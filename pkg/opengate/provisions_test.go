package opengate

import (
	"encoding/json"
	"testing"
)

func TestParseProvisionProcessorSummary(t *testing.T) {
	raw := json.RawMessage(`{
	  "provisionProcessorId": "pp-1",
	  "name": "AddDevices",
	  "configurationParams": {"spreadsheet": {"sheetName": "IEC102", "headerRow": 1, "resultColumnName": "Result"}}
	}`)
	s := ParseProvisionProcessorSummary(raw)
	if s.ProvisionProcessorID != "pp-1" || s.Name != "AddDevices" {
		t.Errorf("id/name wrong: %+v", s)
	}
	if s.SheetName != "IEC102" || s.HeaderRow != "1" || s.ResultColumnName != "Result" {
		t.Errorf("spreadsheet fields wrong: %+v", s)
	}
}

func TestParseProvisionProcessorSummaryStringHeaderRow(t *testing.T) {
	// the API has been seen returning headerRow as a quoted string too
	raw := json.RawMessage(`{"name":"x","configurationParams":{"spreadsheet":{"headerRow":"2"}}}`)
	if got := ParseProvisionProcessorSummary(raw).HeaderRow; got != "2" {
		t.Errorf("headerRow = %q, want 2", got)
	}
}

func TestListProvisionProcessorsResponseItems(t *testing.T) {
	// live responses use "provisionProcessors"
	a := ListProvisionProcessorsResponse{ProvisionProcessors: []json.RawMessage{[]byte(`{}`)}}
	if len(a.Items()) != 1 {
		t.Error("should return provisionProcessors array")
	}
	// schema names it "processors" — accept that too
	b := ListProvisionProcessorsResponse{Processors: []json.RawMessage{[]byte(`{}`), []byte(`{}`)}}
	if len(b.Items()) != 2 {
		t.Error("should fall back to processors array")
	}
	// prefer provisionProcessors when both present
	c := ListProvisionProcessorsResponse{
		ProvisionProcessors: []json.RawMessage{[]byte(`{}`)},
		Processors:          []json.RawMessage{[]byte(`{}`), []byte(`{}`)},
	}
	if len(c.Items()) != 1 {
		t.Error("should prefer provisionProcessors over processors")
	}
}
