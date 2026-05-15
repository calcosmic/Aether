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
import type { SwarmManifest, SwarmWorkerPlan } from "./types.js";
import { createDashboard } from "./dashboard.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

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
export async function runSwarmDisplay(
  opts: SwarmDisplayOptions
): Promise<SwarmDisplayResult> {
  const useDashboard = opts.dashboard !== false && process.stdout.isTTY;
  const planOnly = opts.planOnly !== false;

  try {
    const manifest = fetchSwarmManifest(opts, planOnly);

    if (!manifest.dispatches || manifest.dispatches.length === 0) {
      return {
        success: false,
        dispatches_count: 0,
        error: "Swarm manifest contains no dispatches",
      };
    }

    if (useDashboard) {
      const dashboard = createDashboard({ cwd: opts.cwd });
      dashboard.start();

      const frame = renderSwarmFrame(manifest);
      process.stdout.write("\x1b[2J\x1b[H"); // clear screen, move cursor home
      process.stdout.write(frame + "\n");

      dashboard.stop();
    } else {
      const text = renderSwarmText(manifest);
      process.stdout.write(text + "\n");
    }

    return {
      success: true,
      manifest,
      dispatches_count: manifest.dispatches.length,
    };
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      success: false,
      dispatches_count: 0,
      error: message,
    };
  }
}

// ---------------------------------------------------------------------------
// Go invocation
// ---------------------------------------------------------------------------

function fetchSwarmManifest(
  opts: SwarmDisplayOptions,
  planOnly: boolean
): SwarmManifest {
  const args = ["swarm"];
  if (planOnly) {
    args.push("--plan-only");
  }
  if (opts.target) {
    args.push(opts.target);
  }
  return callGoJSON<SwarmManifest>(opts, args);
}

// ---------------------------------------------------------------------------
// Text rendering
// ---------------------------------------------------------------------------

/**
 * Render a SwarmManifest as readable plain text.
 *
 * @param manifest - Swarm manifest from Go
 * @returns Formatted multi-line text
 */
export function renderSwarmText(manifest: SwarmManifest): string {
  const lines: string[] = [];

  lines.push("Swarm Manifest");
  lines.push("==============");
  lines.push(`Swarm ID: ${manifest.swarm_id}`);
  lines.push(`Target:   ${manifest.target || "(none)"}`);
  lines.push(`Mode:     ${manifest.dispatch_mode}`);
  lines.push(`Waves:    ${manifest.wave_count}`);
  lines.push(`Workers:  ${manifest.worker_count}`);
  lines.push("");

  // Group dispatches by wave
  const waves = groupByWave(manifest.dispatches);
  for (const waveNum of Object.keys(waves).sort((a, b) => parseInt(a, 10) - parseInt(b, 10))) {
    const workers = waves[waveNum]!;
    lines.push(`Wave ${waveNum} (${workers[0]?.stage || "swarm"})`);
    for (const worker of workers) {
      lines.push(`  ${worker.caste} ${worker.name} — ${worker.task}`);
    }
    lines.push("");
  }

  if (manifest.finalizer_command) {
    lines.push(`Finalizer: ${manifest.finalizer_command}`);
  }

  return lines.join("\n");
}

/**
 * Render a compact dashboard frame for the swarm manifest.
 *
 * @param manifest - Swarm manifest from Go
 * @returns Compact frame string
 */
export function renderSwarmFrame(manifest: SwarmManifest): string {
  const lines: string[] = [];

  lines.push("Aether Swarm");
  lines.push("------------");
  lines.push(`Swarm: ${manifest.swarm_id}`);
  lines.push(`Target: ${truncate(manifest.target, 40)}`);
  lines.push(`Waves: ${manifest.wave_count} | Workers: ${manifest.worker_count}`);
  lines.push("");

  const waves = groupByWave(manifest.dispatches);
  for (const waveNum of Object.keys(waves).sort((a, b) => parseInt(a, 10) - parseInt(b, 10))) {
    const workers = waves[waveNum]!;
    lines.push(`Wave ${waveNum}: ${workers.length} worker(s)`);
    for (const worker of workers) {
      lines.push(`  ${worker.caste} ${truncate(worker.name, 20)} — ${truncate(worker.task, 40)}`);
    }
  }

  if (manifest.finalizer_command) {
    lines.push("");
    lines.push(`Finalizer: ${truncate(manifest.finalizer_command, 60)}`);
  }

  return lines.join("\n");
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function groupByWave(dispatches: SwarmWorkerPlan[]): Record<string, SwarmWorkerPlan[]> {
  const waves: Record<string, SwarmWorkerPlan[]> = {};
  for (const dispatch of dispatches) {
    const key = String(dispatch.wave);
    if (!waves[key]) {
      waves[key] = [];
    }
    waves[key]!.push(dispatch);
  }
  return waves;
}

function truncate(text: string, maxLen: number): string {
  if (!text || text.length <= maxLen) return text || "";
  return text.slice(0, maxLen - 3) + "...";
}
