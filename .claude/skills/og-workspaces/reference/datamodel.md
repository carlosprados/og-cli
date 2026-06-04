# Reference: Datamodel (with Datastreams)

A **Datamodel** defines the schema of the data collected by entities (devices, assets, tickets) in OpenGate. It is composed of **categories** (logical groupings) and **datastreams** (individual data fields with type, access, storage and unit metadata).

Datamodels are managed via the `og dm` CLI commands and are also browsable in the platform UI via the `datamodelBrowser` widget.

Full documentation here: https://documentation.opengate.es/2-howtos/3-data-modelling/index.html

---

## JSON Schema

Full datamodel payload (used for `og dm create` and `og dm update`):

```json
{
    "identifier": "my-datamodel",
    "name": "My Datamodel Name",
    "version": "1.0",
    "description": "Human-readable description of the datamodel.",
    "allowedResourceTypes": [
        "entity.device",
        "entity.asset",
        "ticket"
    ],
    "categories": [
        {
            "identifier": "category-id",
            "name": "Category Display Name",
            "datastreams": [
                {
                    "identifier": "my.custom.field",
                    "name": "Field Display Name",
                    "description": "What this datastream measures.",
                    "period": "INSTANT",
                    "access": "READ",
                    "schema": {
                        "type": "number"
                    },
                    "storage": {
                        "period": "DAYS",
                        "total": 30
                    },
                    "tags": ["optional-tag"],
                    "unit": {
                        "label": "ºC",
                        "symbol": "º",
                        "type": "grados"
                    },
                    "icon": {
                        "class": "fa-sort-numeric-asc"
                    },
                    "modifiable": true,
                    "calculated": false,
                    "required": true,
                    "indexed": false
                }
            ]
        }
    ]
}
```

> ⚠️ **`organization` is NOT a body field.** Pass it via `--org <org>` in the CLI command. Including it in the JSON body will cause a 400 error.

> ⚠️ **Reserved namespaces.** Datastream identifiers starting with `device.`, `provision.device.`, and other system-reserved prefixes are rejected by the API. Use custom namespaces like `sensor.temperature`, `cpu.usage`, or plain identifiers like `status`.

---

## Datamodel Root Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `identifier` | `string` | ✅ | Unique identifier for the datamodel (kebab-case recommended). |
| `name` | `string` | ✅ | Display name. |
| `version` | `string` | ✅ | Version string (e.g. `"1.0"`, `"0.1"`). |
| `description` | `string` | — | Human-readable description. |
| `allowedResourceTypes` | `array` | ✅ | Resource types this datamodel applies to. Valid values: `entity.device`, `entity.asset`, `ticket`. |
| `categories` | `array` | ✅ | List of category objects (see below). |

> `organization` is passed via `--org` CLI flag, NOT in the JSON body.

---

## Category Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `identifier` | `string` | ✅ | Unique category ID within the datamodel. |
| `name` | `string` | ✅ | Display name for the category. |
| `datastreams` | `array` | ✅ | List of datastream objects (see below). |

---

## Datastream Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `identifier` | `string` | ✅ | Unique datastream ID. Can use dot notation for nested paths (e.g. `provision.temperature.value`). |
| `name` | `string` | ✅ | Display name. |
| `description` | `string` | — | Human-readable description. |
| `period` | `string` | ✅ | Collection period. Valid values: `INSTANT`, `CUMULATIVE`, `PULSE`. |
| `access` | `string` | ✅ | Access mode. Valid values: `READ`, `WRITE`, `READ,WRITE` (note: comma, not underscore). |
| `schema` | `object` | ✅ | JSON Schema object defining the data type (see Schema Types below). |
| `storage` | `object` | ✅ | Storage configuration (see Storage below). `total` is required when `period` is not `NEVER`. |
| `tags` | `array` | — | Optional string tags for categorisation. |
| `unit` | `object` | — | Unit metadata: `label`, `symbol`, `type`. |
| `icon` | `object` | — | Icon config: `class` (FontAwesome class, e.g. `fa-thermometer`). |
| `modifiable` | `boolean` | — | Whether the value can be written via the platform UI. Default `true`. |
| `calculated` | `boolean` | — | Whether this is a calculated/derived field. Default `false`. |
| `required` | `boolean` | — | Whether the field is mandatory for entities using this datamodel. Default `false`. |
| `indexed` | `boolean` | — | Whether the field is indexed for faster queries. Default `false`. |

---

## Schema Types

```json
{ "type": "number" }
{ "type": "string" }
{ "type": "boolean" }
{ "type": "integer" }
{ "type": "array",  "items": { "type": "string" } }
{ "type": "object" }
```

