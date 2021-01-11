package v1

import (
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
)

/*
This files contains request/response structs for all endpoints (except PublishKeys). The structs have to be changed in
backward-compatible way and when it's not possible, copied to `v2` and changed there.
*/

//RegisterEhridRequest Request for RegisterEhrid function
type RegisterEhridRequest struct {
	Platform              string `json:"platform" validate:"required,oneof=android ios"`
	PlatformVersion       string `json:"platformVersion" validate:"required"`
	Manufacturer          string `json:"manufacturer"`
	Model                 string `json:"model"`
	Locale                string `json:"locale" validate:"required"`
	PushRegistrationToken string `json:"pushRegistrationToken"`
}

//RegisterEhridResponse Response for RegisterEhrid function
type RegisterEhridResponse struct {
	CustomToken string `json:"customToken"`
}

//IsEhridActiveRequest Request for IsEhridActive function
type IsEhridActiveRequest struct {
	IDToken string `json:"idToken" validate:"required"`
}

//IsEhridActiveResponse Response for IsEhridActive function
type IsEhridActiveResponse struct {
	Active bool `json:"active"`
}

//ChangePushTokenRequest Request for ChangePushToken function
type ChangePushTokenRequest struct {
	IDToken               string `json:"idToken" validate:"required"`
	PushRegistrationToken string `json:"pushRegistrationToken" validate:"required"`
}

//RegisterNotificationRequest Request for RegisterNotification function
type RegisterNotificationRequest struct {
	IDToken string `json:"idToken" validate:"required"`
}

//GetCovidDataRequest Request for GetCovidData function
type GetCovidDataRequest struct {
	IDToken string `json:"idToken" validate:"required"`
	Date    string `json:"date"`
}

//GetCovidDataResponse Response for GetCovidData function
type GetCovidDataResponse struct {
	Date                       string `json:"date"`
	TestsIncrease              int    `json:"testsIncrease"  validate:"required"`
	ConfirmedCasesIncrease     int    `json:"confirmedCasesIncrease"  validate:"required"`
	TestsTotal                 int    `json:"testsTotal"  validate:"required"`
	ConfirmedCasesTotal        int    `json:"confirmedCasesTotal"  validate:"required"`
	ActiveCasesTotal           int    `json:"activeCasesTotal"  validate:"required"`
	CuredTotal                 int    `json:"curedTotal"  validate:"required"`
	DeceasedTotal              int    `json:"deceasedTotal"  validate:"required"`
	CurrentlyHospitalizedTotal int    `json:"currentlyHospitalizedTotal"  validate:"required"`
	TestsIncreaseDate          string `json:"testsIncreaseDate" validate:"required"`
	ConfirmedCasesIncreaseDate string `json:"confirmedCasesIncreaseDate" validate:"required"`
	AntigenTestsTotal          int    `json:"antigenTestsTotal" validate:"required"`
	AntigenTestsIncrease       int    `json:"antigenTestsIncrease"  validate:"required"`
	AntigenTestsDate           string `json:"antigenTestsDate" validate:"required"`
	PCRTestsTotal              int    `json:"pcrTestsTotal"  validate:"required"`
	PCRTestsIncrease           int    `json:"pcrTestsIncrease"  validate:"required"`
	PCRTestsIncreaseDate       string `json:"pcrTestsIncreaseDate" validate:"required"`
}

//DownloadMetricsRequest Request for DownloadMetrics function
type DownloadMetricsRequest struct {
	Date string `json:"date"`
}

//DownloadMetricsResponse Response for DownloadMetrics function
type DownloadMetricsResponse = structs.MetricsData
