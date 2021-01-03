package wakeup

import (
	"context"
	fbmessaging "firebase.google.com/go/messaging"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/messaging"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"net/http"
	"time"
)

const topicName = "budicek"

//SendWakeUpSignal Sends wake-up signal to devices
func SendWakeUpSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	secretClient := secrets.Client{}
	pushSender := messaging.Client{}

	httpStatus, httpBody := sendWakeUpSignalAuthenticated(ctx, r, secretClient, pushSender)

	http.Error(w, httpBody, httpStatus)
}

func sendWakeUpSignalAuthenticated(ctx context.Context, r *http.Request, secretClient secrets.Client, pushSender messaging.Client) (int, string) {
	logger := logging.FromContext(ctx).Named("wake-up.sendWakeUpSignalAuthenticated")

	// authentication

	apikey, err := secretClient.Get("manual-wakeup-apikey")
	if err != nil {
		logger.Warnf("Could not obtain api key: %v", err)
		return 500, "Could not obtain api key"
	}

	providedAPIKeys := r.URL.Query()["apikey"]
	if len(providedAPIKeys) != 1 || providedAPIKeys[0] != string(apikey) {
		return 401, "Bad api key"
	}

	// authenticated, go ahead

	if err := sendWakeUpSignal(ctx, pushSender); err != nil {
		msg := fmt.Sprintf("Could not send wake-up signal: %v", err)
		logger.Error(msg)
		return 500, msg
	}

	return 200, "ok"
}

func sendWakeUpSignal(ctx context.Context, msgClient messaging.PushSender) error {
	logger := logging.FromContext(ctx).Named("wake-up.sendWakeUpSignal")

	ttl, _ := time.ParseDuration("1d")

	message := fbmessaging.Message{
		Data: map[string]string{
			"downloadKeyExport": "true",
		},
		Topic: topicName,
		Android: &fbmessaging.AndroidConfig{
			Priority: "high",
			TTL:      &ttl,
		},
	}

	logger.Debugf("Sending wake-up signal to topic %v", topicName)

	return msgClient.Send(ctx, &message)
}
