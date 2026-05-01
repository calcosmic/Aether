---
phase: 86
slug: depth-selection-ui-and-persistence
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-05-01
---

# Phase 86 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./cmd/ -run "TestReviewDepth|TestPlanDepth|TestDepth" -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run "TestReviewDepth|TestPlanDepth|TestDepth" -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 86-01-01 | 01 | 1 | DEPTH-04 | T-86-01, T-86-02 | Flag validation + state persistence | unit | `go test ./cmd/ -run "TestResolveVerificationDepthSmart" -count=1` | No — Wave 0 | pending |
| 86-01-02 | 01 | 1 | DEPTH-04 | T-86-03 | N/A | unit | `go test ./cmd/ -run "TestRenderPlan" -count=1` | Yes | pending |
| 86-02-01 | 02 | 2 | DEPTH-05 | T-86-03, T-86-04 | Manifest integrity | unit | `go test ./cmd/ -run "TestBuildManifest|TestReviewDepth" -count=1` | Yes | pending |
| 86-02-02 | 02 | 2 | DEPTH-04, DEPTH-05 | T-86-05 | N/A | unit | `go test ./cmd/ -run "TestBuildVisual" -count=1` | Yes | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements. No new test setup needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Depth banner visual output | DEPTH-04 | Console formatting is cosmetic | Run `aether plan` and verify banner appears with both depths and reasons |
| Flag override behavior | DEPTH-04 | CLI flag interaction | Run `aether plan --verification-depth heavy` and verify override takes effect |
| Continue reads stored depth | DEPTH-05 | End-to-end CLI flow | Run `aether plan` then `aether continue` and verify depth persists |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
