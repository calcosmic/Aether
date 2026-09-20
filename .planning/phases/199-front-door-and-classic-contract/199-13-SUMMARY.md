---
phase: 199-front-door-and-classic-contract
plan: "13"
subsystem: lifecycle-recovery
tags: [go, pause, resume, transactions, provenance, exactly-once, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "03"
    provides: Versioned PauseHandoff, receipt, state-effect, and recovery-provenance contracts
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Causally read-only LifecycleFacts and shared lifecycle projection policy
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Digest-backed multi-root lifecycle transactions, deterministic recovery, and receipt replay
  - phase: 199-front-door-and-classic-contract
    plan: "12"
    provides: Evidence-qualified signals and exact active-work localization
provides:
  - One versioned safe-boundary pause handoff containing partial work, attempts, decisions, context, repository/worktree evidence, lineage, blockers, signals, and verification
  - Confirmed/reconstructed/conflicting/unknown resume classification with zero runnable-state writes on conflict
  - Handoff-ID-keyed pause/resume transactions whose retries return the existing receipt and effect
  - Canonical pause/resume Cobra metadata plus bounded hidden pre-Cobra legacy input redirects expiring at 1.29
  - One canonical `/ant-pause` wrapper contract that closes through `/ant-resume`
affects: [plan-199-21-resume-parity, lifecycle-closeouts, command-catalog-snapshots, phase-202-checkpoints]

tech-stack:
  added: []
  patterns: [safe-boundary transaction, handoff-ID idempotency key, evidence-before-mutation, hidden parser redirect, runtime-owned wrapper]

key-files:
  created:
    - cmd/pause_resume_199_test.go
  modified:
    - cmd/session_flow_cmds.go
    - cmd/session_compat.go
    - cmd/normalize_args.go
    - .aether/commands/pause.yaml
    - .claude/commands/ant-pause.md
    - .claude/commands/ant/pause.md
    - .opencode/commands/ant/pause.md

key-decisions:
  - "Pause and resume use the handoff ID as their idempotency key, so interrupted commits resume one journal and verified replay returns one existing receipt."
  - "Resume mutates only confirmed or explicitly reconstructed evidence; conflicting state, session, repository, worktree, activity, or receipt facts stop with no runnable-state write."
  - "Legacy pause-colony/resume-colony inputs are exact-token pre-Cobra redirects only through 1.28 and never appear in Cobra aliases, help, completion, or canonical pause wrappers."

patterns-established:
  - "Pause boundary: load facts read-only, refuse a live worker boundary, build one complete handoff, then declare state/session/context/handoff targets before durable intent."
  - "Resume boundary: validate independent evidence first, reconstruct only from named durable sources, and keep stale-session/context cleanup inside the same transaction."
  - "Replay boundary: inspect the handoff-ID transaction journal and revalidate its receipt/targets instead of re-running decisions, cleanup, events, or artifact writes."

requirements-completed: [CEC-04, LIFE-04]

duration: 40min
completed: 2026-09-04
---

# Phase 199 Plan 13: Safe-Boundary Pause and Exactly-Once Resume Summary

**Pause now commits one complete recovery receipt at a named safe boundary, while resume consumes that handoff once only when state, session, repository, worktree, activity, and transaction evidence agree.**

## Performance

- **Duration:** 40 minutes
- **Started:** 2026-09-04T09:08:59Z
- **Completed:** 2026-09-04T09:48:45Z
- **Tasks:** 2/2
- **Files changed:** 8 active production/test/wrapper files plus 3 intentional legacy-wrapper deletions

## Accomplishments

- Replaced sequential pause state/session/context writes with one `LifecycleTransaction` that persists a validated `PauseHandoff`, both cross-references, the human handoff, and context as a single restartable operation.
- Captured unfinished task IDs, current attempt/run, scoped decisions, blockers, active signals, actual spawn lineage/status, last gate evidence, repository bytes, worktree facts, restart point, and the named safe boundary in one versioned record.
- Made resume classify evidence as confirmed, reconstructed, conflicting, or unknown before declaring writes; conflicting fixtures retain a byte-identical runnable-state fingerprint.
- Made crash retry and ordinary replay return the same durable receipt without a second event, decision, cleanup, worker, handoff, or state effect.
- Removed loader-side legacy session creation and moved session reconstruction, stale spawn clearing, context restoration, and summary updates inside the recovery transaction.
- Made `pause` and `resume` the only Cobra names and kept old exact input tokens in a bounded pre-Cobra redirect with a machine-tested `1.29.0` expiry marker.
- Updated the canonical pause YAML and Claude/OpenCode wrappers to invoke `aether pause`, report runtime-owned safe-boundary/receipt fields, and close through `/ant-resume`.

## Task Commits

1. **Task 1: Commit and restore one handoff exactly once**
   - `557e0d6c` — test(199-13): define pause resume recovery contract (RED)
   - `9a6b3b58` — feat(199-13): make pause resume exactly once (GREEN)
2. **Task 2: Synchronize the canonical pause surface**
   - `d437d297` — feat(199-13): synchronize canonical pause surface

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/pause_resume_199_test.go` — safe-boundary, full handoff, transaction-fault, replay, provenance, zero-write conflict, hidden-redirect, and wrapper ratchets.
- `cmd/session_flow_cmds.go` — transactional pause/resume implementation, complete handoff evidence, read-only attempt selection, provenance classifier, receipt replay, and content-sensitive repository/worktree fingerprints.
- `cmd/session_compat.go` — legacy session lookup remains a no-write compatibility shim; reconstruction belongs to the lifecycle transaction.
- `cmd/normalize_args.go` — exact-token pre-Cobra redirect and semver expiry check without Cobra alias registration.
- `.aether/commands/pause.yaml` — canonical public pause ceremony and runtime authority contract.
- `.claude/commands/ant-pause.md`, `.claude/commands/ant/pause.md`, `.opencode/commands/ant/pause.md` — generated thin adapters for the canonical command.
- `.claude/commands/ant-pause-colony.md`, `.claude/commands/ant/pause-colony.md`, `.opencode/commands/ant/pause-colony.md` — intentionally deleted stale public alias wrappers.
- `.planning/phases/199-front-door-and-classic-contract/deferred-items.md` — records aggregate-suite expectations that still encode the superseded public alias/recovery contract.

## Decisions Made

- A live build process is not a safe pause point. Pause reports `worker_completion_boundary` and writes no receipt until the worker reaches a boundary; interrupted execution records an explicit interrupted-attempt boundary.
- Git repositories use a content-sensitive HEAD/dirty fingerprint that ignores only transaction-owned runtime artifacts. Non-Git workspaces use a content-sensitive tree digest rather than fabricating a Git identity or refusing durable reconstruction.
- A structured pause requires state/session handoff references, handoff bytes, repository/worktree evidence, activity facts, and the pause receipt to agree. Any disagreement is a named conflict and produces no runnable-state mutation.
- Reconstructed recovery is allowed from consistent durable state/session or a readable legacy handoff, but it receives reconstructed provenance and its own transaction-bound handoff evidence rather than being flattened into confirmed recovery.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking recovery] Removed abandoned pre-intent staging before a deterministic retry**

- **Found during:** Task 1 transaction fault matrix
- **Issue:** A crash after staging but before durable intent left a transaction-owned directory that the coordinator correctly refused to reinterpret as a file target, blocking a same-ID retry even though no live target had changed.
- **Fix:** Added a contained cleanup that removes only the exact transaction-owned scratch directories when no durable intent exists; once intent exists it refuses cleanup and routes through journal recovery.
- **Files modified:** `cmd/session_flow_cmds.go`
- **Verification:** Both pause and resume recover from `after_stage:target-0001`, then replay with byte-identical runnable state and one receipt per lifecycle effect.
- **Committed in:** `9a6b3b58`

**2. [Rule 2 - Missing critical evidence] Added a non-Git repository fingerprint**

- **Found during:** Task 1 broader subprocess compatibility verification
- **Issue:** Reconstructed resume in a workspace without Git could not satisfy the required repository evidence class.
- **Fix:** Added a deterministic content tree digest that excludes only lifecycle-owned state, lock, handoff, and transaction paths while retaining all project bytes.
- **Files modified:** `cmd/session_flow_cmds.go`
- **Verification:** Versioned plan-revision and compiled install-to-seal subprocess recovery journeys pass in non-Git fixtures.
- **Committed in:** `9a6b3b58`

**3. [Rule 2 - Missing critical contract cleanup] Deleted stale generated pause-colony wrappers**

- **Found during:** Task 2 `TestCommandSourceHygiene`
- **Issue:** Removing the YAML alias left three generated public wrappers with no canonical source, contradicting D-13 and failing source hygiene.
- **Fix:** Deleted the flat Claude, nested Claude, and OpenCode `pause-colony` wrappers and pinned their absence in `TestPauseWrapperContract199`.
- **Files modified:** `.claude/commands/ant-pause-colony.md`, `.claude/commands/ant/pause-colony.md`, `.opencode/commands/ant/pause-colony.md`, `cmd/pause_resume_199_test.go`
- **Verification:** `TestPauseWrapperContract199` and `TestCommandSourceHygiene` pass.
- **Committed in:** `d437d297`

---

**Total deviations:** 3 auto-fixed (1 blocking recovery, 2 missing critical correctness/contract fixes)
**Impact on plan:** All three fixes were necessary to satisfy the planned crash recovery, evidence completeness, and hidden-public-surface guarantees; no new product surface or dependency was added.

## Verification

- PASS — `go test ./cmd -run '^TestPauseResume199(SafeBoundary|HandoffContents|ReplayExactlyOnce|Confirmed|Reconstructed|ConflictZeroWrite|HiddenRedirect)$' -count=1` (18 cases including subtests)
- PASS — `go test ./cmd -run '^TestPauseWrapperContract199$' -count=1`
- PASS — `go test ./cmd -run '^TestCommandSourceHygiene$' -count=1` (7 cases)
- PASS — `go test ./pkg/colony -count=1` (264 cases)
- PASS — direct metadata scan finds no public `pause-colony`/`resume-colony` Cobra `Use` or `Aliases` entry and no `pause-colony` text in canonical/generated pause surfaces.

## TDD Gate Compliance

- RED `557e0d6c` precedes GREEN `9a6b3b58`; the RED gate failed on the intentionally absent transaction, outcome, handoff-path, and hidden-redirect APIs.
- The wrapper contract failed against the legacy description, alias declaration, host-owned wording, and absent receipt/closeout fields before `d437d297` made the canonical and generated surfaces pass.

## Known Stubs

None. The plan-owned implementation and wrappers contain no TODO/FIXME, placeholder/coming-soon copy, mock-only result, or hardcoded empty UI data source. The remaining resume wrapper migration is explicitly owned by Plan 199-21 rather than presented here as completed behavior.

## Issues Encountered

- The aggregate `go test ./cmd -count=1` still contains staged Phase 199 failures. The directly related cases require the old public `pause-colony`/`resume-colony` aliases or the older contradictory interrupted-state shape; other failures match previously recorded status/Next Up, shared golden, reachability, and archived-path migrations. Exact evidence and owning follow-ups are recorded in `deferred-items.md`; no unrelated tests or production paths were weakened.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-21 can mirror the same canonical/evidence contract onto the resume wrapper surface and remove its remaining planned suffixed compatibility wrappers.
- Phase 202 can consume the named safe-boundary, handoff, activity, and receipt evidence without inventing a second recovery vocabulary.
- Shared alias/catalog/reachability snapshots still need their owning later Phase 199 migrations before the final normal/race repository gate.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remain untouched and unstaged.

## Self-Check: PASSED

- All eight active Plan 13 implementation/test/wrapper artifacts and this summary exist; all three intentionally retired alias wrappers are absent.
- RED `557e0d6c`, GREEN `9a6b3b58`, and wrapper `d437d297` resolve in Git in the documented order.
- The summary and deferred-item update pass whitespace validation.
- All focused plan tests, wrapper/source-hygiene checks, and colony-model tests pass.
- Protected pre-existing paths remain unstaged and uncommitted.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
