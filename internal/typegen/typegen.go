// Package typegen writes TypeScript declaration files for the platform globals
// in scope inside an OpenGate artifact's JavaScript.
//
// Two halves, deliberately separated:
//
//   - The platform catalogue (functions, the entity value shape, alarm
//     severities) is static per execution context and lives in an embedded
//     template. It changes with platform releases, not with the user.
//   - The organization's datastream identifiers and their value types come
//     from its datamodel, so `entity['sensro.temperature']` is flagged in the
//     editor before it is ever deployed. No generic editor or LLM can produce
//     this half: only the platform knows it.
//
// tsserver picks the result up through a generated jsconfig.json, which is why
// this delivers completion and diagnostics in Neovim, VS Code, Cursor and Zed
// with no editor-specific code.
package typegen

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
)

//go:embed templates/*.d.ts
var templates embed.FS

// Context identifies the execution environment whose globals are declared.
type Context string

const (
	// ContextRuleAdvanced is the body of an ADVANCED automation rule.
	ContextRuleAdvanced Context = "rule/ADVANCED"
	// ContextCFRequest transforms an outgoing operation request.
	ContextCFRequest Context = "connector-function/REQUEST"
	// ContextCFResponse processes a device's response to an operation.
	ContextCFResponse Context = "connector-function/RESPONSE"
	// ContextCFCollection processes collected data into datapoints.
	ContextCFCollection Context = "connector-function/COLLECTION"
)

// templateFor maps a context to its embedded platform catalogue. The three
// connector function types share one: the objects that differ between them are
// declared in all three, since declaring too little is what makes typings redden
// working code.
var templateFor = map[Context]string{
	ContextRuleAdvanced: "templates/rule-advanced.d.ts",
	ContextCFRequest:    "templates/connector-function.d.ts",
	ContextCFResponse:   "templates/connector-function.d.ts",
	ContextCFCollection: "templates/connector-function.d.ts",
}

// protocolTemplate holds the per-protocol objects, appended for connector
// function contexts.
const protocolTemplate = "templates/cf-protocols.d.ts"

// isConnectorFunction reports whether a context is one of the connector
// function types.
func isConnectorFunction(c Context) bool {
	switch c {
	case ContextCFRequest, ContextCFResponse, ContextCFCollection:
		return true
	}
	return false
}

// ContextForConnectorFunction maps a connector function's `type` field to its
// context.
func ContextForConnectorFunction(cfType string) Context {
	switch strings.ToUpper(cfType) {
	case "REQUEST":
		return ContextCFRequest
	case "RESPONSE":
		return ContextCFResponse
	case "COLLECTION":
		return ContextCFCollection
	default:
		return ContextCFCollection
	}
}

// Contexts lists the contexts typegen can emit, for CLI help and validation.
func Contexts() []string {
	out := make([]string, 0, len(templateFor))
	for c := range templateFor {
		out = append(out, string(c))
	}
	sort.Strings(out)
	return out
}

// platformDatastreams are present on every entity regardless of datamodel.
// Taken from the entity example in the platform guide.
var platformDatastreams = []struct {
	ID      string
	Type    string
	Indexed bool
}{
	{ID: "provision.administration.identifier", Type: "string"},
	{ID: "provision.administration.organization", Type: "string"},
	{ID: "provision.administration.channel", Type: "string"},
	{ID: "provision.administration.plan", Type: "string"},
	{ID: "provision.administration.serviceGroup", Type: "string"},
	{ID: "provision.device.identifier", Type: "string"},
	{ID: "device.identifier", Type: "string"},
	{ID: "provision.device.communicationModules[].identifier", Type: "string", Indexed: true},
	{ID: "provision.device.communicationModules[].subscription.address", Type: "unknown", Indexed: true},
}

