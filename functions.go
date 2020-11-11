package functions

import (
	"context"
	"github.com/covid19cz/erouska-backend/internal/pubsub"

	"github.com/covid19cz/erouska-backend/internal/functions/changepushtoken"
	"github.com/covid19cz/erouska-backend/internal/functions/coviddata"
	"github.com/covid19cz/erouska-backend/internal/functions/efgs"
	"github.com/covid19cz/erouska-backend/internal/functions/isehridactive"
	"github.com/covid19cz/erouska-backend/internal/functions/metricsapi"
	"github.com/covid19cz/erouska-backend/internal/functions/publishkeys"
	"github.com/covid19cz/erouska-backend/internal/functions/registerehrid"
	"github.com/covid19cz/erouska-backend/internal/functions/registernotification"

	"net/http"
)

// RegisterEhrid Registration handler.
func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	registerehrid.RegisterEhrid(w, r)
}

// IsEhridActive IsEhridActive handler.
func IsEhridActive(w http.ResponseWriter, r *http.Request) {
	isehridactive.IsEhridActive(w, r)
}

// ChangePushToken ChangePushToken handler.
func ChangePushToken(w http.ResponseWriter, r *http.Request) {
	changepushtoken.ChangePushToken(w, r)
}

// RegisterNotification RegisterNotification handler.
func RegisterNotification(w http.ResponseWriter, r *http.Request) {
	registernotification.RegisterNotification(w, r)
}

// RegisterNotificationAfterMath RegisterNotificationAfterMath handler.
func RegisterNotificationAfterMath(ctx context.Context, m pubsub.Message) error {
	return registernotification.AfterMath(ctx, m)
}

// DownloadCovidDataTotal handler.
func DownloadCovidDataTotal(w http.ResponseWriter, r *http.Request) {
	coviddata.DownloadCovidDataTotal(w, r)
}

// GetCovidData handler.
func GetCovidData(w http.ResponseWriter, r *http.Request) {
	coviddata.GetCovidData(w, r)
}

//PrepareNewMetricsVersion handler.
func PrepareNewMetricsVersion(w http.ResponseWriter, r *http.Request) {
	metricsapi.PrepareNewVersion(w, r)
}

//DownloadMetrics handler.
func DownloadMetrics(w http.ResponseWriter, r *http.Request) {
	metricsapi.DownloadMetrics(w, r)
}

//RegisterEhridAfterMath handler.
func RegisterEhridAfterMath(ctx context.Context, m pubsub.Message) error {
	return registerehrid.AfterMath(ctx, m)
}

// ***************
// EFGS functions:
// ***************

// PublishKeys handler.
func PublishKeys(w http.ResponseWriter, r *http.Request) {
	publishkeys.PublishKeys(w, r)
}

//EfgsUploadKeys handler.
func EfgsUploadKeys(w http.ResponseWriter, r *http.Request) {
	efgs.UploadBatch(w, r)
}

// EfgsDownloadKeys downloads EFGS keys batch
func EfgsDownloadKeys(ctx context.Context, m pubsub.Message) error {
	return efgs.DownloadAndSaveKeys(ctx, m)
}

// EfgsDownloadYesterdaysKeys downloads EFGS keys batch from whole yesterday
func EfgsDownloadYesterdaysKeys(w http.ResponseWriter, r *http.Request) {
	efgs.DownloadAndSaveYesterdaysKeys(w, r)
}

//EfgsRemoveOldKeys handler.
func EfgsRemoveOldKeys(w http.ResponseWriter, r *http.Request) {
	efgs.CleanupDatabase(w, r)
}

//EfgsIssueTestingVerificationCode handler.
func EfgsIssueTestingVerificationCode(w http.ResponseWriter, r *http.Request) {
	efgs.IssueTestingVerificationCode(w, r)
}
