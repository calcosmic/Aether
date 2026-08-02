---
phase: 164-research-feeds-planning
plan: 04
subsystem: cli
tags: [go, plan-depth, granularity, verification-depth, queen-proposal]

# Dependency graph
requires:
  - phase: 164-research-feeds-planning (research pass)
    provides: interface catalogue for colony.GranularityRange, review_depth.go smart-default helpers, codex_visuals.go reason renderers
provides:
  - "computeDepthProposal — a single function returning all three plan-time depth knobs (granularity, planning_depth, verification_depth) together, each pre-marked with a recommendation and a plain-English reason"
  - "renderDepthProposalCard — renders the three-knob proposal as one selection-only card with a single zero-typing accept line and a single change-by-number instruction"
affects: [164-09 (wires this proposal into the manifest and platform wrappers)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "depthKnob/depthKnobOption struct pair for uniform multi-option proposal rendering"
    - "lookup-table reason functions (granularityReasons map) mirroring the existing getSmartDefaultReason shape in cmd/review_depth.go"

key-files:
  created:
    - cmd/plan_depth_proposal.go
    - cmd/plan_depth_proposal_test.go
  modified: []

key-decisions:
  - "Reused renderSmartDepthReason for both planning-depth and verification-depth reasons instead of writing new heuristics, per the plan's explicit instruction to reuse the existing helper"
  - "Renamed the new label helper to depthProposalGranularityLabel to avoid a package-level name collision with the existing (hardcoded-range) granularityLabel in cmd/status.go"

patterns-established:
  - "Three-knob depth proposal card: one Go-computed, Go-rendered artifact combining recommendation + reason + selection-only UI, matching the wrapper-runtime UX contract (runtime prints the string; wrappers do not compose their own)"

requirements-completed: [RESEARCH-09, RESEARCH-10]

# Metrics
duration: 13min
completed: 2026-08-02
---

# Phase 164 Plan 04: Three-Knob Depth Proposal Summary

**One Go file computing and rendering a combined granularity/planning-depth/verification-depth proposal card with pre-marked recommendations and plain-English reasons, replacing plan.md's two ungrounded static ceremonies.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-08-02T11:13:15Z
- **Completed:** 2026-08-02T11:26:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 (both newly created)

## Accomplishments
- `computeDepthProposal` returns exactly three knobs (`granularity`, `planning_depth`, `verification_depth`) in order, each with a pre-marked recommendation and a non-empty reason
- Granularity option labels are sourced from `colony.GranularityRange` — never hard-coded — with a new `renderGranularityReason` that names the existing route's phase count when phases exist, or buckets the goal's word count otherwise
- `renderDepthProposalCard` renders the whole proposal as a single selection-only card: numbered options with a visible recommendation marker, one `Reason:` line per knob, exactly one `Accept all:` line with all three recommended values already substituted, and exactly one `Change one:` instruction — no free-text prompt anywhere

## Task Commits

Each task followed the RED/GREEN TDD cycle:

1. **Task 1: Three-knob depth proposal with computed reasons**
   - `c38cd690` test(164-04): add failing test for three-knob depth proposal (RED)
   - `07dde040` feat(164-04): compute three-knob depth proposal with reasons (GREEN)
2. **Task 2: Render the proposal as a zero-typing selection card**
   - `c9198ebc` test(164-04): add failing test for selection-only proposal card (RED)
   - `c1900f11` feat(164-04): render depth proposal as selection-only card (GREEN)

**Plan metadata:** committed alongside this SUMMARY (see final commit in the worktree).

## Files Created/Modified
- `cmd/plan_depth_proposal.go` — `depthKnobOption`/`depthKnob`/`depthProposal` types, `computeDepthProposal`, `renderGranularityReason`, `depthProposalGranularityLabel`, `smartOrExplicitReason`, `renderDepthProposalCard`
- `cmd/plan_depth_proposal_test.go` — `TestDepthProposalCardKnobs`, `TestDepthProposalReasons`, `TestDepthProposalCardIsSelectionOnly`

## Decisions Made
- Reused `renderSmartDepthReason(colony.Phase{ID: 1}, len(state.Plan.Phases))` verbatim for both planning-depth and verification-depth reasons rather than writing new smart-default heuristics, exactly as the plan's action section directed
- Renamed the Task 1 label helper from the plan's implied `granularityLabel` to `depthProposalGranularityLabel` because `cmd/status.go` already declares a package-level `granularityLabel(string) string` with hardcoded ranges (1-3/4-7/8-12/13-20) — a pre-existing violation of the single-source-of-truth principle this plan's own RESEARCH-09 intent argues against, but out of this plan's scope to fix. Renaming avoided a redeclaration/type-mismatch build error without touching status.go's behavior.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Renamed `granularityLabel` to `depthProposalGranularityLabel`**
- **Found during:** Task 1 (first `go test` run after writing the implementation)
- **Issue:** `cmd/status.go` already declares `func granularityLabel(granularity string) string` (hardcoded ranges, different signature — takes `string` not `colony.PlanGranularity`). The plan's action text names the new function `granularityLabel` without checking for an existing symbol, producing a build-blocking redeclaration.
- **Fix:** Renamed the new function to `depthProposalGranularityLabel` and updated its one call site in `buildGranularityKnob`. No other file needed changes.
- **Files modified:** `cmd/plan_depth_proposal.go`
- **Verification:** `go build ./...` and `go vet ./cmd/...` both clean; `go test ./cmd/... -count=1` passes (427.9s, all packages).
- **Committed in:** `07dde040` (Task 1 GREEN commit — the rename happened before that commit was made, so no separate fix commit was needed)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Naming-only fix; no behavior change from what the plan specified. No scope creep.

## Issues Encountered

**Acceptance-criteria grep ambiguity.** Task 1's acceptance criteria states `cmd/plan_depth_proposal.go` "does not contain the literal strings `1-3`, `4-7`, `8-12` or `13-20` outside comments." The same task's action text explicitly mandates the planning-depth option label `"light — coarse tasks, 1-3 per plan"` (quoted verbatim from `plan.md`'s existing copy), which itself contains the literal substring `1-3` — but as a *task count* per plan, unrelated to granularity phase-count ranges. A blind grep over the whole file therefore reports one match. I kept the explicitly mandated label text as written (it satisfies the actual intent — no hard-coded *granularity* range exists anywhere; `colony.GranularityRange` is the only source for phase-count numbers) and am flagging the literal grep check as a plan-authoring inconsistency for future reference rather than dropping the mandated label copy.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `computeDepthProposal` and `renderDepthProposalCard` are ready for Plan 09 to wire into the `aether host plan` manifest and the Claude/OpenCode wrapper output, per the plan's stated split of responsibility ("Plan 09 wires it into the manifest and the wrappers")
- No blockers. The full `cmd` package test suite (427.9s) and `go vet ./cmd/...` are both clean on top of this plan's changes.

## Self-Check: PASSED

- FOUND: cmd/plan_depth_proposal.go
- FOUND: cmd/plan_depth_proposal_test.go
- FOUND: .planning/phases/164-research-feeds-planning/164-04-SUMMARY.md
- FOUND: c38cd690, 07dde040, c9198ebc, c1900f11, d4e94222 (all in `git log --oneline`)
