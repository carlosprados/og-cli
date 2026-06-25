package cmd

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// assumeYes is bound to the global --yes/-y flag; it skips the confirmation
// prompt for destructive operations (delete/cancel).
var assumeYes bool

// confirmDestructive guards a destructive operation (delete, cancel, ...).
//
//   - With --yes it proceeds silently (scripts, CI, agents that have already
//     obtained user consent).
//   - On an interactive terminal it prompts [y/N] and proceeds only on yes.
//   - Without a TTY and without --yes it REFUSES, so non-interactive callers
//     must opt in explicitly — a stray `og dev delete X` in a pipeline or from
//     an agent cannot silently destroy data.
//
// action is a human description, e.g. `delete device "sense-001"`.
func confirmDestructive(action string) error {
	if assumeYes {
		return nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("refusing to %s without confirmation: re-run with --yes (no interactive terminal)", action)
	}
	fmt.Fprintf(os.Stderr, "About to %s. This cannot be undone. Continue? [y/N]: ", action)
	var resp string
	_, _ = fmt.Scanln(&resp)
	switch strings.ToLower(strings.TrimSpace(resp)) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("aborted")
	}
}
