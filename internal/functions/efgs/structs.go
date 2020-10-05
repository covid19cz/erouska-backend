package efgs

import "time"

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
	ReportType                 string    `json:"reportType,omitempty"`
	DaysSinceOnsetOfSymptoms   int32     `json:"days_since_onset_of_symptoms,omitempty"`
}

//ToData convert struct from DiagnosisKeyWrapper to DiagnosisKey
func (wrappedKey *DiagnosisKeyWrapper) ToData() *DiagnosisKey {
	var mapReportType = map[string]ReportType{
		"UNKNOWN":                      ReportType_UNKNOWN,
		"CONFIRMED_TEST":               ReportType_CONFIRMED_TEST,
		"CONFIRMED_CLINICAL_DIAGNOSIS": ReportType_CONFIRMED_CLINICAL_DIAGNOSIS,
		"SELF_REPORT":                  ReportType_SELF_REPORT,
		"RECURSIVE":                    ReportType_RECURSIVE,
		"REVOKED":                      ReportType_REVOKED,
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
