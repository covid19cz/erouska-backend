// Package helloworld provides a set of Cloud Functions samples.
package helloworld

import (
	"net/http"

	server "github.com/covid19cz/erouska-backend/pkg/httpserver"
)

// HelloHTTP is an HTTP Cloud Function with a request parameter.
func HelloHTTP(w http.ResponseWriter, r *http.Request) {

	srv, err := server.NewHandler(nil, nil)

	if err != nil {
		return
	}

	srv.ServeHTTP(w, r)

}
