package efgs

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
	"time"
)

//UploadBatch Called in CRON every 2 hours. Gets keys from database and upload them to EFGS.
func UploadBatch(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("efgs.UploadBatch")
	now := time.Now()
	timeFrom := now.AddDate(0, 0, -14).Format("2006-01-02")

	uploadConfig, err := loadUploadConfig(ctx)
	if err != nil {
		logger.Errorf("Upload configuration error: %v", err)
		sendErrorResponse(w, err)
		return
	}

	if err = uploadAndRemoveBatch(ctx, uploadConfig, timeFrom, now); err != nil {
		logger.Errorf("Upload error: %v", err)
		sendErrorResponse(w, err)
		return
	}
}

func uploadAndRemoveBatch(ctx context.Context, uploadConfig *uploadConfig, timeFrom string, timeTo time.Time) error {
	logger := logging.FromContext(ctx).Named("efgs.uploadAndRemoveBatch")

	keys, err := uploadConfig.Database.GetDiagnosisKeys(timeFrom)
	if err != nil {
		return fmt.Errorf("Downloading keys error: %s", err)
	}

	if len(keys) <= 0 {
		logger.Info("No new keys in database")
		return nil
	}

	var diagnosisKeys []*efgsapi.DiagnosisKey
	for _, k := range keys {
		diagnosisKeys = append(diagnosisKeys, k.ToData())
	}

	sortDiagnosisKey(diagnosisKeys)

	batches := splitBatch(diagnosisKeys, uploadConfig.BatchSizeLimit)
	for _, batch := range batches {
		diagnosisKeyBatch := makeBatch(batch)
		hash := sha1.New()
		_, _ = hash.Write(batchToBytes(&diagnosisKeyBatch))
		uploadConfig.BatchTag = timeTo.Format("20060102") + "-" + hex.EncodeToString(hash.Sum(nil))[:7]
		logger.Debugf("Uploading batch (%d keys) with tag %s", len(batch), uploadConfig.BatchTag)

		resp, err := uploadBatch(ctx, &diagnosisKeyBatch, uploadConfig)
		if err != nil {
			return fmt.Errorf("Batch upload failed: %v", err)
		}
		if resp != nil {
			logger.Debugf("Batch was partially invalid and therefore rejected")
			// TODO Matej
			filterInvalidDiagnosisKeys(resp, keys)
			return nil
		}

		logger.Debugf("Batch %s successfully uploaded", uploadConfig.BatchTag)
	}

	if err := uploadConfig.Database.RemoveDiagnosisKey(keys); err != nil {
		return fmt.Errorf("Removing uploaded keys from database failed: %s", err)
	}

	logger.Infof("%d keys (in %d batches) successfully uploaded", len(keys), len(batches))

	return nil
}

func uploadBatch(ctx context.Context, batch *efgsapi.DiagnosisKeyBatch, config *uploadConfig) (*efgsapi.UploadBatchResponse, error) {
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

		if err = json.Unmarshal(body, &parsedResponse); err != nil {
			logger.Debug("Json parsing error")
			return nil, err
		}

		logger.Debug("Some keys in batch were invalid or duplicated: %+v", parsedResponse)
		return &parsedResponse, nil
	case 403:
		return nil, errors.New("authentication failed")
	default:
		logger.Debug("Batch upload failed. No key was uploaded")
		return nil, errors.New("batch upload failed")
	}
}

func filterInvalidDiagnosisKeys(resp *efgsapi.UploadBatchResponse, keys []*efgsapi.DiagnosisKeyWrapper) []*efgsapi.DiagnosisKeyWrapper {
	var filteredKeys = keys

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
