package coviddata

import (
	"context"
	"encoding/json"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/sethvargo/go-envconfig"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// DownloadAndCountVaccinations downloads vaccinations metrics json and writes it to firestore
func DownloadAndCountVaccinations(w http.ResponseWriter, r *http.Request) {
	var ctx = context.Background()
	logger := logging.FromContext(ctx).Named("DownloadAndCountVaccination")
	client := store.Client{}

	httpClient := http.Client{
		Timeout: time.Second * 10, // Timeout after 10 seconds
	}

	vaccinationData, err := fetchVaccinationsData(ctx, &httpClient)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
		httputils.SendErrorResponse(w, r, err)
		return
	}

	if err = persistVaccinationsData(ctx, client, vaccinationData); err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendResponse(w, r, struct{ status string }{status: "OK"})
}

func fetchVaccinationsData(ctx context.Context, client HTTPClient) (*VaccinationsAggregatedData, error) {
	logger := logging.FromContext(ctx).Named("fetchVaccinationsData")

	var vaccinationMetricsConfig vaccinationMetricsConfig
	if err := envconfig.Process(ctx, &vaccinationMetricsConfig); err != nil {
		logger.Debugf("Could not load vaccinationMetricsConfig: %v", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, vaccinationMetricsConfig.URL, nil)
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

	var r vaccinationDownloadRequest

	jsonErr := json.Unmarshal(body, &r)
	if jsonErr != nil {
		return nil, err
	}

	data := r.Data

	sumByDate := make(map[string]VaccinationsAggregatedData)
	totalFirstDose := 0
	totalSecondDose := 0
	modification, _ := time.Parse(time.RFC3339, r.Modified)
	lastDate := "" // date nearest today

	// always must be processed all data in JSON due to possible fixes
	for _, v := range data {
		date := reformatDate(v.Date)
		vd := sumByDate[date]
		vd.Date = date
		vd.DailyFirstDose += v.FirstDose
		vd.DailySecondDose += v.SecondDose
		// each single dose vaccine is counted as completed after the first dose
		if isSingleDoseVaccine(v.Vaccine) {
			totalSecondDose += v.FirstDose
			vd.DailySecondDose += v.FirstDose
		}
		totalFirstDose += v.FirstDose
		totalSecondDose += v.SecondDose
		sumByDate[date] = vd
		if vd.Date > lastDate {
			lastDate = vd.Date
		}
	}

	lastDateVaccinations := VaccinationsAggregatedData{
		Date:            lastDate,
		Modified:        modification.Unix(),
		DailyFirstDose:  sumByDate[lastDate].DailyFirstDose,
		DailySecondDose: sumByDate[lastDate].DailySecondDose,
		TotalFirstDose:  totalFirstDose,
		TotalSecondDose: totalSecondDose,
	}

	logger.Debugf("Fetched new vaccinations metrics: %+v", lastDateVaccinations)

	return &lastDateVaccinations, nil
}

func persistVaccinationsData(ctx context.Context, client store.Client, data *VaccinationsAggregatedData) error {
	logger := logging.FromContext(ctx).Named("PersistVaccinationsData")

	date := data.Date

	if _, err := client.Doc(constants.CollectionVaccinations, date).Set(ctx, *data); err != nil {
		return err
	}

	logger.Infof("Successfully written vaccination data to firestore (key %v): %+v", date, data)
	return nil
}

func isSingleDoseVaccine(vaccine string) bool {
	singleDoseVaccines := os.Getenv("SINGLE_DOSE_VACCINES")
	list := strings.Split(singleDoseVaccines, ",")
	for _, v := range list {
		if v == vaccine {
			return true
		}
	}
	return false
}
