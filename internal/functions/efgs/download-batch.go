package efgs

import (
	"context"
	"encoding/json"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"github.com/sethvargo/go-envconfig"
	"io/ioutil"
	"net/http"
	urlutils "net/url"
	"os"
	"strings"
	"time"
)

type config struct {
	HaidMappings map[string]string
	MaxBatchSize int `env:"MAX_UPLOAD_KEYS,default=30"`
}

//DownloadAndSaveKeys Downloads batch from EFGS.
func DownloadAndSaveKeys(ctx context.Context, m pubsub.Message) error {
	var payload efgsapi.BatchDownloadParams

	if decodeErr := pubsub.DecodeJSONEvent(m, &payload); decodeErr != nil {
		return fmt.Errorf("Error while parsing event payload: %v", decodeErr)
	}

	return downloadAndSaveKeysBatch(ctx, payload)
}

//DownloadAndSaveYesterdaysKeys Downloads batch from yesterday from EFGS.
func DownloadAndSaveYesterdaysKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := downloadAndSaveKeysBatch(ctx, efgsapi.BatchDownloadParams{
		Date: time.Now().Add(time.Hour * -24).Format("2006-01-02"),
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
	}
}

func downloadAndSaveKeysBatch(ctx context.Context, params efgsapi.BatchDownloadParams) error {
	logger := logging.FromContext(ctx).Named("efgs.downloadAndSaveKeysBatch")

	logger.Infof("About to download batch with tag '%v' for date %v!", params.BatchTag, params.Date)

	config, err := loadConfig(ctx)
	if err != nil {
		logger.Errorf("Could not load config: %v", err)
		return err
	}

	logger.Debugf("Using config: %+v", config)

	keys, err := downloadBatchByTag(ctx, params.Date, params.BatchTag)
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

func publishAllKeys(ctx context.Context, config *config, keys []efgsapi.DiagnosisKey) error {
	logger := logging.FromContext(ctx).Named("efgs.publishAllKeys")

	logger.Debugf("About to sort downloaded keys")

	sortedKeys := make(map[string][]keyserverapi.ExposureKey)

	for _, key := range keys {
		mapKey := strings.ToLower(key.Origin)
		sortedKeys[mapKey] = append(sortedKeys[mapKey], key.ToExposureKey())
	}

	logger.Infof("Sorted downloaded keys into %v groups (countries)", len(sortedKeys))

	var errors []string

	for country, countryKeys := range sortedKeys {
		haid, exists := config.HaidMappings[country]
		if !exists {
			errors = append(errors, fmt.Sprintf("Keys from %v were provided but HAID mapping doesn't exist!", country))
			continue
		}

		logger.Infof("Uploading %v keys from %v to our Key server", len(countryKeys), country)

		if err := PublishKeysToKeyServer(ctx, haid, config.MaxBatchSize, countryKeys); err != nil {
			errors = append(errors, fmt.Sprintf("Could not upload keys from %v country", country))
			continue
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("Following errors have happened:\n%v", strings.Join(errors, "\n"))
	}

	return nil
}

func downloadBatchByTag(ctx context.Context, date string, batchTag string) ([]efgsapi.DiagnosisKey, error) {
	logger := logging.FromContext(ctx).Named("efgs.downloadBatchByTag")
	secretsClient := secrets.Client{}

	nbtlsPair, err := efgsutils.LoadX509KeyPair(ctx, efgsutils.NBTLS)
	if err != nil {
		logger.Fatalf("Error loading authentication certificate: %v", err)
		return nil, err
	}

	client, err := efgsutils.NewEFGSClient(ctx, nbtlsPair)
	if err != nil {
		logger.Errorf("Could not create EFGS client: %v", err)
		return nil, err
	}

	var req *http.Request

	// TODO remove while switching to real efgs
	_, connectToLocal := os.LookupEnv("EFGS_LOCAL")

	var efgsRootURL []byte
	if connectToLocal {
		efgsRootURL, err = secretsClient.Get("efgs-test-url")
	} else {
		efgsRootURL, err = secretsClient.Get("efgs-root-url")
	}

	if err != nil {
		return nil, err
	}

	url, err := urlutils.Parse(string(efgsRootURL))
	if err != nil {
		return nil, err
	}

	url.Path = "diagnosiskeys/download/" + date

	req, err = http.NewRequest("GET", url.String(), nil)
	if err != nil {
		logger.Error("Error creating download request")
		return nil, err
	}

	req.Header.Set("Accept", "application/json; version=1.0")
	if batchTag != "" {
		req.Header.Set("batchTag", batchTag)
	}

	if connectToLocal {
		logger.Debugf("Setting up LOCAL EFGS headers")

		fingerprint, err := efgsutils.GetCertificateFingerprint(ctx, nbtlsPair)
		if err != nil {
			return nil, err
		}
		subject, err := efgsutils.GetCertificateSubject(ctx, nbtlsPair)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-SSL-Client-SHA256", fingerprint)
		req.Header.Set("X-SSL-Client-DN", subject)
	}

	logger.Debugf("Download request: %+v", req)

	resp, err := client.Do(req)

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
		batch, err := downloadBatchByTag(ctx, date, resp.Header.Get("nextBatchTag"))
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch[:]...)
	}

	return keys, nil
}

func loadConfig(ctx context.Context) (*config, error) {
	var config config
	if err := envconfig.Process(ctx, &config); err != nil {
		return nil, err
	}

	secretsClient := secrets.Client{}
	bytes, err := secretsClient.Get("efgs-haid-mappings")
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(bytes, &config.HaidMappings); err != nil {
		return nil, err
	}

	return &config, nil
}
