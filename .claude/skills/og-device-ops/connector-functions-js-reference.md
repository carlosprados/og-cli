# Connector Functions JavaScript — official platform guide

> Harvested live from `GET /north/v80/connectorFunctions/doc/jsApi/client` on 2026-06-04.

# Connector Functions Guide

In this javascript code, it is possible to use some defined functions to define the connector function. We will explain them below.

## Input parameters
The main script will have access to three main vars:

- entity: json with flattened operation target device entity representation.
- gateway: json with flattened gateway entity representation. It can be null.
- payload: it can be of different types: json object, binary content or flat text. It can contain different types of information: request or response inforamtion, collected data....
- contextParams: json object with execution context information. It can have some of this params:
  - apiKey: device or user apikey.
  - remoteIp: remote host when HTTP Rest Resource is invoked.
  - uri: opened Websocket complete uri or invoked HTTP Rest Resource complete uri.
  - path: opened Websocket relative path or invoked HTTP Rest Resource relative path. This is the path used as south criteria to filter CFs.
  - topic: MQTT Topic where the message arrived.


Here there is an example of `entity` or `gateway`:

```
entity = {
  "provision.administration.channel": {
    "_value": {
      "_current": {
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
        }
      }
    }
  ],
  "device.identifier": {
    "_value": {
      "_current": {
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
      "_current": {
        "value": 25.3,
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }
    }
  },
  "_value": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.value;} else {return null;}} else {return ds._value._current.value;}} else {return null;}} else {return null;}},
  "_at": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.at;} else {return null;}} else {return ds._value._current.at;}} else {return null;}} else {return null;}},
  "_date": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.date;} else {return null;}} else {return ds._value._current.date;}} else {return null;}} else {return null;}},
  "_source": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.source;} else {return null;}} else {return ds._value._current.source;}} else {return null;}} else {return null;}},
  "_sourceInfo": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.sourceInfo;} else {return null;}} else {return ds._value._current.sourceInfo;}} else {return null;}} else {return null;}}
}
```

`contextParams` some examples:

REQUEST Connector function:
```
  {
    "apiKey": "97c9ceae-c4ee-49ea-ad50-e499bb55ac63",
  }
```

COLLECTION or RESPONSE Connector function for MQTT connector:
```
  {
    "apiKey": "97c9ceae-c4ee-49ea-ad50-e499bb55ac63",
    "topic": "some/topic"
  }
```

COLLECTION or RESPONSE Connector function for HTTP Rest endpoints:
```
  {
    "apiKey": "97c9ceae-c4ee-49ea-ad50-e499bb55ac63",
    "path": "context/path",
    "uri": "http://http_url/device_id/context/path",
    "remoteIp": "remote_device_ip"
  }
```

COLLECTION or RESPONSE Connector function for Websocket connection:
```
  {
    "apiKey": "97c9ceae-c4ee-49ea-ad50-e499bb55ac63",
    "path": "context/path"
    "uri": "ws://ws_url/device/context/path";
  }
```


## Connector function script output

Depending on the type of CF, the script must have different outputs (regardless of whether other calls are concatenated).

### REQUEST CF

In this case no output is necessary, so `return null;` can be used or directly not define any `return` statement.

### RESPONSE CF

In this case, Opengate Standard Response object must be returned. If nothing or null is returned, then no operation update will be done. 

There are some help functions in order to compose this object. See:

* ogResponse
* ogStep
* ogStepResponse

### COLLECTION CF

In this case, Opengate Standard Iot Data Collection object must be returned. If nothing or null is returned, then no collection will be done. 

There are some help functions in order to compose this object. See:

* ogCollection
* ogCollectionDs
* ogCollectionDp
* addOgCollectionDp


## Connector function execution concatenation

In some cases, it is possible invoke the execution of other CFs once the current CF execution is finished.

These are allowed cases:

* From REQUEST CF: 
** Invoke RESPONSE CF
** Invoke COLLECTION CF
** Invoke RESPONSE CF and COLLECTION CF
* From RESPONSE CF:
** Invoke COLLECTION CF

Other invocations will be ignored (for example, invoke RESPONSE CF from COLLECTION CF).

Use the `cf` object:

* `cf.response(responseFunctionCriteria, responsePayload)`
* `cf.collection(collectionFunctionCriteria, collectionPayload)`

The deprecated globals `responseCF(data, criteria)` and `collectCF(data, criteria)` still work and
are what most existing scripts call. **They take their arguments in the opposite order** — payload
first, criteria second — so migrating a call is not a rename: the arguments have to be swapped.
There is no `collectionCF`; an earlier copy of this guide invented it.



## Javascript functions catalog

### Plain functions

#### entityValue

> **Deprecated:** Use `entity` object functions instead.

`entityValue(entity, datastream, index)` Extract from entity specified datastream "value" field value

The function takes 3 parameters:

1. Object with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'provision.device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	}
};
var deviceId = entityValue(entity, "provision.device.identifier");
```

This will return "device_identifier".


#### entitiesValue

> **Deprecated:** Use `utils` object functions instead.

`entitiesValue(entities, datastream, index)` Extract from first entity that has specified datastream "value" field value

The function takes 3 parameters:

1. Array of bbjects with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'provision.device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	}
};
var deviceId = entityValue(entity, "provision.device.identifier");
```

This will return "device_identifier".

