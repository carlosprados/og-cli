# Advanced Rules JavaScript — official platform guide

> Harvested live from `GET /north/v80/rules/doc/javascriptFunctions` (+ /private/javascriptFunctions) on 2026-06-04.
> This is the AUTHORITATIVE reference for the JS inside ADVANCED rules (og rules deploy).

# Advanced Rules Guide

In advanced rules, you can write conditions and actions in javascript language.

In this javascript code, it is possible to call some defined functions to execute the same actions that you can do in easy mode. On the other hand, we also have some help functions. We will explain them below.


The main function will be receive flattened entity representation adding _\_received_ and _\_previous_ value in each datastream. _\_received_ field is a simple object in provision datastream and an array object y another case.

We will take like entity example this one:

```
entity = {
  "provision.administration.channel": {
    "_value": {
      "_current": {
        "value": "battery_channel",
        "date": "2017-12-01T08:52:37.563Z",
        "at": "2017-12-01T08:52:37.563Z"
      },
      "_previous": {
        "value": "battery_channel",
        "date": "2017-12-01T08:52:37.563Z",
        "at": "2017-12-01T08:52:37.563Z"
      }
    }
  },
  "provision.administration.identifier": {
    "_value": {
      "_current": {
        "value": "device_battery_id",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.57Z",
        "at": "2017-12-01T08:52:37.57Z"
      },
      "_previous": {
        "value": "battery_organization",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.566Z",
        "at": "2017-12-01T08:52:37.566Z"
      }
    }
  },
  "provision.administration.organization": {
    "_value": {
      "_current": {
        "value": "battery_organization",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.566Z",
        "at": "2017-12-01T08:52:37.566Z"
      },
      "_previous": {
        "value": "battery_organization",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.566Z",
        "at": "2017-12-01T08:52:37.566Z"
      }
    }
  },
  "provision.administration.plan": {
    "_value": {
      "_current": {
        "value": "FLOW_RATE_100",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.565Z",
        "at": "2017-12-01T08:52:37.565Z"
      },
      "_previous": {
        "value": "FLOW_RATE_100",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.565Z",
        "at": "2017-12-01T08:52:37.565Z"
      }
    }
  },
  "provision.administration.serviceGroup": {
    "_value": {
      "_current": {
        "value": "emptyServiceGroup",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.561Z",
        "at": "2017-12-01T08:52:37.561Z"
      },
      "_previous": {
        "value": "emptyServiceGroup",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.561Z",
        "at": "2017-12-01T08:52:37.561Z"
      }
    }
  },
  "provision.device.communicationModules[].identifier": [
    {
      "_index": {
        "path": "provision.device.communicationModules[].identifier",
        "value": {
          "_current": {
            "value": "commsMod_battery_id",
            "provType": "MONITORING",
            "date": "2017-12-01T08:52:37.577Z",
            "at": "2017-12-01T08:52:37.577Z"
          }
        }
      },
      "_value": {
        "_current": {
          "value": "commsMod_battery_id",
          "provType": "MONITORING",
          "date": "2017-12-01T08:52:37.577Z",
          "at": "2017-12-01T08:52:37.577Z"
        },
        "_previous": {
          "value": "commsMod_battery_id",
          "provType": "MONITORING",
          "date": "2017-12-01T08:52:37.577Z",
          "at": "2017-12-01T08:52:37.577Z"
        }
      }
    }
  ],
  "provision.device.communicationModules[].subscription.address": [
    {
      "_index": {
        "path": "provision.device.communicationModules[].identifier",
        "value": {
          "_current": {
            "value": "commsMod_battery_id",
            "provType": "MONITORING",
            "date": "2017-12-01T08:52:37.577Z",
            "at": "2017-12-01T08:52:37.577Z"
          }
        }
      },
      "_value": {
        "_current": {
          "value": {
            "type": "IPV4",
            "value": "99.1.1.71",
            "apn": "myapnprov.es"
          },
          "provType": "REFERENCE",
          "date": "2017-12-01T08:52:37.624Z",
          "at": "2017-12-01T08:52:37.624Z"
        },
        "_previous": {
          "value": {
            "type": "IPV4",
            "value": "99.1.1.71",
            "apn": "myapnprov.es"
          },
          "provType": "REFERENCE",
          "date": "2017-12-01T08:52:37.624Z",
          "at": "2017-12-01T08:52:37.624Z"
        }
      }
    }
  ],
  "provision.device.communicationModules[].subscription.identifier": [
    {
      "_index": {
        "path": "provision.device.communicationModules[].identifier",
        "value": {
          "_current": {
            "value": "commsMod_battery_id",
            "provType": "MONITORING",
            "date": "2017-12-01T08:52:37.577Z",
            "at": "2017-12-01T08:52:37.577Z"
          }
        }
      },
      "_value": {
        "_current": {
          "value": "subscription_battery_id",
          "provType": "MONITORING",
          "date": "2017-12-01T08:52:37.626Z",
          "at": "2017-12-01T08:52:37.626Z"
        },
        "_previous": {
          "value": "subscription_battery_id",
          "provType": "MONITORING",
          "date": "2017-12-01T08:52:37.626Z",
          "at": "2017-12-01T08:52:37.626Z"
        }
      }
    }
  ],
  "provision.device.identifier": {
    "_value": {
      "_current": {
        "value": "device_battery_id",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.636Z",
        "at": "2017-12-01T08:52:37.636Z"
      },
      "_previous": {
        "value": "device_battery_id",
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.636Z",
        "at": "2017-12-01T08:52:37.636Z"
      }
    }
  },
  "provision.device.location": {
    "_value": {
      "_current": {
        "value": {
          "position": {
            "type": "Point",
            "coordinates": [
              -3.7028,
              40.41675
            ]
          },
          "postal": "28013"
        },
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z"
      },
      "_previous": {
        "value": {
          "position": {
            "type": "Point",
            "coordinates": [
              -3.7028,
              40.41675
            ]
          },
          "postal": "28013"
        },
        "provType": "MONITORING",
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z"
      }
    }
  },
  "resourceType": {
    "_value": {
      "_current": {
        "value": "entity.device",
        "provType": "IDENTIFIER",
        "date": "2017-12-01T08:52:37.643Z",
        "at": "2017-12-01T08:52:37.643Z"
      },
      "_previous": {
        "value": "entity.device",
        "provType": "IDENTIFIER",
        "date": "2017-12-01T08:52:37.643Z",
        "at": "2017-12-01T08:52:37.643Z"
      }
    }
  },
  "device.communicationModules[].identifier": [
    {
      "_index": {
        "path": "device.communicationModules[].identifier",
        "value": {
          "_current": {
            "value": "commsMod_battery_id",
            "date": "2017-12-01T08:52:37.577Z",
            "at": "2017-12-01T08:52:37.577Z"
          }
        }
      },
      "_value": {
        "_current": {
          "value": "commsMod_battery_id",
          "date": "2017-12-01T08:52:37.577Z",
          "at": "2017-12-01T08:52:37.577Z"
        },
        "_previous": {
          "value": "commsMod_battery_id",
          "date": "2017-12-01T08:52:37.577Z",
          "at": "2017-12-01T08:52:37.577Z"
        }
      }
    }
  ],
  "device.communicationModules[].subscription.address": [
    {
      "_index": {
        "path": "device.communicationModules[].identifier",
        "value": {
          "_current": {
            "value": "commsMod_battery_id",
            "date": "2017-12-01T08:52:37.577Z",
            "at": "2017-12-01T08:52:37.577Z"
          }
        }
      },
      "_value": {
        "_current": {
          "value": {
            "type": "IPV4",
            "value": "99.1.1.71",
            "apn": "myapn.es"
          },
          "date": "2017-12-01T08:52:37.624Z",
          "at": "2017-12-01T08:52:37.624Z",
          "source": "DEVICE_OPENGATE_HTTP",
          "sourceInfo": "IoT Data Message Received"
        },
        "_previous": {
          "value": {
            "type": "IPV4",
            "value": "99.1.1.71",
            "apn": "myapn.es"
          },
          "date": "2017-12-01T08:52:37.624Z",
          "at": "2017-12-01T08:52:37.624Z",
          "source": "DEVICE_OPENGATE_HTTP",
          "sourceInfo": "IoT Data Message Received"
        }
      }
    }
  ],
  "device.communicationModules[].subscription.identifier": [
    {
      "_index": {
        "path": "device.communicationModules[].identifier",
        "value": {
          "_current": {
            "value": "commsMod_battery_id",
            "date": "2017-12-01T08:52:37.577Z",
            "at": "2017-12-01T08:52:37.577Z"
          }
        }
      },
      "_value": {
        "_current": {
          "value": "subscription_battery_id",
          "date": "2017-12-01T08:52:37.626Z",
          "at": "2017-12-01T08:52:37.626Z",
          "source": "DEVICE_OPENGATE_HTTP",
          "sourceInfo": "IoT Data Message Received"
        },
        "_previous": {
          "value": "subscription_battery_id",
          "date": "2017-12-01T08:52:37.626Z",
          "at": "2017-12-01T08:52:37.626Z",
          "source": "DEVICE_OPENGATE_HTTP",
          "sourceInfo": "IoT Data Message Received"
        }
      }
    }
  ],
  "device.identifier": {
    "_value": {
      "_received": [{
        "value": "device_battery_id",
        "date": "2017-12-01T08:52:37.636Z",
        "at": "2017-12-01T08:52:37.636Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }],
      "_current": {
        "value": "device_battery_id",
        "date": "2017-12-01T08:52:37.636Z",
        "at": "2017-12-01T08:52:37.636Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      },
      "_previous": {
        "value": "device_battery_id",
        "date": "2017-12-01T08:52:37.636Z",
        "at": "2017-12-01T08:52:37.636Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }
    }
  },
  "device.temperature.value": {
    "_value": {
      "_received": [{
        "value": 25.3,
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }],
      "_current": {
        "value": 25.3,
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      },
      "_previous": {
        "value": 23.3,
        "date": "2017-12-01T05:52:37.64Z",
        "at": "2017-12-01T05:52:37.64Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }
    }
  }
}
  
```


