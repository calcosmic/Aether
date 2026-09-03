---
phase: 199-front-door-and-classic-contract
plan: "11"
subsystem: lifecycle-autopilot
tags: [go, autopilot, lifecycle-projection, repair-receipts, watch, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Digest-backed lifecycle transaction and receipt vocabulary
  - phase: 199-front-door-and-classic-contract
    plan: "07"
    provides: Guided front door and accepted-charter lifecycle identity
  - phase: 199-front-door-and-classic-contract
    plan: "08"
    provides: Read-only typed territory evidence used by lifecycle projection
  - phase: 199-front-door-and-classic-contract
    plan: "09"
    provides: Authoritative full and compact status projections
provides:
  - Zero-write Autopilot prerequisite preflight over immutable lifecycle facts
  - Visible bounded-operation and owner-authority contract with typed pause decisions
  - Repair receipts, one-use budget accounting, debt, and dependency-path reporting
  - Honest idle Watch fallback using compact status and timestamped recorded activity
  - One-call Claude and OpenCode run wrappers governed by the Go runtime result
affects: [autopilot, watch, lifecycle-closeouts, classic-contract-corpus, platform-wrappers]

tech-stack:
  added: []
  patterns: [read-only preflight before mutation, typed authority fence, persist-before-repair receipt, snapshot-not-liveness, passive runtime wrappers]

key-files:
  created:
    - cmd/autopilot_contract_199_test.go
    - cmd/watch_idle_199_test.go
  modified:
    - cmd/autopilot_policy.go
    - cmd/compatibility_cmds.go
    - cmd/run_visuals.go
    - cmd/codex_visuals_test.go
    - cmd/compatibility_cmds_test.go
    - .aether/commands/run.yaml
    - .claude/commands/ant-run.md
    - .claude/commands/ant/run.md
    - .opencode/commands/ant/run.md

key-decisions:
  - "Autopilot validates one accepted goal and an ordered remaining-phase set from immutable LifecycleFacts before opening its mutating loop."
  - "Owner-authority changes use a separate typed fence and pause outside the legacy ordered stage-trigger catalogue, preserving that catalogue's compatibility contract."
  - "Without a Phase 202 typed event source, Watch always reports idle and treats spawn/history rows as timestamped recorded evidence, never current liveness."
  - "Run wrappers invoke the Go runtime exactly once; invocation itself is consent to the displayed bounds and wrappers stop on typed owner-authority results."

patterns-established:
  - "Preflight then execute: invalid Autopilot entry returns a typed refusal without creating a store-backed mutation path."
  - "Repair accounting: planned receipt is durable before action, verification completes it once, and the budget decrements exactly once."
  - "Snapshot versus stream: idle Watch reuses compact LifecycleProjection and labels historical actor rows as recorded, not live."

requirements-completed: [CEC-02, CEC-04, LIFE-01, LIFE-03]

duration: 29min
completed: 2026-09-03
---

# Phase 199 Plan 11: Bounded Autopilot and Honest Idle Watch Summary

**Autopilot now refuses invalid entry without writes, displays and enforces bounded operating authority with repair/debt receipts, while Watch falls back to authoritative compact status instead of inventing live ants.**

## Performance

- **Duration:** 29 minutes
- **Started:** 2026-09-03T23:24:22Z
- **Completed:** 2026-09-03T23:52:55Z
- **Tasks:** 3/3
- **Files modified:** 11 production, wrapper, and test files

## Accomplishments

- Split `run` entry into a pure `AutopilotPreflight` over `LifecycleFacts` and the existing mutating execution loop, preserving exact zero-write init/plan refusal routes.
- Added the operating card, owner-only goal/scope/risk/acceptance fence, bounded repair ledger, verification receipts, exhaustion/debt reporting, and explicit owner-controlled seal handoff.
- Replaced Watch's snapshot files and simulated refresh behavior with the exact idle sentence, compact projection, timestamps, and recorded-history rows that are explicitly not live.
- Synchronized the canonical, Claude flat/nested, and OpenCode run wrappers to one exact description and one runtime call.

## Task Commits

Each task was committed atomically; the two behavior tasks followed RED then GREEN:

1. **Task 1: Enforce the Autopilot entry and authority contract**
   - `1b2ed269` — test(199-11): add failing autopilot contract tests (RED)
   - `0bb09b91` — feat(199-11): enforce bounded autopilot contract (GREEN)
   - `bbfde7a8` — fix(199-11): preserve autopilot trigger compatibility
2. **Task 2: Make idle watch an honest status-plus-activity fallback**
   - `0ecb456f` — test(199-11): add failing idle watch tests (RED)
   - `d46cc91a` — feat(199-11): make idle watch read-only (GREEN)
   - `d11e2db1` — test(199-11): align watch compatibility assertions
3. **Task 3: Synchronize the run operating contract**
   - `33d1c79d` — test(199-11): pin autopilot wrapper contract (RED)
   - `06a516b2` — docs(199-11): synchronize autopilot wrappers (GREEN)

## Files Created/Modified

- `cmd/autopilot_policy.go` — pure entry preflight, authority fence, repair eligibility, receipts, budget, debt, and path-routing policy.
- `cmd/compatibility_cmds.go` — run preflight/loop integration, typed authority stops, repair reporting, and read-only idle Watch result construction.
- `cmd/run_visuals.go` — invalid-entry block, operating contract, authority/repair receipts, and explicit-seal completion language.
- `cmd/autopilot_contract_199_test.go` — entry, authority, repair, completion, and wrapper contract proofs.
- `cmd/watch_idle_199_test.go` — exact idle fallback, no-fake-liveness, timestamp, and zero-write proofs.
- `cmd/codex_visuals_test.go` and `cmd/compatibility_cmds_test.go` — migrated obsolete Watch snapshot-write assertions to the new read-only contract.
- `.aether/commands/run.yaml` — canonical run wrapper contract and approved description.
- `.claude/commands/ant-run.md`, `.claude/commands/ant/run.md`, `.opencode/commands/ant/run.md` — generated one-call runtime delegates.

## Decisions Made

- Missing authority is a typed Autopilot pause but not a new member of the established ordered stage-trigger catalogue. This keeps existing ordering and disposition consumers compatible while stopping owner-controlled changes before application.
- The authority pause points back to `aether status`, whose persisted run report is the truthful recovery surface; it does not invent an unresolved decision record.
- A stale spawn-tree row may appear only under recent recorded activity with its timestamp and a non-live label. It cannot make `active_count` nonzero without the future typed event stream.
- Autopilot completion reports phases built and verified and names `aether seal`; it never seals automatically.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Compatibility] Kept the authority fence out of the legacy stage-trigger catalogue**

