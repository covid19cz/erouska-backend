package metricsapi

import (
	"context"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"time"
)

//DownloadMetrics Serves most current version of metrics.
func DownloadMetrics(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)

	var date = time.Now()

	data, err := fetchMetrics(ctx, date)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	// fallback if data are not yet ready
	if data == nil {
		logger.Debugf("Data for %v are not ready yet, fallback to yesterday", date)

		var date = time.Now().Add(time.Hour * -24)

		fallbackData, err := fetchMetrics(ctx, date)
		if err != nil {
			logger.Errorf("Error while fetching data: %v", err)
			httputils.SendErrorResponse(w, r, err)
			return
		}

		data = fallbackData
	}

	httputils.SendResponse(w, r, data)
}

func fetchMetrics(ctx context.Context, date time.Time) (*MetricsData, error) {
	logger := logging.FromContext(ctx)

	logger.Debugf("Getting metrics data for %v", date.Format("02.01.2006"))

	client := store.Client{}
	rec, err := client.Doc(constants.CollectionMetrics, date.Format("20060102")).Get(ctx)
	if status.Code(err) == codes.NotFound {
		logger.Debugf("Data for %v not found", date.Format("02.01.2006"))
		return nil, nil
	}
	if err != nil {
		logger.Debugf("Error while fetching data: %v", err)
		return nil, err
	}

	var data MetricsData
	if err := rec.DataTo(&data); err != nil {
		logger.Debugf("Error while parsing data: %v", err)
		return nil, err
	}

	return &data, nil
}
