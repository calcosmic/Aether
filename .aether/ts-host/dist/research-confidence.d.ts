/**
 * Research confidence loop binding and evidence-based research scorer.
 *
 * Two pieces the per-phase research confidence loop (RESEARCH-07) needs
 * before it can be wired into the TS host's plan-command handling:
 *
 * 1. `researchLoopPreset` / `researchLoopOptions` — bind a research depth
 *    ("fast" | "balanced" | "deep" | "exhaustive") to the exact
 *    target/iteration pair RESEARCH-08 specifies, packaged as the existing
 *    `ConfidenceLoopOptions` shape consumed by `ConfidenceLoop`
 *    (confidence-loop.ts). RESEARCH-07 forbids reimplementing that loop —
 *    this module only supplies its options.
 *
 * 2. `ResearchConfidenceEvaluator` — a reproducible, evidence-based scorer
 *    for a phase research artifact (the six-section markdown file written by
 *    `renderPhaseResearchBrief`, cmd/phase_research.go:124-134). Satisfies
 *    D-11: the score is dominated by checkable evidence (filled sections,
 *    verified citations, verified file paths) and blended with — never
 *    replaced by — the researcher's own self-reported gap count.
 */
import type { ConfidenceLoopOptions } from "./confidence-loop.js";
/** Confidence target and iteration cap for one research depth tier. */
export interface ResearchLoopPreset {
    confidenceTarget: number;
    maxIterations: number;
}
/**
 * Resolve a research depth string to its RESEARCH-08 confidence
 * target/iteration pair. Normalises with `trim().toLowerCase()`. Any depth
 * outside the four known tiers (including empty/whitespace-only strings)
 * resolves to the balanced pair.
 */
export declare function researchLoopPreset(depth: string): ResearchLoopPreset;
/**
 * Resolve a research depth to a full `ConfidenceLoopOptions` object, ready
 * to construct a `ConfidenceLoop` for a research iteration. Spreads the
 * resolved preset pair and sets the supplied worker budget.
 */
export declare function researchLoopOptions(depth: string, totalBudget: number): ConfidenceLoopOptions;
/** Input to `ResearchConfidenceEvaluator.evaluate`. */
export interface ResearchEvidenceInput {
    /** The full markdown text of a phase-N-research.md artifact. */
    markdown: string;
    /** Repo root that Files-to-Study paths are resolved against. */
    repoRoot: string;
    /** The researcher's own count of self-identified gaps. Default: 0. */
    selfAssessedGaps?: number;
}
/** Evaluated research confidence with a diagnosable breakdown. */
export interface EvaluatedResearchConfidence {
    score: number;
    sectionsFilled: number;
    sectionsTotal: number;
    citationsVerified: number;
    citationsTotal: number;
    filesVerified: number;
    filesTotal: number;
    gaps: number;
    source: "research-evidence";
}
export declare const RESEARCH_BASE_SCORE = 20;
export declare const SECTION_BONUS_MAX = 30;
export declare const CITATION_BONUS_MAX = 25;
export declare const FILES_BONUS_MAX = 15;
export declare const SELF_ASSESSMENT_BONUS = 10;
export declare const GAP_PENALTY = 5;
export declare const GAP_PENALTY_CAP = 20;
/**
 * Reproducible, evidence-based scorer for a phase research artifact.
 *
 * Not a reuse of `ConfidenceEvaluator` — that class's bonuses are
 * worker-claims-shaped (test pass rate, files touched, blockers), which a
 * research artifact has none of. This is a research-flavoured sibling that
 * mirrors its base/bonus/penalty/clamp shape instead.
 *
 * Deterministic: no `Date`, no `Math.random`, no network, no ordering
 * dependence on `Set`/`Map` iteration. The only external read is
 * `fs.existsSync`, deterministic for a fixed tree.
 */
export declare class ResearchConfidenceEvaluator {
    evaluate(input: ResearchEvidenceInput): EvaluatedResearchConfidence;
}
