package increasenotificationscounter

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
	"strconv"
)

type request struct {
	ThresholdsOK bool `json:"thresholdsOK" validate:"required"`
	Date         int  `json:"date"`
}

//IncreaseNotificationsCounter Handler
func IncreaseNotificationsCounter(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request request

	err := httputils.DecodeJSONBody(w, r, &request)
	if err != nil {
		var mr *errors.MalformedRequestError
		if ers.As(err, &mr) {
			logger.Debugf("Cannot handle IncreaseNotificationsCounter request: %+v", mr.Msg)
			http.Error(w, mr.Msg, mr.Status)
		} else {
			logger.Debugf("Cannot handle IncreaseNotificationsCounter request due to unknown error: %+v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Debugf("Handling IncreaseNotificationsCounter request: %+v", request)

	if request.ThresholdsOK {
		var date string
		if request.Date == 0 {
			date = utils.GetTimeNow().Format("20060102")
		} else {
			date = strconv.Itoa(request.Date)
		}

		doc := client.Doc(constants.CollectionNotificationCounters, date)

		err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			rec, err := tx.Get(doc)

			if err != nil {
				if status.Code(err) != codes.NotFound {
					return fmt.Errorf("Error while querying Firestore: %v", err)
				}
				// not found:

				return tx.Set(doc, structs.NotificationCounter{NotificationsCount: 1})
			}

			var data structs.NotificationCounter
			err = rec.DataTo(&data)
			if err != nil {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			logger.Debugf("Found data: %+v", data)

			data.NotificationsCount++

			logger.Debugf("Saving updated daily state: %+v", data)

			return tx.Set(doc, data)
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	httputils.SendEmptyResponse(w, r)
}
