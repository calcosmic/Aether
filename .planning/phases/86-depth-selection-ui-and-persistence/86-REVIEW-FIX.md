---
phase: 86-depth-selection-ui-and-persistence
fixed_at: 2026-05-01T12:00:00Z
review_path: .planning/phases/86-depth-selection-ui-and-persistence/86-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 86: Code Review Fix Report

**Fixed at:** 2026-05-01T12:00:00Z
**Source review:** .planning/phases/86-depth-selection-ui-and-persistence/86-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4
- Fixed: 4
- Skipped: 0

## Fixed Issues

### CR-01: `resolveVerificationDepth` ignores `lightFlag` when keyword is matched, contradicting priority contract

**Files modified:** `cmd/review_depth.go`, `cmd/review_depth_test.go`
**Commit:** f2c68fde
**Applied fix:** Added `&& !lightFlag` condition to the keyword match check in `resolveVerificationDepth`, so that an explicit `--light` flag overrides keyword auto-detection (user intent takes priority). Updated the priority comment and the test that previously expected keyword+light to yield heavy.

### WR-01: Keyword false-positive risk in `phaseRiskLevel` due to substring matching on phase text

**Files modified:** `cmd/review_depth.go`, `cmd/review_depth_test.go`
**Commit:** f76b1c68
**Applied fix:** Changed `phaseRiskLevel` to match keywords only against `phase.Name` (lowercased) instead of the full phase text (name + description + task goals/constraints/hints). This prevents common words like "session", "token", "password" from triggering false "high" risk classification when they appear in task descriptions. Updated three tests to reflect the new name-only matching behavior.

### WR-02: `findRepoRelativePath` fallback uses unbounded recursive walk on the entire repo

**Files modified:** `cmd/codex_build_finalize.go`, `cmd/codex_build_finalize_test.go`
**Commit:** 8a0f1a2f
**Applied fix:** Removed the `filepath.WalkDir` fallback entirely. If `git ls-files` finds nothing, the file likely does not exist in the repo. The unbounded walk could exhaust file descriptors on large repos or walk massive `node_modules`/`vendor` directories. Updated the subdirectory-relative test to expect the original path to be kept (not resolved) when no git repo is available.

### WR-03: Redundant state save in `runCodexBuildFinalize`

**Files modified:** `cmd/codex_build_finalize.go`
**Commit:** 3e56fe0f
**Applied fix:** Replaced the non-atomic `store.SaveJSON("COLONY_STATE.json", updatedState)` with `store.UpdateJSONAtomically`, matching the pattern used in `runCodexBuildWithOptions` and other build paths. The state mutation is now committed atomically after all dependent writes (claims, outcome reports, manifest, spawn tree) succeed, preventing the race window where a checkpoint says one thing and the live state says another.

---

_Fixed: 2026-05-01T12:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
