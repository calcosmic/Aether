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
import * as fs from "node:fs";
import * as path from "node:path";
/**
 * RESEARCH-08's four depth tiers, copied deliberately from
 * `planningLoopPreset` (cmd/codex_plan.go:1689-1700) — NOT re-derived from it
 * at runtime. That Go function drives the whole-plan Go-native loop
 * (`codexPlanningLoop`), which has no concept of a single phase; per-phase
 * research needs its own binding even though the numbers happen to match.
 */
const RESEARCH_LOOP_PRESETS = Object.freeze({
    fast: Object.freeze({ confidenceTarget: 80, maxIterations: 4 }),
    balanced: Object.freeze({ confidenceTarget: 90, maxIterations: 6 }),
    deep: Object.freeze({ confidenceTarget: 95, maxIterations: 8 }),
    exhaustive: Object.freeze({ confidenceTarget: 99, maxIterations: 12 }),
});
/** The pair used for unknown, empty, or unrecognised depth values. */
const RESEARCH_LOOP_DEFAULT_PRESET = RESEARCH_LOOP_PRESETS.balanced;
/**
 * Resolve a research depth string to its RESEARCH-08 confidence
 * target/iteration pair. Normalises with `trim().toLowerCase()`. Any depth
 * outside the four known tiers (including empty/whitespace-only strings)
 * resolves to the balanced pair.
 */
export function researchLoopPreset(depth) {
    const normalised = depth.trim().toLowerCase();
    return RESEARCH_LOOP_PRESETS[normalised] ?? RESEARCH_LOOP_DEFAULT_PRESET;
}
/**
 * Resolve a research depth to a full `ConfidenceLoopOptions` object, ready
 * to construct a `ConfidenceLoop` for a research iteration. Spreads the
 * resolved preset pair and sets the supplied worker budget.
 */
export function researchLoopOptions(depth, totalBudget) {
    const preset = researchLoopPreset(depth);
    return {
        confidenceTarget: preset.confidenceTarget,
        maxIterations: preset.maxIterations,
        totalBudget,
    };
}
// The six required sections from `renderPhaseResearchBrief`
// (cmd/phase_research.go:124-134) — the artifact contract this scorer grades
// against.
const REQUIRED_SECTIONS = Object.freeze([
    "## Hive Wisdom (Pre-existing Knowledge)",
    "## Key Patterns",
    "## External Context",
    "## Gotchas",
    "## Recommended Approach",
    "## Files to Study",
]);
const CITED_SECTIONS = Object.freeze([
    "## Key Patterns",
    "## Gotchas",
]);
const FILES_TO_STUDY_SECTION = "## Files to Study";
/** Cap on how many Files-to-Study paths get an `fs.existsSync` check (T-164-09). */
const MAX_FILES_CHECKED = 50;
/** Minimum non-whitespace, non-placeholder characters for a section to count as filled. */
const SECTION_MIN_CHARS = 20;
// Scoring constants — declared as named module-level constants exactly like
// confidence-evaluator.ts, so the formula is inspectable and a test can
// assert an exact number.
export const RESEARCH_BASE_SCORE = 20;
export const SECTION_BONUS_MAX = 30;
export const CITATION_BONUS_MAX = 25;
export const FILES_BONUS_MAX = 15;
export const SELF_ASSESSMENT_BONUS = 10;
export const GAP_PENALTY = 5;
export const GAP_PENALTY_CAP = 20;
const SCORE_MIN = 0;
const SCORE_MAX = 100;
/** Strip template placeholder spans (`{...}`) from a section body. */
function stripPlaceholders(text) {
    return text.replace(/\{[^}]*\}/g, "");
}
/**
 * Split markdown into a map of "## Heading" -> body text (everything up to
 * the next "## " heading or end of string). Single pass, no backtracking
 * regex (T-164-09).
 */
