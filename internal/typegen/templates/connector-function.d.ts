// ─────────────────────────────────────────────────────────────────────────────
// Platform globals for a connector function.
//
// Hand-maintained from the official platform guide vendored at
// .claude/skills/og-device-ops/connector-functions-js-reference.md. Update both
// together.
//
// The three connector function types share these globals. `response` applies to
// a RESPONSE function and `collection` to a COLLECTION one; both are declared
// everywhere rather than per type, because declaring too little is what makes
// typings redden working code, and a wrong one is caught by the platform's own
// output contract.
// ─────────────────────────────────────────────────────────────────────────────

/** One observation of a datastream. */
interface OGSample<T = unknown> {
  value: T;
  date: string;
  at: string;
  source?: string;
  sourceInfo?: string;
  /** Present on provision datastreams: MONITORING, REFERENCE, ... */
  provType?: string;
}

interface OGDatastreamValue<T = unknown> {
  _current: OGSample<T>;
  _previous?: OGSample<T>;
}

/** A plain (non-indexed) datastream. */
interface OGDatastream<T = unknown> {
  _value: OGDatastreamValue<T>;
}

/** One element of an indexed datastream — a path containing `[]`. The entity
 *  holds an ARRAY of these, not a single object. */
interface OGIndexedDatastream<T = unknown> {
  _index: { path: string; value: { _current: OGSample<unknown> } };
  _value: OGDatastreamValue<T>;
}

/** Execution context. Which fields are present depends on how the function was
 *  reached: `path` for an HTTP resource or WebSocket, `topic` for MQTT. */
interface OGContextParams {
  /** Device or user API key. */
  apiKey?: string;
  /** Remote host, when an HTTP REST resource was invoked. */
  remoteIp?: string;
  /** Complete URI of the WebSocket or HTTP resource. */
  uri?: string;
  /** Relative path — this is what south criteria filter on. */
  path?: string;
  /** MQTT topic the message arrived on. */
  topic?: string;
  [key: string]: unknown;
}

/** The incoming payload: a JSON object, flat text or binary content, depending
 *  on the function's payloadType. */
declare const payload: any;
declare const contextParams: OGContextParams;

// ── Plain functions ─────────────────────────────────────────────────────────
//
// Documented under "Plain functions" in the platform guide. Production code uses
// these heavily — log() and httpRequest() appear in live connector functions —
// and an earlier version of these declarations omitted them, reporting
// "Cannot find name" on working code.

/** Extract a datastream's value. index is required for an indexed datastream
 *  (a path containing `[]`), such as a communication module. */
declare function entityValue(entity: unknown, datastream: OGDatastreamID, index?: number): unknown;
declare function entityAt(entity: unknown, datastream: OGDatastreamID, index?: number): unknown;
declare function entityDate(entity: unknown, datastream: OGDatastreamID, index?: number): unknown;
declare function entitiesValue(entities: unknown, datastream: OGDatastreamID, index?: number): unknown;

/** Log at the default level, concatenating its arguments. */
declare function log(...msg: unknown[]): void;

/** Perform an HTTP request from inside the function. */
declare function httpRequest(request: unknown, payload?: unknown): any;

declare function webSocketMsg(payload: unknown, deviceId?: string): unknown;
declare function base64ToByteArray(base64String: string): unknown;

declare function encryptString(originalValue: string, datastreamId: string, organization: string): string;
declare function decryptString(encryptedValue: string, datastreamId: string, organization: string): string;

// Operation accessors, for a REQUEST or RESPONSE function's payload.operation.
declare function operationId(operationObj: unknown): string;
declare function operationName(operationObj: unknown): string;
declare function operationDeviceId(operationObj: unknown): string;
declare function operationParameters(operationObj: unknown): any;
declare function operationTimestamp(operationObj: unknown): unknown;

// ── Concatenation helpers ───────────────────────────────────────────────────
//
// The same concatenation is reachable two ways: these plain helpers and the cf
// object below. Both are documented and both appear in production code.

