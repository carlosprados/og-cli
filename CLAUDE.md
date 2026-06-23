# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`og` is a CLI tool (binary name: `og`) for the OpenGate IoT platform REST API by Amplía Soluciones. Built in Go with Cobra (commands), Viper (config), and Bubble Tea + Lip Gloss (interactive TUI).

## Commands

```bash
task build          # build the og binary
task test           # go test ./... -v
task lint           # golangci-lint run ./...
task fmt            # gofmt + goimports
task tidy           # go mod tidy
task install        # go install with ldflags
go test ./internal/client/ -run TestLogin -v   # run a single test
```

Version info is injected via ldflags — see Taskfile.yml `LDFLAGS`.

## Architecture

```
main.go              → cmd.Execute()
cmd/                 → Cobra commands (root, login, version, mcp, datamodels, devices)
internal/client/     → OpenGate REST API client (HTTP methods, auth, resource methods)
internal/config/     → Viper config, profiles, .env loading
internal/mcp/        → MCP server (stdio + HTTP transports) + tool definitions
internal/output/     → JSON/table output formatting
internal/query/      → Search filter parser (-w "field op value", query strings)
internal/tui/        → Bubble Tea interactive TUI
```

### Three interfaces — og has three execution modes:

| Mode | Invocation | Implementation |
|------|------------|----------------|
| CLI | `og <command>` | `cmd/` (Cobra) |
| Interactive TUI | `og` (no args) | `internal/tui/` (Bubble Tea) |
| MCP server | `og mcp` | `internal/mcp/` (mcp-go) |

### Surface model: CLI ↔ MCP ↔ TUI (documented split)

A single API operation lives in `pkg/opengate/<method>`; the interfaces are thin
layers over it. The parity expectation is **deliberate and tiered**, not "all three
always" (see `docs/v1-readiness-audit.md` §2):

- **CLI (`cmd/`) — the complete surface.** Every operation gets a CLI verb, including
  local-filesystem lifecycle verbs (`pull`/`wrap`/`deploy`, import/export).
- **MCP (`internal/mcp/`) — full minus local-fs lifecycle.** Every operation a remote
  LLM can drive gets a tool. `pull`/`deploy` (arbitrary local dirs) are intentionally
  CLI-only; MCP offers the inline-body / import-export equivalents. Long-lived streams
  (logs, MQTT subscribe/device) are exposed as **bounded** tools (count/timeout).
- **TUI (`internal/tui/`) — browse + high-value actions.** List/detail/search for every
  resource, plus a few key mutations (alarm attend/close, launch job, rule toggle,
  connector status, workspace share). Full create/update/delete and pull/deploy are
  **not** in the TUI by design; IoT injection has no TUI view (it's an action, not a
  browseable resource).

When adding a new endpoint:
1. Add the client method in `pkg/opengate/`.
2. Add the CLI command in `cmd/` (always).
3. Add the MCP tool in `internal/mcp/` (unless it is purely a local-fs lifecycle verb).
4. Add or extend the TUI view in `internal/tui/` for browse/inspect; add a TUI action
   only if it is a high-value non-form interaction.
5. Ship them in the same PR.
6. Update the relevant skill under `.claude/skills/` (og-cli / og-workspaces / og-device-ops).

**Authentication parity (non-negotiable):** login and auth features — credentials,
2FA/TOTP (code, stored secret, challenge-retry), TLS escape hatches — MUST stay in
lockstep across **all four** surfaces: CLI (`cmd/login.go`), MCP (`internal/mcp/`),
TUI (`internal/tui/login.go`) **and** SKILLS (`.claude/skills/og-cli`) + README. The
TUI is part of this set: it is the default `og` (no-args) entry point, so any auth
capability the CLI offers must be reachable from the TUI too (a field, an auto-derived
code from the stored secret, or a challenge prompt). When you touch login on one
surface, update the other three in the same PR — no "use the CLI instead" shortcuts.

**Entity scope (v1.0):** devices have full CRUD across the surfaces. Assets,
subscribers and subscriptions are provisioned via **provision functions** (`og provision`)
— direct CRUD for them is intentionally out of v1.0 (revisit post-v1 if demand appears).

### OpenGate API conventions

- All API paths use the prefix `/north/v80/` (including operations, despite the YAML spec showing `/v80/`)
- Provision endpoints: `/north/v80/provision/organizations/{org}/...`
- Search endpoints: `/north/v80/search/...`
- Auth: `POST /north/v80/provision/users/login` with `{"email":"...","password":"..."}` → JWT in `response.user.jwt`
- Subsequent requests use `Authorization: Bearer <token>`
- The credential field is `email` (not `user`), validated with `net/mail.ParseAddress`
- API documentation is in `ogdoc/` directory (OpenAPI YAML specs)

### Config

- File: `~/.og/config.yaml` with profile support (`--profile` flag)
- Env vars: prefix `OG_` overrides config (`OG_HOST`, `OG_PROFILE`, `OG_TOKEN`, `OG_ORG`)
- `.env` file in cwd loaded automatically
- `--org` global flag for organization (used by most provisioning commands)

### Output

All data commands support `--output json|table` (default: `table`). Use the `internal/output` package.

### OpenGate API quirks

- HTTP 204 (No Content) is returned when a search has no results — handle with `client.IsEmptyResponse()` before unmarshaling
- Device endpoints require `?flattened=true` query parameter

## Conventions

- Code, comments, variable/function names: **English**
- Commit messages: **English**, conventional commits (`feat:`, `fix:`, `chore:`, etc.)
- Go idioms: effective Go, short functions, minimal interfaces
- No premature abstraction — add complexity only when a second endpoint needs it
- **Always update README.md** when adding new functionality (commands, MCP tools, TUI views)
- **Always update MCP prompts** (`internal/mcp/prompts.go`) when adding new tools so LLMs know how to use them
- **Always update the skills** (`.claude/skills/og-cli`, `og-workspaces`, `og-device-ops`) when commands, flags, or workflows change — they are the operational knowledge for AI agents using og
