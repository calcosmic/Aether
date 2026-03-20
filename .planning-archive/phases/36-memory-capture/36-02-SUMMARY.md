---
phase: 36-memory-capture
plan: 02
subsystem: learning-system
tags: [memory, learning, continue, queen]
dependency_graph:
  requires: [33-03, 34-03]
  provides: [integrated-learning-approval]
  affects: [.claude/commands/ant/continue.md]
tech_stack:
  added: []
  patterns: [silent-skip, conditional-approval-flow]
key_files:
  created: []
  modified:
    - .claude/commands/ant/continue.md
decisions:
  - "Silent skip pattern: No output when no proposals exist (MEM-01)"
  - "Single promotion path via Step 2.1.5 (removed redundant Step 2.2)"
  - "learning-approve-proposals is the canonical approval workflow"
metrics:
  duration: "15 minutes"
  completed_date: "2026-02-21"
  tasks: 3
  files_modified: 1
---

# Phase 36 Plan 02: Integrate Learning Capture into /ant:continue

## Summary

Integrated automatic learning capture into `/ant:continue` so colony observations are presented for user approval at phase end, with approved learnings written immediately to QUEEN.md.

**One-liner:** Silent-skip learning approval with checkbox UI when proposals exist.

## What Was Built

### Updated continue.md with:

1. **Silent skip pattern (MEM-01)** - Step 2.1.5 now checks proposal count before showing approval UI. When no proposals exist, the command silently continues without any user notice.

2. **Integrated approval workflow** - When proposals exist, the checkbox approval UI from Phase 34's `learning-approve-proposals` is displayed, allowing users to select which learnings to promote to QUEEN.md.

3. **Removed redundant Step 2.2** - The old Step 2.2 "Promote Validated Learnings" duplicated functionality already handled by `learning-approve-proposals`. Removed to maintain a single promotion path.

4. **Consistent step numbering** - After removing Step 2.2, all subsequent steps were renumbered:
   - Step 2.3 → Step 2.2 (Update Handoff Document)
   - Step 2.4 → Step 2.3 (Update Changelog)
   - Step 2.6 → Step 2.4 (Commit Suggestion)
   - Step 2.7 → Step 2.5 (Context Clear Suggestion)
   - Step 2.8 → Step 2.6 (Update Context Document)
   - Step 2.5 → Step 2.7 (Project Completion)

## Key Implementation Details

### Silent Skip Pattern
```bash
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  verbose_flag=""
  [[ "$ARGUMENTS" == *"--verbose"* ]] && verbose_flag="--verbose"
  bash .aether/aether-utils.sh learning-approve-proposals $verbose_flag
fi
# If no proposals, silently skip without notice (per user decision)
```

This ensures:
- No noise when no learnings have accumulated
- Checkbox UI only appears when there's something to approve
- User approval required before any promotion happens (INT-03)

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

- [x] continue.md has silent skip pattern (check proposal_count before showing UI)
- [x] Old redundant Step 2.2 is removed
- [x] Step numbering is consistent
- [x] learning-approve-proposals is called when proposals exist
- [x] No output when no proposals (silent skip)

## Commits

1. `065c0d5` - feat(36-02): add silent skip pattern for empty proposals in continue.md
2. `30eb37c` - feat(36-02): remove redundant Step 2.2 promotion section
3. `d381f81` - feat(36-02): update step numbering after Step 2.2 removal

## Next Steps

The memory pipeline is now fully integrated:
1. Colony observes during builds (Phase 33)
2. User approves at continue (this plan)
3. QUEEN.md accumulates wisdom

Ready for Phase 36 Plan 03: Memory verification and testing.
