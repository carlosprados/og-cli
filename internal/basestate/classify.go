package basestate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// State is the outcome of comparing local, remote and base.
type State int

const (
	// Unknown means there is no base snapshot, so nothing can be attributed.
	// A tree pulled before the store existed, or one built by hand, lands here.
	Unknown State = iota

	// Clean: local and remote both match the base. Nothing to do.
	Clean

	// LocalChanges: you edited it, nobody else did. Safe to deploy.
	LocalChanges

	// RemoteChanges: somebody else edited it, you did not. Pull.
	RemoteChanges

	// Conflict: both sides moved since the pull. Deploying would discard the
	// remote edit with no record of it.
	Conflict
)

// Marker is the single character used in diff output.
func (s State) Marker() string {
	switch s {
	case LocalChanges:
		return "~"
	case RemoteChanges:
		return "↓"
	case Conflict:
		return "!"
	case Clean:
		return " "
	default:
		return "?"
	}
}

func (s State) String() string {
	switch s {
	case Clean:
		return "clean"
	case LocalChanges:
		return "local changes"
	case RemoteChanges:
		return "remote changes"
	case Conflict:
		return "conflict"
	default:
		return "unknown"
	}
}

// DeployAdvice is what a caller should do about a state. watch refuses on
// Conflict unconditionally; deploy warns and asks.
func (s State) SafeToDeploy() bool { return s == LocalChanges || s == Clean || s == Unknown }

// Comparison is the full result, kept so callers can explain themselves.
type Comparison struct {
	State State

	LocalHash  string
	RemoteHash string
	BaseHash   string

	// LocalMatchesBase and RemoteMatchesBase are the two questions the state is
	// derived from, exposed for messages.
	LocalMatchesBase  bool
	RemoteMatchesBase bool
}

// Classify compares a local payload and a remote payload against the recorded
// base.
//
// The base is what makes this a three-way comparison: without it, "they differ"
// is all anyone can say, and an automatic deploy becomes a coin flip between
// publishing your work and discarding someone else's.
func Classify(kind unwrap.Kind, local, remote json.RawMessage, base []byte) (Comparison, error) {
	o := canon.Options{Kind: kind}

	localHash, err := canon.Hash(local, o)
	if err != nil {
		return Comparison{}, fmt.Errorf("hashing local: %w", err)
	}
	remoteHash, err := canon.Hash(remote, o)
	if err != nil {
		return Comparison{}, fmt.Errorf("hashing remote: %w", err)
	}

	c := Comparison{LocalHash: localHash, RemoteHash: remoteHash}

	if len(base) == 0 {
		c.State = Unknown
		return c, nil
	}

	// The base is stored already canonical, so hash it as-is rather than
	// canonicalizing twice — doing it again would be harmless but implies the
	// stored form might not be canonical, which it is.
	baseHash, err := canon.Hash(base, o)
	if err != nil {
		return Comparison{}, fmt.Errorf("hashing base: %w", err)
	}
	c.BaseHash = baseHash
	c.LocalMatchesBase = localHash == baseHash
	c.RemoteMatchesBase = remoteHash == baseHash

	switch {
	case c.LocalMatchesBase && c.RemoteMatchesBase:
		c.State = Clean
	case !c.LocalMatchesBase && c.RemoteMatchesBase:
		c.State = LocalChanges
	case c.LocalMatchesBase && !c.RemoteMatchesBase:
		c.State = RemoteChanges
	default:
		// Both moved. They could have moved to the SAME value — two people
		// making the same edit — which is not a conflict worth blocking.
		if localHash == remoteHash {
			c.State = Clean
		} else {
			c.State = Conflict
		}
	}
	return c, nil
}

// ReadBase returns the stored snapshot for an entry, or nil when absent.
func (s *Store) ReadBase(e Entry) ([]byte, error) {
	if e.BaseFile == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(s.baseDir(), e.BaseFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading base snapshot: %w", err)
	}
	return data, nil
}

// ClassifyArtifact is the whole comparison for one artifact directory: find its
// entry, read its base, and classify.
func (s *Store) ClassifyArtifact(kind unwrap.Kind, artifactDir string, local, remote json.RawMessage) (Comparison, Entry, error) {
	e, ok := s.LookupByDir(artifactDir)
	if !ok {
		c, err := Classify(kind, local, remote, nil)
		return c, Entry{}, err
	}
	base, err := s.ReadBase(e)
	if err != nil {
		return Comparison{}, e, err
	}
	c, err := Classify(kind, local, remote, base)
	return c, e, err
}
