---
phase: 162
slug: switch-on-learning
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-04
---

# Phase 162 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./cmd/ -run 'Consolidation|Hive|ContinueFinalize|Seal' -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~120 seconds full, ~20 seconds quick |

---

## Sampling Rate

- **After every task commit:** Run quick command
- **After every plan wave:** Run full suite
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

*(Filled by planner — one row per task with automated command per requirement.)*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | — | — | LEARN-01 | — | Consolidation failure warns loudly, never blocks advance | integration | `go test ./cmd/ -run TestContinueFinalizeInvokesConsolidation -count=1` | ❌ W0 | ⬜ pending |
| TBD | — | — | LEARN-02 | — | Seal runs eight-ant pass without double-promotion | integration | `go test ./cmd/ -run TestSealInvokesConsolidation -count=1` | ❌ W0 | ⬜ pending |
| TBD | — | — | LEARN-03 | — | Decision record exists and docs agree | doc-assert | `test -f .aether/docs/decisions/*learning*` (exact path set by planner) | ❌ W0 | ⬜ pending |
| TBD | — | — | LEARN-04 | — | Hive default is promote; consent gate retired | unit | `go test ./cmd/ -run TestHivePolicyDefault -count=1` | ❌ W0 | ⬜ pending |
| TBD | — | — | LEARN-05 | — | Brief differs populated vs wiped memory | unit (deterministic) | `go test ./cmd/ -run TestWorkerBriefMemoryInjection -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Existing infrastructure covers framework needs (go test already in place)
- [ ] New test files created alongside wiring changes per plan tasks

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Layer-2 before/after artifact (one real worker run each way) | LEARN-05 | Requires live model calls — deliberately excluded from CI per D-08 | Run same worker task with memory populated, then wiped; save differing outputs to phase directory as recorded exhibit |
| Ceremony visual appearance (🧠 beat, eight-ant announcements) | LEARN-01/02 | ANSI/emoji rendering is visual | Run `/ant-continue` at phase end and `/ant-seal`; confirm beats render per D-06/D-07 |
