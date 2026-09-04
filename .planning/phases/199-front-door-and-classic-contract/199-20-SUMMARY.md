---
phase: 199-front-door-and-classic-contract
plan: "20"
subsystem: lifecycle-front-door
tags: [go, cobra, init, wrappers, codex, accepted-charter, territory]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "07"
    provides: Guided runtime init, automatic setup, accepted-charter persistence, and lifecycle-derived closeout
  - phase: 199-front-door-and-classic-contract
    plan: "08"
    provides: Typed territory freshness evidence and its four public outcome labels
provides:
  - One five-stage guided init contract shared by canonical YAML, Claude, OpenCode, and Codex orchestration
  - Executable parity ratchets for accepted-charter persistence, territory truth, active-colony refusal, and host-native closeout vocabulary
  - Codex creation guidance that delegates all durable mutation and result truth to raw aether init
affects: [phase-200, lifecycle-front-door, command-guide, platform-sync]

tech-stack:
  added: []
  patterns: [runtime-owned lifecycle truth, host-native closeout vocabulary, executable wrapper parity]

key-files:
  created:
    - cmd/front_door_init_wrapper_199_test.go
  modified:
    - .aether/commands/init.yaml
    - .claude/commands/ant-init.md
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md
    - .aether/skills/colony/aether-colony-creation/SKILL.md
    - cmd/command_guide.go
    - cmd/command_guide_test.go
    - cmd/init_proposals_test.go

key-decisions:
  - "Claude and OpenCode init close with exact /ant-plan, while Codex closes with exact aether plan."
  - "Guided surfaces may clarify and synthesize intent, but Go alone owns setup, registry updates, accepted-charter and colony-state persistence, territory evidence, and result truth."
  - "Codex parity extends the existing raw aether orchestration without restoring a native $ant-* lifecycle surface."

patterns-established:
  - "Five-stage parity: Queen opening, setup, accepted intent, territory, and closeout appear in the same order on every host."
  - "Host boundary: shared lifecycle semantics stay identical while the final next command uses each host's native vocabulary."

requirements-completed: [CEC-01, CEC-02, LIFE-01]

duration: 12min
completed: 2026-09-04
---

# Phase 199 Plan 20: Guided Init Surface Parity Summary

**Canonical YAML, Claude, OpenCode, and Codex now share one five-stage guided init contract while raw Go runtime authority and host-native plan closeouts remain explicit.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-03T23:57:11Z
- **Completed:** 2026-09-04T00:08:48Z
- **Tasks:** 3/3
- **Files modified:** 9 implementation/test files

## Accomplishments

- Added exact cross-platform tests for the five ordered stages, `accepted-charter/v1`, typed territory outcomes, pre-storage active-colony refusal, automatic setup, and runtime-only mutation authority.
- Synchronized canonical init YAML and all three managed Claude/OpenCode wrappers around the approved description and exact `/ant-plan` closeout.
- Aligned the existing Codex creation skill and command guide with the same lifecycle semantics while preserving raw `aether init`, exact `aether plan`, and the no-native-`$ant-*` milestone boundary.

## Task Commits

Each task was committed atomically:

1. **Task 1: Define cross-platform init parity (RED)** — `f0921683` (test)
2. **Task 2: Synchronize canonical and generated init wrappers (GREEN)** — `0e633598` (feat)
3. **Task 3: Align the existing Codex creation skill and command guide** — `565e9b2e` (feat)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/front_door_init_wrapper_199_test.go` — exact canonical/wrapper parity and negative-authority fixtures.
- `.aether/commands/init.yaml` — canonical five-stage guided init contract and host closeout metadata.
- `.claude/commands/ant-init.md` — managed flat Claude init wrapper.
- `.claude/commands/ant/init.md` — managed nested Claude init wrapper.
- `.opencode/commands/ant/init.md` — managed OpenCode init wrapper with its sanctioned approval-call difference.
- `.aether/skills/colony/aether-colony-creation/SKILL.md` — Codex-native five-stage synthesis and delegation contract.
- `cmd/command_guide.go` — runtime command-guide projection for guided Codex init.
- `cmd/command_guide_test.go` — named Plan 199 init guide and skill parity coverage.
- `cmd/init_proposals_test.go` — legacy closeout expectation updated to the approved direct plan handoff.

## Decisions Made

- Kept common lifecycle semantics literal and testable across hosts, but required `/ant-plan` for Claude/OpenCode and `aether plan` for Codex.
- Left every durable read, write, refusal decision, territory classification, and success claim in Go; wrappers and the Codex skill only gather and synthesize owner intent.
- Preserved the existing Codex skill-first orchestration and raw-command bypass rather than adding a deferred native `$ant-*` lifecycle command.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Replaced the stale post-init proposal-choice expectation**

- **Found during:** Task 2 (Synchronize canonical and generated init wrappers)
- **Issue:** `TestInitWrapperAsksNextMove` required the retired wrapper-owned proposal picker, contradicting the approved exact `/ant-plan` closeout.
- **Fix:** Renamed the test to `TestInitWrapperClosesAtPlan`; proposal information may remain context, but the wrapper must end with the direct plan handoff.
- **Files modified:** `cmd/init_proposals_test.go`
- **Verification:** The relevant init wrapper compatibility cluster passes all 22 cases from the Task 2 run and all 23 cases in final verification.
- **Committed in:** `0e633598`

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug)
**Impact on plan:** The fix removes a contradictory legacy assertion without expanding production scope.

## Issues Encountered

- A broader command-guide sweep found the pre-existing Plan 199-19 maintenance YAML absent from the Codex guide catalog. The exact Plan 199-20 init tests do not exercise or cause that gap; it is recorded in `deferred-items.md` for its owning migration.
- The requirements handler could not match this repository's bold-description checkbox format; `CEC-01`, `CEC-02`, and `LIFE-01` were already checked complete, so no direct requirements-file edit was needed.

## Verification

- RED gate: `go test ./cmd -run '^(TestFrontDoorInitWrapperParity|TestCommandGuideInit199)$' -count=1` failed on the missing contract before implementation, as required.
- Task 2 compatibility cluster passed 22 cases after the stale proposal expectation was corrected.
- Final parity command passed 23 cases across the named init parity tests, source hygiene, ceremony/stage compatibility, low-signal behavior, mode choice, and exact closeouts.
- Live `AETHER_OUTPUT_MODE=json go run ./cmd/aether command-guide init --platform codex` smoke assertion passed for raw init delegation and exact `Next Up: aether plan`.
- `git diff --check` passed.
- Stub scan found no TODO, FIXME, placeholder copy, coming-soon behavior, or goal-blocking unwired data in the plan-owned files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Downstream front-door work can rely on one executable init contract across all supported hosts.
- Runtime changes to accepted intent or territory truth will now fail focused parity tests if wrappers or Codex guidance drift.
- The unrelated maintenance command-guide catalog gap remains deferred to its owning migration; no Plan 199-20 blocker remains.

## Self-Check: PASSED

- All nine implementation/test files and this summary exist.
- Task commits `f0921683`, `0e633598`, and `565e9b2e` are present in Git history.
- Both exact named tests exist, final parity and live CLI verification pass, and the summary is whitespace-clean.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
