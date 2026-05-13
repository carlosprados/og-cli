package client

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
)

const (
	loginPath        = "/north/v80/provision/users/login"
	webSignInPath    = "/api/auth/signin/internal"
)

// LoginRequest holds credentials for JWT authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse holds the response from the login endpoint.
type LoginResponse struct {
	User LoginUser `json:"user"`
}

// LoginUser holds user info returned after login.
type LoginUser struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	JWT     string `json:"jwt"`
	APIKey  string `json:"apiKey"`
	Profile string `json:"profile"`
	Domain  string `json:"domain"`
}

// LoginResult holds the credentials returned by a successful login.
type LoginResult struct {
	JWT     string
	APIKey  string
	Domain  string
	Profile string
}

// WebSignInRequest is the body sent to /api/auth/signin/internal.
type WebSignInRequest struct {
	Email     string `json:"email"`
	Domain    string `json:"domain"`
	Profile   string `json:"profile"`
	Workgroup string `json:"workgroup"`
}

// WebSignInResult holds the credentials returned by a successful web signin.
type WebSignInResult struct {
	JWT       string `json:"jwt"`
	Email     string `json:"email"`
	Domain    string `json:"domain"`
	Profile   string `json:"profile"`
	Workgroup string `json:"workgroup"`
}

// WebSignIn exchanges the north-API bearer token (already set on the Client)
// for a Web API JWT. The body fields email/domain/profile/workgroup are all
// required by the server.
func (c *Client) WebSignIn(req WebSignInRequest) (*WebSignInResult, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("web signin requires a north API token (run og login first)")
	}
	if req.Email == "" || req.Domain == "" || req.Profile == "" || req.Workgroup == "" {
		return nil, fmt.Errorf("web signin requires email, domain, profile and workgroup")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling web signin: %w", err)
	}

	data, statusCode, err := c.Post(webSignInPath, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("web signin request: %w", err)
	}

	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var resp WebSignInResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing web signin response: %w", err)
	}

	if resp.JWT == "" {
		return nil, fmt.Errorf("empty JWT in web signin response")
	}
	return &resp, nil
}

// Login authenticates against OpenGate and returns JWT token, API key, and domain.
func (c *Client) Login(email, password string) (*LoginResult, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("invalid email address %q", email)
	}

	payload, err := json.Marshal(LoginRequest{Email: email, Password: password})
	if err != nil {
		return nil, fmt.Errorf("marshaling login request: %w", err)
	}

	data, statusCode, err := c.Post(loginPath, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("login request: %w", err)
	}

	if err := CheckResponse(data, statusCode); err != nil {
		return nil, err
	}

	var resp LoginResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing login response: %w", err)
	}

	if resp.User.JWT == "" {
		return nil, fmt.Errorf("empty JWT in login response")
	}

	return &LoginResult{
		JWT:     resp.User.JWT,
		APIKey:  resp.User.APIKey,
		Domain:  resp.User.Domain,
		Profile: resp.User.Profile,
	}, nil
}
