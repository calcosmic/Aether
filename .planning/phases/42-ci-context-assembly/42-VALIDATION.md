---
phase: 42
slug: ci-context-assembly
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-31
---

# Phase 42 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | bash (bats-core / custom test runner) |
| **Config file** | none — existing test infrastructure |
| **Quick run command** | `bash .aether/aether-utils.sh test-pr-context` |
| **Full suite command** | `bash tests/test-pr-context.sh` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `bash .aether/aether-utils.sh test-pr-context`
- **After every plan wave:** Run `bash tests/test-pr-context.sh`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 42-01-01 | 01 | 1 | CI-01 | unit | `bash tests/test-pr-context.sh` | ❌ W0 | ⬜ pending |
| 42-01-02 | 01 | 1 | CI-01 | unit | `bash tests/test-pr-context.sh` | ❌ W0 | ⬜ pending |
| 42-02-01 | 02 | 1 | CI-02 | unit | `bash tests/test-pr-context.sh` | ❌ W0 | ⬜ pending |
| 42-03-01 | 03 | 2 | CI-03 | unit | `bash tests/test-pr-context.sh` | ❌ W0 | ⬜ pending |
| 42-03-02 | 03 | 2 | CI-03 | integration | `bash tests/test-pr-context.sh` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/test-pr-context.sh` — test harness for CI-01, CI-02, CI-03
- [ ] Budget extraction regression: verify colony-prime output unchanged after `_budget_enforce()` refactor

*Existing infrastructure covers shared test patterns (setup_pheromone_env, tmpdir isolation, AETHER_ROOT override).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| JSON output is valid and parseable | CI-01 | Schema validation | Run `aether pr-context \| jq .` and verify no parse errors |
| Output under character budget | CI-03 | Character counting | Run `aether pr-context \| wc -c` and verify < 6000 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
