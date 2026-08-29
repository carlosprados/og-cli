# Handoff — og-cli Developer Experience Roadmap

**Target repo:** `github.com/carlosprados/og-cli` (local checkout)
**Audience:** Claude Code, working in the repo with the human in the loop
**Status:** design + implementation brief. Phase 0 is mandatory before any code is written.

---

## 1. Mission

Turn `og` into the primary development surface for **all embedded code in OpenGate**, replacing
the in-platform CodeMirror editor with a local, versionable, IDE-native workflow.

The repo already implements the core cycle (`pull` / `unwrap` → edit → `wrap` → `deploy`) for
workspaces, dashboards and ADVANCED rules. This work extends it in three directions:

1. **Coverage** — first-class support for **connector functions** and **provision functions**,
   at the same level of quality as rules and workspaces/dashboards/widgets.
2. **Safety and visibility** — `diff` (with three-way conflict detection) and `watch`
   (continuous deploy on save).
3. **Editor intelligence** — generated TypeScript declaration files so any LSP-capable editor
   (Neovim, VS Code, Cursor, Zed) gets real completion and diagnostics for platform globals.

A VS Code extension is the final deliverable, built as a thin shell over the binary.

### Non-goals

- Do **not** improve or replace the in-platform CodeMirror editor. That is out of scope.
- Do **not** reimplement any OpenGate API call outside the Go client library.
- Do **not** build a language server from scratch. Generate `.d.ts` and let `tsserver` work.

---

## 2. Ground rules

Read `CLAUDE.md` and `README.md` first — they take precedence over anything here that conflicts.

- **Never invent an API endpoint, field name, or payload shape.** Verify against
  `opengate-wapi-openapi.yaml`, `https://documentation.opengate.es`, or a live call against a
  staging profile. If it cannot be verified, stop and ask. A fabricated endpoint that silently
  fails is worse than an unimplemented command.
- **All code, comments, commit messages and documentation in English.**
- Conventional Commits with named scopes. No `Co-authored-by` trailers.
- Staged-approval workflow: propose the change, get approval, then implement. Do not run
  multi-phase work end-to-end without checkpoints.
- Build automation via `Taskfile.yml` (`task build`, `task test`, `task lint`, `task fmt`).
- Repo convention: **any PR that adds or changes commands must update both `README.md` and the
  relevant skill under `.claude/skills/`.** This is not optional.
- Prefer standard library and existing dependencies. This is a lean single-binary tool.

---

## 3. Phase 0 — Discovery (mandatory, no implementation)

Produce `doc/DX_ANALYSIS.md` and stop for review. Nothing in phases 1+ starts before this is
approved.

### 3.1 Audit the existing implementation

- Map how `pull` / `unwrap`, `wrap`, `deploy`, `export`, `import` are implemented today for
  workspaces, dashboards and rules. Identify duplicated logic across the three.
- Locate the JS extraction heuristic (field names `formatter`, `script`, `operation`, `code`,
  `fn`, `expression`, `_widgetConfigCode`, plus the keyword-based fallback) and document its
  exact behaviour and failure modes.
- Locate the SHA256 config comparison used to assert `wrap` is content-lossless. This is the
  seed of the canonicalization work in Phase 2.
- List every place a nested resource ID, slug or array ordering is encoded in a path
  (`NN__<slug>` prefixes) and how collisions are handled.

### 3.2 Map the API surface for connector functions and provision functions

**This is the highest-risk unknown in the whole plan. Treat it as research, not coding.**

For each of the two families, determine and document:

| Question | Where to look |
|---|---|
| CRUD endpoints (list, get, create, update, delete) | `opengate-wapi-openapi.yaml`, official docs |
| Which API surface — North IoT API or Web API (`/api/v1`)? | affects auth path and client reuse |
| Scoping — organization, channel, both? | mirrors the `--channel` handling in `rules` |
| Where the JavaScript body lives in the JSON payload | field name and nesting |
| Enable/disable semantics, if any | mirrors `og rules enable/disable` |
| Versioning or revision fields | needed for canonicalization and conflict detection |
| Ordering constraints between resources | connector functions can be **concatenated** |

