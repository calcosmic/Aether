---
phase: 51
slug: recovery-verification
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-25
---

# Phase 51 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/ -run TestE2ERecovery -v -count=1` |
| **Full suite command** | `go test ./cmd/ -v -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run TestE2ERecovery -v -count=1`
- **After every plan wave:** Run `go test ./cmd/ -v -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 51-01-01 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryMissingBuildPacket -v` | ❌ W0 | ⬜ pending |
| 51-01-02 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryStaleSpawned -v` | ❌ W0 | ⬜ pending |
| 51-01-03 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryPartialPhase -v` | ❌ W0 | ⬜ pending |
| 51-01-04 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryBadManifest -v` | ❌ W0 | ⬜ pending |
| 51-01-05 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryDirtyWorktree -v` | ❌ W0 | ⬜ pending |
| 51-01-06 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryBrokenSurvey -v` | ❌ W0 | ⬜ pending |
| 51-01-07 | 01 | 1 | TEST-01 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryMissingAgents -v` | ❌ W0 | ⬜ pending |
| 51-01-08 | 01 | 1 | TEST-02 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryCompoundState -v` | ❌ W0 | ⬜ pending |
| 51-01-09 | 01 | 1 | TEST-03 | — | N/A | E2E | `go test ./cmd/ -run TestE2ERecoveryHealthyColony -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/e2e_recovery_test.go` — all E2E test stubs for TEST-01, TEST-02, TEST-03

Existing infrastructure covers all phase requirements (Go stdlib, no framework install needed).

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
