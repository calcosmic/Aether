# Phase 102: Worker Economy & Visual Ceremony Audit - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-07
**Phase:** 102-worker-economy-visual-ceremony-audit
**Areas discussed:** Worker audit scope, Visual ceremony boundary, Wave shape documentation, Audit output format

---

## Worker Audit Scope

### Which castes to audit

| Option | Description | Selected |
|--------|-------------|----------|
| All 27 castes | Catalog all 27, flag unused as findings. Complete picture. | ✓ |
| Active-only (~15) | Only audit castes in dispatch paths. Faster but incomplete. | |
| Split report | All 27 but separated into Active and Defined sections. | |

**User's choice:** All 27 castes
**Notes:** Flag any that are never spawned as 'unused'

### What to verify per caste

| Option | Description | Selected |
|--------|-------------|----------|
| Purpose + output + consumer | Simple table: purpose, durable output, downstream consumer | ✓ |
| Add frequency + cost | Add spawn frequency and context spend assessment | |
| You decide | Right depth to catch unjustified castes | |

**User's choice:** Purpose + output + consumer

### Chat-only worker handling

| Option | Description | Selected |
|--------|-------------|----------|
| Flag as finding | Flag any caste with no durable output | ✓ |
| Flag + suggest pattern | Also suggest conversion patterns | |
| Only flag unjustified | Only flag ones that SHOULD have durable output | |

**User's choice:** Flag as finding

---

## Visual Ceremony Boundary

### Which visual elements count as ceremony

| Option | Description | Selected |
|--------|-------------|----------|
| 3 elements | Caste identity, stage markers, closeout banners | |
| All 5 elements | Add progress bars and Aether wordmark | ✓ |
| You decide | Focus on what could mislead | |

**User's choice:** All 5 elements

### Pure decoration policy

| Option | Description | Selected |
|--------|-------------|----------|
| Allow pure decoration | Wordmark is fine; only flag fake state transitions | ✓ |
| No pure decoration | Every element must trace to runtime state | |

**User's choice:** Allow pure decoration

---

## Wave Shape Documentation

### Documentation format

| Option | Description | Selected |
|--------|-------------|----------|
| Per-command tables | One table per command showing spawn/why/produce | ✓ |
| Unified matrix | Rows=castes, columns=commands matrix | |
| Both formats | Detail tables plus summary matrix | |

**User's choice:** Per-command tables

### Which commands to document

| Option | Description | Selected |
|--------|-------------|----------|
| 5 commands | build, continue, seal, colonize, plan (per WORK-03) | ✓ |
| 8 commands | Add run, swarm, watch | |
| You decide | Whatever makes sense | |

**User's choice:** 5 commands

---

## Audit Output Format

### Output artifacts

| Option | Description | Selected |
|--------|-------------|----------|
| Report + tests | WORKER-ECONOMY.md + spawn coverage test (like Phase 101) | ✓ |
| Per-caste contracts | One doc per caste with inputs/outputs/mutations | |
| You decide | Most useful for Phase 105 | |

**User's choice:** Report + tests

### Report structure

| Option | Description | Selected |
|--------|-------------|----------|
| Single combined report | One WORKER-ECONOMY.md with worker + visual + wave sections | ✓ |
| Two separate reports | WORKER-ECONOMY.md and VISUAL-CEREMONY.md | |

**User's choice:** Single combined report

### Test scope

| Option | Description | Selected |
|--------|-------------|----------|
| Spawn coverage test | Verify every spawned caste has documented purpose/output | ✓ |
| Add ceremony wiring test | Check visual functions map to real state transitions | |
| Both tests | Spawn coverage + ceremony wiring | |

**User's choice:** Spawn coverage test

---

## Claude's Discretion

- Exact test file structure and naming
- How to extract caste spawn sites from dispatch code
- Whether to include spawn frequency data
- Visual ceremony verification method
- Report section ordering and formatting details

## Deferred Ideas

None — discussion stayed within phase scope.
