package main

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
)

// APIDoc is one parsed documentation page.
type APIDoc struct {
	Path  string // repo-relative, for provenance in the generated header
	Title string
	Type  string // the front matter `type`: cf-js-api, pf-js-api, …
	Decls []Decl
}

// Decl is one documented function or method.
type Decl struct {
	// Object is empty for a plain function, otherwise the receiver: `mqtt` for
	// mqtt.publish.
	Object string
	Name   string
	Params []Param
	// Returns is the documented return type, when the page states one.
	Returns string
	Summary string
	// Deprecated is the replacement the documentation names, verbatim. Empty
	// when the declaration is current.
	Deprecated string
	// Internal marks a declaration from a page the documentation itself flags
	// as internal API.
	Internal bool
	Source   string // file it came from
}

// Param is one documented parameter.
type Param struct {
	Name     string
	Type     string // as written in the docs
	Default  string
	Comment  string
	Optional bool
	// NamedType overrides the documented type with a named TypeScript one, from
	// the overrides table.
	NamedType string
	// Variadic renders as a rest parameter. The documentation shows a fixed
	// parameter for logger methods while stating in prose that they concatenate
	// however many they are given.
	Variadic bool
}

// Property is one documented, assignable object property. These are what a
// signature-only reading misses: production code assigns mqtt.topic.
type Property struct {
	Object  string
	Name    string
	Type    string
	Default string
	Comment string
	Source  string
}

