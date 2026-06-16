# og — v1.0 readiness audit

_Audit date: 2026-06-15 · current release: v0.10.0 · branch: main_

Goal: confirm the **CLI ↔ TUI ↔ MCP ↔ SKILLS** invariant is truly level across every
OpenGate surface, and enumerate what must be resolved before tagging **v1.0.0**.

Charlie's stated v1.0 blockers (confirmed): **(1) MCP multi-tenant auth**, **(2) HTTP
south IoT custom-route collect**. CA packaging for MQTT is acknowledged but not urgent.

---

## 1. Parity matrix (resource → surface)

Legend: ✅ full · ✸ partial / read-only · ➖ absent (by design) · ❌ gap (should exist)

| Resource | CLI | MCP | TUI | Skill |
|---|---|---|---|---|
| auth / login | ✅ login | ✅ login | ✅ viewLogin | og-cli |
| datamodels | ✅ CRUD+search | ✅ 5 | ✸ browse only | og-cli |
| devices | ✅ CRUD+search | ✅ 5 | ✸ browse+detail | og-cli |
| alarms | ✅ search/summary/attend/close | ✅ 4 | ✅ list+attend/close | og-device-ops |
| time series | ✅ CRUD+data+export | ✅ 7 | ✸ list+data | og-cli |
| datasets | ✅ CRUD+data | ✅ 6 | ✸ list+data | og-cli |
| jobs (operations) | ✅ search/get/create/cancel/operations | ✅ 5 | ✅ list+detail+launch | og-device-ops |
| scheduled tasks | ✅ search/get/create/cancel | ✅ 4 | ❌ **view orphaned** | og-device-ops |
| operation types | ✅ CRUD+catalog | ✅ 6 | ✸ list only | og-device-ops |
| rules | ✅ CRUD+catalog+toggle+pull/deploy+logs | ✸ 6 (no catalog/logs) | ✸ list+toggle | og-device-ops |
| connector functions | ✅ CRUD+catalog+status+pull/deploy+logs | ✸ 6 (no catalog/logs) | ✸ list+status | og-device-ops |
| provision functions | ✅ CRUD+pull/deploy+plan/bulk | ✅ 9 | ✸ list only | og-device-ops |
| workspaces | ✅ CRUD+import/export+pull/deploy+share | ✅ 7 | ✸ list+detail+share | og-workspaces |
| dashboards | ✅ CRUD+import/export+pull/deploy+share | ✅ 7 | ✸ detail (under ws) | og-workspaces |
| field views | ✅ list/show | ✸ resource only (no tool) | ✸ picker in devices | og-cli |
| IoT south — HTTP | ✅ collect/collect-file | ✅ 2 | ➖ (injection, no TUI) | og-device-ops |
| IoT south — MQTT | ✅ publish/subscribe/device | ✅ 3 | ➖ (injection, no TUI) | og-device-ops |
| IoT south — **HTTP CF route** | ❌ | ❌ | ➖ | ❌ |

---

## 2. The honest parity picture

**CLI is the complete surface. MCP is ~95% of it. TUI is a read-mostly browser.**

The CLAUDE.md invariant ("every operation exposed through CLI, TUI, and MCP") is
**aspirational, not literally enforced**. In reality:

- **TUI mutations are limited to 5 actions**: alarm attend/close, launch job, rule
  toggle-active, connector status-cycle, workspace share. There is **no
  create/update/delete, and no pull/wrap/deploy, anywhere in the TUI.** Everything
  else is list/detail/search.
- **Local-file lifecycle verbs** (`pull`/`pull-all`/`wrap`/`deploy`) are **CLI-only**
  for rules/connectors/provision/workspaces/dashboards — inherent (they touch arbitrary
  local directories). MCP offers the equivalent via inline-body create/update and
  workspace/dashboard import/export.
