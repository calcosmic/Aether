---
phase: 75
slug: intelligence-core
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-29
---

# Phase 75 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Go stdlib |
| **Quick run command** | `go test ./pkg/memory/... -run Trust -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/memory/... -run Trust -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 75-01-01 | 01 | 1 | INTEL-04 | — | N/A | unit | `go test ./pkg/memory/... -run Trust -count=1` | ✅ | ⬜ pending |
| 75-01-02 | 01 | 1 | INTEL-04 | — | N/A | unit | `go test ./cmd/... -run TestMemoryCapture -count=1` | ❌ W0 | ⬜ pending |
| 75-01-03 | 01 | 1 | INTEL-04 | — | N/A | unit | `go test ./pkg/memory/... -count=1` | ✅ | ⬜ pending |
| 75-02-01 | 02 | 1 | INTEL-05 | — | No task redistribution to tripped worker | unit | `go test ./cmd/ -run TestCircuitBreaker -count=1 -race` | ❌ W0 | ⬜ pending |
| 75-02-02 | 02 | 1 | INTEL-05 | — | Per-wave reset clears breaker state | unit | `go test ./cmd/ -run TestCircuitBreaker -count=1 -race` | ❌ W0 | ⬜ pending |
| 75-02-03 | 02 | 2 | INTEL-05 | — | N/A | integration | `go test ./cmd/ -run TestCircuitBreaker -count=1 -race && grep -c 'cb.Allow' cmd/codex_build.go` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/circuit_breaker_test.go` — stubs for circuit breaker unit tests (INTEL-05)
- [ ] `cmd/learning_test.go` — stubs for memory-capture flag tests (INTEL-04)

*Existing infrastructure covers trust scoring tests (`pkg/memory/trust_test.go`, `pkg/memory/observe_test.go`)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Circuit breaker visual output | INTEL-05 | Rendering is visual | Run build with forced worker failure, verify warning line appears |
| Playbook flag additions | INTEL-04 | Playbooks are markdown | Grep continue-advance.md and continue-full.md for --source-type flag |

*Most phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-29
