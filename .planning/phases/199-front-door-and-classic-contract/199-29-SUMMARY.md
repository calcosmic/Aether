---
phase: 199-front-door-and-classic-contract
plan: "29"
subsystem: exact completion gates and lifecycle integration
tags: [go, testing, race-detector, gate-receipt, lifecycle, worktree-safety]
requires:
  - phase: 199-28
    provides: exact Classic coverage ledger and current lifecycle vocabulary ratchet
provides:
  - source-bound receipt for exact normal and race repository gates
  - machine-checked preservation proof for user-owned working-tree state
  - hermetic, bounded integration coverage for lifecycle, recovery, seal, and worktree behavior
affects: [phase-199-verification, release-gates, lifecycle-contracts, worktree-safety]
tech-stack:
  added: []
  patterns:
    - source revision and tree are certified separately from evidence-only receipt commits
    - subprocess-heavy integration tests inherit deadlines and isolate mutable process state
key-files:
  created:
    - .planning/phases/199-front-door-and-classic-contract/199-GATE-RECEIPT.json
    - cmd/phase199_gate_receipt_test.go
    - cmd/isolated_process_test.go
  modified:
    - cmd/session_flow_cmds.go
    - cmd/next_action.go
    - cmd/status.go
    - cmd/worktree.go
    - cmd/lifecycle_next_action_coverage_test.go
    - cmd/classic_coverage_199_test.go
    - .github/workflows/ci.yml
key-decisions:
  - "Any source change invalidates prior gate evidence; clear the receipt, bind the new source commit/tree, and rerun both exact gates."
  - "Fingerprint protected user-owned paths and permit only explicit evidence files plus the pre-existing config edit after the tested source revision."
  - "Keep retired recovery mechanics internal while current output teaches status, maintenance inspection, resume, seal review, and explicit entombing."
patterns-established:
  - "Gate receipt: exact command, zero exit, timestamps, source identity, output SHA-256, and ownership fingerprints are all required."
  - "Hermetic gate tests: global state and git/process fixtures run in isolated child processes with parent-bound deadlines."
requirements-completed: [SYNTH-01, CEC-01, CEC-02, CEC-04, CEC-08, LIFE-01, LIFE-02, LIFE-03, LIFE-04, LIFE-05, LIFE-06, PROOF-01]
duration: 8h 56m
completed: 2026-09-06
---

# Phase 199 Plan 29: Exact Completion Gates Summary

**A source-bound receipt now proves exact normal and race full-suite success, while machine-checking that user-owned state stayed untouched and closing the lifecycle integration drift exposed by those gates.**

## Performance

- **Duration:** 8h 56m (including an interrupted transport continuation)
- **Started:** 2026-09-05T22:50:14Z
- **Completed:** 2026-09-06T07:45:44Z
- **Tasks:** 2/2
- **Files modified:** 104
- **Scoped commits:** 62

## Accomplishments

- Added a non-forgeable completion receipt that requires the exact `go test ./...` and `go test ./... -race` commands, zero exits, fresh timestamps, source revision/tree identity, output hashes, and unchanged ownership fingerprints.
- Reconciled the cross-cutting Phase 199 lifecycle contract uncovered by the focused and full gates: canonical resume/status guidance, coequal next actions, seal/entomb evidence, safe recovery boundaries, and public command reachability now agree.
- Made the large integration suite hermetic and bounded through isolated child processes, registered cleanup ownership, parallel-safe fixtures, indexed proof lookup, and parent-deadline propagation.
- Preserved `.planning/STATE.md`, untracked `.gsd/`, untracked `199-PATTERNS.md`, and the pre-existing `.planning/config.json` edit byte-for-byte and status-for-status.
- Recorded final green evidence for source `f42bd0b2c5f03b4dbcef8a4d06b3aac3061b4106` / tree `8e4a5d1c0afa06f4488dec0ee6fdb8e729c3ddba`.

## Task Commits

The plan used repeated source-repair and receipt-rebind commits so no stale gate run could certify changed source. The complete scoped history is `a244e9ed^..a227b66b` (62 commits).

1. **Task 1: Define the gate receipt and ownership validator** — `a244e9ed`, `d45184b2`, `0baf4e3c`, `0f2a989d`, `1ea8829a`.
2. **Task 2: Repair gate-exposed lifecycle contracts and rebind evidence** — `39ae84ca` through `f1fa54cf` (38 sequential scoped commits).
3. **Task 2: Make exact gates hermetic and bounded** — `c7fe2651` through `2e249677` (16 sequential scoped commits).
4. **Task 2: Fix final-state schema validation and record final gates** — `f42bd0b2`, `bcfce07e`, `a227b66b`.

## Files Created/Modified

