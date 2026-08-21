/**
 * TypeScript orchestration host entry point.
 *
 * Invoked as: node .aether/ts-host/dist/host.js <command> [options]
 *
 * Commands:
 *   colonize   -- Call `aether colonize --plan-only` and print JSON manifest
 *   plan       -- Call `aether plan --plan-only` and print JSON manifest
 *   build <N>  -- Call `aether build N --plan-only` and print JSON manifest
 *   continue   -- Call `aether continue --plan-only` and print JSON manifest
 *   seal       -- Call `aether seal --plan-only` and print JSON manifest
 *   oracle     -- Run the Oracle lifecycle loop
 *   lifecycle  -- Run the experimental simulate-only lifecycle smoke harness
 *   watch      -- Show colony status through the host display surface
 *   swarm      -- Show or plan swarm activity through the host display surface
 *
 * Options:
 *   --cwd <path>  Working directory (default: process.cwd())
 */

import { callGoJSON, discoverGoBinary, writeCompletionFile, approvedCompletionDirPrefix, cleanupCompletionDir } from "./go-bridge.js";
import type { GoBridgeOptions } from "./go-bridge.js";
import {
  buildHostGoArgs,
  getHostCommandDefinition,
  listHostCommandDefinitions,
  HOST_COMMANDS,
  type ParsedHostArgs,
} from "./command-registry.js";
import { runGoJSONCommand } from "./go-command.js";
import {
  dispatchWorkers,
  preflightGoWorkerProvider,
  toWorkerResults,
  type DispatchOptions,
  type DispatchResult,
} from "./worker-dispatch.js";
import { createSpawnOrchestrator, type SpawnOrchestrator } from "./spawn-orchestrator.js";
import {
  createCeremonyAdapter,
  type CeremonyAdapter,
  type CeremonyWorkflow,
  renderDryRunBadge,
} from "./ceremony-adapter.js";
import { ConfidenceLoop, type ConfidenceLoopOptions, type ConfidenceResult } from "./confidence-loop.js";
import { ConfidenceEvaluator, type EvaluatedConfidence, type ConfidenceInput } from "./confidence-evaluator.js";
import {
  ResearchConfidenceEvaluator,
  researchLoopOptions,
  researchLoopPreset,
} from "./research-confidence.js";
import type { WorkerClaims } from "./claims-parser.js";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type {
  BuildDispatch,
  BuildManifest,
  ContinueCompletion,
  ContinueExternalDispatch,
  PlanCompletion,
  PlanningDispatch,
  WorkerResult,
} from "./types.js";

export { buildHostGoArgs } from "./command-registry.js";
export type { ParsedHostArgs } from "./command-registry.js";

// Mutable reference for test injection.
let _callGoJSONRef = callGoJSON;

/** Test-only: inject a mock callGoJSON. */
export function __setCallGoJSON(fn: typeof callGoJSON): void {
	_callGoJSONRef = (<T>(opts: GoBridgeOptions, args: string[]): T => {
		const result = fn<T>(opts, args);
		if (args[0] !== "build-completion-stage") return result;
		const record = result as Record<string, unknown> | undefined;
		if (typeof record?.completion_path === "string") return result;
		const flagIndex = args.indexOf("--completion-file");
		const completionPath = flagIndex >= 0 ? args[flagIndex + 1] : undefined;
		return { completion_path: completionPath } as T;
	}) as typeof callGoJSON;
}

/** Test-only: restore the real callGoJSON. */
export function __restoreCallGoJSON(): void {
  _callGoJSONRef = callGoJSON;
}

// Mutable references for dispatch worker injection (testing).
let _dispatchWorkersRef = dispatchWorkers;
type Platform = "codex" | "claude" | "opencode";
type DetectAvailablePlatforms = () => Promise<Platform[]>;
type PreflightWorkerPlatform = (platform: Platform, cwd: string) => Promise<void>;
let _detectAvailablePlatformsRef: DetectAvailablePlatforms | undefined;
let _preflightWorkerPlatformRef: PreflightWorkerPlatform | undefined;

/** Test-only: inject a mock dispatchWorkers. */
export function __setDispatchWorkers(fn: typeof dispatchWorkers): void {
  _dispatchWorkersRef = fn;
}

/** Test-only: restore the real dispatchWorkers. */
export function __restoreDispatchWorkers(): void {
  _dispatchWorkersRef = dispatchWorkers;
}

/** Test-only: inject a mock detectAvailablePlatforms. */
export function __setDetectAvailablePlatforms(fn: DetectAvailablePlatforms): void {
  _detectAvailablePlatformsRef = fn;
}

/** Test-only: restore the real detectAvailablePlatforms. */
export function __restoreDetectAvailablePlatforms(): void {
  _detectAvailablePlatformsRef = undefined;
}

/** Test-only: inject a mock worker provider preflight. */
export function __setPreflightWorkerPlatform(fn: PreflightWorkerPlatform): void {
  _preflightWorkerPlatformRef = fn;
}

/** Test-only: restore the real worker provider preflight. */
export function __restorePreflightWorkerPlatform(): void {
  _preflightWorkerPlatformRef = undefined;
}

/** Restore all test mocks at once. */
export function __restoreAllMocks(): void {
  __restoreCallGoJSON();
  __restoreDispatchWorkers();
  __restoreDetectAvailablePlatforms();
  __restorePreflightWorkerPlatform();
}

// Test-only: exported runner functions for integration testing.
export { runDispatchedBuildCommand, runDispatchedPlanCommand, runDispatchedContinueCommand, runDryRunDispatchedCommand, toWorkerDispatches };

import { runLifecycle, type LifecycleOptions } from "./lifecycle.js";
import { runOracleLifecycle, type OracleLifecycleOptions } from "./oracle-lifecycle.js";
import { runWatchDisplay, type WatchDisplayOptions } from "./watch-display.js";
import { runSwarmDisplay, type SwarmDisplayOptions } from "./swarm-display.js";
import { createNarrator } from "./narrator.js";
import { startEventBridge } from "./event-bridge.js";

