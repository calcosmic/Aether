---
phase: 34
slug: cleanup
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-23
---

# Phase 34 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./... -count=1` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -count=1`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 34-01-01 | 01 | 1 | R056 | — | N/A | manual | `git worktree list \| wc -l` | — | ⬜ pending |
| 34-01-02 | 01 | 1 | R057 | — | N/A | manual | `git branch --list \| wc -l` | — | ⬜ pending |
| 34-02-01 | 02 | 1 | R056 | — | N/A | manual | `git worktree list` | — | ⬜ pending |
| 34-02-02 | 02 | 1 | R057 | — | N/A | manual | `git branch --list` | — | ⬜ pending |
| 34-03-01 | 03 | 2 | R058 | — | N/A | manual | `aether flag-check-blockers` | — | ⬜ pending |
| 34-03-02 | 03 | 2 | — | — | N/A | regression | `go test ./... -race -count=1` | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. This phase uses git CLI commands and existing Go test suite for verification. No new test files needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Worktree count is 1 (main only) | R056 | Git state verification | `git worktree list` — should show only repo root |
| Branch count is ~3 (main + preserve/*) | R057 | Git state verification | `git branch --list` — should show only main + preserve branches |
| All blocker flags reviewed | R058 | Requires user judgment | `aether flag-check-blockers` — count should match user decisions |
| Disk space recovered | R056 | Physical verification | `du -sh .claude/worktrees/` — should be ~0 |

---

## Validation Sign-Off

- [ ] All tasks have verification commands defined
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
