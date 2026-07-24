/**
 * Event bridge: consume Go ceremony events via JSONL replay + streaming.
 *
 * The bridge calls `aether event-bus-replay` for historical events, then spawns
 * `aether event-bus-subscribe --stream` to tail new events as NDJSON.
 *
 * Boundary contract: this module NEVER writes to `.aether/data/`. It only reads
 * from the JSONL stream. Any write-mode attempt on a Go-owned path throws
 * BoundaryViolationError.
 */
import { type GoBridgeOptions } from "./go-bridge.js";
import type { CeremonyEvent } from "./types.js";
import { BoundaryViolationError } from "./boundary-reference.js";
export interface EventBridgeOptions extends GoBridgeOptions {
    /** Topic filter (supports trailing * wildcard). Default: "ceremony.*" */
    filter?: string;
    /** Polling interval for the stream in milliseconds. Default: 250 */
    pollIntervalMs?: number;
    /** Replay events since this ISO-8601 timestamp. Default: "" (all) */
    replaySince?: string;
    /** Called for every validated ceremony event. */
    onEvent: (event: CeremonyEvent) => void;
}
export interface EventBridgeController {
    /** Stop the stream and clean up resources. */
    stop(): Promise<void>;
}
/**
 * Start the event bridge: replay historical events, then stream new ones.
 *
 * @returns A controller with a `stop()` method.
 */
export declare function startEventBridge(opts: EventBridgeOptions): Promise<EventBridgeController>;
/**
 * Convenience wrapper to stop an event bridge controller.
 */
export declare function stopEventBridge(controller: EventBridgeController): Promise<void>;
export { BoundaryViolationError };
