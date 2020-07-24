// Package httpserver provides a set of Cloud Functions samples.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/covid19cz/erouska-backend/internal/hello"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"go.opencensus.io/plugin/ochttp"
)

// Config holds server config
type Config struct {
	Port string
}

type handler struct {
	config *Config
}

// Server provides a gracefully-stoppable http server implementation. It is safe
// for concurrent use in goroutines.
type Server struct {
	ip       string
	port     string
	listener net.Listener
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		fmt.Fprint(w, hello.SayHello(""))
		return
	}
	if d.Name == "" {
		fmt.Fprint(w, hello.SayHello(""))
		return
	}

	fmt.Fprint(w, hello.SayHello(html.EscapeString(d.Name)))
}

// NewHandler creates the HTTP handler
func NewHandler(ctx context.Context, config *Config) (http.Handler, error) {
	//logger := logging.FromContext(ctx)

	return &handler{
		config: config,
	}, nil
}

// NewServer creates the HTTP server
func NewServer(ctx context.Context, config *Config) (*Server, error) {

	// Create the net listener first, so the connection ready when we return. This
	// guarantees that it can accept requests.
	addr := fmt.Sprintf(":" + config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	return &Server{
		ip:       listener.Addr().(*net.TCPAddr).IP.String(),
		port:     strconv.Itoa(listener.Addr().(*net.TCPAddr).Port),
		listener: listener,
	}, nil
}

// ServeHTTPHandler serves with the http handler
func (s *Server) ServeHTTPHandler(ctx context.Context, handler http.Handler) error {
	return s.ServeHTTP(ctx, &http.Server{
		Handler: &ochttp.Handler{
			Handler: handler,
		},
	})
}

func (s *Server) ServeHTTP(ctx context.Context, srv *http.Server) error {
	logger := logging.FromContext(ctx)

	// Spawn a goroutine that listens for context closure. When the context is
	// closed, the server is stopped.
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()

		logger.Debugf("server.Serve: context closed")
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()

		logger.Debugf("server.Serve: shutting down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Run the server. This will block until the provided context is closed.
	if err := srv.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to serve: %w", err)
	}

	logger.Debugf("server.Serve: serving stopped")

	// Return any errors that happened during shutdown.
	select {
	case err := <-errCh:
		return fmt.Errorf("failed to shutdown: %w", err)
	default:
		return nil
	}
}
