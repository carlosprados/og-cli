package unwrap

import (
	"encoding/json"
	"fmt"
)

const ruleMetaFile = "rule.json"

// UnwrapRule explodes a rule JSON into an editable directory:
//
//	<dir>/<rule-slug>/
//	  rule.json        — rule metadata (JS fields stripped)
//	  javascript.js    — ADVANCED rule code (keypath naming, like widgets)
//
// EASY rules produce only rule.json. Returns the created rule directory.
// Pass one shared Options across a batch so slug deduplication works.
func UnwrapRule(raw json.RawMessage, dir string, opts *Options) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing rule: %w", err)
	}

	contract := RuleContract()
	cleaned, jsFiles, _ := contract.Extract(node, opts.Warn)

	m, _ := node.(map[string]any)
	name, _ := m["name"].(string)
	id, _ := m["identifier"].(string)

	ruleDir, err := opts.claim(name, id, dir)
	if err != nil {
		return "", err
	}
	if err := writeArtifact(ruleDir, ruleMetaFile, cleaned, jsFiles); err != nil {
		return "", err
	}
	return ruleDir, nil
}

// WrapRule rebuilds the rule JSON from an unwrapped rule directory,
// reinjecting every .js file at its original keypath.
func WrapRule(dir string, warn WarnFunc) (json.RawMessage, error) {
	return wrapFlatArtifact(dir, ruleMetaFile, "rule", RuleContract(), warn)
}
