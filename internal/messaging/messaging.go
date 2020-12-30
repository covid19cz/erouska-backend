package messaging

import (
	"context"
	"firebase.google.com/go/messaging"
	"github.com/covid19cz/erouska-backend/internal/firebase"
)

//PushSender Interface for FB messaging client
type PushSender interface {
	Send(ctx context.Context, msg *messaging.Message) error
}

//Client Real implementation of FB messaging client
type Client struct{}

//Send Sends the message
func (c Client) Send(ctx context.Context, msg *messaging.Message) error {
	_, err := firebase.FirebaseMessaging.Send(ctx, msg)
	return err
}
