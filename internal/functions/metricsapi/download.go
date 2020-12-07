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
	date := time.Now()

	var req v1.DownloadMetricsRequest

	if !httputils.DecodeJSONOrReportError(w, r, &req) {
		return
	}

	if req.Date != "" {
		parse, err := time.Parse("2006-01-02", req.Date)
		if err == nil {
			date = parse
		} else {
			logger.Debugf("Could not parse requested date, fallback to today: %v", err)
		}
	}

	data, err := downloadMetrics(ctx, client, date)
	if err != nil {
		logger.Error(err)
		httputils.SendErrorResponse(w, r, fmt.Errorf("Error while fetching data"))
		return
	}

	if data == nil {
		http.Error(w, "Data not found", 404)
		return
	}

	httputils.SendResponse(w, r, data)
}

func downloadMetrics(ctx context.Context, client store.Client, date time.Time) (*v1.DownloadMetricsResponse, error) {
	logger := logging.FromContext(ctx).Named("fetchMetrics")

	logger.Infof("Getting metrics data for %v", date.Format("02.01.2006"))

	rec, err := client.Doc(constants.CollectionMetrics, date.Format("20060102")).Get(ctx)
	if status.Code(err) == codes.NotFound {
		logger.Warnf("Data for %v not found", date.Format("02.01.2006"))
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
