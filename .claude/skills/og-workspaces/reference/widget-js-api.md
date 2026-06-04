# Widget JavaScript API — el Grimorio

*The definitive spellbook for OpenGate widget JavaScript. Named by Charlie's
decree (2026-06-04): every incantation below was paid for in blood against a
live tenant — five curses broken in one session. Deviate and the platform
will answer with "Data not found".*

Everything needed to write `_widgetConfigCode` (customChart) and customTable
scripts that work FIRST TIME. Every rule here was verified live against
OpenGate v13.1.0 on 2026-06-04 after a five-bug debugging session — do not
deviate without re-verifying.

## 1. Execution context

Widget code runs wrapped in an async function:

```js
// customChart — must RETURN an ECharts config object
async function (entityData, relatedEntities, timeserieData, alarmData,
                dashboardFilters, filters, callback) { /* your code */ }

// customTable — must call callback(rows) with an array of row objects
async function (entityData, relatedEntities, timeserieData, alarmData,
                dashboardFilters, filters, pageElements, page, callback) { /* your code */ }
```

Globals available: `$api` (opengate-js — THE data API), `$user` (email,
workgroup, domain, profile, langCode, timezone), `$moment` (moment.js),
`console`, `Promise`, `http` (Nuxt useFetch wrapper — see §4: do NOT use for
north calls), navigation helpers `openDashboard()` / `openEntityDashboard()`.

## 2. THE LINT — write ES5 or die

The platform lints widget code with JSHint **at render time**. Code that fails
shows "Data not found" and a `JshintError` in the browser console (with line
number). Verified pass/fail:

| Construct | Verdict |
|---|---|
| `var`, `function` declarations, `.forEach(function () {})`, `.map(function () {})` | ✅ pass |
| ONE top-level `await` (the wrapper is async) | ✅ pass |
| Promise chains (`.then`/`.catch`), `Promise.all` | ✅ pass |
| `const` / `let` | ❌ fail |
| Arrow functions `=>` | ❌ fail |
| `for (... of ...)` | ❌ fail |
| nested `async function` declarations | ❌ fail |

Run `scripts/lint-widget.sh <file.js>` before every deploy to catch these.

## 3. Fetching data — `$api` with EXPLICIT `.filter()`

**Golden rule**: use `$api.<domain>SearchBuilder().filter({...explicit...})` —
NEVER the `with*()` convenience shortcuts. The shortcuts emit outdated filter
field names (e.g. `datapoint.device`) that current platforms reject with
**400 "Field in filter unknown"**.

Verified working template (latest datapoint of a device datastream):

```js
function lastValue(deviceId, datastreamId) {
  return $api.datapointsSearchBuilder()
    .filter({
      and: [
        { eq: { 'datapoints.entityIdentifier': deviceId } },
        { eq: { 'datapoints.datastreamId': datastreamId } }
      ]
    })
    .build()
    .execute()
    .then(function (res) {
      var dps = (res && res.data && res.data.datapoints) ? res.data.datapoints : [];
      var latest = null;
      dps.forEach(function (dp) {
        var c = dp._current || {};
        if (c.value !== null && c.value !== undefined) {
          if (latest === null || new Date(c.at) > new Date(latest.at)) { latest = c; }
        }
      });
      return latest ? latest.value : null;
    })
    .catch(function (e) { console.error('datapoints error', e); return null; });
}
```

`execute()` resolves to `{ statusCode, data }`; for datapoints, `data.datapoints`
is an array of `{ datastreamId, entityIdentifier, _current: { value, at, date,
source } }`.

### Common SearchBuilder methods (from opengate-js source)

`.filter(obj)` · `.limit(size, start)` · `.addSortAscendingBy(field)` /
`.addSortDescendingBy(field)` · `.select(...)` · `.withTimeout(ms)` ·
`.build()` → `.execute()` (Promise).

Filter operators: `eq, neq, like, gt, lt, gte, lte, in, nin, exists` composed
with `and` / `or` arrays — same JSON the North API search endpoints take.

### Field names per domain — ALWAYS verify against the OpenAPI spec

The filter field names are domain-specific and the ONLY source of truth is the
platform OpenAPI spec (`ogdoc/` in the og-cli repo). Known-good names:

