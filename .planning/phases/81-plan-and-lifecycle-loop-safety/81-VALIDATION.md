---
phase: 81
slug: plan-and-lifecycle-loop-safety
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-30
---

# Phase 81 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./pkg/colony/ -run "TestDetectCycles" -count=1 && go test ./cmd/ -run "TestRecovery|TestClassifyError|TestNormalize|TestRenderRecovery" -count=1` |
| **Full suite command** | `go test ./... -count=1 -timeout 120s` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/colony/ -run "TestDetectCycles" -count=1` or `go test ./cmd/ -run "TestRecovery" -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 81-01-01 | 01 | 1 | LOOP-04 | — | Cycle detection rejects circular task deps | unit | `go test ./pkg/colony/ -run "TestDetectCycles" -count=1` | No — W0 | pending |
| 81-01-02 | 01 | 1 | LOOP-04 | — | Plan validation gate rejects cyclic plans | unit | `go test ./cmd/ -run "TestPlan" -count=1` | Yes (existing) | pending |
| 81-02-01 | 02 | 1 | LOOP-05 | — | Recovery menu never suggests same command | unit | `go test ./cmd/ -run "TestRecovery\|TestClassifyError\|TestNormalize\|TestRenderRecovery" -count=1` | No — W0 | pending |
| 81-02-02 | 02 | 1 | LOOP-05 | — | Lifecycle commands use recovery engine | integration | `go test ./cmd/ -count=1 -timeout 120s` | Yes (existing) | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `pkg/colony/cycle_test.go` — cycle detection unit tests (created by 81-01 Task 1)
- [ ] `cmd/recovery_engine_test.go` — recovery engine unit tests (created by 81-02 Task 1)

*Both test files are created as part of their respective TDD tasks. No separate Wave 0 scaffolding needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Recovery menu renders correctly in terminal | LOOP-05 | Visual output | Run `aether seal` in a colony with missing state, verify menu displays numbered options |
| Cycle rejection message is clear to AI planner | LOOP-04 | Qualitative | Generate a plan with circular deps and verify the error message identifies the cycle path |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
