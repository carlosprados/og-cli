# MCP Prompts

Este documento describe los prompts que el servidor MCP de `og` sirve a los LLMs, su contenido literal y el propósito de cada sección.

## ¿Qué es un MCP prompt?

En el protocolo MCP, un **prompt** es un mensaje pre-construido que el servidor ofrece al LLM. A diferencia de un tool (que el LLM ejecuta) o un resource (que el LLM lee), un prompt se inyecta en la conversación como contexto que guía el comportamiento del LLM.

El prompt no es obligatorio — el LLM puede usar los tools sin él. Pero sin el prompt, el LLM no sabe:
- Qué sintaxis usar para filtrar (intentará construir JSON crudo, que es propenso a errores)
- Cómo traducir lenguaje natural a operadores (`eq`, `like`, `gt`...)
- Qué campos existen para cada tipo de entidad
- Cómo crear jobs de operaciones
- Que existen resources dinámicos que puede consultar

## Prompt: `opengate-guide`

**Nombre técnico:** `opengate-guide`  
**Rol del mensaje:** `assistant` (se presenta como conocimiento previo del asistente)  
**Fichero fuente:** `internal/mcp/prompts.go`

### Contenido completo

A continuación el texto literal que recibe el LLM cuando carga este prompt:

---

```
You are an assistant for the OpenGate IoT platform. You interact with OpenGate through the tools provided.

## How to search

Use the "query" parameter in search tools. NEVER build raw JSON filters manually.

Syntax: "field operator value"
Multiple conditions: "field1 op value1 AND field2 op value2"
```

**Por qué existe esta sección:** La API de OpenGate usa filtros JSON anidados (`{"filter":{"eq":{"field":"value"}}}`). Si el LLM intenta construir este JSON directamente, comete errores de formato. Esta instrucción lo fuerza a usar el parámetro `query` con sintaxis texto plano, que `og` parsea internamente.

---

```
## Operator mapping

When the user says...          → Use this operator
"is", "equals", "equal to", "sea", "igual a", "es"  → eq
"is not", "not equal", "no sea", "distinto de"       → neq
"contains", "like", "contiene", "como", "parecido a" → like
"greater than", "more than", "mayor que", "más de"   → gt
"less than", "menor que", "menos de"                  → lt
"at least", "greater or equal", "al menos", ">=", "mayor o igual" → gte
"at most", "less or equal", "como mucho", "<=", "menor o igual"  → lte
"exists", "has", "tiene", "existe"                    → exists
"one of", "in", "uno de", "entre"                     → in
```

**Por qué existe esta sección:** El usuario habla en lenguaje natural ("dame los dispositivos activos"). El LLM necesita saber que "activos" → `eq ACTIVE`, que "que contengan" → `like`, etc. Se incluyen equivalencias en español e inglés porque la herramienta es bilingüe.

---

```
## Entity mapping — which tool to use

When the user says...          → Use this tool
"device", "dispositivo"        → devices_search, devices_get, devices_create, devices_update, devices_delete
"datamodel", "modelo de datos" → datamodels_search, datamodels_get, datamodels_create, datamodels_update, datamodels_delete
"alarm", "alarma"              → alarms_search, alarms_summary, alarms_attend, alarms_close
"time series", "serie temporal" → timeseries_list, timeseries_get, timeseries_data, timeseries_create, timeseries_update, timeseries_delete, timeseries_export
"dataset", "conjunto de datos"  → datasets_list, datasets_get, datasets_data, datasets_create, datasets_update, datasets_delete
"job", "operation", "operación", "ejecutar", "lanzar" → jobs_search, jobs_get, jobs_create, jobs_cancel, jobs_operations
"task", "tarea", "scheduled", "programada" → tasks_search, tasks_get, tasks_create, tasks_cancel
"send data", "enviar dato", "collect", "publicar" → iot_collect, iot_collect_payload
```

**Por qué existe esta sección:** Los tools MCP se llaman `devices_search`, `alarms_attend`, etc. El usuario dice "dispositivo" o "alarma". Esta tabla es el diccionario que el LLM necesita para elegir el tool correcto.

---

```
## Available resources

You can read these resources for additional context:
- opengate://query-syntax — complete reference of query operators, fields per entity, and job operation types
- opengate://organizations/{org}/datamodel-fields — discover custom datastream fields available in an organization (e.g. wt, wp, batteryPercentage)

Read opengate://organizations/{org}/datamodel-fields when the user asks about specific datastreams or you need to know which fields to use in select/query for devices.
```

**Por qué existe esta sección:** Los campos de OpenGate son dinámicos (dependen de los datamodels). Esta sección le dice al LLM que puede consultar un resource para descubrir qué campos existen, en vez de inventarlos o pedírselos al usuario.

---

