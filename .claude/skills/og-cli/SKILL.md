---
name: "og-cli"
description: "Drive the og CLI for the OpenGate IoT platform: authentication, profiles and multi-tenant work, search query syntax (-w), field projection with views/@at timestamps, output formats, and OpenGate API quirks. Use whenever running og commands or helping a user query, inspect, or manage OpenGate resources (devices, datamodels, alarms, timeseries, datasets)."
---

# og — OpenGate CLI driving skill

`og` talks to the OpenGate IoT platform. Three interfaces, same functionality:
CLI (`og <cmd>`), interactive TUI (`og` with no args), MCP server (`og mcp`).
This skill teaches how to DRIVE the tool. For its full command surface, read
[README.md](../../../README.md). Related skills: **og-workspaces** (dashboards,
widgets, pull/deploy lifecycle) and **og-device-ops** (jobs, alarms, IoT data).

## Intent → command map

| User intent | Command | Notes |
|---|---|---|
| "what devices do I have / how are they" | `og dev search --view summary` | never fetch full documents for listings |
| "battery / power state" | `og dev search --view power` | includes charge_at timestamps |
| "where are the devices" | `og dev search --view location,status` | |
| "devices with X > N" | `og dev search -w "X gt N" --view summary` | filters work on collected values too |
| "show device X in full" | `og dev get <id> -o json` | the only case for full documents |
| "what fields/datastreams exist" | `og dm get <datamodel> --org <org>` or `og views list` | |
| "open/critical alarms" | `og alarms search -w "alarm.status eq OPEN"` | see og-device-ops |
| "reboot / run operation / send data" | jobs / iot commands | see og-device-ops |
| "automation rule / alert me when X" | `og rules ...` (EASY/ADVANCED + local JS) | see og-device-ops |
| "define a new operation" | `og optypes create` | see og-device-ops |
| "edit a dashboard / migrate workspace" | `og workspace|dashboard pull/deploy` | see og-workspaces |

## Authentication & profiles

```bash
og login -e user@example.com            # password prompted; stores North JWT + Web JWT + API key
og login -e user@example.com --profile staging
og login -e user@example.com --no-web   # skip Web API signin (no workspace/dashboard cmds)
```

- Config: `~/.og/config.yaml`, profile-based. Select with `--profile` or `OG_PROFILE`.
- Env overrides: `OG_HOST`, `OG_TOKEN`, `OG_ORG`, `OG_EMAIL`, `OG_PASSWORD`. A `.env` in cwd auto-loads.
- Most provision commands need an organization: `--org <name>` or profile default.
- **Two token surfaces**: North API (devices, alarms, ...) and Web API (workspaces,
  dashboards). Both obtained by one `og login`.
- **Single web session quirk**: OpenGate allows ONE active web session per user.
  Logging into the web UI invalidates the CLI's web token — og auto re-signs-in
  on 401 and retries, so don't treat a one-off 401 in logs as fatal.
- Multi-tenant work = one profile per tenant; every command accepts `--profile`.

## Searching

```bash
og dev search -w "field op value" -w "field2 op value2"   # multiple -w = AND
```

- Operators: `eq neq like gt lt gte lte in exists`.
- Devices filter on BOTH provision metadata and the LATEST collected value of any
  datastream: `-w "wt gt 20"` works directly — do NOT reach for timeseries unless
  the user asks for historical/windowed data.
- OR / nested logic: only via raw JSON `--filter '{"filter":{"or":[...]}}'`.
- Limit results: `--limit N`.

Ready-made recipes: [query-cookbook.md](query-cookbook.md).

## Projecting fields — the golden rule

**Never return full device documents for listings** — they are thousands of tokens
each. Always project:

```bash
og dev search --view summary                  # named view (intent-based)
og dev search -s provision.device.identifier -s wt@at   # explicit fields
og dev search --view power -s wt              # combinable; explicit -s wins
```

- `@at` suffix (or `--at` for all fields) adds the **value timestamp** — vital for
  collected data: a battery at 87% reported 3 weeks ago is a dead data point.
  Always include `@at` when the question involves current sensor values.
- Discover views: `og views list`, `og views show <name>`. Users can add custom
  views in `~/.og/views/*.yaml` and `./.og/views/*.yaml`.
- Full guide: [views-guide.md](views-guide.md).

## Output

- `-o table` (default) for humans, `-o json` for parsing/piping.
- Do not confuse `-o` (output FORMAT) with `--out` (output FILE in export commands).
- Table column names come from field aliases (last path segment, or `<alias>_at`).

## OpenGate API quirks worth knowing

| Quirk | Implication |
|---|---|
| Empty search result = HTTP 204 | og prints nothing / empty table; not an error |
| Device docs are flattened (`?flattened=true`) | fields are root-level dotted keys: `wt`, `provision.device.name` |
| Each field wraps value as `_value._current.{value,at,date,source}` | `at` = when measured; og's `@at` selects it |
| Datastream names are dynamic (defined per-org in datamodels) | discover with `og dm get` or the MCP resource `datamodel-fields` |
| Timeseries/datasets filter by COLUMN names, not device paths | run `og ts get <id>` / `og ds get <id>` first to learn columns |

## MCP

To expose og to LLMs: `og mcp --stdio` (or `--http :8080`). Setup, primitives and
multi-profile config: [mcp-setup.md](mcp-setup.md).
