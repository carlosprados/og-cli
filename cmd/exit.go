package cmd

import "errors"

// Exit codes. Consistent across diff and validate so they can gate CI:
//
//	0  success, or no differences
//	1  differences found
//	2  error
//
// Without this distinction `og diff --exit-code` cannot be used as a drift
// gate: a pipeline could not tell "the tenant has drifted" from "the command
// could not run".
const (
	ExitOK      = 0
	ExitDiff    = 1
	ExitFailure = 2
)

// ExitError carries an explicit exit code out of a command.
//
// A nil Err is silent: the code is the whole message. That is what `--exit-code`
// needs — finding differences is the command working correctly, so printing
// "Error: differences found" would be wrong.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// Silent reports whether this error should be printed.
func (e *ExitError) Silent() bool { return e.Err == nil }

// ExitCode maps an error returned by Execute to a process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitFailure
}

// ShouldPrint reports whether Execute's error needs printing.
//
// Cobra's own error printing is disabled (SilenceErrors) so that a silent
// ExitError does not surface as a bare "Error:". Everything else prints
// exactly as before.
func ShouldPrint(err error) bool {
	if err == nil {
		return false
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return !ee.Silent()
	}
	return true
}

// diffFound returns the sentinel for "differences exist", used when
// --exit-code is set.
func diffFound() error { return &ExitError{Code: ExitDiff} }
