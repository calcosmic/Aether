---
phase: 165-core-lifecycle-commands
plan: 03
subsystem: docs
tags: [go-testing, wrapper-contracts, ceremony-restoration, continue-wrapper]

# Dependency graph
requires:
  - phase: 165-01
    provides: shared lifecycle wrapper contract test toolkit and wrapper-host-contract.md manifest-shape documentation
provides:
  - "Rewritten .claude/commands/ant/continue.md and .opencode/commands/ant/continue.md carrying the D-05 stage skeleton (Purpose/Reads/Spawns/Stop conditions) and the Classic 'builders verifying their own work is confirmation bias' verification beat"
  - "TestContinueWrapperStageSkeletonAndParity (5 subtests: stage_skeleton_density, ordered_heading_parity, structured_blocks_present, method_outweighs_envelope_mechanics, context_clear_stays_runtime_owned)"
  - "D-10 item 9 git-mutation fence (git stash / git add -A / git commit forbidden) across both canonical continue.md files and the flat mirror -- the one D-10 regression-fence item with no prior test coverage anywhere in the codebase"
  - "Resynced flat mirror .claude/commands/ant-continue.md and stage_skeleton wrapper_addition + 2 guardrail bullets in .aether/commands/continue.yaml"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "D-05 stage skeleton (colony beat, **Purpose:**, **Reads:**, **Spawns:**, imperative steps, **Stop conditions:**) applied per-stage to a lifecycle wrapper, reusing Plan 01's assertStageSkeletonDensity/assertOrderedHeadingParity/countMarkerLines helpers instead of redefining them"
    - "Single-pointer consolidation: envelope-shape detail lives in wrapper-host-contract.md with exactly one pointer sentence, reduced from two occurrences in the pre-rewrite file"

key-files:
  created: []
  modified:
    - cmd/continue_wrapper_ceremony_test.go
    - .claude/commands/ant/continue.md
    - .opencode/commands/ant/continue.md
    - .claude/commands/ant-continue.md
    - .aether/commands/continue.yaml

key-decisions:
  - "D-10 item 9 (wrapper-driven git stash/commit) fenced at both the two-file forbidden slice inside the existing TestContinueWrapperCeremonyContract AND the three-file (2 canonical + flat mirror) context_clear_stays_runtime_owned subtest, so the git fence and the context-clear fence now cover identical file sets"
  - "Kept the byte-exact fast-path command line untouched to the character across all four files asserted by TestContinueWrapperSourcesUseFastDevContinue (continue.yaml, ant-continue.md, both canonical continue.md)"
  - "Moved the single surviving wrapper-host-contract.md pointer sentence to under Heavy External Review (where the envelope-parsing mechanics actually live) rather than Ownership Split, per the plan's D-03/D-04 instruction to subordinate mechanics to stage method"

patterns-established:
  - "Verification-beat placement: the rule-with-reason 🐜 idiom belongs immediately at the moment the rule fires (opening ## Heavy External Review), not as an abstract statement elsewhere in the file"

requirements-completed: [CMD-01, CMD-02]

# Metrics
duration: 45min
completed: 2026-08-03
---

# Phase 165 Plan 03: Continue Wrapper Rewrite Summary

**Rewrote continue.md on both platforms so the fast runtime-owned default and the heavy wrapper-conducted review each read as engineering method (Purpose/Reads/Spawns/Stop conditions) with the Classic "builders verifying their own work is confirmation bias" verification beat restored, and closed the one D-10 regression-fence item (wrapper-driven git stash/commit) that had zero test coverage anywhere in the codebase before this plan.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-03
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- `cmd/continue_wrapper_ceremony_test.go` gained `TestContinueWrapperStageSkeletonAndParity` (5 subtests) plus 7 new required strings and 3 new forbidden strings (`git stash`, `git add -A`, `git commit`) on the existing `TestContinueWrapperCeremonyContract`, without touching the pre-existing Go-render assertions (`renderContinueVisual`/`renderContinueBlockedVisual`) that independently prove the context-clear line stays runtime-owned
- Both canonical `continue.md` files were rewritten byte-identically: three stages (`## Default Continue`, `## Heavy External Review`, `## After Continue`) now each carry a colony beat, `**Purpose:**`, `**Reads:**`, `**Spawns:**`, and `**Stop conditions:**`; a new `## Required Cross-Stage State` section names the current-vocabulary carry-forward values (`phase_id`, `verification_depth`, `verification_status`, `manifest_file`, `next_action`)
- The Classic verification beat ("Builders verifying their own work is confirmation bias. Independent Watchers catch bugs builders miss. 'Build passing' is not the same as 'app working'.") now opens `## Heavy External Review` under a `Why this matters` line, mined from `continue-gates.md:49-52` per the archaeology report
- `wrapper-host-contract.md` is referenced exactly once (down from two occurrences), consolidated under `## Heavy External Review` where the envelope-parsing mechanics it documents actually live
- The context-clear fence (`It's safe to clear your context now.` / `/ant-resume`) now also covers the flat mirror `.claude/commands/ant-continue.md`, alongside the new D-10 item 9 git-mutation fence on the same three files
- The flat mirror and `.aether/commands/continue.yaml` are resynced in lockstep: `stage_skeleton` wrapper_addition documents the D-05 order and cross-stage state vocabulary; two new guardrail bullets forbid hand-rendering ceremony visuals and restate the context-clear runtime-ownership rule

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend the continue wrapper contract test** - `e8aa6981` (test)
2. **Task 2: Rewrite continue.md on both platforms** - `c32592a6` (feat)
3. **Task 3: Resync flat mirror and continue.yaml guardrails** - `0b7584e4` (fix)

