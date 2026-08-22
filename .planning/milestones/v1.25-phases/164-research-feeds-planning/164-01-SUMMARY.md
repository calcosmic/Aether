---
phase: 164-research-feeds-planning
plan: 01
subsystem: cmd (Go runtime — plan-time research dispatch)
tags: [research, survey-context, replan, phase-research]
dependency-graph:
  requires: []
  provides:
    - "cmd/phase_research.go: renderPhaseResearchBrief(goal, candidate, survey)"
    - "cmd/phase_research.go: plannedPhaseResearchDispatches(root, planDepth, goal, candidates, survey, reresearch)"
  affects:
    - "cmd/codex_plan.go: runCodexPlanPlanOnly call site (survey + reresearch threaded)"
tech-stack:
  added: []
  patterns:
    - "Survey-injection shape mirrored verbatim from renderPlanningWorkerBrief (surveyDir/surveyDocs/already-mapped territory)"
key-files:
  created: []
  modified:
    - cmd/phase_research.go
    - cmd/codex_plan.go
    - cmd/phase_research_dispatch_test.go
decisions:
  - "reresearch computed as opts.Refresh && iteration == 1 at the call site — matches D-05's boundary (first iteration of a replan re-researches; later iterations of the same run reuse it)"
  - "Territory Survey section inserted between Mission and Research Areas, never emitted empty — explicit fallback sentence when survey is zero-value"
metrics:
  duration: "~35 min"
  completed: 2026-08-02
---

# Phase 164 Plan 01: Research Feeds Planning — Survey Context and Replan Re-research Summary

Threaded territory survey context into the research Scout's brief and inverted the staleness skip so replans re-research from scratch, closing RESEARCH-05 and RESEARCH-04.

## What Changed

**Task 1 — Survey reaches the research Scout's brief (RESEARCH-05).** `renderPhaseResearchBrief` gained a `survey codexSurveyContext` parameter and a new `## Territory Survey` section (inserted between `## Mission` and `## Research Areas`), rendered by a new `renderPhaseResearchSurveySection` helper that mirrors `renderPlanningWorkerBrief`'s survey-injection shape verbatim: primary survey source directory, survey docs to read first (comma-separated, only when non-empty), and already-mapped languages/frameworks/dependencies with an instruction to spend the research budget on what the survey does not cover. A zero-value survey renders the explicit fallback sentence "No territory survey available — scan the repository directly." instead of an empty section. `plannedPhaseResearchDispatches` gained the same `survey` parameter and threads it through to `renderPhaseResearchBrief`. The call site in `runCodexPlanPlanOnly` (`cmd/codex_plan.go:900`) now passes the `survey` value already loaded in scope.

**Task 2 — Replans re-research from scratch (RESEARCH-04).** `plannedPhaseResearchDispatches` gained a `reresearch bool` parameter as its final argument. The existing `hasWorkerAuthoredResearch` staleness skip is now gated on `!reresearch`: when `reresearch` is true, the `continue` never fires and every candidate phase is re-dispatched, discarding stale findings. The call site computes this as `opts.Refresh && iteration == 1` — the first iteration of a `--refresh` plan run re-researches everything; subsequent iterations of that same run reuse what the first iteration's Scouts wrote (unchanged single-run-once-per-phase behavior). The `Outputs` entry remains the plain `phase-N-research.md` filename in both cases — no timestamped archive sibling is ever introduced; findings overwrite the existing file in place per D-07.

## For Dummies

Two small gaps closed in the machinery that runs before a phase gets planned:

1. **The research robot now gets a map.** Before writing up what it learned about a phase, the research Scout is told what the codebase survey already found (languages, frameworks, dependencies, and which survey documents to read) — so it doesn't waste time re-discovering things Aether already knows, and instead focuses on what's actually new.
2. **Re-planning now actually re-researches.** If you tell Aether to redo its plan from scratch (`--refresh`), it used to quietly reuse old research notes even though they might be stale. Now the first pass of a redo throws out the old notes and researches every phase fresh; only if you run the plan step again within that same redo does it keep what it just wrote.

## Deviations from Plan

None — plan executed exactly as written. One minor detail: the plan's read_first note mentioned "four existing call sites" in the test file for Task 1, but only three call sites to `plannedPhaseResearchDispatches` existed in that file (`TestPlanEmitsPhaseResearchDispatchesFromDraft`, `TestPlanFastPresetSkipsPhaseResearch`, `TestPhaseResearchDispatchedOncePerPhase`); all three were updated. This did not affect scope or correctness — verified via `grep -rn` across `cmd/` that no other call sites exist.

## Known Stubs

None.

## Threat Flags

None — the plan's own threat model (T-164-01, T-164-02, T-164-03) fully covers the new surface introduced (survey interpolation into the brief, survey doc paths disclosure, replan re-dispatch DoS bound). No additional network endpoints, auth paths, file access patterns, or schema changes were introduced beyond what the plan anticipated.

## Verification

- `go test ./cmd/... -run 'TestRenderPhaseResearchBriefIncludesSurvey|TestPlanEmitsPhaseResearchDispatchesFromDraft|TestPlanFastPresetSkipsPhaseResearch' -count=1` — PASS
- `go test ./cmd/... -run 'TestReplanReResearchesPhases|TestPhaseResearchDispatchedOncePerPhase|TestPlanFastPresetSkipsPhaseResearch' -count=1` — PASS
- `go test ./cmd/... -count=1` (full package) — PASS (434s)
- `go build ./cmd/aether` — PASS
- `grep -c 'codexSurveyContext' cmd/phase_research.go` — 3 (>= 2 required)
- `grep -n 'phase-%d-research' cmd/phase_research.go` — only plain (non-timestamped) filename pattern present

## Self-Check: PASSED

- FOUND: cmd/phase_research.go (modified, exists)
- FOUND: cmd/codex_plan.go (modified, exists)
- FOUND: cmd/phase_research_dispatch_test.go (modified, exists)
- FOUND commit d9d8498f: feat(164-01): thread territory survey into research Scout brief
- FOUND commit b189406d: feat(164-01): re-research every phase on a replan's first iteration
