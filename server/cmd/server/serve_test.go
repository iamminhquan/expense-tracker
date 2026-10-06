package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewServerSetsEveryTimeout(t *testing.T) {
	srv := newServer(":0", http.NewServeMux())
	if srv.ReadHeaderTimeout == 0 || srv.ReadTimeout == 0 || srv.WriteTimeout == 0 || srv.IdleTimeout == 0 {
		t.Fatalf("a timeout is unset: %+v", srv)
	}
}

// TestServeDrainsInFlightRequestOnCancel pins the point of graceful
// shutdown: a request already being handled when the stop signal arrives
// still gets its response.
func TestServeDrainsInFlightRequestOnCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	srv := newServer(addr, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serve(ctx, srv) }()

	status := make(chan int, 1)
	go func() {
		for range 50 {
			resp, err := http.Get("http://" + addr)
			if err == nil {
				resp.Body.Close()
				status <- resp.StatusCode
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		status <- 0
	}()

	<-started
	cancel()
	select {
	case err := <-served:
		t.Fatalf("serve returned %v before the in-flight request finished", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)

	if got := <-status; got != http.StatusOK {
		t.Fatalf("in-flight request got status %d, want 200", got)
	}
	if err := <-served; err != nil {
		t.Fatalf("serve returned %v after a clean shutdown, want nil", err)
	}
}
