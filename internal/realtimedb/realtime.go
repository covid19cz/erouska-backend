package realtimedb

import (
	"context"
	"firebase.google.com/go/db"

	"github.com/covid19cz/erouska-backend/internal/firebase"
)

// RealtimeDB is a Realtime DB abstraction layer interface
type RealtimeDB interface {
	NewRef(path string) *db.Ref
	RunTransaction(ctx context.Context, path string, f db.UpdateFn) error
}

// Client to interact with storage API
type Client struct{}

// NewRef returns a reference to path in Realtime DB
func (i Client) NewRef(path string) *db.Ref {
	client := firebase.FirebaseDbClient
	return client.NewRef(path)
}

// RunTransaction runs f in a transaction at given path in Realtime DB
func (i Client) RunTransaction(ctx context.Context, path string, f db.UpdateFn) (err error) {
	client := firebase.FirebaseDbClient

	return client.NewRef(path).Transaction(ctx, f)
}

// MockClient mocks storage client functionality for unit tests
type MockClient struct{}

// NewRef returns a reference to path in Realtime DB (it's a mock!)
func (i MockClient) NewRef(path string) *db.Ref {

	ret := db.Ref{
		Key:  "",
		Path: path,
	}

	return &ret

}

// RunTransaction runs f in a transaction (but not, because it's a mock)
func (i MockClient) RunTransaction(ctx context.Context, path string, f db.UpdateFn) (err error) {
	return nil
}