Known from the official documentation (verify all of it against the live API before relying on it):

**Connector functions** execute JavaScript in the device-integration path. They come in
`REQUEST` and `RESPONSE` variants and are selected by **south criteria** (an HTTP/WebSocket
path, or a protocol-specific URI such as `dlms://obis/<code>` or `dlms://template/<id>`).
The script has access to the globals `entity`, `gateway`, `payload` and `contextParams`, plus
helper functions including `entityValue()`, `collectCF()`, a `collection` object with
`addDatapoint()`, and a `response` object with `errorProcessing()`.

Crucially, **the available globals depend on the connector protocol**. Each protocol injects
its own object: `snmp` (`get`, `set`), `mqtt`, `http`, `websocket`, `coap`, `dlms`, `iec102`,
`icmp`, `kite`, `ssh`, `telnet`. There are also cross-cutting APIs documented as Core JS API,
Entity JS API, Inner Collections API, Operation Steps API, Operation JS API, Concatenated
Connector Functions API, UTILS JS API and Cryptography API.

**Provision functions** are documented under bulk provisioning, with their own JavaScript API.
Determine their execution context, input/output contract, and whether they are scoped per
organization or per bulk-provisioning job.

Reference index: `https://documentation.opengate.es/` → *Connector Functions* and
*Bulk provisioning → Provision functions*.

### 3.3 Deliverable

`doc/DX_ANALYSIS.md` containing:

- The endpoint tables above, with a confidence level per row (`verified-live`,
  `from-openapi`, `from-docs`, `unknown`).
- A proposed unified resource abstraction (Phase 1) with the interface signature.
- A proposed on-disk layout for connector functions and provision functions.
- An explicit list of open questions for the human.

---

## 4. Phase 1 — Unified resource abstraction

Today workspaces, dashboards and rules each carry their own `pull` / `wrap` / `deploy`
implementation. Adding two more families by copy-paste produces five divergent code paths and
five sets of bugs. Refactor first.

Introduce `internal/artifact` with a single interface that every family implements:

```go
// Kind identifies a family of platform artifacts that supports the
// pull → edit → deploy lifecycle.
type Kind string

const (
    KindWorkspace         Kind = "workspace"
    KindDashboard         Kind = "dashboard"
    KindWidget            Kind = "widget"
    KindRule              Kind = "rule"
    KindConnectorFunction Kind = "connector-function"
    KindProvisionFunction Kind = "provision-function"
)

// Resource is the contract every artifact family implements. The generic
// pull/diff/watch/deploy machinery is written once against this interface.
type Resource interface {
    Kind() Kind

    // Fetch retrieves the remote representation of a single artifact.
    Fetch(ctx context.Context, c *client.Client, id string) (Remote, error)

    // List enumerates artifacts of this kind within the current scope.
    List(ctx context.Context, c *client.Client, scope Scope) ([]Remote, error)

    // Explode writes the remote representation to a directory tree,
    // extracting embedded code into standalone files.
    Explode(r Remote, dir string) (Tree, error)

    // Implode reconstructs the API payload from a directory tree.
    // Must be the exact inverse of Explode, modulo canonicalization.
    Implode(dir string) (Payload, error)

    // Push writes the payload back. create=false performs an update.
    Push(ctx context.Context, c *client.Client, p Payload, create bool) error

    // CodeFiles returns the extracted code files in the tree, with the
    // language and execution context needed for validation and typegen.
    CodeFiles(dir string) ([]CodeFile, error)
}

// CodeFile is an extracted, editable source file plus the context needed
// to type-check it.
type CodeFile struct {
    Path     string   // relative to the artifact root
    Language string   // "javascript"
    Context  ExecCtx  // rule | connector-function/<protocol> | provision | widget-formatter
}
```

**Acceptance criteria:** existing workspace, dashboard and rule commands are reimplemented on
top of `Resource` with **no change to their CLI surface or output**, and the existing test
suite passes unchanged. This refactor must be behaviour-preserving.

---

## 5. Phase 2 — Connector functions (`og cf`)

First-class parity with `og rules`.

