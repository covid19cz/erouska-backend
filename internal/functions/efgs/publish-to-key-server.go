package efgs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/utils"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
var client = http.Client{}

//PublishKeysToKeyServer Publish exposure keys to Keys server.
func PublishKeysToKeyServer(ctx context.Context, haid string, maxBatchSize int, keys []keyserverapi.ExposureKey) error {
	logger := logging.FromContext(ctx)

	keyServerConfig, err := utils.LoadKeyServerConfig(ctx)
	if err != nil {
		logger.Fatalf("Could not load key server config: %v", err)
		return err
	}

	verificationServerConfig, err := utils.LoadVerificationServerConfig(ctx)
	if err != nil {
		logger.Fatalf("Could not load verification server config: %v", err)
		return err
	}

	if len(keys) > maxBatchSize {
		batches := splitKeys(keys, maxBatchSize)
		for _, batch := range batches {
			resp, err := signAndPublishKeys(ctx, verificationServerConfig, keyServerConfig, haid, batch)
			if err != nil {
				logger.Errorf("Error when publishing keys: %v", err)
				return err
			}

			logger.Infof("Batch of %v keys uploaded (%v sent), going on", resp.InsertedExposures, len(batch))
		}
	} else {
		// single batch
		_, err = signAndPublishKeys(ctx, verificationServerConfig, keyServerConfig, haid, keys)
		if err != nil {
			logger.Errorf("Error when publishing keys: %v", err)
			return err
		}
	}

	logger.Info("Keys uploaded to Key server")

	return nil
}

func signAndPublishKeys(ctx context.Context, verificationServerConfig *utils.VerificationServerConfig, keyServerConfig *utils.KeyServerConfig, haid string, keys []keyserverapi.ExposureKey) (*keyserverapi.PublishResponse, error) {
	logger := logging.FromContext(ctx)

	vc, err := requestNewVC(ctx, *verificationServerConfig)
	if err != nil {
		logger.Debugf("Error when getting VC: %v", err)
		return nil, err
	}

	token, err := verifyCode(ctx, *verificationServerConfig, vc)
	if err != nil {
		logger.Debugf("Error when getting token: %v", err)
		return nil, err
	}

	hmacKey := make([]byte, 16)
	_, _ = seededRand.Read(hmacKey)

	certificate, err := getCertificate(ctx, *verificationServerConfig, keys, token, hmacKey)
	if err != nil {
		logger.Debugf("Error when getting certificate: %v", err)
		return nil, err
	}

	resp, err := publishKeys(ctx, *keyServerConfig, haid, keys, certificate, hmacKey)

	if err != nil {
		logger.Debugf("Error when publishing keys to Key server: %v", err)
		return nil, err
	}

	return resp, nil
}

func requestNewVC(ctx context.Context, config utils.VerificationServerConfig) (string, error) {
	logger := logging.FromContext(ctx)

	body, err := json.Marshal(&issueCodeRequest{
		Phone:    "",
		TestType: "confirmed",
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.GetAdminURL("api/issue"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-api-key", config.AdminKey)

	logger.Debugf("Requesting VC. Request: %+v", req)

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if err := response.Body.Close(); err != nil {
		return "", err
	}

	var r issueCodeResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return "", err
	}
	logger.Debugf("Response: %+v", r)

	if r.ErrorCode != "" || r.Error != "" {
		return "", fmt.Errorf("%v: %+v", r.ErrorCode, r.Error)
	}

	return r.Code, nil
}

func verifyCode(ctx context.Context, config utils.VerificationServerConfig, code string) (string, error) {
	logger := logging.FromContext(ctx)

	body, err := json.Marshal(&verifyRequest{
		Code: code,
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.GetDeviceURL("api/verify"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-api-key", config.DeviceKey)

	logger.Debugf("Requesting token. Request: %+v", req)

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if err := response.Body.Close(); err != nil {
		return "", err
	}

	var r verifyResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return "", err
	}
	logger.Debugf("Response: %+v", r)

	if r.ErrorCode != "" || r.Error != "" {
		return "", fmt.Errorf("%v: %+v", r.ErrorCode, r.Error)
	}

	return r.Token, nil
}

func getCertificate(ctx context.Context, config utils.VerificationServerConfig, keys []keyserverapi.ExposureKey, token string, hmacKey []byte) (string, error) {
	logger := logging.FromContext(ctx)

	hmac, err := efgsutils.CalculateExposureKeysHMAC(keys, hmacKey)
	if err != nil {
		logger.Debugf("Error: %v", err)
		return "", err
	}

	body, err := json.Marshal(&certificateRequest{
		Token:   token,
		KeyHmac: base64.StdEncoding.EncodeToString(hmac),
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.GetDeviceURL("api/certificate"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-api-key", config.DeviceKey)

	logger.Debugf("Getting certificate. Request: %+v", req)

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if err := response.Body.Close(); err != nil {
		return "", err
	}

	var r certificateResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return "", err
	}
	logger.Debugf("Response: %+v", r)

	if r.ErrorCode != "" || r.Error != "" {
		return "", fmt.Errorf("%v: %+v", r.ErrorCode, r.Error)
	}

	return r.Certificate, nil
}

func publishKeys(ctx context.Context, config utils.KeyServerConfig, haid string, keys []keyserverapi.ExposureKey, certificate string, secret []byte) (*keyserverapi.PublishResponse, error) {
	logger := logging.FromContext(ctx)

	keysCount := len(keys)
	logger.Infof("Publishing %v keys with HAID %v", keysCount, haid)

	var publishRequest = keyserverapi.Publish{
		Keys:                 keys,
		HealthAuthorityID:    haid,
		VerificationPayload:  certificate,
		HMACKey:              base64.StdEncoding.EncodeToString(secret),
		SymptomOnsetInterval: 0,
		Traveler:             false,
		RevisionToken:        "",
		Padding:              "",
	}

	body, err := json.Marshal(&publishRequest)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", config.GetURL("v1/publish"), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")

	logger.Debugf("Request: %+v", req)

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err := response.Body.Close(); err != nil {
		return nil, err
	}

	var r keyserverapi.PublishResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Response: %+v", r)

	if r.Code != "" || r.ErrorMessage != "" {
		return nil, fmt.Errorf("%v: %+v", r.Code, r.ErrorMessage)
	}

	if r.InsertedExposures != keysCount {
		logger.Infof("Not all exposures were inserted: %v sent, %v inserted", keysCount, r.InsertedExposures)
	}

	return &r, nil
}

func splitKeys(buf []keyserverapi.ExposureKey, lim int) [][]keyserverapi.ExposureKey {
	var chunk []keyserverapi.ExposureKey
	chunks := make([][]keyserverapi.ExposureKey, 0, len(buf)/lim+1)

	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}

	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}

	return chunks
}