- **IoT injection has no TUI** by design (it's an action, not a browseable resource).

→ **Decision required for v1.0:** either (a) **re-scope the invariant** to
"CLI = full; MCP = full minus local-fs lifecycle; TUI = browse + high-value actions",
and state it in CLAUDE.md + README; or (b) invest in TUI editing forms (large). The
audit **recommends (a)** — the current split is sane; it's just undocumented.

---

## 3. Findings & remediation (prioritised for v1.0)

> **Progress (branch `og-cli/v1-parity`):** B1 ✅, B2 ✅, A2 ✅, A3 ✅, A3b ✅. P2/P3 closed
> for v1.0: surface split documented (CLAUDE.md §"Surface model" + README); test backfill ✅
> (provider auth, MQTT/provision parsing, payload builders); MQTT CA **deferred to v1.1**
> with `--insecure` default kept and a handoff brief (`docs/mqtt-tls-handoff.md`);
> assets/subscribers direct CRUD **out of v1.0** (provision functions cover creation).
> **→ v1.0 is feature-complete on this branch; ready to merge + tag.**

### P0 — v1.0 blockers (Charlie-confirmed)

- **[B1] MCP HTTP multi-tenant auth.** `og mcp --http` serves a **single static
  identity** (profile host/token/webToken/apiKey, baked at startup in `newServer`).
  No per-request auth / header passthrough. Blocks multi-tenant MCP hosting.
  Design already drafted (memory `project-mcp-multitenant-auth`). _Files:_
  `internal/mcp/server.go`, `cmd/mcp.go`.
  **✅ DONE** — `og mcp --http --multi-tenant`: stateless per-request credentials from
  headers (Authorization Bearer / X-OG-Web-Token / X-OG-Api-Key) via a `provider` seam
  (`internal/mcp/session.go`) + `WithHTTPContextFunc`. Per-tool requirement (Bearer always;
  web/apikey only where used); no startup fallback; login tool dropped in MT. Verified live:
  missing Bearer → error, valid Bearer → real devices_search. OIDC-agnostic (token provenance
  is a backend concern).
- **[B2] HTTP south custom-route collect.** No command posts raw data to
  `/south/v80/devices/{id}/{uri-path}` (a connector function's HTTP southCriteria
  route). MQTT covers its own custom topics via `og iot publish --topic --raw`, but
  the HTTP CF-trigger leg is missing — so a COLLECTION/RESPONSE CF on an HTTP route
  still needs `curl`. Add `og iot collect-raw <device> --route <path> --body/-f` with
  CLI + MCP parity. _Files:_ `pkg/opengate/iot.go`, `cmd/iot.go`, `internal/mcp/tools_iot.go`.
  **✅ DONE** (`og iot collect-raw` + `iot_collect_raw`/`CollectRaw`) — verified live: posted to
  charlie-01/ogcli-demo, CF fired (21→69.8°F, name `cf-…`), confirmed in north.

### P1 — parity fixes (quick, real gaps)

- **[A2] `viewTasks` is orphaned.** The TUI screen exists but has **no menu entry and
  no navigation path** — unreachable dead code, while `og tasks` and `tasks_*` exist.
  Fix: add a "Scheduled Tasks" menu item + `fetchTasks` wiring (mirrors jobs), or
  remove the view. _Files:_ `internal/tui/menu.go`, `internal/tui/tui.go`, `internal/tui/operations.go`.
  **✅ DONE** — added "Scheduled Tasks" menu item + navigation to the existing view.
- **[A3] MCP missing `rules_catalog` and `connectors_catalog`** (both exist in CLI).
  Trivial adds — `RulesCatalog()` / `ConnectorFunctionsCatalog()` clients already exist.
  _Files:_ `internal/mcp/tools_rules.go`, `internal/mcp/tools_connectors.go`.
  **✅ DONE** — `rules_catalog` + `connectors_catalog`.
- **[A3b] No MCP tool for rules/connectors logs.** CLI streams live; MCP has nothing.
  Add bounded `rules_logs` / `connectors_logs` (N lines / timeout) backed by the
  existing `CollectFunctionLogs`. _Files:_ `internal/mcp/tools_rules.go`, `tools_connectors.go`.
  **✅ DONE** — bounded `rules_logs` + `connectors_logs` (shared `internal/mcp/tools_logs.go`;
  apiKey threaded into the register funcs).

### P2 — documentation & hygiene

- **[A1/A4] Document the intentional surface split** (section 2) in CLAUDE.md + README
  so the invariant is honest and reviewers stop expecting TUI CRUD.
- **[C2] MQTT `--insecure` defaults to true** (skips TLS verify). Smell for v1.0.
  **RESOLVED for v1.0: deferred to v1.1.** Root cause found — the broker cert is a
  valid Let's Encrypt cert; the broker just **omits the intermediate** in the handshake
  (server misconfig), so it's not a private-CA problem. v1.0 keeps `--insecure` default;
  full pickup brief in `docs/mqtt-tls-handoff.md`. _Files:_ `pkg/opengate/mqtt.go`.
- **[A5] Skills consistency pass.** og-device-ops, og-cli, og-workspaces are correctly
  partitioned and now include provision + MQTT. Do a final read to ensure every MCP
  tool name and CLI verb shipped this cycle appears in the matching skill.

### P3 — coverage (post-v1 candidates / scope decisions)

- **[C1] Assets / subscribers / subscriptions** have **no direct CRUD** on any surface
  — only creatable via provision-function JS. Devices have full CRUD.
  **RESOLVED: out of v1.0** — documented in CLAUDE.md §"Surface model" (Entity scope).
  Provision functions cover creation; revisit post-v1 if demand appears.
- **[C3] Test coverage** was thin (only `unwrap`, `dashboards`, `devices`, `query`,
  `views`). **Backfilled for v1.0:** `internal/mcp/session_test.go` (the multi-tenant
  provider — single/multi-tenant resolution, no startup fallback, per-tool apiKey),
  `pkg/opengate/mqtt_test.go` (host/topic helpers, bulk-id, content-type, operation
  request/response), `pkg/opengate/provisions_test.go` (summary parse, dual-key list).
  Further client-layer HTTP tests remain a post-v1 nice-to-have.
- **[C4] Organizations / users / roles** management is not covered (likely out of scope
  for a device-ops CLI — confirm).

---

## 4. Suggested v1.0 sequence

1. **B1** MCP multi-tenant auth (header passthrough, stateless) — _priority_.
2. **B2** HTTP south custom-route collect (`og iot collect-raw`) — _priority_.
3. **A2 + A3 + A3b** parity quick-fixes (tasks view, MCP catalog + logs tools).
4. **A1/A4** document the surface split (CLAUDE.md + README).
5. **C2** MQTT CA packaging → secure TLS default.
6. **C3** test backfill for the client layer.
7. **C1/C4** scope decisions on assets/subscribers and org/user management.

Nothing here is structural debt — the architecture is sound and the breadth is real.
These are the deltas between "feature-complete" and "v1.0-confident".
