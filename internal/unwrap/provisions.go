package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// provisionProcessorMetaFile is the metadata file produced/consumed by the
// provision processor wrap/unwrap cycle.
const provisionProcessorMetaFile = "provisionfunction.json"

// UnwrapProvisionProcessor explodes a provision processor JSON into an editable
// directory:
//
//	<dir>/<pp-slug>/
//	  provisionfunction.json — metadata (the scriptProcessor.script field stripped)
//	  scriptProcessor__script.js — provision processor code
//
// Returns the created provision processor directory.
func UnwrapProvisionProcessor(raw json.RawMessage, dir string) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing provision processor: %w", err)
	}

	cleaned, jsFiles := ExtractJSFields(node)

	m, _ := node.(map[string]any)
	name, _ := m["name"].(string)
	id, _ := m["provisionProcessorId"].(string)
	slug := DedupedSlug(name, id, map[string]bool{})

	ppDir := filepath.Join(dir, slug)
	if err := os.MkdirAll(ppDir, 0o755); err != nil {
		return "", err
	}

	if err := writeJSON(filepath.Join(ppDir, provisionProcessorMetaFile), cleaned); err != nil {
		return "", err
	}
	for filename, code := range jsFiles {
		if err := os.WriteFile(filepath.Join(ppDir, filename), []byte(code), 0o644); err != nil {
			return "", err
		}
	}
	return ppDir, nil
}

// WrapProvisionProcessor rebuilds the provision processor JSON from an unwrapped
// directory, reinjecting every .js file at its original keypath.
func WrapProvisionProcessor(dir string) (json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(dir, provisionProcessorMetaFile))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", provisionProcessorMetaFile, err)
	}

	var node any
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", provisionProcessorMetaFile, err)
	}

	jsFiles := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
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

	node = ReinjectJSFields(node, jsFiles)

	out, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling provision processor: %w", err)
	}
	return out, nil
}
