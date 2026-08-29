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
go install github.com/carlosprados/og-cli/v2@latest
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
  on-prem:
    host: https://opengate.customer.local
    api_version: v79        # instance pinned to another API version
    retries: 3              # retry 429/5xx with exponential backoff
    ca_file: /etc/ssl/customer-ca.pem
```

Global flags override the profile: `--api-version`, `--retry N`, `--insecure`,
`--ca-file`. `--api-version` applies to both REST planes (North and South).

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
| `OG_API_VERSION` | API version segment (default `v80`; for on-premises instances) |
| `OG_RETRIES` | Attempts per request; retries HTTP 429 and 5xx with backoff |

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

# With a page size
og dev search -w "provision.device.identifier like sense" --limit 10
```

**Operators:** `eq`, `neq`, `like`, `gt`, `lt`, `gte`, `lte`, `in`, `exists`

Multiple `-w` flags are combined with AND. For OR or nested queries, use `--filter` with raw JSON.

### Pagination

OpenGate searches are paged, so **the first page is not necessarily the whole
answer**. Responses carry a `page` block with `number` (the 1-based page you got)
and, on most endpoints, `of` (total pages).

```bash
# Page size (limit.size) — the platform maximum is 2000
og dev search --limit 500

# A specific page. --page is a PAGE NUMBER counting from 1, not an element offset
og dev search --limit 500 --page 3

# Every page, combined into one result — use this whenever completeness matters
og dev search --all
og dev search --all --limit 500        # --limit becomes the page size
```

`--all` and `--page` are mutually exclusive. Any question of the form "how many
devices…" needs `--all`, otherwise the answer is silently "as many as fit in one
page".

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
| `--yes` | `-y` | Skip confirmation prompts for destructive operations (delete/cancel) |

#### Destructive operations

`delete` / `cancel` commands ask for confirmation: on an interactive terminal they
prompt `[y/N]`; **without a terminal (scripts, pipelines, agents) they refuse unless
you pass `--yes`** — so a stray `og dev delete X` cannot silently destroy data. Pass
`--yes`/`-y` to proceed non-interactively.

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

# Pagination (see "Pagination" above): one page, a given page, or all of them
og dev search --limit 500
og dev search --limit 500 --page 2
og dev search --all --view summary

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

### whoami

```bash
og whoami           # who, where, and how long the token has left — local, offline, instant
og whoami --check   # also asks the platform whether it still accepts it
og whoami -o json   # for a script or an editor plugin
```

```
sensehat@amplia.es (Sense)
  profile       default
  host          https://api.opengate.es
  organization  sensehat
  token         valid for 21h36m0s (until 2026-08-30T11:35:13+02:00)
  web session   yes — workspace and dashboard commands are available
```

Reads the token's own claims, so it needs no network and no organization. That
answers the question that comes up most — is there a session, whose, and has it
expired — which a 401 from some other command cannot: it does not distinguish
"never logged in" from "expired an hour ago".

Exit 0 when there is a usable session, 1 when there is none or it has expired
(or, with `--check`, was rejected), 2 on failure. The API key the token also
carries is never printed; the JSON reports only whether one is stored.

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
#     + og-globals.d.ts + jsconfig.json  (editor completion & diagnostics)
$EDITOR rules/<rule-slug>/javascript.js
og rules deploy rules/<rule-slug> --update --org sensehat

# --no-typings skips the two generated files (and the datamodel lookup)
og rules pull <rule-id> --dir rules/ --org sensehat --no-typings

# pull also records what it fetched under <dir>/.og/ — a canonical snapshot plus
# where it came from (host, profile, organization, channel). It is a per-developer
# sync cache, so pull adds .og/ to .gitignore. Deleting it loses only the ability
# to tell a local edit from a remote one.
#
# deploy warns when it is aimed somewhere other than where the artifact was
# pulled from, which nothing recorded before:
#   warning: this rule was pulled from org staging, profile staging
#            deploying to a different organization staging → production
# It warns rather than blocks: promoting an artifact between tenants is a real
# workflow, it just should not happen by accident.

