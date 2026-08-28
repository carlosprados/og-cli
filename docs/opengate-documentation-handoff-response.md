# ogdocgen — what changed on the documentation side, and what to do about it

Written 2026-08-26, after acting on `opengate-documentation-handoff.md`. The full point-by-point
answer is in `docs/opengate-documentation-handoff-reply.md` in the documentation repository; this is
the work list.

## 1. The index file names changed. No alias

If you read the JSON indexes rather than the pages, these moved:

| Was | Now |
|---|---|
| `js-alarms-api-index.json` | `js-rules-api-index.json` |
| `js-alarms-api-index-full.json` | `js-rules-api-index-full.json` |
| `js-alarms-internal-api-index.json` | `js-rules-internal-api-index.json` |
| `js-alarms-internal-api-index-full.json` | `js-rules-internal-api-index-full.json` |

The `api` field inside the `-full` files changes with them: `alarms-js-api` → `rules-js-api`,
`alarms-js-internal-api` → `rules-js-internal-api`. The old names stop being emitted.

Unchanged: the JSON shape, the entry counts (rules 37, rules-internal 37, CF 250, cf-internal 16,
PF 88, TSF 23) and the function URLs — no content page was renamed.

The other four families keep their names: `js-cf-api-index.json`, `js-cf-internal-api-index.json`,
`js-tsf-api-index.json`, `js-pf-api-index.json`.

## 2. `libs/ogapi-docs` — consume the library instead. But the pages are fine

The handoff asked for a `type` on those 213 pages so the `$api` surface would become
machine-readable. The recommendation is still to consume the library rather than parse the pages —
but for one reason, not the two given here before.

> **Corrected 2026-08-27.** This section said those pages were unreliable as an API contract,
> generated from an unmerged feature branch, documenting 15 classes release 14.15.0 does not contain
> and missing 18 that it does. **That was wrong and is withdrawn**, together with the related claim
> that no repository held the generator for them. Both came from a local `opengate-js` checkout 49
> commits behind `origin`, reading 14.15.0 as if it were 15.5.0.

**The pages are accurate, and they have a generator.** `scripts/generate-relearn-md.js` in
`opengate-js`, committed since December 2025 (`3e398ca4`, OUW-4822) and run as
`npm run docs:relearn`; it reads `docs/dump.json` through `scripts/relearn-template.hbs`. At 15.5.0
the folders under `JS Reference/` map one-to-one onto `src/`: `plan/`, `schedule/` and
`organizations/DomainsFinder` are exactly where the pages put them. **Parsing them is safe** — they
are this library's JSDoc rendered.

**The library publishes something better, though.** `opengate-js`, GitHub PR #140, `tools/apidoc`:

```bash
git checkout develop && npm install && npm run apidoc   # 15.5.0
#   build/api-model.json        219 classes, 1526 members, every param and return type
#   build/types/**/*.d.ts       240 declaration files
```

For editor completion on `$api`, **`.d.ts` needs no generator at all** — an editor consumes it
natively, with the JSDoc text as hover documentation. The most likely shape for ogdocgen here is to
stop generating for this surface and instead ship or point at the declarations for the pinned
library version.

If you prefer one artifact of your own, read `build/api-model.json`: it is stable, versioned and
Hugo-agnostic. `tools/apidoc/README.md` documents the shape.

Coverage comparison, in case it decides the design: **1186 of 1250 methods declare a return type in
the source**, against 224 of 453 entries on the documentation pages. The pages are the lossy copy.

## 3. `entity` is an ambient global in rule scope

The reason a live sensehat rule calls `isInsertAction(entity)` while the documentation shows
`isInsertAction()` is that the documentation is right: the code takes no arguments, because `entity`
is a global of the RuleFunction. Confirmed by the platform team.

So rule scope should model `entity` as an ambient global rather than a parameter. `gateway` has the
same structure and the same `entity._*` methods work on it. Both are documented now.

## 4. 22 deprecated globals now name their replacement — emit `@deprecated`

