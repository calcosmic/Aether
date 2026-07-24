/**
 * Oracle iteration runner and loop controller.
 *
 * Provides:
 * - runOracleIteration: simulate a single research pass
 * - runOracleLoop: run iterations until max or target confidence reached
 * - createOracleState: create initial OracleState
 * - createOraclePlan: create initial OraclePlan
 * - oracleReadyForCompletion: check if target confidence is met
 * - snapshotOracleProgress: capture progress metrics
 * - oracleProgressedSince: check if progress improved since a snapshot
 */

import type {
  OraclePlan,
  OracleQuestion,
  OracleState,
  OracleWorkerFinding,
  OracleWorkerResponse,
} from "./types.js";
import { evaluateConfidence } from "./evaluateConfidence.js";
import {
  buildResearchPlan,
  selectNextQuestion,
} from "./buildResearchPlan.js";

/**
 * Simulate a single oracle research iteration.
 *
 * 1. Select next question
 * 2. Simulate a worker response
 * 3. Update state and plan
 * 4. Return updated state, plan, and result
 */
export async function runOracleIteration(
  query: string,
  iteration: number,
  state: OracleState,
  plan: OraclePlan,
): Promise<{ state: OracleState; plan: OraclePlan; result: OracleWorkerResponse }> {
  const question = selectNextQuestion(plan, state);

  // Simulate worker response
  const simulatedConfidence = Math.min(
    state.targetConfidence,
    50 + iteration * 5,
  );
  const finding: OracleWorkerFinding = {
    text: `Simulated finding for "${question.text}" (query: ${query})`,
    evidence: [
      {
        title: `Evidence-${iteration}`,
        location: "src/oracle/runOracleIteration.ts",
        type: "codebase",
      },
    ],
  };
  const result: OracleWorkerResponse = {
    questionID: question.id,
    status: "answered",
    confidence: simulatedConfidence,
    summary: `Simulated finding for ${question.text}`,
    findings: [finding],
    gaps: [],
    contradictions: [],
    recommendation: "Continue research",
  };

  // Update plan: mark question answered, add finding, record iteration touch
  const updatedQuestions: OracleQuestion[] = plan.questions.map((q) => {
    if (q.id === question.id) {
      const newFinding = {
        text: finding.text,
        sourceIDs: [`sim-${iteration}`],
        iteration,
        blocker: false,
      };
      return {
        ...q,
        status: "answered" as const,
        confidence: simulatedConfidence,
        keyFindings: [...q.keyFindings, newFinding],
        iterationsTouched: [...q.iterationsTouched, iteration],
      };
    }
    return q;
  });

  const now = new Date().toISOString();
  const updatedPlan: OraclePlan = {
    ...plan,
    questions: updatedQuestions,
    lastUpdated: now,
  };

  const overallConfidence = evaluateConfidence(updatedQuestions);

  const updatedState: OracleState = {
    ...state,
    iteration,
    overallConfidence,
    lastUpdated: now,
    activeQuestionID: question.id,
    activeQuestionText: question.text,
  };

  return { state: updatedState, plan: updatedPlan, result };
}

/**
 * Run the oracle loop for a given topic.
 *
 * Iterates until maxIterations or targetConfidence is reached.
 */
export async function runOracleLoop(
  topic: string,
  options?: { maxIterations?: number; targetConfidence?: number; depth?: string },
): Promise<{ state: OracleState; plan: OraclePlan; iterationsRun: number; status: string }> {
  const depthConfig = options?.depth
    ? { maxIterations: options.maxIterations ?? 8, targetConfidence: options.targetConfidence ?? 75 }
    : { maxIterations: options?.maxIterations ?? 8, targetConfidence: options?.targetConfidence ?? 75 };

  let state = createOracleState(topic, {
    maxIterations: depthConfig.maxIterations,
    targetConfidence: depthConfig.targetConfidence,
    depth: options?.depth,
  });
  let plan = createOraclePlan(topic, depthConfig.targetConfidence);

  for (let i = 1; i <= depthConfig.maxIterations; i++) {
    const result = await runOracleIteration(topic, i, state, plan);
    state = result.state;
    plan = result.plan;
    if (state.overallConfidence >= depthConfig.targetConfidence) {
      return { state, plan, iterationsRun: i, status: "complete" };
    }
  }

  return { state, plan, iterationsRun: depthConfig.maxIterations, status: "max-iterations" };
}

/**
 * Create an initial OracleState with sensible defaults.
 */
export function createOracleState(
  topic: string,
  options?: { maxIterations?: number; targetConfidence?: number; depth?: string },
): OracleState {
  const now = new Date().toISOString();
  const depthCfg = options?.depth
    ? undefined
    : undefined;
  const maxIterations = options?.maxIterations ?? 8;
  const targetConfidence = options?.targetConfidence ?? 75;
  return {
    version: "1.1",
    topic,
    scope: "both",
    template: "custom",
    phase: "survey",
    iteration: 0,
    maxIterations,
    targetConfidence,
    overallConfidence: 0,
    startedAt: now,
    lastUpdated: now,
    status: "active",
    strategy: "adaptive",
    focusAreas: [],
    platform: "claude",
    stopReason: "",
    summary: "",
    activeQuestionID: "",
    activeQuestionText: "",
    activeAttempt: 0,
    activeReasoning: "",
    activeTimeoutSec: 300,
    activeElapsedSec: 0,
    activeStartedAt: now,
    activeDeadlineAt: now,
    lastArtifactPath: "",
    openGaps: [],
    contradictions: [],
    recommendation: "",
    controllerPID: 0,
    depth: options?.depth ?? "standard",
    novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
  };
}

/**
 * Create an initial OraclePlan with generated questions.
 */
export function createOraclePlan(
  topic: string,
  targetConfidence: number,
): OraclePlan {
  return buildResearchPlan(topic, targetConfidence);
}

/**
 * Check if the oracle is ready for completion.
 */
export function oracleReadyForCompletion(
  _plan: OraclePlan,
  state: OracleState,
): boolean {
  return state.overallConfidence >= state.targetConfidence;
}

/**
 * Capture a progress snapshot.
 */
export function snapshotOracleProgress(
  plan: OraclePlan,
  state: OracleState,
): { answered: number; touched: number; findings: number; confidence: number } {
  const answered = plan.questions.filter((q) => q.status === "answered").length;
  const touched = plan.questions.filter((q) => q.iterationsTouched.length > 0).length;
  const findings = plan.questions.reduce(
    (sum, q) => sum + q.keyFindings.length,
    0,
  );
  return {
    answered,
    touched,
    findings,
    confidence: state.overallConfidence,
  };
}

/**
 * Check if progress has improved since a previous snapshot.
 */
export function oracleProgressedSince(
  before: { answered: number; touched: number; findings: number; confidence: number },
  plan: OraclePlan,
  state: OracleState,
): boolean {
  const current = snapshotOracleProgress(plan, state);
  return (
    current.answered > before.answered ||
    current.touched > before.touched ||
    current.findings > before.findings ||
    current.confidence > before.confidence
  );
}
