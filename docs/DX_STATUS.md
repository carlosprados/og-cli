# DX roadmap — state and next steps

Short, current, and meant to be read first. The reasoning behind every decision is in
`DX_ANALYSIS.md`; this file says where things stand.

**Last updated:** 2026-08-29
**Branch:** `og-cli/ogdocgen-doc-response`, off `main` (`458463b`), **6 commits, none pushed**
**Released:** `v2.1.0`. Everything below is unreleased.
**Health:** build clean, 14 packages green, lint at the 42-issue baseline, all 13 live sensehat
artifacts type-check clean, and a freshly pulled workspace diffs to "No differences".

**Sibling repo:** `og.nvim` now exists at `../og.nvim` — one commit, git initialised, **no GitHub
remote yet and nothing pushed**. Target remote: `github.com/carlosprados/og.nvim`.

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
| 15 | `og <family> show <id> --path` — the remote side of an editor diff | unreleased |
| 11a | **og.nvim** — Neovim plugin, thin shell over the binary | `../og.nvim`, uncommitted to any remote |

## Not done

| Item | Why it is not done | Size |
|---|---|---|
| **Phase 11b: VS Code extension** | Unblocked 2026-08-28 and now also unblocked technically: `og show --path` exists, and og.nvim proved the "thin shell over `-o json`" contract against a real editor. **This is the next piece of work.** Plan below. | large |
| **og.nvim: try it in anger** | Written and verified headlessly against the live tenant, but Charlie has not yet run it in his own config. **Monday 2026-08-31.** Until then, treat it as unproven in daily use. | — |
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

## Next up — Phase 11b, the VS Code extension

Separate repository, as the original brief says — mixing TypeScript into the binary's repo
complicates the GoReleaser release for no gain. **Confirmed with Charlie 2026-08-29:
`github.com/carlosprados/og-vscode`**, checked out beside og-cli and og.nvim.

**Start with the minimal useful slice, not the scaffolding**: binary discovery, the native diff and
diagnostics. That is what made og.nvim useful in an afternoon. The TreeView is the biggest piece and
the least load-bearing — it comes after.

**og.nvim is the reference implementation.** It is smaller, it is already verified against the live
tenant, and it settled the questions that matter — so port its decisions rather than rediscovering
them:

| Decision, already made in og.nvim | Why it carries over |
|---|---|
| Zero API logic outside the binary | Two sources of truth is the failure mode; the TypeScript one would go stale |
| Remote side of a diff from `og <family> show --path` | Exists and is verified byte-identical to what `pull` writes |
| Never re-render a diff | `og diff` carries three-way markers, the pruned workspace tree and the ignored-fields note |
| Artifact resolution walks up, nearest wins | A widget edit must act on the widget, not the workspace |
| Deploy-on-save off by default | A plugin that pushes on every save eventually pushes a half-thought |
| No credentials stored | og's profile is the single place they can be wrong |
| Trailing-newline handling on remote content | Otherwise every diff opens showing a phantom last-line change |

What the extension needs that the Neovim one did not:

1. **Binary management.** Find `og` on `PATH`; otherwise download the matching GoReleaser asset,
   verify the checksum, cache in `globalStorage`. An `og.path` setting overrides.
2. **TreeView.** Profiles → Workspaces → Dashboards → Widgets, plus Rules, Connector Functions and
   Provision Functions as sibling roots. Populated from `-o json`, children fetched lazily.
3. **Native diff.** A `TextDocumentContentProvider` for an `og-remote:` scheme, handed to
   `vscode.diff(remoteUri, localUri, title)`. Never render a diff by hand.
4. **Diagnostics.** `og <family> validate -o json` → `DiagnosticCollection`. The finding's `file`
   field is separate from its message, and the message reads as a continuation of it — prefix it
   when the diagnostic does not land on that file's own document, as og.nvim does.
5. **Save hooks, not `og watch`.** `onDidSaveTextDocument` → `og … deploy --update`. Two watchers
   over one tree produce duplicate deploys. `og watch` serves terminal and Neovim users.
6. **Publish to Open VSX as well as the Marketplace**, or Cursor, Windsurf and VSCodium users are
   excluded. Carry the unofficial/community disclaimer into the marketplace listing and the
   extension's own README, not only into this repository's.

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

---

## Session log — 2026-08-26 → 2026-08-29

Six commits on `og-cli/ogdocgen-doc-response`, plus a new sibling repository. Nothing pushed.

**The documentation round closed.** All four items of `opengate-documentation-handoff.md` landed on
the documentation side; their answer is in `opengate-documentation-handoff-response.md` and ours back
in `opengate-documentation-handoff-2.md`. Consuming it moved the catalogue from 3 typed return types
to 126, from 129 typed parameters to 376, and added 41 `@deprecated` symbols carrying the
documentation's own wording — including the warning that `cf.collection` and `cf.response` reverse
their arguments. Measuring it turned up two parser bugs of ours: tables headed `Param` (206 of them)
were ignored, and return types were read only from the heading form.

**A defect found and fixed on the live tenant.** The typings flagged
`isInsertAction(entity) || isUpdateAction(entity)` in sensehat's PROVISION_RULE. The platform team
confirmed `entity` is an ambient global, so the arguments were vestigial; they were deployed away.
First defect the generated typings found and closed in production.

**opengate-js 16.0.0.** Charlie released it during the session. Verified from the public registry:
246 declarations ship, resolve under all four module-resolution strategies, and a misspelled builder
is caught with a "Did you mean". The published 15.5.0 had shipped no `types/` at all — the `prepack`
hook that generates them landed after that release.

**Four capabilities built**: widget `$api` typings, `og widget check`, the hierarchical
`og workspace diff`, and `og workspace watch` with the sync snapshots the pull had never recorded.
Then `og show --path`, which both editor plugins need, and og.nvim itself.

### Open questions

1. Release the unreleased commits as **v2.1.1**, or hold for more and ship **v2.2.0**?
2. Push the branch and open a PR? `danvifer` also works on this repo — fetch before pushing.
3. Create `github.com/carlosprados/og.nvim` and push the plugin.
4. `og pf` as an alias of `og provision`, for symmetry with `og cf`?
5. `validate` non-overridable on a provision-function deploy, given bulk-scale data corruption?
6. Widget code-field allowlist: keep the content heuristic, or enumerate widget types live first?
7. Send the documentation team the post-regeneration numbers they asked for.
