/**
 * Oracle module tests — confidence evaluation, rubric generation,
 * research plan building, iteration runner, and loop behavior.
 */
import { describe, it, expect } from "vitest";
import type { OracleQuestion, OraclePlan, OracleState } from "../../src/oracle/types.js";
import {
  evaluateConfidence,
  buildOracleRubric,
  clampConfidence,
  inferEvidenceType,
} from "../../src/oracle/evaluateConfidence.js";
import {
  buildResearchPlan,
  selectNextQuestion,
  identifyGaps,
  collectEvidence,
  buildSynthesizedPrompt,
  resolveOracleDepth,
  resolveOracleScope,
  inferOracleTemplate,
} from "../../src/oracle/buildResearchPlan.js";
import {
  runOracleIteration,
  runOracleLoop,
  createOracleState,
  createOraclePlan,
  oracleReadyForCompletion,
  snapshotOracleProgress,
  oracleProgressedSince,
} from "../../src/oracle/runOracleIteration.js";

describe("evaluateConfidence", () => {
  it("returns average confidence across questions", () => {
    const questions: OracleQuestion[] = [
      { id: "q1", text: "Q1", status: "answered", confidence: 80, keyFindings: [], iterationsTouched: [] },
      { id: "q2", text: "Q2", status: "answered", confidence: 60, keyFindings: [], iterationsTouched: [] },
    ];
    expect(evaluateConfidence(questions)).toBe(70);
  });

  it("returns 0 for empty questions", () => {
    expect(evaluateConfidence([])).toBe(0);
  });

  it("clamps confidence to 0-100", () => {
    const questions: OracleQuestion[] = [
      { id: "q1", text: "Q1", status: "answered", confidence: 150, keyFindings: [], iterationsTouched: [] },
      { id: "q2", text: "Q2", status: "answered", confidence: -30, keyFindings: [], iterationsTouched: [] },
    ];
    expect(evaluateConfidence(questions)).toBe(60); // (100 + 0) / 2
  });
});

describe("buildOracleRubric", () => {
  it("returns entries with required fields", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "What is X?", status: "open", confidence: 40, keyFindings: [], iterationsTouched: [1] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 40,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
      status: "active",
      strategy: "adaptive",
      focusAreas: [],
      platform: "claude",
      stopReason: "",
      summary: "",
      activeQuestionID: "q1",
      activeQuestionText: "What is X?",
      activeAttempt: 1,
      activeReasoning: "",
      activeTimeoutSec: 300,
      activeElapsedSec: 0,
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const rubric = buildOracleRubric(plan, state);
    expect(rubric).toHaveLength(1);
    expect(rubric[0]).toMatchObject({
      question_id: "q1",
      question: "What is X?",
      status: "open",
      confidence: 40,
      source_count: 0,
      finding_count: 0,
      iterations_touched: 1,
    });
    expect(rubric[0].assessment).toBeOneOf(["met", "partial", "insufficient"]);
  });

  it("assesses 'met' when confidence >= target", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 90, keyFindings: [], iterationsTouched: [] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 90,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
      status: "active",
      strategy: "adaptive",
      focusAreas: [],
      platform: "claude",
      stopReason: "",
      summary: "",
      activeQuestionID: "q1",
      activeQuestionText: "Q1",
      activeAttempt: 1,
      activeReasoning: "",
      activeTimeoutSec: 300,
      activeElapsedSec: 0,
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const rubric = buildOracleRubric(plan, state);
    expect(rubric[0].assessment).toBe("met");
  });

  it("assesses 'partial' when confidence >= target/2 but < target", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "partial", confidence: 45, keyFindings: [], iterationsTouched: [] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 45,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
      status: "active",
      strategy: "adaptive",
      focusAreas: [],
      platform: "claude",
      stopReason: "",
      summary: "",
      activeQuestionID: "q1",
      activeQuestionText: "Q1",
      activeAttempt: 1,
      activeReasoning: "",
      activeTimeoutSec: 300,
      activeElapsedSec: 0,
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const rubric = buildOracleRubric(plan, state);
    expect(rubric[0].assessment).toBe("partial");
  });

  it("assesses 'insufficient' when confidence < target/2", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "open", confidence: 30, keyFindings: [], iterationsTouched: [] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 30,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
      status: "active",
      strategy: "adaptive",
      focusAreas: [],
      platform: "claude",
      stopReason: "",
      summary: "",
      activeQuestionID: "q1",
      activeQuestionText: "Q1",
      activeAttempt: 1,
      activeReasoning: "",
      activeTimeoutSec: 300,
      activeElapsedSec: 0,
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const rubric = buildOracleRubric(plan, state);
    expect(rubric[0].assessment).toBe("insufficient");
  });
});

