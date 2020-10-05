package secrets

import (
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/logging"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"log"
	"os"
)

//SecretsManagerClient -_-
var SecretsManagerClient *secretmanager.Client
var projectID string

func init() {
	ctx := context.Background()

	projectID = constants.ProjectID
	id, exists := os.LookupEnv("PROJECT_ID")
	if exists {
		projectID = id
	}

	if projectID == "NOOP" {
		log.Printf("Mocking Secrets Manager")
		return
	}

	var err error
	SecretsManagerClient, err = secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("secretmanager.NewClient: %v", err)
	}
}

//Manager is an abstraction over PubSub
type Manager interface {
	Get(name string) ([]byte, error)
}

//Client Real Secrets Manager client.
type Client struct{}

//Get Gets value of specified secret.
func (c Client) Get(name string) ([]byte, error) {
	ctx := context.Background()
	var logger = logging.FromContext(ctx)

	logger.Debugf("Accessing secret '%v'", name)

	var req = secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%v/secrets/%v/versions/latest", projectID, name),
	}

	secret, err := SecretsManagerClient.AccessSecretVersion(ctx, &req)
	if err != nil {
		log.Fatalf("Failed to get secret value: %v", err)
		return nil, err
	}

	logger.Debugf("Got secret '%v': %+v", name, secret)

	return secret.GetPayload().GetData(), nil
}

//MockClient NOOP Secrets Manager client.
type MockClient struct{}

//Get Gets value of specified secret.
func (c MockClient) Get(name string) ([]byte, error) {
	return []byte("mock42"), nil
}
