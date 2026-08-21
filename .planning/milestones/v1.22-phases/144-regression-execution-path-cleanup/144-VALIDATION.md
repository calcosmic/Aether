---
phase: 144
slug: regression-execution-path-cleanup
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-05-18
---

# Phase 144 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + Node.js `node:test` |
| **Config file** | None (built-in conventions) |
| **Quick run command** | `go test ./cmd -run "TestVenv|TestGround|TestCLIFlag|TestWrapper|TestCodexLifecycle" -count=1` |
| **Full suite command** | `go test ./... -count=1 && cd .aether/ts-host && npm test` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd -count=1`
- **After every plan wave:** Run `go test ./... -count=1 && cd .aether/ts-host && npm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 144-01-01 | 01 | 1 | CLEAN-03 | — | N/A | unit | `go test ./cmd -run "TestWrapperSources|TestCodexLifecycle|TestLifecycleWrapper" -count=1` | YES | ⬜ pending |
| 144-01-02 | 01 | 1 | CLEAN-03, CLEAN-04 | — | N/A | unit | `go test ./cmd -run "TestCLIFlagAudit" -count=1` | YES | ⬜ pending |
| 144-02-01 | 02 | 1 | CLEAN-01, CLEAN-05 | — | N/A | integration | `go test ./cmd -run "TestM4LRegression|TestCommandGuideBuild|TestCommandGuidePlan" -v -count=1` | W0 | ⬜ pending |
| 144-02-02 | 02 | 1 | CLEAN-02, CLEAN-06 | — | N/A | audit | `go test ./cmd -run "TestExecutionPathAudit" -count=1 && cd .aether/ts-host && node --test test/cross-platform-parity.test.ts` | YES | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `cmd/codex_colonize_test.go` — add `TestM4LRegression_VenvProducesGroundedPlan` (CLEAN-01)
- [x] `cmd/command_guide_test.go` — fix `TestCLIFlagAudit` false positive (CLEAN-03)
- [x] `.aether/commands/build.yaml` — restore missing anchors (CLEAN-03)
- [x] `.aether/commands/plan.yaml` — restore missing anchors (CLEAN-03)

*Existing infrastructure covers all remaining phase requirements.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Execution path audit documents all 5 workflows x 2 platforms | CLEAN-02 | Documentation audit | Review generated test output for all 10 declared paths |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
