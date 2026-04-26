---
phase: 54
slug: colony-prime-prior-reviews-section
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-26
---

# Phase 54 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./cmd/ -run TestPriorReviews -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run TestPriorReviews -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 54-01-01 | 01 | 1 | PRIME-05, PRIME-09 | — | N/A | build | `go build ./cmd/ && go vet ./cmd/` | ❌ W0 | ⬜ pending |
| 54-01-02 | 01 | 1 | PRIME-01, PRIME-02, PRIME-03, PRIME-04, PRIME-05 | — | N/A | unit | `go test ./cmd/ -run TestPriorReviews -v -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/colony_prime_prior_reviews_test.go` — 14 test functions covering PRIME-01 through PRIME-05

*Existing go test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Colony-prime output contains prior-reviews section with real ledger data | PRIME-01 | Requires running `aether colony-prime` with populated ledgers | Create test ledger files, run `aether colony-prime`, inspect output for prior-reviews section |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