### Commands

```
og cf search   [-w "cf.name like weather"] [--channel <ch>] [--org <org>]
og cf get      <id> [--org <org>]
og cf create   -f cf.json | og cf update <id> -f cf.json | og cf delete <id>
og cf enable   <id> | og cf disable <id>          # only if the API supports it

og cf pull     <id> --dir cfs/ [--force]
og cf pull-all --dir cfs/ [--channel <ch>]
og cf pull-file cf.json --dir cfs/
og cf wrap     cfs/<slug> [--out cf.json]
og cf deploy   cfs/<slug> [--update]

og cf diff     cfs/<slug> | cfs/
og cf watch    cfs/
og cf test     cfs/<slug> --payload sample.json [--entity entity.json]
og cf graph    [--channel <ch>] [-o dot]          # concatenation dependency graph
```

### On-disk layout

Mirror the existing rules layout so the mental model transfers:

```
cfs/
  <channel-slug>/
    <cf-slug>/
      connector-function.json    # metadata: type, south criteria, protocol, scope, active
      code.js                    # the extracted JavaScript body
      og-globals.d.ts            # generated, protocol-specific (Phase 6)
      jsconfig.json              # generated, wires the .d.ts to tsserver
```

### Requirements specific to connector functions

- **`REQUEST` vs `RESPONSE`** must be visible in the metadata file and in `search` output.
  They have different execution contexts and therefore different generated typings.
- **South criteria are the routing key.** Surface them prominently in list output and validate
  their format on deploy where the format is known (`dlms://obis/<dotted-code>` uses dots only,
  never the complex `0-0:66.` form). A malformed south criterion produces a connector function
  that silently never fires — exactly the class of bug this tooling exists to prevent.
- **Concatenated connector functions** form a dependency graph. `og cf graph` must render it,
  and `deploy` must warn when pushing a function that others depend on. Deleting a node with
  dependents requires `--force`.
- **Protocol detection** drives typegen. Record the protocol in the metadata file at pull time.

### `og cf test` — local execution

The single highest-value command in this phase. Connector functions are currently untestable
without deploying to a live platform and sending real device traffic.

- Execute `code.js` in `goja` against a sample payload and a sample entity JSON.
- Provide **stub implementations** of the protocol objects (`snmp`, `mqtt`, `dlms`, ...) that
  record calls instead of performing I/O, and print the recorded call log.
- Capture `collection.addDatapoint()` calls and print the resulting datapoints — this is what
  the developer actually wants to verify.
- Capture `response.errorProcessing()` and non-zero exits.
- **Be explicit in the output that this is a simulation**, not platform-fidelity execution.
  Overclaiming here will cost trust the first time behaviour diverges. If full stubbing proves
  unreliable, degrade to syntax-and-static-check only rather than shipping a misleading harness.

---

## 6. Phase 3 — Provision functions (`og pf`)

Same command set, same layout shape, adjusted to the provisioning context determined in
Phase 0.

```
og pf search | get | create | update | delete
og pf pull | pull-all | wrap | deploy | diff | watch
og pf test <slug> --input sample.csv       # or the real input format, per Phase 0
```

Layout:

```
pfs/
  <pf-slug>/
    provision-function.json
    code.js
    og-globals.d.ts
    jsconfig.json
```

Provision functions run during bulk provisioning, which means **a bad one corrupts entity data
at scale**. `og pf test` and pre-deploy validation are not conveniences here; treat them as
required before `og pf deploy` is considered complete. Consider making `--validate`
non-overridable for this family, or at minimum requiring an interactive confirmation.

---

## 7. Phase 4 — Canonicalization (`internal/canon`)

Prerequisite for `diff`, `watch` and conflict detection.

`wrap` is documented as lossless *modulo cosmetic differences in null/default serialisation*.
That footnote is the entire difficulty of `diff`: a naive textual comparison produces hundreds
of lines of noise from key ordering, `null` vs absent fields, and server-injected metadata.

