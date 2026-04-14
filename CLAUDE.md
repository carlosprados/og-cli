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

### Core invariant: CLI ↔ TUI ↔ MCP parity

**Every OpenGate API operation must be exposed through all three interfaces: CLI command, TUI view, and MCP tool.** All three call the same method in `internal/client/`. This is a hard invariant — never ship functionality in one interface without the other two.

```
cmd/<command>.go  ──→  internal/client/<method>  ←──  internal/mcp/tools.go
                              ↑
                    internal/tui/<view>.go
```

When adding a new endpoint:
1. Add the client method in `internal/client/`
2. Add the Cobra command in `cmd/`
3. Add the MCP tool in `internal/mcp/`
4. Add the TUI view in `internal/tui/`
5. All four must be in the same PR — never ship one without the others

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