function splitSections(markdown) {
    const sections = new Map();
    const headingRe = /^##\s+.+$/gm;
    const matches = [];
    let match;
    while ((match = headingRe.exec(markdown)) !== null) {
        matches.push({ heading: match[0].trim(), index: match.index });
    }
    for (let i = 0; i < matches.length; i++) {
        const current = matches[i];
        const nextIndex = i + 1 < matches.length ? matches[i + 1].index : markdown.length;
        const bodyStart = current.index + current.heading.length;
        const body = markdown.slice(bodyStart, nextIndex);
        sections.set(current.heading, body);
    }
    return sections;
}
/** Extract top-level bullet lines ("- " or "* " prefixed) from a section body. */
function extractBullets(body) {
    return body
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.startsWith("- ") || line.startsWith("* "));
}
/** A bullet is cited when it has `(Source: ...)` with a path-like token or URL. */
function isCited(bullet) {
    const sourceIdx = bullet.indexOf("(Source:");
    if (sourceIdx === -1)
        return false;
    const rest = bullet.slice(sourceIdx);
    const closeIdx = rest.indexOf(")");
    const sourceSpan = closeIdx === -1 ? rest : rest.slice(0, closeIdx + 1);
    if (/https?:\/\//.test(sourceSpan))
        return true;
    // path-like: contains "/" and a "." extension after the last "/"
    const pathMatch = sourceSpan.match(/([^\s()]+\/[^\s()]*\.[A-Za-z0-9]+)/);
    return pathMatch !== null;
}
/** Extract a file path from a "Files to Study" bullet line. */
function extractPath(bullet) {
    return bullet.replace(/^[-*]\s+/, "").trim();
}
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
export class ResearchConfidenceEvaluator {
    evaluate(input) {
        const sections = splitSections(input.markdown);
        // --- Sections filled ---
        let sectionsFilled = 0;
        for (const heading of REQUIRED_SECTIONS) {
            const body = sections.get(heading);
            if (body === undefined)
                continue;
            const cleaned = stripPlaceholders(body).replace(/\s/g, "");
            if (cleaned.length >= SECTION_MIN_CHARS) {
                sectionsFilled++;
            }
        }
        const sectionsTotal = REQUIRED_SECTIONS.length;
        const sectionScore = SECTION_BONUS_MAX * (sectionsFilled / sectionsTotal);
        // --- Citations (Key Patterns + Gotchas bullets) ---
        let citationsTotal = 0;
        let citationsVerified = 0;
        for (const heading of CITED_SECTIONS) {
            const body = sections.get(heading);
            if (body === undefined)
                continue;
            const bullets = extractBullets(body);
            for (const bullet of bullets) {
                citationsTotal++;
                if (isCited(bullet))
                    citationsVerified++;
            }
        }
        const citationScore = citationsTotal > 0
            ? CITATION_BONUS_MAX * (citationsVerified / citationsTotal)
            : 0;
        // --- Files to Study ---
        let filesTotal = 0;
        let filesVerified = 0;
        const filesBody = sections.get(FILES_TO_STUDY_SECTION);
        if (filesBody !== undefined) {
            const bullets = extractBullets(filesBody);
            filesTotal = bullets.length;
            const checkable = bullets.slice(0, MAX_FILES_CHECKED);
            for (const bullet of checkable) {
                const rawPath = extractPath(bullet);
                const resolved = path.resolve(input.repoRoot, rawPath);
                const relative = path.relative(input.repoRoot, resolved);
                // T-164-08: skip any path that escapes repoRoot after resolution.
                const escapesRoot = relative.startsWith("..") || path.isAbsolute(relative);
                if (!escapesRoot && fs.existsSync(resolved)) {
                    filesVerified++;
                }
            }
        }
        const filesScore = filesTotal > 0 ? FILES_BONUS_MAX * (filesVerified / filesTotal) : 0;
        // --- Self-check blend (D-11) ---
        const gaps = input.selfAssessedGaps ?? 0;
        const selfAssessmentScore = gaps === 0 ? SELF_ASSESSMENT_BONUS : 0;
        const gapPenalty = Math.min(gaps * GAP_PENALTY, GAP_PENALTY_CAP);
        // --- Total ---
        let score = RESEARCH_BASE_SCORE +
            sectionScore +
            citationScore +
            filesScore +
            selfAssessmentScore -
            gapPenalty;
        score = Math.max(SCORE_MIN, Math.min(SCORE_MAX, Math.round(score)));
        return {
            score,
            sectionsFilled,
            sectionsTotal,
            citationsVerified,
            citationsTotal,
            filesVerified,
            filesTotal,
            gaps,
            source: "research-evidence",
        };
    }
}