# Compare a local tree against the platform
og rules diff rules/env-anomaly --org sensehat
og connectors diff connectors/weather --org sensehat
og provision diff provision/createUpdate --org sensehat
#
# Metadata is reported as a structural diff over the canonical form, the code as
# a textual one — mixing them is what makes generated diffs unreadable. Both read
# remote → local, so the report is what deploying would do (same direction as
# `git diff`). Server-managed and requester-derived fields never participate.
#
#   ~ Environmental anomaly  (rule r-1)  [local changes]
#     metadata:
#       ~ parameters[0].value: 30 → 28
#     javascript.js  +2 −1
#           if (t) {
#         -   logger.info('remote version');
#         +   logger.warn('local version');
#
# State comes from the snapshot pull recorded:
#   ~ local changes   you edited it, nobody else did
#   ↓ remote changes  somebody else edited it — pull
#   ! conflict        both moved since the pull
#   ? unknown         no snapshot; only the raw comparison
#
og rules diff rules/env-anomaly --name-only        # just the artifact and its state
og rules diff rules/env-anomaly --context 8        # more context around code changes
og rules diff rules/env-anomaly --against production   # what differs between tenants
og rules diff rules/env-anomaly --exit-code -o json    # CI drift gate
#
# --against compares the same artifact in another profile's tenant, ignoring the
# identifiers and ownership that differ there by construction. --exit-code returns
# 1 for differences, 0 for none, 2 on error. The -o json shape is a versioned
# contract: see docs/json-output.md.

# Read one remote code file, raw — no envelope, no table, nothing around it
og rules show <rule-id> --org sensehat                       # list the files it carries
og rules show <rule-id> --org sensehat --path javascript.js  # print that one to stdout
#
# The names are the ones `pull` writes on disk, so a path from the local tree
# addresses the same file remotely. This is what an editor plugin uses for the
# remote side of a native diff view — the same command serves connectors,
# provision functions and dashboards (see `og dashboard show`, where the path is
# `<widget-dir>/<file>.js`).

# Check an artifact before deploying it — local only, no credentials needed
og rules validate rules/env-anomaly
og connectors validate connectors/weather
og provision validate provision/createUpdate --exit-code   # CI gate
#
# Metadata parses, declared code files present, brackets balance, and the
# per-family rules that catch an artifact which deploys happily and never fires:
# a REQUEST connector function with no operationName, a COLLECTION one with no
# southCriterias, an ADVANCED rule with no code, a provision script missing
# normalizeRawObject or actionsPlanning. Not a JavaScript parser — the script
# itself is covered by the generated typings and your editor's type-checker.

# Deploy on save
og rules watch rules/ --org sensehat --dry-run     # see what it would do first
og rules watch rules/ --org sensehat
og rules watch rules/ --org sensehat --json        # NDJSON, one event per line
#
#   12:52:19  started       rules  7 directories, org sensehat, debounce 300ms
#   12:52:22  deployed      env-anomaly  [local changes]
#   12:52:24  refused       collectareas [conflict]  the remote changed since you pulled
#   12:52:27  invalid       battery-low  error: javascript.js:33: "{" is never closed
#
# This is the only og command that writes without a decision per action, so:
#   • a changed file deploys the artifact it belongs to, not the whole tree
#   • one save is one deploy (editors replace files, producing several events)
#   • every artifact is validated first; --no-validate must be explicit
#   • a CONFLICT refuses to deploy, and there is NO --force
#   • add `production: true` to a profile and watch refuses to start on it
#     without --allow-production

# pull-all / wrap mirror the workspace verbs
og rules pull-all --dir rules/ --org sensehat
og rules wrap rules/<rule-slug> --out rule.json
```

ADVANCED rule JavaScript context: `entity['<datastream>']._value._current.value`
(and `._previous.value`), `parameterObject` + `getVariableValue()`, `ruleName`,
`openAlarm(null, name, ruleName, severity, priority, message)`. See
[demo/rules/](demo/rules/default_channel/env-anomaly/javascript.js) for a working
multi-datastream rule with hysteresis.

**Editor support.** `pull` also writes `og-globals.d.ts` and `jsconfig.json` next
to the code, so any LSP-capable editor — Neovim, VS Code, Cursor, Zed — gives
completion and diagnostics through `tsserver` with no editor-specific setup. The
declarations are generated from **your** platform: the organization's datamodel
becomes the set of valid datastream identifiers with their value types, and the
rule's own `parameters` become a typed `parameterObject`. So this is an error in
the editor, before deploying:

```js
entity['sensro.temperature']   // Property 'sensro.temperature' does not exist
                               // on type 'OGEntity'. Did you mean
                               // 'sensor.temperature'?
