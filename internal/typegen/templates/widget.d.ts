// ─────────────────────────────────────────────────────────────────────────────
// The globals a widget's JavaScript runs with.
//
// Hand-written, and unlike every other context there is no generated catalogue
// beside it: the data API a widget uses is `$api`, which is the `opengate-js`
// package, and that package publishes its own TypeScript declarations from
// version 16.0.0 onwards (246 of them, emitted from its JSDoc by tsc). An editor
// consumes those natively, so generating a second, lossier copy from the
// documentation pages would be strictly worse — the pages were built from an
// unmerged branch and describe classes the release does not contain.
//
// So this file only names what the platform injects and points `$api` at the
// library's own exported class. `og typegen` writes an import that resolves
// through node_modules, which is why the artifact directory gets a package.json
// declaring the dependency.
// ─────────────────────────────────────────────────────────────────────────────

import OpenGateAPI from 'opengate-js';

declare global {
  /** The OpenGate data API — the `opengate-js` client, already authenticated as
   *  the viewing user.
   *
   *  Use the search builders with an explicit `.filter({...})`. The `with*()`
   *  convenience shortcuts emit outdated filter field names (`datapoint.device`)
   *  that current platforms reject with HTTP 400 "Field in filter unknown". */
  const $api: OpenGateAPI;

  /** The viewing user. */
  const $user: {
    email: string;
    workgroup: string;
    domain: string;
    profile: string;
    langCode: string;
    timezone: string;
    [key: string]: any;
  };

  /** moment.js, as the platform bundles it. */
  const $moment: any;

  /** Nuxt useFetch wrapper. Do NOT use it for north endpoints — it answers 403
   *  there. `$api` is the data API. */
  const http: any;

  /** Navigate to another dashboard. */
  function openDashboard(dashboardId: string, params?: any): void;
  /** Navigate to an entity's dashboard. */
  function openEntityDashboard(entityId: string, params?: any): void;
}

export {};
