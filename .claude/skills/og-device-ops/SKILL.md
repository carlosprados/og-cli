---
name: "og-device-ops"
description: "Operate on OpenGate devices with the og CLI: launch operation jobs, DEFINE custom operation types (og optypes), manage automation rules (EASY/ADVANCED with locally-edited JavaScript — og rules pull/deploy), edit connector functions (REQUEST/RESPONSE/COLLECTION JS hooks — og connectors pull/deploy), author provision functions (bulk provisioning processors with locally-edited JS — og provision pull/deploy, plan dry-run before bulk), schedule recurring tasks, triage alarms (summary → search → attend/close), inject IoT data via the South API (HTTP or MQTT), and run og as a virtual MQTT device (og iot publish/subscribe/device — auto-answer operations). Use when executing actions on devices, automating with rules, handling alarms, or sending telemetry."
---

# OpenGate device operations skill

Acting ON devices (vs querying them — that's the **og-cli** skill): operations,
alarms, and data injection. All commands need a logged-in profile (`og login`).

## Destructive operations — confirm BEFORE running (HITL)

`delete` and `cancel` verbs are irreversible. og guards them: on an interactive
terminal it prompts `[y/N]`; **non-interactively it REFUSES unless `--yes` is
passed** (`Error: refusing to delete ... re-run with --yes (no interactive
terminal)`). You drive og non-interactively, so that terminal prompt never reaches
the user — the safeguard is YOURS to honor:

1. **STOP** before any destructive command. State exactly what it affects — verb, id,
   org/channel, and what is lost (e.g. *"delete device `sense-001` in org `acme` —
   removes the entity and its collected data"*).
2. **Ask the user** and wait for an explicit yes.
3. **Only then** run it with `--yes` (required non-interactively).

Never add `--yes` reflexively to silence the "refusing…" error — that error means
*ask the user first*, not *append a flag*.

Destructive verbs: `dev delete`, `jobs cancel`, `tasks cancel`, `dm delete`,
`connectors delete`, `rules delete`, `optypes delete`, `provision delete`,
`timeseries delete`, `datasets delete`, `workspace delete`, `dashboard delete`.
Treat `deploy --update` (overwrites) and bulk `provision` runs as high-impact too —
confirm scope first.

## Jobs — one-shot operations on devices

A job = an operation (REBOOT_EQUIPMENT, EQUIPMENT_DIAGNOSTIC, ...) targeted at N
devices, with schedule/timeout/retry semantics. Lifecycle:

```bash
og jobs create -f job.json          # launch, ONE batch (max 100 entities)
og jobs launch -f job.json --entities-file ids.txt   # any fleet size, batched
og jobs get <job-id>                # report: report.summary.status
og jobs operations <job-id> --all   # per-device results — check THIS for partial failures
og jobs history --job <job-id> --all    # same results, filterable, across jobs
og jobs cancel <job-id>             # stop a running job
og jobs search -w "jobStatus eq IN_PROGRESS"
og jobs search -w "operationName eq REBOOT_EQUIPMENT"
```

- Status values: `IN_PROGRESS`, `FINISHED`, `CANCELLED`, `PAUSED`, `CANCELLING_BY_USER`.
- **`jobs search` filter names differ from the response paths**, same trap as the
  operation history: use `jobStatus` (alias `job.status`), `operationName`, `jobId`,
  `taskId`. Filtering on `jobs.report.summary.status` or `jobs.request.name` — the
  names you *read* in the output — returns HTTP 400 *"Field in filter unknown"*.
  Verified live: `jobStatus eq IN_PROGRESS` returns results, the response path does not.
- `FINISHED` does NOT mean every device succeeded — always inspect `jobs operations`
  for per-device outcomes when it matters.

**Reading results back — two commands, different jobs to do.**

- `og jobs operations <job-id>` lists ONE job by id (GET, paged by `--page`/`--limit`,
  page size max **1000**).
- `og jobs history` takes a **filter**, so it selects across jobs. `--job <id>` is the
  shortcut for one job's outcome. Page size max 2000.

**VERIFIED LIVE — `jobs operations` can return HTTP 204 for a job whose operations
exist.** On a real tenant, `/operation/jobs/{id}/operations` returned 204 (no content)
for a FINISHED job while `jobs history` returned its operation with full steps. Do NOT
conclude "the job had no operations" from an empty `jobs operations`: cross-check with
`og jobs history --job <id>`. Treat `jobs history` as the reliable way to read results.

**History filter names do NOT match the response field names.** Confirmed by the OpenGate
team and verified live:

| What you filter on | Field name |
|---|---|
| job, entity, operation id, resource type | `jobId`, `entityId`, `operationId`, `resourceType` — **unprefixed** |
| operation name | `operationName` or `operation.name` |
| lifecycle state | `operationStatus` |
| outcome | `operationResult` or `operation.result` |
| date, notify | `operationDate`, `operationNotify` |

Everything else is HTTP 400 *"Field in filter unknown"*: the bare `name`, `status`,
`result`, `user`, `date`, `description`, anything with an `operations.` prefix (those are
projection/CSV names), and the near-misses `operation.status`, `operationEntityId`,
`operationJobId`, `operationUser`.

So **filter failures server-side** instead of pulling everything:

```bash
og jobs history -w "operationResult eq ERROR" --all               # only the failures
og jobs history --job <id> -w "operationResult eq ERROR"          # ...within one job
og jobs history -w "operationName eq DIAGNOSIS" -w "operationStatus eq FINISHED"
```

**Both are paged, and the first page is not the whole job.** Pass `--all` whenever the
question is "did every device succeed" or "how many failed" — otherwise you are counting
one page and will report a wrong number that looks right.

**Per-operation detail lives in the steps.** Each operation has `status` (lifecycle, e.g.
`FINISHED`) and `result` (outcome, e.g. `SUCCESSFUL`), and `steps[]` where each step has
a `name`, a `result` and a `description`. The step `result` values are `SUCCESSFUL`,
`ERROR`, `SKIPPED` and `NOT_EXECUTED` — note the field is `result`, not `status`, and
that `NOT_EXECUTED` exists. The table output shows a compact `NAME=OK/ERR/SKIP/NOTRUN`
summary; use `--output json` for the full descriptions, which is where the actual reason
for a failure is written.
**More than 100 entities needs `og jobs launch`.** A target list caps at 100, so a big
fleet requires create-inactive → append in batches → activate. `og jobs launch` does it
and merges the activation into the LAST batch, so the job never goes live with a partial
target (an active job operates on whatever is attached at that moment, and an operation
already sent cannot be recalled). It also surfaces the entities the platform refused —
`notProvisioned`/`notAllowed`/`duplicated` arrive inside SUCCESSFUL responses, so a job
can silently cover fewer devices than you asked for.

**Scattering: `maxSpread` is a percentage of the job window and must never be 100** — the
last operations would be launched exactly as the window expires and never run. Use 80,
ceiling 90. And `strategy` is lowercase on the wire despite the schema saying `Strategy`;
a capitalised key is ignored and the job runs unscattered. Details and the window-sizing
rule: [job-templates.md](job-templates.md).

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
#                                                   + og-globals.d.ts + jsconfig.json
# pull also generates TypeScript declarations so tsserver (Neovim/VS Code/Cursor/
# Zed) type-checks the rule. They are generated from THIS org: every datastream id
# in the datamodel with its value type, plus the rule's own parameters typed from
# their schema. entity['sensro.temperature'] and severity:'HIGH' become editor
# errors instead of silent runtime undefined. Both files are ignored by wrap and
# deploy — they never reach the platform.
og typegen --context rule/ADVANCED --org <org> --out rules/<slug>/   # regenerate after a datamodel change
og rules pull <rule-id> --dir rules/ --org <org> --no-typings        # skip them
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
# Two connector functions may legitimately share a name (the identifier is the
# key). pull-all suffixes the slug of the second with a short identifier instead
# of overwriting the first — so the directory name is not always Slugify(name).
# Resolve a directory back to its artifact by reading `identifier` from the
# metadata JSON, never by assuming the slug.
#
# Code paths are declared, not guessed. A connector function's code is always
# `javascript` → javascript.js; a rule's is `javascript` → javascript.js; a
# provision function's is `scriptProcessor.script` → scriptProcessor__script.js.
# The file is written even when the field is empty, so its path never moves.
# Any other .js you drop in the directory is IGNORED by wrap/deploy (reported
# with a `hint:`) — it will not reach the platform.
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
**verified against the system root store by default**; the **global** `--ca-file <pem>`
adds an extra CA/chain and `--insecure` skips verification (escape hatch only) —
these now apply to HTTP and MQTT alike (see og-cli skill). Default
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
| job created | `og jobs get <id>` until FINISHED, then `og jobs operations <id> --all` |
| alarm attended/closed | `og alarms search -w "alarm.entityIdentifier eq <dev>"` |
| data injected | `og dev search -s <ds>@at -w "provision.device.identifier eq <dev>"` |

## Gotchas (hard-won, verified live)

- **Device logs need `TESTING` state.** A device emits logs — the `logger.*` output of
  its connector functions AND ADVANCED rules — ONLY while its
  `provision.device.administrativeState` is `TESTING`. With an `ACTIVE` device the
  CF/rule still runs and collects/acts normally, but `og connectors logs` /
  `og rules logs` stream nothing (just the connect header). Flip the device to TESTING
  to debug, and use `--level DEBUG`/`TRACE` (default INFO hides `logger.debug`).
- **COLLECTION CF: `return collection` — do NOT `send()` then return.**
  `collection.send()` flushes and empties the queue, so a trailing `return collection`
  hands back an empty payload and the south route answers HTTP 400 *"Json is malformed
  (datastreams: null)"* even though the data was already collected. Use `return collection`
  with no final `send()` (or `send()` and return nothing).
- **`og iot collect*` to a non-existent device → HTTP 401 `0x04 "Unauthorized user for
  this operation"`.** The wording is misleading: it usually means *device not found /
  not collectable*, NOT a permissions/API-key problem — verify the device exists
  (`og dev get <id>`) before suspecting your credentials.
- **Deleted entity identifiers stay reserved.** After `og dev delete`, re-creating the
  same identifier returns HTTP 400 *"Entity duplicated"*, yet the device is not
  searchable, gettable or collectable (a ghost). For repeatable demos/scripts,
  parameterise device ids per run (e.g. a timestamp suffix) — don't reuse them.
- **`og connectors deploy --update` needs the `identifier` in connectorfunction.json**
  (pull first). For repeatable, environment-portable assets prefer **delete-then-create
  by name** over update — the identifier is environment-specific.
