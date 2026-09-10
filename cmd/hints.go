package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// hintedError appends a line of guidance to an error without hiding it: the
// original stays reachable, so ExitCode and ShouldPrint keep seeing through to
// whatever the command actually returned.
type hintedError struct {
	err  error
	hint string
}

func (e *hintedError) Error() string { return e.err.Error() + "\n  " + e.hint }

func (e *hintedError) Unwrap() error { return e.err }

// explainNotFound answers the question the platform's rejection leaves open.
//
// Most families address an artifact by an identifier the platform generated —
// a UUID, a 24-char hex string — while the thing a person knows is the name.
// Pass the name and the server says "No resource found (fields: identifier)",
// naming the field it wanted and nothing about how to get one. That wording
// sent a real user hunting for an API version mismatch that did not exist.
//
// The hint lives here, at the single point every command's error passes
// through, rather than in each of the 43 subcommands that take an identifier.
func explainNotFound(c *cobra.Command, err error) error {
	if c == nil || err == nil {
		return err
	}
	var apiErr *opengate.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	if !notFoundish(apiErr) {
		return err
	}
	// Only a command that was handed something to look up can be answered this
	// way. It also keeps the message-text match below from firing on a search.
	arg := firstArg(c)
	if arg == "" {
		return err
	}
	lister := siblingLister(c)
	if lister == "" {
		return err
	}

	var hint string
	if looksLikeIdentifier(arg) {
		hint = fmt.Sprintf("No artifact has that identifier. List the current ones with: %s", lister)
	} else {
		hint = fmt.Sprintf("If %q is a name, map it to an identifier with: %s", arg, lister)
	}
	return &hintedError{err: err, hint: hint}
}

// notFoundish reports whether the platform is saying "nothing has that
// identifier". The status code does not settle it: timeseries answers 404 with
// the field in its context, while datasets ("Element not found.") and rules
// ("No rule has been found with this id") answer 400 with the whole
// explanation in the message and no context at all. Verified live 2026-09-10.
func notFoundish(e *opengate.APIError) bool {
	if e.StatusCode != http.StatusNotFound && e.StatusCode != http.StatusBadRequest {
		return false
	}
	if namesIDField(e) {
		return true
	}
	msg := strings.ToLower(e.Message)
	return strings.Contains(msg, "not found") || strings.Contains(msg, "has been found")
}

// namesIDField reports whether the error's context names an identifier field.
// The key is not `identifier` everywhere: a connector function answers
// `connectorFunctionId`, a provision processor `provisionProcessorId`.
func namesIDField(e *opengate.APIError) bool {
	for _, c := range e.Context {
		name := strings.ToLower(c.Name)
		if name == "identifier" || strings.HasSuffix(name, "id") {
			return true
		}
	}
	return false
}

var (
	objectIDShape = regexp.MustCompile(`^[0-9a-f]{24}$`)
	uuidShape     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// looksLikeIdentifier recognises the two generated shapes, so the hint does not
// tell someone who passed a well-formed id to go and map their "name".
//
// Shape is allowed to pick the wording here and nowhere else: a wrong guess in
// either direction still leaves a hint that helps, whereas a resolution path
// that guessed wrong would act on the wrong artifact. Workspace identifiers
// include `shared` and `_multisensor_demo_ws`, so this question has no reliable
// answer — only a harmless one.
func looksLikeIdentifier(s string) bool {
	s = strings.ToLower(s)
	return objectIDShape.MatchString(s) || uuidShape.MatchString(s)
}

// siblingLister names the command that maps names to identifiers for this
// family: `list` where there is one, `search` otherwise — rules and datamodels
// enumerate only through search.
func siblingLister(c *cobra.Command) string {
	parent := c.Parent()
	if parent == nil {
		return ""
	}
	for _, want := range []string{"list", "search"} {
		for _, sibling := range parent.Commands() {
			if sibling.Name() == want {
				return sibling.CommandPath()
			}
		}
	}
	return ""
}

// firstArg returns the first positional argument the command was given, which
// is the identifier the user tried.
func firstArg(c *cobra.Command) string {
	if args := c.Flags().Args(); len(args) > 0 {
		return args[0]
	}
	return ""
}
