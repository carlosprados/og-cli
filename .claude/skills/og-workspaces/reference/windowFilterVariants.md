### Window filter variants

Custom window filter are defined as follows:

``` json
"windowFilter": {
    "type": "custom",
    "period": "custom",
    "total": 1,
    "from": "2026-04-01T07:43:20.000Z",
    "to": "2026-04-24T22:00:00.000Z"
}
```

Below modes calculate dynamically the "type", "total", "from" and "to" parameters based on 'period'.

Today window filter ("from" is the begining of the day)
``` json
"windowFilter": {
    "type": "today",
    "period": "today",
    "total": 1,
    "from": "2026-05-22T00:00:00+02:00",
    "to": null
}
```

Yesterday window filter ("from" is the begining of the previous day, "to" is the end of the previous day)
``` json
"windowFilter": {
    "type": "yesterday",
    "period": "yesterday",
    "total": 1,
    "from": "2026-05-24T00:00:00+02:00",
    "to": "2026-05-25T00:00:00+02:00"
}
```

Last week window filter ("from" is the begining of the week)
``` json
"windowFilter": {
    "type": "weeks",
    "period": "last_week",
    "total": 1,
    "from": "2026-05-18T00:00:00+02:00",
    "to": null
}
```

Last 7 days window filter ("from" is the begining of the last 7 days)
``` json
"windowFilter": {
    "type": "days",
    "period": "last_7_days",
    "total": 7,
    "from": "2026-05-18T00:00:00+02:00",
    "to": null
}
```

Last 15 days window filter ("from" is the begining of the last 15 days)
``` json
"windowFilter": {
    "type": "days",
    "period": "last_15_days",
    "total": 15,
    "from": "2026-05-10T00:00:00+02:00",
    "to": null
}
```

Last 30 days window filter ("from" is the begining of the last 30 days)
``` json
"windowFilter": {
    "type": "days",
    "period": "last_30_days",
    "total": 30,
    "from": "2026-04-25T00:00:00+02:00",
    "to": null
}
```

Older week window filter ("to" is the end of the previous week)
``` json
"windowFilter": {
    "type": "weeks",
    "period": "older_week",
    "total": 1,
    "from": null,
    "to": "2026-05-18T00:00:00+02:00"
}
```

Older 7 days window filter ("to" is the end of the previous 7 days)
``` json
"windowFilter": {
    "type": "days",
    "period": "older_7_days",
    "total": 7,
    "from": null,
    "to": "2026-05-18T00:00:00+02:00"
}
```

Older 15 days window filter ("to" is the end of the previous 15 days)
``` json
"windowFilter": {
    "type": "days",
    "period": "older_15_days",
    "total": 15,
    "from": null,
    "to": "2026-05-10T00:00:00+02:00"
}
```

Older 30 days window filter ("to" is the end of the previous 30 days)
``` json
"windowFilter": {
    "type": "days",
    "period": "older_30_days",
    "total": 30,
    "from": null,
    "to": "2026-04-25T00:00:00+02:00"
}
```
