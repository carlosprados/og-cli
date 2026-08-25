package typegen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
)

func demoDatamodel(t *testing.T) *opengate.Datamodel {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "demo", "datamodels", "multisensor.json"))
	if err != nil {
		t.Fatalf("reading demo datamodel: %v", err)
	}
	var dm opengate.Datamodel
	if err := json.Unmarshal(raw, &dm); err != nil {
		t.Fatalf("parsing demo datamodel: %v", err)
	}
	return &dm
}

// The organization's own datastream identifiers must reach the declarations —
// this is the half no generic editor or LLM can produce.
func TestGenerateDeclaresDatamodelDatastreams(t *testing.T) {
	out, err := Generate(Options{Context: ContextRuleAdvanced, Datamodel: demoDatamodel(t), OrgName: "sensehat"})
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

	want := map[string]string{"enabled": "boolean", "label": "string", "opaque": "unknown", "tempThreshold": "number"}
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
		Context: ContextRuleAdvanced, Datamodel: demoDatamodel(t), OrgName: "sensehat", Version: "v9.9.9",
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
