---
phase: 164
slug: research-feeds-planning
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-02
---

# Phase 164 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go runtime) + vitest (.aether/ts-host) |
| **Config file** | go.mod / .aether/ts-host/vitest.config.ts |
| **Quick run command** | `go test ./cmd/... -run 'TestPlan|TestResearch|TestConfidence' -count=1` |
| **Full suite command** | `go test ./... && (cd .aether/ts-host && npx vitest run)` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/... -run 'TestPlan|TestResearch|TestConfidence' -count=1`
- **After every plan wave:** Run `go test ./... && (cd .aether/ts-host && npx vitest run)`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (filled by planner) | — | — | RESEARCH-01..10 | — | — | unit | — | ⬜ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Existing `go test ./...` and ts-host vitest infrastructure covers phase requirements — no new framework install expected. Planner confirms per-task commands.

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live confidence readout visible during plan run | RESEARCH-07 | Terminal UX rendering during a live run | Run `/ant-plan` on a research-warranted phase; observe confidence readout updates while research runs |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
