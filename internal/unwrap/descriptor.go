package unwrap

import (
	"encoding/json"
	"fmt"
)

// Kind identifies a family of platform artifacts that supports the
// pull → edit → deploy lifecycle.
type Kind string

const (
	KindWorkspace         Kind = "workspace"
	KindDashboard         Kind = "dashboard"
	KindRule              Kind = "rule"
	KindConnectorFunction Kind = "connector-function"
	KindProvisionFunction Kind = "provision-function"
)

// Descriptor declares everything that distinguishes one flat artifact family
// from another: a single JSON document whose code lives in known fields.
//
// Rules, connector functions and provision functions differ in three literals
// and nothing else. Before this they were three copies of the same forty lines,
// which is how the pull-all slug bug came to exist in triplicate. They are a
// table, not a class hierarchy, so the lifecycle is written once against this
// struct and each family is a declaration.
//
// The nested workspace → dashboard → widget family is genuinely different — it
// walks a tree and splits each widget into its own directory — and keeps its
// own implementation in unwrap.go/wrap.go rather than being forced through
// this shape.
type Descriptor struct {
	Kind Kind

	// MetaFile is the JSON file holding everything but the code.
	MetaFile string

	// NameKeys are the payload keys that may carry the display name, in
	// priority order: connector functions answer to `name` on write and
	// sometimes `connectorFunctionName` on read.
	NameKeys []string

	// IDKey is the payload key holding the identifier. It is not `identifier`
	// everywhere: a provision processor uses `provisionProcessorId`.
	IDKey string

	// Contract reports where this artifact keeps its code. It takes the decoded
	// payload because the answer can depend on it — a connector function's
	// execution context follows its `type` field.
	Contract func(meta map[string]any) CodeContract
}

// RuleDescriptor describes an automation rule. EASY rules carry no code; the
// declared path is simply absent.
func RuleDescriptor() Descriptor {
	return Descriptor{
		Kind:     KindRule,
		MetaFile: "rule.json",
		NameKeys: []string{"name"},
		IDKey:    "identifier",
		Contract: func(map[string]any) CodeContract { return RuleContract() },
	}
}

// ConnectorFunctionDescriptor describes a connector function.
func ConnectorFunctionDescriptor() Descriptor {
	return Descriptor{
		Kind:     KindConnectorFunction,
		MetaFile: "connectorfunction.json",
		NameKeys: []string{"name", "connectorFunctionName"},
		IDKey:    "identifier",
		Contract: func(meta map[string]any) CodeContract {
			cfType, _ := meta["type"].(string)
			return ConnectorFunctionContract(cfType)
		},
	}
}

// ProvisionFunctionDescriptor describes a provision processor.
func ProvisionFunctionDescriptor() Descriptor {
	return Descriptor{
		Kind:     KindProvisionFunction,
		MetaFile: "provisionfunction.json",
		NameKeys: []string{"name"},
		IDKey:    "provisionProcessorId",
		Contract: func(map[string]any) CodeContract { return ProvisionFunctionContract() },
	}
}

// NameOf returns the artifact's display name, trying each name key in order.
// Exported so callers can name an artifact in a message without re-deriving
// which key holds its name.
func (d Descriptor) NameOf(raw json.RawMessage) string {
	var node any
	if json.Unmarshal(raw, &node) != nil {
		return ""
	}
	meta, _ := node.(map[string]any)
	return d.nameOf(meta)
}

// nameOf returns the first non-empty name key.
func (d Descriptor) nameOf(meta map[string]any) string {
	for _, key := range d.NameKeys {
		if s, ok := meta[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func (d Descriptor) idOf(meta map[string]any) string {
	s, _ := meta[d.IDKey].(string)
	return s
}

// Unwrap explodes one artifact of this family into its own directory under dir,
// writing the metadata file and one file per declared code path. It returns the
// directory created.
//
// Pass one shared Options across a batch: that is what lets DedupedSlug see the
// slugs an artifact's siblings have already claimed.
func (d Descriptor) Unwrap(raw json.RawMessage, dir string, opts *Options) (string, error) {
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return "", fmt.Errorf("parsing %s: %w", d.Kind, err)
	}
	meta, _ := node.(map[string]any)

	cleaned, codeFiles, _ := d.Contract(meta).Extract(node, opts.Warn)

	target, err := opts.claim(d.nameOf(meta), d.idOf(meta), dir)
	if err != nil {
		return "", err
	}
	if err := writeArtifact(target, d.MetaFile, cleaned, codeFiles); err != nil {
		return "", err
	}
	return target, nil
}

// Wrap rebuilds the artifact's payload from its directory, reinjecting each
// declared code file at its keypath. It is the inverse of Unwrap.
func (d Descriptor) Wrap(dir string, warn WarnFunc) (json.RawMessage, error) {
	data, err := readMeta(dir, d.MetaFile)
	if err != nil {
		return nil, err
	}

	var node any
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", d.MetaFile, err)
	}
	meta, _ := node.(map[string]any)

	codeFiles, err := readJSFiles(dir)
	if err != nil {
		return nil, err
	}
	node = d.Contract(meta).Reinject(node, codeFiles, warn)

	out, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling %s: %w", d.Kind, err)
	}
	return out, nil
}
