---
phase: 35
plan: 02
subsystem: lifecycle-integration
name: "Entomb Wisdom Integration"
type: execute
wave: 1
requires: []
provides: [INT-05]
affects: [.claude/commands/ant/entomb.md]
tech-stack:
  added: []
  patterns: [wisdom-approval-workflow, blocking-lifecycle-boundary]
key-files:
  created: []
  modified:
    - .claude/commands/ant/entomb.md
decisions:
  - "Use Step 3.25 (decimal) to fit between Step 3 and Step 3.5 without renumbering"
  - "Simplify Step 4 to QUEEN.md initialization only (remove auto-promotion)"
  - "learning-approve-proposals handles all promotion logic in Step 3.25"
metrics:
  duration: "10 minutes"
  completed: "2026-02-21"
  tasks: 3
  files-modified: 1
  commits: 2
---

# Phase 35 Plan 02: Entomb Wisdom Integration Summary

## One-Liner
Integrated wisdom approval workflow into entomb.md so users must review and approve wisdom proposals before the colony is archived.

## What Was Built

### Step 3.25: Wisdom Approval (NEW)
Added a new blocking step between user confirmation (Step 3) and XML tools check (Step 3.5):

- Checks for pending proposals using `learning-check-promotion`
- Displays "FINAL WISDOM REVIEW" header when proposals exist
- Calls `learning-approve-proposals` for the full approval workflow
- Shows "No wisdom proposals to review" and continues when empty
- Blocks progression until approval workflow completes

### Step 4: Simplified
The existing Step 4 auto-promotion logic was redundant since Step 3.25 now handles approval. Simplified to:

- QUEEN.md initialization check only
- Removed manual extraction and promotion loops (Step 4.2, 4.3)
- Removed promotion summary display

## Commits

| Commit | Message | Description |
|--------|---------|-------------|
| effd88f | feat(35-02): add Step 3.25 Wisdom Approval to entomb.md | Added wisdom review step with learning-check-promotion and learning-approve-proposals integration |
| 64d94fa | feat(35-02): simplify Step 4 in entomb.md | Removed redundant auto-promotion logic, kept only QUEEN.md initialization |

## Files Modified

### .claude/commands/ant/entomb.md
- **Lines added:** 43 (Step 3.25)
- **Lines removed:** 69 (simplified Step 4)
- **Net change:** -26 lines (cleaner, more focused)

## Step Flow (Updated)

```
Step 0: Initialize Visual Mode
Step 1: Read State
Step 2: Seal-First Enforcement
Step 3: User Confirmation
Step 3.25: Wisdom Approval (NEW - blocking)
Step 3.5: Check XML Tools
Step 4: Ensure QUEEN.md Exists (simplified)
Step 5: Generate Chamber Name
Step 6: Create Chamber
Step 7: Archive Additional Files
Step 7.5: Export XML Archive
Step 8: Verify Chamber Integrity
Step 9: Record in Eternal Memory
Step 10: Reset Colony State
Step 11: Write HANDOFF.md
Step 12: Display Result
```

## Deviations from Plan

None - plan executed exactly as written.

## Requirements Satisfied

| Requirement | Status | Notes |
|-------------|--------|-------|
| INT-05 | Complete | entomb.md now promotes wisdom before archiving with user approval |

## Verification Results

- [x] Step 3.25 exists with proper wisdom approval flow
- [x] learning-approve-proposals is called correctly
- [x] Empty state shows "No wisdom proposals to review"
- [x] Existing auto-promotion simplified
- [x] All step references consistent

## Self-Check: PASSED

- [x] Modified file exists: .claude/commands/ant/entomb.md
- [x] Commit effd88f exists in git log
- [x] Commit 64d94fa exists in git log
- [x] Step 3.25 present in file
- [x] Step 4 simplified

## Notes

This integration follows the same pattern established in Phase 34 for continue.md, ensuring consistency across all lifecycle boundaries. The wisdom approval workflow is now active at:

1. **Phase end** (continue.md) - review proposals from completed phase
2. **Seal ceremony** (seal.md - to be implemented in 35-01) - final review before milestone
3. **Archive** (entomb.md - this plan) - final wisdom review before eternal storage

All three boundaries use the same `learning-approve-proposals` function for consistent UX.
