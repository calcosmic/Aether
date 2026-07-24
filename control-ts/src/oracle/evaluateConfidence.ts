/**
 * Confidence evaluation and rubric generation for Oracle research.
 *
 * Provides:
 * - evaluateConfidence: average confidence across questions
 * - buildOracleRubric: per-question breakdown with assessment
 * - clampConfidence: clamp to 0-100 range
 * - inferEvidenceType: classify evidence by location string
 */

import type {
  OraclePlan,
  OracleQuestion,
  OracleRubricEntry,
  OracleState,
} from "./types.js";

/**
 * Clamp a confidence value to the valid 0-100 range.
 */
export function clampConfidence(value: number): number {
  return Math.max(0, Math.min(100, Math.round(value)));
}

/**
 * Evaluate overall confidence as the average of all question confidences.
 * Returns 0 if there are no questions.
 */
export function evaluateConfidence(questions: OracleQuestion[]): number {
  if (questions.length === 0) return 0;
  const avg =
    questions.reduce((sum, q) => sum + q.confidence, 0) / questions.length;
  return clampConfidence(avg);
}

/**
 * Build a per-question rubric from the current plan and state.
 *
 * Each entry includes:
 * - question_id, question text, status
 * - confidence (clamped)
 * - source_count (unique source IDs across findings)
 * - finding_count (number of key findings)
 * - iterations_touched (count)
 * - assessment: "met" if confidence >= target, "partial" if >= target/2, else "insufficient"
 */
export function buildOracleRubric(
  plan: OraclePlan,
  state: OracleState,
): OracleRubricEntry[] {
  return plan.questions.map((q) => {
    const sourceIDs = new Set<string>();
    let findingCount = 0;
    for (const f of q.keyFindings) {
      findingCount++;
      for (const sid of f.sourceIDs) {
        sourceIDs.add(sid);
      }
    }
    const confidence = clampConfidence(q.confidence);
    const target = state.targetConfidence;
    let assessment: "met" | "partial" | "insufficient";
    if (confidence >= target) {
      assessment = "met";
    } else if (confidence >= target / 2) {
      assessment = "partial";
    } else {
      assessment = "insufficient";
    }
    return {
      question_id: q.id,
      question: q.text,
      status: q.status,
      confidence,
      source_count: sourceIDs.size,
      finding_count: findingCount,
      iterations_touched: q.iterationsTouched.length,
      assessment,
    };
  });
}

/**
 * Infer the evidence type from a location string.
 *
 * - "official" for HTTP/HTTPS URLs
 * - "runtime" for test commands or shell-like patterns
 * - "codebase" as the default
 */
export function inferEvidenceType(location: string): string {
  const loc = location.trim().toLowerCase();
  if (loc.startsWith("http://") || loc.startsWith("https://")) {
    return "official";
  }
  if (
    loc.startsWith("test") ||
    loc.startsWith("npm test") ||
    loc.startsWith("go test") ||
    loc.startsWith("vitest") ||
    loc.startsWith("jest")
  ) {
    return "runtime";
  }
  return "codebase";
}
