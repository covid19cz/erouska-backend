package api

import (
	"encoding/base64"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"time"
)

//GenericVerServerResponse Generic part of the Verification server response.
type GenericVerServerResponse struct {
	Error     string `json:"error"`
	ErrorCode string `json:"errorCode"`
}

//IssueCodeRequest Issue code request to the Verification server
type IssueCodeRequest struct {
	Phone    string `json:"phone"`
	TestType string `json:"testType"`
}

//IssueCodeResponse Issue code response from the Verification server
type IssueCodeResponse struct {
	GenericVerServerResponse
	Code string `json:"code"`
}

//VerifyRequest  Verify request to the Verification server
type VerifyRequest struct {
	Code string `json:"code"`
}

//VerifyResponse Verify response from the Verification server
type VerifyResponse struct {
	GenericVerServerResponse
	Token string `json:"token"`
}

//CertificateRequest Cerificate request to the Verification server
type CertificateRequest struct {
	Token   string `json:"token"`
	KeyHmac string `json:"ekeyhmac"`
}

//CertificateResponse Certificate response to the Verification server
type CertificateResponse struct {
	GenericVerServerResponse
	Certificate string `json:"certificate"`
}

//DownloadBatchResponse Response for download batch call to EFGS
type DownloadBatchResponse struct {
	Keys []DiagnosisKeyWrapper `json:"keys"`
}

//UploadBatchResponse Response for upload batch call to EFGS
type UploadBatchResponse struct {
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
