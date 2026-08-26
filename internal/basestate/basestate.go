// Package basestate records what a pull actually fetched, so later commands can
// tell local edits from remote ones.
//
// Two-way comparison — local tree against remote artifact — can only say that
// they differ. It cannot say who changed what, which makes any automatic deploy
// a coin flip between publishing your work and silently overwriting someone
// else's. Keeping the canonical remote state from pull time turns that into a
// three-way comparison with four distinguishable outcomes.
//
// It also fixes a footgun that exists today: nothing in an unwrapped directory
// records where it came from, so a tree pulled from staging can be deployed to
// production with no warning at all.
//
// The store lives in a `.og/` directory at the root of a pull, found by walking
// up from an artifact directory the way git finds its repository. It is a cache,
// not a source of truth: deleting it loses the ability to classify, nothing else.
package basestate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosprados/og-cli/v2/internal/canon"
	"github.com/carlosprados/og-cli/v2/internal/unwrap"
)

// DirName is the store directory, created at the root of a pull.
const DirName = ".og"

const (
	baseDirName      = "base"
	manifestFileName = "manifest.json"
	manifestVersion  = 1
)

// Entry records one artifact's provenance and the state it was pulled at.
type Entry struct {
	// Kind, ID and Name identify the artifact.
	Kind unwrap.Kind `json:"kind"`
	ID   string      `json:"id"`
	Name string      `json:"name,omitempty"`

	// Hash is the canonical digest of the payload as fetched.
	Hash string `json:"hash"`

	// BaseFile is the snapshot's filename inside .og/base, recorded rather than
	// derived: an artifact identifier is not guaranteed to be a safe filename,
	// and two sanitised identifiers could collide.
	BaseFile string `json:"baseFile"`

	// Dir is the artifact's directory, relative to the store root.
	Dir string `json:"dir"`

	// Where it came from. Deploying somewhere else is usually a mistake, and
	// nothing recorded this before.
	Profile string `json:"profile,omitempty"`
	Host    string `json:"host,omitempty"`
	Org     string `json:"org,omitempty"`
	Channel string `json:"channel,omitempty"`

	PulledAt time.Time `json:"pulledAt"`
}

// Target is where a command is about to act, for comparison against an entry.
type Target struct {
	Profile string
	Host    string
	Org     string
	Channel string
}

// Manifest is the store index.
type Manifest struct {
	SchemaVersion int              `json:"schemaVersion"`
	Entries       map[string]Entry `json:"entries"`
}

// Store is a `.og/` directory.
type Store struct {
	Root string // the directory containing .og
}

// Open returns the store rooted at dir, creating nothing.
func Open(dir string) *Store { return &Store{Root: dir} }

// Find walks up from startDir looking for a `.og/` directory, the way git finds
// its repository root. Returns ok=false when there is none, which is normal:
// trees pulled before this existed have no store.
func Find(startDir string) (*Store, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, false
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, DirName)); err == nil && info.IsDir() {
			return &Store{Root: dir}, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, false
		}
		dir = parent
	}
}

func (s *Store) dir() string          { return filepath.Join(s.Root, DirName) }
func (s *Store) baseDir() string      { return filepath.Join(s.dir(), baseDirName) }
func (s *Store) manifestPath() string { return filepath.Join(s.dir(), manifestFileName) }

// LoadManifest reads the index, returning an empty one when the store does not
// exist yet.
func (s *Store) LoadManifest() (*Manifest, error) {
	data, err := os.ReadFile(s.manifestPath())
	if os.IsNotExist(err) {
		return &Manifest{SchemaVersion: manifestVersion, Entries: map[string]Entry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestFileName, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", manifestFileName, err)
	}
	if m.Entries == nil {
		m.Entries = map[string]Entry{}
	}
	return &m, nil
}

func (s *Store) saveManifest(m *Manifest) error {
	m.SchemaVersion = manifestVersion
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.manifestPath(), append(data, '\n'), 0o644)
}

