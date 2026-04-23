---
phase: 44-doc-alignment-and-archive-consistency
plan: 02
subsystem: docs
tags: [medic, integrity, archive, v1.5, consistency]

# Dependency graph
requires:
  - phase: 43
    provides: "scanIntegrity wired into medic --deep, aether integrity command"
provides:
  - "Medic agent documents scanIntegrity behavior in medic --deep"
  - "Medic agent recommends aether publish as primary recovery path"
  - "Medic skill documents aether integrity command with flags"
  - "Medic skill updated remedies with aether publish primary"
  - "v1.5-ROADMAP.md says completed 2026-04-23"
  - "v1.5-REQUIREMENTS.md header says v1.5 not v1.4"
  - "All v1.4-era requirements marked completed (v1.5)"
affects: [operators, medic-workers, release-workflow, future-milestones]

# Tech tracking
tech-stack: [markdown]
files-modified:
  - path: .claude/agents/ant/aether-medic.md
    change: "Updated hub publish recovery to recommend aether publish, added scanIntegrity documentation, added integrity check step to execution flow"
  - path: .aether/skills/colony/medic/SKILL.md
    change: "Added aether integrity command docs, updated remedies to use aether publish primary"
  - path: .planning/milestones/v1.5-ROADMAP.md
    change: "Fixed v1.5 status from in progress to completed 2026-04-23"
  - path: .planning/milestones/v1.5-REQUIREMENTS.md
    change: "Fixed header from v1.4 to v1.5, updated all requirements from active to completed"

key-files:
  created: []
  modified:
    - .claude/agents/ant/aether-medic.md
    - .aether/skills/colony/medic/SKILL.md
    - .planning/milestones/v1.5-ROADMAP.md
    - .planning/milestones/v1.5-REQUIREMENTS.md
---

## Summary

Updated medic agent and skill documentation to reflect release integrity behavior from Phase 43. Fixed v1.5 archived milestone contradictions.

### Changes Made

**Task 1 — Medic docs:**
- Medic agent: Added scanIntegrity() documentation for `medic --deep`, added `aether publish` as primary recovery path, added `aether integrity` step to execution flow
- Medic skill: Documented `aether integrity` command with flags (--json, --channel, --source), updated remedies to lead with `aether publish`, documented medic deep scan integration

**Task 2 — Archive fixes:**
- v1.5-ROADMAP.md: Changed status from "in progress" to "completed 2026-04-23, product v1.0.20"
- v1.5-REQUIREMENTS.md: Fixed header from v1.4 to v1.5, changed all 6 requirements (R039-R044) from active to completed

### Verification

- `grep -c "scanIntegrity" .claude/agents/ant/aether-medic.md` → 1
- `grep -c "aether integrity" .aether/skills/colony/medic/SKILL.md` → 1
- `grep "completed 2026-04-23" .planning/milestones/v1.5-ROADMAP.md` → found
- `head -6 .planning/milestones/v1.5-REQUIREMENTS.md | grep "v1.5"` → found

### Deviations

None — all changes executed as planned.

## Self-Check: PASSED
