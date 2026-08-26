package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/basestate"
	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// Result is one artifact's complete comparison. It is the shape emitted under
// `-o json`, so it is a contract: additive changes only, and the envelope's
// schemaVersion moves if that is ever broken.
type Result struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Dir  string `json:"dir"`

	// State is the three-way classification, absent when there is no base
	// snapshot to compare against.
	State string `json:"state"`

	// Metadata differences, over the canonical form.
	Metadata []Change `json:"metadata,omitempty"`
	// Code differences, one entry per extracted source file.
	Code []FileDiff `json:"code,omitempty"`

	// Origin records where the local tree was pulled from, when known.
	Origin string `json:"origin,omitempty"`
	// Ignored lists fields excluded from the comparison, so an unexpected
	// "no differences" can be explained.
	Ignored string `json:"ignored,omitempty"`
}

// Changed reports whether anything differs.
func (r Result) Changed() bool {
	if len(r.Metadata) > 0 {
		return true
	}
	for _, f := range r.Code {
		if f.Changed() {
			return true
		}
	}
	return false
}

// Compare produces the full result for one artifact: the local tree against a
// remote payload.
//
// local and remote are the wrapped payloads. The code files are compared from
// the extracted files on both sides rather than from the payload strings, so
// the output is a diff of the files the developer actually edits.
func Compare(d unwrap.Descriptor, artifactDir string, local, remote json.RawMessage, opts Options) (Result, error) {
	r := Result{
		Kind: string(d.Kind),
		Name: d.NameOf(local),
		Dir:  artifactDir,
	}

	canonOpts := canon.Options{Kind: d.Kind, Scope: opts.Scope}

	localCanon, err := canon.Canonicalize(local, canonOpts)
	if err != nil {
		return r, err
	}
	remoteCanon, err := canon.Canonicalize(remote, canonOpts)
	if err != nil {
		return r, err
	}

	// Metadata: compare with the code fields removed. They are reported as
	// textual diffs, and a 1500-character string changing tells nobody anything.
	localMeta, err := stripCode(d, localCanon)
	if err != nil {
		return r, err
	}
	remoteMeta, err := stripCode(d, remoteCanon)
	if err != nil {
		return r, err
	}
	// Remote is the "before" side, local the "after", so the whole report reads
	// as what deploying would do to the platform — the same direction as the
	// code diffs below, and the same convention as `git diff`. Mixing the two
	// directions in one report is worse than either choice.
	if r.Metadata, err = Structural(remoteMeta, localMeta); err != nil {
		return r, err
	}

	// Code: one textual diff per declared code file. Remote is the "before"
	// side and local the "after", so the diff reads as what deploying would do.
	r.Code = compareCode(d, remote, local, opts.ContextLines)

	r.Ignored = canon.Diagnose(remote, canonOpts)

	// Three-way state, when a base snapshot exists.
	if store, ok := basestate.Find(artifactDir); ok {
		cmp, entry, err := store.ClassifyArtifact(d.Kind, artifactDir, local, remote)
		if err == nil {
			r.State = cmp.State.String()
			if entry.ID != "" {
				r.ID = entry.ID
				r.Origin = entry.Describe()
			}
		}
	}
	if r.State == "" {
		r.State = basestate.Unknown.String()
	}
	return r, nil
}

// Options controls a comparison.
type Options struct {
	Scope        canon.Scope
	ContextLines int
}

// stripCode removes the declared code paths from a canonical payload, so the
// structural diff covers metadata only.
func stripCode(d unwrap.Descriptor, canonical []byte) ([]byte, error) {
	var node any
	if err := json.Unmarshal(canonical, &node); err != nil {
		return nil, err
	}
	meta, _ := node.(map[string]any)
	cleaned, _, _ := d.Contract(meta).Extract(node, nil)
	return json.Marshal(cleaned)
}

// compareCode extracts the code from both payloads and diffs each file.
func compareCode(d unwrap.Descriptor, before, after json.RawMessage, contextLines int) []FileDiff {
	beforeFiles := extractCode(d, before)
	afterFiles := extractCode(d, after)

	names := map[string]bool{}
	for name := range beforeFiles {
		names[name] = true
	}
	for name := range afterFiles {
		names[name] = true
	}

	var out []FileDiff
	for name := range names {
		b, inBefore := beforeFiles[name]
		a, inAfter := afterFiles[name]
		switch {
		case inBefore && !inAfter:
			out = append(out, FileDiff{File: name, OnlyIn: "remote", Removed: len(splitLines(b))})
		case !inBefore && inAfter:
			out = append(out, FileDiff{File: name, OnlyIn: "local", Added: len(splitLines(a))})
		default:
			if fd := Text(name, b, a, contextLines); fd.Changed() {
				out = append(out, fd)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}

func extractCode(d unwrap.Descriptor, payload json.RawMessage) map[string]string {
	var node any
	if json.Unmarshal(payload, &node) != nil {
		return nil
	}
	meta, _ := node.(map[string]any)
	_, files, _ := d.Contract(meta).Extract(node, nil)
	return files
}

// RenderText writes one artifact's comparison for a human.
func (r Result) RenderText(nameOnly bool) string {
	var b strings.Builder

	marker := stateMarker(r.State)
	title := r.Name
	if title == "" {
		title = r.Dir
	}
	fmt.Fprintf(&b, "%s %s", marker, title)
	if r.ID != "" {
		fmt.Fprintf(&b, "  (%s %s)", r.Kind, r.ID)
	}
	if r.State != "" && r.State != "unknown" {
		fmt.Fprintf(&b, "  [%s]", r.State)
	}
	b.WriteString("\n")

	if nameOnly {
		return b.String()
	}

	if len(r.Metadata) > 0 {
		b.WriteString("  metadata:\n")
		b.WriteString(Render(r.Metadata, "    "))
	}
	for _, f := range r.Code {
		switch f.OnlyIn {
		case "local":
			fmt.Fprintf(&b, "  %s  (local only, +%d lines)\n", f.File, f.Added)
			continue
		case "remote":
			fmt.Fprintf(&b, "  %s  (remote only, -%d lines)\n", f.File, f.Removed)
			continue
		}
		fmt.Fprintf(&b, "  %s  +%d −%d\n", f.File, f.Added, f.Removed)
		b.WriteString(f.Unified)
	}
	if r.Origin != "" {
		fmt.Fprintf(&b, "  pulled from %s\n", r.Origin)
	}
	if r.Ignored != "" {
		fmt.Fprintf(&b, "  %s\n", r.Ignored)
	}
	return b.String()
}

// stateMarker maps a state name to its single-character marker.
func stateMarker(state string) string {
	switch state {
	case "local changes":
		return basestate.LocalChanges.Marker()
	case "remote changes":
		return basestate.RemoteChanges.Marker()
	case "conflict":
		return basestate.Conflict.Marker()
	case "clean":
		return basestate.Clean.Marker()
	default:
		return basestate.Unknown.Marker()
	}
}
