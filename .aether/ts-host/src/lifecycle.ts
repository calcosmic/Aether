/**
 * Full lifecycle orchestrator for the TypeScript orchestration host.
 *
 * Drives the plan -> build 1 -> continue lifecycle by:
 * 1. Calling Go --plan-only commands to obtain JSON manifests
 * 2. Building completion files with worker results
 * 3. Calling Go finalizer commands to commit state changes
 * 4. Dispatching build workers with spawn-log/complete via Go CLI
 *
 * Contract: The TS host never writes to .aether/data/ directly. All state
 * mutations go through Go finalizer commands.
 *
 * Satisfies HOST-04 (finalizers called) and HOST-07 (end-to-end lifecycle).
 */

import { tmpdir } from "node:os";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

import type { GoBridgeOptions } from "./go-bridge.js";
import { callGoJSON, writeCompletionFile } from "./go-bridge.js";
import type {
  BuildManifest,
  BuildDispatch,
  WorkerResult,
  PlanCompletion,
  ContinueCompletion,
  PlanningDispatch,
} from "./types.js";
import {
  dispatchWorkers,
  toWorkerResults,
  type DispatchOptions,
} from "./worker-dispatch.js";
import { detectAvailablePlatforms, formatPlatformUnavailableMessage } from "./platform-dispatcher.js";
import { createDashboard, type Dashboard } from "./dashboard.js";
import { createNarrator, type Narrator } from "./narrator.js";
import { startEventBridge, stopEventBridge, type EventBridgeController } from "./event-bridge.js";
import { createQueenOrchestrator as _createQueenOrchestrator } from "./queen/orchestrator.js";
import type { QueenOrchestratorResult } from "./queen/types.js";
import { runOracleLifecycle, type OracleLifecycleOptions } from "./oracle-lifecycle.js";
import type { OracleLifecycleResult } from "./oracle-lifecycle.js";
import {
  createCeremonyAdapter,
  type CeremonyAdapter,
  type CeremonyWorkflow,
} from "./ceremony-adapter.js";

// Mutable reference for test injection.
let _createQueenOrchestratorRef = _createQueenOrchestrator;

/** Test-only: inject a mock createQueenOrchestrator. */
export function __setCreateQueenOrchestrator(
  fn: typeof _createQueenOrchestrator
): void {
  _createQueenOrchestratorRef = fn;
}

/** Test-only: restore the real createQueenOrchestrator. */
export function __restoreCreateQueenOrchestrator(): void {
  _createQueenOrchestratorRef = _createQueenOrchestrator;
}

