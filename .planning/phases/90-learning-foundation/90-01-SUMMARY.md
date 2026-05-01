---
phase: 90-learning-foundation
plan: 01
subsystem: pkg/learn
tags: [tdd, learning-store, types, interface, crud]
dependency_graph:
  requires: []
  provides: [90-02, 90-03, 91-01]
  affects: []
tech_stack:
  added: []
  patterns: [LearnStore interface, JSON persistence via storage.Store, atomic read-modify-write via UpdateFile]
key_files:
  created:
    - pkg/learn/learn.go
    - pkg/learn/colony_store.go
    - pkg/learn/colony_store_test.go
  modified: []
decisions: []
metrics:
  duration: 251s
  completed_date: 2026-05-01
---

# Phase 90 Plan 01: Learn Store Foundation Summary

LearnStore interface with Entry/Evidence/Classification types and ColonyStore JSON persistence using storage.Store atomic operations.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Define types, interface, failing tests | 6dc7b8f2 | pkg/learn/learn.go, pkg/learn/colony_store_test.go |
| 1 (GREEN) | Implement ColonyStore CRUD + compact + isolation | 5936f2c5 | pkg/learn/colony_store.go, pkg/learn/colony_store_test.go |

## What Was Built

- **pkg/learn/learn.go**: `LearnStore` interface with 6 methods (Add/Get/List/Replace/Remove/Compact), `Entry`, `Evidence`, `WorkerEvidence`, `Classification` (4 constants), `EntryFilter` types with JSON tags and omitempty
- **pkg/learn/colony_store.go**: `ColonyStore` implementing `LearnStore` with JSON persistence via `storage.Store.UpdateFile` for atomic read-modify-write, `storage.Store.LoadJSON` for reads
- **pkg/learn/colony_store_test.go**: 15 tests covering Add (with ID assignment), Get (found/not-found), List (all/phase/classification/min-confidence), Replace, Remove, Compact (budget enforcement), RepoIsolation, UniqueIDs, CompactNoop, error cases

## TDD Gate Compliance

- RED commit (6dc7b8f2): `test(90-01): add failing tests for ColonyStore CRUD, compact, and isolation` -- confirmed tests fail at compile time (ColonyStore undefined)
- GREEN commit (5936f2c5): `feat(90-01): implement ColonyStore with CRUD, compact, and repo isolation` -- all 15 tests pass
- REFACTOR: minor cleanup (removed dead `saveEntries` method, used `filepath.Join`) folded into GREEN commit

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Go pass-by-value in test assertions**
- **Found during:** Task 1 GREEN phase
- **Issue:** Tests created `Entry` structs with empty IDs, called `cs.Add(entry)`, then checked `entry.ID`. Go passes structs by value, so the caller's `entry` is never modified by `Add`. Tests failed with "Add should assign an ID".
- **Fix:** Updated tests to verify ID assignment through persistence (LoadJSON) or List/Get retrieval instead of checking the local variable. Affected tests: TestColonyStoreAdd, TestColonyStoreGet, TestColonyStoreReplace, TestColonyStoreRemove, TestColonyStoreAddUniqueIDs.
- **Files modified:** pkg/learn/colony_store_test.go
- **Commit:** 5936f2c5

**2. [Rule 1 - Bug] Lock conflict in UpdateJSONAtomically**
- **Found during:** Task 1 GREEN phase
- **Issue:** Initial implementation used `UpdateJSONAtomically` which holds a write lock via `UpdateFile`, then called `loadEntries()` inside the mutate closure which calls `LoadJSON` (acquires read lock). This caused a deadlock.
- **Fix:** Replaced with direct `UpdateFile` usage that manually unmarshals `existing []byte` (already read by `UpdateFile`) instead of calling `LoadJSON` inside the lock. Created a `updateEntries` helper that encapsulates the read-modify-write pattern.
- **Files modified:** pkg/learn/colony_store.go
- **Commit:** 5936f2c5

## Verification

- `go test ./pkg/learn/... -v -count=1 -timeout 30s` -- 15/15 tests pass
- `go vet ./pkg/learn/...` -- no issues
- `go build ./cmd/aether` -- pre-existing embedded_assets.go error (unrelated to this plan)
- `grep -c "cobra" pkg/learn/colony_store.go` returns 0 (no cobra imports in pkg/)

## Known Stubs

None.

## Threat Flags

None. No new network endpoints, auth paths, or trust boundary changes introduced.
