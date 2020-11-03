package efgs

import (
	"context"
	"encoding/json"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//DownloadAndSaveKeys Downloads batch from EFGS.
func DownloadAndSaveKeys(ctx context.Context, m pubsub.Message) error {
	var payload efgsapi.BatchDownloadParams

	if decodeErr := pubsub.DecodeJSONEvent(m, &payload); decodeErr != nil {
		return fmt.Errorf("Error while parsing event payload: %v", decodeErr)
	}

	config, err := loadDownloadConfig(ctx)
	if err != nil {
		return err
	}

	return downloadAndSaveKeysBatch(ctx, config, payload)
}

//DownloadAndSaveYesterdaysKeys Downloads batch from yesterday from EFGS.
func DownloadAndSaveYesterdaysKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := loadDownloadConfig(ctx)
	if err == nil {
		err = downloadAndSaveKeysBatch(ctx, config, efgsapi.BatchDownloadParams{
			Date: time.Now().Add(time.Hour * -24).Format("2006-01-02"),
		})
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
	}
}

func downloadAndSaveKeysBatch(ctx context.Context, config *downloadConfig, params efgsapi.BatchDownloadParams) error {
	logger := logging.FromContext(ctx).Named("efgs.downloadAndSaveKeysBatch")

	logger.Infof("About to download batch with tag '%v' for date %v!", params.BatchTag, params.Date)

	logger.Debugf("Using config: %+v", config)

	keys, err := downloadBatchByTag(ctx, config, params.Date, params.BatchTag)
	if err != nil {
		logger.Errorf("Could not download batch from EFGS: %v", err)
		return err
	}

	if len(keys) == 0 {
		logger.Infof("No keys returned from EFGS for date %v and batchTag '%v'", params.Date, params.BatchTag)
		return nil
	}

	logger.Infof("Successfully downloaded %v keys from EFGS, going to upload them", len(keys))

	if err = publishAllKeys(ctx, config, keys); err != nil {
		logger.Errorf("Could not download batch from EFGS: %v", err)
		return err
	}

	logger.Infof("Successfully published downloaded keys to our Key server")

	return nil
}

func publishAllKeys(ctx context.Context, config *downloadConfig, keys []efgsapi.DiagnosisKey) error {
	logger := logging.FromContext(ctx).Named("efgs.publishAllKeys")

	logger.Debugf("About to sort downloaded keys")

	sortedKeys := make(map[string][]keyserverapi.ExposureKey)

	for _, key := range keys {
		mapKey := strings.ToUpper(key.Origin)
		sortedKeys[mapKey] = append(sortedKeys[mapKey], key.ToExposureKey())
	}

	logger.Infof("Sorted downloaded keys into %v groups (countries)", len(sortedKeys))

	var errors []string

	for country, countryKeys := range sortedKeys {
		haid, exists := config.HaidMappings[strings.ToLower(country)]
		if !exists {
			errors = append(errors, fmt.Sprintf("Keys from %v were provided but HAID mapping doesn't exist!", country))
			continue
		}

		logger.Infof("Uploading %v keys from %v to our Key server", len(countryKeys), country)

		if err := PublishKeysToKeyServer(ctx, config.PublishConfig, haid, countryKeys); err != nil {
			logger.Warnf("Could not upload %v keys from %v: %+v", len(countryKeys), country, err)
			errors = append(errors, fmt.Sprintf("Could not upload %v keys from %v country", len(countryKeys), country))
			continue
		}

		logger.Debugf("Successfully imported %v keys from %v", len(countryKeys), country)
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

	keys := batchResponse.Keys

	if len(resp.Header.Get("nextBatchTag")) > 0 && resp.Header.Get("nextBatchTag") != "null" {
		batch, err := downloadBatchByTag(ctx, config, date, resp.Header.Get("nextBatchTag"))
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch[:]...)
	}

	return keys, nil
}
