package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// fakeSkills builds an embed-shaped FS rooted at .claude/skills with two skills,
// mirroring how the real binary embeds them.
func fakeSkills() fstest.MapFS {
	return fstest.MapFS{
		".claude/skills/og-cli/SKILL.md":            {Data: []byte("cli v2")},
		".claude/skills/og-cli/query-cookbook.md":   {Data: []byte("recipes")},
		".claude/skills/og-device-ops/SKILL.md":     {Data: []byte("ops v2")},
		".claude/skills/og-device-ops/templates.md": {Data: []byte("tpl")},
	}
}

// withSkills installs the fake embed FS and resets the extract flags for the
// duration of a test.
func withSkills(t *testing.T) {
	t.Helper()
	prevFS := skillsFS
	prevDir, prevGlobal, prevForce, prevBackup := skillsDir, skillsGlobal, skillsForce, skillsBackup
	skillsFS = fakeSkills()
	skillsDir, skillsGlobal, skillsForce, skillsBackup = "", false, false, false
	t.Cleanup(func() {
		skillsFS = prevFS
		skillsDir, skillsGlobal, skillsForce, skillsBackup = prevDir, prevGlobal, prevForce, prevBackup
	})
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

func runExtract(dir string) error {
	skillsDir = dir
	return skillsExtractCmd.RunE(skillsExtractCmd, nil)
}

func TestEmbeddedSkillNames(t *testing.T) {
	withSkills(t)
	names, err := embeddedSkillNames()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"og-cli", "og-device-ops"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Fatalf("got %v, want %v", names, want)
	}
}

func TestExtractFreshDestination(t *testing.T) {
	withSkills(t)
	dest := t.TempDir()
	if err := runExtract(dest); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if got := read(t, filepath.Join(dest, "og-cli", "SKILL.md")); got != "cli v2" {
		t.Fatalf("og-cli/SKILL.md = %q", got)
	}
	if got := read(t, filepath.Join(dest, "og-device-ops", "templates.md")); got != "tpl" {
		t.Fatalf("og-device-ops/templates.md = %q", got)
	}
}

func TestExtractAbortsOnCollision(t *testing.T) {
	withSkills(t)
	dest := t.TempDir()
	mkSkill(t, dest, "og-cli", "SKILL.md", "local edit")

	err := runExtract(dest)
	if err == nil {
		t.Fatal("expected abort on collision, got nil")
	}
	// Untouched: the abort must happen before any write.
	if got := read(t, filepath.Join(dest, "og-cli", "SKILL.md")); got != "local edit" {
		t.Fatalf("collision overwrote without --force: %q", got)
	}
}

func TestExtractForceReplacesWholesale(t *testing.T) {
	withSkills(t)
	dest := t.TempDir()
	// Pre-existing skill with a stale file that is NOT in the embed.
	mkSkill(t, dest, "og-cli", "SKILL.md", "old")
	mkSkill(t, dest, "og-cli", "stale.md", "orphan")

	skillsForce = true
	if err := runExtract(dest); err != nil {
		t.Fatalf("extract --force: %v", err)
	}
	if got := read(t, filepath.Join(dest, "og-cli", "SKILL.md")); got != "cli v2" {
		t.Fatalf("SKILL.md not refreshed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "og-cli", "stale.md")); !os.IsNotExist(err) {
		t.Fatal("stale orphan file survived wholesale replace")
	}
}

func TestExtractBackup(t *testing.T) {
	withSkills(t)
	dest := t.TempDir()
	mkSkill(t, dest, "og-cli", "SKILL.md", "precious")

	skillsForce, skillsBackup = true, true
	if err := runExtract(dest); err != nil {
		t.Fatalf("extract --force --backup: %v", err)
	}
	if got := read(t, filepath.Join(dest, "og-cli.bak", "SKILL.md")); got != "precious" {
		t.Fatalf("backup did not preserve previous content: %q", got)
	}
	if got := read(t, filepath.Join(dest, "og-cli", "SKILL.md")); got != "cli v2" {
		t.Fatalf("SKILL.md not refreshed: %q", got)
	}
}

func TestExtractLeavesForeignSkillsUntouched(t *testing.T) {
	withSkills(t)
	dest := t.TempDir()
	mkSkill(t, dest, "third-party", "SKILL.md", "do not touch")

	skillsForce = true
	if err := runExtract(dest); err != nil {
		t.Fatalf("extract --force: %v", err)
	}
	if got := read(t, filepath.Join(dest, "third-party", "SKILL.md")); got != "do not touch" {
		t.Fatalf("foreign skill was modified: %q", got)
	}
}

func mkSkill(t *testing.T, dest, skill, file, content string) {
	t.Helper()
	dir := filepath.Join(dest, skill)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
