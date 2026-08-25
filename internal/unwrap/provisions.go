package unwrap

import "encoding/json"

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
	return ProvisionFunctionDescriptor().Unwrap(raw, dir, opts)
}

// WrapProvisionProcessor rebuilds the provision processor JSON from an unwrapped
// directory, reinjecting its code file.
func WrapProvisionProcessor(dir string, warn WarnFunc) (json.RawMessage, error) {
	return ProvisionFunctionDescriptor().Wrap(dir, warn)
}
