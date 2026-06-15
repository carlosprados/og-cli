# Provision Functions JavaScript — platform guide

> Distilled from the OpenGate Provision Processor JavaScript API
> (`provision_functions_js_api.md`). A provision function ("provision processor")
> transforms inbound rows (typically Excel) into ODM provisioning actions.

## The contract — two mandatory functions

A provision processor script MUST define these two functions (called by the Java
`Main_Module.processRow`):

- `normalizeRawObject(rawObject)` — receives a map of the raw inbound row, validates
  and transforms it, and returns a normalized object. Do value validation/cleanup here.
- `actionsPlanning(normalizedObject)` — receives the normalized object and returns an
  **array of actions** to apply. Empty array = nothing to do for this row.

Optional:
- `customErrorTransformer(errorManager)` — returns a custom String message when a
  provision error is caught (e.g. duplicated entity), for the Excel result column.

`processRow` returns `{ "scriptDirectResult": "OK" | "<error text>", "actionsToDo": [ ...actions ] }`.

## Authoring rules (platform constraints)

- Use **single quotes** (`'`) for strings, not double quotes.
- Use **block comments** (`/* */`), not line comments (`//`).
- The script is stored in `scriptProcessor.script`. With `og provision`, you edit it
  as a normal multi-line `.js` file (`scriptProcessor__script.js`); `deploy` re-injects
  it as a JSON string.
- `name` must match `^[a-zA-Z0-9]+$` (alphanumeric only, unique per organization).

## Minimal template

```javascript
function normalizeRawObject(rawObject) {
  try {
    return {
      organization: readMapValue(rawObject, 'Organization', '', 'A'),
      channel:      readMapValue(rawObject, 'Channel Name', '', 'B'),
      device_identifier: readMapValue(rawObject, 'Serial number', '', 'D').replace(/\s/g, '')
    };
  } catch (e) {
    printLog('>> normalizeRawObject(): exception: ' + e);
    throw e;
  }
}

function actionsPlanning(normalizedObject) {
  var actions = [];
  var deviceEntity = new Entity()
    .addDatastream('provision.administration.channel', normalizedObject.channel)
    .addDatastream('provision.administration.organization', normalizedObject.organization)
    .addDatastream('provision.administration.identifier', normalizedObject.device_identifier);

  if (!checkDevice(normalizedObject.device_identifier)) {
    actions.push(CREATE_DEVICE_ACTION(deviceEntity.entityJson));
  } else {
    actions.push(UPDATE_DEVICE_ACTION(deviceEntity.entityJson));
  }
  return actions;
}
```

## Runtime API cheat-sheet

**Entity builder** — `new Entity().addDatastream(datastreamId, value)` (chainable);
`.entityJson` yields the ODM JSON to pass into an action builder.

**Action builders** (`Action_Utils`) — each returns an action object; push them from
`actionsPlanning`. `entityJson` is the `Entity.entityJson`; `description` is optional:
- Assets: `CREATE_ASSET_ACTION`, `UPDATE_ASSET_ACTION`, `PATCH_ASSET_ACTION`, `DELETE_ASSET_ACTION`
- Devices: `CREATE_DEVICE_ACTION`, `UPDATE_DEVICE_ACTION`, `PATCH_DEVICE_ACTION`,
  `DELETE_DEVICE_ACTION(entityJson, full, description)` (`full=true` also deletes related subs)
- Subscriptions: `CREATE_/UPDATE_/PATCH_/DELETE_SUBSCRIPTION_ACTION`
- Subscribers: `CREATE_/UPDATE_/PATCH_/DELETE_SUBSCRIBER_ACTION`

Action shape: `{ action: 'POST'|'PUT'|'PATCH'|'DELETE', resourceType|entityType, actionDescription, json, full }`.

**Existence / lookup utils** (`V8_Utils`, query the platform during planning):
- `checkAsset(id)` / `checkDevice(id)` / `checkSubscription(id)` / `checkSubscriber(id)` → boolean
- `getAssetEntity(id)` / `getDeviceEntity(id)` / `getSubscriptionEntity(id)` / `getSubscriberEntity(id)` → entity | null
- `duplicatedDsInDevices(currentId, {dsId: value}, ...)` (and `...InAssets/InSubscriptions/InSubscribers`) → boolean

**Low-level V8 API**:
- `printLog(msg)` — write an INFO log line (the only logging channel; no live WS logs).
- `getEntity(entityId, resourceType, queryContextParams)` → entity JSON | null
- `entitiesGenericSearch(searchFilter, queryContextParams)` → search result JSON | null
  (`queryContextParams`: `{ utc, flattened, defaultSorted }`).

**Error handling** — `throw new Error('Provision Processor Error: ...')` from
`actionsPlanning` to abort the row with a descriptive message.

## Iterate with `og provision plan` (no data mutated)

```bash
og provision pull <pp-id> --dir provision/ --org <org>
$EDITOR provision/<slug>/scriptProcessor__script.js
og provision deploy provision/<slug> --update --org <org>
og provision plan <pp-id> --file sample.xlsx --rows 3 --org <org>   # inspect the action plan JSON
# happy with the plan? run it:
og provision bulk <pp-id> --file data.xlsx --org <org>
```
