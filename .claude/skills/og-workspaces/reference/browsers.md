# Reference: Browser Widgets (`datamodelBrowser`, `devicePlansBrowser`, `connectorFunctionsBrowser` & `connectorFunctionsCatalogBrowser`)

OpenGate provides native read-only catalog browser widgets that let users explore system models, plans, and connector functions directly from a dashboard.

---

## 1. Data Model Browser (`datamodelBrowser`)

The `datamodelBrowser` displays a comprehensive navigation tree and search interface for the platform's data models and datastreams catalogs.

### JSON Schema Configuration

Refer to [Common Widget Fields](./commonFields.md) for grid layout wrapping properties.

* **Ftype**: Must be set to `"datamodels"`.
* **config**: Not required/supported. Both `config` and its fields are omitted.

```json
{
    "type": "datamodelBrowser",
    "Ftype": "datamodels",
    "wid": "my-datamodel-browser-001"
}
```

### Grid Wrapper Example

```json
{
    "width": 3,
    "height": 2,
    "x": 0,
    "y": 4,
    "w": 3,
    "h": 2,
    "i": "my-datamodel-browser-001",
    "moved": false,
    "definition": {
        "type": "datamodelBrowser",
        "Ftype": "datamodels",
        "wid": "my-datamodel-browser-001"
    }
}
```

---

## 2. Device Plans Browser (`devicePlansBrowser`)

The `devicePlansBrowser` is a catalog browser widget used to inspect, filter, and view the active device plans configured in the workspace.

### JSON Schema Configuration

* **Ftype**: Omitted (no functional data filtering type is declared).
* **config**: Not required/supported. Both `config` and its fields are omitted.

```json
{
    "type": "devicePlansBrowser",
    "wid": "my-device-plans-browser-001"
}
```

### Grid Wrapper Example

```json
{
    "width": 3,
    "height": 2,
    "x": 0,
    "y": 2,
    "w": 3,
    "h": 2,
    "i": "my-device-plans-browser-001",
    "moved": false,
    "definition": {
        "type": "devicePlansBrowser",
        "wid": "my-device-plans-browser-001"
    }
}
```

---

## 3. Connector Functions Browser (`connectorFunctionsBrowser`)

The `connectorFunctionsBrowser` displays the list and status of all configured connector functions inside the current workspace/organization.

### JSON Schema Configuration

* **Ftype**: Omitted.
* **config**: Not required/supported. Both `config` and its fields are omitted.

```json
{
    "type": "connectorFunctionsBrowser",
    "wid": "my-connector-functions-browser-001"
}
```

### Grid Wrapper Example

```json
{
    "width": 3,
    "height": 2,
    "x": 0,
    "y": 0,
    "w": 3,
    "h": 2,
    "i": "my-connector-functions-browser-001",
    "moved": false,
    "definition": {
        "type": "connectorFunctionsBrowser",
        "wid": "my-connector-functions-browser-001"
    }
}
```

---

## 4. Connector Functions Catalog Browser (`connectorFunctionsCatalogBrowser`)

The `connectorFunctionsCatalogBrowser` is a catalog browser widget used to inspect the catalog of available connector function templates that can be instantiated.

### JSON Schema Configuration

* **Ftype**: Omitted.
* **config**: Not required/supported. Both `config` and its fields are omitted.

```json
{
    "type": "connectorFunctionsCatalogBrowser",
    "wid": "my-connector-catalog-browser-001"
}
```

### Grid Wrapper Example

```json
{
    "width": 3,
    "height": 2,
    "x": 0,
    "y": 2,
    "w": 3,
    "h": 2,
    "i": "my-connector-catalog-browser-001",
    "moved": false,
    "definition": {
        "type": "connectorFunctionsCatalogBrowser",
        "wid": "my-connector-catalog-browser-001"
    }
}
```

---

## Verified Examples

From `recursos/workspace_1` (SmartCity-Demo, OpenGate v13.1.0):

```json
[
    {
        "width": 3,
        "height": 2,
        "x": 0,
        "y": 0,
        "definition": {
            "type": "connectorFunctionsBrowser",
            "wid": "1779702769868-15"
        },
        "w": 3,
        "h": 2,
        "i": "1779702769868-15",
        "moved": false
    },
    {
        "width": 3,
        "height": 2,
        "x": 0,
        "y": 2,
        "definition": {
            "type": "connectorFunctionsCatalogBrowser",
            "wid": "1779702761450-14"
        },
        "w": 3,
        "h": 2,
        "i": "1779702761450-14",
        "moved": false
    },
    {
        "width": 3,
        "height": 2,
        "x": 0,
        "y": 6,
        "definition": {
            "type": "devicePlansBrowser",
            "wid": "1779699818136-12"
        },
        "w": 3,
        "h": 2,
        "i": "1779699818136-12",
        "moved": false
    },
    {
        "width": 3,
        "height": 2,
        "x": 0,
        "y": 8,
        "definition": {
            "type": "datamodelBrowser",
            "Ftype": "datamodels",
            "wid": "1779699809544-11"
        },
        "w": 3,
        "h": 2,
        "i": "1779699809544-11",
        "moved": false
    }
]
```
