/**
 * Research confidence loop binding for the TypeScript orchestration host.
 *
 * Binds a research depth ("fast" | "balanced" | "deep" | "exhaustive") to
 * the exact target/iteration pair RESEARCH-08 specifies, packaged as the
 * existing `ConfidenceLoopOptions` shape consumed by `ConfidenceLoop`
 * (confidence-loop.ts). RESEARCH-07 forbids reimplementing that loop — this
 * module only supplies its options.
 */

import type { ConfidenceLoopOptions } from "./confidence-loop.js";

// ---------------------------------------------------------------------------
// Depth -> target/iteration binding (RESEARCH-08 / D-12)
// ---------------------------------------------------------------------------

/** Confidence target and iteration cap for one research depth tier. */
export interface ResearchLoopPreset {
  confidenceTarget: number;
  maxIterations: number;
}

/**
 * RESEARCH-08's four depth tiers, copied deliberately from
 * `planningLoopPreset` (cmd/codex_plan.go:1689-1700) — NOT re-derived from it
 * at runtime. That Go function drives the whole-plan Go-native loop
 * (`codexPlanningLoop`), which has no concept of a single phase; per-phase
 * research needs its own binding even though the numbers happen to match.
 */
const RESEARCH_LOOP_PRESETS: Readonly<Record<string, ResearchLoopPreset>> =
  Object.freeze({
    fast: Object.freeze({ confidenceTarget: 80, maxIterations: 4 }),
    balanced: Object.freeze({ confidenceTarget: 90, maxIterations: 6 }),
    deep: Object.freeze({ confidenceTarget: 95, maxIterations: 8 }),
    exhaustive: Object.freeze({ confidenceTarget: 99, maxIterations: 12 }),
  });

/** The pair used for unknown, empty, or unrecognised depth values. */
const RESEARCH_LOOP_DEFAULT_PRESET: ResearchLoopPreset =
  RESEARCH_LOOP_PRESETS.balanced!;

/**
 * Resolve a research depth string to its RESEARCH-08 confidence
 * target/iteration pair. Normalises with `trim().toLowerCase()`. Any depth
 * outside the four known tiers (including empty/whitespace-only strings)
 * resolves to the balanced pair.
 */
export function researchLoopPreset(depth: string): ResearchLoopPreset {
  const normalised = depth.trim().toLowerCase();
  return RESEARCH_LOOP_PRESETS[normalised] ?? RESEARCH_LOOP_DEFAULT_PRESET;
}

/**
 * Resolve a research depth to a full `ConfidenceLoopOptions` object, ready
 * to construct a `ConfidenceLoop` for a research iteration. Spreads the
 * resolved preset pair and sets the supplied worker budget.
 */
export function researchLoopOptions(
  depth: string,
  totalBudget: number
): ConfidenceLoopOptions {
  const preset = researchLoopPreset(depth);
  return {
    confidenceTarget: preset.confidenceTarget,
    maxIterations: preset.maxIterations,
    totalBudget,
  };
}
