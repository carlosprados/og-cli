
// ── Protocol objects ────────────────────────────────────────────────────────
//
// Each connector protocol injects its own object. Which ones are actually in
// scope depends on the protocol the function is reached through — the scheme of
// its south criteria (`mqtts://`, `https://`, `dlms://`, …). They are all
// declared when the protocol cannot be determined, because declaring too few
// would flag working code.

declare const http: {
  client: {
    request(config?: Record<string, unknown>): unknown;
    get(config?: Record<string, unknown>): unknown;
    post(config?: Record<string, unknown>): unknown;
    put(config?: Record<string, unknown>): unknown;
    patch(config?: Record<string, unknown>): unknown;
    delete(config?: Record<string, unknown>): unknown;
  };
  server: { response: { send(): void } };
};

declare const mqtt: {
  publish(message: unknown): void;
  /** Assignable: production code sets mqtt.topic before publishing. */
  topic: string;
};

declare const websocket: {
  sendMsg(): void;
};

declare const coap: {
  server: { response: { send(): void } };
};

declare const snmp: {
  addOid(oid: string, type: string, value?: unknown): void;
  get(): unknown;
  set(): unknown;
};

declare const icmp: {
  send(): unknown;
};

declare const ssh: {
  connect(waitFor?: number): unknown;
  send(command: string, pattern?: string, waitFor?: number): unknown;
  disconnect(): void;
};

declare const telnet: {
  connect(waitFor?: number): unknown;
  send(command: string, pattern?: string, waitFor?: number): unknown;
  disconnect(): void;
};

declare const dlms: {
  connect(): unknown;
  disconnect(): void;
  get(descriptive?: unknown): unknown;
  set(descriptive?: unknown): unknown;
  method(descriptive?: unknown): unknown;
  addAttr(classId: number, obisCode: string, attrId: number, type: string, value?: unknown): void;
  addMethod(classId: number, obisCode: string, methodId: number, type: string, value?: unknown): void;
  getCompactData(typeDescription: unknown, value: unknown, descriptive?: unknown, italianMode?: boolean): unknown;
  getDate(value: unknown): unknown;
  getDateTime(value: unknown): unknown;
  undefinedDateTime(): unknown;
  unspecifiedDateTime(): unknown;
};

declare const iec102: {
  connect(registerType?: unknown, waitFor?: number): unknown;
  connectWithEndpoints(endpoints: unknown): unknown;
  connectWithIpAndPorts(ports: unknown): unknown;
  disconnect(): void;
  send(command: unknown, waitFor?: number, pattern?: string): unknown;
  requestTerminalDetails(): unknown;
  asdus: {
    add(name: string, execConfig?: unknown): void;
    addFromParams(): void;
    configuration(execConfig?: unknown): void;
    currentPricing(execConfig?: unknown): void;
    storedPricing(execConfig?: unknown): void;
    dayLightSavingTime(execConfig?: unknown): void;
    deviceManufacturer(execConfig?: unknown): void;
    loadCurve(execConfig?: unknown): void;
    loadCurveQuarter(execConfig?: unknown): void;
    loadCurveIncremental(execConfig?: unknown): void;
    loadCurveIncrementalQuarter(execConfig?: unknown): void;
    login(execConfig?: unknown): void;
    logout(execConfig?: unknown): void;
    parameters(execConfig?: unknown): void;
    timeRequest(execConfig?: unknown): void;
    execute(): unknown;
  };
};

declare const kite: {
  getCredentials(): unknown;
  getUriSuffix(details?: unknown): string;
  getKiteStatus(status: unknown): unknown;
  getOgStatus(status: unknown): unknown;
  getRatType(type: unknown): unknown;
  addDataToCollection(body: unknown, at?: number | string, source?: string, sourceInfo?: string): void;
  addCgiToCollection(body: unknown, at?: number | string, source?: string, sourceInfo?: string): void;
  addLocation(locat: unknown, postalCode?: unknown, source?: string, sourceInfo?: string): void;
  requestChangeTerminalStatus(status: unknown): unknown;
};
