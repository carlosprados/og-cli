package watch

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Editors do not write files, they replace them. These are the artifacts a real
// save leaves behind, and treating any of them as content means deploying a
// half-written file or a swap file.
func TestIgnoredCoversEditorArtifacts(t *testing.T) {
	ignored := []string{
		"rules/env/4913",               // Neovim's writability probe
		"rules/env/.javascript.js.swp", // vim swap
		"rules/env/javascript.js~",     // backup
		"rules/env/javascript.js.tmp",
		"rules/env/javascript.js.orig",
		"rules/env/og-globals.d.ts", // generated
		"rules/env/jsconfig.json",   // generated
		"rules/.og/manifest.json",   // our own cache — must never retrigger
		"rules/.og/base/rule__r-1.canon.json",
		"rules/.git/index",
		"rules/env/node_modules/x/index.js",
	}
	for _, path := range ignored {
		if !Ignored(path) {
			t.Errorf("%s should be ignored", path)
		}
	}

	kept := []string{
		"rules/env/javascript.js",
		"rules/env/rule.json",
		"connectors/weather/connectorfunction.json",
		"provision/p/scriptProcessor__script.js",
	}
	for _, path := range kept {
		if Ignored(path) {
			t.Errorf("%s must NOT be ignored — it is artifact content", path)
		}
	}
}

// A burst of events for one save must produce exactly one deploy.
func TestCoalescesABurstIntoOneDeploy(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "env-anomaly")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, artifact, "rule.json", `{"identifier":"r-1"}`)

	w := newTestWatcher(t, root)
	defer func() { _ = w.Close() }()

	var mu sync.Mutex
	var got []Target
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = w.Run(ctx, func(target Target) error {
			mu.Lock()
			got = append(got, target)
			mu.Unlock()
			return nil
		})
		close(done)
	}()

	// Simulate one editor save: temp file, probe, rename, backup.
	write(t, artifact, "javascript.js", "return 1;")
	write(t, artifact, "4913", "")
	write(t, artifact, "javascript.js", "return 2;")
	write(t, artifact, "javascript.js~", "return 1;")

	time.Sleep(DefaultDebounce * 3)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("a single save produced %d deploys, want 1: %v", len(got), got)
	}
	if filepath.Base(string(got[0])) != "env-anomaly" {
		t.Errorf("target = %s, want the artifact directory", got[0])
	}
}

// Editing a file deep inside an artifact must deploy that artifact — the
// smallest deployable unit — not the watch root.
func TestResolvesToTheNearestTarget(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "channel", "env-anomaly", "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "channel", "env-anomaly"), "rule.json", `{"identifier":"r-1"}`)

	w := newTestWatcher(t, root)
	defer func() { _ = w.Close() }()

	target, ok := w.resolveTarget(filepath.Join(nested, "javascript.js"))
	if !ok {
		t.Fatal("no target resolved")
	}
	if filepath.Base(string(target)) != "env-anomaly" {
		t.Errorf("target = %s, want env-anomaly", target)
	}
}

// A file outside any artifact must not deploy anything.
func TestNoTargetOutsideAnArtifact(t *testing.T) {
	root := t.TempDir()
	w := newTestWatcher(t, root)
	defer func() { _ = w.Close() }()

	if target, ok := w.resolveTarget(filepath.Join(root, "README.md")); ok {
		t.Errorf("a stray file resolved to %s", target)
	}
}

// A chmod is not a content change.
func TestChmodDoesNotTrigger(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "env")
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, artifact, "rule.json", `{"identifier":"r-1"}`)

	w := newTestWatcher(t, root)
	defer func() { _ = w.Close() }()

	path := filepath.Join(artifact, "javascript.js")
	write(t, artifact, "javascript.js", "return 1;")
	drain(w)

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if len(w.pending) != 0 {
		t.Errorf("a chmod queued a deploy: %v", w.pending)
	}
}

// An artifact directory created while watching must become watched too.
func TestNewDirectoriesAreWatched(t *testing.T) {
	root := t.TempDir()
	w := newTestWatcher(t, root)
	defer func() { _ = w.Close() }()

	before := w.Watched()
	fresh := filepath.Join(root, "new-artifact")
	if err := os.MkdirAll(fresh, 0o755); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case ev := <-w.fsw.Events:
			w.handleEvent(ev)
		case <-time.After(50 * time.Millisecond):
		}
		if w.Watched() > before {
			return
		}
	}
	t.Errorf("a directory created while watching was not picked up (%d → %d)", before, w.Watched())
}

func newTestWatcher(t *testing.T, root string) *Watcher {
	t.Helper()
	w, err := New(Options{
		Root: root,
		IsTarget: func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, "rule.json"))
			return err == nil
		},
		Debounce: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func drain(w *Watcher) {
	for {
		select {
		case ev := <-w.fsw.Events:
			w.handleEvent(ev)
		case <-time.After(80 * time.Millisecond):
			w.pending = map[Target]time.Time{}
			return
		}
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