#### entityAt

> **Deprecated:** Use `entity` object functions instead.

`entityAt(entity, datastream, index)` Extract from entity specified datastream "at" field value

The function takes 3 parameters:

1. Object with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'provision.device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	}
};
var at = entityAt(entity, "provision.device.identifier");
```

This will return "2022-03-17T15:17:04.531Z".

#### entityDate

> **Deprecated:** Use `entity` object functions instead.

`entityDate(entity, datastream, index)` Extract from entity specified datastream "date" field value

The function takes 3 parameters:

1. Object with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'provision.device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	}
};
var date = entityDate(entity, "provision.device.identifier");
```

This will return "2022-03-17T15:17:05.531Z".

#### operationParameters

`operationParameters(operationObj)` Extract from operationObj parameters field. This function could be used in REQUEST CFs, when the payload is Operation Request json. If operationObj is not correct Request object, null will be returned.

The function takes 2 parameters:

1. operationObj: Object with operation request.
2. parameter: String with parameter name.

The example:

```
var apiKey = operationParameter({ "operation": { "request": { "timestamp": 1647452743387, "name": "SL_REFRESH_INFO", "parameters": { "paramName": "paramValue" }, "id": "2ec29c65-c261-438c-9302-0dfbc48db865", "deviceId": "9a318f83-3c1d-4543-a327-0a4e07a2c143" } } }, "paramName");
```

This will return "paramValue".


#### operationTimestamp

`operationTimestamp(operationObj)` Extract from operationObj request timestamp. This function could be used in REQUEST CFs, when the payload is Operation Request json. If operationObj is not correct Request object, null will be returned.

The function takes 1 parameter:

1. operationObj: Object with operation request.

The example:

```
var ts = operationTimestamp({ "operation": { "request": { "timestamp": 1647452743387, "name": "SL_REFRESH_INFO", "parameters": { "paramName": "paramValue" }, "id": "2ec29c65-c261-438c-9302-0dfbc48db865", "deviceId": "9a318f83-3c1d-4543-a327-0a4e07a2c143" } } });
```

This will return 1647452743387.


#### operationName

`operationName(operationObj)` Extract from operationObj request operation name. This function could be used in REQUEST CFs, when the payload is Operation Request json. If operationObj is not correct Request object, null will be returned.

The function takes 1 parameter:

1. operationObj: Object with operation request.

The example:

```
var opName = operationName({ "operation": { "request": { "timestamp": 1647452743387, "name": "SL_REFRESH_INFO", "parameters": { "paramName": "paramValue" }, "id": "2ec29c65-c261-438c-9302-0dfbc48db865", "deviceId": "9a318f83-3c1d-4543-a327-0a4e07a2c143" } } });
```

This will return "SL_REFRESH_INFO".


#### operationId

`operationId(operationObj)` Extract from operationObj request operation id. This function could be used in REQUEST CFs, when the payload is Operation Request json. If operationObj is not correct Request object, null will be returned.

The function takes 1 parameter:

1. operationObj: Object with operation request.

The example:

```
var opId = operationId({ "operation": { "request": { "timestamp": 1647452743387, "name": "SL_REFRESH_INFO", "parameters": { "paramName": "paramValue" }, "id": "2ec29c65-c261-438c-9302-0dfbc48db865", "deviceId": "9a318f83-3c1d-4543-a327-0a4e07a2c143" } } });
```

This will return "2ec29c65-c261-438c-9302-0dfbc48db865".


#### operationDeviceId

`operationDeviceId(operationObj)` Extract operation id from operationObj

The function takes 1 parameter:

1. operationObj: Object with operation request.

The example:

```
var opDevId = operationDeviceId({ "operation": { "request": { "timestamp": 1647452743387, "name": "SL_REFRESH_INFO", "parameters": { "paramName": "paramValue" }, "id": "2ec29c65-c261-438c-9302-0dfbc48db865", "deviceId": "9a318f83-3c1d-4543-a327-0a4e07a2c143" } } });
```

This will return "9a318f83-3c1d-4543-a327-0a4e07a2c143".


#### webSocketMsg

> **Deprecated:** Use `websocket` object functions instead.

`webSocketMsg(payload, deviceId)` Send message to opened websocket

The function takes 2 parameters:

1. Object with payload to be published. It will be converted to string.
2. String with device identifier.

The example:

```
webSocketMsg({someField:someValue}, 'someDeviceId');
```


#### httpRequest

> **Deprecated:** Use `http` object functions instead.

`httpRequest(request, payload)` Executes specified request with specified payload.

The function takes 2 parameters:
1. request: Object with requests parameters: method, uri, headers.
*  'method': GET, POST, PUT, DELETE.
*  'uri': request uri.
*  'headers': json with http request headers.
2. payload: data to be sent. It can be null.

The example:

This will do http request like this

```
var request = {
  'uri':'http://request/uri',
  'method': 'POST',
  'headers': {
    'ContentType':'application/json',
    'Accept': 'application/json',
    'X-Api-Key': '1111-22-333-4444'
  }
}
var httpResponse = httpRequest(request, {'someField':'someValue'});
```

And the result will be:

```
{
  'statusCode': 201,
  'body': 'some content'
}
```


#### log

> **Deprecated:** Use `logger` object functions instead.

`log(...msg)` Creates Info level logging messages. It concatenates msg parameters in the final string to logged. 

