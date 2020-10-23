package coviddata

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"io/ioutil"
	"net/http"
	"testing"
)

type ClientMock struct {
}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {

	json := `
{
    "modified": "2020-10-23T01:03:31+02:00",
    "source": "https:\/\/onemocneni-aktualne.mzcr.cz\/",
    "data": [
        {
            "datum": "2020-10-23",
            "provedene_testy_celkem": 805609,
            "potvrzene_pripady_celkem": 20483,
            "aktivni_pripady": 4934,
            "vyleceni": 15148,
            "umrti": 401,
            "aktualne_hospitalizovani": 122,
            "provedene_testy_vcerejsi_den": 15401,
            "potvrzene_pripady_vcerejsi_den": 1163,
            "potvrzene_pripady_dnesni_den": 701,
			"provedene_testy_vcerejsi_den_datum": "2020-10-22",
			"potvrzene_pripady_vcerejsi_den_datum": "2020-10-22",
			"potvrzene_pripady_dnesni_den_datum": "2020-10-23"
        }
    ]
}
`

	r := ioutil.NopCloser(bytes.NewReader([]byte(json))) // r type is io.ReadCloser

	return &http.Response{Body: r}, nil
}

func TestFetchData(t *testing.T) {

	client := &ClientMock{}

	tables := []struct {
		x TotalsData
	}{
		{
			TotalsData{
				Date:                       "20201023",
				TestsTotal:                 805609,
				ConfirmedCasesTotal:        20483,
				ActiveCasesTotal:           4934,
				CuredTotal:                 15148,
				DeceasedTotal:              401,
				CurrentlyHospitalizedTotal: 122,
				TestsIncrease:              15401,
				ConfirmedCasesIncrease:     1163,
				ConfirmedCasesIncreaseDate: "20201022",
				TestsIncreaseDate:          "20201022",
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
