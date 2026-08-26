package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/basestate"
)

// The store is a per-developer sync cache. Committing it would put one
// developer's pull state in everyone else's tree, so pull adds it to
// .gitignore — creating the file when there is none.
func TestEnsureGitIgnore(t *testing.T) {
	t.Run("creates the file", func(t *testing.T) {
		root := t.TempDir()
		ensureGitIgnore(root)

		data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
		if err != nil {
			t.Fatalf("gitignore not created: %v", err)
		}
		if !strings.Contains(string(data), basestate.GitIgnoreLine) {
			t.Errorf("gitignore = %q, want it to contain %q", data, basestate.GitIgnoreLine)
		}
	})

	t.Run("appends to an existing file", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ".gitignore")
		if err := os.WriteFile(path, []byte("node_modules/\n*.log\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		ensureGitIgnore(root)

		data, _ := os.ReadFile(path)
		got := string(data)
		for _, want := range []string{"node_modules/", "*.log", basestate.GitIgnoreLine} {
			if !strings.Contains(got, want) {
				t.Errorf("gitignore lost or missed %q: %q", want, got)
			}
		}
	})

	t.Run("appends a newline first when the file lacks one", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ".gitignore")
		if err := os.WriteFile(path, []byte("node_modules/"), 0o644); err != nil {
			t.Fatal(err)
		}
		ensureGitIgnore(root)

		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), "node_modules/"+basestate.GitIgnoreLine) {
			t.Errorf("entries must not be concatenated onto one line: %q", data)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 2 {
			t.Errorf("expected two entries, got %q", data)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		root := t.TempDir()
		ensureGitIgnore(root)
		ensureGitIgnore(root)
		ensureGitIgnore(root)

		data, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
		if n := strings.Count(string(data), basestate.GitIgnoreLine); n != 1 {
			t.Errorf("entry written %d times, want 1: %q", n, data)
		}
	})

	t.Run("recognises the bare directory form", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ".gitignore")
		if err := os.WriteFile(path, []byte(basestate.DirName+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		ensureGitIgnore(root)

		data, _ := os.ReadFile(path)
		if strings.Count(string(data), basestate.DirName) != 1 {
			t.Errorf("`.og` without a slash already covers it: %q", data)
		}
	})
}