The function takes as parameters a list of elements to be concatenated to generate the string message to be printed.

This function returns void.

#### encryptString

> **Deprecated:** Use `utils` object functions instead.

`encryptString(originalValue, datastreamId, organization)`  Encrypt an original string with the configuration established by the datastream of the organization

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

`decryptString(encryptedValue, datastreamId, organization)`  Decrypt an encrypted string with the configuration established by the datastream of the organization

The function takes 3 parameters:

1. The encrypted value to decrypt.
2. The datastream of provision type, of the datamodel that define this value. This datastream has the configuration of encryption.
3. The organization name to wich belongs to the datamodel. 

The example:

```
decryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```

This will return "value sent decrypted".

#### base64ToByteArray

`base64ToByteArray(base64String)`  Decode a base64 string to byte array

The function takes 1 parameter:

1. The base64 string to decode.

The example:

```
base64ToByteArray('gCdFyGGkIodI3gUF/SuAOUMWKQ==');
```

This will return `[80, 27, 45, c8, 61, a4, 22, 87, 48, de, 05, 05, fd, 2b, 80, 39, 43, 16, 29]`


### websocket

#### websocket.sendMsg

`websocket.sendMsg()` Send message to opened websocket

Websocket object has 2 properties:
websocket.payload: Object with payload to be published. It will be converted to string.
websocket.deviceId: String with device identifier.

The example:

```
websocket.payload = {someField:someValue};
websocket.deviceId = 'someDeviceId'
websocket.sendMsg();
```




### response

#### response.addStep
`response.addStep(name, result, description, stepResponseList)`  Add step to response object
1. Step name
2. Result code
3. Result description
4. Response list

The example:

```
response.addStep('LOAD_CURVE', 'SUCCESS', 'Successful status');
```

#### response.sendSteps
`response.sendSteps()`  Send added steps and remove them from response object

The example:

```
response.sendSteps();
```

#### response.successful
`response.successful(description)`  Set response status to SUCCESSFUL with defined description

The example:

```
response.successful('Operation finished correctly');
```

#### response.errorProcessing
`response.errorProcessing(description)`  Set response status to ERROR_PROCESSING with defined description

The example:

```
response.errorProcessing('Operation failed');
```

#### response.errorInParam
`response.errorInParam(description)`  Set response status to ERROR_IN_PARAM with defined description

The example:

```
response.errorInParam('Unexpected param');
```

#### response.notSupported
`response.notSupported(description)`  Set response status to NOT_SUPPORTED with defined description

The example:

```
response.notSupported('Invalid operation');
```

#### response.errorTimeout
`response.errorTimeout(description)`  Set response status to ERROR_TIMEOUT with defined description

The example:

```
response.errorTimeout('Timeout executing operation');
```

#### response.unknownResult
`response.unknownResult(description)`  Set response status to UNKNOWN_ERROR with defined description

The example:

```
response.unknownResult('Unexpected result');
```

### collection
#### collection.addDatapoint
`collection.addDatapoint(datastreamId, value, at, source, sourceInfo)`  Add datapoint to collection object
1. Datastream identifier
2. Datapoint value
3. Datapoint at time
4. Datapoint source value
5. Datapoint sourceInfo value

The example:

```
collection.addDatapoint('device.cpu.usage', 95);
```

#### collection.setFeed
`collection.setFeed(datastreamId, feed)`  Set feed for specified datastream
1. Datastream identifier
2. Feed value

The example:

```
collection.setFeed('device.cpu.usage', 'feed_name');
```

#### collection.send
`collection.send()` Send added datapoints and remove them from collection object

The example:

```
collection.send();
```

#### collection.getDatastream
`collection.getDatastream(datastream)` Search specified datastream in collection object and return full datastream with all datapoints
1. Datastream identifier

The example:

```
var dsObject = collection.getDatastream('device.cpu.usage');
```

#### collection.getValue
`collection.getValue(datastream, dpIndex)` Search specified datastream in collection object and return specifierd datapoint value
1. Datastream identifier
3. Datapoints array index, by default 0 index

The example:

```
var dpValue = collection.getValue('device.cpu.usage');
```

#### collection.clone
`collection.clone()` Creates new object with same data and with all functions

The example:

```
var otherCollection = collection.clone();
otherCollection.device='otherEntity';
otherCollection.addDatapoint('device.cpu.usage', 95);
otherCollectino.send();
//return main collection object
return collection;
```


### cf
#### cf.response
`cf.response(responseFunctionCriteria, responsePayload)`  Concatenate Response CF execution
1. Response Connector function South criteria
2. Response Connector function payload. If not specified, default response object is used

The example:

```
cf.response('https://some/response/cf');
```

#### cf.collection
`cf.collection(responseFunctionCriteria, responsePayload)`  Concatenate Collection CF execution
1. Collection Connector function South criteria
2. Collection Connector function payload. If not specified, default collection object is used

The example:

```
cf.collection('https://some/response/cf');
```

### utils
#### utils.atcmd.toDBm
`utils.atcmd.toDBm(value)` Convert value to DB

1. Value to be converted

The example:

```
var dbVal = utils.atcmd.toDBm(1234);
```

#### utils.endesa.torscp
`utils.endesa.torscp(value)` Usedn in Endesa

1. Value to be converted