/** Parse command-line arguments for the TS host. */
export function parseArgs(argv: string[]): ParsedHostArgs {
  const args = argv.slice(2); // skip node and script path
  let command = "";
  let cwd = process.cwd();
  let simulate = false;
  let dryRun = false;
  let synthetic = false;
  let noDashboard = false;
  let skipMiddenCheck = false;
  let skipWatchers = false;
  let refresh = false;
  let force = false;
  let forceResurvey = false;
  const tasks: string[] = [];
  let depth: string | undefined = undefined;
  let planningDepth: string | undefined = undefined;
  let verificationDepth: string | undefined = undefined;
  let targetConfidence: string | undefined = undefined;
  let maxIterations: string | undefined = undefined;
  let accept = false;
  let revisionType: string | undefined = undefined;
  let revisionReason: string | undefined = undefined;
  const revisionEvidence: string[] = [];
  let verificationTimeout: string | undefined = undefined;
  let light = false;
  let heavy = false;
  let workerTimeout: string | undefined = undefined;
  let circuitBreakerThreshold: string | undefined = undefined;
  let noSuggest = false;
  let verbose = false;
  const reconcileTasks: string[] = [];
  let noLearn = false;
  let classicCeremony = false;
  let help = false;
  const positional: string[] = [];
  const unknownFlags: string[] = [];

  for (let i = 0; i < args.length; i++) {
    let arg = args[i]!;
    const rawArg = arg;
    let inlineValue: string | undefined;
    if (arg.startsWith("--")) {
      const eq = arg.indexOf("=");
      if (eq > 2) {
        inlineValue = arg.slice(eq + 1);
        arg = arg.slice(0, eq);
      }
    }
    const nextValue = (): string | undefined => {
      if (inlineValue !== undefined) return inlineValue;
      if (i + 1 < args.length) return args[++i]!;
      return undefined;
    };
    const readValue = (flag: string): string | undefined => {
      const value = nextValue();
      if (value === undefined) {
        unknownFlags.push(`${flag} requires a value`);
      }
      return value;
    };
    if (arg === "--cwd") {
      cwd = readValue(arg) ?? cwd;
    } else if (arg === "--simulate") {
      simulate = true;
    } else if (arg === "--dry-run") {
      dryRun = true;
    } else if (arg === "--synthetic") {
      synthetic = true;
    } else if (arg === "--no-dashboard") {
      noDashboard = true;
    } else if (arg === "--skip-midden-check") {
      skipMiddenCheck = true;
    } else if (arg === "--skip-watchers") {
      skipWatchers = true;
    } else if (arg === "--refresh") {
      refresh = true;
    } else if (arg === "--force") {
      force = true;
    } else if (arg === "--force-resurvey") {
      forceResurvey = true;
    } else if (arg === "--task") {
      const value = readValue(arg);
      if (value !== undefined) tasks.push(value);
    } else if (arg === "--depth") {
      depth = readValue(arg);
    } else if (arg === "--planning-depth") {
      planningDepth = readValue(arg);
    } else if (arg === "--verification-depth") {
      verificationDepth = readValue(arg);
    } else if (arg === "--target") {
      targetConfidence = readValue(arg);
    } else if (arg === "--max-iterations") {
      maxIterations = readValue(arg);
    } else if (arg === "--accept") {
      accept = true;
    } else if (arg === "--revision-type") {
      revisionType = readValue(arg);
    } else if (arg === "--revision-reason") {
      revisionReason = readValue(arg);
    } else if (arg === "--revision-evidence") {
      const value = readValue(arg);
      if (value !== undefined) revisionEvidence.push(value);
    } else if (arg === "--verification-timeout") {
      verificationTimeout = readValue(arg);
    } else if (arg === "--light") {
      light = true;
    } else if (arg === "--heavy") {
      heavy = true;
    } else if (arg === "--worker-timeout") {
      workerTimeout = readValue(arg);
    } else if (arg === "--circuit-breaker-threshold") {
      circuitBreakerThreshold = readValue(arg);
    } else if (arg === "--no-suggest") {
      noSuggest = true;
    } else if (arg === "--verbose") {
      verbose = true;
    } else if (arg === "--reconcile-task") {
      const value = readValue(arg);
      if (value !== undefined) reconcileTasks.push(value);
    } else if (arg === "--no-learn") {
      noLearn = true;
    } else if (arg === "--classic-ceremony") {
      classicCeremony = true;
    } else if (arg === "--help" || arg === "-h") {
      help = true;
    } else if (arg.startsWith("-")) {
      unknownFlags.push(rawArg);
    } else if (!command) {
      command = rawArg;
    } else {
      positional.push(rawArg);
    }
  }

  return {
    command,
    cwd,
    simulate,
    dryRun,
    synthetic,
    noDashboard,
    skipMiddenCheck,
    skipWatchers,
    refresh,
    force,
    forceResurvey,
    tasks,
    depth,
    planningDepth,
    verificationDepth,
    targetConfidence,
    maxIterations,
    accept,
    revisionType,
    revisionReason,
    revisionEvidence,
    verificationTimeout,
    light,
    heavy,
    workerTimeout,
    circuitBreakerThreshold,
    noSuggest,
    verbose,
    reconcileTasks,
    noLearn,
    classicCeremony,
    help,
    positional,
    unknownFlags,
  };
}

function printUsage(): void {
  const commandLines = listHostCommandDefinitions()
    .map((definition) => `  ${definition.usage.padEnd(14)} ${definition.description}`)
    .join("\n");
  process.stderr.write(
    "Usage: host <command> [options]\n\n" +
      "Commands:\n" +
      commandLines + "\n\n" +
      "Options:\n" +
      "  --cwd <path>           Working directory\n" +
      "  --simulate             Run in simulation mode (no real worker spawning)\n" +
      "  --dry-run              Preview ceremony output without spawning workers\n" +
      "  --synthetic            Forward Go synthetic mode for plan/build/continue\n" +
      "  --no-dashboard         Disable live dashboard, use plain text output\n" +
      "  --skip-midden-check    Skip pre-build midden threshold check\n" +
      "  --skip-watchers        Skip continue watcher workers when Go allows it\n" +
      "  --refresh              Refresh an existing plan\n" +
      "  --revision-type TYPE   manual, user_feedback, research, verification_failure, or scope_change\n" +
      "  --revision-reason TEXT Explain why completed work requires replanning\n" +
      "  --revision-evidence PATH  Repository-relative evidence file (repeatable)\n" +
      "  --force                Forward Go force aliases for plan/build\n" +
      "  --force-resurvey       Refresh colonize survey artifacts\n" +
      "  --task <id>            Limit build dispatch to a task id (repeatable)\n" +
      "  --depth <level>        fast | balanced | deep | exhaustive\n" +
      "  --planning-depth <lvl> light | standard | deep\n" +
      "  --verification-depth <lvl> light | standard | heavy\n" +
      "  --target <n>           Planning confidence target 70-99\n" +
      "  --max-iterations <n>   Planning iteration budget 2-12\n" +
      "  --accept               Accept current best plan below target\n" +
      "  --verification-timeout <dur> Override continue verification timeout\n" +
      "  --light                Force light review\n" +
      "  --heavy                Force heavy review\n" +
      "  --worker-timeout <dur> Override per-worker timeout (e.g. 5m, 15m)\n" +
      "  --circuit-breaker-threshold <n> Forward build circuit-breaker threshold\n" +
      "  --no-suggest           Skip build suggestion analysis\n" +
      "  --verbose              Forward verbose build output mode\n" +
      "  --reconcile-task <id>  Mark continue task reconciliation (repeatable)\n" +
      "  --no-learn             Disable continue learning capture when supported\n" +
      "  --classic-ceremony     Use continue's heavy visible review manifest\n"
  );
}

// ---------------------------------------------------------------------------
// Ceremony helpers (shared with lifecycle.ts pattern)
// ---------------------------------------------------------------------------

interface CeremonyDispatchLike {
  execution_wave?: number;
  wave?: number;
  skill_section?: string;
  task?: string;
  task_brief?: string;
}

function emitCeremonyOutput(output: string): void {
  if (output.trim() === "") return;
  process.stderr.write(output.endsWith("\n") ? output : `${output}\n`);
}