## Javascript functions catalog

### Getting dates

#### toDate

`toDate(localDateTime)` Obtain date type from string date representation with format YYYY-MM-DDThh:mm:ssTZD.

The function take 1 parameters:

1. Date string representation

The example get date type from _2021-06-30T10:10:23.256Z_ date:

```
var date = getDate('2021-06-30T10:10:23.256Z');
```

Result javascript date object with selected string date representation.



#### getDailyResetDate

`getDailyResetDate` obtains date to reset daily counters.

This function returns date with current date, but with 00:00:00 hours.


#### getDailyResetDateWithZuluHour

`getDailyResetDateWithZuluHour` obtains date to reset daily counters with defined hour in gmt+0.

The function take 1 parameters:

1. Hour in format HH:mm:ss

```
var date = getDailyResetDateWithZuluHour('02:00:00');
```

This function returns date with current date, but with 2:00:00 hours. For example, if current day is _2021-06-30_, the returned date in previously called function is _21-06-30T02:00:00Z_


#### getMonthlyResetDate

`getMonthlyResetDate()` obtains date to reset monthly counters.

This function returns date with first day of current month and 00:00:00 hours.

#### getMonthlyResetDateWithZuluHour

`getMonthlyResetDateWithZuluHour` obtains date to reset monthly counters with defined hour in gmt+0.

The function take 1 parameters:

1. Hour in format HH:mm:ss

```
var date = getMonthlyResetDateWithZuluHour('02:00:00');
```

This function returns date with first day of current month, but with 2:00:00 hours. For example, if current day is _2021-06-30_, the returned date in previously called function is _21-06-01T02:00:00Z_

#### getMonthlyResetDateWithZuluHourAndDayOfMonth

`getMonthlyResetDateWithZuluHourAndDayOfMonth` obtains date to reset monthly counters with defined hour in gmt+0.

The function take 2 parameters:

1. Hour in format HH:mm:ss
2. Day of Month

```
var date = getMonthlyResetDateWithZuluHourAndDayOfMonth('02:00:00', 21);
```

This function returns date of the day 21 of current month, but with 2:00:00 hours, when day of the month is null, then return first day of the month. For example, the returned date in previously called function is _23-02-21T02:00:00Z_


### Getting and formatting datastreams and values

#### getVariableValue

`getVariableValue(variable)` obtains empty string value if is undefined value

This function take 1 parameter:

1. Value of variable

This function returning empty string

```
var myVar = undefined;
var finalVar = getVariable(myVar);
```

Result is '';


This function returning same value

```
var myVar = 2;
var finalVar = getVariable(myVar);
```

Result is 2;


#### getDatastreamFromEntity

`getDatastreamFromEntity(datastreamId)` returns completed datastream of received entity.

The function take 1 parameter: 
1. Datastream identifier requested.

The example get _device.temperature_ datastream.

```
getDatastreamFromEntity(device.temperature);
```

Result:

```
{
  "_received": [
    {
      "value": 25.3,
      "date": "2017-12-01T08:52:37.64Z",
      "at": "2017-12-01T08:52:37.64Z",
      "source": "DEVICE_OPENGATE_HTTP",
      "sourceInfo": "IoT Data Message Received"
    }
  ],
  "_current": {
    "value": 25.3,
    "date": "2017-12-01T08:52:37.64Z",
    "at": "2017-12-01T08:52:37.64Z",
    "source": "DEVICE_OPENGATE_HTTP",
    "sourceInfo": "IoT Data Message Received"
  },
  "_previous": {
    "value": 23.3,
    "date": "2017-12-01T05:52:37.64Z",
    "at": "2017-12-01T05:52:37.64Z",
    "source": "DEVICE_OPENGATE_HTTP",
    "sourceInfo": "IoT Data Message Received"
  }
}
```

