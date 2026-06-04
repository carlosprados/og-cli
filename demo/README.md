# og-cli — End-to-end IoT demo

A complete OpenGate scenario driven 100% from `og`: provision a datamodel and a
device fleet, inject telemetry, automate with rules (including JavaScript edited
locally), define and launch a custom operation, and publish a dashboard that
shows everything live.

**Audience**: IoT engineers and OpenGate users. Every step is copy/paste.

## The local project structure

This folder is also the reference layout for working with OpenGate locally —
every platform artifact lives in versionable files, JavaScript included:

```
demo/
├── datamodels/        # datamodel definitions (og dm create -f ...)
├── devices/           # device provisioning JSONs (og dev create -f ...)
├── collect/           # IoT payloads (og iot collect-file -f ...) + timestamp refresher
├── rules/             # automation rules per channel — rule.json + javascript.js
│   └── default_channel/
│       ├── battery-low/            # EASY rule: declarative condition + actions
│       └── env-anomaly/            # ADVANCED rule: local JS, deployed with og rules deploy
├── operations/
│   ├── types/         # custom operation definitions (og optypes create -f ...)
│   └── jobs/          # job launches (og jobs create -f ...)
└── workspaces/        # unwrapped workspace trees (og workspace deploy <dir>)
    └── multisensor-demo/           # dashboard with 6 widgets, widget JS in .js files
```

(Reserved for the future: `processors/` for provision functions — also local JS.)

## Prerequisites

```bash
task build                      # or: go install
og login -e <email>             # password prompted; needs web session for workspaces
og dm search --limit 1          # sanity check: API reachable
```

All commands run from the repo root. Add `--org <your-org>` or set the org in
your profile (`~/.og/config.yaml`).

## 1 — Datamodel

```bash
og dm create --org sensehat -f demo/datamodels/multisensor.json
og dm get multisensor --org sensehat
```

5 datastreams in 2 categories: `sensor.temperature`, `sensor.humidity`,
`sensor.luminosity` (Environment) + `energy.consumption`, `power.battery` (Power).

## 2 — Devices

```bash
og dev create --org sensehat -f demo/devices/multisensor-001.json
og dev create --org sensehat -f demo/devices/multisensor-002.json
og dev create --org sensehat -f demo/devices/multisensor-003.json

og dev search -w "provision.device.identifier like multisensor" --view summary
```

## 3 — Telemetry (South API)

Refresh the payload timestamps so the data looks live, then inject:

```bash
demo/collect/refresh-timestamps.sh
og iot collect-file multisensor-001 -f demo/collect/multisensor-001.json
og iot collect-file multisensor-002 -f demo/collect/multisensor-002.json
og iot collect-file multisensor-003 -f demo/collect/multisensor-003.json

# Verify — note the *_at freshness columns:
og dev search -w "provision.device.identifier like multisensor" \
  -s provision.device.identifier -s sensor.temperature@at -s power.battery@at
```

## 4 — Automation rules

Deploy both demo rules (EASY + ADVANCED):

```bash
og rules deploy demo/rules/default_channel/battery-low --org sensehat
og rules deploy demo/rules/default_channel/env-anomaly --org sensehat
og rules search -w "rule.active eq true"
```

The ADVANCED rule's logic lives in
[`env-anomaly/javascript.js`](rules/default_channel/env-anomaly/javascript.js):
it correlates temperature AND humidity with rising-edge detection — open the file,
it's plain JavaScript you can edit in your IDE.

**The local-editing loop** (the og signature move):

```bash
og rules pull <rule-id> --dir /tmp/myrules --org sensehat   # id from rules search
$EDITOR /tmp/myrules/<rule-slug>/javascript.js              # tweak the logic
og rules deploy /tmp/myrules/<rule-slug> --update --org sensehat
```

## 5 — Trigger an alarm

Inject a low battery value; the EASY rule opens a CRITICAL alarm:

```bash
og iot collect multisensor-001 power.battery 12
sleep 5
og alarms search -w "alarm.entityIdentifier eq multisensor-001"
```

(Trigger the ADVANCED rule too: `og iot collect multisensor-002 sensor.humidity 30`
then `og iot collect multisensor-002 sensor.temperature 31` — hot AND dry.)

## 6 — Custom operation

Define a new operation type, launch it on a device, inspect the result:

```bash
og optypes create --org sensehat -f demo/operations/types/calibrate-sensor.json
og optypes get CALIBRATE_SENSOR --org sensehat

og jobs create -f demo/operations/jobs/calibrate-001.json
og jobs search -w "jobs.request.name eq CALIBRATE_SENSOR"
og jobs operations <job-id>
```

> Note: without a south connector listening, the per-device operation stays
> pending until the schedule stop kicks in — what the demo shows is the full
> definition→launch→tracking flow.

## 7 — Publish the dashboard

```bash
og workspace deploy demo/workspaces/multisensor-demo
```

Open the OpenGate web UI → workspace **Multisensor Demo** → dashboard
**Multisensor Overview**. Six widgets, all fed by what you just created:

| Widget | Shows |
|---|---|
| markdown | What this demo is |
| FullDevicesList | The fleet with live datastream values (JS formatters) |
| customChart | Temperature/humidity/battery per device (ECharts, JS in `_widgetConfigCode.js`) |
| DeviceAlarmsList | The alarms your rules opened |
| OperationsList | The CALIBRATE_SENSOR job |
| rulesBrowser | The rules you deployed |
| customTable | Fleet history — a per-cell ECharts sparkline for every device × datastream (JS in `_widgetConfigCode.js`, written to the widget-JS grimoire rules) |

## Repeatable demo — setup & teardown

The demo is fully reversible, run it as many times as you need:

```bash
demo/teardown.sh                    # undo EVERYTHING (workspace, alarms, rules,
                                    # optype, devices, datamodel) — tolerant,
                                    # safe to run on a half-torn-down state
demo/teardown.sh --keep-provision   # faster variant: keep datamodel + devices

demo/setup.sh                       # run the whole demo unattended (rehearsal);
                                    # for a live audience, do the steps manually
```

Both scripts resolve dynamic IDs (rules, alarms) by name, so they work across
runs. Two platform footprints survive teardown by design: CLOSED alarm entries
and launched-job history — OpenGate keeps the audit trail.
