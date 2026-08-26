// Package validate checks an artifact directory before it is deployed.
//
// It exists because deploying a broken artifact is not a typo, it is an
// incident: a connector function with a syntax error stops processing device
// traffic, and a bad provision function corrupts entity data at bulk scale.
//
// What it deliberately is NOT: a JavaScript parser. Embedding one would mean a
// megabyte-scale dependency in a tool that is otherwise standard library and
// Cobra, and a parser still would not catch the errors that actually matter
// (a valid script reading the wrong datastream). The checks here are the ones
// that are cheap, certain, and catch the mistakes people really make while
// editing files by hand — with the JavaScript itself covered by the generated
// typings, which a real type-checker validates in the editor.
package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// Severity distinguishes what must block a deploy from what should be seen.
type Severity string

const (
	// Error means the artifact is broken; deploying it would fail or damage
	// something.
	Error Severity = "error"
	// Warning means it is deployable but probably not what was intended.
	Warning Severity = "warning"
)

// Finding is one problem found in an artifact directory.
type Finding struct {
	Severity Severity `json:"severity"`
	// File is relative to the artifact directory, empty when the finding is
	// about the artifact as a whole.
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

func (f Finding) String() string {
	loc := f.File
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	if loc == "" {
		return fmt.Sprintf("%s: %s", f.Severity, f.Message)
	}
	return fmt.Sprintf("%s: %s: %s", f.Severity, loc, f.Message)
}

// Result is everything found in one artifact directory.
type Result struct {
	Dir      string    `json:"dir"`
	Kind     string    `json:"kind"`
	Findings []Finding `json:"findings,omitempty"`
}

// OK reports whether the artifact is safe to deploy.
func (r Result) OK() bool {
	for _, f := range r.Findings {
		if f.Severity == Error {
			return false
		}
	}
	return true
}

// Errors counts blocking findings.
func (r Result) Errors() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == Error {
			n++
		}
	}
	return n
}

// Artifact validates one unwrapped artifact directory.
func Artifact(d unwrap.Descriptor, dir string) Result {
	r := Result{Dir: dir, Kind: string(d.Kind)}

	metaPath := filepath.Join(dir, d.MetaFile)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		r.Findings = append(r.Findings, Finding{Severity: Error, File: d.MetaFile,
			Message: fmt.Sprintf("cannot be read: %v", err)})
		return r
	}

	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		// Point at the line, so a stray comma is findable.
		r.Findings = append(r.Findings, Finding{Severity: Error, File: d.MetaFile,
			Line: jsonErrorLine(raw, err), Message: fmt.Sprintf("is not valid JSON: %v", err)})
		return r
	}

	r.Findings = append(r.Findings, checkIdentity(d, meta)...)
	r.Findings = append(r.Findings, checkCodeFiles(d, dir, meta)...)
	r.Findings = append(r.Findings, checkFamilyRules(d, meta, dir)...)
	return r
}

// checkIdentity reports a missing identifier, which is what makes an update
// impossible.
func checkIdentity(d unwrap.Descriptor, meta map[string]any) []Finding {
	if id, _ := meta[d.IDKey].(string); id == "" {
		return []Finding{{Severity: Warning, File: d.MetaFile,
			Message: fmt.Sprintf("has no %q — deploy can create it, but not update it", d.IDKey)}}
	}
	if name, _ := meta["name"].(string); name == "" {
		if alt, _ := meta["connectorFunctionName"].(string); alt == "" {
			return []Finding{{Severity: Warning, File: d.MetaFile, Message: "has no name"}}
		}
	}
	return nil
}

// checkCodeFiles validates every declared code file present on disk, and warns
// about .js files the family does not declare — those are silently not deployed.
func checkCodeFiles(d unwrap.Descriptor, dir string, meta map[string]any) []Finding {
	var out []Finding
	contract := d.Contract(meta)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Finding{{Severity: Error, Message: fmt.Sprintf("cannot be listed: %v", err)}}
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		if !contract.Declares(e.Name()) {
			out = append(out, Finding{Severity: Warning, File: e.Name(),
				Message: "is not a code file this artifact declares — it will NOT be deployed"})
			continue
		}
		code, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			out = append(out, Finding{Severity: Error, File: e.Name(),
				Message: fmt.Sprintf("cannot be read: %v", err)})
			continue
		}
		out = append(out, checkBalance(e.Name(), string(code))...)
	}
	return out
}

// checkFamilyRules holds the per-family checks worth making.
func checkFamilyRules(d unwrap.Descriptor, meta map[string]any, dir string) []Finding {
	var out []Finding
	switch d.Kind {
	case unwrap.KindProvisionFunction:
		// A provision processor's script must implement both entry points, or
		// the bulk run fails after the upload — with entities half-processed.
		code, _ := os.ReadFile(filepath.Join(dir, "scriptProcessor__script.js"))
		for _, fn := range []string{"normalizeRawObject", "actionsPlanning"} {
			if len(code) > 0 && !strings.Contains(string(code), fn) {
				out = append(out, Finding{Severity: Error, File: "scriptProcessor__script.js",
					Message: fmt.Sprintf("does not define %s(), which the platform calls on every row", fn)})
			}
		}
	case unwrap.KindConnectorFunction:
		cfType, _ := meta["type"].(string)
		switch strings.ToUpper(cfType) {
		case "REQUEST":
			// REQUEST functions are matched by operationName; without one the
			// function never fires and nothing reports why.
			if name, _ := meta["operationName"].(string); name == "" {
				out = append(out, Finding{Severity: Warning, File: d.MetaFile,
					Message: "a REQUEST connector function with no operationName will never be matched"})
			}
		case "RESPONSE", "COLLECTION":
			if !hasNonEmptyList(meta, "southCriterias") {
				out = append(out, Finding{Severity: Warning, File: d.MetaFile,
					Message: fmt.Sprintf("a %s connector function with no southCriterias will never be matched", strings.ToUpper(cfType))})
			}
		case "":
			out = append(out, Finding{Severity: Error, File: d.MetaFile,
				Message: "has no type — must be REQUEST, RESPONSE or COLLECTION"})
		default:
			out = append(out, Finding{Severity: Error, File: d.MetaFile,
				Message: fmt.Sprintf("has an unknown type %q — must be REQUEST, RESPONSE or COLLECTION", cfType)})
		}
	case unwrap.KindRule:
		mode, _ := meta["mode"].(string)
		if strings.ToUpper(mode) == "ADVANCED" {
			if _, err := os.Stat(filepath.Join(dir, "javascript.js")); err != nil {
				out = append(out, Finding{Severity: Error, File: "javascript.js",
					Message: "is missing, but the rule is in ADVANCED mode"})
			}
		}
	}
	return out
}

func hasNonEmptyList(meta map[string]any, key string) bool {
	list, ok := meta[key].([]any)
	return ok && len(list) > 0
}