_Note: this is a `type="auto"` plan, not `type="tdd"` at the plan level, but Task 1 was itself run as a TDD task — its RED state (`stage_skeleton_density` failing, plus the extended `TestContinueWrapperCeremonyContract` newly failing on the 7 added required strings) was intentional and closed by Task 2's rewrite, per the plan's own acceptance criteria._

## Files Created/Modified

- `cmd/continue_wrapper_ceremony_test.go` - Extended required/forbidden slices on `TestContinueWrapperCeremonyContract`; added `TestContinueWrapperStageSkeletonAndParity` with 5 subtests reusing Plan 01's shared toolkit (`canonicalWrapperPaths`, `flatMirrorPath`, `countMarkerLines`, `stageSkeletonMarkers`, `envelopeMechanicsMarkers`, `assertOrderedHeadingParity`, `assertStageSkeletonDensity`)
- `.claude/commands/ant/continue.md` - Rewritten with D-05 stage skeleton, Required Cross-Stage State, Classic verification beat, success/failure/read-only blocks; wrapper-host-contract.md pointer reduced to 1 occurrence
- `.opencode/commands/ant/continue.md` - Byte-identical copy of the canonical rewrite
- `.claude/commands/ant-continue.md` - Resynced to exact byte content of the rewritten canonical file
- `.aether/commands/continue.yaml` - Added `stage_skeleton` wrapper_addition and 2 guardrail bullets; no existing entry deleted or reworded, fast-path command string untouched

## Decisions Made

- Included a `**Spawns:**` line on `## After Continue` (explicitly "nothing; this stage only communicates the runtime's result") even though the plan called `**Spawns:**` "where applicable" — chose to keep the marker on all three stages for consistency and to strengthen the `method_outweighs_envelope_mechanics` proportion margin, since an absent spawn stage is itself useful method per the plan's own framing for Default Continue
- Placed the single surviving `wrapper-host-contract.md` pointer under `## Heavy External Review` (not `## Ownership Split`) since that is where the manifest-parsing mechanics it documents actually occur, satisfying the plan's "subordinate to the stage's method" instruction literally

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a self-inflicted line-wrap bug that broke a required substring across two lines**
- **Found during:** Task 2 verification
- **Issue:** The first draft of `continue.md` wrapped "Spawn reviewers as visible live Task/subagent panels. Do not set `run_in_background`." across two lines mid-sentence, splitting the required substring `` Do not set `run_in_background` `` so `TestContinueWrapperCeremonyContract` failed on both canonical files.
- **Fix:** Joined the two lines back into one, then re-copied to the OpenCode mirror.
- **Files modified:** `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`
- **Commit:** folded into `c32592a6` (caught and fixed before the task commit)

No other deviations - plan executed as written.

## Issues Encountered

The full `go test ./cmd/...` run for Task 3's final verification took ~7.2 minutes, exceeding the default 120s Bash timeout, so it was run in the background and awaited via notification. Confirmed `ok github.com/calcosmic/Aether/cmd 433.093s` on completion, plus clean `go vet ./...` and successful `go build ./cmd/aether`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Continue's wrapper is now fully rewritten and consistent with Plan 01's shared foundation; no further work on continue.md is expected within Phase 165
- The `TestContinueWrapperStageSkeletonAndParity` pattern (5 subtests) is a directly copyable template for any remaining Wave 2 wrapper-rewrite plans (build.md, plan.md, init.md) that have not yet added their own stage-skeleton-density test
- No blockers identified

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-03*

## Self-Check: PASSED

- FOUND: `cmd/continue_wrapper_ceremony_test.go`
- FOUND: `.claude/commands/ant/continue.md`
- FOUND: `.opencode/commands/ant/continue.md`
- FOUND: `.claude/commands/ant-continue.md`
- FOUND: `.aether/commands/continue.yaml`
- FOUND: `.planning/phases/165-core-lifecycle-commands/165-03-SUMMARY.md`
- FOUND: commit `e8aa6981` (Task 1)
- FOUND: commit `c32592a6` (Task 2)
- FOUND: commit `0b7584e4` (Task 3)
