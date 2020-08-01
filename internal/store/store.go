package store

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/covid19cz/erouska-backend/internal/firebase"
)

// Storer is a storage abstraction layer interface
type Storer interface {
	Doc(string, string) *firestore.DocumentRef
	RunTransaction(context.Context, func(context.Context, *firestore.Transaction) error, ...firestore.TransactionOption) error
}

// Client to interact with storage API
type Client struct{}

// Doc returns a DocumentRef that refers to the document in the collection with the given identifier.
func (i Client) Doc(collectionName string, path string) *firestore.DocumentRef {
	client := firebase.FirestoreClient
	return client.Collection(collectionName).Doc(path)
}

// RunTransaction runs f in a transaction.
func (i Client) RunTransaction(ctx context.Context, f func(context.Context, *firestore.Transaction) error, opts ...firestore.TransactionOption) (err error) {
	client := firebase.FirestoreClient
	return client.RunTransaction(ctx, f, opts...)
}

// MockClient mocks storage client functionaly for unit tests
type MockClient struct{}

// Doc returns a DocumentRef that refers to the document in the collection with the given identifier.
func (i MockClient) Doc(_ string, path string) *firestore.DocumentRef {

	ret := firestore.DocumentRef{
		Parent: nil,
		Path:   path,
		//shortPath: "coll-1/doc-1",
		ID: "abc",
	}

	return &ret

}

// RunTransaction runs f in a transaction.
func (i MockClient) RunTransaction(ctx context.Context, f func(context.Context, *firestore.Transaction) error, opts ...firestore.TransactionOption) (err error) {
	return nil
}