#### utils.endesa.toDbmPlusQuality
`utils.endesa.toDbmPlusQuality(value)` Usedn in Endesa

1. Value to be converted

#### utils.endesa.prepareMsisdn
`utils.endesa.prepareMsisdn(value)` Usedn in Endesa

1. MSISDN to be transformed

#### utils.endesa.commandFrom
`utils.endesa.commandFrom(value)` Usedn in Endesa

1. Command code

#### utils.odm.addValueToContext
`utils.odm.addValueToContext(key,value)`

1. Conntext key
2. Context value as String

#### utils.odm.addIdentificationHint
`utils.odm.addIdentificationHint(key,value,structured)`

1. Conntext key
2. Identification hint
3. Boolean

#### utils.odm.sleep
`utils.odm.sleep(time)` Sleep for specified time

1. Time in milliseconds

#### utils.odm.entitiesValue

`utils.odm.entitiesValue(entities, datastream, index)` Extract from first entity that has specified datastream "value" field value

The function takes 3 parameters:

1. Array of bbjects with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'provision.device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	}
};
var deviceId = utils.odm.entitiesValue(entity, "provision.device.identifier");
```

This will return "device_identifier".

#### utils.odm.httpRequest
`utils.odm.httpRequest(request, payload)` Executes specified request with specified payload.

The function takes 2 parameters:
1. request: Object with requests parameters: method, uri, headers.
*  'method': GET, POST, PUT, DELETE.
*  'uri': request uri.
*  'headers': json with http request headers.
2. payload: data to be sent. It can be null.

The example:

This will do http request like this

```
var request = {
  'uri':'http://request/uri',
  'method': 'POST',
  'headers': {
    'ContentType':'application/json',
    'Accept': 'application/json',
    'X-Api-Key': '1111-22-333-4444'
  }
}
var httpResponse = utils.odm.httpRequest(request, {'someField':'someValue'});
```

And the result will be:

```
{
  'statusCode': 201,
  'body': 'some content'
}
```


#### utils.odm.encryptString

`utils.odm.encryptString(originalValue, datastreamId, organization)`  Encrypt an original string with the configuration established by the datastream of the organization

The function takes 3 parameters:

1. The original value to encrypt.
2. The datastream of provision type, of the datamodel that define this value. This datastream has the configuration of encryption.
3. The organization name to wich belongs to the datamodel.

The example:

```
utils.odm.encryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```

This will return "value sent encrypted".

#### utils.odm.decryptString

`utils.odm.decryptString(encryptedValue, datastreamId, organization)`  Decrypt an encrypted string with the configuration established by the datastream of the organization

The function takes 3 parameters:

1. The encrypted value to decrypt.
2. The datastream of provision type, of the datamodel that define this value. This datastream has the configuration of encryption.
3. The organization name to wich belongs to the datamodel.

The example:

```
utils.odm.decryptString('texto to encrypt', 'provision.data.encrypt', 'organization_name');
```


#### utils.date.period.previousQuarter
`utils.date.period.previousQuarter(referenceTime)` Return previous quarter to referenceTime

1. Reference time

#### utils.date.period.previousDay
`utils.date.period.previousDay(referenceTime)` Return previous day period to referenceTime

1. Reference time

#### utils.date.period.previousWeek
`utils.date.period.previousWeek(referenceTime)` Return previous week period to referenceTime

1. Reference time

#### utils.date.period.previousMonth
`utils.date.period.previousMonth(referenceTime)` Return previous month period to referenceTime

1. Reference time

#### utils.date.period.customPeriodUtc
`utils.date.period.customPeriodUtc(periodInit, periodEnd)` Return custom period with init and finish time

1. ISO Date string
2. ISO Date string

#### utils.date.period.lastMinutes
`utils.date.period.lastMinutes(minutes, referenceTime)` Return period of last minutes from reference time

1. Period number of minutes
2. Reference time

#### utils.date.period.lastHours
`utils.date.period.lastHours(hours, referenceTime)` Return period of last hours from reference time

1. Period number of minutes
2. Reference time

#### utils.date.period.lastDays
`utils.date.period.lastDays(days, referenceTime)` Return period of last days from reference time

1. Period number of minutes
2. Reference time

#### utils.bytes.fromHexString

`utils.bytes.fromHexString(hexString)` Convert hexadecimal String to Uint8Array.

The function takes 1 parameter:

1. The hexadecimal string to decode (spaces are alowed).

The example:

```
utils.bytes.fromHexString('09 4A 48');
```

This will return `[09, 74, 72]`

#### utils.bytes.toHexString

`utils.bytes.toHexString(byteArray)` Convert byte array to hexadecimal string.

The function takes 1 parameter:

1. The array of bytes to be converted.

The example:

```
utils.bytes.toHexString([09, 74, 72]);
```

This will return `094A48`

#### utils.bytes.fromText

`utils.bytes.fromText(str)` Convert strings to array of bytes.

The function takes 1 parameter:

1. String of text.

The example:

```
utils.bytes.fromText('hello');
```

This will return `[104,101,108,108,111]`

#### utils.bytes.toText

`utils.bytes.toText(bytes)` Convert array of bytes to a string of text.

The function takes 1 parameter:

1. Array of bytes.

The example:

```
utils.bytes.toText([104,101,108,108,111]);
```

This will return `'hello'`

### iec102
#### iec102.disconnect
`iec102.disconnect()` Close opened session.

#### iec102.send
`iec102.send(command, waitFor, pattern)` Send specified command
1. Command to be sent
2. Array of expected results. It can be null.
3. Regex to extract values. It can be null.


#### iec102.connect
`iec102.connect(registerType, waitFor)` Open session.
1. Connection type. Not mandatory.
2. Array of expected results. Not mandatory.

#### iec102.connectWithEndpoints
`iec102.connectWithEndpoints(endpoints)` Connect wrapper using a list of endpoints
1. List of endpoints. Each endpoints specifies IP, PORT... It uses GSM registerType.

#### iec102.connectWithIpAndPorts
`iec102.connectWithIpAndPorts(ports)` Connect wrapper using a list ports
1. List of ports to be tryed.

#### iec102.asdus.addFromParams
`iec102.asdus.addFromParams()` Calculates ASDUs to be executed from operations params. To allow this calculations operations params must fit with GET_METER_INFO params 

#### iec102.asdus.add
`iec102.asdus.add(name, execConfig)` Add new ASDU to asdus execution list
1. Asdu name
2. Asdu execution configuracion (default collection, steps, period...)

#### iec102.asdus.execute
`iec102.asdus.execute()` Execute one by one all ASDUS from asdus execution list.

#### iec102.asdus.login
`iec102.asdus.login(execConfig)` Execute login ASDU

1. Execution configuration


#### iec102.asdus.timeRequest
`iec102.asdus.timeRequest(execConfig)` Execute timeRequest ASDU

1. Execution configuration

#### iec102.asdus.dayLightSavingTime
`iec102.asdus.dayLightSavingTime(execConfig)` Execute dayLightSavingTime ASDU

1. Execution configuration

#### iec102.asdus.loadCurve
`iec102.asdus.loadCurve(execConfig)` Execute loadCurve ASDU

1. Execution configuration

#### iec102.asdus.loadCurveQuarter
`iec102.asdus.loadCurveQuarter(execConfig)` Execute loadCurveQuarter ASDU

1. Execution configuration

#### iec102.asdus.loadCurveIncremental
`iec102.asdus.loadCurveIncremental(execConfig)` Execute loadCurveIncremental ASDU

1. Execution configuration

#### iec102.asdus.loadCurveIncrementalQuarter
`iec102.asdus.loadCurveIncrementalQuarter(execConfig)` Execute loadCurveIncrementalQuarter ASDU

1. Execution configuration

#### iec102.asdus.configuration
`iec102.asdus.configuration(execConfig)` Execute configuration ASDU

1. Execution configuration

#### iec102.asdus.deviceManufacturer
`iec102.asdus.deviceManufacturer(execConfig)` Execute deviceManufacturer ASDU

1. Execution configuration

#### iec102.asdus.parameters
`iec102.asdus.parameters(execConfig)` Execute parameters ASDU

1. Execution configuration

#### iec102.asdus.currentPricing
`iec102.asdus.currentPricing(execConfig)` Execute currentPricing ASDU

1. Execution configuration

#### iec102.asdus.storedPricing
`iec102.asdus.storedPricing(execConfig)` Execute storedPricing ASDU

1. Execution configuration

#### iec102.asdus.logout
`iec102.asdus.logout(execConfig)` Execute logout ASDU

1. Execution configuration


### icmp
#### icmp.send
`icmp.send()` Send ping

### mqtt
#### mqtt.publish
`mqtt.publish(message)` Publish message on specified topic

### snmp
#### snmp.addOid
`snmp.addOid(oid, type, value)` Configure oid

1. Identifier
2. Type
3. Value

#### snmp.get
`snmp.get()` Obtain values for specified oids

#### snmp.set
`snmp.set()` Set values values for specified oids

### ssh
#### ssh.connect
`ssh.connect(waitFor)` Stablish ssh connection

1. Expected prompt messages

#### ssh.disconnect
`ssh.disconnect()` Close openened connection

#### ssh.send
`ssh.send(command, pattern, waitFor)` Send specified command

1. Command to be sent
2. Pattern list to process messages
3. Expected prompt messages 


### telnet
#### telnet.connect
`telnet.connect(waitFor)` Stablish telnet connection

1. Expected prompt messages

#### telnet.disconnect
`telnet.disconnect()` Close openened connection

#### telnet.send
`telnet.send(command, pattern, waitFor)` Send specified command

1. Command to be sent
2. Pattern list to process messages
3. Expected prompt messages 

### dlms
#### dlms.addAttr
`dlms.addAttr(classId, obisCode, attrId, type, value)` Add dlms attribute definition. It's used both for *dmls.get()* and *dmls.set()* methods

1. dlms object's class identifier
2. dlms object's obisCode
3. dlms object's attibute identifier
4. attibute type
5. attibute value

The example:

```
dlms.addAttr(1, "1.2.3.4.5.6", 2, "unsigned", 254);
```

#### dlms.addMethod
`dlms.addMethod(classId, obisCode, methodId, type, value)` add dlms method definition. It's used for *dmls.method()*

1. dlms object's class identifier
2. dlms object's obisCode
3. dlms object's method identifier
4. method's parameter type
5. method's parameter value

The example:
```
dlms.addMethod(7, '1.2.3.4.5.6', 2, 'array', [{type: 'unsigned', value: 1},{type: 'unsigned', value: 2}]);
```

#### dlms.connect
`dlms.connect()` open a dlms connection

#### dlms.disconnect
`dlms.disconnect()` close a dlms connection

#### dlms.get
`dlms.get(descriptive)` executes a multi DLMS get attribute request with the previously specified payload (*dmls.addAttr(...)*).

1. Descriptive mode returns complex data as `Array` of `object` containing `type` and `value` for each object, non descritive mode flattens the returned array

The example:
```
dlms.addAttr(1, "0.0.0.0.0.0", 2);
var return = dlms.get(true);
```

This will return an `array`

#### dlms.set
`dlms.set(descriptive)` executes a multi DLMS set attribute request with the previously specified payload (*dmls.addAttr(...)*).

1. Descriptive mode  **does not matter on `set()`**. It's included to have the same signature as `get()` and in case a device returns something in a set.

The example:
```
dlms.addAttr(1, "0.0.0.0.0.0", 2, "boolean", false);
var return = dlms.set(true);
```

This will return an `array`

#### dlms.method
`dlms.method(descriptive)` Executes a multi DLMS method (or action) request with the previously specified payload (*dlms.addMethod()*)

1. Descriptive mode returns complex data as `Array` of `object` containing `type` and `value` for each object, non descritive mode flattens the returned array

The example:
```
dlms.addMethod(7, "1.0.99.1.0.254", 2, "unsigned", 0);
var result = dlms.method();
```

This will return an `array`

#### dlms.getCompactData
`dlms.getCompactData(typeDescription, value, descriptive, italianMode)` Extract the compact data serialized in a byte array acording to the description given.

1. Object with the description of the data
2. Array of bytes with the compact data
3. Descriptive mode returns complex data as `Array` of `object` containing `type` and `value` for each object, non descritive mode flattens the returned array
4. Determines if the compact data decoding of arrays must be in italian mode or not (`explicitArrayLengthInContent`)

The example:
```
var typeDescription = {"type": "structure", "items": [
		{"type": "long-unsigned"},
		{"type": "unsigned"},
		{"type": "array", "length": 1, "subtype": {"type": "long-unsigned"}},
		{"type": "octet-string"}
	]
};
var data = dlms.getCompactData(typeDescription, payload, true, false);
```

This will return an `Object` (descriptive mode) or any type described as default `type` of a DLMS data type

#### dlms.getDate
`dlms.getDate(value)` Extract the *Date* of a *dateTime* object or an *octet-string*.

1. The best-effort `Date` possible

The example:
```
var result = dlms.get()",
var date = dlms.getDate(result[0]);
```

This will return a `Date` Object

#### dlms.getDateTime
`dlms.getDateTime(value)` Extract the dmls *dateTime* object of a *Date* or an *octet-string*.

1. Value containing `Date` value

The example:
```
var dateTime = dlms.getDateTime(new Date('1970-01-01T13:59:22'));
```

This will return a dlms _dateTime_ object

#### dlms.undefinedDateTime
`dlms.undefinedDateTime()` Get _dateTime_ object with all it's fields set to not specified.

The example:
```
var dateTime = dlms.undefinedDateTime();
```

This will return a dlms _dateTime_ object


#### dlms.unspecifiedDateTime
`dlms.unspecifiedDateTime()` Get _dateTime_ object with all it's fields set to not specified.

The example:
```
var dateTime = dlms.unspecifiedDateTime();
```

This will return a dlms _dateTime_ object

### kite
#### kite.requestChangeTerminalStatus
`kite.requestChangeTerminalStatus(status)`

#### kite.requestTerminalDetails
`iec102.requestTerminalDetails()` 
1. Command to be sent
2. Array of expected results. It can be null.
3. Regex to extract values. It can be null.

#### kite.getCredentials
`kite.getCredentials()` Get validation credentials in Kite

#### kite.getUriSuffix
`kite.getUriSuffix(details)` Get uriSuffix depend on sub-entity data icc, imsi or msidn
1. details is not mandatory. Define the kind of variable data in url, by url (not null) or by parameters 

#### kite.addDataToCollection
`kite.addDataToCollection(body, at, source, sourceInfo)` Add necessary data as datapoint to collect
1. body is mandatory.

#### kite.addCgiToCollection
`kite.addCgiToCollection(body, at, source, sourceInfo)` Add CGI necessary data as datapoint to collect
1. body is mandatory.

#### kite.getOgStatus
`kite.getOgStatus(status)`
1. status is mandatory.

#### kite.getRatType
`kite.getRatType(type)` Return ratType depend on numeric type
1. type is mandatory.

#### kite.getKiteStatus
`kite.getKiteStatus(status)`
1. status is mandatory.

#### kite.addLocation
`kite.addLocation(locat, postalCode, source, sourceInfo)` Add location as datapoint to collect
1. locat is mandatory.


### entity

#### entity._value

`entity._value(datastream, index)` Extract from entity specified datastream "value" field value

The function takes 2 parameters:

1. Datastream name, for example: 'provision.device.identifier'
2. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	},
	"value": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.value;} else {return null;}} else {return ds._value._current.value;}} else {return null;}} else {return null;}}
};
var deviceId = entity._value("provision.device.identifier");
```

