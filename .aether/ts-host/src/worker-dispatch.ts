/**
 * Worker dispatch module for the TypeScript orchestration host.
 *
 * Iterates over Go manifest dispatches, records spawn-log before each worker,
 * dispatches the worker (simulated or real), and records spawn-complete
 * after. Restores the visible worker activity lost in the Bash-to-Go migration.
 *
 * When simulateWorkers is false, dispatches real workers via platform CLI
 * subprocess using platform-dispatcher.ts, prompt-assembler.ts, and
 * claims-parser.ts.
 *
 * Satisfies HOST-03 (visible dispatch from manifest) and HOST-06 (spawn
 * lifecycle events via Go CLI).
 */

import type { GoBridgeOptions } from "./go-bridge.js";
import { callGoJSON } from "./go-bridge.js";
import type { BuildDispatch, WorkerResult, TerminalWorkerStatus } from "./types.js";
import {
  createPlatformDispatcher,
  detectAvailablePlatforms,
  spawnWorker,
  type Platform,
  type WorkerConfig,
} from "./platform-dispatcher.js";
import {
  assemblePrompt,
  getAgentNameForCaste,
} from "./prompt-assembler.js";
import { parseWorkerClaims } from "./claims-parser.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Result of dispatching a single worker. */
export interface DispatchResult {
  /** Worker name from the manifest dispatch. */
  name: string;
  /** Terminal status after dispatch attempt. */
  status: TerminalWorkerStatus;
  /** Summary of what the worker did (or why it failed). */
  summary: string;
  /** Approximate duration in seconds. */
  duration?: number;
  /** Files modified by the worker (simulated or real). */
  files_modified?: string[];
  /** Files created by the worker (simulated or real). */
  files_created?: string[];
  /** Tests written by the worker (simulated or real). */
  tests_written?: string[];
  /** Detected platform for debugging. */
  detectedPlatform?: string;
}

/** Options for worker dispatch, extending Go bridge options. */
export interface DispatchOptions extends GoBridgeOptions {
  /**
   * When true (default), simulate worker execution instead of spawning
   * a real platform CLI. The prototype uses simulation.
   */
  simulateWorkers?: boolean;
  /**
   * File paths that actually exist in the repo, used as simulated worker
   * file claims. Must be real repo-relative paths that exist on disk,
   * because the Go build-finalizer validates all file claims.
   */
  simulatedFileClaims?: string[];
}

// ---------------------------------------------------------------------------
// Single worker dispatch
// ---------------------------------------------------------------------------

/**
 * Dispatch a single worker from a manifest dispatch entry.
 *
 * Lifecycle:
 * 1. Call `aether spawn-log` to record the spawn before dispatch.
 * 2. Execute the worker (simulated or real).
 * 3. Call `aether spawn-complete` to record the outcome.
 *
 * Spawn-log failure does not block dispatch. Spawn-complete is always
 * attempted, even on dispatch error.
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatch - Build dispatch entry from the Go manifest
 * @returns Dispatch result with name, status, and summary
 */
