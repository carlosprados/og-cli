# MCP setup — exposing og to LLMs

`og mcp` starts a Model Context Protocol server so any MCP client (Claude Code,
Claude Desktop, LM Studio, ...) can drive OpenGate through tools.

## Start & prerequisites

```bash
og login -e user@example.com     # REQUIRED first — server reads tokens from ~/.og/config.yaml
og mcp                           # stdio transport (default, what clients spawn)
og mcp --http :8080              # HTTP transport for remote/debug setups
```

## Client configuration

Claude Code / Claude Desktop (`.mcp.json` in the project or `~/.claude/settings.json`):

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
