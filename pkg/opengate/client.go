package opengate

import (
	"bytes"
	"context"
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
	APIKey     string // when set, North API calls use X-ApiKey instead of the bearer token
	HTTPClient *http.Client

	apiVersion string
	initErr    error

	webRefreshMu      sync.Mutex
	webRefreshRequest *WebSignInRequest
	onWebRefresh      func(newToken string)
}

// New creates a Client from a host URL and an optional JWT token.
//
// Without options the client inherits the process-wide TLS settings configured
// via ConfigureTLS and targets North API DefaultAPIVersion. Pass WithHTTPClient,
// WithTLS, WithAPIVersion or WithAPIKey to configure it independently of any
// other client in the process.
//
// New never returns nil and never fails: a bad configuration (e.g. an unreadable
// --ca-file) is recorded and returned by Err and by every request the client
// makes. Long-running consumers should check Err right after construction.
func New(host, token string, opts ...Option) *Client {
	c := &Client{
		BaseURL: strings.TrimRight(host, "/"),
		Token:   token,
	}

	var o clientOptions
	for _, opt := range opts {
		opt(&o)
	}
	if err := o.apply(c); err != nil {
		c.initErr = err
		if c.HTTPClient == nil {
			c.HTTPClient = NewHTTPClient()
		}
	}
	return c
}

// Err reports a configuration error captured during New, if any. Every request
// the client makes fails with the same error until it is fixed.
func (c *Client) Err() error { return c.initErr }

// authHeader is the credential a single request carries. The zero value sends
// no credential at all (used by login).
type authHeader struct {
	name  string
	value string
}

func (a authHeader) set(req *http.Request) {
	if a.name != "" {
		req.Header.Set(a.name, a.value)
	}
}

// bearer builds the Authorization header for a JWT.
func bearer(token string) authHeader {
	if token == "" {
		return authHeader{}
	}
	return authHeader{name: "Authorization", value: "Bearer " + token}
}

// northAuth returns the credential for North API calls. An API key wins over
// the JWT: a caller that configured WithAPIKey did so precisely to avoid
// depending on a token that expires.
func (c *Client) northAuth() authHeader {
	if c.APIKey != "" {
		return authHeader{name: "X-ApiKey", value: c.APIKey}
	}
	return bearer(c.Token)
}

// versionToken is the placeholder every North API path constant carries in
// place of a hardcoded version segment. resolvePath substitutes it with the
// client's version, so retargeting an instance pinned to another API version is
// one option instead of a fork.
const versionToken = "{v}"

// resolvePath substitutes the API version placeholder in a path constant.
func (c *Client) resolvePath(path string) string {
	v := c.apiVersion
	if v == "" {
		v = DefaultAPIVersion
	}
	return strings.Replace(path, versionToken, v, 1)
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

// doRequest executes an HTTP request with the north API credential and returns the response body.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
	return c.doRequestWithAuth(ctx, method, path, body, c.northAuth())
}

// webDoRequest executes an HTTP request with the Web API token. If the server
// responds with 401 and a refresh request is configured, it re-signs in once
// and retries the request transparently.
func (c *Client) webDoRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, int, error) {
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

	data, statusCode, err := c.doRequestWithAuth(ctx, method, path, makeReader(), bearer(c.WebToken))
	if err != nil || !isAuthFailure(statusCode) || c.webRefreshRequest == nil {
		return data, statusCode, err
	}

	// 401/403 with refresh configured — try to re-signin once.
	fmt.Fprintln(os.Stderr, "Web token rejected (HTTP", statusCode, "); refreshing and retrying once...")
	if refreshErr := c.refreshWebToken(ctx); refreshErr != nil {
		// Return the original 401 response; surface refresh error in a wrapping message.
		return data, statusCode, fmt.Errorf("web token refresh failed: %w", refreshErr)
	}

	return c.doRequestWithAuth(ctx, method, path, makeReader(), bearer(c.WebToken))
}

// isAuthFailure returns true for HTTP status codes that indicate the bearer
// token is stale or rejected. OpenGate has been observed returning either
// 401 or 403 in this case.
func isAuthFailure(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}

// refreshWebToken re-signs in using the stored refresh credentials and
// updates the client's WebToken. Calls the onWebRefresh callback if set.
func (c *Client) refreshWebToken(ctx context.Context) error {
	c.webRefreshMu.Lock()
	defer c.webRefreshMu.Unlock()

	if c.webRefreshRequest == nil {
		return fmt.Errorf("no refresh credentials configured")
	}

	res, err := c.WebSignIn(ctx, *c.webRefreshRequest)
	if err != nil {
		return err
	}
	c.WebToken = res.JWT
	if c.onWebRefresh != nil {
		c.onWebRefresh(res.JWT)
	}
	return nil
}

func (c *Client) doRequestWithAuth(ctx context.Context, method, path string, body io.Reader, auth authHeader) ([]byte, int, error) {
	if c.initErr != nil {
		return nil, 0, c.initErr
	}
	url := c.BaseURL + c.resolvePath(path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	auth.set(req)

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
func (c *Client) WebGet(ctx context.Context, path string) ([]byte, int, error) {
	return c.webDoRequest(ctx, http.MethodGet, path, nil)
}

// WebPost performs a POST against the Web API (uses WebToken).
func (c *Client) WebPost(ctx context.Context, path string, body io.Reader) ([]byte, int, error) {
	return c.webDoRequest(ctx, http.MethodPost, path, body)
}

// WebPut performs a PUT against the Web API (uses WebToken).
func (c *Client) WebPut(ctx context.Context, path string, body io.Reader) ([]byte, int, error) {
	return c.webDoRequest(ctx, http.MethodPut, path, body)
}

// WebDelete performs a DELETE against the Web API (uses WebToken).
func (c *Client) WebDelete(ctx context.Context, path string) ([]byte, int, error) {
	return c.webDoRequest(ctx, http.MethodDelete, path, nil)
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, int, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// GetWithAccept performs a GET request with an explicit Accept header. Used by
// endpoints that return a non-JSON body (e.g. an Excel file) and reject the
// request with HTTP 409 when Accept does not match the response content type.
func (c *Client) GetWithAccept(ctx context.Context, path, accept string) ([]byte, int, error) {
	if c.initErr != nil {
		return nil, 0, c.initErr
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+c.resolvePath(path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	c.northAuth().set(req)

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
func (c *Client) Post(ctx context.Context, path string, body io.Reader) ([]byte, int, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body io.Reader) ([]byte, int, error) {
	return c.doRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, int, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
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
func (c *Client) PostMultipartFile(ctx context.Context, path, fieldName, filePath, accept string) (body []byte, status int, location string, err error) {
	if c.initErr != nil {
		return nil, 0, "", c.initErr
	}
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+c.resolvePath(path), &buf)
	if err != nil {
		return nil, 0, "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	c.northAuth().set(req)

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
