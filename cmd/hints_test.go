package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/carlosprados/og-cli/v2/pkg/opengate"
	"github.com/spf13/cobra"
)

// family builds a command tree shaped like a real one: a family with a lister
// and a get, so the hint has a sibling to point at.
func family(t *testing.T, listerName string, args ...string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "og"}
	fam := &cobra.Command{Use: "timeseries"}
	get := &cobra.Command{Use: "get", RunE: func(*cobra.Command, []string) error { return nil }}
	if listerName != "" {
		fam.AddCommand(&cobra.Command{Use: listerName})
	}
	fam.AddCommand(get)
	root.AddCommand(fam)
	if err := get.Flags().Parse(args); err != nil {
		t.Fatalf("parsing args: %v", err)
	}
	return get
}

func notFound(field, value string) error {
	return &opengate.APIError{
		StatusCode: http.StatusNotFound,
		Message:    "No resource found",
		Fields:     []string{field},
		Context:    []opengate.ErrorContext{{Name: field, Value: value}},
	}
}

func TestExplainNotFoundSuggestsMappingAName(t *testing.T) {
	err := explainNotFound(family(t, "list", "ConsumoDiarioMeter"), notFound("identifier", "ConsumoDiarioMeter"))

	want := `If "ConsumoDiarioMeter" is a name, map it to an identifier with: og timeseries list`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
	// The original error must stay both visible and reachable.
	if !strings.Contains(err.Error(), "No resource found") {
		t.Error("the platform's own message was lost")
	}
	var apiErr *opengate.APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As no longer finds the APIError through the hint")
	}
	if got := ExitCode(err); got != ExitFailure {
		t.Errorf("ExitCode = %d, want %d", got, ExitFailure)
	}
	if !ShouldPrint(err) {
		t.Error("ShouldPrint = false, want true")
	}
}

// A well-formed identifier that simply does not exist must not be told to map
// its "name" — that read as nonsense against a live tenant.
func TestExplainNotFoundWordsAnIdentifierDifferently(t *testing.T) {
	for _, id := range []string{"000000000000000000000000", "0dcf4b14-8ecb-4351-85fa-95a7f7b8fd44"} {
		err := explainNotFound(family(t, "list", id), notFound("identifier", id))
		if want := "No artifact has that identifier"; !strings.Contains(err.Error(), want) {
			t.Errorf("for %s: error = %q, want it to contain %q", id, err.Error(), want)
		}
		if strings.Contains(err.Error(), "is a name") {
			t.Errorf("for %s: hint calls a well-formed identifier a name", id)
		}
	}
}

// datasets and rules answer 400 with the explanation in the message and no
// context at all; the hint has to reach those too. Captured live 2026-09-10.
func TestExplainNotFoundHandlesMessageOnly400(t *testing.T) {
	for _, msg := range []string{"Element not found.", "No rule has been found with this id"} {
		apiErr := &opengate.APIError{StatusCode: http.StatusBadRequest, Message: msg}
		err := explainNotFound(family(t, "list", "Battery low"), apiErr)
		if want := "is a name"; !strings.Contains(err.Error(), want) {
			t.Errorf("for %q: error = %q, want a hint", msg, err.Error())
		}
	}
}

// The identifier key is not `identifier` everywhere.
func TestExplainNotFoundRecognisesOtherIDKeys(t *testing.T) {
	for _, field := range []string{"connectorFunctionId", "provisionProcessorId"} {
		err := explainNotFound(family(t, "list", "AddDevices"), notFound(field, "AddDevices"))
		if !strings.Contains(err.Error(), "is a name") {
			t.Errorf("for %s: no hint added: %q", field, err.Error())
		}
	}
}

func TestExplainNotFoundLeavesOtherErrorsAlone(t *testing.T) {
	cases := map[string]struct {
		cmd *cobra.Command
		err error
	}{
		"a 400 that is not about a missing artifact": {
			family(t, "list", "MRG"),
			&opengate.APIError{StatusCode: http.StatusBadRequest, Message: "Organization not exists"},
		},
		"an error that is not an APIError": {
			family(t, "list", "X"), fmt.Errorf("dial tcp: connection refused"),
		},
		"a family with nothing to list": {
			family(t, "", "X"), notFound("identifier", "X"),
		},
		"a command given no argument": {
			family(t, "list"), notFound("identifier", "X"),
		},
		"a 500": {
			family(t, "list", "X"),
			&opengate.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"},
		},
	}
	for name, tc := range cases {
		got := explainNotFound(tc.cmd, tc.err)
		if got.Error() != tc.err.Error() {
			t.Errorf("%s: error was annotated: %q", name, got.Error())
		}
	}
}

// A silent ExitError must stay silent through the wrapper.
func TestExplainNotFoundKeepsExitErrorSemantics(t *testing.T) {
	if got := ExitCode(explainNotFound(family(t, "list", "X"), &ExitError{Code: ExitDiff})); got != ExitDiff {
		t.Errorf("ExitCode = %d, want %d", got, ExitDiff)
	}
}

func TestExplainNotFoundToleratesNilCommand(t *testing.T) {
	err := notFound("identifier", "X")
	if got := explainNotFound(nil, err); got != err {
		t.Error("a nil command must pass the error through untouched")
	}
	if got := explainNotFound(family(t, "list", "X"), nil); got != nil {
		t.Errorf("a nil error must stay nil, got %v", got)
	}
}

// Workspaces, dashboards, datamodels and devices answer a missing artifact
// with 204 and an empty body, which the client reports as a NotFoundError
// rather than an APIError. The hint has to reach those too.
func TestExplainNotFoundReachesTheEmptyBodyShape(t *testing.T) {
	err := explainNotFound(
		family(t, "list", "Operations"),
		&opengate.NotFoundError{Kind: "workspace", Identifier: "Operations"},
	)
	want := `If "Operations" is a name, map it to an identifier with: og timeseries list`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
	if !opengate.IsNotFound(err) {
		t.Error("IsNotFound no longer sees through the hint")
	}
}
