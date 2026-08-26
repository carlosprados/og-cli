package unwrap

import (
	"fmt"
	"strconv"
	"strings"
)

// ExecCtx names the environment an extracted source file runs in. It is what
// tells validation and type generation which globals are in scope, so it is
// recorded per code path rather than inferred later from the file's location.
type ExecCtx string

const (
	CtxRuleAdvanced ExecCtx = "rule/ADVANCED"
	CtxCFRequest    ExecCtx = "connector-function/REQUEST"
	CtxCFResponse   ExecCtx = "connector-function/RESPONSE"
	CtxCFCollection ExecCtx = "connector-function/COLLECTION"
	CtxProvision    ExecCtx = "provision-function"
	CtxWidget       ExecCtx = "widget"
	CtxUnknown      ExecCtx = "unknown"
)

// CodePath binds one payload keypath to one on-disk filename and one execution
// context.
type CodePath struct {
	Path    keyPath
	File    string
	Context ExecCtx
}

// CodeContract declares where an artifact family keeps its source code.
//
// It exists because the previous extraction rule — "any string that looks like
// JavaScript" — made the on-disk layout a function of the file's *contents*:
// editing a formatter down to an assignment with no `function`/`return` made
// its .js file vanish on the next pull, and the code reappeared inside the
// JSON. diff, watch and typegen all need a path→field mapping that holds still,
// so the schema decides the layout, not the content.
type CodeContract struct {
	// ExactPaths are keypaths known to carry code. They are extracted whenever
	// present, to a stable filename, even when the content does not look like
	// code. Used by the flat families, whose payload shape is fixed.
	ExactPaths []CodePath

	// FieldNames are leaf field names that carry code at any depth. Widget
	// configs need this: their code fields are widget-type specific and nest
	// inside arbitrary arrays and objects.
	FieldNames map[string]ExecCtx

	// HeuristicFallback keeps extracting undeclared strings that look like
	// code, with a warning naming the field. Enabled for widgets only, until
	// the per-widget-type field census is complete: refusing to extract an
	// unlisted field would cost a user their editable .js file, which is worse
	// than an incomplete table. It is never enabled where the declared paths
	// are known to be exhaustive.
	HeuristicFallback bool
}

// Warning reports something the caller should see but that is not fatal.
type Warning struct {
	Path    string // keypath, dotted, for humans
	Message string
}

func (w Warning) String() string { return w.Path + ": " + w.Message }

// WarnFunc receives warnings as they are produced. A nil WarnFunc is silent.
type WarnFunc func(Warning)

func (f WarnFunc) emit(path keyPath, format string, args ...any) {
	if f == nil {
		return
	}
	f(Warning{Path: strings.Join(path, "."), Message: fmt.Sprintf(format, args...)})
}

// RuleContract declares the code path of an automation rule. EASY rules simply
// have no javascript field; the path is extracted when present.
func RuleContract() CodeContract {
	return CodeContract{ExactPaths: []CodePath{
		{Path: keyPath{"javascript"}, File: "javascript.js", Context: CtxRuleAdvanced},
	}}
}

// ConnectorFunctionContract declares the code path of a connector function.
// cfType is the artifact's `type` field — REQUEST, RESPONSE or COLLECTION —
// which decides the execution context and therefore the globals in scope.
func ConnectorFunctionContract(cfType string) CodeContract {
	ctx := CtxUnknown
	switch strings.ToUpper(cfType) {
	case "REQUEST":
		ctx = CtxCFRequest
	case "RESPONSE":
		ctx = CtxCFResponse
	case "COLLECTION":
		ctx = CtxCFCollection
	}
	return CodeContract{ExactPaths: []CodePath{
		{Path: keyPath{"javascript"}, File: "javascript.js", Context: ctx},
	}}
}

// ProvisionFunctionContract declares the code path of a provision processor.
// The script implements normalizeRawObject and actionsPlanning.
func ProvisionFunctionContract() CodeContract {
	return CodeContract{ExactPaths: []CodePath{
		{Path: keyPath{"scriptProcessor", "script"}, File: "scriptProcessor__script.js", Context: CtxProvision},
	}}
}

// widgetCodeFields are the widget config field names known to carry code.
//
// Verified against the demo corpus and the widget-JS reference:
//
//	_widgetConfigCode  customChart / customTable body
//	_formatterCode     per-column formatter in the list widgets
//	formatter          formatter callback
//
// Inherited from the previous allowlist and kept so no existing tree changes
// shape, though none is attested in the demo corpus or the references:
//
//	script, code, fn, expression
//
// Deliberately NOT here: `operation`, which holds an operation *name*
// (REFRESH_INFO), not code — it was extracting one-word .js files; and
// `javascript`, which belongs to the flat families, not to widgets.
var widgetCodeFields = map[string]ExecCtx{
	"_widgetconfigcode": CtxWidget,
	"_formattercode":    CtxWidget,
	"formatter":         CtxWidget,
	"script":            CtxWidget,
	"code":              CtxWidget,
	"fn":                CtxWidget,
	"expression":        CtxWidget,
}