---

## Storage

Valid `period` values (API-verified): `SECONDS`, `MINUTES`, `HOURS`, `DAYS`, `MONTHS`, `YEARS`, `NEVER`.

```json
{ "period": "NEVER" }              // Not stored — no total needed
{ "period": "DAYS",  "total": 30 } // Store for 30 days — total required
{ "period": "MONTHS", "total": 6 } // Store for 6 months
{ "period": "YEARS",  "total": 1 } // Store for 1 year
```

> ⚠️ `total` is **required** whenever `period` is anything other than `NEVER`.

---

## CLI Commands (`og dm` / `og datamodels`)

```bash
# Search datamodels
og dm search
og dm search -w "datamodels.identifier like weather"
og dm search -w "datamodels.organizationName eq myorg" --limit 5

# Get a specific datamodel (shows categories and datastreams)
og dm get <identifier> --org <org>
og dm get weather --org sensehat -o json

# Create from JSON file
og dm create --org <org> -f datamodel.json

# Update existing
og dm update <identifier> --org <org> -f datamodel.json

# Delete
og dm delete <identifier> --org <org>
```

**Example `og dm get` output:**
```
Category   Datastream  Name         Period    Schema  Access
weather    wt          Temperature  INSTANT   number  READ
weather    wp          Pressure     INSTANT   number  READ
```

---

## Verified Working Example

Deployed live to OpenGate v13.1.0 via `og dm create --org carmencorp -f datamodel.json`:

```json
{
    "identifier": "test-sensor-antigravity",
    "name": "Test Sensor Antigravity",
    "version": "1.0",
    "description": "Datamodel de prueba verificado en OpenGate v13.1.0.",
    "allowedResourceTypes": ["entity.device"],
    "categories": [
        {
            "identifier": "environment",
            "name": "Environment",
            "datastreams": [
                {
                    "identifier": "sensor.temperature",
                    "name": "Temperature",
                    "description": "Ambient temperature in Celsius.",
                    "period": "INSTANT",
                    "access": "READ",
                    "schema": { "type": "number" },
                    "storage": { "period": "DAYS", "total": 30 },
                    "unit": { "label": "Celsius", "symbol": "ºC", "type": "temperature" },
                    "icon": { "class": "fa-thermometer" },
                    "modifiable": false,
                    "calculated": false,
                    "required": true,
                    "indexed": false
                },
                {
                    "identifier": "sensor.humidity",
                    "name": "Humidity",
                    "description": "Relative humidity percentage.",
                    "period": "INSTANT",
                    "access": "READ",
                    "schema": { "type": "number" },
                    "storage": { "period": "DAYS", "total": 30 },
                    "unit": { "label": "Percent", "symbol": "%", "type": "percentage" },
                    "icon": { "class": "fa-tint" },
                    "modifiable": false,
                    "calculated": false,
                    "required": false,
                    "indexed": false
                }
            ]
        },
        {
            "identifier": "system",
            "name": "System",
            "datastreams": [
                {
                    "identifier": "cpu.usage",
                    "name": "CPU Usage",
                    "description": "CPU usage percentage.",
                    "period": "INSTANT",
                    "access": "READ",
                    "schema": { "type": "number" },
                    "storage": { "period": "DAYS", "total": 30 },
                    "unit": { "label": "Percent", "symbol": "%", "type": "percentage" },
                    "icon": { "class": "fa-microchip" },
                    "modifiable": false,
                    "calculated": false,
                    "required": false,
                    "indexed": false
                },
                {
                    "identifier": "status.message",
                    "name": "Status Message",
                    "description": "Free-text status from the device.",
                    "period": "CUMULATIVE",
                    "access": "READ,WRITE",
                    "schema": { "type": "string" },
                    "storage": { "period": "NEVER" },
                    "unit": { "label": "", "symbol": "", "type": "" },
                    "icon": {},
                    "modifiable": true,
                    "calculated": false,
                    "required": false,
                    "indexed": false
                }
            ]
        }
    ]
}
```

**CLI output after create:**
```
Category     Datastream          Name            Period      Schema  Access
environment  sensor.temperature  Temperature     INSTANT     number  READ
environment  sensor.humidity     Humidity        INSTANT     number  READ
system       cpu.usage           CPU Usage       INSTANT     number  READ
system       status.message      Status Message  CUMULATIVE  string  READ,WRITE
```

---

## Source

- `recursos/Data model wizard.json` — verified example from OpenGate platform export.
- `README.md` — `og dm` CLI command reference.
