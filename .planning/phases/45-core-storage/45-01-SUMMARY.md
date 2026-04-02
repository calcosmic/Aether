---
phase: 45-core-storage
plan: "01"
subsystem: colony
tags: [go-types, golden-tests, round-trip-parity]
dependency_graph:
  requires: [pkg/colony/colony.go, pkg/storage/storage.go]
  provides: [pkg/colony/pheromones.go, pkg/colony/learning.go, pkg/colony/midden.go, pkg/colony/constraints.go, pkg/colony/flags.go, pkg/colony/session.go, pkg/colony/instincts.go, pkg/colony/testdata/*.golden.json]
  affects: [phases 46-51 all read/write colony data through these types]
tech_stack:
  added: [go-encoding-json, json.RawMessage for nested content]
  patterns: [golden-file-testing, typed-structs-over-map-interface, nullable-pointer-fields]
key_files:
  created:
    - pkg/colony/pheromones.go
    - pkg/colony/learning.go
    - pkg/colony/midden.go
    - pkg/colony/constraints.go
    - pkg/colony/flags.go
    - pkg/colony/session.go
    - pkg/colony/instincts.go
    - pkg/colony/testdata/COLONY_STATE.golden.json
    - pkg/colony/testdata/pheromones.golden.json
    - pkg/colony/testdata/learning-observations.golden.json
    - pkg/colony/testdata/midden.golden.json
    - pkg/colony/testdata/session.golden.json
  modified:
    - pkg/colony/colony.go
    - pkg/colony/colony_test.go
decisions:
  - json.RawMessage for pheromone content prevents double-escaping of nested JSON
  - Separate MiddenEntry and MiddenArchivedSignal types (different schemas)
  - []interface{} for InstinctEntry ApplicationHistory/RelatedInstincts (always empty in real data)
  - Empty ConstraintsFile struct for current {} data
metrics:
  duration: 5min
  completed: "2026-04-01"
---

# Phase 45 Plan 01: Colony Data Type Definitions Summary

Typed Go structs for all 7 colony data files with golden file round-trip parity tests proving byte-level compatibility with real shell-produced JSON data.

## Commits

| Hash | Message |
|------|---------|
| e484265 | feat(45-01): add typed Go structs for all colony data files |
| 8c37048 | test(45-01): golden file tests proving byte-level parity with shell data |

## What Was Done

### Task 1: Type Definitions
- ColonyState already had Milestone/MilestoneUpdatedAt fields (confirmed)
- Created 7 new type files in pkg/colony/ with proper snake_case JSON tags
- PheromoneSignal.Content uses json.RawMessage (not string) to preserve nested JSON
- MiddenFile has separate MiddenEntry and MiddenArchivedSignal types
- SessionFile.ResumedAt is *string for null handling
- InstinctsFile covers standalone instincts.json with TrustScore, TrustTier, Provenance, Archived
- No map[string]interface{} in any data file struct

### Task 2: Golden File Tests
- Generated 5 golden fixtures from real .aether/data/ files via Go normalization
- TestGoldenColonyState, TestGoldenPheromones, TestGoldenLearning, TestGoldenMidden, TestGoldenSession
- All golden tests pass byte-for-byte comparison
- Race detector clean

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

- InstinctEntry.ApplicationHistory and RelatedInstincts are []interface{} (always empty in real data; typed structs to be added when real data populates them)
