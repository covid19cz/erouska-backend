package publishkeys

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"firebase.google.com/go/db"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/functions/efgs"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsdatabase "github.com/covid19cz/erouska-backend/internal/functions/efgs/database"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/realtimedb"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	countryOfOrigin              = "CZ"
	defaultTransmissionRiskLevel = 2 // see docs for ExposureKey - "CONFIRMED will lead to TR 2"
	defaultDSOS                  = 3 // "days since onset of symptoms" - this is a good default/fallback value because it's taken as serious by the EN API
)

type config struct {
	keyServerConfig         *utils.KeyServerConfig
	client                  *http.Client
	realtimeDBClient        *realtimedb.Client
	efgsdatabase            *efgsdatabase.Connection
	defaultVisitedCountries []string
}

//PublishKeys Handler
func PublishKeys(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("publish-keys.PublishKeys")

	var request v1.PublishKeysRequestDevice

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic("fuck")
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Request base64: %+v", base64.StdEncoding.EncodeToString(body))
	}

	if err := json.Unmarshal(body, &request); err != nil {
		logger.Errorf("Could not deserialize request from device: %v", err)
		http.Error(w, "Could not deserialize", http.StatusBadRequest)
		return
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Handling PublishKeys request: %+v", request)
	}

	config, err := loadConfig(ctx)
	if err != nil {
		logger.Errorf("Could not load config: %v", err)
		http.Error(w, "Could not load config", http.StatusInternalServerError)
		return
	}

	publishKeys(ctx, config, w, request)
}

func publishKeys(ctx context.Context, config *config, w http.ResponseWriter, request v1.PublishKeysRequestDevice) {
	logger := logging.FromContext(ctx).Named("publish-keys.publishKeys")

	var serverRequest = toServerRequest(&request)

	serverResponse, err := passToKeyServer(ctx, config, serverRequest)
	if err != nil {
		logger.Errorf("Could not obtain response from Key server: %v", err)
		return
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Received response from Key server: %+v", serverResponse)
	}

	// send response to client ASAP
	sendResponseToClient(ctx, w, toDeviceResponse(serverResponse))

	if serverResponse.Code == "" && serverResponse.ErrorMessage == "" {
		logger.Infof("Successfully uploaded %v keys to Key server (%v keys sent)", serverResponse.InsertedExposures, len(serverRequest.Keys))

		if err := updateCounters(ctx, config.realtimeDBClient, serverResponse.InsertedExposures+1); err != nil {
			logger.Errorf("Could not update publishers counter: %+v", err)
			// don't fail, this is not so important
		}

		if request.ConsentToFederation {
			logger.Debug("Going to save uploaded keys to EFGS database")

			if err = persistKeysForEfgs(ctx, config, request); err != nil {
				logger.Errorf("Error while processing keys persistence: %v", err)
				// don't fail, this is not so important
			} else {
				logger.Info("Saved uploaded keys to EFGS database")
			}
		} else {
			logger.Info("Federation is disabled for this request")
		}
	} else {
		// error has occurred! don't fail, just pass the error to client
		logger.Errorf("Key server has refused the keys; code %v, message '%v'", serverResponse.Code, serverResponse.ErrorMessage)
	}
}

func persistKeysForEfgs(ctx context.Context, config *config, request v1.PublishKeysRequestDevice) error {
	logger := logging.FromContext(ctx).Named("publish-keys.persistKeysForEfgs")

	logger.Debugf("Handling keys upload")

	visitedCountries := request.VisitedCountries
	if len(visitedCountries) == 0 {
		visitedCountries = config.defaultVisitedCountries
	}

	// Days since onset of symptoms
	// Try to read it from VC and if not present, use the default value.
	dos := extractDSOS(request)

	if dos <= 0 { // one would use MAX function if Go has some...
		dos = defaultDSOS
	}

	logger.Debugf("Extracted DSOS %v", dos)

	var keys []*efgsapi.DiagnosisKey
	for _, k := range request.Keys {
		diagnosisKey := efgs.ToDiagnosisKey(&k, countryOfOrigin, visitedCountries, dos)
		if diagnosisKey.TransmissionRiskLevel == 0 {
			diagnosisKey.TransmissionRiskLevel = defaultTransmissionRiskLevel
		}
		keys = append(keys, diagnosisKey)
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Saving keys into DB: %+v", keys)
	}

	return config.efgsdatabase.PersistDiagnosisKeys(keys)
}

