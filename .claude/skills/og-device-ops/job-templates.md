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

## Large fleets — more than 100 entities

**A target list holds at most 100 entities** (and the body has a 300 KB limit). For a
bigger fleet the platform prescribes: create the job INACTIVE, append the targets in
batches, activate on the last one. `og jobs launch` does exactly that:

```bash
og jobs launch -f diagnosis.json --entities-file meters.txt        # 2500 ids, 25 batches
og jobs launch -f diagnosis.json --entity dev-1 --entity dev-2
```

The activation is merged into the **last** batch, so the job is never active with a
partial target — an active job runs the operation on whatever devices are attached at
that instant, and an operation already sent to a device cannot be recalled. The job id
is printed before the first batch so an interrupted run can be resumed by hand.

**Entities the platform refuses are reported inside SUCCESSFUL responses**
(`report.target.notProvisioned` / `notAllowed` / `duplicated`). `og jobs launch` warns
about them; if you build the job any other way, check for them, or you will believe the
job covers devices it never touched.

## Scattering — spreading operations over time (large fleets)

For a fleet big enough to hurt a mobile cell, add a `scattering` block:

```json
"schedule": {
  "stop": { "delayed": 21600000 },
  "scattering": {
    "maxSpread": 80,
    "strategy": { "factor": 75, "field": "subscription.collected.cellInfo", "warningMaxRate": 3 }
  }
}
```

- **`maxSpread` is a PERCENTAGE of the job's window, and must never be 100.** At 100 the
  last operations are launched exactly as the window expires and never run. The platform
  does not measure durations to reserve that margin, so keep **80, ceiling 90**. The
  library refuses anything above 90.
- `strategy` is **lowercase** on the wire. The published schema says `Strategy`; the
  platform reads `strategy`, and a capitalised key is silently ignored — the job then
  runs unscattered.
- `strategy.field` accepts only `subscription.collected.cellInfo` today.
- Sizing rule: the window must satisfy both
  `window x maxSpread > N x timeout / concurrent-threads` (dispatch slower than capacity)
  and `window x (1 - maxSpread)` big enough to drain the operations still in flight.
  They pull in opposite directions, so when the per-device timeout is larger, **grow the
  window, never shrink the tail**.

These values are documented **assumptions** (the platform's own example values plus that
arithmetic), not measurements. Revisit them against a real job body.

## Launch & follow

```bash
og jobs create -f job.json            # single batch, up to 100 entities
og jobs launch -f job.json --entities-file ids.txt   # any size, batched
og jobs get <job-id>                  # poll: jobs.report.summary.status
og jobs operations <job-id> --all     # per-device outcome — the real success signal
og jobs history --job <job-id> --all  # same, and the dependable one (see SKILL.md)
```

Other operation names exist per platform configuration (custom operations defined
in datamodels/connectors). If the user names one not listed here, verify it exists
before launching: ask the user or check the platform — do NOT guess parameters.