describe("clampConfidence", () => {
  it("clamps high values to 100", () => {
    expect(clampConfidence(150)).toBe(100);
  });

  it("clamps low values to 0", () => {
    expect(clampConfidence(-10)).toBe(0);
  });

  it("rounds and keeps valid values", () => {
    expect(clampConfidence(75.4)).toBe(75);
    expect(clampConfidence(75.6)).toBe(76);
  });
});

describe("inferEvidenceType", () => {
  it("returns 'official' for HTTP URLs", () => {
    expect(inferEvidenceType("https://example.com")).toBe("official");
    expect(inferEvidenceType("http://example.com")).toBe("official");
  });

  it("returns 'runtime' for test commands", () => {
    expect(inferEvidenceType("npm test")).toBe("runtime");
    expect(inferEvidenceType("go test ./...")).toBe("runtime");
    expect(inferEvidenceType("vitest run")).toBe("runtime");
  });

  it("returns 'codebase' by default", () => {
    expect(inferEvidenceType("src/main.ts")).toBe("codebase");
    expect(inferEvidenceType("")).toBe("codebase");
  });
});

describe("buildResearchPlan", () => {
  it("returns an OraclePlan with at least 5 questions", () => {
    const plan = buildResearchPlan("test topic", 50);
    expect(plan.questions.length).toBeGreaterThanOrEqual(5);
    for (const q of plan.questions) {
      expect(q.status).toBe("open");
      expect(q.confidence).toBe(0);
      expect(q.keyFindings).toEqual([]);
      expect(q.iterationsTouched).toEqual([]);
    }
  });

  it("generates question IDs like q1, q2, ...", () => {
    const plan = buildResearchPlan("test topic", 50);
    expect(plan.questions[0].id).toBe("q1");
    expect(plan.questions[1].id).toBe("q2");
  });
});

describe("selectNextQuestion", () => {
  it("picks the lowest-confidence unanswered question", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 80, keyFindings: [], iterationsTouched: [1] },
        { id: "q2", text: "Q2", status: "open", confidence: 10, keyFindings: [], iterationsTouched: [] },
        { id: "q3", text: "Q3", status: "open", confidence: 20, keyFindings: [], iterationsTouched: [] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 0,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const next = selectNextQuestion(plan, state);
    expect(next.id).toBe("q2");
  });

  it("returns a placeholder when all questions are answered", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 80, keyFindings: [], iterationsTouched: [1] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 2,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 80,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const next = selectNextQuestion(plan, state);
    expect(next.id).toBe("iteration-2");
    expect(next.text).toBe("All oracle questions have been answered.");
  });
});

describe("identifyGaps", () => {
  it("returns gaps for questions below target confidence", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "open", confidence: 30, keyFindings: [], iterationsTouched: [] },
        { id: "q2", text: "Q2", status: "answered", confidence: 90, keyFindings: [], iterationsTouched: [1] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 60,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: ["missing-context"],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const gaps = identifyGaps(plan, state);
    expect(gaps.some((g) => g.questionID === "q1")).toBe(true);
    expect(gaps.some((g) => g.questionID === "q2")).toBe(false);
    expect(gaps.some((g) => g.reason === "missing-context")).toBe(true);
  });
});

