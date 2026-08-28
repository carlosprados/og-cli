# DX Analysis — Phase 0 deliverable

**Status:** analysis and design. No implementation.
**Date:** 2026-08-25
**Input:** `og-cli-dx-handoff.md`
**Method:** source audit of the current checkout, plus diagnostic probes executed against
`internal/unwrap` (written, run, and removed — findings marked `verified-probe`).

Confidence markers used throughout:

| Marker           | Meaning                                                      |
| ---------------- | ------------------------------------------------------------ |
| `verified-code`  | read directly from this checkout                             |
| `verified-probe` | reproduced by executing a diagnostic test                    |
| `verified-live`  | previously confirmed against a live instance (v0.9.0 work)   |
| `from-docs`      | from the platform JS reference vendored in `.claude/skills/` |
| `unknown`        | not established; needs a live probe or a human decision      |

---

## 1. The handoff is out of date

The handoff was written against an earlier state of the repo. Six of its premises are wrong,
and two of its mandatory Phase 0 research tasks are already-answered questions. Correcting
this first, because it changes the plan's shape and its cost.

| Handoff claim                                                         | Reality                                                                                                                                                                          | Evidence                                                            |
| --------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Connector functions are unimplemented (Phase 2)                       | Shipped in v0.9.0 as `og connectors` (alias **`og cf`** already exists)                                                                                                          | `cmd/connectors.go`, `verified-code`                                |
| Provision functions are unimplemented (Phase 3)                       | Shipped in v0.9.0 as `og provision`, including `plan` (dry-run) and `bulk`                                                                                                       | `cmd/provision.go`, `verified-code`                                 |
| "Locate the SHA256 config comparison used to assert wrap is lossless" | **No SHA256 exists anywhere in the repo.** Losslessness is asserted by `reflect.DeepEqual` over decoded trees in tests only. **Source of the confusion found** (2026-08-25): the `og-workspaces` skill claimed *"wrap reproduces identical widget configs (same SHA256)"* — a documentation overstatement, now corrected there                                                      | `grep sha256` → no Go matches, `verified-code`                      |
| Three divergent `pull`/`wrap`/`deploy` implementations                | One shared core (`ExtractJSFields`/`ReinjectJSFields`/`Slugify`) plus **three byte-identical adapters** and one genuinely different nested pipeline (workspace→dashboard→widget) | `internal/unwrap/{rules,connectors,provisions}.go`, `verified-code` |
| Connector functions have `REQUEST` and `RESPONSE` variants            | Three types: `REQUEST`, `RESPONSE`, **`COLLECTION`**                                                                                                                             | `pkg/opengate/connectors.go`, `verified-live`                       |
| Helper is `cf.collection()`; `collectCF()` is its deprecated form | **Settled by the platform team.** `collectionCF` does not exist and never did — it comes from the vendored copy of the guide, which this row was based on. `collectCF(data, criteria)` exists and is deprecated in favour of `cf.collection(criteria, payload)`, **arguments reversed**. See `docs/opengate-documentation-handoff-response.md` §4 | `from-docs`, settled 2026-08-26 |
| "South criteria are the routing key"                                  | Only for `RESPONSE`/`COLLECTION`. `REQUEST` matches on `operationName` + `northCriterias`                                                                                        | `cmd/connectors.go` long help, `verified-live`                      |
| `og pf test <slug> --input sample.csv`                                | The input is an **Excel spreadsheet**, not CSV                                                                                                                                   | `configurationParams.spreadsheet`, `verified-live`                  |

**Consequence:** Phases 2 and 3 as written are already delivered. What remains is the handoff's
real contribution: the safety layer (canonicalization → base state → diff → watch) and editor
intelligence (typegen), plus a smaller-than-advertised refactor. The plan gets *cheaper*, not
bigger.

---

## 2. Audit of the existing lifecycle

### 2.1 What is shared and what is duplicated

`internal/unwrap` (1484 lines including tests) is already factored around a common core:

| File                    | Role                                           | Reused by              |
| ----------------------- | ---------------------------------------------- | ---------------------- |
| `jsfields.go`           | JS extraction/reinjection by keypath           | **all five families**  |
| `slug.go`               | `Slugify`, `DedupedSlug`                       | all five               |
| `unwrap.go` / `wrap.go` | workspace → dashboard → widget nested pipeline | workspaces, dashboards |
| `rules.go`              | flat single-artifact adapter                   | rules                  |
| `connectors.go`         | flat single-artifact adapter                   | connector functions    |
| `provisions.go`         | flat single-artifact adapter                   | provision functions    |

The three flat adapters are the same 80 lines three times over. They differ in exactly three
values:

|                    | metadata filename        | name field                          | id field               |
| ------------------ | ------------------------ | ----------------------------------- | ---------------------- |
| rule               | `rule.json`              | `name`                              | `identifier`           |
| connector function | `connectorfunction.json` | `name` \|\| `connectorFunctionName` | `identifier`           |
| provision function | `provisionfunction.json` | `name`                              | `provisionProcessorId` |

**This is a table, not a class hierarchy.** The refactor the handoff calls Phase 1 collapses to
a descriptor struct plus one generic implementation — far less than the `Resource` interface it
proposes (see §5.1).

The `cmd/` layer duplicates more than `internal/unwrap` does: each family carries a near-identical
`unwrapXTo` helper that **recomputes the slug** that `unwrap.UnwrapX` computes again internally.
Two independent slug computations that agree only by coincidence.

### 2.2 The JS extraction heuristic — the central design problem

`shouldExtract(key, value)` extracts a string when **either** the key is in a hardcoded
allowlist **or** the value looks like JS by content (`len >= 40` and matching
`function|return|=>|const |let |var `).

Allowlist (`jsfields.go`): `formatter`, `script`, `operation`, `code`, `fn`, `expression`,
`_widgetconfigcode`, `javascript`.

Probes run against this logic (`verified-probe`):

| Case               | Input                                                                                                     | Result                                                           |
| ------------------ | --------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| **False negative** | `_formatterCode` = `cellFormatter.customValue = value; cellFormatter.style = '…';` (77 chars, no keyword) | **not extracted** — stays embedded as a `\n`-escaped JSON string |
| **False negative** | `_formatterCode` = `v => v + 1;` (11 chars)                                                               | not extracted (below the 40-char floor)                          |
| **False positive** | `description` = `"Please return to the main dashboard and let the operator know"`                         | extracted to `description.js`                                    |
| **False positive** | `markdown` = markdown prose containing "let"/"return"                                                     | extracted to `markdown.js`                                       |
| **False positive** | `operation` = `"REFRESH_INFO"` (an operation *name*)                                                      | extracted to `operation.js`                                      |
| **False positive** | `formatter` = `""`                                                                                        | extracted to an empty `formatter.js`                             |

`_formatterCode` — a field that carries real code in the demo corpus (`verified-code`:
five `columns__N___formatterCode.js` files) — **is not in the allowlist**. It is extracted only
because its content happens to trip the keyword pattern.

The consequence is the finding that governs the whole design:

> **The on-disk layout is a function of the file's content, not of the artifact's schema.**
> Edit a formatter down to an assignment without a keyword and, on the next `pull`, that `.js`
> file ceases to exist and the code reappears inside the JSON. The reverse also holds.

Everything downstream depends on a stable mapping between a file path and a payload field:
`diff` compares paths, `watch` resolves a changed path to a deployable field, `typegen` writes
a `.d.ts` next to the files it types. A content-dependent layout breaks all three. This must be
fixed **before** Phases 4–8, not during them.

Two further consequences of the "any `.js` in the directory gets reinjected" rule
(`wrap.go`, `rules.go`, `connectors.go`, `provisions.go` all glob `*.js`):

- A user cannot keep an auxiliary `helper.js` or `notes.js` in an artifact directory — it will be
  reinjected as a bogus payload field at its parsed keypath.
- Shared code between artifacts is therefore impossible today. Worth stating explicitly before
  someone designs a module system on top.

### 2.3 Round-trip losslessness — two real data-loss paths

`wrap` is documented as lossless "modulo cosmetic null/default differences". Two probes show it
is not (`verified-probe`):

**A. Numeric object keys become arrays.**

```
in : {"series":{"0":{"formatter":"function(v){return v;}"}}}
out: {"series":[{"formatter":"function(v){return v;}"}]}
```

