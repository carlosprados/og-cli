package views

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"

	"github.com/carlosprados/og-cli/internal/config"
)

// SourceBuiltin marks views embedded in the binary.
const SourceBuiltin = "builtin"

// Load builds the registry from all layers: builtin, then ~/.og/views/*.yaml,
// then ./.og/views/*.yaml. Later layers override earlier ones by view name.
func Load() (*Registry, error) {
	merged, err := parseFile(builtinYAML, SourceBuiltin)
	if err != nil {
		return nil, fmt.Errorf("builtin views: %w", err) // would be a packaging bug
	}

	for _, dir := range layerDirs() {
		layer, err := loadDir(dir)
		if err != nil {
			return nil, err
		}
		maps.Copy(merged, layer)
	}

	return &Registry{views: merged}, nil
}

// layerDirs returns the user and project views directories, in precedence order.
func layerDirs() []string {
	var dirs []string
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, config.DefaultConfigDir, "views"))
	}
	dirs = append(dirs, filepath.Join(".", config.DefaultConfigDir, "views"))
	return dirs
}

// loadDir parses every *.yaml file in dir. Two files in the same directory
// defining the same view name is an error.
func loadDir(dir string) (map[string]Definition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading views dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := filepath.Ext(e.Name()); ext == ".yaml" || ext == ".yml" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	layer := make(map[string]Definition)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}
		defs, err := parseFile(data, file)
		if err != nil {
			return nil, err
		}
		for name, def := range defs {
			if existing, ok := layer[name]; ok {
				return nil, fmt.Errorf("view %q defined twice in the same layer: %s and %s", name, existing.Source, file)
			}
			layer[name] = def
		}
	}
	return layer, nil
}