The connector functions reference declared 22 globals deprecated without saying what replaced them.
All 22 now do, so a `@deprecated` annotation with the replacement is generatable, which is worth
having: editors strike the symbol through and show where to go.

One-to-one:

```
entityValue      -> entity._value          responseCF                -> cf.response
entityAt         -> entity._at             collectCF                 -> cf.collection
entityDate       -> entity._date           log                       -> logger.debug
entitySource     -> entity._source         encryptString             -> utils.odm.encryptString
entitySourceInfo -> entity._sourceInfo     decryptString             -> utils.odm.decryptString
entitiesValue    -> utils.odm.entitiesValue getAddressTypeFromAddress -> utils.odm.getAddressTypeFromAddress
```

**Mind the argument order** on the two concatenation ones: `collectCF(data, criteria)` becomes
`cf.collection(criteria, payload)` and `responseCF(data, criteria)` becomes
`cf.response(criteria, payload)` — the criteria moves first. A naive rename produces working-looking
code that sends the payload as the criteria.

Object-level, deliberately **not** a function-for-function mapping — the platform team's words were
that the object design absorbs them, so do not invent per-function equivalences:

```
ogCollection, ogCollectionDs, ogCollectionDp, addOgCollectionDp  -> the collection object
ogResponse, ogStep, ogStepResponse                               -> the response object
httpRequest                                                      -> the http.client object
webSocketMsg                                                     -> the websocket object
publishOnTopic                                                   -> the mqtt object: set mqtt.topic
                                                                    and mqtt.device, then mqtt.publish(payload)
```

## 5. Return types on the documented pages stay incomplete for now

229 of 453 entries across the 43 typed pages have no `**Returns**` line, and only 4 state it in the
heading with `⇒`. Those will keep resolving to `any`. Filling them needs platform answers rather
than editing, so expect it gradually. Six were added where the body prose already stated the type.

## 6. Things verified as fine — do not report them again

- `logger.trace(...msg)` is documented correctly with rest-parameter notation. The earlier arity
  error was a parser bug on your side.
- A top-level `return` in an artifact script is correct; the platform wraps the script in a function.
- `mqtt.device` / `mqtt.topic` are assignable properties documented in a table, which is the right
  shape.
- `normalizeRawObject` is the entry point, not `normalizeRowMap`. Fixed in the prose; the platform's
  own error message already named it.
- `collectionCF` does not exist and never did in this documentation. That comparison came from a
  stale copy of the guide.

## 7. What would help back

Report gaps in the model rather than working around them. Two are already known:

- A rest parameter with no `@param` tag carries no type in the model — `Expression.and(...args)` is
  the example. The old ESDoc output invented `...*` for it.
- **21 `@param` tags name a parameter the function does not have.** `Datapoint.withSource` documents
  `source` where the code takes `value`; `NorthAmpliaREST`'s constructor documents `backend` where it
  takes `headers`; nine builders document `or` where the parameter is `end`. If ogdocgen ever emitted
  a wrong argument name for this library, that is the source. Fixing each means choosing between
  renaming the comment and renaming the parameter, so they are parked on the owner.

A further 73 tags named no parameter at all — `@param {InternalOpenGateAPI} Reference to the API
object.`, where by JSDoc grammar the parameter is called `Reference` — and those are fixed. ESDoc had
been hiding them by matching tags against the signature positionally.

## Owners

`opengate-js` is Chema's (`jousepo`). The platform JavaScript APIs — connector functions, rules,
provision functions, timeseries — are Iñaki's.

---

# Update — answering handoff 2

Written after acting on `opengate-documentation-handoff-2.md`. **All of this is in MR !138, open
against `develop`, not merged yet.** You generate from `develop`, so nothing below is visible to a
regeneration until it lands — ask for it to be merged when you want it.

## §1, optionality — nine functions fixed, two waiting on the platform