export async function dispatchSingleWorker(
  opts: DispatchOptions,
  dispatch: BuildDispatch
): Promise<DispatchResult> {
  const depth = "1"; // Default depth for prototype

  // Step 1: Record spawn-log before dispatch.
  // Spawn-log failure is logged but does not block dispatch.
  try {
    const logResult = callGoJSON<{ recorded?: boolean }>(opts, [
      "spawn-log",
      "--parent", "Queen",
      "--caste", dispatch.caste,
      "--name", dispatch.name,
      "--task", dispatch.task,
      "--depth", depth,
    ]);
    if (!logResult.recorded) {
      process.stderr.write(
        `Warning: spawn-log for ${dispatch.name} returned recorded=false\n`
      );
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    process.stderr.write(
      `Warning: spawn-log failed for ${dispatch.name}: ${msg}\n`
    );
  }

  // Step 2: Execute the worker.
  let result: DispatchResult;
  const simulate = opts.simulateWorkers !== false; // default true

  try {
    if (simulate) {
      // Simulated worker execution: brief delay to mimic work.
      await new Promise<void>((resolve) => setTimeout(resolve, 100));
      // Use configurable simulated file claims, or empty if not provided.
      // The Go build-finalizer validates that all file claims exist in the
      // repo, so simulated claims must reference real files.
      const simClaims = opts.simulatedFileClaims ?? [];
      result = {
        name: dispatch.name,
        status: "completed",
        summary: `Simulated worker completion for ${dispatch.name}`,
        duration: 0.1,
      };
      if (simClaims.length > 0) {
        result.files_modified = [simClaims[0]!];
      }
      if (simClaims.length > 1) {
        result.tests_written = [simClaims[1]!];
      }
    } else {
      // Real worker dispatch via platform CLI.
      result = await dispatchRealWorker(opts, dispatch);
    }
  } catch (dispatchErr: unknown) {
    const errMsg =
      dispatchErr instanceof Error ? dispatchErr.message : String(dispatchErr);
    result = {
      name: dispatch.name,
      status: "failed",
      summary: `Worker dispatch failed: ${errMsg}`,
    };
  }

  // Step 3: Record spawn-complete after dispatch.
  // Always attempt spawn-complete, even if the worker failed.
  try {
    const completeResult = callGoJSON<{ completed?: boolean }>(opts, [
      "spawn-complete",
      "--name", result.name,
      "--status", result.status,
      "--summary", result.summary,
    ]);
    if (!completeResult.completed) {
      process.stderr.write(
        `Warning: spawn-complete for ${result.name} returned completed=false\n`
      );
    }
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    process.stderr.write(
      `Warning: spawn-complete failed for ${result.name}: ${msg}\n`
    );
  }

  return result;
}

// ---------------------------------------------------------------------------
// Real worker dispatch
// ---------------------------------------------------------------------------

/**
 * Dispatch a real worker via platform CLI subprocess.
 *
 * 1. Detect available platform (default "claude")
 * 2. Assemble prompt from agent definition + task brief + response contract
 * 3. Spawn subprocess via platform-dispatcher
 * 4. Parse claims JSON from stdout
 * 5. Build DispatchResult from claims
 *
 * @param opts - Dispatch options
 * @param dispatch - Build dispatch entry
 * @returns Dispatch result from real worker claims
 */
async function dispatchRealWorker(
  opts: DispatchOptions,
  dispatch: BuildDispatch
): Promise<DispatchResult> {
  // Detect platform: prefer "claude" if available, else first available.
  let platform: Platform = "claude";
  const available = await detectAvailablePlatforms();
  if (available.length > 0 && !available.includes("claude")) {
    platform = available[0]!;
  }

  const agentName = getAgentNameForCaste(dispatch.caste);

  const prompt = assemblePrompt({
    cwd: opts.cwd,
    caste: dispatch.caste,
    name: dispatch.name,
    task: dispatch.task,
    platform,
    agentName,
  });

  const config: WorkerConfig = {
    platform,
    agentName,
    caste: dispatch.caste,
    name: dispatch.name,
    task: dispatch.task,
    root: opts.cwd,
    prompt,
  };

  const spawnResult = await spawnWorker(config);

  if (spawnResult.exitCode !== 0) {
    return {
      name: dispatch.name,
      status: "failed",
      summary: `Worker exited with code ${spawnResult.exitCode}: ${spawnResult.stderr.slice(0, 200)}`,
      duration: spawnResult.duration / 1000,
      detectedPlatform: platform,
    };
  }

  let claims;
  try {
    claims = parseWorkerClaims(spawnResult.stdout);
  } catch (parseErr: unknown) {
    const msg = parseErr instanceof Error ? parseErr.message : String(parseErr);
    return {
      name: dispatch.name,
      status: "failed",
      summary: `Failed to parse worker claims: ${msg}`,
      duration: spawnResult.duration / 1000,
      detectedPlatform: platform,
    };
  }

  const result: DispatchResult = {
    name: dispatch.name,
    status: (claims.status as TerminalWorkerStatus) ?? "completed",
    summary: claims.summary ?? `Worker ${dispatch.name} completed`,
    duration: spawnResult.duration / 1000,
    detectedPlatform: platform,
  };

  if (claims.files_created !== undefined) {
    result.files_created = claims.files_created;
  }
  if (claims.files_modified !== undefined) {
    result.files_modified = claims.files_modified;
  }
  if (claims.tests_written !== undefined) {
    result.tests_written = claims.tests_written;
  }

  return result;
}

// ---------------------------------------------------------------------------
// Multi-worker dispatch
// ---------------------------------------------------------------------------

/**
 * Dispatch multiple workers from a manifest, grouped by wave.
 *
 * Waves are processed sequentially. Within each wave, workers are
 * dispatched sequentially (parallel within-wave dispatch can be
 * added later).
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatches - Array of build dispatch entries from the manifest
 * @returns Array of dispatch results, one per input dispatch
 */
export async function dispatchWorkers(
  opts: DispatchOptions,
  dispatches: BuildDispatch[]
): Promise<DispatchResult[]> {
  // Group dispatches by wave number.
  const waveMap = new Map<number, BuildDispatch[]>();
  for (const d of dispatches) {
    const wave = d.wave ?? d.execution_wave ?? 0;
    const existing = waveMap.get(wave);
    if (existing) {
      existing.push(d);
    } else {
      waveMap.set(wave, [d]);
    }
  }

  // Sort waves ascending.
  const sortedWaves = Array.from(waveMap.keys()).sort((a, b) => a - b);

  const results: DispatchResult[] = [];

  for (const wave of sortedWaves) {
    const waveDispatches = waveMap.get(wave)!;
    process.stderr.write(
      `Dispatching wave ${wave}: ${waveDispatches.length} workers\n`
    );

    // Dispatch workers sequentially within each wave.
    for (const d of waveDispatches) {
      const result = await dispatchSingleWorker(opts, d);
      results.push(result);
    }

    process.stderr.write(`Wave ${wave} complete\n`);
  }

  return results;
}

// ---------------------------------------------------------------------------
// Result mapping
// ---------------------------------------------------------------------------

/**
 * Map dispatch entries and their results to WorkerResult objects for
 * the Go build finalizer.
 *
 * Matches results to dispatches by name (not by index) to handle
 * wave-grouped re-ordering from dispatchWorkers.
 *
 * Preserves manifest fields (caste, task_id, stage, wave) alongside
 * dispatch outcomes (status, summary).
 *
 * @param dispatches - Original build dispatch entries from the manifest
 * @param results - Dispatch results from dispatchWorkers
 * @returns WorkerResult array suitable for the Go finalizer completion file
 */
export function toWorkerResults(
  dispatches: BuildDispatch[],
  results: DispatchResult[]
): WorkerResult[] {
  // Build a name -> result lookup for O(1) matching
  const resultByName = new Map<string, DispatchResult>();
  for (const r of results) {
    resultByName.set(r.name, r);
  }

  return dispatches.map((dispatch): WorkerResult => {
    const result = resultByName.get(dispatch.name);
    // Build result object; omit optional fields when undefined for
    // exactOptionalPropertyTypes compatibility.
    const worker: WorkerResult = {
      name: dispatch.name,
      status: result?.status ?? "completed",
      summary: result?.summary ?? `Simulated completion for ${dispatch.name}`,
      caste: dispatch.caste,
      task: dispatch.task,
      stage: dispatch.stage,
    };
    if (dispatch.task_id !== undefined) worker.task_id = dispatch.task_id;
    if (dispatch.wave !== undefined) worker.wave = dispatch.wave;
    if (dispatch.execution_wave !== undefined) worker.execution_wave = dispatch.execution_wave;
    if (result?.duration !== undefined) worker.duration = result.duration;
    if (result?.files_modified !== undefined) worker.files_modified = result.files_modified;
    if (result?.files_created !== undefined) worker.files_created = result.files_created;
    if (result?.tests_written !== undefined) worker.tests_written = result.tests_written;
    return worker;
  });
}
