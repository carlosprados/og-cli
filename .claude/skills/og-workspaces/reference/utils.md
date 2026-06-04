# Reference: Global Scripting Context & Utilities

All custom scripts inside OpenGate's Advanced Widgets (such as `customTable`, `customChart`, etc.) run in a specialized sandbox. This document covers the globally available parameters, objects, and helper utilities.

Each widget has its own configuration. These objects are available globally in runtime.

---

## Global Context Objects

### 1. `$api`
Provides a JavaScript client to interface with the OpenGate REST API.
- [Reference API Documentation](https://amplia-iiot.github.io/opengate-js/)

### 2. `$user`
Provides information about the currently logged-in user:
```json
{
    "email": "email@domain.com",
    "workgroup": "workgroup_name",
    "domain": "domain_name",
    "profile": "profile_name",
    "countryCode": "ES",
    "langCode": "en",
    "timezone": "Europe/Madrid"
}
```

### 3. `$moment`
An instance of the Moment.js library for simple date parsing, validation, manipulation, and formatting.
- [Moment.js Documentation](https://momentjs.com/docs/)

### 4. `console`
Standard browser debugger console (e.g. `console.log("Debug message")`).

### 5. `Promise`
Global ES6 Promise object enabling execution of concurrent async scripts.

### 6. `http`
A lightweight wrapper encapsulating `useFetch` (Nuxt 4) for simple network requests.

---

## Navigation & UI Utilities

These helper functions can be triggered within table row actions or button click handlers to route the user within the OpenGate platform.

- **`openDashboard(workspaceId, dashboardId, newPage)`**
  Opens the selected dashboard inside the specified workspace.
- **`openEntityDashboard(entityIdentifier [, organization [, resourceType [, newPage]]])`**
  Opens a temporary dashboard loaded with data for the specified entity.
  *Defaults: `resourceType` defaults to `'entity.device'`, `organization` defaults to the user's organization.*

---

## Global Script Parameters

Advanced scripts receive the following data objects natively based on their execution context:

### 1. `entityData`
Only available when the widget is rendered inside an **Entity Dashboard Template**.
```json
{
  "provision.administration.identifier": {
    "_value": {
      "_current": {
        "value": "device_1"
      }
    }
  },
  "provision.administration.organization": {
    "_value": {
      "_current": {
        "value": "organization_name"
      }
    }
  },
  "provision.administration.channel": {
    "_value": {
      "_current": {
        "value": "channel_name"
      }
    }
  }
}
```

### 2. `alarmData`
Available when the template is opened for a specific alarm.
```json
{
  "identifier": "270dd9f9-1396-4660-bb5f-8d8b471e1dcd",
  "name": "activityForbidden",
  "rule": "activityForbidden",
  "severity": "INFORMATIVE",
  "priority": "LOW",
  "organization": "organization_name",
  "status": "CLOSED",
  "openingDate": "2019-06-27T08:57:36+02:00"
}
```

### 3. `relatedEntities`
An array containing data of entities related to the selected entity.
```json
[{
  "provision.administration.identifier": {
    "_value": { "_current": { "value": "related_device_1" } }
  }
}]
```

### 4. `timeserieData`
Only available when a template is opened via a timeseries table widget row.
```json
{
  "config": {
    "identifier": "timeserie_id",
    "name": "Battery charge history",
    "identifierColumn": "EntityID"
  },
  "data": {
    "bucketEnd": "2025-12-11T13:00:00+01:00",
    "EntityID": "entity_1",
    "Powersupply battery charge Current Value": 34
  }
}
```

### 5. `dashboardFilters`

Contains filters selected in the dashboard (e.g. tasks, jobs, operations, alarms, etc).

```json
{
  "tasksSelected": [],
  "jobsSelected": [],
  "operationNameSelected": [],
  "operationStatusSelected": [],
  "operationResultSelected": [],
  "alarmNameSelected": [],
  "ruleNameSelected": [],
  "alarmSeveritySelected": [],
  "alarmStatusSelected": []
}
```