package http

import (
	"encoding/json"
	ers "errors"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	"github.com/golang/gddo/httputil/header"
	"io"
	"net/http"
	"strings"
)

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
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case ers.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}

		case ers.Is(err, io.ErrUnexpectedEOF):
			msg := "Request body contains badly-formed JSON"
			return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}

		case ers.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}

		case ers.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &errors.MalformedRequestError{Status: http.StatusRequestEntityTooLarge, Msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}
	}

	err = utils.Validate.Struct(dst)
	if err != nil {
		msg := fmt.Sprintf("Validation of the request has failed: %v", err.Error())
		return &errors.MalformedRequestError{Status: http.StatusBadRequest, Msg: msg}
	}

	return nil
}
