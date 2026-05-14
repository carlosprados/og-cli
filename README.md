# og — OpenGate CLI

Unofficial command-line interface for the [OpenGate](https://opengate.es) IoT platform REST API. See the [official OpenGate documentation](https://documentation.opengate.es) for API reference.

Three modes of operation:

| Mode | Invocation | Description |
|------|------------|-------------|
| **Interactive** | `og` | TUI with Bubble Tea — browse, search, and manage resources visually |
| **CLI** | `og <command>` | Direct commands for scripts and one-liners |
| **MCP** | `og mcp` | Model Context Protocol server for LLM integration |

All three interfaces expose the same functionality through the same client library.

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
| **Devices** | List → enter for tabbed detail view | enter, o, r |
| **Device detail** | Three tabs: Overview (cards), Datastreams (table), JSON (scrollable) | 1/2/3, tab |
| **Alarms** | List with severity/status → attend or close | a, c, r |
| **Time Series** | List → enter to browse collected data | enter, r |
| **Datasets** | List → enter to browse data | enter, r |
| **Jobs** | List → enter for job detail with per-device operations | enter, r |

From the **Devices** screen, press `o` on a device to launch an operation (REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC).

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

## CLI commands

### Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Custom config file path |
| `--profile` | | Config profile to use |
| `--org` | | Organization name |
| `--output` | `-o` | Output format: `json` or `table` (default: `table`) |

### login

Authenticate against OpenGate and store JWT token, API key, and organization in the active profile.

```bash
og login -e user@example.com
og login -e user@example.com -p mypassword
og login -e user@example.com --profile staging
```

The password is prompted securely if not provided. The API key (needed for IoT data collection) is obtained automatically from the login response.

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

# Select specific fields (dynamic columns)
og dev search -s provision.device.identifier -s wt -s wp \
              -w "provision.device.identifier like sense"

# Get
og dev get sense-001
og dev get sense-001 -o json

# CRUD
og dev create --org sensehat -f device.json
og dev update sense-001 --org sensehat -f device.json
og dev delete sense-001 --org sensehat
```

**Select** (`-s`): choose which datastreams/fields to return. Without `-s`, default columns are Identifier, Name, Organization, and State.

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
og tasks get <task-id>
og tasks create -f task.json
og tasks cancel <task-id>
og tasks jobs <task-id>
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
og workspace unwrap <workspace-id> --dir wsroot/
og workspace unwrap-all --dir wsroot/
og workspace unwrap-file ws.json --dir wsroot/      # from a local JSON file

# All unwrap commands accept --force to overwrite an existing destination

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

# Deploy a single dashboard directory in one step (wrap + import)
og dashboard deploy wsroot/<ws>/<dashboard-dir>
og dashboard deploy wsroot/<ws>/<dashboard-dir> --update
og dashboard deploy wsroot/<ws>/<dashboard-dir> --workspace <other-ws-id>
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
og workspace export ws-id -o ws.json
og dashboard export dash-id -o dash.json

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
```

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
og mcp              # stdio transport (default)
og mcp --http :8080 # HTTP transport
```

**Prerequisites:** run `og login` first to store credentials.

Configuration for MCP clients:

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

For a detailed guide on how prompts, resources, and tools work together, see [doc/mcp-integration.md](doc/mcp-integration.md).

#### Tools

| Tool | Description |
|------|-------------|
| `login` | Authenticate with email/password |
| `devices_search` | Search devices with query/filter/select |
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
| `tasks_search` | Search operation tasks |
| `tasks_get` | Get task detail |
| `tasks_create` | Create operation task |
| `tasks_cancel` | Cancel a task |
| `iot_collect` | Send a single data point to a device |
| `iot_collect_payload` | Send a full IoT payload to a device |
| `workspaces_list` | List workspaces (optionally with embedded dashboards) |
| `workspaces_get` | Get a workspace by ID |
| `workspaces_export` | Export a workspace via `/workspaces/export/{id}` |
| `workspaces_import` | Create a workspace from JSON payload |
| `workspaces_update` | Update a workspace |
| `workspaces_delete` | Delete a workspace |
| `dashboards_list` | List dashboards (all, or filtered by workspace) |
| `dashboards_get` | Get a dashboard with grid layout and widgets |
| `dashboards_export` | Export a dashboard via `/dashboards/export/{id}` |
| `dashboards_import` | Create a dashboard from JSON payload, optionally overriding target workspace |
| `dashboards_update` | Update a dashboard |
| `dashboards_delete` | Delete a dashboard |

Search tools accept a `query` parameter with the same syntax as `-w` flags:

```
devices_search(
  query: "provision.device.administrativeState eq ACTIVE AND provision.device.identifier like sense",
  select: "provision.device.identifier,wt",
  limit: 10
)

# Filtering by the latest collected datastream value is also supported
devices_search(query: "wt gt 20")
devices_search(query: "device.temperature.value gt 50 AND provision.device.operationalStatus eq NORMAL")
```

#### Prompts

| Prompt | Description |
|--------|-------------|
| `opengate-guide` | Complete guide covering all tools, query syntax with operator mapping (ES/EN → eq/like/gt/...), fields per entity, job creation format, IoT data collection, and worked examples. See [doc/mcp-prompts.md](doc/mcp-prompts.md) for full content. |

#### Resources

| Resource | Description |
|----------|-------------|
| `opengate://query-syntax` | Static reference of query operators, fields per entity, and job operation types |
| `opengate://organizations/{org}/datamodel-fields` | Dynamic: lists all datastream fields available in an organization's datamodels, fetched live from the API |

### version

```bash
og version
```

## Documentation

| Document | Description |
|----------|-------------|
| [doc/mcp-integration.md](doc/mcp-integration.md) | MCP architecture: how prompts, resources, and tools work together |
| [doc/mcp-prompts.md](doc/mcp-prompts.md) | Full content of MCP prompts with explanation of each section |
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
