// Package views maps named field sets (views) to OpenGate select clauses,
// so users and LLMs can ask for intent ("summary", "power") instead of
// memorizing datastream paths.
//
// Views are loaded from three layers, later layers overriding earlier ones
// by view name:
//
//  1. built-in views embedded in the binary (builtin.yaml)
//  2. user views: ~/.og/views/*.yaml
//  3. project views: ./.og/views/*.yaml
//
// Two views with the same name inside the same layer are an error.
package views

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/carlosprados/og-cli/internal/query"
	"go.yaml.in/yaml/v3"
)

//go:embed builtin.yaml
var builtinYAML []byte

// Field is one datastream projected by a view.
type Field struct {
	Name  string
	At    bool   // also project the at timestamp
	Alias string // optional alias override for the value column
}

// Definition is a named view with its projected fields.
type Definition struct {
	Name        string
	Description string
	Fields      []Field
	Source      string // "builtin" or the file it was loaded from
}

// Clauses expands the view into OpenGate select clauses.
func (d Definition) Clauses() []query.SelectClause {
	clauses := make([]query.SelectClause, len(d.Fields))
	for i, f := range d.Fields {
		var subs []string
		if f.At {
			subs = append(subs, "at")
		}
		c := query.NewSelectClause(f.Name, subs...)
		if f.Alias != "" {
			c.Fields[0].Alias = f.Alias
			if f.At {
				c.Fields[1].Alias = f.Alias + "_at"
			}
		}
		clauses[i] = c
	}
	return clauses
}

// Registry holds the merged view dictionary.
type Registry struct {
	views map[string]Definition
}

// Get returns a view by name.
func (r *Registry) Get(name string) (Definition, bool) {
	d, ok := r.views[name]
	return d, ok
}

// All returns every view sorted by name.
func (r *Registry) All() []Definition {
	out := make([]Definition, 0, len(r.views))
	for _, d := range r.views {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResolveSelect expands view names and merges them with explicit select
// clauses. Explicit clauses win on datastream name collisions; duplicates
// across views are removed. Unknown view names are an error.
func (r *Registry) ResolveSelect(viewNames []string, explicit []query.SelectClause) ([]query.SelectClause, error) {
	seen := make(map[string]bool)
	result := make([]query.SelectClause, 0, len(explicit))

	for _, c := range explicit {
		if !seen[c.Name] {
			seen[c.Name] = true
			result = append(result, c)
		}
	}

	for _, name := range viewNames {
		view, ok := r.Get(name)
		if !ok {
			return nil, r.unknownViewError(name)
		}
		for _, c := range view.Clauses() {
			if !seen[c.Name] {
				seen[c.Name] = true
				result = append(result, c)
			}
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func (r *Registry) unknownViewError(name string) error {
	if s := r.closest(name); s != "" {
		return fmt.Errorf("unknown view %q (did you mean %q?); run 'og views list'", name, s)
	}
	return fmt.Errorf("unknown view %q; run 'og views list'", name)
}

// closest returns the registered view name nearest to the given one,
// or "" when nothing is close enough to be a plausible typo.
func (r *Registry) closest(name string) string {
	best, bestDist := "", 3 // tolerate up to 2 edits
	for candidate := range r.views {
		if d := editDistance(strings.ToLower(name), candidate); d < bestDist {
			best, bestDist = candidate, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

// --- YAML parsing ---

type fileDoc struct {
	Views map[string]viewDoc `yaml:"views"`
}

type viewDoc struct {
	Description string     `yaml:"description"`
	Fields      []fieldDoc `yaml:"fields"`
}

// fieldDoc accepts the string shorthand ("wt", "wt@at") or the long form
// {name: ..., at: true, alias: ...}.
type fieldDoc struct {
	Field
}

func (f *fieldDoc) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		var s string
		if err := node.Decode(&s); err != nil {
			return err
		}
		if s = strings.TrimSpace(s); s == "" {
			return fmt.Errorf("empty field name")
		}
		if name, found := strings.CutSuffix(s, "@at"); found {
			f.Field = Field{Name: name, At: true}
		} else {
			f.Field = Field{Name: s}
		}
		return nil
	}

	var long struct {
		Name  string `yaml:"name"`
		At    bool   `yaml:"at"`
		Alias string `yaml:"alias"`
	}
	if err := node.Decode(&long); err != nil {
		return err
	}
	if long.Name == "" {
		return fmt.Errorf("field entry is missing 'name'")
	}
	if name, found := strings.CutSuffix(long.Name, "@at"); found {
		long.Name, long.At = name, true
	}
	f.Field = Field{Name: long.Name, At: long.At, Alias: long.Alias}
	return nil
}

// parseFile parses one views YAML document.
func parseFile(data []byte, source string) (map[string]Definition, error) {
	var doc fileDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", source, err)
	}
	if len(doc.Views) == 0 {
		return nil, fmt.Errorf("parsing %s: no views defined (expected top-level 'views:' map)", source)
	}

	out := make(map[string]Definition, len(doc.Views))
	for name, v := range doc.Views {
		if len(v.Fields) == 0 {
			return nil, fmt.Errorf("parsing %s: view %q has no fields", source, name)
		}
		fields := make([]Field, len(v.Fields))
		for i, fd := range v.Fields {
			fields[i] = fd.Field
		}
		out[name] = Definition{
			Name:        name,
			Description: v.Description,
			Fields:      fields,
			Source:      source,
		}
	}
	return out, nil
}