#### getDatastreamValueFromEntity
`getDatastreamValueFromEntity(datastreamObject)` returns `_value._current._value` from datastream object

The function take 1 parameter:
1. Datastream object

Return value object from datastream object.


#### getCommsDatastreamFromEntity

`getCommsDatastreamFromEntity(id, commsId)` returns completed datastream in selected communication module of received entity.

The function take 2 parameters:
1. Datastream identifier requested 
2. Communication module identifier

The example get _device.communicationModules[].subscription.address_ datastream in _commsMod\_battery\_id_ communications module.

```
datastream = getDatastreamFromEntity('device.communicationModules[].subscription.address', 'commsMod_battery_id');
```

Result:

```
{
  "_current": {
    "value": {
      "type": "IPV4",
      "value": "99.1.1.71",
      "apn": "myapn.es"
    },
    "date": "2017-12-01T08:52:37.624Z",
    "at": "2017-12-01T08:52:37.624Z",
    "source": "DEVICE_OPENGATE_HTTP",
    "sourceInfo": "IoT Data Message Received"
  },
  "_previous": {
    "value": {
      "type": "IPV4",
      "value": "99.1.1.71",
      "apn": "myapn.es"
    },
    "date": "2017-12-01T08:52:37.624Z",
    "at": "2017-12-01T08:52:37.624Z",
    "source": "DEVICE_OPENGATE_HTTP",
    "sourceInfo": "IoT Data Message Received"
  }
}
```

#### getCounterValue

`getCounterValue(datastreamValue, incValue, resetDate)` obtains incValue if datastreamValue date is before than resetDate or increments received value in datastream to incValue if the date is after.

The function take 3 parameters:
1. Value of datastream 
2. Value to increment
3. Date of reset

This example get incremented daily counter to _myCounterDatastream_ datastream:

```
var datastream = {'value':3, 'date': toDate('2021-06-30T11:20:21.352Z')}
var resetDate = toDate('2021-06-30T00:00:0.000Z');
var counter = getCounterValue(datastream, 1, resetDate); 
````

Result: 4


This example get reseted daily counter to _myCounterDatastream_ datastream:

```
var datastream = {'value':3, 'date': toDate('2021-06-29T11:20:21.352Z')}
var resetDate = toDate('2021-06-30T00:00:0.000Z');
var counter = getCounterValue(datastream, 1, resetDate); 
````

Result: 1


### Executing actions

#### openAlarm

> **Deprecated:** Use `alarm` object functions instead.

`openAlarm(subEntityIdentifier, alarmName, ruleName, severity, priority, alarmDescription)` execute an opening of selected rule on the entity.

The function take 6 parameters: 
1. You can open an alarm on device, subscription or subscriber identifier. With _subEntityIdentifier_ you define selected identifier. If is undefined it will open on _provision.administration.identifier_ identifier.
2. Name that you want to see in opened alarm
3. Name of rule that produce opening of alarm.
4. Severity of alarm. Values can be INFORMATIVE, URGENT or CRITICAL.
5. Priority of alarm. Values can be LOW, MEDIUM or HIGH.
6. Alarm description.

This example open alarm _apnMismatch_ to subscription.

```
openAlarm('subscription_battery_id', 'apnMismatch' , 'apnMismatchRule', 'URGENT', 'MEDIUM', 'APN mismatch with provisioned value')
```

And this example open alarm _highTemperature_ to device.

```
openAlarm(undefiend, 'highTemperature' , 'highTemperatureRule', 'URGENT', 'MEDIUM', 'Device temperature is high')
```

Function returns void.

#### closeAlarmByRuleName

> **Deprecated:** Use `alarm` object functions instead.
 
`closeAlarmByRuleName(entityIdDatastream, ruleName)` execute a clousure of selected rule on the entity.

The function take 2 parameters:
1. You can close an alarm on device, subscription or subscriber identifier. With _entityIdDatastream_ you define selected identifier. If is undefined it will close on _provision.administration.identifier_ identifier.
2. Rule name that open alarm.

The example close alarm generated by _highTemperatureRule_ to device.

```
closeAlarm(undefined, 'highTemperatureRule')
```

Function returns void.

#### closeAlarmByAlarmName

> **Deprecated:** Use `alarm` object functions instead.

`closeAlarmByAlarmName(entityIdDatastream, alarmName)` execute a clousure of selected opened alarm on the entity.

The function take 2 parameters:
1. You can close an alarm on device, subscription or subscriber identifier. With _entityIdDatastream_ you define selected identifier. If is undefined it will close on _provision.administration.identifier_ identifier.
2. Opened alarm name.

The example close opened alarm _apnMismatch_ to device.

```
closeAlarm(undefined, 'highTemperatureRule')
```

Function returns void.

#### addEmailNotification

> **Deprecated:** Use `notification` object functions instead.

`addEmailNotification(recipients, notificationName, notificationBody, ruleName, mailParameters)` send email notification to defined recipients.

The function take 5 parameters:

1. Array of recipients to send email. This param has same format as defined in easy mode.
2. Name of notification that will be received in subject field.
3. Mustache template with email's body
4. Name of rule that launch email request.
5. Json object with key-value pairs that will be replaces in mustache evaluation.

The example send email to _example@recipient.es_ with _highTemperature_ notification.

```
parameters = {};
parameters['deviceIdentifier'] = getDatastreamFromEntity('device.identifier')._current.value;
parameters['deviceTemperature'] = getDatastreamFromEntity('device.temperature')._current.value;
addEmailNotification(['example@recipient.es'], 'highTemperature', 'Received high temperature from {{deviceIdentifier}} with value {{devideTemerature}}', 'highTemperatureRule', parameters);
```

Function returns void.


#### addTrapNotification

> **Deprecated:** Use `notification` object functions instead.

`addTrapNotification(recipients, variables, notificationName, trapOID, enterpriseOID, ruleName)` send trap notification to defined recipients.

The function take 6 parameters:

1. Array of recipients (<ip>:<port) to send trap. This param has same format as defined in easy mode.
2. Map of variables. Each pair defines OID variable and sent value for this OID.
3. Name of notification
4. OID of trap
5. OID of enterprise
6. Name of rule

The example send trap to _25.35.98.5:8585_ with _highTemperature_ notification.

