---
phase: 165-core-lifecycle-commands
plan: 02
subsystem: docs
tags: [wrapper-rewrite, ceremony-restoration, build-wrapper, tdd]

# Dependency graph
requires:
  - phase: 165-core-lifecycle-commands
    plan: "01"
    provides: cmd/lifecycle_wrapper_contract_test.go shared toolkit, wrapper-host-contract.md manifest shapes
provides:
  - "Rewritten .claude/commands/ant/build.md and .opencode/commands/ant/build.md carrying D-05 stage skeleton (colony beat + Purpose + Reads + Spawns + Stop conditions) and Classic Queen-voice ceremony"
  - "D-08/CMD-05 ownership handshake: PHASE-160/PHASE-165/PHASE-168 chain recorded in the file and pinned by TestBuildMdOwnershipHandshake"
  - "TestBuildWrapperStageSkeletonAndParity: stage-skeleton density, heading parity, structured blocks, and method-vs-envelope-mechanics proportion assertions"
  - "D-10 item 9 forbidden-string coverage for wrapper-driven git stash/add/commit"
  - "Resynced .claude/commands/ant-build.md flat mirror; updated .aether/commands/build.yaml with stage_skeleton and ownership_chain wrapper_additions plus 2 new guardrail bullets"
affects: [165-03, 165-04, 165-05, 165-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "D-05 stage skeleton: colony beat -> **Purpose:** -> **Reads:** -> **Spawns:** (where applicable) -> instructions -> **Stop conditions:**, exempting structural/guardrail headings"
    - "Envelope-shape detail relocated to a single pointer sentence at wrapper-host-contract.md, keeping wrapper prose as method not schema"
    - "Ownership-chain HTML comment placed after the frontmatter delimiter (not before) to stay compatible with cmd/source_check.go's single-header-line parser assumption"

key-files:
  created:
    - .planning/phases/165-core-lifecycle-commands/165-02-SUMMARY.md
  modified:
    - cmd/build_wrapper_ceremony_test.go
    - .claude/commands/ant/build.md
    - .opencode/commands/ant/build.md
    - .claude/commands/ant-build.md
    - .aether/commands/build.yaml

key-decisions:
  - "Placed the PHASE-160/PHASE-165/PHASE-168 ownership comment as the first line of the file body (after the closing frontmatter '---') rather than before the frontmatter, because cmd/source_check.go's parseSourceCheckWrapper only strips exactly one header line before requiring the frontmatter delimiter to open immediately — this is a Rule 1 bug fix, not a deviation from the ownership-chain requirement itself, which only constrains PHASE-160 appearing before PHASE-168 and PHASE-168 being the last non-empty line"
  - "Kept the 'Parse `result.completion_path`' line inline in Finalize (not moved to wrapper-host-contract.md) per the plan's explicit carve-out that terminal-result/finalize mechanics are method, not envelope mechanics needing relocation"
  - "wrapper-host-contract.md is referenced exactly once, under Dispatch Manifest, removed from the Ownership Split table where the original file's single existing reference lived"

patterns-established:
  - "Any future rewrite adding a leading HTML comment to a wrapper file must place it after the frontmatter close, not stacked before it, to avoid breaking cmd/source_check.go's parser"

requirements-completed: [CMD-01, CMD-02, CMD-03, CMD-05]

# Metrics
duration: 45min
completed: 2026-08-03
---

# Phase 165 Plan 02: Build Wrapper Ceremony Restoration Summary

**Rewrote `build.md` on both platforms with a uniform D-05 stage skeleton and restored Classic Queen-voice ceremony beats, pinned by three new/extended test functions (`TestBuildMdOwnershipHandshake`, `TestBuildWrapperStageSkeletonAndParity`, extended `TestBuildWrapperCeremonyContract`) that fail if the ceremony, the method, or the Phase 160/165/168 ownership record is ever removed.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-03
- **Tasks:** 3 (plus one Rule-1 bug-fix commit discovered during Task 3 verification)
- **Files modified:** 5 (1 test file extended, 4 wrapper/config files rewritten)

## Accomplishments

- `cmd/build_wrapper_ceremony_test.go` gained `TestBuildMdOwnershipHandshake` (D-08/CMD-05) and `TestBuildWrapperStageSkeletonAndParity` (D-05/D-09, four subtests: stage-skeleton density, heading parity, structured-blocks presence, method-outweighs-envelope-mechanics proportion), plus three new forbidden strings (`git stash`, `git add -A`, `git commit`) closing the one D-10 regression-fence item (item 9) that previously had zero test coverage anywhere in the codebase
- `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` are now byte-identical rewrites: every non-exempt stage carries a colony beat, `**Purpose:**`, `**Reads:**`, and `**Stop conditions:**` (10 Purpose markers total); Worker Spawning additionally carries `**Spawns:**` and a restored "Why this matters" rule-with-reason block; the Classic "You DIRECTLY spawn multiple workers" opener is restored
- Envelope-shape detail is relocated behind exactly one pointer sentence to `.aether/docs/wrapper-host-contract.md`; the file contains zero hand-rendered `━━━`/`────` separators and zero occurrences of the retired `colony_depth` vocabulary
- The Phase 160 -> 165 -> 168 ownership chain is recorded as an HTML comment plus the exact reserved `<!-- PHASE-168: visual-guidance trailer appends below this line -->` trailer marker (the file's last non-empty line), both asserted by `TestBuildMdOwnershipHandshake`
- `.claude/commands/ant-build.md` flat mirror resynced byte-for-byte; `.aether/commands/build.yaml` gained `stage_skeleton` and `ownership_chain` `wrapper_additions` entries plus exactly 2 new guardrail bullets (18 total, up from 16), with all original entries untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend build wrapper contract test (RED)** - `4ec50399` (test)
2. **Task 2: Rewrite build.md on both platforms (GREEN)** - `49fe28c5` (feat)
3. **[Rule 1 - Bug] Fix source-check parser incompatibility** - `fb455cb8` (fix)
4. **Task 3: Resync flat mirror and update build.yaml guardrails** - `e56c835f` (fix)

## Files Created/Modified

- `cmd/build_wrapper_ceremony_test.go` - Extended `required`/`forbidden` slices; added `TestBuildMdOwnershipHandshake` and `TestBuildWrapperStageSkeletonAndParity`
- `.claude/commands/ant/build.md` - Full rewrite: D-05 stage skeleton, Classic colony beats, ownership handshake, structured blocks
- `.opencode/commands/ant/build.md` - Byte-identical copy of the above
- `.claude/commands/ant-build.md` - Resynced flat mirror
- `.aether/commands/build.yaml` - Added `stage_skeleton`/`ownership_chain` wrapper_additions entries and 2 guardrail bullets

## Decisions Made

- The ownership-chain HTML comment's exact position (before vs. after frontmatter) was not fixed by the plan text, which said "immediately after the Aether-managed line." Placing it there broke `cmd/source_check.go`'s single-header-line parser assumption (`TestSourceCheckValidatesCurrentSourceSurfaces` failed). Moved it to the first line of the body instead — this satisfies every test the plan named (`TestBuildMdOwnershipHandshake` only requires PHASE-160 before PHASE-168, with PHASE-168 as the last non-empty line) while not regressing an unrelated, pre-existing test that enforces the wrapper/YAML generation contract.
- Extrapolated Reads/Spawns/Stop-conditions prose for each of the 10 non-exempt stages from the archaeology report and the existing mechanics text, since the plan specified the skeleton order but not verbatim wording per stage (explicitly "Claude's Discretion" per 165-CONTEXT.md).
- Did not restore the "Context Confirmation Rule" mentioned in D-8/T-165-02-01 of the threat model, since no test or explicit `<action>` instruction required it and no verbatim user transcript exists anywhere in the current file to guard against re-introducing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed source-check parser incompatibility with the ownership-chain comment placement**
- **Found during:** Task 3's full `go test ./cmd/...` verification run
- **Issue:** Placing the `PHASE-160:` HTML comment immediately before the frontmatter delimiter (as a second header line) broke `cmd/source_check.go`'s `parseSourceCheckWrapper`, which only strips exactly one leading comment line before requiring `---` to open the frontmatter. `TestSourceCheckValidatesCurrentSourceSurfaces` failed for both `.claude` and `.opencode` build.md with "missing opening frontmatter delimiter."
- **Fix:** Moved the `PHASE-160:` comment to the first line of the file body, immediately after the closing frontmatter `---`, instead of before the opening one. All ownership-handshake test assertions (`TestBuildMdOwnershipHandshake`) and acceptance criteria (PHASE-160 before PHASE-168, PHASE-168 as last non-empty line, grep counts) still pass.
- **Files modified:** `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`
- **Commit:** `fb455cb8`

## Verification Results

- `go vet ./...` clean; `go build ./cmd/aether` succeeds
- `go test ./cmd/...` passes in full (two full runs: 435.8s pre-fix showing the one expected failure, 330.7s post-fix all green)
- `.claude`, `.opencode`, and the flat mirror `build.md` are byte-identical (verified by `diff`)
- Manual read-through of `.claude/commands/ant/build.md` top to bottom (CMD-03 criterion): each of the 10 non-exempt stages states its purpose and inputs in one sentence (`**Purpose:**` / `**Reads:**`) without requiring consultation of `cmd/*.go`

## Issues Encountered

None beyond the source-check parser fix documented above. All three tasks' acceptance criteria passed after that one fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 165-03/04/05 (continue.md, plan.md, init.md rewrites, presumably) can reuse the same D-05 skeleton pattern and the shared `assertStageSkeletonDensity` / `assertOrderedHeadingParity` / `countMarkerLines` helpers from Plan 01's toolkit without redefining them
- The ownership-chain HTML-comment placement lesson (after frontmatter, not before) applies to any of those plans if they add a similar PHASE-ownership record to their own wrapper files
- No blockers identified for this plan; build.md is now the sole structural product of Phase 165 for milestone v1.25, with Phase 168 having an unambiguous trailer insertion point

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-03*

## Self-Check: PASSED

- FOUND: `.claude/commands/ant/build.md`
- FOUND: `.opencode/commands/ant/build.md`
- FOUND: `.claude/commands/ant-build.md`
- FOUND: `.aether/commands/build.yaml`
- FOUND: `cmd/build_wrapper_ceremony_test.go`
- FOUND: commit `4ec50399` (Task 1)
- FOUND: commit `49fe28c5` (Task 2)
- FOUND: commit `fb455cb8` (Rule 1 fix)
- FOUND: commit `e56c835f` (Task 3)
