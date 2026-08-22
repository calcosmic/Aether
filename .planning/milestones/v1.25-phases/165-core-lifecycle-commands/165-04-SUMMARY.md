---
phase: 165-core-lifecycle-commands
plan: 04
subsystem: docs
tags: [wrapper-contracts, go-testing, ceremony-restoration, lifecycle-commands]

# Dependency graph
requires:
  - phase: 165-core-lifecycle-commands (Plan 01)
    provides: wrapper-host-contract.md manifest-shape table, shared lifecycle wrapper contract test toolkit (assertStageSkeletonDensity, stageSkeletonMarkers, envelopeMechanicsMarkers, countMarkerLines)
provides:
  - Rewritten plan.md (Claude + OpenCode, byte-identical) on the D-05 stage skeleton with Purpose/Reads/Spawns/Stop-conditions per stage
  - "## Required Cross-Stage State" block naming the current planning-loop vocabulary (planning_run_id, iteration, target_confidence, manifest_file, selected_gaps, stop_reason, next_action)
  - Restored termination-condition prose naming all four planning-loop stop conditions (target confidence, stall detection, iteration cap, escape hatch) without numeric thresholds, with the --accept escape-hatch flag verified against the live runtime
  - <success_criteria>/<failure_modes>/<read_only> structured blocks
  - TestPlanWrapperStageSkeleton (4 subtests) extending the plan wrapper contract test suite
  - Resynced .claude/commands/ant-plan.md flat mirror and updated .aether/commands/plan.yaml wrapper_additions/guardrails
