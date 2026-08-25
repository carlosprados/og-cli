package unwrap

import "encoding/json"

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
	return ConnectorFunctionDescriptor().Unwrap(raw, dir, opts)
}

// WrapConnectorFunction rebuilds the connector function JSON from an unwrapped
// directory, reinjecting its code file.
func WrapConnectorFunction(dir string, warn WarnFunc) (json.RawMessage, error) {
	return ConnectorFunctionDescriptor().Wrap(dir, warn)
}
