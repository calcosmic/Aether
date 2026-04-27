# Phase 62: Lifecycle Ceremony -- Seal and Init - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 62-lifecycle-ceremony-seal-and-init
**Areas discussed:** Seal blocker UX, Init-research depth, CROWNED-ANTHILL format, Promotion ordering

---

## Seal Blocker UX

| Option | Description | Selected |
|--------|-------------|----------|
| Hard stop with summary | Print table of blockers (title, desc, age), exit. User resolves or --force. | ✓ |
| Interactive prompt | Print blockers, ask y/n. Softer but blocks automation. | |
| Warn and continue | Print as warnings, proceed. Only --strict blocks. | |

**User's choice:** Hard stop with summary + resolution command suggestions
**Notes:** User wants actionable output — not just "FAILED" but specific commands to resolve each blocker.

---

## Init-Research Depth

| Option | Description | Selected |
|--------|-------------|----------|
| Structured scan only | README, test framework, CI configs, 2-level dirs. No source reading. | |
| Entry point + architecture hint | Same + read main entry to detect architecture patterns. | |
| Deep scan with source reading | Recursive walk, skip .git/node_modules/vendor, read main entry + top 5 files. | ✓ |

**User's choice:** Deep scan with source reading (walk + top files)
**Notes:** User then provided extensive context about what the old shell-era init used to do — charter, governance detection, pheromone suggestions, complexity metrics. Wants all of this restored.

---

## Init Ceremony Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Rich scan, no approval gate | Deep scan outputs charter, no user approval. | |
| Rich scan + approval ceremony | Charter + approval before colony creation. | |
| Full ceremony with pheromone approval | Charter + pheromone tick-to-approve during init. | ✓ |

**User's choice:** Full ceremony with pheromone approval
**Notes:** User wants the founding document restored. Charter with Intent/Vision/Governance/Goals. Pheromone suggestions from 10 deterministic patterns during init (not build).

---

## Init Ceremony Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Runtime scans, wrapper approves | Go outputs structured data, wrappers handle interaction. | ✓ |
| Runtime does everything | Go handles scan + interactive approval. | |

**User's choice:** Runtime scans, wrapper approves (Claude's recommendation)
**Notes:** Follows established wrapper-runtime contract. Runtime stays portable.

---

## CROWNED-ANTHILL Format

| Option | Description | Selected |
|--------|-------------|----------|
| Summary table only | Counts table at bottom. Machine-parseable. | |
| Table + detail lists | Counts plus instinct names, signal types. Human-readable. | ✓ |

**User's choice:** Table + detail lists
**Notes:** User wants both machine-parseable counts and human-readable detail.

---

## Promotion Ordering

| Option | Description | Selected |
|--------|-------------|----------|
| Auto local, manual global | Repo QUEEN.md auto, global + hive manual. | ✓ |
| Auto local + global, manual hive | Both QUEENs auto, hive manual. | |
| Full auto everywhere | All three auto-promote at seal. | |

**User's choice:** Auto local, manual global
**Notes:** User wants global QUEEN.md to be "more human orchestrated" — not every seal. Seal should log suggestions ("3 instincts eligible for global promotion") but not auto-execute.

---

## Claude's Discretion

- Blocker table formatting (column order, widths)
- Number of deterministic pheromone patterns (old system had 10)
- Charter markdown section structure
- Top-5-files selection heuristic

## Deferred Ideas

- Suggest-analyze during builds (Phase 64)
- Bayesian confidence parity check (separate investigation)