```go
// Canonicalize produces a stable, comparable representation of an artifact:
// object keys sorted, volatile server-managed fields stripped, null and
// default serialisation collapsed to a single form.
func Canonicalize(v any, kind artifact.Kind) ([]byte, error)

// Hash returns the canonical digest used for change detection.
func Hash(v any, kind artifact.Kind) (string, error)

// VolatileFields are server-managed and never participate in comparison.
var VolatileFields = map[artifact.Kind][]string{ /* per-kind, from Phase 0 */ }
```

**Acceptance criteria:** for every artifact family, `Hash(Explode → Implode) == Hash(original)`
across the whole `demo/` corpus. Add a property test that round-trips every fixture.

---

## 8. Phase 5 — Base snapshots and three-way state

Two-way comparison (local vs remote) can tell you that things differ. It cannot tell you *who
changed what*, which makes any automatic deploy a silent overwrite of someone else's work.

At `pull` time, persist the canonical remote state:

```
<artifact-root>/
  .og/
    base/
      <artifact-id>.canon.json
    manifest.json     # {artifactID: {hash, pulledAt, profile, org, channel}}
```

`.og/` goes in `.gitignore` — it is a sync cache, not a source of truth. Add it to the repo's
own `.gitignore` template and document it.

Classification, which drives every downstream decision:

| local vs base | remote vs base | State | `diff` marker | `watch` action |
|---|---|---|---|---|
| = | = | clean | (none) | none |
| ≠ | = | local changes | `~` | deploy |
| = | ≠ | remote changes | `↓` | warn, suggest `pull` |
| ≠ | ≠ | **conflict** | `!` | **refuse to deploy** |

---

## 9. Phase 6 — `diff`

Available on every family: `og <resource> diff`.

### Two kinds of diff, presented separately

| Content | Diff type |
|---|---|
| `*.json` metadata | **structural** over the canonical form: `+ field`, `~ field: a → b`, `- field` |
| `*.js` extracted code | **textual** unified diff |

Mixing them in one text stream is what makes generated diffs unreadable. Keep them visually
distinct.

### Output

```
$ og workspace diff wsroot/dashboards-adif
dashboards-adif  (workspace 6712a...)
  ~ 00__visualizaci-n-pbi
      ~ 00__customchart__1727269767709-0
          M _widgetConfigCode.js          +12 −4
          ~ widget.json  config.yAxis.max: 100 → 140
  + 03__comparativa-nueva                 (local only, not deployed)
  ! 01__visualizaci-n-pbi-m-ximos         (remote changed since pull)

3 dashboards changed, 1 added, 1 conflict
```

### Flags

- `--name-only`
- `-o json` — **stable contract**, consumed by CI and by the VS Code extension. Version the
  schema from day one (`"schemaVersion": 1`) and document it in `doc/json-output.md`.
- `--exit-code` → `0` no differences, `1` differences, `2` error. Enables
  `og cf diff --exit-code` as a CI drift gate.
- `--against <profile>` — **cross-tenant diff**. The repo already documents a source→destination
  migration pattern; "what differs between staging and production?" is currently unanswerable
  without manual JSON archaeology. Expect this to be the most-used flag.

---

## 10. Phase 7 — `watch`

```
og cf watch cfs/ --org sensehat
og workspace watch wsroot/dashboards-adif
```

Pipeline: `fsnotify` → debounce → resolve target → validate → enqueue → `deploy --update`.

### Granularity

Resolve the changed file to the **smallest deployable unit**, not the root. Editing
`_widgetConfigCode.js` deploys the owning dashboard, not the whole workspace.

```go
// resolveTarget walks up from the changed path to the nearest deployable unit.
func resolveTarget(root, changed string) (Target, error)
```

### Four implementation details that decide whether this is usable

**a) Editors replace files, they do not write them.** Neovim writes to a temporary file and
renames, creates a probe file named `4913`, and leaves `.swp` / `.swx` / `~` artifacts. Listen
for `CREATE|WRITE|RENAME`, and filter:

```go
var ignorePatterns = []string{"4913", "*.swp", "*.swx", "*~", ".og/**", "*.tmp", "*.d.ts"}
```

