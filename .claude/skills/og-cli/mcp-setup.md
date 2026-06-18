# MCP setup — exposing og to LLMs

`og mcp` starts a Model Context Protocol server so any MCP client (Claude Code,
Claude Desktop, LM Studio, ...) can drive OpenGate through tools.

## Start & prerequisites

```bash
og login -e user@example.com     # REQUIRED first — server reads tokens from ~/.og/config.yaml
og mcp                           # stdio transport (default, what clients spawn)
og mcp --http :8080              # HTTP transport for remote/debug setups
og mcp --http :8080 --multi-tenant   # HTTP, per-request credentials from headers
```

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
    "opengate": { "command": "og", "args": ["mcp", "--stdio"] }
  }
}
```

Multi-tenant — one server per profile:

```json
{
  "mcpServers": {
    "opengate-production": { "command": "og", "args": ["mcp", "--stdio", "--profile", "production"] },
    "opengate-staging":    { "command": "og", "args": ["mcp", "--stdio", "--profile", "staging"] }
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
| **Tools** (50+) | `devices_search`, `alarms_*`, `jobs_*`, `iot_collect`, `workspaces_*`, `dashboards_*`, ... | The actual operations; search tools take `query` (preferred), `filter` (raw JSON), `limit` |

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
