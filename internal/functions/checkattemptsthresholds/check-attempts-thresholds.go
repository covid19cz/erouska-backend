package checkattemptsthresholds

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type queryRequest struct {
	Ehrid string `json:"ehrid" validate:"required"`
	Date  int    `json:"date"`
}

type queryResponse struct {
	ThresholdOk bool `json:"thresholdsOK"`
}

//CheckAttemptsThresholds Check if attempts are over threshold
func CheckAttemptsThresholds(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request queryRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	logger.Debugf("Handling CheckAttemptsThresholds request: %+v", request)

	var date string
	if request.Date == 0 {
		date = utils.GetTimeNow().Format("20060102")
	} else {
		date = strconv.Itoa(request.Date)
	}

	ehridNotifCount, err := getNotifsCount(ctx, client, constants.CollectionDailyNotificationAttemptsEhrid, request.Ehrid, date)
	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	var isOk = ehridNotifCount < 1

	response := queryResponse{ThresholdOk: isOk}

	httputils.SendResponse(w, r, response)
}

func getNotifsCount(ctx context.Context, client store.Client, collection string, key string, date string) (int, error) {
	rec, err := client.Doc(collection, key).Get(ctx)

	var notifsCount int
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return 0, fmt.Errorf("Error while querying Firestore: %v", err)
		}

		notifsCount = 0
	} else {
		var states map[string]int
		err = rec.DataTo(&states)
		if err != nil {
			return 0, fmt.Errorf("Error while querying Firestore: %v", err)
		}

		notifsCount = states[date]
	}
	return notifsCount, nil
}
