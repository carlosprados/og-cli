// Package watch turns file saves into deploys.
//
// This is the only part of og that writes to the platform without a human
// decision per action, so the design is mostly about not doing that wrongly:
// resolve a changed file to the smallest deployable unit, coalesce a burst of
// events into one deploy, validate before pushing, and refuse outright when the
// remote has moved.
package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultDebounce is how long to wait for a burst of events to settle.
//
// Editors do not write files, they replace them: Neovim writes a temporary
// file, renames it over the target, and touches a probe file — three or four
// events for one save. Deploying on the first would upload a half-written file.
const DefaultDebounce = 300 * time.Millisecond

// ignoredNames are exact filenames that are never artifact content.
//
// 4913 is Neovim's writability probe (the number is arbitrary, from the source);
// it appears and vanishes on every save.
var ignoredNames = map[string]bool{
	"4913":          true,
	".DS_Store":     true,
	"jsconfig.json": true, // generated
}

// ignoredSuffixes cover editor scratch files.
var ignoredSuffixes = []string{
	".swp", ".swx", ".swo", // vim swap
	"~",             // emacs/vim backup
	".tmp", ".temp", // generic
	".orig", ".rej", // merge leftovers
	".d.ts", // generated typings
}

// ignoredDirs are directory names never walked or watched.
var ignoredDirs = map[string]bool{
	".og":          true, // our own sync cache: writing to it must not retrigger
	".git":         true,
	"node_modules": true,
}

// Ignored reports whether a path should be disregarded entirely.
func Ignored(path string) bool {
	base := filepath.Base(path)
	if ignoredNames[base] {
		return true
	}
	// Vim writes .filename.swp — hidden, and by suffix too.
	for _, suffix := range ignoredSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if ignoredDirs[part] {
			return true
		}
	}
	return false
}

// Target is one deployable unit — an artifact directory.
type Target string

// Options configures a Watcher.
type Options struct {
	// Root is the directory tree to watch.
	Root string

	// IsTarget reports whether a directory is a deployable unit. Resolution
	// walks up from a changed file until this says yes, so editing a file deep
	// inside an artifact deploys that artifact and not the whole tree.
	IsTarget func(dir string) bool

	// Debounce defaults to DefaultDebounce.
	Debounce time.Duration
}

// Watcher turns filesystem events into coalesced target notifications.
type Watcher struct {
	opts    Options
	fsw     *fsnotify.Watcher
	pending map[Target]time.Time
}

// New creates a watcher over opts.Root, watching every subdirectory.
func New(opts Options) (*Watcher, error) {
	if opts.IsTarget == nil {
		return nil, fmt.Errorf("watch: IsTarget is required")
	}
	if opts.Debounce == 0 {
		opts.Debounce = DefaultDebounce
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{opts: opts, fsw: fsw, pending: map[Target]time.Time{}}
	if err := w.addTree(opts.Root); err != nil {
		_ = fsw.Close()
		return nil, err
	}
	return w, nil
}

// addTree watches root and every subdirectory under it. Directories are watched
// rather than files because a rename-over-target — how editors save — is an
// event on the directory, not on the old file.
func (w *Watcher) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if Ignored(path) {
			return filepath.SkipDir
		}
		return w.fsw.Add(path)
	})
}

// Close releases the underlying watcher.
func (w *Watcher) Close() error { return w.fsw.Close() }

// Watched reports how many directories are being watched, for startup output.
func (w *Watcher) Watched() int { return len(w.fsw.WatchList()) }

// Run delivers coalesced targets to handle until ctx is done.
//
// One handler runs at a time, and further events for a target already queued
// collapse into the pending entry rather than queueing a second deploy: the
// platform will not merge two concurrent updates to one artifact, it will keep
// whichever arrives last.
func (w *Watcher) Run(ctx context.Context, handle func(Target) error) error {
	ticker := time.NewTicker(w.opts.Debounce / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			// A watcher error is worth reporting but not worth stopping for:
			// losing one event is better than ending the session.
			fmt.Fprintf(os.Stderr, "  watch: %v\n", err)

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(ev)

		case <-ticker.C:
			for target, due := range w.pending {
				if time.Now().Before(due) {
					continue
				}
				delete(w.pending, target)
				if err := handle(target); err != nil {
					fmt.Fprintf(os.Stderr, "  %v\n", err)
				}
			}
		}
	}
}

// handleEvent filters an event and queues its target.
func (w *Watcher) handleEvent(ev fsnotify.Event) {
	// Chmod alone is not a content change; ignore it so a `chmod` or a backup
	// tool does not trigger a deploy.
	if ev.Op == fsnotify.Chmod {
		return
	}
	if Ignored(ev.Name) {
		return
	}

	// A new directory must be watched too, or artifacts pulled while watching
	// are invisible.
	if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
		if ev.Has(fsnotify.Create) {
			_ = w.addTree(ev.Name)
		}
		return
	}

	target, ok := w.resolveTarget(ev.Name)
	if !ok {
		return
	}
	w.pending[target] = time.Now().Add(w.opts.Debounce)
}

// resolveTarget walks up from a changed path to the nearest deployable unit.
//
// The smallest one, not the root: editing one widget's code should deploy that
// widget's dashboard, not every dashboard in the tree.
func (w *Watcher) resolveTarget(changed string) (Target, bool) {
	dir := filepath.Dir(changed)
	root, err := filepath.Abs(w.opts.Root)
	if err != nil {
		return "", false
	}
	for {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", false
		}
		if w.opts.IsTarget(dir) {
			return Target(dir), true
		}
		if abs == root {
			return "", false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
