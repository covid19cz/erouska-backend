package registerehrid

import (
	"context"
	"testing"

	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/store"

	"github.com/google/go-cmp/cmp"
)

func TestRegister(t *testing.T) {

	ctx := context.Background()

	store := store.MockClient{}

	mockGenEhrid := func() string { return "eABCDEF123" }

	tables := []struct {
		x structs.Registration
		y func() string
	}{
		{
			structs.Registration{
				Platform:        "ios",
				PlatformVersion: "13.5.1",
				Manufacturer:    "Apple",
				Model:           "iPhone 8",
				Locale:          "cs_CZ",
			},
			mockGenEhrid,
		},
		{
			structs.Registration{
				Platform:        "android",
				PlatformVersion: "10.2",
				Manufacturer:    "Samsung",
				Model:           "Yololo",
				Locale:          "en_US",
			},
			mockGenEhrid,
		},
	}

	for _, table := range tables {
		ehrid, err := register(ctx, store, table.y, table.x)

		diff := cmp.Diff(ehrid, table.y())
		if diff != "" {
			t.Fatalf("register mismatch (-want +got):\n%v", diff)
		}
		if err != nil {
			t.Fatalf("register no error expected")
		}
	}
}
