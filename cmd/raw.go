package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// `--raw` on a get prints the platform's response bytes verbatim.
//
// It exists because three families — datamodels, datasets, workspaces — decode
// into Go structs on the way out, and a struct only carries the fields this
// package knows about. Everything else is dropped silently, which is harmless
// for reading and for editing but wrong for a backup: a datamodel read through
// the struct used to lose `indexed` and `notFilterable`, and a dataset lost its
// whole `sorts` catalogue. Those fields are modelled now, but the next one the
// platform adds would go the same way, so the fix is an escape hatch that does
// not depend on this package being up to date.
//
// The other families need no flag: they already return json.RawMessage and pass
// the bytes through, as does the pull/wrap/deploy lifecycle in internal/unwrap.
//
// This is a bytes contract, so it ignores -o: no envelope, no re-indentation,
// no table. Pipe it through jq to read it. That is the same choice `show
// --path` makes, and for the same reason — a command that sometimes emits a
// formatted view and sometimes exact bytes serves neither well.
const rawFlagUsage = "print the platform's exact response bytes (ignores -o; pipe through jq to read)"

// printRaw writes response bytes to stdout unchanged, adding only a trailing
// newline so the output does not run into the next shell prompt.
func printRaw(b json.RawMessage) error {
	if _, err := os.Stdout.Write(b); err != nil {
		return err
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		_, err := fmt.Fprintln(os.Stdout)
		return err
	}
	return nil
}
