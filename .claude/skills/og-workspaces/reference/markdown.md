# Reference: `markdown`

Renders static or dynamic Markdown content inside a dashboard tile for documentation, headers, or rich-text.

## JSON Schema Configuration

Refer to [Common Widget Fields](./commonFields.md) for grid and standard widget layout wrapping properties.

```json
{
    "type": "markdown",
    "wid": "my-markdown-001",
    "config": {
        "title": "Markdown",
        "reloadPeriod": "0",
        "about": "",
        "hideWidgetTitle": "visible",
        "boxed": false,
        "customActions": null,
        "content": "# My Title\n\nSome **bold** text and *italic* text.\n\n- Item 1\n- Item 2"
    }
}
```

> Note: `markdown` has **no `Ftype`** field (omit entirely).


---

## Config Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `content` | `string` | ✅ | Markdown content to render (use `\n` for newlines inside JSON string). |
| `title` | `string` | — | Widget header title. |
| `hideWidgetTitle` | `string` | — | `"visible"` or `"hidden"`. Default `"visible"`. |
| `boxed` | `boolean` | — | Wraps widget in a card border. Default `false`. |
| `reloadPeriod` | `string` | — | Auto-refresh interval in seconds. `"0"` to disable. |
| `customActions` | `null\|array` | — | Toolbar actions array, default `null`. |


---

## Supported Markdown Syntax

The `content` field supports standard CommonMark Markdown:

```markdown
# H1 Heading
## H2 Heading
### H3 Heading

**Bold text**
*Italic text*
~~Strikethrough~~
`inline code`

- Unordered list item
    - Nested item

1. Ordered item 1
2. Ordered item 2

```python
def example():
    pass
```

| Col 1 | Col 2 |
|-------|-------|
| A     | B     |

[Link text](https://url.com)
```

---

## Verified Example

From `recursos/workspace_0` (SmartCity-Demo, OpenGate v13.1.0):

```json
{
    "type": "markdown",
    "wid": "1779437118780-9",
    "config": {
        "title": "Markdown",
        "reloadPeriod": "0",
        "about": "",
        "hideWidgetTitle": "visible",
        "boxed": false,
        "customActions": null,
        "content": "```markdown\n# Documento de Prueba en Markdown\n\n## Características de Markdown\n\n### 1. Elementos de Texto\n* **Texto en negrita**\n* *Texto en cursiva*\n* ~~Texto tachado~~\n* Texto con `código en línea`\n```"
    }
}
```
