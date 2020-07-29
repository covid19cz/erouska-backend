package scorer

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/pkg/firebase"
)

func ScoreReview(ctx context.Context, e fst.Event) error {
	review := e.Value.Fields
	reviewScore := score(review.Text.Value)
	docs, err := firebase.FirestoreClient.Collection("Scores").Documents(ctx).GetAll()
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
	_, err = firebase.FirestoreClient.Collection("Scores").Doc(review.Author.Value).Set(ctx, mapValue)
	if err != nil {
		return fmt.Errorf("Set: %v", err)
	}
	return nil
}

func score(text string) int {
	return len(text)
}
