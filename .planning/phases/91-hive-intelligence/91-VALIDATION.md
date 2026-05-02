---
phase: 91
slug: hive-intelligence
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-05-02
---

# Phase 91 -- Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none -- existing infrastructure |
| **Quick run command** | `go test ./pkg/learn/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -timeout 120s` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/learn/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 91-01-1a | 01 | 1 | HIVE-04 | T-91-02 | Parameterized queries | unit | `go test ./pkg/learn/... -run "TestSQLite\|TestMigration\|TestWALMode" -v -count=1` | yes | pending |
| 91-01-1b | 01 | 1 | HIVE-04, HIVE-06 | -- | N/A | unit | `go test ./pkg/learn/... -run "TestCompact\|TestMigrateFromJSON" -v -count=1` | yes | pending |
| 91-01-2 | 01 | 1 | HIVE-05 | T-91-01 | sanitizeFTS5Query | unit | `go test ./pkg/learn/... -run TestFTS5 -v -count=1` | yes | pending |
| 91-02-1 | 02 | 2 | SKIL-01, SKIL-02, SKIL-03 | T-91-06 | validateSkillName | unit | `go test ./pkg/learn/... -run TestSkill -v -count=1` | yes | pending |
| 91-02-2 | 02 | 2 | HIVE-05, SKIL-01 | T-91-07 | sanitizeFTS5Query | unit | `go build ./cmd/aether` | yes | pending |
| 91-03-1 | 03 | 3 | SKIL-04, SKIL-05, SKIL-06 | — | N/A | unit | `go test ./pkg/learn/... -run TestCurator -v -count=1` | yes | pending |
| 91-03-2 | 03 | 3 | SKIL-04 | — | N/A | unit | `go build ./cmd/aether && go test ./pkg/learn/... -count=1 -timeout 60s` | yes | pending |
| 91-04-1 | 04 | 4 | AUTO-01, AUTO-02, AUTO-03 | T-91-15 | IsAutoSkillRejected | unit | `go test ./pkg/learn/... -run "TestAssess\|TestAutoSkill\|TestLoadAutoSkillMode" -v -count=1` | yes | pending |
| 91-04-2 | 04 | 4 | AUTO-01, AUTO-04 | T-91-16 | Structural isolation | unit | `go build ./cmd/aether && go test ./pkg/learn/... -count=1 -timeout 60s` | yes | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No Wave 0 stubs needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WAL mode persistence across restarts | HIVE-04 | Requires process lifecycle | Create db, verify WAL mode with `PRAGMA journal_mode`, kill process, reopen, verify still WAL |
| FTS5 search relevance quality | HIVE-05 | Relevance quality is subjective | Insert test data, run searches, verify results contain expected entries |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 45s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
