---
phase: 55
slug: agent-definition-updates
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-26
updated: 2026-04-26
---

# Validation Strategy: Phase 55 — Agent Definition Updates

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (conventional `go test`) |
| Quick run command | `go test ./cmd/... -run TestFindingsInjection -count=1` |
| Full suite command | `go test ./... -count=1` |

## Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? | Status |
|--------|----------|-----------|-------------------|-------------|--------|
| AGENT-01 through AGENT-07 | Agent body text contains Write tool, guardrails, findings instructions | Manual verification (grep) | `grep "Write" .claude/agents/ant/aether-{agent}.md` | ✅ verified | ✅ green |
| AGENT-08 | All 4 surfaces in sync for all 7 agents | Integration (diff) | `diff .claude/agents/ant/aether-{agent}.md .aether/agents-claude/aether-{agent}.md` | ✅ verified | ✅ green |
| AGENT-09 | Write-scope guardrails present in boundaries section | Manual verification (grep) | `grep "MUST NOT write to" .claude/agents/ant/aether-{agent}.md` | ✅ verified | ✅ green |
| AGENT-10 | Dispatch injection text appears for review castes | Unit test | `go test ./cmd/... -run TestFindingsInjection -count=1` | ✅ exists | ✅ green |

## Sampling Rate

- **Per task commit:** `go test ./cmd/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green + manual mirror parity verification (28 files)

## Wave 0 Gaps

- [x] Unit test for `findingsInjectionForCaste` — covers AGENT-10
- [x] Unit tests for continue review brief (gatekeeper has findings, probe does not) — covers AGENT-10
- [x] Existing review ledger tests cover the CLI commands agents will call — no gap

## Manual Verification Checklist

These items require grep/file comparison since they are content checks on markdown/TOML files:

- [x] All 7 canonical agents have `Write` in `tools:` frontmatter
- [x] No canonical agent contains unqualified "no Write" without scoped Write exception
- [x] All 7 agents-claude mirrors are byte-identical to canonical (`diff` returns no output)
- [x] All 7 OpenCode mirrors have `write: true` in YAML frontmatter
- [x] All 7 Codex TOML files reference `review-ledger-write`
- [x] Tracker contains bugs domain carve-out and "except reviews/bugs/"

## Validation Audit 2026-04-26

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 10 requirements verified green (20 automated tests + 28 file verification checks). No new tests needed.
