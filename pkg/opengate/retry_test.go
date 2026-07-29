package opengate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fastRetry is the default policy with a tiny base delay so tests do not sleep.
func fastRetry() RetryPolicy {
	p := DefaultRetryPolicy()
	p.Base = time.Millisecond
	p.MaxDelay = 50 * time.Millisecond
	return p
}

// TestRetryOn429ThenSuccess is the T9 acceptance criterion: two 429s followed by
// a 200 are retried, and the Retry-After the server sent is respected.
func TestRetryOn429ThenSuccess(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.Header().Set("Retry-After", "1") // one second, clamped by MaxDelay below
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	p := fastRetry()
	p.MaxDelay = 40 * time.Millisecond // cap the server's 1s so the test stays quick
	c := New(srv.URL, "tok", WithRetry(p))

	start := time.Now()
	data, status, err := c.Get(context.Background(), "/north/{v}/x")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body = %s", data)
	}
	if attempts != 3 {
		t.Errorf("server saw %d attempts, want 3 (two 429s then a 200)", attempts)
	}

	// Retry-After was honoured rather than the much smaller exponential backoff:
	// two waits clamped to MaxDelay each.
	if elapsed < 2*p.MaxDelay {
		t.Errorf("elapsed %v: the Retry-After delay was not respected (expected at least %v)",
			elapsed, 2*p.MaxDelay)
	}
}

func TestRetryExhaustsAndReturnsLastResponse(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"slow down"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", WithRetry(fastRetry()))
	data, status, err := c.Get(context.Background(), "/north/{v}/x")
	if err != nil {
		t.Fatalf("a 429 is a response, not a transport error: %v", err)
	}
	if status != http.StatusTooManyRequests {
		t.Errorf("status = %d, want the final 429", status)
	}
	if attempts != DefaultRetryAttempts {
		t.Errorf("server saw %d attempts, want %d", attempts, DefaultRetryAttempts)
	}
	// The body of the last attempt must survive, so CheckResponse can explain it.
	if err := CheckResponse(data, status); err == nil || !strings.Contains(err.Error(), "slow down") {
		t.Errorf("the last response body must be returned, got %v", err)
	}
}

func TestRetryOn5xxForIdempotentRequest(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", WithRetry(fastRetry()))
	if _, status, err := c.Get(context.Background(), "/north/{v}/x"); err != nil || status != http.StatusOK {
		t.Fatalf("status %d, err %v", status, err)
	}
	if attempts != 2 {
		t.Errorf("server saw %d attempts, want 2", attempts)
	}
}

// TestNoRetryOn4xx checks a client error is not repeated: retrying a 400 or a
// 404 only wastes the platform's time.
func TestNoRetryOn4xx(t *testing.T) {
	resetTLS(t)

	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict} {
		var attempts int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(code)
		}))

		c := New(srv.URL, "tok", WithRetry(fastRetry()))
		if _, status, _ := c.Get(context.Background(), "/north/{v}/x"); status != code {
			t.Errorf("status = %d, want %d", status, code)
		}
		if attempts != 1 {
			t.Errorf("HTTP %d was attempted %d times, want 1", code, attempts)
		}
		srv.Close()
	}
}

// TestNoRetryOf5xxOnMutatingPost is the safety property: a 500 does not say
// whether the server acted before failing, so repeating a job creation could
// launch the operation twice.
func TestNoRetryOf5xxOnMutatingPost(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", WithRetry(fastRetry()))
	if _, err := c.CreateJob(context.Background(), json.RawMessage(`{"job":{}}`)); err == nil {
		t.Fatal("expected an error")
	}
	if attempts != 1 {
		t.Errorf("a job creation was attempted %d times after a 500, want 1", attempts)
	}
}

// TestRetryOf5xxOnSearchPost covers the other half: OpenGate searches are POSTs
// that change nothing, so they must still be retried.
func TestRetryOf5xxOnSearchPost(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"devices":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", WithRetry(fastRetry()))
	if _, err := c.SearchDevices(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("a search was attempted %d times, want 2", attempts)
	}
}

