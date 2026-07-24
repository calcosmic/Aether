/**
 * Worker dispatch module for the TypeScript orchestration host.
 *
 * Iterates over Go manifest dispatches, records spawn-log before each worker,
 * dispatches the worker (simulated or real), and records spawn-complete
 * after. Restores the visible worker activity lost in the Bash-to-Go migration.
 *
 * When simulateWorkers is false, delegates real worker execution to Go's
 * typed platform adapter boundary. The host coordinates waves but never
 * selects or launches provider CLIs itself.
 *
 * Satisfies HOST-03 (visible dispatch from manifest) and HOST-06 (spawn
 * lifecycle events via Go CLI).
 */

import type { GoBridgeOptions } from "./go-bridge.js";
import {
  callGoJSON,
  callGoJSONAsync,
  cleanupCompletionDir,
  writeCompletionFile,
} from "./go-bridge.js";
import type { BuildDispatch, ExecutionBinding, PermissionProfile, WorkerResult, TerminalWorkerStatus, SpawnClaim, WorkerHandoff } from "./types.js";
import { normalizeSpawnClaims } from "./claims-parser.js";
import { dispatchWaves, type WaveResult } from "./wave-orchestrator.js";

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
  /** Sub-workers requested by this worker via structured spawn claims. */
  spawns?: SpawnClaim[];
  /** Worker handoff data, including child_results for spawned children (SPAWN-04). */
  handoff?: WorkerHandoff;
  /** Provider-returned structured artifacts. */
  artifacts?: Record<string, unknown>;
  /** Planning Scout evidence, when this is a Scout dispatch. */
  scout_report?: unknown;
  /** Route-Setter plan artifact, when returned by the provider. */
  phase_plan?: unknown;
  /** Number of provider tool calls reported by the worker. */
  tool_count?: number;
  /** Blocking findings reported by the worker. */
  blockers?: string[];
  /** Go's host-enforcement decision for the requested caste profile. */
  permission_decision?: PermissionDecision;
  /** Durable build-run identity echoed by the Go adapter. */
  execution_binding?: ExecutionBinding;
  /** Unique provider invocation within the durable build run. */
  provider_run_id?: string;
}

/** Options for worker dispatch, extending Go bridge options. */
export interface DispatchOptions extends GoBridgeOptions {
  /** Workflow owning this dispatch. Build requests require a durable binding. */
  workflow?: string;
  /** Build phase associated with the durable binding. */
  phase?: number;
  /** Go-authored immutable execution identity from the build manifest. */
  executionBinding?: ExecutionBinding;
  /**
   * When explicitly true, simulate worker execution instead of delegating
   * to the Go adapter. Production execution is the default.
   */
  simulateWorkers?: boolean;
  /**
   * Spawn orchestrator for processing worker spawn claims after wave completion.
   * Passed through to wave-orchestrator for child worker dispatch (SPAWN-01).
   */
  spawnOrchestrator?: import("./spawn-orchestrator.js").SpawnOrchestrator;
  /**
   * File paths that actually exist in the repo, used as simulated worker
   * file claims. Must be real repo-relative paths that exist on disk,
   * because the Go build-finalizer validates all file claims.
   */
  simulatedFileClaims?: string[];
  /**
   * When true (default), workers within the same wave are dispatched
   * concurrently via Promise.all. When false, they run sequentially.
   */
  parallel?: boolean;
  /**
   * Maximum number of retry attempts for a failed worker.
   * Default: 2
   */
  retryLimit?: number;
  /**
   * Base delay between retry attempts in milliseconds.
   * Default: 5000
   */
  retryDelayMs?: number;
  /**
   * Timeout for each worker dispatch in milliseconds.
   * Default: 600000 (10 minutes)
   */
  timeoutMs?: number;
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
  // NOTE: This function is only called from dispatchWaves with manifest
  // dispatches (from the Go build manifest). spawn-log/spawn-complete
  // therefore only record manifest workers, never internal/system workers.

  // SPAWN-05: Child spawns record actual parent worker name in spawn tree.
  // Manifest workers (no parent field) log parent="Queen", depth=1.
  // Spawned workers (have parent field) log parent="Builder-01", depth=2.
  const parent = dispatch.parent || "Queen";
  const spawnDepth = String(dispatch.depth || 1);

