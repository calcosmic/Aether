---
phase: "138"
plan_id: "138-01"
subsystem: "ts-host"
tags: ["spawning", "types", "claims-parser", "backward-compatibility"]
dependency_graph:
  requires: []
  provides: ["SpawnClaim", "SpawnedWorker", "normalizeSpawnClaims", "WorkerClaims.spawns"]
  affects: ["claims-parser.ts", "worker-dispatch.ts", "types.ts"]
tech_stack:
  added: []
  patterns: ["structured spawn claims", "backward-compatible type evolution", "spawn claim validation"]
key_files:
  created:
    - ".aether/ts-host/test/spawn-claims.test.ts"
  modified:
    - ".aether/ts-host/src/types.ts"
    - ".aether/ts-host/src/claims-parser.ts"
    - ".aether/ts-host/src/worker-dispatch.ts"
decisions:
  - "String spawns default to builder caste during normalization"
  - "Invalid spawn entries filtered with warnings, not causing parse failure"
  - "Spawn claims capped at 50 entries to prevent DoS (T-138-02)"
  - "normalizeSpawnClaims runs automatically inside parseWorkerClaims"
metrics:
  duration: "4m 37s"
  completed: "2026-05-18"
  tasks: 4
  files: 4
---

# Phase 138 Plan 01: Spawn Claim Types and Parsing Summary

Structured spawn claim types and backward-compatible parsing so workers can request sub-workers via `{ caste, task, reason }` objects instead of plain strings.

## Changes Made

### Task 1: SpawnClaim and SpawnedWorker types (types.ts)
- Added `SpawnClaim` interface: `{ caste: string, task: string, reason?: string }`
- Added `SpawnedWorker` interface: `{ claim, name, parent, depth, status, summary?, handoff? }`
- Added `spawns?: SpawnClaim[]` to `WorkerResult` and `BuildDispatch`

### Task 2: Claims parser structured spawn support (claims-parser.ts)
- `WorkerClaims.spawns` changed from `string[]` to `(string | SpawnClaim)[]`
- `validateWorkerClaims` validates SpawnClaim shape, filters invalid entries with stderr warnings
- Added `normalizeSpawnClaims` helper: converts strings to `{ caste: "builder", task: <string>, reason: "auto-converted from string spawn" }`
- `parseWorkerClaims` auto-normalizes spawns on return via internal `normalizeClaimsSpawns`
- Spawn claims capped at 50 entries (DoS mitigation T-138-02)

### Task 3: DispatchResult spawns field (worker-dispatch.ts)
- Added `spawns?: SpawnClaim[]` to `DispatchResult` interface
- `dispatchRealWorker` extracts spawns from parsed claims after normalization
- TODO comment for future extraction once worker output format stabilizes

### Task 4: Spawn claims tests (spawn-claims.test.ts)
- 9 tests across 4 suites: structured parsing (3), backward compatibility (2), validation (2), normalization (2)
- Covers all three output formats: direct JSON, code-fenced, trailing JSON block
- Tests string normalization, mixed arrays, invalid entry filtering with stderr capture

## Test Results

- spawn-claims.test.ts: 9/9 pass
- claims-parser.test.ts: 9/9 pass (no regressions)
- Total: 18/18 pass

## Deviations from Plan

None - plan executed exactly as written.

## Commits

| Commit | Message |
|--------|---------|
| 83b34af4 | feat(138-01): add SpawnClaim and SpawnedWorker types to types.ts |
| 55883167 | feat(138-01): update claims-parser to validate structured spawn claims |
| f56ced41 | feat(138-01): add spawns field to DispatchResult |
| b6366e17 | test(138-01): add spawn claims parsing tests |