// TestRetryOf429OnMutatingPost checks the exception: a rate-limited request was
// never processed, so repeating it cannot duplicate anything.
func TestRetryOf429OnMutatingPost(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"job-1"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", WithRetry(fastRetry()))
	if _, err := c.CreateJob(context.Background(), json.RawMessage(`{"job":{}}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("server saw %d attempts, want 2", attempts)
	}
}

// TestRetryReplaysTheBody checks the buffered body survives a retry: an
// io.Reader is consumed by the first attempt.
func TestRetryReplaysTheBody(t *testing.T) {
	resetTLS(t)

	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf [512]byte
		n, _ := r.Body.Read(buf[:])
		bodies = append(bodies, string(buf[:n]))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"devices":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", WithRetry(fastRetry()))
	filter := json.RawMessage(`{"filter":{"eq":{"a":"b"}}}`)
	if _, err := c.SearchDevices(context.Background(), filter); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("server saw %d requests, want 2", len(bodies))
	}
	if bodies[0] != bodies[1] {
		t.Errorf("the retry sent a different body:\n first: %q\nsecond: %q", bodies[0], bodies[1])
	}
	if bodies[1] != string(filter) {
		t.Errorf("retried body = %q, want %q", bodies[1], filter)
	}
}

// TestRetryStopsOnContextCancellation checks a cancelled context ends the
// backoff instead of finishing the wait.
func TestRetryStopsOnContextCancellation(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := DefaultRetryPolicy()
	p.MaxDelay = 10 * time.Second // long enough that only cancellation ends it
	c := New(srv.URL, "tok", WithRetry(p))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err := c.Get(ctx, "/north/{v}/x")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("waited %v: cancellation did not interrupt the backoff", elapsed)
	}
	if attempts != 1 {
		t.Errorf("server saw %d attempts, want 1", attempts)
	}
}

// TestRetryDisabledByDefault checks a library does not silently multiply a
// caller's requests.
func TestRetryDisabledByDefault(t *testing.T) {
	resetTLS(t)

	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	for _, name := range []string{"default", "explicitly off"} {
		attempts = 0
		c := New(srv.URL, "tok")
		if name == "explicitly off" {
			c = New(srv.URL, "tok", WithoutRetry())
		}
		if _, _, err := c.Get(context.Background(), "/north/{v}/x"); err != nil {
			t.Fatal(err)
		}
		if attempts != 1 {
			t.Errorf("%s: server saw %d attempts, want 1", name, attempts)
		}
	}
}

func TestRetryAfterParsing(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
		ok     bool
	}{
		{"", 0, false},
		{"5", 5 * time.Second, true},
		{" 5 ", 5 * time.Second, true},
		{"0", 0, true},
		{"-3", 0, false},
		{"not-a-number", 0, false},
		{time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0, true},
	}
	for _, tc := range tests {
		h := http.Header{}
		if tc.header != "" {
			h.Set("Retry-After", tc.header)
		}
		got, ok := retryAfter(h)
		if ok != tc.ok {
			t.Errorf("Retry-After %q: ok = %v, want %v", tc.header, ok, tc.ok)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("Retry-After %q = %v, want %v", tc.header, got, tc.want)
		}
	}

	// An HTTP date in the future yields a positive wait.
	h := http.Header{}
	h.Set("Retry-After", time.Now().Add(30*time.Second).UTC().Format(http.TimeFormat))
	got, ok := retryAfter(h)
	if !ok || got <= 0 || got > 31*time.Second {
		t.Errorf("future HTTP-date Retry-After = %v (ok=%v)", got, ok)
	}
}

func TestBackoffGrowsAndIsCapped(t *testing.T) {
	p := RetryPolicy{Attempts: 10, Base: 100 * time.Millisecond, MaxDelay: time.Second}

	// Full jitter means each delay is in (0, base<<n], so compare the ceilings.
	for n, ceiling := range map[int]time.Duration{1: 100 * time.Millisecond, 2: 200 * time.Millisecond, 3: 400 * time.Millisecond} {
		for range 20 {
			d := p.backoff(n, http.Header{})
			if d <= 0 || d > ceiling {
				t.Fatalf("backoff(%d) = %v, want it within (0, %v]", n, d, ceiling)
			}
		}
	}

	// A large attempt number must not overflow into a negative or huge delay.
	for _, n := range []int{20, 40, 62, 63, 64, 100} {
		d := p.backoff(n, http.Header{})
		if d <= 0 || d > p.MaxDelay {
			t.Errorf("backoff(%d) = %v, want it within (0, %v]", n, d, p.MaxDelay)
		}
	}
}

func TestIsSearchPath(t *testing.T) {
	tests := map[string]bool{
		"/north/v80/search/devices?flattened=true":            true,
		"/north/v80/search/entities/operations/history":       true,
		"/north/v80/search/entities/alarms/summary":           true,
		"/north/v80/rules/search":                             true,
		"/north/v80/operationTypes/search":                    true,
		"/north/v80/operation/jobs":                           false,
		"/north/v80/operation/tasks":                          false,
		"/north/v80/provision/users/login":                    false,
		"/north/v80/alarms":                                   false,
		"/north/v80/provisionProcessors/provision/org/x/bulk": false,
		"/north/v80/searching/devices":                        false, // not a whole segment
	}
	for path, want := range tests {
		if got := isSearchPath(path); got != want {
			t.Errorf("isSearchPath(%q) = %v, want %v", path, got, want)
		}
	}
}
