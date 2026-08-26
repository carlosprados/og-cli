# JSON output contract

Commands whose machine-readable output is consumed by something other than a
human — a CI pipeline, an editor extension — emit a **versioned envelope**:

```json
{
  "schemaVersion": 1,
  "kind": "diff",
  "data": { }
}
```

`schemaVersion` is versioned from the first release that emits it, rather than
after the first breaking change. Consumers should read it and refuse a version
they do not know.

## Compatibility rules

- **Additive changes do not bump the version.** A new field in `data`, or a new
  value for an existing enumerated field, is a minor release. Consumers must
  ignore unknown fields.
- **Removing or renaming a field, or changing its type, bumps `schemaVersion`.**
  That is a breaking change and is called out in the release notes.
- `kind` names the payload shape. A new `kind` is additive.

Commands that predate the envelope print their payload bare. They are not
retrofitted: doing so would break every existing consumer.

## Exit codes

Consistent across `diff` (and `validate`, when it lands):

| Code | Meaning |
|---|---|
| `0` | success, or no differences |
| `1` | differences found |
| `2` | error |

`1` and `2` are distinct so a pipeline can tell "the tenant has drifted" from
"the command could not run". Only commands given `--exit-code` return `1`;
without the flag, finding differences is a normal, successful run.

## `kind: "diff"`

Emitted by `og rules diff`, `og connectors diff` and `og provision diff` under
`-o json`.

```json
{
  "schemaVersion": 1,
  "kind": "diff",
  "data": {
    "kind": "rule",
    "id": "r-1",
    "name": "Environmental anomaly",
    "dir": "rules/env-anomaly",
    "state": "local changes",
    "metadata": [
      {
        "kind": "changed",
        "path": "parameters[0].value",
        "before": "30",
        "after": "28"
      }
    ],
    "code": [
      {
        "file": "javascript.js",
        "added": 2,
        "removed": 1,
        "unified": "      var t = entity['sensor.temperature'];\n    - …\n    + …\n"
      }
    ],
    "origin": "org acme, channel default_channel, profile staging",
    "ignored": "ignored (same-tenant): __v, lastAccess"
  }
}
```

### Fields

| Field | Notes |
|---|---|
| `kind` | artifact family: `rule`, `connector-function`, `provision-function`, `workspace`, `dashboard` |
| `id` | present when a base snapshot records it |
| `dir` | the local artifact directory that was compared |
| `state` | `clean`, `local changes`, `remote changes`, `conflict`, `unknown` |
| `metadata[]` | structural differences over the canonical form; omitted when empty |
| `metadata[].kind` | `added`, `removed`, `changed` |
| `metadata[].path` | dotted, with array indices in brackets: `config.columns[2].title` |
| `metadata[].before` / `after` | rendered compactly; a value over 60 characters is truncated with its real length appended |
| `code[]` | one entry per extracted source file that differs; omitted when empty |
| `code[].onlyIn` | `local` or `remote` when the file exists on one side only; absent otherwise |
| `origin` | where the local tree was pulled from, when a base snapshot records it |
| `ignored` | the volatile fields excluded from this comparison, when any were present |

### Direction

`before` is the **remote** side and `after` the **local** one, throughout — both
in `metadata` and in `code`. The report therefore reads as what deploying would
do to the platform, the same direction as `git diff`.

### `state`

`state` is the three-way classification against the snapshot taken at pull time.
`unknown` means there is no snapshot — a tree pulled before the store existed,
or built by hand — and only the raw two-way comparison is available.

`conflict` means both sides moved since the pull. Two sides that moved to the
*same* value are `clean`, not a conflict.
