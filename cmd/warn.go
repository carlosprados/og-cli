package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// hintWarner prints extraction warnings to stderr, so they stay out of piped
// JSON output. Warnings are how an undeclared code field becomes visible
// instead of silently changing the shape of the artifact directory.
func hintWarner() unwrap.WarnFunc {
	return func(w unwrap.Warning) {
		fmt.Fprintf(os.Stderr, "  hint: %s\n", w)
	}
}

// unwrapArtifactTo explodes one flat artifact into dir, naming it in any error.
// One helper for rules, connector functions and provision functions: they
// differ only in their descriptor, which already knows where the name lives.
func unwrapArtifactTo(d unwrap.Descriptor, raw json.RawMessage, dir string, opts *unwrap.Options) (string, error) {
	artifactDir, err := d.Unwrap(raw, dir, opts)
	if err != nil {
		name := d.NameOf(raw)
		if name == "" {
			name = "(unnamed)"
		}
		return "", fmt.Errorf("%s %s: %w", d.Kind, name, err)
	}
	return artifactDir, nil
}
