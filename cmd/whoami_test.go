package cmd

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func jwtWith(payload map[string]any) string {
	body, _ := json.Marshal(payload)
	return "header." + base64.RawURLEncoding.EncodeToString(body) + ".signature"
}

func TestReadClaims(t *testing.T) {
	token := jwtWith(map[string]any{
		"sub":      "someone@example.com",
		"name":     "Someone",
		"iat":      1787996113,
		"exp":      1788082513,
		"X-ApiKey": "a-secret-that-must-not-escape",
	})

	claims, err := readClaims(token)
	if err != nil {
		t.Fatalf("readClaims: %v", err)
	}
	if claims.Sub != "someone@example.com" || claims.Name != "Someone" {
		t.Errorf("claims = %+v", claims)
	}
	if claims.Expires != 1788082513 || claims.IssuedAt != 1787996113 {
		t.Errorf("timestamps = %d / %d", claims.IssuedAt, claims.Expires)
	}
}

// The token's payload carries an API key. A command whose job is to describe a
// session must not become a way to print a secret to a terminal or a log, so
// the struct it decodes into deliberately has no field for it.
func TestReadClaimsCannotCarryTheAPIKey(t *testing.T) {
	token := jwtWith(map[string]any{"sub": "x@y.z", "X-ApiKey": "a-secret-that-must-not-escape"})

	claims, err := readClaims(token)
	if err != nil {
		t.Fatal(err)
	}
	rendered, _ := json.Marshal(claims)
	if strings.Contains(string(rendered), "a-secret") {
		t.Errorf("the API key survived into the claims struct: %s", rendered)
	}
}

func TestReadClaimsRejectsNonJWT(t *testing.T) {
	for name, token := range map[string]string{
		"not a jwt":    "just-a-string",
		"two segments": "header.payload",
		"bad base64":   "header.!!!not-base64!!!.signature",
		"not json":     "header." + base64.RawURLEncoding.EncodeToString([]byte("nope")) + ".signature",
	} {
		if _, err := readClaims(token); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
