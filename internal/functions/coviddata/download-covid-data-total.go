package coviddata

import (
	"context"
	"encoding/json"
	"github.com/sethvargo/go-envconfig"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
)

func fetchCovidData(client HTTPClient) (*TotalsData, error) {

	var ctx = context.Background()
	logger := logging.FromContext(ctx)

	var covidMetricsConfig covidMetricsConfig
	if err := envconfig.Process(ctx, &covidMetricsConfig); err != nil {
		logger.Debugf("Could not load covidMetricsConfig: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, covidMetricsConfig.URL, nil)
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

	var r covidDataDownloadRequest

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
	data.AntigenTestsIncreaseDate = reformatDate(data.AntigenTestsIncreaseDate)
	data.PCRTestsIncreaseDate = reformatDate(data.PCRTestsIncreaseDate)
	data.VaccinationsIncreaseDate = reformatDate(data.VaccinationsIncreaseDate)

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

	totalsData, err := fetchCovidData(&spaceClient)
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