This will return "device_identifier".

#### entity._at

`entity._at(datastream, index)` Extract from entity specified datastream "at" field value

The function takes 2 parameters:

1. Datastream name, for example: 'provision.device.identifier'
2. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	},
	"at": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.at;} else {return null;}} else {return ds._value._current.at;}} else {return null;}} else {return null;}}
};
var at = entity._at("provision.device.identifier");
```

This will return "2022-03-17T15:17:04.531Z".

#### entity._date

`entity._date(datastream, index)` Extract from entity specified datastream "date" field value

The function takes 2 parameters:

1. Datastream name, for example: 'provision.device.identifier'
2. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	},
	"date": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.date;} else {return null;}} else {return ds._value._current.date;}} else {return null;}} else {return null;}}
};
var date = entity._date("provision.device.identifier");
```

This will return "2022-03-17T15:17:05.531Z".

#### entity._entitySource

`entity._entitySource(datastream, index)` Extract from entity specified datastream "source" field value. "source" field only is not found in datastream wit prefix 'provision'.

The function takes 3 parameters:

1. Object with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
  "device.temperature.value": {
    "_value": {
      "_current": {
        "value": 25.3,
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }
    }
  }
};
var source = entitySource(entity, "device.temperature.value");
```

This will return "DEVICE_OPENGATE_HTTP".

#### entity._entitySourceInfo

`entity._entitySourceInfo(datastream, index)` Extract from entity specified datastream "sourceInfo" field value. "sourceInfo" field only is not found in datastream wit prefix 'provision'.

The function takes 3 parameters:

1. Object with flattened entity. It can be device entity or gateway entity.
2. Datastream name, for example: 'provision.device.identifier'
3. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
  "device.temperature.value": {
    "_value": {
      "_current": {
        "value": 25.3,
        "date": "2017-12-01T08:52:37.64Z",
        "at": "2017-12-01T08:52:37.64Z",
        "source": "DEVICE_OPENGATE_HTTP",
        "sourceInfo": "IoT Data Message Received"
      }
    }
  }
};
var sourceInfo = entitySourceInfo(entity, "device.temperature.value");
```