```
var recipients = ['25.35.98.5:8585'];
var trapVariables = {
  '1.1.1': entity['provision.administration.organization']._value._current.value,
  '1.1.2': entity['provision.administration.channel']._value._current.value,
  '1.1.3': entity['provision.administration.identifier']._value._current.value,
  '1.1.4': entity['device.temperature']._value._current.value,
  '1.1.4': entity['device.temperature']._value._previous.value,
  '1.1.5': Date.now()
};         

addTrapNotification(recipients, trapVariables, 'highTemperature', '1.100.1', '1.2.7.3.1.2.25841', 'highTemperatureRule');
```

Function returns void.


#### sendHttp

> **Deprecated:** Use `notification` object functions instead.

`sendHttp(httpJson)` send http notification to defined recipients.

The function take 1 parameter:

1. Same json that easy mode.

The example send http request to _http://myService/request_ with _highTemperature_ notification.

```
httpJson = {
  'url': 'http://myService/request'
  'method' : 'POST',
  'headers': {
    'Content-type': 'application/json'
  },
  'queryParams' : {
    'deviceId' : entity['provision.device.identifier']._value._current.value;
  },
  'body' : 'New alert received by high temperature received. Current value: ' + entity['device.temperature']._value._current.value
};
sendHttp(httpJson);
```

Function returns void.


#### executeOperation

> **Deprecated:** Use `operation` object functions instead.

`executeOperation(subEntityIdentifier, operationType, operationTimeout, jobUser, retries, ackTimeout, retriesDelay, stopValue, stopMode, parameters, callback)` execute selected operation to received identifier.

The function take 10 parameters:

1. You can execute operation on device, subscription or subscriber identifier. With _subEntityIdentifier_ you define selected identifier. If is undefined it will open on _provision.administration.identifier_ identifier.
2. Operation Identifier. You can get this values getting operation catalog. **Mandatory field**. The rule will be created, but the operation will not.
3. Operation timeout in milliseconds. **Default**: 60000 milliseconds
4. User apiKey that launch this operation. **Mandatory field**. The rule will be created, but the operation will not.
5. Operation retries number. **Default**: 0. 
6. ACK timeout in milliseconds. **Default**: null. 
7. Delay in seconds between retries. **Default**: null. 
8. Stop value, this value depends of stop mode selected. **Default**: Operation timeout + 5000. 
9. Stop mode. Possible values are. **Default**: delayed
  - date: If this mode is selected, stop value is a date in YYYY-MM-DDThh:mm:ssTZD format.
  - delayed: If this mode is selected, stop value is a time defined in milliseconds.
10. parameters. This value is a json with accepted value in selected operation. You can see parameters getting operation catalog.
11. callback. URI where the result of the operation execution is received.

**NOTE**: The Job of the operation will be created with an active status by default.

The example execute _REFRESH\_PRESENCE_ operation to received device.

```
apiKey = getVariableValue(parameters['apiKey']);
parameters = {
  'active': false
};
executeOperation(entity['provision.administration.identifier]._value._current.value, 'REFRESH_PRESENCE', 20000, apiKey, 0, null, null, 25000, 'delayed', parameters, callback);
```

This function returns void.

#### cancelDelay

> **Deprecated:** Use `utils` object functions instead.

`cancelDelay(ruleName)` Cancel active delayed action produces by another rule activation.

The function take 1 parameters:

1. Name of rule that produce delayed actions.

The example cancel delay of 'highTemperatureRule' rule.

```
cancelDelay('highTemperatureRule');
```

This function returns void.

#### encryptString

> **Deprecated:** Use `utils` object functions instead.

`encryptString(originalValue, datastreamId, organization)` Encrypt an original string with the configuration established by the datastream of the organization

The function takes 3 parameters:

1. The original value to encrypt.
2. The datastream of provision type, of the datamodel that define this value. This datastream has the configuration of encryption.
3. The organization name to wich belongs to the datamodel. 

The example:

```
encryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```

This will return "value sent encrypted".

#### decryptString

> **Deprecated:** Use `utils` object functions instead.

`decryptString(encryptedValue, datastreamId, organization)` Decrypt an encrypted string with the configuration established by the datastream of the organization

The function takes 3 parameters:

1. The encrypted value to decrypt.
2. The datastream of provision type, of the datamodel that define this value. This datastream has the configuration of encryption.
3. The organization name to wich belongs to the datamodel. 

The example:

```
decryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```

This will return "value sent decrypted".

### Logging

The `logger` object is the main object for logging functions.


#### logger.trace
`logger.trace(msg)` Creates TRACE level logging messages. It concatenates msg parameters to compound message to be logged.

The function takes multiple parameters:

1. It takes a list of elements to be concatenated to generate the string message to be printed.

Example of use:
```javascript
logger.trace('Operation sent to', host_var, ' and port '. port_var, '. Operation content: ', operation_var);
```
Function returns void.


#### logger.debug

`logger.debug(msg)` Creates DEBUG level logging messages. It concatenates msg parameters to compound message to be logged.

The function takes multiple parameters:

1. It takes a list of elements to be concatenated to generate the string message to be printed.

Example of use:
```javascript
logger.debug('Operation sent to', host_var, ' and port '. port_var, '. Operation content: ', operation_var);
```
Function returns void.

#### logger.info

`logger.info(msg)` Creates INFO level logging messages. It concatenates msg parameters to compound message to be logged.

The function takes multiple parameters:

1. It takes a list of elements to be concatenated to generate the string message to be printed.

Example of use:
```javascript
logger.info('Operation sent to', host_var, ' and port '. port_var, '. Operation content: ', operation_var);
```
Function returns void.

#### logger.warn

`logger.warn(msg)` Creates WARN level logging messages. It concatenates msg parameters to compound message to be logged.

The function takes multiple parameters:

1. It takes a list of elements to be concatenated to generate the string message to be printed.

Example of use:
```javascript
logger.warn('Operation sent to', host_var, ' and port '. port_var, '. Operation content: ', operation_var);
```
Function returns void.

#### logger.error

`logger.error(msg)` Creates ERROR level logging messages. It concatenates msg parameters to compound message to be logged.

The function takes multiple parameters:

1. It takes a list of elements to be concatenated to generate the string message to be printed.

Example of use:
```javascript
logger.error('Error connecting to host ', host_var, ' and port '. port_var, '. Stop processing');
```
Function returns void.


### Alarm

The `alarm` object is the main object for managing alarms.

#### alarm.open

`alarm.open(alarmConfig)` Open alarm

This function takes as parameter an object with the following fields:

