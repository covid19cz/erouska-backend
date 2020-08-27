package coviddata

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/covid19cz/erouska-backend/internal/utils/errors"

	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type getRequest struct {
	Date string `json:"date"`
}

type response struct {
	Date                          string `json:"date"`
	TestsIncrease                 int    `json:"testsIncrease"  validate:"required"`
	ConfirmedCasesIncrease        int    `json:"confirmedCasesIncrease"  validate:"required"`
	ActiveCasesIncrease           int    `json:"activeCasesIncrease"  validate:"required"`
	CuredIncrease                 int    `json:"curedIncrease"  validate:"required"`
	DeceasedIncrease              int    `json:"deceasedIncrease"  validate:"required"`
	CurrentlyHospitalizedIncrease int    `json:"currentlyHospitalizedIncrease"  validate:"required"`
	TestsTotal                    int    `json:"testsTotal"  validate:"required"`
	ConfirmedCasesTotal           int    `json:"confirmedCasesTotal"  validate:"required"`
	ActiveCasesTotal              int    `json:"activeCasesTotal"  validate:"required"`
	CuredTotal                    int    `json:"curedTotal"  validate:"required"`
	DeceasedTotal                 int    `json:"deceasedTotal"  validate:"required"`
	CurrentlyHospitalizedTotal    int    `json:"currentlyHospitalizedTotal"  validate:"required"`
}

func fetchGeneric(ctx context.Context, client store.Client, col string, date string) (interface{}, error) {
	logger := logging.FromContext(ctx)

	snap, err := client.Doc(col, date).Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, &errors.NotFoundError{Msg: fmt.Sprintf("Could not find covid data for %v", date)}
		}

		return nil, fmt.Errorf("Error while querying Firestore: %v", err)
	}

	logger.Infof("fetched firestore data: %+v", snap.Data())

	var data interface{}

	if err := snap.DataTo(&data); err != nil {
		panic(fmt.Sprintf("could not parse input: %s", err))
	}

	logger.Infof("fetched data: %+v", data)

	return &data, nil
}

func fetchCovidData(ctx context.Context, client store.Client, date string) (*TotalsData, *IncreaseData, error) {
	logger := logging.FromContext(ctx)

	totalsData, err := fetchGeneric(ctx, client, constants.CollectionCovidDataTotal, date)
	if err != nil {
		logger.Errorf("Error fetching totalsData from firestore: %v", err)
		return nil, nil, err
	}

	increaseData, err := fetchGeneric(ctx, client, constants.CollectionCovidDataIncrease, date)
	if err != nil {
		logger.Errorf("Error fetching increaseData from firestore: %v", err)
		return nil, nil, err
	}

	return totalsData.(*TotalsData), increaseData.(*IncreaseData), nil
}

// GetCovidData handler.
func GetCovidData(w http.ResponseWriter, r *http.Request) {

	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var req getRequest

	if !httputils.DecodeJSONOrReportError(w, r, &req) {
		return
	}

	date := req.Date

	// if no date was specified in input
	// and there is no data for today, try to get
	// data for yesterday
	shouldFallback := false

	if date == "" {
		date = utils.GetTimeNow().Format("20060102")
		shouldFallback = true
	}

	failed := false

	totalsData, increaseData, err := fetchCovidData(ctx, client, date)
	if err != nil {
		logger.Errorf("Error fetching data from firestore: %v", err)
		failed = true
		if !shouldFallback {
			httputils.SendErrorResponse(w, r, err)
			return
		}
	}

	if failed && shouldFallback {
		// we try to fetch data from yesterday
		t, _ := time.Parse("20060102", date)
		date = t.AddDate(0, 0, -1).Format("20060102")

		totalsData, increaseData, err = fetchCovidData(ctx, client, date)
		if err != nil {
			logger.Errorf("Error refetching data from firestore: %v", err)
			failed = true
			if !shouldFallback {
				httputils.SendErrorResponse(w, r, err)
				return
			}
		}
	}

	res := response{
		Date:                          date,
		TestsIncrease:                 increaseData.TestsIncrease,
		TestsTotal:                    totalsData.TestsTotal,
		ConfirmedCasesIncrease:        increaseData.ConfirmedCasesIncrease,
		ConfirmedCasesTotal:           totalsData.ConfirmedCasesTotal,
		ActiveCasesIncrease:           increaseData.ActiveCasesIncrease,
		ActiveCasesTotal:              totalsData.ActiveCasesTotal,
		CuredIncrease:                 increaseData.CuredIncrease,
		CuredTotal:                    totalsData.CuredTotal,
		DeceasedIncrease:              increaseData.DeceasedIncrease,
		DeceasedTotal:                 totalsData.DeceasedTotal,
		CurrentlyHospitalizedIncrease: increaseData.CurrentlyHospitalizedIncrease,
		CurrentlyHospitalizedTotal:    totalsData.CurrentlyHospitalizedTotal,
	}

	httputils.SendResponse(w, r, res)

	return
}
