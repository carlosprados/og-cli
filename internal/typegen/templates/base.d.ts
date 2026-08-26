// ─────────────────────────────────────────────────────────────────────────────
// The types the documentation does not describe.
//
// Everything the OpenGate documentation states — some 450 functions, methods
// and properties — is generated into the *.generated.d.ts files beside this one
// by tools/ogdocgen. What the documentation does NOT describe is the shape of a
// datastream value and the small enumerations, so those are written here by
// hand, and the overrides table in the generator points the relevant parameters
// at them.
//
// Hand-written, on purpose. Keep it small: anything the documentation covers
// belongs in the generator, not here.
// ─────────────────────────────────────────────────────────────────────────────

/** One observation of a datastream. */
interface OGSample<T = any> {
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

interface OGDatastreamValue<T = any> {
  /** Values arriving in the message that triggered the artifact. Monitoring
   *  datastreams only — absent on provision datastreams. */
  _received?: Array<OGSample<T>>;
  _current: OGSample<T>;
  _previous?: OGSample<T>;
}

/** A plain (non-indexed) datastream. */
interface OGDatastream<T = any> {
  _value: OGDatastreamValue<T>;
}

/** One element of an indexed datastream — a path containing `[]`, such as
 *  `provision.device.communicationModules[].identifier`. The entity holds an
 *  ARRAY of these, not a single object. */
interface OGIndexedDatastream<T = any> {
  _index: { path: string; value: { _current: OGSample<any> } };
  _value: OGDatastreamValue<T>;
}

/** Alarm severity. Anything else is rejected — severity: 'HIGH' is a real
 *  mistake that a plain string type would let through. */
type OGSeverity = 'INFORMATIVE' | 'URGENT' | 'CRITICAL';

/** Alarm priority. */
type OGPriority = 'LOW' | 'MEDIUM' | 'HIGH';

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

/** Execution context of a connector function. Which fields are present depends
 *  on how the function was reached: `path` for an HTTP resource or WebSocket,
 *  `topic` for MQTT. */
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
  [key: string]: any;
}

/** The incoming payload: a JSON object, flat text or binary content, depending
 *  on the artifact's payloadType. */
declare const payload: any;
declare const contextParams: OGContextParams;

/** Name of the artifact being executed. */
declare const ruleName: string;