```
## Common fields per entity

### Devices (used in devices_search query)
- provision.device.identifier — device ID
- provision.device.name — device name
- provision.device.administrativeState — ACTIVE, TESTING, BANNED
- provision.device.operationalStatus — NORMAL, ALARM, DOWN
- provision.administration.organization — organization name
- provision.administration.channel — channel name

### Datamodels (used in datamodels_search query)
- datamodels.identifier — datamodel ID
- datamodels.name — datamodel name
- datamodels.organizationName — organization
- datamodels.version — version

### Alarms (used in alarms_search query)
- alarm.severity — INFORMATIVE, URGENT, CRITICAL
- alarm.status — OPEN, ATTEND, CLOSED
- alarm.name — alarm name
- alarm.rule — rule that triggered the alarm
- alarm.entityIdentifier — device/entity identifier
- alarm.organization — organization name
- alarm.channel — channel name
- alarm.priority — LOW, MEDIUM, HIGH
- alarm.openingDate — ISO 8601 datetime

### Jobs (used in jobs_search query)
- jobs.request.name — operation name (REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC, etc.)
- jobs.report.summary.status — IN_PROGRESS, FINISHED, CANCELLED, PAUSED, CANCELLING_BY_USER

### Tasks (used in tasks_search query)
- tasks.name — task name
- tasks.state — ACTIVE, PAUSED, FINISHED
- tasks.id — task UUID
```

**Por qué existe esta sección:** Sin estos campos, el LLM no sabe qué poner en el `query`. Por ejemplo, para buscar alarmas críticas necesita saber que el campo se llama `alarm.severity` y que el valor es `CRITICAL`, no `critical` ni `Critical`.

---

```
## Creating jobs (operations on devices)

To execute an operation on one or more devices, use jobs_create with a JSON body:

{"job":{"request":{"name":"OPERATION_NAME","parameters":{},"active":true,"schedule":{"stop":{"delayed":90000}},"operationParameters":{"timeout":85000,"retries":0},"target":{"append":{"entities":["device_id_1","device_id_2"]}}}}}

Common operation names: REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC
For REBOOT_EQUIPMENT, add "parameters": {"type": "HARDWARE"}

After creating a job, use jobs_get to check its status and jobs_operations to see per-device results.
```

**Por qué existe esta sección:** Crear un job requiere un JSON específico con estructura anidada. Sin esta plantilla, el LLM tendría que inventar la estructura, lo cual fallaría. Le damos el formato exacto con los campos mínimos obligatorios.

---

```
## Sending IoT data (South API)

Use iot_collect to send a single value to a device datastream:
  iot_collect(device_id: "sense-001", datastream_id: "wt", value: "25.3")

Use iot_collect_payload for multiple datastreams at once:
  iot_collect_payload(device_id: "sense-001", payload: '{"version":"1.0.0","datastreams":[{"id":"wt","datapoints":[{"value":25.3}]},{"id":"wp","datapoints":[{"value":1013}]}]}')
```

**Por qué existe esta sección:** La API Sur usa autenticación diferente (X-ApiKey en vez de JWT) y un formato de payload específico. Esta sección aclara cuándo usar cada tool y cómo.

---

```
## Time series and datasets

For timeseries_data and datasets_data, the filter uses column names defined in the time series/dataset (not device field paths).
Use timeseries_get or datasets_get first to discover available column names, then use those names in the query parameter.
```

**Por qué existe esta sección:** Los campos de filtrado en time series y datasets NO son los mismos que en devices. Se usan los nombres de columna definidos en la serie/dataset. Sin esta nota, el LLM intentaría filtrar con `provision.device.identifier` en vez del nombre de columna correcto (e.g. `Prov Identifier`).

---

```
## Examples

User: "Give me active devices" → devices_search(query: "provision.device.administrativeState eq ACTIVE")
User: "Dispositivos con identificador que contenga sense" → devices_search(query: "provision.device.identifier like sense")
...
(28 ejemplos cubriendo todos los tools en español e inglés)
```

**Por qué existe esta sección:** Los ejemplos son el mecanismo de aprendizaje más efectivo para un LLM — few-shot learning. Cada ejemplo muestra la entrada en lenguaje natural y la llamada exacta al tool con sus parámetros. 28 ejemplos cubren:

| Área | Ejemplos |
|------|----------|
| Devices (search, get, select) | 5 |
| Datamodels | 1 |
| Alarms (search, summary, attend, close) | 5 |
| IoT collect | 2 |
| Time series | 2 |
| Datasets | 1 |
| Jobs (search, create, get, operations, cancel) | 5 |

## Cómo se sirve el prompt

El prompt se registra en `internal/mcp/prompts.go`:

```go
s.AddPrompt(mcp.NewPrompt("opengate-guide",
    mcp.WithPromptDescription("Complete guide for interacting with the OpenGate IoT platform..."),
), handleQueryGuide)
```

El handler devuelve el texto como un mensaje con rol `assistant`:

```go
Messages: []mcp.PromptMessage{
    {
        Role: mcp.RoleAssistant,
        Content: mcp.TextContent{
            Type: "text",
            Text: queryGuideText,
        },
    },
},
```

El rol `assistant` hace que el LLM interprete el contenido como conocimiento propio, no como una instrucción del usuario. Esto es más natural y produce mejores resultados.

## Cómo añadir más prompts

Si en el futuro se necesitan prompts especializados (e.g. uno para administradores que gestionen organizaciones, otro para operadores que solo monitoricen), basta con registrar más prompts en `registerPrompts()`:

```go
s.AddPrompt(mcp.NewPrompt("opengate-admin", ...), handleAdminGuide)
s.AddPrompt(mcp.NewPrompt("opengate-monitoring", ...), handleMonitoringGuide)
```

Cada prompt es independiente — el LLM elige cuál cargar según el contexto.
