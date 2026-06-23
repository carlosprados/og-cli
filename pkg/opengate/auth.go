package opengate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

const (
	loginPath     = "/north/v80/provision/users/login"
	webSignInPath = "/api/auth/signin/internal"
)

// OpenGate 2FA (TOTP) login error codes, returned by the login endpoint.
// (0x000067 — code sent but account has no 2FA — is left to the generic error path.)
const (
	errCode2FARequired = "0x000065" // 2FA configured but no code sent (or first-time setup)
	errCode2FAInvalid  = "0x000066" // 2FA code is wrong or expired
)

// LoginRequest holds credentials for JWT authentication.
type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	TwoFaCode string `json:"2FaCode,omitempty"`
}

// LoginResponse holds the response from the login endpoint.
type LoginResponse struct {
	User LoginUser `json:"user"`
}

// LoginUser holds user info returned after login.
type LoginUser struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	Surname   string `json:"surname"`
	JWT       string `json:"jwt"`
	APIKey    string `json:"apiKey"`
	Profile   string `json:"profile"`
	Domain    string `json:"domain"`
	TwoFaType string `json:"2FaType"` // "TOTP" when 2FA is enabled, "NONE" otherwise
}

// LoginResult holds the credentials returned by a successful login.
type LoginResult struct {
	JWT       string
	APIKey    string
	Domain    string
	Profile   string
	TwoFaType string
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

// Is2FAChallenge reports whether err means the server wants a TOTP 2FA code —
// either none was sent (the account has 2FA enabled) or the one sent was
// wrong/expired. Callers should prompt for a code and retry Login.
func Is2FAChallenge(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == errCode2FARequired || apiErr.Code == errCode2FAInvalid
	}
	return false
}

// GenerateTOTPCode derives the current 6-digit TOTP code from a base32 secret
// (the seed shown when enabling 2FA in the web UI). It lets og log in
// non-interactively when the secret is stored in the profile or supplied via
// OG_2FA_SECRET. Spaces are stripped and the secret is upper-cased to tolerate
// the way authenticator apps display it.
func GenerateTOTPCode(secret string) (string, error) {
	secret = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	if secret == "" {
		return "", fmt.Errorf("empty TOTP secret")
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("generating TOTP code from secret: %w", err)
	}
	return code, nil
}

// Login authenticates against OpenGate and returns JWT token, API key, and
// domain. twoFaCode is the 6-digit TOTP code for accounts with 2FA enabled;
// pass "" when the account has no 2FA. When 2FA is required but the code is
// missing or invalid, the returned error satisfies Is2FAChallenge.
func (c *Client) Login(email, password, twoFaCode string) (*LoginResult, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("invalid email address %q", email)
	}

	payload, err := json.Marshal(LoginRequest{Email: email, Password: password, TwoFaCode: twoFaCode})
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
		JWT:       resp.User.JWT,
		APIKey:    resp.User.APIKey,
		Domain:    resp.User.Domain,
		Profile:   resp.User.Profile,
		TwoFaType: resp.User.TwoFaType,
	}, nil
}