// Options controls a generation run.
type Options struct {
	Context Context

	// Datamodels are the organization's datamodels, whose datastreams become
	// the valid keys of the entity object. Plural because a real tenant has
	// many — sensehat has 27, and a device's entity can carry datastreams from
	// any of them — so typing against one would flag correct code as wrong.
	// Empty means only platform datastreams are declared.
	Datamodels []opengate.Datamodel
	OrgName    string
	// Parameters are the rule's declared parameters, when generating for a
	// specific rule rather than for a context in general.
	Parameters []Parameter

	// Protocols are the connector protocols this function is reached through,
	// for the header. Every protocol object is declared regardless: which ones
	// are really in scope depends on the protocol, and declaring too few would
	// flag working code.
	Protocols []string

	// ExtraDatastreams are identifiers the artifact itself references but that
	// no datamodel declares.
	//
	// This is not a nicety. Verified in production: sensehat has a live rule
	// triggering on `temperature.from.pressure`, which appears in no datamodel
	// at all. Declaring only what the datamodels hold would flag that rule's
	// working code as an error, and typings that redden correct code get
	// deleted. Sources are the artifact's own trigger and the identifiers its
	// code already reads.
	ExtraDatastreams []string
	// Version is the og version stamped in the header, for traceability.
	Version string
}

// Generate returns the contents of og-globals.d.ts.
func Generate(o Options) (string, error) {
	tmplPath, ok := templateFor[o.Context]
	if !ok {
		return "", fmt.Errorf("no typings for context %q (known: %s)", o.Context, strings.Join(Contexts(), ", "))
	}
	catalogue, err := templates.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("reading embedded template: %w", err)
	}

	var b strings.Builder
	writeHeader(&b, o)
	b.WriteString(string(catalogue))
	b.WriteString("\n")

	if isConnectorFunction(o.Context) {
		protocols, err := templates.ReadFile(protocolTemplate)
		if err != nil {
			return "", fmt.Errorf("reading protocol template: %w", err)
		}
		b.WriteString(string(protocols))
		b.WriteString("\n")
	}
	writeEntity(&b, o)
	writeParameters(&b, o.Parameters)
	return b.String(), nil
}

func writeHeader(b *strings.Builder, o Options) {
	fmt.Fprintf(b, "// og-globals.d.ts — generated by `og typegen`. DO NOT EDIT.\n")
	fmt.Fprintf(b, "// context:   %s\n", o.Context)
	if o.Version != "" {
		fmt.Fprintf(b, "// generator: og %s\n", o.Version)
	}
	switch len(o.Datamodels) {
	case 0:
		b.WriteString("// datamodel: none — only platform datastreams are declared\n")
	case 1:
		dm := o.Datamodels[0]
		fmt.Fprintf(b, "// datamodel: %s", dm.Identifier)
		if dm.Version != "" {
			fmt.Fprintf(b, " v%s", dm.Version)
		}
		if o.OrgName != "" {
			fmt.Fprintf(b, " (organization %s)", o.OrgName)
		}
		b.WriteString("\n")
	default:
		names := make([]string, 0, len(o.Datamodels))
		for _, dm := range o.Datamodels {
			names = append(names, dm.Identifier)
		}
		sort.Strings(names)
		fmt.Fprintf(b, "// datamodels: %d", len(names))
		if o.OrgName != "" {
			fmt.Fprintf(b, " in organization %s", o.OrgName)
		}
		fmt.Fprintf(b, "\n//            %s\n", strings.Join(names, ", "))
	}
	source := "rules-js-reference.md"
	if isConnectorFunction(o.Context) {
		source = "connector-functions-js-reference.md"
	}
	fmt.Fprintf(b, "// source:    .claude/skills/og-device-ops/%s\n", source)
	if len(o.Protocols) > 0 {
		fmt.Fprintf(b, "// protocols: %s (from the south criteria)\n", strings.Join(o.Protocols, ", "))
	}
	fmt.Fprintf(b, "//\n// Regenerate after a datamodel change: og typegen --context %q --org <org>\n\n", o.Context)
}

// datastreamEntry is one declared key of the entity object.
type datastreamEntry struct {
	ID      string
	TSType  string
	Indexed bool
	Doc     string
}

