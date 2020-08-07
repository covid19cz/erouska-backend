package http

import (
	"bytes"
	"encoding/json"
	ers "errors"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	"github.com/golang/gddo/httputil/header"
	rpccode "google.golang.org/genproto/googleapis/rpc/code"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"io"
	"net/http"
	"strings"
)

// DecodeJSONBody Decode request body from JSON to struct
// copied from https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &errors.MalformedRequestError{Status: http.StatusUnsupportedMediaType, Msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)

	if r.Header.Get("X-Erouska-Wrapped") != "false" {
		wrappedRequest := make(map[string]json.RawMessage)

		err := dec.Decode(&wrappedRequest)
		_, found := wrappedRequest["data"]
		if err != nil || !found {
			msg := "Request body must be wrapped in 'data' field"
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}
		}

		dec = json.NewDecoder(bytes.NewBuffer(wrappedRequest["data"]))
	}

	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case ers.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		case ers.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		case ers.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		case ers.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}
	}

	err = utils.Validate.Struct(dst)
	if err != nil {
		msg := fmt.Sprintf("Validation of the request has failed: %v", err.Error())
		return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}
	}

	return nil
}

//DecodeJSONOrReportError Decodes request JSON and writes possible errors to the ResponseWriter. Return bool - if the request was decoded successfully.
func DecodeJSONOrReportError(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)

	err := DecodeJSONBody(w, r, dst)
	if err != nil {
		var mr *errors.MalformedRequestError
		if ers.As(err, &mr) {
			SendErrorResponse(w, r, mr.Status, mr.Msg)
		} else {
			logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
			SendErrorResponse(w, r, rpccode.Code_INTERNAL, "Unknown error")
		}
		return false
	}

	return true
}

//SendResponse Marshals response into JSON and sends it to the client.
func SendResponse(w http.ResponseWriter, r *http.Request, response interface{}) {
	logger := logging.FromContext(r.Context())

	var wrapped = r.Header.Get("X-Erouska-Wrapped") != "false"

	var responseBytes []byte

	if wrapped {
		r := map[string]interface{}{"data": response}
		js, err := json.Marshal(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		responseBytes = js
	} else {
		js, err := json.Marshal(response)
		if err != nil {
			logger.Warnf("Unknown error: %+v", err)
			http.Error(w, "Unknown error", http.StatusInternalServerError)
			return
		}
		responseBytes = js
	}

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(responseBytes)
	if err != nil {
		logger.Warnf("Unknown error: %+v", err)
		http.Error(w, "Unknown error", http.StatusInternalServerError)
		return
	}

	logger.Debugf("Returning response: %+v", response)
}

//SendEmptyResponse Marshals empty response into JSON and sends it to the client.
func SendEmptyResponse(w http.ResponseWriter, r *http.Request) {
	SendResponse(w, r, struct{}{})
}

//SendErrorResponse Sends error response as required by Firebase SDK.
func SendErrorResponse(w http.ResponseWriter, r *http.Request, error rpccode.Code, message string) {
	logger := logging.FromContext(r.Context())

	errorCode := rpccode.Code_value[error.String()]

	status := rpcstatus.Status{
		Code:    errorCode,
		Message: message,
	}

	// This is ALWAYS wrapped, ignoring the possible `X-Erouska-Wrapped` header
	wrappedStatus := map[string]interface{}{"error": status}
	js, err := json.Marshal(wrappedStatus)
	if err != nil {
		logger.Warnf("Unknown error: %+v", err)
		http.Error(w, "Unknown error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		logger.Warnf("Unknown error: %+v", err)
		http.Error(w, "Unknown error", http.StatusInternalServerError)
		return
	}

	logger.Debugf("Returning error response: %+v", status)
}
