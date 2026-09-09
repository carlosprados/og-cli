package opengate

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The delete endpoint takes its ids under "dasboardsDelete" — the platform's
// spelling, missing the "h". Every other body shape returns 400 with an empty
// message, so this key is the whole contract and a typo here is silent.
//
// Read off the web client's bundle and confirmed against a live tenant; the
// server is stubbed here so the assertion is about what og sends.
func TestDeleteDashboardSendsPlatformKey(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string][]string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Deleted"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "token")
	c.WebToken = "token"

	if err := c.DeleteDashboard(context.Background(), "dash-1"); err != nil {
		t.Fatalf("DeleteDashboard: %v", err)
	}

	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	// The trailing slash matters: without it the platform answers 400.
	if !strings.HasSuffix(gotPath, "/api/dashboards/") {
		t.Errorf("path = %q, want it to end in /api/dashboards/", gotPath)
	}
	ids, ok := gotBody["dasboardsDelete"]
	if !ok {
		t.Fatalf("body has no dasboardsDelete key: %v", gotBody)
	}
	if len(ids) != 1 || ids[0] != "dash-1" {
		t.Errorf("dasboardsDelete = %v, want [dash-1]", ids)
	}
}
