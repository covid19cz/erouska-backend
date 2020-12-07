package metricsapi

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	v1 "github.com/covid19cz/erouska-backend/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"time"
)

//DownloadMetrics Serves most current version of metrics.
func DownloadMetrics(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("DownloadMetrics")

	client := store.Client{}

	var date = time.Now()

	data, err := downloadMetrics(ctx, client, date)
	if err != nil {
		logger.Error(err)
		httputils.SendErrorResponse(w, r, fmt.Errorf("Error while fetching data"))
		return
	}

	httputils.SendResponse(w, r, data)
}

func downloadMetrics(ctx context.Context, client store.Client, date time.Time) (*v1.DownloadMetricsResponse, error) {
	logger := logging.FromContext(ctx).Named("downloadMetrics")

	data, err := fetchMetrics(ctx, client, date)
	if err != nil {
		return nil, fmt.Errorf("Error while fetching data: %v", err)
	}

	// fallback if data are not yet ready
	if data == nil {
		logger.Debugf("Data for %v are not ready yet, fallback to yesterday", date)

		var date = time.Now().Add(time.Hour * -24)

		fallbackData, err := fetchMetrics(ctx, client, date)
		if err != nil {
			return nil, fmt.Errorf("Error while fetching data: %v", err)
		}

		data = fallbackData
	}

	return data, nil
}

func fetchMetrics(ctx context.Context, client store.Storer, date time.Time) (*v1.DownloadMetricsResponse, error) {
	logger := logging.FromContext(ctx).Named("fetchMetrics")

	logger.Debugf("Getting metrics data for %v", date.Format("02.01.2006"))

	rec, err := client.Doc(constants.CollectionMetrics, date.Format("20060102")).Get(ctx)
	if status.Code(err) == codes.NotFound {
		logger.Debugf("Data for %v not found", date.Format("02.01.2006"))
		return nil, nil
	}
	if err != nil {
		logger.Debugf("Error while fetching data: %v", err)
		return nil, err
	}

	var data structs.MetricsData
	if err := rec.DataTo(&data); err != nil {
		logger.Debugf("Error while parsing data: %v", err)
		return nil, err
	}

	return &data, nil
}
