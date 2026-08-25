package unwrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const sampleProvisionProcessor = `{
  "provisionProcessorId": "pp-123",
  "name": "createUpdate",
  "configurationParams": {
    "spreadsheet": {
      "sheetName": "PARA",
      "headerRow": 2,
      "resultColumnName": "ODM_Result"
    }
  },
  "scriptProcessor": {
    "script": "function normalizeRawObject(rawObject) {\n  return {};\n}\nfunction actionsPlanning(normalizedObject) {\n  return [];\n}"
  }
}`

func TestUnwrapProvisionProcessor(t *testing.T) {
	dir := t.TempDir()
	ppDir, err := UnwrapProvisionProcessor(json.RawMessage(sampleProvisionProcessor), dir, &Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Base(ppDir) != "createupdate" {
		t.Errorf("slug = %s", ppDir)
	}

	// provisionfunction.json must NOT contain the nested script field
	meta, err := os.ReadFile(filepath.Join(ppDir, "provisionfunction.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(meta, &m)
	cfg, _ := m["scriptProcessor"].(map[string]any)
	if cfg != nil {
		if _, has := cfg["script"]; has {
			t.Error("script field not stripped from provisionfunction.json")
		}
	}

	// JS extracted to scriptProcessor__script.js
	if _, err := os.ReadFile(filepath.Join(ppDir, "scriptProcessor__script.js")); err != nil {
		t.Fatalf("scriptProcessor__script.js missing: %v", err)
	}
}

func TestWrapProvisionProcessorRoundTrip(t *testing.T) {
	dir := t.TempDir()
	ppDir, err := UnwrapProvisionProcessor(json.RawMessage(sampleProvisionProcessor), dir, &Options{})
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}

	out, err := WrapProvisionProcessor(ppDir)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}

	var want, got map[string]any
	json.Unmarshal([]byte(sampleProvisionProcessor), &want)
	json.Unmarshal(out, &got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("round-trip mismatch:\nwant %v\ngot  %v", want, got)
	}
}

func TestWrapProvisionProcessorEditedJS(t *testing.T) {
	dir := t.TempDir()
	ppDir, _ := UnwrapProvisionProcessor(json.RawMessage(sampleProvisionProcessor), dir, &Options{})

	edited := "function normalizeRawObject(o) { return o; }\nfunction actionsPlanning(o) { return []; }"
	os.WriteFile(filepath.Join(ppDir, "scriptProcessor__script.js"), []byte(edited), 0o644)

	out, err := WrapProvisionProcessor(ppDir)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	var got map[string]any
	json.Unmarshal(out, &got)
	sp, _ := got["scriptProcessor"].(map[string]any)
	if sp == nil || sp["script"] != edited {
		t.Errorf("edited JS not reinjected: %v", got["scriptProcessor"])
	}
}
