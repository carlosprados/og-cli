package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Client is the OpenGate REST API client.
//
// The OpenGate platform exposes two REST surfaces:
//   - North API (/north/v80/...) — IoT plane, authenticated with Token (JWT).
//   - Web API (/api/...)         — UI plane (workspaces, dashboards),
//     authenticated with a separate WebToken obtained via WebSignIn.
//
// Both share host and HTTP transport.
//
// The Web API invalidates a WebToken whenever a fresh signin happens (e.g. the
// user logs into the OpenGate web UI in parallel). When a refresh request is
// configured via WithWebRefresh, the client will transparently re-signin on
// HTTP 401 and retry the original request once.
type Client struct {
	BaseURL    string
	Token      string
	WebToken   string
	HTTPClient *http.Client

	webRefreshMu      sync.Mutex
	webRefreshRequest *WebSignInRequest
	onWebRefresh      func(newToken string)
}

// New creates a Client from a host URL and an optional JWT token.
func New(host, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(host, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithWebToken returns the client with WebToken set. Used for Web API calls.
func (c *Client) WithWebToken(token string) *Client {
	c.WebToken = token
	return c
}

// WithWebRefresh enables transparent re-signin on 401 from the Web API.
// req carries the credentials needed to call WebSignIn again. onRefresh is
// called with the new token after a successful refresh so the caller can
// persist it (typically into ~/.og/config.yaml).
func (c *Client) WithWebRefresh(req WebSignInRequest, onRefresh func(string)) *Client {
	if req.Email == "" || req.Domain == "" || req.Profile == "" || req.Workgroup == "" {
		return c
	}
	c.webRefreshRequest = &req
	c.onWebRefresh = onRefresh
	return c
}

// doRequest executes an HTTP request with the north API token and returns the response body.
func (c *Client) doRequest(method, path string, body io.Reader) ([]byte, int, error) {
	return c.doRequestWithToken(method, path, body, c.Token)
}

// webDoRequest executes an HTTP request with the Web API token. If the server
// responds with 401 and a refresh request is configured, it re-signs in once
// and retries the request transparently.
func (c *Client) webDoRequest(method, path string, body io.Reader) ([]byte, int, error) {
	if c.WebToken == "" {
		return nil, 0, fmt.Errorf("web API token is missing — re-run `og login` to obtain it (or set OG_WEB_TOKEN)")
	}

	// Buffer body to allow one retry.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, 0, fmt.Errorf("buffering request body: %w", err)
		}
	}

	makeReader := func() io.Reader {
		if bodyBytes == nil {
			return nil
		}
		return bytes.NewReader(bodyBytes)
	}

	data, statusCode, err := c.doRequestWithToken(method, path, makeReader(), c.WebToken)
	if err != nil || !isAuthFailure(statusCode) || c.webRefreshRequest == nil {
		return data, statusCode, err
	}

	// 401/403 with refresh configured — try to re-signin once.
	fmt.Fprintln(os.Stderr, "Web token rejected (HTTP", statusCode, "); refreshing and retrying once...")
	if refreshErr := c.refreshWebToken(); refreshErr != nil {
		// Return the original 401 response; surface refresh error in a wrapping message.
		return data, statusCode, fmt.Errorf("web token refresh failed: %w", refreshErr)
	}

	return c.doRequestWithToken(method, path, makeReader(), c.WebToken)
}

// isAuthFailure returns true for HTTP status codes that indicate the bearer
// token is stale or rejected. OpenGate has been observed returning either
// 401 or 403 in this case.
func isAuthFailure(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}

// refreshWebToken re-signs in using the stored refresh credentials and
// updates the client's WebToken. Calls the onWebRefresh callback if set.
func (c *Client) refreshWebToken() error {
	c.webRefreshMu.Lock()
	defer c.webRefreshMu.Unlock()

	if c.webRefreshRequest == nil {
		return fmt.Errorf("no refresh credentials configured")
	}

	res, err := c.WebSignIn(*c.webRefreshRequest)
	if err != nil {
		return err
	}
	c.WebToken = res.JWT
	if c.onWebRefresh != nil {
		c.onWebRefresh(res.JWT)
	}
	return nil
}

func (c *Client) doRequestWithToken(method, path string, body io.Reader, token string) ([]byte, int, error) {
	url := c.BaseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	return data, resp.StatusCode, nil
}

// WebGet performs a GET against the Web API (uses WebToken).
func (c *Client) WebGet(path string) ([]byte, int, error) {
	return c.webDoRequest(http.MethodGet, path, nil)
}

// WebPost performs a POST against the Web API (uses WebToken).
func (c *Client) WebPost(path string, body io.Reader) ([]byte, int, error) {
	return c.webDoRequest(http.MethodPost, path, body)
}

// WebPut performs a PUT against the Web API (uses WebToken).
func (c *Client) WebPut(path string, body io.Reader) ([]byte, int, error) {
	return c.webDoRequest(http.MethodPut, path, body)
}

// WebDelete performs a DELETE against the Web API (uses WebToken).
func (c *Client) WebDelete(path string) ([]byte, int, error) {
	return c.webDoRequest(http.MethodDelete, path, nil)
}

// Get performs a GET request.
func (c *Client) Get(path string) ([]byte, int, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body io.Reader) ([]byte, int, error) {
	return c.doRequest(http.MethodPost, path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body io.Reader) ([]byte, int, error) {
	return c.doRequest(http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) ([]byte, int, error) {
	return c.doRequest(http.MethodDelete, path, nil)
}

// APIError represents an error response from the OpenGate API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("OpenGate API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// CheckResponse returns an APIError if the status code indicates failure.
func CheckResponse(data []byte, statusCode int) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	msg := string(data)
	// Try to extract a message from JSON error response
	var errBody struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &errBody) == nil && errBody.Message != "" {
		msg = errBody.Message
	}
	return &APIError{StatusCode: statusCode, Message: msg}
}

// IsEmptyResponse returns true when the API returned no content (204 or empty body).
func IsEmptyResponse(data []byte, statusCode int) bool {
	return statusCode == 204 || len(data) == 0
}
