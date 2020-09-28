package pubsub

import (
	"bytes"
	"context"
	"encoding/json"
	ers "errors"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	rpccode "google.golang.org/genproto/googleapis/rpc/code"
	"io"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/covid19cz/erouska-backend/internal/constants"
)

//PubSubClient -_-
var PubSubClient *pubsub.Client

// Message is the payload of a Pub/Sub event.
type Message struct {
	Data []byte `json:"data"`
}

func init() {
	ctx := context.Background()

	projectID := constants.ProjectID
	id, exists := os.LookupEnv("PROJECT_ID")
	if exists {
		projectID = id
	}

	if projectID == "NOOP" {
		log.Printf("Mocking PubSub")
		return
	}

	var err error
	PubSubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("pubsub.NewClient: %v", err)
	}
}

//EventPublisher is an abstraction over PubSub
type EventPublisher interface {
	Publish(topic string, msg interface{}) error
}

//Client Real PubSub client.
type Client struct{}

//Publish Publish message to some topic.
func (c Client) Publish(topic string, msg interface{}) error {
	var t = PubSubClient.Topic(topic)
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	result := t.Publish(context.Background(), &pubsub.Message{Data: payload})

	// The Get method blocks until a server-generated ID or
	// an error is returned for the published message.
	_, err = result.Get(context.Background())
	return err
}

//MockClient NOOP PubSub client.
type MockClient struct{}

//Publish Publish message to some topic.
func (c MockClient) Publish(topic string, msg interface{}) error {
	return nil
}

//DecodeJSONEvent Decodes and validates PubSub message into given interface.
func DecodeJSONEvent(m Message, dst interface{}) errors.ErouskaError {
	dec := json.NewDecoder(bytes.NewBuffer(m.Data))
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
			msg := "Body must not be empty"
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &errors.MalformedRequestError{Status: rpccode.Code_INVALID_ARGUMENT, Msg: msg}

		default:
			return &errors.UnknownError{Msg: err.Error()}
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
