package coviddata

import (
	"context"
	"encoding/json"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
)

type downloadRequest struct {
	Modified string       `json:"modified"`
	Source   string       `json:"source"`
	Data     []TotalsData `json:"data"`
}

// TotalsData holds all the info about tests, cases and results
type TotalsData struct {
	Date                       string `json:"datum" validate:"required"`
	TestsTotal                 int    // for backward compatibility
	ConfirmedCasesTotal        int    `json:"potvrzene_pripady_celkem"  validate:"required"`
	ActiveCasesTotal           int    `json:"aktivni_pripady"  validate:"required"`
	CuredTotal                 int    `json:"vyleceni"  validate:"required"`
	DeceasedTotal              int    `json:"umrti"  validate:"required"`
	CurrentlyHospitalizedTotal int    `json:"aktualne_hospitalizovani"  validate:"required"`
	TestsIncrease              int    // for backward compatibility
	ConfirmedCasesIncrease     int    `json:"potvrzene_pripady_vcerejsi_den" validate:"required"`
	TestsIncreaseDate          string // for backward compatibility
	ConfirmedCasesIncreaseDate string `json:"potvrzene_pripady_vcerejsi_den_datum" validate:"required"`
	AntigenTestsTotal          int    `json:"provedene_antigenni_testy_celkem" validate:"required"`
	AntigenTestsIncrease       int    `json:"provedene_antigenni_testy_vcerejsi_den" validate:"required"`
	AntigenTestsDate           string `json:"provedene_antigenni_testy_vcerejsi_den_datum" validate:"required"`
	PCRTestsTotal              int    `json:"provedene_testy_celkem" validate:"required"`
	PCRTestsIncrease           int    `json:"provedene_testy_vcerejsi_den" validate:"required"`
	PCRTestsIncreaseDate       string `json:"provedene_testy_vcerejsi_den_datum" validate:"required"`
}

// TotalsDataFields are wrapped TotalsData from firestore response
type TotalsDataFields struct {
	Date                       structs.StringValue  `json:"date" validate:"required"`
	TestsTotal                 structs.IntegerValue `json:"testsTotal"  validate:"required"`
	ConfirmedCasesTotal        structs.IntegerValue `json:"confirmedCasesTotal"  validate:"required"`
	ActiveCasesTotal           structs.IntegerValue `json:"activeCasesTotal"  validate:"required"`
	CuredTotal                 structs.IntegerValue `json:"curedTotal"  validate:"required"`
	DeceasedTotal              structs.IntegerValue `json:"deceasedTotal"  validate:"required"`
	CurrentlyHospitalizedTotal structs.IntegerValue `json:"currentlyHospitalizedTotal"  validate:"required"`
	TestsIncrease              structs.IntegerValue `json:"testsIncrease" validate:"required"`
	ConfirmedCasesIncrease     structs.IntegerValue `json:"confirmedCasesIncrease" validate:"required"`
	TestsIncreaseDate          structs.StringValue  `json:"provedene_testy_vcerejsi_den_datum" validate:"required"`
	ConfirmedCasesIncreaseDate structs.StringValue  `json:"potvrzene_pripady_vcerejsi_den_datum" validate:"required"`
}

// HTTPClient interface for mocking fetchData
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func fetchData(client HTTPClient) (*TotalsData, error) {

	var ctx = context.Background()
	logger := logging.FromContext(ctx)

	// TODO: make this configurable
	url := "https://onemocneni-aktualne.mzcr.cz/api/v2/covid-19/zakladni-prehled.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, err
	}

	var r downloadRequest

	jsonErr := json.Unmarshal(body, &r)
	if jsonErr != nil {
		return nil, err
	}

	logger.Debugf("Handling DownloadCovidDataTotal request: %+v", r)

	data := r.Data[0]

	date := data.Date
	if date == "" {
		date = utils.GetTimeNow().Format("20060102")
	} else {
		date = reformatDate(date)
	}

	data.Date = date
	data.ConfirmedCasesIncreaseDate = reformatDate(data.ConfirmedCasesIncreaseDate)
	data.AntigenTestsDate = reformatDate(data.AntigenTestsDate)
	data.PCRTestsIncreaseDate = reformatDate(data.PCRTestsIncreaseDate)

	return &data, nil
}

// DownloadCovidDataTotal downloads coviddata json and writes it to firestore
func DownloadCovidDataTotal(w http.ResponseWriter, r *http.Request) {

	var ctx = context.Background()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	spaceClient := http.Client{
		Timeout: time.Second * 10, // Timeout after 10 seconds
	}

	totalsData, err := fetchData(&spaceClient)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
	}

	date := totalsData.Date

	_, err = client.Doc(constants.CollectionCovidDataTotal, date).Set(ctx, *totalsData)

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	logger.Infof("Successfully written totals data to firestore (key %v): %+v", date, totalsData)

	httputils.SendResponse(w, r, struct{ status string }{status: "OK"})
}

// convert 2020-08-19 to 20200819
func reformatDate(date string) string {
	if date == "" {
		return ""
	}
	return strings.ReplaceAll(date, "-", "")
}
