package opengate

import (
	"errors"
	"fmt"
)

// NotFoundError reports that the platform holds no artifact under the
// identifier that was asked for.
//
// OpenGate answers a single-artifact lookup that matches nothing with **HTTP
// 204 and an empty body**, not with a 404 (verified live 2026-09-10 for
// datamodels, workspaces, dashboards and devices). A getter that unmarshals
// that body straight away fails with "unexpected end of JSON input", and one
// that passes the bytes through returns nothing at all — `og dev get` printed
// an empty table and exited 0. Both report a missing artifact as anything but.
//
// Callers distinguish it with errors.As or the IsNotFound helper.
type NotFoundError struct {
	// Kind names the artifact family, for the message only.
	Kind string
	// Identifier is what the caller asked for.
	Identifier string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no %s with identifier %q (the platform answered HTTP 204 No Content)", e.Kind, e.Identifier)
}

// IsNotFound reports whether err is the platform saying it has no such
// artifact.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}

// notFoundIfEmpty turns a 204-or-empty response into a NotFoundError. It is
// called after CheckResponse, so a genuine error has already been returned.
//
// Only single-artifact reads may use it: on a list or a catalog an empty
// response means "none yet", which is an answer and not a failure.
func notFoundIfEmpty(data []byte, statusCode int, kind, id string) error {
	if IsEmptyResponse(data, statusCode) {
		return &NotFoundError{Kind: kind, Identifier: id}
	}
	return nil
}
