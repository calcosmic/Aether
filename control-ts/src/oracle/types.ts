/**
 * Oracle domain types — mirror Go oracle loop shapes.
 * These types define the structure of oracle research sessions,
 * questions, findings, sources, and worker responses.
 */

export interface OracleState {
  version: string;
  topic: string;
  scope: string;
  template: string;
  phase: string;
  iteration: number;
  maxIterations: number;
  targetConfidence: number;
  overallConfidence: number;
  startedAt: string;
  lastUpdated: string;
  status: string;
  strategy: string;
  focusAreas: string[];
  platform: string;
  stopReason: string;
  summary: string;
  activeQuestionID: string;
  activeQuestionText: string;
  activeAttempt: number;
  activeReasoning: string;
  activeTimeoutSec: number;
  activeElapsedSec: number;
  activeStartedAt: string;
  activeDeadlineAt: string;
  lastArtifactPath: string;
  openGaps: string[];
  contradictions: string[];
  recommendation: string;
  controllerPID: number;
  depth: string;
  novelty: {
    lastKeywords: Record<string, boolean>;
    consecutiveLow: number;
    threshold: number;
  };
}

export interface OraclePlan {
  version: string;
  sources: Record<string, OracleSource>;
  questions: OracleQuestion[];
  createdAt: string;
  lastUpdated: string;
}

export interface OracleSource {
  url: string;
  title: string;
  type: string;
  accessedAt: string;
}

export interface OracleQuestion {
  id: string;
  text: string;
  status: "open" | "answered" | "partial" | "blocked";
  confidence: number;
  keyFindings: OracleFinding[];
  iterationsTouched: number[];
}

export interface OracleFinding {
  text: string;
  sourceIDs: string[];
  iteration: number;
  blocker: boolean;
}

export interface OracleWorkerResponse {
  questionID: string;
  status: "answered" | "partial" | "blocked";
  confidence: number;
  summary: string;
  findings: OracleWorkerFinding[];
  gaps: string[];
  contradictions: string[];
  recommendation: string;
}

export interface OracleWorkerFinding {
  text: string;
  evidence: OracleWorkerEvidence[];
}

export interface OracleWorkerEvidence {
  title: string;
  location: string;
  type: string;
}

export interface OracleDepthConfig {
  maxIterations: number;
  targetConfidence: number;
  label: string;
  description: string;
}

export interface OracleScopeProfile {
  scope: string;
  label: string;
  description: string;
  evidenceStrategy: string;
  includeRepoContext: boolean;
  includeExternalContext: boolean;
  includeColonyGoal: boolean;
  includeRecentLearnings: boolean;
}

export interface OracleRubricEntry {
  question_id: string;
  question: string;
  status: string;
  confidence: number;
  source_count: number;
  finding_count: number;
  iterations_touched: number;
  assessment: "met" | "partial" | "insufficient";
}

export interface OracleGapEntry {
  questionID: string;
  question: string;
  reason: string;
  confidence: number;
}

export interface OracleEvidenceEntry {
  questionID: string;
  summaryCount: number;
  sourceCount: number;
  topSources?: string;
}