This will return "IoT Data Message Received".




`entity._date(datastream, index)` Extract from entity specified datastream "date" field value

The function takes 2 parameters:

1. Datastream name, for example: 'provision.device.identifier'
2. Array Index. If provided datastream is an array (i.e. comm module), element index must be specified

The example:

```
var entity = {
	"provision.device.identifier":{
		"_value":{
			"_current":{
				"value": "device_identifier", 
				"at": "2022-03-17T15:17:04.531Z", 
				"date": "2022-03-17T15:17:05.531Z"
			}
		}
	},
	"date": function (datastream, index) {if (this) {var ds = this[datastream];if (ds) {if (index || index === 0) {var elem = ds[index];if (elem) {return elem._value._current.date;} else {return null;}} else {return ds._value._current.date;}} else {return null;}} else {return null;}}
};
var date = entity._date("provision.device.identifier");
```

This will return "2022-03-17T15:17:05.531Z".

### logger

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


### http

#### http.server.response.send

`http.server.response.send()` Creates HTTP Response with defined properties. The response will be sent once the CF is finished correctly. 

The function does not take any parameters.

Example of use:
```javascript
http.server.response.status=200;
http.server.response.body={'msg': 'OK'};
http.server.response.send();
//continue CF execution
return collection;
```
Function returns void.

