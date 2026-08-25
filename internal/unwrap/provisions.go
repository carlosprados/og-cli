package unwrap

import (
	"encoding/json"
	"fmt"
)

// provisionProcessorMetaFile is the metadata file produced/consumed by the
// provision processor wrap/unwrap cycle.
const provisionProcessorMetaFile = "provisionfunction.json"

// UnwrapProvisionProcessor explodes a provision processor JSON into an editable
// directory:
//
//	<dir>/<pp-slug>/
//	  provisionfunction.json     — metadata (the scriptProcessor.script field stripped)
//	  scriptProcessor__script.js — provision processor code
//
// Returns the created provision processor directory. Pass one shared Options
// across a batch so slug deduplication works.
func UnwrapProvisionProcessor(raw json.RawMessage, dir string, opts *Options) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing provision processor: %w", err)
	}

	cleaned, jsFiles := ExtractJSFields(node)

	m, _ := node.(map[string]any)
	name, _ := m["name"].(string)
	id, _ := m["provisionProcessorId"].(string)

	ppDir, err := opts.claim(name, id, dir)
	if err != nil {
		return "", err
	}
	if err := writeArtifact(ppDir, provisionProcessorMetaFile, cleaned, jsFiles); err != nil {
		return "", err
	}
	return ppDir, nil
}

// WrapProvisionProcessor rebuilds the provision processor JSON from an unwrapped
// directory, reinjecting every .js file at its original keypath.
func WrapProvisionProcessor(dir string) (json.RawMessage, error) {
	return wrapFlatArtifact(dir, provisionProcessorMetaFile, "provision processor")
}
