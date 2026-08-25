package unwrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
// Returns the created connector function directory. Pass one shared Options
// across a batch so slug deduplication works.
func UnwrapConnectorFunction(raw json.RawMessage, dir string, opts *Options) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing connector function: %w", err)
	}

	m, _ := node.(map[string]any)
	cfType, _ := m["type"].(string)
	contract := ConnectorFunctionContract(cfType)
	cleaned, jsFiles, _ := contract.Extract(node, opts.Warn)

	name, _ := m["name"].(string)
	if name == "" {
		name, _ = m["connectorFunctionName"].(string)
	}
	id, _ := m["identifier"].(string)

	cfDir, err := opts.claim(name, id, dir)
	if err != nil {
		return "", err
	}
	if err := writeArtifact(cfDir, connectorFunctionMetaFile, cleaned, jsFiles); err != nil {
		return "", err
	}
	return cfDir, nil
}

// WrapConnectorFunction rebuilds the connector function JSON from an unwrapped
// directory, reinjecting every .js file at its original keypath.
func WrapConnectorFunction(dir string, warn WarnFunc) (json.RawMessage, error) {
	return wrapFlatArtifact(dir, connectorFunctionMetaFile, "connector function", ConnectorFunctionContract(cfTypeOf(dir)), warn)
}

// cfTypeOf reads the connector function's type from its metadata file, so wrap
// resolves the same execution context that pull recorded. An unreadable file is
// not this function's problem: wrapFlatArtifact reports it.
func cfTypeOf(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, connectorFunctionMetaFile))
	if err != nil {
		return ""
	}
	var meta struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	return meta.Type
}