- `.planning/phases/199-front-door-and-classic-contract/199-GATE-RECEIPT.json` — exact command results, source identity, output digests, and protected ownership fingerprints.
- `cmd/phase199_gate_receipt_test.go` — schema mutation tests plus strict live receipt, repository-freshness, and fingerprint validation.
- `cmd/isolated_process_test.go` and `.github/workflows/ci.yml` — shared isolated-process runner and CI enforcement for hermetic spawn coverage.
- `cmd/session_flow_cmds.go`, `cmd/next_action.go`, `cmd/status.go`, and lifecycle tests/fixtures — canonical resume/status/next-action behavior and read-only no-colony handling.
- `cmd/lifecycle_closeout.go`, `cmd/entomb_cmd.go`, and seal/entomb tests — single, renderable closure evidence with retained review context.
- `cmd/worktree.go`, `cmd/worktree_safety.go`, `cmd/clash.go`, and worktree tests — isolated git operations, safe clash inspection, and owned cleanup.
- `cmd/classic_coverage_199_test.go`, `cmd/classic_contract_test.go`, and related fixtures — indexed, executable proof resolution for the final coverage gate.
- Platform command sources and inventories — aligned current public routes across Aether, Claude, OpenCode, and Codex-facing evidence.

## Verification

- Source-existence assertions for every required exact and prefix-selected test family — passed.
- Focused Phase 199 command lane plus `pkg/colony` lifecycle lane — passed.
- `go test ./...` — passed in 427 seconds; receipt SHA-256 `908c4b6e59b7e082df9d315ea75ce6b2b169016c8eac6f65e2b9fef618a7f9e9`.
- `go test ./... -race` — passed in 548 seconds; receipt SHA-256 `bf34c203efca7772e02866733ca80d9a240a0f9a1179089e998259d140f3cd7a`.
- `go test ./cmd -run '^TestPhase199GateReceiptSchema$' -count=1` — passed.
- `go test ./cmd -run '^TestPhase199GateReceipt$' -count=1` — passed before and after the evidence commit.

## Decisions Made

- Receipt evidence points at the final source-only commit, not at its later evidence commit; the validator allows only the receipt, coverage document, summary, and fingerprinted pre-existing config delta after that source revision.
- Every source repair reset previously collected gate evidence. Both final exact suites were rerun after the last validator change.
- The unrelated `pkg/codex` timeout test was not changed: 21 isolated race-instrumented runs passed, so the plan preserved the scope boundary and recorded the transient full-suite failure below.
- Generic project-state metadata updates were deliberately not run because this plan explicitly requires `.planning/STATE.md` to retain its captured ownership fingerprint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reconciled canonical lifecycle choices, resume state, and status output**

- **Found during:** Task 2 focused/full verification.
- **Issue:** Runtime projections, wrappers, fixtures, and command inventories disagreed about canonical resume/status routes and coequal lifecycle choices.
- **Fix:** Restored durable resume facts, explicit next-action overrides, the status dashboard envelope, current command inventories, and matching execution contracts.
- **Files modified:** Lifecycle runtime, wrapper, inventory, fixture, and test files under `cmd/` plus the three platform command sources.
- **Verification:** Required focused selectors and both final exact gates passed.
- **Committed in:** Representative commits `39ae84ca`, `1fcb7758`, `e9664511`, `38caa5c7`, `9db1b0ed`, `3a79a835`, `cae09b0b`, `80ecb4df`.

**2. [Rule 1 - Bug] Preserved complete seal and entomb transaction evidence**

- **Found during:** Task 2 full-suite verification.
- **Issue:** Curation, ceremony context, retained review state, and archive results could be dropped, duplicated, or rendered through stale expectations.
- **Fix:** Kept typed preflight/closure facts through the lifecycle transaction and aligned seal/entomb execution tests.
- **Files modified:** `cmd/lifecycle_closeout.go`, `cmd/entomb_cmd.go`, seal/entomb runtime tests, and Classic fixtures.
- **Verification:** Seal transaction, ceremony, entomb, focused Phase 199, normal, and race gates passed.
- **Committed in:** `4eab661e`, `bfd509f4`, `cefcadde`, `889a58d0`, `8ef23335`, `061747f7`, `2cf3a0c6`.

**3. [Rule 1 - Bug] Isolated worktree inspection and cleanup ownership**

- **Found during:** Task 2 full/race verification.
- **Issue:** Worktree clash inspection and git commands could leak repository-global state, while cleanup evidence did not prove ownership strongly enough.
- **Fix:** Isolated git operations, hardened clash inspection, registered created worktrees, and retained unregistered decoys.
- **Files modified:** `cmd/worktree.go`, `cmd/worktree_safety.go`, `cmd/clash.go`, `cmd/codex_build_worktree.go`, and worktree tests.
- **Verification:** Worktree safety/ownership tests and both exact gates passed.
- **Committed in:** `f4ecfc7f`, `c7a54ad2`, `99c4833c`.

