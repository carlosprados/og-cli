# Reference: `DatapointsList`

The `DatapointsList` widget allows displaying datapoints data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'datapoints'
* `type`: Always must be 'DatapointsList'

### Specific configuration in config

* `syncZoomFilter`: true/false (default false). If true, the zoom filter will be synchronized with the zoom filter of other widgets.

### Paths in columns config

Columns path (._current.* not allowed) for DatapointsList:

| Field | Description |
|-------|-------------|
| entityIdentifier | Identifier of the main entity involved in the record. |
| subEntityIdentifier | Identifier of a related or subordinate entity. |
| channel | Channel through which the data is processed. |
| organization | Organization associated with the entity. |
| datastreamId | Identifier of the datastream where the information belongs. |
| entityRelated | Indicates whether the record is related to another entity. |
| _current.value | Current value associated with the datastream. |
| _current.feedId | Identifier of the feed providing the current data. |
| _current.from | Source or origin of the current value. |
| _current.date | Date when the current value was recorded. |
| _current.at | Timestamp indicating the exact moment the value was captured. |


### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Fields for filter:

| Field | Description |
|-------|-------------|
| datapoints.entityIdentifier | Identifier of the main entity represented in the datapoint. |
| datapoints.subEntityIdentifier | Identifier of a related or subordinate entity within the datapoint. |
| datapoints.channel | Channel through which the datapoint was generated or transmitted. |
| datapoints.organization | Organization associated with the datapoint. |
| datapoints.datastreamId | Identifier of the datastream where the datapoint belongs. |
| datapoints.entityRelated | Indicates whether the datapoint is related to another entity. |
| datapoints._current.value | Current value stored in the datapoint. |
| datapoints._current.feedId | Identifier of the feed providing the current datapoint value. |
| datapoints._current.from | Source or origin of the current datapoint value. |
| datapoints._current.date | Date when the current datapoint value was recorded. |
| datapoints._current.at | Exact timestamp when the datapoint value was captured. |
