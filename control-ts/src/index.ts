/**
 * Retired experimental control-plane package.
 *
 * This package intentionally exports no state mutation, orchestration, or
 * adapter API. Production Aether execution is owned by the Go runtime, with
 * `.aether/ts-host` acting only as a manifest/wave conductor.
 */
export const AETHER_CONTROL_VERSION = "0.1.0";
export const AETHER_CONTROL_STATUS = "retired" as const;
export { RETIRED_CONTROL_PLANE_MESSAGE } from "./retired.js";
