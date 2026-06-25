# MCP setup — exposing og to LLMs

`og mcp` starts a Model Context Protocol server so any MCP client (Claude Code,
Claude Desktop, LM Studio, ...) can drive OpenGate through tools.

## Start & prerequisites

```bash
og login -e user@example.com     # REQUIRED first — server reads tokens from ~/.og/config.yaml
og mcp                           # stdio transport (default → lean mode)
og mcp --http :8080              # HTTP transport for remote/debug setups
og mcp --http :8080 --multi-tenant   # HTTP, per-request credentials from headers
```

## Surface modes — control the context-token cost

Tool definitions load into the model's context every request. Exposing all ~90
named tools costs ~14k tokens/turn whether used or not. Pick the surface:

| Mode | Flag | Tools | ~tokens | When |
|---|---|---|---|---|
| **lean** | `--lean` | 2 (`og_exec`, `og_help`) | ~440 | shell-capable agents — the model runs the whole CLI through `og_exec` and discovers it with `og_help`. **Default for stdio.** |
| **observe** | `--toolsets observe` | ~18 | ~3.7k | curated read core (devices, alarms, datamodels, timeseries, datasets, jobs). **Default for HTTP.** |
| **readonly** | `--toolsets readonly` | ~42 | ~7k | every non-mutating tool |
| **groups** | `--toolsets devices,alarms-ops` | varies | — | named groups: read = `<resource>`, mutations = `<resource>-write` / `<resource>-ops` |
| **all** | `--all-tools` | ~90 | ~14k | the full named surface |

Defaults when no surface flag is given: **stdio → lean**, **HTTP → observe**. The
`og_toolsets` tool lists active/available groups at runtime so a client can ask for
more.

### Which mode should I pick?

Decision order — stop at the first match:

1. **Agent has a shell / runs locally (Claude Code, Cursor, scripts)** → `--lean`.
   Cheapest, and the model gets the entire CLI. This is the stdio default; rarely
   override it.
2. **Remote / less-trusted caller (hosted chatbot, multi-tenant HTTP, third party)**
   → `--toolsets observe` (the HTTP default). Read-only and governable; widen only if
   the use case needs it. Do **not** use `--lean` here unless the backing credentials
   are scoped (it exposes arbitrary subcommands).
3. **Caller needs to mutate a specific area** → add the write/ops groups it needs, e.g.
   `--toolsets observe,devices-write,alarms-ops`. Grant the minimum.
4. **A client that genuinely wants every named tool** (and can afford ~14k tokens) →
   `--all-tools`.

Rule of thumb: prefer `--lean` whenever the caller has a shell; prefer the narrowest
`--toolsets` otherwise. `--all-tools` is the last resort, not the default.

### Driving the CLI in lean mode (`og_exec` / `og_help`)

In lean mode there are exactly two tools. Workflow:

1. **Discover** with `og_help`: `og_help("")` (top level) → `og_help("device")` →
   `og_help("device search")`. Read flags before guessing.
2. **Run** with `og_exec`, passing the command WITHOUT the leading `og`, quoting args
   with spaces:

```
og_exec("device search -w \"provision.device.administrativeState eq ACTIVE\" --view summary --output json")
og_exec("device get sensehat sense-001 --output json")
og_exec("alarm summary")
og_exec("datamodel get sensehat weather --output json")
```

- Same query syntax, views and field projection as the CLI apply verbatim (see
  query-cookbook.md and views-guide.md). Always project (`--view`/`--select`) and use
  `--output json` to keep results small.
- **Destructive** subcommands (`delete`, `cancel`) refuse to run without a terminal
  unless you add `--yes`. Only add it after the user has explicitly consented:
  `og_exec("device delete sensehat sense-001 --yes")`.
- The child process inherits the server's identity (host + JWT injected as env), so
  no login step is needed inside `og_exec`.

### Toolset reference

Read groups (non-mutating) — also reachable via the `readonly` alias:

| Toolset | Tools |
|---|---|
| `devices` | devices_search, devices_get |
| `datamodels` | datamodels_search, datamodels_get |
| `alarms` | alarms_search, alarms_summary |
| `timeseries` | timeseries_list, timeseries_get, timeseries_data, timeseries_export |
| `datasets` | datasets_list, datasets_get, datasets_data |
| `jobs` | jobs_search, jobs_get, jobs_operations |
| `rules` | rules_search, rules_get, rules_catalog, rules_logs |
| `connectors` | connectors_list, connectors_get, connectors_catalog, connectors_logs |
| `provision` | provision_list, provision_get, provision_bulk_status, provision_bulk_details |
| `optypes` | optypes_catalog, optypes_search, optypes_get |
| `workspaces` | workspaces_list, workspaces_get, workspaces_export |
| `dashboards` | dashboards_list, dashboards_get, dashboards_export |
| `tasks` | tasks_search, tasks_get |

Write / ops groups (mutations & high-impact actions):

