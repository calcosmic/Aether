---
phase: 83-planning-depth-system
plan: 02
status: complete
commits:
  - bea2771f: feat(83-02): add planning depth ceremony to YAML source and wrapper files
---

# Summary: Plan 83-02 — YAML Source + Wrapper Updates

## What Was Built

Updated all three command source files to expose the `--planning-depth` flag with light/standard/deep task decomposition levels:

1. **`.aether/commands/plan.yaml`** — Added `--planning-depth <light|standard|deep>` to runtime command, added `planning_depth_ceremony` key, updated orchestration step 2 and post-plan step 1.

2. **`.claude/commands/ant/plan.md`** — Added `## Planning Depth` section after Depth Ceremony, updated manifest invocation with `--planning-depth <choice2>`, updated wave execution injection list, updated after-planning summary.

3. **`.opencode/commands/ant/plan.md`** — Identical changes to Claude wrapper (structural parity confirmed via diff).

## Key Decisions

- Planning depth is clearly distinguished from planning granularity: depth controls phase count, planning depth controls task detail within each plan.
- Default is "standard" — no ceremony required if user doesn't specify.
- The ceremony text explicitly states "independent of phase count" to prevent confusion.

## Self-Check: PASSED

- All three files contain "planning-depth" string
- Both wrappers have `## Planning Depth` section
- Structural parity between Claude and OpenCode wrappers confirmed
- YAML source has `planning_depth_ceremony` key
