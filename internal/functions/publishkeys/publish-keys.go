package publishkeys

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/functions/efgs"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsdatabase "github.com/covid19cz/erouska-backend/internal/functions/efgs/database"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

const countryOfOrigin = "CZ"

var defaultVisitedCountries = []string{"AT", "DE", "DK", "ES", "IE", "NL", "PL"} // this could be a constant but we're in fckn Go

//PublishKeys Handler
func PublishKeys(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("PublishKeys")

	var request v1.PublishKeysRequestDevice

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		logger.Errorf("Could not deserialize request from device: %v", err)
		http.Error(w, "Could not deserialize", http.StatusBadRequest)
		return
	}

	logger.Debugf("Handling PublishKeys request: %+v", request)

	var serverRequest = toServerRequest(&request)

	serverResponse, err := passToKeyServer(ctx, serverRequest)
	if err != nil {
		logger.Errorf("Could not obtain response from Key server: %v", err)
		return
	}

	logger.Debugf("Received response from Key server: %+v", serverResponse)

	if serverResponse.Code == "" && serverResponse.ErrorMessage == "" {
		logger.Infof("Successfully uploaded %v keys to Key server (%v keys sent)", serverResponse.InsertedExposures, len(serverRequest.Keys))

		if request.ConsentToFederation {
			if err = handleKeysUpload(request); err != nil {
				logger.Errorf("Error while processing keys persistence: %v", err)
			} else {
				logger.Info("Saved uploaded keys to efgs database")
			}
		} else {
			logger.Info("Federation is disabled for this request")
		}
	} else {
		// error has occurred!
		logger.Errorf("Key server has refused the keys; code %v, message '%v'", serverResponse.Code, serverResponse.ErrorMessage)
	}

	sendResponseToClient(logger, w, toDeviceResponse(serverResponse))
}

func handleKeysUpload(request v1.PublishKeysRequestDevice) error {
	visitedCountries := request.VisitedCountries
	if len(visitedCountries) == 0 {
		visitedCountries = defaultVisitedCountries
	}

	var keys []*efgsapi.DiagnosisKey
	for _, k := range request.Keys {
		keys = append(keys, efgs.ToDiagnosisKey(&k, countryOfOrigin, visitedCountries, request.SymptomOnsetInterval))
	}

	return efgsdatabase.Database.PersistDiagnosisKeys(keys)
}

func passToKeyServer(ctx context.Context, request *v1.PublishKeysRequestServer) (*v1.PublishKeysResponseServer, error) {
	logger := logging.FromContext(ctx).Named("passToKeyServer")

	blob, err := json.Marshal(request)
	if err != nil {
		logger.Debugf("Could not serialize request for Key server: %v", err)
		return nil, err
	}

	keyServerConfig, err := utils.LoadKeyServerConfig(ctx)
	if err != nil {
		logger.Fatalf("Could not load key server config: %v", err)
		return nil, err
	}

	response, err := http.Post(keyServerConfig.GetURL("v1/publish"), "application/json", bytes.NewBuffer(blob))
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

func sendResponseToClient(logger *zap.SugaredLogger, w http.ResponseWriter, response *v1.PublishKeysResponseDevice) {
	blob, err := json.Marshal(response)
	if err != nil {
		logger.Warnf("Could not serialize response for device: %v", err)
		return
	}

	logger.Debugf("Sending response to client: %+v", response)

	_, err = w.Write(blob)
	if err != nil {
		logger.Warnf("Could not send response to device: %v", err)
		return
	}
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
