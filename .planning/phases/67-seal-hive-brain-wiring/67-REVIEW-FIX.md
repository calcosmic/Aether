---
phase: 67-seal-hive-brain-wiring
fixed_at: 2026-04-28T01:42:00Z
review_path: .planning/phases/67-seal-hive-brain-wiring/67-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 4
skipped: 1
status: partial
---

# Phase 67: Code Review Fix Report

**Fixed at:** 2026-04-28T01:42:00Z
**Source review:** .planning/phases/67-seal-hive-brain-wiring/67-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5
- Fixed: 4
- Skipped: 1

## Fixed Issues

### WR-02: Three json.Unmarshal calls silently ignore errors in hive.go

**Files modified:** `cmd/hive.go`
**Commit:** 8d375843
**Applied fix:** Added error checking after all three `json.Unmarshal` calls in `hive-store` (line 95), `hive-read` (line 180), and `promoteToHive` (line 271). On corruption, `hive-store` and `hive-read` now report via `outputError` and return nil. `promoteToHive` returns a `fmt.Errorf` so callers can decide whether to block. This prevents corrupted wisdom.json from being overwritten with empty data.

### WR-03: writeWisdom error silently ignored in hive-read

**Files modified:** `cmd/hive.go`
**Commit:** 962f54ed
**Applied fix:** Changed the bare `writeWisdom(wisdomPath, wf)` call in hive-read to capture and check the error. On failure, reports via `outputError` and returns nil, consistent with other writeWisdom call sites in hive.go.

### WR-04: hive-store computes textHash but deduplication does not use it

**Files modified:** `cmd/hive.go`
**Commit:** f0b1fa41
**Applied fix:** Moved `textHash := fmt.Sprintf(...)` from before the dedup loop to after it, placing it immediately before the new entry construction. This avoids the SHA-256 computation on every call when the entry already exists (reinforcement path). The hash is still used for ID generation on new entries.

### WR-01: sourceRepo always empty from seal, disabling multi-repo confidence boosting

**Files modified:** `cmd/codex_workflow_cmds.go`
**Commit:** 9f578ad9
**Applied fix:** Added repo name detection at the start of the seal ceremony. First tries `git remote get-url origin` and extracts the last path component (stripping `.git` suffix). Falls back to `filepath.Base(os.Getwd())` if git remote is unavailable. Passes the derived `repoName` to `promoteToHive` instead of the empty string, enabling the multi-repo confidence boosting feature documented in CLAUDE.md.

## Skipped Issues

### CR-01: Hive promoted count inflated for instincts with empty Action

**File:** `cmd/codex_workflow_cmds.go:321`
**Reason:** Code context differs from review. The reviewer cited line 299 having only `entry.Confidence >= 0.8`, but the actual code at line 321 already includes `&& entry.Action != ""` in the hive eligibility guard. This fix was already applied in the current codebase. The local promotion guard (line 316) and the hive eligibility guard (line 321) both correctly check for non-empty Action text.
**Original issue:** The hive promotion count could be inflated when instincts have high confidence but empty Action text.

---

_Fixed: 2026-04-28T01:42:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
