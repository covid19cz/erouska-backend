package secrets

import (
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"log"
	"os"
)

//SecretsManagerClient -_-
var SecretsManagerClient *secretmanager.Client
var projectID string
var secretsPayloadLogging = false

func init() {
	ctx := context.Background()

	projectIDtmp, ok := os.LookupEnv("PROJECT_ID")
	if !ok {
		panic("PROJECT_ID env must be configured!")
	}

	if projectIDtmp == "NOOP" {
		log.Printf("Mocking Secrets Manager")
		return
	}

	projectID = projectIDtmp // Fuck you, Go!

	v, ok := os.LookupEnv("SECRETS_PAYLOAD_LOGGING")
	if ok && v == "true" {
		secretsPayloadLogging = true
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

	fullName := fmt.Sprintf("projects/%v/secrets/%v/versions/latest", projectID, name)

	logger.Debugf("Accessing secret '%v'", fullName)

	var req = secretmanagerpb.AccessSecretVersionRequest{
		Name: fullName,
	}

	secret, err := SecretsManagerClient.AccessSecretVersion(ctx, &req)
	if err != nil {
		log.Fatalf("Failed to get secret value: %v", err)
		return nil, err
	}

	if secretsPayloadLogging {
		logger.Debugf("Got secret '%v': %+v", name, secret)
	}

	return secret.GetPayload().GetData(), nil
}

//MockClient NOOP Secrets Manager client.
type MockClient struct{}

//Get Gets value of specified secret.
func (c MockClient) Get(name string) ([]byte, error) {
	return []byte("mock42"), nil
}
