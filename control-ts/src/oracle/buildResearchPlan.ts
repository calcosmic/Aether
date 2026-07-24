/**
 * Research plan builder and question management for Oracle research.
 *
 * Provides:
 * - buildResearchPlan: generate a default set of questions for a topic
 * - selectNextQuestion: pick the most impactful unanswered question
 * - identifyGaps: list questions below target confidence
 * - collectEvidence: aggregate evidence per question
 * - buildSynthesizedPrompt: markdown synthesis of research state
 * - resolveOracleDepth: map depth label to config
 * - resolveOracleScope: map scope label to profile
 * - inferOracleTemplate: infer template from topic keywords
 */

import type {
  OracleDepthConfig,
  OracleEvidenceEntry,
  OracleGapEntry,
  OraclePlan,
  OracleQuestion,
  OracleScopeProfile,
  OracleState,
} from "./types.js";

/**
 * Build a research plan with at least 5 default questions for the given topic.
 */
export function buildResearchPlan(
  topic: string,
  targetConfidence: number,
): OraclePlan {
  const now = new Date().toISOString();
  const questions: OracleQuestion[] = [
    {
      id: "q1",
      text: `What is the core problem or goal related to "${topic}"?`,
      status: "open",
      confidence: 0,
      keyFindings: [],
      iterationsTouched: [],
    },
    {
      id: "q2",
      text: `What are the known constraints or requirements for "${topic}"?`,
      status: "open",
      confidence: 0,
      keyFindings: [],
      iterationsTouched: [],
    },
    {
      id: "q3",
      text: `What existing solutions or patterns apply to "${topic}"?`,
      status: "open",
      confidence: 0,
      keyFindings: [],
      iterationsTouched: [],
    },
    {
      id: "q4",
      text: `What risks or blockers could affect "${topic}"?`,
      status: "open",
      confidence: 0,
      keyFindings: [],
      iterationsTouched: [],
    },
    {
      id: "q5",
      text: `What are the recommended next steps for "${topic}"?`,
      status: "open",
      confidence: 0,
      keyFindings: [],
      iterationsTouched: [],
    },
  ];
  return {
    version: "1.1",
    sources: {},
    questions,
    createdAt: now,
    lastUpdated: now,
  };
}

/**
 * Select the next most impactful unanswered question.
 *
 * Priority:
 * 1. Questions that have never been touched (iterationsTouched.length === 0)
 * 2. Lowest confidence among non-answered questions
 * 3. If all answered, return a placeholder
 */
export function selectNextQuestion(
  plan: OraclePlan,
  state: OracleState,
): OracleQuestion {
  const unanswered = plan.questions.filter(
    (q) => q.status !== "answered",
  );
  if (unanswered.length === 0) {
    return {
      id: `iteration-${state.iteration}`,
      text: "All oracle questions have been answered.",
      status: "answered",
      confidence: 100,
      keyFindings: [],
      iterationsTouched: [],
    };
  }
  const untouched = unanswered.filter(
    (q) => q.iterationsTouched.length === 0,
  );
  const pool = untouched.length > 0 ? untouched : unanswered;
  const next = pool.reduce((best, q) =>
    q.confidence < best.confidence ? q : best,
  );
  return next;
}

/**
 * Identify gaps in the research plan.
 *
 * Returns questions that haven't reached target confidence or are unanswered,
 * plus any open gaps from the state.
 */
export function identifyGaps(
  plan: OraclePlan,
  state: OracleState,
): OracleGapEntry[] {
  const gaps: OracleGapEntry[] = [];
  for (const q of plan.questions) {
    if (q.confidence < state.targetConfidence || q.status !== "answered") {
      gaps.push({
        questionID: q.id,
        question: q.text,
        reason: q.status === "blocked"
          ? "blocked"
          : q.status === "open"
          ? "unanswered"
          : "below-target-confidence",
        confidence: q.confidence,
      });
    }
  }
  for (const reason of state.openGaps) {
    gaps.push({
      questionID: "state-gap",
      question: "",
      reason,
      confidence: 0,
    });
  }
  return gaps;
}

/**
 * Collect evidence aggregated per question.
 */
