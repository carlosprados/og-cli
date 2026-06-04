# Reference: Rules Browser (`rulesBrowser`)

The `rulesBrowser` widget provides a read‑only view of the rules defined in the OpenGate platform. It is typically used within a dashboard to let operators inspect rule configurations, statuses, and associated actions.

## JSON Schema Minimum Configuration
Refer to [Common Widget Fields](./commonFields.md) for grid layout properties. The specific properties for `rulesBrowser` in the `definition` object are minimal:

```json
{
  "type": "rulesBrowser",
  "wid": "my-rules-browser-001"
}
```

* **type**: Must be set to `"rulesBrowser"`.
* **wid**: Unique widget identifier.
* **config**: Not required or supported for this widget; omit it.

## Grid Wrapper Example
When placing the widget on a dashboard grid, wrap the definition as follows:

```json
{
  "width": 3,
  "height": 2,
  "x": 0,
  "y": 0,
  "w": 3,
  "h": 2,
  "i": "my-rules-browser-001",
  "moved": false,
  "definition": {
    "type": "rulesBrowser",
    "wid": "my-rules-browser-001"
  }
}
```

The `rulesBrowser` widget does not accept additional configuration fields, and its appearance follows the platform’s default styling. It can be combined with other widgets on the same dashboard grid.

---

### Usage Context
Include the widget in a dashboard JSON under the `grid` array:

```json
"grid": [
  {
    "width": 3,
    "height": 2,
    "x": 0,
    "y": 0,
    "definition": {
      "type": "rulesBrowser",
      "wid": "my-rules-browser-001"
    },
    "w": 3,
    "h": 2,
    "i": "my-rules-browser-001",
    "moved": false
  }
]
```

---
