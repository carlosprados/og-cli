package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// Only the inline tags the pages actually use. A general `<[a-z]…>` pattern
	// eats the type argument of Array.<String> and leaves "Array".
	htmlTag      = regexp.MustCompile(`(?i)</?(?:code|tt|b|i|em|strong|p|br|span)\b[^>]*>`)
	jsdocGeneric = regexp.MustCompile(`^([a-z]+)\.?<([^>]*)>$`)
)

// tsType maps a type as written in the documentation to a TypeScript one.
//
// Anything unrecognised becomes `any`, not `unknown`. `unknown` is the safer
// choice in the abstract, but it cannot be compared or used in arithmetic
// without a cast, and production artifacts do both — a rule comparing
// entity['x']._value._current.value against a threshold is correct JavaScript
// that `unknown` rejects. These declarations exist to catch a mistyped
// identifier, not to enforce value types the platform does not state.
// paramType renders a parameter's type, preferring an override.
func paramType(p Param) string {
	if p.NamedType != "" {
		return p.NamedType
	}
	return tsType(p.Type)
}

func tsType(docType string) string {
	// The pages mix notations for the same thing: `Object`, <code>Object</code>,
	// Array.<String>. Normalise before matching, or two thirds of the documented
	// return types fall through to `any` on punctuation alone.
	t := htmlTag.ReplaceAllString(docType, "")
	t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "\u21d2")) // heading form: `foo(x) ⇒ String`
	t = strings.ToLower(strings.Trim(t, "`*_\\. "))
	t = strings.TrimSuffix(t, "()")

	// JSDoc generics: Array.<String>, Array<String>.
	if m := jsdocGeneric.FindStringSubmatch(t); m != nil {
		if m[1] == "array" {
			return tsType(m[2]) + "[]"
		}
		return "any"
	}

	switch t {
	case "string", "str", "text":
		return "string"
	case "number", "int", "integer", "long", "float", "double", "decimal":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "date":
		return "Date"
	case "uint8array":
		return "Uint8Array"
	case "array", "list":
		return "any[]"
	case "void", "none", "-":
		return "void"
	case "", "*", "any", "object", "json", "json object", "map", "mixed", "entity":
		return "any"
	}

	// "array of string", "string[]", "list of actions"
	if strings.HasSuffix(t, "[]") {
		return tsType(strings.TrimSuffix(t, "[]")) + "[]"
	}
	if rest, ok := strings.CutPrefix(t, "array of "); ok {
		return tsType(rest) + "[]"
	}
	if rest, ok := strings.CutPrefix(t, "list of "); ok {
		return tsType(rest) + "[]"
	}
	return "any"
}

// emitParams renders a declaration's parameter list.
//
// Once one parameter is optional every later one must be too, or TypeScript
// rejects the signature. The docs are not always ordered that way, so this
// propagates optionality rightwards rather than emitting something that does
// not compile.
func emitParams(params []Param) string {
	optionalFrom := len(params)
	for i, p := range params {
		if p.Optional {
			optionalFrom = i
			break
		}
	}

	parts := make([]string, 0, len(params))
	for i, p := range params {
		name := sanitiseIdent(p.Name)
		if p.Variadic {
			parts = append(parts, fmt.Sprintf("...%s: %s[]", name, paramType(p)))
			continue
		}
		marker := ""
		if i >= optionalFrom {
			marker = "?"
		}
		parts = append(parts, fmt.Sprintf("%s%s: %s", name, marker, paramType(p)))
	}
	return strings.Join(parts, ", ")
}

// sanitiseIdent makes a documented parameter name usable as an identifier.
func sanitiseIdent(name string) string {
	var b strings.Builder
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r == '$':
			b.WriteRune(r)
		case r >= '0' && r <= '9' && i > 0:
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "arg"
	}
	// A TypeScript reserved word cannot be a parameter name.
	switch out {
	case "function", "default", "class", "new", "delete", "in", "of", "var", "let", "const", "return", "this":
		return out + "_"
	}
	return out
}

