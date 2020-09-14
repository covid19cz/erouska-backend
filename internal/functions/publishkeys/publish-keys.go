package publishkeys

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"github.com/sethvargo/go-envconfig"
	"go.uber.org/zap"
	"net/http"
)

//PublishKeys Handler
func PublishKeys(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)

	var request v1.PublishKeysRequestDevice

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&request)
	if err != nil {
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

		err = handleKeysUpload(ctx, request)
		if err != nil {
			logger.Errorf("Error while processing keys upload: %v", err)
		}
	} else {
		// error has occurred!
		logger.Errorf("Key server has refused the keys; code %v, message '%v'", serverResponse.Code, serverResponse.ErrorMessage)
	}

	sendResponseToClient(logger, w, toDeviceResponse(serverResponse))
}

func handleKeysUpload(ctx context.Context, request v1.PublishKeysRequestDevice) error {
	// TODO save to DB

	return nil
}

func passToKeyServer(ctx context.Context, request *v1.PublishKeysRequestServer) (*v1.PublishKeysResponseServer, error) {
	logger := logging.FromContext(ctx)

	blob, err := json.Marshal(request)
	if err != nil {
		logger.Debugf("Could not serialize request for Key server: %v", err)
		return nil, err
	}

	var keyServerConfig constants.KeyServerConfig
	if err := envconfig.Process(ctx, &keyServerConfig); err != nil {
		logger.Fatalf("Could not read KeyServerConfig: %v", err)
		return nil, err
	}

	response, err := http.Post(keyServerConfig.URL, "application/json", bytes.NewBuffer(blob))
	if err != nil {
		logger.Debugf("Could not obtain response from Key server: %v", err)
		return nil, err
	}

	var serverResponse v1.PublishKeysResponseServer

	dec := json.NewDecoder(response.Body)
	err = dec.Decode(&serverResponse)
	if err != nil {
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
