package functions

import (
	"net/http"

	"github.com/covid19cz/erouska-backend/internal/registerehrid"
)

// RegisterEhrid Registration handler.
func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	registerehrid.RegisterEhrid(w, r)
}
