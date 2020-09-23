package pubsub

import (
	"context"
	"encoding/json"
	"log"
	"os"

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
