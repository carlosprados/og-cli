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
- JS extraction is DECLARED, not guessed. Widget code fields, matched at any depth:
  `_widgetConfigCode`, `_formatterCode`, `formatter`, `script`, `code`, `fn`,
  `expression`. A declared field becomes a sibling `.js` file whenever present —
  even when empty — so its filename never moves because you edited the code.
  Nested fields keep their keypath in the filename (`columns__0__formatter.js`,
  `columns__2___formatterCode.js`). Edit the `.js` files, NOT the JSON copies.
- `operation` is NOT a code field: it carries an operation name (`REFRESH_INFO`).
- Transitional fallback (widgets only): an UNDECLARED field whose content looks
  like JavaScript (long, with `function`/`return`/`=>`/`const`/`let`/`var`) is
  still extracted, but `pull` prints a `hint:` on stderr naming it. Report those
  so the field gets declared — the heuristic misses code written as a bare
  assignment, and fires on prose.
- A `.js` file the family does not declare (your own `helper.js`, generated
  typings) is IGNORED by wrap — reported with a `hint:`, never uploaded as a
  payload field.
- The cycle is content-lossless in the sense that `wrap` reproduces an equivalent
  widget config: the decoded JSON trees compare equal. It is NOT a byte-for-byte
  or SHA256 match — key order and null/default serialisation differ. Two real
  round-trip defects (an object keyed by number turning into an array; a key
  containing `__` colliding with the separator) were fixed in v2.1.0.

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

> **Widget JS survival rules** — abridged from **el Grimorio**
> ([reference/widget-js-api.md](reference/widget-js-api.md)): execution
> contexts, the full $api builder catalog (51), field-name tables per domain,
> lint constraints, return contracts, debug runbook. READ IT before writing
> any widget JS. The essentials (ALL verified live — break any and the widget
> shows "Data not found" or empty values):
>
> 1. **Fetch with `$api` + EXPLICIT `.filter()`** — never the `with*` shortcuts
>    (they emit outdated field names → 400 "Field in filter unknown") and never
>    the `http` wrapper (Nuxt AsyncData; 403 on `/north/v80/*` from widgets):
>
>    ```js
>    var res = await $api.datapointsSearchBuilder()
>      .filter({ and: [
>        { eq: { 'datapoints.entityIdentifier': deviceId } },
>        { eq: { 'datapoints.datastreamId': datastreamId } }
>      ]})
>      .build().execute();
>    var dps = (res && res.data && res.data.datapoints) ? res.data.datapoints : [];
>    // each dp._current = { value, at, date, source }
>    ```
>
>    Filter field names come from the platform OpenAPI specs (datapoints search:
>    `datapoints.entityIdentifier`, `datapoints.datastreamId`). Other builders:
>    devicesSearchBuilder, entitiesSearchBuilder (https://amplia-iiot.github.io/opengate-js/).
>    Since opengate-js 16.0.0 the library ships its own TypeScript declarations, so
>    `og typegen` in a customChart/customTable directory writes og-globals.d.ts +
>    jsconfig.json + package.json: run `npm install` there and $api completes and
>    type-checks in the editor. A misspelled builder becomes an error with a
>    "Did you mean" suggestion instead of a runtime "Data not found".
>    In the editor you get completion only (a widget returns at top level, which
>    TypeScript rejects on a plain file). For real diagnostics run
>    `og widget check` in the widget directory — it type-checks the code inside
>    the platform's own async wrapper and maps positions back to your file.
>    `--exit-code` makes it a CI gate; `--strict` also shows the two idioms it
>    sets aside (Date subtraction, and a property read on a `null`-initialised var).
>    Before deploying, `og workspace diff <dir>` shows what the deploy would do as
>    a tree: workspace → dashboard → widget, with '+' created, '−' deleted and '~'
>    changed. Widgets are matched by identity, so a reorder reads as a move rather
>    than as a rewrite of every widget. Same flags as the other families
>    (`--name-only`, `--against <profile>`, `--exit-code`, `--context`).
>    `og workspace watch <dir>` deploys on save, with the DASHBOARD as the unit:
>    a widget edit deploys its dashboard, not the whole workspace; workspace.json
>    edits are skipped (use deploy). It refuses on conflict, with no --force, and
>    refuses to start against a `production: true` profile without
>    --allow-production. Start with --dry-run.
>
> 2. **ES5 only** — the platform lints widget code with JSHint at render time:
>    `const`/`let`/arrow functions/`for...of` FAIL; use `var`, `function`
>    declarations and `.forEach(function () {})`. A single top-level `await`
>    is accepted (the code runs inside an async wrapper).
>
> 3. **Working reference**: `demo/workspaces/multisensor-demo/00__multisensor-overview/`
>    `02__customchart__demo-temp-chart/_widgetConfigCode.js` in the og-cli repo.
>
> 4. **Debug loop**: drive Chrome (devtools MCP) → console (`JshintError` =
>    lint failure with line number; `OGAPI ERROR` = request failure) → network
>    panel for the failing POST body and the platform's error JSON. Each
>    `og dashboard deploy --update` steals the browser web session (re-login).

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
# Ownership filter: pull only writes items the active profile owns (owner == profile email).
#   single-item (pull / pull-file): REFUSES with the owner shown if not yours
#   bulk (pull-all):                SKIPS non-owned items with a warning, continues
#   --force-owner overrides the single-item refusal (on `workspace pull` it also
#                 forces the nested dashboards). Items with owner=null (system/shared)
#                 are never "owned" — they always need --force-owner.

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

# Share — OPTIONAL, separate from deploy (publishing never shares by itself).
# The ONLY way to grant other users visibility: setting users[] in the JSON
# and PUTting the workspace does NOT work; only PUT .../share does.
# Lists REPLACE current sharing on every call; --unshare clears.
og workspace share <ws-id> --user someone@org.com [--domain d]
og dashboard share <dash-id> --user someone@org.com
og workspace share <ws-id> --unshare

# Direct update / delete
og workspace update <ws-id> -f ws.json
og workspace delete <ws-id>              # destructive: prompts on a TTY, needs --yes non-interactively
og dashboard update <dash-id> -f dash.json
og dashboard delete <dash-id>            # idem
```

> **Destructive verbs (delete) need confirmation.** Non-interactively they refuse
> unless `--yes` is passed. When driving og as an agent, get explicit user
> confirmation BEFORE running, then pass `--yes` — don't add it reflexively. See the
> HITL section in the **og-device-ops** skill.

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
