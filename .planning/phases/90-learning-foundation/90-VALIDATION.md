---
phase: 90
slug: learning-foundation
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-05-01
---

# Phase 90 -- Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none -- existing |
| **Quick run command** | `go test ./pkg/learn/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -race -count=1 -timeout 120s` |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/learn/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -race -count=1 -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 90-01-01 | 01 | 1 | HIVE-01, HIVE-02, LRN-05 | T-90-01, T-90-02, T-90-03 | N/A | unit (TDD) | `go test ./pkg/learn/... -run "TestColonyStore" -v -count=1` | pkg/learn/colony_store_test.go | ⬜ pending |
| 90-02-01 | 02 | 1 | LRN-01, LRN-02, PRIV-03, PRIV-05 | T-90-04, T-90-05, T-90-06 | privacyScan blocks secrets before classification | unit (TDD) | `go test ./pkg/learn/... -run "TestIsLearningEligible\|TestCollectEvidence\|TestClassifyEntry" -v -count=1` | pkg/learn/trigger_test.go, pkg/learn/classify_test.go | ⬜ pending |
| 90-03-01 | 03 | 2 | LRN-01, LRN-02, LRN-04, HIVE-03, PRIV-03 | T-90-07, T-90-08 | Privacy scan + classification before storage; ClassBlocked never stored | integration | `go build ./cmd/aether && go test ./cmd/... -count=1 -timeout 60s` | cmd/codex_continue_finalize.go | ⬜ pending |
| 90-03-02 | 03 | 2 | LRN-04, HIVE-03 | T-90-09, T-90-10 | Learned content is read-only frozen snapshot | integration | `go build ./cmd/aether && go test ./cmd/... -count=1 -timeout 60s` | cmd/colony_prime_context.go | ⬜ pending |
| 90-04-01 | 04 | 3 | HIVE-01, LRN-03, LRN-06, PRIV-04 | T-90-11, T-90-12, T-90-14 | Export runs privacyScan; blocked entries skipped; redaction report generated | unit | `go test ./pkg/learn/... -run "TestExportPack\|TestImportPack" -v -count=1 && go build ./cmd/aether` | pkg/learn/export_test.go, cmd/learn_export.go | ⬜ pending |
| 90-04-02 | 04 | 3 | PRIV-05, D-08 | T-90-13 | Config + flag control learning enablement; cmd/ call sites use pkg/learn/ | integration | `go build ./cmd/aether && go test ./cmd/... ./pkg/learn/... ./pkg/memory/... -count=1 -timeout 60s` | cmd/learning.go, cmd/learning_cmds.go, cmd/graph_consolidation_cmds.go | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Wave 0 test stubs are created inline during TDD execution (Plans 01 and 02 are type: tdd with RED phase producing test files first). No separate Wave 0 scaffold task needed.

- [x] `pkg/learn/colony_store_test.go` -- created in Plan 01 Task 1 RED phase (TDD)
- [x] `pkg/learn/trigger_test.go` -- created in Plan 02 Task 1 RED phase (TDD)
- [x] `pkg/learn/classify_test.go` -- created in Plan 02 Task 1 RED phase (TDD)
- [x] `pkg/learn/export_test.go` -- created in Plan 04 Task 1
- [x] Plans 03 and 04 (type: execute) verify via `go build ./cmd/aether` and existing test suites

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `--no-learn` flag suppresses learning writes | PRIV-05 | CLI flag behavior | Run `/ant-continue --no-learn`, verify no entries written to .aether/data/learn/ |
| Repo isolation between two repos | HIVE-02 | Cross-repo state | Create entries in repo A, verify repo B cannot read them |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
