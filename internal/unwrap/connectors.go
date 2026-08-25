package unwrap

import (
	"encoding/json"
	"fmt"
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

	cleaned, jsFiles := ExtractJSFields(node)

	m, _ := node.(map[string]any)
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
func WrapConnectorFunction(dir string) (json.RawMessage, error) {
	return wrapFlatArtifact(dir, connectorFunctionMetaFile, "connector function")
}
