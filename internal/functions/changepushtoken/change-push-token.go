package changepushtoken

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"google.golang.org/api/iterator"
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

	var storeClient store.Storer = store.Client{}

	if isEhrid {
		err = handleForEhrid(ctx, storeClient, uid, request.PushRegistrationToken)
	} else {
		logger.Infof("Provided ID is not eHrid: %v", uid)
		err = handleForFUID(ctx, storeClient, uid, request.PushRegistrationToken)
	}

	if err != nil {
		logger.Errorf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendEmptyResponse(w, r)
}

func handleForEhrid(ctx context.Context, storeClient store.Storer, ehrid string, pushToken string) error {
	logger := logging.FromContext(ctx).Named("change-push-token.handleForEhrid")

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

		registration.PushRegistrationToken = pushToken

		logger.Debugf("Saving updated push token: %+v", registration)

		return tx.Set(doc, registration)
	})
}

func handleForFUID(ctx context.Context, storeClient store.Storer, fuid string, pushToken string) error {
	logger := logging.FromContext(ctx).Named("change-push-token.handleForFUID")

	logger.Debugf("Looking for FUID %v in collection %v", fuid, constants.CollectionRegistrationsV1)

	doc, err := findDocByFUID(ctx, storeClient, fuid)
	if err != nil {
		return err
	}

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

		registration.PushRegistrationToken = pushToken

		logger.Debugf("Saving updated push token: %+v", registration)

		return tx.Set(doc, registration)
	})
}

func findDocByFUID(ctx context.Context, storeClient store.Storer, fuid string) (*firestore.DocumentRef, error) {
	it := storeClient.Find(constants.CollectionRegistrationsV1, "fuid", fuid).Snapshots(ctx)

	resp, err := it.Next()
	if err == iterator.Done || (resp != nil && resp.Size == 0) {
		return nil, fmt.Errorf("Could not find record for FUID %v: resp=%v %v", fuid, resp, err)
	}

	if err != nil {
		return nil, err
	}

	snap, err := resp.Documents.Next()
	if err == iterator.Done || snap == nil {
		return nil, fmt.Errorf("Could not find record for FUID %v: snap=%v %v", fuid, snap, err)
	}

	if err != nil {
		return nil, err
	}

	return snap.Ref, nil
}
