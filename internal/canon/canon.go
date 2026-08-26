// Package canon produces a stable, comparable form of an artifact payload.
//
// It exists because "wrap is lossless" is only true modulo cosmetic
// differences, and those differences are what make a naive textual diff
// unreadable: object key order is arbitrary, a field can be absent or null with
// the same meaning, and several fields are managed by the server or derived
// from whoever is asking. Comparing raw JSON reports hundreds of lines of noise
// and buries the one line that matters.
//
// Canonicalize answers "are these the same artifact?". It deliberately does NOT
// reorder arrays: the order of a dashboard's grid, or of a rule's datastreams,
// is meaningful.
package canon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// Scope selects which fields participate in a comparison.
type Scope int

const (
	// SameTenant compares an artifact against itself — a local tree against the
	// remote it was pulled from. Identity fields are stable and must be
	// compared: a changed id means something is wrong.
	SameTenant Scope = iota

	// CrossTenant compares the same logical artifact in two organizations, as
	// `diff --against <profile>` does. Identifiers and ownership differ by
	// construction there, so comparing them reports noise on every artifact.
	CrossTenant
)

// volatileFields are server-managed or requester-derived: they change without
// anyone editing the artifact, so they never participate in a comparison.
//
// Verified against real API responses in pkg/opengate/testdata:
//
//	__v              Mongo document version, bumped on every save
//	lastAccess       timestamp of the last read
//	allowedProfiles  derived from the profile making the request
//	editable         derived from whether the requester owns the artifact
//
// The last two are the reason two developers pulling the same dashboard get
// different JSON.
var volatileFields = map[unwrap.Kind][]string{
	unwrap.KindWorkspace: {"__v", "lastAccess", "allowedProfiles", "editable"},
	unwrap.KindDashboard: {"__v", "lastAccess", "allowedProfiles", "editable"},

	// A connector function GET returns `errors`, which the platform writes when
	// the function fails — not something a developer edits. Observed in
	// production (always null on sensehat's current functions, so clean() drops
	// it today anyway); listed so a populated one is never reported as your
	// change. Remove it if it turns out to be editable.
	unwrap.KindConnectorFunction: {"errors"},

	// Rules and provision functions: no server-managed field observed in a real
	// payload. Verified against production, not assumed — a rule GET returns
	// exactly what was written, and a provision processor only
	// configurationParams, name, provisionProcessorId and scriptProcessor.
	// Left empty rather than guessed: a wrongly listed field hides a real change.
	unwrap.KindRule:              {},
	unwrap.KindProvisionFunction: {},
}

// identityFields locate an artifact in one organization. They are compared
// within a tenant and ignored across tenants.
var identityFields = map[unwrap.Kind][]string{
	unwrap.KindWorkspace: {"_id", "id", "owner"},
	unwrap.KindDashboard: {"_id", "id", "owner", "workspaces"},

	// Verified against sensehat in production: a rule GET returns `organization`
	// and `channel`, NOT the organizationId/channelId of the search summary
	// struct. Both spellings are listed because the API is inconsistent about
	// this elsewhere, and an extra name here costs nothing.
	unwrap.KindRule:              {"identifier", "organization", "channel", "organizationId", "channelId"},
	unwrap.KindConnectorFunction: {"identifier"},
	unwrap.KindProvisionFunction: {"provisionProcessorId"},
}

// Options controls one canonicalization.
type Options struct {
	Kind  unwrap.Kind
	Scope Scope
	// Extra names additional top-level fields to drop, for a caller that knows
	// something this package does not.
	Extra []string
}

// Canonicalize returns the comparable form of a payload: keys sorted, nulls and
// empty containers dropped, volatile fields removed.
func Canonicalize(payload json.RawMessage, o Options) ([]byte, error) {
	var node any
	if err := json.Unmarshal(payload, &node); err != nil {
		return nil, fmt.Errorf("parsing payload: %w", err)
	}

	drop := map[string]bool{}
	for _, f := range volatileFields[o.Kind] {
		drop[f] = true
	}
	if o.Scope == CrossTenant {
		for _, f := range identityFields[o.Kind] {
			drop[f] = true
		}
	}
	for _, f := range o.Extra {
		drop[f] = true
	}

	// Dropping happens at every depth, not just at the top level: a workspace
	// payload embeds its dashboards, each carrying its own __v, lastAccess and
	// requester-derived flags. Comparing two workspaces while only stripping the
	// outer ones reports noise from every nested dashboard.
	//
	// The trade-off is that a field with one of these names and a genuine
	// meaning deeper in the tree — an `editable` inside a widget config — is
	// also ignored. That costs a missed difference in an unlikely place, against
	// guaranteed noise in a common one. It only ever affects comparison; nothing
	// here touches a payload that gets deployed.
	cleaned := clean(node, drop)

	// json.Marshal sorts map keys, which is the key-order half of the job.
	out, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("encoding canonical form: %w", err)
	}
	return out, nil
}

// Hash is the digest of the canonical form, for change detection.
func Hash(payload json.RawMessage, o Options) (string, error) {
	c, err := Canonicalize(payload, o)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return hex.EncodeToString(sum[:]), nil
}

// Equal reports whether two payloads are the same artifact.
func Equal(a, b json.RawMessage, o Options) (bool, error) {
	ha, err := Hash(a, o)
	if err != nil {
		return false, err
	}
	hb, err := Hash(b, o)
	if err != nil {
		return false, err
	}
	return ha == hb, nil
}

// clean drops null values and empty objects/arrays, recursively.
//
// A field that is absent and a field that is null carry the same meaning in
// these payloads, and the platform is inconsistent about which it returns —
// that inconsistency is the single largest source of diff noise. Empty
// containers go for the same reason: the API omits an empty list on one
// endpoint and returns [] on another.
//
// Arrays keep their order and their length: a null inside an array is a
// position, not an absence.
func clean(node any, drop map[string]bool) any {
	switch v := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, child := range v {
			if drop[k] {
				continue
			}
			c := clean(child, drop)
			if isEmptyValue(c) {
				continue
			}
			out[k] = c
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = clean(child, drop)
		}
		return out
	default:
		return v
	}
}

// isEmptyValue reports whether a cleaned value carries no information.
func isEmptyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case map[string]any:
		return len(t) == 0
	case []any:
		return len(t) == 0
	default:
		return false
	}
}

// VolatileFields reports the fields excluded from comparison for a kind, so a
// command can explain to the user what it is ignoring.
func VolatileFields(kind unwrap.Kind, scope Scope) []string {
	out := append([]string{}, volatileFields[kind]...)
	if scope == CrossTenant {
		out = append(out, identityFields[kind]...)
	}
	return out
}

// String renders a scope for messages.
func (s Scope) String() string {
	if s == CrossTenant {
		return "cross-tenant"
	}
	return "same-tenant"
}

// Diagnose returns a human-readable summary of what canonicalization removed,
// for troubleshooting an unexpected "no differences".
func Diagnose(payload json.RawMessage, o Options) string {
	var node map[string]any
	if json.Unmarshal(payload, &node) != nil {
		return ""
	}
	var dropped []string
	for _, f := range VolatileFields(o.Kind, o.Scope) {
		if _, present := node[f]; present {
			dropped = append(dropped, f)
		}
	}
	if len(dropped) == 0 {
		return ""
	}
	return "ignored (" + o.Scope.String() + "): " + strings.Join(dropped, ", ")
}
