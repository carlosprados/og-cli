package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/v2/internal/basestate"
	"github.com/carlosprados/og-cli/v2/internal/config"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// syncTarget describes where the current invocation is pointed, for recording
// at pull time and checking at deploy time.
func syncTarget(p *config.Profile, orgName, channel string) basestate.Target {
	name := profile
	if name == "" && cfg != nil {
		name = cfg.DefaultProfile
	}
	t := basestate.Target{Profile: name, Org: orgName, Channel: channel}
	if p != nil {
		t.Host = p.Host
	}
	return t
}

// recordBase stores the canonical snapshot of a freshly pulled artifact, so a
// later diff or watch can tell a local edit from a remote one — and so deploy
// can notice it is aimed somewhere other than where this came from.
//
// Failures are reported and swallowed: the artifact was fetched correctly, and
// losing the ability to classify must not fail the pull.
func recordBase(kind unwrap.Kind, id, name, artifactDir, root string, payload json.RawMessage, t basestate.Target) {
	if id == "" {
		// Nothing to key the snapshot on. Rare, and not worth a warning on
		// every pull of a catalogue entry.
		return
	}
	store := basestate.Open(root)
	if err := store.Record(kind, id, name, artifactDir, payload, t); err != nil {
		fmt.Fprintf(os.Stderr, "  hint: sync state not recorded: %v\n", err)
		return
	}
	ensureGitIgnore(root)
}

// ensureGitIgnore adds .og/ to a .gitignore in the pull root, creating it if
// needed. The store is a cache; committing it would put one developer's sync
// state in everyone else's tree.
func ensureGitIgnore(root string) {
	path := filepath.Join(root, ".gitignore")
	existing, err := os.ReadFile(path)
	if err == nil {
		for _, line := range splitLines(string(existing)) {
			if line == basestate.GitIgnoreLine || line == basestate.DirName {
				return
			}
		}
		content := string(existing)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content += "\n"
		}
		content += basestate.GitIgnoreLine + "\n"
		_ = os.WriteFile(path, []byte(content), 0o644)
		return
	}
	if os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(basestate.GitIgnoreLine+"\n"), 0o644)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, trimSpace(s[start:]))
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// warnIfMovedTarget prints a warning when an artifact is about to be deployed
// somewhere other than where it was pulled from.
//
// This closes a real footgun: nothing in an unwrapped directory recorded its
// origin, so a tree pulled from staging could be deployed to production with no
// indication at all. It warns rather than blocks — deliberately deploying a
// staging artifact to production is a legitimate promotion, and that is exactly
// the cross-tenant workflow the repo documents.
func warnIfMovedTarget(artifactDir string, t basestate.Target) {
	store, ok := basestate.Find(artifactDir)
	if !ok {
		return
	}
	entry, ok := store.LookupByDir(artifactDir)
	if !ok {
		return
	}
	diffs := entry.MovedTo(t)
	if len(diffs) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "  warning: this %s was pulled from %s\n", entry.Kind, entry.Describe())
	for _, d := range diffs {
		fmt.Fprintf(os.Stderr, "           deploying to a different %s\n", d)
	}
}
