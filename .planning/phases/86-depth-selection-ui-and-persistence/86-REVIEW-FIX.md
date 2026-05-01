---
phase: 86-depth-selection-ui-and-persistence
fixed_at: 2026-05-01T12:30:00Z
review_path: .planning/phases/86-depth-selection-ui-and-persistence/86-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 86: Code Review Fix Report

**Fixed at:** 2026-05-01T12:30:00Z
**Source review:** .planning/phases/86-depth-selection-ui-and-persistence/86-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4
- Fixed: 4
- Skipped: 0

## Fixed Issues

### CR-01: renderBuildVisualWithDispatches and renderBuildPlanOnlyVisual hardcode smartDefault=true, misrepresenting user intent

**Files modified:** `cmd/codex_visuals.go`
**Commit:** a7eb9278
**Applied fix:** Replaced `renderReviewDepthLineWithReason(reviewDepth, ..., true)` with `renderReviewDepthLine(reviewDepth, ...)` in both `renderBuildVisualWithDispatches` (line 1163) and `renderBuildPlanOnlyVisual` (line 1217). These callers have no way to determine whether the depth was smart-defaulted or user-specified, so showing the reason annotation was misleading. The plan visual at `codex_visuals.go:896` already correctly gates on `verificationSmartDefault` -- the build visuals now match that pattern by using the no-reason variant.

### WR-01: resolveReviewDepth (2-level legacy) ignores lightFlag for keyword phases, inconsistent with resolveVerificationDepth (3-level)

**Files modified:** `cmd/review_depth.go`, `cmd/review_depth_test.go`
**Commit:** db7b3868
**Applied fix:** Added `&& !lightFlag` guard to the keyword auto-detection branch in `resolveReviewDepth`, matching the 3-level `resolveVerificationDepth` behavior where user intent (light flag) overrides keyword match. Updated the test case from "keyword phase with light flag still heavy" expecting `ReviewDepthHeavy` to "keyword phase with light flag overrides to light" expecting `ReviewDepthLight`.

### WR-02: TestDepthKeysPresentInFreshPlanResultMap scans source text, creating fragile coupling

**Files modified:** `cmd/review_depth_test.go`
**Commit:** 15d89d15
**Applied fix:** Added a clear comment documenting the intentional tradeoff: this is a structural regression guard that intentionally scans source text to prevent a prior regression (missing depth keys in result map paths) from recurring. The comment instructs maintainers to update the expected count if a refactor moves keys into a shared helper.

### WR-03: resolveVerificationDepthSmart validates raw input separately from NormalizeVerificationDepth, creating drift risk

**Files modified:** `cmd/review_depth.go`, `cmd/review_depth_test.go`
**Commit:** 7e83d7d6
**Applied fix:** Removed the redundant hardcoded alias switch in `resolveVerificationDepthSmart`, making `NormalizeVerificationDepth` the single source of truth for depth normalization. Unknown inputs now fall through to the default "standard" depth via `NormalizeVerificationDepth` instead of returning an error. Updated the test to verify "extreme" normalizes to "standard" rather than expecting an error. Removed the now-unused `fmt` import from `review_depth.go`.

---

_Fixed: 2026-05-01T12:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
