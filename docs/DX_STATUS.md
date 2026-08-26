# DX roadmap — state and next steps

Short, current, and meant to be read first. The reasoning behind every decision is in
`DX_ANALYSIS.md`; this file says where things stand.

**Last updated:** 2026-08-26
**Branch:** `main`, pushed (`ff780f8`)
**Released:** `v2.1.0`. Two commits sit on top of it, unreleased: connector-function typings and the
documentation-driven generator.
**Health:** build clean, 13 packages green, lint at the 42-issue baseline.

---

## Done

| # | Work | Where |
|---|---|---|
| B1 | `pull-all` slug collisions | v2.0.2 |
| B2 | `wrap` fails on a malformed widget directory | v2.0.2 |
| B3 | Filename encoding — no renames needed | v2.1.0 |
| 1 | Declared code-path contract per family | v2.1.0 |
| 2 | Typings for ADVANCED rules | v2.1.0 |
| 3 | Descriptor table for flat families | v2.1.0 |
| 4 | `internal/canon` + corpus round-trip property test | v2.1.0 |
| 5 | Versioned JSON envelope + typed exit codes | v2.1.0 |
| 6 | `.og/` sync store, provenance, three-way classification | v2.1.0 |
| 7 | `og <family> validate` | v2.1.0 |
| 8 | `og <family> diff`, incl. `--against` and `--exit-code` | v2.1.0 |
| 9 | `og <family> watch` | v2.1.0 |
| 10a | Typings for connector functions | unreleased |
| 10b | `tools/ogdocgen`: all typings generated from the official documentation | unreleased |

## Not done

| Item | Why it is not done | Size |
|---|---|---|
| **Widget `$api` typings** | `libs/ogapi-docs/JS Reference/` (92 pages) has no `type` in its front matter, so it is not machine-readable. Asked for in the documentation handoff. | small once the front matter lands |
| **`diff` for workspaces/dashboards** | The engine is family-agnostic and their payloads canonicalize fine; what is missing is the hierarchical renderer — a tree of dashboards and widgets each with its own marker, rather than a flat list of `dashboards[0].dashboard.grid[2]…` paths. A rendering job. | medium |
| **`watch` for workspaces/dashboards** | Depends on the above, and on the session-invalidation warning that only applies to these two families. | small after diff |
| **Phase 11: VS Code extension** | Gated on the governance decision (community tool vs Amplía product), which Charlie has yet to discuss internally. | large |
| Volatile fields for rules and provision functions | None observed in a real payload. Deliberately empty, with a test pinning that intent. Fill from a live GET if one ever shows one. | trivial |

---

## Decisions taken

| Decision | When | Note |
|---|---|---|
| **Always verify against the real OpenGate** (`sensehat` on api.opengate.es), never a mock | 2026-08-26, standing | Found 4 API bugs, a false positive on working code, and a path bug that silently disabled conflict detection. Log in with `--no-web` so the North-API-only path cannot invalidate a browser session. Ask before anything that writes. |
| Governance: **community tool, for now** | 2026-08-25 | Provisional, pending Charlie's conversation at Amplía. Gates the extension. |
| No JavaScript parser dependency | 2026-08-26 | `validate` does JSON, declared files, bracket balance and per-family traps. The script itself is covered by typings + the editor's real type-checker. |
| Untyped values are `any`, not `unknown` | 2026-08-26 | `unknown` rejects `entity['x']._value._current.value > n`, which is correct JavaScript. The goal is catching mistyped identifiers. |
| The generated `jsconfig` adapts to the code | 2026-08-26 | `checkJs` off where a top-level `return`, an untyped helper parameter or a dynamic index would flag working code. 8 of 13 live artifacts are fully checked, 5 completion-only. |
| Typings generated from the documentation, committed | 2026-08-26 | `tools/ogdocgen`, run by hand. Hand-maintenance got 4 signatures wrong. |
| `watch` refuses on conflict, with no `--force` | 2026-08-26 | Overwriting someone else's edit should not be one keystroke away. |
| Descriptor table lives in `internal/unwrap`, not a new `internal/artifact` | 2026-08-25 | `unwrap` already is the artifact package; a sibling would import it all back. `DX_ANALYSIS` §12. |

---

## How to pick up

```bash
# Authenticate read-only against the real platform
OG_PASSWORD=sensehat og login --email sensehat@amplia.es --no-web

# Regenerate the typings after a documentation change
go run ./tools/ogdocgen \
  -docs /home/charlie/Dropbox/Charlie/0-Env/amplia/1-Projects/opengate/odm-documentation-hugo \
  -out internal/typegen/templates

# The verification that matters: pull every live artifact and type-check it
og rules pull-all --dir r --org sensehat
og connectors pull-all --dir c --org sensehat
# then, in each directory: tsc -p jsconfig.json   (expect 13/13 clean)
```

`docs/opengate-documentation-handoff.md` is with the documentation team. When its item 1 lands
(`libs/ogapi-docs` front matter), the widget `$api` typings become a small job.

## Open questions

1. Release the two unreleased commits as **v2.1.1**, or hold them for the widget typings and ship a
   single **v2.2.0**?
2. `og pf` as an alias of `og provision`, for symmetry with `og cf`?
3. `validate` non-overridable on a provision-function deploy, given bulk-scale data corruption?
4. Widget code-field allowlist: keep the transitional content heuristic, or enumerate widget types
   against a live tenant first?