describe("collectEvidence", () => {
  it("aggregates evidence per question", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {
        s1: { url: "https://example.com", title: "Example", type: "official", accessedAt: new Date().toISOString() },
      },
      questions: [
        {
          id: "q1",
          text: "Q1",
          status: "answered",
          confidence: 80,
          keyFindings: [
            { text: "F1", sourceIDs: ["s1"], iteration: 1, blocker: false },
          ],
          iterationsTouched: [1],
        },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 80,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const evidence = collectEvidence(plan, state);
    expect(evidence).toHaveLength(1);
    expect(evidence[0].questionID).toBe("q1");
    expect(evidence[0].summaryCount).toBe(1);
    expect(evidence[0].sourceCount).toBe(1);
  });
});

describe("runOracleIteration", () => {
  it("accepts a query and returns updated state, plan, and result", async () => {
    const state = createOracleState("test topic", { maxIterations: 5, targetConfidence: 80 });
    const plan = createOraclePlan("test topic", 80);
    const result = await runOracleIteration("test query", 1, state, plan);
    expect(result.state.iteration).toBe(1);
    expect(result.result.status).toBeOneOf(["answered", "partial", "blocked"]);
    expect(typeof result.state.overallConfidence).toBe("number");
  });

  it("increments iteration counter", async () => {
    const state = createOracleState("test topic", { maxIterations: 5, targetConfidence: 80 });
    const plan = createOraclePlan("test topic", 80);
    expect(state.iteration).toBe(0);
    const result = await runOracleIteration("test query", 1, state, plan);
    expect(result.state.iteration).toBe(1);
  });
});

describe("runOracleLoop", () => {
  it("runs multiple iterations and stops when maxIterations reached", async () => {
    const result = await runOracleLoop("test topic", { maxIterations: 3, targetConfidence: 95 });
    expect(result.iterationsRun).toBeLessThanOrEqual(3);
    expect(result.iterationsRun).toBeGreaterThanOrEqual(1);
    expect(typeof result.status).toBe("string");
    expect(typeof result.state.overallConfidence).toBe("number");
  });

  it("stops early if target confidence is reached", async () => {
    const result = await runOracleLoop("test topic", { maxIterations: 10, targetConfidence: 30 });
    expect(result.iterationsRun).toBeGreaterThanOrEqual(1);
    expect(result.iterationsRun).toBeLessThanOrEqual(10);
    if (result.state.overallConfidence >= 30) {
      expect(result.status).toBe("complete");
    }
  });
});

describe("createOracleState", () => {
  it("returns a valid initial OracleState with defaults", () => {
    const state = createOracleState("test topic");
    expect(state.version).toBe("1.1");
    expect(state.phase).toBe("survey");
    expect(state.iteration).toBe(0);
    expect(state.status).toBe("active");
    expect(state.strategy).toBe("adaptive");
    expect(state.focusAreas).toEqual([]);
    expect(state.topic).toBe("test topic");
  });

  it("respects provided options", () => {
    const state = createOracleState("test topic", { maxIterations: 15, targetConfidence: 90, depth: "deep" });
    expect(state.maxIterations).toBe(15);
    expect(state.targetConfidence).toBe(90);
    expect(state.depth).toBe("deep");
  });
});

describe("createOraclePlan", () => {
  it("returns a valid initial OraclePlan with empty sources and generated questions", () => {
    const plan = createOraclePlan("test topic", 80);
    expect(plan.version).toBe("1.1");
    expect(Object.keys(plan.sources)).toHaveLength(0);
    expect(plan.questions.length).toBeGreaterThanOrEqual(5);
    expect(plan.createdAt).toBeTruthy();
    expect(plan.lastUpdated).toBeTruthy();
  });
});

describe("oracleReadyForCompletion", () => {
  it("returns true when overall confidence >= target confidence", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 90, keyFindings: [], iterationsTouched: [1] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 90,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    expect(oracleReadyForCompletion(plan, state)).toBe(true);
  });

  it("returns false when overall confidence < target confidence", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "open", confidence: 30, keyFindings: [], iterationsTouched: [] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 30,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    expect(oracleReadyForCompletion(plan, state)).toBe(false);
  });
});

