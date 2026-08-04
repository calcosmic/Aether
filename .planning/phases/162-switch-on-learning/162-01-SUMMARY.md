---
phase: 162-switch-on-learning
plan: 01
subsystem: learning
tags: [go, hive-brain, cross-colony-wisdom, policy-default, consent-gate-retirement]

# Dependency graph
requires:
  - phase: 160-fail-loudly
    provides: shared foundation (loud failures, drift-test fixes) all phase-162 plans depend on
provides:
  - AETHER_HIVE_POLICY as the single hive control surface (promote by default, explicit off, fail-safe on typos)
  - Retirement of the per-colony consent-file gate (hiveRetrievalOptedIn) and its two cobra commands
  - AGENTS.md hive gating section rewritten to describe actual single-switch behavior
affects: [162-06-decision-record, 170-reclaim-queen-seed-from-hive]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "sync.Once-guarded stderr warning for unrecognized env-var values (fail-safe + discoverable, not silent)"
    - "Table test with load-bearing regression proof (temporarily reverting the fix must fail the test)"

key-files:
  created: []
  modified:
    - cmd/hive_policy.go
    - cmd/hive_policy_test.go
    - cmd/hive.go
    - cmd/context_weighting.go
    - cmd/colony_prime_context.go
    - cmd/hive_test_optin.go (deleted)
    - cmd/codex_prompt_context_test.go
    - cmd/colony_prime_audit_test.go
    - cmd/colony_prime_budget_test.go
    - cmd/colony_prime_context_test.go
    - cmd/hive_runtime_test.go
    - cmd/pheromone_loader_test.go
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/command_catalog.json
    - cmd/testdata/regression_snapshot.json
    - AGENTS.md

key-decisions:
  - "Unrecognized AETHER_HIVE_POLICY values fail safe to off (not promote), per RESEARCH.md A3 — a typo must not silently widen cross-repo data flow"
  - "colony_prime_context.go's withheld-wisdom warning surfaces unconditionally now, not gated on opt-in state, since no opt-in state exists anymore"
  - "LEARN-04 is only partially satisfied by this plan (the control half); the formal decision record is plan 06's task per this plan's own success_criteria — not marked complete here"

patterns-established:
  - "Single control surface for machine-wide policy switches: one env var, explicit case for every named value, default case treated as unrecognized-and-fail-safe rather than falling through"

requirements-completed: []  # LEARN-04 control half only — decision record is plan 06's scope; not marking complete here

# Metrics
duration: ~75min
completed: 2026-08-04
---

# Phase 162 Plan 01: Switch On Learning — Hive Default Flip Summary

**Flipped `AETHER_HIVE_POLICY` to promote-by-default and deleted the per-colony consent-file gate that silently vetoed it, leaving exactly one control surface for cross-colony wisdom.**

## Performance

- **Duration:** ~75 min
- **Started:** 2026-08-04T11:03:00Z (approx, first tool call)
- **Completed:** 2026-08-04T11:27:55Z
- **Tasks:** 3 (plus 1 deviation fix)
- **Files modified:** 16 (15 planned + 1 deviation: `cmd/testdata/regression_snapshot.json`)

## Accomplishments

- `currentHiveRuntimePolicy()` now resolves unset/empty to `promote` (D-01), keeps explicit `off` working via its own switch case (Pitfall 4), and fails unrecognized values safe to `off` with a stderr warning naming the offending value, guarded by `sync.Once` so it fires exactly once per process (T-162-04)
- Retired the entire `hiveRetrievalOptedIn()` consent-file mechanism (path resolution, read, write, the `hive-opt-in`/`hive-opt-out` cobra commands and their registration) and its three call sites in `cmd/hive.go`, `cmd/context_weighting.go`, and `cmd/colony_prime_context.go` — D-02's "exactly one control surface"
- `colony_prime_context.go`'s withheld-wisdom warning now surfaces unconditionally (a genuine meaning change, not just a deletion) since there is no opt-in state left to gate the warning on
- Deleted `cmd/hive_test_optin.go`'s `enableHiveForTest` helper and removed its calls from all 7 sites across 6 test files — tests no longer need to opt in since promote is now the default
- Rewrote `AGENTS.md`'s "Retrieval is opt-in, twice over" section into "Retrieval is on by default, through one switch," naming all four `AETHER_HIVE_POLICY` resolutions and removing every reference to the retired consent file/commands

## Task Commits

Each task was committed atomically:

1. **Task 1: Flip the hive policy default to promote, keep explicit off, fail safe on typos** — TDD cycle:
   - `2784b53b` (test): failing table test pinning the D-02 resolution table
   - `2dc1273b` (feat): implementation — unset/empty → promote, explicit `off` case, unrecognized → warn-once + off
2. **Task 2: Retire the consent-file gate at all four references plus its two commands** — `9a5bccaf` (feat)
3. **Task 3: Rewrite AGENTS.md's hive gating section to describe the single switch** — `b96e69fe` (docs)

**Deviation fix:** `3b1f2e94` (fix) — regression snapshot golden-value update, see below.

_Note: This SUMMARY.md commit itself follows per the git_commit_metadata step._

## Files Created/Modified

