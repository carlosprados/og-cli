# DX roadmap — state and next steps

Short, current, and meant to be read first. The reasoning behind every decision is in
`DX_ANALYSIS.md`; this file says where things stand.

**Last updated:** 2026-08-26
**Branch:** `og-cli/ogdocgen-doc-response`, off `main` (`458463b`)
**Released:** `v2.1.0`. Three commits sit on top of it, unreleased: connector-function typings, the
documentation-driven generator, and the documentation-response round below.
**Health:** build clean, 14 packages green (ogdocgen now has tests), lint at the 42-issue baseline,
all 13 live sensehat artifacts type-check clean.

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
| 10c | Documentation response consumed: `@deprecated`, documented return types, `Param` tables, `entity` as ambient global | unreleased |
| 11 | Widget `$api` typings, from `opengate-js` 16.0.0's own declarations | unreleased |
| 12 | `og widget check` — diagnostics inside the platform's async wrapper | unreleased |
| 13 | `og workspace diff` — hierarchical renderer for workspaces and dashboards | unreleased |
| 14 | `og workspace watch` + sync snapshots recorded on workspace pull | unreleased |

## Not done

| Item | Why it is not done | Size |
|---|---|---|
| **Phase 11: VS Code extension** | **Unblocked 2026-08-28.** Charlie has had the conversation at Amplía: we publish it ourselves as an unofficial community product, carrying that disclaimer. Nothing gates it now — it is the next substantial piece of work. | large |
| Volatile fields for rules and provision functions | None observed in a real payload. Deliberately empty, with a test pinning that intent. Fill from a live GET if one ever shows one. | trivial |

---

## Decisions taken

| Decision | When | Note |
|---|---|---|
| Widget diagnostics come from a rebuilt wrapper, not from a laxer config | 2026-08-28 | TS1108 on a top-level `return` is unavoidable in every module configuration tried, and every widget has one. Rather than force `checkJs` on and ask the author to ignore a known error, `og widget check` wraps the code in the platform's own async function and checks that — the return and the `await` become ordinary and nothing is suppressed. Two JS-vs-TS idioms are set aside by name, narrowly enough that a misspelled `$api` member still reports. |
| Workspace watch deploys dashboards, not workspaces | 2026-08-28 | The watch rule is that a change resolves to the smallest deployable unit, and for a widget edit that is its dashboard; deploying a whole workspace per keystroke is the blast radius the rule exists to avoid. It also needed `og workspace pull` to record a sync snapshot, which it never did — without a base every classification is Unknown and the conflict guard is a blind overwrite. Verified live: a genuine conflict is refused. |
| Workspace diff matches children by identity, not by position | 2026-08-28 | Matching by index turns one widget inserted at the top of a dashboard into "every widget changed", which is the output that makes a diff useless. Position is reported separately as a move, and a move counts as a change in both the marker column and the JSON status — anything else traps a consumer filtering on status. Verified live: reorder, insert and delete each report as themselves. |
| The widget `$api` surface is consumed, never generated | 2026-08-28 | `opengate-js` 16.0.0 ships 246 `.d.ts` from its own JSDoc, resolved from node_modules under all four module-resolution strategies. Verified against sensehat's two live `$api` widgets. Generating a second copy from `libs/ogapi-docs` would have been strictly worse — those pages came from an unmerged branch. og writes the injected globals and the async wrapper's parameters, and nothing else. |
| **Always verify against the real OpenGate** (`sensehat` on api.opengate.es), never a mock | 2026-08-26, standing | Found 4 API bugs, a false positive on working code, and a path bug that silently disabled conflict detection. Log in with `--no-web` so the North-API-only path cannot invalidate a browser session. Ask before anything that writes. |
| Governance: **community tool**, and we publish the extension ourselves | 2026-08-25, settled 2026-08-28 | The Amplía conversation happened: og stays a community tool, and the VS Code extension ships from us under an explicit "unofficial, community product" disclaimer rather than as an Amplía product. No longer provisional, and Phase 11 is no longer gated. The `production: true` guard on watch stays — a community tool writing to a customer's platform has no support channel to catch the fallout. |
| No JavaScript parser dependency | 2026-08-26 | `validate` does JSON, declared files, bracket balance and per-family traps. The script itself is covered by typings + the editor's real type-checker. |
| Untyped values are `any`, not `unknown` | 2026-08-26 | `unknown` rejects `entity['x']._value._current.value > n`, which is correct JavaScript. The goal is catching mistyped identifiers. |
| The generated `jsconfig` adapts to the code | 2026-08-26 | `checkJs` off where a top-level `return`, an untyped helper parameter or a dynamic index would flag working code. 8 of 13 live artifacts are fully checked, 5 completion-only. |
| Typings generated from the documentation, committed | 2026-08-26 | `tools/ogdocgen`, run by hand. Hand-maintenance got 4 signatures wrong. |
| The generator refuses to write when a documented family yields no page | 2026-08-26 | The rules front matter was renamed `alarms-js-api` → `rules-js-api` with no alias. The generator kept exiting zero, wrote three families instead of four, and left a stale `rule-advanced.generated.d.ts` on disk claiming to be current. Silence is the wrong answer to a family disappearing. |
| Optionality read from the parameter description, not asserted | 2026-08-26, largely resolved 2026-08-28 | No signature marked an optional parameter, so real parameter types made TypeScript enforce an arity the documentation never stated. Reported as gap 1 in handoff 2. The documentation answered it: nine functions now bracket their trailing optionals, and the platform team confirmed **`openAlarm` takes six parameters, not seven**. The heuristic now changes nothing that the brackets do not already say — declarations where every parameter collapses to optional fell from 7 to 4 — so it stays as a safety net rather than a load-bearing guess. |
| `entity` and `gateway` are ambient globals; the vestigial arguments were removed from the live rule | 2026-08-26 | The platform team confirmed the documentation was right: the helpers take no argument because `entity` is a global of the rule function. The typings flagged `isInsertAction(entity) \|\| isUpdateAction(entity)` in sensehat's PROVISION_RULE as a true positive, and the arguments were deployed away (`og rules deploy --update`). Back to 13/13 clean — this is the first defect the generated typings found and closed on a live tenant. |
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

The documentation round is closed: `docs/opengate-documentation-handoff.md` (ours) →
`docs/opengate-documentation-handoff-response.md` (theirs) → `docs/opengate-documentation-handoff-2.md`
(ours, the model gaps that are left). All four original items landed on their side.

## Open questions

1. Release the three unreleased commits as **v2.1.1**, or hold them for the widget `$api`
   declarations and ship a single **v2.2.0**? The catalogue moved enough this round to argue for
   shipping it: 41 `@deprecated` tags, 126 real return types against 3, and 376 typed parameters
   against 129.
2. `og pf` as an alias of `og provision`, for symmetry with `og cf`?
3. `validate` non-overridable on a provision-function deploy, given bulk-scale data corruption?
4. Widget code-field allowlist: keep the transitional content heuristic, or enumerate widget types
   against a live tenant first?