alarm.open({ severity: 'HIGH' })   // 'HIGH' is not assignable to OGSeverity
                                    // ('INFORMATIVE' | 'URGENT' | 'CRITICAL')
```

The platform half is generated from the official OpenGate documentation, so it
carries what the documentation says about each symbol — including deprecation.
The 41 globals the platform has superseded are struck through in the editor with
their replacement on hover, in the documentation's own words:

```js
responseCF(payload, criteria)   // @deprecated Use `cf.response` instead. Note the
                                // arguments are in the opposite order: `cf.response`
                                // takes the criteria first and the payload second.
```

That warning is worth reading before a bulk rename: `cf.collection` and
`cf.response` swap payload and criteria, so a naive substitution produces code
that runs and sends the payload where the criteria belongs.

Every datamodel in the organization contributes its datastreams (sensehat has 27,
holding 664 between them), plus two sources specific to the artifact: the rule's
own trigger, and the identifiers its code already reads. That last part matters —
live rules do reference datastreams no datamodel declares, and typings that
redden working code get deleted.

Regenerate standalone after a datamodel change — the header records what each
file was generated from:

```bash
og typegen --context rule/ADVANCED --org sensehat --out rules/<rule-slug>/
og typegen --context rule/ADVANCED --org sensehat --datamodel multisensor --out .   # restrict to one
og typegen --help        # available contexts

# Connector functions get them too, on pull: the context follows the function's
# type and the protocol objects follow the scheme of its south criteria
# (mqtts:// → mqtt, https:// → http). og typegen inside a connector function
# directory detects both from connectorfunction.json.
og connectors pull <cf-id> --dir cfs/ --org sensehat
og connectors pull <cf-id> --dir cfs/ --org sensehat --no-typings

# Where a script cannot be type-checked without flagging correct code — a
# top-level `return` (the platform wraps the script in a function), an untyped
# helper parameter, or a dynamic entity index — the generated jsconfig turns
# checkJs off and keeps completion. It says so in the file. Of sensehat's 13
# live artifacts, 8 are fully checked and 5 are completion-only.
```

Both generated files are safe to keep in the artifact directory: `wrap` and
`deploy` ignore everything they do not declare as a code path, so they never
reach the platform. Commit them or gitignore them, as you prefer.

**Widgets are typed too, and from a different source.** A widget's data API is
`$api` — the `opengate-js` package — which publishes its own declarations from
version 16.0.0. og generates nothing for that surface: it writes the globals the
platform injects (`$api`, `$user`, `$moment`, `http`, the navigation helpers) and
the parameters of the async function the platform wraps the code in
(`entityData`, `filters`, `callback`, and for a customTable `page` and
`pageElements`), then points `$api` at the library's own `OpenGateAPI` class.

```bash
og workspace pull <workspace-id> --dir ws/ --org sensehat
cd ws/<workspace>/<dashboard>/<NN>__customchart__<wid>/
npm install          # og wrote a package.json declaring opengate-js
# $api now completes, navigates and documents itself from the library's JSDoc:
#   $api.datapointsSearchBuildr()
#   → Property 'datapointsSearchBuildr' does not exist on type 'OpenGateAPI'.
#     Did you mean 'datapointsSearchBuilder'?
```

Only `customChart` and `customTable` carry a script; every other widget kind is
configured rather than programmed, and `og typegen` says so instead of writing
declarations that would not apply.

In the editor a widget gets completion but not diagnostics: a widget returns its
result at the top level — its contract with the platform — and TypeScript reports
that as an error on a plain file, so checking is turned off. `og widget check`
gets the diagnostics back by rebuilding the wrapper the platform puts around the
code and checking inside it, where the return and the `await` are ordinary:

```bash
og widget check                 # in the widget directory
og widget check <dir> --exit-code   # 1 when there are diagnostics — a CI gate
og widget check --strict        # also show what is normally set aside
```

Positions point at your file; the wrapper's own lines are subtracted. Two idioms
are set aside by default because they are correct JavaScript that TypeScript
rejects on principle — subtracting two `Date`s, and reading a property from a
variable initialised to `null` — and both occur in production widgets. Everything
else is reported, including the one that motivated the command:

```
_widgetConfigCode.js:17:15  TS2551  Property 'datapointsSearchBuildr' does not
exist on type 'OpenGateAPI'. Did you mean 'datapointsSearchBuilder'?
```

which otherwise surfaces as "Data not found" at render time.

### workspace diff

A workspace is a tree — dashboards holding widgets — so its diff is rendered as
one, rather than as the flat key paths the other families use. Dashboards and
widgets are matched by identity, so moving a widget is a move and not two
rewrites, and unchanged branches are pruned to leave the path to what differs:

```bash
og workspace diff ws/multisensor-demo
```

```
~ workspace Multisensor Demo  (_multisensor_demo_ws)
  metadata:
    + color: #ff0000
  ~ dashboard Multisensor Overview (staging)  (multisensor-overview)
    metadata:
      ~ title: Multisensor Overview → Multisensor Overview (staging)
    ~ widget 02__customChart__demo-temp-chart  (demo-temp-chart)
      _widgetConfigCode.js  +1 −1
        - var devices = ['multisensor-001', …];
        + var devices = ['multisensor-999', …];
