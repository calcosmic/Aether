---
phase: 72
slug: smart-init-charter
status: verified
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-28
---

# Phase 72 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/... -run TestInit -count=1 -v` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run TestInit -count=1 -v`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 72-01-01 | 01 | 1 | INIT-01 | T-72-01 / — | Charter generation from scan data is deterministic | unit | `go test ./cmd/... -run TestCharter` | ✅ W0 | ✅ green |
| 72-01-02 | 01 | 1 | INIT-01 | T-72-01 / — | Charter persistence uses valid JSON | unit | `go test ./pkg/colony/... -run TestCharter` | ✅ W0 | ✅ green |
| 72-01-03 | 01 | 1 | INIT-02 | T-72-02 / — | Rejected charter does not write COLONY_STATE.json | unit | `go test ./cmd/... -run TestInitReject` | ✅ W0 | ✅ green |
| 72-02-01 | 02 | 1 | INIT-02 | — / — | Go ceremony prompt reads user input correctly | unit | `go test ./cmd/... -run TestInitCeremony` | ✅ W0 | ✅ green |
| 72-02-02 | 02 | 2 | INIT-02 | — / — | Wrapper charter display with --charter-json flag | grep | `grep -c "charter.tech_stack" .claude/commands/ant/init.md && grep -c "charter.tech_stack" .opencode/commands/ant/init.md` | ✅ W0 | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `cmd/init_research_test.go` — stubs for charter generation (INIT-01) — created by Plan 01 Task 1 TDD RED phase
- [x] `pkg/colony/colony_test.go` — stubs for charter persistence (INIT-01) — created by Plan 01 Task 1 TDD RED phase
- [x] `cmd/init_cmd_test.go` — stubs for ceremony/reject flow (INIT-02) — created by Plan 02 Task 1 TDD RED phase

*TDD RED phase in Plan 01 Task 1 and Plan 02 Task 1 creates these test stubs before implementation (GREEN phase).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Charter ceremony displays correctly in wrapper mode | INIT-02 | CLI rendering is platform-specific | Run `/ant-init "test"` in Claude Code and verify charter markdown renders |
| Go native ceremony prompt is readable | INIT-02 | TUI output requires human judgment | Run `aether init "test"` and verify prompt text is clear |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** verified
