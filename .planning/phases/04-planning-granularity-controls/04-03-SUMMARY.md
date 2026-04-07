---
phase: 04-planning-granularity-controls
plan: 03
status: complete
completed: 2026-04-07
---

# Plan 04-03: Autopilot granularity awareness

## What Was Built

- Autopilot (Step 0) reads persisted `plan_granularity` from COLONY_STATE.json
- If granularity is set, compares total phase count against the range
- Warns (non-blocking) if plan exceeds range — does not halt execution
- Displays granularity label in AUTOPILOT ENGAGED startup banner
- Both Claude and OpenCode run.md files have identical granularity check text (parity)

## Key Files

- `.aether/commands/claude/run.md` — Claude autopilot with granularity check in Step 0
- `.opencode/commands/ant/run.md` — OpenCode autopilot with identical check

## Self-Check

- [x] All acceptance criteria met
- [x] Both files contain `plan-granularity get` references
- [x] Both files contain `Granularity:` in the AUTOPILOT ENGAGED display
- [x] Both files have identical granularity check text (parity verified: 8 refs + 1 display each)
- [x] No blocking behavior for out-of-range plans (warn only)
