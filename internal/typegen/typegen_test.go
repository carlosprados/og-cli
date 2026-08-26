package typegen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
)

func demoDatamodel(t *testing.T) []opengate.Datamodel {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "demo", "datamodels", "multisensor.json"))
	if err != nil {
		t.Fatalf("reading demo datamodel: %v", err)
	}
	var dm opengate.Datamodel
	if err := json.Unmarshal(raw, &dm); err != nil {
		t.Fatalf("parsing demo datamodel: %v", err)
	}
	return []opengate.Datamodel{dm}
}

// The organization's own datastream identifiers must reach the declarations —
// this is the half no generic editor or LLM can produce.
func TestGenerateDeclaresDatamodelDatastreams(t *testing.T) {
	out, err := Generate(Options{Context: ContextRuleAdvanced, Datamodels: demoDatamodel(t), OrgName: "sensehat"})
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"sensor.temperature", "sensor.humidity", "sensor.luminosity", "energy.consumption", "power.battery"} {
		if !strings.Contains(out, "'"+id+"'") {
			t.Errorf("datastream %q missing from the declarations", id)
		}
	}
	// The JSON Schema type must reach the declaration, not just the name.
	if !strings.Contains(out, "'sensor.temperature'?: OGDatastream<number>") {
		t.Error("sensor.temperature should be typed as a number from its schema")
	}
	// Platform datastreams are always declared.
	if !strings.Contains(out, "'provision.administration.identifier'") {
		t.Error("platform datastreams missing")
	}
	// An indexed datastream is an array of elements, not a single object.
	if !strings.Contains(out, "'provision.device.communicationModules[].identifier'?: Array<OGIndexedDatastream<string>>") {
		t.Error("indexed datastream should be declared as an array")
	}
	// The unit and description become the doc comment.
	if !strings.Contains(out, "Ambient temperature in Celsius") || !strings.Contains(out, "[ºC]") {
		t.Error("datastream doc comment should carry the description and unit")
	}
}

// Without a datamodel the output is still valid and useful: platform globals
// and platform datastreams only.
func TestGenerateWithoutDatamodel(t *testing.T) {
	out, err := Generate(Options{Context: ContextRuleAdvanced})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "only platform datastreams are declared") {
		t.Error("header should say the datamodel is missing")
	}
	if !strings.Contains(out, "declare const entity: OGEntity") {
		t.Error("entity must still be declared")
	}
	if strings.Contains(out, "sensor.temperature") {
		t.Error("no organization datastreams should appear")
	}
}

func TestGenerateUnknownContext(t *testing.T) {
	if _, err := Generate(Options{Context: "widget/formatter"}); err == nil {
		t.Fatal("expected an error for a context with no template")
	}
}

// A rule states each parameter's schema, so parameterObject can be typed as
// precisely as the entity.
func TestParametersFromRule(t *testing.T) {
	raw := `{"name":"r","parameters":[
	  {"name":"tempThreshold","value":28,"schema":"number"},
	  {"name":"label","value":"hot","schema":"string"},
	  {"name":"enabled","value":true},
	  {"name":"opaque"}
	]}`
	params := ParametersFrom(json.RawMessage(raw))

	// `opaque` has neither schema nor value: it becomes `any`, not `unknown`.
	// `unknown` cannot be compared or used in arithmetic without a cast, and
	// production rules do both with parameters.
	want := map[string]string{"enabled": "boolean", "label": "string", "opaque": "any", "tempThreshold": "number"}
	if len(params) != len(want) {
		t.Fatalf("got %d parameters, want %d: %+v", len(params), len(want), params)
	}
	for _, p := range params {
		if want[p.Name] != p.TSType {
			t.Errorf("parameter %s: type = %q, want %q", p.Name, p.TSType, want[p.Name])
		}
	}

	// `enabled` has no schema — its type comes from the default value.
	out, err := Generate(Options{Context: ContextRuleAdvanced, Parameters: params})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "'tempThreshold': number") {
		t.Error("declared parameter type missing from OGParameters")
	}
	if !strings.Contains(out, "default: 28") {
		t.Error("parameter default should appear as its doc comment")
	}
}

