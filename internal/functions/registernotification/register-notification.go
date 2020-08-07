package registernotification

import (
	"cloud.google.com/go/firestore"
	"context"
	ers "errors"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

type request struct {
	Ehrid string `json:"ehrid" validate:"required"`
	IP    string `json:"ip" validate:"required"`
}

//RegisterNotification Handler
func RegisterNotification(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request request

	err := httputils.DecodeJSONBody(w, r, &request)
	if err != nil {
		var mr *errors.MalformedRequestError
		if ers.As(err, &mr) {
			logger.Debugf("Cannot handle RegisterNotification request: %+v", mr.Msg)
			http.Error(w, mr.Msg, mr.Status)
		} else {
			logger.Debugf("Cannot handle RegisterNotification request due to unknown error: %+v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Debugf("Handling RegisterNotification request: %+v", request)

	doc := client.Doc(constants.CollectionRegistrations, request.Ehrid)

	err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return fmt.Errorf("Could not find registration for %v: %v", request.Ehrid, err)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	httputils.SendEmptyResponse(w, r)
}
