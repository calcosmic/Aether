---
phase: 43-make-learning-flow
plan: 02
type: execute
wave: 2
subsystem: learning-pipeline
tags: [learning, promotion, queen, observations]
dependency_graph:
  requires: [43-01]
  provides: [FLOW-02]
  affects: [.claude/commands/ant/build.md, .claude/commands/ant/continue.md, .aether/aether-utils.sh]
tech_stack:
  added: []
  patterns: [one-at-a-time-ui, threshold-alignment, atomic-write]
key_files:
  created: []
  modified:
    - .claude/commands/ant/build.md
    - .claude/commands/ant/continue.md
    - .aether/aether-utils.sh
decisions:
  - One-at-a-time proposal UI per user decision
  - Aligned thresholds (1 for most types, 0 for decree)
  - Failure type maps to Patterns section in QUEEN.md
  - Retry prompt on QUEEN.md write failure
metrics:
  duration_minutes: 45
  completed_date: 2026-02-22
  tasks_completed: 5
  files_modified: 3
  commits: 5
---

# Phase 43 Plan 02: Make Learning Flow - FLOW-02

> Wire the learning pipeline so observations flow to QUEEN.md promotions automatically

## One-Liner Summary

Implemented end-of-build promotion checking with one-at-a-time proposal UI, aligned thresholds across all learning functions, and verified integration with continue.md.

## What Was Built

### Task 1: End-of-Build Promotion Check (build.md)
- Added Step 5.10: Check for Promotion Proposals
- Runs after build completion (success or failure)
- Calls `learning-check-promotion` to find threshold-meeting observations
- Displays proposal count and invokes `learning-approve-proposals` when proposals exist
- Silent skip when no proposals (per user decision)

### Task 2: One-at-a-Time Proposal UI (aether-utils.sh)
- Completely rewrote `learning-approve-proposals` function
- Replaced batch display with numeric selection
- New flow: Shows proposal X of Y with full content
- Three actions: [A]pprove, [R]eject, [S]kip
- Auto-advances to next proposal after user action
- Summary at end: "X approved, Y rejected, Z skipped"
- Handles retry on QUEEN.md write failure per user decision
- Maintains support for --dry-run, --yes, --deferred, --undo flags

### Task 3: Threshold Alignment (aether-utils.sh)
- Updated `learning-display-proposals`: thresholds now 1 for most types
- Updated `learning-select-proposals`: thresholds now 1 for most types
- Updated `learning-check-promotion`: added explicit failure type
- Added failure type to type arrays in learning-display-proposals
- All functions now use consistent thresholds:
  - philosophy: 1, pattern: 1, redirect: 1, stack: 1, failure: 1
  - decree: 0 (immediate promotion)

### Task 4: Continue.md Verification
- Verified continue.md Step 2.1.5 has correct promotion check pattern
- Uses `learning-check-promotion` to find proposals
- Calls `learning-approve-proposals` when proposals exist
- Silent skip when no proposals (per user decision)
- Pattern matches build.md Step 5.10 implementation

### Task 5: Queen-Promote Verification (aether-utils.sh)
- Added failure type to valid_types array
- Failure observations map to Patterns section when promoted
- Entry format verified: `- **colony_name** (timestamp): content`
- Writes to QUEEN.md with atomic temp file + rename pattern
- Updates METADATA stats and evolution_log on promotion

## Deviations from Plan

None - plan executed exactly as written.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | b424fb8 | feat(43-02): add end-of-build promotion check to build.md |
| 2 | 7e80248 | feat(43-02): implement one-at-a-time proposal UI |
| 3 | 71c5b02 | fix(43-02): align thresholds across all learning functions |
| 4 | 53fdc3b | docs(43-02): verify continue.md promotion check alignment |
| 5 | 19b0779 | feat(43-02): add failure type support to queen-promote |

## Verification Results

All verification checks passed:

1. ✅ build.md has Step 5.10 with promotion check
2. ✅ build.md calls learning-check-promotion
3. ✅ One-at-a-time UI implemented with [A]pprove/[R]eject/[S]kip
4. ✅ Thresholds aligned (all use 1 for most types)
5. ✅ failure type in all threshold checks
6. ✅ continue.md pattern verified
7. ✅ queen-promote format correct

## Success Criteria

- [x] build.md has end-of-build promotion checking (Step 5.10)
- [x] learning-approve-proposals shows one proposal at a time with [A]pprove/[R]eject/[S]kip actions
- [x] After user acts on a proposal, next proposal auto-shows
- [x] Thresholds are aligned across learning-observe, learning-check-promotion, learning-display-proposals, and learning-select-proposals
- [x] failure type is handled in all threshold checks
- [x] continue.md pattern verified and consistent
- [x] queen-promote writes correct format to QUEEN.md

## Integration Flow

The learning pipeline now works end-to-end:

1. **Observation Recording**: `learning-observe` records observations with content hash deduplication
2. **Threshold Checking**: `learning-check-promotion` finds observations meeting thresholds
3. **User Review**: `learning-approve-proposals` presents one-at-a-time UI
4. **Promotion**: `queen-promote` writes approved wisdom to QUEEN.md
5. **Context Loading**: `colony-prime` loads QUEEN.md wisdom for worker context

## Files Modified

- `.claude/commands/ant/build.md` - Added Step 5.10 promotion check
- `.aether/aether-utils.sh` - Updated learning-approve-proposals, aligned thresholds, added failure type support

## Notes

- The one-at-a-time UI follows the user's explicit decision for better UX
- Threshold alignment ensures consistent behavior across all learning functions
- Failure observations are now first-class citizens in the learning pipeline
- Retry logic on QUEEN.md write failures prevents lost promotions