```

Read remote → local, so it is what deploying would do: `+` would be created,
`−` deleted, `~` changed. Widget JavaScript is compared as a textual diff of the
extracted files — the ones the editor works on — and everything else as a
structural diff of the metadata. `--name-only`, `--against <profile>`,
`--exit-code` and `--context` work as they do for the flat families.

### workspace watch

Deploys on save, with the dashboard as the unit: editing a widget's JavaScript
deploys the dashboard that widget belongs to, not the whole workspace. Edits to
`workspace.json` are reported and skipped — `og workspace deploy` is for those.

```bash
og workspace watch ws/ --dry-run
og workspace watch ws/
```

```
15:34:09  refused  00__multisensor-overview  [conflict]  the remote changed since
you pulled — deploying would discard it. Run `og workspace diff`, then pull or
resolve by hand.
```

The conflict guard needs the snapshot `og workspace pull` records under `.og/`.
A dashboard pulled before that existed, or unwrapped from an export file rather
than fetched, has no snapshot and is reported as unknown rather than silently
overwritten. Same flags as the other families, including the `production: true`
profile guard and `--allow-production`.

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
og jobs search -w "jobStatus eq IN_PROGRESS"
og jobs search -w "operationName eq REBOOT_EQUIPMENT"

# Get report / create / cancel
og jobs get <job-id>
og jobs create -f job.json                     # single batch: up to 100 entities
og jobs cancel <job-id>

# Launch over a large fleet: creates inactive, appends targets in batches, activates last
og jobs launch -f job.json --entities-file meters.txt
og jobs launch -f job.json --entity dev-1 --entity dev-2 --batch 50

# List per-device operations within a job, with their execution steps
og jobs operations <job-id>
og jobs operations <job-id> --all              # every page — a job over a big fleet is paged
og jobs operations <job-id> --page 2 --limit 100

# Operation history: closed operations across jobs, filterable
og jobs history --job <job-id> --all           # read back one job's results
og jobs history -w "operationName eq DIAGNOSIS" --limit 100
og jobs history -w "operationResult eq ERROR" --all      # only the failures
og jobs history -w "entityId eq dev-1" --output json
```

