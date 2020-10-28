package efgs

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsdatabase "github.com/covid19cz/erouska-backend/internal/functions/efgs/database"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
	urlutils "net/url"
	"os"
	"strconv"
	"time"
)

type uploadConfiguration struct {
	URL       *urlutils.URL
	Env       efgsutils.Environment
	NBTLSPair *efgsutils.X509KeyPair
	Client    *http.Client
	BatchTag  string
}

//UploadBatch Called in CRON every 2 hours. Gets keys from database and upload them to EFGS.
func UploadBatch(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("efgs.UploadBatch")
	now := time.Now()
	timeFrom := now.AddDate(0, 0, -14).Format("2006-01-02")
	maxBatchSize, isSet := os.LookupEnv("EFGS_UPLOAD_BATCH_SIZE")
	if !isSet {
		logger.Error("Maximum batch size not set")
		sendErrorResponse(w, errors.New("maximum batch size not set"))
		return
	}

	batchSizeLimit, err := strconv.Atoi(maxBatchSize)
	if err != nil {
		logger.Errorf("Error converting batch size to int: %s", err)
		sendErrorResponse(w, err)
		return
	}

	keys, err := efgsdatabase.Database.GetDiagnosisKeys(timeFrom)
	if err != nil {
		logger.Errorf("Downloading keys error: %s", err)
		sendErrorResponse(w, err)
		return
	}

	if len(keys) <= 0 {
		logger.Info("No new keys in database")
		return
	}

	var diagnosisKeys []*efgsapi.DiagnosisKey
	for _, k := range keys {
		diagnosisKeys = append(diagnosisKeys, k.ToData())
	}

	sortDiagnosisKey(diagnosisKeys)
	uploadConfig, err := uploadBatchConfiguration(ctx)
	if err != nil {
		logger.Errorf("Upload configuration error: %v", err)
		sendErrorResponse(w, err)
		return
	}

	batches := splitBatch(diagnosisKeys, batchSizeLimit)
	for _, batch := range batches {
		diagnosisKeyBatch := makeBatch(batch)
		hash := sha1.New()
		_, _ = hash.Write(batchToBytes(&diagnosisKeyBatch))
		uploadConfig.BatchTag = now.Format("20060102") + "-" + hex.EncodeToString(hash.Sum(nil))[:7]
		logger.Debugf("Uploading batch (%d keys) with tag %s", len(batch), uploadConfig.BatchTag)

		resp, err := uploadBatch(ctx, &diagnosisKeyBatch, uploadConfig)
		if err != nil {
			logger.Errorf("Batch upload failed: %v", err)
			sendErrorResponse(w, err)
			return
		}
		if resp != nil {
			filterInvalidDiagnosisKeys(ctx, resp, keys)
		}
		logger.Debugf("Batch %s successfully uploaded", uploadConfig.BatchTag)
	}

	if err := efgsdatabase.Database.RemoveDiagnosisKey(keys); err != nil {
		logger.Errorf("Removing uploaded keys from database failed: %s", err)
		sendErrorResponse(w, err)
		return
	}

	logger.Infof("%d keys (in %d batches) successfully uploaded", len(keys), len(batches))
}

func uploadBatchConfiguration(ctx context.Context) (*uploadConfiguration, error) {
	logger := logging.FromContext(ctx).Named("efgs.uploadBatchConfiguration")

	efgsEnv := efgsutils.GetEfgsEnvironmentOrFail()

	url := efgsutils.GetEfgsURLOrFail(efgsEnv)
	url.Path = "diagnosiskeys/upload"

	var err error
	config := uploadConfiguration{
		URL: url,
		Env: efgsEnv,
	}

	config.Env = efgsEnv

	config.NBTLSPair, err = efgsutils.LoadX509KeyPair(ctx, efgsEnv, efgsutils.NBTLS)
	if err != nil {
		logger.Debug("Error loading authentication certificate")
		return nil, err
	}

	config.Client, err = efgsutils.NewEFGSClient(ctx, config.NBTLSPair)
	if err != nil {
		logger.Debug("Could not create EFGS client")
		return nil, err
	}

	return &config, nil
}

