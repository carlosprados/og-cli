# og — OpenGate CLI

Command-line interface for the [OpenGate](https://opengate.es) IoT platform REST API by Amplía Soluciones.

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
- **Menu** — login, datamodels, devices
- **Login** — email/password form, stores token on success
- **Datamodels** — searchable list → detail with all categories and datastreams
- **Devices** — searchable list → detail with full JSON

## CLI commands

### Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | | Custom config file path |
| `--profile` | | Config profile to use |
| `--org` | | Organization name |
| `--output` | `-o` | Output format: `json` or `table` (default: `table`) |

### login

Authenticate against OpenGate and store the JWT token in the active profile.

```bash
og login -e user@example.com
og login -e user@example.com -p mypassword
og login -e user@example.com --profile staging
```

The password is prompted securely if not provided.

### datamodels (alias: dm)

Manage OpenGate data models.

#### search

```bash
# List all data models
og dm search

# Filter by identifier
og dm search -w "datamodels.identifier like weather"

# Filter by organization
og dm search -w "datamodels.organizationName eq sensehat"

# Combine filters (AND)
og dm search -w "datamodels.identifier like teliot" -w "datamodels.organizationName eq myorg"

# Limit results
og dm search --limit 5

# Raw JSON filter (for OR, nested queries)
og dm search --filter '{"filter":{"or":[{"eq":{"datamodels.identifier":"weather"}},{"eq":{"datamodels.identifier":"bts"}}]}}'

# JSON output
og dm search -o json
```

#### get

Show all categories and datastreams of a data model.

```bash
og dm get weather --org sensehat
```

```
Category  Datastream  Name         Period   Schema  Access
weather   wt          Temperature  INSTANT  number  READ
weather   wp          Pressure     INSTANT  number  READ
```

```bash
og dm get weather --org sensehat -o json
```

#### create / update / delete

```bash
og dm create --org sensehat -f datamodel.json
og dm update weather --org sensehat -f datamodel.json
og dm delete weather --org sensehat
```

### devices (alias: dev)

Manage OpenGate devices.

#### search

```bash
# List all devices
og dev search

# Filter by state
og dev search -w "provision.device.administrativeState eq ACTIVE"

# Filter by identifier (partial match)
og dev search -w "provision.device.identifier like sense"

# Combine filters (AND)
og dev search -w "provision.device.administrativeState eq TESTING" \
              -w "provision.device.identifier like 865"

# Limit results
og dev search --limit 10

# Select specific fields (dynamic columns)
og dev search -s provision.device.identifier -s provision.device.administrativeState
og dev search -s provision.device.identifier -s wt -s wp \
              -w "provision.device.identifier like sense"

# Raw JSON filter (for OR, nested queries)
og dev search --filter '{"filter":{"or":[...]}}'

# JSON output
og dev search -o json
```

**Filter operators** (`-w`): `eq`, `neq`, `like`, `gt`, `lt`, `gte`, `lte`, `in`, `exists`

Multiple `-w` flags are combined with AND. For OR or nested queries, use `--filter` with raw JSON.

**Select** (`-s`): choose which datastreams/fields to return. Without `-s`, the default columns are Identifier, Name, Organization, and State.

#### get

```bash
og dev get sense-001 --org sensehat
og dev get sense-001 --org sensehat -o json
```

#### create / update / delete

```bash
og dev create --org sensehat -f device.json
og dev update sense-001 --org sensehat -f device.json
og dev delete sense-001 --org sensehat
```

### alarms (alias: al)

Monitor and manage OpenGate alarms.

#### search

```bash
# List all alarms
og alarms search

# Filter by severity
og alarms search -w "alarm.severity eq CRITICAL"

# Open and urgent alarms
og alarms search -w "alarm.status eq OPEN" -w "alarm.severity eq URGENT"

# Alarms for a specific device
og alarms search -w "alarm.entityIdentifier like sense" --limit 10

# JSON output
og alarms search -o json
```

**Alarm filter fields**: `alarm.severity` (INFORMATIVE, URGENT, CRITICAL), `alarm.status` (OPEN, ATTEND, CLOSED), `alarm.name`, `alarm.rule`, `alarm.entityIdentifier`, `alarm.organization`, `alarm.channel`, `alarm.priority` (LOW, MEDIUM, HIGH), `alarm.openingDate`.

#### summary

```bash
# Overall summary (counts by severity, status, rule, name)
og alarms summary

# Summary of open alarms only
og alarms summary -w "alarm.status eq OPEN"
```

#### attend / close

```bash
og alarms attend <alarm-uuid> --notes "Investigating"
og alarms attend <uuid1> <uuid2> <uuid3>
og alarms close <alarm-uuid> --notes "Resolved"
```

### iot

Device integration via the South API (X-ApiKey authentication). The API key is obtained automatically from the login response.

#### collect

Send a single value to a device datastream:

```bash
og iot collect sense-001 wt 25.3
og iot collect sense-001 wp 1013
og iot collect sense-001 mystream "hello world"
```

#### collect-file

Send a full IoT payload from a JSON file:

```bash
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
# stdio transport (for direct LLM integration, e.g. Claude Code)
og mcp

# HTTP transport
og mcp --http :8080
```

Configuration for MCP clients (Claude Code, LM Studio, etc.):

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

#### Tools

| Tool | Description |
|------|-------------|
| `login` | Authenticate with email/password |
| `devices_search` | Search devices with query/filter |
| `devices_get` | Get device detail |
| `devices_create` | Create device from JSON |
| `devices_update` | Update device from JSON |
| `devices_delete` | Delete device |
| `datamodels_search` | Search data models with query/filter |
| `datamodels_get` | Get data model detail |
| `datamodels_create` | Create data model from JSON |
| `datamodels_update` | Update data model from JSON |
| `datamodels_delete` | Delete data model |
| `alarms_search` | Search alarms with query/filter |
| `alarms_summary` | Alarm counts by severity/status/rule |
| `alarms_attend` | Mark alarms as attended |
| `alarms_close` | Close alarms |
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
| `opengate-guide` | Teaches the LLM how to use the OpenGate tools: query syntax, operator mapping (natural language → eq/like/gt/...), common field names per resource, and worked examples in English and Spanish |

LLMs that load this prompt can interpret natural language like "Give me active devices" or "Dispositivos cuyo estado sea ACTIVE" and translate it to the correct tool call.

#### Resources

| Resource | Description |
|----------|-------------|
| `opengate://query-syntax` | Static reference of query operators and common fields |
| `opengate://organizations/{org}/datamodel-fields` | Dynamic: lists all datastream fields available in an organization's datamodels, fetched live from the API |

The dynamic resource lets the LLM discover which custom datastreams (e.g. `wt`, `wp`, `batteryPercentage`) exist in a given organization, so it can use them in `select` and `query` parameters without the user having to remember field names.

### version

```bash
og version
```

## License

Proprietary — Amplía Soluciones.
