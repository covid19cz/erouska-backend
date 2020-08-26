package registernotification

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
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

type request struct {
	Ehrid string `json:"ehrid" validate:"required"`
}

//RegisterNotification Handler
func RegisterNotification(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request request

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	logger.Debugf("Handling RegisterNotification request: %+v", request)

	doc := client.Doc(constants.CollectionRegistrations, request.Ehrid)

	err := client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return &errors.NotFoundError{Msg: fmt.Sprintf("Could not find registration for %v: %v", request.Ehrid, err)}
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

	httputils.SendEmptyResponse(w, r)
}
