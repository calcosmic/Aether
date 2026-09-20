---
phase: 200-iterative-planning
plan: 38
subsystem: planning-presentation
tags: [go, tdd, lifecycle-facts, candidate-standing, next-action, cobra, json, terminal]

requires:
  - phase: 200-37
    provides: Pure candidate-standing assessment, exact acceptance deadline, typed refusal facts, and atomic expiry transition
  - phase: 200-31
    provides: Immutable candidate identity, exact acceptance command, and active plan-revision authority
provides:
  - One captured candidate-standing lifecycle fact model for current, stale, expired, and accepted authority
  - Semantically identical terminal and JSON candidate projections with exact untruncated authority values and commands
  - Real Cobra nonzero refusal rendering that preserves Plan 37 typed expiry details
  - Next Up guidance driven by captured facts, with refresh replacing every stale acceptance or execution action
affects: [planning-review, lifecycle-facts, next-action, plan-acceptance, terminal-output, json-output]

tech-stack:
  added: []
  patterns: [single captured standing, shared semantic projection, fact-driven next action, typed nonzero command refusal]

key-files:
  created:
    - cmd/planning_expiry_presentation_200_test.go
  modified:
    - cmd/lifecycle_facts.go
    - cmd/planning_visuals.go
    - cmd/next_action.go
    - cmd/codex_workflow_cmds.go
    - cmd/planning_visuals_test.go

key-decisions:
  - "Lifecycle facts assess candidate standing once at CapturedAt; renderers and Next Up consume that captured result without another clock read or state write."
  - "The newest reviewable authority frontier wins: a newer expired candidate continues to require refresh instead of silently exposing an older accepted plan's build action."
  - "The complete D-16 acceptance command is captured in lifecycle facts and passed through unchanged; the resolver never reconstructs security-sensitive flags."
  - "The existing lifecyclePlanningFromSnapshot signature remains as a compatibility adapter over the captured-time implementation, keeping production callers outside the six-file scope unchanged."

patterns-established:
  - "Presentation authority: terminal, JSON, Cobra errors, and Next Up derive from one standing vocabulary and exact command values."
  - "Expiry recovery: stale or expired candidate authority recursively excludes accept/build/run and exposes only aether plan --refresh."

requirements-completed: [PLAN-03, PLAN-05, PLAN-06]

duration: 31min
completed: 2026-09-09
---

# Phase 200 Plan 38: Candidate Expiry Presentation Summary

**Candidate authority now reads the same in terminal, JSON, lifecycle facts, Cobra failures, and Next Up: current work can be reviewed and accepted exactly, while stale or expired work can only be refreshed.**

## Performance

- **Duration:** 31 min
- **Started:** 2026-09-09T03:13:37Z
- **Completed:** 2026-09-09T03:44:37Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added a state/format matrix for current, every early-stale reason, exact-deadline expiry, persisted expiry, and accepted candidates at widths 32, 47, 48, 63, 64, and 80, including color, `NO_COLOR`, JSON, and exact live-command checks.
- Extended lifecycle planning facts with one captured standing, expiry, reason, evidence, state and active-plan effects, acceptance availability, exact acceptance command, and recovery command.
- Routed visual and JSON candidate review through one semantic projection, retaining complete IDs, hashes, timestamps, labels, and commands without truncation.
- Preserved Plan 37's typed refusal object through the real Cobra error boundary: exact-deadline acceptance now returns nonzero while JSON and the dedicated visual error name expiry, state mutation, unchanged active plan, and `aether plan --refresh`.
- Made Next Up use the captured fact model: current candidates offer review plus exact acceptance, stale/expired candidates offer refresh only, and accepted candidates retain the existing build/run path.

## Task Commits

Each task followed a fail-first TDD boundary and was committed atomically:

1. **Task 1 RED: Define candidate expiry presentation contract** - `feb16710` (test)
2. **Task 1 GREEN: Render candidate standing without authority drift** - `4258f6dc` (feat)
3. **Task 2 RED: Require captured standing in lifecycle guidance** - `a14f6848` (test)
4. **Task 2 GREEN: Drive next action from captured candidate standing** - `2d542a61` (feat)

**Plan metadata:** `dcbf8939` (docs), followed by a scoped provenance-restoration commit.

## Files Created/Modified

