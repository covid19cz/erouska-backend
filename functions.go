package functions

import (
	"github.com/covid19cz/erouska-backend/internal/isehridactive"
	"github.com/covid19cz/erouska-backend/internal/registerehrid"

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