describe("snapshotOracleProgress", () => {
  it("returns a progress snapshot", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 80, keyFindings: [], iterationsTouched: [1] },
        { id: "q2", text: "Q2", status: "open", confidence: 0, keyFindings: [], iterationsTouched: [] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 40,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const snap = snapshotOracleProgress(plan, state);
    expect(snap.answered).toBe(1);
    expect(snap.touched).toBe(1);
    expect(snap.findings).toBe(0);
    expect(snap.confidence).toBe(40);
  });
});

describe("oracleProgressedSince", () => {
  it("returns true if any metric improved", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 80, keyFindings: [], iterationsTouched: [1] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 80,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const before = { answered: 0, touched: 0, findings: 0, confidence: 0 };
    expect(oracleProgressedSince(before, plan, state)).toBe(true);
  });

  it("returns false if no metric improved", () => {
    const plan: OraclePlan = {
      version: "1.1",
      sources: {},
      questions: [
        { id: "q1", text: "Q1", status: "answered", confidence: 80, keyFindings: [], iterationsTouched: [1] },
      ],
      createdAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
    };
    const state: OracleState = {
      version: "1.1",
      topic: "test",
      scope: "repo",
      template: "custom",
      phase: "survey",
      iteration: 1,
      maxIterations: 10,
      targetConfidence: 80,
      overallConfidence: 80,
      startedAt: new Date().toISOString(),
      lastUpdated: new Date().toISOString(),
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
      activeStartedAt: new Date().toISOString(),
      activeDeadlineAt: new Date().toISOString(),
      lastArtifactPath: "",
      openGaps: [],
      contradictions: [],
      recommendation: "",
      controllerPID: 0,
      depth: "standard",
      novelty: { lastKeywords: {}, consecutiveLow: 0, threshold: 3 },
    };
    const before = { answered: 1, touched: 1, findings: 0, confidence: 80 };
    expect(oracleProgressedSince(before, plan, state)).toBe(false);
  });
});

describe("resolveOracleDepth", () => {
  it("maps known depth labels to config", () => {
    const quick = resolveOracleDepth("quick");
    expect(quick.label).toBe("quick");
    expect(quick.maxIterations).toBeGreaterThan(0);

    const deep = resolveOracleDepth("deep");
    expect(deep.label).toBe("deep");
    expect(deep.maxIterations).toBeGreaterThanOrEqual(quick.maxIterations);
  });

  it("returns a sensible default for unknown depth", () => {
    const cfg = resolveOracleDepth("unknown");
    expect(cfg.label).toBe("standard");
    expect(cfg.maxIterations).toBeGreaterThan(0);
  });
});

describe("resolveOracleScope", () => {
  it("maps known scopes to profiles", () => {
    const repo = resolveOracleScope("test", "repo");
    expect(repo.scope).toBe("repo");
    expect(repo.includeRepoContext).toBe(true);

    const web = resolveOracleScope("test", "web");
    expect(web.scope).toBe("web");
    expect(web.includeExternalContext).toBe(true);
  });

  it("defaults to 'both' for unknown scope", () => {
    const both = resolveOracleScope("test", "unknown");
    expect(both.scope).toBe("both");
  });
});

describe("inferOracleTemplate", () => {
  it("infers template from topic keywords", () => {
    expect(inferOracleTemplate("Write a PRD for feature X")).toBe("prd");
    expect(inferOracleTemplate("Investigate bug in login")).toBe("bug-investigation");
    expect(inferOracleTemplate("Review the architecture")).toBe("architecture-review");
    expect(inferOracleTemplate("Evaluate React vs Vue")).toBe("tech-eval");
    expect(inferOracleTemplate("Research brief on AI")).toBe("research-brief");
  });

  it("returns 'custom' when no keyword matches", () => {
    expect(inferOracleTemplate("random topic")).toBe("custom");
  });
});

describe("buildSynthesizedPrompt", () => {
  it("returns a markdown synthesis string", () => {
    const state = createOracleState("test topic");
    const plan = createOraclePlan("test topic", 80);
    const prompt = buildSynthesizedPrompt(plan, state);
    expect(prompt).toContain("test topic");
    expect(prompt).toContain("## Research State");
  });
});