**b) Serialized queue with coalescing.** One worker per target, ~300ms debounce, discard stale
events for the same target. Never issue concurrent updates against the same artifact — the
platform will not merge them, it will keep whichever arrives last.

**c) Validation on by default.** Parse the JavaScript (esbuild-as-library or `goja` parse-only)
and validate the JSON schema before every push. Deploying a syntax error into a live connector
function or provision function is an incident, not a typo. `--no-validate` exists but must be
explicit.

**d) Safety guards.** `watch` is the only part of `og` that writes to the platform without a
per-action human decision.

- `--dry-run` prints what would be deployed.
- Conflict state (`!` above) blocks deploy unconditionally. No `--force` on watch.
- Add `production: true` to the profile schema in `config.yaml`. `watch` **refuses to start**
  against a production profile without `--allow-production`.
- `--json` emits NDJSON events for the extension and any external consumer.

### Known landmine to document

OpenGate allows **one active web session per user**, and `og` transparently re-signs on 401.
With `watch` running continuously, this produces a loop: developer saves → `watch` re-signs →
their browser session is invalidated → they reload to see the result → the CLI token dies →
next save re-signs. The natural dashboard workflow is "edit, then look at the browser", so this
will be hit immediately.

Mitigation: recommend a dedicated service account per developer for the CLI, and print a
warning at `watch` startup when the profile's credentials match a likely-interactive user.
Document it in `README.md` before someone discovers it the hard way.

---

## 11. Phase 8 — Type generation (do this early, it is cheap and high-leverage)

Generate `og-globals.d.ts` plus a `jsconfig.json` into every pulled artifact directory.
`tsserver` picks these up automatically — this delivers completion and diagnostics in Neovim,
VS Code, Cursor and Zed with **zero editor-specific code**.

The typings are **per execution context**, which is the whole point:

| Context | Globals to declare |
|---|---|
| ADVANCED rule | `entity[ds]._value._current/._previous.value`, `parameterObject`, `getVariableValue()`, `ruleName`, `openAlarm()` |
| Connector function (common) | `entity`, `gateway`, `payload`, `contextParams`, `entityValue()`, `collectCF()`, `collection.addDatapoint()`, `response.errorProcessing()` |
| Connector function (per protocol) | `snmp` / `mqtt` / `http` / `websocket` / `coap` / `dlms` / `iec102` / `icmp` / `kite` / `ssh` / `telnet` |
| Provision function | per Phase 0 findings |
| Widget formatter | widget-specific formatter signature |

Example shape:

```typescript
// og-globals.d.ts — generated by `og cf pull`, do not edit
// context: connector-function/REQUEST/snmp

interface DatastreamValue { _current: { value: unknown; at: string }; _previous: { value: unknown } }
declare const entity: Record<string, { _value: DatastreamValue }>;
declare const gateway: Record<string, { _value: DatastreamValue }> | null;
declare const payload: unknown;
declare const contextParams: { path?: string; [k: string]: unknown };

declare function entityValue(e: typeof entity, path: string, index?: number): unknown;
declare function collectCF(data: unknown, criteria: string): void;

declare const snmp: {
  get(oid: string): unknown;
  set(oid: string, value: unknown): unknown;
};
```

### The multiplier

`pull` already knows the organization's datamodel. Generate the **real datastream identifiers**
instead of a generic `Record<string, ...>`:

```typescript
type DatastreamID = 'wt' | 'wp' | 'device.temperature.value' | /* ... from the datamodel */;
declare const entity: Record<DatastreamID, { _value: DatastreamValue }>;
```

Now `entity['wtt']` is flagged red in the editor before it is ever deployed. No generic editor
or LLM can produce this — it can only be generated from the platform. Add
`og typegen --org <org> --context <ctx> --out og-globals.d.ts` as a standalone command so it can
be regenerated when the datamodel changes.

Source the protocol signatures from the official documentation and record, in a comment header,
which doc version they were generated from.

---

## 12. Phase 9 — VS Code extension (separate repository)

**Single principle: the extension is a thin shell over the binary. Zero API logic in
TypeScript.** The moment a call is reimplemented in TS there are two sources of truth.

