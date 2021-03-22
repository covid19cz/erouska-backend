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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"os"
	"time"
)

type config struct {
	projectID        string
	now              time.Time
	realtimedbClient realtimedb.RealtimeDB
	firestoreClient  store.Storer
	monitoringClient monitoring.Reader
}

//PrepareNewVersion Prepares new version of metrics JSON document.
func PrepareNewVersion(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("PrepareNewVersion")

	projectID, ok := os.LookupEnv("METRICS_PROJECT_ID")
	if !ok {
		logger.Error("Could not find METRICS_PROJECT_ID env")
		httputils.SendErrorResponse(w, r, fmt.Errorf("Function not properly configured"))
		return
	}

	config := config{
		projectID:        projectID,
		now:              time.Now(),
		realtimedbClient: realtimedb.Client{},
		firestoreClient:  store.Client{},
		monitoringClient: monitoring.Client{},
	}

	if err := prepareNewVersion(ctx, &config); err != nil {
		logger.Errorf("Error while fetching data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendResponse(w, r, struct{ status string }{status: "OK"})
}

func prepareNewVersion(ctx context.Context, config *config) error {
	logger := logging.FromContext(ctx).Named("prepareNewVersion")

	yesterday := config.now.UTC().Add(-24 * time.Hour)
	doc, err := config.firestoreClient.Doc(constants.CollectionMetrics, yesterday.Format("20060102")).Get(ctx)
	if err != nil {
		logger.Debugf("Could not fetch yesterdays data for yestPublishers: %v", err)
		return err
	}
	var yestData structs.MetricsData
	if err = doc.DataTo(&yestData); err != nil {
		logger.Debugf("Could not fetch yesterdays data for yestPublishers: %v", err)
		return err
	}

	logger.Debugf("Loaded yesterdays data: %+v", yestData)

	yestNotifications, err := getNotificationsCount(ctx, config, yesterday.Format("20060102"))
	if err != nil {
		return fmt.Errorf("Error while fetching data: %v", err)
	}

	yestActivations, err := getActivationsCount(ctx, config, yesterday.Format("20060102"))
	if err != nil {
		return fmt.Errorf("Error while fetching data: %v", err)
	}

	yestPublishers, err := getPublishersCount(ctx, config, yesterday.Format("20060102"))
	if err != nil {
		return fmt.Errorf("Error while fetching data: %v", err)
	}

	yestEfgs, err := getEfgsCounter(ctx, config, yesterday.Format("20060102"))
	if err != nil {
		return fmt.Errorf("Error while fetching data: %v", err)
	}

	var today = config.now.Format("20060102")

	data := structs.MetricsData{
		Modified:                    config.now.Unix(),
		Date:                        today,
		ActivationsYesterday:        yestActivations,
		ActivationsTotal:            yestData.ActivationsTotal + yestActivations,
		KeyPublishersYesterday:      yestPublishers,
		KeyPublishersTotal:          yestData.KeyPublishersTotal + yestPublishers,
		NotificationsYesterday:      yestNotifications,
		NotificationsTotal:          yestData.NotificationsTotal + yestNotifications,
		EfgsKeysDownloadedYesterday: int32(yestEfgs.KeysDownloaded),
		EfgsKeysDownloadedTotal:     yestData.EfgsKeysDownloadedTotal + int32(yestEfgs.KeysDownloaded),
		EfgsKeysUploadedYesterday:   int32(yestEfgs.KeysUploaded),
		EfgsKeysUploadedTotal:       yestData.EfgsKeysUploadedTotal + int32(yestEfgs.KeysUploaded),
		EfgsPublishersYesterday:     int32(yestEfgs.Publishers),
		EfgsPublishersTotal:         yestData.EfgsPublishersTotal + int32(yestEfgs.Publishers),
	}

	logger.Debugf("Collected data: %+v", data)

	_, err = config.firestoreClient.Doc(constants.CollectionMetrics, today).Set(ctx, &data)
	if err != nil {
		return fmt.Errorf("Error while saving data: %v", err)
	}

	logger.Infof("Successfully written metrics data to firestoreClient: %+v", data)
	return nil
}

func getNotificationsCount(ctx context.Context, config *config, key string) (int32, error) {
	logger := logging.FromContext(ctx).Named("getNotificationsCount")

	logger.Debugf("Getting notification counter with key %v", key)

	doc := config.firestoreClient.Doc(constants.CollectionNotificationCounters, key)

	var data structs.NotificationCounter

	rec, err := doc.Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return 0, fmt.Errorf("Error while querying Firestore: %v", err)
		}

		logger.Warnf("Notifications counter for '%v' was not found, using default value", key)

		data = structs.NotificationCounter{
			NotificationsCount: 0,
		}
	} else {
		err = rec.DataTo(&data)
		if err != nil {
			return 0, fmt.Errorf("Error while querying Firestore: %v", err)
		}
	}

	return int32(data.NotificationsCount), nil
}

func getActivationsCount(ctx context.Context, config *config, key string) (int32, error) {
	logger := logging.FromContext(ctx).Named("getActivationsCount")

	logger.Debugf("Getting activation counter with key %v", key)

	var data structs.UserCounter

	if err := config.realtimedbClient.NewRef(constants.DbUserCountersPrefix+key).Get(ctx, &data); err != nil {
		logger.Debugf("Error while querying DB: %v", err)
		return 0, err
	}

	return int32(data.UsersCount), nil
}

func getPublishersCount(ctx context.Context, config *config, key string) (int32, error) {
	logger := logging.FromContext(ctx)

	logger.Debugf("Getting publishers counter with key %v", key)

	var data structs.PublisherCounter

	if err := config.realtimedbClient.NewRef(constants.DbPublisherCountersPrefix+key).Get(ctx, &data); err != nil {
		logger.Debugf("Error while querying DB: %v", err)
		return 0, err
	}

	return int32(data.PublishersCount), nil
}

func getEfgsCounter(ctx context.Context, config *config, key string) (*structs.EfgsCounter, error) {
	logger := logging.FromContext(ctx)

	logger.Debugf("Getting EFGS publishers with key %v", key)

	var data structs.EfgsCounter

	if err := config.realtimedbClient.NewRef(constants.DbEfgsCountersPrefix+key).Get(ctx, &data); err != nil {
		if status.Code(err) != codes.NotFound {
			return nil, fmt.Errorf("Error while querying Firestore: %v", err)
		}

		logger.Debug("EFGS counter for '%v' was not found, using default value", key)
	}

	return &data, nil
}