  // Step 1: Record spawn-log before dispatch.
  // Spawn-log failure is logged but does not block dispatch.
  try {
    const logResult = callGoJSON<{ recorded?: boolean }>(opts, [
      "spawn-log",
      "--parent", parent,
      "--caste", dispatch.caste,
      "--name", dispatch.name,
      "--task", dispatch.task,
      "--depth", spawnDepth,
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
  // Default to real execution; simulation only with explicit opt-in.
  const simulate = opts.simulateWorkers === true;

  if (simulate) {
    process.stderr.write(
      `ℹ️  Simulating worker ${dispatch.name} (simulateWorkers=true)\n`
    );
  }

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
      // Real worker dispatch through the Go-owned platform adapter.
      result = await dispatchRealWorker(opts, dispatch);
    }
  } catch (dispatchErr: unknown) {
    const errMsg =
      dispatchErr instanceof Error ? dispatchErr.message : String(dispatchErr);
    // Auth/config errors halt the build immediately (D-01).
    if (isAuthError(dispatchErr)) {
      throw new Error(
        `Worker dispatch halted: ${sanitizeWorkerDiagnosticOutput(errMsg)}`
      );
    }

    // Timeout/transient errors mark the worker as failed and continue (D-02).
    result = {
      name: dispatch.name,
      status: "failed",
      summary: `Worker dispatch failed: ${sanitizeWorkerDiagnosticOutput(errMsg)}`,
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
 * Dispatch a real worker through Go's typed provider adapter.
 *
 * Go owns platform selection, agent validation, provider invocation, claims
 * parsing, and the versioned platform capability contract. TypeScript owns
 * only the temporary request file and mapping the typed response into the
 * host's wave result.
 *
 * @param opts - Dispatch options
 * @param dispatch - Build dispatch entry
 * @returns Dispatch result from real worker claims
 */
async function dispatchRealWorker(
  opts: DispatchOptions,
  dispatch: BuildDispatch
): Promise<DispatchResult> {
  const timeoutMs = opts.timeoutMs ?? 600_000;
  const request: GoWorkerDispatchRequest = {
    schema_version: 1,
    caste: dispatch.caste,
    worker_name: dispatch.name,
    task_id: dispatch.task_id ?? dispatch.name,
    task: dispatch.task,
    task_brief: dispatch.task_brief ?? dispatch.task,
    timeout_ms: timeoutMs,
    permission_profile: dispatch.permission_profile ?? permissionProfileForCaste(dispatch.caste),
  };
  if (opts.workflow !== undefined) request.workflow = opts.workflow;
  if (opts.phase !== undefined) request.phase = opts.phase;
  if (opts.executionBinding !== undefined) request.execution_binding = opts.executionBinding;
  if (dispatch.context_capsule !== undefined) request.context_capsule = dispatch.context_capsule;
  if (dispatch.skill_section !== undefined) request.skill_section = dispatch.skill_section;
  if (dispatch.hive_section !== undefined) request.hive_section = dispatch.hive_section;
  if (dispatch.pheromone_section !== undefined) request.pheromone_section = dispatch.pheromone_section;
  if (dispatch.handoff_section !== undefined) request.handoff_section = dispatch.handoff_section;

  const requestPath = writeCompletionFile(
    "aether-worker-request",
    "worker-request.json",
    request
  );
  let response: GoWorkerAdapterResponse;
  try {
    response = await callGoJSONAsync<GoWorkerAdapterResponse>(
      opts,
      ["internal-worker-adapter", "--request-file", requestPath],
      timeoutMs + 30_000
    );
  } finally {
    cleanupCompletionDir(requestPath);
  }
  if (response.execution_owner !== "go-adapter" || response.schema_version !== 1) {
    throw new Error("Go worker adapter returned an unsupported ownership contract");
  }
  if (opts.workflow === "build") {
    if (!opts.executionBinding || !response.execution_binding) {
      throw new Error("Go worker adapter returned no durable build execution binding");
    }
    if (!sameExecutionBinding(opts.executionBinding, response.execution_binding)) {
      throw new Error("Go worker adapter returned a result for a different build run");
    }
  }
  const worker = response.worker;
  if (!worker) {
    throw new Error("Go worker adapter returned no terminal worker result");
  }
  if (worker.name !== dispatch.name || worker.caste.toLowerCase() !== dispatch.caste.toLowerCase()) {
    throw new Error("Go worker adapter returned a result for a different worker");
  }
  const status = normalizeTerminalStatus(worker.status);
  const result: DispatchResult = {
    name: dispatch.name,
    status,
    summary:
      worker.summary?.trim() ||
      worker.error?.trim() ||
      `Worker ${dispatch.name} returned ${status}`,
    detectedPlatform: response.platform,
    permission_decision: response.permission_decision,
  };
  if (response.execution_binding !== undefined) result.execution_binding = response.execution_binding;
  if (response.provider_run_id !== undefined) result.provider_run_id = response.provider_run_id;
  if (worker.duration !== undefined) result.duration = worker.duration;
  if (worker.files_created !== undefined) {
    result.files_created = worker.files_created;
  }
  if (worker.files_modified !== undefined) {
    result.files_modified = worker.files_modified;
  }
  if (worker.tests_written !== undefined) {
    result.tests_written = worker.tests_written;
  }
  if (worker.spawns !== undefined && worker.spawns.length > 0) {
    result.spawns = normalizeSpawnClaims(worker.spawns);
  }
  if (worker.handoff !== undefined) result.handoff = worker.handoff;
  if (worker.artifacts !== undefined) {
    result.artifacts = worker.artifacts;
    if (worker.artifacts["phase_plan"] !== undefined) {
      result.phase_plan = worker.artifacts["phase_plan"];
    }
    if (worker.scout_report === undefined && worker.artifacts["scout_report"] !== undefined) {
      result.scout_report = worker.artifacts["scout_report"];
    }
  }
  if (worker.scout_report !== undefined) result.scout_report = worker.scout_report;
  if (worker.tool_count !== undefined) result.tool_count = worker.tool_count;
  if (worker.blockers !== undefined) result.blockers = worker.blockers;

  return result;
}

interface GoWorkerDispatchRequest {
  schema_version: 1;
  workflow?: string;
  phase?: number;
  caste: string;
  worker_name: string;
  task_id: string;
  task: string;
  task_brief: string;
  context_capsule?: string;
  skill_section?: string;
  hive_section?: string;
  pheromone_section?: string;
  handoff_section?: string;
  timeout_ms: number;
  permission_profile: PermissionProfile;
  execution_binding?: ExecutionBinding;
}

// Go remains authoritative and rejects stale or broadened values. This fallback
// covers dynamically spawned children that are not present in a Go manifest.
function permissionProfileForCaste(caste: string): PermissionProfile {
  const normalized = caste.trim().toLowerCase().replace(/^aether-/, "").replaceAll("-", "_");
  const readOnly = normalized === "scout" || normalized === "includer";
  const filesystem = readOnly ? "repository_read_only" : "workspace_write";
  return {
    schema_version: 1,
    name: filesystem,
    filesystem,
    shell: "within_filesystem_boundary",
    network: "provider_default",
    approval: "never",
  };
}

export interface PermissionDecision {
  schema_version: number;
  platform: string;
  profile: PermissionProfile;
  allowed: boolean;
  enforcement: string;
  mechanism: string;
  limitations?: string[];
}

interface GoWorkerAdapterWorker {
  name: string;
  caste: string;
  task_id?: string;
  status: string;
  summary?: string;
  files_created?: string[];
  files_modified?: string[];
  tests_written?: string[];
  artifacts?: Record<string, unknown>;
  scout_report?: unknown;
  tool_count?: number;
  blockers?: string[];
  spawns?: string[];
  duration?: number;
  error?: string;
  handoff?: WorkerHandoff;
}

export interface GoWorkerAdapterResponse {
  schema_version: number;
  execution_owner: string;
  platform: string;
  platform_contract: Record<string, unknown>;
  availability: Record<string, unknown>;
  provider_diagnostics?: string;
  permission_decision: PermissionDecision;
  execution_binding?: ExecutionBinding;
  provider_run_id?: string;
  worker?: GoWorkerAdapterWorker;
}

function sameExecutionBinding(left: ExecutionBinding, right: ExecutionBinding): boolean {
  return left.schema_version === right.schema_version &&
    left.run_id === right.run_id &&
    left.attempt_id === right.attempt_id &&
    left.manifest_sha256 === right.manifest_sha256 &&
    left.workspace_fingerprint === right.workspace_fingerprint &&
    left.execution_owner === right.execution_owner;
}

function normalizeTerminalStatus(value: string): TerminalWorkerStatus {
  const status = value.trim().toLowerCase();
  switch (status) {
    case "completed":
    case "failed":
    case "blocked":
    case "timeout":
      return status;
    default:
      throw new Error(`Go worker adapter returned invalid terminal status "${value}"`);
  }
}

/** Ask the Go-owned adapter layer to select and preflight the worker provider. */
export async function preflightGoWorkerProvider(
  opts: GoBridgeOptions,
  context: string
): Promise<GoWorkerAdapterResponse> {
  try {
    return await callGoJSONAsync<GoWorkerAdapterResponse>(
      opts,
      ["internal-worker-adapter", "--preflight"],
      30_000
    );
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`${context} cannot start: ${sanitizeWorkerDiagnosticOutput(message)}`);
  }
}

export function sanitizeWorkerDiagnosticOutput(value: string): string {
  let output = stripAnsi(value.trim());
  if (!output) {
    return "";
  }
  output = output.replace(
    /\b(token|api[_-]?key|secret|password|authorization)\b\s*[:=]\s*["']?[^"'\s;]+/gi,
    (_match, key: string) => `${key}=[redacted]`
  );
  return output
    .replace(/sk-[A-Za-z0-9_-]+/g, "[redacted]")
    .replace(/github_pat_[A-Za-z0-9_]+/g, "[redacted]")
    .replace(/gh[pousr]_[A-Za-z0-9_]+/g, "[redacted]")
    .replace(/npm_[A-Za-z0-9_]+/g, "[redacted]")
    .replace(/\b[A-Za-z0-9._-]*secret[A-Za-z0-9._-]*\b/gi, "[redacted]");
}

/**
 * Check if an error is classified as an authentication error.
 *
 * Auth errors should halt the build immediately (D-01).
 * The build pipeline (Plan 03+) uses this to decide whether to halt.
 *
 * @param error - The error to check
 * @returns true if the error is an auth/config error
 */
export function isAuthError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error);
  return /\b(auth(?:entication|orization)?|credentials?|login|permission denied|api[_ -]?key)\b/i.test(message);
}

function stripAnsi(value: string): string {
  return value.replace(/\x1b\[[0-9;]*m/g, "");
}

// ---------------------------------------------------------------------------
// Multi-worker dispatch
// ---------------------------------------------------------------------------

/**
 * Dispatch multiple workers from a manifest, grouped by wave.
 *
 * Waves are processed sequentially. Within each wave, workers are
 * dispatched in parallel by default (parallel=true). Delegates to
 * wave-orchestrator.ts for wave grouping and parallel dispatch.
 *
 * @param opts - Dispatch options including Go binary path and cwd
 * @param dispatches - Array of build dispatch entries from the manifest
 * @returns Array of dispatch results, one per input dispatch
 */
export async function dispatchWorkers(
  opts: DispatchOptions,
  dispatches: BuildDispatch[]
): Promise<DispatchResult[]> {
  const waveResults = await dispatchWaves(opts, dispatches);

  // Log wave summaries to stderr with per-worker failure details (D-02).
  for (const wr of waveResults) {
    const succeeded = wr.results.length - wr.failures.length;
    const failed = wr.failures.length;
    if (failed > 0) {
      const failureNames = wr.failures
        .map((f) => `${f.name}: ${f.status}`)
        .join(", ");
      process.stderr.write(
        `Wave ${wr.wave}: ${succeeded} succeeded, ${failed} failed (${failureNames})\n`
      );
    } else {
      process.stderr.write(
        `Wave ${wr.wave}: ${succeeded}/${wr.results.length} succeeded, ${wr.retried} retries\n`
      );
    }
  }

  // Flatten WaveResult array back to DispatchResult array
  const results: DispatchResult[] = [];
  for (const wr of waveResults) {
    results.push(...wr.results);
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
    const status = result?.status ?? "timeout";
    const summary =
      result?.summary ??
      `No terminal worker result was provided for ${dispatch.name}; treating as timeout.`;
    const worker: WorkerResult = {
      name: dispatch.name,
      status,
      summary,
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
    if (result?.blockers !== undefined) worker.blockers = result.blockers;
    if (result?.tool_count !== undefined) worker.tool_count = result.tool_count;
    if (result?.spawns !== undefined) worker.spawns = result.spawns;
    if (result?.handoff !== undefined) worker.handoff = result.handoff;
    if (result?.artifacts !== undefined) worker.artifacts = result.artifacts;
    if (result?.scout_report !== undefined) {
      worker.scout_report = result.scout_report;
    }
    if (result?.phase_plan !== undefined) {
      worker.phase_plan = result.phase_plan;
    }
    return worker;
  });
}
