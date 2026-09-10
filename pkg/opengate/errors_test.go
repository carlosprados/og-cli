package opengate

import (
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"
)

// An OpenGate error body's context names the field the platform objected to
// and, on a rejected query parameter, the value it refused. Reporting only the
// name is what turned a rejected expand value into a hunt for the cause.

// expandRejected is the platform's answer to an expand value it does not
// support, captured live from api.opengate.es with ?expand=bogus.
const expandRejectedBody = `{"errors":[{"code":"0xA0","message":"Invalid query parameters.","context":[{"value":"sorts","name":"expand"}]}]}`

// The value is the actionable half of the rejection: "(fields: expand)" sends
// the reader hunting, "(fields: expand=sorts)" names the culprit.
func TestCheckResponseReportsRejectedValue(t *testing.T) {
	err := CheckResponse([]byte(expandRejectedBody), http.StatusBadRequest)
	if err == nil {
		t.Fatal("CheckResponse: want an error, got none")
	}
	if want := "(fields: expand=sorts)"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if !apiErr.HasField("expand") {
		t.Error("HasField(expand) = false, want true")
	}
	if apiErr.HasField("sorts") {
		t.Error("HasField(sorts) = true, want false: sorts is the value, not the field")
	}
	// Fields stays populated: it is part of the published surface.
	if len(apiErr.Fields) != 1 || apiErr.Fields[0] != "expand" {
		t.Errorf("Fields = %v, want [expand]", apiErr.Fields)
	}
}

// A context entry with no value keeps the bare name, which is what the
// datamodel PUT rejection looks like.
func TestCheckResponseKeepsBareFieldName(t *testing.T) {
	body := `{"errors":[{"code":"0x1","message":"Forbidden field.","context":[{"name":"datamodel.allowedResourceTypes"}]}]}`
	err := CheckResponse([]byte(body), http.StatusBadRequest)
	if want := "(fields: datamodel.allowedResourceTypes)"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

// A numeric value must not print as 1e+06.
func TestCheckResponseFormatsNumericValue(t *testing.T) {
	body := `{"errors":[{"code":"0x1","message":"Invalid query parameters.","context":[{"name":"limit","value":1000000}]}]}`
	err := CheckResponse([]byte(body), http.StatusBadRequest)
	if want := "(fields: limit=1000000)"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

// A long value is cut for display without splitting a multi-byte character.
func TestErrorContextTruncatesByRune(t *testing.T) {
	long := strings.Repeat("á", maxContextValueLen+10)
	got := ErrorContext{Name: "f", Value: long}.String()
	want := "f=" + strings.Repeat("á", maxContextValueLen) + "..."
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Error("String() produced invalid UTF-8")
	}
}
