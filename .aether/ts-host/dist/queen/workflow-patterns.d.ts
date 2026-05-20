/**
 * Workflow pattern derivation and recommendation formatting for the Queen.
 *
 * Derives a WorkflowPattern from the composition of build dispatches and
 * maps verification/review depth strings to canonical values.
 *
 * Satisfies ORC-01 (workflow pattern derivation) and ORC-02 (Queen recommendation).
 */
import type { BuildDispatch } from "../types.js";
import type { QueenRecommendation, QueenExecutionPolicy, WorkflowPattern } from "./types.js";
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
export declare function deriveWorkflowPattern(dispatches: BuildDispatch[]): WorkflowPattern;
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
export declare function mapVerificationDepth(depth: string): string;
/**
 * Format a Queen recommendation as a human-readable string.
 *
 * Format: "{review_depth}: {reason}"
 *
 * @param rec - Queen recommendation
 * @returns Formatted recommendation string
 */
export declare function formatQueenRecommendation(rec: QueenRecommendation): string;
/**
 * Derive a QueenExecutionPolicy from a recommendation and pattern.
 *
 * @param rec - Queen recommendation
 * @param pattern - Workflow pattern
 * @returns Execution policy with canonical depths
 */
export declare function deriveExecutionPolicy(rec: QueenRecommendation, pattern: WorkflowPattern, existingPolicy?: Pick<QueenExecutionPolicy, "spawn_budget">): QueenExecutionPolicy;
