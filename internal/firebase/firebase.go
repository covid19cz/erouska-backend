package firebase

import (
	"cloud.google.com/go/firestore"
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"log"
	"os"
)

//FirebaseDbClient -_-
var FirebaseDbClient *db.Client

//FirestoreClient -_-
var FirestoreClient *firestore.Client

func init() {
	ctx := context.Background()

	firebaseURL := constants.FirebaseURL
	url, exists := os.LookupEnv("FIREBASE_URL")
	if exists {
		firebaseURL = url
	}

	if firebaseURL == "NOOP" {
		log.Printf("Mocking Firebase")
		return
	}

	conf := &firebase.Config{
		DatabaseURL: firebaseURL,
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
