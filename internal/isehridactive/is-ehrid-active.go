package isehridactive

import (
	"encoding/json"
	ers "errors"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

type queryRequest struct {
	Ehrid string `json:"ehrid" validate:"required"`
}

type queryResponse struct {
	Active bool `json:"active"`
}

//IsEhridActive Queries if specified eHrid is active
func IsEhridActive(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request queryRequest

	err := httputils.DecodeJSONBody(w, r, &request)
	if err != nil {
		var mr *errors.MalformedRequestError
		if ers.As(err, &mr) {
			logger.Debugf("Cannot handle isEhridActive request: %+v", mr.Msg)
			http.Error(w, mr.Msg, mr.Status)
		} else {
			logger.Debugf("Cannot handle isEhridActive request due to unknown error: %+v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Debugf("Handling isEhridActive request: %+v", request)

	_, err = client.Doc(constants.CollectionRegistrations, request.Ehrid).Get(ctx)

	var active bool

	if err != nil {
		if status.Code(err) != codes.NotFound {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		active = false
	} else {
		active = true
	}

	response := queryResponse{Active: active}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		response := fmt.Sprintf("Error: %v", err)
		http.Error(w, response, http.StatusInternalServerError)
		return
	}
}
