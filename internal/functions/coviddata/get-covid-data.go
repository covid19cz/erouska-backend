package coviddata

import (
	"fmt"
	"net/http"

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

// GetCovidData handler.
func GetCovidData(w http.ResponseWriter, r *http.Request) error {

	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var req getRequest

	if !httputils.DecodeJSONOrReportError(w, r, &req) {
		return nil
	}

	date := req.Date

	if date == "" {
		date = utils.GetTimeNow().Format("20060102")
	}

	snap, err := client.Doc(constants.CollectionCovidDataTotal, date).Get(ctx)

	if err != nil {
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("NotFound error while querying Firestore: %v", err)
		}

		return fmt.Errorf("Error while querying Firestore: %v", err)
	}

	logger.Infof("fetched firestore event: %+v", snap.Data())

	var totals TotalsData

	if err := snap.DataTo(&totals); err != nil {
		panic(fmt.Sprintf("could not parse input: %s", err))
	}

	logger.Infof("fetched data: %+v", totals)

	snap, err = client.Doc(constants.CollectionCovidDataIncrease, date).Get(ctx)

	if err != nil {
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("NotFound error while querying Firestore: %v", err)
		}

		return fmt.Errorf("Error while querying Firestore: %v", err)
	}

	logger.Infof("fetched firestore event: %+v", snap.Data())

	var increase IncreaseData

	if err := snap.DataTo(&increase); err != nil {
		panic(fmt.Sprintf("could not parse input: %s", err))
	}

	logger.Infof("fetched data: %+v", increase)

	totalsData := totals

	if err != nil {
		panic(fmt.Sprintf("could not convert firestore fields to data: %s", err))
	}

	increaseData := increase

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

	return nil
}
