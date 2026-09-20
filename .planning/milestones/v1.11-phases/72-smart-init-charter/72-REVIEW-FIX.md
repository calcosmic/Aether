---
phase: 72-smart-init-charter
fixed_at: 2026-04-28T12:30:00Z
review_path: .planning/phases/72-smart-init-charter/72-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 4
skipped: 1
status: partial
---

# Phase 72: Code Review Fix Report

**Fixed at:** 2026-04-28T12:30:00Z
**Source review:** .planning/phases/72-smart-init-charter/72-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5
- Fixed: 4
- Skipped: 1

## Fixed Issues

### CR-01: `init-ceremony` bypasses idempotency and sealed colony detection

**Files modified:** `cmd/init_ceremony.go`
**Commit:** 7ed42985
**Applied fix:** Added the same idempotency check logic from `init_cmd.go` into `createCeremonyColony()`: checks for existing colony state, blocks if an active (non-sealed) colony exists, detects sealed colonies with in-progress seal operations, and creates a backup of existing state before overwriting. Uses `copyFile` for backup and `sealInProgress` for seal detection, matching the `aether init` behavior exactly.

### WR-01: `--charter-json` flag registered on `init-ceremony` but never read

**Files modified:** `cmd/init_ceremony.go`
**Commit:** 20bb808b
**Applied fix:** Removed the unused `--non-interactive` and `--charter-json` flag registrations. Simplified the TTY check to only use `isTestMode` (for test injection via `stdinReader`), removing the dead `nonInteractive` code path that would have caused a hang on stdin prompt.

### WR-02: Dead code -- `tmpCmd` variable in `runCeremonyResearch`

**Files modified:** `cmd/init_ceremony.go`
**Commit:** a67b4237
**Applied fix:** Removed the unused `tmpCmd` cobra command block (variable creation, flag registration, and `SetArgs` call) that was never executed. Only the later `researchCmd` is actually used.

### WR-03: `validateCharterFieldLength` not called by `init-ceremony`

**Files modified:** `cmd/init_ceremony.go`
**Commit:** 9adf8722
**Applied fix:** Added a call to `validateCharterFieldLength(charter)` in `createCeremonyColony` after the backup logic and before state creation, ensuring charter fields from the ceremony flow are also subject to the 2000-character limit.

## Skipped Issues

### WR-04: Global `stdout` mutation in `runCeremonyResearch` is not concurrency-safe

**File:** `cmd/init_ceremony.go:195-207`
**Reason:** Reviewer noted "No immediate fix required for correctness (single-threaded), but consider a more robust approach in future iterations." The ceremony flow is single-threaded, and the same pattern exists in other commands. This is a design consideration for a future refactor, not a fix to apply now.
**Original issue:** `runCeremonyResearch` temporarily replaces the global `stdout` variable to capture `init-research` output. If two ceremony calls were ever made concurrently, the stdout redirect would be corrupted.

---

_Fixed: 2026-04-28T12:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
