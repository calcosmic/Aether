---
phase: 165-core-lifecycle-commands
plan: 01
subsystem: docs
tags: [go-testing, wrapper-contracts, ceremony-restoration, flat-mirror-drift]

# Dependency graph
requires:
  - phase: 160-fail-loudly
    provides: shared foundation all Phase 161-170 plans depend on
provides:
  - "Manifest and Completion Packet Shapes" section in .aether/docs/wrapper-host-contract.md (15 field keys, producer, wrapper obligation)
  - "Terminal Worker Result Belongs to the Wrapper" ruling recording the Open Question 2 decision
  - cmd/lifecycle_wrapper_contract_test.go shared toolkit (canonicalWrapperPaths, flatMirrorPath, countMarkerLines, stageSkeletonMarkers, envelopeMechanicsMarkers, assertOrderedHeadingParity, assertStageSkeletonDensity)
  - Permanent flat-mirror byte-equality invariant (TestLifecycleFlatMirrorsMatchCanonical) across all four lifecycle verbs
  - Permanent retired-depth-vocabulary fence (TestLifecycleWrappersAvoidRetiredDepthVocabulary)
  - Resynced .claude/commands/ant-build.md and ant-init.md (previously drifted, now byte-identical to canonical)
affects: [165-02, 165-03, 165-04, 165-05, 165-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared Go test toolkit in package cmd reused across wrapper-rewrite plans instead of each plan inventing its own helpers"
    - "Byte-equality invariant test (bytes.Equal) pins flat installed-consumer mirrors to canonical nested wrapper sources permanently"
    - "Proportion/invariant assertion (assertStageSkeletonDensity) over named-section grep, per project Definition of Done"

key-files:
  created:
    - cmd/lifecycle_wrapper_contract_test.go
  modified:
    - .aether/docs/wrapper-host-contract.md
    - .claude/commands/ant-build.md
    - .claude/commands/ant-init.md

key-decisions:
  - "Flat legacy mirrors (Open Question 1) are rewritten in lockstep with canonical wrappers and pinned by byte-equality test, not descoped or left to a publish round-trip"
  - "Terminal worker result field list (Open Question 2) stays inline in wrappers as method, not envelope mechanics — ruling recorded in wrapper-host-contract.md so it cannot be re-litigated silently"
  - "wrapper-host-contract.md gains real new content (Open Question 3), not just a pointer, per the project's Definition of Done rejecting paper claims"

patterns-established:
  - "Retired-vocabulary fence lives in a dedicated shared test file (not platform_doc_hygiene_test.go) so four parallel Wave 2 wrapper-editing plans don't serialize on one shared forbidden-list file"

requirements-completed: [CMD-01, CMD-02]

# Metrics
duration: 25min
completed: 2026-08-02
---

# Phase 165 Plan 01: Wave 1 Shared Foundation Summary

**Documented 15 manifest field shapes plus the terminal-result boundary ruling in wrapper-host-contract.md, built a shared Go test toolkit with a permanent flat-mirror byte-equality invariant, and resynced two already-drifted flat wrapper mirrors so the drift is now mechanically prevented.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-08-02T22:05:54Z
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- `.aether/docs/wrapper-host-contract.md` now documents 15 manifest/completion field keys in one place, plus a written ruling that the terminal worker result is method (stays inline), resolving all three research-flagged open questions
- New `cmd/lifecycle_wrapper_contract_test.go` gives Wave 2 and Wave 3 plans seven reusable helpers without redefining any of the existing package-level test helpers
- `TestLifecycleFlatMirrorsMatchCanonical` makes flat-mirror drift a permanent, mechanically-enforced invariant across all four lifecycle verbs (init, plan, build, continue) — previously only two of four verbs had any test coverage at all
- The two proven-drifted flat mirrors (`ant-build.md`, `ant-init.md`) are resynced byte-for-byte with their canonical sources

## Task Commits

Each task was committed atomically:

1. **Task 1: Author manifest-shape section and terminal-result boundary ruling** - `54b6b2ec` (docs)
2. **Task 2: Create shared lifecycle wrapper contract test toolkit** - `0834b9d3` (test)
3. **Task 3: Resync the two drifted flat mirrors** - `ba8874e4` (fix)

_Note: Task 2 is a TDD task whose intentional RED state (`TestLifecycleFlatMirrorsMatchCanonical` failing for `ant-build.md`/`ant-init.md`) is by design closed by Task 3's fix commit, per the plan's own acceptance criteria — not a standard single-task RED/GREEN/REFACTOR cycle._

## Files Created/Modified
- `.aether/docs/wrapper-host-contract.md` - Added "Manifest and Completion Packet Shapes" table (15 keys) and "Terminal Worker Result Belongs to the Wrapper" ruling; all 7 pre-existing sections (Context, Decision, Boundary Rules, Provider/Auth Boundary, Rationale, Migration Notes, Consequences) untouched
- `cmd/lifecycle_wrapper_contract_test.go` - New shared toolkit: `lifecycleWrapperVerbs`, `canonicalWrapperPaths`, `flatMirrorPath`, `countMarkerLines`, `stageSkeletonMarkers`, `envelopeMechanicsMarkers`, `assertOrderedHeadingParity`, `assertStageSkeletonDensity`, plus `TestLifecycleFlatMirrorsMatchCanonical`, `TestWrapperHostContractDocumentsManifestShapes`, `TestLifecycleWrappersAvoidRetiredDepthVocabulary`
- `.claude/commands/ant-build.md` - Resynced to exact byte content of `.claude/commands/ant/build.md`
- `.claude/commands/ant-init.md` - Resynced to exact byte content of `.claude/commands/ant/init.md`

## Decisions Made
- Kept all three Open Question rulings exactly as the plan's objective specified (options already locked by the planner) — no new judgment calls were needed during execution
- Assigned one short imperative wrapper obligation per manifest key row (15 total) using the plan's seven example phrases as anchors, extrapolating reasonable equivalents for the remaining eight keys (e.g., `dispatch_manifest.execution_plan` → "respect wave order", `result.plan_manifest` → describes what it is the sole source for) since the plan specified the phrase bank but not a 1:1 row mapping

## Deviations from Plan

None - plan executed exactly as written. All acceptance criteria for all three tasks passed on first attempt; no auto-fixes were needed.

## Issues Encountered

The full `go test ./cmd/...` run (Task 3's final verification step) took ~4.7 minutes and exceeded the default 120s Bash timeout, so it was run in the background and awaited via notification rather than a foreground call. No functional issue — confirmed `ok github.com/calcosmic/Aether/cmd 282.388s` on completion, plus a clean `go build ./cmd/aether` and `go vet ./...`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Wave 2's four wrapper-rewrite plans (165-02 through 165-05, presumably) can now point wrapper prose at real `wrapper-host-contract.md` content instead of an empty promise, and can call `canonicalWrapperPaths`, `flatMirrorPath`, `assertOrderedHeadingParity`, and `assertStageSkeletonDensity` directly without redefining them
- Each Wave 2 plan's own Task 3 (per the plan's objective note) still needs to resync its own wrapper's flat mirror after its rewrite — Task 3 here only closed the two mirrors that were pre-existing drift, not future drift introduced by Wave 2 edits
- `envelopeMechanicsMarkers()` is defined and ready but intentionally not yet invoked by any test — Wave 3's CMD-02 proportion test (per the plan's Task 2 read_first notes referencing "165-06-PLAN.md Task 1") is expected to consume it
- No blockers identified for Wave 2 start

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-02*

## Self-Check: PASSED

- FOUND: `.aether/docs/wrapper-host-contract.md`
- FOUND: `cmd/lifecycle_wrapper_contract_test.go`
- FOUND: `.planning/phases/165-core-lifecycle-commands/165-01-SUMMARY.md`
- FOUND: commit `54b6b2ec` (Task 1)
- FOUND: commit `0834b9d3` (Task 2)
- FOUND: commit `ba8874e4` (Task 3)
