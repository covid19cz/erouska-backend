package http

import (
	"bytes"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	"github.com/stretchr/testify/assert"
	rpccode "google.golang.org/genproto/googleapis/rpc/code"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testRequest struct {
	Field1 string `json:"field1" validate:"required"`
	Field2 int    `json:"field2" validate:"required"`
}

type testResponse struct {
	Result string `json:"result"`
}

/* DecodeJSONBody: */

func TestDecodeJSONBodyOk(t *testing.T) {
	var body = bytes.NewBufferString(`{"data": {"field1": "ahoj", "field2": 42}}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	var request testRequest

	err = DecodeJSONBody(rr, req, &request)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, request, testRequest{
		Field1: "ahoj",
		Field2: 42,
	})

	assert.Equal(t, 0, rr.Body.Len())
	assert.False(t, rr.Flushed)
}

func TestDecodeJSONBodyMissingContentType(t *testing.T) {
	var body = bytes.NewBufferString(`{"data": {"field1": "ahoj", "field2": 42}}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}

	var request testRequest

	err = DecodeJSONBody(httptest.NewRecorder(), req, &request)

	if err == nil {
		t.Fatal("Must not end well")
	}

	assert.Equal(t, err, &errors.MalformedRequestError{
		Status: 415,
		Msg:    "Content-Type header is not application/json",
	})
}

func TestDecodeJSONBodyBadContentType(t *testing.T) {
	var body = bytes.NewBufferString(`{"data": {"field1": "ahoj", "field2": 42}}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/protobuf")

	var request testRequest

	err = DecodeJSONBody(httptest.NewRecorder(), req, &request)

	if err == nil {
		t.Fatal("Must not end well")
	}

	assert.Equal(t, err, &errors.MalformedRequestError{
		Status: 415,
		Msg:    "Content-Type header is not application/json",
	})
}

func TestDecodeJSONBodyUnwrapped(t *testing.T) {
	var body = bytes.NewBufferString(` {"field1": "ahoj", "field2": 42}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	var request testRequest

	err = DecodeJSONBody(httptest.NewRecorder(), req, &request)

	if err == nil {
		t.Fatal("Must not end well")
	}

	assert.Equal(t, err, &errors.MalformedRequestError{
		Status: rpccode.Code_INVALID_ARGUMENT,
		Msg:    "Request body must be wrapped in 'data' field",
	})
}

func TestDecodeJSONBodyMissingField(t *testing.T) {
	var body = bytes.NewBufferString(`{"data": {"field1": "ahoj"}}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	var request testRequest

	err = DecodeJSONBody(httptest.NewRecorder(), req, &request)

	if err == nil {
		t.Fatal("Must not end well")
	}

	assert.Equal(t, err, &errors.MalformedRequestError{
		Status: rpccode.Code_INVALID_ARGUMENT,
		Msg:    "Validation of the request has failed: Key: 'testRequest.Field2' Error:Field validation for 'Field2' failed on the 'required' tag",
	})
}

/* DecodeJSONOrReportError */

func TestDecodeJSONOrReportErrorOk(t *testing.T) {
	var body = bytes.NewBufferString(`{"data": {"field1": "ahoj", "field2": 42}}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	var request testRequest

	assert.True(t, DecodeJSONOrReportError(rr, req, &request))

	assert.Equal(t, request, testRequest{
		Field1: "ahoj",
		Field2: 42,
	})

	assert.Equal(t, 0, rr.Body.Len())
	assert.False(t, rr.Flushed)
}

func TestDecodeJSONOrReportErrorMissingField(t *testing.T) {
	var body = bytes.NewBufferString(`{"data": {"field1": "ahoj"}}`)

	req, err := http.NewRequest("GET", "/Url", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	var request testRequest

	assert.False(t, DecodeJSONOrReportError(rr, req, &request))
	assert.Equal(t, `{"error":{"status":3,"message":"Validation of the request has failed: Key: 'testRequest.Field2' Error:Field validation for 'Field2' failed on the 'required' tag"}}`, rr.Body.String())
}

func TestSendResponse(t *testing.T) {
	req, err := http.NewRequest("GET", "/Url", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	SendResponse(rr, req, testResponse{Result: "42"})

	var response = rr.Result()

	assert.Equal(t, `200 OK`, response.Status)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, `{"data":{"result":"42"}}`, rr.Body.String())
}

func TestSendEmptyResponse(t *testing.T) {
	req, err := http.NewRequest("GET", "/Url", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	SendEmptyResponse(rr, req)

	var response = rr.Result()

	assert.Equal(t, `200 OK`, response.Status)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, `{"data":{}}`, rr.Body.String())
}

func TestSendErrorResponseUnknownError(t *testing.T) {
	req, err := http.NewRequest("GET", "/Url", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	SendErrorResponse(rr, req, &errors.UnknownError{Msg: "ahoj"})

	var response = rr.Result()

	assert.Equal(t, `200 OK`, response.Status)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, `{"error":{"status":13,"message":"ahoj"}}`, rr.Body.String())
}

func TestSendErrorResponseBadRequestError(t *testing.T) {
	req, err := http.NewRequest("GET", "/Url", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	SendErrorResponse(rr, req, &errors.MalformedRequestError{Msg: "request is screwed"})

	var response = rr.Result()

	assert.Equal(t, `200 OK`, response.Status)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, `{"error":{"status":3,"message":"request is screwed"}}`, rr.Body.String())
}

func TestSendErrorResponseNotFoundError(t *testing.T) {
	req, err := http.NewRequest("GET", "/Url", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	SendErrorResponse(rr, req, &errors.NotFoundError{Msg: "entity not found"})

	var response = rr.Result()

	assert.Equal(t, `200 OK`, response.Status)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, `{"error":{"status":5,"message":"entity not found"}}`, rr.Body.String())
}

func TestSendErrorResponseGenericError(t *testing.T) {
	req, err := http.NewRequest("GET", "/Url", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	SendErrorResponse(rr, req, fmt.Errorf("Error while doing something"))

	var response = rr.Result()

	assert.Equal(t, `200 OK`, response.Status)
	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
	assert.Equal(t, `{"error":{"status":13,"message":"Error while doing something"}}`, rr.Body.String())
}