| Domain (builder) | Filter fields (prefix) | Response key |
|---|---|---|
| `datapointsSearchBuilder` | `datapoints.entityIdentifier`, `datapoints.datastreamId`, `datapoints.feed` | `data.datapoints[]` with `_current` |
| `devicesSearchBuilder` / `entitiesSearchBuilder` | `provision.device.*`, `provision.administration.*`, plus bare datastream ids (`wt`) | `data.devices[]` / `data.entities[]` (flattened) |
| `alarmsSearchBuilder` | `alarm.name`, `alarm.severity`, `alarm.status`, `alarm.entityIdentifier`, `alarm.openingDate` | `data.alarms[]` |
| `rulesSearchBuilder` | `rule.name`, `rule.mode`, `rule.active` | `data.rules[]` |
| `operationsSearchBuilder` (jobs) | `jobs.request.name`, `jobs.report.summary.status` | `data.jobs[]` |

When in doubt: grep the corresponding `ogdoc/**/*.yaml` for the search request
examples — a wrong field name is a 400 with
`{"errors":[{"code":"0x050014","message":"Field in filter unknown"}]}`.

### Full `$api` builder catalog (51 factories, opengate-js)

alarms, areas, assets, basicTypes, bulk, bulkExecution, bundles, certificates,
channels, communicationsModuleType, countryCodes, datamodels, **datapoints**,
datasetEntities, datasets, datasetsCatalog, datastreams, **devices**,
**entities**, executions, executionsHistory, feeds, fieldsDefinition,
ioTDatastreamAccess/Period/StoragePeriod, mobilePhoneProvider,
operationalStatus, **operations**, operationTypes, organizations, raw,
resourceType, **rules**, serviceGroup, softwares, subscribers, subscriptions,
tasks, ticket*(4), timeserie, timezone, userLanguages, userProfiles, users,
workgroups — each as `$api.<name>SearchBuilder()`.

## 4. What NOT to use

- **`http` wrapper**: returns a Nuxt AsyncData (`{data: ref, pending, status,
  error, refresh, execute, clear}`) and gets **403** on `/north/v80/search/*`
  from widget context. Useless for platform data.
- **Raw `fetch`**: no session auth.
- **`with*()` builder shortcuts**: outdated field names → 400 (see §3).

## 5. Return contracts (both verified live)

- **customChart**: a single top-level `await` is fine; `return <ECharts v5
  config object>`. Empty/error states: return
  `{ title: { text: '...', subtext: '...', left: 'center' }, series: [] }`.
- **customTable**: the executor calls `.then()` on your RETURN VALUE — you
  MUST `return` the rows array or a Promise of it. `callback(rows)` alone is
  NOT enough (returns undefined → `TypeError: Cannot read properties of
  undefined (reading 'then')`). Avoid top-level await here; the bulletproof
  shape is:

  ```js
  return Promise.all(calls).then(function (flat) {
    var rows = /* build rows */;
    if (typeof callback === 'function') { callback(rows); }
    return rows;
  });
  ```

  Each row is `{ colField: value, ... }`; a cell can be
  `{ value: '', _chart: <echarts config> }` (embedded sparkline) or
  `{ value: 'x', _style: 'css...' }`.

## 6. Debug runbook (proven)

1. Drive Chrome via the chrome-devtools MCP; login at **https://opengate.es/**
   (UI host; api.opengate.es has no UI). NOTE: every `og dashboard deploy
   --update` steals the browser session (single web session per user) → re-login.
2. `list_console_messages`: `JshintError` = lint (§2); `OGAPI ERROR` +
   `Failed to load resource 400/403` = request problem.
3. `list_network_requests` → `get_network_request` on the failing POST: the
   request body shows what the builder actually sent; the response body names
   the unknown field.
4. Iterate locally (`demo/.../_widgetConfigCode.js` style), redeploy with
   `og dashboard deploy <dir> --update`, reload, re-check.

## 7. Living, verified examples

- `demo/workspaces/multisensor-demo/00__multisensor-overview/02__customchart__demo-temp-chart/_widgetConfigCode.js`
  — customChart: bar chart, multi-device × multi-stream, Promise.all fan-out.
  Renders live.
- The "Multi-Sensor Summary" dashboard in the sensehat Demo workspace —
  customTable: per-cell `_chart` sparklines with `$moment` tooltips, returned
  via the Promise contract above. Renders live (fixed 2026-06-04).
- FullDevicesList column formatters:
  `demo/.../01__fulldeviceslist__*/columns__N___formatterCode.js` — per-column
  value formatting (`n.toFixed(1) + ' ºC'`, color-coded battery).

Related platform JS surfaces (different contexts, same care needed):
ADVANCED rules → `og-device-ops/rules-js-reference.md` (harvested official
guide); connector functions → `og-device-ops/connector-functions-js-reference.md`.
