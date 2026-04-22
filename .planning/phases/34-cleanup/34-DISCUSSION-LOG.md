# Phase 34: Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-23
**Phase:** 34-cleanup
**Areas discussed:** Preservation strategy, Integration vs. preserve-only, Cleanup safety model, Blocker flags

---

## Preservation Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Create `preserve/` branches automatically | Auto-create branches before deleting | |
| Preserve manually, then clean | Manual preservation, then cleanup command only deletes disposable | |
| Review first, then decide | Review commits for value/merge-readiness/safety; integrate if good, preserve if not | ✓ |

**User's choice:** Review first, then decide. The user explicitly said: "It's about also checking if it's valuable and it's ready to merge and if we can do that safely. If it is good work. ... It's something for you to look at and check."
**Notes:** This shifts from blind preservation to an assessment-based approach. The user wants me (Claude) to evaluate the 2 candidate commits before any cleanup happens.

---

## Cleanup Safety Model

| Option | Description | Selected |
|--------|-------------|----------|
| Dry-run by default | Show what would be deleted, require `--force` to actually delete | |
| Delete immediately with summary | Clean everything in one go, report after | |
| Interactive confirmation | Show the list, pause for user 'proceed?' before deleting | ✓ |

**User's choice:** Interactive confirmation — show the list and ask 'proceed?' before deleting.
**Notes:** The user wants to see exactly what will be removed before anything happens. No `--force` bypass.

---

## Blocker Flags

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-archive if older than 14 days | Stale blockers auto-archived by age | |
| Manual review | Show all 13, user decides each one | ✓ |
| Archive all unresolved | Bulk archive all 13 | |

**User's choice:** Manual review — show all 13 blockers and let the user decide each one.
**Notes:** Consistent with the interactive approach. No auto-archive by age.

---

## Claude's Discretion

- Specific porting strategy for each commit (which files to cherry-pick, which to skip)
- Exact order of cleanup operations
- Output formatting for interactive review screens

## Deferred Ideas

- Automated recurring cleanup (future enhancement)
- Cross-repo worktree cleanup (out of scope)
- Visual cleanup dashboard (nice-to-have)
