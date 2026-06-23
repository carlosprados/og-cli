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

## Getting the skills onto a new machine

These skills ship embedded in the `og` binary, so a user who installed `og` but
did not clone the repo can still get them: `og skills extract` writes them to
`./.claude/skills/`, `og skills extract --global` to `~/.claude/skills/`.
It aborts on collision; `--force` (optionally `--backup`) overwrites. Editing a
skill requires rebuilding `og` for the embedded copy to pick up the change.

## Intent → command map

| User intent | Command | Notes |
|---|---|---|
| "what devices do I have / how are they" | `og dev search --view summary` | never fetch full documents for listings |
| "battery / power state" | `og dev search --view power` | includes charge_at timestamps |
| "where are the devices" | `og dev search --view location,status` | |
| "devices with X > N" | `og dev search -w "X gt N" --view summary` | filters work on collected values too |
| "devices that collected after T / reported recently" | `og dev search --filter '{...gte device.identifier._current.at...}'` | filter `at` (reception) of the always-collected identifier; see Searching → at vs date |
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
og login -e user@example.com --2fa-code 123456   # TOTP 2FA-enabled accounts
og login -e user@example.com --2fa-secret JBSWY3DPEHPK3PXP   # store seed; og generates codes itself
```

- **2FA (TOTP):** if the account has 2FA enabled, login prompts for the 6-digit code
  and retries. For non-interactive runs pass `--2fa-code` / `OG_2FA_CODE` (codes expire
  in 30s — generate fresh per run). Enabling/resetting 2FA is done in the web UI, not og.
- **Hands-off 2FA:** `--2fa-secret <base32>` stores the TOTP seed in the profile
  (config file forced to `0600`) and og derives the code on every login — no prompt, no
  per-run code. `OG_2FA_SECRET` does the same per-run **without** persisting. ⚠️ A stored
  seed is a permanent second factor (anyone reading the config can mint codes) — trusted
  hosts / CI only.
- Config: `~/.og/config.yaml`, profile-based. Select with `--profile` or `OG_PROFILE`.
- Env overrides: `OG_HOST`, `OG_TOKEN`, `OG_ORG`, `OG_EMAIL`, `OG_PASSWORD`, `OG_2FA_CODE`, `OG_2FA_SECRET`, `OG_INSECURE`, `OG_CA_FILE`. A `.env` in cwd auto-loads.
- **Self-signed / private-CA servers**: global `--insecure` skips TLS verification
  (no CA needed) and `--ca-file <pem>` trusts an extra CA — both cover HTTP (North/South)
  **and** MQTT. Resolution is flag → profile → env; `og login --insecure` persists it
  into the profile so later commands inherit it. og warns on stderr when verification is off.
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

**Value typing — the search lake is type-strict.** A field stored as a string only
matches a JSON *string*; sending a number returns 0 silently (HTTP 204). The field's
type lives in the datamodel (`og dm get <model>` → `schema.type` / `$ref`), NOT in the
value's textual shape. `provision.device.identifier` is `ogIdentifier` (**string**) — so
an IP-/MAC-looking id is a string, not a number.

- ⚠️ **`-w` mis-casts numeric-looking strings.** A value with a numeric prefix
  (`192.168.0.1`, `1.2.3`, an ISO timestamp) is silently coerced to a number, so
  `-w "provision.device.identifier eq 192.168.0.1-..."` returns nothing. Until this is
  fixed, use `like` for partial string match, or pass the exact JSON string via raw
  `--filter '{"filter":{"eq":{"provision.device.identifier":"192.168.0.1-..."}}}'`.
- **Knowing a field's type**: for *declared* streams read the datamodel; for *ad-hoc*
  streams (collected, not in any datamodel) the stored type = whatever was collected —
  sample one `_current.value` to learn it. Cache the org's field→type map for the session.

### Filtering by collection time — `at` vs `date`

Every collected field carries two timestamps under `_value._current`:
**`at`** = when the platform *received* it, **`date`** = when the measurement was *taken*.
Both are filterable via the path `<datastream>._current.at` / `._current.date`, value =
**ISO-8601 with offset** (e.g. `2026-03-12T11:33:13.441+01:00`). The path WITHOUT
`._current` is invalid. Use raw `--filter` (the `-w` parser mis-casts ISO strings, see above):

```bash
# Devices that COLLECTED ANYTHING after a moment → filter the always-collected identifier's at
og dev search --filter '{"filter":{"gte":{"device.identifier._current.at":"2026-06-21T18:00:00.000+02:00"}}}'
```

- "Did the device report recently?" → `at` on `device.identifier` (always collected).
- "Was *this measurement* taken after T?" → first ask the user **which datastream**,
  then filter `<stream>._current.date` (measurement) and/or `._current.at` (reception).

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
| Each field wraps value as `_value._current.{value,at,date,source}` | **`at` = reception (platform ingest), `date` = measurement time**; og's `@at` selects `at`. `date` is in the data but og has no `@date` projection yet — read it via `og dev get` (full doc) |
| Search lake is **type-strict** | a string field only matches a JSON string; field types come from the datamodel (`schema.type`/`$ref`), not the value's shape |
| `at`/`date` are filterable | path `<datastream>._current.{at,date}`, ISO-8601 value, via raw `--filter` (not `-w`) |
| Datastream names are dynamic (defined per-org in datamodels) | discover with `og dm get` or the MCP resource `datamodel-fields` |
| Timeseries/datasets filter by COLUMN names, not device paths | run `og ts get <id>` / `og ds get <id>` first to learn columns |

## MCP

To expose og to LLMs: `og mcp --stdio` (or `--http :8080`). Setup, primitives and
multi-profile config: [mcp-setup.md](mcp-setup.md).
