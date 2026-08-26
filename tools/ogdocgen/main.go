// Command ogdocgen generates TypeScript declaration files from the OpenGate
// documentation, so an editor can offer completion and diagnostics for the
// JavaScript embedded in platform artifacts.
//
// It is a development tool, not part of the og binary: run it when the
// documentation changes, commit the result. That keeps the generated
// declarations reviewable — the diff of a regeneration shows what changed in
// the platform — and means og needs no access to the documentation repository.
//
//	go run ./tools/ogdocgen -docs <path-to-odm-documentation-hugo> -out internal/typegen/templates
//
// The documentation is authoritative about what exists, but not always about
// the details: it contains signatures the platform does not match, and in one
// place its prose contradicts its own example. Corrections live in
// overrides.go, each with the evidence for it.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// contextForDocType maps a documentation `type` to the artifact execution
// context whose declarations it belongs in.
//
// The names are the documentation's own, and one is misleading: the nine
// `alarms-js-api` pages are the ADVANCED rules API, not an alarms API.
var contextForDocType = map[string]string{
	"cf-js-api":              "connector-function",
	"cf-internal-js-api":     "connector-function",
	"alarms-js-api":          "rule-advanced",
	"alarms-js-internal-api": "rule-advanced",
	"pf-js-api":              "provision-function",
	"tsf-js-api":             "timeseries-function",
}

// internalDocTypes are the pages the documentation itself marks as internal.
var internalDocTypes = map[string]bool{
	"cf-internal-js-api":     true,
	"alarms-js-internal-api": true,
}

// outputFile is the declaration file each context is written to.
var outputFile = map[string]string{
	"connector-function":  "connector-function.generated.d.ts",
	"rule-advanced":       "rule-advanced.generated.d.ts",
	"provision-function":  "provision-function.generated.d.ts",
	"timeseries-function": "timeseries-function.generated.d.ts",
}

func main() {
	docs := flag.String("docs", "", "path to the odm-documentation-hugo checkout")
	out := flag.String("out", "internal/typegen/templates", "directory to write the declaration files into")
	report := flag.Bool("report", false, "print what was parsed instead of writing files")
	flag.Parse()

	if *docs == "" {
		fmt.Fprintln(os.Stderr, "-docs is required: the path to the odm-documentation-hugo checkout")
		os.Exit(2)
	}

	content := filepath.Join(*docs, "content")
	if _, err := os.Stat(content); err != nil {
		fmt.Fprintf(os.Stderr, "%s does not look like the documentation repository: %v\n", *docs, err)
		os.Exit(2)
	}

	bundles := map[string]*Bundle{}
	var pages int
	var skipped []string

	err := filepath.WalkDir(content, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// The theme ships its own example content; it is not OpenGate's.
			if d.Name() == "themes" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		rel, relErr := filepath.Rel(*docs, path)
		if relErr != nil {
			rel = path
		}

		doc, props, perr := ParseFile(path, rel)
		if perr != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", rel, perr))
			return nil
		}
		context, ok := contextForDocType[doc.Type]
		if !ok {
			return nil
		}
		pages++

		b := bundles[context]
		if b == nil {
			b = NewBundle(context)
			bundles[context] = b
		}
		b.AddSource(rel)

		internal := internalDocTypes[doc.Type]
		for _, decl := range doc.Decls {
			decl.Internal = internal
			b.Add(decl)
		}
		for _, p := range props {
			b.AddProperty(p)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walking the documentation: %v\n", err)
		os.Exit(1)
	}

	for _, s := range skipped {
		fmt.Fprintf(os.Stderr, "  skipped %s\n", s)
	}

	if len(bundles) == 0 {
		fmt.Fprintln(os.Stderr, "no JavaScript API pages found — has the documentation's `type` front matter changed?")
		os.Exit(1)
	}

	contexts := make([]string, 0, len(bundles))
	for c := range bundles {
		contexts = append(contexts, c)
	}
	sort.Strings(contexts)

	fmt.Printf("Parsed %d documentation pages\n", pages)
	for _, context := range contexts {
		b := bundles[context]
		applyOverrides(b)

		methods := 0
		for _, list := range b.Objects {
			methods += len(list)
		}
		props := 0
		for _, list := range b.Props {
			props += len(list)
		}
		fmt.Printf("  %-20s %3d plain, %3d methods in %2d objects, %3d properties, from %d pages\n",
			context, len(b.Plain), methods, len(b.Objects), props, len(b.Sources))

		if *report {
			continue
		}

		name, ok := outputFile[context]
		if !ok {
			continue
		}
		body := header(b) + b.Emit()
		target := filepath.Join(*out, name)
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", target, err)
			os.Exit(1)
		}
		fmt.Printf("    → %s\n", target)
	}
}

// header records what the file was generated from, so a stale one is
// recognisable and a reviewer can find the source page.
func header(b *Bundle) string {
	var s strings.Builder
	s.WriteString("// Generated by tools/ogdocgen from the OpenGate documentation. DO NOT EDIT.\n")
	fmt.Fprintf(&s, "// context: %s\n", b.Context)
	s.WriteString("//\n// Source pages:\n")
	sorted := append([]string{}, b.Sources...)
	sort.Strings(sorted)
	for _, src := range sorted {
		fmt.Fprintf(&s, "//   %s\n", src)
	}
	s.WriteString("//\n// Regenerate: go run ./tools/ogdocgen -docs <odm-documentation-hugo> -out internal/typegen/templates\n\n")
	return s.String()
}
