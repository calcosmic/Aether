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
import { callGoJSON } from "./go-bridge.js";
import type { GoBridgeOptions } from "./go-bridge.js";
import { HOST_COMMANDS, type ParsedHostArgs } from "./command-registry.js";
import { dispatchWorkers } from "./worker-dispatch.js";
import type { PlanningDispatch } from "./types.js";
export { buildHostGoArgs } from "./command-registry.js";
export type { ParsedHostArgs } from "./command-registry.js";
/** Test-only: inject a mock callGoJSON. */
export declare function __setCallGoJSON(fn: typeof callGoJSON): void;
/** Test-only: restore the real callGoJSON. */
export declare function __restoreCallGoJSON(): void;
type Platform = "codex" | "claude" | "opencode";
type DetectAvailablePlatforms = () => Promise<Platform[]>;
type PreflightWorkerPlatform = (platform: Platform, cwd: string) => Promise<void>;
/** Test-only: inject a mock dispatchWorkers. */
export declare function __setDispatchWorkers(fn: typeof dispatchWorkers): void;
/** Test-only: restore the real dispatchWorkers. */
export declare function __restoreDispatchWorkers(): void;
/** Test-only: inject a mock detectAvailablePlatforms. */
export declare function __setDetectAvailablePlatforms(fn: DetectAvailablePlatforms): void;
/** Test-only: restore the real detectAvailablePlatforms. */
export declare function __restoreDetectAvailablePlatforms(): void;
/** Test-only: inject a mock worker provider preflight. */
export declare function __setPreflightWorkerPlatform(fn: PreflightWorkerPlatform): void;
/** Test-only: restore the real worker provider preflight. */
export declare function __restorePreflightWorkerPlatform(): void;
/** Restore all test mocks at once. */
export declare function __restoreAllMocks(): void;
export { runDispatchedBuildCommand, runDispatchedPlanCommand, runDispatchedContinueCommand, runDryRunDispatchedCommand };
/** Parse command-line arguments for the TS host. */
export declare function parseArgs(argv: string[]): ParsedHostArgs;
type HostInjectedDispatchFields = {
    hive_section?: string;
    task_brief?: string;
};
type PlanDispatchLike = PlanningDispatch & HostInjectedDispatchFields;
/**
 * Run the dry-run path for any dispatched command: fetch the manifest,
 * render ceremony, show DRY RUN badge, exit without dispatching workers.
 */
declare function runDryRunDispatchedCommand(bridge: GoBridgeOptions, parsed: ParsedHostArgs, definition: typeof HOST_COMMANDS[number]): Promise<void>;
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
    /** Phase IDs escalated to Oracle. Always empty here \u2014 Plan 08 populates it. */
    escalations: number[];
}
/**
 * Give the plan path a real confidence loop (RESEARCH-07). Constructs one
 * `ConfidenceLoop` per approved research phase \u2014 never a batch average \u2014 and
 * iterates each phase's Scout until its depth-bound target is met or its
 * iteration budget runs out (RESEARCH-08 / D-12), printing a ceremony line
 * every iteration (D-09) and an early-accept prompt at most once per phase
 * when progress stalls or nears target (D-10).
 */
export declare function runResearchConfidenceLoop(bridge: GoBridgeOptions, parsed: ParsedHostArgs, researchDispatches: PlanDispatchLike[]): Promise<ResearchLoopSummary>;
/**
 * Run the dispatched build pipeline: fetch manifest, dispatch workers,
 * write completion file, call finalizer, render ceremony.
 *
 * Now iteration-aware: after each dispatch wave, evaluate confidence.
 * If confidence is too low and budget/iterations remain, re-dispatch
 * with failure feedback injected into worker task briefs.
 */
declare function runDispatchedBuildCommand(bridge: GoBridgeOptions, parsed: ParsedHostArgs, definition: typeof HOST_COMMANDS[number]): Promise<void>;
/**
 * Run the dispatched plan pipeline: fetch plan manifest, dispatch planning
 * workers (Scout/Route-Setter), write completion file, call plan-finalizer.
 */
declare function runDispatchedPlanCommand(bridge: GoBridgeOptions, parsed: ParsedHostArgs): Promise<void>;
/**
 * Run the dispatched continue pipeline: fetch continue manifest, dispatch
 * review workers, write completion file, call continue-finalizer.
 */
declare function runDispatchedContinueCommand(bridge: GoBridgeOptions, parsed: ParsedHostArgs): Promise<void>;
