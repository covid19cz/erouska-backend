// Package functions provides a set of Cloud Function entrypoints.
package functions

import (
	"context"
	"net/http"

	"github.com/covid19cz/erouska-backend/internal/hello"
	"github.com/covid19cz/erouska-backend/internal/scorer"

	"github.com/covid19cz/erouska-backend/pkg/firestore"
)

// HelloHTTP is an HTTP Cloud Function with a request parameter.
func HelloHTTP(w http.ResponseWriter, r *http.Request) {

	hello.Hello(w, r)
}

// ScoreReview generates the scores for movie reviews and transactionally writes them to the
// Firebase Realtime Database.
func ScoreReview(ctx context.Context, e firestore.Event) error {

	return scorer.ScoreReview(ctx, e)
}
