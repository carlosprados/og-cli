# Reference: `AreasList`

The `AreasList` widget allows displaying areas data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'areas'
* `type`: Always must be 'AreasList'

### Paths in columns config

Columns path (._current.* not allowed) for AreasList:

| Field | Description |
|-------|-------------|
| identifier | Identifier of the area. |
| name | Name of the area. |
| description | Description of the area. |
| entities | Entities associated with the area. |
| geometry | Geometric definition or spatial shape of the area. |
| color | Color assigned to the area. |
| organization | Organization associated with the area. |

### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Fields for filter:

| Field | Description |
|-------|-------------|
| areas.identifier | Identifier of the area. |
| areas.name | Name of the area. |
| areas.description | Description of the area. |
| areas.entities | Entities associated with the area. |
| areas.geometry | Geometric definition or spatial shape of the area. |
| areas.color | Color assigned to the area. |
| areas.organization | Organization associated with the area. |