func uploadBatch(ctx context.Context, batch *efgsapi.DiagnosisKeyBatch, config *uploadConfiguration) (*efgsapi.UploadBatchResponse, error) {
	logger := logging.FromContext(ctx).Named("efgs.uploadBatch")

	raw, err := proto.Marshal(batch)
	if err != nil {
		logger.Debug("Error converting DiagnosisKeyBatch to bytes")
		return nil, err
	}

	req, err := http.NewRequest("POST", config.URL.String(), bytes.NewBuffer(raw))
	if err != nil {
		logger.Debug("Request creating failed")
		return nil, err
	}

	signedBatch, err := signBatch(ctx, config.Env, batch)
	if err != nil {
		logger.Debug("Batch signing error")
		return nil, err
	}

	if config.Env == efgsutils.EnvLocal {
		logger.Debugf("Setting up LOCAL EFGS headers")

		fingerprint, err := efgsutils.GetCertificateFingerprint(ctx, config.NBTLSPair)
		if err != nil {
			logger.Debugf("Fingerprint error")
			return nil, err
		}
		subject, err := efgsutils.GetCertificateSubject(ctx, config.NBTLSPair)
		if err != nil {
			logger.Debugf("Subject error")
			return nil, err
		}
		req.Header.Set("X-SSL-Client-SHA256", fingerprint)
		req.Header.Set("X-SSL-Client-DN", subject)
	}

	req.Header.Set("BatchTag", config.BatchTag)
	req.Header.Set("batchSignature", signedBatch)
	req.Header.Set("Content-Type", "application/protobuf; version=1.0")

	res, err := config.Client.Do(req)
	if err != nil {
		logger.Debug("Batch upload failed")
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Debug("Response parsing body failed")
		return nil, err
	}

	if err := res.Body.Close(); err != nil {
		logger.Debug("Response parsing body failed")
		return nil, err
	}

	logger.Debugf("Code: %d", res.StatusCode)
	logger.Debugf("Body: %s", body)

	switch res.StatusCode {
	case 201:
		logger.Infof("%d keys (tag: %s) successfully uploaded", len(batch.Keys), config.BatchTag)
		return nil, nil
	case 207:
		var parsedResponse efgsapi.UploadBatchResponse

		jsonErr := json.Unmarshal(body, &parsedResponse)
		if jsonErr != nil {
			logger.Debug("Json parsing error")
			return nil, jsonErr
		}

		logger.Debug("Some keys in batch were invalid or duplicated")
		return &parsedResponse, nil
	case 403:
		return nil, errors.New("authentication failed")
	default:
		logger.Debug("Batch upload failed. No key was uploaded")
		return nil, errors.New("batch upload failed")
	}
}

func filterInvalidDiagnosisKeys(ctx context.Context, resp *efgsapi.UploadBatchResponse, keys []*efgsapi.DiagnosisKeyWrapper) []*efgsapi.DiagnosisKeyWrapper {
	logger := logging.FromContext(ctx).Named("efgs.filterInvalidDiagnosisKeys")
	var filteredKeys = keys

	logger.Debugf("Part of batch was successfully uploaded")
	for _, e := range resp.Error {
		filteredKeys = append(filteredKeys[:e], filteredKeys[e+1:]...)
	}
	return filteredKeys
}

func splitBatch(buf []*efgsapi.DiagnosisKey, lim int) [][]*efgsapi.DiagnosisKey {
	var chunk []*efgsapi.DiagnosisKey
	chunks := make([][]*efgsapi.DiagnosisKey, 0, len(buf)/lim+1)

	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}

	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}

	return chunks
}

func sendErrorResponse(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