#### http.client.post

`http.client.post()` Performs POST using defined configuration. Overrides defined `method`. 

The function does not take any parameters.

Example of use:
```javascript
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.post();
```
And the result will be:

```
{
  'statusCode': 201,
  'body': 'some content',
  'headers': {
    'someHeader':'someHeaderValue'
  }
}
```

#### http.client.put

`http.client.put()` Performs PUT using defined configuration. Overrides defined `method`. 

The function does not take any parameters.

Example of use:
```javascript
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.put();
```
And the result will be:

```
{
  'statusCode': 201,
  'body': 'some content',
  'headers': {
    'someHeader':'someHeaderValue'
  }
}
```

#### http.client.patch

`http.client.patch()` Performs PATCH using defined configuration. Overrides defined `method`. 

The function does not take any parameters.

Example of use:
```javascript
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
http.client.body = {'operation_name':'custom_operation_request'};
var httpResp = http.client.patch();
```
And the result will be:

```
{
  'statusCode': 201,
  'body': 'some content',
  'headers': {
    'someHeader':'someHeaderValue'
  }
}
```

#### http.client.get

`http.client.get()` Performs GET using defined configuration. Overrides defined `method`. 

The function does not take any parameters.

Example of use:
```javascript
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
var httpResp = http.client.get();
```
And the result will be:

```
{
  'statusCode': 200,
  'body': 'some content',
  'headers': {
    'someHeader':'someHeaderValue'
  }
}
```

#### http.client.delete

`http.client.delete()` Performs DELETE using defined configuration. Overrides defined `method`. 

The function does not take any parameters.

Example of use:
```javascript
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
var httpResp = http.client.delete();
```
And the result will be:

```
{
  'statusCode': 200,
  'body': 'some content',
  'headers': {
    'someHeader':'someHeaderValue'
  }
}
```

#### http.client.request

`http.client.request()` Performs configured http request. `method` property must be defined. 

The function does not take any parameters.