| Property            | Type      | Default       | Mandatory | Description                                                                                                                                                                                                          |
|---------------------|-----------|---------------|-----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| subEntityIdentifier | String    | entityId      | No        | You can open an alarm on device, subscription or subscriber identifier. With _subEntityIdentifier_ you define selected identifier. If is undefined it will open on _provision.administration.identifier_ identifier  |
| alarmName           | String    |               | No        | Name that you want to see in opened alarm                                                                                                                                                                            |
| ruleName            | String    |               | No        | Name of rule that produce opening of alarm                                                                                                                                                                           |
| severity            | String    | 'INFORMATIVE' | No        | Severity of alarm. Values can be INFORMATIVE, URGENT or CRITICAL                                                                                                                                                     |
| priority            | String    | 'LOW'         | No        | Priority of alarm. Values can be LOW, MEDIUM or HIGH                                                                                                                                                                 |
| description         | String    |               | No        | Alarm description                                                                                                                                                                                                    |

This example open alarm _apnMismatch_ to subscription.

```
alarmConfig = {
    subEntityIdentifier: "id",
    alarmName: "apnMismatch",
    ruleName: "apnMismatchRule",
    severity: "URGENT",
    priority: "MEDIUM",
    description: "APN mismatch with provisioned value"
}
  
alarm.open(alarmConfig);
```

And this example open alarm _highTemperature_ to device.

```
alarmConfigDevice = {
    alarmName: "highTemperature",
    ruleName: "highTemperatureRule",
    severity: "URGENT",
    priority: "MEDIUM",
    description: "Device temperature is high"
}

alarm.open(alarmConfigDevice);
```

Function returns void.

#### alarm.closeByRuleName
`alarm.closeByRuleName(closeByRuleConfig)` Close alarm

This function takes as parameter an object with the following fields:

| Property           | Type   | Default | Mandatory | Description                                                                                                                                                                                                           |
|--------------------|--------|---------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ruleName           | String |         | Yes       | Rule name that open alarm.                                                                                                                                                                                            |
| entityIdDatastream | String |         | No        | You can close an alarm on device, subscription or subscriber identifier. With _entityIdDatastream_ you define selected identifier. If is undefined it will close on _provision.administration.identifier_ identifier. | 

The example close alarm generated by _highTemperatureRule_ to device.

```
closeByRuleConfig = {
    ruleName: "highTemperatureRule"
}

alarm.closeByRuleName(closeByRuleConfig);
```

Function returns void.

#### alarm.closeByAlarmName
`alarm.closeByRuleName(closeByNameConfig)` Close alarm

This function takes as parameter an object with the following fields:

| Property           | Type   | Default | Mandatory | Description                                                                                                                                                                                                           |
|--------------------|--------|---------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| alarmName          | String |         | Yes       | Opened alarm name.                                                                                                                                                                                                    |
| entityIdDatastream | String |         | No        | You can close an alarm on device, subscription or subscriber identifier. With _entityIdDatastream_ you define selected identifier. If is undefined it will close on _provision.administration.identifier_ identifier. | 

The example close opened alarm _apnMismatch_ to device.

```
closeByNameConfig = {
    alarmName: "highTemperatureRule"
}

alarm.closeByAlarmName(closeByNameConfig);
```

Function returns void.

### Notification

The `notification` object is the main object for managing notifications.

#### notification.addEmailNotification

`notification.addEmailNotification(emailConfig)` send email notification to defined recipients.

This function takes as parameter an object with the following fields:

| Property         | Type   | Default | Mandatory | Description                                                                           |
|------------------|--------|---------|-----------|---------------------------------------------------------------------------------------|
| jsRecipients     | List   |         | Yes       | Array of recipients to send email. This param has same format as defined in easy mode |
| notificationName | String |         | Yes       | Name of notification that will be received in subject field                           |          
| notificationBody | String |         | Yes       | Mustache template with email's body                                                   |            
| ruleName         | String |         | Yes       | Name of rule that launch email request                                                |        
| mailParameters   | Object |         | No        | Json object with key-value pairs that will be replaces in mustache evaluation         |  

The example send email to _example@recipient.es_ with _highTemperature_ notification.

```
emailNotifConfig = {
    jsRecipients: ['example@recipient.es'],
    notificationName: 'highTemperature',
    notificationBody: 'Received high temperature from {{deviceIdentifier}} with value {{devideTemerature}}',
    ruleName: 'highTemperatureRule',
    mailParameters: {
        deviceIdentifier: getDatastreamFromEntity('device.identifier')._current.value,
        deviceTemperature: getDatastreamFromEntity('device.temperature')._current.value
    }
}
    
notification.addEmailNotification(emailNotifConfig);
```

Function returns void.

#### notification.addTrapNotification

`notification.addTrapNotification(trapConfig)` send email notification to defined recipients.

This function takes as parameter an object with the following fields:

| Property         | Type   | Default | Mandatory | Description                                                                                       |
|------------------|--------|---------|-----------|---------------------------------------------------------------------------------------------------|
| jsRecipients     | List   |         | Yes       | Array of recipients (<ip>:<port) to send trap. This param has same format as defined in easy mode |
| jsVariables      | Object |         | Yes       | Map of variables. Each pair defines OID variable and sent value for this OID                      |
| notificationName | String |         | Yes       | Name of notification                                                                              |          
| trapOID          | String |         | Yes       | OID of trap                                                                                       |              
| enterpriseOID    | String |         | Yes       | OID of enterprise                                                                                 |  
| ruleName         | String |         | Yes       | Name of rule                                                                                      |      

The example send trap to _25.35.98.5:8585_ with _highTemperature_ notification.

```
trapNotifConfig = {
    jsRecipients: ['25.35.98.5:8585'],
    jsVariables: {
      '1.1.1': entity['provision.administration.organization']._value._current.value,
      '1.1.2': entity['provision.administration.channel']._value._current.value,
      '1.1.3': entity['provision.administration.identifier']._value._current.value,
      '1.1.4': entity['device.temperature']._value._current.value,
      '1.1.4': entity['device.temperature']._value._previous.value,
      '1.1.5': Date.now()
    },
    notificationName: 'highTemperature',
    trapOID: '1.100.1',
    enterpriseOID: '1.2.7.3.1.2.25841',
    ruleName: 'highTemperatureRule'
}

notification.addTrapNotification(trapNotifConfig);
```

Function returns void.

#### notification.sendHttp

`notification.sendHttp(httpConfig)` send email notification to defined recipients.

This function takes as parameter an object with the following fields:

| Property    | Type   | Default | Mandatory | Description                                   |
|-------------|--------|---------|-----------|-----------------------------------------------|
| url         | String |         | Yes       | URL to which the HTTP request will be sent    |
| method      | String |         | Yes       | HTTP method to be used for the request        |          
| headers     | Object |         | No        | Key-value pairs representing HTTP headers     |            
| queryParams | Object |         | No        | Key-value pairs representing query parameters |        
| body        | String |         | No        | Body of the HTTP request                      |            

The example send http request to _http://myService/request_ with _highTemperature_ notification.

