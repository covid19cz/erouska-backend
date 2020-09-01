package isehridactive

import (
	"github.com/covid19cz/erouska-backend/internal/auth"
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
	IDToken string `json:"idToken" validate:"required"`
}

type queryResponse struct {
	Active bool `json:"active"`
}

//IsEhridActive Queries if specified eHrid is active
func IsEhridActive(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	storeClient := store.Client{}
	authClient := auth.Client{}

	var request queryRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	ehrid, err := authClient.AuthenticateToken(ctx, request.IDToken)
	if err != nil {
		logger.Debugf("Unverifiable token provided: %+v %+v", request.IDToken, err.Error())
		httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
		return
	}

	logger.Debugf("Handling isEhridActive request: %v %+v", ehrid, request)

	_, err = storeClient.Doc(constants.CollectionRegistrations, ehrid).Get(ctx)

	var active bool

	if err != nil {
		if status.Code(err) != codes.NotFound {
			logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
			httputils.SendErrorResponse(w, r, err)
			return
		}

		active = false
	} else {
		active = true
	}

	response := queryResponse{Active: active}

	httputils.SendResponse(w, r, response)
}
