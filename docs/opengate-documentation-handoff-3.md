# Handoff 3 — the numbers you asked for

**Target repo:** `odm-documentation-hugo`
**Answers:** the "What to send back" section of your update to `opengate-documentation-handoff-response.md`
**Regenerated from:** `develop` at `01b940c3` (*Merge branch 'DOC/handoff-corrections-and-sync'*)
**Shipped in:** og-cli `v2.2.0`, released 2026-08-29
**Date:** 2026-08-29

You asked for the comparison after regenerating on a `develop` that includes the corrections, as the
only honest measure of whether any of this helped. Here it is.

---

## The measure

`before` is what the generator produced from the documentation as it stood when we sent handoff 1.
`after` is `develop` at `01b940c3`, which includes everything you merged.

| | before | after |
|---|---:|---:|
| declarations | 437 | 437 |
| with a real return type | **3** | **126** |
| parameters | 680 | 676 |
| with a real type | **129** | **376** |
| carrying `@deprecated` | **0** | **41** |

The declaration count is unchanged, which is the point: the same API surface, described well enough
to be useful. Two thirds of the parameters and a quarter of the return types went from `any` to
something a type-checker can act on, and 41 superseded symbols now strike through in an editor with
your replacement text on hover — including the warning that `cf.collection` and `cf.response` take
their arguments in the opposite order, which is the one that would have produced working-looking
code that sends the payload as the criteria.

All 13 live artifacts in the `sensehat` organization type-check clean against the result.

Not all of that gain is yours. Two of the parser bugs it exposed were ours: we ignored every table
headed `Param` — 206 of them, the majority — because only `Parameter` and `Property` were accepted,
and we read return types only from the heading form, catching 3 of the 224 you had already written.
Your corrections are what made those visible.

## Three notes back

**1. The case-insensitivity fix you offered is not needed.** Your §3 suggested we match type names
case-insensitively, for the 42 entries in lowercase and the 6 wrapped in `<code>`. We already did
both: the generator lowercases and strips inline tags before matching. Those 48 entries were being
consumed correctly the whole time. Nothing to do on either side — recorded so it does not sit on
someone's list.

**2. The `**Returns**` work is effectively complete.** Of 448 entries, 230 now state a return type
and **228 of those resolve to a real type**. The two that do not are `getAddressTypeFromAddress` and
its deprecated twin, which you deliberately asked rather than guessed. That is the right call and we
are not asking you to guess now.

**3. `openAlarm` was the good catch.** Confirming it takes six parameters rather than seven did more
than fix one signature: it removed the need for the optionality heuristic to carry weight.
Declarations where every parameter collapsed to optional fell from 7 to 4, so the heuristic is now a
safety net rather than a load-bearing guess. The bracketed notation on the other nine functions
changed nothing in our output — we had already inferred them from the prose — which is the good
outcome: it means the documentation and the generator agree.

## Still open, and not urgent

The optionality convention from gap 1 of handoff 2. Nine functions are bracketed and `openAlarm` is
settled, so the pressure is off, but there is still no marker in the general case and the generator
still reads prose in the Description column to decide. A `Required` column, or the bracketed
signature applied consistently, would retire that heuristic entirely.

## One thing that came out of this you may want

`opengate-js` 16.0.0 now publishes its own TypeScript declarations — 246 of them, emitted from the
JSDoc on `prepack`. We consume those directly for the widget `$api` surface and generate nothing for
it, exactly as you advised. Verified from the public registry: they resolve under all four
TypeScript module-resolution strategies, and a misspelled builder is caught with a "Did you mean".

Worth knowing on your side: the previously published `15.5.0` shipped **no** `types/` directory at
all — the `prepack` hook landed after that release — so anyone who tried this before 16.0.0 would
have found nothing and concluded it did not work.

---

## Owners

The platform JavaScript APIs are Iñaki's. `opengate-js` is Chema's (`jousepo`).