Each operation carries `status` (lifecycle, e.g. `FINISHED`) and `result` (outcome,
e.g. `SUCCESSFUL`), plus `steps[]` where every step has a `name`, a `result`
(`SUCCESSFUL`, `ERROR`, `SKIPPED`, `NOT_EXECUTED`) and a `description`. The table
output compacts the steps into `NAME=OK/ERR/SKIP/NOTRUN`; use `--output json` for
the descriptions, which is where the reason for a failure is written.

`jobs operations` lists one job by id and caps its page size at 1000; `jobs history`
takes a filter and caps at 2000. **A `FINISHED` job does not mean every device
succeeded, and the first page is not the whole job** — pass `--all` when the question
is "did they all work" or "how many failed".

**The history filter field names do not match the response field names.** Identifiers go
unprefixed — `jobId`, `entityId`, `operationId`, `resourceType` — while the rest take an
`operation` prefix: `operationName` (or `operation.name`), `operationStatus`,
`operationResult` (or `operation.result`), `operationDate`, `operationNotify`. Anything
else is rejected with HTTP 400 *"Field in filter unknown"*, including a bare `result` or
`status`, any `operations.` prefix, and `operation.status`. Outcome **is** filterable, so
ask for the failures rather than fetching everything.

Note that `jobs operations` has been observed returning HTTP 204 (no content) for a
job whose operations do exist and are returned by `jobs history`. When results matter,
prefer `og jobs history --job <id>`.

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

# Duplicate names: the platform keys artifacts by identifier, so two rules,
# connector functions, dashboards or workspaces may share a name. pull-all
# disambiguates them by appending a short identifier suffix to the slug of the
# second and later ones — it never writes two artifacts into the same directory.

# wrap refuses a malformed artifact tree rather than deploying a partial one: a
# widget directory with a missing or unparseable widget.json fails the wrap,
# naming the offending directory. Dot-directories (.og/, .git/) are ignored.

# Which fields become .js files is declared per artifact family, not guessed
# from the content: a rule's `javascript`, a connector function's `javascript`,
# a provision function's `scriptProcessor.script`, and the widget code fields
# (_widgetConfigCode, _formatterCode, formatter, ...). A declared field is
# always extracted — even when empty — so the file's path stays put across
# edits. Widget configs additionally keep a content heuristic as a transitional
# fallback: an unlisted field that looks like code is still extracted, with a
# `hint:` on stderr naming it (please report those).
#
# A .js file the family does not declare — your own helper.js, generated
# typings — is left alone on wrap and never deployed as a payload field. It is
# reported with a `hint:` so it is clear it is not being uploaded.

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

# Compare a dashboard directory against the platform — what would deploying change?
og dashboard diff wsroot/<ws>/<dashboard-dir>
og dashboard diff wsroot/<ws>/<dashboard-dir> --name-only
og dashboard diff wsroot/<ws>/<dashboard-dir> --against production   # promotion
og dashboard diff wsroot/<ws>/<dashboard-dir> --exit-code            # CI drift gate