```
httpJson = {
  'url': 'http://myService/request'
  'method' : 'POST',
  'headers': {
    'Content-type': 'application/json'
  },
  'queryParams' : {
    'deviceId' : entity['provision.device.identifier']._value._current.value;
  },
  'body' : 'New alert received by high temperature received. Current value: ' + entity['device.temperature']._value._current.value
};

notification.sendHttp(httpJson);
```

Function returns void.

### Operation

The `operation` object is the main object for execute operations.

#### operation.execute
`operation.execute(operationConfig)` execute selected operation to received identifier.

This function takes as parameter an object with the following fields:

| Property            | Type   | Default                 | Mandatory | Description                                                                                                                                                                                                                           |
|---------------------|--------|-------------------------|-----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| subEntityIdentifier | String |                         | No        | You can execute operation on device, subscription or subscriber identifier. With _subEntityIdentifier_ you define selected identifier. **Default**: If is undefined it will open on _provision.administration.identifier_ identifier. |
| operationType       | String |                         | Yes       | Operation Identifier. You can get this values getting operation catalog. **Mandatory field**. The rule will be created, but the operation will not.                                                                                   |          
| operationTimeout    | Number | 60000                   | No        | Operation timeout in milliseconds.                                                                                                                                                                                                    |            
| jobUser             | String |                         | Yes       | User apiKey that launch this operation. **Mandatory field**. The rule will be created, but the operation will not.                                                                                                                    |        
| retries             | Number | 0                       | No        | Operation retries number.                                                                                                                                                                                                             |   
| ackTimeout          | Number | null                    | No        | ACK timeout in milliseconds.                                                                                                                                                                                                          |
| retriesDelay        | Number | 0                       | No        | Delay in seconds between retries.                                                                                                                                                                                                     |
| stopValue           | String | operationTimeout + 5000 | No        | Stop value, this value depends of stop mode selected.                                                                                                                                                                                 |
| stopMode            | String | delayed                 | No        | Stop mode. Possible values are: **date**: If this mode is selected, stop value is a date in YYYY-MM-DDThh:mm:ssTZD format. **delayed**: If this mode is selected, stop value is a time defined in milliseconds.                       |
| parameters          | Object | {}                      | No        | This value is a json with accepted value in selected operation. You can see parameters getting operation catalog.                                                                                                                     | 
| callback            | String | null                    | No        | URI where the result of the operation execution is received.                                                                                                                                                                          |

**NOTE**: The Job of the operation will be created with an active status by default.

The example execute _ADMINISTRATIVE_STATUS_CHANGE_ operation to received device.

```
operationConfig = {
    subEntityIdentifier: entity['provision.administration.identifier]._value._current.value,
    operationType: 'ADMINISTRATIVE_STATUS_CHANGE',     
    operationTimeout: 20000,   
    jobUser: apiKey,           
    retries: 1,            
    ackTimeout: 5000,        
    retriesDelay: 0,      
    stopValue: '25000',         
    stopMode: 'delayed',
    callback: 'http://127.0.0.1:7070/directory',
    parameters: {
        admsts: 'BANNED'
    }
}

operation.execute(operationConfig);
```
Another example with stoMode = date

```
operationConfig = {
    subEntityIdentifier: entity['provision.administration.identifier]._value._current.value,
    operationType: 'REFRESH_PRESENCE',     
    operationTimeout: 20000,   
    jobUser: apiKey,           
    retries: 1,            
    ackTimeout: 5000,        
    retriesDelay: 0,      
    stopMode: 'date',
    stopValue: '2045-11-08T08:30:00+02:00',
    callback: 'http://127.0.0.1:7070/directory',
    parameters: {}
}

operation.execute(operationConfig);
```

### Utils

The `utils` object is the main object for utilities.

#### utils.cancelDelay

`utils.cancelDelay(ruleName)` Cancel active delayed action produces by another rule activation.

The function take 1 parameters:

1. Name of rule that produce delayed actions.

The example cancel delay of 'highTemperatureRule' rule.

```
utils.cancelDelay('highTemperatureRule');
```

This function returns void.

#### utils.encryptString

`utils.encryptString(encryptConfig)`

This function takes as parameter an object with the following fields:

| Property      | Type    | Default | Mandatory | Description                                                           |
|---------------|---------|---------|-----------|-----------------------------------------------------------------------|
| originalValue | String  |         | Yes       | The value to encrypt                                                  |
| datastreamId  | String  |         | Yes       | The datastream of provision type, of datamodel that define this value |
| organization  | String  |         | Yes       | The organization name to wich belongs to the datamodel                |

```
encryptConfig = {
    originalValue: 'texto to encrypt',
    datastreamId: 'provision.data.encrypt',
    organization: 'organization_name'
}

utils.encryptString(encryptConfig);
```

This will return "value sent encrypted".

#### utils.decryptString

`utils.decryptString(decryptConfig)`

This function takes as parameter an object with the following fields:

| Property       | Type   | Default | Mandatory | Description                                                           |
|----------------|--------|---------|-----------|-----------------------------------------------------------------------|
| encryptedValue | String |         | Yes       | The value to decrypt                                                  |
| datastreamId   | String |         | Yes       | The datastream of provision type, of datamodel that define this value |
| organization   | String |         | Yes       | The organization name to wich belongs to the datamodel                |

The example:

```
decryptConfig = {
    encryptedValue: 'texto to decrypt',
    datastreamId: 'provision.data.encrypt',
    organization: 'organization_name'
}

utils.decryptString(decryptConfig);
```

This will return "value sent decrypted".

### Provision

The `provision` object is the main object for changing provision data.

#### provision.datastreams

`provision.datastreams(provisionConfig)`

This function takes as parameter an object with the following fields:

| Property		| Type    | Mandatory | Description                             					|
|---------------|---------|-----------|-------------------------------------------------------------------------|
| datastreams	| Object  | Yes       | Map where key is datastreamId and value is value to set in provision.	|
| apiKey  		| String  | Yes       | User apiKey that launch this provision. 								|

For example:

```
provisionConfig = {
    datastreams: {'datastream1':'value1','datastreamN':'valueN'},
    apiKey: 'the_api_key'
}

provision.datastreams(provisionConfig);
```

---

## Private/extended functions

### Getting datastreams

#### getCounterValueWithReset

`getCounterValueWithReset(datastreamValue, incValue, resetDate)` is same that getCounterValue, but returns a object with counter value, flag to get if is activated reset and previous value

The function take 3 parameters:

1. Datastream object
2. Value to increment
3. Reset date

This function returns a object with
- value: final counter value
- reset: flag to get if reset is activated
- preValue: previous value before calculate counter value.

#### getDatastreamValueCmmsModuleFromEntity

`getDatastreamValueCmmsModuleFromEntity(datastreamObject, position)` Get datastream object in communicationModules datastream (check that this datastreams is array and not object) selecting index of array.

The function take 2 parameters:

1. Datastream object
2. Index of array to get.

This function returns datastream with value, date, at... fields

