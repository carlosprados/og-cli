---
name: "og-workspaces"
description: "Design, edit, and deploy OpenGate Workspaces, Dashboards, and Widgets with the og CLI: verified widget type catalog (with per-type JSON references), local directory structure, full pull/edit/wrap/deploy lifecycle, JS extraction rules, multi-phase import, and cross-tenant migration. Use when creating or editing OpenGate dashboards/widgets, or moving workspaces between tenants."
---

# OpenGate Workspaces skill

Workspaces are the OpenGate UI top-level containers (Web API `/api/v1`); each
workspace owns N dashboards; each dashboard is a grid of widgets. The `og` CLI
turns them into an editable local directory tree (`pull`) and back (`wrap`/`deploy`),
extracting embedded JavaScript into real `.js` files.

Prerequisite: `og login` WITHOUT `--no-web` (workspace/dashboard commands use the
Web API token). Quirk: OpenGate allows one active web session per user — a browser
login steals the CLI session; og re-signs-in automatically on 401.

## Local directory structure (what `pull` produces / `wrap` expects)

```text
wsroot/
└── <workspace-slug>/                          # e.g. dashboards-adif
    ├── workspace.json                         # metadata only — dashboards array is STRIPPED
    └── NN__<dashboard-slug>/                  # NN__ prefix preserves array order, e.g. 00__visualizaci-n-pbi
        ├── dashboard.json                     # dashboard metadata + _workspaceLayout block
        └── NN__<widget-type>__<wid>/          # e.g. 00__customchart__1727269767709-0
            ├── widget.json                    # GridItem: layout (x,y,w,h,i) + definition
            └── _widgetConfigCode.js           # extracted JS (when present)
```

Rules:
- `workspace.json` MUST NOT embed the `dashboards` array — the CLI derives it from subdirectories.
- `dashboard.json` MUST include `_workspaceLayout` (`x`,`y`,`w`,`h`,`id`) linking it to the workspace grid.
- `widget.json` MUST be a full GridItem (layout coords AND `definition`), not a bare widget config.
- JS extraction: fields named `formatter`, `script`, `operation`, `code`, `fn`,
  `expression`, `_widgetConfigCode` — or any long string containing JS keywords
  (`function`, `return`, `=>`, `const`, `let`, `var`) — become sibling `.js` files;
  nested fields keep their keypath in the filename (e.g. `columns__0__formatter.js`).
  Edit the `.js` files, NOT the JSON copies.
- The cycle is content-lossless: `wrap` reproduces identical widget configs (same SHA256).

JSON shape details: [reference/workspaces.md](reference/workspaces.md),
[reference/dashboards.md](reference/dashboards.md),
[reference/commonFields.md](reference/commonFields.md).

## Widget catalog — DO NOT INVENT TYPES

> **CRITICAL**: never invent, guess, or synthesize widget type names or config
> fields. Use ONLY the verified types below; consult the per-type reference before
> writing any config. Anything else fails platform validation or renders empty.

| Type | Purpose | Reference |
|---|---|---|
| `customTable` | tables with custom JS evaluations | [customTable.md](reference/customTable.md) |
| `customChart` | ECharts v5 charts (line/bar/stats) | [customChart.md](reference/customChart.md) |
| `actionButton` | buttons + dynamic forms (replaces `customAction`) | — (see [utils.md](reference/utils.md) for action context) |
| `clock` | live clock, zero-config | [clock.md](reference/clock.md) |
| `markdown` | static CommonMark content | [markdown.md](reference/markdown.md) |
| `iframeWidget` | embedded external pages | [iframeWidget.md](reference/iframeWidget.md) |
| `DatapointsList` | entity telemetry tables | [DatapointsList.md](reference/DatapointsList.md) |
| `FullDevicesList` | device/asset list with status | [FullDevicesList.md](reference/FullDevicesList.md) |
| `DeviceAlarmsList` | live alarm list | [DeviceAlarmsList.md](reference/DeviceAlarmsList.md) |
| `maps` | Google Maps with device GPS | — |
| `entityTimeseriesMultipleHistory` | historical TS charts | [TimeseriesList.md](reference/TimeseriesList.md) |
| `datamodelBrowser` | datamodel/datastream browser | [browsers.md](reference/browsers.md) |
| `devicePlansBrowser` | device plans browser | [browsers.md](reference/browsers.md) |
| `connectorFunctionsBrowser` | connector functions monitor | [browsers.md](reference/browsers.md) |
| `connectorFunctionsCatalogBrowser` | connector templates catalog | [browsers.md](reference/browsers.md) |
| `rulesBrowser` | read-only platform rules view | [rulesBrowser.md](reference/rulesBrowser.md) |

