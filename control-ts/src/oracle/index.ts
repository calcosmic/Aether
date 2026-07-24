/**
 * Oracle module barrel export.
 *
 * Re-exports all oracle types, confidence evaluation,
 * research plan building, and iteration/loop runners.
 */

// Types
export type {
  OracleState,
  OraclePlan,
  OracleQuestion,
  OracleFinding,
  OracleSource,
  OracleWorkerResponse,
  OracleWorkerFinding,
  OracleWorkerEvidence,
  OracleDepthConfig,
  OracleScopeProfile,
  OracleRubricEntry,
  OracleGapEntry,
  OracleEvidenceEntry,
} from "./types.js";

// Confidence evaluation
export { evaluateConfidence, buildOracleRubric } from "./evaluateConfidence.js";

// Research plan building
export {
  buildResearchPlan,
  selectNextQuestion,
  identifyGaps,
  collectEvidence,
  buildSynthesizedPrompt,
  resolveOracleDepth,
  resolveOracleScope,
  inferOracleTemplate,
} from "./buildResearchPlan.js";

// Iteration and loop
export {
  runOracleIteration,
  runOracleLoop,
  createOracleState,
  createOraclePlan,
  oracleReadyForCompletion,
  snapshotOracleProgress,
  oracleProgressedSince,
} from "./runOracleIteration.js";