```
VS Code extension (TypeScript)          og binary (single source of truth)
  TreeView, commands, diff UI     ──▶     auth, pull/wrap/deploy
  diagnostics, status bar         child     diff -o json, watch --json
                                  process   typegen
```

- **Binary management:** find `og` on `PATH`; otherwise download the matching GoReleaser asset,
  verify the checksum, cache in `globalStorage`. Setting `og.path` for override.
- **TreeView:** Profiles → Workspaces → Dashboards → Widgets; plus Rules, Connector Functions
  and Provision Functions as sibling roots. Populated from `-o json` output, lazy children.
- **Native diff:** register a `TextDocumentContentProvider` for an `og-remote:` scheme and call
  `vscode.diff(remoteUri, localUri, title)`. Do not render diffs manually.
  This needs one small CLI addition:
  `og <resource> show <id> --path <relative-path>` printing remote content to stdout.
- **Do not spawn `og watch` from the extension.** Use `onDidSaveTextDocument` and call
  `og ... deploy --update`. Two watchers over the same files produce duplicate events.
  Division: `og watch` serves terminal and Neovim users; save hooks serve the extension. Both
  share the same Go deploy path.
- **Diagnostics:** `og <resource> validate -o json` → `DiagnosticCollection`.
- **Publish to Open VSX as well as the VS Code Marketplace**, or Cursor, Windsurf and VSCodium
  users are excluded.

---

## 13. Cross-cutting requirements

- **JSON output contract.** Every new command supports `-o json` with a versioned schema.
  Document it in `doc/json-output.md`. Breaking it breaks the extension and CI.
- **Exit codes.** `0` success / no differences, `1` differences found, `2` error. Consistent
  across `diff` and `validate`.
- **Tests.** Round-trip property tests per family over `demo/` fixtures; table-driven tests for
  `resolveTarget`; a `watch` test using a fake clock and synthetic Neovim-style write sequences
  (temp file + rename + `4913` probe).
- **Documentation.** `README.md` command reference, plus new/updated skills under
  `.claude/skills/`. Connector functions and provision functions likely warrant a fourth skill,
  `og-code` — decide during Phase 0.
- **`demo/`** gains an end-to-end connector-function and provision-function scenario, consistent
  with the existing runbook style.

---

## 14. Suggested sequencing

Ordered by impact-to-effort, with checkpoints for human approval between phases:

1. **Phase 0** — discovery, `doc/DX_ANALYSIS.md`. **Stop for review.**
2. **Phase 8 typegen for rules only** — ships value in days, works immediately in Neovim,
   validates the approach before broadening.
3. **Phase 1** — `internal/artifact` refactor, behaviour-preserving. **Stop for review.**
4. **Phase 4** — `internal/canon` + round-trip property tests.
5. **Phase 2** — connector functions, including `og cf test`.
6. **Phase 3** — provision functions.
7. **Phase 5** — base snapshots and three-way state.
8. **Phase 6** — `diff`, including `--against <profile>`.
9. **Phase 7** — `watch`.
10. **Phase 8 (full)** — typegen for all contexts, with datamodel-derived datastream IDs.
11. **Phase 9** — VS Code extension, only once 1–10 are stable.

The extension cannot be better than the CLI it wraps. Do not start it early.

---

## 15. Open questions for the human

Raise these during Phase 0 rather than guessing:

1. Do connector functions and provision functions live on the North IoT API or the Web API?
   This determines auth path and client reuse.
2. Is there a revision/version field that enables optimistic concurrency on update? If so,
   three-way conflict detection can be enforced server-side rather than only locally.
3. Are connector functions scoped per channel like rules, or per organization?
4. What is the real input contract for provision functions in bulk provisioning?
5. Should `og cf test` ship stubbed protocol execution, or start with syntax-and-static-check
   only? Stubs risk diverging from platform behaviour.
6. Governance: this repo is explicitly unofficial and disclaims all liability, while the
   roadmap points at a marketplace extension and a `watch` command that writes to production.
   Decide before Phase 9 whether this is internalized as an Amplía product with support, or
   deliberately remains a community tool. Both are defensible; drifting into the question after
   a customer incident is not.
