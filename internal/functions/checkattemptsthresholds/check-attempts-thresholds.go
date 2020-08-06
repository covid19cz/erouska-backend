package checkattemptsthresholds

import (
	"context"
	"encoding/json"
	ers "errors"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
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

type queryRequest struct {
	Ehrid string `json:"ehrid" validate:"required"`
	IP    string `json:"ip" validate:"required"`
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

	err := httputils.DecodeJSONBody(w, r, &request)
	if err != nil {
		var mr *errors.MalformedRequestError
		if ers.As(err, &mr) {
			logger.Debugf("Cannot handle CheckAttemptsThresholds request: %+v", mr.Msg)
			logger.Error(err)
			http.Error(w, mr.Msg, mr.Status)
		} else {
			logger.Debugf("Cannot handle CheckAttemptsThresholds request due to unknown error: %+v", err.Error())
			logger.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
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
		response := fmt.Sprintf("Error: %v", err)
		logger.Error(err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}

	ipNotifCount, err := getNotifsCount(ctx, client, constants.CollectionDailyNotificationAttemptsIP, request.IP, date)
	if err != nil {
		response := fmt.Sprintf("Error: %v", err)
		logger.Error(err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}

	var isOk = ehridNotifCount < 1 && ipNotifCount < 100

	response := queryResponse{ThresholdOk: isOk}

	js, err := json.Marshal(response)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		response := fmt.Sprintf("Error: %v", err)
		logger.Error(err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}
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
