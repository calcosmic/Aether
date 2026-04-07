---
phase: 04-planning-granularity-controls
plan: 02
status: complete
completed: 2026-04-07
---

# Plan 04-02: Plan command and route-setter wiring

## What Was Built

- Added `--granularity` flag parsing to both plan command files
- Added granularity selection step (read persisted value or ask user) before planning depth selection
- Replaced all hardcoded "3-6" references with dynamic `{granularity_min}-{granularity_max}` placeholders
- Replaced "Maximum 6 phases" with "Maximum {granularity_max} phases"
- Updated all 4 route-setter agent definition files (agents, agents-claude, .claude/agents, .opencode/agents)
- Zero remaining hardcoded "3-6" or "Maximum 6" references in any file

## Key Files

- `.claude/commands/ant/plan.md` — Plan command with granularity selection and dynamic bounds
- `.aether/commands/claude/plan.md` — Mirror of plan command
- `.aether/agents/aether-route-setter.md` — Route-setter with dynamic bounds
- `.aether/agents-claude/aether-route-setter.md` — Claude mirror
- `.claude/agents/ant/aether-route-setter.md` — Claude Code mirror
- `.opencode/agents/aether-route-setter.md` — OpenCode mirror

## Self-Check

- [x] All acceptance criteria met
- [x] `grep -rn "3-6"` returns 0 in route-setter files
- [x] `grep -rn "Maximum 6"` returns 0 in plan files
- [x] All 4 agent files have {granularity_min} and {granularity_max} placeholders
- [x] Both plan files have --granularity flag and granularity selection step