function ceremonyExecutionWaves(dispatches: CeremonyDispatchLike[]): number[] {
  const waves = new Set<number>();
  for (const dispatch of dispatches) {
    const wave = dispatch.execution_wave ?? dispatch.wave ?? 0;
    if (wave > 0) waves.add(wave);
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

// ---------------------------------------------------------------------------
// Manifest result types (match Go JSON output shapes)
// ---------------------------------------------------------------------------

interface BuildManifestResult {
  dispatch_manifest?: BuildManifest;
  dispatches?: BuildDispatch[];
  provider_diagnostics?: string;
}

type HostInjectedDispatchFields = {
  task_brief?: string;
};

type BuildDispatchLike = BuildDispatch;

type PlanManifestResult = PlanCompletion;

type PlanDispatchLike = PlanningDispatch & HostInjectedDispatchFields;

type ContinueManifestResult = ContinueCompletion;

type ContinueDispatchLike = ContinueExternalDispatch & HostInjectedDispatchFields;

// This function is the Go -> worker fidelity boundary: every field a
// dispatch carries must be explicitly copied through, renamed, or
// consciously classified as unmapped. A field silently dropped here fails
// loudly at Go's ResolvePermissionProfile exact-equality check (CR-01) or
// not at all (CR-02) -- both were true for months because every test that
// touched this function mocked it instead of calling it. `permission_profile`
// must never be invented locally: it is copied through verbatim from the Go
// manifest, never constructed here. test/dispatch-field-fidelity.test.ts
// guards this function directly (no mock) so a newly added Go field that
// isn't mapped here fails that test until someone consciously classifies it.
function toWorkerDispatches(
  dispatches: Array<PlanDispatchLike | ContinueDispatchLike>
): BuildDispatch[] {
  return dispatches.map((dispatch): BuildDispatch => {
    const workerDispatch: BuildDispatch = {
      stage: dispatch.stage ?? "dispatch",
      caste: dispatch.caste,
      name: dispatch.name,
      task: dispatch.task,
      status: dispatch.status,
    };
    if (dispatch.wave !== undefined) workerDispatch.wave = dispatch.wave;
    if (dispatch.execution_wave !== undefined) {
      workerDispatch.execution_wave = dispatch.execution_wave;
    }
    if (dispatch.task_id !== undefined) workerDispatch.task_id = dispatch.task_id;
    if (dispatch.summary !== undefined) workerDispatch.summary = dispatch.summary;
    if (dispatch.blockers !== undefined) workerDispatch.blockers = dispatch.blockers;
    if (dispatch.duration !== undefined) workerDispatch.duration = dispatch.duration;
    if (dispatch.skill_section !== undefined) {
      workerDispatch.skill_section = dispatch.skill_section;
    }
    // Precedence: host-injected `task_brief` (build/continue paths) wins over
    // Go-emitted `brief` (plan-time phase_research dispatches). Only fall
    // back to `brief` when `task_brief` was never injected.
    if (dispatch.task_brief !== undefined) {
      workerDispatch.task_brief = dispatch.task_brief;
    } else if (typeof dispatch.brief === "string" && dispatch.brief !== "") {
      workerDispatch.task_brief = dispatch.brief;
    }
    if (dispatch.permission_profile !== undefined) {
      workerDispatch.permission_profile = dispatch.permission_profile;
    }
    if (dispatch.matched_skills !== undefined) {
      workerDispatch.matched_skills = dispatch.matched_skills;
    }
    if (dispatch.skill_count !== undefined) {
      workerDispatch.skill_count = dispatch.skill_count;
    }
    if (dispatch.colony_skill_count !== undefined) {
      workerDispatch.colony_skill_count = dispatch.colony_skill_count;
    }
    if (dispatch.domain_skill_count !== undefined) {
      workerDispatch.domain_skill_count = dispatch.domain_skill_count;
    }
    return workerDispatch;
  });
}

// ---------------------------------------------------------------------------
// Dispatched runner: real dispatch pipeline for build/plan/continue
// ---------------------------------------------------------------------------

/** Build a skill injection summary line (D-07) */
function emitSkillSummary(dispatches: CeremonyDispatchLike[]): void {
  const skillCount = dispatches.filter(
    (d) => typeof d.skill_section === "string" && d.skill_section.trim() !== ""
  ).length;
  if (skillCount > 0) {
    process.stderr.write(`Injecting ${skillCount} skills into worker prompts.\n`);
  }
}

async function preflightHostWorkerDispatch(
  bridge: GoBridgeOptions,
  context: string,
  fallbackDiagnostic?: string
): Promise<void> {
  // D-06: a skipped preflight must never be silent, on either the
  // compat/test-hook path or the production Go-adapter path below.
  const skipRaw = process.env["AETHER_SKIP_PREFLIGHT"]?.trim().toLowerCase();
  if (skipRaw && ["1", "true", "yes", "on"].includes(skipRaw)) {
    process.stderr.write(
      "Warning: preflight skipped via AETHER_SKIP_PREFLIGHT — provider auth and model config were NOT verified before dispatch\n"
    );
    return;
  }

  // Compatibility-only test hooks. Production always delegates selection and
  // provider preflight to the Go adapter boundary below.
  if (_detectAvailablePlatformsRef) {
    const available = await _detectAvailablePlatformsRef();
    if (available.length === 0) {
      throw new Error(
        fallbackDiagnostic ??
          `Worker dispatch cannot start for ${context.toLowerCase()}; no provider is available.`
      );
    }
    const requested = process.env["AETHER_WORKER_PLATFORM"]?.trim().toLowerCase();
    const selected = requested
      ? available.find((platform) => platform === requested)
      : available[0];
    if (!selected) {
      throw new Error(
        requested
          ? `AETHER_WORKER_PLATFORM is set to ${requested}, but that provider is not available. Available providers: ${available.join(", ")}.`
          : `No selectable worker platform is available. Available providers: ${available.join(", ") || "none"}.`
      );
    }
    if (_preflightWorkerPlatformRef) {
      try {
        await _preflightWorkerPlatformRef(selected, bridge.cwd);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        throw new Error(`${context} cannot start: ${message}`);
      }
    }
    return;
  }
  await preflightGoWorkerProvider(bridge, context);
}

// ---------------------------------------------------------------------------
// Dry-run ceremony preview (HOST-07, D-06)
// ---------------------------------------------------------------------------

/**
 * Run the dry-run path for any dispatched command: fetch the manifest,
 * render ceremony, show DRY RUN badge, exit without dispatching workers.
 */
async function runDryRunDispatchedCommand(
  bridge: GoBridgeOptions,
  parsed: ParsedHostArgs,
  definition: typeof HOST_COMMANDS[number],
): Promise<void> {
  const ceremony = createCeremonyAdapter(bridge);
  const goArgs = buildHostGoArgs(parsed)!;
  const workflow = (definition.ceremonyWorkflow ?? "build") as CeremonyWorkflow;

  // Fetch manifest (read-only)
  const manifestResult = _callGoJSONRef<Record<string, unknown>>(bridge, goArgs);

  // Extract dispatches from manifest for ceremony rendering
  const manifestEnvelope = manifestResult;
  const topLevelDispatches = (manifestResult as Record<string, unknown>)?.dispatches as CeremonyDispatchLike[] | undefined;
  const dispatchManifest = (manifestResult as Record<string, unknown>)?.dispatch_manifest as Record<string, unknown> | undefined;
  const planManifest = (manifestResult as Record<string, unknown>)?.plan_manifest as Record<string, unknown> | undefined;
  const planningManifest = (manifestResult as Record<string, unknown>)?.planning_manifest as Record<string, unknown> | undefined;
  const continueManifest = (manifestResult as Record<string, unknown>)?.continue_manifest as Record<string, unknown> | undefined;
  const dispatches = topLevelDispatches
    ?? dispatchManifest?.dispatches as CeremonyDispatchLike[] | undefined
    ?? planManifest?.dispatches as CeremonyDispatchLike[] | undefined
    ?? planningManifest?.dispatches as CeremonyDispatchLike[] | undefined
    ?? continueManifest?.dispatches as CeremonyDispatchLike[] | undefined
    ?? [];
  const manifestObj = dispatchManifest ?? planManifest ?? planningManifest ?? continueManifest;

  // Hive wisdom is no longer computed or attached here (Phase 190). It
  // arrives exactly once, already embedded in each dispatch's
  // context_capsule by Go's colony-prime capsule -- attaching a second,
  // independently-computed copy here duplicated the same "## HIVE WISDOM
  // (Cross-Colony Patterns)" section in every worker's assembled prompt.

  // Playbook injection removed to match cmd/codex_build.go. Playbooks are
  // orchestrator guidance; appending them to every worker's task_brief told a
  // single Builder "YOU (the Queen) will spawn workers directly" at five times
  // the mass of its actual assignment. Measured before removal: 5,733 of 7,485
  // brief characters. The Go side stopped doing this; this path is the one the
  // /ant-build wrapper actually uses, so it has to stop too or the fix only
  // covers direct CLI use.

  if (manifestObj) {
    const envelope = { ...manifestEnvelope, [`${workflow}_manifest`]: manifestObj };
    renderManifestCeremony(ceremony, workflow, envelope, dispatches);
  } else {
    renderManifestCeremony(ceremony, workflow, manifestEnvelope, dispatches);
  }

  // DRY RUN badge (D-06)
  renderDryRunBadge();

  // Skill injection summary
  emitSkillSummary(dispatches);

  // Output manifest JSON to stdout
  process.stdout.write(JSON.stringify({ ok: true, dry_run: true, manifest: manifestResult }, null, 2) + "\n");
}

// ---------------------------------------------------------------------------
// Iteration ceremony helpers (ITER-06)
// ---------------------------------------------------------------------------

/**
 * Render an iteration marker between dispatch waves.
 *
 * @param result - The ConfidenceResult from the last evaluate call.
 * @param prefix - Optional label prepended before "Iteration" (research path
 *   passes `${scoutName} phase ${id}`). Omitting it produces a byte-identical
 *   line to the build path's original output.
 */
function renderIterationCeremony(result: ConfidenceResult, prefix?: string): void {
  const parts: string[] = [
    `${prefix ? `${prefix} ` : ""}Iteration ${result.iterationCount}:`,
    `confidence ${result.currentConfidence}%`,
    `(delta ${result.delta >= 0 ? "+" : ""}${result.delta}%,`,
    `budget ${result.budgetRemaining} workers remaining)`,
  ];
  if (result.stopReason) {
    parts.push(`[${result.stopReason}]`);
  }
  emitCeremonyOutput(`\u2500\u2500 ${parts.join(" ")} \u2500\u2500`);
}

/** Render the final iteration stop reason. */
function renderIterationComplete(stopReason: string): void {
  emitCeremonyOutput(`\u2500\u2500 Iteration complete: ${stopReason} \u2500\u2500`);
}

// ---------------------------------------------------------------------------
// Research confidence loop (RESEARCH-07 / RESEARCH-08)
// ---------------------------------------------------------------------------

/**
 * Default worker budget for a single research phase's ConfidenceLoop.
 * Mirrors the build path's `?? 20` fallback (see spawnBudget in
 * `runDispatchedBuildCommand`) \u2014 the depth preset's maxIterations (4/6/8/12,
 * always well under 20) is the binding cap in practice; this budget is the
 * secondary safety net T-164-16 requires.
 */
const RESEARCH_LOOP_DEFAULT_BUDGET = 20;

/** Documented range for --target (see host.ts usage text: "70-99"). */
function clampResearchConfidenceTarget(value: number): number {
  return Math.min(99, Math.max(70, value));
}

/** Documented range for --max-iterations (see host.ts usage text: "2-12"). */
function clampResearchMaxIterations(value: number): number {
  return Math.min(12, Math.max(2, value));
}

/** Per-phase outcome reported by `runResearchConfidenceLoop`. */
export interface ResearchLoopPhaseSummary {
  phaseId: number;
  iterations: number;
  finalConfidence: number;
  stopReason: string;
}

/** Summary returned by `runResearchConfidenceLoop`. */
export interface ResearchLoopSummary {
  phases: ResearchLoopPhaseSummary[];
  /**
   * Phase IDs escalated from Scout to Oracle (D-04). A phase appears here at
   * most once: its loop stopped with reason "diminishing_returns" below the
   * confidence target at deep or exhaustive depth, and the escalated Oracle
   * dispatch was sent exactly once \u2014 Oracle's own RALF loop (runOracleLoop)
   * drives the rest, not a second ConfidenceLoop.
   */
  escalations: number[];
}

/** Depths at which a stalled phase is worth the heavier Oracle researcher. */
function isEscalationEligibleDepth(depth: string): boolean {
  const normalised = depth.trim().toLowerCase();
  return normalised === "deep" || normalised === "exhaustive";
}

/**
 * Extract the phase ID a research dispatch targets from its task_id
 * ("plan-research-phase-<ID>", cmd/phase_research.go:90). Returns undefined
 * when the task_id doesn't match the expected shape.
 */
function researchDispatchPhaseId(dispatch: { task_id?: string }): number | undefined {
  const match = /^plan-research-phase-(\d+)$/.exec(dispatch.task_id ?? "");
  if (!match) return undefined;
  return parseInt(match[1]!, 10);
}

/**
 * Read a phase's research artifact from disk. A missing file is treated as
 * empty markdown \u2014 the evidence scorer naturally grades an empty artifact at
 * its base score, no special-casing needed.
 */
function readPhaseResearchMarkdown(cwd: string, phaseId: number): string {
  const filePath = join(cwd, ".aether", "data", "phase-research", `phase-${phaseId}-research.md`);
  try {
    return readFileSync(filePath, "utf8");
  } catch {
    return "";
  }
}

/** Derive a self-assessed gap count from a worker's claimed blockers, if any. */
function selfAssessedGapsFromDispatchResult(result: DispatchResult | undefined): number {
  return result?.blockers?.length ?? 0;
}

/** Per-phase state tracked across dispatch rounds inside the research loop. */
interface ResearchPhaseState {
  dispatch: PlanDispatchLike;
  phaseId: number;
  loop: ConfidenceLoop;
  prompted: boolean;
  lastResult?: ConfidenceResult;
  finalStopReason?: string;
}

/**
 * Give the plan path a real confidence loop (RESEARCH-07). Constructs one
 * `ConfidenceLoop` per approved research phase \u2014 never a batch average \u2014 and
 * iterates each phase's Scout until its depth-bound target is met or its
 * iteration budget runs out (RESEARCH-08 / D-12), printing a ceremony line
 * every iteration (D-09) and an early-accept prompt at most once per phase
 * when progress stalls or nears target (D-10).
 */
export async function runResearchConfidenceLoop(
  bridge: GoBridgeOptions,
  parsed: ParsedHostArgs,
  researchDispatches: PlanDispatchLike[],
): Promise<ResearchLoopSummary> {
  if (researchDispatches.length === 0) {
    return { phases: [], escalations: [] };
  }

  const depth = parsed.depth ?? "balanced";
  const preset = researchLoopPreset(depth);
  const loopOpts: ConfidenceLoopOptions = researchLoopOptions(depth, RESEARCH_LOOP_DEFAULT_BUDGET);
  if (parsed.targetConfidence) {
    const parsedTarget = parseInt(parsed.targetConfidence, 10);
    if (!Number.isNaN(parsedTarget)) {
      loopOpts.confidenceTarget = clampResearchConfidenceTarget(parsedTarget);
    }
  }
  if (parsed.maxIterations) {
    const parsedMax = parseInt(parsed.maxIterations, 10);
    if (!Number.isNaN(parsedMax)) {
      loopOpts.maxIterations = clampResearchMaxIterations(parsedMax);
    }
  }
  const confidenceTarget = loopOpts.confidenceTarget ?? preset.confidenceTarget;
  const escalationEligible = isEscalationEligibleDepth(depth);

  const evaluator = new ResearchConfidenceEvaluator();

  const phases = new Map<number, ResearchPhaseState>();
  for (const dispatch of researchDispatches) {
    const phaseId = researchDispatchPhaseId(dispatch);
    if (phaseId === undefined) continue; // Malformed dispatch; nothing to key the loop on.
    phases.set(phaseId, {
      dispatch,
      phaseId,
      loop: new ConfidenceLoop(loopOpts),
      prompted: false,
    });
  }

  const active = new Map(phases);
  // Phases whose loop stalled below target at deep/exhaustive depth (D-04).
  // Populated inside the round loop below, consumed by the escalation round
  // once every phase has finished its own confidence loop.
  const escalationCandidates: ResearchPhaseState[] = [];

  while (active.size > 0) {
    const activeStates = [...active.values()];
    const roundDispatches = toWorkerDispatches(activeStates.map((state) => state.dispatch));
    const dispatchOpts: DispatchOptions = {
      goBinaryPath: bridge.goBinaryPath,
      cwd: bridge.cwd,
      simulateWorkers: parsed.simulate,
      workflow: "plan",
    };
    const workerResults = await _dispatchWorkersRef(dispatchOpts, roundDispatches);
    const resultByName = new Map(workerResults.map((r) => [r.name, r] as const));

    for (const state of activeStates) {
      const markdown = readPhaseResearchMarkdown(bridge.cwd, state.phaseId);
      const selfAssessedGaps = selfAssessedGapsFromDispatchResult(resultByName.get(state.dispatch.name));
      const evaluated = evaluator.evaluate({
        markdown,
        repoRoot: bridge.cwd,
        selfAssessedGaps,
      });

      const loopResult = state.loop.evaluate(evaluated.score, 1);
      state.lastResult = loopResult;
      renderIterationCeremony(loopResult, `${state.dispatch.name} phase ${state.phaseId}`);

      let stopReason = loopResult.stopReason;
      let finished = !loopResult.shouldContinue;

      // Early-accept prompt (D-10): at most once per phase, evaluated before
      // the --accept override below forces a stop.
      if (!state.prompted) {
        const stalledBelowTarget =
          loopResult.stopReason === "diminishing_returns" && loopResult.currentConfidence < confidenceTarget;
        const nearTarget =
          loopResult.shouldContinue && loopResult.currentConfidence >= confidenceTarget - 5;
        if (stalledBelowTarget || nearTarget) {
          state.prompted = true;
          emitCeremonyOutput(
            `phase ${state.phaseId} research at ${loopResult.currentConfidence}%, ` +
              `gaining ${loopResult.delta}%/iteration \u2014 accept now or keep digging? ` +
              `(re-run with --accept to accept)`
          );
        }
      }

      // Non-interactive accept (D-10): stop every still-active phase after
      // its current iteration. No per-iteration nagging.
      if (parsed.accept && loopResult.shouldContinue) {
        finished = true;
        stopReason = "accepted";
      }

      if (finished) {
        state.finalStopReason = stopReason;
        renderIterationComplete(stopReason);
        active.delete(state.phaseId);

        // D-04: escalate Scout -> Oracle only when the loop truly stalled
        // (diminishing_returns) below target, and only at deep/exhaustive
        // depth. confidence_target_met and max_iterations_met never escalate.
        if (escalationEligible && stopReason === "diminishing_returns" && loopResult.currentConfidence < confidenceTarget) {
          escalationCandidates.push(state);
        }
      }
    }
  }

  // Escalation round (D-04): dispatched once, outside the per-phase
  // ConfidenceLoop above. Oracle owns its own RALF iteration
  // (cmd/oracle_loop.go runOracleLoop) -- nesting a second ConfidenceLoop
  // around it here would mean two drivers for one worker.
  //
  // CR-03: an unwrapped call here previously crashed the whole plan run --
  // any plan-research-escalate failure (a phase Go can't resolve, a
  // subprocess error, a malformed envelope) propagated straight out of
  // runResearchConfidenceLoop and killed the node process. D-08 makes every
  // path through this block non-fatal: research (and its escalation) is
  // enrichment, never a gate. Both the per-candidate call and the final
  // escalation dispatch wave below degrade to a named warning and continue.
  const escalations: number[] = [];
  if (escalationCandidates.length > 0) {
    const escalationDispatches: PlanDispatchLike[] = [];
    for (const state of escalationCandidates) {
      const stalledAt = state.lastResult?.currentConfidence ?? 0;
      emitCeremonyOutput(
        `Oracle escalation: phase ${state.phaseId} research stalled at ${stalledAt}% against a ` +
          `${confidenceTarget}% target — escalating Scout to Oracle`
      );
      try {
        const escalateResult = _callGoJSONRef<{ dispatch: PlanDispatchLike }>(bridge, [
          "plan-research-escalate",
          "--phase", String(state.phaseId),
          "--confidence", String(stalledAt),
          "--target", String(confidenceTarget),
        ]);
        escalationDispatches.push(escalateResult.dispatch);
        escalations.push(state.phaseId);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        emitCeremonyOutput(
          `Warning: Oracle escalation unavailable for phase ${state.phaseId}: ${message}. ` +
            `Planning continues -- research is enrichment, never a gate (D-08).`
        );
        continue;
      }
    }
    if (escalationDispatches.length > 0) {
      const escalationDispatchOpts: DispatchOptions = {
        goBinaryPath: bridge.goBinaryPath,
        cwd: bridge.cwd,
        simulateWorkers: parsed.simulate,
        workflow: "plan",
      };
      try {
        await _dispatchWorkersRef(escalationDispatchOpts, toWorkerDispatches(escalationDispatches));
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        emitCeremonyOutput(
          `Warning: Oracle escalation dispatch failed: ${message}. ` +
            `Planning continues -- research is enrichment, never a gate (D-08).`
        );
      }
    }
  }

  const summaryPhases: ResearchLoopPhaseSummary[] = [...phases.values()].map((state) => ({
    phaseId: state.phaseId,
    iterations: state.lastResult?.iterationCount ?? 0,
    finalConfidence: state.lastResult?.currentConfidence ?? 0,
    stopReason: state.finalStopReason ?? state.lastResult?.stopReason ?? "",
  }));

  return { phases: summaryPhases, escalations };
}

// ---------------------------------------------------------------------------
// dispatchBuildWave: single worker-dispatch iteration
// ---------------------------------------------------------------------------

/** Result of a single dispatch wave. */
interface WaveResult {
  /** Worker results from dispatch mapped to manifest dispatches. */
  mappedResults: WorkerResult[];
  /** Path to the completion file written for this wave. */
  completionPath: string;
  /** The dispatch manifest used for this wave. */
  buildManifest: BuildManifest;
  /** Number of workers dispatched in this wave. */
  workerCount: number;
  /** Aggregated worker claims for confidence evaluation. */
  workerClaims: WorkerClaims[];
}

/**
 * Dispatch a single build iteration, render ceremony, and write its completion
 * packet. The caller finalizes only the last accepted iteration.
 *
 * @param bridge - Go bridge options
 * @param parsed - Parsed host arguments
 * @param ceremony - Ceremony adapter for rendering
 * @param buildManifest - The dispatch manifest
 * @param dispatches - The dispatches for this wave
 * @param spawnOrchestrator - Spawn budget orchestrator (carries cumulative budget)
 * @param iterationFeedback - Optional feedback from a previous iteration to inject into task briefs
 * @returns Wave result with mapped workers, completion path, and claims
 */
async function dispatchBuildWave(
  bridge: GoBridgeOptions,
  parsed: ParsedHostArgs,
  ceremony: CeremonyAdapter,
  buildManifest: BuildManifest,
  dispatches: BuildDispatchLike[],
  spawnOrchestrator: SpawnOrchestrator,
  iterationFeedback?: string,
): Promise<WaveResult> {
  // Inject iteration feedback into task briefs if provided
  if (iterationFeedback) {
    for (const d of dispatches) {
      const existingBrief = d.task_brief ?? d.task ?? "";
      d.task_brief = existingBrief
        ? `${existingBrief}\n\nPrevious iteration feedback:\n${iterationFeedback}`
        : `Previous iteration feedback:\n${iterationFeedback}`;
    }
  }

  // Skill injection summary (D-07)
  emitSkillSummary(dispatches);

  // Dispatch workers
  if (!buildManifest.execution_binding) {
    throw new Error("Build manifest contains no durable execution_binding");
  }
  const dispatchOpts: DispatchOptions = {
    goBinaryPath: bridge.goBinaryPath,
    cwd: bridge.cwd,
    simulateWorkers: parsed.simulate,
    spawnOrchestrator,
    workflow: "build",
    phase: buildManifest.phase,
    executionBinding: buildManifest.execution_binding,
  };
  const workerResults = await _dispatchWorkersRef(dispatchOpts, dispatches);
  const mappedResults = toWorkerResults(dispatches, workerResults);

  // Render worker-complete ceremony
  renderWorkerCeremony(ceremony, "build", mappedResults);

  // Write a completion candidate. Finalization happens once after iteration.
  const completion = {
    dispatch_manifest: buildManifest,
    dispatches: mappedResults,
  };
  const completionPath = writeCompletionFile(
    approvedCompletionDirPrefix("build"),
    "build-completion.json",
    { result: completion }
  );
  // Build worker claims from raw dispatch results for confidence evaluation.
  // Raw results carry extra fields (blockers, test_results) that toWorkerResults drops.
  const workerClaims: WorkerClaims[] = (workerResults as unknown as Record<string, unknown>[]).map((w) => {
    const claim: WorkerClaims = {
      status: (w.status as string) ?? "completed",
    };
    const blockers = w.blockers;
    if (Array.isArray(blockers)) claim.blockers = blockers as string[];
    const filesCreated = w.files_created;
    if (Array.isArray(filesCreated)) claim.files_created = filesCreated as string[];
    const filesModified = w.files_modified;
    if (Array.isArray(filesModified)) claim.files_modified = filesModified as string[];
    const testsWritten = w.tests_written;
    if (Array.isArray(testsWritten)) claim.tests_written = testsWritten as string[];
    // Preserve test_results for ConfidenceEvaluator raw cast
    const testResults = w.test_results;
    if (typeof testResults === "object" && testResults !== null) {
      (claim as unknown as Record<string, unknown>).test_results = testResults;
    }
    return claim;
  });

  return {
    mappedResults,
    completionPath,
    buildManifest,
    workerCount: dispatches.length,
    workerClaims,
  };
}

/**
 * Run the dispatched build pipeline: fetch manifest, dispatch workers,
 * write completion file, call finalizer, render ceremony.
 *
 * Now iteration-aware: after each dispatch wave, evaluate confidence.
 * If confidence is too low and budget/iterations remain, re-dispatch
 * with failure feedback injected into worker task briefs.
 */
async function runDispatchedBuildCommand(
  bridge: GoBridgeOptions,
  parsed: ParsedHostArgs,
  definition: typeof HOST_COMMANDS[number],
): Promise<void> {
  const ceremony = createCeremonyAdapter(bridge);
  const goArgs = buildHostGoArgs(parsed)!;
  const phase = parsed.positional[0] ?? "1";

  // Step 1: Fetch manifest
  const buildResult = _callGoJSONRef<BuildManifestResult>(bridge, goArgs);
  const buildManifest = buildResult.dispatch_manifest;
  if (!buildManifest) {
    throw new Error("Build --plan-only returned no dispatch_manifest. Check colony state and try again.");
  }
  if (buildManifest.orchestrator_boundary_guidance?.active) {
    const guidance = buildManifest.orchestrator_boundary_guidance;
    throw new Error(guidance.summary ?? `Build is paused for unresolved boundary questions. Run ${guidance.next ?? "aether discuss"}.`);
  }
  const issuedDispatches = buildManifest.dispatches ?? buildResult.dispatches ?? [];
  let dispatches: BuildDispatchLike[] = JSON.parse(JSON.stringify(issuedDispatches)) as BuildDispatchLike[];
  if (dispatches.length === 0) {
    throw new Error("Build manifest contains no dispatches. Nothing to build.");
  }

  // Step 1b removed: hive wisdom is no longer separately resolved and
  // attached here (Phase 190). Each dispatch's context_capsule already
  // carries Go's colony-prime "## HIVE WISDOM (Cross-Colony Patterns)"
  // section; recomputing and attaching a second copy duplicated it.

  // Step 1c removed: build playbooks are no longer injected into worker briefs.
  // See the note in runDryRunDispatchedCommand and cmd/codex_build.go.

  // Step 2: Ask Go to select and preflight the provider (unless simulating)
  if (!parsed.simulate) {
    const diagnostic = buildResult.provider_diagnostics;
    await preflightHostWorkerDispatch(
      bridge,
      `Build phase ${phase}`,
      diagnostic ? `No platform workers available. ${diagnostic}` : undefined
    );
  }

  // Step 3: Render spawn-plan and wave-start ceremony
  const ceremonyEnvelope = { dispatch_manifest: buildManifest };
  renderManifestCeremony(ceremony, "build", ceremonyEnvelope, dispatches);

  // Step 5: Initialize spawn budget from manifest QueenSpawnBudget.max_workers (SPAWN-03)
  const spawnBudget =
    buildManifest.queen_execution_policy?.spawn_budget?.max_workers ?? 20;
  const spawnOrchestrator = createSpawnOrchestrator({
    goBinaryPath: bridge.goBinaryPath,
    cwd: bridge.cwd,
    totalBudget: spawnBudget,
    consumedBudget: dispatches.length,
    currentDepth: 1,
  });

  // Step 6: Initialize ConfidenceLoop and ConfidenceEvaluator
  const loopOpts: ConfidenceLoopOptions = {
    totalBudget: spawnBudget,
  };
  if (parsed.maxIterations) {
    loopOpts.maxIterations = parseInt(parsed.maxIterations, 10);
  }
  if (parsed.targetConfidence) {
    loopOpts.confidenceTarget = parseInt(parsed.targetConfidence, 10);
  }
  const confidenceLoop = new ConfidenceLoop(loopOpts);
  const confidenceEvaluator = new ConfidenceEvaluator();

  // Step 7: Iteration loop
  let lastWaveResult: WaveResult | undefined;
  let iterationCount = 0;

  while (true) {
    iterationCount++;

    // Build iteration feedback from previous iteration's blockers
    let iterationFeedback: string | undefined;
    if (lastWaveResult && lastWaveResult.workerClaims.length > 0) {
      const allBlockers: string[] = [];
      for (const claim of lastWaveResult.workerClaims) {
        if (claim.blockers) {
          for (const b of claim.blockers) allBlockers.push(b);
        }
      }
      if (allBlockers.length > 0) {
        iterationFeedback = `Blockers from previous iteration:\n${allBlockers.map((b) => `- ${b}`).join("\n")}`;
      }
    }

    // Dispatch the wave
    const previousWaveResult = lastWaveResult;
    lastWaveResult = await dispatchBuildWave(
      bridge,
      parsed,
      ceremony,
      buildManifest,
      dispatches,
      spawnOrchestrator,
      iterationFeedback,
    );
    if (previousWaveResult && previousWaveResult.completionPath !== lastWaveResult.completionPath) {
      cleanupCompletionDir(previousWaveResult.completionPath);
    }

    // Evaluate confidence from worker claims
    const evaluated: EvaluatedConfidence = confidenceEvaluator.evaluate({
      workerClaims: lastWaveResult.workerClaims,
    });

    // Feed into confidence loop
    const loopResult = confidenceLoop.evaluate(
      evaluated.score,
      lastWaveResult.workerCount,
    );

    // Render iteration ceremony marker
    renderIterationCeremony(loopResult);

    // Check if loop should continue
    if (!loopResult.shouldContinue) {
      renderIterationComplete(loopResult.stopReason);
      break;
    }

    // The Go-issued manifest is immutable for the whole attempt. Subsequent
    // iterations reuse cloned dispatches with feedback injected into briefs.
  }

  if (!lastWaveResult) {
    throw new Error("Build dispatch produced no completion packet.");
  }

  // Step 8: Commit the accepted completion exactly once. Preserve the packet
  // on failure so the same attempt can be retried without rerunning workers.
  const finalCompletionPath = lastWaveResult.completionPath;
	const staged = _callGoJSONRef<{ completion_path: string }>(bridge, [
		"build-completion-stage", phase,
		"--completion-file", finalCompletionPath,
	]);
	const durableCompletionPath = staged.completion_path;
	if (!durableCompletionPath) {
		throw new Error("Go did not return a durable build completion path.");
	}
	_callGoJSONRef(bridge, [
    "build-finalize", phase,
		"--completion-file", durableCompletionPath,
  ]);
  cleanupCompletionDir(finalCompletionPath);
  emitCeremonyOutput(ceremony.renderCloseout("build", finalCompletionPath));

  // Output result JSON to stdout with iteration summary
  const loopState = confidenceLoop.getState();
  const lastResult = lastWaveResult;

  // Determine stop reason from the loop state
  let stopReason = "unknown";
  if (loopState.confidenceHistory.length > 0) {
    // Re-evaluate the stop condition from the last recorded confidence
    const lastConfidence = loopState.confidenceHistory[loopState.confidenceHistory.length - 1]!;
    if (lastConfidence >= (loopOpts.confidenceTarget ?? 80)) {
      stopReason = "confidence_target_met";
    } else if (loopState.iterationCount >= (loopOpts.maxIterations ?? 3)) {
      stopReason = "max_iterations_met";
    } else if (loopOpts.totalBudget && loopOpts.totalBudget > 0 && loopState.budgetRemaining <= 0) {
      stopReason = "budget_exhausted";
    } else {
      stopReason = "diminishing_returns";
    }
  }

  process.stdout.write(JSON.stringify({
    ok: true,
		completion_file: durableCompletionPath,
    iterations: {
      count: loopState.iterationCount,
      final_confidence: loopState.confidenceHistory[loopState.confidenceHistory.length - 1] ?? 0,
      stop_reason: stopReason,
    },
  }, null, 2) + "\n");
}

/**
 * Run the dispatched plan pipeline: fetch plan manifest, dispatch planning
 * workers (Scout/Route-Setter), write completion file, call plan-finalizer.
 */
async function runDispatchedPlanCommand(
  bridge: GoBridgeOptions,
  parsed: ParsedHostArgs,
): Promise<void> {
  const ceremony = createCeremonyAdapter(bridge);
  const goArgs = buildHostGoArgs(parsed)!;

  // Step 1: Fetch plan manifest
  const planResult = _callGoJSONRef<PlanManifestResult>(bridge, goArgs);
  const planManifest = planResult.plan_manifest ?? planResult.planning_manifest;
  if (!planManifest) {
    throw new Error("Plan --plan-only returned no plan manifest. Check colony state and try again.");
  }
  const dispatches: PlanDispatchLike[] = planResult.dispatches ?? [];
  if (dispatches.length === 0) {
    throw new Error("Plan manifest contains no dispatches. Nothing to plan.");
  }

  // Step 1b removed: hive wisdom is no longer separately resolved and
  // attached here (Phase 190). Each dispatch's context_capsule already
  // carries Go's colony-prime "## HIVE WISDOM (Cross-Colony Patterns)"
  // section; recomputing and attaching a second copy duplicated it.

  // Step 1c removed: plan playbooks are no longer injected into worker briefs.
  // See the note in runDryRunDispatchedCommand and cmd/codex_build.go.

  // Step 2: Ask Go to select and preflight the provider (unless simulating)
  if (!parsed.simulate) {
    await preflightHostWorkerDispatch(bridge, "Plan");
  }

  // Step 3: Render spawn-plan and wave-start ceremony
  const ceremonyEnvelope = { plan_manifest: planManifest, dispatches };
  renderManifestCeremony(ceremony, "plan", ceremonyEnvelope, dispatches);

  // Step 4: Research phases get their own confidence loop (RESEARCH-07)
  // before the rest of the wave dispatches. This preserves today's ordering
  // contract -- all wave-1 research completes before the wave-2 Route-Setter
  // -- while giving research its own depth-bound iteration budget (D-12).
  const researchPart = dispatches.filter((d) => d.stage === "phase_research");
  const rest = dispatches.filter((d) => d.stage !== "phase_research");

  let researchSummary: ResearchLoopSummary = { phases: [], escalations: [] };
  let researchMappedResults: WorkerResult[] = [];
  if (researchPart.length > 0) {
    researchSummary = await runResearchConfidenceLoop(bridge, parsed, researchPart);
    researchMappedResults = researchPart.map((dispatch): WorkerResult => {
      const phaseId = researchDispatchPhaseId(dispatch);
      const phaseSummary = researchSummary.phases.find((p) => p.phaseId === phaseId);
      const summary = phaseSummary
        ? `Research phase ${phaseSummary.phaseId} reached ${phaseSummary.finalConfidence}% confidence after ${phaseSummary.iterations} iteration(s) (${phaseSummary.stopReason}).`
        : "Research phase completed.";
      return {
        name: dispatch.name,
        status: "completed",
        summary,
        caste: dispatch.caste,
        task: dispatch.task,
        stage: "phase_research",
      };
    });
  }

  // Step 5: Dispatch the remaining planning workers (Route-Setter, etc.)
  const dispatchOpts: DispatchOptions = {
    goBinaryPath: bridge.goBinaryPath,
    cwd: bridge.cwd,
    simulateWorkers: parsed.simulate,
    workflow: "plan",
  };
  let mappedResults: WorkerResult[] = researchMappedResults;
  if (rest.length > 0) {
    const buildDispatches = toWorkerDispatches(rest);
    const workerResults = await _dispatchWorkersRef(dispatchOpts, buildDispatches);
    mappedResults = [...researchMappedResults, ...toWorkerResults(buildDispatches, workerResults)];
  }

  // Step 6: Render worker-complete ceremony
  renderWorkerCeremony(ceremony, "plan", mappedResults);

  // Step 7: Write completion file and call finalizer
  const completion = {
    plan_manifest: planManifest,
    dispatches: mappedResults,
  };
  const completionPath = writeCompletionFile(
    approvedCompletionDirPrefix("plan"),
    "plan-completion.json",
    { result: completion }
  );
  _callGoJSONRef(bridge, [
    "plan-finalize",
    "--completion-file", completionPath,
  ]);
  cleanupCompletionDir(completionPath);

  // Step 8: Render closeout
  emitCeremonyOutput(ceremony.renderCloseout("plan", completionPath));

  process.stdout.write(JSON.stringify({
    ok: true,
    completion_file: completionPath,
    research_iterations: researchSummary,
  }, null, 2) + "\n");
}

/**
 * Run the dispatched continue pipeline: fetch continue manifest, dispatch
 * review workers, write completion file, call continue-finalizer.
 */
async function runDispatchedContinueCommand(
  bridge: GoBridgeOptions,
  parsed: ParsedHostArgs,
): Promise<void> {
  const ceremony = createCeremonyAdapter(bridge);
  const goArgs = buildHostGoArgs(parsed)!;

  // Step 1: Fetch continue manifest
  const continueResult = _callGoJSONRef<ContinueManifestResult>(bridge, goArgs);
  const continueManifest = continueResult.continue_manifest;
  if (!continueManifest) {
    throw new Error("Continue --plan-only returned no continue manifest. Check colony state and try again.");
  }
  const dispatches: ContinueDispatchLike[] = continueResult.dispatches ?? [];
  if (dispatches.length === 0) {
    throw new Error("Continue manifest contains no dispatches. Nothing to continue.");
  }

  // Step 1b removed: hive wisdom is no longer separately resolved and
  // attached here (Phase 190). Each dispatch's context_capsule already
  // carries Go's colony-prime "## HIVE WISDOM (Cross-Colony Patterns)"
  // section; recomputing and attaching a second copy duplicated it.

  // Step 2: Ask Go to select and preflight the provider (unless simulating)
  if (!parsed.simulate) {
    await preflightHostWorkerDispatch(bridge, "Continue");
  }

  // Step 3: Render spawn-plan and wave-start ceremony
  const ceremonyEnvelope = { continue_manifest: continueManifest, dispatches };
  renderManifestCeremony(ceremony, "continue", ceremonyEnvelope, dispatches);

  // Step 4: Dispatch review workers
  const dispatchOpts: DispatchOptions = {
    goBinaryPath: bridge.goBinaryPath,
    cwd: bridge.cwd,
    simulateWorkers: parsed.simulate,
    workflow: "continue",
  };
  const buildDispatches = toWorkerDispatches(dispatches);
  const workerResults = await _dispatchWorkersRef(dispatchOpts, buildDispatches);
  const mappedResults = toWorkerResults(buildDispatches, workerResults);

  // Step 5: Render worker-complete ceremony
  renderWorkerCeremony(ceremony, "continue", mappedResults);

  // Step 6: Write completion file and call finalizer
  const completion = {
    continue_manifest: continueManifest,
    dispatches: mappedResults,
  };
  const completionPath = writeCompletionFile(
    approvedCompletionDirPrefix("continue"),
    "continue-completion.json",
    { result: completion }
  );
  _callGoJSONRef(bridge, [
    "continue-finalize",
    "--completion-file", completionPath,
  ]);
  cleanupCompletionDir(completionPath);

  // Step 7: Render closeout
  emitCeremonyOutput(ceremony.renderCloseout("continue", completionPath));

  process.stdout.write(JSON.stringify({ ok: true, completion_file: completionPath }, null, 2) + "\n");
}

async function main(): Promise<void> {
  const parsed = parseArgs(process.argv);
  const { command, cwd, simulate, noDashboard, skipMiddenCheck, help, positional } = parsed;

  if (help || !command) {
    printUsage();
    process.exit(help ? 0 : 1);
  }

  const goBinaryPath = discoverGoBinary();
  const bridge: GoBridgeOptions = { goBinaryPath, cwd };
  const definition = getHostCommandDefinition(command);

  if (!definition) {
    process.stderr.write(`Unknown command: ${command}\n`);
    printUsage();
    process.exit(1);
  }

  if (parsed.unknownFlags.length > 0) {
    process.stderr.write(`Error: Unsupported host flag(s): ${parsed.unknownFlags.join(", ")}\n`);
    process.exit(1);
  }

  switch (definition.runner) {
    case "dispatched": {
      const workflow = definition.ceremonyWorkflow ?? "build";
      try {
        if (parsed.dryRun) {
          await runDryRunDispatchedCommand(bridge, parsed, definition);
        } else if (workflow === "build") {
          await runDispatchedBuildCommand(bridge, parsed, definition);
        } else if (workflow === "plan") {
          await runDispatchedPlanCommand(bridge, parsed);
        } else if (workflow === "continue") {
          await runDispatchedContinueCommand(bridge, parsed);
        } else {
          // Fallback for colonize/seal/swarm -- go-json passthrough
          let args: string[];
          try {
            args = buildHostGoArgs(parsed)!;
          } catch (err) {
            const message = err instanceof Error ? err.message : String(err);
            process.stderr.write(`Error: ${message}\n`);
            process.exit(1);
          }
          const result = runGoJSONCommand(bridge, args, _callGoJSONRef);
          process.stdout.write(JSON.stringify(result, null, 2) + "\n");
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Error: ${message}\n`);
        process.exit(1);
      }
      break;
    }

    case "go-json": {
      let args: string[];
      try {
        args = buildHostGoArgs(parsed)!;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Error: ${message}\n`);
        process.exit(1);
      }
      const result = runGoJSONCommand(bridge, args, _callGoJSONRef);
      if (parsed.dryRun) {
        renderDryRunBadge();
        process.stdout.write(JSON.stringify({ ok: true, dry_run: true, manifest: result }, null, 2) + "\n");
      } else {
        process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      }
      break;
    }

    case "oracle-lifecycle": {
      const topic = positional[0] || "auto";

      if (parsed.dryRun) {
        // Dry-run: fetch first iteration manifest, render ceremony, show badge
        const ceremony = createCeremonyAdapter(bridge);
        const manifestResult = _callGoJSONRef<Record<string, unknown>>(bridge, [
          "oracle-iterate",
          "--plan-only",
          "--topic",
          topic,
        ]);
        const state = (manifestResult as Record<string, unknown>)?.iteration_manifest;
        const ceremonyEnvelope = state ? { iteration_manifest: state } : manifestResult;
        emitCeremonyOutput(ceremony.renderSpawnPlan("build" as CeremonyWorkflow, ceremonyEnvelope));
        emitCeremonyOutput(ceremony.renderWaveStart("build" as CeremonyWorkflow, ceremonyEnvelope, 1));
        renderDryRunBadge();
        process.stdout.write(JSON.stringify({ ok: true, dry_run: true, manifest: manifestResult }, null, 2) + "\n");
        break;
      }

      const oracleOpts: OracleLifecycleOptions = {
        goBinaryPath,
        cwd,
        topic,
        simulateWorkers: simulate,
        dashboard: !noDashboard,
      };

      const result = await runOracleLifecycle(oracleOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "lifecycle": {
      const phaseArg = positional[0];
      const oracleTopicArg = positional[1];
      if (phaseArg && isNaN(parseInt(phaseArg, 10))) {
        process.stderr.write("Error: lifecycle phase must be a number\n");
        process.exit(1);
      }
      if (!simulate) {
        process.stderr.write(
          "Error: lifecycle is experimental and simulate-only. Use --simulate for the smoke harness, or run aether plan/build/continue for real orchestration.\n"
        );
        process.exit(1);
      }
      if (simulate) {
        process.stderr.write("Running in simulation mode\n");
      }
      const lifecycleOpts: LifecycleOptions = {
        goBinaryPath,
        cwd,
        simulateWorkers: simulate,
        dashboard: !noDashboard,
        skipMiddenCheck,
      };
      if (phaseArg) {
        lifecycleOpts.phase = parseInt(phaseArg, 10);
      }
      if (oracleTopicArg) {
        lifecycleOpts.runOracle = true;
        lifecycleOpts.oracleTopic = oracleTopicArg;
      }

      const narrator = createNarrator({
        cwd,
        outputMode: process.env["AETHER_OUTPUT_MODE"],
        suppressOutput: !noDashboard && process.stdout.isTTY,
      });

      const eventBridge = await startEventBridge({
        goBinaryPath,
        cwd,
        onEvent: (evt) => {
          narrator.onEvent(evt);
        },
      });

      const result = await runLifecycle(lifecycleOpts);

      await eventBridge.stop();
      narrator.stop();

      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "watch-display": {
      const watchOpts: WatchDisplayOptions = {
        goBinaryPath,
        cwd,
        dashboard: !noDashboard,
      };

      const result = await runWatchDisplay(watchOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "swarm-display": {
      const target = positional[0] || "";
      const swarmOpts: SwarmDisplayOptions = {
        goBinaryPath,
        cwd,
        target,
        dashboard: !noDashboard,
        planOnly: true,
      };

      const result = await runSwarmDisplay(swarmOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }
  }
}

import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import { realpathSync } from "node:fs";

function normalizeEntrypointPath(path: string | undefined): string {
  if (!path) return "";
  try {
    return realpathSync(path);
  } catch {
    return resolve(path);
  }
}

const isMainModule =
  normalizeEntrypointPath(fileURLToPath(import.meta.url)) ===
  normalizeEntrypointPath(process.argv[1]);
if (isMainModule) {
  main().catch((err: unknown) => {
    const message = err instanceof Error ? err.message : String(err);
    process.stderr.write(`Fatal: ${message}\n`);
    process.exit(1);
  });
}