Example of use:
```javascript
http.client.method = "GET";
http.client.uri = "https://remotehost/uri";
http.client.headers = {'X-Api-Key':'apikey'};
var httpResp = http.client.request();
```
And the result will be:

```
{
  'statusCode': 200,
  'body': 'some content',
  'headers': {
    'someHeader':'someHeaderValue'
  }
}
```

### operation
Is the main object. It allows to do request of the Operation-API.

#### operation.getAllPending
`operation.getAllPending()` Read and return a list with the selected device operations pending (with status WAITING_FOR_CONNECTION) of the user to execute the CFx.
This function does not require parameters.

**Function operation.getAllPending return object, descript like:**
errorMessage: Description of the exception error caught, or error sent by the request. will be `null`, when the request no contains errors.
opResult: { statusCode, List } => Contains *statusCode*, and the *result list* object of the request, when it's OK.

Example of use with default values:
```javascript
var opResult = operation.getAllPending();
```

##### opResult -- Object With optional error and List of Object Functions
**opResult** is a JSON Object containing an optional error and a result object.

The result object inside opResult (returned by the operation.getAllPending function) is a List Object, you can use the next functions:

*activate*: Active the target operation. Return `null` whe the activation is OK, in other case return the cause error message.
*getRequest*: Build and return the request to send of the device to response the operation.

The object -**opResult**- contains all attributes and functions of the response (operation) object.

###### activate() 
**Active the target operation selected with parameters of the context**
(Update to IN_PROGRESS the operation)

This function does not require parameters.

Example of use with default values:
```javascript
var opResult = operation.getAllPending();
if (opResult.error) {
    return error;
}
opResult.result.forEach(op => op.activate());
```

###### getRequest() 
**Build and return the request to send of the device to response the operation**

This function does not require parameters.

### reqRes -- Object properties
**reqRes** is a JSON Object returned by the getRequest function, descript like:

*operation*: The main objet to do the request.

-- Attributes of `operation`
request: The secondary main objet to do the request.

-- Attributes of `request`
name: The attribute to define the operation name target of the request
id: The attribute to define the operation identifier target of the request
parameters: The attribute to define the operation parameters to use in the request
timestamp: The attribute to define the operation time on do the request

*By default*, the attributes will be set with the value of the target operation.

Example getRequest object return
```javascript
{ 
    'operation': {
        'request': {
            'name': 'opName',
            'id': 'ce02792a-3a37-11f0-a48d-52540044ee01',
            'parameters': {},
            'timestamp': Date.now()
        }
    }
}
```                            

Example of use getRequest function:
```javascript
var opResult = operation.getAllPending();
if (opResult.error) {
    return error;
}
opResult.result.forEach(op => {
        op.activate();
        http.client.body = op.getRequest();
        var httpResp = http.client.post();
        if (httpResp.statusCode != 201) {
            throw Error('Unexpected response:', httpResp);
        }
    }
);
```

### crypto
#### crypt.aes.decrypt
`crypt.aes.decrypt(algorithm, key, ivParameterSpec, data)` Decrypt the `data` using the selected AES `algorithm` with the provided shared `key`.

1. algorithm identifier, as required in java `javax.crypto.Cipher.getInstance(algorithm)`
2. key used to decrypt the data.
3. Initialization vector (IV) used in encryption. Only alowed or required depending on the selected algorithm.
4. encoded data to be decrypted

The example:

```
crypt.aes.decrypt("AES/CBC/NoPadding"
    , utils.bytes.fromText('012345678902345a')
    , utils.bytes.fromText('0123456789023452')
    , [178,19,136,33,80,100,25,183,126,178,19,125,139,24,212,253]);
```

#### crypt.aes.encrypt
`crypt.aes.encrypt(algorithm, key, ivParameterSpec, data)` Encrypt the `data` using the selected AES `algorithm` with the provided shared `key`.

1. algorithm identifier, as required in java `javax.crypto.Cipher.getInstance(algorithm)`
2. key used to encrypt the data.
3. Initialization vector (IV) used in encryption. Only alowed or required depending on the selected algorithm.
4. data to be encrypted

The example:

```
crypt.aes.encrypt("AES/CBC/NoPadding"
    , utils.bytes.fromText('012345678902345a')
    , utils.bytes.fromText('0123456789023452')
    , [104, 111, 108, 97, 32, 99, 97, 114, 97, 108, 99, 111, 108, 52, 53, 54]); 
```


#### crypt.hmac.sha256
`crypt.hmac.sha256(data, key)` Hash `data` using provided key with sha256 algorithm.

1. Data to be hashed
2. key to be used.

The example:

```
var hashResult = crypt.hmac.sha256("Some data to be hashed", "hashingKey"); 
```


#### crypt.hmac.sha512
`crypt.hmac.sha512(data, key)` Hash `data` using provided key with sha256 algorithm.

1. Data to be hashed
2. key to be used.

The example:

```
var hashResult = crypt.hmac.sha512("Some data to be hashed", "hashingKey");
```


### coap

#### coap.server.response.send

`coap.server.response.send()` Creates Coap Response with defined properties. The response will be sent once the CF is finished correctly. 

The function does not take any parameters.

Example of use:
```javascript
coap.server.response.status=200;
coap.server.response.body={'msg': 'OK'};
coap.server.response.contentFormat=42;
coap.server.response.send();
//continue CF execution
return collection;
```
Function returns void.