// front matter delimiters: TOML (+++) or YAML (---).
var (
	frontMatterTOML = "+++"
	frontMatterYAML = "---"

	typeLine  = regexp.MustCompile(`^type\s*=\s*['"]([^'"]+)['"]`)
	titleLine = regexp.MustCompile(`^title\s*=\s*['"]([^'"]+)['"]`)

	// A signature heading, at any heading level. Tolerates a space before the
	// parenthesis (`telnet.send (command, …)`) and a trailing return type
	// (`#### foo(x)  String`), both of which occur.
	sigHeading = regexp.MustCompile(`^#+\s+([a-zA-Z_$][a-zA-Z0-9_$.]*)\s*\(([^)]*)\)\s*(.*)$`)

	// A properties section, whose table lists assignable fields.
	propsHeading = regexp.MustCompile(`(?i)^#+\s+(?:the\s+)?([a-zA-Z_$][a-zA-Z0-9_$]*)\s+object\s+properties`)

	// Any heading, to close the scope of a properties table.
	anyHeading = regexp.MustCompile(`^#+\s+`)

	tableRow = regexp.MustCompile(`^\s*\|(.+)\|\s*$`)

	// A deprecation notice, which the documentation writes as a blockquote
	// immediately under the signature heading:
	//
	//	> **Deprecated:** Use [`cf.collection`](…) instead. Note the arguments
	//	> are in the opposite order: …
	//
	// Reading it is what turns "the documentation says this is deprecated" into
	// a symbol the editor strikes through, with the replacement on hover.
	deprecatedNotice = regexp.MustCompile(`^>\s*\*\*Deprecated:?\*\*\s*(.*)$`)
	// A continuation line of that blockquote.
	blockquoteLine = regexp.MustCompile(`^>\s?(.*)$`)

	// The canonical way a page states a return type, on its own line under the
	// signature:
	//
	//	**Returns**: String - Returns deviceId field. If operationObj is not …
	//
	// 224 entries across the pages we read state one this way. Reading only the
	// heading form caught 3 of them.
	returnsLine = regexp.MustCompile(`^\*\*Returns?\*\*:?\s*(.*)$`)

	// How the pages say a parameter may be left out. There is no optional marker
	// in the signatures and no Default column in most tables: the fact lives in
	// the description, phrased half a dozen ways.
	//
	// Reading it matters because the alternative is asserting an arity the
	// documentation never states. Four live artifacts call openAlarm with 6 of
	// its 7 documented parameters and executeOperation with 10 of 11, and they
	// work.
	optionalHint = regexp.MustCompile(`(?i)\*\*default\*\*|\bdefault:|\boptional\b|\bif (?:it )?is undefined\b|\bif not (?:provided|defined|set)\b|\bif provided\b|\bcan be null\b|\bit is not mandatory\b`)

	// An inline markdown link: keep the text, drop the URL. The replacement
	// names are inside the links, so a naive strip loses exactly the useful
	// half.
	markdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// ParseFile reads one documentation page.
func ParseFile(path, relPath string) (*APIDoc, []Property, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = f.Close() }()

	doc := &APIDoc{Path: relPath}
	var properties []Property

	var (
		inFrontMatter bool
		fmDelim       string
		pendingDecl   *Decl
		propsObject   string
		tableHeader   []string
		inDeprecated  bool
	)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	flush := func() {
		if pendingDecl != nil {
			pendingDecl.Deprecated = plainText(pendingDecl.Deprecated)
			doc.Decls = append(doc.Decls, *pendingDecl)
			pendingDecl = nil
		}
		inDeprecated = false
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// ── front matter ──
		if trimmed == frontMatterTOML || trimmed == frontMatterYAML {
			if !inFrontMatter && len(doc.Decls) == 0 && doc.Type == "" && fmDelim == "" {
				inFrontMatter, fmDelim = true, trimmed
				continue
			}
			if inFrontMatter && trimmed == fmDelim {
				inFrontMatter = false
				continue
			}
		}
		if inFrontMatter {
			if m := typeLine.FindStringSubmatch(trimmed); m != nil {
				doc.Type = m[1]
			}
			if m := titleLine.FindStringSubmatch(trimmed); m != nil {
				doc.Title = m[1]
			}
			continue
		}

		// ── headings ──
		if m := sigHeading.FindStringSubmatch(line); m != nil {
			flush()
			propsObject = ""
			tableHeader = nil

			object, name := splitQualified(unescapeMarkdown(m[1]))
			pendingDecl = &Decl{
				Object:  object,
				Name:    name,
				Params:  parseParamNames(m[2]),
				Returns: strings.TrimSpace(m[3]),
				Source:  relPath,
			}
			continue
		}
		// ── deprecation notice ──
		//
		// A blockquote under the signature, before its prose. Read before the
		// summary rule below, which would otherwise take the notice for the
		// description of the function.
		if pendingDecl != nil {
			if m := deprecatedNotice.FindStringSubmatch(trimmed); m != nil {
				pendingDecl.Deprecated = strings.TrimSpace(m[1])
				inDeprecated = true
				continue
			}
			if inDeprecated {
				if m := blockquoteLine.FindStringSubmatch(trimmed); m != nil {
					pendingDecl.Deprecated = collapseSpaces(pendingDecl.Deprecated + " " + m[1])
					continue
				}
				inDeprecated = false
			}
		}

		// ── documented return type ──
		if pendingDecl != nil && pendingDecl.Returns == "" {
			if m := returnsLine.FindStringSubmatch(trimmed); m != nil {
				pendingDecl.Returns = returnTypeText(m[1])
				continue
			}
		}

		if m := propsHeading.FindStringSubmatch(line); m != nil {
			flush()
			propsObject = strings.ToLower(unescapeMarkdown(m[1]))
			tableHeader = nil
			continue
		}
		if anyHeading.MatchString(line) {
			flush()
			propsObject = ""
			tableHeader = nil
			continue
		}

		// ── tables ──
		if m := tableRow.FindStringSubmatch(line); m != nil {
			cells := splitCells(m[1])
			if isSeparatorRow(cells) {
				continue
			}
			if tableHeader == nil {
				tableHeader = lowerAll(cells)
				continue
			}
			switch {
			case propsObject != "":
				if p := propertyFromRow(tableHeader, cells); p.Name != "" {
					p.Object, p.Source = propsObject, relPath
					properties = append(properties, p)
				}
			case pendingDecl != nil:
				enrichParam(pendingDecl, tableHeader, cells)
			}
			continue
		}

		// The first prose line after a signature is its summary. A blockquote is
		// not prose: it is the deprecation notice, handled above.
		if pendingDecl != nil && pendingDecl.Summary == "" && trimmed != "" &&
			!strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, ">") {
			pendingDecl.Summary = collapseSpaces(trimmed)
		}
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	sort.SliceStable(doc.Decls, func(i, j int) bool {
		if doc.Decls[i].Object != doc.Decls[j].Object {
			return doc.Decls[i].Object < doc.Decls[j].Object
		}
		return doc.Decls[i].Name < doc.Decls[j].Name
	})
	return doc, properties, nil
}

