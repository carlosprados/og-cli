package canon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

func TestKeyOrderIsIrrelevant(t *testing.T) {
	a := json.RawMessage(`{"name":"r","identifier":"r-1","mode":"ADVANCED"}`)
	b := json.RawMessage(`{"mode":"ADVANCED","identifier":"r-1","name":"r"}`)

	same, err := Equal(a, b, Options{Kind: unwrap.KindRule})
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		t.Error("key order must not affect the comparison")
	}
}

// Absent and null carry the same meaning in these payloads, and the platform is
// inconsistent about which it returns. That inconsistency was the largest
// source of diff noise.
func TestNullAndAbsentAreEqual(t *testing.T) {
	cases := [][2]string{
		{`{"id":"1","description":null}`, `{"id":"1"}`},
		{`{"id":"1","users":[]}`, `{"id":"1"}`},
		{`{"id":"1","extra":{}}`, `{"id":"1"}`},
		{`{"id":"1","nested":{"a":null,"b":[]}}`, `{"id":"1"}`},
	}
	for _, c := range cases {
		same, err := Equal(json.RawMessage(c[0]), json.RawMessage(c[1]), Options{Kind: unwrap.KindRule})
		if err != nil {
			t.Fatal(err)
		}
		if !same {
			t.Errorf("%s and %s should compare equal", c[0], c[1])
		}
	}
}

// A real difference must survive canonicalization, or the whole thing is a
// no-op that reports "no changes" forever.
func TestRealDifferencesSurvive(t *testing.T) {
	cases := [][2]string{
		{`{"id":"1","active":true}`, `{"id":"1","active":false}`},
		{`{"id":"1","javascript":"return 1;"}`, `{"id":"1","javascript":"return 2;"}`},
		{`{"id":"1","n":1}`, `{"id":"1","n":2}`},
		// Array order is meaningful: a reordered grid is a different dashboard.
		{`{"id":"1","grid":["a","b"]}`, `{"id":"1","grid":["b","a"]}`},
		// A null inside an array is a position, not an absence.
		{`{"id":"1","grid":[null,"a"]}`, `{"id":"1","grid":["a"]}`},
	}
	for _, c := range cases {
		same, err := Equal(json.RawMessage(c[0]), json.RawMessage(c[1]), Options{Kind: unwrap.KindRule})
		if err != nil {
			t.Fatal(err)
		}
		if same {
			t.Errorf("%s and %s must NOT compare equal", c[0], c[1])
		}
	}
}

// Server-managed and requester-derived fields never participate.
func TestVolatileFieldsIgnored(t *testing.T) {
	a := json.RawMessage(`{"id":"d1","title":"T","__v":26,"lastAccess":"2025-11-21T13:11:21.879Z",
	                       "allowedProfiles":["root"],"editable":true}`)
	b := json.RawMessage(`{"id":"d1","title":"T","__v":41,"lastAccess":"2026-08-26T09:00:00.000Z",
	                       "allowedProfiles":["viewer"],"editable":false}`)

	same, err := Equal(a, b, Options{Kind: unwrap.KindDashboard})
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		ca, _ := Canonicalize(a, Options{Kind: unwrap.KindDashboard})
		cb, _ := Canonicalize(b, Options{Kind: unwrap.KindDashboard})
		t.Errorf("volatile fields must not affect the comparison\n  a: %s\n  b: %s", ca, cb)
	}
}

