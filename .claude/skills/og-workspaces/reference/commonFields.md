# Common Widget Fields Reference

All OpenGate widgets embedded in a dashboard's layout grid share a set of common wrapper fields and base configuration properties. To optimize the skill and avoid repetition, these common properties are documented here.

---

## 1. Grid Layout Fields
Inside the dashboard's `grid` array, each item defines the placement and container size of a widget:

| Field | Type | Description |
|---|---|---|
| `x` | `integer` | Horizontal position in the grid layout (0-indexed). |
| `y` | `integer` | Vertical position in the grid layout (0-indexed). |
| `width` / `w` | `integer` | Grid cell width span. Must match each other. |
| `height` / `h` | `integer` | Grid cell height span. Must match each other. |
| `i` | `string` | Grid item identifier. **Must exactly match** the `wid` inside the `definition`. |
| `moved` | `boolean` | Indicates if the widget has been moved (default `false`). |
| `definition` | `object` | Contains the actual widget's functional configuration. |

---

## 2. Base Widget Wrapper (`definition`)
Every widget definition has the following root fields:

| Field | Type | Description |
|---|---|---|
| `type` | `string` | The widget class type identifier (e.g., `customTable`, `customChart`, `maps`, `DeviceAlarmsList`). |
| `Ftype` | `string` | The primary functional resource category context (e.g. `*`, `alarms`, `maps`, `datasets`). It determines the compatibility between widgets and searching types when sharing filters between different widgets. |
| `wid` | `string` | Unique widget instance ID. Usually formatted as `{timestamp}-{counter}`. |
| `config` | `object` | Specific parameters required by the widget class type. |

---

## 3. Common Configuration Properties (`definition.config`)
These standard options are shared across the `config` blocks of most widgets:

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | `string` | - | The user-visible header text of the widget. |
| `hideWidgetTitle` | `string` | `"visible"` | Title bar visibility state (`"dynamic"`, `"visible"` or `"hidden"`). |
| `boxed` | `boolean` | `true` / `false` | Wraps the widget container in a visual card border outline. |
| `reloadPeriod` | `string` / `integer` | `"0"` | Period (seconds) to automatically refresh widget data (`"0"` to disable). |
| `about` | `string` | `""` | User-defined descriptive markdown text shown in the widget's info panel. |
| `customActions` | `array` | [] | Array of custom actions to add to the widget's title bar. Each action has an icon, a title, and the code to execute when clicked. Default is `[]`. Definition can be found in [Custom Actions Object](#custom-actions-object). |

#### Custom Actions Object
| Field | Type | Default | Description |
|---|---|---|---|
| `icon` | `string` | - | The icon to display for the action. |
| `_actionCode` | `string` | - | The javascript code to execute when the action is clicked. |
| `title` | `string` | - | The title of the action. |

## 4. Filter field configuration

For those widgets that have a filter (Ftype <> ""), they have the following configuration:

| Field | Type | Default | Description |
|---|---|---|---|
| `filter` | `object` | {} | Filter configuration. It contains the following fields: |
| `filter.type` | `string` | `"basic"` | Filter type (`"basic"`, `"advanced"` or `"shared"`). |
| `filter.oql` | `string` | `` | OQL query string. |
| `filter.value` | `string` | `` | Opengate Filter format json string scaped. |

### Basic mode

In basic mode, the filter is applied to all available entities in the widget. You have to introduce the value of the filter in the `value` field. Oql field is calculated automatically based on the widget type and the Ftype field.

``` json
{
    "type": "basic",
    "oql": "provision.administration.identifier ~ \"entity_3\" or provision.device.specificType ~ \"entity_3\" or provision.asset.specificType ~ \"entity_3\" or device.specificType ~ \"entity_3\" or asset.specificType ~ \"entity_3\" or resourceType ~ \"entity_3\" or provision.device.communicationModules[].subscriber.identifier ~ \"entity_3\" or provision.device.communicationModules[].subscription.identifier ~ entity_3",
    "value": "entity_3"
}
```

### Advanced mode

In advanced mode, `oql` has the filter defined in OQL format and `value` field has an Opengate Filter format json string scaped based on oql and its calculated automatically from oql.

``` json
{
    "type": "advanced",
    "oql": "provision.administration.identifier._current.value eq \"entity_3\"",
    "value": "{\"eq\":{\"provision.administration.identifier._current.value\":\"entity_3\"}}"
}
```

### Shared mode

In shared mode, only the `id` field is added to the filter configuration to identify the widget from which the filter is copied. This `id` must match with the `wid` of the widget in the grid layout.
Other fields in filter object (type, oql, value) must be the copied from the widget that it refers to (`wid`).
Both widgets must be in the same dashboard and have the same `Ftype`.

``` json
{
    "type": "advanced",
    "oql": "provision.administration.identifier._current.value eq \"entity_3\"",
    "value": "{\"eq\":{\"provision.administration.identifier._current.value\":\"entity_3\"}}",
    "id": "1779433574493-1"
}
```

## 5. Widget internal filters

Widgets with `Ftype` != "" can have internal filters that are applied to the data retrieved by the widget.

- **privateFilter**: It can be used to filter the data in a private way, without affecting other widgets. It must contain OQL format query string.

``` json
{
    "privateFilter": "alarm.identifier eq \"asdf\"",
}
```

- **templateFilter**: It can be used to filter the data in a template. It must contain OQL format query string.

The differene from `privateFilter` is that `templateFilter` can use variables from the widget's context. The variable are enclosed in `$variable_name$` (without quotes) and can be used in the OQL query string. Numbers must not be enclosed in quotes, strings must be enclosed in quotes.

``` json
{
    "templateFilter": "alarm.entityIdentifier eq \"$provision.device.identifier$\""
}
```
or
``` json
{
    "templateFilter": "device.powersupply.battery.charge gt $device.battery.current$"
}
```


## 6. OQL Query string

OQL is a query language used in OpenGate to retrieve data from the platform and more sql friendly.

``` oql
resourceType eq "entity.device" or Battery._current.value gt 5 or (provision.string.enum._current.value in ("hola2","adios2"))
```

#### Operators supported by OQL:

**1. Relational operators**

* **eq** = equal
* **neq** = not equal
* **exists** = exists
* **like or ~** = like
* **gt or >** = greater than
* **lt or <** = less than
* **gte or > =** = greater than or equal to
* **lte or <=** = less than or equal to
* **in** = in (parameter must be an array of strings between parentheses: ('value1','value2','value3'))
* **nin** = not in (parameter must be an array of strings between parentheses: ('value1','value2','value3'))

**2. Logic operators**
* **and** = and
* **or** = or

**3. Grouping**
* **()** = parentheses

## 7. Window filter

Window filter is used to filter the data in a widget based on a time window. Not all the widgets support window filter (see specific widget fields). If a widget support window filter, it will have the following structure. 'windowFilter' can also be found in filter object because it is used to share with other widgets.

All information about window filter variants can be found in [Window Filter Variants](./windowFilterVariants.md).

