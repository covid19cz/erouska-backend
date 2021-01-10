package registernotification

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/api/iterator"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/covid19cz/erouska-backend/internal/auth"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/covid19cz/erouska-backend/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//AftermathPayload Struct holding aftermath input data.
type AftermathPayload struct {
	Ehrid string `json:"ehrid" validate:"required"`
}

//RegisterNotification Handler
func RegisterNotification(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("RegisterNotification")
	authClient := auth.Client{}
	pubSubClient := pubsub.Client{}

	var request v1.RegisterNotificationRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	uid, err := authClient.AuthenticateToken(ctx, request.IDToken)
	if err != nil {
		logger.Debugf("Unverifiable token provided: %+v %+v", request.IDToken, err.Error())

		if strings.Contains(err.Error(), "ID token has expired") {
			expSince := parseExpirationTime(request.IDToken)
			now := time.Now().Unix()

			logger.Debugf("ID token is expired for %v s (since %v, now %v), request headers %+v", now-expSince, expSince, now, r.Header)
		}

		httputils.SendErrorResponse(w, r, &errors.UnauthenticatedError{Msg: "Invalid token"})
		return
	}

	logger.Debugf("Handling RegisterNotification request: UID %v %+v", uid, request)

	isEhrid, _ := regexp.MatchString(utils.EhridRegex, uid)

	if !isEhrid {
		logger.Infof("Provided ID is not eHrid: %v", uid)
		err = handleForFUID(ctx, uid)
	} else {
		err = handleForEhrid(ctx, uid)
	}

	if err != nil {
		logger.Errorf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	aftermathPayload := AftermathPayload{Ehrid: uid}

	topicName := constants.TopicRegisterNotification
	logger.Infof("Publishing event to %v: %+v", topicName, aftermathPayload)
	err = pubSubClient.Publish(topicName, aftermathPayload)
	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, err)
		return
	}

	httputils.SendEmptyResponse(w, r)
}

func handleForEhrid(ctx context.Context, ehrid string) error {
	logger := logging.FromContext(ctx).Named("register-notification.handleForEhrid")
	storeClient := store.Client{}

	doc := storeClient.Doc(constants.CollectionRegistrations, ehrid)

	return storeClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		rec, err := tx.Get(doc)

		if err != nil {
			if status.Code(err) != codes.NotFound {
				return fmt.Errorf("Error while querying Firestore: %v", err)
			}
			// not found:

			return &errors.NotFoundError{Msg: fmt.Sprintf("Could not find registration for %v: %v", ehrid, err)}
		}

		var registration structs.Registration
		err = rec.DataTo(&registration)
		if err != nil {
			return fmt.Errorf("Error while querying Firestore: %v", err)
		}
		logger.Debugf("Found registration: %+v", registration)

		registration.LastNotificationStatus = "sent"
		registration.LastNotificationUpdatedAt = utils.GetTimeNow().Unix()

		logger.Debugf("Saving updated notification state: %+v", registration)

		return tx.Set(doc, registration)
	})
}

func handleForFUID(ctx context.Context, fuid string) error {
	logger := logging.FromContext(ctx).Named("register-notification.handleForFUID")
	storeClient := store.Client{}

	logger.Debugf("Looking for FUID %v in collection %v", fuid, constants.CollectionRegistrationsV1)

	it := storeClient.Find(constants.CollectionRegistrationsV1, "fuid", fuid).Snapshots(ctx)

	resp, err := it.Next()
	if err == iterator.Done {
		return fmt.Errorf("Could not find record for FUID %v", fuid)
	}

	if err != nil {
		return err
	}

	if resp.Size == 0 {
		return fmt.Errorf("Could not find record for FUID %v", fuid)
	}

	logger.Debugf("Record for FUID %+v found", fuid)

	return nil
}

func parseExpirationTime(idToken string) int64 {
	token, _ := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		return []byte("hello-world"), nil
	})

	// We certainly got validation error but we don't care. It's just about whether we are able to parse the token or not.

	if token == nil || token.Claims == nil {
		return 0
	}

	claims := token.Claims.(jwt.MapClaims)

	switch exp := claims["exp"].(type) {
	case float64:
		return int64(exp)
	case json.Number:
		v, _ := exp.Int64()
		return v
	default:
		logging.FromContext(context.Background()).Named("register-notification.parseExpirationTime").Warnf("Unknown type of exp claim: %+v", claims)
		return 0
	}
}
