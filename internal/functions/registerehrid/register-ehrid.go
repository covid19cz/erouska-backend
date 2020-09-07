package registerehrid

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"net/http"

	"github.com/avast/retry-go"
	"github.com/covid19cz/erouska-backend/internal/auth"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const needsRetry = "needs_retry"

//RegisterEhrid Register new user.
func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}
	authClient := auth.Client{}

	var request v1.RegisterEhridRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	logger.Debugf("Handling registration request: %+v", request)

	registration := structs.Registration{
		Platform:              request.Platform,
		PlatformVersion:       request.PlatformVersion,
		Manufacturer:          request.Manufacturer,
		Model:                 request.Model,
		Locale:                request.Locale,
		PushRegistrationToken: request.PushRegistrationToken,
		CreatedAt:             utils.GetTimeNow().Unix(),
	}

	ehrid, err := register(ctx, storeClient, utils.GenerateEHrid, registration)
	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	customToken, err := authClient.CustomToken(ctx, ehrid)
	if err != nil {
		logger.Fatalf("error minting custom token: %v\n", err.Error())
		httputils.SendErrorResponse(w, r, &errors.UnknownError{Msg: "ahoj"})
	}

	logger.Debugf("Got custom token: %v\n", customToken)

	response := v1.RegisterEhridResponse{CustomToken: customToken}

	httputils.SendResponse(w, r, response)
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
