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
