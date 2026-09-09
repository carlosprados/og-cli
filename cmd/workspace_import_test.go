package cmd

import (
	"strings"
	"testing"
)

// A user can end up holding either of two files that both look like "the
// workspace": the document `export --full` writes, or the platform's export
// bundle. Only the first is importable, because the platform strips `_id` from
// the bundle on purpose — and the old code answered both with the same bare
// "workspace JSON has no _id", which pointed nowhere.
func TestParseWorkspaceForImport(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantID  string
		wantErr string
	}{
		{
			name:   "bare workspace document, as export --full writes it",
			body:   `{"_id":"_123","name":"Ops","dashboards":[]}`,
			wantID: "_123",
		},
		{
			name:    "platform export bundle, which carries no _id",
			body:    `{"bundles":[],"views":[{"a":1}],"workspaces":[{"name":"Ops","dashboards":[]}]}`,
			wantErr: "platform export bundle",
		},
		{
			name:   "bundle that does carry an _id is still importable",
			body:   `{"views":[],"workspaces":[{"_id":"_456","name":"Ops"}]}`,
			wantID: "_456",
		},
		{
			name:    "a document with neither shape",
			body:    `{"name":"Ops"}`,
			wantErr: "no _id",
		},
		{
			name:    "not JSON at all",
			body:    `not json`,
			wantErr: "parsing workspace JSON",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, err := parseWorkspaceForImport([]byte(tc.body))

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got workspace %+v", tc.wantErr, w)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not mention %q", err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w.ID != tc.wantID {
				t.Errorf("got _id %q, want %q", w.ID, tc.wantID)
			}
		})
	}
}

// The bundle error has to name the way out, or it just moves the confusion.
func TestParseWorkspaceForImportBundleErrorNamesTheFix(t *testing.T) {
	_, err := parseWorkspaceForImport([]byte(`{"workspaces":[{"name":"Ops"}]}`))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "--full") {
		t.Errorf("error should point at --full, got: %v", err)
	}
}