// writeEntity emits the entity object type and the DatastreamID union.
func writeEntity(b *strings.Builder, o Options) {
	entries := make([]datastreamEntry, 0, len(platformDatastreams))
	for _, ds := range platformDatastreams {
		entries = append(entries, datastreamEntry{ID: ds.ID, TSType: ds.Type, Indexed: ds.Indexed, Doc: "platform datastream"})
	}
	entries = append(entries, datamodelEntries(o.Datamodels)...)
	entries = append(entries, extraEntries(o.ExtraDatastreams, entries)...)

	b.WriteString("// ── Datastreams ─────────────────────────────────────────────────────────────\n\n")
	b.WriteString("/** Every datastream identifier declared for this organization, plus the\n")
	b.WriteString(" *  platform ones. A typo is a compile error rather than a runtime undefined. */\n")
	b.WriteString("type OGDatastreamID =\n")
	for i, e := range entries {
		sep := "|"
		if i == 0 {
			sep = " "
		}
		fmt.Fprintf(b, "  %s %s\n", sep, quote(e.ID))
	}
	b.WriteString("  ;\n\n")

	b.WriteString("/** The flattened entity the rule receives. Keys are optional: a datastream\n")
	b.WriteString(" *  the device has never reported is absent, so guard before reading. */\n")
	b.WriteString("interface OGEntity {\n")
	for _, e := range entries {
		if e.Doc != "" {
			fmt.Fprintf(b, "  /** %s */\n", e.Doc)
		}
		decl := fmt.Sprintf("OGDatastream<%s>", e.TSType)
		if e.Indexed {
			decl = fmt.Sprintf("Array<OGIndexedDatastream<%s>>", e.TSType)
		}
		fmt.Fprintf(b, "  %s?: %s;\n", quote(e.ID), decl)
	}
	b.WriteString("}\n\n")
	// Besides its datastreams, the entity carries plain properties. Production
	// rules read entity.resourceType and entity.device, so the declared type is
	// the datastream map plus those.
	b.WriteString("/** Properties the entity carries besides its datastreams. */\n")
	b.WriteString("interface OGEntityProperties {\n")
	b.WriteString("  /** e.g. 'entity.device'. */\n")
	b.WriteString("  resourceType?: OGDatastream<string>;\n")
	b.WriteString("  device?: any;\n")
	b.WriteString("}\n\n")
	b.WriteString("declare const entity: OGEntity & OGEntityProperties;\n")
	b.WriteString("/** The gateway entity, when the rule runs behind one. */\n")
	b.WriteString("declare const gateway: (OGEntity & OGEntityProperties) | null;\n")
}

