/**
 * Queen orchestrator module — barrel exports.
 *
 * Re-exports all public types and functions from the Queen subsystem.
 */

export type {
  QueenRecommendation,
  QueenExecutionPolicy,
  WorkflowPattern,
  BuilderProbeLockResult,
  MiddenCheckResult,
  RecoveryAction,
  FailureClassification,
  QueenOrchestratorOptions,
  QueenOrchestratorResult,
  QueenOrchestrator,
} from "./types.js";

export {
  deriveWorkflowPattern,
  mapVerificationDepth,
  formatQueenRecommendation,
  deriveExecutionPolicy,
} from "./workflow-patterns.js";

export {
  applyBuilderProbeLock,
  hasProbeVerification,
  isBuilderProbeLockSatisfied,
} from "./builder-probe-lock.js";

export {
  checkMiddenThreshold,
  formatMiddenSummary,
} from "./midden-check.js";

export {
  classifyFailure,
  handleWaveFailures,
  formatRecoveryActions,
} from "./escalation.js";

export {
  createQueenOrchestrator,
  runBuild,
} from "./orchestrator.js";
