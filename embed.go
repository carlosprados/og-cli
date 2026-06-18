package main

import "embed"

// skillsFS embeds the Claude Code skills shipped with og so that `og skills
// extract` can drop them onto disk without cloning the repo. The "all:" prefix
// is required because the .claude directory starts with a dot, which go:embed
// excludes by default. The embed lives in package main because go:embed cannot
// reference paths outside its own package directory, and only main sits at the
// module root where .claude/skills lives — keeping a single source of truth.
//
//go:embed all:.claude/skills
var skillsFS embed.FS
