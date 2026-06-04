# Reference: `TimeseriesList`

The `TimeseriesList` widget allows displaying timeseries data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'datasets'
* `type`: Always must be 'TableTimeserie'

### Timeseries specific configuration

In `dataset` field you must define the timeseries to display:

```json
"dataset": {
    "identifier": "65a62af5fcf8cc749c287dbe",
    "organization": "chemacorp",
    "columnIdentifier": {
        "name": "EntityID",
        "path": "provision.administration.identifier._current.value"
    },
    "name": "estado_bateria"
},
```

Where `dataset` is an object with the following fields:

* `identifier`: Identifier of the timeseries.
* `name`: Name of the timeseries.
* `organization`: Organization associated with the timeseries.
* `columnIdentifier`: The name of the column and path in timeseries that identifies the entity to which the timeseries belongs. Column must be string type.

Also timeseries widget can have the next config fields

* `preferredOrganization`: tells wich organization the widget should use to find the timeseries definition:
    * `selected`: means the organization is the selected in "dataset.organization" field
    * `user`: means the organization is the selected in "organization" field in "User" resource in side bar (overrides "dataset.organization" if available)
    * `entity`: means the organization comes from the entity opened in Opengate (overrides "dataset.organization" if available)

* `predefinedSort` is the selected sort from the available sorts in timeseries to tell how to sort the timeseries.

If you want to use predefined sort:

```json
{
    "identifier": "EntityID_asc__bucketEnd_asc",
    "description": "Sort by EntityID(ASC) and bucketEnd(ASC)",
    "columns": [
        {
            "name": "EntityID",
            "direction": "ASC"
        },
        {
            "name": "bucketEnd",
            "direction": "ASC"
        }
    ]
}
```

### Paths in columns config and filter config

Columns path (._current.* not allowed) for timeseries and must match the column names provisioned in selected timeseries.

Example of column definition for a timeseries that has "EntityID" as column identifier:

```json
{
    "title": "EntityID",
    "path": "EntityID",
    "customPath": "provision.administration.identifier._current.value",
    "alias": "EntityID",
    "notFilterable": false,
    "filtrable": "YES",
    "schema": {
        "type": "string"
    },
    "type": "string",
    "isOgIdentifier": true
}
```

Some fields in json come from timeseries definition and others are added by the widget.