- `cmd/hive_policy.go` — single-switch policy resolution with `sync.Once`-guarded warning
- `cmd/hive_policy_test.go` — added `TestHiveRuntimePolicyDefault` (11-case table + 1 derived-helper subtest) and `TestHiveRuntimePolicyUnrecognizedWarns`; rewrote `TestHiveWorkerReadIsOffByDefaultButManualReadRemainsAvailable` → `TestHiveWorkerReadIsOnByDefaultAndOffSwitchDisablesIt` to assert the new default-on behavior and the `off` compensating control
- `cmd/hive.go` — deleted consent-file constant/path/read/write, the two opt-in/opt-out commands, their `AddCommand` registration, and the `hive-read --for-worker` consent check
- `cmd/context_weighting.go` — deleted the consent check in `readHiveWisdomEntriesForDomains`
- `cmd/colony_prime_context.go` — withheld-wisdom warning now surfaces unconditionally instead of gated on opt-in
- `cmd/hive_test_optin.go` — deleted (its only purpose was satisfying the retired gate)
- `cmd/codex_prompt_context_test.go`, `cmd/colony_prime_audit_test.go`, `cmd/colony_prime_budget_test.go`, `cmd/colony_prime_context_test.go`, `cmd/hive_runtime_test.go`, `cmd/pheromone_loader_test.go` — removed 7 `enableHiveForTest(t)` calls
- `cmd/colony_prime_audit_test.go` — additionally updated `TestColonyPrimeGracefulWithMissingData` to expect the new unconditional hive-fallback warning instead of zero warnings
- `cmd/testdata/parity_snapshot.json`, `cmd/testdata/command_catalog.json` — removed `hive-opt-in`/`hive-opt-out` entries
- `cmd/testdata/regression_snapshot.json` — command_count golden value updated 401 → 399
- `AGENTS.md` — hive command table and gating section rewritten to describe actual behavior

## Decisions Made

- **Unrecognized policy values fail safe to `off`, not `promote`** (resolves RESEARCH.md's open assumption A3 in favor of the fail-safe branch): a typo must not silently widen cross-repo data flow, and the stderr warning makes the failure discoverable rather than silent.
- **`colony_prime_context.go`'s Site 3 is a meaning change, not a deletion**: the withheld-wisdom warning used to be suppressed for colonies that hadn't opted in (the default state of every colony); with no opt-in state left, the warning now always surfaces, matching the plan's explicit instruction and the design rule that wisdom adoption is never silent.
- **LEARN-04 is only partially closed by this plan.** The plan's own `<success_criteria>` states "the written decision record itself is plan 06's task" — so `requirements-completed` intentionally lists nothing here; the control implementation (this plan) and the formal decision record (plan 06) are treated as two halves of one requirement, consistent with CLAUDE.md's Definition of Done (a checked box without an executable proof is not done).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Regression snapshot golden value stale after command removal**
- **Found during:** Full-suite verification after Task 2
- **Issue:** `TestRegressionSnapshot` pins the total command count as a golden value (401). Retiring `hive-opt-in` and `hive-opt-out` (Task 2, required by the plan) drops the real command count to 399 — exactly the expected delta — so the test failed with `command_count mismatch: got 399, want 401`.
- **Fix:** Updated only the `command_count` field in `cmd/testdata/regression_snapshot.json` (401 → 399); left all five other audit dimensions untouched since nothing else in this plan's scope changes castes, gates, artifacts, or sync pairs.
- **Files modified:** `cmd/testdata/regression_snapshot.json`
- **Verification:** `go test ./cmd/ -run 'TestRegressionSnapshot$' -count=1 -v` passes; full `go test ./cmd/... -count=1` subsequently green.
- **Committed in:** `3b1f2e94`

**2. [Investigation, no code change] Two apparent full-suite failures traced to concurrent-edit races, not real regressions**
- **Found during:** An early full-suite background run (`go test ./cmd/... -count=1`) that was started immediately after Task 1 and then overlapped with Task 2's file edits in progress
- **Issue:** `TestPackedNPMReleaseCandidateContract` failed with a Go compile error referencing symbols mid-deletion, and later `TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild` failed asserting "black-box command changed the source checkout" — both because the test suite's own internal release-binary build and git-diff snapshot ran while I was still editing/committing files for Task 2 concurrently with the background test process.
- **Fix:** No code change needed. Re-ran both tests in isolation once the working tree was stable (all Task 2 edits committed) and both passed (`TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild` — PASS, 16.04s). A final clean full-suite run (`go test ./cmd/... -count=1`, 266.7s) confirmed `ok` with zero failures.
- **Files modified:** None
- **Verification:** Isolated reruns of both tests passed; final full `cmd` package suite green.

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug), plus 1 investigated-and-cleared false alarm (concurrent-edit race, no code change required).
**Impact on plan:** The regression-snapshot fix is a direct, necessary consequence of Task 2's required command removal — no scope creep. The race-condition investigation added verification time but changed no code.

## Issues Encountered

- Two full-repo `go test ./cmd/... -count=1` runs took 4.5–6 minutes each due to the test suite building a full release binary internally (`TestPackedNPMReleaseCandidateContract`) and running many subprocess-spawning integration tests, compounded by concurrent load from other processes on the shared machine. Resolved by waiting for background completion and cross-checking suspicious failures in isolation rather than assuming they were real.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The hive control surface (`AETHER_HIVE_POLICY`) and its documentation are now internally consistent and test-pinned (`TestHiveRuntimePolicyDefault`, `TestHiveRuntimePolicyUnrecognizedWarns`, `TestHiveWorkerReadIsOnByDefaultAndOffSwitchDisablesIt`).
- Plan 06 (later wave) still owns the formal D-01/D-02/D-03 decision-record artifact and any remaining AGENTS.md sections outside hive gating (learning-pipeline, consolidation) — this plan deliberately did not touch those.
- Phase 170's RECLAIM-09 (`queen-seed-from-hive`) can now build on a settled hive-on-by-default posture.
- No blockers identified for downstream plans in this phase.

---
*Phase: 162-switch-on-learning*
*Completed: 2026-08-04*
