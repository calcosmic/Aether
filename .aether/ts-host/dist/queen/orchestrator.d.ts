/**
 * Queen orchestrator for the TypeScript orchestration host.
 *
 * Drives the build phase by:
 * 1. Checking midden thresholds before build
 * 2. Deriving the workflow pattern from dispatch composition
 * 3. Dispatching workers via wave orchestrator
 * 4. Applying the Builder-Probe Lock
 * 5. Handling wave failures with recovery actions
 * 6. Returning a structured result with pattern, recommendation, and actions
 *
 * Satisfies ORC-06 (Queen orchestrator) and integrates ORC-01 through ORC-05.
 */
import type { BuildDispatch } from "../types.js";
import type { QueenOrchestratorOptions, QueenOrchestratorResult, QueenOrchestrator } from "./types.js";
/**
 * Create a Queen orchestrator for the given options.
 *
 * @param opts - Queen orchestrator options
 * @returns Queen orchestrator with runBuild method
 */
export declare function createQueenOrchestrator(opts: QueenOrchestratorOptions): QueenOrchestrator;
/**
 * Run the build phase under Queen orchestration.
 *
 * Steps:
 * 1. Pre-build midden check (if not skipped)
 * 2. Derive workflow pattern from manifest dispatches
 * 3. Dispatch workers via dispatchWaves
 * 4. Apply Builder-Probe Lock
 * 5. Handle wave failures
 * 6. Build and return result
 *
 * @param opts - Queen orchestrator options
 * @param manifest - Build manifest with dispatches
 * @returns Queen orchestrator result
 */
export declare function runBuild(opts: QueenOrchestratorOptions, manifest: {
    dispatches?: BuildDispatch[];
}): Promise<QueenOrchestratorResult>;
