package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const requestConnectorFunction = `{
  "name": "refreshInfo",
  "identifier": "cf-123",
  "type": "REQUEST",
  "operationalStatus": "PRODUCTION",
  "payloadType": "JSON",
  "operationName": "REFRESH_INFO",
  "northCriterias": [{"path": "provision.device.model._current.value.manufacturer", "value": "Acme"}],
  "javascript": "function buildRequest(operation) {\n  return {cmd: 'refresh'};\n}"
}`

func TestUnwrapConnectorFunction(t *testing.T) {
	dir := t.TempDir()
	cfDir, err := UnwrapConnectorFunction(json.RawMessage(requestConnectorFunction), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(cfDir) != "refreshinfo" {
		t.Errorf("slug = %s", cfDir)
	}

	// connectorfunction.json must NOT contain the javascript field
	meta, err := os.ReadFile(filepath.Join(cfDir, "connectorfunction.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(meta, &m)
	if _, has := m["javascript"]; has {
		t.Error("javascript field not stripped from connectorfunction.json")
	}

	// JS extracted to javascript.js
	if _, err := os.ReadFile(filepath.Join(cfDir, "javascript.js")); err != nil {
		t.Fatalf("javascript.js missing: %v", err)
	}
}

func TestWrapConnectorFunctionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfDir, err := UnwrapConnectorFunction(json.RawMessage(requestConnectorFunction), dir)
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}

	out, err := WrapConnectorFunction(cfDir)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}

	var want, got map[string]any
	json.Unmarshal([]byte(requestConnectorFunction), &want)
	json.Unmarshal(out, &got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("round-trip mismatch:\nwant %v\ngot  %v", want, got)
	}
}

func TestWrapConnectorFunctionEditedJS(t *testing.T) {
	dir := t.TempDir()
	cfDir, _ := UnwrapConnectorFunction(json.RawMessage(requestConnectorFunction), dir)

	edited := "function buildRequest(operation) { return {}; }"
	os.WriteFile(filepath.Join(cfDir, "javascript.js"), []byte(edited), 0o644)

	out, err := WrapConnectorFunction(cfDir)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	var got map[string]any
	json.Unmarshal(out, &got)
	if got["javascript"] != edited {
		t.Errorf("edited JS not reinjected: %v", got["javascript"])
	}
}