// A workspace embeds its dashboards, each with its own volatile fields. Only
// stripping the outer ones leaves noise from every nested dashboard.
func TestVolatileFieldsIgnoredWhenNested(t *testing.T) {
	a := json.RawMessage(`{"id":"w1","name":"W","dashboards":[{"id":"d1","dashboard":{"__v":1,"lastAccess":"a","title":"T"}}]}`)
	b := json.RawMessage(`{"id":"w1","name":"W","dashboards":[{"id":"d1","dashboard":{"__v":99,"lastAccess":"z","title":"T"}}]}`)

	same, err := Equal(a, b, Options{Kind: unwrap.KindWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	if !same {
		t.Error("nested volatile fields must be ignored too")
	}
}

// Identity is compared within a tenant and ignored across tenants: staging and
// production hold different ids for the same logical artifact.
func TestIdentityScoping(t *testing.T) {
	staging := json.RawMessage(`{"identifier":"r-staging","name":"R","active":true}`)
	prod := json.RawMessage(`{"identifier":"r-prod","name":"R","active":true}`)
	opts := func(s Scope) Options { return Options{Kind: unwrap.KindRule, Scope: s} }

	if same, _ := Equal(staging, prod, opts(SameTenant)); same {
		t.Error("same-tenant: a different identifier is a real difference")
	}
	if same, _ := Equal(staging, prod, opts(CrossTenant)); !same {
		t.Error("cross-tenant: identifiers differ by construction and must be ignored")
	}

	// A real difference must still be reported cross-tenant.
	changed := json.RawMessage(`{"identifier":"r-prod","name":"R","active":false}`)
	if same, _ := Equal(staging, changed, opts(CrossTenant)); same {
		t.Error("cross-tenant must still report a genuine difference")
	}
}

// Rules, connector functions and provision functions have no verified volatile
// field. An invented one would hide a real change, so the tables are empty on
// purpose — this test states that intent so a future addition is deliberate.
func TestFlatFamiliesHaveNoVolatileFieldsYet(t *testing.T) {
	for _, kind := range []unwrap.Kind{unwrap.KindRule, unwrap.KindConnectorFunction, unwrap.KindProvisionFunction} {
		if got := VolatileFields(kind, SameTenant); len(got) != 0 {
			t.Errorf("%s: volatile fields = %v; add them only when verified against a real payload", kind, got)
		}
		if got := VolatileFields(kind, CrossTenant); len(got) == 0 {
			t.Errorf("%s: identity fields must be ignored cross-tenant", kind)
		}
	}
}

func TestExtraFieldsDropped(t *testing.T) {
	a := json.RawMessage(`{"id":"1","weird":"x"}`)
	b := json.RawMessage(`{"id":"1","weird":"y"}`)
	if same, _ := Equal(a, b, Options{Kind: unwrap.KindRule, Extra: []string{"weird"}}); !same {
		t.Error("Extra should drop the named field")
	}
}

func TestHashIsStableAndDiffers(t *testing.T) {
	payload := json.RawMessage(`{"id":"1","n":1}`)
	o := Options{Kind: unwrap.KindRule}

	h1, err := Hash(payload, o)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := Hash(json.RawMessage(`{"n":1,"id":"1"}`), o)
	if h1 != h2 {
		t.Error("the hash must not depend on key order")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64 hex chars", len(h1))
	}
	h3, _ := Hash(json.RawMessage(`{"id":"1","n":2}`), o)
	if h1 == h3 {
		t.Error("a different artifact must hash differently")
	}
}

func TestInvalidJSONReported(t *testing.T) {
	if _, err := Canonicalize(json.RawMessage(`{"id":`), Options{Kind: unwrap.KindRule}); err == nil {
		t.Error("expected an error for malformed JSON")
	}
}

// Diagnose explains what was ignored, for troubleshooting an unexpected
// "no differences".
func TestDiagnose(t *testing.T) {
	payload := json.RawMessage(`{"id":"d1","__v":3,"lastAccess":"x","title":"T"}`)

	got := Diagnose(payload, Options{Kind: unwrap.KindDashboard})
	for _, want := range []string{"__v", "lastAccess", "same-tenant"} {
		if !strings.Contains(got, want) {
			t.Errorf("diagnosis %q should mention %q", got, want)
		}
	}
	if strings.Contains(got, "allowedProfiles") {
		t.Error("only fields actually present should be listed")
	}

	if Diagnose(json.RawMessage(`{"id":"d1","title":"T"}`), Options{Kind: unwrap.KindDashboard}) != "" {
		t.Error("nothing to ignore should produce no diagnosis")
	}
}
