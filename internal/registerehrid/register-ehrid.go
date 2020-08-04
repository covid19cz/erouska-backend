package registerehrid

import (
	"context"
	"encoding/json"
	ers "errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/avast/retry-go"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const needsRetry = "needs_retry"

type registrationRequest struct {
	Platform        string `json:"platform" validate:"required,oneof=android ios"`
	PlatformVersion string `json:"platformVersion" validate:"required"`
	Manufacturer    string `json:"manufacturer" validate:"required"`
	Model           string `json:"model" validate:"required"`
	Locale          string `json:"locale" validate:"required"`
}

type registrationResponse struct {
	Ehrid string `json:"ehrid"`
}

//RegisterEhrid Register new user.
func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request registrationRequest

	err := httputils.DecodeJSONBody(w, r, &request)
	if err != nil {
		var mr *errors.MalformedRequestError
		if ers.As(err, &mr) {
			logger.Debugf("Cannot handle registration request: %+v", mr.Msg)
			http.Error(w, mr.Msg, mr.Status)
		} else {
			logger.Debugf("Cannot handle registration request due to unknown error: %+v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Debugf("Handling registration request: %+v", request)

	registration := structs.Registration{
		Platform:        request.Platform,
		PlatformVersion: request.PlatformVersion,
		Manufacturer:    request.Manufacturer,
		Model:           request.Model,
		Locale:          request.Locale,
		CreatedAt:       utils.GetTimeNow().Unix(),
	}

	ehrid, err := register(ctx, client, utils.GenerateEHrid, registration)
	if err != nil {
		response := fmt.Sprintf("Error: %v", err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}

	response := registrationResponse{ehrid}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		response := fmt.Sprintf("Error: %v", err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}
}

func register(ctx context.Context, store store.Storer, generateEhrid func() string, registration structs.Registration) (string, error) {
	logger := logging.FromContext(ctx)

	var ehrid string

	err := retry.Do(
		func() error {
			ehrid = generateEhrid()
			var doc = store.Doc(constants.CollectionRegistrations, ehrid)

			logger.Debugf("Trying eHrid: %v", ehrid)

			return store.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
				_, err := tx.Get(doc)

				if err == nil {
					// doc found, need retry
					return &errors.CustomError{Msg: needsRetry}
				}

				if status.Code(err) != codes.NotFound {
					return fmt.Errorf("Error while querying Firestore: %v", err)
				}
				// not found, great!

				logger.Infof("Generated new eHrid %v, saving registration %+v", ehrid, registration)

				return tx.Set(doc, registration)
			})
		},
		retry.RetryIf(func(err error) bool {
			return err.Error() == needsRetry
		}),
	)

	return ehrid, err
}
