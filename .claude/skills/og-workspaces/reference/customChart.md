# Reference: `customChart`

Displays data graphically by integrating with Apache ECharts (v5). Supports external, internal, or mock data.

Full documentation here: https://documentation.opengate.es/ux/workspaces/dashboards/widgets/advanced/customchart/index.html

## JSON Schema Configuration
Refer to [Common Widget Fields](./commonFields.md) for grid and standard widget layout wrapping properties. Specific properties for `customChart` in `config` include:

```json
{
  "type": "customChart",
  "Ftype": "*",
  "wid": "widget-chart-id",
  "config": {
    "title": "Custom Chart",
    "reloadPeriod": "0",
    "about": "gwegwergewrg",
    "hideWidgetTitle": "dynamic",
    "boxed": true,
    "customActions": [
      {
        "icon": "ogicon-abacus",
        "_actionCode": "console.log(\"Hola Mundo!!!\");",
        "title": "Console log"
      }
    ],
    "model": "icon",
    "_extraValue": "{ \"provision.device.identifier\": {\"_value\": {\"_current\": {\"value\": \"entity_identifier\",\"provType\": \"IDENTIFIER\",\"date\": \"2019-11-26T18:11:40.501+01:00\"}}} }",
    "_extraValueRelated": "[{ \"provision.device.identifier\": {\"_value\": {\"_current\": {\"value\": \"entity_related\",\"provType\": \"IDENTIFIER\",\"date\": \"2019-11-26T18:11:40.501+01:00\"}}} }]",
    "_timeserieValue": "{ \"config\": {\"identifier\":\"timeserie_identifier\",\"name\":\"timeserie_name\",\"timeBucket\":300,\"bucketColumn\":\"end bucket name\",\"bucketInitColumn\":\"init bucket name\",\"identifierColumn\":\"identifier column name\",\"retention\":30672000,\"origin\":\"2025-04-29T22:00:00Z\"}, \"data\": { \"end bucket name\": \"value\", \"identifier column name\": \"value 2\" } }",
    "_alarmValue": "{\"identifier\":\"70fa3325-f6a8-4003-acfb-342864b3d669\",\"name\":\"alarm name\",\"rule\":\"rule name\",\"description\":\"rule description\",\"severity\":\"INFORMATIVE\",\"priority\":\"LOW\",\"organization\":\"organization\",\"channel\":\"channel\",\"entityIdentifier\":\"identifier\",\"subEntityIdentifier\":\"identifier\",\"resourceType\":\"entity.device\",\"status\":\"OPEN\",\"openingDate\":\"2025-05-20T13:39:06.947+02:00\"}",
    "_dashboardFilters": "{\"tasksSelected\":[],\"jobsSelected\":[],\"operationNameSelected\":[],\"operationStatusSelected\":[],\"operationResultSelected\":[],\"alarmNameSelected\":[],\"ruleNameSelected\":[],\"alarmSeveritySelected\":[],\"alarmStatusSelected\":[]}",
    "_widgetConfigCode": "return {\n  legend: {},\n  tooltip: {},\n  dataset: {\n    source: [\n      ['product', '2012', '2013', '2014', '2015'],\n      ['Matcha Latte', 41.1, 30.4, 65.1, 53.3],\n      ['Milk Tea', 86.5, 92.1, 85.7, 83.1],\n      ['Cheese Cocoa', 24.1, 67.2, 79.5, 86.4]\n    ]\n  },\n  xAxis: [\n    { type: 'category', gridIndex: 0 },\n    { type: 'category', gridIndex: 1 }\n  ],\n  yAxis: [{ gridIndex: 0 }, { gridIndex: 1 }],\n  grid: [{ bottom: '55%' }, { top: '55%' }],\n  series: [\n    // These series are in the first grid.\n    { type: 'bar', seriesLayoutBy: 'row' },\n    { type: 'bar', seriesLayoutBy: 'row' },\n    { type: 'bar', seriesLayoutBy: 'row' },\n    // These series are in the second grid.\n    { type: 'bar', xAxisIndex: 1, yAxisIndex: 1 },\n    { type: 'bar', xAxisIndex: 1, yAxisIndex: 1 },\n    { type: 'bar', xAxisIndex: 1, yAxisIndex: 1 },\n    { type: 'bar', xAxisIndex: 1, yAxisIndex: 1 }\n  ]\n};",
    "allowWidgetFilter": true
  }
}
```

* Field definitions:
  * `_extraValue`: Entity data (used for testing).
  * `_extraValueRelated`: Related entities data (used for testing).
  * `_timeserieValue`: Timeserie data (used for testing).
  * `_alarmValue`: Alarm data (used for testing).
  * `_dashboardFilters`: Dashboard filters (used for testing).
  * `_widgetConfigCode`: contains the javascript code that generates the chart configuration. Code is wrapped in a function:
  ```javascript
  async function (entityData, relatedEntities, timeserieData, alarmData, dashboardFilters, filters, callback){
    // _widgetConfigCode here -> returns an ECharts configuration object
  }
  ```

### Script Parameters Context

When providing code in the widget (e.g., inside the platform UI, which is then mapped to the JSON), the script execution receives several parameters depending on the configuration:

Apart of [Global Script Parameters](./utils.md), the following parameters are available:

1. `filters`: Introduced by the user. Includes:
   - `generic`: generic text filter.
   - `period`: `{ "from": "...", "to": "..." }`.
   - `column`: JSON object with filters per column (e.g., `{ "column_value": { "operator": "eq", "value": "text" } }`).
   - `sort`: Array containing sorting preferences.
2. `callback`: Function used to send table data only when the API/HTTP calls are resolved (e.g., `callback(data);`).

### Utilities

Includes `Global Context Objects` and `Navigation & UI Utilities` from [Utils](./utils.md).


## Expected Return Format

The script MUST return a valid ECharts configuration object.

### Example ECharts Return Structure
```javascript
return {
  title: {
    text: 'Example Chart',
    left: 'center'
  },
  tooltip: {
    trigger: 'item'
  },
  legend: {
    orient: 'vertical',
    left: 'left'
  },
  series: [
    {
      name: 'Access From',
      type: 'pie',
      radius: '50%',
      data: [
        { value: 1048, name: 'Search Engine' },
        { value: 735, name: 'Direct' },
        { value: 580, name: 'Email' }
      ]
    }
  ]
};

