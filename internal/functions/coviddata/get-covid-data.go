package coviddata

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fetchTotals(ctx context.Context, client store.Client, date string) (*TotalsData, error) {
	logger := logging.FromContext(ctx)

	snap, err := client.Doc(constants.CollectionCovidDataTotal, date).Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, &errors.NotFoundError{Msg: fmt.Sprintf("Could not find covid data for %v", date)}
		}

		return nil, fmt.Errorf("Error while querying Firestore: %v", err)
	}

	logger.Infof("fetched firestore data: %+v", snap.Data())

	var totals TotalsData

	if err := snap.DataTo(&totals); err != nil {
		panic(fmt.Sprintf("could not parse input: %s", err))
	}

	logger.Infof("fetched data: %+v", totals)

	return &totals, nil
}

func fetchVaccinations(ctx context.Context, client store.Client, date string) (*VaccinationsAggregatedData, error) {
	logger := logging.FromContext(ctx).Named("fetchVaccinations")

	snap, err := client.Doc(constants.CollectionVaccinations, date).Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, &errors.NotFoundError{Msg: fmt.Sprintf("Could not find vaccination data for %v", date)}
		}

		return nil, fmt.Errorf("Error while querying Firestore: %v", err)
	}

	var vaccinationData VaccinationsAggregatedData

	if err := snap.DataTo(&vaccinationData); err != nil {
		panic(fmt.Sprintf("could not parse input: %s", err))
	}

	logger.Infof("fetched vaccinations data: %+v", vaccinationData)

	return &vaccinationData, nil
}

// GetCovidData handler.
func GetCovidData(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}
	//authClient := auth.Client{}

	var req v1.GetCovidDataRequest

	if !httputils.DecodeJSONOrReportError(w, r, &req) {
		return
	}

	//ehrid, err := authClient.AuthenticateToken(ctx, req.IDToken)
	//if err != nil {
	//	logger.Debugf("Unverifiable token provided: %+v %+v", req.IDToken, err.Error())
	//	httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
	//	return
	//}
	//
	//logger.Debugf("Handling GetCovidData request: %v %+v", ehrid, req)

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
	vaccinationsFailed := false

	totalsData, err := fetchTotals(ctx, storeClient, date)
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
		totalsDataDate := t.AddDate(0, 0, -1).Format("20060102")

		totalsData, err = fetchTotals(ctx, storeClient, totalsDataDate)
		if err != nil {
			logger.Errorf("Error refetching data from firestore: %v", err)
			httputils.SendErrorResponse(w, r, err)
			return
		}
	}

	vaccinationData, err := fetchVaccinations(ctx, storeClient, date)
	if err != nil {
		logger.Errorf("Error fatching vaccination data from firestore: %v", err)
		vaccinationsFailed = true
		if !shouldFallback {
			httputils.SendErrorResponse(w, r, err)
			return
		}
	}

	if vaccinationsFailed && shouldFallback {
		// we try to fetch data from yesterday
		t, _ := time.Parse("20060102", date)
		vaccinationsDataDate := t.AddDate(0, 0, -1).Format("20060102")

		vaccinationData, err = fetchVaccinations(ctx, storeClient, vaccinationsDataDate)
		if err != nil {
			logger.Errorf("Error refetching vaccinations firestore: %v", err)
			httputils.SendErrorResponse(w, r, err)
			return
		}
	}

	// backward compatibility (when the date where PCR tests attributes are not defined is queried)
	testsTotal := totalsData.PCRTestsTotal
	testsIncrease := totalsData.PCRTestsIncrease
	testsIncreaseDate := totalsData.PCRTestsIncreaseDate

	if testsTotal == 0 {
		testsTotal = totalsData.TestsTotal
	}

	if testsIncrease == 0 {
		testsIncrease = totalsData.TestsIncrease
	}

	if testsIncreaseDate == "" {
		testsIncreaseDate = totalsData.TestsIncreaseDate
	}

	res := v1.GetCovidDataResponse{
		Date:                        totalsData.Date,
		ActiveCasesTotal:            totalsData.ActiveCasesTotal,
		CuredTotal:                  totalsData.CuredTotal,
		DeceasedTotal:               totalsData.DeceasedTotal,
		CurrentlyHospitalizedTotal:  totalsData.CurrentlyHospitalizedTotal,
		TestsTotal:                  testsTotal,        // this value is duplicated for backward compatibility
		TestsIncrease:               testsIncrease,     // this value is duplicated for backward compatibility
		TestsIncreaseDate:           testsIncreaseDate, // this value is duplicated for backward compatibility
		ConfirmedCasesTotal:         totalsData.ConfirmedCasesTotal,
		ConfirmedCasesIncrease:      totalsData.ConfirmedCasesIncrease,
		ConfirmedCasesIncreaseDate:  totalsData.ConfirmedCasesIncreaseDate,
		AntigenTestsTotal:           totalsData.AntigenTestsTotal,
		AntigenTestsIncrease:        totalsData.AntigenTestsIncrease,
		AntigenTestsIncreaseDate:    totalsData.AntigenTestsIncreaseDate,
		PCRTestsTotal:               testsTotal,
		PCRTestsIncrease:            testsIncrease,
		PCRTestsIncreaseDate:        testsIncreaseDate,
		VaccinationsTotal:           totalsData.VaccinationsTotal,
		VaccinationsIncrease:        totalsData.VaccinationsIncrease,
		VaccinationsIncreaseDate:    totalsData.VaccinationsIncreaseDate,
		VaccinationsDailyDosesDate:  vaccinationData.Date,
		VaccinationsDailyFirstDose:  vaccinationData.DailyFirstDose,
		VaccinationsDailySecondDose: vaccinationData.DailySecondDose,
		VaccinationsTotalFirstDose:  vaccinationData.TotalFirstDose,
		VaccinationsTotalSecondDose: vaccinationData.TotalSecondDose,
	}

	httputils.SendResponse(w, r, res)
}
