package cmd

import (
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/config"
)

// isOwnedByProfile reports whether the given resource owner (email string from
// the workspace/dashboard JSON) matches the active profile's email. An empty
// owner (null in the JSON, typical of system/shared items) is never considered
// owned. Comparison is case-insensitive and trims whitespace.
func isOwnedByProfile(owner string, p *config.Profile) bool {
	if p == nil {
		return false
	}
	o := strings.TrimSpace(owner)
	e := strings.TrimSpace(p.Email)
	if o == "" || e == "" {
		return false
	}
	return strings.EqualFold(o, e)
}

// requireOwnership returns an error if the resource is not owned by the active
// profile, unless force is true. Used by the single-item unwrap commands when
// the user has explicitly requested a specific id or file — in that case we
// refuse loudly so the user knows the item is not editable.
func requireOwnership(kind, id, owner string, p *config.Profile, force bool) error {
	if force || isOwnedByProfile(owner, p) {
		return nil
	}
	profileEmail := ""
	if p != nil {
		profileEmail = p.Email
	}
	switch {
	case owner == "":
		return fmt.Errorf("%s %q has no owner (likely a system/shared item) — refusing to unwrap; pass --force to override",
			kind, id)
	case profileEmail == "":
		return fmt.Errorf("%s %q owner is %q but the active profile has no email set — refusing to unwrap; pass --force to override",
			kind, id, owner)
	default:
		return fmt.Errorf("%s %q owner is %q, not %q — not editable by you; pass --force to override",
			kind, id, owner, profileEmail)
	}
}