- `cmd/planning_expiry_presentation_200_test.go` - State, width, color, JSON, real-Cobra refusal, lifecycle capture, and Next Up parity matrix.
- `cmd/planning_visuals.go` - Shared candidate semantic projection plus dedicated typed refusal rendering.
- `cmd/codex_workflow_cmds.go` - Preserves typed candidate refusal details through JSON or visual nonzero errors.
- `cmd/lifecycle_facts.go` - Loads the canonical candidate frontier read-only and assesses it once at `LifecycleFacts.CapturedAt`.
- `cmd/next_action.go` - Selects review, exact acceptance, refresh, or execution strictly from captured candidate facts.
- `cmd/planning_visuals_test.go` - Updates golden candidate review expectations for the explicit standing and authority fields.

## Decisions Made

- Candidate expiry is presentation authority, not decoration: if the newest candidate is stale or expired, an older accepted revision cannot make build/run look like the next safe step.
- Lifecycle loading may read the canonical candidate artifact because a not-yet-accepted candidate is deliberately absent from `COLONY_STATE`; the original state fact remains unchanged and no reader persists anything.
- Exact acceptance command assembly remains at the authority boundary. Lifecycle facts carry the full command, and Next Up treats it as an opaque executable value.
- The old five-argument lifecycle projection helper remains available for existing callers and delegates to the captured-time implementation. This avoided changing `seal_outcome.go` or any test outside Plan 38 ownership.

## Verification

- Task 1 plan command passed: `rtk go test ./cmd -run '^(TestPlanningExpiryPresentation200|TestGoldenPlanVisualOutput)$' -count=1` — 16 passed.
- All lifecycle-fact regressions passed: `rtk go test ./cmd -run '^TestLifecycleFacts' -count=1` — 19 passed.
- Exact scoped Plan 38 matrix passed: `rtk go test ./cmd -run '^(TestPlanningExpiryPresentation200|TestGoldenPlanVisualOutput|Test.*NextAction.*|Test.*LifecycleFacts.*Planning.*)$' -skip '^(TestNextActionNeverHardcoded|TestEveryLifecycleCommandEndsWithNextAction)$' -count=1` — 100 passed, 0 failed.
- Core state/format/Cobra/facts/Next Up matrix repeated three times — 54 passed, 0 failed.
- The same core matrix under `go test -race` — 18 passed, 0 failed, no data races.
- `rtk go vet ./cmd`, `git diff --check 94cada84..HEAD`, exact six-file ownership, and deletion checks passed.
- The repository-wide `go test ./...` was not run: this plan calls for bounded presentation/lifecycle verification and the repository records that full command as an approximately 11-minute gate for Plan 40.

## TDD Gate Compliance

- RED commit `feb16710` failed to compile on the deliberately absent shared candidate projection fields.
- GREEN commit `4258f6dc` passed the terminal/JSON/width/color and real Cobra refusal boundary.
- RED commit `a14f6848` failed to compile on the nine deliberately absent additive lifecycle standing fields.
- GREEN commit `2d542a61` passed the candidate-fact and fact-driven Next Up matrix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Correctness Bug] Read the canonical candidate artifact before lifecycle projection**
- **Found during:** Task 2 GREEN lifecycle fact implementation
- **Issue:** Pending candidates intentionally are not written into `COLONY_STATE` before acceptance, so state-only lifecycle loading could not expose their standing even though the canonical review artifact existed.
- **Fix:** Added a read-only artifact load inside `cmd/lifecycle_facts.go`, merged it into an assessment snapshot, and retained the original state fact and provenance unchanged.
- **Files modified:** `cmd/lifecycle_facts.go`
- **Verification:** Current, deadline-expired, persisted-expired, early-stale, and accepted lifecycle fixtures passed; all 19 `TestLifecycleFacts*` tests passed.
- **Committed in:** `2d542a61`

**2. [Rule 1 - Compatibility Bug] Preserved the existing lifecycle projection helper signature**
- **Found during:** Task 2 GREEN compile boundary
- **Issue:** Replacing the existing helper signature with captured-time arguments would require edits to production and tests outside the plan's strict six-file ownership.
- **Fix:** Kept the established helper as a compatibility adapter and added an internal captured-time variant used by the lifecycle reader.
- **Files modified:** `cmd/lifecycle_facts.go`
- **Verification:** All lifecycle-fact regressions and the scoped 100-test Plan 38 matrix passed.
- **Committed in:** `2d542a61`

