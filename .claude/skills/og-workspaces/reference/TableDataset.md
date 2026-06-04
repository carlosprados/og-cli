# Reference: `TableDataset`

The `TableDataset` widget allows displaying datasets data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'datasets'
* `type`: Always must be 'TableDataset'

### Datasets specific configuration

In `dataset` field you must define the timeseries to display:

```json
"dataset": {
    "identifier": "66eaab46beaeb30653d1343c",
    "organization": "corporation",
    "columnIdentifier": {
        "name": "EntityID",
        "path": "provision.administration.identifier._current.value"
    },
    "name": "corporation dataset"
},
```

Where `dataset` is an object with the following fields:

* `identifier`: Identifier of the dataset.
* `name`: Name of the dataset.
* `organization`: Organization associated with the dataset.
* `columnIdentifier`: The name of the column and path in dataset that identifies the entity to which the dataset belongs. Column must be string type.

Also dataset widget can have the next config fields

* `preferredOrganization`: tells wich organization the widget should use to find the dataset definition:
    * `selected`: means the organization is the selected in "dataset.organization" field
    * `user`: means the organization is the selected in "organization" field in "User" resource in side bar (overrides "dataset.organization" if available)
    * `entity`: means the organization comes from the entity opened in Opengate (overrides "dataset.organization" if available)

### Paths in columns config and filter config

Columns path (._current.* not allowed) for dataset and must match the column names provisioned in selected dataset.

Example of column definition for a dataset that has "EntityID" as column identifier:

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

Some fields in json come from dataset definition and others are added by the widget.

