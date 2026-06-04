# Reference: `clock`

Displays a live real-time clock inside a dashboard tile.

You have to consider this especial configuration in config:

* `type`: Always must be 'clock'

## JSON Schema Minimum Configuration
Refer to [Common Widget Fields](./commonFields.md) for grid and standard widget layout wrapping properties. Specific properties for `Clock` in `config` include:

```json
{
    "type": "clock",
    "wid": "widget-id",
    "config": {
        "title": "",
        "timePattern": "HH:mm:ss",
        "datePattern": "DD-MM-YYYY",
    }
}
```

* `timePattern and datePattern use the same format as moment.js: https://momentjs.com/docs/#/displaying/format/