export function collectEvidence(
  plan: OraclePlan,
  _state: OracleState,
): OracleEvidenceEntry[] {
  return plan.questions.map((q) => {
    const sourceIDs = new Set<string>();
    for (const f of q.keyFindings) {
      for (const sid of f.sourceIDs) {
        sourceIDs.add(sid);
      }
    }
    const topSources = sourceIDs.size > 0
      ? Array.from(sourceIDs).slice(0, 3).join(", ")
      : undefined;
    return {
      questionID: q.id,
      summaryCount: q.keyFindings.length,
      sourceCount: sourceIDs.size,
      topSources,
    };
  });
}

/**
 * Build a markdown synthesis of the current research state.
 */
export function buildSynthesizedPrompt(
  plan: OraclePlan,
  state: OracleState,
): string {
  const lines: string[] = [
    `# Oracle Research: ${state.topic}`,
    "",
    `## Research State`,
    `- Topic: ${state.topic}`,
    `- Phase: ${state.phase}`,
    `- Iteration: ${state.iteration} / ${state.maxIterations}`,
    `- Overall Confidence: ${state.overallConfidence}%`,
    `- Target Confidence: ${state.targetConfidence}%`,
    `- Strategy: ${state.strategy}`,
    `- Depth: ${state.depth}`,
    "",
    `## Questions`,
  ];
  for (const q of plan.questions) {
    lines.push(`- [${q.status}] ${q.text} (confidence: ${q.confidence}%)`);
    for (const f of q.keyFindings) {
      lines.push(`  - Finding: ${f.text}`);
    }
  }
  if (state.focusAreas.length > 0) {
    lines.push("");
    lines.push(`## Focus Areas`);
    for (const area of state.focusAreas) {
      lines.push(`- ${area}`);
    }
  }
  return lines.join("\n");
}

/**
 * Resolve a depth label to an OracleDepthConfig.
 */
export function resolveOracleDepth(depth: string): OracleDepthConfig {
  const configs: Record<string, OracleDepthConfig> = {
    quick: {
      maxIterations: 3,
      targetConfidence: 50,
      label: "quick",
      description: "Fast scan with limited iterations",
    },
    balanced: {
      maxIterations: 5,
      targetConfidence: 65,
      label: "balanced",
      description: "Balanced depth and speed",
    },
    standard: {
      maxIterations: 8,
      targetConfidence: 75,
      label: "standard",
      description: "Standard research depth",
    },
    deep: {
      maxIterations: 12,
      targetConfidence: 85,
      label: "deep",
      description: "Deep research with more iterations",
    },
    exhaustive: {
      maxIterations: 15,
      targetConfidence: 90,
      label: "exhaustive",
      description: "Exhaustive research coverage",
    },
    marathon: {
      maxIterations: 20,
      targetConfidence: 95,
      label: "marathon",
      description: "Maximum research depth",
    },
  };
  return configs[depth] ?? configs["standard"];
}

/**
 * Resolve a scope label to an OracleScopeProfile.
 */
export function resolveOracleScope(
  topic: string,
  requested: string,
): OracleScopeProfile {
  const profiles: Record<string, OracleScopeProfile> = {
    repo: {
      scope: "repo",
      label: "Repository Context",
      description: `Focus on repository-specific evidence for "${topic}"`,
      evidenceStrategy: "repo-first",
      includeRepoContext: true,
      includeExternalContext: false,
      includeColonyGoal: true,
      includeRecentLearnings: true,
    },
    web: {
      scope: "web",
      label: "Web Context",
      description: `Focus on external web sources for "${topic}"`,
      evidenceStrategy: "web-first",
      includeRepoContext: false,
      includeExternalContext: true,
      includeColonyGoal: false,
      includeRecentLearnings: false,
    },
    both: {
      scope: "both",
      label: "Combined Context",
      description: `Combine repository and web sources for "${topic}"`,
      evidenceStrategy: "balanced",
      includeRepoContext: true,
      includeExternalContext: true,
      includeColonyGoal: true,
      includeRecentLearnings: true,
    },
  };
  return profiles[requested] ?? profiles["both"];
}

/**
 * Infer the oracle template from topic keywords.
 */
export function inferOracleTemplate(topic: string): string {
  const t = topic.toLowerCase();
  if (t.includes("prd") || t.includes("product requirement")) return "prd";
  if (t.includes("bug") || t.includes("investigate")) return "bug-investigation";
  if (t.includes("architecture") || t.includes("review")) return "architecture-review";
  if (t.includes("eval") || t.includes("compare") || t.includes("vs")) return "tech-eval";
  if (t.includes("research") || t.includes("brief")) return "research-brief";
  return "custom";
}
