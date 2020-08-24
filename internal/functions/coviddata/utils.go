package coviddata

import (
	"fmt"
	"strconv"
)

// ToData converts Fields interface to TotalsData
func (f TotalsDataFields) ToData() (*TotalsData, error) {

	testsTotal, err := strconv.Atoi(f.TestsTotal.IntegerValue)
	if err != nil {
		return nil, fmt.Errorf("Error converting string to int: %v", err)
	}
	confirmedCasesTotal, err := strconv.Atoi(f.ConfirmedCasesTotal.IntegerValue)
	if err != nil {
		return nil, fmt.Errorf("Error converting string to int: %v", err)
	}
	activeCasesTotal, err := strconv.Atoi(f.ActiveCasesTotal.IntegerValue)
	if err != nil {
		return nil, fmt.Errorf("Error converting string to int: %v", err)
	}
	curedTotal, err := strconv.Atoi(f.CuredTotal.IntegerValue)
	if err != nil {
		return nil, fmt.Errorf("Error converting string to int: %v", err)
	}
	deceasedTotal, err := strconv.Atoi(f.DeceasedTotal.IntegerValue)
	if err != nil {
		return nil, fmt.Errorf("Error converting string to int: %v", err)
	}
	currentlyHospitalizedTotal, err := strconv.Atoi(f.CurrentlyHospitalizedTotal.IntegerValue)
	if err != nil {
		return nil, fmt.Errorf("Error converting string to int: %v", err)
	}

	data := TotalsData{
		Date:                       f.Date.StringValue,
		TestsTotal:                 testsTotal,
		ConfirmedCasesTotal:        confirmedCasesTotal,
		ActiveCasesTotal:           activeCasesTotal,
		CuredTotal:                 curedTotal,
		DeceasedTotal:              deceasedTotal,
		CurrentlyHospitalizedTotal: currentlyHospitalizedTotal,
	}

	return &data, nil
}
