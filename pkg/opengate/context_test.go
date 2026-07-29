package opengate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// slowServer blocks until the request's context is done, so a test can prove the
// cancellation reached the transport rather than merely timing out locally.
//
// The fallback is deliberately short: with a request body the server does not
// always observe the client's disconnect promptly, and a long fallback would
// leave the handler — and srv.Close — hanging for its full duration. 2s is 40x
// the cancellation delay used by the tests, so it never races.
func slowServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestContextCancellationMidRequest is the T1 acceptance criterion: a ctx
// cancelled while a request is in flight aborts it, and the error wraps
// context.Canceled so a caller can tell "the user went away" from "the platform
// failed".
func TestContextCancellationMidRequest(t *testing.T) {
	resetTLS(t)
	srv := slowServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	c := New(srv.URL, "tok")
	start := time.Now()
	_, _, err := c.Get(ctx, "/north/{v}/slow")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from the cancelled request")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error must wrap context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Errorf("request did not abort on cancellation (took %v)", elapsed)
	}
}

// TestContextCancellationReachesResourceMethods checks the ctx is threaded all
// the way through the typed resource layer, not just the raw request helpers.
func TestContextCancellationReachesResourceMethods(t *testing.T) {
	resetTLS(t)
	srv := slowServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	c := New(srv.URL, "tok")
	_, err := c.SearchDevices(ctx, nil)
	if err == nil {
		t.Fatal("expected an error from the cancelled search")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error must wrap context.Canceled, got %v", err)
	}
}

// TestContextDeadlineIsHonoured covers the per-request deadline a long-running
// service needs, which is the other half of why ctx exists.
func TestContextDeadlineIsHonoured(t *testing.T) {
	resetTLS(t)
	srv := slowServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := New(srv.URL, "tok")
	_, _, err := c.Get(ctx, "/north/{v}/slow")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error must wrap context.DeadlineExceeded, got %v", err)
	}
}

// TestAlreadyCancelledContextDoesNotHitTheNetwork guards the cheap case: a ctx
// that is already dead must fail before a connection is made.
func TestAlreadyCancelledContextDoesNotHitTheNetwork(t *testing.T) {
	resetTLS(t)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := New(srv.URL, "tok")
	if _, _, err := c.Get(ctx, "/north/{v}/x"); !errors.Is(err, context.Canceled) {
		t.Errorf("error must wrap context.Canceled, got %v", err)
	}
	if hits != 0 {
		t.Errorf("server was contacted %d times with a dead context", hits)
	}
}
