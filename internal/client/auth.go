package client

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
)

const loginPath = "/north/v80/provision/users/login"

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
	JWT    string
	APIKey string
	Domain string
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
		JWT:    resp.User.JWT,
		APIKey: resp.User.APIKey,
		Domain: resp.User.Domain,
	}, nil
}
