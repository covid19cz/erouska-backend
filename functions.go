package functions

import (
	"net/http"

	"github.com/covid19cz/erouska-backend/internal/register-ehrid"
)

func RegisterEhrid(w http.ResponseWriter, r *http.Request) {
	register_ehrid.RegisterEhrid(w, r)
}