// WidgetContract declares the code fields of a widget config.
func WidgetContract() CodeContract {
	return CodeContract{FieldNames: widgetCodeFields, HeuristicFallback: true}
}

// lookupExact returns the declared code path at p, if any.
func (c CodeContract) lookupExact(p keyPath) (CodePath, bool) {
	for _, cp := range c.ExactPaths {
		if len(cp.Path) != len(p) {
			continue
		}
		match := true
		for i := range cp.Path {
			if cp.Path[i] != p[i] {
				match = false
				break
			}
		}
		if match {
			return cp, true
		}
	}
	return CodePath{}, false
}

// Declares reports whether filename is one this contract would produce, so a
// caller can tell an artifact's own code file from a stray .js sitting in the
// directory.
func (c CodeContract) Declares(filename string) bool {
	for _, cp := range c.ExactPaths {
		if cp.File == filename {
			return true
		}
	}
	if len(c.FieldNames) == 0 {
		return false
	}
	path := parseFilename(filename)
	if path == nil {
		return false
	}
	leaf := strings.ToLower(path[len(path)-1])
	_, ok := c.FieldNames[leaf]
	return ok
}

// Extract walks node and pulls out every string the contract declares as code,
// returning the cleaned document, the files to write, and any warnings.
func (c CodeContract) Extract(node any, warn WarnFunc) (cleaned any, files map[string]string, ctxByFile map[string]ExecCtx) {
	files = make(map[string]string)
	ctxByFile = make(map[string]ExecCtx)
	cleaned = c.walk(node, nil, files, ctxByFile, warn)
	return cleaned, files, ctxByFile
}

func (c CodeContract) walk(node any, path keyPath, files map[string]string, ctxByFile map[string]ExecCtx, warn WarnFunc) any {
	switch v := node.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for k, child := range v {
			childPath := append(append(keyPath{}, path...), k)
			if s, ok := child.(string); ok {
				if file, ctx, extract := c.decide(childPath, k, s, warn); extract {
					files[file] = s
					ctxByFile[file] = ctx
					continue
				}
			}
			result[k] = c.walk(child, childPath, files, ctxByFile, warn)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, child := range v {
			childPath := append(append(keyPath{}, path...), strconv.Itoa(i))
			if s, ok := child.(string); ok {
				if file, ctx, extract := c.decide(childPath, strconv.Itoa(i), s, warn); extract {
					files[file] = s
					ctxByFile[file] = ctx
					continue
				}
			}
			result[i] = c.walk(child, childPath, files, ctxByFile, warn)
		}
		return result
	default:
		return v
	}
}

// decide applies the contract to one string value.
func (c CodeContract) decide(path keyPath, key, value string, warn WarnFunc) (file string, ctx ExecCtx, extract bool) {
	if cp, ok := c.lookupExact(path); ok {
		return cp.File, cp.Context, true
	}
	if ctx, ok := c.FieldNames[strings.ToLower(key)]; ok {
		return path.filename(), ctx, true
	}
	if !looksLikeJS(value) {
		return "", "", false
	}
	if c.HeuristicFallback {
		warn.emit(path, "extracted by content heuristic — %q is not a declared code field, please report it so it can be added", key)
		return path.filename(), CtxUnknown, true
	}
	warn.emit(path, "looks like code but is not a declared code path; left inline")
	return "", "", false
}

// Reinject puts the code files back at their declared keypaths. A .js file the
// contract does not declare is ignored with a warning rather than injected as
// a payload field — which is what makes an auxiliary helper.js harmless.
func (c CodeContract) Reinject(node any, files map[string]string, warn WarnFunc) any {
	for filename, code := range files {
		if !c.Declares(filename) {
			warn.emit(keyPath{filename}, "not a code file this artifact declares; ignored (it will not be deployed)")
			continue
		}
		path := c.pathFor(filename)
		if path == nil {
			warn.emit(keyPath{filename}, "cannot be mapped back to a field; ignored")
			continue
		}
		node = setAt(node, path, code)
	}
	return node
}

// pathFor resolves a filename to the keypath it belongs at.
func (c CodeContract) pathFor(filename string) keyPath {
	for _, cp := range c.ExactPaths {
		if cp.File == filename {
			return cp.Path
		}
	}
	return parseFilename(filename)
}
