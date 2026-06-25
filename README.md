# og — OpenGate CLI

Unofficial command-line interface for the [OpenGate](https://opengate.es) IoT platform REST API. See the [official OpenGate documentation](https://documentation.opengate.es) for API reference.

Three modes of operation:

| Mode | Invocation | Description |
|------|------------|-------------|
| **Interactive** | `og` | TUI with Bubble Tea — browse, search, and manage resources visually |
| **CLI** | `og <command>` | Direct commands for scripts and one-liners |
| **MCP** | `og mcp` | Model Context Protocol server for LLM integration |

All three interfaces expose the same functionality through the same client library —
including [named views](#views--project-fields-by-intent) (`--view summary,power`), which
project common field sets with their value timestamps without memorizing datastream paths.

## Build

Requires Go 1.21+ and [Task](https://taskfile.dev/).

```bash
task build      # build ./og binary
task install    # install to $GOPATH/bin
task test       # run tests
task lint       # golangci-lint
task fmt        # gofmt + goimports
task tidy       # go mod tidy
task clean      # remove build artifacts
```

Or install directly:

```bash
go install github.com/carlosprados/og-cli@latest
```

## Configuration

Config file: `~/.og/config.yaml`

```yaml
default_profile: production

profiles:
  production:
    host: https://api.opengate.es
    organization: my-org
  staging:
    host: https://staging-api.opengate.es
    organization: my-org-staging
```

Environment variables (prefix `OG_`) override config values:

| Variable | Description |
|----------|-------------|
| `OG_HOST` | API host URL |
| `OG_PROFILE` | Active profile name |
| `OG_TOKEN` | JWT token |
| `OG_ORG` | Organization name |
| `OG_EMAIL` | Login email |
| `OG_PASSWORD` | Login password |
| `OG_2FA_CODE` | 6-digit TOTP code for 2FA-enabled accounts |
| `OG_2FA_SECRET` | base32 TOTP secret; og derives the code itself (not persisted) |
| `OG_INSECURE` | Skip TLS verification (`true`/`false`) |
| `OG_CA_FILE` | Path to an extra CA/chain PEM to trust |

A `.env` file in the current directory is also loaded automatically.

## Interactive mode

Launch with no arguments:

```bash
og
```

Navigate with `↑↓` or `j/k`, `enter` to select, `esc` to go back, `r` to refresh lists, `q` to quit.

Screens:

| Screen | Description | Keys |
|--------|-------------|------|
| **Menu** | Main menu with all sections | enter |
| **Login** | Email/password form, stores JWT + API key + organization | tab, enter |
| **Datamodels** | List → enter for detail with categories and datastreams | enter, r |
| **Devices** | List → enter for tabbed detail view; `v` picks a named view (dynamic columns) | enter, o, v, r |
| **Workspaces** | List → enter for detail; `s` shares the selected workspace with users | enter, s, r |
| **Device detail** | Three tabs: Overview (cards), Datastreams (table), JSON (scrollable) | 1/2/3, tab |
| **Alarms** | List with severity/status → attend or close | a, c, r |
| **Rules** | Automation rules list; `t` toggles active on/off | t, r |
| **Operation Types** | Operation definitions catalog (yours + predefined) | r |
| **Time Series** | List → enter to browse collected data | enter, r |
| **Datasets** | List → enter to browse data | enter, r |
| **Jobs** | List → enter for job detail with per-device operations | enter, r |

From the **Devices** screen, press `o` on a device to launch an operation (REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC),
or `v` to switch the table to a named view (summary, power, resources, ... — including your custom ones).

## Query syntax

All search commands support a common filter syntax via `-w` flags (CLI) or `query` parameter (MCP):

```bash
# Single condition
og dev search -w "provision.device.administrativeState eq ACTIVE"

# Multiple conditions (AND)
og dev search -w "provision.device.identifier like sense" -w "provision.device.administrativeState eq TESTING"

# With limit
og dev search -w "provision.device.identifier like sense" --limit 10
```

**Operators:** `eq`, `neq`, `like`, `gt`, `lt`, `gte`, `lte`, `in`, `exists`

Multiple `-w` flags are combined with AND. For OR or nested queries, use `--filter` with raw JSON.

## Views — project fields by intent

Views are named field sets that expand into select clauses, so you (or an LLM) can ask
for *intent* instead of memorizing datastream paths. They work in the CLI (`--view`),
the TUI (`v` key on the Devices screen), and MCP (`view` parameter of `devices_search`).

```bash
# "What devices do I have and how are they?" — instead of six -s flags
og dev search --view summary

# "How are the batteries doing?"
og dev search --view power

# Views combine, and mix with explicit selects (explicit -s wins on collision)
og dev search --view summary,power -s wt@at
```

Built-in views: `summary`, `identifier`, `name`, `type`, `location`, `organization`,
`topology`, `status`, `hardware`, `software`, `relations`, `temperature`, `power`,
`resources`. Discover them with:

```bash
og views list           # all views with their source and description
og views show power     # exact fields a view expands to
```

### Value timestamps (`at`)

Collected values are only meaningful with their timestamp — a battery at 87% reported
three weeks ago is a dead data point. Views mark key collected fields with `at`, which
adds a `<field>_at` column. The same works in explicit selects:

```bash
og dev search -s wt@at -s provision.device.identifier   # wt + wt_at columns
og dev search -s wt -s wp --at                          # at for every selected field
```

### Custom views

Add your own views as YAML files (any name, as many files as you need) in:

| Layer | Location | Wins over |
|-------|----------|-----------|
| Project | `./.og/views/*.yaml` | user and built-in |
| User | `~/.og/views/*.yaml` | built-in |
| Built-in | embedded in the binary | — |

Same view name in a higher layer replaces the lower one entirely. Two files in the
*same* directory defining the same view is an error.

```yaml
# ~/.og/views/sensehat.yaml
views:
  water:
    description: Water sensor readings (sensehat custom datamodel)
    fields:
      - wt@at                          # shorthand: value + at timestamp
      - wp                             # shorthand: value only
      - name: device.temperature.value
        at: true
        alias: temp                    # long form, custom column name
```

The new view is immediately available everywhere: `og dev search --view water`, the
TUI picker, and `devices_search(view: "water")` in MCP. Unknown view names fail loudly
with a suggestion (`unknown view "sumary" (did you mean "summary"?)`).

## CLI commands

### Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Custom config file path |
| `--profile` | | Config profile to use |
| `--org` | | Organization name |
| `--output` | `-o` | Output format: `json` or `table` (default: `table`) |
| `--insecure` | | Skip TLS certificate verification (HTTP + MQTT) |
| `--ca-file` | | PEM file with extra CA/chain certs to trust (HTTP + MQTT) |

#### TLS / self-signed servers

By default og verifies every TLS connection — REST (North/South HTTP planes) and
MQTT — against the system root store. For an on-prem OpenGate behind a self-signed
or private-CA certificate:

- `--insecure` skips verification entirely (accept any cert, no CA needed). It is a
  global flag and applies to **all** connections; og prints a warning to stderr.
- `--ca-file <pem>` trusts an extra CA/chain without disabling verification.

Both are resolved as **flag → profile → env** (`OG_INSECURE`, `OG_CA_FILE`). The
flag wins per invocation; the profile value persists across commands. Running
`og login --insecure` (or `--ca-file`) writes the setting into the active profile,
so subsequent commands inherit it without repeating the flag.

```bash
# One-off against a self-signed server
og --insecure dev list

# Persist it in the profile at login time
og login -e user@example.com --insecure --profile onprem
og --profile onprem dev list          # no --insecure needed

# Trust a private CA instead of skipping verification
og --ca-file /etc/og/ca.pem dev list
```

### login

Authenticate against OpenGate and store JWT token, API key, and organization in the active profile.

```bash
og login -e user@example.com
og login -e user@example.com -p mypassword
og login -e user@example.com --profile staging
og login -e user@example.com --2fa-code 123456     # accounts with 2FA (TOTP) enabled
og login -e user@example.com --2fa-secret JBSWY3DPEHPK3PXP   # store the seed; og generates the code from now on
```

The password is prompted securely if not provided. The API key (needed for IoT data collection) is obtained automatically from the login response.

**Two-factor authentication (2FA).** If the account has TOTP 2FA enabled, `og login`
asks for the 6-digit code interactively and retries. For non-interactive use
(scripts, CI) supply it up front with `--2fa-code` or the `OG_2FA_CODE` env var —
codes expire after 30 seconds, so generate a fresh one per run.

**Hands-off 2FA — store the seed.** To skip the prompt entirely, give og the base32
TOTP secret once with `--2fa-secret`: it is saved in the profile (the config file is
forced to `0600`) and og derives the 6-digit code on every subsequent login. For
secret-management setups that must not touch disk, pass it per run via `OG_2FA_SECRET`
instead — that path is never persisted. ⚠️ A stored seed is a *permanent* second
factor: anyone who can read `~/.og/config.yaml` can mint codes, so it degrades 2FA to
effectively single-factor on that machine. Use it on trusted hosts / CI runners only.
Enabling/resetting 2FA itself is done in the OpenGate web UI, not from `og`.

**2FA everywhere, not just the CLI.** The interactive TUI (`og` with no args) and the
MCP `login` tool handle 2FA too: the TUI auto-derives the code from a stored
`--2fa-secret` and reveals a "2FA code" field when the server issues a challenge, and
the MCP tool accepts `2FaCode` / `2FaSecret` parameters — same behaviour as the CLI.

### datamodels (alias: dm)

Manage OpenGate data models.

```bash
# Search
og dm search
og dm search -w "datamodels.identifier like weather"
og dm search -w "datamodels.organizationName eq sensehat" --limit 5

# Get (shows categories and datastreams)
og dm get weather --org sensehat
og dm get weather --org sensehat -o json

# CRUD
og dm create --org sensehat -f datamodel.json
og dm update weather --org sensehat -f datamodel.json
og dm delete weather --org sensehat
```

Example output for `og dm get`:

```
Category  Datastream  Name         Period   Schema  Access
weather   wt          Temperature  INSTANT  number  READ
weather   wp          Pressure     INSTANT  number  READ
```

### devices (alias: dev)

Manage OpenGate devices.

```bash
# Search
og dev search
og dev search -w "provision.device.administrativeState eq ACTIVE"
og dev search -w "provision.device.identifier like sense" --limit 10

# Filter by the latest collected datastream value (works for any default or custom stream)
og dev search -w "wt gt 20"
og dev search -w "wt gte 10 AND wt lte 30 AND provision.administration.organization eq sensehat"
og dev search -w "device.temperature.value gt 50 AND provision.device.operationalStatus eq NORMAL"

# Select specific fields (dynamic columns); @at adds the value timestamp
og dev search -s provision.device.identifier -s wt@at -s wp \
              -w "provision.device.identifier like sense"

# Named views — common field sets without memorizing paths (see "Views" above)
og dev search --view summary
og dev search --view summary,power -s wt

# Get
og dev get sense-001
og dev get sense-001 -o json

# CRUD
og dev create --org sensehat -f device.json
og dev update sense-001 --org sensehat -f device.json
og dev delete sense-001 --org sensehat
```

**Select** (`-s`): choose which datastreams/fields to return. Append `@at` to a field
(or pass `--at` for all of them) to also get the timestamp of the current value.
Without `-s`/`--view`, default columns are Identifier, Name, Organization, and State.

### views

Inspect the named views available for `--view` (see [Views](#views--project-fields-by-intent)).

```bash
og views list           # NAME, SOURCE (builtin / file path), DESCRIPTION
og views show summary   # expansion: datastream, projected fields, alias
```

### alarms (alias: al)

Monitor and manage OpenGate alarms.

```bash
# Search
og alarms search
og alarms search -w "alarm.severity eq CRITICAL"
og alarms search -w "alarm.status eq OPEN" -w "alarm.severity eq URGENT"
og alarms search -w "alarm.entityIdentifier like sense" --limit 10

# Summary (counts by severity, status, rule, name)
og alarms summary
og alarms summary -w "alarm.status eq OPEN"

# Actions
og alarms attend <alarm-uuid> --notes "Investigating"
og alarms close <alarm-uuid> --notes "Resolved"
```

**Alarm fields:** `alarm.severity` (INFORMATIVE, URGENT, CRITICAL), `alarm.status` (OPEN, ATTEND, CLOSED), `alarm.name`, `alarm.rule`, `alarm.entityIdentifier`, `alarm.organization`, `alarm.channel`, `alarm.priority` (LOW, MEDIUM, HIGH), `alarm.openingDate`.

### rules

Manage automation rules. Two modes: **EASY** (declarative condition + actions) and
**ADVANCED** (a JavaScript function decides). Rules are channel-scoped — all
commands take `--channel` (default: `default_channel`) plus the global `--org`.

```bash
# Search (note the rule. prefix in filters)
og rules search
og rules search -w "rule.active eq true"
og rules search -w "rule.mode eq ADVANCED"

# Inspect
og rules get <rule-id> --org sensehat
og rules catalog                        # predefined rule templates

# CRUD from a JSON file
og rules create --org sensehat -f rule.json
og rules update <rule-id> --org sensehat -f rule.json
og rules delete <rule-id> --org sensehat

# Enable / disable without editing JSON
og rules enable <rule-id> --org sensehat
og rules disable <rule-id> --org sensehat

# Local editing cycle — ADVANCED rules' JavaScript becomes a real .js file
og rules pull <rule-id> --dir rules/ --org sensehat
#   → rules/<rule-slug>/rule.json + javascript.js
$EDITOR rules/<rule-slug>/javascript.js
og rules deploy rules/<rule-slug> --update --org sensehat

# pull-all / wrap mirror the workspace verbs
og rules pull-all --dir rules/ --org sensehat
og rules wrap rules/<rule-slug> --out rule.json
```

ADVANCED rule JavaScript context: `entity['<datastream>']._value._current.value`
(and `._previous.value`), `parameterObject` + `getVariableValue()`, `ruleName`,
`openAlarm(null, name, ruleName, severity, priority, message)`. See
[demo/rules/](demo/rules/default_channel/env-anomaly/javascript.js) for a working
multi-datastream rule with hysteresis.

### connectors (alias: cf)

Manage **connector functions** — JavaScript hooks in the device-integration
pipeline. Three types: **REQUEST** (transform an outgoing operation request,
matched by `operationName` + `northCriterias`), **RESPONSE** (process an
operation response from the device) and **COLLECTION** (process collected data
and emit datapoints) — both matched by `southCriterias` (URIs, topics, OIDs).
The code lives in the `javascript` field; `operationalStatus` is one of
`DISABLED | PRODUCTION | TEST`. Connector functions are channel-scoped — all
commands take `--channel` (default: `default_channel`) plus the global `--org`.

```bash
# List and inspect
og connectors list --org sensehat
og connectors get <cf-id> --org sensehat
og connectors catalog                     # predefined connector function templates

# CRUD from a JSON file
og connectors create --org sensehat -f cf.json
og connectors update <cf-id> --org sensehat -f cf.json
og connectors delete <cf-id> --org sensehat

# Change operationalStatus without editing JSON
og connectors status <cf-id> TEST --org sensehat
og connectors enable <cf-id> --org sensehat    # → PRODUCTION
og connectors disable <cf-id> --org sensehat   # → DISABLED

# Local editing cycle — the connector function's JavaScript becomes a real .js file
og connectors pull <cf-id> --dir connectors/ --org sensehat
#   → connectors/<cf-slug>/connectorfunction.json + javascript.js
$EDITOR connectors/<cf-slug>/javascript.js
og connectors deploy connectors/<cf-slug> --update --org sensehat

# pull-all / wrap mirror the rules and workspace verbs
og connectors pull-all --dir connectors/ --org sensehat
og connectors wrap connectors/<cf-slug> --out cf.json
```

COLLECTION JavaScript uses the `collection` global
(`collection.addDatapoint(datastreamId, value, at, source, sourceInfo)`,
`collection.send()`); concatenated executions use the `cf` global
(`cf.response(criteria, payload)`, `cf.collection(criteria, payload)`).

### provision (alias: pf)

Manage **provision functions** — "provision processors" in the API. A provision
function is a JavaScript script that turns inbound rows (typically an Excel
sheet) into ODM provisioning actions (create/update/delete assets, devices,
subscriptions, subscribers). The script must implement two functions —
`normalizeRawObject(rawObject)` (validate + shape one row) and
`actionsPlanning(normalizedObject)` (return the array of actions) — and lives in
the `scriptProcessor.script` field. Provision functions are **organization-scoped**
(global `--org`, no `--channel`) and have no status field.

```bash
# List and inspect
og provision list --org sensehat
og provision get <pp-id> --org sensehat

# CRUD from a JSON file
og provision create --org sensehat -f pp.json
og provision update <pp-id> --org sensehat -f pp.json
og provision delete <pp-id> --org sensehat

# Local editing cycle — the script becomes a real .js file
og provision pull <pp-id> --dir provision/ --org sensehat
#   → provision/<pp-slug>/provisionfunction.json + scriptProcessor__script.js
$EDITOR provision/<pp-slug>/scriptProcessor__script.js
og provision deploy provision/<pp-slug> --update --org sensehat

# pull-all / wrap mirror the rules and connector verbs
og provision pull-all --dir provision/ --org sensehat
og provision wrap provision/<pp-slug> --out pp.json
```

Closing the loop with execution — **`plan`** is a dry-run that returns the
computed action plan as JSON **without mutating any data**, ideal for iterating
on a script; **`bulk`** runs the full provisioning:

```bash
# Dry-run the first 3 rows of an Excel file — no data is changed
og provision plan <pp-id> --file data.xlsx --rows 3 --org sensehat

# Run for real, then track the bulk process and download its result Excel
og provision bulk <pp-id> --file data.xlsx --org sensehat
og provision bulk-status <bulk-id> --org sensehat
og provision bulk-details <bulk-id> --out result.xlsx --org sensehat
```

> `plan`/`bulk`/`bulk-details` take a local file path and are CLI/MCP operations;
> the TUI covers the list/inspect surface.

### optypes (alias: operation-types)

Manage operation **type definitions** — the DDL side of operations. Define a
custom operation here, then launch it on devices with `og jobs create`.

```bash
og optypes catalog                      # predefined operation types
og optypes search                       # catalog + your custom ones
og optypes get REBOOT_EQUIPMENT --org sensehat

og optypes create --org sensehat -f optype.json
og optypes update CALIBRATE_SENSOR --org sensehat -f optype.json
og optypes delete CALIBRATE_SENSOR --org sensehat
```

The definition's `parameters` field is a **JSON Schema object** (not an array) —
see [demo/operations/types/calibrate-sensor.json](demo/operations/types/calibrate-sensor.json)
for a complete working example.

### timeseries (alias: ts)

Manage OpenGate time series — aggregated temporal data.

```bash
# List and get
og ts list
og ts get <id>
og ts get <id> -o json

# Query data
og ts data <id>
og ts data <id> -w "Prov Identifier eq MyDevice1"
og ts data <id> --sort EntityAscBucketDesc --limit 50

# CRUD
og ts create -f timeseries.json
og ts update <id> -f timeseries.json
og ts delete <id>

# Export to Parquet
og ts export <id>
```

### datasets (alias: ds)

Manage OpenGate datasets — columnar snapshots of device data.

```bash
# List and get
og ds list
og ds get <id>
og ds get <id> -o json

# Query data
og ds data <id>
og ds data <id> -w "Prov Identifier eq MyDevice1" --limit 50

# CRUD
og ds create -f dataset.json
og ds update <id> -f dataset.json
og ds delete <id>
```

### jobs

Manage OpenGate operation jobs — execute operations on devices.

```bash
# Search
og jobs search
og jobs search --limit 10
og jobs search -w "jobs.report.summary.status eq IN_PROGRESS"
og jobs search -w "jobs.request.name eq REBOOT_EQUIPMENT"

# Get report / create / cancel
og jobs get <job-id>
og jobs create -f job.json
og jobs cancel <job-id>

# List per-device operations within a job
og jobs operations <job-id>
```

Example job JSON for REBOOT_EQUIPMENT:

```json
{
  "job": {
    "request": {
      "name": "REBOOT_EQUIPMENT",
      "parameters": { "type": "HARDWARE" },
      "active": true,
      "schedule": { "stop": { "delayed": 90000 } },
      "operationParameters": { "timeout": 85000, "retries": 0 },
      "target": { "append": { "entities": ["sense-001"] } }
    }
  }
}
```

### tasks

Manage OpenGate operation tasks — scheduled/recurring operations.

```bash
og tasks search
og tasks search -w "tasks.state eq ACTIVE"
og tasks get <task-id>
og tasks create -f task.json
og tasks cancel <task-id>
og tasks jobs <task-id>     # list jobs spawned by a task
```

### workspace (alias: ws)

Manage OpenGate **workspaces** (Web API `/api/v1`). Workspaces are the top-level UI
container — every workspace owns one or more dashboards.

```bash
# List and get
og workspace list
og workspace list --full          # include embedded dashboards
og workspace get <workspace-id>
og workspace get <workspace-id> --full

# Export (cross-tenant migration / backups)
og workspace export <workspace-id> --out ws.json
og workspace export <workspace-id> --dir backups/      # auto-naming: backups/<id>.json
og workspace export <workspace-id> --full --out ws.json

# Batch export every workspace
og workspace export-all --dir backups/

# Unwrap into editable directory tree (IDE / AI friendly)
# Aliases: unwrap → pull, unwrap-all → pull-all, unwrap-file → pull-file
og workspace pull <workspace-id> --dir wsroot/      # (alias of unwrap)
og workspace pull-all --dir wsroot/
og workspace pull-file ws.json --dir wsroot/

# All unwrap/pull commands accept --force to overwrite an existing destination

# Ownership filter: unwrap only writes items you can actually edit. A workspace
# (or nested dashboard) whose `owner` is not the active profile's email is not
# editable by you — re-importing it would fail or clobber someone else's work.
#   - single-item (pull / pull-file): refuses loudly with the owner shown
#   - bulk (pull-all): skips non-owned items with a warning, continues the rest
# Pass --force-owner to override on single-item commands (e.g. to inspect a
# shared/system workspace read-only). On pull, --force-owner also forces the
# workspace's nested dashboards.
og workspace pull <workspace-id> --dir wsroot/ --force-owner

# Share with other users/domains — OPTIONAL and separate from deploy/import:
# publishing never shares by itself (the workspace stays owner-only until you
# share it). The lists REPLACE current sharing on every call.
og workspace share <workspace-id> --user claudia@amplia.es
og workspace share <workspace-id> --user a@x.com --user b@x.com --domain partners
og workspace share <workspace-id> --unshare

# Wrap back into a single JSON ready for import
og workspace wrap wsroot/<workspace-slug> --out ws.json

# Or deploy in one step (wrap + import, no intermediate file)
og workspace deploy wsroot/<workspace-slug>            # POST: create
og workspace deploy wsroot/<workspace-slug> --update   # PUT: overwrite

# Import / update / delete
og workspace import -f ws.json              # POST: creates workspace + its dashboards (multi-phase)
og workspace import -f ws.json --update     # PUT: overwrites workspace + all its dashboards
og workspace update <workspace-id> -f ws.json
og workspace delete <workspace-id>
```

`og workspace import` replays the same multi-phase flow the OpenGate web-UI
wizard uses, so the workspace's dashboards (and the JavaScript inside their
widgets) are actually persisted:

```
POST /api/workspaces                ← workspace shell (no dashboards inline)
POST /api/dashboards × N            ← each dashboard with full grid + widgets
PUT  /api/workspaces/{id}           ← shell + dashboards[] as grid-layout refs
```

`--update` is the symmetric variant for re-deploying after edit:

```
PUT /api/dashboards/{id} × N        ← each dashboard with its (edited) widgets
PUT /api/workspaces/{id}            ← shell + dashboards[] as grid-layout refs
```

This makes the `unwrap → edit JS → wrap → import --update` cycle work
correctly: changes to the extracted `.js` files land on the server.

#### Unwrap structure (for IDE editing and AI agents)

`og workspace unwrap` explodes a workspace into one folder per nesting level
and extracts any embedded JavaScript code into standalone `.js` files. This
lets you edit widget formatters and operation scripts as regular `.js` files
with syntax highlighting, lints, and AI assistance.

```
wsroot/
  dashboards-adif/                                   # <workspace-slug>
    workspace.json                                   # workspace metadata (dashboards stripped)
    00__visualizaci-n-pbi/                           # NN__<dashboard-slug> — preserves array order
      dashboard.json                                 # dashboard metadata + _workspaceLayout
      00__customchart__1727269767709-0/              # NN__<widget-type>__<wid>
        widget.json                                  # grid item + cleaned config
        _widgetConfigCode.js                         # extracted JS (9 KB of chart code)
    01__visualizaci-n-pbi-m-ximos/
      dashboard.json
      00__customchart__1727358473084-0/
        widget.json
        _widgetConfigCode.js
    02__comparativa-vibraciones/
      ...
```

JavaScript is extracted automatically when the field is named `formatter`,
`script`, `operation`, `code`, `fn`, `expression`, `_widgetConfigCode`, **or**
when a string is long enough and contains JS keywords (`function`, `return`,
`=>`, `const`, `let`, `var`). Nested fields keep their keypath in the
filename, e.g. `columns__0__formatter.js`.

The cycle is content-lossless: `og workspace wrap <dir>` produces a workspace
JSON with identical configuration trees (same SHA256 per widget config) as the
original export, modulo cosmetic differences in `null`/default field
serialisation.

### dashboard (alias: dash)

Manage OpenGate **dashboards** (Web API `/api/v1`). Every dashboard belongs to exactly
one workspace (1-N hierarchy).

```bash
# List — iterates workspaces with ?full=1 and shows their dashboards
og dashboard list
og dashboard list --workspace <workspace-id>          # filter by workspace

# Get
og dashboard get <dashboard-id>

# Export (single)
og dashboard export <dashboard-id> --out dash.json
og dashboard export <dashboard-id> --dir backups/     # auto-naming: backups/<id>.json

# Batch export every dashboard (or just a workspace's)
og dashboard export-all --dir backups/
og dashboard export-all --dir backups/ --workspace <workspace-id>

# Import / update / delete
og dashboard import -f dash.json                      # POST, uses workspace from JSON
og dashboard import -f dash.json --workspace <id>     # POST, override target workspace
og dashboard import -f dash.json --update             # PUT: overwrites the dashboard whose _id is in the file
og dashboard update <dashboard-id> -f dash.json
og dashboard delete <dashboard-id>

# Pull dashboards for editing (aliases: unwrap, unwrap-all, unwrap-file)
og dashboard pull <dashboard-id> --dir dashroot/             # one dashboard
og dashboard pull-all --dir dashroot/                        # every dashboard
og dashboard pull-all --dir dashroot/ --workspace <ws-id>    # only one workspace
og dashboard pull-file dash.json --dir dashroot/             # from local JSON file
og dashboard pull <dashboard-id> --dir dashroot/ --force     # overwrite existing
og dashboard pull <dashboard-id> --dir dashroot/ --force-owner  # unwrap even if not owned by you

# Ownership filter (same as workspaces): single-item pull/pull-file refuse a
# dashboard whose owner is not the active profile (use --force-owner to
# override); pull-all skips non-owned dashboards with a warning and continues.

# Share a single dashboard (same semantics as workspace share)
og dashboard share <dashboard-id> --user claudia@amplia.es
og dashboard share <dashboard-id> --unshare

# Wrap an edited dashboard directory back into JSON (no import)
og dashboard wrap dashroot/<dashboard-dir>                 # stdout
og dashboard wrap dashroot/<dashboard-dir> --out d.json    # to file

# Deploy a single dashboard directory in one step (wrap + import)
og dashboard deploy wsroot/<ws>/<dashboard-dir>
og dashboard deploy wsroot/<ws>/<dashboard-dir> --update
og dashboard deploy wsroot/<ws>/<dashboard-dir> --workspace <other-ws-id>
```

Dashboard verb pair `pull ↔ deploy` (same as workspace):

```bash
og dashboard pull <id> --dir dashroot/   # GET dashboard, write tree with widget JS extracted
# ... edit in IDE / IA ...
og dashboard deploy dashroot/<slug> --update   # PUT dashboard
```

### Web API authentication

Workspaces and dashboards live in the OpenGate **Web API** (`/api/...`), a
separate surface from the North IoT API. The Web API uses its own JWT, obtained
automatically by `og login` via `POST /api/auth/signin/internal`.

OpenGate only allows **one active web session per user**. If you log into the
OpenGate web UI in another tab, your CLI web token is invalidated. `og`
detects HTTP 401 from the Web API and transparently re-signs in once before
retrying, so commands keep working without manual `og login` reruns.

Login flags:

```bash
og login --domain X --workgroup Y --user-profile Z   # override defaults
og login --no-web                                     # skip web signin entirely
```

**Cross-tenant migration pattern**:

```bash
# 1. Log in to source tenant and export
og login --profile source
og workspace export ws-id --out ws.json
og dashboard export dash-id --out dash.json

# 2. Log in to destination tenant and import
og login --profile destination
og --profile destination workspace import -f ws.json
og --profile destination dashboard import -f dash.json --workspace <new-ws-id>
```

### iot

Device integration via the South API (X-ApiKey authentication). The API key is obtained automatically from the login response.

```bash
# Send a single value
og iot collect sense-001 wt 25.3
og iot collect sense-001 wp 1013

# Send a full payload from file
og iot collect-file sense-001 -f payload.json

# Trigger a connector function over its HTTP south route (raw body, custom path).
# Unlike collect/iot (which bypasses connector functions), this hits the CF's
# southCriteria path so the CF transforms the body and emits datapoints.
og iot collect-raw charlie-01 --route ogcli-demo --body '{"raw":21,"id":"abc"}'
og iot collect-raw charlie-01 --route raw/feed -f reading.json --content-type text/plain
```

#### MQTT (South plane)

og also speaks the OpenGate **MQTT** south connector — publish telemetry, observe
traffic, and act as a full **virtual device**. Auth is `username = device-id`,
`password = API key` (broker derived from the profile host, port 1883; `--tls` for
8883). TLS is **verified against the system root store** by default; the global
`--ca-file <pem>` / `--insecure` flags (see [TLS / self-signed servers](#tls--self-signed-servers))
apply here too. Topics default to `odm/iot/<id>` (data), `odm/request/<id>`
(operations) and `odm/response/<id>` (responses), but **any `--topic` is accepted** —
connector functions define their own southCriterias, so topics are not fixed.

```bash
# Publish data over MQTT (instead of HTTP collect)
og iot publish sense-001 temperature 21.5
og iot publish sense-001 -f payload.json
og iot publish sense-001 --topic my/cf/route --raw '{"raw":21,"id":"abc"}'   # custom CF route

# Observe any topic live (debug CFs / operations); --count N to stop after N
og iot subscribe sense-001                      # watch incoming operations
og iot subscribe sense-001 --topic odm/iot/sense-001

# Run as a virtual device: auto-answer operations launched against it
og iot device sense-001                          # answers SUCCESSFUL on odm/response/<id>
og iot device sense-001 --refresh-data refresh.json   # also fulfils REFRESH_INFO with data
```

`og iot device` subscribes to the request topic and publishes an acknowledging
response for every operation it receives, so a `og jobs create` REBOOT_EQUIPMENT /
REFRESH_INFO targeting the device completes against the virtual device.

Payload format:

```json
{
  "version": "1.0.0",
  "datastreams": [
    {"id": "wt", "datapoints": [{"value": 25.3}]},
    {"id": "wp", "datapoints": [{"value": 1013}]}
  ]
}
```

### mcp

Start the MCP (Model Context Protocol) server, exposing all commands as LLM tools.

```bash
og mcp                            # stdio transport (default)
og mcp --http :8080               # HTTP transport (single-tenant: startup profile)
og mcp --http :8080 --multi-tenant # HTTP, per-request credentials from headers
```

**Prerequisites:** run `og login` first to store credentials.

#### Multi-tenant HTTP mode

For a multi-user chatbot, `--multi-tenant` makes the server a **stateless conduit**:
credentials are read **per request from HTTP headers**, never from tool arguments
(which would flow through the LLM) and never from the startup profile. Each request
must carry the caller's own identity:

- `Authorization: Bearer <north JWT>` — required for every tool.
- `X-OG-Web-Token: <web JWT>` — required by workspace/dashboard tools (OpenGate's
  north and web planes use different JWTs).
- `X-OG-Api-Key: <api key>` — required by South/IoT tools.

A tool returns an error if a credential it needs is absent (no cross-user fallback).
The `login` tool is **not exposed** in this mode (it would return a JWT to the LLM).
**Run TLS in front** — tokens travel in headers. The host stays a fixed server config
(one og MCP server = one OpenGate instance); only credentials vary per request.

#### Automatic client setup (`og mcp install`)

Instead of editing JSON by hand, register og in a Claude client automatically:

```bash
og mcp install                          # Claude Desktop (default) — per-OS config path
og mcp install --client claude-code     # Claude Code — project-scoped .mcp.json in cwd
og mcp install --profile production      # bake --profile into the server args
og mcp install --print                  # dry-run: show path + entry, write nothing
og mcp uninstall                         # remove the entry again
```

It writes the **absolute path** of the og binary (Claude Desktop does not inherit
your shell `PATH`, so `"command": "og"` would fail), merges non-destructively
(other servers and keys are preserved, the original is backed up to `<file>.bak`),
and refuses to touch a config file that is not valid JSON. Config locations:

| Client | Location |
|--------|----------|
| `claude-desktop` (macOS) | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| `claude-desktop` (Windows) | `%APPDATA%\Claude\claude_desktop_config.json` |
| `claude-desktop` (Linux, unofficial build) | `~/.config/Claude/claude_desktop_config.json` |
| `claude-code` | `./.mcp.json` (or `--dir`) |

Flags: `--client`, `--name` (entry key, default `opengate`), `--dir` (override the
directory), `--profile` (global), `--print`. Restart Claude Desktop after installing.

##### Scope: where the server becomes available

The two clients differ in how far the registration reaches:

- **Claude Desktop** has a single, app-wide config, so `og mcp install` makes the
  server available **everywhere in the app**. Nothing else to do (just restart it).
- **Claude Code** is registered at **project scope**: `og mcp install --client
  claude-code` writes `.mcp.json` in the current directory, so the server is
  available **only when you open Claude Code in that directory** (or a subdirectory).
  This works, but it is per-project on purpose — `og` does not edit the user-global
  `~/.claude.json`, which holds a lot of Claude Code state.

To make the server available in **every** directory under Claude Code, register it
at user scope yourself with the Claude CLI (which manages `~/.claude.json` safely):

```bash
claude mcp add --scope user opengate -- og mcp --stdio
# add a profile: ... -- og mcp --stdio --profile production
```

Tip: pass the absolute path of the binary (`$(command -v og)`) if `og` is not on
the PATH that Claude Code sees. Use `og mcp install --client claude-code --print`
to get the exact command/args to reuse.

#### Manual client configuration

**Claude Code** (`~/.claude/settings.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "opengate": {
      "command": "og",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

**LM Studio** (MCP server configuration):

```json
{
  "mcpServers": {
    "opengate": {
      "command": "/path/to/og",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

**Multiple environments** (use `--profile`):

```json
{
  "mcpServers": {
    "opengate-production": {
      "command": "og",
      "args": ["mcp", "--stdio", "--profile", "production"]
    },
    "opengate-staging": {
      "command": "og",
      "args": ["mcp", "--stdio", "--profile", "staging"]
    }
  }
}
```

For a detailed guide on how prompts, resources, and tools work together, see [docs/mcp-integration.md](docs/mcp-integration.md).

#### Tools

| Tool | Description |
|------|-------------|
| `login` | Authenticate with email/password |
| `devices_search` | Search devices with query/filter/select/view |
| `devices_get` | Get device detail |
| `devices_create` | Create device from JSON |
| `devices_update` | Update device from JSON |
| `devices_delete` | Delete device |
| `datamodels_search` | Search data models with query/filter |
| `datamodels_get` | Get data model with categories/datastreams |
| `datamodels_create` | Create data model from JSON |
| `datamodels_update` | Update data model from JSON |
| `datamodels_delete` | Delete data model |
| `alarms_search` | Search alarms with query/filter |
| `alarms_summary` | Alarm counts by severity/status/rule |
| `alarms_attend` | Mark alarms as attended |
| `alarms_close` | Close alarms |
| `timeseries_list` | List time series in organization |
| `timeseries_get` | Get time series definition |
| `timeseries_create` | Create time series from JSON |
| `timeseries_update` | Update time series from JSON |
| `timeseries_delete` | Delete time series |
| `timeseries_data` | Query data from a time series |
| `timeseries_export` | Trigger Parquet export |
| `datasets_list` | List datasets in organization |
| `datasets_get` | Get dataset definition |
| `datasets_create` | Create dataset from JSON |
| `datasets_update` | Update dataset from JSON |
| `datasets_delete` | Delete dataset |
| `datasets_data` | Query data from a dataset |
| `jobs_search` | Search operation jobs |
| `jobs_get` | Get job report with execution summary |
| `jobs_create` | Create and launch operation job |
| `jobs_cancel` | Cancel a running job |
| `jobs_operations` | List per-device operations within a job |
| `rules_search` | Search automation rules (rule.name, rule.mode, rule.active) |
| `rules_get` | Get a rule (ADVANCED rules include their JavaScript) |
| `rules_create` | Create an EASY or ADVANCED rule |
| `rules_update` | Update a rule (full body) |
| `rules_delete` | Delete a rule |
| `rules_set_active` | Enable/disable a rule |
| `rules_catalog` | Platform rules catalog (predefined templates) |
| `rules_logs` | Collect a rule's execution logs (bounded) |
| `connectors_list` | List connector functions in a channel |
| `connectors_get` | Get a connector function (includes its JavaScript) |
| `connectors_create` | Create a REQUEST/RESPONSE/COLLECTION connector function |
| `connectors_update` | Update a connector function (full body) |
| `connectors_delete` | Delete a connector function |
| `connectors_set_status` | Set operationalStatus (DISABLED/PRODUCTION/TEST) |
| `connectors_catalog` | Platform connector functions catalog (templates) |
| `connectors_logs` | Collect a connector function's execution logs (bounded) |
| `provision_list` | List provision functions in an organization |
| `provision_get` | Get a provision function (includes its script) |
| `provision_create` | Create a provision function |
| `provision_update` | Update a provision function (full body) |
| `provision_delete` | Delete a provision function |
| `provision_plan` | Dry-run an Excel against a provision function (no data mutated) |
| `provision_bulk` | Run a full provisioning bulk from an Excel file |
| `provision_bulk_status` | Read a bulk process status summary |
| `provision_bulk_details` | Download the result Excel of a bulk process |
| `optypes_catalog` | Catalog of predefined operation types |
| `optypes_search` | Search operation type definitions |
| `optypes_get` | Get an operation type (parameters schema, steps) |
| `optypes_create` | Define a new custom operation type |
| `optypes_update` | Update an operation type definition |
| `optypes_delete` | Delete an operation type definition |
| `tasks_search` | Search operation tasks |
| `tasks_get` | Get task detail |
| `tasks_create` | Create operation task |
| `tasks_cancel` | Cancel a task |
| `iot_collect` | Send a single data point to a device (HTTP South) |
| `iot_collect_payload` | Send a full IoT payload to a device (HTTP South) |
| `iot_collect_raw` | POST a raw body to a connector function's HTTP south route |
| `iot_mqtt_publish` | Publish data/raw body over MQTT (topic overridable for CFs) |
| `iot_mqtt_subscribe` | Subscribe to a topic and collect up to N messages (bounded) |
| `iot_mqtt_device` | Virtual device: auto-answer operations over MQTT (bounded) |
| `workspaces_list` | List workspaces (optionally with embedded dashboards) |
| `workspaces_get` | Get a workspace by ID |
| `workspaces_export` | Export a workspace via `/workspaces/export/{id}` |
| `workspaces_import` | Create a workspace from JSON payload |
| `workspaces_update` | Update a workspace |
| `workspaces_delete` | Delete a workspace |
| `workspaces_share` | Share a workspace with users/domains (grants UI visibility) |
| `dashboards_list` | List dashboards (all, or filtered by workspace) |
| `dashboards_get` | Get a dashboard with grid layout and widgets |
| `dashboards_export` | Export a dashboard via `/dashboards/export/{id}` |
| `dashboards_import` | Create a dashboard from JSON payload, optionally overriding target workspace |
| `dashboards_update` | Update a dashboard |
| `dashboards_delete` | Delete a dashboard |
| `dashboards_share` | Share a dashboard with users/domains |

Search tools accept a `query` parameter with the same syntax as `-w` flags:

```
devices_search(
  query: "provision.device.administrativeState eq ACTIVE AND provision.device.identifier like sense",
  select: "provision.device.identifier,wt@at",
  limit: 10
)

# Filtering by the latest collected datastream value is also supported
devices_search(query: "wt gt 20")
devices_search(query: "device.temperature.value gt 50 AND provision.device.operationalStatus eq NORMAL")

# Named views project common field sets without knowing datastream paths
devices_search(query: "provision.administration.organization eq sensehat", view: "summary")
devices_search(query: "device.powersupply.battery.charge lt 20", view: "summary,power")
```

#### Prompts

| Prompt | Description |
|--------|-------------|
| `opengate-guide` | Complete guide covering all tools, query syntax with operator mapping (ES/EN → eq/like/gt/...), fields per entity, job creation format, IoT data collection, and worked examples. See [docs/mcp-prompts.md](docs/mcp-prompts.md) for full content. |

#### Resources

| Resource | Description |
|----------|-------------|
| `opengate://query-syntax` | Static reference of query operators, fields per entity, and job operation types |
| `opengate://organizations/{org}/datamodel-fields` | Dynamic: lists all datastream fields available in an organization's datamodels, fetched live from the API |
| `opengate://views` | Merged dictionary of named field views (built-in + user + project layers) with what each one expands to |

### version

```bash
og version
```

### completion

Generate shell autocompletion (bash, zsh, fish, powershell):

```bash
og completion zsh > "${fpath[1]}/_og"            # zsh
og completion bash > /etc/bash_completion.d/og   # bash
```

## Demo — end-to-end IoT scenario

[demo/](demo/README.md) is a complete, copy/paste runbook that exercises the whole
platform from `og`: datamodel → device fleet → telemetry injection → automation
rules (EASY + ADVANCED with locally-edited JavaScript) → rule-triggered alarms →
custom operation definition + launch → published dashboard with 6 widgets.

It doubles as the **reference layout for local OpenGate projects**: every
artifact (provisioning, payloads, rules with their `.js`, operation types, jobs,
unwrapped workspaces) lives in versionable files.

## Claude Code skills

The repo ships three [Claude Code skills](https://code.claude.com/docs/en/skills)
under `.claude/skills/` (the canonical, versioned location for project skills).
They teach Claude — and any AI agent reading them — how to use `og` to its full
potential. Skills load on demand: their one-line description is always visible to
the agent, and the full content is pulled in only when the task matches.

| Skill | Teaches | Supporting files |
|-------|---------|------------------|
| [og-cli](.claude/skills/og-cli/SKILL.md) | Driving the tool: auth/profiles/multi-tenant, query syntax, field projection with views and `@at` timestamps, output formats, API quirks | [query-cookbook.md](.claude/skills/og-cli/query-cookbook.md) (copy/paste recipes by intent), [views-guide.md](.claude/skills/og-cli/views-guide.md), [mcp-setup.md](.claude/skills/og-cli/mcp-setup.md) |
| [og-workspaces](.claude/skills/og-workspaces/SKILL.md) | Dashboards & widgets: verified widget catalog, local directory structure, pull/edit/wrap/deploy lifecycle, JS extraction, multi-phase import, cross-tenant migration | `reference/` — 23 per-widget JSON deep-dives (customTable, customChart, browsers, ...) |
| [og-device-ops](.claude/skills/og-device-ops/SKILL.md) | Acting on devices: operation jobs, scheduled tasks, alarm triage workflow, IoT data injection via South API | [job-templates.md](.claude/skills/og-device-ops/job-templates.md) (ready job JSONs with realistic timeouts) |

Division of labor: *querying* OpenGate → og-cli; *building UI* → og-workspaces;
*acting on devices* → og-device-ops. The skills complement (never duplicate) this
README: they encode workflows, decision criteria, and platform quirks; the README
remains the exhaustive command reference.

**Convention:** any PR that adds or changes commands must update the relevant
skill, the same way it must update this README (see [CLAUDE.md](CLAUDE.md)).

### Installing the skills without the repo

The skills are embedded in the `og` binary and versioned together with it, so a
colleague who installed `og` can drop them onto disk without cloning the repo:

```bash
og skills list                 # show the skills embedded in this binary
og skills extract              # write them to ./.claude/skills/ (project-local)
og skills extract --global     # write them to ~/.claude/skills/ (any directory)
og skills extract --dir PATH   # write them to an arbitrary directory
```

`extract` replaces each skill as a whole (no stale files left behind) and only
touches the skills `og` manages — other skills in the destination are left
alone. If a skill already exists it **aborts and lists** what it would replace;
re-run with `--force` to overwrite, adding `--backup` to keep a `<skill>.bak`
copy of the previous version.

## Documentation

| Document | Description |
|----------|-------------|
| [docs/mcp-integration.md](docs/mcp-integration.md) | MCP architecture: how prompts, resources, and tools work together |
| [docs/mcp-prompts.md](docs/mcp-prompts.md) | Full content of MCP prompts with explanation of each section |
| [INTEGRATION_PLAN.md](INTEGRATION_PLAN.md) | API integration roadmap and progress |
| [CLAUDE.md](CLAUDE.md) | Instructions for Claude Code when working in this repo |

## Links

- [Amplía Soluciones](https://amplia-iiot.com) — company behind OpenGate
- [OpenGate Documentation](https://documentation.opengate.es) — official API reference
- [OpenGate Platform](https://opengate.es) — product page

## Disclaimer

This software is **NOT** an official product of, endorsed by, or affiliated with [Amplía Soluciones](https://amplia-iiot.com) or the OpenGate platform. "OpenGate" is a trademark of Amplía Soluciones. This project is an independent, community-driven tool.

THIS SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NONINFRINGEMENT.

IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES, OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT, OR OTHERWISE, ARISING FROM, OUT OF, OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

**By using this software, you acknowledge and agree that:**

1. **You are solely responsible** for any and all consequences arising from the use of this tool, including but not limited to data loss, service disruption, unauthorized access, or any damage to production or non-production environments.
2. **Any misuse**, including but not limited to unauthorized access to systems, malicious operations, denial-of-service actions, or any activity that violates applicable laws or the terms of service of the OpenGate platform, is **strictly prohibited** and is the sole responsibility of the individual performing such actions.
3. **You assume all risk** associated with connecting this tool to any OpenGate instance, whether in development, staging, or production environments. The authors bear no responsibility for any impact on such environments.
4. **You are responsible** for securing any credentials (JWT tokens, API keys) stored by this tool in configuration files and for ensuring compliance with your organization's security policies.

For official support, documentation, and tools, contact [Amplía Soluciones](https://amplia-iiot.com) directly.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
