package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the OpenGate REST API client.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
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

// doRequest executes an HTTP request with auth headers and returns the response body.
func (c *Client) doRequest(method, path string, body io.Reader) ([]byte, int, error) {
	url := c.BaseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
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
