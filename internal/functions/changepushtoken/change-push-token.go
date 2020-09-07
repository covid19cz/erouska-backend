package changepushtoken

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"net/http"

	"github.com/covid19cz/erouska-backend/internal/auth"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//ChangePushToken Handler
func ChangePushToken(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}
	authClient := auth.Client{}

	var request v1.ChangePushTokenRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	ehrid, err := authClient.AuthenticateToken(ctx, request.IDToken)
	if err != nil {
		logger.Debugf("Unverifiable token provided: %+v %+v", request.IDToken, err.Error())
		httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
		return
	}

	logger.Debugf("Handling ChangePushToken request: %v %+v", ehrid, request)

	doc := storeClient.Doc(constants.CollectionRegistrations, ehrid)

	err = storeClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return fmt.Errorf("Could not find registration for %v: %v", ehrid, err)
		}

		var registration structs.Registration
		err = rec.DataTo(&registration)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}
		logger.Debugf("Found registration: %+v", registration)

		registration.PushRegistrationToken = request.PushRegistrationToken

		logger.Debugf("Saving updated push token: %+v", registration)

		return tx.Set(doc, registration)
	})

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendEmptyResponse(w, r)
}
