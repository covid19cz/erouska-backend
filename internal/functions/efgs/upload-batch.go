package efgs

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"math"
	"net/http"
	"time"
)

//UploadBatch Called in CRON every 2 hours. Gets keys from database and upload them to EFGS.
func UploadBatch(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("efgs.UploadBatch")

	uploadConfig, err := loadUploadConfig(ctx)
	if err != nil {
		logger.Errorf("Upload configuration error: %v", err)
		sendErrorResponse(w, err)
		return
	}

	now := time.Now()
	loadKeysSince := time.Now().AddDate(0, 0, -14) // TODO make 14 configurable?

	if err = uploadAndRemoveBatch(ctx, uploadConfig, now, loadKeysSince); err != nil {
		logger.Errorf("Upload error: %v", err)
		sendErrorResponse(w, err)
		return
	}
}

func uploadAndRemoveBatch(ctx context.Context, uploadConfig *uploadConfig, now time.Time, loadKeysSince time.Time) error {
	logger := logging.FromContext(ctx).Named("efgs.uploadAndRemoveBatch")

	keys, err := uploadConfig.Database.GetDiagnosisKeys(loadKeysSince)
	if err != nil {
		return fmt.Errorf("DB loading keys error: %s", err)
	}

	if len(keys) <= 0 {
		logger.Info("No new keys in database")
		return nil
	}

	batches := splitBatch(keys, uploadConfig.BatchSizeLimit)

	for _, batch := range batches {
		var diagnosisKeys []*efgsapi.DiagnosisKey
		for _, k := range batch {
			diagnosisKeys = append(diagnosisKeys, k.ToData())
		}
		sortDiagnosisKey(diagnosisKeys)

		diagnosisKeyBatch := makeBatch(diagnosisKeys)
		uploadConfig.BatchTag = calculateBatchTag(now, &diagnosisKeyBatch)

		logger.Debugf("Uploading batch (%d keys) with tag %s", len(batch), uploadConfig.BatchTag)

		resp, err := uploadBatch(ctx, &diagnosisKeyBatch, uploadConfig)
		if err != nil {
			return fmt.Errorf("Batch upload failed: %v", err)
		}

		if resp.StatusCode != 201 { // nothing was saved to EFGS
			logger.Debugf("Batch was partially invalid and therefore rejected")

			if err := handleErrorUploadResponse(resp, uploadConfig, batch); err != nil {
				return fmt.Errorf("Handling upload response ended with error: %s", err)
			}
		} else {
			if err := uploadConfig.Database.RemoveDiagnosisKeys(batch); err != nil {
				return fmt.Errorf("Removing uploaded keys from database failed: %s", err)
			}
			logger.Debugf("Batch %s successfully uploaded", uploadConfig.BatchTag)
		}
	}

	logger.Infof("%d batches successfully uploaded", len(batches))

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

	defer res.Body.Close()

	logger.Debugf("Code: %d", res.StatusCode)
	logger.Debugf("Body: %s", body)

	var parsedResponse efgsapi.UploadBatchResponse
	parsedResponse.StatusCode = res.StatusCode

	switch res.StatusCode {
	case 201:
		logger.Infof("%d keys (tag: %s) successfully uploaded", len(batch.Keys), config.BatchTag)
		return &parsedResponse, nil
	case 207:
		if err = json.Unmarshal(body, &parsedResponse); err != nil {
			logger.Debug("Json parsing error: %+v", err)
			return nil, err
		}

		logger.Debug("Some keys in batch were invalid or duplicated: %+v", parsedResponse)
		return &parsedResponse, nil
	default:
		logger.Debug("Batch upload failed. No key was uploaded")
		return &parsedResponse, nil
	}
}

func handleErrorUploadResponse(resp *efgsapi.UploadBatchResponse, uploadConfig *uploadConfig, keys []*efgsapi.DiagnosisKeyWrapper) error {
	if resp.StatusCode == 400 {
		for _, key := range keys {
			key.Retries++
		}

		return uploadConfig.Database.UpdateKeys(keys)
	}

	for _, keyIndex := range resp.Error {
		key := keys[keyIndex]
		key.Retries++
	}

	for _, keyIndex := range resp.Duplicate {
		keys[keyIndex].Retries = math.MaxInt64
	}

	for _, keyIndex := range resp.Success {
		keys[keyIndex].Retries = math.MaxInt64
	}

	if err := uploadConfig.Database.UpdateKeys(keys); err != nil {
		return err
	}

	return uploadConfig.Database.RemoveInvalidKeys(keys)
}

func splitBatch(buf []*efgsapi.DiagnosisKeyWrapper, lim int) [][]*efgsapi.DiagnosisKeyWrapper {
	var chunk []*efgsapi.DiagnosisKeyWrapper
	chunks := make([][]*efgsapi.DiagnosisKeyWrapper, 0, len(buf)/lim+1)

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

func calculateBatchTag(date time.Time, batch *efgsapi.DiagnosisKeyBatch) string {
	hash := sha1.New()
	_, _ = hash.Write(batchToBytes(batch))
	return date.Format("20060102") + "-" + hex.EncodeToString(hash.Sum(nil))[:7]
}
