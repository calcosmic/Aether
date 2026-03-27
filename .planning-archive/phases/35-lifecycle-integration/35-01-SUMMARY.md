---
phase: 35-lifecycle-integration
plan: 01
type: execute
subsystem: wisdom-system
phase_name: Lifecycle Integration
plan_name: Seal.md Wisdom Approval Integration

requires:
  - INT-04

provides:
  - seal.md with integrated wisdom approval workflow
  - Blocking wisdom review at colony lifecycle boundary

affects:
  - .claude/commands/ant/seal.md

tech-stack:
  added: []
  patterns:
    - Blocking approval flow at lifecycle boundaries
    - Decimal step numbering for non-breaking insertion

key-files:
  created: []
  modified:
    - path: .claude/commands/ant/seal.md
      change: "Added Step 3.5 wisdom approval, simplified Step 4 to activity logging only"

decisions: []

metrics:
  duration: 83 seconds
  completed_at: 2026-02-21T14:36:07Z
  tasks_completed: 3
  files_modified: 1
  commits: 1
---

# Phase 35 Plan 01: Seal.md Wisdom Approval Integration

## Summary

Integrated the wisdom approval workflow into seal.md so users must review and approve wisdom proposals before the colony is sealed. Wisdom is permanent — this ensures users validate what gets promoted to QUEEN.md at the final lifecycle boundary.

## What Changed

### seal.md Structure Update

**Before:**
- Step 3: Confirmation
- Step 4: Promote Colony Wisdom (auto-promotion without approval)
- Step 5: Update Milestone
- Step 6: Write CROWNED-ANTHILL.md

**After:**
- Step 3: Confirmation
- **Step 3.5: Wisdom Approval** (NEW — blocking approval workflow)
- Step 4: Log Seal Activity (simplified — old auto-promotion removed)
- Step 5: Update Milestone
- Step 6: Write CROWNED-ANTHILL.md

### Key Implementation Details

1. **Step 3.5: Wisdom Approval**
   - Checks for pending proposals using `learning-check-promotion`
   - Displays "🧠 WISDOM REVIEW" header when proposals exist
   - Calls `learning-approve-proposals` for blocking approval workflow
   - Shows "No wisdom proposals to review" when empty
   - Blocks progression until user approves or defers

2. **Step 4 Simplification**
   - Removed 70+ lines of auto-promotion logic
   - Now only logs seal ceremony to activity log
   - Avoids duplicate promotion (already handled by Step 3.5)

3. **Step Numbering**
   - Used decimal step number (3.5) to avoid renumbering existing steps
   - All other steps retain original numbers
   - No internal step references needed updating

## Commits

| Hash | Message |
|------|---------|
| 8f595b9 | feat(35-01): add Step 3.5 wisdom approval to seal.md |

## Verification

- [x] Step 3.5 exists with proper wisdom approval flow
- [x] `learning-approve-proposals` is called correctly
- [x] Empty state shows "No wisdom proposals to review"
- [x] Existing auto-promotion simplified to avoid duplication
- [x] Step numbering is consistent throughout
- [x] Step 3.5 positioned between Step 3 (confirmation) and Step 4 (logging)

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- [x] seal.md contains Step 3.5: Wisdom Approval
- [x] seal.md contains learning-approve-proposals call
- [x] seal.md shows "No wisdom proposals to review" for empty state
- [x] Step 4 is simplified to activity logging only
- [x] Commit 8f595b9 exists in git history

## Next Steps

Plan 35-02 will integrate the same wisdom approval workflow into entomb.md for the archival lifecycle boundary (INT-05).
