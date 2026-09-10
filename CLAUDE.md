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
pkg/opengate/        → OpenGate REST API client (HTTP methods, auth, resource methods)
pkg/query/           → Search filter parser (-w "field op value", query strings)
internal/config/     → Viper config, profiles, .env loading
internal/mcp/        → MCP server (stdio + HTTP transports) + tool definitions
internal/output/     → JSON/table output formatting
internal/tui/        → Bubble Tea interactive TUI
internal/unwrap/     → Workspace/rule/connector unwrap to local dirs
internal/views/      → Named field views for searches
```

`pkg/` is the public library surface: it is consumed by external services (e.g.
Punto de Luz), so its API is a contract. `internal/` is CLI-private.

**Library conventions for `pkg/`:**

- Every method that performs I/O takes `ctx context.Context` as its **first**
  parameter, propagated to `http.NewRequestWithContext`. No exceptions.
- Per-client configuration goes through functional options on `New`
  (`WithHTTPClient`, `WithTLS`, `WithAPIVersion`, `WithAPIKey`) — never
  process-wide state. `ConfigureTLS`/`NewHTTPClient` are deprecated leftovers
  kept for the CLI, which has exactly one endpoint per invocation.
- North API path constants carry the `{v}` version placeholder, resolved per
  client. Never hardcode a version segment.
- A misconfigured client is not a panic and not a second constructor: `New`
  records the error, and `Err()` plus every request return it.
- **A typed getter is lossy by construction.** Most families return
  `json.RawMessage` and pass the platform's bytes through; `datamodels`,
  `datasets`, `timeseries` and `workspaces` decode into structs, so any field
  the struct does not name is dropped without a word. Where a struct exists,
  ship a `…Raw` sibling (`GetDatamodelRaw`, `GetDatasetRaw`, `GetTimeSeriesRaw`,
  `ListTimeSeriesRaw`, `GetWorkspaceRaw`) so callers who need fidelity are not
  hostage to this package being current — and surface it as `--raw` on the CLI
  `get`. Note `og workspace pull` reads through the struct too, so workspaces
  are the one family whose pull is not a passthrough.

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
0. If it is a new **artifact family** with a pull/edit/deploy cycle (a single JSON
   document with code in known fields), declare it as an `unwrap.Descriptor` —
   metadata filename, name keys, id key, code contract — rather than copying an
   existing family's adapter. The lifecycle is written once against that struct.
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

**The dashboard family's edit verbs (2026-08-29).** `og dashboard show --path` and
`og dashboard diff` exist so a widget is editable from an editor the way a rule
already was. Three decisions came with them, and each is load-bearing:

- **The widget is not the unit; the dashboard is.** A widget is a grid item, not
  something the platform can address on its own, so there is no `og widget show`,
  `diff` or `deploy`. Same boundary `og workspace watch` already draws when a widget
  edit deploys its dashboard.
- **Paths are matched by widget identity, not by grid position.** The `NN__` prefix in
  a widget directory is the remote grid order at the moment of the pull; a reorder on
  the platform would otherwise make every path in a local tree address nothing. Where
  identity is ambiguous (same type, neither widget carrying an id) the path is reported
  as not found rather than guessed — see `unwrap.ResolveCodePath`.
- **A missing subcommand is not a loud failure.** cobra answers a subcommand a family
  does not have by printing that family's help and exiting **0**, so `og dashboard diff`
  used to look like a successful comparison. Both editor plugins were confirming a
  deploy against a help page. When a family gains a verb the others have, check the
  exit code, not just the output.

The TUI was deliberately left alone: no family shows artifact code in the TUI, so a
widget code viewer would put dashboards ahead of rules and connector functions for no
reason. When a code viewer lands it should land for all four families at once.

**Deliberate MCP exclusions.** Two operations are CLI-only on purpose, beyond the
local-filesystem lifecycle verbs:

- `og jobs launch` (batched launch over a large fleet). Firing an irreversible
  operation at thousands of devices is a decision to take explicitly, not one to
  hand an LLM a single tool call for. `jobs_create` already covers one batch
  (≤100 entities) and an agent can call it repeatedly under supervision.
- `--all` on searches. An unbounded result set is actively harmful in an LLM
  context window; MCP exposes `page` instead, plus the response's `page` block so
  the caller knows more pages exist.

**Entity scope (v1.0):** devices have full CRUD across the surfaces. Assets,
subscribers and subscriptions are provisioned via **provision functions** (`og provision`)
— direct CRUD for them is intentionally out of v1.0 (revisit post-v1 if demand appears).

### OpenGate API conventions

- All API paths use the prefix `/north/<version>/` (including operations, despite the YAML spec showing `/v80/`). In code the version is the `{v}` placeholder, resolved per client — `v80` by default
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
- **Filter names are not response names.** On several endpoints the field you filter
  on differs from the path you read in the output, and using the response path
  returns HTTP 400 *"Field in filter unknown"*. Verified live:
  - `search/jobs`: filter on `jobStatus` (alias `job.status`), `operationName`,
    `jobId`, `taskId` — NOT `jobs.report.summary.status` / `jobs.request.name`.
  - `search/entities/operations/history`: identifiers unprefixed (`jobId`,
    `entityId`, `operationId`, `resourceType`), the rest `operation`-prefixed
    (`operationName`, `operationStatus`, `operationResult`, `operationDate`,
    `operationNotify`) — NOT the bare `status`/`result`, nor any `operations.` prefix.
  When adding a filterable field, probe it against a live instance AND check it
  really filters (a bogus value must return nothing): an accepted-but-ignored field
  is worse than a 400.
- **The Web API's delete keys are the platform's, typos included.** Deleting a
  dashboard is `DELETE /api/dashboards/` (trailing slash required) with
  `{"dasboardsDelete":["<id>"]}` — note the missing "h". Every reasonable guess
  (`{"_id":…}`, `{"ids":[…]}`, the document itself, even `{}`) returns 400 with
  an empty message, and so does a nonexistent id, so the error tells you
  nothing. Found by reading the web client's bundle: when a Web API call fails
  this way, fetch the front-end JS and search it rather than guessing.
- **The Web API's PUT merges, it does not replace.** A dashboard field this
  package does not model survives a `pull` → `deploy` cycle rather than being
  erased by the write (verified live 2026-09-09 with `extraConfig.showBanner`).
  Two consequences: an unmodelled field is a fidelity bug, not a destructive
  one, and `backgroundColor`/`backgroundImageSize` cannot be set through this
  API at all — the platform keeps returning null whatever you PUT.
- **An endpoint can answer "successfully" with a fraction of the document.**
  `/api/workspaces/export/{id}` returns the workspace shell — dashboards: 0, no
  views, no bundles — unless asked with
  `?dashboard=1&template=1&wiwi=1&view=1` (2.8 KB vs 15 KB, verified live
  2026-09-08). The `timeseries` list behaves the same way through `expand`:
  `expand=columns,context` silently omits every sort. Neither returns an error,
  so a command can look like it works and quietly produce a useless backup.
  When wiring a read that is meant to be complete, compare it against the same
  endpoint with every option turned on.
- **A query parameter's accepted values are per-build, and a rejected one
  fails the whole request.** The `timeseries` list validates `expand` against a
  whitelist: `api.opengate.es` accepts `sorts`, an on-premises instance on the
  same `v80` does not and answers HTTP 400 "Invalid query parameters" for the
  entire call — not a response without sorts (reported live from an MRG staging
  tenant, 2026-09-10; regression shipped in v2.6.0, fixed by degrading in
  `listTimeSeries`). Two lessons: the API version segment does **not** tell you
  what an instance supports, so feature-detect on the rejection rather than on
  `--api-version`; and asking for an optional expansion unconditionally trades
  "returns less" for "returns nothing" on any instance that has not caught up.
  When a read must be complete AND must work everywhere, ask for everything and
  degrade on the 400 that names the parameter.
- **An error's `context` carries the offending value, not just the field
  name.** `{"context":[{"value":"sorts","name":"expand"}]}` is the difference
  between "(fields: expand)", which sends the reader hunting, and
  "(fields: expand=sorts)", which names the culprit. `APIError.Context` keeps
  both; `APIError.Fields` is the older names-only view, kept because `pkg/` is a
  published contract. A diagnosis built on the names-only message is how a
  rejected `expand` value got misread as og targeting the wrong API version.
- **The OpenAPI spec under `ogdoc/` is incomplete, so it is not a coverage
  criterion.** Verified live 2026-09-07: every datastream comes back with
  `indexed`, some with `notFilterable`, and every dataset with a `sorts` array
  of named orderings — none of the three appears in the YAML. All three were
  missing from their structs and were being dropped on every read. When adding
  or reviewing a typed struct, diff it against a live response
  (`curl … | jq -S .` vs `og … -o json | jq -S .`), not against the spec.
- **"Not found" has no single shape, so no caller can key on the status
  code.** Handing a family's `get` a name instead of an identifier was probed
  across seven families live (2026-09-10): `timeseries` answers HTTP 404 with
  `fields: identifier`, connector and provision functions 404 with
  `connectorFunctionId` / `provisionProcessorId`, `datasets` HTTP **400**
  "Element not found.", `rules` HTTP **400** "No rule has been found with this
  id" — and `workspaces`, `dashboards`, `datamodels` and `devices` answer
  **HTTP 204 with an empty body**, which is not an error status at all.
  Anything that has to recognise "no such artifact" must look at the context's
  id field, the message text, AND the empty body.

  That last shape was reporting a missing artifact as anything but: a typed
  getter unmarshalled the empty body and failed with `unexpected end of JSON
  input`, and a `…Raw` one returned zero bytes, so `og dev get <missing>`
  printed an empty table and **exited 0**. Every single-artifact read now goes
  through `notFoundIfEmpty` and returns a `*NotFoundError` (`IsNotFound`).
  Lists and catalogs must NOT use it: there an empty body means "none yet",
  which is an answer. `og jobs get <missing>` is still open — it answers 200
  with `{}`, so no transport-level check can see it.
- **A hint about an identifier belongs at the one place errors pass through.**
  43 subcommands across 7 families take a generated identifier (UUID, 24-char
  hex, or in workspaces no fixed shape at all), and half of them mutate. So the
  fix for "the 404 does not say how to get an identifier" is `explainNotFound`
  in `cmd/hints.go`, hooked into `Execute` via cobra's `ExecuteContextC` —
  which hands back the command that ran, so the hint can name that family's own
  `list`. Resolving a name to an identifier automatically was considered and
  rejected: it would touch those 43 entry points, and doing it for a `delete`
  or `update` would act on a guess. Note the shape check in `looksLikeIdentifier`
  only picks the wording — never behaviour — precisely because workspace ids
  like `shared` make shape undecidable.
- **Closed work leaves the "current" views.** A FINISHED job was observed absent from
  `search/jobs` and its `operation/jobs/{id}/operations` returned HTTP 204, while
  `search/entities/operations/history` returned the operation with its steps. Never
  read an empty per-job listing as "it had no operations".

## Conventions

- Code, comments, variable/function names: **English**
- Commit messages: **English**, conventional commits (`feat:`, `fix:`, `chore:`, etc.)
- Go idioms: effective Go, short functions, minimal interfaces
- No premature abstraction — add complexity only when a second endpoint needs it
- **Always update README.md** when adding new functionality (commands, MCP tools, TUI views)
- **Always update MCP prompts** (`internal/mcp/prompts.go`) when adding new tools so LLMs know how to use them
- **Always update the skills** (`.claude/skills/og-cli`, `og-workspaces`, `og-device-ops`) when commands, flags, or workflows change — they are the operational knowledge for AI agents using og
