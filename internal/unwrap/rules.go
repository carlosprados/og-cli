package unwrap

import "encoding/json"

// UnwrapRule explodes a rule JSON into an editable directory:
//
//	<dir>/<rule-slug>/
//	  rule.json        — rule metadata (code fields stripped)
//	  javascript.js    — ADVANCED rule code
//
// EASY rules produce only rule.json. Returns the created rule directory.
// Pass one shared Options across a batch so slug deduplication works.
func UnwrapRule(raw json.RawMessage, dir string, opts *Options) (string, error) {
	return RuleDescriptor().Unwrap(raw, dir, opts)
}

// WrapRule rebuilds the rule JSON from an unwrapped rule directory,
// reinjecting its code file.
func WrapRule(dir string, warn WarnFunc) (json.RawMessage, error) {
	return RuleDescriptor().Wrap(dir, warn)
}
