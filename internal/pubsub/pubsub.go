package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"log"
	"os"
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
	PubSubClient, err = pubsub.NewClient(ctx, constants.ProjectID)
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

	t.Publish(context.Background(), &pubsub.Message{Data: payload})

	return nil
}

//MockClient NOOP PubSub client.
type MockClient struct{}

//Publish Publish message to some topic.
func (c MockClient) Publish(topic string, msg interface{}) error {
	return nil
}