# Read one remote widget code file, raw — the other half of an editor diff
og dashboard show <dashboard-id>                        # list the code files it carries
og dashboard show <dashboard-id> --path 01__customtable__sales/_widgetConfigCode.js
```

`og dashboard diff` is `og workspace diff` narrowed to one dashboard: same tree
renderer, same flags, same identity matching for widgets. Use it when the dashboard
is what you edited and the workspace one when you want the whole tree.

`og dashboard show --path` is the dashboard's answer to `og rules show --path`. The
paths are the ones `og workspace pull` writes — `<widget-dir>/<file>.js` — and the
leading `NN` is the widget's grid position, ignored when matching: a path from a tree
pulled before someone reordered the dashboard still resolves. Where it cannot tell two
widgets apart (same type, neither carrying an id) it reports the path as not found
rather than guessing.

There is no `og widget diff`, `og widget show` or `og widget deploy`, and that is
deliberate: a widget is a grid item, not an artifact the platform can address on its
own, so the dashboard is the smallest unit that can be fetched, compared or deployed —
the same boundary `og workspace watch` draws when a widget edit deploys its dashboard.

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

Start the MCP (Model Context Protocol) server, exposing og operations to an LLM.

```bash
og mcp                            # stdio transport (default → lean mode)
og mcp --http :8080               # HTTP transport (single-tenant: startup profile)
og mcp --http :8080 --multi-tenant # HTTP, per-request credentials from headers
```

**Prerequisites:** run `og login` first to store credentials.

#### Surface modes (context-token footprint)

Every tool definition (name + description + JSON schema) is loaded into the model's
context on **every** request, used or not. Exposing all ~90 tools costs ~14k tokens
per turn. og lets you pick how much surface to expose:

| Mode | Flag | Tools | ~tokens | Use |
|------|------|-------|---------|-----|
| **lean** | `--lean` | 2 (`og_exec`, `og_help`) | ~440 | shell-capable agents (Claude Code, Cursor); the model drives the whole CLI through two tools and discovers it with `og_help` |
| **observe** | `--toolsets observe` | ~18 | ~3.7k | curated read core (devices, alarms, datamodels, timeseries, datasets, jobs) — no mutations |
| **readonly** | `--toolsets readonly` | ~42 | ~7k | every non-mutating tool |
| **groups** | `--toolsets devices,alarms-ops,…` | varies | — | named groups; read = `<resource>`, mutations = `<resource>-write`/`-ops` |
| **all** | `--all-tools` | ~90 | ~14k | the full named surface |

**Per-transport defaults** (when no surface flag is given):

- **stdio → `--lean`** — a shell-capable, trusted, single user. Cheapest, and the model
  has the whole CLI.
- **HTTP / multi-tenant → `--toolsets observe`** — a curated, read-only, governable
  surface for a remote (less-trusted) caller. Widen with `--toolsets` only if needed.

Call the **`og_toolsets`** tool at runtime to list which groups are active and which
can be enabled. In lean mode, destructive subcommands (`delete`/`cancel`) still refuse
to run without `--yes` — pass it in the `og_exec` command only with explicit consent.

> **Changed in v1.7.0:** the stdio default moved from "all tools" to `--lean`, and HTTP
> now defaults to `--toolsets observe` instead of all tools — a large drop in per-turn
> context tokens. To keep the previous behaviour, start the server with `--all-tools`.
> `og mcp install` now bakes `--lean` into the generated config.

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
claude mcp add --scope user opengate -- og mcp --stdio --lean
# add a profile: ... -- og mcp --stdio --lean --profile production
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
      "args": ["mcp", "--stdio", "--lean"]
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
      "args": ["mcp", "--stdio", "--lean"]
    }
  }
}
```

To expose named tools instead of the lean exec surface, replace `--lean` with
`--toolsets observe` (or `--all-tools` for everything).

**Multiple environments** (use `--profile`):

```json
{
  "mcpServers": {
    "opengate-production": {
      "command": "og",
      "args": ["mcp", "--stdio", "--lean", "--profile", "production"]
    },
    "opengate-staging": {
      "command": "og",
      "args": ["mcp", "--stdio", "--lean", "--profile", "staging"]
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
| `dashboards_code` | Read widget JavaScript one file at a time: list them, or return one by path |
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

## Use as a Go library

Besides the CLI, TUI and MCP server, `og` exposes its OpenGate client as an
importable library. `pkg/opengate` is the API client and `pkg/query` the search
filter builder; everything under `internal/` is CLI-private.

```bash
go get github.com/carlosprados/og-cli/v2/pkg/opengate
```

> The module path carries `/v2`. Version 2 broke the library API deliberately — every
> I/O method now takes a `context.Context` first, `New` accepts options, and the South
> API calls became client methods. Code on v1 keeps working on v1; it does not compile
> against v2 without those changes.

```go
import (
    "github.com/carlosprados/og-cli/v2/pkg/opengate"
    "github.com/carlosprados/og-cli/v2/pkg/query"
)

// Options configure this client only — nothing is process-wide.
c := opengate.New("https://api.opengate.es", jwtToken,
    opengate.WithHTTPClient(&http.Client{Timeout: 15 * time.Second}),
    opengate.WithTLS(false, "/etc/ssl/extra-ca.pem"), // --insecure / --ca-file equivalents
    opengate.WithAPIVersion("v80"),                   // for an on-premises instance on another version
)
if err := c.Err(); err != nil {
    return err // e.g. an unreadable ca-file: New defers the error instead of panicking
}

