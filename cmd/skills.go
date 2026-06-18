package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// embedRoot is the path, inside the embedded FS, under which the skill
// directories live. It mirrors the on-disk layout in the repo.
const embedRoot = ".claude/skills"

// skillsFS holds the embedded Claude Code skills, injected from package main
// (the only package that can go:embed the repo-root .claude/skills tree).
var skillsFS fs.FS

// SetSkillsFS wires the embedded skills filesystem from main. It is a seam so
// that the embed directive can stay in package main while the command logic
// lives here.
func SetSkillsFS(f fs.FS) { skillsFS = f }

var (
	skillsDir    string
	skillsGlobal bool
	skillsForce  bool
	skillsBackup bool
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage the Claude Code skills bundled with og",
	Long: "The og binary ships with the Claude Code skills (og-cli, og-workspaces,\n" +
		"og-device-ops) embedded inside it, versioned together with the binary.\n" +
		"Use these commands to list them or extract them onto disk so Claude Code\n" +
		"can discover them — without cloning the repository.",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the skills embedded in this og binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := embeddedSkillNames()
		if err != nil {
			return err
		}
		fmt.Printf("Skills embedded in og %s:\n", version)
		for _, n := range names {
			fmt.Printf("  - %s\n", n)
		}
		return nil
	},
}

var skillsExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract the embedded skills onto disk",
	Long: "Extract the bundled Claude Code skills so Claude Code can discover them.\n\n" +
		"By default they are written to ./.claude/skills/ (project-local). Use\n" +
		"--global to write to ~/.claude/skills/ (available in any directory), or\n" +
		"--dir to choose an arbitrary destination.\n\n" +
		"Each skill is replaced as a whole: if it already exists, the entire\n" +
		"directory is removed and rewritten (no file-by-file merge that could\n" +
		"leave stale files behind). For safety, extract aborts if any skill\n" +
		"already exists — pass --force to overwrite, optionally with --backup to\n" +
		"keep a <skill>.bak copy of what was there. Only the skills og manages are\n" +
		"touched; other skills in the destination are left untouched.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if skillsDir != "" && skillsGlobal {
			return fmt.Errorf("--dir and --global are mutually exclusive")
		}

		root, err := skillsSubFS()
		if err != nil {
			return err
		}
		names, err := embeddedSkillNames()
		if err != nil {
			return err
		}

		destRoot, err := resolveDestRoot()
		if err != nil {
			return err
		}

		// Detect collisions up front so we can abort cleanly before touching
		// anything when --force was not given.
		var existing []string
		for _, n := range names {
			if _, err := os.Stat(filepath.Join(destRoot, n)); err == nil {
				existing = append(existing, n)
			}
		}
		if len(existing) > 0 && !skillsForce {
			return fmt.Errorf(
				"these skills already exist in %s:\n  %s\n\n"+
					"Refusing to overwrite. Re-run with --force to replace them "+
					"(add --backup to keep a .bak copy of each).",
				destRoot, strings.Join(existing, "\n  "))
		}

		if err := os.MkdirAll(destRoot, 0o755); err != nil {
			return fmt.Errorf("creating destination %s: %w", destRoot, err)
		}

		for _, n := range names {
			if err := extractSkill(root, n, destRoot); err != nil {
				return fmt.Errorf("extracting %s: %w", n, err)
			}
			fmt.Printf("  ✓ %s\n", filepath.Join(destRoot, n))
		}
		fmt.Printf("Extracted %d skill(s) to %s\n", len(names), destRoot)
		return nil
	},
}

func init() {
	skillsExtractCmd.Flags().StringVar(&skillsDir, "dir", "", "destination directory (default ./.claude/skills)")
	skillsExtractCmd.Flags().BoolVar(&skillsGlobal, "global", false, "extract to ~/.claude/skills (available in any directory)")
	skillsExtractCmd.Flags().BoolVar(&skillsForce, "force", false, "overwrite skills that already exist")
	skillsExtractCmd.Flags().BoolVar(&skillsBackup, "backup", false, "with --force, rename an existing skill to <skill>.bak before replacing")

	skillsCmd.AddCommand(skillsListCmd, skillsExtractCmd)
	rootCmd.AddCommand(skillsCmd)
}

// skillsSubFS returns the embedded FS rooted at the skills directory, so its
// top-level entries are the skill names.
func skillsSubFS() (fs.FS, error) {
	if skillsFS == nil {
		return nil, fmt.Errorf("no skills embedded in this binary")
	}
	sub, err := fs.Sub(skillsFS, embedRoot)
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	return sub, nil
}

// embeddedSkillNames lists the top-level skill directories in the embed,
// sorted. This set defines exactly which skills og manages on disk.
func embeddedSkillNames() ([]string, error) {
	root, err := skillsSubFS()
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(root, ".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no skills embedded in this binary")
	}
	return names, nil
}

// resolveDestRoot computes the destination skills directory from the flags:
// --dir wins, then --global (~/.claude/skills), else ./.claude/skills.
func resolveDestRoot() (string, error) {
	switch {
	case skillsDir != "":
		return skillsDir, nil
	case skillsGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		return filepath.Join(home, ".claude", "skills"), nil
	default:
		return filepath.Join(".claude", "skills"), nil
	}
}

// extractSkill writes a single skill onto disk, replacing any existing copy
// atomically: the subtree is written to a temp sibling directory and only then
// swapped in, so a failure mid-write never leaves a half-written skill. If the
// skill already exists it is removed first (or moved aside to <skill>.bak when
// --backup is set).
func extractSkill(root fs.FS, name, destRoot string) error {
	final := filepath.Join(destRoot, name)

	tmp, err := os.MkdirTemp(destRoot, "."+name+".extract-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmp) // no-op once renamed away

	if err := writeSubtree(root, name, tmp); err != nil {
		return err
	}

	if _, err := os.Stat(final); err == nil {
		if skillsBackup {
			bak := final + ".bak"
			if err := os.RemoveAll(bak); err != nil {
				return fmt.Errorf("clearing previous backup %s: %w", bak, err)
			}
			if err := os.Rename(final, bak); err != nil {
				return fmt.Errorf("backing up to %s: %w", bak, err)
			}
		} else if err := os.RemoveAll(final); err != nil {
			return fmt.Errorf("removing existing %s: %w", final, err)
		}
	}

	if err := os.Rename(tmp, final); err != nil {
		return fmt.Errorf("installing %s: %w", final, err)
	}
	return nil
}

// writeSubtree copies the embedded subtree srcDir into the on-disk dstDir.
func writeSubtree(srcFS fs.FS, srcDir, dstDir string) error {
	return fs.WalkDir(srcFS, srcDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(srcFS, p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
