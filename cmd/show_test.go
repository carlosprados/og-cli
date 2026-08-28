package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// The extension's diff provider addresses a remote file by the name `pull`
// wrote on disk. If those two ever drift, every remote diff opens empty, so
// this pins that they come from the same extraction.
func TestRemoteCodeFilesUsesThePullNames(t *testing.T) {
	payload := json.RawMessage(`{
		"identifier": "r1",
		"name": "a rule",
		"mode": "ADVANCED",
		"javascript": "var x = 1;\nreturn x;\n"
	}`)

	files, err := remoteCodeFiles(unwrap.RuleDescriptor(), payload)
	if err != nil {
		t.Fatalf("remoteCodeFiles: %v", err)
	}
	code, ok := files["javascript.js"]
	if !ok {
		t.Fatalf("no javascript.js; got %v", sortedFileNames(files))
	}
	if code != "var x = 1;\nreturn x;\n" {
		t.Errorf("content = %q", code)
	}
}

func TestRemoteCodeFilesRejectsAMalformedPayload(t *testing.T) {
	if _, err := remoteCodeFiles(unwrap.RuleDescriptor(), json.RawMessage(`not json`)); err == nil {
		t.Error("expected an error on a payload that is not JSON")
	}
}

// The error for a wrong --path has to name what is available, or the caller is
// left guessing at a filename it cannot see.
func TestJoinOrNone(t *testing.T) {
	if got := joinOrNone(nil); !strings.Contains(got, "no code files") {
		t.Errorf("empty case = %q", got)
	}
	if got := joinOrNone([]string{"b.js", "a.js"}); got != "b.js, a.js" {
		t.Errorf("joined = %q", got)
	}
}
