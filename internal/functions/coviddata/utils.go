package coviddata

import (
	"net/http"
	"strings"
)

// HTTPClient interface for mocking fetchCovidData
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// convert 2020-08-19 to 20200819
func reformatDate(date string) string {
	if date == "" {
		return ""
	}
	return strings.ReplaceAll(date, "-", "")
}