// A rule with no parameters must not make reading one an error: it is typed
// permissively so a parameter added later still compiles.
func TestNoParametersIsPermissive(t *testing.T) {
	out, err := Generate(Options{Context: ContextRuleAdvanced})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "declare const parameterObject: Record<string, any>") {
		t.Error("with no declared parameters, parameterObject should be permissive")
	}
}

// The generated jsconfig must keep the two options the diagnostics depend on.
func TestJSConfigEnablesDiagnostics(t *testing.T) {
	var cfg struct {
		CompilerOptions map[string]any `json:"compilerOptions"`
		Include         []string       `json:"include"`
	}
	if err := json.Unmarshal([]byte(JSConfig), &cfg); err != nil {
		t.Fatalf("jsconfig.json is not valid JSON: %v", err)
	}
	if cfg.CompilerOptions["checkJs"] != true {
		t.Error("checkJs must be on, or there are no diagnostics at all")
	}
	if cfg.CompilerOptions["noImplicitAny"] != true {
		t.Error("noImplicitAny must be on, or a mistyped datastream identifier is silently typed as any")
	}
	if len(cfg.Include) == 0 {
		t.Error("include must cover the artifact's .js files and the declarations")
	}
}

// The header records what the file was generated from, so a stale file is
// recognisable.
func TestHeaderRecordsProvenance(t *testing.T) {
	out, err := Generate(Options{
		Context: ContextRuleAdvanced, Datamodels: demoDatamodel(t), OrgName: "sensehat", Version: "v9.9.9",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"DO NOT EDIT", "rule/ADVANCED", "og v9.9.9", "multisensor v1.0", "organization sensehat", "rules-js-reference.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q", want)
		}
	}
}

func TestContextsIsSorted(t *testing.T) {
	got := Contexts()
	if len(got) == 0 {
		t.Fatal("no contexts advertised")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("contexts not sorted: %v", got)
		}
	}
}

