package increaseehridattemptscount

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
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
	Ehrid string `json:"ehrid" validate:"required"`
	Date  int    `json:"date"`
}

//IncreaseEhridAttemptsCount Increases the attemptCount attribute in the dailyNotificationAttemptsEhrid
func IncreaseEhridAttemptsCount(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request request

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	logger.Debugf("Handling IncreaseEhridAttemptsCount request: %+v", request)

	var date string
	if request.Date == 0 {
		date = utils.GetTimeNow().Format("20060102")
	} else {
		date = strconv.Itoa(request.Date)
	}

	doc := client.Doc(constants.CollectionDailyNotificationAttemptsEhrid, request.Ehrid)

	err := client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return tx.Set(doc, map[string]int{date: 1})
		}
		// record for eHrid found, let's update it

		var states map[string]int
		err = rec.DataTo(&states)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}

		logger.Debugf("Found daily states: %+v", states)

		dailyCount, exists := states[date]
		if exists {
			states[date] = dailyCount + 1
		} else {
			states[date] = 1
		}

		logger.Debugf("Saving updated daily states: %+v", states)

		return tx.Set(doc, states)
	})

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendEmptyResponse(w, r)
}
