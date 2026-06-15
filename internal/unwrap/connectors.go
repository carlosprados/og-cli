package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// connectorFunctionMetaFile is the metadata file produced/consumed by the
// connector function wrap/unwrap cycle.
const connectorFunctionMetaFile = "connectorfunction.json"

// UnwrapConnectorFunction explodes a connector function JSON into an editable
// directory:
//
//	<dir>/<cf-slug>/
//	  connectorfunction.json — metadata (the javascript field stripped)
//	  javascript.js          — connector function code
//
// Returns the created connector function directory.
func UnwrapConnectorFunction(raw json.RawMessage, dir string) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing connector function: %w", err)
	}

	cleaned, jsFiles := ExtractJSFields(node)

	m, _ := node.(map[string]any)
	name, _ := m["name"].(string)
	if name == "" {
		name, _ = m["connectorFunctionName"].(string)
	}
	id, _ := m["identifier"].(string)
	slug := DedupedSlug(name, id, map[string]bool{})

	cfDir := filepath.Join(dir, slug)
	if err := os.MkdirAll(cfDir, 0o755); err != nil {
		return "", err
	}

	if err := writeJSON(filepath.Join(cfDir, connectorFunctionMetaFile), cleaned); err != nil {
		return "", err
	}
	for filename, code := range jsFiles {
		if err := os.WriteFile(filepath.Join(cfDir, filename), []byte(code), 0o644); err != nil {
			return "", err
		}
	}
	return cfDir, nil
}

// WrapConnectorFunction rebuilds the connector function JSON from an unwrapped
// directory, reinjecting every .js file at its original keypath.
func WrapConnectorFunction(dir string) (json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(dir, connectorFunctionMetaFile))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", connectorFunctionMetaFile, err)
	}

	var node any
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", connectorFunctionMetaFile, err)
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
		return nil, fmt.Errorf("marshaling connector function: %w", err)
	}
	return out, nil
}
