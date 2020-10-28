package wakeup

import (
	"context"
	fbmessaging "firebase.google.com/go/messaging"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/messaging"
	"net/http"
	"time"
)

const topicName = "budicek"

//SendWakeUpSignal Sends wake-up signal to devices
func SendWakeUpSignal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named("wake-up.SendWakeUpSignal")

	if err := wakeUp(ctx, messaging.Client{}); err != nil {
		msg := fmt.Sprintf("Could not send wake-up signal: %v", err)
		logger.Error(msg)
		http.Error(w, msg, 500)
		return
	}

	//ok
}

func wakeUp(ctx context.Context, msgClient messaging.PushSender) error {
	logger := logging.FromContext(ctx).Named("wake-up.wakeUp")

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
