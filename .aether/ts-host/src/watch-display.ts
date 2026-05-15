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
import { createDashboard } from "./dashboard.js";

// Mutable reference for test injection.
let _callGoJSONRef = callGoJSON;

/** Test-only: inject a mock callGoJSON. */
export function __setCallGoJSON(fn: typeof callGoJSON): void {
  _callGoJSONRef = fn;
}

/** Test-only: restore the real callGoJSON. */
export function __restoreCallGoJSON(): void {
  _callGoJSONRef = callGoJSON;
}

function callGoJSONRef<T>(opts: GoBridgeOptions, args: string[]): T {
  return _callGoJSONRef(opts, args);
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

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
export async function runWatchDisplay(
  opts: WatchDisplayOptions
): Promise<WatchDisplayResult> {
  const refreshIntervalMs = Math.max(
    opts.refreshIntervalMs ?? 5000,
    1000
  );
  const useDashboard =
    opts.dashboard !== false && process.stdout.isTTY;

  let snapshotsCount = 0;
  let finalStatus: StatusResult | undefined;

  try {
    if (useDashboard) {
      return await runDashboardWatch(opts, refreshIntervalMs);
    }

    // Plain-text mode: single fetch
    const status = fetchStatus(opts);
    snapshotsCount = 1;
    finalStatus = status;

    const text = renderStatusText(status);
    process.stdout.write(text + "\n");

    return {
      success: true,
      final_status: status,
      snapshots_count: snapshotsCount,
    };
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      success: false,
      final_status: finalStatus,
      snapshots_count: snapshotsCount,
      error: message,
    };
  }
}

// ---------------------------------------------------------------------------
// Dashboard mode
// ---------------------------------------------------------------------------

async function runDashboardWatch(
  opts: WatchDisplayOptions,
  refreshIntervalMs: number
): Promise<WatchDisplayResult> {
  const dashboard = createDashboard({ cwd: opts.cwd });
  dashboard.start();

  let snapshotsCount = 0;
  let finalStatus: StatusResult | undefined;
  let running = true;

  // Handle Ctrl+C gracefully
  const abortController = new AbortController();
  const onSigint = (): void => {
    running = false;
    abortController.abort();
  };
  process.on("SIGINT", onSigint);

  try {
    while (running) {
      try {
        const status = fetchStatus(opts);
        snapshotsCount++;
        finalStatus = status;

        const frame = renderStatusFrame(status);
        // Clear previous frame and write new one
        process.stdout.write("\x1b[2J\x1b[H"); // clear screen, move cursor home
        process.stdout.write(frame + "\n");
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Watch refresh error: ${msg}\n`);
      }

      // Wait for interval or abort
      await sleep(refreshIntervalMs, abortController.signal);
    }
  } finally {
    process.removeListener("SIGINT", onSigint);
    dashboard.stop();
  }

  return {
    success: true,
    final_status: finalStatus,
    snapshots_count: snapshotsCount,
  };
}

// ---------------------------------------------------------------------------
// Go invocation
// ---------------------------------------------------------------------------

function fetchStatus(opts: GoBridgeOptions): StatusResult {
  return callGoJSONRef<StatusResult>(opts, ["status"]);
}

// ---------------------------------------------------------------------------
// Text rendering
// ---------------------------------------------------------------------------

/**
 * Render a StatusResult as readable plain text.
 *
 * @param status - Status result from Go
 * @returns Formatted multi-line text
 */
export function renderStatusText(status: StatusResult): string {
  const lines: string[] = [];

  lines.push("Colony Status");
  lines.push("=============");
  lines.push(`Goal:    ${status.goal || "(none)"}`);
  lines.push(`State:   ${status.state}`);
  lines.push(`Phase:   ${status.phase_name || "(none)"} (${status.phases_completed}/${status.total_phases})`);

  if (status.tasks_total > 0) {
    const pct = Math.round((status.tasks_completed / status.tasks_total) * 100);
    lines.push(`Tasks:   ${status.tasks_completed}/${status.tasks_total} (${pct}%)`);
  }

  if (status.colony_mode) {
    lines.push(`Mode:    ${status.colony_mode}`);
  }

  if (status.warnings && status.warnings.length > 0) {
    lines.push("");
    lines.push("Warnings:");
    for (const warning of status.warnings) {
      lines.push(`  - ${warning}`);
    }
  }

  return lines.join("\n");
}

/**
 * Render a compact single-frame dashboard display for live watch mode.
 *
 * @param status - Status result from Go
 * @returns Compact frame string with progress bar
 */
export function renderStatusFrame(status: StatusResult): string {
  const lines: string[] = [];

  lines.push("Aether Watch");
  lines.push("------------");

  const phasePct = status.total_phases > 0
    ? Math.round((status.phases_completed / status.total_phases) * 100)
    : 0;
  lines.push(`Phase: ${progressBar(status.phases_completed, status.total_phases, 20)} ${phasePct}%`);

  if (status.tasks_total > 0) {
    const taskPct = Math.round((status.tasks_completed / status.tasks_total) * 100);
    lines.push(`Tasks: ${progressBar(status.tasks_completed, status.tasks_total, 20)} ${taskPct}%`);
  }

  lines.push(`State: ${status.state}`);
  lines.push(`Goal:  ${truncate(status.goal, 50)}`);

  if (status.warnings && status.warnings.length > 0) {
    lines.push(`Warn:  ${status.warnings.length} warning(s)`);
  }

  return lines.join("\n");
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function progressBar(current: number, total: number, width: number): string {
  if (total <= 0) {
    return "[" + "░".repeat(width) + "]";
  }
  const filled = Math.min(width, Math.round((current / total) * width));
  const empty = width - filled;
  return "[" + "█".repeat(filled) + "░".repeat(empty) + "]";
}

function truncate(text: string, maxLen: number): string {
  if (!text || text.length <= maxLen) return text || "";
  return text.slice(0, maxLen - 3) + "...";
}

function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, ms);
    if (signal) {
      const onAbort = (): void => {
        clearTimeout(timer);
        resolve();
      };
      if (signal.aborted) {
        onAbort();
        return;
      }
      signal.addEventListener("abort", onAbort, { once: true });
    }
  });
}
