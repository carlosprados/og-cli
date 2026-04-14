# MCP Integration Guide

This document explains how `og` exposes the OpenGate IoT platform to LLMs via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/).

## Overview

When you run `og mcp --stdio`, the binary starts an MCP server that exposes three types of primitives:

| Primitive | Purpose | How the LLM uses it |
|-----------|---------|---------------------|
| **Tools** | Actions the LLM can execute | Called directly to search, create, update, delete resources |
| **Prompts** | Pre-built instructions the LLM can load | Loaded at the start of a conversation to understand how to use the tools |
| **Resources** | Data the LLM can read on demand | Read when the LLM needs to discover dynamic information (e.g. available fields) |

## Configuration

### Claude Code / Claude Desktop

Add to your MCP settings (`~/.claude/settings.json` or project `.mcp.json`):

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

### LM Studio

In LM Studio's MCP server configuration:

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

### Any MCP-compatible client

The pattern is the same: point the client at the `og` binary with `mcp --stdio` as arguments. The server communicates over stdin/stdout using JSON-RPC.

### Using a specific profile

If you have multiple OpenGate environments, pass `--profile`:

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

### Prerequisites

Before the MCP server can work, you need a valid session:

```bash
og login -e user@example.com -p password
```

This stores the JWT token and API key in `~/.og/config.yaml`. The MCP server reads them from there.

## Prompts

### `opengate-guide`

This is the main prompt. When an LLM loads it, it receives a complete guide that teaches it:

1. **Query syntax** — how to build filter expressions (`"field op value"`) instead of raw JSON
2. **Operator mapping** — natural language (Spanish and English) to operators:
   - "is" / "sea" / "igual a" → `eq`
   - "contains" / "contiene" → `like`
   - "greater than" / "mayor que" → `gt`
   - etc.
3. **Entity mapping** — which tool to call for each concept:
   - "device" / "dispositivo" → `devices_search`
   - "alarm" / "alarma" → `alarms_search`
   - "launch operation" / "lanzar operación" → `jobs_create`
4. **Field reference** — filterable fields for each entity type with their possible values
5. **Job creation format** — exact JSON structure for creating operation jobs
6. **IoT data collection** — how to send data to devices via the South API
7. **Worked examples** — 28 examples covering every tool in both Spanish and English

**How it works in practice:**

Without the prompt, an LLM might try to call `devices_search(filter: '{"filter":{"eq":{"provision.device.administrativeState":"ACTIVE"}}}')` — constructing raw JSON that's error-prone.

With the prompt loaded, the same request becomes `devices_search(query: "provision.device.administrativeState eq ACTIVE")` — the LLM uses the simple query syntax, which is parsed by `og` into the correct JSON internally.

**How the LLM loads it:**

MCP clients typically offer prompts to the user or auto-load them. In Claude Code, the LLM has access to the prompt automatically. In LM Studio, you may need to reference it in your system prompt or the client will list it as an available prompt.

## Resources

### `opengate://query-syntax`

A static text resource containing:
- All query operators with descriptions
- Filterable fields for every entity type (devices, datamodels, alarms, jobs, tasks)
- Available job operation types (REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC)

This duplicates some of the prompt content but is available as a separate resource that the LLM can read at any time during a conversation, not just at the start.

### `opengate://organizations/{org}/datamodel-fields`

A dynamic resource that queries the OpenGate API and returns all datastream fields available in a specific organization.

Example: reading `opengate://organizations/sensehat/datamodel-fields` returns:

```
Available datastream fields for organization: sensehat
==========================================================

Datamodel: weather (v1.0) — Weather Station
  wt                                                Temperature
  wp                                                Pressure

Datamodel: provisionDevice (v8.0.0.0) — provisionDevice
  provision.device.identifier                       Prov. Identifier
  provision.device.name                             Prov. Name
  ...
```

This is critical for LLMs because OpenGate datastream names are dynamic — they come from data models, not from a fixed schema. Without this resource, the LLM would need the user to tell it which fields exist. With it, the LLM can discover them autonomously.

**When the LLM should read it:**
- When the user asks about specific measurements (temperature, pressure, etc.)
- When building a `select` clause for `devices_search`
- When it needs to know which datastream ID to use with `iot_collect`

## Tools

### Search tools

All search tools (`devices_search`, `datamodels_search`, `alarms_search`, `jobs_search`, `tasks_search`) share the same parameter pattern:

| Parameter | Description |
|-----------|-------------|
| `query` | Simple filter: `"field op value AND field op value"`. Preferred. |
| `limit` | Max number of results |
| `filter` | Raw OpenGate JSON filter. Only for OR/nested queries that `query` cannot express. |

`devices_search` also has:
| Parameter | Description |
|-----------|-------------|
| `select` | Comma-separated fields to return (e.g. `"provision.device.identifier,wt,wp"`) |

### CRUD tools

Resources that support CRUD (devices, datamodels, time series, datasets) follow this pattern:

- `*_get(organization, id)` — read a single resource
- `*_create(organization, body)` — create from JSON
- `*_update(organization, id, body)` — update from JSON
- `*_delete(organization, id)` — delete

### Action tools

| Tool | Parameters | Description |
|------|-----------|-------------|
| `alarms_attend` | `ids`, `notes` | Mark alarms as attended |
| `alarms_close` | `ids`, `notes` | Close alarms |
| `jobs_create` | `body` | Create and launch an operation job |
| `jobs_cancel` | `id` | Cancel a running job |
| `jobs_operations` | `id` | List per-device operations within a job |
| `iot_collect` | `device_id`, `datastream_id`, `value` | Send a data point |
| `iot_collect_payload` | `device_id`, `payload` | Send a full IoT payload |

## Architecture

```
LLM Client (Claude, LM Studio, etc.)
    │
    │ MCP protocol (JSON-RPC over stdio)
    │
    ▼
og mcp --stdio
    │
    ├── Prompts    → opengate-guide (query syntax, entity mapping, examples)
    ├── Resources  → opengate://query-syntax, opengate://organizations/{org}/datamodel-fields
    └── Tools      → 30+ tools calling internal/client methods
                          │
                          ▼
                    OpenGate REST API
                    (North API: JWT auth)
                    (South API: X-ApiKey auth)
```

All three MCP primitives (prompts, resources, tools) work together: the prompt teaches the LLM how to use the tools, the resources provide dynamic data discovery, and the tools execute the actual operations.
