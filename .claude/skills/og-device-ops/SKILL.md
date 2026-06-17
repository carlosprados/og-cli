---
name: "og-device-ops"
description: "Operate on OpenGate devices with the og CLI: launch operation jobs, DEFINE custom operation types (og optypes), manage automation rules (EASY/ADVANCED with locally-edited JavaScript — og rules pull/deploy), edit connector functions (REQUEST/RESPONSE/COLLECTION JS hooks — og connectors pull/deploy), author provision functions (bulk provisioning processors with locally-edited JS — og provision pull/deploy, plan dry-run before bulk), schedule recurring tasks, triage alarms (summary → search → attend/close), inject IoT data via the South API (HTTP or MQTT), and run og as a virtual MQTT device (og iot publish/subscribe/device — auto-answer operations). Use when executing actions on devices, automating with rules, handling alarms, or sending telemetry."
---

# OpenGate device operations skill

Acting ON devices (vs querying them — that's the **og-cli** skill): operations,
alarms, and data injection. All commands need a logged-in profile (`og login`).

## Jobs — one-shot operations on devices

A job = an operation (REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC, ...) targeted at N
devices, with schedule/timeout/retry semantics. Lifecycle:

```bash
og jobs create -f job.json          # launch (see job-templates.md for ready JSONs)
og jobs get <job-id>                # report: jobs.report.summary.status
og jobs operations <job-id>         # per-device results — check THIS for partial failures
og jobs cancel <job-id>             # stop a running job
og jobs search -w "jobs.report.summary.status eq IN_PROGRESS"
og jobs search -w "jobs.request.name eq REBOOT_EQUIPMENT"
```

- Status values: `IN_PROGRESS`, `FINISHED`, `CANCELLED`, `PAUSED`, `CANCELLING_BY_USER`.
- `FINISHED` does NOT mean every device succeeded — always inspect `jobs operations`
  for per-device outcomes when it matters.
- Job JSON anatomy and per-operation templates with realistic timeouts:
  [job-templates.md](job-templates.md).
- DESTRUCTIVE: rebooting production devices needs explicit user confirmation. Never
  fabricate operation names — only use ones known to the platform.

## Operation types — defining NEW operations (DDL)

`og jobs` LAUNCHES operations; `og optypes` DEFINES them. Before launching an
unfamiliar operation name, check it exists:

```bash
og optypes search                              # catalog + custom ones
og optypes get REBOOT_EQUIPMENT --org <org>    # parameters schema + steps
og optypes create --org <org> -f optype.json   # define a custom operation
```

CRITICAL shape detail (learned the hard way): `parameters` must be a **JSON
Schema object** (`{"type":"object","properties":{...}}`), NOT an array of
name/schema pairs — the create endpoint accepts the wrong shape silently but
`jobs create` then fails with HTTP 500. Step `name` should match the operation
name. Working template: `demo/operations/types/calibrate-sensor.json`.

## Automation rules — EASY and ADVANCED

Rules are channel-scoped (`--channel`, default `default_channel`) + `--org`.
Search filters use the **rule.** prefix: `rule.name`, `rule.mode`, `rule.active`.

```bash
og rules search -w "rule.active eq true"
og rules catalog                                # predefined templates
og rules enable|disable <rule-id> --org <org>   # toggles 'active' via GET+PUT
```

**EASY** = declarative `condition.filter` + `actions` (open/close alarm, email,
HTTP, operation) + optional `parameters` referenced as `"$parameter:<name>"`.
**ADVANCED** = a `javascript` field decides; EASY rules ALSO carry server-generated
javascript (read-only byproduct — don't edit it, edit condition/actions).

**Local JS editing loop** (the og signature move):

```bash
og rules pull <rule-id> --dir rules/ --org <org>   # → rules/<slug>/rule.json + javascript.js
# edit javascript.js in the IDE
og rules deploy rules/<slug> --update --org <org>  # PUT (requires identifier in rule.json)
og rules deploy rules/<slug> --org <org>           # POST: create new (no identifier needed)
```

ADVANCED JS context: `entity['<ds>']._value._current.value` and `._previous.value`
(use _previous for rising-edge/hysteresis logic), `parameterObject` +
`getVariableValue()`, `ruleName`, `openAlarm(null, name, ruleName, severity,
priority, message)`. Working example with multi-datastream correlation:
`demo/rules/default_channel/env-anomaly/javascript.js`.

**Authoritative JS references (harvested from the platform's live docs):**
- [rules-js-reference.md](rules-js-reference.md) — every function available in
  ADVANCED rule JS (open/close alarms, datastream getters, counters, emails...)
- [connector-functions-js-reference.md](connector-functions-js-reference.md) —
  the connector functions JS API (south-side payload processing)
- [provision-functions-js-reference.md](provision-functions-js-reference.md) —
  the provision processor JS API (normalizeRawObject + actionsPlanning, Entity
  builder, *_ACTION builders, V8 search/get utils)

For widget JS (dashboards), the grimoire lives in
`og-workspaces/reference/widget-js-api.md`.

Verify a rule fires: inject a triggering value with `og iot collect`, wait ~5 s,
then `og alarms search -w "alarm.entityIdentifier eq <device>"`.

## Connector functions — south-side JS hooks

Connector functions (`og connectors`, alias `cf`) are JavaScript hooks in the
device-integration pipeline. Channel-scoped like rules (`--channel`, default
`default_channel`) + `--org`. **No search endpoint** — use `list` to enumerate.

Three `type`s (immutable after creation):
- **REQUEST** — transform an outgoing operation request before it reaches the
  device. Matched by `operationName` + `northCriterias` ([{path, value}] over
  device/operation metadata). `payloadType` must be JSON.
- **RESPONSE** — process an operation response from the device. Matched by
  `southCriterias` (URIs/topics/OIDs). `operationName` must be null.
- **COLLECTION** — process collected data and emit datapoints. Matched by
  `southCriterias`. `operationName` must be null.

`operationalStatus` is `DISABLED | PRODUCTION | TEST` (not a boolean like rules).

```bash
og connectors list --org <org>
og connectors catalog                                  # predefined templates
og connectors status <cf-id> TEST --org <org>          # generic status setter
og connectors enable|disable <cf-id> --org <org>       # → PRODUCTION | DISABLED
```

**Local JS editing loop** (same signature move as rules):

```bash
og connectors pull <cf-id> --dir connectors/ --org <org>
#   → connectors/<slug>/connectorfunction.json + javascript.js
# edit javascript.js in the IDE
og connectors deploy connectors/<slug> --update --org <org>  # PUT (needs identifier)
og connectors deploy connectors/<slug> --org <org>           # POST: create new
og connectors pull-all --dir connectors/ --org <org>         # whole channel
```

COLLECTION JS uses the `collection` global (`addDatapoint`, `setFeed`, `send`,
`getValue`); concatenated executions use the `cf` global (`cf.response`,
`cf.collection`). Full grimoire:
[connector-functions-js-reference.md](connector-functions-js-reference.md).

## Provision functions — bulk provisioning JS

Provision functions (`og provision`, alias `pf`) — "provision processors" in the
API — turn inbound rows (typically an Excel sheet) into ODM provisioning actions
(create/update/delete assets, devices, subscriptions, subscribers). **Organization-scoped**
(`--org`, NO `--channel`). **No status field, no catalog, no live logs** (the
script's `printLog` writes to platform logs). Identifier field is
`provisionProcessorId` (not `identifier`); `name` must match `^[a-zA-Z0-9]+$`.

The script lives in `scriptProcessor.script` and MUST implement two functions:
- `normalizeRawObject(rawObject)` — validate + shape one inbound row.
- `actionsPlanning(normalizedObject)` — return the array of actions
  (`CREATE_DEVICE_ACTION`, `UPDATE_ASSET_ACTION`, ...).

**Local JS editing loop** (same move as rules/connectors):

```bash
og provision list --org <org>
og provision pull <pp-id> --dir provision/ --org <org>
#   → provision/<slug>/provisionfunction.json + scriptProcessor__script.js
# edit scriptProcessor__script.js in the IDE
og provision deploy provision/<slug> --update --org <org>  # PUT (needs provisionProcessorId)
og provision deploy provision/<slug> --org <org>           # POST: create new
og provision pull-all --dir provision/ --org <org>
```

**Execution loop — ALWAYS dry-run with `plan` before `bulk` (which mutates data):**

```bash
og provision plan <pp-id> --file data.xlsx --rows 3 --org <org>   # JSON action plan, no mutation
og provision bulk <pp-id> --file data.xlsx --org <org>            # real run → bulk id
og provision bulk-status <bulk-id> --org <org>                    # processed/successful/error
og provision bulk-details <bulk-id> --out result.xlsx --org <org> # 204 while still running
```

`plan` is the key iteration tool: write the script, plan it against a sample
sheet, inspect the computed actions, fix, repeat — no entity is touched until
`bulk`. Full grimoire (the two mandatory functions, V8/Entity/Action utils):
[provision-functions-js-reference.md](provision-functions-js-reference.md).

## Tasks — scheduled / recurring operations

```bash
og tasks search -w "tasks.state eq ACTIVE"      # states: ACTIVE, PAUSED, FINISHED
og tasks get <task-id>
og tasks create -f task.json
og tasks cancel <task-id>
og tasks jobs <task-id>             # jobs spawned by this task
```

## Alarms — triage workflow

```bash
# 1. Overview first: counts by severity/status/rule/name
og alarms summary
og alarms summary -w "alarm.status eq OPEN"

# 2. Narrow down
og alarms search -w "alarm.status eq OPEN" -w "alarm.severity eq CRITICAL"
og alarms search -w "alarm.entityIdentifier eq sense-001"          # one device
og alarms search -w "alarm.openingDate gt 2026-06-01T00:00:00Z"    # recent

# 3. Act (uuid from search results); ALWAYS pass --notes for the audit trail
og alarms attend <alarm-uuid> --notes "Investigating"
og alarms close <alarm-uuid> --notes "Resolved: cause X"
```

Fields: `alarm.severity` (INFORMATIVE, URGENT, CRITICAL), `alarm.status` (OPEN,
ATTEND, CLOSED), `alarm.priority` (LOW, MEDIUM, HIGH), `alarm.name`, `alarm.rule`,
`alarm.entityIdentifier`, `alarm.organization`, `alarm.channel`, `alarm.openingDate`.

## IoT data injection — South API

Sends telemetry INTO the platform as if the device reported it. Auth is `X-ApiKey`
(captured automatically at login), not the JWT.

```bash
og iot collect <device-id> <datastream-id> <value>     # single point
og iot collect sense-001 wt 25.3

og iot collect-file <device-id> -f payload.json        # multiple datastreams at once

og iot collect-raw <device-id> --route <cf-path> --body '<raw>'   # trigger an HTTP CF route
```

`collect`/`collect-file` post to `collect/iot` and **bypass connector functions**.
To TRIGGER a COLLECTION/RESPONSE connector function over HTTP, use `collect-raw`:
it POSTs a raw body to `/south/v80/devices/<id>/<route>` (the CF's HTTP
southCriteria path), so the CF transforms it and emits datapoints — verify the
result in the north with `og devices search`. (The MQTT equivalent is
`og iot publish --topic <cf-topic> --raw`.)

```json
{
  "version": "1.0.0",
  "datastreams": [
    {"id": "wt", "datapoints": [{"value": 25.3}]},
    {"id": "wp", "datapoints": [{"value": 1013, "at": 1717500000000}]}
  ]
}
```

- `at` (epoch millis) is optional — omit for "now"; set it to backfill history.
- The datastream id must exist in the org's datamodel (`og dm get` to verify);
  collecting to an undefined stream is silently useless.
- After collecting, verify with `og dev search -w "<ds> exists" -s <ds>@at` —
  the `_at` column should show your injection time.
- This WRITES platform data: on production tenants, confirm with the user first.

### MQTT south client (publish / subscribe / virtual device)

`og iot` also speaks the OpenGate MQTT south connector. Broker = profile host
(port 1883; `--tls` → 8883), **auth user = device-id, pass = API key**. TLS is
**verified against the system root store by default**; `--ca-file <pem>` adds an
extra CA/chain, `--insecure` skips verification (escape hatch only). Default
topics `odm/iot/<id>` (data), `odm/request/<id>` (operations), `odm/response/<id>`
(responses) — but **every verb takes `--topic`**: connector functions define their
own southCriterias, so topics are NOT fixed.

```bash
og iot publish <id> <ds> <value>                 # publish data over MQTT
og iot publish <id> --topic <cf-route> --raw '{...}'   # custom CF south route
og iot subscribe <id> [--topic T] [--count N]    # observe a topic live (debug CFs/ops)
og iot device <id>                               # VIRTUAL DEVICE: auto-answer operations
og iot device <id> --refresh-data refresh.json   # also fulfils REFRESH_INFO with data
```

- `og iot device` subscribes to `odm/request/<id>` and publishes an acknowledging
  response (`{operation:{response:{name,id,resultCode:SUCCESSFUL,steps,deviceId}}}`)
  to `odm/response/<id>` for each operation — so a `og jobs create` REBOOT_EQUIPMENT /
  REFRESH_INFO targeting the device reaches FINISHED/SUCCESSFUL. Runs until Ctrl-C.
- MCP equivalents are **bounded** (return after N msgs / ops or a timeout):
  `iot_mqtt_publish`, `iot_mqtt_subscribe`, `iot_mqtt_device`.
- To drive a connector function end-to-end over MQTT: `og iot publish <id>
  --topic <cf-southCriteria> --raw '<rawBody>'`, then check the result in the north
  with `og devices search` (see the connector-functions reference).

## Verification pattern (close the loop)

After any operation, verify the effect rather than assuming success:

| Action | Verify with |
|---|---|
| job created | `og jobs get <id>` until FINISHED, then `og jobs operations <id>` |
| alarm attended/closed | `og alarms search -w "alarm.entityIdentifier eq <dev>"` |
| data injected | `og dev search -s <ds>@at -w "provision.device.identifier eq <dev>"` |
