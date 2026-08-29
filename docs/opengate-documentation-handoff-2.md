# Handoff 2 — what ogdocgen consumed, and what is still missing from the model

**Target repo:** `odm-documentation-hugo`
**Answers:** `opengate-documentation-handoff-reply.md` (your side) / `opengate-documentation-handoff-response.md` (ours)
**Date:** 2026-08-26

All four items landed and all four were consumed. The generator now reads 43 pages into 437
declarations, of which **41 carry a `@deprecated` tag with your wording** and **107 a real return
type** — against 0 and 3 before. The 13 live `sensehat` artifacts still type-check, with one
deliberate exception described in §5.

You asked to be told what is missing rather than have it worked around. Five things, in the order we
would fix them.

---

## 1. Optionality is never stated, and it is the one thing that breaks working code

**Severity: high.** This is the single largest source of false positives.

No signature on any of the 43 pages marks a parameter optional. Once parameters have real types —
which they now do, thanks to the `Param` tables — TypeScript enforces arity, and correct production
code fails:

| Call in a live `sensehat` artifact | Documented signature | Result |
|---|---|---|
| `openAlarm(null,'Battery low',ruleName,'CRITICAL','HIGH','…')` — 6 args | 7 parameters | 4 rules rejected |
| `executeOperation('0101…','LORAWAN_PROVISION',10,key,null,null,null,10000,'DELAYED',params)` — 10 args | 11 parameters | 1 rule rejected |
| `entityValue(entity,'provision.device.identifier')` — 2 args | 3 parameters | 1 connector function rejected |

All three work in production, so the documented arity is not the required arity.

The fact *is* in the pages — but in the Description column, phrased six different ways:
`**Default**: 60000`, `If is undefined it will open on …`, `Optional organization name`,
`If not provided null will be set`, `If not defined current time will be used`, `It can be null`.
ogdocgen now matches that phrasing to decide optionality, which is a heuristic over prose and will
drift the moment someone writes it a seventh way.

`openAlarm`'s `extraInfo` has no signal at all — its description is `Extra information.` We only get
it right because the *first* parameter says "If is undefined", and TypeScript cannot express
"optional, then required", so everything after it becomes optional too. That is luck, not reading.

**What would fix it:** one machine-readable convention. Either bracket the parameter in the
signature — `openAlarm(subEntityIdentifier, alarmName, ruleName, severity, priority,
alarmDescription, [extraInfo])`, which the generator already understands — or add a `Required`
column to the parameter tables. Either one, applied consistently, removes the heuristic entirely.

## 2. `getVariableValue(variable)` — the table contradicts the page's own example

**Severity: medium.** It rejected a live rule.

`rules_advanced_datastreams_api.md:19` types the parameter as `String`. Fourteen lines further down,
the page's own example passes a number:

```javascript
var myVar = 2;
var finalVar = getVariableValue(myVar);
```

> Result is 2;
> This function returns the same value

A live `sensehat` rule reads a numeric rule parameter through it and works. The parameter is `*`,
not `String`. Currently corrected in `tools/ogdocgen/overrides.go`; the override goes away when the
table does.

## 3. 40 `**Returns**` lines state prose where the type should be

**Severity: low, but cheap to fix and it is pure gain.**

224 entries now carry a `**Returns**` line and the generator uses every one of them. 40 of those
resolve to `any` anyway, because the line describes the value instead of naming its type. These are
not the 224 you already know are missing — these are the ones you have *already written*, one word
short of being usable:

| Entry | Says | Almost certainly |
|---|---|---|
| `utils.bytes.fromHexString`, `utils.bytes.fromText` | "return an `Uint8Array` object equivalent to…" | `Uint8Array` |
| `utils.bytes.toHexString`, `utils.bytes.toText` | "return the hexadecimal string representation." | `String` |
| `utils.date.period.*` (8 entries) | "return an object with initial and final times…" | `Object` |
| `entity._at` / `_date` / `_source` / `_sourceInfo` (and the 4 deprecated globals) | "Specified datastream … field. It can be complex object." | `*` |
| `getDatastreamFromEntity`, `getCommsDatastreamFromEntity` | "Completed datastream or undefined…" | `Object` |
| `getCounterValue` | "Incremented value or reset value." | `Number` |
| `utils.atcmd.toDBm` | "The translation to dBm value" | `Number` |
| `utils.odm.encryptString` / `decryptString`, `getAddressTypeFromAddress` | "The originalValue encrypted" | `String` |
| `ogStep`, `ogStepResponse` | "OG step object" | `Object` |
| `dlms.getInvocationCounter` | `` `number` (integer). `` | `Number` — the parenthetical is what defeats a parser |

The convention that already works elsewhere on these pages is `**Returns**: Type - prose`. Putting
the type before the dash is the whole fix.

## 4. Two headings for the same table

**Not a defect, recorded so it is not "tidied" later.** Parameter tables are headed `Param` in 206
places, `Parameter` in 99 and `Property` in 81. The generator accepts all three. It previously
accepted only the last two, which silently dropped the types of 206 tables — that was our bug, and
finding it is what raised typed parameters from 129 to 377. No change wanted; just do not assume one
spelling is dead.

## 5. `entity` as an ambient global — confirmed, and it found a vestigial argument

Modelled as you described: `entity` and `gateway` are ambient globals of the rule scope, and the
three `is*Action()` helpers take no argument. The override that widened them is gone.

That leaves one genuine diagnostic on live code — `sensehat`'s `PROVISION_RULE` calls
`isInsertAction(entity) || isUpdateAction(entity)`. The arguments are ignored by the platform, so
this is dead weight rather than a bug, and it is ours to remove. Recorded here only so nobody reads
it as a disagreement with §3 of your reply: it is the opposite, it is that section working.

## 6. Not merged yet

`DOC/rules-entity-global` is still open against `develop`. We generate from `develop`, so the
`entity`-is-a-global prose and the `**Returns**: Boolean` on the three `is*Action` entries are not in
what we ship. Nothing is blocked on it — ping us when it merges and a regeneration picks it up.

## 7. On the `$api` surface

Taken: ogdocgen generates nothing for `$api` and never parsed `libs/ogapi-docs`. We will consume
`opengate-js`'s own `.d.ts` for the pinned version rather than build anything. Nothing to report back
on `api-model.json` yet — when we do consume it, the two gaps you named (a rest parameter with no
`@param`, and the 21 `@param` tags naming a parameter the function does not have) are the first
things we will hit, and we will bring them back with the specific call sites.

---

## Owners

Items 1–3 and 5–6: the platform JavaScript APIs, Iñaki's. Item 7: `opengate-js`, Chema's
(`jousepo`).