func passToKeyServer(ctx context.Context, config *config, request *v1.PublishKeysRequestServer) (*v1.PublishKeysResponseServer, error) {
	logger := logging.FromContext(ctx).Named("publish-keys.passToKeyServer")

	blob, err := json.Marshal(request)
	if err != nil {
		logger.Debugf("Could not serialize request for Key server: %v", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", config.keyServerConfig.GetURL("v1/publish"), bytes.NewBuffer(blob))
	if err != nil {
		logger.Debugf("Could not create request for Key server: %v", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := config.client.Do(req)
	if err != nil {
		logger.Debugf("Could not obtain response from Key server: %v", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err := response.Body.Close(); err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %v: %v", response.StatusCode, string(body))
	}

	var serverResponse v1.PublishKeysResponseServer

	if err = json.Unmarshal(body, &serverResponse); err != nil {
		logger.Debugf("Could not deserialize response from Key server: %v", err)
		return nil, err
	}

	return &serverResponse, nil
}

func loadConfig(ctx context.Context) (*config, error) {
	secretsClient := secrets.Client{}

	visitedCountries, err := secretsClient.Get("efgs-default-visited-countries")
	if err != nil {
		return nil, err
	}

	keyServerConfig, err := utils.LoadKeyServerConfig(ctx)
	if err != nil {
		return nil, err
	}

	config := config{
		keyServerConfig:  keyServerConfig,
		client:           &http.Client{},
		realtimeDBClient: &realtimedb.Client{},
		efgsdatabase:     &efgsdatabase.Database,
	}

	if err = json.Unmarshal(visitedCountries, &config.defaultVisitedCountries); err != nil {
		return nil, err
	}

	return &config, nil
}

func sendResponseToClient(ctx context.Context, w http.ResponseWriter, response *v1.PublishKeysResponseDevice) {
	logger := logging.FromContext(ctx).Named("publish-keys.sendResponseToClient")

	blob, err := json.Marshal(response)
	if err != nil {
		logger.Warnf("Could not serialize response for device: %v", err)
		return
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Sending response to client: %+v", response)
	}

	_, err = w.Write(blob)
	if err != nil {
		logger.Warnf("Could not send response to device: %v", err)
		return
	}
}

func extractDSOS(request v1.PublishKeysRequestDevice) int {
	// We parse the token but we don't care about signature validation.
	token, _ := jwt.Parse(request.VerificationPayload, func(token *jwt.Token) (interface{}, error) {
		return []byte("hello-world"), nil
	})

	// Here we certainly got validation error but we don't care, the validation was already done by Key server.
	// If we got the token too, it's just enough.

	if token == nil {
		return -1
	}

	// Extract DSOS.
	if token.Claims == nil {
		return -1
	}

	claims := token.Claims.(jwt.MapClaims)
	value, ok := claims["symptomOnsetInterval"]
	if !ok {
		return -1
	}

	soi := int64(value.(float64))
	return int((time.Now().Unix() - soi*600) / 86400)
}

func updateCounters(ctx context.Context, client *realtimedb.Client, keysCount int) error {
	logger := logging.FromContext(ctx).Named("publish-keys.updateCounters")

	var date = utils.GetTimeNow().Format("20060102")

	// update daily counter
	if err := updateCounter(ctx, client, constants.DbPublisherCountersPrefix+date, keysCount); err != nil {
		logger.Warnf("Cannot increase publishers counter due to unknown error: %+v", err.Error())
		return err
	}

	// update total counter
	if err := updateCounter(ctx, client, constants.DbPublisherCountersPrefix+"total", keysCount); err != nil {
		logger.Warnf("Cannot increase publishers counter due to unknown error: %+v", err.Error())
		return err
	}

	return nil
}

func updateCounter(ctx context.Context, client *realtimedb.Client, dbKey string, keysCount int) error {
	logger := logging.FromContext(ctx).Named("publish-keys.updateCounter")

	return client.RunTransaction(ctx, dbKey, func(tn db.TransactionNode) (interface{}, error) {
		var state structs.PublisherCounter

		if err := tn.Unmarshal(&state); err != nil {
			return nil, err
		}

		logger.Debugf("Found counter state, dbKey %v: %+v", dbKey, state)

		state.PublishersCount++
		state.KeysCount += keysCount

		logger.Debugf("Saving updated counter state, dbKey %v: %+v", dbKey, state)

		return state, nil
	})
}

func toServerRequest(request *v1.PublishKeysRequestDevice) *v1.PublishKeysRequestServer {
	return &v1.PublishKeysRequestServer{
		Keys:                 request.Keys,
		HealthAuthorityID:    request.HealthAuthorityID,
		VerificationPayload:  request.VerificationPayload,
		HMACKey:              request.HMACKey,
		SymptomOnsetInterval: request.SymptomOnsetInterval,
		Traveler:             request.Traveler,
		RevisionToken:        request.RevisionToken,
		Padding:              request.Padding,
	}
}

func toDeviceResponse(response *v1.PublishKeysResponseServer) *v1.PublishKeysResponseDevice {
	return &v1.PublishKeysResponseDevice{
		RevisionToken:     response.RevisionToken,
		InsertedExposures: response.InsertedExposures,
		ErrorMessage:      response.ErrorMessage,
		Code:              response.Code,
		Padding:           response.Padding,
	}
}
