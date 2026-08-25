package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// This file holds the shared pull/wrap cycle for "flat" artifact families —
// rules, connector functions and provision functions. Each is a single JSON
// document with its code in one or more string fields, so they differ only in
// their metadata filename and in which keys carry the name and the identifier.
// The nested workspace → dashboard → widget family lives in unwrap.go/wrap.go.

// Options controls how a single flat artifact is exploded onto disk.
//
// Taken carries the slugs already claimed during the current run. Callers that
// unwrap more than one artifact — pull-all — MUST reuse one Options across
// every call, otherwise DedupedSlug cannot see the collision and two artifacts
// with the same name resolve to the same directory.
type Options struct {
	Taken map[string]bool
	Force bool // overwrite a destination left by a previous run

	// Warn receives extraction warnings — an undeclared field that looks like
	// code, a stray .js file wrap will not deploy. Nil is silent.
	Warn WarnFunc
}

// claim resolves the destination directory for an artifact, reserving its slug
// in o.Taken and rejecting a pre-existing destination unless Force is set.
//
// Resolving the destination here — rather than in each cmd/ helper — keeps a
// single slug computation per artifact. Two independent computations that had
// to agree is how the pull-all collision bug arose.
func (o *Options) claim(name, id, dir string) (string, error) {
	if o.Taken == nil {
		o.Taken = make(map[string]bool)
	}
	slug := DedupedSlug(name, id, o.Taken)
	target := filepath.Join(dir, slug)
	if _, err := os.Stat(target); err == nil && !o.Force {
		return "", fmt.Errorf("destination %s already exists (use --force to overwrite)", target)
	}
	return target, nil
}

// writeArtifact writes the metadata file and every extracted JS file into an
// already-resolved artifact directory.
func writeArtifact(dir, metaFile string, cleaned any, jsFiles map[string]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	if err := writeJSON(filepath.Join(dir, metaFile), cleaned); err != nil {
		return err
	}
	for filename, code := range jsFiles {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(code), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}
	return nil
}

// readJSFiles loads every *.js file sitting directly in dir, keyed by filename.
func readJSFiles(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	jsFiles := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		code, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		jsFiles[e.Name()] = string(code)
	}
	return jsFiles, nil
}

// wrapFlatArtifact rebuilds a flat artifact's JSON from its directory,
// reinjecting the code files the contract declares. kind names the family in
// error messages.
func wrapFlatArtifact(dir, metaFile, kind string, contract CodeContract, warn WarnFunc) (json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(dir, metaFile))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", metaFile, err)
	}

	var node any
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", metaFile, err)
	}

	jsFiles, err := readJSFiles(dir)
	if err != nil {
		return nil, err
	}
	node = contract.Reinject(node, jsFiles, warn)

	out, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling %s: %w", kind, err)
	}
	return out, nil
}