`keyPath.filename()` encodes array indices and object keys identically (as decimal strings), and
`setAt` reverses any numeric segment into an array index (`strconv.Atoi`). A JSON object keyed by
number silently changes type across the cycle.

**B. The `__` separator collides with keys containing `__`.**

```
in : {"a__b":{"code":"…1"},"a":{"b":{"code":"…2"}}}
out: {"a":{"b":{"code":"…2"}}, "a__b":{}}
```

Both produce the filename `a__b__code.js`; one overwrites the other on disk, and the loser is
reconstructed **empty**. Silent code loss.

Neither is hypothetical-only in kind — both are structural properties of the filename encoding,
which is why Phase 4 (canonicalization) is worth doing with a real round-trip property test over
the whole corpus, as the handoff says. It just needs to *fail* on these two cases first, and the
encoding needs fixing.

### 2.4 Slug collisions in `pull-all` — a live bug

`DedupedSlug(name, id, taken)` is correct and tested. Its callers are not.

| Caller                                             | `taken` map                                 | Behaviour   |
| -------------------------------------------------- | ------------------------------------------- | ----------- |
| `cmd/dashboards.go:394` (pull-all)                 | shared across iterations                    | **correct** |
| `cmd/workspaces.go:508`                            | shared (`wsSlugs`)                          | correct     |
| `cmd/rules.go:378`                                 | `map[string]bool{}` literal, fresh per call | **broken**  |
| `cmd/connectors.go:384`                            | `map[string]bool{}` literal                 | **broken**  |
| `cmd/provision.go:387`                             | `map[string]bool{}` literal                 | **broken**  |
| `internal/unwrap/{rules,connectors,provisions}.go` | `map[string]bool{}` literal                 | **broken**  |

For rules, connector functions and provision functions, deduplication is therefore inert during
`pull-all`. Two artifacts with the same name resolve to the same directory, and:

- without `--force`: the pre-existence check aborts the **entire** `pull-all` mid-way, leaving a
  partial tree;
- with `--force`: the second artifact **silently overwrites** the first, and the user ends up
  believing they have a local copy of both.

Duplicate names are legal on the platform (the identifier is the key), so this is reachable.
`verified-code`.

**This is a bug fix, not a phase.** It should ship on its own, before any of this roadmap.

**Limitation that survives the fix** (found while implementing it, 2026-08-25): `DedupedSlug`
gives the *first* artifact of a colliding group the clean slug and suffixes the rest, so which
directory a given artifact lands in depends on the **order the API returns the list in**. If that
order changes between two `pull-all` runs, two same-named artifacts swap directories. This already
applied to dashboards and workspaces, which is why it was not introduced here.

A deterministic naming scheme — suffix *every* member of a colliding group, decided in a first
pass over the full list — would remove the dependency, at the cost of changing the directory name
of the artifact that currently keeps the clean slug. That is a layout change, so it belongs with
B3 and Phase 1, not in a patch. Until then: **resolve a directory to its artifact by reading the
identifier from its metadata JSON, never by assuming the slug**. Documented in `README.md` and in
the `og-device-ops` skill.

### 2.5 Ordering and path encoding

- Dashboard order inside a workspace, and widget order inside a dashboard, are encoded as a
  zero-padded `NN__` prefix; `wrap` recovers order by sorting directory names alphabetically
  (`collectGrid`, `Wrap`). Width is `max(len(itoa(n-1)), 2)`, so >100 widgets widen the prefix
  and change every sibling's name — a rename storm in `diff` and in git history. Low frequency,
  worth knowing.
- The workspace layout entry is smuggled into `dashboard.json` under a `_workspaceLayout` key and
  stripped on wrap. It is the one place a side channel already exists; the base-state design
  (§5.3) should not add a second one.
- `collectGrid` **silently skips** malformed widget directories (`continue` on error). A typo in
  one `widget.json` currently produces a successful deploy with that widget missing. This is
  exactly the failure class the roadmap exists to prevent, and it is one line.

### 2.6 Output, exit codes, validation

- `internal/output` offers `PrintJSON` / `PrintTable` / `Print`. There is no versioned JSON
  envelope; each command shapes its own payload. The handoff's `"schemaVersion": 1` contract has
  no existing scaffolding to hang on — it needs building (§5.5).
- Exit codes: `main.go` uses only `os.Exit(1)` on any error. The proposed `0`/`1`/`2` convention
  requires a typed error propagated through `cmd.Execute()`.
- There is **no** `validate` verb anywhere, and `deploy` performs no JS syntax check and offers
  no `--dry-run`. `verified-code`.

---

## 3. API surface — connector functions and provision functions

The handoff's highest-risk unknown. It is not unknown; it was implemented and verified live in
v0.9.0. Recorded here so Phase 0 is genuinely closed.

### 3.1 Endpoints

All on the **North API** (`/north/{v}/…`, version resolved per client) — *not* the Web API.
Auth is the JWT bearer token; none of these use the web-token path.

