---
phase: 174
slug: spend-ledger
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-13
---

# Phase 174 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go's built-in `testing` package (`go test`) |
| **Config file** | none — standard `go.mod`-driven test discovery |
| **Quick run command** | `go test ./pkg/codex/... ./cmd/... -run Spend` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~90 seconds (full suite) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./pkg/codex/... ./cmd/... -run Spend`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

*(Task IDs filled by planner; requirement mapping from RESEARCH.md Validation Architecture.)*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | — | — | SPEND-01 | — | Usage survives past process exit | unit | `go test ./cmd/... -run TestSpendLedgerPersistsAcrossProcesses -v` | ❌ W0 | ⬜ pending |
| TBD | — | — | SPEND-02 | T-174-01 | Wrapper-path usage row is non-empty and never `provider`-tagged; transcript path bounds-checked | unit (fixture transcript) | `go test ./cmd/... -run TestWrapperPathUsageFromClaudeTranscript -v` | ❌ W0 | ⬜ pending |
| TBD | — | — | SPEND-03 | — | Documented 50/100000/2000/500 → 102,550 literal assertion | unit (exists, direct path) | `go test ./pkg/codex/... -run TestClaudeUsageTotalIncludesCacheReadAndCreation -v` | ✅ | ⬜ pending |
| TBD | — | — | SPEND-04 | — | Measured/estimated subtotals separate; derived metric refused without `--include-estimates` | unit | `go test ./cmd/... -run TestLedgerMeasuredEstimatedSubtotalsAreSeparate -v` | ❌ W0 | ⬜ pending |
| TBD | — | — | SPEND-05 | — | Grand total equals sum of worker rows (invariant); per-parent roll-up derivable | unit (property-style) | `go test ./cmd/... -run TestLedgerGrandTotalEqualsSumOfRows -v` | ❌ W0 | ⬜ pending |
| TBD | — | — | SPEND-06 | — | `aether spend` per-worker report; byte-identical on repeat run | integration (CLI purity) | `go test ./cmd/... -run TestSpendCommandIsIdempotent -v` | ❌ W0 | ⬜ pending |
| TBD | — | — | SPEND-07 | — | No spend figure derives from `colonyPrimeBudgetChars`; docs renamed | unit + doc-lint | `go test ./cmd/... -run TestSpendFigureNeverDerivesFromCharBudget -v` | ❌ W0 | ⬜ pending |
| TBD | — | — | SPEND-08 | — | Closeout prints one plain-English token line on build and continue | unit | `go test ./cmd/... -run TestCeremonyCloseoutIncludesTokenLine -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/spend_ledger_test.go` — stubs for SPEND-01, SPEND-04, SPEND-05
- [ ] `cmd/spend_cmd_test.go` — stubs for SPEND-06 (idempotency, reusing `assertConsolidationDryRunIsPure` pattern)
- [ ] `cmd/wrapper_usage_claude_test.go` — SPEND-02, committed fixture `.jsonl` (trimmed/redacted real transcript shape, foreground `tool_result` and async `<task-notification>` forms)
- [ ] `cmd/wrapper_usage_opencode_test.go` — SPEND-02 (OpenCode), committed fixture `project.json`/`session.json`/`message.json`
- [ ] `cmd/ceremony_cmd_test.go` addition — SPEND-08
- [ ] No new test framework install needed — `go test` already fully configured

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Doc audit: no remaining "token budget" naming for the character budget | SPEND-07 | Prose rename across ~9 files; grep is the check, judgment is the fix | `grep -ri "token budget" CLAUDE.md .aether/docs/` returns only intentional references |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