#### getDatastreamByIdFromDB

> **Deprecated:** Use `utils` object functions instead.

`getDatastreamByIdFromDB(datastreamId, subEntityIdentifier)` get datastream reading from DB.

The function take 2 parameters:
1. datastream identifier
2. subEntity identifier

The function returns datastream object with date, at and value fields

#### getMonthlyCounterValueFromDB

`getMonthlyCounterValueFromDB(datastreamId, incValue, subEntityIdentifier)` get incremented value to readed datastream if date is before to get in function getDailyResetDate

The function take 3 parameters:
1. Datastream identifier
2. Value to increments
3. SubEntity identifier

The function returns datastream object with date, at and value fields


#### getMonthlyCounterValueFromDBWithReset
`getMonthlyCounterValueFromDBWithReset (datastream, incValue, subEntityIdentifier)` same that _getMonthlyCounterValueFromDB_ but returning object of _getCounterValueWithReset_

The function take 3 parameters:
1. Datastream identifier
2. Value to increments
3. SubEntity identifier

The function returns same object that _getCounterValueWithReset_ 

#### getMonthlyCounterValueFromMessage
`getMonthlyCounterValueFromMessage(datastream, incValue)` get incremented value to received datastream if date is before to get in function _getMonthlyResetDate_

The function take 2 parameters:
1. Datastream readed from entity
2. Value to increments

The function returns datastream object with date, at and value fields

#### getMonthlyCounterValueFromMessageWithReset
`getMonthlyCounterValueFromMessageWithReset(datastream, incValue)` same that _getMonthlyCounterValueFromMessage_ but returning object of _getCounterValueWithReset_

The function take 2 parameters:
1. Datastream readed from entity
2. Value to increments

The function returns same object that _getCounterValueWithReset_ 

#### getDailyCounterValueFromDB
`getDailyCounterValueFromDB(datastream, incValue, subEntityIdentifier)` get incremented value to readed datastream if date is before to get in function _getDailyResetDate_

The function take 3 parameters:
1. Datastream identifier
2. Value to increments
3. SubEntity identifier

The function returns datastream object with date, at and value fields

#### getDailyCounterValueFromDBWithReset
`getDailyCounterValueFromDBWithReset (datastream, incValue, subEntityIdentifier)` same that _getDailyCounterValueFromMessage_ but returning object of _getCounterValueWithReset_

The function take 3 parameters:
1. Datastream identifier
2. Value to increments
3. SubEntity identifier

The function returns same object that _getCounterValueWithReset_ 

#### getDailyCounterValueFromMessage
`getDailyCounterValueFromMessage(datastream, incValue)` get incremented value to received datastream if date is before to get in function _getDailyResetDate_

The function take 2 parameters:
1. Datastream readed from entity
2. Value to increments

The function returns datastream object with date, at and value fields

#### getDailyCounterValueFromMessageWithReset
`getDailyCounterValueFromMessageWithReset(datastream, incValue)` same that _getDailyCounterValueFromMessage_ but returning object of _getCounterValueWithReset_

The function take 2 parameters:
1. Datastream readed from entity
2. Value to increments

The function returns same object that _getCounterValueWithReset_

### Check and calculate

#### getDistance

> **Deprecated:** Use `location` object functions instead.

`getDistance(lat1, long1, lat2, long2)` get distance between 2 coordinates in meters.

The function take 4 parameters:
1. latitude from first coordinate
2. longitude from first coordinate
3. latitude from second coordinate
3. longitude from second coordinate

#### isInsertAction
`isInsertAction()` obtains if message is from insert provision action

This function returns boolean

#### isUpdateAction
`isUpdateAction()` obtains if message is from update provision action

This function returns boolean

#### isPatchAction
`isPatchAction()` obtains if message is from patch provision action

This function returns boolean


#### getAreas

> **Deprecated:** Use `location` object functions instead.

`getAreas(coordinates)` get provisioned areas in OpenGate to selected coordinate.

The function take 1 parameter:
1. Array of 2 double elements, first longitude an second latitude.

The function returns array of object with this fields:

- identifier
- name
- description
- order

#### getSortedAreas

> **Deprecated:** Use `location` object functions instead.

`getSortedAreas(coordinates, sorted)` Same that getAreas but is possible get sorted by ordered field.

The function take 2 parameters:
1. Array of 2 double elements, first longitude an second latitude.
2. boolean sorted to request sorted array or not.

The function returns array of object with this fields:

- identifier
- name
- description
- order


#### arrayEquals

`arrayEquals(array1, array2)` check if 2 arrays are equals

The function take 2 parameters:
1. One array
2. Another array

This function returns true if two arrays has the same elements in same order.

#### printLog

> **Deprecated:** Use `logger` object functions instead.

`printLog(log)` show in traces log object

The function take 1 parameters:
1. object to show in traces

This function returns void.

### Actions

#### openAlarmWithExtraInfo

> **Deprecated:** Use `alarm` object functions instead.

`openAlarmWithExtraInfo(subEntityIdentifier, alarmName, ruleName, severity, priority, alarmDescription, extra_info)` is same that _openAlarm_ adding extraInfo parameter.

#### collectDatastreams

> **Deprecated:** Use `collect` object functions instead.

`collectDatastreams(jsMapValues, jsCommsValues, directMode, reinject)` collect values in received entity

The function take 4 parameters:

1. jsMapValues. Map where key is datastreamId and value is value to collect.
2. jsCommsValue. Map where key is commsId and value is another map where key is datastreamId and value is value to collect
3. select if use complete flow to collect or simple flow
4. select if message with new collected datastreams must be received in rules or not.


#### collectDataPoints

> **Deprecated:** Use `collect` object functions instead.

`collectDataPoints(resourceType, identifier, mapValues, commsValues, ttl)` collect datapoints in received entity

The function take 4 parameters:

1. select entity resourceType
2. select entity identifier
3. mapValues. Map where key is datastreamId and value is value to collect.
4. commsValue. Map where key is commsId and value is another map where key is datastreamId and value is value to collect
5. ttl to datapoint, getting minor of selected ttl, ttl in datamodel definition and ttl in organization plan

#### cancelJobGroup

> **Deprecated:** Use `utils` object functions instead.

`cancelJobGroup(ruleName)` Cancel active delayed action produces by another rule activation.

The function take 1 parameters:

1. Name of rule that produce delayed actions.

The example cancel delay of 'highTemperatureRule' rule.

```
cancelJobGroup('highTemperatureRule');
```

This function returns void.

#### encryptString

> **Deprecated:** Use `utils` object functions instead.

`encryptString(originalValue, datastreamId, organization)`

The function takes 3 parameters:

1. The value to encrypt.
2. The datastream of provision type, of datamodel that define this value.
3. The organization name to wich belongs to the datamodel. 

The example:

```
encryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```