---

**Total deviations:** 2 auto-fixed (2 Rule 1 correctness/compatibility bugs)
**Impact on plan:** Both fixes were required to deliver one honest captured fact model while preserving strict ownership; no dependency, public schema authority, mutation path, or out-of-scope file was added.

## Issues Encountered

The plan's literal Task 2 regex includes two failures that predate Plan 38 and are outside its declared files:

1. `rtk go test ./cmd -run '^TestNextActionNeverHardcoded$' -count=1` fails because `cmd/codex_plan.go` now contains `runCodexPlanAgentDelegateInSession` while `cmd/testdata/next_action_hardcode_baseline.json` still names `runCodexPlanAgentDelegate`. The exact assertions report one new hand-typed site at `next_action_hardcode_ratchet_test.go:525` and one stale baseline entry at line 530.
2. `rtk go test ./cmd -run '^TestEveryLifecycleCommandEndsWithNextAction$' -count=1` fails in its isolated child because the building fixture prints only `Step 1/5: Prepare (1s)` instead of ending in the shared card, then resume refuses a test data root outside its repository with `repository containment refused ... path escapes its root` (`lifecycle_next_action_coverage_test.go:302` and line 335).

These are confirmed baseline failures, not Plan 38 regressions. A clean archive of pre-plan commit `94cada84` at `/private/tmp/aether-plan38-baseline.bqdAtp` reproduced both exact commands and assertions unchanged: each command reported 0 passed / 1 failed. The plan's full literal regex therefore reports **100 passed / 2 failed**, while the exact scoped regex shown under Verification reports **100 passed / 0 failed**. Fixing either baseline would require unowned production/test files or weakening physical containment, so both were left untouched for the owning plan.

The installed GSD state synchronizer again removed Phase 200 provenance keys and treated the completed Phase 199 as current-milestone progress while recording this plan. The normal SDK update was committed first as `dcbf8939`; Phase 200 identity, the zero-complete-current-phase progress convention, and `state_head` were then restored using the same scoped follow-up pattern as Plans 34, 35, and 37. `state.advance-plan` was intentionally not called because numeric increment would select Plan 39, while declared wave order makes Plan 36 the next incomplete plan.

No authentication, package, architectural, or external-service blocker occurred.

## Known Stubs

None. The two existing user-facing phrases containing “not available” are real defensive guidance, not placeholders; no empty/mock value was introduced into a rendered path.

## Threat Surface

- No network endpoint, authentication path, package, schema, or new write boundary was introduced.
- The only added file access is a read-only load of the existing canonical planning-candidate artifact under the already-contained repository planning directory.
- Spoofing, action-escalation, and narrow-output disclosure risks are covered by explicit standing/evidence fields, recursive forbidden-action tests, exact command-tree resolution, and nontruncation checks.

## Human Verification Still Required

The two plan assumptions remain explicitly unresolved; automated success is not being presented as owner approval:

- **PLAN-03:** A human must judge whether the live terminal hierarchy makes the current specification authority immediately understandable.
- **PLAN-05:** A human must judge whether the live candidate decision and stale-before-expiry explanation are clear enough to act on confidently.

## User Setup Required

None - no packages, credentials, migrations, or external services were added.

## Next Phase Readiness

- Plan 36 can continue the remaining build-start fixture migration without changing the candidate presentation contract.
- Plan 39 can consume one authoritative candidate standing across its full public-journey and attack proofs.
- Plan 40 retains the repository-wide normal/race gate and final owner-visible verification, including the two unresolved visual-clarity assumptions above.
- No Plan 38 implementation, focused, repeated, race, vet, diff, ownership, deletion, or protected-state blocker remains.

## Self-Check: PASSED

- All six declared source/test files and this summary exist.
- Commits `feb16710`, `4258f6dc`, `a14f6848`, `2d542a61`, and metadata commit `dcbf8939` exist in repository history in RED/GREEN/closeout order.
- Focused, repeated, race, vet, diff, ownership, deletion, and protected-state checks pass.
- Protected `.planning/config.json`, `.gsd/`, and Phase 199 PATTERNS dirt remains present and unstaged; the two byte-hashed files exactly match their pre-execution hashes.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-09*
