package coviddata

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	rpccode "google.golang.org/genproto/googleapis/rpc/code"

	"cloud.google.com/go/firestore"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	TestsTotal                 int    `json:"provedene_testy_celkem"  validate:"required"`
	ConfirmedCasesTotal        int    `json:"potvrzene_pripady_celkem"  validate:"required"`
	ActiveCasesTotal           int    `json:"aktivni_pripady"  validate:"required"`
	CuredTotal                 int    `json:"vyleceni"  validate:"required"`
	DeceasedTotal              int    `json:"umrti"  validate:"required"`
	CurrentlyHospitalizedTotal int    `json:"aktualne_hospitalizovani"  validate:"required"`
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
		// convert 2020-08-19 to 20200819
		date = strings.ReplaceAll(date, "-", "")
	}

	data.Date = date

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

	d, err := fetchData(&spaceClient)
	if err != nil {
		logger.Errorf("Error while fetching data: %v", err)
	}

	date := d.Date

	doc := client.Doc(constants.CollectionCovidDataTotal, date)

	err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		_, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}

			return tx.Set(doc, *d)
		}
		return nil
	})

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, rpccode.Code_INTERNAL, "Unknown error")
		return
	}

	logger.Infof("Succesfully written data to firestore: %+v", d)

	httputils.SendResponse(w, r, struct{ status string }{status: "OK"})
}
