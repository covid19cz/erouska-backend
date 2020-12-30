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
	"strings"
	"time"
)

var startOfData = time.Date(2020, 10, 23, 0, 0, 0, 0, time.UTC)

//DownloadMetrics Serves most current version of metrics.
func DownloadMetrics(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()

	client := store.Client{}
	date := time.Now()

	downloadMetrics(ctx, w, r, client, date)
}

func downloadMetrics(ctx context.Context, w http.ResponseWriter, r *http.Request, client store.Client, date time.Time) {
	logger := logging.FromContext(ctx).Named("metricsapi.downloadMetrics")

	var req v1.DownloadMetricsRequest

	providedDate := ""

	if err := httputils.DecodeJSONBody(w, r, &req); err != nil {
		date := r.URL.Query()["date"]
		if len(date) > 0 && date[0] != "" {
			providedDate = date[0]
		}
	} else {
		providedDate = req.Date
	}

	if strings.ToLower(providedDate) == "all" {
		downloadAll(ctx, w, r, client, date)
		return
	}

	if providedDate != "" {
		parsed, err := time.Parse("2006-01-02", providedDate)
		if err == nil {
			date = parsed
		} else {
			logger.Debugf("Could not parse requested date '%v': %v", providedDate, err)
		}
	}

	// The fallback is allowed only when the date is not explicit
	fallbackToYesterday := providedDate == ""

	downloadSingle(ctx, w, r, client, date, fallbackToYesterday)
}

func downloadSingle(ctx context.Context, w http.ResponseWriter, r *http.Request, client store.Client, date time.Time, fallbackToYesterday bool) {
	logger := logging.FromContext(ctx).Named("metricsapi.downloadSingle")

	data, err := loadData(ctx, client, date)
	if err != nil {
		logger.Error(err)
		httputils.SendErrorResponse(w, r, fmt.Errorf("Error while fetching data"))
		return
	}

	if data == nil {
		if fallbackToYesterday {
			logger.Infof("Data for %v not found, fallback to yesterday", date.Format("02.01.2006"))
			downloadSingle(ctx, w, r, client, date.Add(-24*time.Hour), false)
			return
		}

		http.Error(w, "Data not found", 404)
		return
	}

	httputils.SendResponse(w, r, data)
}

func downloadAll(ctx context.Context, w http.ResponseWriter, r *http.Request, client store.Client, today time.Time) {
	logger := logging.FromContext(ctx).Named("metricsapi.downloadAll")

	var allData []structs.MetricsData

	date := today
	for days := 0; !date.Before(startOfData); days++ {
		data, err := loadData(ctx, client, date)
		if err != nil {
			logger.Error(err)
			httputils.SendErrorResponse(w, r, fmt.Errorf("Error while fetching data"))
			return
		}

		date = date.Add(-24 * time.Hour)

		if data == nil {
			continue
		}

		allData = append(allData, *data)
	}

	httputils.SendResponse(w, r, allData)
}

func loadData(ctx context.Context, client store.Client, date time.Time) (*structs.MetricsData, error) {
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