// docComment renders a JSDoc comment, or nothing when there is nothing to say.
func docComment(indent, summary string, extra ...string) string {
	lines := []string{}
	if summary != "" {
		lines = append(lines, summary)
	}
	for _, e := range extra {
		if e != "" {
			lines = append(lines, e)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) == 1 && len(lines[0]) < 90 {
		return fmt.Sprintf("%s/** %s */\n", indent, lines[0])
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s/**\n", indent)
	for _, l := range lines {
		fmt.Fprintf(&b, "%s *  %s\n", indent, l)
	}
	fmt.Fprintf(&b, "%s */\n", indent)
	return b.String()
}

// Bundle is everything to emit for one execution context.
type Bundle struct {
	Context string
	Sources []string
	Plain   []Decl
	Objects map[string][]Decl
	Props   map[string][]Property
}

// NewBundle groups declarations by their receiver.
func NewBundle(context string) *Bundle {
	return &Bundle{
		Context: context,
		Objects: map[string][]Decl{},
		Props:   map[string][]Property{},
	}
}

// Add files a declaration under its object, or as a plain function.
func (b *Bundle) Add(d Decl) {
	if d.Object == "" {
		b.Plain = append(b.Plain, d)
		return
	}
	b.Objects[d.Object] = append(b.Objects[d.Object], d)
}

// AddProperty files an assignable property.
func (b *Bundle) AddProperty(p Property) {
	b.Props[p.Object] = append(b.Props[p.Object], p)
}

// AddSource records a page this bundle was built from.
func (b *Bundle) AddSource(path string) {
	for _, s := range b.Sources {
		if s == path {
			return
		}
	}
	b.Sources = append(b.Sources, path)
}

// Emit renders the bundle as a TypeScript declaration file.
func (b *Bundle) Emit() string {
	var out strings.Builder

	// ── plain functions ──
	if len(b.Plain) > 0 {
		out.WriteString("// ── Plain functions ─────────────────────────────────────────────────────────\n\n")
		sort.SliceStable(b.Plain, func(i, j int) bool { return b.Plain[i].Name < b.Plain[j].Name })
		seen := map[string]bool{}
		for _, d := range b.Plain {
			if seen[d.Name] {
				continue
			}
			seen[d.Name] = true
			out.WriteString(docComment("", d.Summary, deprecatedNote(d), internalNote(d), sourceNote(d)))
			fmt.Fprintf(&out, "declare function %s(%s): %s;\n\n",
				sanitiseIdent(d.Name), emitParams(d.Params), returnType(d))
		}
	}

	// ── objects ──
	nameSet := map[string]bool{}
	for name := range b.Objects {
		nameSet[name] = true
	}
	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	// Objects form a tree: the documentation describes http only as
	// http.client.get() and http.server.response.send(), so `http` and
	// `http.client` exist purely as containers. An earlier version emitted one
	// level of nesting and dropped both, losing the whole HTTP and CoAP APIs.
	roots := map[string]bool{}
	for _, n := range names {
		roots[strings.SplitN(n, ".", 2)[0]] = true
	}
	sortedRoots := make([]string, 0, len(roots))
	for r := range roots {
		sortedRoots = append(sortedRoots, r)
	}
	sort.Strings(sortedRoots)

	if len(sortedRoots) > 0 {
		out.WriteString("// ── Objects ─────────────────────────────────────────────────────────────────\n\n")
	}
	for _, root := range sortedRoots {
		// `entity` is two things at once: a map keyed by datastream identifier,
		// and an object with accessor methods. The map half is generated per
		// organization from its datamodel, so the methods are emitted as an
		// interface for that declaration to intersect. Declaring a second
		// `entity` here shadowed the map and lost the identifiers entirely.
		if iface, ok := interfaceForObject[root]; ok {
			fmt.Fprintf(&out, "/** Accessor methods carried by `%s`, intersected into its declaration. */\ninterface %s ", root, iface)
			out.WriteString(b.emitObjectBody(root, "", names))
			out.WriteString("\n\n")
			continue
		}
		fmt.Fprintf(&out, "declare const %s: ", sanitiseIdent(root))
		out.WriteString(b.emitObjectBody(root, "", names))
		out.WriteString(";\n\n")
	}
	return out.String()
}

// emitObjectBody renders the body of one object, recursing into its children.
func (b *Bundle) emitObjectBody(path, indent string, allNames []string) string {
	var out strings.Builder
	out.WriteString("{\n")
	inner := indent + "  "

	// Assignable properties first — these are what a signature-only reading of
	// the documentation misses entirely, and what production code assigns.
	props := b.propsFor(path)
	sort.SliceStable(props, func(i, j int) bool { return props[i].Name < props[j].Name })
	seen := map[string]bool{}
	for _, p := range props {
		if p.Name == "" || seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		extra := ""
		if p.Default != "" {
			extra = "Default: " + p.Default
		}
		out.WriteString(docComment(inner, p.Comment, extra))
		fmt.Fprintf(&out, "%s%s: %s;\n", inner, sanitiseIdent(p.Name), tsType(p.Type))
	}

	// Methods.
	methods := append([]Decl{}, b.Objects[path]...)
	sort.SliceStable(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	for _, d := range methods {
		if seen[d.Name] {
			continue
		}
		seen[d.Name] = true
		out.WriteString(docComment(inner, d.Summary, deprecatedNote(d), internalNote(d)))
		fmt.Fprintf(&out, "%s%s(%s): %s;\n", inner, sanitiseIdent(d.Name), emitParams(d.Params), returnType(d))
	}

	// Children, one level down.
	prefix := path + "."
	children := map[string]bool{}
	for _, n := range allNames {
		if !strings.HasPrefix(n, prefix) {
			continue
		}
		children[strings.SplitN(strings.TrimPrefix(n, prefix), ".", 2)[0]] = true
	}
	sortedChildren := make([]string, 0, len(children))
	for c := range children {
		sortedChildren = append(sortedChildren, c)
	}
	sort.Strings(sortedChildren)

	for _, child := range sortedChildren {
		if seen[child] {
			continue
		}
		fmt.Fprintf(&out, "%s%s: ", inner, sanitiseIdent(child))
		out.WriteString(b.emitObjectBody(prefix+child, inner, allNames))
		out.WriteString(";\n")
	}

	fmt.Fprintf(&out, "%s}", indent)
	return out.String()
}

// propsFor returns the documented properties of an object path, tolerating the
// case difference between a heading ("Websocket Object Properties") and the
// signatures ("websocket.sendMsg").
func (b *Bundle) propsFor(path string) []Property {
	if p, ok := b.Props[path]; ok {
		return p
	}
	if p, ok := b.Props[strings.ToLower(path)]; ok {
		return p
	}
	// A nested heading is written as "server.response Object Properties", so the
	// leaf alone may be the key.
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		if p, ok := b.Props[strings.ToLower(path[idx+1:])]; ok {
			return p
		}
	}
	return nil
}

// returnType renders a declaration's return type. Undocumented means `any`
// rather than void: a documented function whose result is used would otherwise
// be an error.
func returnType(d Decl) string {
	if d.Returns == "" {
		return "any"
	}
	return tsType(d.Returns)
}

// deprecatedNote renders the documentation's own deprecation notice as a JSDoc
// `@deprecated` tag, which an editor shows struck through with the replacement
// on hover.
//
// The text is the documentation's, verbatim, and that is the point. Two of
// these replacements — `collectCF` → `cf.collection` and `responseCF` →
// `cf.response` — take their arguments in the opposite order, and the notice
// says so; a rename driven by a table of names alone produces code that
// compiles, runs, and sends the payload where the criteria belongs. The other
// group (`ogCollection*`, `ogResponse`, `ogStep*`, `httpRequest`,
// `webSocketMsg`, `publishOnTopic`) is deliberately not a function-for-function
// mapping — the objects that replace them absorb the calls — so nothing here
// invents an equivalence the documentation does not state.
func deprecatedNote(d Decl) string {
	if d.Deprecated == "" {
		return ""
	}
	return "@deprecated " + d.Deprecated
}

func internalNote(d Decl) string {
	if !d.Internal {
		return ""
	}
	return "@internal — the documentation marks this API as internal; it is not part of the public surface."
}

func sourceNote(d Decl) string { return "" }

// interfaceForObject names the objects emitted as an interface rather than a
// const, because their declaration is completed elsewhere.
var interfaceForObject = map[string]string{
	"entity": "OGEntityMethods",
}
