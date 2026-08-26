# Handoff — corrections for the OpenGate documentation

**Target repo:** `odm-documentation-hugo`
**Found by:** building `tools/ogdocgen` in `og-cli`, which generates TypeScript declarations
from this documentation, then type-checking **13 live artifacts** from the `sensehat` organization
against the result. Every item below made correct production code fail to type-check, or is a
self-contradiction inside the documentation.
**Date:** 2026-08-26

---

## Summary

The documentation is in good shape for machine consumption. 43 pages carry a JavaScript API `type`
in their front matter, the heading and table structure is consistent enough to parse without
special cases, and **every page with JavaScript signatures outside `libs/ogapi-docs` is correctly
typed** — nothing is being missed through absent front matter.

Four issues, one of which is a genuine error that will mislead anyone writing a provision function.

---

## 1. `normalizeRowMap` vs `normalizeRawObject` — wrong function name

**Severity: high.** Anyone following the prose writes a provision processor that never runs.

Two pages state the mandatory entry point is `normalizeRowMap`:

- `content/api/management/organizations/channels/entities/bulk/provision_functions/provision_functions_js_api.md:29`
  > `normalizeRowMap(rawObject)`: This function receives a map with the data to be processed.
- `content/api/management/organizations/channels/entities/bulk/provision_functions/provision_functions_main_js_api.md:29`
  > `normalizeRowMap(rawObject)`: This function will read and transform inbound `rawObject` …

And line 32 of the first page refers back to it: *"Takes the result from `normalizeRowMap` function"*.

But **the same page's own code example** defines `function normalizeRawObject`, and so does the live
provision function in `sensehat` (`AddDevices`), which uses `normalizeRawObject` three times and
works.

**Suggested fix:** replace `normalizeRowMap` with `normalizeRawObject` in the prose of both pages,
or — if both names are genuinely accepted by the platform — say so explicitly, because right now the
prose and the example contradict each other on the one function a provision processor cannot do
without.

**Worth checking with the platform team which it is.** If `normalizeRowMap` is a newer name that the
platform also accepts, the example should be updated instead.

---

## 2. `collectCF` vs `collectionCF` — inconsistent between sources

**Severity: medium.** Affects anyone concatenating a collection function.

This documentation says **`collectCF`**, consistently — 6 occurrences, including a worked example:

- `content/api/device_integration/connector_functions/protocol_apis/snmp.md:19`
  `collectCF(result.data, "snmps://<oidValue>");`
- `content/api/device_integration/meter_protocols/_index.md:106`

The concatenation page lists the helpers as `responseCF` and `collectionCF`
(`core_javascript_api/concatenated.md`), and a copy of the guide circulating internally says
`collectionCF` too.

So one of the two is wrong, and they are in the same documentation set. **Please confirm the real
name with the platform team and make both pages agree.** (`og-cli` currently declares what this
repository says — `collectCF` — since it has the working example.)

---

## 3. `isInsertAction()` is documented with no parameters but called with one

**Severity: low.** Cosmetic for a human, but it makes generated typings reject working code.

`content/api/management/organizations/channels/rules/rules_internal_js_functions.md:157` documents:

```
#### isInsertAction()
```

The live `PROVISION_RULE` rule in `sensehat` calls `isInsertAction(entity)` and
`isUpdateAction(entity)`. Either the parameter is optional and undocumented, or the rule is passing
something that is ignored.

**Suggested fix:** document the parameter, or state that the functions take none and that any
argument is ignored. Same for `isUpdateAction` and `isPatchAction`.

---

## 4. `type = 'alarms-js-api'` on the ADVANCED rules pages — misleading label

**Severity: low.** Documentation-internal, no reader-visible effect.

Nine pages use `type = 'alarms-js-api'`:

```
content/api/management/organizations/channels/rules/rules_advanced_actions_api.md
content/api/management/organizations/channels/rules/rules_advanced_alarms_api.md
content/api/management/organizations/channels/rules/rules_advanced_cypher_api.md
content/api/management/organizations/channels/rules/rules_advanced_datastreams_api.md
content/api/management/organizations/channels/rules/rules_advanced_dates_api.md
content/api/management/organizations/channels/rules/rules_advanced_notifications_api.md
content/api/management/organizations/channels/rules/rules_advanced_operations_api.md
content/api/management/organizations/channels/rules/rules_advanced_provision_api.md
content/api/management/organizations/channels/rules/rules_advanced_utils_api.md
```

Only one of them is about alarms; the rest are the ADVANCED rules JavaScript API. Anything keying
off the `type` — a generator, a template, a search facet — has to know that `alarms-js-api` means
rules.

**Suggested fix:** rename to `rules-js-api` (and `rules-js-internal-api` for
`rules_internal_js_functions.md`, currently `alarms-js-internal-api`). Cheap now, and it keeps the
front matter meaning what it says.

---

## Not a documentation problem

Recorded so nobody chases them:

- **`logger.trace(...msg)` is documented correctly**, with the rest-parameter notation, in
  `content/api/debugging/logger_api.md`. An earlier version of the generator dropped the dots and
  reported an arity error on working code. That was a parser bug.
- **A top-level `return` in an artifact script** is correct — the platform wraps the script in a
  function — and TypeScript reports it as an error regardless. Handled on the tooling side.
- **`mqtt.device` / `mqtt.topic`** are documented as assignable properties in a table, which is
  exactly the right shape; the generator reads them from there.

---

## What the documentation enables, if you want it

`tools/ogdocgen` in `og-cli` turns these pages into TypeScript declarations, giving completion and
diagnostics for artifact JavaScript in VS Code, Neovim, Cursor and Zed. Current coverage from this
repository:

| Context | Pages | Declarations |
|---|---|---|
| connector functions (core + 15 protocols) | 26 | 24 functions, 244 methods in 35 objects, 57 properties |
| ADVANCED rules | 10 | 46 functions, 23 methods in 8 objects |
| provision functions | 6 | 42 functions, 46 methods |
| timeseries functions | 1 | 23 methods in 5 objects |

Two things would extend it:

1. **`libs/ogapi-docs/JS Reference/` has no `type` in its front matter** — 92 pages, 149 signatures
   on `InternalOpenGateAPI.md` alone. That is the `$api` (opengate-js) surface used inside **widget**
   code, and it is the one context where og-cli still cannot offer completion. Adding a `type`
   (say `ogapi-js-reference`) would make it machine-readable at no cost to readers.
2. **Return types are rarely stated.** Where a page does state one, the generator uses it; otherwise
   everything is `any`. A `Returns` row in the method tables would improve every generated
   signature.

Neither is urgent. Item 1 is the one with real leverage.
