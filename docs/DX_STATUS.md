# DX roadmap — state and next steps

Short, current, and meant to be read first. The reasoning behind every decision is in
`DX_ANALYSIS.md`; this file says where things stand.

**Last updated:** 2026-08-29
**Branch:** `main`, pushed and clean
**Released:** `v2.4.0`. Nothing unreleased.
**Health:** build clean, tests green, lint at the 42-issue baseline, 13/13 live sensehat
artifacts type-check clean under TypeScript 5.9 and 6.0.

**Three repositories now, all public and all pushed:**

| | | |
|---|---|---|
| [`og-cli`](https://github.com/carlosprados/og-cli) | `v2.4.0` | the binary; everything else drives it |
| [`og.nvim`](https://github.com/carlosprados/og.nvim) | unversioned | Neovim plugin |
| [`og-vscode`](https://github.com/carlosprados/og-vscode) | `0.4.0` packaged; `0.3.0` still live on the Marketplace | VS Code extension |

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
| 15 | `og <family> show <id> --path` — the remote side of an editor diff | v2.2.0 |
| 16 | `og whoami` — session state, local and offline, with usable exit codes | v2.3.0 |
| 11a | **og.nvim** — Neovim plugin | published |
| 11b | **og-vscode** — VS Code extension | published, `0.3.0` |
| 17 | `og dashboard show --path` + `og dashboard diff` + `dashboards_code` — a widget is editable | v2.4.0 |
| 18 | og.nvim's README and `doc/og.txt` rewritten around the nine commands it has | og.nvim `main` |
| 19 | Both plugins level again: widgets diff and deploy, `anchor` replaces the per-family special cases | og.nvim `main`, og-vscode `0.4.0` |

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
| The dashboard is the smallest addressable unit; the widget is not | 2026-08-29 | A widget is a grid item, not something the platform can name, so it gets no `show`, `diff` or `deploy` of its own — its dashboard does. Editing a widget still works exactly as editing a rule does; only the deploy and the comparison move up a level, and both say so. The same boundary `og workspace watch` already drew. |
| `show --path` matches widgets by identity, not by grid position | 2026-08-29 | The `NN__` prefix in a widget directory is the remote grid order at the moment of the pull. Match on it and a reorder on the platform orphans every path in a local tree. Where identity is ambiguous — same type, neither widget carrying an id — the path is reported as not found rather than guessed. Consistent with the workspace diff, which already matched children by identity. |
| A missing cobra subcommand exits 0, so a new verb is checked by exit code | 2026-08-29 | `og dashboard diff` did not exist, and cobra answered it with the family's help page and exit 0. Both editor plugins rendered that help as a diff, and og-vscode confirmed a real dashboard deploy against it. Nothing looked broken from the outside. |
| The TUI stays out of the artifact-code business | 2026-08-29 | No family shows artifact code in the TUI — not rules, not connector functions — so a widget code viewer would put dashboards ahead of the rest for no reason, and it needs a viewport the TUI does not have. When a code viewer lands it should land for all four families at once. |
| Both plugins are thin shells; all API logic stays in the binary | 2026-08-29 | Reimplementing a call in Lua or TypeScript makes two sources of truth, and the copy is the one that goes stale. Every platform interaction is a child process. |
| og.nvim browses with a picker, og-vscode with a sidebar | 2026-08-29 | Parity is of capability, not of chrome. `vim.ui.select` is overridden by LazyVim, Telescope, fzf-lua and snacks, so it inherits the user's own finder and imposes no layout; a hand-rolled tree would be more code and worse. |
| The session is checked before the work, not after the 401 | 2026-08-29 | A 401 collapses "you never logged in" and "your session expired", which need different things from the reader. `og whoami` is local and instant, so asking first costs a process and no request. Validate stays unguarded: it needs no credentials. |
| Passwords travel in the environment, never in argv | 2026-08-29 | Arguments are readable by anything that can list processes. Neither plugin stores a credential; og writes the token to its own profile at 0600. |
| Publish to the Marketplace through the web portal, not a PAT | 2026-08-29 | Creating an Azure DevOps organization now demands an Azure subscription and disables *Continue* without saying why — it blocked Charlie for half an hour. The portal needs neither, and global PATs retire 2026-12-01 anyway. |
| An unparseable version is not a missing binary | 2026-08-29 | A source build prints `og dev (commit: unknown)`. Both plugins treated that as "CLI not found" while it sat on the PATH. Running and being parseable are different questions. |
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

## Next up

Phase 11 is done: both plugins exist and are published. What is left is polish and
one gap, in the order I would take them.

### 1. Upload og-vscode 0.4.0 to the Marketplace

Packaged and verified — `og-vscode-0.4.0.vsix`, run against the live tenant from the
archive itself — but **not uploaded**: the portal route needs a browser session.
`PUBLISHING.md` §5: **+ New extension** → **Visual Studio Code** → the `.vsix`.

### 2. Decide whether the plugins get tags

Neither repository has ever had one, so there is no way to say "fixed in". og-vscode
at least has a `version` in `package.json`; og.nvim has nothing, and it is installed
from a git ref. Open question 4 below, still open.

### 3. Open VSX

Never done, and it is what Cursor, Windsurf and VSCodium install from. Needs a GitHub
login, the Publisher Agreement signed, a token, and `ovsx create-namespace carlosprados`.
Steps are in `og-vscode/PUBLISHING.md` §2.

### 4. Publishing without a human

Both plugins are uploaded by hand. A workflow on a `v*` tag would end that — see
`og-vscode/PUBLISHING.md` §6. Note the Marketplace half needs a token, and Azure DevOps
global PATs retire on 1 December 2026, so the durable answer there is Entra ID.

### 5. Unproven in daily use

Neither plugin has been used in anger. Charlie tries og.nvim in his own LazyVim on
Monday 2026-08-31. Both grew features nobody has exercised.

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

1. `og pf` as an alias of `og provision`, for symmetry with `og cf`?
2. `validate` non-overridable on a provision-function deploy, given bulk-scale data
   corruption?
3. Widget code-field allowlist: keep the content heuristic, or enumerate widget types
   against a live tenant first?
4. Does og.nvim want a version and tags at all? It is installed from a git ref, so
   nothing forces it — but without one there is no way to say "fixed in".

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

---

## Session log — 2026-08-29, the editor layer shipped

Three releases of og-cli and two published plugins, from a standing start.

**og-cli v2.2.0** — the editor layer: `og widget check`, the hierarchical
`og workspace diff`, `og workspace watch` with the sync snapshots the pull had never
recorded, `og <family> show --path`, and the platform catalogue regenerated after the
documentation team answered the handoff (3 → 126 typed return types, 129 → 376 typed
parameters, 41 `@deprecated`).

**v2.2.1** — a defect found by running the VS Code extension rather than by compiling
anything: VS Code 1.135 ships TypeScript 6, which reports `target: es5` as an error, so
every artifact directory og had ever generated showed one in the editor the typings
exist to serve. The fix TypeScript suggests breaks TypeScript 5, so only the target
moved and `lib` stayed at es5.

**v2.3.0** — `og whoami`, prompted by building the plugins: a 401 cannot distinguish
"never logged in" from "expired an hour ago".

**og.nvim and og-vscode** — built, published, and then levelled. The Neovim one was
written first and the extension grew past it; the four missing pieces (login, binary
management, browse, session check) were ported back the same day.

Things that only surfaced by running the software: the TypeScript 6 error above, a
415 from GitHub's API for sending one `Accept` everywhere, the diff resolving a
metadata file instead of code when invoked from the tree, a diagnostic printing the
filename twice, and the unparseable-version bug. None of them was findable by
compiling.

Also sent: `docs/opengate-documentation-handoff-3.md`, the numbers the documentation
team asked for. Written but **not delivered** — Charlie decides the channel.

A draft email to `general@amplia.es` announcing all of it sits unsent in Gmail
(draft `r-3699741285082560721`).