| Toolset | Tools |
|---|---|
| `devices-write` | devices_create, devices_update, devices_delete |
| `datamodels-write` | datamodels_create, datamodels_update, datamodels_delete |
| `alarms-ops` | alarms_attend, alarms_close |
| `timeseries-write` | timeseries_create, timeseries_update, timeseries_delete |
| `datasets-write` | datasets_create, datasets_update, datasets_delete |
| `jobs-run` | jobs_create, jobs_cancel |
| `rules-write` | rules_create, rules_update, rules_delete |
| `rules-ops` | rules_set_active |
| `connectors-write` | connectors_create, connectors_update, connectors_delete |
| `connectors-ops` | connectors_set_status |
| `provision-write` | provision_create, provision_update, provision_delete |
| `provision-bulk` | provision_plan, provision_bulk |
| `optypes-write` | optypes_create, optypes_update, optypes_delete |
| `workspaces-write` | workspaces_import, workspaces_update, workspaces_delete |
| `workspaces-share` | workspaces_share |
| `dashboards-write` | dashboards_import, dashboards_update, dashboards_delete |
| `dashboards-share` | dashboards_share |
| `tasks-write` | tasks_create, tasks_cancel |
| `iot` | iot_collect, iot_collect_payload, iot_collect_raw, iot_mqtt_publish, iot_mqtt_subscribe, iot_mqtt_device |

Aliases: `observe` = devices + alarms + datamodels + timeseries + datasets + jobs (read
core); `readonly` = all read groups; `all` = everything. `login` is present in
single-tenant non-lean modes; `og_toolsets` is always present in named modes.

> **Migration (since v1.7.0):** the stdio default changed from "all tools" to `--lean`,
> and HTTP now defaults to `--toolsets observe` instead of all tools. To restore the old
> behaviour explicitly, start the server with `--all-tools`. `og mcp install` now bakes
> `--lean` into the generated config.

## Multi-tenant HTTP mode (one server, many users)

Two distinct ways to serve multiple identities:

- **Per-profile (stdio):** one og process per profile (`--profile production`), each
  with its own startup credentials. Good for a few fixed tenants on one machine.
- **`--multi-tenant` (HTTP):** ONE stateless server; credentials arrive **per request
  in HTTP headers**, never in tool args (which would pass through the LLM) and never
  from the startup profile. For multi-user chatbots. Required headers per request:
  - `Authorization: Bearer <north JWT>` — every tool.
  - `X-OG-Web-Token: <web JWT>` — workspace/dashboard tools.
  - `X-OG-Api-Key: <api key>` — South/IoT tools.
  A tool errors if a credential it needs is absent (no cross-user fallback); the
  `login` tool is dropped (it would leak a JWT to the LLM). Put TLS in front.

## Client configuration

**Easiest — let og write the config:**

```bash
og mcp install                       # Claude Desktop (default): per-OS config path
og mcp install --client claude-code  # Claude Code: project .mcp.json in cwd
og mcp install --profile production   # bake --profile into the server args
og mcp install --print                # dry-run: show what would be written
og mcp uninstall                      # remove it again
```

`install` writes the binary's ABSOLUTE path (Claude Desktop has no shell PATH, so
a bare `og` would fail — the nº1 cause of "it doesn't work"), merges
non-destructively (other servers preserved, original backed up to `<file>.bak`),
and refuses to clobber a config that is not valid JSON. Tell the user to restart
Claude Desktop afterwards. Desktop paths: macOS `~/Library/Application Support/Claude/`,
Windows `%APPDATA%\Claude\`, Linux (unofficial build) `~/.config/Claude/`.

**Manual** — Claude Code / Claude Desktop (`.mcp.json` in the project or `~/.claude/settings.json`):

```json
{
  "mcpServers": {
    "opengate": { "command": "og", "args": ["mcp", "--stdio", "--lean"] }
  }
}
```

(`og mcp install` bakes in `--lean`. Swap it for `--toolsets observe` or `--all-tools`
to expose named tools instead of the exec surface.)

Multi-tenant — one server per profile:

```json
{
  "mcpServers": {
    "opengate-production": { "command": "og", "args": ["mcp", "--stdio", "--lean", "--profile", "production"] },
    "opengate-staging":    { "command": "og", "args": ["mcp", "--stdio", "--lean", "--profile", "staging"] }
  }
}
```

Any other MCP client: same pattern — spawn the `og` binary with `mcp --stdio`
(JSON-RPC over stdin/stdout).

## The three primitives and how they cooperate

| Primitive | What | Why it matters |
|---|---|---|
| **Prompt** `opengate-guide` | Teaches query syntax, ES/EN operator mapping, entity→tool mapping, job JSON format, worked examples | Load it at conversation start; without it LLMs hand-build raw JSON filters and fail |
| **Resources** | `opengate://query-syntax`, `opengate://views`, `opengate://organizations/{org}/datamodel-fields` | On-demand discovery: views dictionary (incl. user's custom views) and per-org dynamic datastream names |
| **Tools** | named mode: `devices_search`, `alarms_*`, `jobs_*`, `iot_collect`, `workspaces_*`, ... (which ones depends on the active toolsets). lean mode: just `og_exec` / `og_help` | The actual operations; search tools take `query` (preferred), `filter` (raw JSON), `limit` |
| **`og_toolsets`** | discovery tool (named modes) | Lists active/available toolset groups so the client can request more |

Key tool parameters for efficient usage:

- `devices_search(query, view, select, limit, filter)` — ALWAYS pass `view`
  and/or `select`; otherwise each result is a full device document (thousands of
  tokens). `select` supports `@at` suffixes: `"wt@at,wp@at"`.
- Read `opengate://organizations/{org}/datamodel-fields` before guessing custom
  datastream names; read `opengate://views` to see what views expand to.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| Every tool returns auth errors | No/expired session: run `og login` again; check `OG_PROFILE` matches the configured one |
| Workspace/dashboard tools fail, device tools work | Logged in with `--no-web`, or web session stolen by a browser login (og retries once automatically; persistent failure → re-login) |
| Search returns nothing but data exists | Field name wrong (check datamodel-fields resource) or filtering timeseries/datasets by device paths instead of column names |
| Huge token usage per search | Missing `view`/`select` projection |
