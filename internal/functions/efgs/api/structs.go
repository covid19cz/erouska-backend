package api

import (
	"encoding/base64"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"strconv"
	"strings"
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
	Keys []DiagnosisKey `json:"keys"`
}

var mapReportTypeInt = map[int]ReportType{
	0: ReportType_UNKNOWN,
	1: ReportType_CONFIRMED_TEST,
	2: ReportType_CONFIRMED_CLINICAL_DIAGNOSIS,
	3: ReportType_SELF_REPORT,
	4: ReportType_RECURSIVE,
	5: ReportType_REVOKED,
}

var mapReportTypeString = map[string]ReportType{
	"UNKNOWN":                      ReportType_UNKNOWN,
	"CONFIRMED_TEST":               ReportType_CONFIRMED_TEST,
	"CONFIRMED_CLINICAL_DIAGNOSIS": ReportType_CONFIRMED_CLINICAL_DIAGNOSIS,
	"SELF_REPORT":                  ReportType_SELF_REPORT,
	"RECURSIVE":                    ReportType_RECURSIVE,
	"REVOKED":                      ReportType_REVOKED,
}

//UnmarshalJSON Accepts ReportType in both integer and string form.
func (s *ReportType) UnmarshalJSON(data []byte) error {
	str := strings.TrimLeft(strings.TrimRight(string(data), "\""), "\"")

	number, err := strconv.Atoi(str)

	if err == nil {
		*s = mapReportTypeInt[number]
	} else {
		*s = mapReportTypeString[str]
	}

	return nil
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
	rt := wrappedKey.TransmissionRiskLevel
	if rt > 8 || rt < 0 {
		// This is a sad story of how EFGS has put this into their protocol and then recommended to not use it because the value meaning can
		// be different across different countries. Because it cannot be unused in fact, the final recommendation is to put max value possible
		// there. That works for others but, because we put those keys into the Key server, not for us - the Key server doesn't like it so we
		// have to adjust the value to basically anything consumable. The value itself is not used at all for anything and even though it's
		// said to be optional in comments, the Key server requires it and fails when the value is not there or is invalid.
		// Sad story.
		//
		rt = 0 // This is an override for putting 0x7fffffff there
	}

	return &DiagnosisKey{
		KeyData:                    wrappedKey.KeyData,
		RollingStartIntervalNumber: wrappedKey.RollingStartIntervalNumber,
		RollingPeriod:              wrappedKey.RollingPeriod,
		TransmissionRiskLevel:      rt,
		Origin:                     wrappedKey.Origin,
		DaysSinceOnsetOfSymptoms:   wrappedKey.DaysSinceOnsetOfSymptoms,
		VisitedCountries:           wrappedKey.VisitedCountries,
		ReportType:                 mapReportTypeInt[wrappedKey.ReportType],
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
