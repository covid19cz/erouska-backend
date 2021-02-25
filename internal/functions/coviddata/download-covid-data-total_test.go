package coviddata

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

type ClientMock struct {
}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {

	json := `
{
    "modified": "2021-01-11T12:06:57+01:00",
    "source": "https:\/\/onemocneni-aktualne.mzcr.cz\/",
    "data": [
        {
            "datum": "2021-01-11",
            "provedene_testy_celkem": 4076606,
            "potvrzene_pripady_celkem": 835454,
            "aktivni_pripady": 159753,
            "vyleceni": 662429,
            "umrti": 13272,
            "aktualne_hospitalizovani": 6622,
            "provedene_testy_vcerejsi_den": 12185,
            "potvrzene_pripady_vcerejsi_den": 4283,
            "potvrzene_pripady_dnesni_den": 0,
            "provedene_testy_vcerejsi_den_datum": "2021-01-12",
            "potvrzene_pripady_vcerejsi_den_datum": "2021-01-13",
            "potvrzene_pripady_dnesni_den_datum": "2021-01-11",
            "provedene_antigenni_testy_celkem": 1037613,
            "provedene_antigenni_testy_vcerejsi_den": 9743,
            "provedene_antigenni_testy_vcerejsi_den_datum": "2021-01-14",
            "vykazana_ockovani_celkem": 581542,
            "vykazana_ockovani_vcerejsi_den": 16663,
            "vykazana_ockovani_vcerejsi_den_datum": "2021-01-15"
        }
    ]
}
`
	r := ioutil.NopCloser(bytes.NewReader([]byte(json))) // r type is io.ReadCloser

	return &http.Response{Body: r}, nil
}

func TestFetchData(t *testing.T) {

	if err := os.Setenv("UZIS_METRICS_URL", ""); err != nil {
		return
	}

	client := &ClientMock{}

	tables := []struct {
		x TotalsData
	}{
		{
			TotalsData{
				Date:                       "20210111",
				PCRTestsTotal:              4076606,
				ConfirmedCasesTotal:        835454,
				ActiveCasesTotal:           159753,
				CuredTotal:                 662429,
				DeceasedTotal:              13272,
				CurrentlyHospitalizedTotal: 6622,
				PCRTestsIncrease:           12185,
				ConfirmedCasesIncrease:     4283,
				ConfirmedCasesIncreaseDate: "20210113",
				PCRTestsIncreaseDate:       "20210112",
				AntigenTestsTotal:          1037613,
				AntigenTestsIncrease:       9743,
				AntigenTestsIncreaseDate:   "20210114",
				VaccinationsTotal:          581542,
				VaccinationsIncrease:       16663,
				VaccinationsIncreaseDate:   "20210115",
			},
		},
	}

	for _, table := range tables {
		data, err := fetchData(client)

		diff := cmp.Diff(*data, table.x)
		if diff != "" {
			t.Fatalf("register mismatch (-want +got):\n%v", diff)
		}
		if err != nil {
			t.Fatalf("register no error expected")
		}
	}
}
