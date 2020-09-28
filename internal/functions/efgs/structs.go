package efgs

import (
	"encoding/base64"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"time"
)

type genericVerServerResponse struct {
	Error     string `json:"error"`
	ErrorCode string `json:"errorCode"`
}

type issueCodeRequest struct {
	Phone    string `json:"phone"`
	TestType string `json:"testType"`
}

type issueCodeResponse struct {
	genericVerServerResponse
	Code string `json:"code"`
}

type verifyRequest struct {
	Code string `json:"code"`
}

type verifyResponse struct {
	genericVerServerResponse
	Token string `json:"token"`
}

type certificateRequest struct {
	Token   string `json:"token"`
	KeyHmac string `json:"ekeyhmac"`
}

type certificateResponse struct {
	genericVerServerResponse
	Certificate string `json:"certificate"`
}

type downloadBatchResponse struct {
	Keys []DiagnosisKeyWrapper `json:"keys"`
}

type uploadBatchResponse struct {
	Error     []int `json:"500"`
	Duplicate []int `json:"409"`
	Success   []int `json:"201"`
}

//BatchDownloadParams Struct holding download input data.
type BatchDownloadParams struct {
	Date     string `json:"date" validate:"required"`
	BatchTag string `json:"batchTag"`
}

//DiagnosisKeyWrapper map json response from EFGS to local DiagnosisKey structure
type DiagnosisKeyWrapper struct {
	tableName                  struct{}  `pg:"diagnosis_keys,alias:dk"`
	ID                         int32     `pg:",pk" json:"id"`
	CreatedAt                  time.Time `pg:"default:now()" json:"created_at"`
	KeyData                    []byte    `json:"keyData,omitempty"`
	RollingStartIntervalNumber uint32    `json:"rollingStartIntervalNumber,omitempty"`
	RollingPeriod              uint32    `json:"rollingPeriod,omitempty"`
	TransmissionRiskLevel      int32     `json:"transmissionRiskLevel,omitempty"`
	VisitedCountries           []string  `json:"visitedCountries,omitempty"`
	Origin                     string    `pg:"default:'CZ'" json:"origin,omitempty"`
	ReportType                 int       `json:"reportType,omitempty"`
	DaysSinceOnsetOfSymptoms   int32     `json:"days_since_onset_of_symptoms,omitempty"`
}

//ToData convert struct from DiagnosisKeyWrapper to DiagnosisKey
func (wrappedKey *DiagnosisKeyWrapper) ToData() *DiagnosisKey {
	var mapReportType = map[int]ReportType{
		0: ReportType_UNKNOWN,
		1: ReportType_CONFIRMED_TEST,
		2: ReportType_CONFIRMED_CLINICAL_DIAGNOSIS,
		3: ReportType_SELF_REPORT,
		4: ReportType_RECURSIVE,
		5: ReportType_REVOKED,
	}

	return &DiagnosisKey{
		KeyData:                    wrappedKey.KeyData,
		RollingStartIntervalNumber: wrappedKey.RollingStartIntervalNumber,
		RollingPeriod:              wrappedKey.RollingPeriod,
		TransmissionRiskLevel:      wrappedKey.TransmissionRiskLevel,
		Origin:                     wrappedKey.Origin,
		DaysSinceOnsetOfSymptoms:   wrappedKey.DaysSinceOnsetOfSymptoms,
		VisitedCountries:           wrappedKey.VisitedCountries,
		ReportType:                 mapReportType[wrappedKey.ReportType],
	}
}

//ToExposureKey convert struct from DiagnosisKeyWrapper to DiagnosisKey
func (key *DiagnosisKey) ToExposureKey() keyserverapi.ExposureKey {
	return keyserverapi.ExposureKey{
		Key:              base64.StdEncoding.EncodeToString(key.KeyData),
		IntervalNumber:   int32(key.RollingStartIntervalNumber),
		IntervalCount:    int32(key.RollingPeriod),
		TransmissionRisk: int(key.TransmissionRiskLevel),
	}
}
