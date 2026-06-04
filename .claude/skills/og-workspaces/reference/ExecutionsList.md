# Reference: `ExecutionsList`

The `ExecutionsList` widget allows displaying executions data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'operations'
* `type`: Always must be 'ExecutionsList'

### Specific configuration in config

* `resource.id`: "onDevices"/"onSubscriptions"/"onSubscribers" tells where the executions are coming from.

* `executionsHistory`: true/false tells if the executions are historical or not.

### Paths in columns config

Columns path (._current.* not allowed) for ExecutionsList when `executionsHistory: false`:

| Field | Description |
|-------|-------------|
| jobId | Job identifier. |
| entityId | Device ID (if "onDevices"), subscription ID (if "onSubscriptions"), or subscriber ID (if "onSubscribers"). |
| resourceType | Type of resource involved in the job or operation. |
| operationId | Identifier of the operation executed within the job. |
| name | Name of the operation. |
| userNotes | User‑provided notes related to the operation. |
| notify | Indicates whether notifications are enabled for the operation. |
| user | User who started the job. |
| date | Date when the operation was executed. |
| status | Current status of the operation. |
| result | Final result of the operation. |

Columns path (._current.* not allowed) for ExecutionsList when `executionsHistory: true`:

| Field | Description |
|-------|-------------|
| jobId | Job identifier. |
| entityId | Device ID (if "onDevices"), subscription ID (if "onSubscriptions"), or subscriber ID (if "onSubscribers"). |
| resourceType | Type of resource involved in the job or operation. |
| operationId | Identifier of the operation executed within the job. |
| name | Name of the operation. |
| notify | Indicates whether notifications are enabled for the operation. |
| date | Date when the operation was executed. |
| status | Current status of the operation. |
| result | Final result of the operation. |


### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Supported fields are the following (when `executionsHistory: false`):

| Field | Description |
|-------|-------------|
| applicationEmail | Email address associated with the application or job request. |
| channelKey | Unique identifier of the communication or processing channel. |
| channelName | Human‑readable name of the channel used for the operation. |
| domainKey | Identifier of the domain or functional area where the job belongs. |
| entityId | Internal ID of the entity being processed. |
| resourceType | Type of resource involved (user, device, service, etc.). |
| entityUk | Unique key of the entity, often used for deduplication. |
| jobActivatedDate | Timestamp when the job became active. |
| jobEntityTotal | Total number of entities included in the job. |
| jobErrorCode | Error code returned if the job fails. |
| jobErrorDescription | Human‑readable explanation of the job error. |
| jobFinishedCancelledByEngine | Jobs cancelled automatically by the engine. |
| jobFinishedCancelledByExternal | Jobs cancelled by an external system. |
| jobFinishedCancelledByExternalTimeout | Jobs cancelled due to an external timeout. |
| jobFinishedCancelledByTimeout | Jobs cancelled because they exceeded the allowed execution time. |
| jobFinishedCancelledByUser | Jobs cancelled manually by a user. |
| jobFinishedCancelledTotal | Total number of cancelled jobs. |
| jobFinishedDate | Timestamp when the job completed. |
| jobFinishedError | Jobs that finished with an error. |
| jobFinishedOotError | Out‑of‑tolerance errors detected during execution. |
| jobFinishedOotSuccessful | Jobs completed successfully but with out‑of‑tolerance warnings. |
| jobFinishedOotTotal | Total out‑of‑tolerance job results. |
| jobFinishedSuccessful | Jobs completed successfully. |
| jobFinishedTotal | Total number of finished jobs. |
| jobId | Unique identifier of the job. |
| jobInProgressPendingExecution | Jobs waiting to be executed. |
| jobInProgressScheduled | Jobs scheduled but not yet started. |
| jobInProgressStarted | Jobs currently running. |
| jobInProgressTotal | Total number of jobs in progress. |
| jobInProgressWaitingForConnection | Jobs paused while waiting for a required connection. |
| jobJson | Raw JSON payload describing the job. |
| jobStartedDate | Timestamp when the job execution began. |
| jobState | Current state of the job. |
| jobStatus | Status indicator summarizing the job outcome. |
| jobTotal | Total number of jobs tracked. |
| jobUser | User who initiated or owns the job. |
| jobUserNotes | User‑provided notes or comments about the job. |
| notDeletable | Indicates whether the item cannot be deleted. |
| operationAttemptCurrent | Current retry attempt number. |
| operationAttemptMax | Maximum allowed retry attempts. |
| operationDate | Date when the operation was executed. |
| operationDetails | Additional descriptive information about the operation. |
| operationEndDate | Timestamp when the operation ended. |
| operationId | Unique identifier of the operation. |
| operationInitDate | Timestamp when the operation was initialized. |
| operationJson | JSON payload describing the operation. |
| operationName | Human‑readable name of the operation. |
| operationNotify | Whether notifications should be sent for this operation. |
| operationResult | Final result of the operation. |
| operationScheduleTimestamp | Scheduled execution time. |
| operationStatus | Current status of the operation. |
| operationStatusKey | Internal key representing the operation status. |
| operationStepDescription | Description of the current step. |
| operationStepName | Name of the operation step. |
| operationStepResponse | Response returned by the step. |
| operationStepResult | Result of the step execution. |
| operationStepTimestamp | Timestamp of the step execution. |
| operationTimeoutTimestamp | Time when the operation will be considered timed out. |
| operationUserNotes | Notes added by the user regarding the operation. |
| organizationKey | Unique identifier of the organization. |
| organizationName | Name of the organization. |
| profileKey | Identifier of the user or system profile. |
| provAdministrativeState | Administrative provisioning state. |
| provAdministrativeStateKey | Key representing the provisioning administrative state. |
| provDefaultFeed | Default provisioning feed or source. |
| serviceGroupName | Name of the service group associated with the operation. |
| tag | Generic tag used for classification or filtering. |
| taskEmpty_null | Indicates an empty or null task placeholder. |
| taskId | Unique identifier of the task. |
| taskKey | Internal key representing the task. |
| taskName | Human‑readable name of the task. |
| taskState | Current state of the task. |
| taskStatus | Status indicator summarizing the task outcome. |
| workgroupKey | Identifier of the workgroup. |
| workgroupName | Name of the workgroup. |

Supported fields are the following (when `executionsHistory: true`):

| Field | Description |
|-------|-------------|
| applicationEmail | Email address associated with the application or job request. |
| entityId | Internal ID of the entity being processed. |
| resourceType | Type of resource involved (user, device, service, etc.). |
| jobActivatedDate | Timestamp when the job became active. |
| jobID | Unique identifier of the job. |
| jobUserNotes | User‑provided notes or comments about the job. |
| operationAttemptCurrent | Current retry attempt number. |
| operationAttemptMax | Maximum allowed retry attempts. |
| operationDate | Date when the operation was executed. |
| operationDetails | Additional descriptive information about the operation. |
| operationEndDate | Timestamp when the operation ended. |
| operationId | Unique identifier of the operation. |
| operationInitDate | Timestamp when the operation was initialized. |
| operationName | Human‑readable name of the operation. |
| operationNotify | Whether notifications should be sent for this operation. |
| operationParameters | Parameters provided for the operation. |
| operationResult | Final result of the operation. |
| operationStatus | Current status of the operation. |
| operationSteps | Steps involved in the operation execution. |   