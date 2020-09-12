package coviddata

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/covid19cz/erouska-backend/internal/auth"
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

// GetCovidData handler.
func GetCovidData(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}
	authClient := auth.Client{}

	var req v1.GetCovidDataRequest

	if !httputils.DecodeJSONOrReportError(w, r, &req) {
		return
	}

	ehrid, err := authClient.AuthenticateToken(ctx, req.IDToken)
	if err != nil {
		logger.Debugf("Unverifiable token provided: %+v %+v", req.IDToken, err.Error())
		httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
		return
	}

	logger.Debugf("Handling GetCovidData request: %v %+v", ehrid, req)

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
		date = t.AddDate(0, 0, -1).Format("20060102")

		totalsData, err = fetchTotals(ctx, storeClient, date)
		if err != nil {
			logger.Errorf("Error refetching data from firestore: %v", err)
			httputils.SendErrorResponse(w, r, err)
			return
		}
	}

	res := v1.GetCovidDataResponse{
		Date:                       date,
		TestsIncrease:              totalsData.TestsIncrease,
		TestsTotal:                 totalsData.TestsTotal,
		ConfirmedCasesIncrease:     totalsData.ConfirmedCasesIncrease,
		ConfirmedCasesTotal:        totalsData.ConfirmedCasesTotal,
		ActiveCasesTotal:           totalsData.ActiveCasesTotal,
		CuredTotal:                 totalsData.CuredTotal,
		DeceasedTotal:              totalsData.DeceasedTotal,
		CurrentlyHospitalizedTotal: totalsData.CurrentlyHospitalizedTotal,
	}

	httputils.SendResponse(w, r, res)
}
