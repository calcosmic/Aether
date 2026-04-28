---
phase: 68
slug: gate-recovery-verification
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-28
---

# Phase 68 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none (Go convention) |
| **Quick run command** | `go test ./cmd/... -run "TestGate" -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run "TestGate" -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 68-01-01 | 01 | 1 | GATE-01 | — | Recovery templates render per gate | unit | `go test ./cmd/... -run "TestGateRecoveryTemplate" -count=1` | ✅ | ⬜ pending |
| 68-01-02 | 01 | 1 | GATE-01 | — | gate-recovery-template CLI works | unit | `go test ./cmd/... -run "TestGateRecoveryTemplateCmd" -count=1` | ✅ | ⬜ pending |
| 68-01-03 | 01 | 1 | GATE-03 | — | gateResultsWrite merges entries (CR-01 fix) | unit | `go test ./cmd/... -run "TestGateResultsMerge" -count=1` | ❌ W0 | ⬜ pending |
| 68-01-04 | 01 | 1 | GATE-03 | — | shouldSkipGate skips passed, never skips tests_pass | unit | `go test ./cmd/... -run "TestShouldSkipGate" -count=1` | ✅ | ⬜ pending |
| 68-02-01 | 02 | 1 | GATE-03 | — | Finalize path persists gate results (WR-01 fix) | unit | `go test ./cmd/... -run "TestFinalizeGateResults" -count=1` | ❌ W0 | ⬜ pending |
| 68-02-02 | 02 | 1 | GATE-03 | — | Finalize path clears gate results on advance (WR-02 fix) | unit | `go test ./cmd/... -run "TestFinalizeGateResultsClear" -count=1` | ❌ W0 | ⬜ pending |
| 68-03-01 | 03 | 1 | GATE-01, GATE-02, GATE-03 | — | Phase 59 VERIFICATION.md created with all evidence | manual | File existence check | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/gate_test.go` — test for `gateResultsWrite` merge behavior (CR-01 verification)
- [ ] `cmd/gate_incremental_test.go` — test for finalize path gate result persistence (WR-01 verification)
- [ ] `cmd/gate_incremental_test.go` — test for finalize path gate result clearing (WR-02 verification)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Playbook renders recovery templates | GATE-01 | Markdown playbook, not Go code | grep continue-verify.md for "gate-recovery-template" calls |
| Watcher Veto shows 3 choices | GATE-02 | Markdown playbook behavior | grep continue-gates.md for AskUserQuestion with 3 options |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