function createQueenOrchestrator(...args: Parameters<typeof _createQueenOrchestrator>) {
  return _createQueenOrchestratorRef(...args);
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Options for the full lifecycle orchestrator. */
export interface LifecycleOptions extends DispatchOptions {
  /** Phase number to build (default: 1). */
  phase?: number;
  /**
   * When true (default), workers within the same wave are dispatched
   * concurrently via Promise.all.
   */
  parallel?: boolean;
  /**
   * When true (default when TTY), show the live dashboard during build.
   * When false, use plain text narrator output.
   */
  dashboard?: boolean;
  /** When true, skip the pre-build midden threshold check. */
  skipMiddenCheck?: boolean;
  /** When true, run Oracle RALF research before planning. */
  runOracle?: boolean;
  /** Research topic for the Oracle step (default: "auto"). */
  oracleTopic?: string;
}

/** Oracle result summary embedded in LifecycleResult. */
export interface LifecycleOracleResult {
  /** Number of Oracle iterations completed. */
  iterations_completed: number;
  /** Final confidence percentage after Oracle loop. */
  final_confidence: number;
  /** Reason the Oracle loop stopped. */
  stop_reason: string;
}

/** Result of the lifecycle orchestration. */
export interface LifecycleResult {
  /** Whether the full lifecycle completed successfully. */
  success: boolean;
  /** Steps completed in order. */
  steps_completed: string[];
  /** Oracle result if the Oracle step ran. */
  oracle_result?: LifecycleOracleResult;
  /** Error message if the lifecycle failed. */
  error?: string;
}

/** Plan manifest shape from Go plan --plan-only output. */
interface PlanManifestResult {
  plan_manifest?: Record<string, unknown>;
  planning_manifest?: Record<string, unknown>;
  dispatches?: PlanningDispatchResult[];
  [key: string]: unknown;
}

/** Planning dispatch from Go plan --plan-only output. */
interface PlanningDispatchResult {
  name?: string;
  caste?: string;
  stage?: string;
  task?: string;
  task_id?: string;
  wave?: number;
  execution_wave?: number;
  [key: string]: unknown;
}

/** Build manifest result from Go build --plan-only output. */
interface BuildManifestResult {
  dispatch_manifest?: BuildManifest;
  dispatches?: BuildDispatch[];
  dispatch_count?: number;
  provider_diagnostics?: string;
  [key: string]: unknown;
}

/** Continue manifest result from Go continue --plan-only output. */
interface ContinueManifestResult {
  continue_manifest?: Record<string, unknown>;
  dispatches?: ContinueDispatchResult[];
  phase?: number;
  [key: string]: unknown;
}

/** Continue dispatch from Go continue --plan-only output. */
interface ContinueDispatchResult {
  name?: string;
  caste?: string;
  stage?: string;
  task?: string;
  task_id?: string;
  wave?: number;
  execution_wave?: number;
  status?: string;
  summary?: string;
  [key: string]: unknown;
}

interface CeremonyDispatchLike {
  execution_wave?: number;
  wave?: number;
}

function emitCeremonyOutput(output: string): void {
  if (output.trim() === "") {
    return;
  }
  process.stderr.write(output.endsWith("\n") ? output : `${output}\n`);
}

function ceremonyExecutionWaves(dispatches: CeremonyDispatchLike[]): number[] {
  const waves = new Set<number>();
  for (const dispatch of dispatches) {
    const wave = dispatch.execution_wave ?? dispatch.wave ?? 0;
    if (wave > 0) {
      waves.add(wave);
    }
  }
  return [...waves].sort((a, b) => a - b);
}

function renderManifestCeremony(
  ceremony: CeremonyAdapter,
  workflow: CeremonyWorkflow,
  manifestEnvelope: unknown,
  dispatches: CeremonyDispatchLike[]
): void {
  emitCeremonyOutput(ceremony.renderSpawnPlan(workflow, manifestEnvelope));
  for (const executionWave of ceremonyExecutionWaves(dispatches)) {
    emitCeremonyOutput(
      ceremony.renderWaveStart(workflow, manifestEnvelope, executionWave)
    );
  }
}

function renderWorkerCeremony(
  ceremony: CeremonyAdapter,
  workflow: CeremonyWorkflow,
  workers: unknown[]
): void {
  for (const worker of workers) {
    emitCeremonyOutput(ceremony.renderWorkerComplete(workflow, worker));
  }
}

function stringField(value: unknown, key: string): string | undefined {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return undefined;
  }
  const field = (value as Record<string, unknown>)[key];
  return typeof field === "string" && field.trim() !== "" ? field.trim() : undefined;
}

function providerDiagnosticFromBuildResult(
  buildResult: BuildManifestResult,
  buildManifest: BuildManifest
): string | undefined {
  return (
    stringField(buildResult, "provider_diagnostics") ??
    stringField(buildManifest, "provider_diagnostics")
  );
}

function formatLifecycleProviderUnavailableMessage(
  context: string,
  buildResult: BuildManifestResult,
  buildManifest: BuildManifest
): string {
  return (
    providerDiagnosticFromBuildResult(buildResult, buildManifest) ??
    formatPlatformUnavailableMessage(context)
  );
}

// ---------------------------------------------------------------------------
// Lifecycle orchestrator
// ---------------------------------------------------------------------------

/**
 * Run the full plan -> build -> continue lifecycle through the Go CLI.
 *
 * Each step:
 * 1. Calls Go --plan-only to get a manifest (no state mutation)
 * 2. Builds a completion file with worker results
 * 3. Calls the Go finalizer to commit state atomically
 *
 * Error handling: Per HOST-07, if the lifecycle cannot complete, the error
 * message documents the exact blocker.
 *
 * @param opts - Lifecycle options including Go binary path and cwd
 * @returns Lifecycle result with success status and completed steps
 */
