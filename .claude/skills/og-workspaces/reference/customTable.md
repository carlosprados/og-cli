# Reference: `customTable`

The `customTable` widget allows displaying data in a list format based on logic encoded by the user. The data source can be external, internal, both, or even fabricated.

## JSON Schema Minimum Configuration
Refer to [Common Widget Fields](./commonFields.md) for grid and standard widget layout wrapping properties. Specific properties for `customTable` in `config` include:

```json
{
    "type": "customTable",
    "wid": "widget-id",
    "config": {
        "title": "Custom Table",
        "reloadPeriod": "0",
        "about": "",
        "hideWidgetTitle": "hidden",
        "boxed": false,
        "customActions": null,
        "model": "icon",
        "_extraValue": "{ \"provision.device.identifier\": {\"_value\": {\"_current\": {\"value\": \"entity_identifier\",\"provType\": \"IDENTIFIER\",\"date\": \"2019-11-26T18:11:40.501+01:00\"}}} }",
        "_extraValueRelated": "[{ \"provision.device.identifier\": {\"_value\": {\"_current\": {\"value\": \"entity_related\",\"provType\": \"IDENTIFIER\",\"date\": \"2019-11-26T18:11:40.501+01:00\"}}} }]",
        "_timeserieValue": "{ \"config\": {\"identifier\":\"timeserie_identifier\",\"name\":\"timeserie_name\",\"timeBucket\":300,\"bucketColumn\":\"end bucket name\",\"bucketInitColumn\":\"init bucket name\",\"identifierColumn\":\"identifier column name\",\"retention\":30672000,\"origin\":\"2025-04-29T22:00:00Z\"}, \"data\": { \"end bucket name\": \"value\", \"identifier column name\": \"value 2\" } }",
        "_alarmValue": "{\"identifier\":\"70fa3325-f6a8-4003-acfb-342864b3d669\",\"name\":\"alarm name\",\"rule\":\"rule name\",\"description\":\"rule description\",\"severity\":\"INFORMATIVE\",\"priority\":\"LOW\",\"organization\":\"organization\",\"channel\":\"channel\",\"entityIdentifier\":\"identifier\",\"subEntityIdentifier\":\"identifier\",\"resourceType\":\"entity.device\",\"status\":\"OPEN\",\"openingDate\":\"2025-05-20T13:39:06.947+02:00\"}",
        "_dashboardFilters": "{\"tasksSelected\":[],\"jobsSelected\":[],\"operationNameSelected\":[],\"operationStatusSelected\":[],\"operationResultSelected\":[],\"alarmNameSelected\":[],\"ruleNameSelected\":[],\"alarmSeveritySelected\":[],\"alarmStatusSelected\":[]}",
        "_widgetConfigCode": "return [{\nfield_one: 'campo 1'}];",
        "allowPagination": 1,
        "allowGrouping": true,
        "allowWidgetFilter": true,
        "expandable": true,
        "groupBy": [
        
        ],
        "pageElements": 19,
        "compactTable": true,
        "columns": [
        {
            "groupable": true,
            "divider": true,
            "sortable": true,
            "operator": "eq",
            "type": "number",
            "text": "field 1",
            "value": "field_one",
            "isOgIdentifier": true,
            "filterable": true,
            "width": 34,
            "disableOperators": true
        }
        ],
        "tableId": "field_one",
        "showFilter": false
    }
}
```

## Specific config properties

* `tableId`: The field name to use as primary key for the table.
* `expandable`: Enables the display of a new column at the left of the table that, when clicked, expands a new table with the data related to the entity.
* `allowGrouping`: Allows grouping the table by a field.
* `compactTable`: Reduces the height of the table rows.
* `pageElements`: Number of elements to display per page.
* `allowPagination`: Allows pagination for the table. 0 is disabled, 1 is local pagination (code handles it), 2 is enabled server side (API should support).
* `allowWidgetFilter`: Enables filters in widget and columns.
* `groupBy`: Array of fields to group the table by.
* `_extraValue`: Entity data (used for testing).
* `_extraValueRelated`: Related entities data (used for testing).
* `_timeserieValue`: Timeserie data (used for testing).
* `_alarmValue`: Alarm data (used for testing).
* `_dashboardFilters`: Dashboard filters (used for testing).
* `_widgetConfigCode`: contains the javascript code that generates the table data. Code is wrapped in a function:
  ```javascript
  async function (entityData,relatedEntities,timeserieData,alarmData,dashboardFilters,filters,pageElements,page,callback)  {
    // _widgetConfigCode here -> returns an array of JSON objects
  }
  ```

### Script Parameters Context

When providing code in the widget (e.g., inside the platform UI, which is then mapped to the JSON), the script execution receives several parameters depending on the configuration:

> [!CAUTION]
> For the `_widgetConfigCode` field you MUST read the full documentation here: https://documentation.opengate.es/ux/workspaces/dashboards/widgets/advanced/customtable/index.html 

*Below you have a summary of the official documentation*

Apart of [Global Script Parameters](./utils.md), the following parameters are available:

1. `filters`: Introduced by the user. Includes:
   - `generic`: generic text filter.
   - `period`: `{ "from": "...", "to": "..." }`.
   - `column`: JSON object with filters per column (e.g., `{ "column_value": { "operator": "eq", "value": "text" } }`).
   - `sort`: Array containing sorting preferences.
2. `pageElements` and `page`: Used for pagination (e.g. `pageElements=10`, `page=1`), unavailable when `allowPagination` is 0 (disabled).
3. `callback`: Function used to send table data only when the API/HTTP calls are resolved (e.g., `callback(data);`).


## Columns Configuration
For each column you can define:
- **`text`**: Name to show in headers.
- **`value`**: The JSON field to read from the data returned by the function.
- **`sortable`**: Permits sorting for this column.
- **`groupable`**: Allows grouping by this field in the table.
- **`filterable`**: Allows filtering by this field in the table.
- **`type`**: Data type of the filter (only used when filterable is true).
- **`divider`**: Draws a separator between this column and the next.
- **`operator`**: Operator to use for filtering (only used when filterable is true).
- **`disableOperators`**: Disables the operator dropdown (only used when filterable is true).
- **`width`**: Width of the column (only used when filterable is true).
- **`isOgIdentifier`**: Whether the column is an OpenGate identifier.

### Simple JSON Data
```json
{
    "jsonfield": "value to display. It can be HTML."
}
```

### Complex JSON Data
```json
{
    "jsonfield": {
        "value": "value to display. It can be HTML",
        "_style": "cell custom style in css format",
        "_chart": "displays an echarts chart. Overrides others in this item",
        "_extension": "combines _chart and _table elements when expandable rows are enabled"
    }
}
```

### Embedded Table Data
Using `_table` displays a nested table inside the column:
```javascript
{
    columns: ['each', 'item', 'is', 'a', 'column'],
    data: [
        ['data 1', 'in', 'columns', 'order', { /* simple/complex object */ }],
        ['data 2', 'in', 'columns', 'order', { /* simple/complex object */ }]
    ]
}
```

## Expected Return Format

The script MUST return an array of JSON objects compatible with the configured columns.

### Example Table Data Return
```javascript
return [
  { 
    field_one: 'value 1', 
    field_two: { 
        _chart: { 
            // ECharts configuration object
        }
    } 
  },
  { field_one: 'value 1', field_two: 'value 2' }
];
```