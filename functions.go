package functions

import (
	"github.com/covid19cz/erouska-backend/internal/functions/checkattemptsthresholds"
	"github.com/covid19cz/erouska-backend/internal/functions/increaseehridattemptscount"
	"github.com/covid19cz/erouska-backend/internal/functions/increaseipattemptscount"
	"github.com/covid19cz/erouska-backend/internal/functions/increasenotificationscounter"
	"github.com/covid19cz/erouska-backend/internal/functions/isehridactive"
	"github.com/covid19cz/erouska-backend/internal/functions/registerehrid"

	"net/http"
)

// RegisterEhrid Registration handler.
func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	registerehrid.RegisterEhrid(w, r)
}

// IsEhridActive IsEhridActive handler.
func IsEhridActive(w http.ResponseWriter, r *http.Request) {
	isehridactive.IsEhridActive(w, r)
}

// IncreaseEhridAttemptsCount IncreaseEhridAttemptsCount handler.
func IncreaseEhridAttemptsCount(w http.ResponseWriter, r *http.Request) {
	increaseehridattemptscount.IncreaseEhridAttemptsCount(w, r)
}

// IncreaseIPAttemptsCount IncreaseIPAttemptsCount handler.
func IncreaseIPAttemptsCount(w http.ResponseWriter, r *http.Request) {
	increaseipattemptscount.IncreaseIPAttemptsCount(w, r)
}

// CheckAttemptsThresholds CheckAttemptsThresholds handler.
func CheckAttemptsThresholds(w http.ResponseWriter, r *http.Request) {
	checkattemptsthresholds.CheckAttemptsThresholds(w, r)
}

// IncreaseNotificationsCounter IncreaseNotificationsCounter handler.
func IncreaseNotificationsCounter(w http.ResponseWriter, r *http.Request) {
	increasenotificationscounter.IncreaseNotificationsCounter(w, r)
}