// Record stores the canonical snapshot of a freshly pulled artifact and indexes
// it. artifactDir is where the artifact was written; payload is what the API
// returned.
func (s *Store) Record(kind unwrap.Kind, id, name, artifactDir string, payload json.RawMessage, target Target) error {
	if id == "" {
		return fmt.Errorf("cannot record a %s with no identifier", kind)
	}

	c, err := canon.Canonicalize(payload, canon.Options{Kind: kind})
	if err != nil {
		return err
	}
	hash, err := canon.Hash(payload, canon.Options{Kind: kind})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.baseDir(), 0o755); err != nil {
		return err
	}
	baseFile := baseFileName(kind, id)
	if err := os.WriteFile(filepath.Join(s.baseDir(), baseFile), append(c, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing base snapshot: %w", err)
	}

	// Normalise both sides before relativising. The store root can arrive
	// absolute (from Find, which walks up) while the artifact directory arrives
	// relative (from a command's argument), and filepath.Rel of one against the
	// other fails — silently storing a path that LookupByDir can never match, so
	// every later classification reports "unknown".
	rel := relativeTo(s.Root, artifactDir)

	m, err := s.LoadManifest()
	if err != nil {
		return err
	}
	m.Entries[entryKey(kind, id)] = Entry{
		Kind: kind, ID: id, Name: name,
		Hash: hash, BaseFile: baseFile, Dir: rel,
		Profile: target.Profile, Host: target.Host, Org: target.Org, Channel: target.Channel,
		PulledAt: time.Now().UTC(),
	}
	return s.saveManifest(m)
}

// relativeTo expresses dir relative to root, with both made absolute first.
// Falls back to dir unchanged when either cannot be resolved.
func relativeTo(root, dir string) string {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return dir
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	rel, err := filepath.Rel(absRoot, absDir)
	if err != nil {
		return dir
	}
	return rel
}

// Lookup returns the recorded entry for an artifact.
func (s *Store) Lookup(kind unwrap.Kind, id string) (Entry, bool) {
	m, err := s.LoadManifest()
	if err != nil {
		return Entry{}, false
	}
	e, ok := m.Entries[entryKey(kind, id)]
	return e, ok
}

// LookupByDir returns the entry recorded for an artifact directory, which is
// what a deploy has in hand.
func (s *Store) LookupByDir(artifactDir string) (Entry, bool) {
	rel := relativeTo(s.Root, artifactDir)
	m, err := s.LoadManifest()
	if err != nil {
		return Entry{}, false
	}
	for _, e := range m.Entries {
		if filepath.Clean(e.Dir) == filepath.Clean(rel) {
			return e, true
		}
	}
	return Entry{}, false
}

// BaseHash returns the hash recorded at pull time.
func (e Entry) BaseHash() string { return e.Hash }

// MovedTo reports how a target differs from where the artifact was pulled from,
// or an empty slice when it matches. Empty fields in either are not compared:
// an entry written before a field existed should not raise a false alarm.
func (e Entry) MovedTo(t Target) []string {
	var diffs []string
	cmp := func(label, was, now string) {
		if was == "" || now == "" || was == now {
			return
		}
		diffs = append(diffs, fmt.Sprintf("%s %s → %s", label, was, now))
	}
	cmp("host", e.Host, t.Host)
	cmp("profile", e.Profile, t.Profile)
	cmp("organization", e.Org, t.Org)
	cmp("channel", e.Channel, t.Channel)
	return diffs
}

// entryKey namespaces an identifier by kind: nothing guarantees a rule and a
// connector function cannot share one.
func entryKey(kind unwrap.Kind, id string) string { return string(kind) + ":" + id }

// baseFileName builds a filesystem-safe snapshot name. The manifest records the
// result, so a collision between two sanitised identifiers cannot silently
// alias one artifact onto another.
func baseFileName(kind unwrap.Kind, id string) string {
	return unwrap.Slugify(string(kind)) + "__" + unwrap.Slugify(id) + ".canon.json"
}

// GitIgnoreLine is what a project should add to .gitignore: the store is a sync
// cache, not a source of truth.
const GitIgnoreLine = DirName + "/"

// Describe renders an entry's provenance for a human.
func (e Entry) Describe() string {
	parts := []string{}
	if e.Org != "" {
		parts = append(parts, "org "+e.Org)
	}
	if e.Channel != "" {
		parts = append(parts, "channel "+e.Channel)
	}
	if e.Profile != "" {
		parts = append(parts, "profile "+e.Profile)
	}
	if e.Host != "" {
		parts = append(parts, e.Host)
	}
	if len(parts) == 0 {
		return "unknown origin"
	}
	return strings.Join(parts, ", ")
}
