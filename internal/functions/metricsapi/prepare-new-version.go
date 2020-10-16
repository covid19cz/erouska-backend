package metricsapi

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/monitoring"
	"github.com/covid19cz/erouska-backend/internal/realtimedb"
	"github.com/covid19cz/erouska-backend/internal/store"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"net/http"
	"os"
	"time"
)

//MetricsData Data of metrics.
type MetricsData struct {
	Modified               int64  `json:"modified"`
	Date                   string `json:"date"`
	ActivationsYesterday   int32  `json:"activations_yesterday"`
	ActivationsTotal       int32  `json:"activations_total"`
	KeyPublishersYesterday int32  `json:"key_publishers_yesterday"`
	KeyPublishersTotal     int32  `json:"key_publishers_total"`
	NotificationsYesterday int32  `json:"notifications_yesterday"`
	NotificationsTotal     int32  `json:"notifications_total"`
}

type counts struct {
	yesterday int32
	total     int32
}

const publishersOffset = -2708 // number of reported publishers before the metric was changed

//PrepareNewVersion Prepares new version of metrics JSON document.
func PrepareNewVersion(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)

	var now = time.Now()

	projectID, ok := os.LookupEnv("METRICS_PROJECT_ID")
	if !ok {
		logger.Error("Could not find METRICS_PROJECT_ID env")
		httputils.SendErrorResponse(w, r, fmt.Errorf("Function not properly configured"))
		return
	}

	notifications, err := getNotificationCounters(ctx, now)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	activations, err := getActivationCounters(ctx, now)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	publishers, err := getPublishersCount(ctx, projectID)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	var today = time.Now().Format("20060102")

	var data = MetricsData{
		Modified:               now.Unix(),
		Date:                   today,
		ActivationsYesterday:   activations.yesterday,
		ActivationsTotal:       activations.total,
		KeyPublishersYesterday: publishers.yesterday,
		KeyPublishersTotal:     publishers.total,
		NotificationsYesterday: notifications.yesterday,
		NotificationsTotal:     notifications.total,
	}

	firestoreClient := store.Client{}

	_, err = firestoreClient.Doc(constants.CollectionMetrics, today).Set(ctx, &data)
	if err != nil {
		logger.Errorf("Error while saving data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	logger.Infof("Successfully written metrics data to firestore: %+v", data)

	httputils.SendResponse(w, r, struct{ status string }{status: "OK"})
}

func getNotificationCounters(ctx context.Context, now time.Time) (*counts, error) {
	logger := logging.FromContext(ctx)

	var date = now.Add(time.Hour * -24).Format("20060102")

	// yesterday

	yesterdayData, err := getNotificationCounter(ctx, date)
	if err != nil {
		return nil, err
	}

	logger.Infof("Notifications count for yesterday: %v", yesterdayData.NotificationsCount)

	// total

	totalData, err := getNotificationCounter(ctx, "total")
	if err != nil {
		return nil, err
	}

	logger.Infof("Notifications total count: %v", totalData.NotificationsCount)

	return &counts{
		yesterday: int32(yesterdayData.NotificationsCount),
		total:     int32(totalData.NotificationsCount),
	}, nil
}

func getNotificationCounter(ctx context.Context, key string) (*structs.NotificationCounter, error) {
	logger := logging.FromContext(ctx)
	var storeClient = store.Client{}

	logger.Debugf("Getting notification counter with key %v", key)

	doc := storeClient.Doc(constants.CollectionNotificationCounters, key)

	// TODO handle not found

	rec, err := doc.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error while querying Firestore: %v", err)
	}

	var data structs.NotificationCounter
	err = rec.DataTo(&data)
	if err != nil {
		return nil, fmt.Errorf("Error while querying Firestore: %v", err)
	}
	return &data, nil
}

func getActivationCounters(ctx context.Context, now time.Time) (*counts, error) {
	logger := logging.FromContext(ctx)

	var date = now.Add(time.Hour * -24).Format("20060102")

	// yesterday

	yesterdayData, err := getActivationCounter(ctx, date)
	if err != nil {
		return nil, err
	}

	logger.Infof("Activations count for yesterday: %v", yesterdayData.UsersCount)

	// total

	totalData, err := getActivationCounter(ctx, "total")
	if err != nil {
		return nil, err
	}

	logger.Infof("Activations total count: %v", totalData.UsersCount)

	return &counts{
		yesterday: int32(yesterdayData.UsersCount),
		total:     int32(totalData.UsersCount),
	}, nil
}

func getActivationCounter(ctx context.Context, key string) (*structs.UserCounter, error) {
	logger := logging.FromContext(ctx)
	client := realtimedb.Client{}

	logger.Debugf("Getting activation counter with key %v", key)

	var data structs.UserCounter

	// TODO handle not found

	if err := client.NewRef(constants.DbUserCountersPrefix+key).Get(ctx, &data); err != nil {
		logger.Debugf("Error while querying DB: %v", err)
		return nil, err
	}

	return &data, nil
}

func getPublishersCount(ctx context.Context, projectID string) (*counts, error) {
	logger := logging.FromContext(ctx)

	startOfTomorrow := time.Now().UTC().Add(time.Hour * -24).Truncate(time.Hour * 24)
	startOfErouska, err := time.Parse("02.01.2006", "01.09.2020")
	if err != nil {
		panic(err)
	}

	values, err := getPublishersValues(ctx, projectID, startOfTomorrow, 84600 /* 1 day */)
	if err != nil {
		logger.Debugf("Could not fetch data for publishers of last day: %v", err)
		return nil, err
	}

	yesterdayCount := values[0] // get the newest from daily buckets

	logger.Infof("Publishers count for yesterday %v", yesterdayCount)

	values, err = getPublishersValues(ctx, projectID, startOfErouska, 2592000 /* 1 month */)
	if err != nil {
		logger.Debugf("Could not fetch data for publishers of all time: %v", err)
		return nil, err
	}

	totalCount := sum(values) // if there's multiple values, sum them all up - this is all time startOfErouska
	totalCount -= publishersOffset

	logger.Infof("Publishers total count: %v", totalCount)

	return &counts{
		yesterday: yesterdayCount,
		total:     totalCount,
	}, nil
}

func getPublishersValues(ctx context.Context, projectID string, from time.Time, sumWindow int64) ([]int32, error) {
	monitoringClient := monitoring.Client{}

	startOfToday := time.Now().UTC().Truncate(time.Hour * 24)

	// this just adds the configuration... one would use function currying of Go supports such thing
	return monitoringClient.ReadSummarized(ctx,
		projectID,
		`resource.type="cloud_run_revision" metric.type="logging.googleapis.com/user/publish-exposures-inserted-flattened"`,
		from,
		startOfToday,
		sumWindow)
}

func sum(array []int32) int32 {
	result := int32(0)
	for _, v := range array {
		result += v
	}
	return result
}
