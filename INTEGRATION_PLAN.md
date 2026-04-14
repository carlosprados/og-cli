# Plan de integración de la API de OpenGate

## Fase 1: Alarms ✅ COMPLETADA

Alto valor inmediato — monitorización.

| Comando | Endpoint | Método |
|---------|----------|--------|
| `og alarms search` | `/north/v80/search/entities/alarms` | POST |
| `og alarms summary` | `/north/v80/search/entities/alarms/summary` | POST |
| `og alarms attend <id>` | `/north/v80/alarms` | POST |
| `og alarms close <id>` | `/north/v80/alarms` | POST |

Soporta `-w` para filtrar por severidad, estado, regla, etc. El `summary` da conteos agrupados por severidad/estado. `attend` y `close` cambian el estado de la alarma.

TUI: pantalla de alarmas con tabla coloreada por severidad, enter para detalle, teclas rápidas `a` (attend) y `c` (close).

MCP: tools `alarms_search`, `alarms_summary`, `alarms_attend`, `alarms_close`.

## Fase 2: Time Series ✅ COMPLETADA

Análisis temporal.

| Comando | Endpoint | Método |
|---------|----------|--------|
| `og timeseries list` | `/north/v80/timeseries/provision/organizations/{org}` | GET |
| `og timeseries get <id>` | `.../{org}/{id}` | GET |
| `og timeseries create -f` | `.../{org}` | POST |
| `og timeseries update <id> -f` | `.../{org}/{id}` | PUT |
| `og timeseries delete <id>` | `.../{org}/{id}` | DELETE |
| `og timeseries data <id>` | `.../{org}/{id}/data` | POST |
| `og timeseries export <id>` | `.../{org}/{id}/export` | POST |

El subcomando `data` consulta datos con filtros temporales, agregaciones (AVG, SUM, MIN, MAX, COUNT, FIRST, LAST) y soporte CSV. `export` genera Parquet.

TUI: lista de time series, enter para ver config, submenú para consultar datos con rango temporal.

MCP: tools `timeseries_list`, `timeseries_get`, `timeseries_create`, `timeseries_update`, `timeseries_delete`, `timeseries_data`, `timeseries_export`.

## Fase 3: Data Sets ✅ COMPLETADA

Reporting.

| Comando | Endpoint | Método |
|---------|----------|--------|
| `og datasets list` | `/north/v80/datasets/provision/organizations/{org}` | GET |
| `og datasets get <id>` | `.../{org}/{id}` | GET |
| `og datasets create -f` | `.../{org}` | POST |
| `og datasets update <id> -f` | `.../{org}/{id}` | PUT |
| `og datasets delete <id>` | `.../{org}/{id}` | DELETE |
| `og datasets data <id>` | `.../{org}/{id}/data` | POST |
| `og datasets summary <id>` | `/north/v80/search/organizations/{org}/datasets/{id}/summary` | POST |

TUI: lista de datasets, enter para ver config, submenú para consultar datos.

MCP: tools `datasets_list`, `datasets_get`, `datasets_create`, `datasets_update`, `datasets_delete`, `datasets_data`, `datasets_summary`.

## Fase 4: Complementos (futura)

| Área | Comando | Valor | Complejidad |
|------|---------|-------|-------------|
| Datapoints | `og datapoints search` | Ver último valor de cualquier datastream | Baja |
| Datastreams | `og datastreams search` | Catálogo de datastreams disponibles | Baja |
| Operations | `og jobs`, `og tasks` | Ejecutar y monitorizar operaciones | Media |
| Organizations | `og orgs` | Gestión multi-tenant | Baja |
| Channels | `og channels` | Agrupación de dispositivos | Baja |
| Users | `og users` | Gestión de usuarios | Baja |
| Tickets | `og tickets` | Sistema de tickets | Media |

## South API (Data Collection) ✅ COMPLETADA

`og iot collect` permite publicar datos a devices vía South API (X-ApiKey). Esto habilita pruebas end-to-end: publicar datos → buscar en devices/datasets/timeseries → verificar alarmas.

## Notas

- Operations usa `/v80/` sin `/north/` — tenerlo en cuenta en el client.
- El patrón search/summary se repite en toda la API — la función `buildSearchFilter` ya es reutilizable.
- Cada fase incluye los 4 componentes obligatorios: client + CLI + MCP + TUI.
