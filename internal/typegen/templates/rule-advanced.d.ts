// ─────────────────────────────────────────────────────────────────────────────
// Platform globals for an ADVANCED automation rule.
//
// Hand-maintained from the official platform guide vendored at
// .claude/skills/og-device-ops/rules-js-reference.md. Update both together.
// ─────────────────────────────────────────────────────────────────────────────

/** One observation of a datastream. */
interface OGSample<T = unknown> {
  value: T;
  /** ISO-8601 instant the platform recorded the value. */
  date: string;
  /** ISO-8601 instant the value refers to. */
  at: string;
  source?: string;
  sourceInfo?: string;
  /** Present on provision datastreams: MONITORING, REFERENCE, ... */
  provType?: string;
}

interface OGDatastreamValue<T = unknown> {
  /** Values arriving in the message that triggered the rule. Monitoring
   *  datastreams only — absent on provision datastreams. */
  _received?: Array<OGSample<T>>;
  _current: OGSample<T>;
  _previous?: OGSample<T>;
}

/** A plain (non-indexed) datastream. */
interface OGDatastream<T = unknown> {
  _value: OGDatastreamValue<T>;
}

/** One element of an indexed datastream — a path containing `[]`, such as
 *  `provision.device.communicationModules[].identifier`. Note the entity holds
 *  an ARRAY of these, not a single object. */
interface OGIndexedDatastream<T = unknown> {
  _index: { path: string; value: { _current: OGSample<unknown> } };
  _value: OGDatastreamValue<T>;
}

type OGSeverity = 'INFORMATIVE' | 'URGENT' | 'CRITICAL';
type OGPriority = 'LOW' | 'MEDIUM' | 'HIGH';

// ── Actions: alarms ──────────────────────────────────────────────────────────

interface OGAlarmConfig {
  /** Device, subscription or subscriber identifier. Defaults to
   *  provision.administration.identifier. */
  subEntityIdentifier?: string;
  alarmName?: string;
  ruleName?: string;
  /** Default: 'INFORMATIVE'. */
  severity?: OGSeverity;
  /** Default: 'LOW'. */
  priority?: OGPriority;
  description?: string;
}

declare const alarm: {
  open(alarmConfig: OGAlarmConfig): void;
  closeByRuleName(config: { subEntityIdentifier?: string; ruleName: string }): void;
  closeByAlarmName(config: { subEntityIdentifier?: string; alarmName: string }): void;
};

/** Deprecated in favour of alarm.open. */
declare function openAlarm(
  subEntityIdentifier: string | undefined,
  alarmName: string,
  ruleName: string,
  severity: OGSeverity,
  priority: OGPriority,
  alarmDescription: string,
): void;
declare function openAlarmWithExtraInfo(
  subEntityIdentifier: string | undefined,
  alarmName: string,
  ruleName: string,
  severity: OGSeverity,
  priority: OGPriority,
  alarmDescription: string,
  extra_info: unknown,
): void;
declare function closeAlarmByRuleName(entityIdDatastream: string, ruleName: string): void;
declare function closeAlarmByAlarmName(entityIdDatastream: string, alarmName: string): void;

// ── Actions: notifications ───────────────────────────────────────────────────

declare const notification: {
  addEmailNotification(emailConfig: Record<string, unknown>): void;
  addTrapNotification(trapConfig: Record<string, unknown>): void;
  sendHttp(httpConfig: Record<string, unknown>): unknown;
};

declare function addEmailNotification(
  recipients: string | string[],
  notificationName: string,
  notificationBody: string,
  ruleName: string,
  mailParameters?: Record<string, unknown>,
): void;
declare function addTrapNotification(
  recipients: string | string[],
  variables: unknown,
  notificationName: string,
  trapOID: string,
  enterpriseOID: string,
  ruleName: string,
): void;
declare function sendHttp(httpJson: Record<string, unknown>): unknown;

// ── Actions: operations ──────────────────────────────────────────────────────

declare const operation: {
  execute(operationConfig: Record<string, unknown>): unknown;
};

declare function executeOperation(
  subEntityIdentifier: string | undefined,
  operationType: string,
  operationTimeout: number,
  jobUser: string,
  retries: number,
  ackTimeout: number,
  retriesDelay: number,
  stopValue: unknown,
  stopMode: string,
  parameters: unknown,
  callback?: unknown,
): unknown;

// ── Collection ───────────────────────────────────────────────────────────────

declare const collect: {
  datapoints(datapointConfig: Record<string, unknown>): void;
  datastreams(datastreamConfig: Record<string, unknown>): void;
};

declare function collectDataPoints(
  resourceType: string,
  identifier: string,
  mapValues: unknown,
  commsValues?: unknown,
  ttl?: number,
): void;
declare function collectDatastreams(
  jsMapValues: unknown,
  jsCommsValues?: unknown,
  directMode?: boolean,
  reinject?: boolean,
): void;

