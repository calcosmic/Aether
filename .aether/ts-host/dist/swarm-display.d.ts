/**
 * Swarm display module for the TypeScript orchestration host.
 *
 * Calls `aether swarm --plan-only` with AETHER_OUTPUT_MODE=json and renders
 * a structured swarm manifest. Supports both dashboard mode (one-shot display)
 * and plain-text mode.
 *
 * Contract: Go is the source of truth for swarm manifest generation. The TS
 * host only renders structured JSON output — never parses wrapper visual text.
 */
import type { GoBridgeOptions } from "./go-bridge.js";
import { callGoJSON } from "./go-bridge.js";
import type { SwarmManifest } from "./types.js";
/** Test-only: inject a mock callGoJSON. */
export declare function __setCallGoJSON(fn: typeof callGoJSON): void;
/** Test-only: restore the real callGoJSON. */
export declare function __restoreCallGoJSON(): void;
/** Options for the swarm display. */
export interface SwarmDisplayOptions extends GoBridgeOptions {
    /** Target problem description (optional). */
    target?: string;
    /** When true and stdout is a TTY, use the dashboard for display. */
    dashboard?: boolean;
    /** When true (default), only generate the plan without running workers. */
    planOnly?: boolean;
}
/** Result of running the swarm display. */
export interface SwarmDisplayResult {
    /** Whether the swarm display completed successfully. */
    success: boolean;
    /** The swarm manifest (present on success). */
    manifest?: SwarmManifest | undefined;
    /** Number of dispatches in the manifest. */
    dispatches_count: number;
    /** Error message if the display failed. */
    error?: string | undefined;
}
/**
 * Run the swarm display by calling `aether swarm --plan-only` via Go.
 *
 * In dashboard mode (TTY + dashboard=true): creates a dashboard and renders
 * the swarm manifest as a one-shot display.
 *
 * In plain-text mode: fetches the manifest and prints a formatted text summary.
 *
 * @param opts - Swarm display options
 * @returns Swarm display result
 */
export declare function runSwarmDisplay(opts: SwarmDisplayOptions): Promise<SwarmDisplayResult>;
/**
 * Render a SwarmManifest as readable plain text.
 *
 * @param manifest - Swarm manifest from Go
 * @returns Formatted multi-line text
 */
export declare function renderSwarmText(manifest: SwarmManifest): string;
/**
 * Render a compact dashboard frame for the swarm manifest.
 *
 * @param manifest - Swarm manifest from Go
 * @returns Compact frame string
 */
export declare function renderSwarmFrame(manifest: SwarmManifest): string;
