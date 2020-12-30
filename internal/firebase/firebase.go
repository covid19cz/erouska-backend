package firebase

import (
	"cloud.google.com/go/firestore"
	"context"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/db"
	"firebase.google.com/go/messaging"
	"log"
	"os"
)

//FirebaseDbClient -_-
var FirebaseDbClient *db.Client

//FirestoreClient -_-
var FirestoreClient *firestore.Client

//FirebaseAuth -_-
var FirebaseAuth *auth.Client

//FirebaseMessaging -_-
var FirebaseMessaging *messaging.Client

func init() {
	ctx := context.Background()

	projectID, ok := os.LookupEnv("PROJECT_ID")
	if !ok {
		panic("PROJECT_ID env must be configured!")
	}

	firebaseDbURL := "https://" + projectID + ".firebaseio.com/"
	url, exists := os.LookupEnv("FIREBASE_URL")
	if exists {
		firebaseDbURL = url
	}

	if firebaseDbURL == "NOOP" {
		log.Printf("Mocking Firebase")
		return
	}

	conf := &firebase.Config{
		DatabaseURL: firebaseDbURL,
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

	FirebaseAuth, err = app.Auth(ctx)
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	FirebaseMessaging, err = app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}
}
