# Reference: `FullDevicesList`

The `FullDevicesList` widget allows displaying entities data in a table format with several features like grouping, filtering, and pagination.

## JSON Schema Minimum Configuration
Refer to [Common Widget Fields](./commonFields.md) for grid and standard widget layout wrapping properties. Specific properties for `FullDevicesList` in `config` include:

```json
{
    "type": "widget-list-type",
    "Ftype": "entities|alarms",
    "wid": "widget-id",
    "config": {
        "title": "",
        "reloadPeriod": "0",
        "about": "",
        "hideWidgetTitle": false,
        "boxed": false,
        "customActions": null,
        "columns": [
            {
            "config_title": "(provisionGenericIdentifier - provisionAdministrationIdentifier - provision.administration.identifier)",
            "title": "FORM.TITLE.CURRENT.VALUE",
            "path": "provision.administration.identifier._current.value",
            "availableFields": [
                
            ],
            "alias": "Prov. Identifier",
            "firstPath": "provision.administration.identifier._current.value",
            "expand": false,
            "isOgIdentifier": true,
            "hiddenColumn": false
            },
            {
            "config_title": "(provisionGenericIdentifier - provisionAdministrationIdentifier - provision.administration.identifier)",
            "title": "FORM.TITLE.AT",
            "path": "provision.administration.identifier._current.at",
            "availableFields": [
                
            ],
            "alias": "fecha Identifier",
            "firstPath": "provision.administration.identifier._current.at",
            "expand": false,
            "isOgIdentifier": false,
            "widths": {
                "width": "100",
                "minWidth": "1",
                "maxWidth": "150",
                "hAlign": "text-center",
                "textWrap": true
            }
            },
            {
            "config_title": "(provisionGenericIdentifier - provisionAdministrationIdentifier - provision.administration.identifier)",
            "title": "FORM.TITLE.CURRENT.VALUE",
            "path": "provision.administration.identifier._current.value",
            "availableFields": [
                
            ],
            "alias": "Custom Action",
            "firstPath": "provision.administration.identifier._current.value",
            "expand": false,
            "isOgIdentifier": false,
            "hiddenColumn": false,
            "expandable": false,
            "buttonColumn": true,
            "_actionCode": "console.log(\"hola mundo!!!\");\nconsole.log(rowData);"
            },
            {
            "config_title": "(provisionGenericIdentifier - provisionAdministrationIdentifier - provision.administration.identifier)",
            "title": "FORM.TITLE.AT",
            "path": "provision.administration.identifier._current.at",
            "availableFields": [
                
            ],
            "alias": "Hidden column",
            "firstPath": "provision.administration.identifier._current.at",
            "expand": false,
            "isOgIdentifier": false,
            "widths": {
                "width": "100",
                "minWidth": "1",
                "maxWidth": "150",
                "hAlign": "text-center",
                "textWrap": true
            },
            "hiddenColumn": true
            },
            {
            "config_title": "(provisionGenericIdentifier - provisionAdministrationIdentifier - provision.administration.identifier)",
            "title": "FORM.TITLE.AT",
            "path": "provision.administration.identifier._current.at",
            "availableFields": [
                
            ],
            "alias": "Expandable column",
            "firstPath": "provision.administration.identifier._current.at",
            "expand": false,
            "isOgIdentifier": false,
            "widths": {
                "width": "100",
                "minWidth": "1",
                "maxWidth": "150",
                "hAlign": "text-center",
                "textWrap": true
            },
            "hiddenColumn": false,
            "expandable": true
            },
            {
                "config_title": "(provisionDevice - deviceProvisionedInfo - provision.device.location)",
                "title": "FORM.TITLE.CURRENT.VALUE",
                "path": "provision.device.location._current.value",
                "availableFields": [
                    "position.type",
                    "position.coordinates",
                    "country",
                    "region",
                    "province",
                    "town",
                    "postal",
                    "address",
                    "source",
                    "accuracy",
                    "zoom"
                ],
                "alias": "Prov. Location",
                "firstPath": "provision.device.location._current.value",
                "expand": false,
                "field": "region",
                "isOgIdentifier": false
            }
        ],
        "pagination": 10,
        "sharePrivateFilter": false,
        "noShareFilter": false,
        "noShareHeaderFilter": false,
        "noShareHeaderSorting": false,
        "hideSlowSearch": null,
        "heights": {
            "height": "50",
            "minHeight": "10",
            "maxHeight": "75",
            "vAlign": "align-end"
        },
        "rowFormatter": {
            "_formatterCode": "console.log(rowData);\nif (rowData.index%2==0) {\n\trowFormatter.style =\"background-color: lightgrey;\";\n}",
            "_formatterRef": null
        }
    }
}
```

## Specific config properties

