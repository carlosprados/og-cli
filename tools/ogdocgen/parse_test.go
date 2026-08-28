package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The page below is the shape the OpenGate documentation actually uses, reduced
// to the parts the generator reads: a front matter `type`, signature headings,
// the deprecation blockquote, a `**Returns**` line and a parameter table.
const samplePage = `+++
title = 'Sample'
type = 'cf-js-api'
+++

## JS API

#### collectCF(collectionData, collectionFunctionCriteria)

> **Deprecated:** Use [` + "`cf.collection`" + `](../concatenated/#cfcollection) instead. Note the
> arguments are in the opposite order: ` + "`cf.collection`" + ` takes the criteria first.

Used in Request or Response connector functions to define a concatenated Collection action.

**Kind**: global function

| Param                      | Type    | Description            |
| -------------------------- |---------| ---------------------- |
| collectionData             | \*      | data to be used.       |
| collectionFunctionCriteria | String  | the criteria.          |

#### getDeviceId(operationObj)

Reads the device identifier.

**Kind**: global function
**Returns**: String - Returns deviceId field. If operationObj is not correct, null.

| Param        | Type    |
| ------------ |---------|
| operationObj | Object  |

#### logger.trace(...msg)

Writes a TRACE trace.

**Returns**: <code>Void</code>

#### mqtt.publish(payload)

Publishes on the configured topic.

**Returns**: Boolean

### The mqtt Object Properties

| Property | Type   | Description        |
| -------- |--------| ------------------ |
| topic    | String | Topic to publish.  |
`

func parseSample(t *testing.T) (*APIDoc, []Property) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.md")
	if err := os.WriteFile(path, []byte(samplePage), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, props, err := ParseFile(path, "content/sample.md")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return doc, props
}

func find(t *testing.T, doc *APIDoc, object, name string) Decl {
	t.Helper()
	for _, d := range doc.Decls {
		if d.Object == object && d.Name == name {
			return d
		}
	}
	t.Fatalf("declaration %s.%s not parsed; got %+v", object, name, doc.Decls)
	return Decl{}
}

func TestParseFrontMatterType(t *testing.T) {
	doc, _ := parseSample(t)
	if doc.Type != "cf-js-api" {
		t.Errorf("Type = %q, want cf-js-api", doc.Type)
	}
}

// A deprecation notice must reach the declaration, keep the replacement name
// that lives inside the markdown link, and survive being wrapped over two
// blockquote lines. It must NOT be mistaken for the summary.
func TestParseDeprecationNotice(t *testing.T) {
	doc, _ := parseSample(t)
	d := find(t, doc, "", "collectCF")

	want := "Use `cf.collection` instead. Note the arguments are in the opposite order: `cf.collection` takes the criteria first."
	if d.Deprecated != want {
		t.Errorf("Deprecated =\n  %q\nwant\n  %q", d.Deprecated, want)
	}
	if got := "Used in Request or Response connector functions to define a concatenated Collection action."; d.Summary != got {
		t.Errorf("Summary = %q, want the prose line, not the notice", d.Summary)
	}
	if note := deprecatedNote(d); note != "@deprecated "+want {
		t.Errorf("deprecatedNote = %q", note)
	}
}

// The `**Returns**` line is how 224 of the entries we read state their type;
// reading only the heading form found 3 of them.
func TestParseReturnsLine(t *testing.T) {
	doc, _ := parseSample(t)

	for _, tc := range []struct {
		object, name string
		want         string
	}{
		{"", "getDeviceId", "string"},  // "String - prose" — the type is before the dash
		{"logger", "trace", "void"},    // <code>Void</code>
		{"mqtt", "publish", "boolean"}, // bare Boolean
		{"", "collectCF", "any"},       // no Returns line at all
	} {
		d := find(t, doc, tc.object, tc.name)
		if got := returnType(d); got != tc.want {
			t.Errorf("%s.%s returns %q, want %q (raw %q)", tc.object, tc.name, got, tc.want, d.Returns)
		}
	}
}

func TestParseParamsAndProperties(t *testing.T) {
	doc, props := parseSample(t)

	d := find(t, doc, "", "collectCF")
	if len(d.Params) != 2 {
		t.Fatalf("collectCF has %d params, want 2", len(d.Params))
	}
	if got := emitParams(d.Params); got != "collectionData: any, collectionFunctionCriteria: string" {
		t.Errorf("emitParams = %q", got)
	}

	// A rest parameter keeps its dots: dropping them made logger.trace('a: ', b)
	// an arity error on working production code.
	trace := find(t, doc, "logger", "trace")
	if len(trace.Params) != 1 || !trace.Params[0].Variadic {
		t.Fatalf("logger.trace params = %+v, want one variadic", trace.Params)
	}

	if len(props) != 1 || props[0].Object != "mqtt" || props[0].Name != "topic" {
		t.Fatalf("properties = %+v, want mqtt.topic", props)
	}
}

func TestTSType(t *testing.T) {
	for in, want := range map[string]string{
		"String":                  "string",
		"`Object`":                "any",
		"<code>Uint8Array</code>": "Uint8Array",
		"Array.<String>":          "string[]",
		"Array<Object>":           "any[]",
		"Void":                    "void",
		"⇒ String":                "string", // heading form: foo(x) ⇒ String
		"returns msisdn parameter without format": "any", // prose, not a type
	} {
		if got := tsType(in); got != want {
			t.Errorf("tsType(%q) = %q, want %q", in, got, want)
		}
	}
}

// The documentation renamed the rules front matter with no alias, and the
// generator kept exiting zero with a stale rule-advanced file on disk. A family
// that yields no page has to be an error.
func TestCheckCoverage(t *testing.T) {
	full := map[string]int{}
	for docType := range contextForDocType {
		full[docType] = 1
	}
	if missing := checkCoverage(full); len(missing) != 0 {
		t.Errorf("checkCoverage(complete) = %v, want none", missing)
	}

	delete(full, "rules-js-api")
	missing := checkCoverage(full)
	if len(missing) != 1 || missing[0] != "rules-js-api" {
		t.Errorf("checkCoverage(no rules pages) = %v, want [rules-js-api]", missing)
	}
}