// splitQualified splits `mqtt.publish` into object and name. A dotted path
// deeper than one level keeps everything but the last segment as the object,
// so `utils.odm.sleep` belongs to `utils.odm`.
func splitQualified(qualified string) (object, name string) {
	idx := strings.LastIndex(qualified, ".")
	if idx < 0 {
		return "", qualified
	}
	return qualified[:idx], qualified[idx+1:]
}

// parseParamNames reads the names out of a signature's parameter list. Types
// come from the table underneath, when there is one.
func parseParamNames(list string) []Param {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	var out []Param
	for _, raw := range strings.Split(list, ",") {
		name := strings.TrimSpace(raw)
		// A documented rest parameter, e.g. log(...msg). Dropping the dots
		// without recording them made log('a: ', b) an arity error on working
		// production code.
		variadic := strings.HasPrefix(name, "...")
		name = strings.TrimPrefix(name, "...")
		if name == "" {
			continue
		}
		// Some signatures write an optional parameter as [x].
		optional := strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]")
		name = strings.Trim(name, "[]")
		out = append(out, Param{Name: unescapeMarkdown(name), Optional: optional, Variadic: variadic})
	}
	return out
}

// enrichParam attaches a table row's type, default and description to the
// matching parameter of the pending declaration.
func enrichParam(d *Decl, header, cells []string) {
	get := func(names ...string) string {
		for _, want := range names {
			for i, h := range header {
				if h == want && i < len(cells) {
					return strings.TrimSpace(cells[i])
				}
			}
		}
		return ""
	}

	// "Param" is the most common heading by a distance — 206 tables against 99
	// for "Parameter" — and leaving it out of this list meant every one of them
	// contributed nothing, so their parameters silently stayed `any`.
	name := unescapeMarkdown(strings.Trim(get("param", "parameter", "property", "name", "field"), "`*_ "))
	if name == "" {
		return
	}
	typ := get("type")
	def := get("default", "default value")
	comment := get("description", "comment")

	for i := range d.Params {
		if strings.EqualFold(d.Params[i].Name, name) {
			d.Params[i].Type = typ
			d.Params[i].Default = def
			d.Params[i].Comment = collapseSpaces(comment)
			if def != "" || optionalHint.MatchString(comment) {
				d.Params[i].Optional = true
			}
			return
		}
	}
}

// propertyFromRow reads one row of an object-properties table.
func propertyFromRow(header, cells []string) Property {
	get := func(names ...string) string {
		for _, want := range names {
			for i, h := range header {
				if h == want && i < len(cells) {
					return strings.TrimSpace(cells[i])
				}
			}
		}
		return ""
	}
	return Property{
		Name:    unescapeMarkdown(strings.Trim(get("property", "param", "name", "field"), "`*_ ")),
		Type:    get("type"),
		Default: get("default", "default value"),
		Comment: collapseSpaces(get("description", "comment")),
	}
}

func splitCells(row string) []string {
	parts := strings.Split(row, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func isSeparatorRow(cells []string) bool {
	for _, c := range cells {
		if c == "" {
			continue
		}
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

func lowerAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.ToLower(s)
	}
	return out
}

// unescapeMarkdown removes the backslash escapes the docs use in headings,
// e.g. `V8\_Utils`.
func unescapeMarkdown(s string) string {
	return strings.NewReplacer(`\_`, "_", `\*`, "*", "`", "").Replace(s)
}

// returnTypeText isolates the type from a **Returns** line.
//
// The line is `Type - prose describing it`, but plenty of entries write prose
// alone ("returns msisdn parameter without format"). Those are left to fall
// through to `any` in tsType rather than guessed at: a wrong return type is
// worse than none, because it rejects correct code.
func returnTypeText(raw string) string {
	if idx := strings.Index(raw, " - "); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimSpace(raw)
}

// plainText renders a fragment of documentation markdown as the one line a
// JSDoc tag can carry: links become their text, escapes are undone. Backticks
// stay, because an editor renders them and the replacement names read better
// as code.
func plainText(s string) string {
	if s == "" {
		return ""
	}
	s = markdownLink.ReplaceAllString(s, "$1")
	s = strings.NewReplacer(`\_`, "_", `\*`, "*").Replace(s)
	return collapseSpaces(s)
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
