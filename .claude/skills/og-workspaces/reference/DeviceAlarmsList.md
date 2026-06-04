# Reference: `DeviceAlarmsList`

The `DeviceAlarmsList` widget allows displaying alarms data in a table format with several features like  filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'alarms'
* `type`: Always must be 'DeviceAlarmsList'

### Paths in columns config

Columns path (._current.* not allowed) for DeviceAlarmsList:
* **identifier** Alarm Identifier
* **channel** Alarm Channel
* **organization** Alarm Organization
* **name** Alarm Name
* **description** Alarm Description
* **rule** Alarm Rule
* **entityIdentifier** Alarm Entity Identifier
* **resourceType** Entity Resource Type
* **subEntityIdentifier** Alarm Sub Entity Identifier
* **status** Alarm Status
* **severity** Alarm Severity
* **priority** Alarm Priority
* **openingDate** Alarm Opening Date
* **attentionDate** Alarm Attention Date
* **attentionUser** Alarm Attention User
* **attentionNote** Alarm Attention Note
* **closureDate** Alarm Closure Date
* **closureUser** Alarm Closure User
* **closureNote** Alarm Closure Note
* **extra_info** Alarm Extra Info

### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Supported fields are the following:

| filter field (used to build the filter) | equivalent alarm path (used to get the value for columns) |
|---|---|
| alarm.identifier | identifier | 
| alarm.channel | channel | 
| alarm.organization | organization | 
| alarm.name | name | 
| alarm.description | description | 
| alarm.rule | rule | 
| alarm.entityIdentifier | entityIdentifier | 
| alarm.resourceType | resourceType | 
| alarm.subEntityIdentifier | subEntityIdentifier | 
| alarm.status | status | 
| alarm.severity | severity | 
| alarm.priority | priority | 
| alarm.openingDate | openingDate | 
| alarm.attentionDate | attentionDate | 
| alarm.attentionUser | attentionUser | 
| alarm.attentionNote | attentionNote | 
| alarm.closureDate | closureDate | 
| alarm.closureUser | closureUser | 
| alarm.closureNote | closureNote | 
| alarm.extra_info | extra_info | 

### Window filter

This widget supports window filter, see [Window Filter](./commonFields.md#Window-filter).