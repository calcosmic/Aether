---
phase: 70
slug: self-hosting-cleanup
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-28
---

# Phase 70 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 70-01-01 | 01 | 1 | CLEAN-01 | T-70-01 | No references to removed paths in Go source | unit | `grep -r '\.aether/agents/' cmd/ pkg/ \| wc -l` returns 0 | ✅ W0 | ⬜ pending |
| 70-01-02 | 01 | 1 | CLEAN-02 | T-70-01 | No chamber files tracked | integration | `git ls-files .aether/chambers/ \| wc -l` returns 0 | ✅ W0 | ⬜ pending |
| 70-01-03 | 01 | 1 | CLEAN-03 | T-70-01 | No runtime state files tracked | integration | `git ls-files .aether/CONTEXT.md .aether/CROWNED-ANTHILL.md \| wc -l` returns 0 | ✅ W0 | ⬜ pending |
| 70-01-04 | 01 | 1 | CLEAN-04 | T-70-01 | agents-claude/ byte-identical to .claude/agents/ant/ | integration | `diff -rq .aether/agents-claude/ .claude/agents/ant/` returns nothing | ✅ W0 | ⬜ pending |
| 70-01-05 | 01 | 1 | CLEAN-05 | T-70-01 | .gitignore covers all local-only dirs | integration | `.aether/.gitignore` contains data/, dreams/, midden/, chambers/, etc. | ✅ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements — validation is git operations and file existence checks, no new test stubs needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Verify clean clone has no self-hosting artifacts | CLEAN-01..05 | Git state verification | Clone repo fresh, check `git ls-files .aether/chambers/ .aether/agents/ .aether/data/` all return 0 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
