---
phase: 199-front-door-and-classic-contract
plan: "26"
subsystem: lifecycle-closeout
tags: [go, lifecycle, projection, closeout, seal, entomb, zero-write]

requires:
  - phase: 199-front-door-and-classic-contract
    plans: ["07", "08", "09", "10", "11", "13", "15", "16"]
    provides: Final front-door, work, recovery, seal, entomb, projection, and next-action implementations
provides:
  - One typed focused closeout grammar across init, colonize, plan, build, run, pause, resume, seal, and entomb
  - Revision-locked adapters that consume existing lifecycle projections without resolving a second next action
  - Verified-versus-forced seal truth and archive failure closeouts with safe retained-state recovery
affects: [phase-199-verification, lifecycle-ui, command-results, wrapper-parity]

tech-stack:
  added: []
  patterns: [projection-owned next action, typed focused closeout, zero-write refusal, journal-backed archive failure]

key-files:
  created:
    - cmd/lifecycle_closeout.go
    - cmd/lifecycle_closeout_199_test.go
  modified:
    - cmd/init_cmd.go
    - cmd/codex_colonize_finalize.go
    - cmd/codex_plan_finalize.go
    - cmd/compatibility_cmds.go
    - cmd/session_flow_cmds.go
    - cmd/seal_render.go
    - cmd/entomb_cmd.go

key-decisions:
  - "Focused closeouts consume an existing LifecycleProjection and fail closed when its revision disagrees with the command result."
  - "Zero-write refusals resolve from already-read, read-only state facts so rendering cannot initialize a store or create lock files."
  - "Sealed colonies keep status primary and entomb optional; only a verified archive-and-clear result may offer init."

patterns-established:
  - "Canonical closeout: applicable Colony, Participants, What happened, Evidence, State changes, Standing instructions, Unresolved, and Next Up slots render in one fixed order."
  - "Terminal adapter: command-specific evidence augments projection truth but cannot replace projection-owned identity, closure, state effect, or next-action policy."
  - "Archive failure: typed stage, state effect, and coordinator-journal evidence close on status without claiming an archive or offering init."

requirements-completed: [CEC-01, CEC-02, CEC-04, CEC-08, LIFE-01, LIFE-03, LIFE-04, LIFE-05]

duration: 1h 2m
completed: 2026-09-04
---

# Phase 199 Plan 26: Focused Lifecycle Closeout Summary

**Nine lifecycle commands now end in one projection-backed focused grammar, with zero-write refusals, truthful seal discrimination, and init offered only after verified archival.**

## Performance

- **Duration:** 1h 2m
- **Started:** 2026-09-04T17:13:03Z
- **Completed:** 2026-09-04T18:16:01Z
- **Tasks:** 3/3
- **Files modified:** 9 implementation/test files

## Accomplishments

- Added the typed `LifecycleCloseout` contract and canonical applicable-slot renderer without embedding a status dashboard or introducing another next-action resolver.
- Wired work, recovery, and front-door commands to reuse the exact projection revision, evidence, state effect, refusal truth, replay truth, and platform-native action spelling already present in their results.
- Unified verified seal, forced-incomplete seal, successful entomb, and failed archive closeouts while preserving status-first sealed review, optional entomb authority, archive receipts/digests, and retained or rolled-back journal evidence.
- Added sixteen focused contract cases covering all selected lifecycle commands, canonical order, revision agreement, refusal/replay behavior, and zero-write rendering.

## Task Commits

Each TDD task was committed with a RED test commit followed by its GREEN implementation commit:

1. **Task 1: Define closeouts and wire work/pause results** — `ddb8aac9` (test), `966f9293` (feat)
2. **Task 2: Wire front-door and planning closeouts** — `bcf725cb` (test), `527301b5` (feat)
3. **Task 3: Wire verified and forced closure closeouts** — `b79017b5` (test), `98385744` (feat)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/lifecycle_closeout.go` — typed closeout slots, projection/revision validation, focused rendering, and read-only refusal adapters.
- `cmd/lifecycle_closeout_199_test.go` — sixteen cross-lifecycle TDD cases for structure, semantics, replay, refusal, archive truth, and zero-write behavior.
- `cmd/init_cmd.go` — accepted-charter/territory success and active-colony refusal closeouts.
- `cmd/codex_colonize_finalize.go` — verified territory finalization and unavailable-input refusal closeouts.
- `cmd/codex_plan_finalize.go` — accepted plan evidence, unresolved gaps, and refused finalization closeouts.
- `cmd/compatibility_cmds.go` — build and bounded Autopilot closeouts.
- `cmd/session_flow_cmds.go` — pause/resume success, replay, and conflicting-evidence closeouts.
- `cmd/seal_render.go` — revision-locked verified/forced seal adapters and shared focused terminal rendering.
- `cmd/entomb_cmd.go` — archive success closeouts plus typed retained/rolled-back/recovery-required failure evidence and rendering.

## Decisions Made

- The shared closeout is an adapter over one existing `LifecycleProjection`; a missing or mismatched revision is an error, never a reason to resolve policy again.
- Front-door refusals construct the resolver input from state already inspected by the caller. This preserves the required zero-write boundary because general store-backed loaders may create lock files.
- A seal's durable `PrimaryNext` and `OptionalNext` are validated as `aether status` and `aether entomb`; forced-incomplete evidence is rendered without verified-success wording.
- Entomb converts only command spelling (`/ant-*` to runtime `aether *`) and never chooses policy. Its success closeout is `archived`; its failure closeout remains sealed and status-safe.

## Deviations from Plan

None - the plan was executed exactly as written.

## Verification

- PASS — all 16 focused Plan 26 cases in `TestLifecycleCloseout199(...)` (0.805s).
- PASS — all `TestSealTransaction199.*` regression cases (41.959s).
- PASS — all `TestEntombTransaction199.*` transaction, replay, proof, and failure cases (10.999s).
- PASS — focused init/colonize/plan finalizer regression selection (28.306s).
- BASELINE-IDENTICAL — the broader legacy `TestEntomb*` selection retains ten migration-wide fixture failures already present at `e003b1df`; the Plan 26 transaction and closeout suites are green.

## Issues Encountered

- The general next-action state loader creates storage locks, which violated refusal fingerprint tests. The closeout refusal path now uses the existing pure resolver with already-read state facts, preserving policy ownership and zero writes.
- Ten old entomb command fixtures predate the current typed sealed-outcome preflight and fail before closeout. Running the same selection from an `e003b1df` archive produced the identical failures, confirming they were not introduced by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The focused terminal grammar is integrated across every lifecycle command named by Plan 26.
- Phase 199 remains in progress; Plan 21 is the first incomplete plan despite this Wave 12 plan executing out of numeric order.
- No Plan 26-owned blocker remains.

## Self-Check: PASSED

- All nine implementation/test files and this summary exist.
- All six TDD task commits are present in repository history.
- Focused closeout and terminal transaction regression suites pass.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
