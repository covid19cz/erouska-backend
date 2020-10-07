package registerehrid

import (
	"context"
	"firebase.google.com/go/db"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	"github.com/covid19cz/erouska-backend/internal/realtimedb"
	"github.com/covid19cz/erouska-backend/internal/utils"
)

// Aftermath handler
func Aftermath(ctx context.Context, m pubsub.Message) error {
	logger := logging.FromContext(ctx)

	var payload AftermathPayload

	decodeErr := pubsub.DecodeJSONEvent(m, &payload)
	if decodeErr != nil {
		return fmt.Errorf("Error while parsing event payload: %v", decodeErr)
	}

	logger.Debugf("Doing user registration aftermath for eHrid '%s'!", payload.Ehrid)

	client := realtimedb.Client{}

	var date = utils.GetTimeNow().Format("20060102")

	// update daily counter
	err := updateCounter(ctx, client, constants.DbUserCountersPrefix+date)
	if err != nil {
		logger.Warnf("Cannot handle register user aftermath due to unknown error: %+v", err.Error())
		return err
	}

	// update total counter
	err = updateCounter(ctx, client, constants.DbUserCountersPrefix+"total")
	if err != nil {
		logger.Warnf("Cannot handle register user aftermath due to unknown error: %+v", err.Error())
		return err
	}

	logger.Debugf("Register user aftermath done")

	// Everything done!

	return nil
}

func updateCounter(ctx context.Context, client realtimedb.Client, key string) error {
	logger := logging.FromContext(ctx)

	return client.RunTransaction(ctx, key, func(tn db.TransactionNode) (interface{}, error) {
		var state structs.UserCounter

		if err := tn.Unmarshal(&state); err != nil {
			return nil, err
		}

		logger.Debugf("Found counter state, key %v: %+v", key, state)

		state.UsersCount++

		logger.Debugf("Saving updated counter state, key %v: %+v", key, state)

		return state, nil
	})
}
