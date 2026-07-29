package opengate

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Retry defaults. Three attempts covers a transient blip without turning a
// genuine outage into a long stall, and the delay cap keeps a Retry-After of
// "come back in an hour" from hanging a request for an hour.
const (
	DefaultRetryAttempts = 3
	DefaultRetryBase     = 500 * time.Millisecond
	DefaultRetryMaxDelay = 30 * time.Second
)

// RetryPolicy configures automatic retries.
//
// The zero value disables retrying. Use DefaultRetryPolicy for the recommended
// settings, or WithoutRetry to turn it off explicitly.
type RetryPolicy struct {
	// Attempts is the total number of tries, not the number of retries. 1 or
	// less means no retrying.
	Attempts int
	// Base is the first backoff delay; it doubles each attempt.
	Base time.Duration
	// MaxDelay caps a single wait, including one derived from Retry-After.
	MaxDelay time.Duration
	// RetryNonIdempotent allows retrying 5xx and transport failures on requests
	// that may have already changed something — creating a job, launching an
	// operation, running a bulk provision.
	//
	// Leave it off. A 500 does not say whether the server acted before failing,
	// so retrying a create can produce two jobs, and an operation already sent
	// to a device cannot be recalled. HTTP 429 is retried regardless, because a
	// rate-limited request was by definition not processed.
	RetryNonIdempotent bool
}

// DefaultRetryPolicy returns the recommended policy: three attempts with
// exponential backoff and jitter, honouring Retry-After, and no retrying of
// requests that may have already had an effect.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Attempts: DefaultRetryAttempts,
		Base:     DefaultRetryBase,
		MaxDelay: DefaultRetryMaxDelay,
	}
}

// enabled reports whether the policy retries at all.
func (p RetryPolicy) enabled() bool { return p.Attempts > 1 }

// normalized fills in zero values so a partially specified policy still works.
func (p RetryPolicy) normalized() RetryPolicy {
	if p.Base <= 0 {
		p.Base = DefaultRetryBase
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = DefaultRetryMaxDelay
	}
	return p
}

// idempotentMethod reports whether repeating the request cannot add an effect.
func idempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

// isSearchPath reports whether a path is one of the read-only search endpoints.
//
// This matters because OpenGate searches are POSTs: they carry their filter in
// the body. Judging retry safety by method alone would either skip retries for
// every search or retry a job creation, so the path is what decides. Search
// endpoints either contain a "search" segment (/search/devices) or end with one
// (/rules/search).
func isSearchPath(path string) bool {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	for _, seg := range strings.Split(path, "/") {
		if seg == "search" {
			return true
		}
	}
	return strings.HasSuffix(path, "/summary")
}

// retryableRequest reports whether this request may be repeated after a 5xx or a
// transport error.
func (p RetryPolicy) retryableRequest(method, path string) bool {
	return p.RetryNonIdempotent || idempotentMethod(method) || isSearchPath(path)
}

// shouldRetry decides whether to try again given the outcome of an attempt.
func (p RetryPolicy) shouldRetry(method, path string, status int, err error) bool {
	if !p.enabled() {
		return false
	}
	// A rate-limited request was not processed, so repeating it is always safe.
	if status == http.StatusTooManyRequests {
		return true
	}
	if !p.retryableRequest(method, path) {
		return false
	}
	if err != nil {
		return true // transport failure: connection refused, reset, timeout
	}
	return status >= 500
}

// backoff returns the wait before attempt n (1-based), preferring the server's
// Retry-After when it sent one.
//
// Jitter is full jitter — a uniform pick in [0, delay) — so a fleet of clients
// that all got throttled at the same moment does not come back in lockstep and
// throttle itself again.
func (p RetryPolicy) backoff(n int, header http.Header) time.Duration {
	if d, ok := retryAfter(header); ok {
		return min(d, p.MaxDelay)
	}

	delay := p.Base << (n - 1)
	if delay <= 0 || delay > p.MaxDelay { // also catches the shift overflowing
		delay = p.MaxDelay
	}
	return time.Duration(rand.Int64N(int64(delay)) + 1)
}

// retryAfter parses a Retry-After header, which is either a number of seconds or
// an HTTP date.
func retryAfter(header http.Header) (time.Duration, bool) {
	v := header.Get("Retry-After")
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d, true
		}
		return 0, true // a date in the past means "retry now"
	}
	return 0, false
}

// wait sleeps for d unless ctx ends first, in which case it reports the context
// error rather than finishing the wait.
func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