filter, err := query.BuildFilter(query.SearchParams{
    Conditions: []query.Condition{{Field: "provision.device.identifier", Op: "eq", Value: "dev-1"}},
    Limit:      100,
})

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
devices, err := c.SearchDevices(ctx, filter)
```

Searches are paged. Walk them with the iterator rather than asking for
everything at once:

```go
for dev, err := range c.SearchDevicesAll(ctx, filter) {
    if err != nil {
        return err // includes ctx cancellation, so a partial walk is never silent
    }
    process(dev)
}
```

`SearchDevicesAll` requests `limit.size` from your filter (or
`opengate.DefaultPageSize`) and stops at the last page. For manual control use
`SearchDevicesPage(ctx, filter, page, size)`; `page` counts from **1** and is a
page number, not an element offset.

Launching an operation over a fleet larger than one target list (100 entities) uses
the batched pattern, with activation merged into the last batch so the job is never
active with a partial target:

```go
res, err := c.LaunchJob(ctx, opengate.JobRequest{
    Name:     "DIAGNOSIS",
    Callback: "https://my-service/jobs/callback",
    Notify:   opengate.Ptr(true),
    Schedule: &opengate.JobSchedule{
        Stop:       &opengate.JobScheduleTime{Delayed: 21_600_000}, // 6 h window
        Scattering: opengate.DefaultScattering(),                   // maxSpread 80
    },
    OperationParameters: &opengate.OperationParams{
        Timeout: 60_000, AckTimeout: 5_000, Retries: opengate.Ptr(0),
    },
}, meterIDs, opengate.LaunchOptions{
    OnProgress: func(p opengate.LaunchProgress) {
        // The id arrives before the first batch, so it can be persisted and resumed.
        log.Printf("job %s: %d/%d", p.JobID, p.Appended, p.Total)
    },
})
if !res.Rejected.Empty() {
    // notProvisioned / notAllowed / duplicated arrive inside SUCCESSFUL responses:
    // ignore them and the job silently covers fewer devices than requested.
}
```

`scattering.maxSpread` is a percentage of the job's window and **must never be 100** —
the last operations would be launched exactly as the window expires and never run, since
the platform does not measure durations to reserve that margin. The library refuses
anything above `MaxScatteringSpread` (90); `DefaultScattering()` uses 80.

For a service making thousands of calls against someone else's instance, enable
retries — they are **off by default**, because a library must not silently multiply
a caller's writes:

```go
c := opengate.New(host, "", 
    opengate.WithAPIKey(key),
    opengate.WithRetry(opengate.DefaultRetryPolicy()), // 3 attempts, backoff + jitter
)
```

The policy honours `Retry-After` when the server sends one, and applies full jitter
so a fleet of throttled clients does not come back in lockstep. What it retries:

| Outcome | Retried? |
|---|---|
| HTTP 429 | always — a rate-limited request was never processed |
| 5xx or a transport error, on `GET`/`PUT`/`DELETE` | yes |
| 5xx on a **search** `POST` (`/search/…`, history, summaries) | yes — those change nothing |
| 5xx on a mutating `POST` (create job, create task, login, alarm action) | **no** |
| any other 4xx | no |

That last distinction is the point: a 500 does not tell you whether the server acted
before failing, so repeating a job creation can launch the operation twice, and an
operation already sent to a device cannot be recalled. Set
`RetryPolicy.RetryNonIdempotent` if you accept that risk. Bulk provisioning uploads
are never retried.

Two more things worth knowing for long-running services:

- **Every I/O method takes a `context.Context` first**, propagated down to the
  HTTP request, so cancellation and per-request deadlines work end to end.
- **Prefer an API key over a JWT for service accounts.** A JWT obtained at
  start-up may be dead by the time a deferred phase runs hours later; an API key
  does not expire. `opengate.WithAPIKey(key)` switches North API calls to the
  `X-ApiKey` header. It never applies to the Web API, which has its own token.

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
