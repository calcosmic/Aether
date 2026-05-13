/**
 * Type definitions for the Queen orchestrator module.
 *
 * The Queen reads manifest recommendations, enforces the Builder-Probe Lock,
 * derives workflow patterns from dispatch composition, checks midden thresholds,
 * and delegates escalation recovery to Go CLI commands.
 */

import type { BuildDispatch, WorkerResult } from "../types.js";

// ---------------------------------------------------------------------------
// Queen recommendation types
// ---------------------------------------------------------------------------

/** A recommendation produced by the Queen for a build phase. */
export interface QueenRecommendation {
  /** Suggested review depth (e.g., "fast", "standard", "final-review"). */
  review_depth: string;
  /** Human-readable reason for the recommendation. */
  reason: string;
}

/** Execution policy derived from the Queen's recommendation. */
export interface QueenExecutionPolicy {
  /** Verification depth for the build (e.g., "Fast", "Standard", "Heavy"). */
  verification_depth: string;
  /** Review depth for the build (e.g., "fast", "standard", "final-review"). */
  review_depth: string;
}

// ---------------------------------------------------------------------------
// Workflow pattern types
// ---------------------------------------------------------------------------

/**
 * Workflow patterns the Queen can derive from dispatch composition.
 *
 * - SPBV: Standard Plan-Build-Verify (default)
 * - Investigate-Fix: Bug investigation and repair
 * - Deep Research: Oracle-driven research with no builders
 * - Refactor: Code restructuring without test castes
 * - Compliance: Security, audit, or accessibility work
 * - Documentation Sprint: Chronicler-led documentation work
 */
export type WorkflowPattern =
  | "SPBV"
  | "Investigate-Fix"
  | "Deep Research"
  | "Refactor"
  | "Compliance"
  | "Documentation Sprint";

// ---------------------------------------------------------------------------
// Builder-Probe Lock types
// ---------------------------------------------------------------------------

/** Result of applying the Builder-Probe Lock to worker results. */
export interface BuilderProbeLockResult {
  /** The (possibly downgraded) worker results. */
  results: WorkerResult[];
  /** Whether any builder statuses were downgraded. */
  downgraded: boolean;
  /** Human-readable summary of lock application. */
  summary: string;
}

// ---------------------------------------------------------------------------
// Midden check types
// ---------------------------------------------------------------------------

/** Result of checking the midden threshold. */
export interface MiddenCheckResult {
  /** Whether the threshold was exceeded. */
  exceeded: boolean;
  /** Total number of midden entries. */
  total: number;
  /** Categories and their counts. */
  categories: Record<string, number>;
  /** Threshold that was checked against. */
  threshold: number;
}

// ---------------------------------------------------------------------------
// Escalation types
// ---------------------------------------------------------------------------

/** Recovery action for a failed worker. */
export interface RecoveryAction {
  /** Type of recovery action. */
  type: "retry" | "peer_reassign" | "fixer_dispatch" | "escalate";
  /** Name of the worker this action applies to. */
  worker: string;
  /** Optional human-readable reason for the action. */
  reason?: string;
}

/** Classification of a worker failure. */
export type FailureClassification =
  | "recoverable"
  | "blocking"
  | "requires-attempt"
  | "unknown";

// ---------------------------------------------------------------------------
// Orchestrator types
// ---------------------------------------------------------------------------

import type { GoBridgeOptions } from "../go-bridge.js";
import type { WaveResult } from "../wave-orchestrator.js";

/** Options for the Queen orchestrator. */
export interface QueenOrchestratorOptions extends GoBridgeOptions {
  /** Phase number to build. */
  phase: number;
  /** When true, simulate worker execution instead of real dispatch. */
  simulateWorkers?: boolean;
  /** When true (default), workers within the same wave run concurrently. */
  parallel?: boolean;
  /** When true (default when TTY), show the live dashboard during build. */
  dashboard?: boolean;
  /** When true, skip the pre-build midden check. */
  skipMiddenCheck?: boolean;
}

/** Result of running the Queen orchestrator build. */
export interface QueenOrchestratorResult {
  /** Whether the build completed successfully. */
  success: boolean;
  /** Flattened worker results from all waves. */
  workerResults: WorkerResult[];
  /** Derived workflow pattern. */
  pattern: WorkflowPattern;
  /** Queen recommendation for this build. */
  recommendation: QueenRecommendation;
  /** Midden check result (if not skipped). */
  middenResult?: MiddenCheckResult | undefined;
  /** Recovery actions for failed workers (if any). */
  recoveryActions?: RecoveryAction[] | undefined;
  /** Error message if the build failed. */
  error?: string | undefined;
}

/** Queen orchestrator interface. */
export interface QueenOrchestrator {
  /** Run the build for the given manifest. */
  runBuild(manifest: { dispatches?: BuildDispatch[] }): Promise<QueenOrchestratorResult>;
}
