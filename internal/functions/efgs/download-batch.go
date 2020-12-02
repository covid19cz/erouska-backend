package efgs

import (
	"context"
	"encoding/json"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsconstants "github.com/covid19cz/erouska-backend/internal/functions/efgs/constants"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	redisclient "github.com/go-redis/redis/v8"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"github.com/stretchr/stew/slice"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const czCode = "CZ"

//DownloadAndSaveKeys Downloads batch from EFGS.
func DownloadAndSaveKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named("efgs.DownloadAndSaveKeys")

	config, err := loadDownloadConfig(ctx)
	if err == nil {
		err = downloadAndSaveKeys(ctx, config)
	}

	if err != nil {
		logger.Errorf("Could not process: %+v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
	}
}

//DownloadAndSaveYesterdaysKeys Downloads batch from whole yesterday from EFGS.
func DownloadAndSaveYesterdaysKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named("efgs.DownloadAndSaveYesterdaysKeys")

	config, err := loadDownloadConfig(ctx)
	if err != nil {
		logger.Errorf("Could not load config: %+v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
		return
	}

	_, err = downloadAllRecursively(ctx, config, &efgsapi.BatchDownloadParams{
		Date: time.Now().Add(time.Hour * -24).Format("2006-01-02"),
	})

	if err != nil {
		logger.Errorf("Could not download all data: %+v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
	}
}

func downloadAndSaveKeys(ctx context.Context, config *downloadConfig) error {
	logger := logging.FromContext(ctx).Named("efgs.downloadAndSaveKeys")

	mutex, err := config.MutexManager.Lock(efgsconstants.MutexNameDownloadAndSaveKeys)
	if err != nil {
		return fmt.Errorf("Could not acquire '%v' mutex: %v", efgsconstants.MutexNameDownloadAndSaveKeys, err)
	}
	defer mutex.Unlock()

	now := time.Now()

	batchParams := efgsapi.BatchDownloadParams{
		Date:     now.Format("2006-01-02"),
		BatchTag: now.Format("20060102") + "-1",
	}

	logger.Debugf("Looking for batch params to be downloaded")

	val, err := config.RedisClient.Get(efgsconstants.RedisKeyNextBatch)
	if err != nil {
		if err == redisclient.Nil {
			logger.Debugf("Batch params wasn't found, using default: %v", batchParams)
		} else {
			err := fmt.Errorf("Error while querying Redis: %+v", err)
			logger.Error(err)
			return err
		}
	} else {
		var savedBatchParams efgsapi.BatchDownloadParams
		if err := json.Unmarshal([]byte(val), &savedBatchParams); err != nil {
			err := fmt.Errorf("Could not unmarshall saved batch params: %+v", err)
			logger.Error(err)
			return err
		}

		if savedBatchParams.Date == batchParams.Date {
			batchParams = savedBatchParams
			logger.Debugf("Batch params found: %v", batchParams)
		} else {
			logger.Debugf("Found batch params from another day, discarding")
		}
	}

	nextBatch, err := downloadAndSaveKeysBatch(ctx, config, batchParams)

	if nextBatch != nil {
		logger.Debugf("Next batch will be: %+v", nextBatch)
		bytes, err := json.Marshal(*nextBatch)
		if err != nil {
			return err
		}

		if err = config.RedisClient.Set(efgsconstants.RedisKeyNextBatch, string(bytes), 0); err != nil {
			logger.Errorf("Could not save next batch params to Redis: %+v", err)
			return err
		}
	}

	return err
}

func downloadAndSaveKeysBatch(ctx context.Context, config *downloadConfig, params efgsapi.BatchDownloadParams) (*efgsapi.BatchDownloadParams, error) {
	logger := logging.FromContext(ctx).Named("efgs.downloadAndSaveKeysBatch")

	logger.Infof("About to download batch with tag '%v' for date %v!", params.BatchTag, params.Date)

	logger.Debugf("Using config: %+v", config)

	keys, err := downloadBatchByTag(ctx, config, params.Date, params.BatchTag)
	if err != nil {
		logger.Errorf("Could not download batch from EFGS: %v", err)
		return nil, err
	}

	if len(keys) == 0 {
		logger.Infof("No keys returned from EFGS for date %v and batchTag '%v'", params.Date, params.BatchTag)
		return nil, nil
	}

	logger.Infof("Successfully downloaded %v keys from EFGS, going to enqueue them", len(keys))

	if err = enqueueForImport(ctx, config, keys); err != nil {
		logger.Errorf("Could not download batch from EFGS: %v", err)
		return nil, err
	}

	logger.Infof("Successfully enqueued downloaded keys for import to our Key server")

	nextBatch := nextBatchParams(&params)

	return &nextBatch, nil
}

func nextBatchParams(last *efgsapi.BatchDownloadParams) efgsapi.BatchDownloadParams {
	today := time.Now().Format("2006-01-02")
	batchTagPrefix := time.Now().Format("20060102")

	var nextBatchTag string

	if last.Date == today && last.BatchTag != "" {
		parts := strings.Split(last.BatchTag, "-")
		nextID, err := strconv.Atoi(parts[1])
		if err != nil {
			panic(fmt.Sprintf("Unexpected format of EFGS batch tag: '%v'", last.BatchTag))
		}
		nextBatchTag = fmt.Sprintf("%v-%v", batchTagPrefix, nextID+1)
	} else {
		nextBatchTag = batchTagPrefix + "-2" // '2' because '1' is what one gets without explicit tag
	}

	return efgsapi.BatchDownloadParams{
		Date:     today,
		BatchTag: nextBatchTag,
	}
}

func enqueueForImport(ctx context.Context, config *downloadConfig, keys []efgsapi.DiagnosisKey) error {
	logger := logging.FromContext(ctx).Named("efgs.enqueueForImport")

	now := time.Now().Add(30 - time.Minute)

	logger.Debugf("About to sort downloaded keys")

	sortedKeys := make(map[string][]keyserverapi.ExposureKey)

	skippedKeys := 0

	for _, key := range keys {
		// filter out keys that are too old
		if !isRecent(&key, now, config.MaxIntervalAge) {
			skippedKeys += 1
			continue
		}

		var origin string

		// Import keys that relates to us as our own keys (#171)
		// The comparison is case-insensitive as the code should be ISO-3166-1 alpha 2
		if slice.ContainsString(key.VisitedCountries, czCode) {
			origin = czCode
		} else {
			origin = key.Origin
		}

		mapKey := strings.ToUpper(origin)
		sortedKeys[mapKey] = append(sortedKeys[mapKey], key.ToExposureKey())
	}

	logger.Debugf("Sorted keys into %v groups (countries), %v keys skipped", len(sortedKeys), skippedKeys)

	var errors []string

	for country, countryKeys := range sortedKeys {
		haid, exists := config.HaidMappings[strings.ToLower(country)]
		if !exists {
			errors = append(errors, fmt.Sprintf("Keys from %v were provided but HAID mapping doesn't exist!", country))
			continue
		}

		batches := splitKeys(countryKeys, config.MaxKeysOnPublish, config.MaxSameStartIntervalKeys)

		logger.Infof("Enqueuing %v batches for import with HAID %v", len(batches), haid)

		for _, batch := range batches {
			batchParams := efgsapi.BatchImportParams{
				HAID: haid,
				Keys: batch,
			}

			logger.Debugf("Enqueuing batch of %v keys from %v for import", len(batch), country)

			if efgsutils.EfgsExtendedLogging {
				logger.Debugf("Enqueued batch: %+v", batchParams)
			}

			if err := config.PubSubClient.Publish(efgsconstants.TopicNameImportKeys, batchParams); err != nil {
				msg := fmt.Sprintf("Error while enqueuing keys from %v: %+v", country, err)
				logger.Warn(msg)
				errors = append(errors, msg)
				continue
			}
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("Following errors have happened:\n%v", strings.Join(errors, "\n"))
	}

	return nil
}

func downloadBatchByTag(ctx context.Context, config *downloadConfig, date string, batchTag string) ([]efgsapi.DiagnosisKey, error) {
	logger := logging.FromContext(ctx).Named("efgs.downloadBatchByTag")

	url := config.URL

	url.Path = "diagnosiskeys/download/" + date

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		logger.Error("Error creating download request")
		return nil, err
	}

	req.Header.Set("Accept", "application/json; version=1.0")
	if batchTag != "" {
		req.Header.Set("batchTag", batchTag)
	}

	if config.Env == efgsutils.EnvLocal {
		logger.Debugf("Setting up LOCAL EFGS headers")

		fingerprint, err := efgsutils.GetCertificateFingerprint(ctx, config.NBTLSPair)
		if err != nil {
			return nil, err
		}
		subject, err := efgsutils.GetCertificateSubject(ctx, config.NBTLSPair)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-SSL-Client-SHA256", fingerprint)
		req.Header.Set("X-SSL-Client-DN", subject)
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Download request: %+v", req)
	}

	resp, err := config.Client.Do(req)

	if err != nil {
		logger.Errorf("Error while downloading batch: %v", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := resp.Body.Close(); err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return []efgsapi.DiagnosisKey{}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %v: %v", resp.StatusCode, string(body))
	}

	var batchResponse efgsapi.DownloadBatchResponse

	if err = json.Unmarshal(body, &batchResponse); err != nil {
		logger.Debugf("Download response parsing error: %v, body: %v", err, string(body))
		return nil, err
	}

	return batchResponse.Keys, nil
}

func downloadAllRecursively(ctx context.Context, config *downloadConfig, params *efgsapi.BatchDownloadParams) (*efgsapi.BatchDownloadParams, error) {
	nextBatch, err := downloadAndSaveKeysBatch(ctx, config, *params)
	if err != nil {
		return nil, err
	}
	if nextBatch != nil {
		return downloadAllRecursively(ctx, config, nextBatch)
	}

	return nil, nil
}
