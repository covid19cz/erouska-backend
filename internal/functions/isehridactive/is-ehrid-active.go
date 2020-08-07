package isehridactive

import (
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	rpccode "google.golang.org/genproto/googleapis/rpc/code"
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

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	logger.Debugf("Handling isEhridActive request: %+v", request)

	_, err := client.Doc(constants.CollectionRegistrations, request.Ehrid).Get(ctx)

	var active bool

	if err != nil {
		if status.Code(err) != codes.NotFound {
			logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
			httputils.SendErrorResponse(w, r, rpccode.Code_INTERNAL, "Unknown error")
			return
		}

		active = false
	} else {
		active = true
	}

	response := queryResponse{Active: active}

	httputils.SendResponse(w, r, response)
}
