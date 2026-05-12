---
phase: 108
slug: golden-workflow-tests
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-05-12
---

# Phase 108 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing 1.26.1 (stdlib) |
| **Config file** | none — uses Go defaults |
| **Quick run command** | `go test ./cmd/ -run "TestGolden" -count=1` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run "TestGolden" -count=1`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 108-01-01 | 01 | 1 | TEST-01, TEST-02, TEST-03 | T-108-01 | N/A — test code | golden (snapshot) | `go test ./cmd/ -run "TestGoldenPlanVisualOutput|TestGoldenBuildVisualOutput|TestGoldenContinueVisualOutput" -count=1` | ❌ W0 | ⬜ pending |
| 108-01-02 | 01 | 1 | TEST-04, TEST-05 | T-108-02, T-108-03 | N/A — test code | unit + integration | `go test ./cmd/ -run "TestGoldenStateMutations" -count=1 && go test ./... -race` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/golden_workflow_test.go` — golden lifecycle snapshot tests + state mutation verification
- [ ] `cmd/testdata/golden_plan.txt` — plan visual output baseline (created by first `-update-golden` run)
- [ ] `cmd/testdata/golden_build.txt` — build visual output baseline (created by first `-update-golden` run)
- [ ] `cmd/testdata/golden_continue.txt` — continue visual output baseline (created by first `-update-golden` run)

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 5s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
