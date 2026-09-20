---
phase: 164-research-feeds-planning
plan: 03
subsystem: ts-host research confidence
tags: [confidence-loop, research-scoring, D-11, RESEARCH-07, RESEARCH-08]
dependency-graph:
  requires: []
  provides:
    - "researchLoopPreset / researchLoopOptions (RESEARCH-08 depth binding)"
    - "ResearchConfidenceEvaluator (D-11 evidence-based research scorer)"
  affects:
    - "future host.ts wiring of a research-specific ConfidenceLoop construction site"
tech-stack:
  added: []
  patterns:
    - "research-flavoured sibling of ConfidenceEvaluator (not reuse), mirroring base/bonus/penalty/clamp shape"
key-files:
  created:
    - .aether/ts-host/src/research-confidence.ts
    - .aether/ts-host/test/research-confidence.test.ts
  modified: []
decisions:
  - "Depth->target/iteration numbers are copied from planningLoopPreset (cmd/codex_plan.go), not re-derived at runtime -- that Go function drives the whole-plan loop, which has no per-phase concept"
  - "ResearchConfidenceEvaluator is a new sibling class, not a ConfidenceEvaluator extension/reuse -- its bonuses (test pass rate, files touched, blockers) are worker-claims-shaped and a research artifact has none of those signals"
metrics:
  duration: "~35 min"
  completed: 2026-08-02
---

# Phase 164 Plan 03: Research Confidence Loop Binding + Evidence Scorer Summary

One-liner: RESEARCH-08 depth-to-budget binding plus a reproducible, evidence-weighted scorer for phase research artifacts, both landing in a new self-contained `research-confidence.ts` module that supplies (but never modifies) the existing `ConfidenceLoop`.

## What Was Built

**Task 1 — `researchLoopPreset` / `researchLoopOptions`:** Binds a research depth string (`"fast" | "balanced" | "deep" | "exhaustive"`) to the exact RESEARCH-08 confidence-target/max-iteration pairs (80/4, 90/6, 95/8, 99/12), normalising with `trim().toLowerCase()` and defaulting any unrecognised value to the balanced pair. `researchLoopOptions` packages the resolved preset into a `ConfidenceLoopOptions` object (the existing interface from `confidence-loop.ts`) plus a caller-supplied `totalBudget`, ready to construct a `ConfidenceLoop` instance without reimplementing it (RESEARCH-07).

**Task 2 — `ResearchConfidenceEvaluator`:** A reproducible scorer that grades a phase research markdown file against the six-section contract `renderPhaseResearchBrief` writes (`cmd/phase_research.go:124-134`). The formula (base 20, up to +30 for filled sections, up to +25 for cited Key Patterns/Gotchas bullets, up to +15 for verified Files-to-Study paths, +10/-up-to-20 self-assessment blend, clamped 0-100) satisfies D-11: checkable evidence dominates 70 of the 100-point range, and a worker's self-reported gap count only ever nudges the score, never sets it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Installed missing `.aether/ts-host` dependencies**
- **Found during:** Task 1 verification (`npm run typecheck`)
- **Issue:** The worktree had no `node_modules` under `.aether/ts-host`, so `tsc --noEmit` failed on unrelated pre-existing files (`js-yaml`, `boxen`, `ora`, `figlet`, `log-update` type declarations missing) — none of it caused by this plan's changes.
- **Fix:** Ran `npm install` inside `.aether/ts-host`. `node_modules` is gitignored, so nothing extra was committed.
- **Files modified:** None (dependency install only)
- **Commit:** N/A (no tracked file changes)

No other deviations — plan executed as written.

## Threat Flags

None. The plan's own `<threat_model>` (T-164-07, T-164-08, T-164-09) already covers the only security-relevant surface this plan introduces (worker-authored markdown parsing and `fs.existsSync` on worker-supplied paths), and all three mitigations were implemented exactly as specified:
- T-164-07: self-assessment can only add up to 10 points or subtract, never substitute for the 70-point evidence-dominated score.
- T-164-08: every Files-to-Study path is resolved against `repoRoot` with `path.resolve`/`path.relative`, and any path that escapes `repoRoot` is skipped before the existence check; only a boolean is returned.
- T-164-09: `splitSections` is a single forward pass with no backtracking-prone regex, and Files-to-Study existence checks are capped at the first 50 bullets.

## Verification

- `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence.test.ts` — 17/17 tests pass (9 preset/options + 8 evaluator)
- `npm --prefix .aether/ts-host run typecheck` — exits 0
- `npm --prefix .aether/ts-host run test:all` — 535/535 tests pass (no regressions)
- `git diff --stat .aether/ts-host/src/confidence-loop.ts` — empty (file untouched, per RESEARCH-07)
- `grep -c 'Math.random\|Date.now\|new Date' research-confidence.ts` (excluding comments) — 0

## Self-Check: PASSED

- FOUND: .aether/ts-host/src/research-confidence.ts (304 lines)
- FOUND: .aether/ts-host/test/research-confidence.test.ts (296 lines)
- FOUND commit e7f88c53: feat(164-03): bind research depth to RESEARCH-08 target/iteration pairs
- FOUND commit 0f3d11c5: feat(164-03): add reproducible evidence-based research scorer