- **Found during:** Overall Autopilot/Watch regression verification
- **Issue:** The initial implementation inserted `missing_authority` into the ordered legacy trigger catalogue, breaking its established exact membership and disposition matrix.
- **Fix:** Retained the typed authority code and pause behavior in a dedicated fence path with an actionable status recovery command, without changing the catalogue.
- **Files modified:** `cmd/autopilot_policy.go`, `cmd/compatibility_cmds.go`, `cmd/autopilot_contract_199_test.go`
- **Verification:** `TestAutopilotTriggerCatalogue`, `TestAutopilotDispositionMatrix`, and `TestAutopilotContract199AuthorityFence` pass.
- **Committed in:** `bbfde7a8`

**2. [Rule 1 - Regression tests] Migrated directly superseded Watch assertions**

- **Found during:** Overall Autopilot/Watch regression verification
- **Issue:** Two existing tests still required Watch to write `watch-status.txt`/`watch-progress.txt` and present stale spawn-tree rows as active, behavior D-12 explicitly replaces.
- **Fix:** Changed those directly related assertions to require the idle sentence, authoritative compact status, timestamped recorded evidence, unsupported live capability, and zero writes.
- **Files modified:** `cmd/codex_visuals_test.go`, `cmd/compatibility_cmds_test.go`
- **Verification:** The migrated Watch tests and all three `TestWatchIdle199*` tests pass; the broader selector returns only the same two failures present at the pre-plan commit.
- **Committed in:** `d11e2db1`

---

**Total deviations:** 2 auto-fixed (2 Rule 1 compatibility/regression corrections)
**Impact on plan:** Both changes preserve the requested contract and prevent Plan 11 from adding failures to the existing Autopilot/Watch baseline; no unrelated legacy expectation was rewritten.

## Verification

- PASS — `go test ./cmd -run '^TestAutopilotContract199(InvalidZeroWrite|OperatingCard|StartsImmediately|AuthorityFence|RepairEligibility|RepairReceipt|RepairVerification|BudgetExhaustion|DebtReport|IndependentContinuation|CompletionDoesNotSeal)$' -count=1`
- PASS — `go test ./cmd -run '^TestWatchIdle199(StatusFallback|NoFakeLiveness|ReadOnly)$' -count=1`
- PASS — `go test ./cmd -run '^TestAutopilotWrapperContract199$' -count=1`
- PASS — `go test ./cmd -run '^TestCommandSourceHygiene$' -count=1`
- PASS — `TestWatchVisualOutputShowsHonestIdleFallback`, `TestWatchCompatibilityUsesReadOnlyFallback`, `TestAutopilotTriggerCatalogue`, and `TestAutopilotDispositionMatrix`.
- BASELINE NON-GREEN — `go test ./cmd -run '(Autopilot|Watch)' -count=1` reports only `TestSwarmCompatibilityWatchShowsRecoveryGuidance` and `TestRunAutopilotReplanDue`; the same two tests fail against pre-plan commit `8966eefc`.
- BASELINE NON-GREEN — `go test ./cmd -count=1` completed with the repository's documented staged Phase 199 migration failures. No named Plan 11 selector failed; later plans own the unchanged broad migration set.
- PASS — `git diff --check`

## TDD Gate Compliance

- Task 1: RED `1b2ed269` precedes GREEN `0bb09b91`.
- Task 2: RED `0ecb456f` precedes GREEN `d46cc91a`.
- Task 3's required wrapper-first test: RED `33d1c79d` precedes implementation `06a516b2`.

## Known Stubs

None. The scan found no placeholder, TODO/FIXME, empty UI value, or mock-only data path in Plan 11's created or modified production surfaces.

## Issues Encountered

- The full command suite is intentionally non-green during the staged Phase 199 migration. A pre-plan archive comparison was used for the relevant Autopilot/Watch surface so Plan 11 regressions could be separated from that baseline without changing unrelated tests.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 12 can consume the stable read-only preflight, operating contract, repair report, and honest snapshot-versus-stream boundary.
- Phase 202 remains responsible for a real typed Watch event stream; until then Watch truthfully reports the capability as unsupported.
- Phase 201 remains responsible for the broader work-cycle redesign; Plan 11 did not introduce auto-seal or a second scheduler.
- The protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remain untouched and uncommitted.

## Self-Check: PASSED

- All created and modified Plan 11 artifacts exist.
- All eight task, TDD, compatibility, and wrapper commits resolve in Git.
- Focused verification and whitespace validation pass.
- Protected pre-existing paths remain unstaged and uncommitted.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