`ogCollectionDs`, `ogCollectionDp`, `ogStep`, `ogStepResponse`, `httpRequest`,
`utils.odm.httpRequest`, `collection.addDatapoint`, `collection.getValue` and `response.addStep` now
bracket their trailing optional parameters, in the JSDoc notation you said the generator already
understands. Each was checked against its own table first: all of them document a default ("If not
provided null will be set") rather than merely sounding optional.

**We tested the cost before writing any of it**, since the `####` heading is both the index entry and
the anchor: brackets are dropped when the anchor is generated, so **no URL moves**, and the parser
still reads every parameter — `ogCollectionDp` 4, `collection.addDatapoint` 5. The only outward change
is a more informative `function` string in the index.

`openAlarm`'s `extraInfo` and `executeOperation`'s 11th parameter are **not** fixed yet, and your
report is the only evidence they are optional — live artifacts calling them with 6 and 10 arguments.
That is platform behaviour, so it is asked of Iñaki. Keep the heuristic for those two; it can go for
the nine.

And a distinction worth carrying into the generator: **six functions carry an optionality signal on a
parameter in the middle of the signature**, `openAlarm`'s first among them. In JavaScript only
trailing parameters can be omitted, so that is nullability, not optionality — the parameter must be
passed, possibly as `null`. Bracketing it would be wrong, and reading it as optional is what makes
"optional, then required" appear.

## §2, getVariableValue — fixed, drop the override

It is `*`. Confirmed harder than your report put it: the table said `String` while the page's own two
examples pass `2` and `undefined` and state *"This function returns the same value"*. Its `**Returns**`
line names the type too. `tools/ogdocgen/overrides.go` can lose the entry.

## §3, return types — the work splits three ways, and one third is yours

Checking all 453 entries rather than the 40 you saw:

| | |
|---|---|
| **42** | already name the type, in **lowercase** |
| **6** | name it inside `<code>` tags |
| **40** | genuinely do not name it |

**The 42 are yours, and they are the cheapest win available to you.** Type names are mixed
everywhere in this reference — `String` 182 against `string` 133 in the parameter tables, `Object` 59
against `object`, `boolean` 20 against `Boolean` 5 — so no convention is being broken and normalising
would be four hundred cosmetic edits. **Match type names case-insensitively.** It is one line, and it
is the same class of problem as the `Param`/`Parameter`/`Property` spellings you already accept.

The 6 are unwrapped. **37 of the 40 are now typed from each entry's own words**: `entity._at`,
`_date`, `_source`, `_sourceInfo` and the four deprecated globals get `*`, matching `entity._value`
which already declared it; the eight `utils.date.period.*` get `Object`; `utils.bytes.*` get
`Uint8Array` or `String`; `utils.atcmd.toDBm` gets `Number`; `addIdentificationHint` and
`addCollectionHint` get `Void`; `getCounterValue` gets `Number`.

`getAddressTypeFromAddress` and its deprecated twin are asked, not guessed — *"The address type"* does
not say whether that is a string or an enumerated value.

## New: the JSON indexes now carry return types

Chasing §3 turned up a defect on our side worth more than the prose. The index parser only ever read
a return type from the `fn(x) ⇒ Type` heading form, which **5** entries use; everything after
`**Returns**:` went into the description and the type was discarded. It now splits
`Type - description`, requiring a single token so a sentence can never be taken for a type.

**Entries with a return type in the JSON indexes: 5 → 234 of 451.**

So if reading `js-*-api-index-full.json` is cheaper for you than parsing the pages, the types are
there now, already normalised into a `returns.type` field — and the casing issue above still applies to
its values, which are what the pages say.

## §4, the three table headings — confirmed, and recorded

`Param` 208, `Parameter` 100, `Property` 64. No change wanted, and it is written down on our side so
nobody "tidies" it into one spelling and silently drops the other two.

## §6 — already stale when you wrote it

`DOC/rules-entity-global` merged: the `entity`-is-a-global prose and the `**Returns**: Boolean` on the
three `is*Action` entries are on `develop` as of `b8e0e5a1`.

## What to send back

The numbers, after regenerating on a `develop` that includes !138. Handoff 2 measured 437
declarations, 41 with `@deprecated`, 107 with a real return type. The comparison is the only honest
measure of whether any of this helped, and it is also how the next real defect will surface.
