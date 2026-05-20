/**
 * Watch display module for the TypeScript orchestration host.
 *
 * Calls `aether status` with AETHER_OUTPUT_MODE=json and renders structured
 * colony status. Supports both live dashboard mode (TTY refresh loop) and
 * plain-text mode (single snapshot).
 *
 * Contract: Go is the source of truth for colony state. The TS host only
 * renders structured JSON output — never parses wrapper visual text.
 */
import type { GoBridgeOptions } from "./go-bridge.js";
import { callGoJSON } from "./go-bridge.js";
import type { StatusResult } from "./types.js";
/** Test-only: inject a mock callGoJSON. */
export declare function __setCallGoJSON(fn: typeof callGoJSON): void;
/** Test-only: restore the real callGoJSON. */
export declare function __restoreCallGoJSON(): void;
/** Options for the watch display. */
export interface WatchDisplayOptions extends GoBridgeOptions {
    /** When true and stdout is a TTY, use the live dashboard with refresh. */
    dashboard?: boolean;
    /** Refresh interval in milliseconds (default: 5000, minimum: 1000). */
    refreshIntervalMs?: number;
}
/** Result of running the watch display. */
export interface WatchDisplayResult {
    /** Whether the watch completed successfully. */
    success: boolean;
    /** The final status snapshot (last fetched). */
    final_status?: StatusResult | undefined;
    /** Number of status snapshots taken. */
    snapshots_count: number;
    /** Error message if the watch failed. */
    error?: string | undefined;
}
/**
 * Run the watch display by calling `aether status` via Go and rendering output.
 *
 * In dashboard mode (TTY + dashboard=true): creates a live dashboard and
 * refreshes at the configured interval until interrupted.
 *
 * In plain-text mode: fetches status once and prints a formatted text summary.
 *
 * @param opts - Watch display options
 * @returns Watch display result
 */
export declare function runWatchDisplay(opts: WatchDisplayOptions): Promise<WatchDisplayResult>;
/**
 * Render a StatusResult as readable plain text.
 *
 * @param status - Status result from Go
 * @returns Formatted multi-line text
 */
export declare function renderStatusText(status: StatusResult): string;
/**
 * Render a compact single-frame dashboard display for live watch mode.
 *
 * @param status - Status result from Go
 * @returns Compact frame string with progress bar
 */
export declare function renderStatusFrame(status: StatusResult): string;
