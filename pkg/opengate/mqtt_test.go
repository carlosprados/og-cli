package opengate

import (
	"encoding/json"
	"testing"
)

func TestMQTTHostFromProfile(t *testing.T) {
	cases := map[string]string{
		"https://api.opengate.es":  "api.opengate.es",
		"http://api.opengate.es/":  "api.opengate.es",
		"api.opengate.es":          "api.opengate.es",
		"https://host:8443/":       "host:8443",
		"ssl://broker.example.com": "broker.example.com",
	}
	for in, want := range cases {
		if got := MQTTHostFromProfile(in); got != want {
			t.Errorf("MQTTHostFromProfile(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMQTTDefaultTopics(t *testing.T) {
	if got := MQTTDataTopic("dev1"); got != "odm/iot/dev1" {
		t.Errorf("data topic = %q", got)
	}
	if got := MQTTRequestTopic("dev1"); got != "odm/request/dev1" {
		t.Errorf("request topic = %q", got)
	}
	if got := MQTTResponseTopic("dev1"); got != "odm/response/dev1" {
		t.Errorf("response topic = %q", got)
	}
}

func TestBulkIDFromLocation(t *testing.T) {
	cases := map[string]string{
		"http://og/north/v80/provisionProcessors/provision/organizations/o/bulk/abc-123": "abc-123",
		"https://og/.../bulk/xyz/": "xyz",
		"just-an-id":               "just-an-id",
		"":                         "",
	}
	for in, want := range cases {
		if got := bulkIDFromLocation(in); got != want {
			t.Errorf("bulkIDFromLocation(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExcelContentType(t *testing.T) {
	if ct := excelContentType("data.xls"); ct != "application/vnd.ms-excel" {
		t.Errorf(".xls content type = %q", ct)
	}
	xlsx := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if ct := excelContentType("data.xlsx"); ct != xlsx {
		t.Errorf(".xlsx content type = %q", ct)
	}
	if ct := excelContentType("data.unknown"); ct != xlsx {
		t.Errorf("unknown ext should default to xlsx type, got %q", ct)
	}
}

func TestBuildOperationResponse(t *testing.T) {
	out := BuildOperationResponse("dev1", "REBOOT_EQUIPMENT", "op-id-1", "SUCCESSFUL", "rebooting")

	var resp struct {
		Operation struct {
			Response struct {
				Name              string `json:"name"`
				ResultCode        string `json:"resultCode"`
				ResultDescription string `json:"resultDescription"`
				DeviceID          string `json:"deviceId"`
				ID                string `json:"id"`
				Steps             []struct {
					Name        string `json:"name"`
					Result      string `json:"result"`
					Description string `json:"description"`
				} `json:"steps"`
			} `json:"response"`
		} `json:"operation"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	r := resp.Operation.Response
	if r.Name != "REBOOT_EQUIPMENT" || r.ID != "op-id-1" || r.DeviceID != "dev1" {
		t.Errorf("response identity wrong: %+v", r)
	}
	if r.ResultCode != "SUCCESSFUL" || r.ResultDescription != "Success" {
		t.Errorf("SUCCESSFUL should map to resultDescription=Success, got code=%q desc=%q", r.ResultCode, r.ResultDescription)
	}
	if resp.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", resp.Version)
	}
	if len(r.Steps) != 1 || r.Steps[0].Result != "SUCCESSFUL" || r.Steps[0].Description != "rebooting" {
		t.Errorf("steps wrong: %+v", r.Steps)
	}
}

func TestBuildOperationResponseError(t *testing.T) {
	out := BuildOperationResponse("dev1", "REFRESH_INFO", "id2", "ERROR", "boom")
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	resp := m["operation"].(map[string]any)["response"].(map[string]any)
	if resp["resultDescription"] != "ERROR" {
		t.Errorf("non-SUCCESSFUL resultDescription should echo the code, got %v", resp["resultDescription"])
	}
}

func TestParseOperationRequest(t *testing.T) {
	payload := []byte(`{"operation":{"request":{"timestamp":1614253699108,"name":"REBOOT_EQUIPMENT","parameters":{"type":"HARDWARE"},"id":"05d17fb4"}}}`)
	req, err := ParseOperationRequest(payload)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	op := req.Operation.Request
	if op.Name != "REBOOT_EQUIPMENT" || op.ID != "05d17fb4" {
		t.Errorf("parsed op wrong: %+v", op)
	}
	if op.Parameters["type"] != "HARDWARE" {
		t.Errorf("parameters not parsed: %+v", op.Parameters)
	}
}

func TestParseOperationRequestInvalid(t *testing.T) {
	if _, err := ParseOperationRequest([]byte("not json")); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
