# Reference: `iframeWidget`

Embeds external web pages or dashboards directly into your OpenGate dashboard using an HTML `<iframe>` or exposes a button that launches the URL.

## JSON Schema Configuration

Refer to [Common Widget Fields](./commonFields.md) for grid and standard widget layout wrapping properties.

```json
{
    "type": "iframeWidget",
    "wid": "my-iframe-001",
    "config": {
        "title": "Iframe Documentation",
        "reloadPeriod": "0",
        "about": "Embedded external web documentation",
        "hideWidgetTitle": "visible",
        "boxed": false,
        "customActions": null,
        "url": "https://documentation.opengate.es/",
        "showAsButton": false
    }
}
```

> Note: `iframeWidget` has **no `Ftype`** field — omit it entirely.

---

## Config Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `url` | `string` | ✅ | Absolute HTTP/HTTPS URL to embed. |
| `showAsButton` | `boolean` | — | If `true`, renders a clickable launcher button instead of embedding inline. Default `false`. |
| `title` | `string` | — | Widget header or button label text. |
| `hideWidgetTitle` | `string` | — | `"visible"` or `"hidden"`. Default `"visible"`. |
| `boxed` | `boolean` | — | Renders a card border boundary. Default `false`. |
| `reloadPeriod` | `string` | — | Auto-refresh rate in seconds. `"0"` to disable. |
| `customActions` | `null\|array` | — | Toolbar actions array, default `null`. |


---

## Verified Example

From `recursos/workspace_0` (SmartCity-Demo, OpenGate v13.1.0):

```json
{
    "width": 6,
    "height": 3,
    "x": 0,
    "y": 0,
    "w": 6,
    "h": 3,
    "i": "1779443119259-10",
    "definition": {
        "type": "iframeWidget",
        "wid": "1779443119259-10",
        "config": {
            "title": "Iframe",
            "reloadPeriod": "0",
            "about": "",
            "hideWidgetTitle": "visible",
            "boxed": false,
            "customActions": null,
            "url": "https://documentation.opengate.es/",
            "showAsButton": true
        }
    }
}
```

---

## Source

Verified from `recursos/workspace_0` (SmartCity-Demo, OpenGate v13.1.0).
