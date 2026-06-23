package opengate

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckResponseParsesErrorListCode(t *testing.T) {
	body := []byte(`{"errors":[{"code":"0x000065","message":"You have configured 2FA, must send the code."}]}`)
	err := CheckResponse(body, http.StatusUnauthorized)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "0x000065" {
		t.Errorf("Code = %q, want 0x000065", apiErr.Code)
	}
	if apiErr.Message != "You have configured 2FA, must send the code." {
		t.Errorf("Message = %q", apiErr.Message)
	}
}

func TestCheckResponseSimpleMessageShape(t *testing.T) {
	// The simple {"message":...} shape must still work (no code).
	err := CheckResponse([]byte(`{"message":"boom"}`), http.StatusBadRequest)
	apiErr := err.(*APIError)
	if apiErr.Code != "" || apiErr.Message != "boom" {
		t.Errorf("got code=%q msg=%q", apiErr.Code, apiErr.Message)
	}
}

func TestIs2FAChallenge(t *testing.T) {
	cases := map[string]bool{
		"0x000065": true,  // code required
		"0x000066": true,  // bad/expired code
		"0x000067": false, // code sent without 2FA — not a challenge
		"0x04":     false, // generic unauthorized
		"":         false,
	}
	for code, want := range cases {
		err := &APIError{StatusCode: 401, Code: code}
		if got := Is2FAChallenge(err); got != want {
			t.Errorf("Is2FAChallenge(code=%q) = %v, want %v", code, got, want)
		}
	}
	if Is2FAChallenge(nil) {
		t.Error("Is2FAChallenge(nil) should be false")
	}
}

// TestLogin2FAFlow drives the two-step flow: the first attempt (no code) is
// rejected with 0x000065, and the retry carrying the TOTP code succeeds.
func TestLogin2FAFlow(t *testing.T) {
	const wantCode = "123456"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)

		if req.TwoFaCode == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":[{"code":"0x000065","message":"You have configured 2FA, must send the code."}]}`))
			return
		}
		if req.TwoFaCode != wantCode {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":[{"code":"0x000066","message":"Code 2FA validation failure error."}]}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"user":{"jwt":"the-jwt","apiKey":"the-key","domain":"d1","profile":"admin","2FaType":"TOTP"}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")

	// First attempt without a code must surface a 2FA challenge.
	if _, err := c.Login("user@example.com", "pw", ""); !Is2FAChallenge(err) {
		t.Fatalf("expected 2FA challenge, got %v", err)
	}

	// A wrong code is still a challenge (so the CLI re-prompts).
	if _, err := c.Login("user@example.com", "pw", "000000"); !Is2FAChallenge(err) {
		t.Fatalf("expected 2FA challenge for bad code, got %v", err)
	}

	// The correct code logs in and reports TOTP.
	res, err := c.Login("user@example.com", "pw", wantCode)
	if err != nil {
		t.Fatalf("login with code: %v", err)
	}
	if res.JWT != "the-jwt" || res.TwoFaType != "TOTP" {
		t.Errorf("unexpected result: %+v", res)
	}
}