// A real tenant has many datamodels — sensehat has 27, holding 664 datastreams
// between them — and a device's entity can carry datastreams from any of them.
// Typing against one would flag correct code as wrong.
func TestGenerateMergesSeveralDatamodels(t *testing.T) {
	dms := []opengate.Datamodel{
		{
			Identifier: "weather", Version: "1.0",
			Categories: []opengate.Category{{Identifier: "env", Name: "Environment", Datastreams: []opengate.Datastream{
				{Identifier: "weather.temperature", Name: "Temp", Schema: json.RawMessage(`{"type":"number"}`)},
			}}},
		},
		{
			Identifier: "bts", Version: "2.0",
			Categories: []opengate.Category{{Identifier: "power", Datastreams: []opengate.Datastream{
				{Identifier: "bts.voltage", Name: "Voltage", Schema: json.RawMessage(`{"type":"number"}`)},
			}}},
		},
	}

	out, err := Generate(Options{Context: ContextRuleAdvanced, Datamodels: dms, OrgName: "sensehat"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"weather.temperature", "bts.voltage"} {
		if !strings.Contains(out, "'"+id+"'") {
			t.Errorf("datastream %q from a merged datamodel is missing", id)
		}
	}
	// The header must say what it merged, so a stale file is recognisable.
	if !strings.Contains(out, "datamodels: 2") || !strings.Contains(out, "bts, weather") {
		t.Errorf("header should list the merged datamodels:\n%s", firstLines(out, 8))
	}
}

// The same identifier in two datamodels with the same type keeps that type; with
// different types it is left untyped, because asserting either would flag
// correct code that used the other.
func TestGenerateMergeTypeConflicts(t *testing.T) {
	mk := func(id, dsID, jsonType string) opengate.Datamodel {
		return opengate.Datamodel{
			Identifier: id,
			Categories: []opengate.Category{{Datastreams: []opengate.Datastream{
				{Identifier: dsID, Schema: json.RawMessage(`{"type":"` + jsonType + `"}`)},
			}}},
		}
	}

	agreeing, err := Generate(Options{Context: ContextRuleAdvanced,
		Datamodels: []opengate.Datamodel{mk("a", "shared.ds", "number"), mk("b", "shared.ds", "number")}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(agreeing, "'shared.ds'?: OGDatastream<number>") {
		t.Error("agreeing types should be kept")
	}
	if !strings.Contains(agreeing, "[in: a, b]") {
		t.Error("the doc comment should name both source datamodels")
	}
	// One entry, not two.
	if n := strings.Count(agreeing, "'shared.ds'?:"); n != 1 {
		t.Errorf("the identifier is declared %d times, want 1", n)
	}

	conflicting, err := Generate(Options{Context: ContextRuleAdvanced,
		Datamodels: []opengate.Datamodel{mk("a", "shared.ds", "number"), mk("b", "shared.ds", "string")}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(conflicting, "'shared.ds'?: OGDatastream<any>") {
		t.Error("conflicting types should fall back to any — unknown would reject correct comparisons")
	}
	if !strings.Contains(conflicting, "conflicting types") {
		t.Error("the doc comment should explain why it is untyped")
	}
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// Verified in production: sensehat has a live rule triggering on
// `temperature.from.pressure`, which appears in NO datamodel. Declaring only
// what the datamodels hold would flag that rule's working code as an error.
func TestExtraDatastreamsCoverWhatNoDatamodelDeclares(t *testing.T) {
	out, err := Generate(Options{
		Context:          ContextRuleAdvanced,
		Datamodels:       demoDatamodel(t),
		ExtraDatastreams: []string{"temperature.from.pressure", "sensor.temperature"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "'temperature.from.pressure'?: OGDatastream<any>") {
		t.Error("an identifier no datamodel declares must still be declared, untyped")
	}
	if !strings.Contains(out, "declared in no datamodel") {
		t.Error("the doc comment should say where it came from")
	}
	// One already in the datamodel must keep its real type, not be downgraded.
	if !strings.Contains(out, "'sensor.temperature'?: OGDatastream<number>") {
		t.Error("an identifier the datamodel declares must keep its type")
	}
	if n := strings.Count(out, "'sensor.temperature'?:"); n != 1 {
		t.Errorf("declared %d times, want 1", n)
	}
}

func TestDatastreamsReferencedBy(t *testing.T) {
	code := `
		var t = entity['sensor.temperature'];
		var g = gateway["gw.status"];
		var again = entity[ 'sensor.temperature' ];
		var spaced = entity [ "spaced.one" ];
		var dynamic = entity[someVariable];
	`
	got := DatastreamsReferencedBy(code)

	want := []string{"gw.status", "sensor.temperature", "spaced.one"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			break
		}
	}
}

func TestDatastreamsTriggering(t *testing.T) {
	// The shape of a real rule's trigger, from sensehat.
	rule := []byte(`{"name":"R","type":{"name":"DATASTREAM","datastreams":[
	  {"name":"temperature.from.pressure","fields":[{"alias":"x","field":"_current.value"}]},
	  {"name":"sensor.humidity"}]}}`)

	got := DatastreamsTriggering(rule)
	if len(got) != 2 || got[0] != "sensor.humidity" || got[1] != "temperature.from.pressure" {
		t.Errorf("got %v, want the two trigger datastreams sorted", got)
	}

	if got := DatastreamsTriggering([]byte(`{"name":"R"}`)); len(got) != 0 {
		t.Errorf("a rule with no trigger datastreams should yield none, got %v", got)
	}
	if got := DatastreamsTriggering([]byte(`not json`)); got != nil {
		t.Errorf("malformed input should yield nothing, got %v", got)
	}
}

// A connector function's context follows its type field.
func TestContextForConnectorFunction(t *testing.T) {
	for cfType, want := range map[string]Context{
		"REQUEST":    ContextCFRequest,
		"RESPONSE":   ContextCFResponse,
		"COLLECTION": ContextCFCollection,
		"request":    ContextCFRequest,
		// An unknown or missing type falls back to COLLECTION, the most common.
		"":      ContextCFCollection,
		"WEIRD": ContextCFCollection,
	} {
		if got := ContextForConnectorFunction(cfType); got != want {
			t.Errorf("type %q: context = %q, want %q", cfType, got, want)
		}
	}
}

// Connector function typings must declare the protocol objects and the plain
// functions that live production code uses.
func TestConnectorFunctionTemplate(t *testing.T) {
	out, err := Generate(Options{Context: ContextCFCollection, Datamodels: demoDatamodel(t)})
	if err != nil {
		t.Fatal(err)
	}

	// Objects and plain functions found in sensehat's live functions. An earlier
	// version omitted the plain ones and reported "Cannot find name" on code
	// that works.
	for _, decl := range []string{
		"declare const collection", "declare const response", "declare const cf",
		"declare function log(", "declare function httpRequest(",
		"declare function responseCF(", "declare function collectionCF(",
		"declare function ogCollection(", "declare function ogResponse(",
		"declare const mqtt", "declare const http", "declare const snmp", "declare const dlms",
	} {
		if !strings.Contains(out, decl) {
			t.Errorf("missing declaration: %s", decl)
		}
	}

	// logger must be variadic: production code calls logger.debug('x: ', value).
	if !strings.Contains(out, "debug(...msg: unknown[])") {
		t.Error("logger methods must be variadic")
	}
	// mqtt.topic is assigned by production code, not just published to.
	if !strings.Contains(out, "topic: string") {
		t.Error("mqtt.topic must be declared and assignable")
	}
}

// The protocol comes from the scheme of the south criteria. Verified against
// sensehat, whose criteria are mqtts:// and https:// URIs.
func TestProtocolsFromCriteria(t *testing.T) {
	cases := map[string][]string{
		`{"southCriterias":["mqtts://endesa"]}`:        {"mqtt"},
		`{"southCriterias":["https://demo"]}`:          {"http"},
		`{"southCriterias":["mqtts://a","https://b"]}`: {"http", "mqtt"},
		`{"southCriterias":["mqtts://a","mqtt://b"]}`:  {"mqtt"},
		`{"southCriterias":["dlms://obis/1.8.0"]}`:     {"dlms"},
		`{"southCriterias":["wss://x"]}`:               {"websocket"},
		// A REQUEST function has no south criteria: its protocol is unknowable
		// from the payload, which is why every protocol object is declared.
		`{"type":"REQUEST","northCriterias":[{"path":"x"}]}`: nil,
		`{"southCriterias":["not-a-uri"]}`:                   nil,
	}
	for payload, want := range cases {
		got := ProtocolsFromCriteria([]byte(payload))
		if len(got) != len(want) {
			t.Errorf("%s: got %v, want %v", payload, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: got %v, want %v", payload, got, want)
				break
			}
		}
	}
}

// The generated jsconfig has to adapt to the code it will check. An artifact's
// script is wrapped in a function by the platform, so a top-level return is
// correct — and TypeScript reports it. Six of sensehat's seven live connector
// functions contain one, or an untyped helper parameter.
func TestJSConfigAdaptsToTheCode(t *testing.T) {
	checked := func(cfg string) bool {
		var parsed struct {
			CompilerOptions map[string]any `json:"compilerOptions"`
		}
		if err := json.Unmarshal([]byte(cfg), &parsed); err != nil {
			t.Fatalf("generated jsconfig is not valid JSON: %v", err)
		}
		return parsed.CompilerOptions["checkJs"] == true
	}

	strictCases := []string{
		"var t = entity['sensor.temperature'];\nlogger.info('x');\n",
		"if (entity['a']) { collection.addDatapoint('b', 1); }\n",
	}
	for _, code := range strictCases {
		cfg := JSConfigFor(code)
		if !checked(cfg) {
			t.Errorf("checkable code should be checked:\n%s", code)
		}
		if !strings.Contains(cfg, `"noImplicitAny": true`) {
			t.Error("noImplicitAny must be on when checking, or a mistyped datastream is not caught")
		}
	}

	// These would report errors on correct code.
	relaxedCases := map[string]string{
		"top-level return":         "var d = {};\nreturn d;\n",
		"untyped helper parameter": "function doRequest(method, uri) { return method + uri; }\n",
	}
	for label, code := range relaxedCases {
		if checked(JSConfigFor(code)) {
			t.Errorf("%s: must not be checked — it would flag working code", label)
		}
	}
}
