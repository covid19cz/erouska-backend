package register_ehrid

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	ers "errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

const NeedsRetry = "needs_retry"

type RegistrationRequest struct {
	Platform        string `json:"platform" validate:"required,oneof=android ios"`
	PlatformVersion string `json:"platformVersion" validate:"required"`
	Manufacturer    string `json:"manufacturer" validate:"required"`
	Model           string `json:"model" validate:"required"`
	Locale          string `json:"locale" validate:"required"`
}

type RegistrationResponse struct {
	Ehrid string `json:"ehrid"`
}

func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)

	var request RegistrationRequest

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

	ehrid, err := register(ctx, structs.Registration(request))
	if err != nil {
		response := fmt.Sprintf("Error: %v", err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}

	response := RegistrationResponse{ehrid}

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

func register(ctx context.Context, registration structs.Registration) (string, error) {
	logger := logging.FromContext(ctx)

	var ehrid string

	err := retry.Do(
		func() error {
			ehrid = utils.GenerateEHrid()
			var doc = firebase.FirestoreClient.Collection(constants.CollectionRegistrations).Doc(ehrid)

			logger.Debugf("Trying eHrid: %v", ehrid)

			return firebase.FirestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
				_, err := tx.Get(doc)

				if err == nil {
					// doc found, need retry
					return &errors.CustomError{Msg: NeedsRetry}
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
			return err.Error() == NeedsRetry
		}),
	)

	return ehrid, err
}
