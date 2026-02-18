---
phase: 12-build-progress
plan: "02"
subsystem: colony-commands
tags: [build, completion-lines, tool-count, tmux-gating, wave-failure, build-summary, progress-indicators]

# Dependency graph
requires: ["12-01"]
provides:
  - "Single-line completion format for builders, watcher, and chaos with tool_count"
  - "Failed worker completion line format with failure_reason and tool_count"
  - "All-wave-failed halt with prominent WAVE FAILURE alert"
  - "tmux-only gating on all swarm-display-text calls"
  - "BUILD SUMMARY block replacing compact/verbose output split"
  - "Retry guidance using /ant:swarm and /ant:flags in BUILD SUMMARY"
affects: [12-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Completion line format: '{emoji} {Ant-Name}: {task} ({tool_count} tools) ✓'"
    - "Failed worker format: '{emoji} {Ant-Name}: {task} ✗ ({failure_reason} after {tool_count} tools)'"
    - "Wave failure halt: check if ALL workers in wave returned status failed, display WAVE FAILURE alert, stop"
    - "tmux gate pattern: 'If $TMUX is set, run swarm-display-text; otherwise skip entirely'"
    - "BUILD SUMMARY block: always shown, shows pass/fail counts, total_tools, elapsed time, failed worker details"

key-files:
  created: []
  modified:
    - ".claude/commands/ant/build.md"
    - ".opencode/commands/ant/build.md"

key-decisions:
  - "BUILD SUMMARY always shown (not split by verbose_mode) — verbose mode appends detail sections after it"
  - "total_tools = sum of tool_count from all worker return JSONs (builders + watcher + chaos)"
  - "elapsed calculated from build_started_at_epoch captured in Step 5 (per Plan 01 decision)"
  - "tmux gate applied to both swarm-display-text calls (Step 5.2 and Step 7) — chat never sees it"
  - "Wave failure halts to Step 5.9 synthesis with status: failed — watcher and chaos do not run"
  - "Retry guidance uses /ant:swarm and /ant:flags — no fictional --retry-failed flag"

patterns-established:
  - "Completion lines: always single-line immediately on result arrival — builder, watcher, chaos all same pattern"
  - "tmux gating: check $TMUX before every swarm-display-text call, skip entirely if not set"
  - "Wave failure: all workers must fail to halt — partial failure continues to verification"

requirements-completed: [PROG-02, PROG-03]

# Metrics
duration: 4min
completed: 2026-02-18
---

# Phase 12 Plan 02: Build Progress — Completion Lines, tmux Gating, Wave Failure Halt, BUILD SUMMARY

**Worker completion lines with tool_count appear immediately, swarm display gated to tmux-only, all-wave-failed halt added, and unified BUILD SUMMARY block replaces old compact/verbose split in both build.md files**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-02-18T11:26:48Z
- **Completed:** 2026-02-18T11:31:20Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Completion lines now show immediately as each worker finishes in format `🔨 Hammer-42: Implement login (12 tools) ✓` — no more old `✅ 🔨🐜` format
- Watcher and Chaos completion lines use same pattern with 👁️ and 🎲 emojis respectively
- Failed worker format shows `✗ (failure_reason after N tools)` — failure reason from first item in blockers array
- swarm-display-text now gated behind `$TMUX` check in both Step 5.2 and Step 7 — chat users never see swarm display calls fire
- All-wave-failed halt added to Step 5.2: if every worker in a wave fails, build stops with prominent WAVE FAILURE alert and skips to Step 5.9
- BUILD SUMMARY block replaces compact/verbose output split — always shown, with pass/fail counts, total tools across all workers, duration, and retry guidance
- verbose_mode now appends detail sections (Colony Work Tree, TDD, etc.) after BUILD SUMMARY rather than replacing it
- All changes mirrored to OpenCode build.md with OpenCode plain-bash syntax preserved

## Task Commits

Each task was committed atomically:

1. **Task 1: Claude Code build.md** — `f047405` (feat)
2. **Task 2: OpenCode build.md mirror** — `687ebc6` (feat)

## Files Created/Modified

- `.claude/commands/ant/build.md` — 7 edits: completion line format (builder/watcher/chaos), tmux gating on 2 swarm-display-text calls, all-wave-failed halt, BUILD SUMMARY block replacing compact/verbose split
- `.opencode/commands/ant/build.md` — Same 7 edits with OpenCode plain-bash syntax conventions preserved

## Decisions Made

- BUILD SUMMARY always shown (not split by verbose_mode) — verbose mode appends detail sections after the summary block
- total_tools calculated by summing tool_count from all worker return JSONs (builders + watcher + chaos)
- elapsed uses build_started_at_epoch from Step 5 (Plan 01 decision) — measures actual worker execution time
- tmux gate applied to both swarm-display-text calls — chat users see structured completion lines instead
- Wave failure halts build when ALL workers in a wave fail — partial failure (some workers succeed) continues normally
- Retry guidance uses /ant:swarm and /ant:flags — no fictional CLI flags invented

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

- lint:sync shows pre-existing 34 vs 33 command count divergence — present before Task 1 began, not introduced by Phase 12 changes

## Self-Check

Files exist:
- `.claude/commands/ant/build.md` — FOUND
- `.opencode/commands/ant/build.md` — FOUND

Commits exist:
- `f047405` — FOUND (feat(12-02): completion lines...)
- `687ebc6` — FOUND (feat(12-02): mirror completion lines...)

## Self-Check: PASSED

## Next Phase Readiness

- BUILD SUMMARY block is live — Plan 03 can reference this as the "after" output pattern
- tool_count from Plan 01 is now consumed visibly in completion lines and BUILD SUMMARY total_tools
- tmux gating ensures swarm display never fires in chat context
- Wave failure halt prevents silent partial-success scenarios

---
*Phase: 12-build-progress*
*Completed: 2026-02-18*
