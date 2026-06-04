# Reference: `FullDevicesList`

The `FullDevicesList` widget allows displaying entities data in a table format with several features like grouping, filtering, and pagination.

You can see in ListsCommons.md the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'entities
* `type`: Always must be 'FullDevicesList'

### Paths in columns config

Paths in columns must have a complete datastream path without a field, and with the _current context, for example: `datastream1.datastream2._current.value` or `datastream1.datastream2._current.at`.

All datastreams must exists in datamodels, otherwise the column will not be displayed or it will throw an error.

### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Supported fields are the datastreams in the datamodels with `_current` context or `_current.value.[field]`