| Family       | Operation             | Path                                                                                | Confidence      |
| ------------ | --------------------- | ----------------------------------------------------------------------------------- | --------------- |
| Connector fn | list                  | `/north/{v}/connectorFunctions/provision/organizations/{org}/channels/{ch}`         | `verified-live` |
| Connector fn | get / update / delete | `…/channels/{ch}/{id}`                                                              | `verified-live` |
| Connector fn | create                | `POST …/channels/{ch}`                                                              | `verified-live` |
| Connector fn | catalog               | `/north/{v}/connectorFunctions/provision/catalog`                                   | `verified-live` |
| Connector fn | enable/disable        | **no endpoint** — GET + patch `operationalStatus` + PUT                             | `verified-code` |
| Provision fn | list                  | `/north/{v}/provisionProcessors/provision/organizations/{org}`                      | `verified-live` |
| Provision fn | get / update / delete | `…/organizations/{org}/{id}`                                                        | `verified-live` |
| Provision fn | plan (dry-run)        | `…/{id}/plan?numberOfEntriesToProcess=N` (multipart)                                | `verified-live` |
| Provision fn | bulk                  | `…/{id}/bulk` (multipart; `Accept` must match the upload's content type or **409**) | `verified-live` |
| Provision fn | bulk status / details | `…/bulk/{bulkId}` / `…/bulk/{bulkId}/details` (Excel; **204** while unfinished)     | `verified-live` |
| Provision fn | enable/disable        | **does not exist** — no status field on the resource                                | `verified-code` |

### 3.2 Scoping, identity, code location

|              | Scope             | ID field               | Code location            | Status field                                           |
| ------------ | ----------------- | ---------------------- | ------------------------ | ------------------------------------------------------ |
| Rule         | org + **channel** | `identifier`           | `javascript` (top level) | `active` (bool)                                        |
| Connector fn | org + **channel** | `identifier`           | `javascript` (top level) | `operationalStatus` ∈ {`DISABLED`,`PRODUCTION`,`TEST`} |
| Provision fn | **org only**      | `provisionProcessorId` | `scriptProcessor.script` | none                                                   |

API inconsistencies already absorbed by the client, worth preserving in any refactor:

- connector function name is `name` on write, sometimes `connectorFunctionName` on read;
- the provision-processor list array is `provisionProcessors` live but `processors` in the schema;
- a 404 on the provision-processor list means "none", not an error.

### 3.3 Connector function semantics

`from-docs` (`.claude/skills/og-device-ops/connector-functions-js-reference.md`, 2103 lines) plus
`verified-live`:

| Type         | Matched by                         | Must return                                                                |
| ------------ | ---------------------------------- | -------------------------------------------------------------------------- |
| `REQUEST`    | `operationName` + `northCriterias` | nothing / `null`                                                           |
| `RESPONSE`   | `southCriterias`                   | OpenGate Standard Response (`ogResponse`/`ogStep`/`ogStepResponse`)        |
| `COLLECTION` | `southCriterias`                   | Standard IoT Collection (`ogCollection`/`ogCollectionDs`/`ogCollectionDp`) |

Concatenation is a typed, restricted graph:

```
REQUEST  → RESPONSE, COLLECTION      (via cf.response / cf.collection)
RESPONSE → COLLECTION                (via cf.collection)
COLLECTION → nothing                 (any call is silently ignored)
```

The deprecated globals `responseCF(data, criteria)` and `collectCF(data, criteria)` do the same,
with the arguments the other way round. `collectionCF` does not exist and never did — that name came
from a stale vendored copy of the guide.

This makes `og cf graph` implementable by static analysis of the four call sites
(`cf.response`/`cf.collection` and their deprecated globals), and — more valuable — makes **illegal concatenation a lintable error** rather than a
silent no-op at runtime. That is a better payoff than the graph rendering itself.

### 3.4 Provision function contract

`verified-live` + `verified-code`. Answers handoff open question 4.

- Input: an **Excel spreadsheet**, configured on the resource under
  `configurationParams.spreadsheet` → `sheetName`, `headerRow` (number), `resultColumnName`.
- The script implements two entry points: **`normalizeRawObject(rawObject)`** and
  **`actionsPlanning(normalizedObject)`** (from the shipped test fixture, mirroring live payloads).
- A server-side dry run already exists: `plan` with `numberOfEntriesToProcess`, surfaced as
  `og provision plan`. **`og pf test` as specified in the handoff is largely redundant** — the
  platform will tell you what it would do, with real fidelity, which no local `goja` harness can
  match. See §6.2.

### 3.5 No optimistic concurrency

No endpoint exposes an ETag, revision, or version field on rules, connector functions or
provision functions (`verified-code`). Answers handoff open question 2: **conflict detection must
be local**; the server cannot enforce it.

A direct consequence, already live today: `SetRuleActive` and `SetConnectorFunctionStatus`
perform GET → patch one field → PUT. Two concurrent `og cf disable` calls, or one racing a web-UI
edit, will lose the other's changes wholesale. Not caused by this roadmap; worth recording.

---

## 4. Answers to the handoff's open questions

| #   | Question                                      | Answer                                                                                                                                                                                                                           |
| --- | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | North API or Web API?                         | **North API**, JWT bearer, for both families. Web API (`/api/…`, web-token + auto-refresh) is used only by workspaces, dashboards and share. `verified-code`                                                                     |
| 2   | Revision field for optimistic concurrency?    | **No.** Conflict detection must be local (§3.5).                                                                                                                                                                                 |
| 3   | Connector functions per channel or per org?   | **Per channel**, like rules. Provision functions are **per org**. `verified-live`                                                                                                                                                |
| 4   | Real input contract for provision functions?  | Excel spreadsheet + `normalizeRawObject`/`actionsPlanning`; server-side `plan` already exists (§3.4).                                                                                                                            |
| 5   | Should `og cf test` stub protocol execution?  | **Recommendation: no.** Ship static validation instead (§6.2).                                                                                                                                                                   |
| 6   | Governance: community tool or Amplía product? | **Decided 2026-08-25: community tool, for now.** Charlie has yet to discuss it with others at Amplía, so treat it as provisional — revisit before the extension ships (§9). Interacts with `docs/premium-open-core-analysis.md`. |

---

## 5. Proposed design

### 5.1 `internal/artifact` — descriptor, not interface hierarchy

The handoff's `Resource` interface has seven methods and would be implemented five times. Given
§2.1, three of those five implementations are the same code. Proposal: a **descriptor table**
driving one generic implementation, with an escape hatch for the nested family.

```go
// Kind identifies a family of platform artifacts supporting the
// pull → edit → deploy lifecycle.
type Kind string

const (
    KindWorkspace         Kind = "workspace"
    KindDashboard         Kind = "dashboard"
    KindRule              Kind = "rule"
    KindConnectorFunction Kind = "connector-function"
    KindProvisionFunction Kind = "provision-function"
)

// Descriptor declares everything that distinguishes one flat artifact family
// from another. The pull/wrap/diff/watch machinery reads this table; it is not
// implemented per family.
type Descriptor struct {
    Kind     Kind
    MetaFile string   // "rule.json", "connectorfunction.json", …
    NameKeys []string // ["name", "connectorFunctionName"] — first non-empty wins
    IDKey    string   // "identifier" | "provisionProcessorId"
    Scope    Scope    // ScopeOrg | ScopeOrgChannel

    // CodePaths is the authoritative list of keypaths carrying source code.
    // This replaces the content heuristic (§2.2) — the on-disk layout becomes
    // a function of the schema, not of the file's contents.
    CodePaths []CodePath

    // Volatile keypaths are server-managed and excluded from comparison.
    Volatile []string
}

// CodePath binds one payload keypath to one on-disk file and one execution
// context, so typegen and validation know what they are looking at.
type CodePath struct {
    Path    string  // "javascript", "scriptProcessor.script"
    File    string  // "javascript.js", "code.js"
    Context ExecCtx // rule-advanced | cf-request | cf-response | cf-collection | provision | widget-formatter
}
```

Rules, connector functions and provision functions become three `Descriptor` literals. Workspaces
and dashboards keep their nested pipeline behind the same outer API — the honest split, rather
than forcing a widget tree through a flat interface.

**Acceptance criteria** (unchanged from the handoff, and the right ones): identical CLI surface,
identical output, existing tests pass untouched.

### 5.2 Code extraction: allowlist per family, heuristic demoted

Replace `shouldExtract`'s content sniffing with `Descriptor.CodePaths`:

1. A declared keypath is **always** extracted, to a **stable** filename — even when empty, even
   when the content does not look like code.
2. An undeclared string that trips the heuristic is **left in place** and reported as a warning
   (`og <family> pull` prints `hint: config.foo looks like code but is not a declared code path`).
   That surfaces gaps in the table without letting content decide the layout.
3. `wrap` reinjects only files declared in `CodePaths` (plus discovered widget keypath files for
   the nested family). Undeclared `.js` files are **ignored with a warning** instead of being
   injected as bogus fields — which incidentally makes `helper.js` and future shared modules
   possible.

Widgets are the hard case: their code fields are widget-type-specific
(`_widgetConfigCode`, `_formatterCode`, `columns[].formatter`, …). Proposal: a per-widget-type
allowlist seeded from `.claude/skills/og-workspaces/reference/` plus the demo corpus, with the
heuristic retained **for widgets only** as a warning-generating fallback during a transition
period. `_formatterCode` goes in the allowlist on day one (§2.2).

### 5.3 Filename encoding — fix before building on it

The `__`-joined keypath (§2.3) is ambiguous in two directions. **Implemented 2026-08-25, and
cheaper than this section originally proposed** — no file in any existing tree is renamed:

- **Numeric object keys.** Tagging array indices as `[0]` turns out to be unnecessary. The cleaned
  document still carries the original container at each position — `walkExtract` only removes leaf
  strings, so `{"series":{"0":{…}}}` stays an object — which means `setAt` can read the container's
  concrete type instead of inferring it from the segment's spelling. It only has to guess when the
  container is absent, and there a numeric segment does mean an array. `columns__0__formatter.js`
  keeps its name.
- **Separator collision.** Percent-escaping is applied only to runs of *two or more* underscores
  inside a segment (and to `%`, to keep the escape unambiguous). A lone underscore cannot be
  confused with the separator, so `_widgetConfigCode.js` and `columns__2___formatterCode.js` are
  spelled exactly as before. Files written before escaping existed contain no `%`, so they parse
  unchanged.

So B3 is not the disruptive layout change §9.2 assumed: no migration note, no re-pull. What *does*
change shape is the extraction contract below, and only for fields that were being extracted by
accident.

### 5.4 Canonicalization and base state

As the handoff specifies, with two adjustments:

- `VolatileFields` comes from `Descriptor.Volatile`, so there is one table, not two.
- The base snapshot must also record **where the artifact came from** — profile, org, channel.
  Today nothing does (§2.1), so a `pull` from staging can be `deploy`ed to production with no
  warning. The manifest fixes a live footgun, not just a future one:

```
<artifact-root>/.og/
  base/<artifact-id>.canon.json
  manifest.json   # {artifactID: {hash, pulledAt, profile, org, channel, kind}}
```

`.og/` is a cache: gitignored, and `deploy` warns when the target profile/org/channel differs
from the manifest.

The three-way classification table in the handoff (§8 there) is adopted as-is. It is correct, and
§3.5 means it is the only conflict detection available.

### 5.5 JSON contract and exit codes

Needs building from scratch (§2.6). Proposal: one envelope in `internal/output`, versioned once,
reused by every `-o json` command:

```json
{"schemaVersion": 1, "kind": "diff", "data": {…}}
```

and a typed `cmd.ExitError{Code int}` honoured by `main.go`, so `0`/`1`/`2` becomes real. Both are
prerequisites for `diff --exit-code` in CI and for the extension.

### 5.6 `watch` — the session landmine is narrower than stated

The handoff warns that continuous re-signing invalidates the developer's browser session. Audit
result (`verified-code`): the transparent 401 re-sign lives in `newWebClient` and is therefore
used **only** by workspaces, dashboards and share. Rules, connector functions and provision
functions go through the North API JWT client, which has no re-sign path.

So the warning is real but scoped: it applies to `og workspace watch` / `og dashboard watch`, not
to `og cf watch`. Print the warning where it applies rather than globally — a warning that fires
on every command is a warning nobody reads.

The rest of §10 of the handoff (ignore patterns, one worker per target with ~300 ms coalescing,
validation on by default, `production: true` profile guard, no `--force`) is sound and adopted.
`Profile` (`internal/config/config.go`) gains one `mapstructure:"production"` bool; `ActiveProfile`
already resolves profiles by name, which is all `--against <profile>` needs.

### 5.7 Typegen — the source is already in the repo

The handoff proposes sourcing protocol signatures from `documentation.opengate.es`. They are
already vendored, versioned, in the skills:

| Source                                                             | Lines | Covers                                                                                                                                                                                                                               |
| ------------------------------------------------------------------ | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `.claude/skills/og-device-ops/connector-functions-js-reference.md` | 2103  | `entity`, `gateway`, `payload`, `contextParams`, `response`, `collection`, `cf`, `utils`, `logger`, `operation`, `crypto` + protocols `http`, `mqtt`, `snmp`, `dlms`, `iec102`, `icmp`, `coap`, `ssh`, `telnet`, `websocket`, `kite` |
| `.claude/skills/og-device-ops/rules-js-reference.md`               | 1849  | ADVANCED rule context                                                                                                                                                                                                                |
| `.claude/skills/og-workspaces/reference/widget-js-api.md`          | 224   | widget formatter/config context                                                                                                                                                                                                      |

Design: the **static** half of each `.d.ts` is a hand-written template per execution context,
embedded with `go:embed` (the catalogue changes with platform releases, not with the user's org).
The **generated** half comes from the org's datamodel. Do not parse the markdown at runtime.

The datamodel gives more than the handoff assumes. `pkg/opengate.Datamodel` exposes
`Categories[].Datastreams[]` with `Identifier`, **`Schema`** (a JSON Schema) and `Unit`. So the
generated half can type the *value*, not just the id:

```typescript
type DatastreamID = 'sensor.temperature' | 'sensor.humidity' | 'power.energy' | …;

interface Sample<T> { _current: { value: T; date: string; at: string; provType?: string }; }
declare const entity: { 'sensor.temperature'?: { _value: Sample<number> }, … };
```

One correction to the handoff's proposed shape, from the platform reference (`from-docs`): an
**indexed** datastream is an *array* of `{_index, _value}`, not a bare `{_value}` —
`entity['provision.device.communicationModules[].identifier'][0]._value._current.value`. And
`_current` carries `date`/`at`/`provType`, not just `value`/`at`. Getting this wrong in the
typings would be worse than having none.

Record the source document version in the generated header, as the handoff says.

---

## 6. Where I disagree with the handoff

### 6.1 Phase 1 before typegen is the wrong order — but so is the handoff's order

The handoff sequences typegen-for-rules first (ship value in days), then the refactor. Correct
instinct, wrong dependency: typegen writes `og-globals.d.ts` and `jsconfig.json` **into artifact
directories**, and `wrap` currently reinjects every `.js` it finds (§2.2). `.d.ts` is safe by
extension, but the moment typegen wants a `helper.js` or the allowlist changes shape, it breaks.

Ship in this order instead: the extraction contract (§5.2) → typegen for rules → the descriptor
refactor. The contract is a prerequisite for both.

### 6.2 `og cf test` / `og pf test` with stubbed protocols: don't

The handoff hedges on this ("degrade to syntax-and-static-check if stubbing proves unreliable").
Take the hedge as the plan:

- For provision functions, the platform already offers a **real** dry run (`plan`, exposed as
  `og provision plan`). A `goja` simulation is strictly worse than a server-side plan.
- For connector functions, faithful stubs for eleven protocols plus `entity`/`collection`/`response`
  semantics is a large surface with no fidelity guarantee, and `operationalStatus: TEST` plus
  `og connectors logs` already provides on-platform testing that og can drive.
- What is *not* covered today and is cheap: **static validation** — JS parse (goja parse-only),
  required entry points per type (`normalizeRawObject`/`actionsPlanning` for PF; return-shape for
  CF), illegal concatenation (§3.3), south-criteria format, declared-vs-actual code paths.

Proposal: ship `og <family> validate` (feeding `--validate` on deploy, `watch`, and the extension's
diagnostics) and **drop `test`**. Revisit only if a concrete demand appears.

### 6.3 `og cf graph` is a nice-to-have, the concatenation lint is the value

Same analysis as above: the graph is a rendering; the lint catches a class of silent runtime
no-ops. Fold the graph into `validate`'s data and render it later if anyone asks.

---

## 7. Revised sequencing

Bugs first — they are cheap, they are live, and two of them corrupt the local tree that
everything else builds on.

| #      | Work                                                                            | Depends on | Notes                                          |
| ------ | ------------------------------------------------------------------------------- | ---------- | ---------------------------------------------- |
| **B1** | Fix `pull-all` slug collisions (§2.4)                                           | —          | share one `taken` map; single slug computation |
| **B2** | Fail loudly on malformed widget dirs (§2.5)                                     | —          | one-line `continue` → error                    |
| **B3** | Fix the filename encoding (§5.3)                                                | —          | prerequisite for any hashing                   |
| **1**  | Extraction contract: `CodePaths` allowlist, heuristic demoted to warning (§5.2) | B3         | the keystone                                   |
| **2**  | Typegen for ADVANCED rules (§5.7)                                               | 1          | first visible win, works in Neovim immediately |
| **3**  | `internal/artifact` descriptor refactor (§5.1)                                  | 1          | behaviour-preserving; **checkpoint**           |
| **4**  | `internal/canon` + round-trip property test over `demo/`                        | 3, B3      | must fail on §2.3 first                        |
| **5**  | JSON envelope + typed exit codes (§5.5)                                         | —          | small; unblocks CI and the extension           |
| **6**  | Base snapshots + manifest with profile/org/channel (§5.4)                       | 4          | also fixes the cross-tenant deploy footgun     |
| **7**  | `og <family> validate` (§6.2)                                                   | 1, 5       | replaces the handoff's `test`                  |
| **8**  | `diff`, incl. `--against <profile>` (§5.5, handoff §9)                          | 4, 5, 6    |                                                |
| **9**  | `watch` (§5.6)                                                                  | 6, 7, 8    | scoped session warning; production guard       |
| **10** | Typegen, all contexts + datamodel-derived types (§5.7)                          | 2, 3       |                                                |
| **11** | VS Code extension                                                               | 1–10       | governance decision (§4 q6) first              |

Phases 2 and 3 of the handoff are struck: already shipped.

---

## 8. Decisions taken

**2026-08-25, Charlie:**

1. **Governance → community tool, for now.** Provisional: to be discussed with others at Amplía.
   Consequences recorded in §9.
2. **B1–B3 ship now**, as a patch release, independently of whether phases 1+ are approved.
   Caveat surfaced after the decision: **B3 is not a bug fix of the same nature as B1/B2** — it
   changes the on-disk filename format. See §9.2.

## 8b. Still open for the human

1. **Command naming.** `og cf` already aliases `og connectors`, `og pf` does not alias
   `og provision`. Add the `pf` alias for symmetry, or leave it?
2. **Is `--against <profile>` cross-tenant diff worth pulling earlier?** It is the flag the
   handoff predicts will be most used, and it needs only canon + two clients — it could land
   before base snapshots, without three-way state.
3. **Scope of `validate` on `deploy`.** Non-overridable for provision functions (the handoff
   suggests this, given bulk data corruption at scale), or a warning with `--no-validate`
   everywhere?
4. **Widget code-field allowlist.** Seeding it from the skills + demo corpus will be incomplete.
   Accept a transition period where the heuristic still fires as a warning, or block on
   enumerating widget types against a live tenant first?

---

## 9. Consequences of the decisions

### 9.1 Community tool (provisional)

- The unofficial/no-warranty framing in `README.md` stands; nothing needs softening yet.
- `watch` still ships the `production: true` profile guard and `--allow-production` (§5.6). A
  community tool writing to a customer's production platform is *more* reason for the guard, not
  less — the tool carries no support channel to catch the fallout.
- Phase 11 (VS Code extension) stays gated: publishing to a marketplace under a community banner
  is a distribution decision, not a technical one. Revisit when the Amplía conversation happens.
- No change to the roadmap's technical content.

### 9.2 B1–B3 as a patch release — with one correction

B1 and B2 are genuine bug fixes: small, local, behaviour-restoring, no format change.

| Fix                                             | Blast radius                                                                                        |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| **B1** slug dedup in `pull-all` (§2.4)          | 3 call sites in `cmd/` + 3 in `internal/unwrap`; collapse the double slug computation while there   |
| **B2** malformed widget dirs fail loudly (§2.5) | one `continue` → error in `collectGrid`; changes `wrap` from "silently drops a widget" to "refuses" |

**B3 is different and should not ride the same release.** Fixing the filename encoding (§5.3)
changes the names of files already sitting in users' working trees and in `demo/`. Even with
read-both-forms/write-new-form compatibility, a patch release that renames files under someone's
editor is not a patch. Options:

- **(a) Recommended:** B1+B2 as a patch (e.g. v1.7.1). B3 lands with Phase 1 (the extraction
  contract), which is where the layout is deliberately redefined anyway, in a minor release with
  a migration note.
- **(b)** B3 in the patch, guarded behind a flag defaulting to the old encoding. Adds a flag that
  exists only to be removed later.

Under (a) the round-trip property test of Phase 4 must be written to **fail** on the §2.3 cases
from day one, so the debt stays visible rather than being quietly deferred.

---

## 10. Phase 1 as built (2026-08-25)

Branch `og-cli/dx-extraction-contract`, on top of v2.0.2.

`internal/unwrap/contract.go` introduces `CodeContract`, which each family supplies:

| Family | Declared code path(s) | Execution context |
|---|---|---|
| Rule | `javascript` → `javascript.js` | `rule/ADVANCED` |
| Connector function | `javascript` → `javascript.js` | `connector-function/{REQUEST,RESPONSE,COLLECTION}`, from the artifact's `type` |
| Provision function | `scriptProcessor.script` → `scriptProcessor__script.js` | `provision-function` |
| Widget | field names at any depth: `_widgetConfigCode`, `_formatterCode`, `formatter`, `script`, `code`, `fn`, `expression` | `widget` |

Behaviour changes:

1. **A declared field is always extracted**, even when empty or when its content does not look
   like code. This is the point: the path stops depending on the contents (§2.2).
2. **`operation` is no longer a code field.** It holds an operation name (`REFRESH_INFO`); it was
   producing one-word `.js` files.
3. **Undeclared strings that look like code** are left inline and reported on stderr for the flat
   families, whose paths are exhaustive. For **widgets** the heuristic still extracts them — with a
   warning naming the field — because the per-widget-type census is incomplete and refusing would
   cost a user their editable file. Per the decision in §8b.4 (transition period), with one
   deliberate deviation from the option as worded: the fallback still *writes* the file for
   widgets rather than only warning.
4. **A `.js` file the contract does not declare is never injected** as a payload field on wrap; it
   is reported and skipped. That makes `helper.js` and generated typings safe to keep in an
   artifact directory — a precondition for Phase 8 (typegen) and for any future shared modules.
5. The execution context is recorded per extracted file (`ExecCtx`), which is what Phase 8 will
   consume. It is computed and currently discarded by the callers; nothing persists it yet.

Verification: the whole `demo/` corpus wraps **byte-identically** to v2.0.2 (workspace with 7
widgets and 5 column formatters, both ADVANCED rules), and the run emits **zero warnings** — the
declared widget fields cover the real corpus. Lint stays at the 42-issue baseline.

Not addressed here, still open for Phase 4: `Hash(Explode → Implode) == Hash(original)` as a
property test over the corpus. The two data-loss paths it was meant to catch are now fixed and
covered by unit tests, but the corpus-wide property test does not exist yet.

---

## 11. Phase 2 as built (2026-08-25)

`internal/typegen` + `og typegen`, and `og rules pull` writes the two files next to the code
(`--no-typings` opts out).

Two halves, as designed in §5.7:

- **Platform catalogue** — `templates/rule-advanced.d.ts`, embedded with `go:embed`, hand-derived
  from the vendored `rules-js-reference.md`: ~65 functions and objects (`alarm`, `notification`,
  `operation`, `collect`, `provision`, `logger`, `utils`, `location`, `http.client`, the date and
  counter helpers), the entity value shape, and `OGSeverity`/`OGPriority` as real unions.
- **Generated from the tenant** — the organization's datamodel becomes the `OGDatastreamID` union
  and the `OGEntity` keys, each typed from its JSON Schema (`{"type":"number"}` → `number`), with
  the datastream's name, description, unit and category as its doc comment.

Two things went beyond the plan:

1. **Rule parameters are typed too.** A rule declares `parameters: [{name, value, schema}]`, so
   `parameterObject` gets a real interface instead of `Record<string, unknown>` — a mistyped
   parameter name is an error, and arithmetic against a numeric parameter type-checks. Discovered
   because the *demo rule* failed to type-check against the first version of the declarations:
   `Record<string, unknown>` made `temp > tempThreshold` an error on correct code.
2. **Indexed datastreams are declared as arrays** of `{_index, _value}`, and `_current` carries
   `date`/`at`/`source`/`sourceInfo`/`provType` — the corrections noted in §5.7.

### Verified with a real type-checker

Not "should work": `tsc 5.9` was run against the generated files.

| Case | Result |
|---|---|
| `entity['sensro.temperature']` | `Property 'sensro.temperature' does not exist on type 'OGEntity'. Did you mean 'sensor.temperature'?` |
| `entity['device.pressure']` (not in this datamodel) | `TS7053: … can't be used to index type 'OGEntity'` |
| `alarm.open({severity: 'HIGH'})` | `Type '"HIGH"' is not assignable to type 'OGSeverity'` |
| `alarm.opne(...)` | `Property 'opne' does not exist` |
| `._curent.value` | `Did you mean '_current'?` |
| `loger.info(...)` | `Cannot find name 'loger'. Did you mean 'logger'?` |
| **the demo's real rule** (`env-anomaly/javascript.js`, unmodified) | **clean, exit 0** |

### Two decisions worth recording

- **`noImplicitAny: true` in the generated `jsconfig.json` is load-bearing.** With `strict: false`
  alone, TypeScript silently types an unknown index as `any` and the single most valuable check —
  the mistyped datastream identifier — never fires. Found because the first run caught only 4 of
  the 6 seeded errors. `strict` itself stays off: `strictNullChecks` would flag every read of an
  optional datastream, which is noise a user would silence by deleting the file.
- **`getVariableValue` is typed as `(v: T) => T`, not `T | ''`.** The honest union (it does return
  `''` for undefined) makes every arithmetic use of a rule parameter an error, putting correct code
  in red. The deviation is documented in the declaration itself.

`og typegen` is CLI-only, consistent with the surface split: it writes to arbitrary local
directories, which is the one category deliberately kept out of MCP.

Still open from §5.7: contexts other than `rule/ADVANCED` (connector functions per protocol,
provision functions, widget formatters) — that is Phase 10.

---

## 12. Phase 3 as built (2026-08-25)

### Deviation from §5.1: no new package

§5.1 proposed `internal/artifact`. Built instead as `internal/unwrap/descriptor.go`, deliberately:

`internal/unwrap` **is** the artifact package — its doc comment already says so, and the shared
machinery (`CodeContract`, `Options`, `DedupedSlug`, `ExtractJSFields`, the nested widget walker)
all lives there. A sibling `internal/artifact` would have to either import `unwrap` for all of it
(leaving the family declarations in one package and the lifecycle in another) or move 1500 lines
across. Renaming the package `unwrap` → `artifact` would be cleaner conceptually and is pure churn
across 20 files for no functional gain. Same benefit, a fraction of the risk. Revisit only if a
second consumer of the abstraction appears.

### What it does

`Descriptor` declares the three literals that distinguish a flat family, plus a contract resolver:

| | MetaFile | NameKeys | IDKey |
|---|---|---|---|
| `RuleDescriptor()` | `rule.json` | `name` | `identifier` |
| `ConnectorFunctionDescriptor()` | `connectorfunction.json` | `name`, `connectorFunctionName` | `identifier` |
| `ProvisionFunctionDescriptor()` | `provisionfunction.json` | `name` | `provisionProcessorId` |

`Descriptor.Unwrap` / `Descriptor.Wrap` implement the lifecycle once. `Contract` is a function of
the decoded payload rather than a constant, because a connector function's execution context
follows its `type` field — which also removed the `cfTypeOf` helper that re-read the metadata file
from disk just to resolve it.

In `cmd/`, the three near-identical `unwrapXTo` helpers collapse into one `unwrapArtifactTo` taking
a descriptor; the descriptor knows which key holds the name, so the error message no longer needs a
per-family summary parser.

The existing exported functions (`UnwrapRule`, `WrapConnectorFunction`, …) survive as two-line
wrappers. That is what makes the refactor behaviour-preserving by construction: no call site and no
test changed.

### Honest accounting

This does **not** reduce the line count — 168 lines removed, 168 added in `descriptor.go`, roughly
neutral. What it buys is a single place where the flat lifecycle lives, and a new family becoming a
ten-line declaration. It is a prerequisite for Phases 4, 6 and 7, which need to iterate over
families rather than special-case three of them.

Two fields §5.1 proposed are **not** implemented: `Scope` and `Volatile`. Nothing consumes them
yet, and `Volatile` would have to be invented rather than verified — it arrives in Phase 4, from
real payloads.

### Verification

Beyond the suite: the Phase 2 binary and the Phase 3 binary were run side by side over nine cases —
the two demo rules, a connector function, a provision function, a workspace, a dashboard, two
error paths (missing metadata file, nonexistent directory) and `rules --help`. **stdout, stderr and
exit code are byte-identical in all nine.** Lint stays at 42.

---

## 13. Phase 4 as built (2026-08-26)

`internal/canon`: `Canonicalize`, `Hash`, `Equal`, plus `VolatileFields`/`Diagnose` so a command can
tell the user what it ignored. 90.8% statement coverage.

Canonical form: keys sorted (`json.Marshal` of a map does this), nulls and empty containers dropped,
volatile fields removed. Arrays are **not** reordered — a dashboard's grid order and a rule's
datastream order are meaningful.

### Volatile fields — verified, not invented

Read off real API responses in `pkg/opengate/testdata`:

| Field | Why it never participates |
|---|---|
| `__v` | Mongo document version, bumped on every save |
| `lastAccess` | timestamp of the last read |
| `allowedProfiles` | **derived from the profile making the request** |
| `editable` | **derived from whether the requester owns the artifact** |

The last two are why two developers pulling the same dashboard get different JSON — a diff would
otherwise report a change nobody made.

Rules, connector functions and provision functions have **no** verified volatile field, so their
tables are **empty on purpose**, with a test asserting that intent. An invented entry would hide a
real change, which is worse than an unfiltered one. Fill them from a live GET when one is available.

### Two scopes, which §7 did not anticipate

`_id`/`id`/`owner` are stable within a tenant but differ by construction between tenants. One list
cannot serve both, so identity fields are separate from volatile ones and only dropped under
`CrossTenant` — the scope `diff --against <profile>` will use. Same-tenant comparison still reports
a changed identifier, which there means something is wrong.

### Volatile removal is recursive — found by the test failing

A workspace payload **embeds its dashboards**, each carrying its own `__v`, `lastAccess`,
`allowedProfiles` and `editable`. Stripping only the top level left every nested dashboard
generating noise; the workspace fixture round-trip failed until removal became depth-wide.

The trade-off, documented in the code: a field with one of these names and a genuine meaning deeper
in the tree — an `editable` inside a widget config — is ignored too. That risks a missed difference
in an unlikely place against guaranteed noise in a common one, and it only ever affects comparison,
never a deployed payload.

### The property test, and its detection power

The acceptance criterion — explode → implode preserves the artifact — now runs over the real corpus:
the `dashboard_get.json` response (three real widgets), `workspace_get.json`, every workspace in the
`workspace_export.json` envelope, both demo rules, a REQUEST and a COLLECTION connector function, a
provision function, and an idempotence check (a second cycle changes nothing further, so `diff`
cannot report a phantom change on first save).

**A property test that always passes is worth nothing, so both bugs were reintroduced to check it
fails.** The first attempt did not: the regression cases had been routed through `RuleDescriptor`,
whose contract declares only `javascript` — nothing was extracted, so `setAt` was never reached and
the test was vacuous. They now go through the real dashboard→widget cycle on disk, where the shapes
actually occur. Verified: reverting the container-type fix fails exactly the two numeric-key
subtests; reverting the separator escaping fails exactly the two underscore subtests.

### Coverage limit, stated rather than skipped

No fixture exercises workspace → dashboard → widget in one pass, because no single API response
contains all three: a workspace GET embeds its dashboards' layout but not their grids. Workspace →
dashboard is covered by the workspace fixtures, dashboard → widget by `dashboard_get.json`. The
`workspace_export.json` envelope (`{"workspaces":[…]}`) is now iterated rather than skipped.

---

## 14. Phase 5 as built (2026-08-26)

`internal/basestate`: a `.og/` store at the root of a pull, holding a canonical snapshot per
artifact plus a manifest recording provenance. 74.2% statement coverage.

```
rules/
  .gitignore          ← pull adds .og/ to it
  .og/
    base/rule__r-1.canon.json
    manifest.json     {schemaVersion, entries: {"rule:r-1": {hash, baseFile, dir,
                       profile, host, org, channel, pulledAt, kind, id, name}}}
  env-anomaly/
    rule.json  javascript.js  og-globals.d.ts  jsconfig.json
```

### The classification, and one case the design missed

| local vs base | remote vs base | State | marker | safe to deploy |
|---|---|---|---|---|
| = | = | clean | (space) | yes |
| ≠ | = | local changes | `~` | yes |
| = | ≠ | remote changes | `↓` | no |
| ≠ | ≠ | **conflict** | `!` | no |
| — | — | **unknown** (no base) | `?` | yes |

Two additions to the handoff's table:

- **Converged edits are not a conflict.** When both sides moved to the *same* value — two people
  making the same change — blocking would be theatre. Compared by hash, so it costs nothing.
- **`Unknown` must not block.** Every tree pulled before this existed has no base snapshot, and
  refusing to deploy those would break the current workflow for no safety gain.

### Design decisions worth recording

- **The store is found by walking up**, the way git finds its repository. `deploy` receives an
  artifact directory (`rules/env-anomaly`), while the store lives at the pull root (`rules/`), so a
  fixed relative path would not work.
- **Entries are namespaced by kind.** Nothing guarantees a rule and a connector function cannot
  share an identifier.
- **The snapshot filename is recorded, not derived.** An artifact identifier is not guaranteed to be
  a safe filename, and two sanitised identifiers could collide — deriving it would silently alias
  one artifact's base onto another's.
- **Empty recorded fields are not compared.** An entry written before a field existed must not raise
  a false alarm about it.

### What is visible today

The footgun closes: `deploy` warns when aimed at a different host, profile, organization or channel
than the artifact was pulled from. Verified end to end against a simulated staging tree — it names
all three differences, then lets the command proceed. It **warns rather than blocks**: promoting an
artifact between tenants is the documented cross-tenant workflow, it just should not happen by
accident. Silence when the target matches, and silence when there is no store at all.

`Classify` itself has no consumer yet — it is what Phase 6 (`diff`) and Phase 7 (`watch`) are
written against. It ships now, with the four-outcome table under test, because the base snapshots it
reads have to be recorded from this release onward for it to have anything to work with later.

---

## 15. Phase 6 as built (2026-08-26)

`internal/diff` plus `og rules diff`, `og connectors diff`, `og provision diff`. 83.2% coverage.
Also lands sequencing item 5 (the JSON envelope and typed exit codes), which `diff` needs.

### Two diffs, kept apart

Metadata is structural over the canonical form (`+`/`-`/`~` with a dotted path); code is a textual
unified diff. Mixing them is what makes generated diffs unreadable — a structural diff of a
1500-character script reports that one string changed, and a textual diff of reordered JSON reports
the whole document.

The textual diff is a line-level LCS written here, about thirty lines: this is a lean single binary,
and a dependency would only earn its place if word-level or move detection were wanted. Long
unchanged stretches collapse to `… N unchanged lines`, or a one-line change in a 500-line rule
prints the whole file.

### One direction for the whole report — found by reading the output

The first working version read **local → remote** for metadata and **remote → local** for code,
because the code direction had been chosen deliberately ("what deploying would do") and the metadata
direction had not been chosen at all. Two directions in one report is worse than either. Both now
read remote → local, the same convention as `git diff`: the report is what deploying would do to the
platform.

### Flags

| Flag | Notes |
|---|---|
| `--name-only` | the artifact and its state, no bodies |
| `--exit-code` | `1` differences, `0` none, `2` error — the CI drift gate |
| `--against <profile>` | cross-tenant: identity fields are ignored, since they differ by construction |
| `--context N` | lines of context around each code change (default 3) |
| `-o json` | the versioned envelope, documented in `docs/json-output.md` |

### Exit codes needed Cobra's error printing moved

`--exit-code` returning `1` for "differences found" must not print `Error:` — finding differences is
the command working. That needs an error that carries a code and no message, which means Cobra can no
longer print errors itself (`SilenceErrors`), so `main` prints them instead, in the same format.
Verified: an ordinary failure still prints `Error: …` and exits `2`; a successful command exits `0`.

### Verified end to end, not just in tests

A fake OpenGate served a known rule payload on localhost while the real binary diffed a local tree
against it. All four states render with their markers — `~ local changes`, `↓ remote changes`,
`! conflict`, and `?` with no snapshot — and `--exit-code` returns 1 with differences, 0 without.

### Scope note

The three flat families have `diff`. Workspaces and dashboards do not yet: the engine is
family-agnostic and their payloads canonicalize fine, but the useful rendering for them is the
hierarchical one from the handoff's §9 example — a tree of dashboards and widgets, each with its own
marker — rather than one flat list of `dashboards[0].dashboard.grid[2]…` paths. That is a rendering
job, not an engine one, and it is deliberately left for its own change.

---

## 16. Validation against the real platform (2026-08-26)

Everything above had been verified against tests, fixtures and a fake OpenGate served on localhost.
Charlie's instruction was to use the real thing instead: the `sensehat` organization on
`api.opengate.es`. Read-only, and logged in with `--no-web` so no browser session could be
invalidated (the rule/CF/PF path is North API only — see §5.6).

It worked, and it found four things no fixture had.

### What held

- `og rules pull` of a live ADVANCED rule produced the tree, the typings and the `.og/` store.
- **`og rules diff` on the untouched tree reported "No differences", exit 0.** That is the real
  test of canonicalization: a live payload, with whatever the platform actually returns.
- After editing the code and the description, the diff reported exactly those two things, with the
  right state (`~ local changes`) and the recorded origin.

### 1. Identity field names were wrong

A rule GET returns **`organization`** and **`channel`**. The table had `organizationId` and
`channelId`, taken from the search summary struct. Consequence: `diff --against` would have reported
both fields as differences on **every single rule**, which is exactly the noise identity fields
exist to remove. Both spellings are now listed.

### 2. A connector function has a server-managed `errors` field

Not in any fixture. Always `null` on sensehat's current functions, so `clean()` drops it today, but
the platform writes it when a function fails — it would surface as "your change". Now listed as
volatile, with a note to remove it if it turns out to be editable.

The `TestFlatFamiliesHaveNoVolatileFieldsYet` test failed when this was added, which is exactly what
it was written for: it forces any addition to be deliberate rather than incidental.

### 3. One datamodel per organization is a fiction

sensehat has **27 datamodels, 664 datastreams**. The first implementation refused to choose and
generated platform datastreams only — throwing away the whole point of the feature on any real
tenant.

They are now merged. The datamodel *search* response already carries every model's categories and
datastreams, so this costs one request, not 27. Where two datamodels declare the same identifier
with different types, the entry is left `unknown` and says so: asserting either would flag correct
code that used the other.

### 4. Live rules reference datastreams no datamodel declares

The rule that was pulled triggers on **`temperature.from.pressure`**, which appears in **none** of
the 27 datamodels. Strict typings generated from datamodels alone would have marked that rule's
working code as an error — and typings that redden correct code get deleted, taking the feature with
them.

Two further sources now feed the declarations, both specific to the artifact being pulled:

- the rule's own trigger (`type.datastreams[].name`), which the platform guarantees will be present;
- the identifiers the code already reads, found by matching `entity['…']` / `gateway['…']`.

Such entries are declared `unknown` — nothing states their type — and say why in their doc comment.

**The trade-off, stated plainly:** a typo already present in the code when the typings were
generated is declared, and therefore not flagged. Inventing errors in code that works is the worse
failure. A typo written *afterwards* is still caught, which is the case that matters while editing.

Verified with `tsc 5.9` against the real artifact: the production rule's code type-checks **clean**,
and typos added afterwards are still reported —
`Property 'temperature.from.presure' does not exist… Did you mean 'temperature.from.pressure'?` —
including on the identifier that no datamodel declares.

---

## 17. Phases 7 and 9 as built (2026-08-26)

`internal/validate` + `og <family> validate`, and `internal/watch` + `og <family> watch`. 85.7% and
77.9% coverage. Both exercised against the real `sensehat` organization, per Charlie's standing
instruction that there be no mocks.

### validate, and the false positive that nearly shipped

Checks: metadata parses (with the line number, so a stray comma is findable), declared code files
present, brackets balance, and the per-family traps — a REQUEST connector function with no
`operationName`, a RESPONSE/COLLECTION one with no `southCriterias`, an ADVANCED rule with no code,
a provision script missing `normalizeRawObject` or `actionsPlanning`. Errors block, warnings do not.

Deliberately **not** a JavaScript parser, per §6.2: a megabyte-scale dependency in a tool that is
otherwise standard library and Cobra, and it still would not catch a valid script reading the wrong
datastream — which the typings do catch, in the editor, with a real type-checker.

Run against sensehat's 14 live artifacts, the bracket check **reported an error in a working
connector function**: `message.replace(/\'/g, '')`. The escaped quote inside the regex was read as a
string delimiter and desynced everything after it. The doc comment claimed regexes were skipped;
they were not. A false positive here is worse than no check — validate blocks, and watch refuses to
push — so telling a regex from a division (which needs the preceding token) is now implemented.
Re-verified: **all 14 real artifacts pass clean**, seeded breakages are still caught with the right
line, and the regex case is a test.

### watch

`fsnotify` → filter → resolve to the nearest deployable unit → debounce → validate → classify →
deploy. All four of §5.6's details are in: editor debris is ignored (`4913`, `*.swp`, `~`, `*.tmp`,
generated `.d.ts`/`jsconfig.json`, and the `.og/` cache — which would otherwise retrigger on its own
writes), one save is one deploy, validation is on unless `--no-validate`, and the guards.

**A conflict refuses to deploy and there is no `--force`.** Overwriting someone else's edit should
not be one keystroke away.

`production: true` on a profile makes watch refuse to start without `--allow-production`. Nothing
else reads the field.

The session-invalidation warning is printed **only** for workspaces and dashboards, per the §5.6
finding: the transparent 401 re-sign lives in the Web API client, and the flat families never touch
it. A warning printed everywhere is a warning nobody reads.

After a successful deploy, watch **re-records the base snapshot**. Without that, the next save is
classified against a snapshot two versions old and reported as a conflict against your own work.

### Verified against production, and what it caught

Watching sensehat's 7 real rule directories:

| Case | Result |
|---|---|
| one edit to a real rule's JS | `would-deploy [local changes]` |
| `4913`, `*.tmp`, `~` written | nothing — filtered |
| conflict (base moved out from under the local tree) | `refused [conflict]`, **with dry-run off** |
| unbalanced brace | `invalid  error: javascript.js:33: "{" is never closed`, no deploy |
| profile marked `production: true` | refuses to start; starts with `--allow-production` |

Two bugs found by running it rather than by testing it:

- **`--json` emitted indented multi-line JSON**, not NDJSON — unparseable line by line, which is the
  entire point of the flag. It goes through `json.Marshal` directly now.
- A comment accidentally left in a `rule.json` during testing was correctly rejected as invalid JSON
  with the line number, which is a nice accidental demonstration of the validation path.

### The write itself, and the bug only writing could find

Exercised on Charlie's go-ahead with a throwaway rule — `claudia-watch-test`, `active: false`, so it
triggered nothing — created in sensehat, driven through `watch`, and deleted afterwards. sensehat is
back to its original six rules.

The first run deployed twice and reported the second as **`[unknown]`** instead of `local changes`.
The cause: `basestate.Find` returns an absolute root (it walks up), while a command hands it a
relative artifact directory. `filepath.Rel` of one against the other fails, and the fallback stored a
path `LookupByDir` could never match — so after the first deploy every classification came back
unknown, and with it the conflict detection that makes watch safe to leave running would have been
silently dead.

`--dry-run` could not have caught it: the re-record only happens after a real deploy. Nor could the
unit tests, which used consistent path styles throughout. Fixed by normalising both sides before
relativising, with a regression test that deliberately mixes absolute and relative.

Re-verified against production: three consecutive saves, three real deploys, each correctly
classified as `local changes`, and the platform ended up holding the last version written.

---

## 18. Phase 10 as built: connector function typings (2026-08-26)

`og connectors pull` now writes declarations too. The context follows the function's `type`, and the
protocol objects follow the **scheme of its south criteria** — verified against sensehat, whose
criteria are URIs like `mqtts://endesa` and `https://demo`. A REQUEST function has no south criteria
(it matches on `operationName` plus north criteria), so its protocol is unknowable from the payload;
every protocol object is declared rather than only the detected ones, because declaring too few
flags working code.

### Seven for seven wrong on the first run

The first version failed `tsc` on **all seven** of sensehat's live connector functions. Four distinct
causes, none of which a fixture would have shown:

1. **The catalogue was missing every plain function.** Extracting signatures from the reference's
   `object.method(...)` headings skipped the "Plain functions" section entirely — so `log()`,
   `httpRequest()` and `responseCF()`, all used in production, were reported as
   `Cannot find name`.
2. **`logger` is variadic.** The reference documents `logger.debug(msg)` but says it concatenates its
   parameters; production calls `logger.debug('payload is: ', payload)`.
3. **`mqtt.topic` is assigned**, not just `mqtt.publish()` called.
4. **A top-level `return` is correct.** The platform wraps the script in a function, so
   `return dataPoints;` at the end of a COLLECTION function is how it works — and TypeScript reports
   TS1108, which is not configurable away.

Then the rules regressed too, for three more reasons found the same way: `isInsertAction(entity)` is
called with an argument the reference documents as taking none; `entity.resourceType` and
`entity.device` are properties the entity carries besides its datastreams; and a live rule indexes
the entity **dynamically** — `entity[getVariableValue(parameterObject['x'])]`.

### `unknown` had to become `any`

A rule compares `entity['ccare.bps']._value._current.value > threshold`. With the value typed
`unknown`, TypeScript rejects the comparison — correct JavaScript, reddened. `unknown` is the safer
type in the abstract; here it made the declarations unusable. Untyped values are `any` now. The point
of these declarations is catching a mistyped **identifier**, which still works, not enforcing value
types the platform does not state.

### The jsconfig adapts to the code

Since a top-level `return`, an untyped helper parameter and a dynamic index are all correct code that
cannot be type-checked, the generated `jsconfig.json` turns `checkJs` **off** for those artifacts and
keeps completion, navigation and signature help — most of the day-to-day value — rather than
producing errors on working code. It says so in the file.

Of the 13 live artifacts, **8 are fully checked and 5 are completion-only**.

### Verified

All 13 real artifacts — 6 rules, 7 connector functions — type-check **clean**. Typos introduced
afterwards are still caught in the checked ones, with suggestions: `logger.dbug` → `debug`,
`mqtt.publsh` → `publish`, `responseCFF` → `responseCF`.

### Still open

Provision functions have **no** typings: there is no vendored JS reference for their execution
context, and the honest options were to invent globals or to ship nothing. Nothing shipped. Widget
formatters are also still open (`widget-js-api.md` exists and would be the source).

---

## 19. Typings generated from the documentation (2026-08-26)

Charlie pointed at the real documentation repository — `odm-documentation-hugo` — and asked whether
the JavaScript APIs could be vendored wholesale for editor support. They can, and the case for it is
the four signatures §18 got wrong by hand.

`tools/ogdocgen` reads the documentation and writes the declaration files. A development tool, not
part of the binary: run it when the documentation changes, commit the result. The generated files
stay reviewable — the diff of a regeneration shows what changed in the platform — and og needs no
access to the documentation repository.

### Coverage

43 pages carry a JavaScript API `type` in their front matter, and every page with JS signatures
outside `libs/ogapi-docs` is correctly typed, so nothing is missed through absent front matter.

| Context | Pages | Declarations |
|---|---|---|
| connector functions (core + 15 protocols) | 26 | 24 functions, 244 methods in 35 objects, **57 properties** |
| ADVANCED rules | 10 | 46 functions, 23 methods in 8 objects |
| **provision functions** | 6 | 42 functions, 46 methods |
| **timeseries functions** | 1 | 23 methods in 5 objects |

Roughly 450 declarations against the ~100 maintained by hand — and two contexts that did not exist
before, including the provision functions §18 declared unshippable for want of a reference. The 57
properties matter: they are the assignable fields (`mqtt.topic`, `http.client.body`) that a
signature-only reading of any reference misses entirely, and `mqtt.topic` had already bitten once.

### What the generator does and does not decide

The documentation is authoritative about what **exists**. It is not always right about details, so
`tools/ogdocgen/overrides.go` carries corrections — currently one — each with its evidence, since an
unexplained override is indistinguishable from a mistake. Separately, `paramTypes` refines a
documented `string` or `*` into a named type where that is the whole point: a datastream identifier
is one of the organization's, not any string, and an alarm severity is one of three words.

The hand-written half shrinks to `templates/base.d.ts`: the datastream value shape and the small
enumerations, which the documentation does not describe. Everything it does describe is generated.

### Four bugs in the generator, all found the same way

Type-checking the 13 live artifacts, not by inspection:

1. **`http` and `coap` vanished entirely.** Both are documented only as `http.client.get()` and
   `http.server.response.send()`, so the parent objects exist purely as containers; the emitter
   handled one level of nesting and dropped them. It builds the object tree recursively now.
2. **`entity` is two things at once** — a map keyed by datastream identifier, and an object with
   accessor methods (`entity._value(ds, i)`). Emitting it as a second `declare const` shadowed the
   per-organization map, silently losing every datastream identifier. The methods are emitted as an
   interface that the generated map intersects.
3. **Rest parameters lost their dots.** The documentation writes `logger.trace(...msg)` correctly;
   the parser stripped the `...` without recording it, making `log('a: ', b)` an arity error on
   working code.
4. **A properties table under a nested heading** (`server.response Object Properties`) keys on the
   leaf, not the full path.

All 13 real artifacts type-check clean afterwards, and typos introduced later are still caught.

### The documentation handoff

Four items for `odm-documentation-hugo`, in `docs/opengate-documentation-handoff.md`. One is a real
error: two pages state the provision processor entry point is `normalizeRowMap`, while the same
page's example — and the live function in sensehat — use `normalizeRawObject`.

That exercise also corrected **this document**: §1 claimed the original handoff was wrong about
`collectCF`. It was right; the vendored copy of the guide I checked against says `collectionCF` once,
the official documentation says `collectCF` six times with a worked example. The platform team then
settled it: `collectionCF` never existed, and `collectCF` is deprecated in favour of `cf.collection`
with the arguments reversed.

### Answered — see `docs/opengate-documentation-handoff-response.md`

All four items landed in the documentation. Two changed what ogdocgen does:

- The rules front matter is now `rules-js-api` / `rules-js-internal-api`, with no alias. The
  generator was keyed on `alarms-js-api` and silently produced three families instead of four; it
  now refuses to write when a known family yields no page.
- 22 connector-function globals (and 3 more in rules) name their replacement, so the generator emits
  `@deprecated` with the documentation's own wording — including the warning that `cf.collection`
  and `cf.response` take their arguments in the opposite order.

### Still open

The widget `$api` surface. **Do not parse `libs/ogapi-docs`**: those pages were generated from an
unmerged branch and document 15 classes release 14.15.0 does not have while missing 18 it does.
`opengate-js` now publishes its own `.d.ts` (PR #140, `types` in `package.json`, 234 declaration
files emitted from the JSDoc by tsc), which an editor consumes natively — so there is no generator
to write for this surface, only a decision about how og points an artifact directory at the
declarations for the pinned library version.