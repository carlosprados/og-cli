# og — OpenGate CLI

Unofficial command-line interface for the [OpenGate](https://opengate.es) IoT platform REST API.

> **Disclaimer:** This software is NOT an official product of [Amplía Soluciones](https://amplia-iiot.com). It is provided "as is", without warranty of any kind, express or implied. Use it at your own risk. For official documentation, visit [documentation.opengate.es](https://documentation.opengate.es).

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

Search tools accept a `query` parameter with the same syntax as `-w` flags:

```
devices_search(
  query: "provision.device.administrativeState eq ACTIVE AND provision.device.identifier like sense",
  select: "provision.device.identifier,wt",
  limit: 10
)
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

## License

Apache License 2.0 — see [LICENSE](LICENSE).