* **pagination**: Number of elements to display in the table. Default 10.
* **sharePrivateFilter**: Whether to share the internal private filter with other users (boolean).
* **noShareFilter**: Whether to not share the filter with other users when dashboard is shared with them (boolean).
* **noShareHeaderFilter**: Whether to not share the header filter with other users when dashboard is shared with them (boolean).
* **noShareHeaderSorting**: Whether to not share the header sorting with other users when dashboard is shared with them (boolean).
* **hideSlowSearch**: Whether to hide the slow search columns (boolean).
* **hideRowActions**: Whether to hide the row actions (boolean).
* **heights**: The heights of the rows (object).
    * **height**: The height of the rows (string).
    * **minHeight**: The minimum height of the rows (string).
    * **maxHeight**: The maximum height of the rows (string).
    * **vAlign**: The vertical alignment of the rows (string).
* **rowFormatter**: The row formatter of the table (object).
    * **_formatterCode**: The code to format the rows (string).
    * **_formatterRef**: The reference to the row formatter if it is saved/shared (string).

### _formatterCode description

Code used to format the rows. Function code is wrapped in a function with a default returned value: 

```javascript
function (rowData, entityData) {
    var rowFormatter = { style: ''}; // always included in rowFormatter wrapper object
    // _formatterCode here your action here
    return rowFormatter; // always included in rowFormatter wrapper object
}
```

Where:
- `rowData`: The data of the entity in table (columns selected in devicesListConfig) format.
- `entityData`: The data of the entity in flattened format (only when opened in an entity context).
- `relatedEntities`: The data of the related entities in flattened format (only when opened in an entity context).


* **columns** array of column configuration objects:
    * **config_title**: The title for the configuration (used for code generation) (string).
    * **title**: The title of the column (string). Can be a translatable string.
    * **path**: The path to the field to display in the column (string).
    * **availableFields**: An array of available fields for the datastream if available (array of strings).
    * **field**: The field to display in the column if available in availableFields (string).
    * **alias**: The alias of the column (string).
    * **firstPath**: The first path to the field to display in the column (string).
    * **expand**: Whether the column is expandable (boolean).
    * **isOgIdentifier**: Whether the column is an OpenGate identifier (boolean).
    * **isDateTime**: Whether the column is a datetime (boolean).
    * **widths**: The widths of the column (object).
        * **width**: The width of the column (string).
        * **minWidth**: The minimum width of the column (string).
        * **maxWidth**: The maximum width of the column (string).
        * **hAlign**: The horizontal alignment of the column (string).
        * **textWrap**: Whether the column text wraps (boolean).
    * **hiddenColumn**: Whether the column is hidden.
    * **expandable**: Whether the column is expandable. A *show*/*hide* button appears instead of value. Click to show/hide value
    * **buttonColumn**: Whether the column is a button column. If true, the column will display a button. It requires _actionCode to be defined.
        * **_actionCode**: The code to execute when the button is clicked.
    * **_formatterCode**: The code to format the column value.
    * **_formatterRef**: The reference to the row formatter if it is saved/shared (string).

### Paths in columns

Each kind of widget manages the path selection in a different way. See the specific widget documentation for more information.

### Column _formatterCode description

Code used to format the column value. Function code is wrapped in a function with a default returned value: 

```javascript
function( value, rowData, entityData ) {
    var cellFormatter = { style: '', leftIcon:'', rightIcon:'', customValue:''}; // always included in cellFormatter wrapper object
    // _formatterCode here your action here
    return cellFormatter; // always included in cellFormatter wrapper object
}
```

Where:
- `rowData`: The data of the entity in table (columns selected in devicesListConfig) format.
- `entityData`: The data of the entity in flattened format (only when opened in an entity context).
- `relatedEntities`: The data of the related entities in flattened format (only when opened in an entity context).


### Column _actionCode description

Action to execute when the button is clicked. Function code is wrapped in a function: 

```javascript
async function (rowData,entityData,relatedEntities){
    // _actionCode here your action here
}
```

Where:
- `rowData`: The data of the entity in table (columns selected in devicesListConfig) format.
- `entityData`: The data of the entity in flattened format (only when opened in an entity context).
- `relatedEntities`: The data of the related entities in flattened format (only when opened in an entity context).

#### Available utils for _actionCode 

Utils listed in [utils.md](utils.md) are available and also you have the following utilities.

- **van** instance of the VanJS framework.
- **vanui** instance of the VanJS UI library.
- **vanOpenModal(vanObject[,title, onCloseCallback])** opens your van object in a modal window
- **openNewWindow(url)**: A window.open instance
- **openWizard(wizardId, wizardData, isEdit)**: Opens a wizard in a modal window.
- **refreshWidget([timeout])**: Refreshes the widget. If timeout is provided, it will refresh the widget after the specified time in milliseconds.
