---
name: "og-device-ops"
description: "Operate on OpenGate devices with the og CLI: launch operation jobs, DEFINE custom operation types (og optypes), manage automation rules (EASY/ADVANCED with locally-edited JavaScript — og rules pull/deploy), schedule recurring tasks, triage alarms (summary → search → attend/close), and inject IoT data via the South API. Use when executing actions on devices, automating with rules, handling alarms, or sending telemetry."
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

Verify a rule fires: inject a triggering value with `og iot collect`, wait ~5 s,
then `og alarms search -w "alarm.entityIdentifier eq <device>"`.

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
```

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

## Verification pattern (close the loop)

After any operation, verify the effect rather than assuming success:

| Action | Verify with |
|---|---|
| job created | `og jobs get <id>` until FINISHED, then `og jobs operations <id>` |
| alarm attended/closed | `og alarms search -w "alarm.entityIdentifier eq <dev>"` |
| data injected | `og dev search -s <ds>@at -w "provision.device.identifier eq <dev>"` |
