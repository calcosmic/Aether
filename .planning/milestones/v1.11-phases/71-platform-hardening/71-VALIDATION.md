---
phase: 71
slug: platform-hardening
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-28
---

# Phase 71 -- Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none -- existing Go test infrastructure |
| **Quick run command** | `go test ./cmd/... -count=1 -timeout 60s` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -count=1 -timeout 60s`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 71-01-01 | 01 | 1 | PLAT-01/02/03 | T-71-01 | Process tracking validates PID targets | unit | `go test ./... -count=1` | YES (existing) | pending |
| 71-01-02 | 01 | 1 | PLAT-01/02/03 | T-71-02, T-71-03 | State-mutate revert validates guard access; dispatch uses hardcoded map | unit | `go test ./cmd/ -run 'TestStateMutate|TestDispatch|TestShelf' -count=1` | YES (existing) | pending |
| 71-02-01 | 02 | 2 | PLAT-04 | T-71-04, T-71-05, T-71-06, T-71-07 | Flag audit proves coverage; subcommands use store guard pattern | unit | `go test ./cmd/ -run TestCLIFlagAudit -count=1` | YES (created by task) | pending |
| 71-02-02 | 02 | 2 | PLAT-04, PLAT-05 | -- | Smoke test validates all subcommands exit cleanly | smoke | `go test ./cmd/ -run 'TestSubcommandSmoke|TestNewSubcommand' -count=1` | YES (created by task) | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [x] `cmd/smoke_test.go` -- created by Plan 02 Task 2 (PLAT-05)
- [x] `cmd/cli_flag_audit_test.go` -- created by Plan 02 Task 1 (PLAT-04)
- [x] Existing Go test infrastructure covers shelf, dispatch, and state-mutate basics (Plan 01)

Wave 0 test files are created during execution rather than pre-existing. The tasks that create them also contain the automated verify commands that exercise them.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| OpenCode shelf operations end-to-end | PLAT-01, PLAT-02 | Requires OpenCode runtime | Run `/ant-init` + `/ant-seal` in OpenCode, verify shelf sections appear |
| Codex agent dispatch from TOML | PLAT-03 | Requires Codex runtime | Verify each of 26 TOML files maps to a valid agent caste |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (created during execution)
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
