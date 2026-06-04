# Reference: `OperationsList`

The `OperationsList` widget allows displaying operations/jobs data in a table format with several features like filtering and pagination.

You can see in [ListsCommons.md] the common configuration for all lists.

You have to consider this especial configuration in config:

* `Ftype`: Always must be 'jobs'
* `type`: Always must be 'OperationsList'

### Paths in columns config

| Field | Description |
|-------|-------------|
| report.execution.activatedDate | Date when the job was activated. |
| report.execution.finishedDate | Date when the job finished. |
| report.execution.startedDate | Date when the job execution began. |
| report.summary.finished.total | Total number of finished jobs. |
| report.summary.finished.error | Number of jobs that finished with an error. |
| report.summary.finished.successful | Number of jobs completed successfully. |
| report.summary.inProgress.total | Total number of jobs currently in progress. |
| report.summary.finishedOutOfTime.error | Out‑of‑time errors detected during execution. |
| report.summary.finishedOutOfTime.successful | Jobs completed successfully but with out‑of‑time warnings. |
| report.summary.finishedOutOfTime.total | Total out‑of‑time job results. |
| report.summary.finished.cancelled.total | Total number of cancelled jobs. |
| report.summary.total | Total number of jobs tracked. |
| request.user | User who initiated or owns the job. |
| report.summary.status | Status indicator summarizing the job outcome. |
| request.active | Current state of the job. |
| task_id | Unique identifier of the task. |
| request.name | Name of the operation. |
| id | Unique identifier of the job. |

### Filter configuration

See [Filter field configuration](./commonFields.md#Filter-field-configuration).

Supported fields are the following:

| Field | Description |
|-------|-------------|
| jobActivatedDate | Date when the job was activated. |
| jobActivatedTime | Time when the job was activated. |
| jobCallback | Callback information or URL associated with the job. |
| jobEntityTotal | Total number of entities included in the job. |
| jobErrorCode | Error code returned if the job fails. |
| jobErrorDescription | Human‑readable explanation of the job error. |
| jobFinishedCancelledByEngine | Jobs cancelled automatically by the engine. |
| jobFinishedCancelledByExternal | Jobs cancelled by an external system. |
| jobFinishedCancelledByExternalTimeout | Jobs cancelled due to an external timeout. |
| jobFinishedCancelledByTimeout | Jobs cancelled because they exceeded the allowed execution time. |
| jobFinishedCancelledByUser | Jobs cancelled manually by a user. |
| jobFinishedCancelledTotal | Total number of cancelled jobs. |
| jobFinishedDate | Date when the job finished. |
| jobFinishedError | Jobs that finished with an error. |
| jobFinishedOotError | Out‑of‑time errors detected during execution. |
| jobFinishedOotSuccessful | Jobs completed successfully but with out‑of‑time warnings. |
| jobFinishedOotTotal | Total out‑of‑time job results. |
| jobFinishedSuccessful | Jobs completed successfully. |
| jobFinishedTime | Time when the job finished. |
| jobFinishedTotal | Total number of finished jobs. |
| jobId | Unique identifier of the job. |
| jobInProgressPendingExecution | Jobs waiting to be executed. |
| jobInProgressScheduled | Jobs scheduled but not yet started. |
| jobInProgressStarted | Jobs currently running. |
| jobInProgressTotal | Total number of jobs in progress. |
| jobInProgressWaitingForConnection | Jobs paused while waiting for a required connection. |
| jobJson | Raw JSON payload describing the job. |
| jobOperationName | Name of the operation associated with the job. |
| jobScheduledTime | Time when the job was scheduled. |
| jobStartedDate | Date when the job execution began. |
| jobState | Current state of the job. |
| jobStatus | Status indicator summarizing the job outcome. |
| jobTotal | Total number of jobs tracked. |
| jobTrap | Trap or error‑handling flag associated with the job. |
| jobUser | User who initiated or owns the job. |
| jobUserNotes | User‑provided notes or comments about the job. |
| jobVisible | Indicates whether the job is visible to users. |
| operationId | Identifier of the operation executed within the job. |
| operationName | Name of the operation. |
| operationResult | Final result of the operation. |
| operationStatusKey | Internal key representing the operation status. |
| taskCoremessage | Core message or internal message for the task. |
| taskId | Unique identifier of the task. |
| taskKey | Internal key representing the task. |
| taskName | Human‑readable name of the task. |
| taskNextExecution | Timestamp of the next scheduled execution of the task. |
| taskState | Current state of the task. |
| taskStatus | Status indicator summarizing the task outcome. |