This will return "value sent encrypted".

#### decryptString

> **Deprecated:** Use `utils` object functions instead.

`decryptString(encryptedValue, datastreamId, organization)`

The function takes 3 parameters:

1. The value to decrypt.
2. The datastream of provision type, of datamodel that define this value.
3. The organization name to wich belongs to the datamodel. 

The example:

```
decryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```

This will return "value sent decrypted".


### Data collection

The `collect` object is the main object for fetching data.

#### collect.datastreams
`collect.datastreams(datastreamConfig)` collect values in received entity

This function takes as parameter an object with the following fields:

| Property         | Type     | Default | Mandatory | Description                                                                                               |
|------------------|----------|---------|-----------|-----------------------------------------------------------------------------------------------------------|
| jsMapValues      | Object   |         | No        | Map where key is datastreamId and value is value to collect.                                              |
| jsCommsValue     | Object   |         | No        | Map where key is commsId and value is another map where key is datastreamId and value is value to collect |
| directMode       | Boolean  | True    | No        | Select if use complete flow to collect or simple flow                                                     |
| reinject         | Boolean  | False   | No        | Select if message with new collected datastreams must be received in rules or not.                        |
| at			   | String	  |         | No        | String with the collected date in ISODate format. Example: 2025-05-08T09:23:40.289Z						|


#### collect.datapoints
`collect.datapoints(datapointConfig)` collect datapoints in received entity

This function takes as parameter an object with the following fields:

| Property     | Type   | Default | Mandatory | Description                                                                                               |
|--------------|--------|---------|-----------|-----------------------------------------------------------------------------------------------------------|
| resourceType | String |         | No        | select entity resourceType                                                                                |
| identifier   | String |         | No        | select entity identifier                                                                                  |
| mapValues    | Object |         | No        | Map where key is datastreamId and value is value to collect                                               |
| commsValues  | Object |         | No        | Map where key is commsId and value is another map where key is datastreamId and value is value to collect |
| ttl          | Number |         | No        | ttl to datapoint, getting minor of selected ttl, ttl in datamodel definition and ttl in organization plan |
| att          | String |         | No        |                                                                                                           |


### Location

The `location` object is the main object for calculations.

#### location.getAreas

`location.getAreas(coordinates)` get provisioned areas in OpenGate to selected coordinate.

The function take 1 parameter:
1. Array of 2 double elements, first longitude an second latitude.

The function returns array of object with this fields:

- identifier
- name
- description
- order

#### location.getSortedAreas

`location.getSortedAreas(coordinates, sorted)` Same that getAreas but is possible get sorted by ordered field.

The function take 2 parameters:
1. Array of 2 double elements, first longitude an second latitude.
2. boolean sorted to request sorted array or not.

The function returns array of object with this fields:

- identifier
- name
- description
- order

#### location.getDistance

`location.getDistance(lat1, long1, lat2, long2)` get distance between 2 coordinates in meters.

The function take 4 parameters:
1. latitude from first coordinate
2. longitude from first coordinate
3. latitude from second coordinate
3. longitude from second coordinate


### Utils

The `utils` object is the main object for utilities.

#### utils.getDatastreamByIdFromDB

`utils.getDatastreamByIdFromDB(datastreamId, subEntityIdentifier)` get datastream reading from DB.

The function take 2 parameters:
1. datastream identifier
2. subEntity identifier

The function returns datastream object with date, at and value fields


### HTTP client request

In enabled to do synchronous http request to get external information and use in OpenGate Rules. To use this method is necesary create http.client object.

The `http` object is the main object of the HTTP client. It allows perform different actions such as do http request through http/https protocol and https with mutual authentication.

**Http client properties**


| Property       	| Type    | Default | Description                                                     												|
|------------------	|-------- |---------|-------------------------------------------------------------------------------------------------------------	|
| method         	| string  | GET     | One of http methods: POST, GET, ...                                                                         	|
| uri            	| string  | null    | Request uri                                                                                                 	|
| headers        	| JSON    | null    | Request headers in json format                                                                              	|
| body           	| *       | null    | Response body                                                                                               	|
| alias          	| string  | null    | Used to set https context for mutual authentication, alias to be used for keystore 														|
| certificate    	| string  | null    | Used to set https context for mutual authentication, certificate content for keystore    												|
| privateKey     	| string  | null    | Used to set https context for mutual authentication, private key content from certificate  												|
| trustedAll	 	| boolean | false   | if true, trusted in all certificates, if false you may have to set trustedCetificate and trustedAlias properties	|
| trustedCertificate| string  | null    | Used to build truststore for https connection 																	|
| trustedAlias		| string  | null    | Used to set https context, alias to be used for truststore
| redirectPolicy 	| string  | null    | Overrides default redirection policy                            												|
| clientVersion  	| string  | null    | Overrides default client version configured                     												|
| timeOut        	| integer | null    | Overrides default timeout configured. Defined in seconds        												|

To use HTTPS protocol should set properties certificate and privateKey.


**Http client response**

Following methods return Http Request result JSON with these fields:

| Field          |  Description                   |
|----------------|--------------------------------|
| statusCode     | Received response HTTP code    |
| body           | Received response body         |
| headers        | Received response headers      |

#### http.client.post

`http.client.post()` Send POST http method. Property `method` is override in function.

To make the POST request it is necessary to complete the parameters defined in http.client which are defined in the *HTTP Client Request* section of this guide.


Example of use with default values:
```
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.post();
```

#### http.client.put

`http.client.put()` Send PUT http method. Property `method` is override in function.

To make the PUT request it is necessary to complete the parameters defined in http.client which are defined in the *HTTP Client Request* section of this guide.

Example of use with default values:
```
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.put();
```

#### http.client.get

`http.client.get()` Send GET http method.

To make the GET request it is necessary to complete the parameters defined in http.client which are defined in the *HTTP Client Request* section of this guide. To view them you can click on the maximize button.

Example of use with default values:
```
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
var httpResp = http.client.get();
```

#### http.client.delete

`http.client.delete()` Send DELETE http method. Property `method` is override in function.

To make the DELETE request it is necessary to complete the parameters defined in http.client which are defined in the *HTTP Client Request* section of this guide.

Example of use with default values:
```
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
var httpResp = http.client.delete();
```

#### http.client.patch

`http.client.patch()` Send PATCH http method. Property `method` is override in function.

To make the PATCH request it is necessary to complete the parameters defined in http.client which are defined in the *HTTP Client Request* section of this guide.

Example of use with default values:
```
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.patch();
```

##### http.client.request

`http.client.request()` Send REQUEST http method. Property `method` must be defined when is called.

To make the PUT request it is necessary to complete the parameters defined in http.client which are defined in the *HTTP Client Request* section of this guide.

Example of use with default values:
```
http.client.method = "GET";
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.request();
```