Deprecated/unverified: `customAction` (→ `actionButton` since v13.1.0),
`summaryChart`, `ExecutionsList`, `BundlesList`.

Widget JS runs sandboxed with globals `$api`, `$user`, `$moment`, `http` — see
[reference/utils.md](reference/utils.md). Window filters:
[reference/windowFilterVariants.md](reference/windowFilterVariants.md).

## CLI lifecycle — complete surface

```bash
# Discover
og workspace list [--full]              # --full embeds dashboards
og workspace get <ws-id> [--full]
og dashboard list [--workspace <ws-id>]
og dashboard get <dash-id>

# Pull (unwrap) — remote → editable tree    [aliases: unwrap = pull]
og workspace pull <ws-id> --dir wsroot/
og workspace pull-all --dir wsroot/
og workspace pull-file ws.json --dir wsroot/        # from a local export, no API call
og dashboard pull <dash-id> --dir dashroot/
og dashboard pull-all --dir dashroot/ [--workspace <ws-id>]
og dashboard pull-file dash.json --dir dashroot/
# all accept --force to overwrite an existing destination

# Wrap — tree → single JSON (no upload)
og workspace wrap wsroot/<workspace-slug> --out ws.json
og dashboard wrap dashroot/<dashboard-dir> --out d.json

# Deploy — wrap + import in one step
og workspace deploy wsroot/<workspace-slug>             # POST: create
og workspace deploy wsroot/<workspace-slug> --update    # PUT: overwrite existing
og dashboard deploy wsroot/<ws>/<dashboard-dir> [--update] [--workspace <other-ws-id>]

# Export / import raw JSON (no tree)
og workspace export <ws-id> --out ws.json
og workspace export <ws-id> --dir backups/          # auto-naming: backups/<id>.json
og workspace export-all --dir backups/
og dashboard export <dash-id> --out d.json
og dashboard export-all --dir backups/ [--workspace <ws-id>]
og workspace import -f ws.json [--update]
og dashboard import -f dash.json [--update] [--workspace <id>]

# Direct update / delete
og workspace update <ws-id> -f ws.json
og workspace delete <ws-id>
og dashboard update <dash-id> -f dash.json
og dashboard delete <dash-id>
```

> **CRITICAL deploy rule**: creating a workspace from scratch OR adding NEW
> dashboards → deploy WITHOUT `--update`. Running `deploy --update` when a
> dashboard doesn't exist yet fails to link it and it shows up empty/missing.
> Use `--update` ONLY for edits to existing workspaces/dashboards.

## Multi-phase import (why import isn't one POST)

`og workspace import` replays the web-UI wizard so dashboards and their widget JS
actually persist:

```
POST /api/workspaces            ← shell (no dashboards inline)
POST /api/dashboards × N        ← each dashboard with full grid + widgets
PUT  /api/workspaces/{id}       ← shell + dashboards[] as grid-layout refs
```

`--update` variant: `PUT /api/dashboards/{id} × N` then `PUT /api/workspaces/{id}`.
If an import fails mid-way, the workspace shell may exist without all its
dashboards — re-run with `--update`, or delete the shell and import again.

## Standard workflows

**Edit cycle (IDE / AI):**
```bash
og workspace pull <ws-id> --dir wsroot/
# ... edit .js / .json files ...
og workspace deploy wsroot/<slug> --update
```

**Cross-tenant migration:**
```bash
og login --profile source
og workspace export <ws-id> --out ws.json
og login --profile destination
og --profile destination workspace import -f ws.json
# single dashboard into a different workspace:
og --profile destination dashboard import -f dash.json --workspace <dst-ws-id>
```

**New dashboard from scratch:** build the directory tree by hand (follow the
structure rules above, pick types from the catalog, copy a reference JSON as a
starting point) → `og dashboard deploy <dir>` (no `--update`) → verify rendering
in the platform.
