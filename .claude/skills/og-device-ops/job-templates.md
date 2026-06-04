# Job JSON templates

Anatomy of every job request:

```json
{
  "job": {
    "request": {
      "name": "<OPERATION_NAME>",          // platform-defined operation; never invent
      "parameters": {},                    // operation-specific (see below)
      "active": true,                      // launch immediately
      "schedule": { "stop": { "delayed": 90000 } },        // hard stop after 90 s
      "operationParameters": { "timeout": 85000, "retries": 0 },
      "target": { "append": { "entities": ["<device-id-1>", "<device-id-2>"] } }
    }
  }
}
```

Conventions that work well in practice:
- `operationParameters.timeout` slightly BELOW `schedule.stop.delayed` (e.g. 85 s vs 90 s)
  so the operation times out cleanly before the job is killed.
- `retries: 0` for interactive use; raise only for flaky connectivity fleets.
- `target.append.entities` takes device identifiers (the `provision.device.identifier`
  values) — collect them first with `og dev search ... -s provision.device.identifier`.

## REBOOT_EQUIPMENT — hardware reboot

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

DESTRUCTIVE — confirm with the user before launching against anything that matters.

## EQUIPMENT_DIAGNOSTIC — self-diagnostic

```json
{
  "job": {
    "request": {
      "name": "EQUIPMENT_DIAGNOSTIC",
      "parameters": {},
      "active": true,
      "schedule": { "stop": { "delayed": 120000 } },
      "operationParameters": { "timeout": 110000, "retries": 0 },
      "target": { "append": { "entities": ["sense-001"] } }
    }
  }
}
```

Non-destructive; safe to run for health checks. Results per device via
`og jobs operations <job-id>`.

## Launch & follow

```bash
og jobs create -f job.json
og jobs get <job-id>                 # poll: jobs.report.summary.status
og jobs operations <job-id>          # per-device outcome — the real success signal
```

Other operation names exist per platform configuration (custom operations defined
in datamodels/connectors). If the user names one not listed here, verify it exists
before launching: ask the user or check the platform — do NOT guess parameters.
