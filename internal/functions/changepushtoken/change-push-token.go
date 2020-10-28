package changepushtoken

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"net/http"
	"regexp"

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
	authClient := auth.Client{}

	var request v1.ChangePushTokenRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	uid, err := authClient.AuthenticateToken(ctx, request.IDToken)
	if err != nil {
		logger.Debugf("Unverifiable token provided: %+v %+v", request.IDToken, err.Error())
		httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
		return
	}

	logger.Debugf("Handling ChangePushToken request: %v %+v", uid, request)

	isEhrid, _ := regexp.MatchString(utils.EhridRegex, uid)

	if !isEhrid {
		logger.Infof("Provided ID is not eHrid: %v", uid)
		err = handleForFUID(ctx, uid, request)
	} else {
		err = handleForEhrid(ctx, uid, request)
	}

	if err != nil {
		logger.Errorf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendEmptyResponse(w, r)
}

func handleForEhrid(ctx context.Context, ehrid string, request v1.ChangePushTokenRequest) error {
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}

	doc := storeClient.Doc(constants.CollectionRegistrations, ehrid)

	logger.Debugf("Trying to find registration for %v in %v", ehrid, constants.CollectionRegistrations)

	return storeClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
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
}

func handleForFUID(ctx context.Context, fuid string, request v1.ChangePushTokenRequest) error {
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}

	doc := storeClient.Doc(constants.CollectionRegistrationsV1, fuid)

	logger.Debugf("Trying to find registration for %v in %v", fuid, constants.CollectionRegistrationsV1)

	return storeClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return fmt.Errorf("Could not find registration for %v: %v", fuid, err)
		}

		var registration structs.RegistrationV1
		err = rec.DataTo(&registration)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}
		logger.Debugf("Found registration: %+v", registration)

		registration.PushRegistrationToken = request.PushRegistrationToken

		logger.Debugf("Saving updated push token: %+v", registration)

		return tx.Set(doc, registration)
	})
}