declare function responseCF(payload: unknown, responseFunctionCriteria?: string): void;
declare function collectionCF(payload: unknown, collectionFunctionCriteria?: string): void;

// ── Standard object builders ────────────────────────────────────────────────
//
// A RESPONSE function must return an OpenGate Standard Response and a COLLECTION
// function a Standard IoT Data Collection; these compose them.

declare function ogResponse(...args: unknown[]): any;
declare function ogStep(...args: unknown[]): any;
declare function ogStepResponse(...args: unknown[]): any;
declare function ogCollection(...args: unknown[]): any;
declare function ogCollectionDs(...args: unknown[]): any;
declare function ogCollectionDp(...args: unknown[]): any;
declare function addOgCollectionDp(...args: unknown[]): any;

// ── Output: RESPONSE connector functions ────────────────────────────────────

/** Build the OpenGate Standard Response a RESPONSE function must return. */
declare const response: {
  /** result is a platform result code, e.g. 'SUCCESSFUL', 'ERROR_PROCESSING'. */
  addStep(name: string, result: string, description?: string, stepResponseList?: unknown): void;
  /** Send the added steps and clear them from the response object. */
  sendSteps(): void;
  successful(description?: string): void;
  errorProcessing(description?: string): void;
  errorInParam(description?: string): void;
  errorTimeout(description?: string): void;
  notSupported(description?: string): void;
  unknownResult(description?: string): void;
};

// ── Output: COLLECTION connector functions ──────────────────────────────────

/** Build the OpenGate Standard IoT Data Collection a COLLECTION function must
 *  return. */
declare const collection: {
  addDatapoint(datastreamId: OGDatastreamID, value: unknown, at?: number | string, source?: string, sourceInfo?: string): void;
  setFeed(datastreamId: OGDatastreamID, feed: string): void;
  /** Send the added datapoints and clear them from the collection object. */
  send(): void;
  getDatastream(datastream: OGDatastreamID): unknown;
  getValue(datastream: OGDatastreamID, dpIndex?: number): unknown;
  clone(): unknown;
};

// ── Concatenation ───────────────────────────────────────────────────────────

/** Invoke another connector function once this one finishes.
 *
 *  The platform restricts which concatenations run: REQUEST may invoke RESPONSE
 *  and COLLECTION, RESPONSE may invoke COLLECTION, and COLLECTION may invoke
 *  nothing — any other call is silently IGNORED, with no error anywhere. */
declare const cf: {
  response(responseFunctionCriteria: string, responsePayload?: unknown): void;
  collection(responseFunctionCriteria: string, responsePayload?: unknown): void;
};

// ── Operations ──────────────────────────────────────────────────────────────

declare const operation: {
  getAllPending(): unknown;
};

// ── Logging ─────────────────────────────────────────────────────────────────

/** Every level concatenates its arguments into one message, so all of them are
 *  variadic — production code calls logger.debug('payload is: ', payload). */
declare const logger: {
  trace(...msg: unknown[]): void;
  debug(...msg: unknown[]): void;
  info(...msg: unknown[]): void;
  warn(...msg: unknown[]): void;
  error(...msg: unknown[]): void;
};

// ── Utils ───────────────────────────────────────────────────────────────────

declare const utils: {
  atcmd: { toDBm(value: unknown): unknown };
  endesa: {
    torscp(value: unknown): unknown;
    toDbmPlusQuality(value: unknown): unknown;
    prepareMsisdn(value: unknown): unknown;
    commandFrom(value: unknown): unknown;
  };
  odm: {
    addValueToContext(key: string, value: unknown): void;
    addIdentificationHint(key: string, value: unknown, structured?: unknown): void;
    sleep(time: number): void;
  };
  [key: string]: unknown;
};

// ── Cryptography ────────────────────────────────────────────────────────────

declare const crypt: {
  aes: {
    encrypt(algorithm: string, key: string, ivParameterSpec: string, data: string): string;
    decrypt(algorithm: string, key: string, ivParameterSpec: string, data: string): string;
  };
  hmac: {
    sha256(data: string, key: string): string;
    sha512(data: string, key: string): string;
  };
};