**4. [Rule 1 - Bug] Restored public front-door reachability while keeping retired recovery internal**

- **Found during:** Task 2 vocabulary and reachability gates.
- **Issue:** Current routes were missing from public reachability evidence while retired recovery mechanics leaked into current-contract fixtures.
- **Fix:** Restored public command reachability, canonicalized active guidance, and ratcheted retired recovery to bounded internal compatibility evidence.
- **Files modified:** Command registration/guidance, recovery runtime, reachability tests, vocabulary inventory, and Classic contract fixtures.
- **Verification:** Front-door, current-vocabulary, recovery-route, compatibility, and exact repository gates passed.
- **Committed in:** `d6f72896`, `9fd57821`, `a1988e40`, `0484e633`.

**5. [Rule 3 - Blocking] Made the exact completion gates hermetic and bounded**

- **Found during:** Task 2 exact normal/race gates.
- **Issue:** Serial subprocess journeys, repeated proof scans, shared global state, and unbounded child setup made the completion gates too slow and susceptible to load-dependent interference.
- **Fix:** Added isolated-process execution, CI spawn auditing, safe parallelism, one-time proof indexing, and parent-bound setup deadlines.
- **Files modified:** `cmd/isolated_process_test.go`, integration tests, `cmd/classic_coverage_199_test.go`, and `.github/workflows/ci.yml`.
- **Verification:** Hermetic audit tests, focused proofs, and final 427-second normal / 548-second race gates passed.
- **Committed in:** `46f38358`, `2fd8779c`, `febdbb4a`, `25ba11b2`, `5b250ee3`, `09e1d385`, `c9dd5024`, `bd2f9ba7`, `2ebebf75`.

**6. [Rule 1 - Bug] Kept schema mutation coverage valid after receipt completion**

- **Found during:** Task 2 final receipt validation.
- **Issue:** The schema test labelled the live checked-in receipt as its incomplete fixture, so that assertion stopped being negative once the real receipt became complete.
- **Fix:** Added an explicit incomplete fixture and made the diagnostic status-neutral, then discarded all earlier evidence and reran both exact gates on the new source tree.
- **Files modified:** `cmd/phase199_gate_receipt_test.go`, `199-GATE-RECEIPT.json`.
- **Verification:** Schema, strict final receipt, focused lane, normal gate, and race gate all passed.
- **Committed in:** `f42bd0b2`, `bcfce07e`, `a227b66b`.

**Total deviations:** 6 auto-fixed (5 Rule 1 bugs, 1 Rule 3 blocker).
**Impact on plan:** The gate exposed real Phase 199 integration drift and test-infrastructure blockers. Repairs stayed within lifecycle/gate correctness, and every source change forced fresh evidence rather than reusing a prior pass.

## Known Stubs

None. Stub-token scan hits were negative-test fixtures, documentation placeholders, or pre-existing catalog descriptions rather than new incomplete implementation.

## Issues Encountered

- An early exact normal run failed once in `TestEveryLifecycleCommandEndsWithNextAction/starting` with empty subprocess JSON. The exact isolated test then passed, 12 concurrent stress repetitions passed, and the unchanged-source exact retry passed. The final post-fix normal gate was rerun from scratch and passed.
- An early exact race run failed once in `TestAvailabilityPreflightAttemptsAreBoundedByFailureClass/timeout_gets_one_retry` at its 500 ms test-only boundary. The exact isolated race test plus 20 repeated race runs all passed (21/21), and the unchanged-source exact retry passed. Per the scope boundary, no unrelated Phase 198 source was changed. The final post-fix race gate was rerun from scratch and passed.
- Completing the first receipt exposed the final-state schema-fixture bug described above. The receipt was reset and rebound before collecting the final pair of gate runs.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-29 now has exact focused, normal, race, freshness, and ownership-preservation proof; all 34 Phase 199 plans have summary artifacts once this file is committed.
- `.planning/STATE.md`, `.gsd/`, `199-PATTERNS.md`, and `.planning/config.json` remain user-owned and untouched. Any future intentional change to their captured fingerprints requires explicitly regenerating the receipt rather than silently invalidating it.
- The plan-specific ownership contract supersedes the generic GSD state updater here; this summary is the completion signal for Plan 29.

## Self-Check: PASSED

All three created artifacts exist, all 62 scoped commits are reachable, the requirements list matches the plan, only this summary is staged, and strict receipt validation still passes with the summary present.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-06*
