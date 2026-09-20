---
phase: 74
slug: suggest-analyze
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-29
---

# Phase 74 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./cmd/ -run TestSuggest -v` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run TestSuggest -v`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 74-01-01 | 01 | 1 | INTEL-01 | — | N/A | unit | `go test ./cmd/ -run TestSuggestAnalyze -v` | ❌ W0 | ⬜ pending |
| 74-02-01 | 02 | 1 | INTEL-02 | — | No prompt injection in suggestions | unit | `go test ./cmd/ -run TestSuggestDedup -v` | ❌ W0 | ⬜ pending |
| 74-03-01 | 03 | 1 | INTEL-03 | — | No unauthorized state mutations | unit | `go test ./cmd/ -run TestSuggestApprove -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/suggest_analyze_test.go` — stubs for INTEL-01, INTEL-02, INTEL-03
- [ ] `cmd/suggest_approve_test.go` — stubs for approve/dismiss flow

*Existing infrastructure covers all phase requirements — Go test framework already in use.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tick-to-approve UI renders suggestions | INTEL-03 | CLI output formatting is visual | Run `aether suggest-approve`, verify each suggestion shows [y/N] prompt |
| Suggest-analyze runs during build Step 4.2 | INTEL-01 | Integration across build playbook | Run `/ant-build` and verify suggest-analyze output appears in Step 4.2 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