// ── Provision ────────────────────────────────────────────────────────────────

declare const provision: {
  datastreams(provisionConfig: Record<string, unknown>): void;
};

// ── Logging ──────────────────────────────────────────────────────────────────

declare const logger: {
  trace(msg: unknown): void;
  debug(msg: unknown): void;
  info(msg: unknown): void;
  warn(msg: unknown): void;
  error(msg: unknown): void;
};

declare function printLog(log: unknown): void;

// ── Utils ────────────────────────────────────────────────────────────────────

declare const utils: {
  cancelDelay(ruleName: string): void;
  encryptString(encryptConfig: Record<string, unknown>): string;
  decryptString(decryptConfig: Record<string, unknown>): string;
  getDatastreamByIdFromDB(datastreamId: OGDatastreamID, subEntityIdentifier?: string): OGDatastream | undefined;
};

declare function cancelDelay(ruleName: string): void;
declare function cancelJobGroup(ruleName: string): void;
declare function encryptString(originalValue: string, datastreamId: string, organization: string): string;
declare function decryptString(encryptedValue: string, datastreamId: string, organization: string): string;
declare function arrayEquals(array1: unknown[], array2: unknown[]): boolean;

// ── Location ─────────────────────────────────────────────────────────────────

declare const location: {
  getAreas(coordinates: unknown): unknown;
  getSortedAreas(coordinates: unknown, sorted: boolean): unknown;
  getDistance(lat1: number, long1: number, lat2: number, long2: number): number;
};

declare function getAreas(coordinates: unknown): unknown;
declare function getSortedAreas(coordinates: unknown, sorted: boolean): unknown;
declare function getDistance(lat1: number, long1: number, lat2: number, long2: number): number;

// ── HTTP client ──────────────────────────────────────────────────────────────

declare const http: {
  client: {
    request(config?: Record<string, unknown>): unknown;
    get(config?: Record<string, unknown>): unknown;
    post(config?: Record<string, unknown>): unknown;
    put(config?: Record<string, unknown>): unknown;
    patch(config?: Record<string, unknown>): unknown;
    delete(config?: Record<string, unknown>): unknown;
  };
};

// ── Dates ────────────────────────────────────────────────────────────────────

declare function toDate(localDateTime: string): Date;
declare function getDailyResetDate(): Date;
declare function getDailyResetDateWithZuluHour(zuluHour: number): Date;
declare function getMonthlyResetDate(): Date;
declare function getMonthlyResetDateWithZuluHour(zuluHour: number): Date;
declare function getMonthlyResetDateWithZuluHourAndDayOfMonth(zuluHour: number, dayOfMonth: number): Date;

// ── Values and datastreams ───────────────────────────────────────────────────

/** Passes the value through, substituting '' when it is undefined.
 *
 *  Typed as returning T rather than `T | ''` on purpose: the honest union makes
 *  every arithmetic use of a rule parameter an error, which would put correct
 *  code in red and get these typings deleted. Guard for '' if the parameter can
 *  be absent. */
declare function getVariableValue<T>(variable: T): T;

/** The complete datastream of the received entity. */
declare function getDatastreamFromEntity(datastreamId: OGDatastreamID): OGDatastream | undefined;
declare function getDatastreamValueFromEntity(datastreamObject: unknown): unknown;
declare function getCommsDatastreamFromEntity(id: string, commsId: string): unknown;
declare function getDatastreamValueCmmsModuleFromEntity(datastreamObject: unknown, position: number): unknown;
declare function getDatastreamByIdFromDB(datastreamId: OGDatastreamID, subEntityIdentifier?: string): OGDatastream | undefined;

// ── Counters ─────────────────────────────────────────────────────────────────

declare function getCounterValue(datastreamValue: unknown, incValue: number, resetDate: Date): number;
declare function getCounterValueWithReset(datastreamValue: unknown, incValue: number, resetDate: Date): number;
declare function getDailyCounterValueFromDB(datastream: unknown, incValue: number, subEntityIdentifier?: string): number;
declare function getDailyCounterValueFromMessage(datastream: unknown, incValue: number): number;
declare function getDailyCounterValueFromMessageWithReset(datastream: unknown, incValue: number): number;
declare function getMonthlyCounterValueFromDB(datastreamId: OGDatastreamID, incValue: number, subEntityIdentifier?: string): number;
declare function getMonthlyCounterValueFromMessage(datastream: unknown, incValue: number): number;
declare function getMonthlyCounterValueFromMessageWithReset(datastream: unknown, incValue: number): number;

// ── Action kind ──────────────────────────────────────────────────────────────

declare function isInsertAction(): boolean;
declare function isUpdateAction(): boolean;
declare function isPatchAction(): boolean;

/** Name of the rule being evaluated. */
declare const ruleName: string;
