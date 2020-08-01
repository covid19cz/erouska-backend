package firebase

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"github.com/covid19cz/erouska-backend/internal/constants"
)

//FirebaseDbClient -_-
var FirebaseDbClient *db.Client

//FirestoreClient -_-
var FirestoreClient *firestore.Client

func init() {
	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL: constants.FirebaseURL,
	}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalf("firebase.NewApp: %v", err)
	}
	FirebaseDbClient, err = app.Database(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}
	FirestoreClient, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}
}
