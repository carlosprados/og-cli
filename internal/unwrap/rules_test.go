package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const advancedRule = `{
  "name": "env-anomaly",
  "identifier": "rule-123",
  "mode": "ADVANCED",
  "active": true,
  "type": {"name": "DATASTREAM", "datastreams": ["sensor.temperature"]},
  "javascript": "function checkAnomaly(context) {\n  var temp = context.datastream.value;\n  return temp > 28;\n}"
}`

const easyRule = `{
  "name": "battery-low",
  "mode": "EASY",
  "active": true,
  "type": {"name": "DATASTREAM", "datastreams": ["power.battery"]},
  "condition": {"filter": {"lt": {"power.battery._current.value": 20}}},
  "actions": {"open": [{"name": "batteryLow", "enabled": true, "severity": "CRITICAL"}]}
}`

func TestUnwrapRuleAdvanced(t *testing.T) {
	dir := t.TempDir()
	ruleDir, err := UnwrapRule(json.RawMessage(advancedRule), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(ruleDir) != "env-anomaly" {
		t.Errorf("slug = %s", ruleDir)
	}

	// rule.json must NOT contain the javascript field
	meta, err := os.ReadFile(filepath.Join(ruleDir, "rule.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(meta, &m)
	if _, has := m["javascript"]; has {
		t.Error("javascript field not stripped from rule.json")
	}

	// JS extracted to javascript.js
	js, err := os.ReadFile(filepath.Join(ruleDir, "javascript.js"))
	if err != nil {
		t.Fatalf("javascript.js missing: %v", err)
	}
	if string(js) == "" || !json.Valid(meta) {
		t.Error("unexpected extraction output")
	}
}

func TestUnwrapRuleEasyNoJS(t *testing.T) {
	dir := t.TempDir()
	ruleDir, err := UnwrapRule(json.RawMessage(easyRule), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, _ := os.ReadDir(ruleDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".js" {
			t.Errorf("EASY rule produced unexpected JS file: %s", e.Name())
		}
	}
}

func TestWrapRuleRoundTrip(t *testing.T) {
	for _, src := range []string{advancedRule, easyRule} {
		dir := t.TempDir()
		ruleDir, err := UnwrapRule(json.RawMessage(src), dir)
		if err != nil {
			t.Fatalf("unwrap: %v", err)
		}

		out, err := WrapRule(ruleDir)
		if err != nil {
			t.Fatalf("wrap: %v", err)
		}

		var want, got map[string]any
		json.Unmarshal([]byte(src), &want)
		json.Unmarshal(out, &got)
		if !reflect.DeepEqual(want, got) {
			t.Errorf("round-trip mismatch:\nwant %v\ngot  %v", want, got)
		}
	}
}

func TestWrapRuleEditedJS(t *testing.T) {
	dir := t.TempDir()
	ruleDir, _ := UnwrapRule(json.RawMessage(advancedRule), dir)

	edited := "function checkAnomaly(context) { return true; }"
	os.WriteFile(filepath.Join(ruleDir, "javascript.js"), []byte(edited), 0o644)

	out, err := WrapRule(ruleDir)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	var got map[string]any
	json.Unmarshal(out, &got)
	if got["javascript"] != edited {
		t.Errorf("edited JS not reinjected: %v", got["javascript"])
	}
}
