package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UnwrapRule explodes a rule JSON into an editable directory:
//
//	<dir>/<rule-slug>/
//	  rule.json        — rule metadata (JS fields stripped)
//	  javascript.js    — ADVANCED rule code (keypath naming, like widgets)
//
// EASY rules produce only rule.json. Returns the created rule directory.
func UnwrapRule(raw json.RawMessage, dir string) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing rule: %w", err)
	}

	cleaned, jsFiles := ExtractJSFields(node)

	m, _ := node.(map[string]any)
	name, _ := m["name"].(string)
	id, _ := m["identifier"].(string)
	slug := DedupedSlug(name, id, map[string]bool{})

	ruleDir := filepath.Join(dir, slug)
	if err := os.MkdirAll(ruleDir, 0o755); err != nil {
		return "", err
	}

	if err := writeJSON(filepath.Join(ruleDir, "rule.json"), cleaned); err != nil {
		return "", err
	}
	for filename, code := range jsFiles {
		if err := os.WriteFile(filepath.Join(ruleDir, filename), []byte(code), 0o644); err != nil {
			return "", err
		}
	}
	return ruleDir, nil
}

// WrapRule rebuilds the rule JSON from an unwrapped rule directory,
// reinjecting every .js file at its original keypath.
func WrapRule(dir string) (json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(dir, "rule.json"))
	if err != nil {
		return nil, fmt.Errorf("reading rule.json: %w", err)
	}

	var node any
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing rule.json: %w", err)
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
		return nil, fmt.Errorf("marshaling rule: %w", err)
	}
	return out, nil
}
