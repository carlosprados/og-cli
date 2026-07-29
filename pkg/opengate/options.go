package opengate

import (
	"fmt"
	"net/http"
	"strings"
)

// DefaultAPIVersion is the North API version segment used when WithAPIVersion
// is not passed. Every path constant carries a version placeholder that is
// resolved against this value.
const DefaultAPIVersion = "v80"

// Option configures a Client at construction time. Options are applied in
// order, so a later one wins over an earlier one.
type Option func(*clientOptions)

// clientOptions accumulates option values before they are applied to a Client,
// so that options can be order-independent and validated as a whole.
type clientOptions struct {
	httpClient *http.Client
	tls        tlsSettings
	tlsSet     bool
	apiVersion string
	apiKey     string
	retry      RetryPolicy
	retrySet   bool
}

// WithHTTPClient makes the client use hc for every HTTP call. Use it to control
// connection pooling and timeouts, to install instrumentation, or to point the
// client at a test server's transport.
//
// It takes precedence over WithTLS: when both are given, hc's own transport is
// used as-is and the TLS settings are ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(o *clientOptions) { o.httpClient = hc }
}

// WithTLS sets the TLS behaviour of this client only, leaving other clients in
// the process untouched. insecure skips server certificate verification
// entirely — the escape hatch for self-signed certs. caFile, when non-empty,
// appends an extra CA/chain PEM to the system pool.
//
// A bad caFile is reported by Err and by every request the client makes.
func WithTLS(insecure bool, caFile string) Option {
	return func(o *clientOptions) {
		o.tls = tlsSettings{insecure: insecure, caFile: caFile}
		o.tlsSet = true
	}
}

// WithAPIVersion overrides the North API version segment (default "v80"), for
// talking to an on-premises instance pinned to another version.
func WithAPIVersion(v string) Option {
	return func(o *clientOptions) { o.apiVersion = strings.Trim(v, "/") }
}

// WithAPIKey authenticates North API calls with an API key (X-ApiKey header)
// instead of the JWT bearer token. An API key does not expire, which makes it
// the right credential for a service account driving long-running work: a JWT
// obtained at start-up may well be dead by the time a deferred phase runs.
//
// It takes precedence over the token passed to New.
func WithAPIKey(key string) Option {
	return func(o *clientOptions) { o.apiKey = key }
}

// WithRetry retries rate-limited and failing requests with exponential backoff
// and jitter, honouring a Retry-After header when the server sends one.
//
// Pass DefaultRetryPolicy() for the recommended settings. Only requests that
// cannot gain an effect by being repeated are retried after a 5xx — see
// RetryPolicy, which explains why creating a job is not one of them.
//
// Retrying is OFF unless this option is given: a library must not silently
// multiply a caller's write.
func WithRetry(p RetryPolicy) Option {
	return func(o *clientOptions) {
		o.retry = p
		o.retrySet = true
	}
}

// WithoutRetry disables retrying explicitly. It is the default, and exists so a
// caller can be unambiguous about it.
func WithoutRetry() Option {
	return func(o *clientOptions) {
		o.retry = RetryPolicy{}
		o.retrySet = true
	}
}

// apply resolves the accumulated options onto c, returning the first
// configuration error found.
func (o *clientOptions) apply(c *Client) error {
	c.apiVersion = DefaultAPIVersion
	if o.apiVersion != "" {
		c.apiVersion = o.apiVersion
	}
	c.APIKey = o.apiKey
	if o.retrySet {
		c.retry = o.retry.normalized()
	}

	switch {
	case o.httpClient != nil:
		c.HTTPClient = o.httpClient
	default:
		// Inherit the process-wide TLS defaults unless this client sets its own.
		s := processTLS
		if o.tlsSet {
			s = o.tls
		}
		if err := s.validate(); err != nil {
			return err
		}
		hc, err := s.httpClient()
		if err != nil {
			return err
		}
		c.HTTPClient = hc
	}

	if c.apiVersion == "" {
		return fmt.Errorf("api version must not be empty")
	}
	return nil
}
