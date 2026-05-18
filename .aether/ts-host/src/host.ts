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

import { callGoJSON, discoverGoBinary, writeCompletionFile, approvedCompletionDirPrefix } from "./go-bridge.js";
import type { GoBridgeOptions } from "./go-bridge.js";
import {
  buildHostGoArgs,
  getHostCommandDefinition,
  listHostCommandDefinitions,
  HOST_COMMANDS,
  type ParsedHostArgs,
} from "./command-registry.js";
import { runGoJSONCommand } from "./go-command.js";
import { dispatchWorkers, toWorkerResults, type DispatchOptions } from "./worker-dispatch.js";
import { detectAvailablePlatforms, formatPlatformUnavailableMessage } from "./platform-dispatcher.js";
import {
  createCeremonyAdapter,
  type CeremonyAdapter,
  type CeremonyWorkflow,
} from "./ceremony-adapter.js";

export { buildHostGoArgs } from "./command-registry.js";
export type { ParsedHostArgs } from "./command-registry.js";

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

// Mutable references for dispatch worker injection (testing).
let _dispatchWorkersRef = dispatchWorkers;
let _detectAvailablePlatformsRef = detectAvailablePlatforms;

/** Test-only: inject a mock dispatchWorkers. */
export function __setDispatchWorkers(fn: typeof dispatchWorkers): void {
  _dispatchWorkersRef = fn;
}

/** Test-only: restore the real dispatchWorkers. */
export function __restoreDispatchWorkers(): void {
  _dispatchWorkersRef = dispatchWorkers;
}

/** Test-only: inject a mock detectAvailablePlatforms. */
export function __setDetectAvailablePlatforms(fn: typeof detectAvailablePlatforms): void {
  _detectAvailablePlatformsRef = fn;
}

/** Test-only: restore the real detectAvailablePlatforms. */
export function __restoreDetectAvailablePlatforms(): void {
  _detectAvailablePlatformsRef = detectAvailablePlatforms;
}

/** Restore all test mocks at once. */
export function __restoreAllMocks(): void {
  __restoreCallGoJSON();
  __restoreDispatchWorkers();
  __restoreDetectAvailablePlatforms();
}
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
      "  --synthetic            Forward Go synthetic mode for plan/build/continue\n" +
      "  --no-dashboard         Disable live dashboard, use plain text output\n" +
      "  --skip-midden-check    Skip pre-build midden threshold check\n" +
      "  --skip-watchers        Skip continue watcher workers when Go allows it\n" +
      "  --refresh              Refresh an existing plan\n" +
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
  [key: string]: unknown;
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
  dispatch_manifest?: {
    dispatches?: BuildDispatchLike[];
    [key: string]: unknown;
  };
  dispatches?: BuildDispatchLike[];
  provider_diagnostics?: string;
  [key: string]: unknown;
}

interface BuildDispatchLike {
  stage?: string;
  wave?: number;
  execution_wave?: number;
  caste?: string;
  name?: string;
  task?: string;
  status?: string;
  summary?: string;
  task_id?: string;
  skill_section?: string;
  [key: string]: unknown;
}

interface PlanManifestResult {
  plan_manifest?: Record<string, unknown>;
  planning_manifest?: Record<string, unknown>;
  dispatches?: PlanDispatchLike[];
  [key: string]: unknown;
}

interface PlanDispatchLike {
  name?: string;
  caste?: string;
  stage?: string;
  task?: string;
  task_id?: string;
  wave?: number;
  execution_wave?: number;
  [key: string]: unknown;
}

interface ContinueManifestResult {
  continue_manifest?: Record<string, unknown>;
  dispatches?: ContinueDispatchLike[];
  phase?: number;
  [key: string]: unknown;
}

interface ContinueDispatchLike {
  name?: string;
  caste?: string;
  stage?: string;
  task?: string;
  task_id?: string;
  wave?: number;
  execution_wave?: number;
  [key: string]: unknown;
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

/**
 * Run the dispatched build pipeline: fetch manifest, dispatch workers,
 * write completion file, call finalizer, render ceremony.
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
  const dispatches: BuildDispatchLike[] = buildManifest.dispatches ?? buildResult.dispatches ?? [];
  if (dispatches.length === 0) {
    throw new Error("Build manifest contains no dispatches. Nothing to build.");
  }

  // Step 2: Render spawn-plan and wave-start ceremony
  const ceremonyEnvelope = { dispatch_manifest: buildManifest };
  renderManifestCeremony(ceremony, "build", ceremonyEnvelope, dispatches);

  // Step 3: Check available platforms (unless simulating)
  if (!parsed.simulate) {
    const available = await _detectAvailablePlatformsRef();
    if (available.length === 0) {
      const diagnostic = buildResult.provider_diagnostics;
      const msg = diagnostic
        ? `No platform workers available. ${diagnostic}`
        : formatPlatformUnavailableMessage(`build phase ${phase}`);
      throw new Error(msg);
    }
  }

  // Step 4: Skill injection summary (D-07)
  emitSkillSummary(dispatches);

  // Step 5: Dispatch workers
  const dispatchOpts: DispatchOptions = {
    goBinaryPath: bridge.goBinaryPath,
    cwd: bridge.cwd,
    simulateWorkers: parsed.simulate,
  };
  const workerResults = await _dispatchWorkersRef(dispatchOpts, dispatches as any[]);
  const mappedResults = toWorkerResults(dispatches as any[], workerResults);

  // Step 6: Render worker-complete ceremony
  renderWorkerCeremony(ceremony, "build", mappedResults);

  // Step 7: Write completion file and call finalizer
  const completion = {
    dispatch_manifest: buildManifest,
    dispatches: mappedResults,
  };
  const completionPath = writeCompletionFile(
    approvedCompletionDirPrefix("build"),
    "build-completion.json",
    { result: completion }
  );
  _callGoJSONRef(bridge, [
    "build-finalize", phase,
    "--completion-file", completionPath,
  ]);

  // Step 8: Render closeout
  emitCeremonyOutput(ceremony.renderCloseout("build", completionPath));

  // Output result JSON to stdout
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
        if (workflow === "build") {
          await runDispatchedBuildCommand(bridge, parsed, definition);
        } else {
          // Plan and continue pipelines added in Task 2 -- fall back to go-json
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
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "oracle-lifecycle": {
      const topic = positional[0] || "auto";
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
