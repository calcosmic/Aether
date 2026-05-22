---
phase: 152-boundary-parity
plan: 01
subsystem: docs
completed_date: 2026-05-22
duration: "0:20:00"
tags: [boundary, parity, architecture, docs]
dependency_graph:
  requires: []
  provides: [153-01, 154-01, 155-01]
  affects: [.aether/docs/]
tech-stack:
  added: []
  patterns: [plain-reference-doc, markdown-tables, cross-references]
key-files:
  created:
    - .aether/docs/ARCHITECTURE_BOUNDARY.md
    - .aether/docs/PARITY_CLASSIC_VS_GO.md
  modified: []
decisions:
  - "Doc format: plain reference doc markdown (not ADR template) per D-13"
  - "Asset-type matrix uses 11 rows to cover all extraction targets"
  - "Parity checklist uses 16 rows (15 required + 1 bonus) with MATCH/GAP/DEGRADED/INTENTIONALLY_CHANGED status"
  - "Cross-references bidirectional between boundary doc and parity doc"
---

# Phase 152 Plan 01: Boundary & Parity Reference Docs Summary

## One-Liner
Created the two primary reference documents that define the hybrid architecture salvage: the architecture boundary doc (what stays where) and the Classic parity checklist (what correct looks like).

## What Was Done

### Task 1: ARCHITECTURE_BOUNDARY.md
Created `.aether/docs/ARCHITECTURE_BOUNDARY.md` with:
- Opening statement containing the hard rule verbatim: "Compiled code may execute behaviour, but editable assets must define behaviour."
- Three-tier boundary table (Go Runtime Spine, TypeScript Control Plane, Editable Assets, Bash Glue)
- Asset-type matrix with 11 rows covering agents, prompts, phases, playbooks, policies, visuals, ceremony, commands, workers, skills, and events
- Integration points section describing NDJSON events, CLI calls, runtime loading, and `aether update` distribution
- Migration sequence mapping phases 152-159 to asset types
- Decision log referencing D-01 through D-13 from 152-CONTEXT.md
- Cross-references to PARITY_CLASSIC_VS_GO.md, REQUIREMENTS.md, and v1.18 milestone docs

### Task 2: PARITY_CLASSIC_VS_GO.md
Created `.aether/docs/PARITY_CLASSIC_VS_GO.md` with:
- Purpose and scope explaining the v5.4.0 golden reference
- How-to-use guide with decision rule (user decides per item, no blanket winner)
- Parity checklist table with 16 rows covering all required areas: init ceremony, queen loads, worker spawn, planner, oracle loop, scout, build wave, watcher, gatekeeper, probe, memory, lessons, skills, events, recovery, cleanup
- Verification methods section with automated golden/snapshot tests for 9 flagship workflows and manual checklist for edge cases
- Known gaps section listing 6 Classic behaviours intentionally not restored
- Decision log for prior milestone decisions
- Cross-references to ARCHITECTURE_BOUNDARY.md and v1.18 milestone docs

## Verification Results

| Acceptance Criteria | Expected | Actual | Status |
|---------------------|----------|--------|--------|
| ARCHITECTURE_BOUNDARY.md: hard rule count | >= 1 | 1 | PASS |
| ARCHITECTURE_BOUNDARY.md: section count | >= 7 | 7 | PASS |
| ARCHITECTURE_BOUNDARY.md: table row count | >= 20 | 45 | PASS |
| ARCHITECTURE_BOUNDARY.md: cross-reference count | >= 1 | 2 | PASS |
| ARCHITECTURE_BOUNDARY.md: line count | < 300 | 110 | PASS |
| PARITY_CLASSIC_VS_GO.md: status keyword count | >= 1 | 24 | PASS |
| PARITY_CLASSIC_VS_GO.md: table row count | >= 30 | 42 | PASS |
| PARITY_CLASSIC_VS_GO.md: area mention count | >= 15 | 20 | PASS |
| PARITY_CLASSIC_VS_GO.md: boundary cross-reference | >= 1 | 1 | PASS |
| PARITY_CLASSIC_VS_GO.md: v1.18 reference | >= 1 | 5 | PASS |
| PARITY_CLASSIC_VS_GO.md: verification keyword count | >= 1 | 5 | PASS |

## Deviations from Plan

None — plan executed exactly as written.

## Commits

| Commit | Message | Files |
|--------|---------|-------|
| 9f44839c | docs(152-01): create architecture boundary reference document | .aether/docs/ARCHITECTURE_BOUNDARY.md |
| fb620726 | docs(152-01): create Classic parity checklist reference document | .aether/docs/PARITY_CLASSIC_VS_GO.md |

## Self-Check: PASSED

- [x] `.aether/docs/ARCHITECTURE_BOUNDARY.md` exists
- [x] `.aether/docs/PARITY_CLASSIC_VS_GO.md` exists
- [x] Commit 9f44839c exists in git log
- [x] Commit fb620726 exists in git log
- [x] No accidental file deletions
- [x] Pre-existing modifications (`.planning/PROJECT.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `cmd/ceremony_cmd.go`, `cmd/ceremony_cmd_test.go`) were NOT committed
