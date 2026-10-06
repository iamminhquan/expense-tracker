package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// The timeouts a bare http.ListenAndServe leaves at zero (wait forever), which
// lets one client hold a connection open by sending its headers a byte at a
// time. The API's biggest request is a 1 MB CSV upload and its slowest
// response a CSV export, so 30 seconds is generous for both.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 2 * time.Minute

	// shutdownTimeout is how long in-flight requests get to finish after a
	// stop signal. Render sends SIGTERM on every redeploy and kills the
	// process 30 seconds later.
	shutdownTimeout = 10 * time.Second
)

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// serve runs srv until it fails or ctx is cancelled. On cancel it stops
// accepting connections and waits up to shutdownTimeout for in-flight
// requests, so a redeploy doesn't cut a transaction write off mid-request.
func serve(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
