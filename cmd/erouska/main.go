package main

import (
	"fmt"
	"github.com/covid19cz/erouska-backend/pkg/logging"
	"net/http"

	"github.com/sethvargo/go-signalcontext"

	server "github.com/covid19cz/erouska-backend/pkg/httpserver"
)

func main() {

	ctx, done := signalcontext.OnInterrupt()
	defer done()

	logger := logging.FromContext(ctx)

	var config server.Config = server.Config{Port: "8081"}

	handler, err := server.NewHandler(ctx, &config)

	if err != nil {
		logger.Errorf("publish.NewHandler: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	srv, err := server.NewServer(ctx, &config)
	if err != nil {
		fmt.Errorf("server.New: %w", err)
	}
	logger.Infof("listening on :%s", config.Port)

	if err := srv.ServeHTTPHandler(ctx, mux); err != nil {
		logger := logging.FromContext(ctx)
		logger.Fatal(err)
	}

}
