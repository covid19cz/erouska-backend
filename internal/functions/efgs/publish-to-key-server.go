package efgs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

//PublishKeysToKeyServer Publish exposure keys to Keys server.
func PublishKeysToKeyServer(ctx context.Context, config *publishConfig, haid string, keys []ExpKey) error {
	logger := logging.FromContext(ctx).Named("efgs.PublishKeysToKeyServer")

	// Create batches from keys but first filter out keys that are too old.
	recentKeys := filterRecentKeys(keys, config.MaxIntervalAge)
	batches := splitKeys(recentKeys, config.MaxKeysOnPublish, config.MaxSameStartIntervalKeys)

	//TODO rate limiting

	for _, batch := range batches {
		resp, err := signAndPublishKeys(ctx, config, haid, batch)
		if err != nil {
			logger.Debugf("Error when publishing keys: %v", err)
			return err
		}

		logger.Infof("Batch of %v keys uploaded (%v sent), going on", resp.InsertedExposures, len(batch))
	}

	logger.Info("Keys uploaded to Key server")

	return nil
}

func signAndPublishKeys(ctx context.Context, config *publishConfig, haid string, keys ExpKeyBatch) (*keyserverapi.PublishResponse, error) {
	logger := logging.FromContext(ctx).Named("efgs.signAndPublishKeys")

	vc, err := requestNewVC(ctx, config)
	if err != nil {
		logger.Debugf("Error when getting VC: %v", err)
		return nil, err
	}

	token, err := verifyCode(ctx, config, vc)
	if err != nil {
		logger.Debugf("Error when getting token: %v", err)
		return nil, err
	}

	hmacKey := make([]byte, 16)
	_, _ = seededRand.Read(hmacKey)

	certificate, err := getCertificate(ctx, config, keys, token, hmacKey)
	if err != nil {
		logger.Debugf("Error when getting certificate: %v", err)
		return nil, err
	}

	resp, err := publishKeys(ctx, config, haid, keys, certificate, hmacKey)

	if err != nil {
		logger.Debugf("Error when publishing keys to Key server: %v", err)
		return nil, err
	}

	return resp, nil
}

func requestNewVC(ctx context.Context, config *publishConfig) (string, error) {
	logger := logging.FromContext(ctx).Named("efgs.requestNewVC")

	body, err := json.Marshal(&efgsapi.IssueCodeRequest{
		TestType:    "confirmed",
		SymptomDate: time.Now().AddDate(0, 0, -3).Format("2006-01-02"),
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.VerificationServer.GetAdminURL("api/issue"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-api-key", config.VerificationServer.AdminKey)

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Requesting new VC-Request: %+v", req)
	} else {
		logger.Debugf("Requesting new VC")
	}

	response, err := config.Client.Do(req)
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

	if response.StatusCode != 200 && response.StatusCode != 400 {
		return "", fmt.Errorf("HTTP %v: %v", response.StatusCode, string(body))
	}

	var r efgsapi.IssueCodeResponse
	if err = json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Response: %+v", r)
	}

	if r.ErrorCode != "" || r.Error != "" {
		return "", fmt.Errorf("%v: %+v", r.ErrorCode, r.Error)
	}

	return r.VerificationCode, nil
}

func verifyCode(ctx context.Context, config *publishConfig, code string) (string, error) {
	logger := logging.FromContext(ctx).Named("efgs.verifyCode")

	body, err := json.Marshal(&efgsapi.VerifyRequest{
		VerificationCode: code,
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.VerificationServer.GetDeviceURL("api/verify"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-api-key", config.VerificationServer.DeviceKey)

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Requesting token. Request: %+v", req)
	} else {
		logger.Debugf("Requesting token")
	}

	response, err := config.Client.Do(req)
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

	if response.StatusCode != 200 && response.StatusCode != 400 {
		return "", fmt.Errorf("HTTP %v: %v", response.StatusCode, string(body))
	}

	var r efgsapi.VerifyResponse
	if err = json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Response: %+v", r)
	}

	if r.ErrorCode != "" || r.Error != "" {
		return "", fmt.Errorf("%v: %+v", r.ErrorCode, r.Error)
	}

	return r.VerificationToken, nil
}

func getCertificate(ctx context.Context, config *publishConfig, keys ExpKeyBatch, token string, hmacKey []byte) (string, error) {
	logger := logging.FromContext(ctx).Named("efgs.getCertificate")

	hmac, err := efgsutils.CalculateExposureKeysHMAC(keys, hmacKey)
	if err != nil {
		logger.Debugf("Error: %v", err)
		return "", err
	}

	body, err := json.Marshal(&efgsapi.CertificateRequest{
		VerificationToken: token,
		ExposureKeyHMAC:   base64.StdEncoding.EncodeToString(hmac),
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", config.VerificationServer.GetDeviceURL("api/certificate"), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-api-key", config.VerificationServer.DeviceKey)

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Getting certificate. Request: %+v", req)
	} else {
		logger.Debugf("Getting certificate")
	}

	response, err := config.Client.Do(req)
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

	if response.StatusCode != 200 && response.StatusCode != 400 {
		return "", fmt.Errorf("HTTP %v: %v", response.StatusCode, string(body))
	}

	var r efgsapi.CertificateResponse
	if err = json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Response: %+v", r)
	}

	if r.ErrorCode != "" || r.Error != "" {
		return "", fmt.Errorf("%v: %+v", r.ErrorCode, r.Error)
	}

	return r.Certificate, nil
}

func publishKeys(ctx context.Context, config *publishConfig, haid string, keys ExpKeyBatch, certificate string, secret []byte) (*keyserverapi.PublishResponse, error) {
	logger := logging.FromContext(ctx).Named("efgs.publishKeys")

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

	req, err := http.NewRequest("POST", config.KeyServer.GetURL("v1/publish"), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Publishing %v keys with HAID %v. Request: %+v", keysCount, haid, req)
	} else {
		logger.Debugf("Publishing %v keys with HAID %v", keysCount, haid)
	}

	response, err := config.Client.Do(req)
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

	if response.StatusCode != 200 && response.StatusCode != 400 {
		return nil, fmt.Errorf("HTTP %v: %v", response.StatusCode, string(body))
	}

	var r keyserverapi.PublishResponse
	if err = json.Unmarshal(body, &r); err != nil {
		return nil, err
	}
	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Response: %+v", r)
	}

	if r.Code != "" || r.ErrorMessage != "" {
		return nil, fmt.Errorf("%v: %+v", r.Code, r.ErrorMessage)
	}

	if r.InsertedExposures != keysCount {
		logger.Infof("Not all exposures were inserted: %v sent, %v inserted", keysCount, r.InsertedExposures)
	}

	return &r, nil
}