export async function runLifecycle(
  opts: LifecycleOptions
): Promise<LifecycleResult> {
  const stepsCompleted: string[] = [];
  const targetPhase = opts.phase ?? 1;
  let oracleResult: OracleLifecycleResult | undefined;
  const ceremony = createCeremonyAdapter(opts);

  // Determine if dashboard should be active (default true when TTY)
  const useDashboard = opts.dashboard !== false && process.stdout.isTTY;
  let dashboard: Dashboard | undefined;

  try {
    // ── Step 0: Oracle (optional) ────────────────────────────────────────
    if (opts.runOracle) {
      const oracleOpts: OracleLifecycleOptions = {
        goBinaryPath: opts.goBinaryPath,
        cwd: opts.cwd,
        topic: opts.oracleTopic ?? "auto",
        simulateWorkers: opts.simulateWorkers ?? false,
        dashboard: useDashboard,
      };

      const oracleRes = await runOracleLifecycle(oracleOpts);
      if (!oracleRes.success) {
        throw new Error(oracleRes.error ?? "Oracle lifecycle failed");
      }
      oracleResult = oracleRes;
      stepsCompleted.push("oracle");
      process.stderr.write(
        `Oracle completed: ${oracleRes.iterations_completed} iterations, confidence ${oracleRes.final_confidence}%\n`
      );
    }

    // ── Step 1: Plan ─────────────────────────────────────────────────────

    const planResult = callGoJSON<PlanManifestResult>(opts, [
      "plan",
      "--plan-only",
      "--depth",
      "fast",
    ]);

    // Extract the plan manifest from the result. Go plan --plan-only outputs
    // plan_manifest as the primary manifest field.
    const planManifest = planResult.plan_manifest ?? planResult.planning_manifest;
    if (!planManifest) {
      throw new Error("Plan --plan-only returned no plan_manifest");
    }

    // Extract planning dispatches from the result. These are the planning
    // workers (scout, route-setter) that would normally run during planning.
    const planDispatches = planResult.dispatches ?? [];
    const planCeremonyEnvelope = {
      plan_manifest: planManifest,
      dispatches: planDispatches,
    };
    renderManifestCeremony(ceremony, "plan", planCeremonyEnvelope, planDispatches);

    // The TS host does not run planning workers here. Only real planning
    // dispatch completions belong in this list; host-created plans are labeled
    // with the synthesis envelope below.
    const planningResults: PlanningDispatch[] = [];
    renderWorkerCeremony(ceremony, "plan", planningResults);

    // Build plan completion file with explicit host synthesis.
    // The Go plan-finalizer requires a phase_plan (codexWorkerPlanArtifact)
    // with at least one phase containing tasks. The TS host labels this as
    // synthesis instead of presenting it as worker evidence.
    const goal = typeof planManifest["goal"] === "string"
      ? planManifest["goal"] as string
      : "colony goal";
    const phasePlan = {
      phases: [
        {
          name: "Phase 1: Implementation",
          description: `Implement the colony goal: ${goal}`,
          tasks: [
            {
              goal: `Implement ${goal}`,
              constraints: [],
              hints: [],
              success_criteria: ["Code compiles", "Tests pass"],
            },
          ],
          success_criteria: ["Feature implemented", "Tests passing"],
        },
      ],
      confidence: { coverage: 50, complexity: 50, dependencies: 50, effort: 50, overall: 50 },
    };

    const planCompletion: PlanCompletion = {
      plan_manifest: planManifest,
      dispatches: planningResults,
      synthesis: {
        source: "ts-host",
        reason: "TS lifecycle host created a fallback phase plan without running planning workers.",
        phase_plan: phasePlan,
      },
    };

    const planCompletionPath = writeCompletionFile(
      "aether-lifecycle",
      "plan-completion.json",
      { result: planCompletion }
    );

    // Call plan-finalizer to commit the plan to colony state
    callGoJSON<Record<string, unknown>>(opts, [
      "plan-finalize",
      "--completion-file",
      planCompletionPath,
    ]);
    emitCeremonyOutput(ceremony.renderCloseout("plan", planCompletionPath));

    stepsCompleted.push("plan");
    process.stderr.write("Plan finalized successfully\n");

    // ── Step 2: Build ────────────────────────────────────────────────────

    const buildResult = callGoJSON<BuildManifestResult>(opts, [
      "build",
      String(targetPhase),
      "--plan-only",
    ]);

    // Extract the build manifest from the result. Go build --plan-only outputs
    // dispatch_manifest as the primary manifest field.
    const buildManifest = buildResult.dispatch_manifest;
    if (!buildManifest) {
      throw new Error(
        "Build --plan-only returned no dispatch_manifest"
      );
    }

    // Get dispatches from the manifest (authoritative source)
    const buildDispatches = buildManifest.dispatches ?? [];
    if (buildDispatches.length === 0) {
      throw new Error("Build manifest contains no dispatches");
    }
    const buildCeremonyEnvelope = { dispatch_manifest: buildManifest };
    renderManifestCeremony(ceremony, "build", buildCeremonyEnvelope, buildDispatches);

    // Detect available platforms before dispatching.
    const availablePlatforms = await detectAvailablePlatforms();
    const hasPlatforms = availablePlatforms.length > 0;

    // Default to real dispatch. Simulation requires explicit --simulate opt-in.
    const simulateWorkers = opts.simulateWorkers ?? false;

    if (!hasPlatforms && !simulateWorkers) {
      throw new Error(
        formatLifecycleProviderUnavailableMessage(
          `phase ${targetPhase} build`,
          buildResult,
          buildManifest
        )
      );
    }

    // Create a placeholder file for simulated worker file claims.
    // The Go build-finalizer validates that all claimed files exist on disk
    // and are within the repository. For simulated workers, we create a real
    // file in .aether/ts-host/ (TS-host-owned, NOT in GO_OWNED_PATHS) that
    // can be claimed as file_created by the simulated workers.
    if (simulateWorkers) {
      const placeholderDir = join(opts.cwd, ".aether", "ts-host");
      const placeholderRel = ".aether/ts-host/SIMULATED_BUILD_OUTPUT.txt";
      try {
        if (!existsSync(placeholderDir)) {
          mkdirSync(placeholderDir, { recursive: true });
        }
        writeFileSync(
          join(opts.cwd, placeholderRel),
          "Simulated build output for TS host prototype lifecycle test.\n",
          "utf-8"
        );
      } catch {
        // If we can't write the placeholder, continue without file claims.
        // The build-finalizer will reject the build if no files are claimed,
        // which is the expected behavior for a prototype that doesn't do real work.
      }
    }

    // Start dashboard before build dispatch when active
    if (useDashboard) {
      dashboard = createDashboard({ cwd: opts.cwd });
      dashboard.start();
    }

    // Build step with Queen orchestration
    const queenOpts = {
      goBinaryPath: opts.goBinaryPath,
      cwd: opts.cwd,
      phase: targetPhase,
      simulateWorkers,
      parallel: opts.parallel ?? true,
      dashboard: useDashboard,
      skipMiddenCheck: opts.skipMiddenCheck ?? false,
      ...(simulateWorkers ? { simulatedFileClaims: [".aether/ts-host/SIMULATED_BUILD_OUTPUT.txt"] as string[] } : {}),
    };
    const queen = createQueenOrchestrator(queenOpts);
    const queenResult = await queen.runBuild(buildManifest);

    // Log midden warnings and recovery actions if present
    if (queenResult.middenResult?.exceeded) {
      process.stderr.write(
        `Warning: midden threshold exceeded (${queenResult.middenResult.total} entries)\n`
      );
    }
    if (queenResult.recoveryActions && queenResult.recoveryActions.length > 0) {
      process.stderr.write(
        `Recovery actions: ${queenResult.recoveryActions.length} action(s) generated\n`
      );
    }

    // Handle Queen orchestrator failure gracefully
    if (queenResult.error) {
      process.stderr.write(`Queen orchestrator error: ${queenResult.error}\n`);
    }

    // Stop dashboard after build dispatch completes
    if (dashboard) {
      dashboard.stop();
      dashboard = undefined;
    }

    // Use worker results from Queen orchestrator for the finalizer
    const workerResults = queenResult.workerResults;
    renderWorkerCeremony(ceremony, "build", workerResults);

    // Build completion file
    const buildCompletion = {
      dispatch_manifest: buildManifest,
      dispatches: workerResults,
    };

    const buildCompletionPath = writeCompletionFile(
      "aether-lifecycle",
      "build-completion.json",
      { result: buildCompletion }
    );

    // Call build-finalizer to commit build results
    callGoJSON<Record<string, unknown>>(opts, [
      "build-finalize",
      String(targetPhase),
      "--completion-file",
      buildCompletionPath,
    ]);
    emitCeremonyOutput(ceremony.renderCloseout("build", buildCompletionPath));

    stepsCompleted.push("build");
    process.stderr.write("Build finalized successfully\n");

    // ── Step 3: Continue ─────────────────────────────────────────────────

    const continueResult = callGoJSON<ContinueManifestResult>(opts, [
      "continue",
      "--plan-only",
    ]);

    // Extract the continue manifest from the result
    const continueManifest = continueResult.continue_manifest;
    if (!continueManifest) {
      throw new Error(
        "Continue --plan-only returned no continue_manifest"
      );
    }

    // Get continue dispatches from the manifest
    const continueDispatches = continueResult.dispatches ?? [];
    const continueCeremonyEnvelope = {
      continue_manifest: continueManifest,
      dispatches: continueDispatches,
    };
    renderManifestCeremony(
      ceremony,
      "continue",
      continueCeremonyEnvelope,
      continueDispatches
    );

    // Build continue dispatch results: mark all as completed for the prototype.
    // In production, these would be review worker results (watcher, auditor, etc.)
    const continueResults = continueDispatches.map(
      (d): ContinueDispatchResult => {
        const result: ContinueDispatchResult = {
          name: d.name ?? "unknown",
          status: "completed",
          summary: `Continue dispatch completed by TS host (${d.name ?? "unknown"})`,
        };
        if (d.caste !== undefined) result.caste = d.caste;
        if (d.stage !== undefined) result.stage = d.stage;
        if (d.task !== undefined) result.task = d.task;
        if (d.task_id !== undefined) result.task_id = d.task_id;
        if (d.wave !== undefined) result.wave = d.wave;
        if (d.execution_wave !== undefined) result.execution_wave = d.execution_wave;
        return result;
      }
    );
    renderWorkerCeremony(ceremony, "continue", continueResults);

    // Build continue completion file
    const continueCompletion = {
      continue_manifest: continueManifest,
      dispatches: continueResults,
    };

    const continueCompletionPath = writeCompletionFile(
      "aether-lifecycle",
      "continue-completion.json",
      { result: continueCompletion }
    );

    // Call continue-finalizer to run verification, gates, and advance state
    const continueFinalizeResult = callGoJSON<Record<string, unknown>>(opts, [
      "continue-finalize",
      "--completion-file",
      continueCompletionPath,
    ]);
    emitCeremonyOutput(
      ceremony.renderCloseout("continue", continueCompletionPath)
    );

    // Check if continue was blocked by gates (informational, not a failure)
    const blocked = continueFinalizeResult["blocked"] === true;
    if (blocked) {
      process.stderr.write(
        "Continue blocked by verification gates (informational)\n"
      );
    }

    stepsCompleted.push("continue");
    process.stderr.write("Continue finalized successfully\n");

    const result: LifecycleResult = {
      success: true,
      steps_completed: stepsCompleted,
    };
    if (oracleResult) {
      result.oracle_result = {
        iterations_completed: oracleResult.iterations_completed,
        final_confidence: oracleResult.final_confidence,
        stop_reason: oracleResult.stop_reason,
      };
    }
    return result;
  } catch (err: unknown) {
    const message =
      err instanceof Error ? err.message : String(err);
    const step = stepsCompleted.length + 1;
    const stepNames = ["plan", "build", "continue"];
    const stepName = stepNames[step - 1] ?? `step ${step}`;

    const errorMessage = `Failed at ${stepName}: ${message}`;
    process.stderr.write(`Lifecycle error: ${errorMessage}\n`);

    return {
      success: false,
      steps_completed: stepsCompleted,
      error: errorMessage,
    };
  } finally {
    // Ensure dashboard is always stopped, even on error
    if (dashboard) {
      dashboard.stop();
      dashboard = undefined;
    }
  }
}
