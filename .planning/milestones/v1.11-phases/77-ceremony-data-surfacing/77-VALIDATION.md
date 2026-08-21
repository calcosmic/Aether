---
phase: 77
slug: ceremony-data-surfacing
status: verified
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-29
---

# Phase 77 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./cmd/... -run "TestRenderResearch\|TestCircuitBreaker" -v` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run "TestRenderResearch\|TestCircuitBreaker" -v`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 77-01-01 | 01 | 1 | INIT-03, INIT-04, INIT-05, INIT-07, INTEL-05 | T-77-01 / T-77-02 | Research data display and circuit breaker event bus routing | unit | `go test ./cmd/... -run "TestRenderResearch\|TestCircuitBreaker" -v` | ✅ W0 | ✅ green |
| 77-01-02 | 01 | 1 | INTEL-01 | T-77-03 | --no-suggest flag registration is boolean-only | unit | `go build ./cmd/aether && aether build --help \| grep no-suggest` | ✅ W0 | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing Go test infrastructure covers all requirements. 6 new tests added during execution (3 renderResearchDisplay, 3 circuitBreaker event bus). No new test infrastructure needed.

---

## Manual-Only Verifications

All phase behaviors have automated verification (77-VERIFICATION.md confirms: "All behaviors are verifiable programmatically through tests, grep, and build output").

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** verified
