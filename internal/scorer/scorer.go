package scorer

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"

	fst "github.com/covid19cz/erouska-backend/pkg/firestore"
)

var client *db.Client
var clientF *firestore.Client

func init() {
	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL: "https://playground-283917.firebaseio.com",
	}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalf("firebase.NewApp: %v", err)
	}
	client, err = app.Database(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}
	clientF, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}
}

// ScoreReview generates the scores for movie reviews and transactionally writes them to the
// Firebase Realtime Database.
func ScoreReview(ctx context.Context, e fst.Event) error {
	review := e.Value.Fields
	reviewScore := score(review.Text.Value)
	docs, err := clientF.Collection("Scores").Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("GetAll: %v", err)
	}
	var mapValue = map[string]string{"value": fmt.Sprintf("%v", reviewScore)}
	for _, doc := range docs {
		value, err := doc.DataAt("value")
		if err != nil {
			return fmt.Errorf("DataAt: %v", err)
		}
		mapValue[doc.Ref.Path] = fmt.Sprintf("%v", value)
	}
	_, err = clientF.Collection("Scores").Doc(review.Author.Value).Set(ctx, mapValue)
	if err != nil {
		return fmt.Errorf("Set: %v", err)
	}
	return nil
	//ref := client.NewRef("scores").Child(review.Author.Value)
	//updateTxn := func(node db.TransactionNode) (interface{}, error) {
	//	var currentScore int
	//	if err := node.Unmarshal(&currentScore); err != nil {
	//		return nil, err
	//	}
	//	return currentScore + reviewScore, nil
	//}
	//return ref.Transaction(ctx, updateTxn)
}

// score computes the score for a review text.
//
// This is currently just the length of the text.
func score(text string) int {
	return len(text)
}
