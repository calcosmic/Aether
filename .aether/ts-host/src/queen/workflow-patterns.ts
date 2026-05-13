/**
 * Workflow pattern derivation and recommendation formatting for the Queen.
 *
 * Derives a WorkflowPattern from the composition of build dispatches and
 * maps verification/review depth strings to canonical values.
 *
 * Satisfies ORC-01 (workflow pattern derivation) and ORC-02 (Queen recommendation).
 */

import type {
  BuildDispatch,
  QueenRecommendation,
  QueenExecutionPolicy,
  WorkflowPattern,
} from "../types.js";

// ---------------------------------------------------------------------------
// Pattern derivation
// ---------------------------------------------------------------------------

/**
 * Derive a workflow pattern from the set of build dispatches.
 *
 * Rules (evaluated in order):
 * 1. oracle + scout (no builder) → "Deep Research"
 * 2. chaos or tracker present → "Investigate-Fix"
 * 3. weaver present AND no test castes (watcher, probe) → "Refactor"
 * 4. gatekeeper, auditor, or includer present → "Compliance"
 * 5. chronicler present → "Documentation Sprint"
 * 6. default → "SPBV"
 *
 * @param dispatches - Build dispatch entries from the manifest
 * @returns Derived workflow pattern
 */
export function deriveWorkflowPattern(
  dispatches: BuildDispatch[]
): WorkflowPattern {
  const castes = new Set(dispatches.map((d) => d.caste));

  // Rule 1: oracle + scout with no builder → Deep Research
  if (castes.has("oracle") && castes.has("scout") && !castes.has("builder")) {
    return "Deep Research";
  }

  // Rule 2: chaos or tracker → Investigate-Fix
  if (castes.has("chaos") || castes.has("tracker")) {
    return "Investigate-Fix";
  }

  // Rule 3: weaver and no test castes → Refactor
  const testCastes = new Set(["watcher", "probe"]);
  const hasTestCaste = [...castes].some((c) => testCastes.has(c));
  if (castes.has("weaver") && !hasTestCaste) {
    return "Refactor";
  }

  // Rule 4: gatekeeper, auditor, or includer → Compliance
  if (
    castes.has("gatekeeper") ||
    castes.has("auditor") ||
    castes.has("includer")
  ) {
    return "Compliance";
  }

  // Rule 5: chronicler → Documentation Sprint
  if (castes.has("chronicler")) {
    return "Documentation Sprint";
  }

  // Rule 6: default
  return "SPBV";
}

// ---------------------------------------------------------------------------
// Depth mapping
// ---------------------------------------------------------------------------

/**
 * Map a raw verification/review depth string to a canonical value.
 *
 * Canonical values:
 * - "fast" → "Fast"
 * - "standard" → "Standard"
 * - "heavy" | "final-review" → "Heavy"
 *
 * Unknown values pass through unchanged.
 *
 * @param depth - Raw depth string from manifest or recommendation
 * @returns Canonical depth string
 */
export function mapVerificationDepth(depth: string): string {
  const normalized = depth.toLowerCase().trim();
  switch (normalized) {
    case "fast":
      return "Fast";
    case "standard":
      return "Standard";
    case "heavy":
    case "final-review":
      return "Heavy";
    default:
      return depth;
  }
}

// ---------------------------------------------------------------------------
// Recommendation formatting
// ---------------------------------------------------------------------------

/**
 * Format a Queen recommendation as a human-readable string.
 *
 * Format: "{review_depth}: {reason}"
 *
 * @param rec - Queen recommendation
 * @returns Formatted recommendation string
 */
export function formatQueenRecommendation(rec: QueenRecommendation): string {
  return `${rec.review_depth}: ${rec.reason}`;
}

// ---------------------------------------------------------------------------
// Policy derivation
// ---------------------------------------------------------------------------

/**
 * Derive a QueenExecutionPolicy from a recommendation and pattern.
 *
 * @param rec - Queen recommendation
 * @param pattern - Workflow pattern
 * @returns Execution policy with canonical depths
 */
export function deriveExecutionPolicy(
  rec: QueenRecommendation,
  pattern: WorkflowPattern
): QueenExecutionPolicy {
  return {
    verification_depth: mapVerificationDepth(rec.review_depth),
    review_depth: rec.review_depth,
  };
}
