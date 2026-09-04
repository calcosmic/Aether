---
phase: 199-front-door-and-classic-contract
plan: "30"
subsystem: lifecycle-vocabulary
tags: [pause, resume, handoff, recovery, wrappers, documentation]
requires:
  - phase: 199-13
    provides: canonical transactional pause and resume runtime commands
  - phase: 199-14
    provides: hidden legacy parser compatibility and pruned public wrappers
provides:
  - D-13 canonical pause and resume vocabulary across organize, handoff, and recovery guidance
  - synchronized raw CLI recovery snapshot and tracked context wording
affects: [Claude wrappers, OpenCode wrappers, handoff templates, recovery snapshots]
tech-stack:
  added: []
  patterns: [canonical-source-first vocabulary updates, hidden compatibility with public-name hygiene]
key-files:
  created: []
  modified: [.aether/commands/organize.yaml, .aether/skills/colony/context-management/SKILL.md, cmd/recovery_snapshot.go, .aether/CONTEXT.md, cmd/session_flow_cmds_test.go]
key-decisions:
  - "Current product guidance exposes only /ant-pause and /ant-resume on wrapper platforms, and aether pause and aether resume on the raw CLI."
  - "The existing pause persistence test exercises the public pause command while retaining durable handoff, session, state, and resume assertions."
patterns-established:
  - "Vocabulary guardrails must avoid spelling forbidden tokens when a literal retired-name grep is the acceptance gate."
requirements-completed: [SYNTH-01, LIFE-03, LIFE-04, PROOF-01]
duration: 5min
completed: 2026-09-04
---

# Phase 199 Plan 30: D-13 Vocabulary Migration Summary

**Current organize, handoff, and recovery guidance now exposes one canonical pause/resume door per supported host, with tracked recovery output matching the raw CLI renderer.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-09-04T22:53:33Z
- **Completed:** 2026-09-04T22:58:34Z
- **Tasks:** 3/3
- **Files modified:** 10

## Accomplishments

- Added canonical lifecycle vocabulary protection to the organize source and synchronized all managed organize closeouts to `/ant-resume`.
- Replaced retired session instructions in context-management and handoff/OpenCode templates with canonical pause and resume commands.
- Aligned generated and tracked recovery guidance on `aether resume`, preserving the full-handoff explanation without a second public command.

## Task Commits

1. **Task 1: Synchronize managed ant-organize vocabulary** — `d987b335` (docs)
2. **Task 2: Correct context-management and handoff templates** — `eb791db5` (docs)
3. **Task 3: Align recovery snapshot renderer and tracked output** — `277c9682` (fix)
4. **Acceptance repair: Keep organize vocabulary guardrail clean** — `62c53cdd` (fix)

## Files Created/Modified

- `.aether/commands/organize.yaml` — canonical public-vocabulary guardrail.
- `.claude/commands/ant-organize.md`, `.claude/commands/ant/organize.md`, `.opencode/commands/ant/organize.md` — synchronized managed organize closeouts.
- `.aether/skills/colony/context-management/SKILL.md` and handoff/OpenCode templates — one-door current handoff instructions.
- `cmd/recovery_snapshot.go` and `.aether/CONTEXT.md` — matching full-recovery guidance using `aether resume`.
- `cmd/session_flow_cmds_test.go` — persistence fixture migrated to the public `pause` command without weakening durable-state assertions.

## Decisions Made

- Wrapper hosts teach exact `/ant-pause` and `/ant-resume`; raw CLI guidance teaches exact `aether pause` and `aether resume`.
- The richer recovery view remains an outcome of `aether resume`, not a second command name.

## Verification

- Nine-file scoped retired-vocabulary grep — passed.
- `go test ./cmd -run '^(TestCommandSourceHygiene|TestPauseColonyWritesHandoffAndSession|TestPauseResume199HandoffContents)$' -count=1` — 9 passed.
- `go build ./cmd/aether` — passed.
- `go test -race ./cmd -run '^TestPauseColonyWritesHandoffAndSession$' -count=1` — 1 passed.
- Stub scan found only pre-existing `TODO/FIXME/HACK` text in organize's report-analysis instructions; no product stub was introduced.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Plan-acceptance test] Migrated the named pause persistence fixture to the public command.**
- **Found during:** Task 3
- **Issue:** `TestPauseColonyWritesHandoffAndSession` invoked the removed `pause-colony` input and failed before exercising persistence.
- **Fix:** Invoked `pause`, asserted its session command value, and updated the existing handoff-header assertion to the current canonical output while retaining all persistence assertions.
- **Files modified:** `cmd/session_flow_cmds_test.go`
- **Verification:** Exact named test and focused race test passed.
- **Committed in:** `277c9682`

**2. [Rule 1 - Acceptance hygiene] Removed forbidden token spellings from the canonical guardrail prose.**
- **Found during:** Final nine-file scoped grep
- **Issue:** The guardrail correctly described legacy routes but caused the plan's literal retired-vocabulary grep to match its own text.
- **Fix:** Kept explicit canonical names and referred to retired commands as legacy lifecycle routes.
- **Files modified:** `.aether/commands/organize.yaml`
- **Verification:** Nine-file scoped retired-vocabulary grep passed.
- **Committed in:** `62c53cdd`

**Total deviations:** 2 auto-fixed (2 Rule 1).
**Impact on plan:** Both changes were narrow correctness fixes that preserve the D-13 public contract and its acceptance proof.

## Issues Encountered

- The original named persistence fixture had drifted from the public command and current handoff title; it now tests the implemented public behavior.

## Known Stubs

None.

## Self-Check: PASSED

- All nine planned product paths and the permitted persistence-test fixture exist.
- All four task and acceptance commits exist in git history.
