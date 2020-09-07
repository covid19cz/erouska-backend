package registernotification

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"net/http"

	"github.com/covid19cz/erouska-backend/internal/auth"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//AftermathPayload Struct holding aftermath input data.
type AftermathPayload struct {
	Ehrid string `json:"ehrid" validate:"required"`
}

//RegisterNotification Handler
func RegisterNotification(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}
	authClient := auth.Client{}
	pubSubClient := pubsub.Client{}

	var request v1.RegisterNotificationRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	ehrid, err := authClient.AuthenticateToken(ctx, request.IDToken)
	if err != nil {
		logger.Debugf("Unverifiable token provided: %+v %+v", request.IDToken, err.Error())
		httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
		return
	}

	logger.Debugf("Handling RegisterNotification request: %v %+v", ehrid, request)

	doc := storeClient.Doc(constants.CollectionRegistrations, ehrid)

	err = storeClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return &errors.NotFoundError{Msg: fmt.Sprintf("Could not find registration for %v: %v", ehrid, err)}
		}

		var registration structs.Registration
		err = rec.DataTo(&registration)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}
		logger.Debugf("Found registration: %+v", registration)

		registration.LastNotificationStatus = "sent"
		registration.LastNotificationUpdatedAt = utils.GetTimeNow().Unix()

		logger.Debugf("Saving updated notification state: %+v", registration)

		return tx.Set(doc, registration)
	})

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	aftermathPayload := AftermathPayload{Ehrid: ehrid}

	topicName := constants.TopicRegisterNotification
	logger.Debugf("Publishing event to %v: %+v", topicName, aftermathPayload)
	err = pubSubClient.Publish(topicName, aftermathPayload)
	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendEmptyResponse(w, r)
}
