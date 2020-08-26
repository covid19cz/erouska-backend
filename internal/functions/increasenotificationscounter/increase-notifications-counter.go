package increasenotificationscounter

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
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

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
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

		err := client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
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
			logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
			httputils.SendErrorResponse(w, r, err)
			return
		}
	}

	httputils.SendEmptyResponse(w, r)
}
