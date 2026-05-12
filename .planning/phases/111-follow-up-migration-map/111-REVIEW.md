---
phase: 111-follow-up-migration-map
reviewed: 2026-05-12T12:00:00Z
depth: quick
files_reviewed: 1
files_reviewed_list:
  - .aether/docs/migration-map.md
findings:
  critical: 1
  warning: 2
  info: 0
  total: 3
status: issues_found
---

# Phase 111: Code Review Report

**Reviewed:** 2026-05-12T12:00:00Z
**Depth:** quick
**Files Reviewed:** 1
**Status:** issues_found

## Summary

Reviewed `.aether/docs/migration-map.md` -- a documentation-only migration planning artifact for three deferred capabilities (Oracle/RALF, Swarm Visibility, Build/Continue Parity). The document is well-structured and thorough, but contains one factual error about existing Go commands and two inconsistencies with the phase plan and research that could mislead implementers.

## Critical Issues

### CR-01: Migration map incorrectly claims seal lacks --plan-only support

**File:** `.aether/docs/migration-map.md:160`
**Issue:** Phase C-2 states "Seal needs a new `seal --plan-only` (returns blocker list, ceremony steps) and `seal-finalize` command." This is factually wrong. Both `seal --plan-only` and `seal-finalize` already exist in the Go codebase:

- `seal --plan-only` is implemented at `cmd/codex_workflow_cmds.go:320-329` calling `runSealPlanOnly()`
- `seal-finalize` is implemented at `cmd/seal_final_review.go:136`

The phase plan (Task 2, line 179 of `111-01-PLAN.md`) correctly states "Seal already has `seal --plan-only` and `seal-finalize` commands," but the migration map contradicts its own plan. This will cause implementers to waste time building commands that already exist, or worse, build duplicate/conflicting commands.

Notably, the research document (`111-RESEARCH.md:192`) also contains this error ("No existing --plan-only support"), so the mistake originated upstream but should have been caught during map authoring by verifying against the actual Go code.

**Fix:**
Change line 160 from:
```
Seal needs a new `seal --plan-only` (returns blocker list, ceremony steps) and `seal-finalize` command.
```
To:
```
Seal already has `seal --plan-only` (returns blocker list via `runSealPlanOnly()`) and `seal-finalize` command. TS host calls plan-only to check for blockers, then calls seal-finalize if unblocked.
```

Also update the C-2 Risk Assessment table entry (line 187) which says "Seal lacks existing `--plan-only` support (unlike colonize)" -- this should be removed or rewritten since seal does have `--plan-only`.

## Warnings

### WR-01: Phase C-2 scope description contradicts phase plan Task 2

**File:** `.aether/docs/migration-map.md:160`
**Issue:** The migration map says Phase C-2 scope is "Add `runSealLifecycle()` to TS host. Seal needs a new `seal --plan-only`..." which implies Go command creation work. But the phase plan (Task 2) says "Seal already has `seal --plan-only` and `seal-finalize` commands. TS host calls plan-only, checks for blockers, dispatches review workers if needed, calls finalizer." This discrepancy means the scope estimate ("Medium") may be understated if implementers follow the map (thinking they need to build Go commands) or correct but misleading (if implementers follow the plan).

This is closely related to CR-01 but highlights the plan/map inconsistency separately because even after fixing the factual error, the scope description should be updated to reflect that C-2 is TS-host-only work (no new Go commands needed), which may lower the scope estimate.

**Fix:** After fixing CR-01, revise the C-2 scope to reflect TS-host-only work:
```
Medium -- Add `runSealLifecycle()` to TS host. Seal already has `seal --plan-only` (returns blocker list, ceremony steps) and `seal-finalize` command. TS host calls plan-only, checks for blockers, dispatches review workers if needed, calls finalizer. No new Go commands required.
```

### WR-02: Migration map references `oracle-iterate` commands that do not exist yet without clearly flagging them as new

**File:** `.aether/docs/migration-map.md:45-46`
**Issue:** Milestone A Phase A-1 proposes adding `oracle-iterate --plan-only` and `oracle-iterate-finalize --completion-file`. These are correctly identified as new Go commands to be built. However, the requirements (ORA-01, ORA-02) and later sections reference these commands as if they already exist, using present tense ("Go command `oracle-iterate --plan-only` returns JSON manifest"). While the phase table makes it clear A-1 creates these commands, the requirements section could mislead someone reading requirements in isolation.

Additionally, the research document (`111-RESEARCH.md:217`) explicitly warns about this as "Pitfall 1" -- over-scoping oracle migration by moving loop logic to TS. The mitigation is correct (Go provides the iteration commands, TS orchestrates), but the requirements should be clearer that these are new commands.

**Fix:** In the Requirements table, ORA-01 and ORA-02 should include explicit phrasing that these are new commands to be implemented in Phase A-1, not existing commands. For example, ORA-01 could read: "New Go command `oracle-iterate --plan-only` returns JSON manifest..." (adding "New" clarifies this is a deliverable, not an existing capability).

---

_Reviewed: 2026-05-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
