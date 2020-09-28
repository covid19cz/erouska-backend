package registernotification

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//AfterMath Handler
func AfterMath(ctx context.Context, m pubsub.Message) error {
	logger := logging.FromContext(ctx)

	var payload AftermathPayload

	decodeErr := pubsub.DecodeJSONEvent(m, &payload)
	if decodeErr != nil {
		return fmt.Errorf("Error while parsing event payload: %v", decodeErr)
	}

	logger.Debugf("Doing registration aftermath for eHrid '%s'!", payload.Ehrid)

	client := store.Client{}

	var date = utils.GetTimeNow().Format("20060102")

	doc := client.Doc(constants.CollectionDailyNotificationAttemptsEhrid, payload.Ehrid)

	var finalDailyCount int

	err := client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			logger.Debugf("Saving default daily state")
			finalDailyCount = 1
			return tx.Set(doc, map[string]int{date: 1})
		}
		// record for eHrid found, let's update it

		var states map[string]int
		err = rec.DataTo(&states)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}

		logger.Debugf("Found daily states: %+v", states)

		// Step 1. Increase daily state
		dailyCount, exists := states[date]
		if !exists {
			dailyCount = 0
		}

		finalDailyCount = dailyCount + 1
		states[date] = finalDailyCount

		logger.Debugf("Saving updated daily states for eHRID %v: %+v", payload.Ehrid, states)

		return tx.Set(doc, states)
	})

	if err != nil {
		logger.Warnf("Cannot handle register notification aftermath due to unknown error: %+v", err.Error())
		return err
	}

	logger.Debugf("Daily count for %v: %v", payload.Ehrid, finalDailyCount)

	// Step 2. Check if daily state is not too high

	var thresholdsOK = finalDailyCount == 1

	logger.Debugf("Thresholds ok: %v", thresholdsOK)

	// Step 3. Possibly increase notificationsCount

	if thresholdsOK {
		// update daily counter
		err = updateCounter(ctx, client, date)

		if err != nil {
			logger.Warnf("Cannot handle register notification aftermath due to unknown error: %+v", err.Error())
			return err
		}

		// update total counter
		err = updateCounter(ctx, client, "total")

		if err != nil {
			logger.Warnf("Cannot handle register notification aftermath due to unknown error: %+v", err.Error())
			return err
		}
	}

	logger.Debugf("Register notification aftermath done")

	// Everything done!

	return nil
}

func updateCounter(ctx context.Context, client store.Client, key string) error {
	logger := logging.FromContext(ctx)

	doc := client.Doc(constants.CollectionNotificationCounters, key)

	return client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			logger.Debugf("Saving default global daily state, key %v", key)
			return tx.Set(doc, structs.NotificationCounter{NotificationsCount: 1})
		}

		var data structs.NotificationCounter
		err = rec.DataTo(&data)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}
		logger.Debugf("Found global daily states, key %v: %+v", key, data)

		data.NotificationsCount++

		logger.Debugf("Saving updated global daily state, key %v: %+v", key, data)

		return tx.Set(doc, data)
	})
}
