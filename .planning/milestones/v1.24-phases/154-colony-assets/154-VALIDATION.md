---
phase: 154
slug: colony-assets
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-23
---

# Phase 154 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest |
| **Config file** | `control-ts/vitest.config.ts` |
| **Quick run command** | `npm run test:schemas` |
| **Full suite command** | `npm run test` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `npm run test:schemas`
- **After every plan wave:** Run `npm run test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 154-01-01 | 01 | 1 | EXTRACT-01 | — | Agent YAML valid per schema | schema | `npm run test:schemas` | ❌ W0 | ⬜ pending |
| 154-01-02 | 01 | 1 | EXTRACT-01 | — | All 27 agent YAMLs exist | file-check | `ls colony/agents/*.yaml` | ❌ W0 | ⬜ pending |
| 154-01-03 | 01 | 1 | EXTRACT-02 | — | Prompt files exist per schema | file-check | `ls colony/prompts/*.md` | ❌ W0 | ⬜ pending |
| 154-02-01 | 02 | 2 | EXTRACT-03 | — | Phase YAMLs valid per schema | schema | `npm run test:schemas` | ❌ W0 | ⬜ pending |
| 154-02-02 | 02 | 2 | EXTRACT-04 | — | Playbook files exist | file-check | `ls colony/playbooks/*.md` | ❌ W0 | ⬜ pending |
| 154-03-01 | 03 | 3 | EXTRACT-05 | — | Model-routing policy valid | schema | `npm run test:schemas` | ❌ W0 | ⬜ pending |
| 154-03-02 | 03 | 3 | EXTRACT-06 | — | Memory-rules policy valid | schema | `npm run test:schemas` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `control-ts/src/schemas/agent.schema.ts` — role enum expanded to 27 castes
- [ ] `control-ts/tests/fixtures/agents/` — sample YAML for all 27 castes
- [ ] `control-ts/tests/fixtures/phases/` — sample YAML for lifecycle phases
- [ ] `control-ts/tests/fixtures/playbooks/` — sample Markdown for playbooks
- [ ] `control-ts/tests/fixtures/policies/` — sample YAML for policies
- [ ] `colony/` directory created with `agents/`, `prompts/`, `phases/`, `playbooks/`, `policies/` subdirectories

*Wave 0 creates the directory structure and expands the agent schema to accept all 27 castes before any asset files are written.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Prompt completeness | EXTRACT-02 | Text quality review | Spot-check 3 prompt files for completeness against existing `.claude/agents/ant/*.md` sources |
| Playbook accuracy | EXTRACT-04 | Text quality review | Spot-check 2 playbook files against existing `.aether/docs/command-playbooks/*.md` sources |
| No hardcoded behaviour in Go | EXTRACT-06 | Static analysis | Run `grep -r "builder\|watcher\|scout" cmd/*.go` and verify no prompt text remains |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