// datamodelEntries turns the organization's datamodels into entity keys,
// merging them.
//
// The same datastream identifier can appear in several datamodels. When their
// declared types agree, the merged entry keeps that type. When they disagree the
// entry falls back to `unknown` and says so in its doc comment: asserting one of
// the two would flag correct code that used the other.
func datamodelEntries(dms []opengate.Datamodel) []datastreamEntry {
	type merged struct {
		entry    datastreamEntry
		types    map[string]bool
		sources  []string
		conflict bool
	}
	byID := map[string]*merged{}

	for _, dm := range dms {
		for _, cat := range dm.Categories {
			for _, ds := range cat.Datastreams {
				if ds.Identifier == "" {
					continue
				}
				tsType := tsTypeOf(ds.Schema)
				m, seen := byID[ds.Identifier]
				if !seen {
					byID[ds.Identifier] = &merged{
						entry: datastreamEntry{
							ID:      ds.Identifier,
							TSType:  tsType,
							Indexed: strings.Contains(ds.Identifier, "[]"),
							Doc:     docFor(ds, cat),
						},
						types:   map[string]bool{tsType: true},
						sources: []string{dm.Identifier},
					}
					continue
				}
				m.types[tsType] = true
				if len(m.types) > 1 {
					m.conflict = true
				}
				if !contains(m.sources, dm.Identifier) {
					m.sources = append(m.sources, dm.Identifier)
				}
				// Keep the first non-empty doc: an identifier documented in one
				// datamodel and bare in another should keep the documentation.
				if m.entry.Doc == "" {
					m.entry.Doc = docFor(ds, cat)
				}
			}
		}
	}

	out := make([]datastreamEntry, 0, len(byID))
	for _, m := range byID {
		e := m.entry
		if m.conflict {
			e.TSType = "any"
			e.Doc = strings.TrimSpace(e.Doc + " — declared with conflicting types across datamodels, so left untyped")
		}
		if len(m.sources) > 1 {
			sort.Strings(m.sources)
			e.Doc = strings.TrimSpace(e.Doc) + " [in: " + strings.Join(m.sources, ", ") + "]"
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// docFor builds the doc comment for a datastream: its name, description,
// unit and category, whichever are present.
func docFor(ds opengate.Datastream, cat opengate.Category) string {
	parts := []string{}
	if ds.Name != "" {
		parts = append(parts, ds.Name)
	}
	if ds.Description != "" {
		parts = append(parts, ds.Description)
	}
	if ds.Unit != nil && ds.Unit.Symbol != "" {
		parts = append(parts, "["+ds.Unit.Symbol+"]")
	}
	if cat.Name != "" {
		parts = append(parts, "(category: "+cat.Name+")")
	}
	return strings.Join(parts, " — ")
}

// tsTypeOf maps a datastream's JSON Schema to a TypeScript type. Anything
// beyond the primitives stays `unknown`: a wrong type would be worse than an
// unspecific one, since the editor would flag correct code.
func tsTypeOf(schema json.RawMessage) string {
	if len(schema) == 0 {
		return "any"
	}
	var s struct {
		Type any `json:"type"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		return "any"
	}
	switch t := s.Type.(type) {
	case string:
		return tsPrimitive(t)
	case []any:
		// A union of JSON Schema types, e.g. ["number","null"].
		var parts []string
		for _, v := range t {
			if name, ok := v.(string); ok {
				parts = append(parts, tsPrimitive(name))
			}
		}
		if len(parts) == 0 {
			return "any"
		}
		return strings.Join(dedupe(parts), " | ")
	default:
		return "any"
	}
}

// tsPrimitive maps a JSON Schema type name to a TypeScript one.
//
// An unmapped type becomes `any`, not `unknown`. `unknown` is the safer choice
// in TypeScript, but it cannot be compared or arithmetic'd without a cast, and
// production rules do exactly that: `entity['ccare.bps']._value._current.value >
// threshold` is correct JavaScript that `unknown` rejects. Since the point of
// these declarations is to catch mistyped identifiers rather than to enforce
// value types, `any` is the right trade.
func tsPrimitive(jsonType string) string {
	switch jsonType {
	case "number", "integer":
		return "number"
	case "string":
		return "string"
	case "boolean":
		return "boolean"
	case "null":
		return "null"
	case "array":
		return "unknown[]"
	default:
		return "any"
	}
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// quote renders a TypeScript string literal, single-quoted like the rest of the
// generated file.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
}

// JSConfigFor returns the jsconfig.json to write alongside an artifact's code.
//
// Two compiler options carry the diagnostic value:
//
//	checkJs        without it the declarations give completion but no
//	               diagnostics, which is half the point.
//	noImplicitAny  this is what makes entity['sensro.temperature'] an error.
//	               Without it TypeScript quietly types an unknown index as any
//	               and the most useful check — a mistyped datastream identifier
//	               — never fires.
//
// But an artifact's script is not a standalone program. The platform wraps it in
// a function, so a top-level `return` is correct and TypeScript reports TS1108
// on it, and production scripts declare helper functions whose parameters carry
// no types, which noImplicitAny reports as TS7006. Both are errors on working
// code, and typings that redden working code get deleted.
//
// So the config adapts to the code it will check. Where the code cannot be
// type-checked without false positives, checking is turned off and completion is
// kept — the declarations still drive autocomplete, navigation and signature
// help, which is most of the day-to-day value.
//
// Verified against sensehat's seven live connector functions: six of the seven
// contain a top-level return or an untyped helper parameter.
func JSConfigFor(code string) string {
	strict := !hasTopLevelReturn(code) && !hasUntypedFunctionParams(code) && !hasDynamicDatastreamAccess(code)

	if strict {
		return `{
  "compilerOptions": {
    "target": "es5",
    "lib": ["es5"],
    "allowJs": true,
    "checkJs": true,
    "noEmit": true,
    "strict": false,
    "noImplicitAny": true
  },
  "include": ["*.js", "og-globals.d.ts"]
}
`
	}

	// completion only: this script cannot be checked without reporting errors on
	// correct code (a top-level return, or an untyped helper parameter).
	return `{
  "compilerOptions": {
    "target": "es5",
    "lib": ["es5"],
    "allowJs": true,
    "checkJs": false,
    "noEmit": true,
    "strict": false
  },
  "include": ["*.js", "og-globals.d.ts"]
}
`
}

// JSConfig is the strict configuration, kept for callers with no code to inspect.
var JSConfig = JSConfigFor("")

// topLevelReturnPattern finds a `return` at the start of a line with no
// indentation, which cannot be inside a function body formatted normally.
var topLevelReturnPattern = regexp.MustCompile(`(?m)^return\b`)

// hasTopLevelReturn reports whether the script returns a value at its top level,
// which the platform allows — it wraps the script in a function — and TypeScript
// does not.
func hasTopLevelReturn(code string) bool {
	return topLevelReturnPattern.MatchString(code)
}

// untypedParamPattern finds a function declaration taking at least one
// parameter. In plain JavaScript none of them carry types, so noImplicitAny
// reports every one.
var untypedParamPattern = regexp.MustCompile(`function\s*[a-zA-Z0-9_$]*\s*\(\s*[a-zA-Z_$]`)

func hasUntypedFunctionParams(code string) bool {
	return untypedParamPattern.MatchString(code)
}

// dynamicIndexPattern finds entity[...] indexed by anything other than a string
// literal — a variable, or a call such as
// entity[getVariableValue(parameterObject['x'])], which a live rule does.
var dynamicIndexPattern = regexp.MustCompile(`(?:entity|gateway)\s*\[\s*[^'"\s\]]`)

// hasDynamicDatastreamAccess reports whether the code indexes the entity with a
// computed key. TypeScript cannot know what it resolves to, and reports it —
// on code that is correct.
func hasDynamicDatastreamAccess(code string) bool {
	return dynamicIndexPattern.MatchString(code)
}

// Parameter is one declared parameter of a rule, from its `parameters` array.
type Parameter struct {
	Name   string
	TSType string
	Doc    string
}

// ParametersFrom converts a rule's `parameters` array into typed entries. The
// rule declares each parameter's schema, so parameterObject can be typed as
// precisely as the entity — a mistyped parameter name becomes an error, and an
// arithmetic comparison against a numeric parameter type-checks.
func ParametersFrom(raw json.RawMessage) []Parameter {
	var rule struct {
		Parameters []struct {
			Name   string          `json:"name"`
			Schema json.RawMessage `json:"schema"`
			Value  json.RawMessage `json:"value"`
		} `json:"parameters"`
	}
	if json.Unmarshal(raw, &rule) != nil {
		return nil
	}

	out := make([]Parameter, 0, len(rule.Parameters))
	for _, p := range rule.Parameters {
		if p.Name == "" {
			continue
		}
		out = append(out, Parameter{
			Name:   p.Name,
			TSType: parameterTSType(p.Schema, p.Value),
			Doc:    parameterDoc(p.Value),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// parameterTSType reads the parameter's schema, which a rule states as a bare
// type name ("number") rather than as a JSON Schema object. Falls back to the
// default value's own JSON type.
func parameterTSType(schema, value json.RawMessage) string {
	var name string
	if json.Unmarshal(schema, &name) == nil && name != "" {
		return tsPrimitive(name)
	}
	// The schema is checked first, then the default value's own JSON type. The
	// comparison is against "any" because that is what tsTypeOf returns when it
	// cannot tell — getting this wrong silently skipped the value fallback.
	if t := tsTypeOf(schema); t != "any" {
		return t
	}
	return jsonValueTSType(value)
}

// jsonValueTSType infers a type from a literal default value.
func jsonValueTSType(value json.RawMessage) string {
	var v any
	if len(value) == 0 || json.Unmarshal(value, &v) != nil {
		return "any"
	}
	switch v.(type) {
	case float64:
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	default:
		return "any"
	}
}

func parameterDoc(value json.RawMessage) string {
	if len(value) == 0 {
		return ""
	}
	return "default: " + strings.TrimSpace(string(value))
}

// writeParameters emits the rule's parameter object type.
func writeParameters(b *strings.Builder, params []Parameter) {
	b.WriteString("\n// ── Rule parameters ─────────────────────────────────────────────────────────\n\n")
	if len(params) == 0 {
		b.WriteString("/** This rule declares no parameters. Typed permissively so reading one\n")
		b.WriteString(" *  added later is not an error — regenerate to get it declared. */\n")
		b.WriteString("declare const parameterObject: Record<string, any>;\n")
		return
	}
	b.WriteString("/** The rule's declared parameters, keyed by name. */\ninterface OGParameters {\n")
	for _, p := range params {
		if p.Doc != "" {
			fmt.Fprintf(b, "  /** %s */\n", p.Doc)
		}
		fmt.Fprintf(b, "  %s: %s;\n", quote(p.Name), p.TSType)
	}
	b.WriteString("}\n\ndeclare const parameterObject: OGParameters;\n")
}

// extraEntries declares identifiers the artifact references that no datamodel
// covers, skipping any the datamodels already declared.
//
// Their type is unknown: nothing states it. That is still worth declaring —
// the alternative is an error on code that works.
func extraEntries(ids []string, existing []datastreamEntry) []datastreamEntry {
	if len(ids) == 0 {
		return nil
	}
	known := make(map[string]bool, len(existing))
	for _, e := range existing {
		known[e.ID] = true
	}

	seen := map[string]bool{}
	var out []datastreamEntry
	for _, id := range ids {
		if id == "" || known[id] || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, datastreamEntry{
			ID:      id,
			TSType:  "any",
			Indexed: strings.Contains(id, "[]"),
			Doc:     "referenced by this artifact but declared in no datamodel — type unknown",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// datastreamPattern finds entity['...'] and gateway['...'] reads in artifact
// code, single or double quoted.
var datastreamPattern = regexp.MustCompile(`(?:entity|gateway)\s*\[\s*['"]([^'"]+)['"]\s*\]`)

// DatastreamsReferencedBy returns the identifiers an artifact's code reads.
//
// Used so the generated declarations cover what the artifact already does. A
// typo present in the code when the typings were generated is therefore
// declared and not flagged — the alternative, inventing errors in code that
// works, is worse. A typo written afterwards is still caught, which is the case
// that matters while editing.
func DatastreamsReferencedBy(code string) []string {
	matches := datastreamPattern.FindAllStringSubmatch(code, -1)
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		if len(m) < 2 || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

// DatastreamsTriggering returns the datastream identifiers a rule declares as
// its trigger, which the platform guarantees the entity will carry.
func DatastreamsTriggering(rulePayload json.RawMessage) []string {
	var rule struct {
		Type struct {
			Datastreams []struct {
				Name string `json:"name"`
			} `json:"datastreams"`
		} `json:"type"`
	}
	if json.Unmarshal(rulePayload, &rule) != nil {
		return nil
	}
	out := make([]string, 0, len(rule.Type.Datastreams))
	for _, ds := range rule.Type.Datastreams {
		if ds.Name != "" {
			out = append(out, ds.Name)
		}
	}
	sort.Strings(out)
	return out
}

// schemePattern captures the URI scheme of a south criterion.
var schemePattern = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.-]*)://`)

// ProtocolsFromCriteria reports the connector protocols a function is reached
// through, read from the scheme of each south criterion.
//
// Verified against sensehat: south criteria are URI strings whose scheme is the
// protocol — `mqtts://endesa`, `https://demo`. A REQUEST function has no south
// criteria at all (it matches on operationName plus north criteria), so its
// protocol cannot be determined from the payload; that is why every protocol
// object is declared rather than only the ones detected.
func ProtocolsFromCriteria(payload json.RawMessage) []string {
	var cf struct {
		SouthCriterias []string `json:"southCriterias"`
	}
	if json.Unmarshal(payload, &cf) != nil {
		return nil
	}

	seen := map[string]bool{}
	var out []string
	for _, criterion := range cf.SouthCriterias {
		m := schemePattern.FindStringSubmatch(criterion)
		if m == nil {
			continue
		}
		proto := normaliseScheme(m[1])
		if seen[proto] {
			continue
		}
		seen[proto] = true
		out = append(out, proto)
	}
	sort.Strings(out)
	return out
}

// normaliseScheme maps a URI scheme to the protocol object the platform injects:
// mqtts and mqtt both mean the mqtt object, https and http the http one.
func normaliseScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "mqtt", "mqtts":
		return "mqtt"
	case "http", "https":
		return "http"
	case "ws", "wss":
		return "websocket"
	case "coap", "coaps":
		return "coap"
	default:
		return strings.ToLower(scheme)
	}
}
