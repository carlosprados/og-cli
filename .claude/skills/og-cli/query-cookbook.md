# og query cookbook

Copy/paste recipes by intent. All verified against a live tenant. Add
`--profile <name>` for non-default tenants and `-o json` for machine output.

## Devices — inventory & health

```bash
# Everything in an org, at a glance
og dev search -w "provision.administration.organization eq sensehat" --view summary

# Only ACTIVE devices
og dev search -w "provision.device.administrativeState eq ACTIVE" --view summary

# Devices in trouble (operational status not NORMAL)
og dev search -w "provision.device.operationalStatus neq NORMAL" --view summary

# Find a device by partial id/name
og dev search -w "provision.device.identifier like sense" --view summary

# Machine health: CPU/RAM/disk
og dev search -w "device.cpu.usage gt 80" --view summary,resources
```

## Devices — sensor values with freshness

```bash
# Battery status with WHEN it was reported (charge_at column)
og dev search --view power

# Low battery, who and how stale
og dev search -w "device.powersupply.battery.charge lt 20" --view summary,power

# Custom datastream (e.g. wt = water temperature in sensehat) + timestamp
og dev search -w "wt exists" -s provision.device.identifier -s wt@at

# Range filter on a custom stream
og dev search -w "wt gte 10 AND wt lte 30" -s provision.device.identifier -s wt@at

# Every selected field with its timestamp
og dev search -s wt -s wp -s anin1 --at
```

**Stale-data check pattern**: project `@at` and compare against now; a value whose
`*_at` is days old means the device stopped reporting — surface that to the user.

## Devices — filter by collection time (`at` = reception, `date` = measurement)

Timestamps are **filterable server-side**, not just projectable. Path is
`<datastream>._current.at` / `._current.date`, value = ISO-8601 with offset. Works with
`-w` (the ISO value is kept as a string) or raw `--filter`. Project them with `@at`/`@date`.

```bash
# Devices that COLLECTED ANYTHING after 18:00 yesterday
# → the device identifier datastream is collected on every report, so its `at` answers it
og dev search -w "device.identifier._current.at gte 2026-06-21T18:00:00.000+02:00" -s device.identifier@at

# Devices whose <stream> MEASUREMENT (date) is after T — ASK which stream first
og dev search -w "wt._current.date gte 2026-06-21T18:00:00.000+02:00" -s wt@date

# Combine a time window with a provision filter (AND)
og dev search --filter '{"filter":{"and":[
  {"eq":{"provision.device.administrativeState":"ACTIVE"}},
  {"gte":{"device.identifier._current.at":"2026-06-21T18:00:00.000+02:00"}}
]}}'
```

- **`at` vs `date`**: `at` is when the platform ingested the datapoint, `date` is when
  the sensor measured it. They differ when data arrives late/buffered. "Reported after T"
  → `at`; "measured after T" → `date`.
- For a generic "which devices collected recently" use `device.identifier._current.at`
  (always present). For a specific quantity, disambiguate the datastream with the user.
- The lake is **type-strict**: a string-typed field needs a JSON string in `eq`; a number
  won't match. Check the field type with `og dm get <model>` (`schema.type`/`$ref`).

## Devices — OR and nested filters (raw JSON)

`-w` only does AND. For OR, pass the OpenGate filter JSON directly:

```bash
og dev search --filter '{
  "filter": {
    "or": [
      {"eq": {"provision.device.administrativeState": "TESTING"}},
      {"eq": {"provision.device.administrativeState": "BANNED"}}
    ]
  },
  "limit": {"size": 20}
}'
```

Note: `--filter` overrides `-w`/`--limit`, and `select` must then be embedded in
the JSON too if needed.

## Datamodels — discover what fields exist

```bash
og dm search -w "datamodels.organizationName eq sensehat"
og dm get <datamodel-id> --org sensehat          # lists categories + datastreams
```

Use this when a filter on a custom field returns nothing — the field name probably
doesn't match the datamodel definition.

## Timeseries / datasets — historical data

```bash
og ts list                          # what timeseries exist
og ts get <id>                      # ← REQUIRED first: learn the COLUMN names
og ts data <id> -w "Prov Identifier eq MyDevice1" --limit 50
og ts data <id> --sort <sort-id> --limit 50      # sort ids defined in the timeseries

og ds list
og ds get <id>
og ds data <id> -w "<Column Name> eq <value>" --limit 50
```

Columns are resource-defined display names (often with spaces — quote them), NOT
device field paths. Filtering with device paths silently returns nothing.

## Cross-tenant comparison

```bash
og --profile production dev search --view summary -o json > prod.json
og --profile staging   dev search --view summary -o json > stag.json
# then diff/process the JSON
```
