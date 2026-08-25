package cmd

import (
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
