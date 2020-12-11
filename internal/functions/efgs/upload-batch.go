package efgs

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"firebase.google.com/go/db"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/realtimedb"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strings"
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

	var errors []string

	for _, batchDbKeys := range batches {
		var diagnosisKeys []*efgsapi.DiagnosisKey
		for _, k := range batchDbKeys {
			diagnosisKeys = append(diagnosisKeys, k.ToData())
		}
		sortDiagnosisKey(diagnosisKeys)

		diagnosisKeyBatch := makeBatch(diagnosisKeys)
		uploadConfig.BatchTag = calculateBatchTag(now, &diagnosisKeyBatch)
		batchKeysCount := len(batchDbKeys)

		logger.Debugf("Uploading batch (%d keys) with tag %s", batchKeysCount, uploadConfig.BatchTag)

		resp, err := uploadBatch(ctx, &diagnosisKeyBatch, uploadConfig)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Batch upload failed: %v", err))
			continue
		}

		if resp.StatusCode == 201 {
			if err := uploadConfig.Database.RemoveDiagnosisKeys(batchDbKeys); err != nil {
				errors = append(errors, fmt.Sprintf("Removing uploaded keys from database failed: %s", err))
				continue
			}
			logger.Debugf("Batch %s successfully uploaded", uploadConfig.BatchTag)

			if err = updateUploadedCounters(ctx, uploadConfig.RealtimeDBClient, batchKeysCount); err != nil {
				logger.Warnf("Could not update EFGS upload counters: %v", err)
			}
		} else { // nothing was saved to EFGS
			msg := fmt.Sprintf("Batch %s was partially invalid and therefore rejected", uploadConfig.BatchTag)
			logger.Debugf(msg)
			errors = append(errors, msg)

			if err := handleErrorUploadResponse(ctx, resp, uploadConfig, batchDbKeys, diagnosisKeys); err != nil {
				errors = append(errors, fmt.Sprintf("Handling upload response ended with error: %s", err))
				continue
			}
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("Following errors have happened, only %v batches has been uploaded:\n%v", len(batches)-len(errors), strings.Join(errors, "\n"))
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

		logger.Debugf("Some keys in batch were invalid or duplicated: %+v", parsedResponse)
		return &parsedResponse, nil
	default:
		logger.Debug("Batch upload failed. No key was uploaded")
		return &parsedResponse, fmt.Errorf("HTTP %v: %v", res.StatusCode, string(body))
	}
}

func handleErrorUploadResponse(ctx context.Context, resp *efgsapi.UploadBatchResponse, uploadConfig *uploadConfig, batchDbKeys []*efgsapi.DiagnosisKeyWrapper, diagnosesKeys []*efgsapi.DiagnosisKey) error {
	logger := logging.FromContext(ctx).Named("efgs.handleErrorUploadResponse")

	// The magic in this method is needed because EFGS uses weird sorting and we need to map indexes reported by EFGS (indexes in uplod batch)
	// to index resp. key in the original batch. I'd love to implement this some less mind-blowing way but in a language where one has to
	// implement a function for getting all keys/values of map, this way is just a smaller pain.

	// This is needed for binary-search below
	sort.Slice(batchDbKeys, func(i, j int) bool {
		return bytes.Compare(batchDbKeys[i].KeyData, batchDbKeys[j].KeyData) > 0
	})

	// Find DB entity relevant to DiagnosisKey with given index
	findRelevantKey := func(index int) *efgsapi.DiagnosisKeyWrapper {
		lookFor := diagnosesKeys[index].KeyData

		// This does binary search.
		pos := sort.Search(len(batchDbKeys), func(i int) bool {
			return bytes.Compare(batchDbKeys[i].KeyData, lookFor) <= 0
		})

		if pos < len(batchDbKeys) && pos >= 0 {
			return batchDbKeys[pos]
		}

		return nil
	}

	// Handle keys when the whole batch was rejected. Increase their retries counter, will be removed if the value is too high
	if resp.StatusCode == 400 {
		for _, key := range batchDbKeys {
			key.Retries++
		}

		logger.Debugf("Updating retries for failed keys")
		return uploadConfig.Database.UpdateKeys(batchDbKeys)
	}

	// Handle errored keys - increase their retries counter, will be removed if the value is too high
	for _, keyIndex := range resp.Error {
		if key := findRelevantKey(keyIndex); key != nil {
			key.Retries++
		}
	}

	// Handle duplicate (already uploaded) keys - should be deleted from our DB
	for _, keyIndex := range resp.Duplicate {
		if key := findRelevantKey(keyIndex); key != nil {
			key.Retries = math.MaxInt64
		}
	}

	// Handle potentially successful keys - must be retried!
	for _, keyIndex := range resp.Success {
		if key := findRelevantKey(keyIndex); key != nil {
			key.Retries = 1
		}
	}

	// Update the keys in DB
	if err := uploadConfig.Database.UpdateKeys(batchDbKeys); err != nil {
		return err
	}

	logger.Debugf("Removing invalid keys from DB")
	return uploadConfig.Database.RemoveInvalidKeys()
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

func updateUploadedCounters(ctx context.Context, client *realtimedb.Client, keysCount int) error {
	logger := logging.FromContext(ctx).Named("efgs.upload-batch.updateUploadedCounters")

	var date = utils.GetTimeNow().Format("20060102")

	// update daily counter
	if err := updateUploadedCounter(ctx, client, constants.DbEfgsCountersPrefix+date, keysCount); err != nil {
		logger.Warnf("Cannot increase EFGS counter due to unknown error: %+v", err.Error())
		return err
	}

	// update total counter
	if err := updateUploadedCounter(ctx, client, constants.DbEfgsCountersPrefix+"total", keysCount); err != nil {
		logger.Warnf("Cannot increase EFGS counter due to unknown error: %+v", err.Error())
		return err
	}

	return nil
}

func updateUploadedCounter(ctx context.Context, client *realtimedb.Client, dbKey string, keysCount int) error {
	logger := logging.FromContext(ctx).Named("efgs.upload-batch.updateUploadedCounter")

	return client.RunTransaction(ctx, dbKey, func(tn db.TransactionNode) (interface{}, error) {
		var state structs.EfgsCounter

		if err := tn.Unmarshal(&state); err != nil {
			return nil, err
		}

		logger.Debugf("Found counter state, dbKey %v: %+v", dbKey, state)

		state.KeysUploaded += keysCount

		logger.Debugf("Saving updated counter state, dbKey %v: %+v", dbKey, state)

		return state, nil
	})
}
