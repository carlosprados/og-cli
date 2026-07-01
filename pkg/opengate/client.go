package opengate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// New creates a Client from a host URL and an optional JWT token. The HTTP
// client honours the process-wide TLS settings configured via ConfigureTLS
// (e.g. --insecure / --ca-file for self-signed servers).
func New(host, token string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(host, "/"),
		Token:      token,
		HTTPClient: NewHTTPClient(),
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

// GetWithAccept performs a GET request with an explicit Accept header. Used by
// endpoints that return a non-JSON body (e.g. an Excel file) and reject the
// request with HTTP 409 when Accept does not match the response content type.
func (c *Client) GetWithAccept(path, accept string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
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

// excelContentType maps a spreadsheet file extension to its MIME type.
// Defaults to the xlsx type for unknown extensions.
func excelContentType(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".xls":
		return "application/vnd.ms-excel"
	default:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
}

// PostMultipartFile uploads a local file as multipart/form-data to the north API
// and returns the response body, status code, and the Location response header
// (used by endpoints that report a created resource via Location). The part's
// content type is derived from the file extension.
//
// accept sets the request's Accept header when non-empty. Some endpoints (e.g.
// the provision bulk execution) reject the request with HTTP 409 unless Accept
// matches the uploaded file's content type.
func (c *Client) PostMultipartFile(path, fieldName, filePath, accept string) (body []byte, status int, location string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("opening %s: %w", filePath, err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name=%q; filename=%q`, fieldName, filepath.Base(filePath)))
	header.Set("Content-Type", excelContentType(filePath))
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, 0, "", fmt.Errorf("creating multipart part: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, 0, "", fmt.Errorf("writing multipart body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, 0, "", fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, &buf)
	if err != nil {
		return nil, 0, "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, "", fmt.Errorf("reading response: %w", err)
	}
	return data, resp.StatusCode, resp.Header.Get("Location"), nil
}

// APIError represents an error response from the OpenGate API.
type APIError struct {
	StatusCode int
	Code       string // OpenGate error code (e.g. "0x000065"), when present
	Message    string
	Fields     []string // offending field names from the error context, when present
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("OpenGate API error (HTTP %d): %s", e.StatusCode, e.Message)
	if len(e.Fields) > 0 {
		msg += fmt.Sprintf(" (fields: %s)", strings.Join(e.Fields, ", "))
	}
	return msg
}

// CheckResponse returns an APIError if the status code indicates failure.
func CheckResponse(data []byte, statusCode int) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	msg := string(data)
	code := ""
	var fields []string
	// OpenGate error bodies come in two shapes:
	//   {"message":"..."}                                                    (simple)
	//   {"errors":[{"code":"0x..","message":"...","context":[{"name":".."}]}]} (ErrorList)
	// The context carries the offending field(s) — e.g. a 400 "Forbidden field."
	// on a datamodel PUT names allowedResourceTypes there, which is otherwise invisible.
	var errList struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Context []struct {
				Name string `json:"name"`
			} `json:"context"`
		} `json:"errors"`
	}
	if json.Unmarshal(data, &errList) == nil && len(errList.Errors) > 0 {
		code = errList.Errors[0].Code
		if errList.Errors[0].Message != "" {
			msg = errList.Errors[0].Message
		}
		for _, ctx := range errList.Errors[0].Context {
			if ctx.Name != "" {
				fields = append(fields, ctx.Name)
			}
		}
	} else {
		var errBody struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &errBody) == nil && errBody.Message != "" {
			msg = errBody.Message
		}
	}
	return &APIError{StatusCode: statusCode, Code: code, Message: msg, Fields: fields}
}

// IsEmptyResponse returns true when the API returned no content (204 or empty body).
func IsEmptyResponse(data []byte, statusCode int) bool {
	return statusCode == 204 || len(data) == 0
}
