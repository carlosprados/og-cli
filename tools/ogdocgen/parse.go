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
	)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	flush := func() {
		if pendingDecl != nil {
			doc.Decls = append(doc.Decls, *pendingDecl)
			pendingDecl = nil
		}
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

		// The first prose line after a signature is its summary.
		if pendingDecl != nil && pendingDecl.Summary == "" && trimmed != "" && !strings.HasPrefix(trimmed, "```") {
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

	name := unescapeMarkdown(strings.Trim(get("property", "parameter", "name", "field"), "`*_ "))
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
			if def != "" {
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
		Name:    unescapeMarkdown(strings.Trim(get("property", "name", "field"), "`*_ ")),
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

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
