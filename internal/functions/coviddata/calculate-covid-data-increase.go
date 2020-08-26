package coviddata

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"

	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FirestoreEvent is an event that triggers this functions
type FirestoreEvent struct {
	OldValue   FirestoreValue `json:"oldValue"`
	Value      FirestoreValue `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// FirestoreValue is the new and old keyval from the triggered event
type FirestoreValue struct {
	CreateTime time.Time        `json:"createTime"`
	Name       string           `json:"name"`
	UpdateTime time.Time        `json:"updateTime"`
	Fields     TotalsDataFields `json:"fields"`
}

// IncreaseData holds all the coviddata increases (diffs from previous day)
type IncreaseData struct {
	TestsIncrease                 int `json:"testsIncrease"  validate:"required"`
	ConfirmedCasesIncrease        int `json:"confirmedCasesIncrease"  validate:"required"`
	ActiveCasesIncrease           int `json:"activeCasesIncrease"  validate:"required"`
	CuredIncrease                 int `json:"curedIncrease"  validate:"required"`
	DeceasedIncrease              int `json:"deceasedIncrease"  validate:"required"`
	CurrentlyHospitalizedIncrease int `json:"currentlyHospitalizedIncrease"  validate:"required"`
}

// IncreaseDataFields is a wrapped around IncreaseData, from firestore request
type IncreaseDataFields struct {
	TestsIncrease                 structs.IntegerValue `json:"testsIncrease"  validate:"required"`
	ConfirmedCasesIncrease        structs.IntegerValue `json:"confirmedCasesIncrease"  validate:"required"`
	ActiveCasesIncrease           structs.IntegerValue `json:"activeCasesIncrease"  validate:"required"`
	CuredIncrease                 structs.IntegerValue `json:"curedIncrease"  validate:"required"`
	DeceasedIncrease              structs.IntegerValue `json:"deceasedIncrease"  validate:"required"`
	CurrentlyHospitalizedIncrease structs.IntegerValue `json:"currentlyHospitalizedIncrease"  validate:"required"`
}

// CalculateCovidDataIncrease handler.
func CalculateCovidDataIncrease(ctx context.Context, e FirestoreEvent) error {

	logger := logging.FromContext(ctx)
	client := store.Client{}

	logger.Infof("received firestore event: %+v", e)

	// This is the data that's in the database itself
	todayFields := e.Value.Fields
	today, err := todayFields.ToData()
	if err != nil {
		return fmt.Errorf("Error converting firestore data: %v", err)
	}

	date := todayFields.Date.StringValue

	if date == "" {
		date = utils.GetTimeNow().Format("20060102")
	}

	t, err := time.Parse("20060102", date)
	if err != nil {
		return fmt.Errorf("Error while parsing date: %v", err)
	}

	yesterdayDate := t.AddDate(0, 0, -1).Format("20060102")

	snap, err := client.Doc(constants.CollectionCovidDataTotal, yesterdayDate).Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &errors.NotFoundError{Msg: fmt.Sprintf("Could not find covid data for %v", yesterdayDate)}
		}

		return fmt.Errorf("Error while querying Firestore: %v", err)
	}

	logger.Infof("fetched firestore event: %+v", snap.Data())

	var yesterday TotalsData

	if err := snap.DataTo(&yesterday); err != nil {
		panic(fmt.Sprintf("could not parse input: %s", err))
	}

	logger.Infof("fetched data: %+v", yesterday)

	newData := IncreaseData{
		TestsIncrease:                 today.TestsTotal - yesterday.TestsTotal,
		ConfirmedCasesIncrease:        today.ConfirmedCasesTotal - yesterday.ConfirmedCasesTotal,
		ActiveCasesIncrease:           today.ActiveCasesTotal - yesterday.ActiveCasesTotal,
		CuredIncrease:                 today.CuredTotal - yesterday.CuredTotal,
		DeceasedIncrease:              today.DeceasedTotal - yesterday.DeceasedTotal,
		CurrentlyHospitalizedIncrease: today.CurrentlyHospitalizedTotal - yesterday.CurrentlyHospitalizedTotal,
	}

	doc := client.Doc(constants.CollectionCovidDataIncrease, date)

	err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		_, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}

			return tx.Set(doc, newData)
		}
		return nil
	})

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		return nil
	}

	logger.Infof("Succesfully written data to firestore: %+v", newData)

	return nil
}