affects: [165-05, 165-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "D-05 stage skeleton (colony beat, Purpose, Reads, Spawns-where-applicable, method, Stop conditions) applied to plan.md, extending the pattern Plan 01's toolkit was built for"
    - "Termination-condition prose names concepts (target confidence, stall, max iteration, escape hatch) as literal test-asserted substrings rather than asserting one long sentence, per project Definition of Done (proportion/invariant over named-section grep)"
    - "CLI flags named in restored ceremony prose are verified against the live binary before being written (--accept confirmed via `aether plan --help` and TS host --help text) rather than carried forward from v5.4.0 archaeology unchecked"

key-files:
  created: []
  modified:
    - cmd/plan_wrapper_ceremony_test.go
    - .claude/commands/ant/plan.md
    - .opencode/commands/ant/plan.md
    - .claude/commands/ant-plan.md
    - .aether/commands/plan.yaml

key-decisions:
  - "Named the --accept escape-hatch flag explicitly in stop-condition prose (rather than describing it purely by behavior) after verifying it exists on both the Go CLI (`aether plan --help`) and the TS host (`aether host plan --help` text in host.ts:347) — the plan allowed behavior-only description only if no current flag matched, and one does"
  - "Kept the pre-existing .aether/docs/wrapper-runtime-ux-contract.md pointer sentence in Decision Moment 1 as-is; it is a different document from wrapper-host-contract.md and the plan's 'exactly one pointer' rule applies only to the latter"
  - "Placed the full termination-condition prose in Finalize's Stop conditions (where requires_next_iteration already lived) rather than creating a new heading, since D-06 called for restoring the concepts, not a specific stage"

patterns-established: []

requirements-completed: [CMD-01, CMD-02]

# Metrics
duration: 17min
completed: 2026-08-03
---

# Phase 165 Plan 04: Plan Wrapper Rewrite Summary

**Rewrote plan.md (Claude + OpenCode) onto the D-05 stage skeleton with Classic Queen-voice ceremony and restored, verified termination-condition prose — the planning loop's four stop conditions (target confidence, stall detection, iteration cap, escape hatch) are now documented honestly without threshold arithmetic or an unverified flag, while the Phase 164 two-decision-moment contract survived unchanged.**

## Performance

- **Duration:** ~17 min
- **Completed:** 2026-08-03T00:30:00+02:00
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- `plan.md` on both platforms now carries a uniform stage skeleton: every one of its 8 non-exempt stages has Purpose, Reads, Spawns-where-applicable, and Stop conditions, plus a Classic 🐜 colony-beat line
- The planning loop's termination conditions — target confidence reached, stall detection, an iteration cap, and an escape hatch — are documented in prose for the first time since the v5.4.0 archaeology baseline, with the escape-hatch flag (`--accept`) verified against the live runtime rather than assumed from history
- `## Required Cross-Stage State`, `<success_criteria>`, `<failure_modes>`, and `<read_only>` blocks added, closing the D-06 method gap the phase research identified
- The Phase 164 two-decision-moment contract (depth proposal card, research batch card) and all four runtime card keys survived the rewrite verbatim, proven by the unmodified `TestPlanWrapperCardsParity` passing unchanged
- Envelope-parsing detail in `## Planning Manifest` was subordinated to one line per mechanical step plus exactly one pointer sentence to `.aether/docs/wrapper-host-contract.md`
- All three plan.md paths (`.claude/commands/ant/plan.md`, `.claude/commands/ant-plan.md`, `.opencode/commands/ant/plan.md`) are byte-identical to each other

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend the plan wrapper contract test with skeleton density, structured blocks, and stop-condition prose** - `178108bb` (test)
2. **Task 2: Rewrite plan.md on both platforms with the D-05 stage skeleton, colony beats, and restored stop conditions** - `84cfe329` (feat)
3. **Task 3: Resync the plan flat mirror and update plan.yaml guardrails in lockstep** - `8cae33e6` (fix)

_Note: Task 1 is the RED half of a TDD task closed by Task 2's GREEN implementation, not a standard single-task cycle — matching the plan's own framing._

## Files Created/Modified
- `cmd/plan_wrapper_ceremony_test.go` - Added 8 required substrings to `TestPlanWrapperCeremonyContract` and a new `TestPlanWrapperStageSkeleton` with 4 subtests (stage_skeleton_density, structured_blocks_present, termination_conditions_documented, method_outweighs_envelope_mechanics)
- `.claude/commands/ant/plan.md` - Full D-05 rewrite: Required Cross-Stage State block, Purpose/Reads/Spawns/Stop-conditions per stage, restored termination-condition prose, success/failure/read-only blocks, subordinated envelope mechanics
- `.opencode/commands/ant/plan.md` - Byte-identical copy of the rewritten canonical file
- `.claude/commands/ant-plan.md` - Resynced flat mirror, byte-identical to the canonical file
- `.aether/commands/plan.yaml` - Added `stage_skeleton` and `termination_conditions` entries under `wrapper_additions`; added one new guardrail bullet forbidding hand-rendering `aether ceremony` visuals; existing `depth_proposal_card`/`research_batch_card` keys and all prior guardrails preserved

## Decisions Made
- Named the `--accept` flag explicitly in the escape-hatch description after verifying it against both the Go CLI help output and the TS host's help text — the plan required behavior-only phrasing only if no current flag matched
- Kept the pre-existing `wrapper-runtime-ux-contract.md` reference in Decision Moment 1 untouched since the "exactly once" rule in the plan applies specifically to `wrapper-host-contract.md`, a different document
- Located the termination-condition prose in `## Finalize`'s Stop conditions section, alongside the existing `requires_next_iteration` handling, rather than introducing a new heading

## Deviations from Plan

None - plan executed exactly as written. `.planning/phases/165-core-lifecycle-commands/165-PATTERNS.md`, referenced in the plan's context block and read_first lists, does not exist on disk (confirmed absent in git history across all branches) — this affected all four Wave 2 plans' referenced context, not just this one. Proceeded using `165-ARCHAEOLOGY.md` and `165-RESEARCH.md`, which contain the equivalent Auto-Termination Safeguards quotes and print-verbatim card idiom the plan's read_first pointed at, plus the plan's own detailed task instructions, which were fully self-contained. No task's acceptance criteria depended on content that could only come from the missing file.

## Issues Encountered

The full `go test ./cmd/...` run (Task 3's final verification step) took ~7.3 minutes and exceeded the default 120s Bash timeout; it was run in the background and awaited via notification rather than a foreground call. No functional issue — confirmed `ok github.com/calcosmic/Aether/cmd 439.579s` on completion, plus a clean `go build ./cmd/aether` and `go vet ./...`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `plan.md` now matches the D-05/D-06 shape the other three Wave 2 plans (build.md, continue.md, init.md) are independently rewriting to; `TestLifecycleWrappersAvoidRetiredDepthVocabulary` and `TestLifecycleCommandDocsPreferRuntimeCLI` both stay green against the rewritten file
- Wave 3's CMD-02 proportion test (expected per 165-01-SUMMARY.md) can now also exercise `plan.md` alongside the other three wrappers via `envelopeMechanicsMarkers()`/`stageSkeletonMarkers()`
- No blockers identified

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-03*

## Self-Check: PASSED

- FOUND: `.claude/commands/ant/plan.md`
- FOUND: `.opencode/commands/ant/plan.md`
- FOUND: `.claude/commands/ant-plan.md`
- FOUND: `.aether/commands/plan.yaml`
- FOUND: `cmd/plan_wrapper_ceremony_test.go`
- FOUND: commit `178108bb` (Task 1)
- FOUND: commit `84cfe329` (Task 2)
- FOUND: commit `8cae33e6` (Task 3)
