---
phase: 87-fix-continue-depth-persistence
fixed_at: 2026-05-01T12:30:00Z
review_path: .planning/phases/87-fix-continue-depth-persistence/87-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 87: Code Review Fix Report

**Fixed at:** 2026-05-01T12:30:00Z
**Source review:** .planning/phases/87-fix-continue-depth-persistence/87-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 3
- Fixed: 3
- Skipped: 0

## Fixed Issues

### CR-01: `--light` flag on `aether continue` does not suppress keyword-based heavy override

**Files modified:** `cmd/review_depth.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`
**Commit:** 73c58476
**Applied fix:** Extracted `resolveEffectiveContinueDepth` helper in `review_depth.go` that combines CLI flag resolution with state fallback, and critically passes the original `lightFlag`/`heavyFlag` boolean values through to `resolveVerificationDepth`. Previously all three continue call sites resolved boolean flags to a string first, then called `resolveVerificationDepth` with `false, false` -- losing the keyword-guard protection where `lightFlag` blocks heavy keyword auto-detection. The new helper preserves this behavior, matching the `aether build` path. All three call sites (`runCodexContinuePlanOnly`, `missingBuildPacketBlockedResult`, `runCodexContinue`) now use the single helper.

### WR-01: No test coverage for `runCodexContinue` (non-plan-only) path with stored depth

**Files modified:** `cmd/codex_continue_test.go`
**Commit:** ab033bb9
**Applied fix:** Added three new tests:
- `TestMissingBuildPacketHonorsStoredHeavyDepth` -- verifies `missingBuildPacketBlockedResult` uses stored "heavy" depth
- `TestMissingBuildPacketLightFlagBlocksKeywordOverride` -- verifies the CR-01 fix specifically: phase named "Security hardening" with `--light` flag returns light (not keyword-overridden to heavy)
- `TestContinueMissingPacketHonorsStoredDepth` -- verifies the full `runCodexContinue` path with missing build packet returns the correct stored depth

### WR-02: Three-fold duplication of the depth resolution block

**Files modified:** `cmd/review_depth.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`
**Commit:** 73c58476 (combined with CR-01)
**Applied fix:** The identical 5-line depth resolution block (resolve flag -> fallback to state -> call resolveVerificationDepth) was extracted into `resolveEffectiveContinueDepth` in `review_depth.go`. All three call sites now call this single helper. This eliminates the duplication risk and makes future changes to depth resolution in continue paths a single-point edit.

## Skipped Issues

None -- all findings were fixed.

---

_Fixed: 2026-05-01T12:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
