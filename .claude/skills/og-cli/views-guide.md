# Views guide — intent-based field projection

A view is a named field set that expands into OpenGate select clauses. Ask for
*intent* (`--view power`) instead of memorizing datastream paths.

## Built-in views and when to use each

| View | Use when the user asks about... |
|---|---|
| `summary` | listings, "what devices", general state — **the default for any listing** |
| `power` | battery, power supply, outages (charge/outage include `_at`) |
| `resources` | CPU, RAM, disk usage |
| `temperature` | device temperature (includes `_at`) |
| `location` | where devices are (provisioned coordinates) |
| `status` | administrative + operational state (operationalStatus includes `_at`) |
| `hardware` | model, serial number, description |
| `software` | firmware/software versions |
| `identifier`, `name`, `type` | id/name/type across provision + collected layers |
| `topology` | position in the entity tree |
| `organization` | owning organization |
| `relations` | device↔asset links |

Discover live (custom views included):

```bash
og views list             # NAME, SOURCE (builtin / file), DESCRIPTION
og views show power       # exact expansion: datastream, fields, alias
```

## Combining and precedence

```bash
og dev search --view summary,power            # views combine (deduped by datastream)
og dev search --view summary -s wt@at         # mix with explicit -s
```

- Explicit `-s` clauses WIN on collision with view fields and appear first.
- Unknown view names fail loudly with a typo suggestion — don't guess view names;
  run `og views list` when unsure.

## Timestamps (`@at` / `--at`)

- In views, key collected fields already carry `at` (column `<alias>_at`).
- In explicit selects: `-s wt@at` → columns `wt` + `wt_at`; `--at` applies to all.
- Rule of thumb: any question about a CURRENT sensor value should project its
  timestamp — the value is meaningless without knowing how fresh it is.

## Custom views (YAML, three layers)

| Layer | Path | Overrides |
|---|---|---|
| Project | `./.og/views/*.yaml` | user + builtin |
| User | `~/.og/views/*.yaml` | builtin |
| Builtin | embedded in the binary | — |

Same name in a higher layer replaces the lower one entirely. Two files in the SAME
directory defining the same view = hard error at load time.

```yaml
# ~/.og/views/sensehat.yaml — any filename, many views per file allowed
views:
  water:
    description: Water sensor readings (sensehat custom datamodel)
    fields:
      - wt@at                          # shorthand: value + at
      - wp                             # shorthand: value only
      - name: device.temperature.value
        at: true
        alias: temp                    # long form when you need a custom column name
```

The view is immediately usable everywhere: CLI `--view water`, TUI `v` picker,
MCP `devices_search(view: "water")`. When a user repeatedly asks for the same
field combination, OFFER to create a custom view for it.

## Where views work

| Interface | How |
|---|---|
| CLI | `og dev search --view <names>` |
| TUI | `v` key on the Devices screen (picker with builtin + custom) |
| MCP | `devices_search(view: "...")` + resource `opengate://views` |
