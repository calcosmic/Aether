---
phase: 173-delegation-guard
plan: 11
subsystem: agent
tags: [go, spawn-tree, corruption-detection, fail-closed, spawn-budget]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: the spawn ledger (pkg/agent/spawn_tree.go), the whole-run budget (cmd/spawn_budget.go), and 173-VERIFICATION.md's reproduced exploit this plan closes
provides:
  - a spawn ledger parser that returns a distinct, non-nil error (wrapping ErrSpawnTreeCorrupt) for present-but-unparseable content, instead of silently discarding it
  - a run-state reader (spawn-runs.json) that draws the identical absent/unreadable/unparseable distinction
  - write closures (RecordSpawn, updateStatus) that abort on a corrupt ledger instead of silently rewriting a clean file over it
  - four package-level red-proofs covering all of the above
affects: [173-12 (turns these errors into denies in spawnTreeBudgetReason), 173-13 (owns correcting the five now-stale claims named below)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Classify a pipe-delimited line by its TRUE (unbounded) field count before reading any field, never by a bounded SplitN that can merge extra pipes into a trailing field"
    - "A corruption guard must accept every shape its own writer can produce (writer round-trip test) before it is allowed to reject anything"
    - "Three-way file-read contract: absent -> nil error/empty; any other read failure -> non-nil error; present-but-unparseable -> non-nil error wrapping a sentinel"

key-files:
  created: []
  modified:
    - pkg/agent/spawn_tree.go
    - pkg/agent/spawn_tree_test.go
    - cmd/spawn_ancestor_test.go

key-decisions:
  - "The absent-ledger exception is narrowed to exactly three cases (file not found, zero bytes, whitespace-only) -- every other unreadable-or-unparseable outcome is now an error, closing the T-173-50/51/52 gap 173-VERIFICATION.md reproduced"
  - "Classification is shape-only and does not close T-173-67: a well-formed but forged line (e.g. a hand-written valid-looking 4-field line) is still accepted. Closing that needs integrity data this file format does not carry (an append-only log or a signature) and is out of scope for this plan"

requirements-completed: [SPAWN-03]

duration: 55min
completed: 2026-08-13
---

# Phase 173 Plan 11: Corruption Detection at the Spawn Ledger Parse Boundary Summary

**Gave the spawn ledger's parser (and its sibling run-state file) a way to tell "nothing has been written yet" apart from "someone tampered with this file" -- closing the exact hole 173-VERIFICATION.md used to reset the whole-run spawn budget to zero with one shell redirect.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-08-13T15:59:34Z
- **Tasks:** 2/2
- **Files modified:** 3

## Accomplishments

- `pkg/agent/spawn_tree.go`'s `parseFile()` and `loadRunStateLocked()` now return three distinguishable outcomes each (absent/empty, unreadable, unparseable) instead of collapsing all three into "empty, no error"
- `RecordSpawn` and `updateStatus` now abort their write closures on a parse error, so a corrupted `spawn-tree.txt` survives on disk as evidence instead of being silently replaced by a clean file carrying one new entry -- the exact step 173-VERIFICATION.md's reproduction relied on
- Four new tests prove the guard actually holds, including a writer round-trip proving the guard can never reject bytes its own code produced
- Full repository test suite (`go test ./... -count=1 -timeout 900s`) passes unchanged

## Task Commits

Each task was committed atomically:

1. **Task 1: Tell an absent ledger apart from a tampered one** - `d2591c48` (fix)
2. **Task 2: Red-proofs, and prove nothing else depended on the silence** - `743102d9` (test)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## The Rule Set The Parser Now Enforces

`parseSpawnTreeBytes` splits the file on `\n` (not on a trimmed whole-string blob, so line numbers reported in errors match the file's own numbering), trims each line, skips blank ones, and for every remaining line takes `fields := strings.Split(line, "|")` -- an **unbounded** split, taken once, before any field is read. It then switches on `len(fields)`:

| Shape | Field count | Outcome |
|---|---|---|
| Spawn entry (`timestamp\|parent\|caste\|name\|task\|depth\|status`) with a numeric depth | 7 | Parsed as a `SpawnEntry`, exactly as before |
| Spawn entry with a **non-numeric** depth field | 7 | **New: corruption error** (`ErrSpawnTreeCorrupt`, names line number, field count 7). No longer silently skipped, and never retried as a completion line |
| Completion line (`timestamp\|name\|status\|summary`) with a non-empty normalised status | 4 | Parsed and merged onto the matching spawn entry, exactly as before |
| Completion line with an **empty** normalised status | 4 | Ignored WITHOUT error -- unchanged, and required: `updateStatus` legitimately writes this shape whenever its status argument normalises to empty |
| Anything else (1, 2, 3, 5, 6, or 8+ fields) | not 7 or 4 | **New: corruption error**, names line number and the observed field count |

The bounded `strings.SplitN(line, "|", 7)` / `SplitN(line, "|", 4)` pair the old code used as classifiers is gone. That mattered concretely: a 7-field line with a non-numeric depth has 6 pipes, so `SplitN(line, "|", 4)` used to fold everything past the third pipe into one trailing field and let the line "pass" as a bogus 4-field completion record with `fields[2]` landing on the caste field (which normalises non-empty) -- silently accepted, no error. Task 2's `TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces` subtest `a_non-numeric_depth_field_is_corrupt,_never_a_bogus_completion` is the regression lock for this, and red-proof 3 below reproduces the old bug on demand to prove the lock is real.

Every error string names only the 1-based line number and the field count -- never the line's own content, since a spawn-tree line can carry a worker's task description. `TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError`'s `LEDGER-CONTENT-CANARY` assertion fails loudly if anyone later "improves" the error by interpolating the line.

**What this does NOT close (T-173-67, deliberately accepted residue):** classification is by shape only. A well-formed but forged line -- a hand-written 4-field completion line, for instance -- is still accepted, because nothing about it is malformed. An attacker who copies the exact pipe-delimited format can still empty the ledger's *effective* content by writing plausible-looking completion lines. Detecting that needs integrity data this file format does not carry (an append-only log, or a signature), which is out of scope for this plan and remains named risk in the phase's threat register, not something this plan claims to have closed.

## The Run-State Reader's New Three-Way Contract

`loadRunStateLocked()` (backing `spawn-runs.json`) now draws the identical distinction as `parseFile`, on the file that establishes which run a spawn belongs to:

1. **Absent** (`errors.Is(err, fs.ErrNotExist)`) -> empty `spawnRunState{}`, **nil error**. This case had to stay a nil error: a colony that has never begun a run legitimately has no run-state file yet, and `BeginRun` must be able to create the very first one. Making this an error would permanently lock every fresh colony out of spawning -- the same T-173-24 justification that already governs `spawn-tree.txt`'s absent case.
2. **Present but unreadable for any other reason** (a directory at the path, a permission denial, an I/O error) -> **non-nil error** naming `spawn-runs.json`. This is the actual fix: before this plan, the collapsed condition `err != nil || len(strings.TrimSpace(string(data))) == 0` turned every read failure into "empty run history, no error" -- indistinguishable from a legitimately fresh colony.
3. **Present, empty or whitespace-only content** -> empty state, nil error. Unchanged.
4. **Present, content that fails `json.Unmarshal`** -> non-nil error. Unchanged -- this was already the one path that DID error before this plan, and the new test suite proves it was not disturbed.

## Red-Proof Transcripts

All four were performed by mutating `pkg/agent/spawn_tree.go` in place, running the named test to confirm it fails by name, then restoring the file from a pre-mutation backup and re-confirming the test (and the full package suite) passes again. Each cycle used `go build ./...` to confirm the mutation still compiled before testing.

### 1. `parseFile`'s corrupt branch reverted to a nil error

**Mutation:** changed the corrupt-content branch of `parseFile` from `return nil, nil, fmt.Errorf("spawn_tree: %s: %w", st.filePath, err)` to `return nil, nil, nil`.

**Result:**
```
=== RUN   TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError/corrupt
    spawn_tree_test.go:753: Parse() error = nil, want a corruption error
--- FAIL: TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError (0.00s)
    --- FAIL: TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError/corrupt (0.00s)
```
Restored; `go test ./pkg/agent -run TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError -count=1` passes again.

### 2. `RecordSpawn`'s closure reverted to ignore the parse error

**Mutation:** changed `entries, completions, err := parseSpawnTreeBytes(existing); if err != nil { return nil, err }` inside `RecordSpawn`'s `UpdateFile` closure to `entries, completions, _ := parseSpawnTreeBytes(existing)`.

**Result:**
```
=== RUN   TestSpawnTreeRefusesToRewriteACorruptLedger/RecordSpawn
    spawn_tree_test.go:894: RecordSpawn() error = nil, want non-nil against a corrupt ledger
--- FAIL: TestSpawnTreeRefusesToRewriteACorruptLedger (0.00s)
    --- FAIL: TestSpawnTreeRefusesToRewriteACorruptLedger/RecordSpawn (0.00s)
```
(The sibling `UpdateStatus` subtest still passed, since only `RecordSpawn`'s closure was mutated -- proof the two closures are independently guarded.) Restored; both subtests pass again.

### 3. The true-field-count switch replaced with the old bounded `SplitN(line, "|", 4)` fallback

**Mutation:** replaced the `switch len(fields)` classifier with the pre-plan two-step `SplitN(line, "|", 7)` then `SplitN(line, "|", 4)` fallback logic (byte-for-byte the old algorithm).

**Result:**
```
=== RUN   TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces/a_non-numeric_depth_field_is_corrupt,_never_a_bogus_completion
    spawn_tree_test.go:857: parseSpawnTreeBytes() error = nil, want a corruption error for a non-numeric depth field
--- FAIL: TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces (0.00s)
    --- FAIL: TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces/a_non-numeric_depth_field_is_corrupt,_never_a_bogus_completion (0.00s)
```
This is the exact "bounded SplitN accepts a corrupt line as a bogus completion" trap the plan's `<interfaces>` section warned about, reproduced on demand. Restored; all three subtests pass again.

### 4. `loadRunStateLocked`'s collapsed condition restored

**Mutation:** replaced the split three-way branch with the original `if err != nil || len(strings.TrimSpace(string(data))) == 0 { return spawnRunState{}, nil }`.

**Result:**
```
=== RUN   TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne/a_directory_obstructing_spawn-runs.json's_path_is_an_error,_not_an_empty_history
    spawn_tree_test.go:977: CurrentRun() error = nil, want non-nil when spawn-runs.json's path is obstructed by a directory
--- FAIL: TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne (0.01s)
    --- FAIL: TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne/a_directory_obstructing_spawn-runs.json's_path_is_an_error,_not_an_empty_history (0.00s)
```
(The absent-file and invalid-JSON subtests still passed, confirming only the directory-obstruction case depended on the fix.) Restored; all three subtests pass again, and `git diff --stat pkg/agent/spawn_tree.go` against the Task 1 commit showed zero diff after the final restore, confirming the file returned to exactly its committed state.

## Pre-Existing Fixtures Corrected As Fallout

**One comment corrected**, no assertions or fixture bytes changed:

- `cmd/spawn_ancestor_test.go` (~line 229, inside `TestSpawnAncestorCheckFailsClosedOnUnreadableTree`): the old comment said a non-numeric depth field "is skipped entirely by parseSpawnTreeBytes, making A1 an unresolvable name." That claim is now false -- the whole ledger fails to parse instead. Rewritten to say the ledger no longer parses at all and the ancestor chain is therefore unreadable. `git diff cmd/spawn_ancestor_test.go` confirms comment-only lines changed.

## Three Malformed-Ledger Fixtures In `cmd` Checked And Deliberately Left Alone

All three write the identical fixture, `not-a-timestamp|Queen|builder|A1|...|not-a-number|spawned` -- a 7-field line with a non-numeric depth. Before this plan, `parseSpawnTreeBytes` silently skipped this line (dropping A1 from the parsed results); after this plan, it makes the whole `parseFile()` call return a non-nil error wrapping `ErrSpawnTreeCorrupt`. All three tests were re-run as part of the full suite and still pass, each reaching a deny through a **different** fail-closed branch than before, since the *reason* the entry is now missing changed (a hard parse error rather than a silent drop), but the observable deny behavior did not:

1. **`cmd/spawn_ancestor_test.go`** (`TestSpawnAncestorCheckFailsClosedOnUnreadableTree`, ~line 235): `spawnAncestorChain`'s `st.Parse()` call now returns the corruption error directly; `spawnAncestorCycleReason` wraps it as `"ancestor chain unreadable (...)"` -- same deny message shape as before, now reached via a real parse error instead of a "no recorded spawn entry" lookup miss.
2. **`cmd/spawn_failclosed_test.go`** (`requester-not-recorded` axis, ~line 314, under the `spawn-can-spawn` command's fault table): same path as above, reached via `--name A1` after A1 is corrupted out of the tree. Deny message still contains `"ancestor chain unreadable"`.
3. **`cmd/spawn_failclosed_test.go`** (`parent-not-recorded` axis, ~line 402, under the `spawn-log` command's fault table): a `--parent` naming the now-unparseable A1 reaches `deriveSpawnDepth`'s "unknown parent" branch (`cmd/spawn.go:230-237`), because `latestSpawnEntryByName` returns nil when `Parse()` errors. D-09 (a refused spawn leaves no trace) is separately asserted by this axis and still holds.

## Five Stale Claims Named For Plan 13 (Not Edited Here)

Plan 13 owns `cmd/spawn_failclosed_test.go` and `cmd/internal_cmds.go`; this plan does not touch either file's logic or the four remaining stale comments below, per the plan's explicit scope boundary. All five statements claimed `agent.SpawnTree.Parse()` "can never itself return a non-nil error" -- a claim this plan makes false by construction:

1. **`cmd/spawn_failclosed_test.go`, lines ~27-39** -- the file's header "Bounded residue" paragraph: *"Parse() itself can therefore never return a non-nil error, and no byte content exists that makes it do so."*
2. **`cmd/spawn_failclosed_test.go`, ~line 104** -- `TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree`'s doc comment: *"agent.SpawnTree.Parse() cannot itself be made to return an error."*
3. **`cmd/spawn_failclosed_test.go`, ~line 223** -- the `NotExercisable` field's doc comment on `delegationGuardFaultAxis`: *"agent.SpawnTree.Parse() can never itself return an error -- see the bounded-residue note atop this file."* (Note: this field is declared but has zero live usages in the current fault table -- `grep -n "NotExercisable:"` returns nothing -- so no test behavior currently depends on this claim, but the comment should still be corrected or the now-unused field removed.)
4. **`cmd/spawn_failclosed_test.go`, ~line 489** -- the `spawn-tree-unreadable` axis comment: *"Parse() itself cannot be made to error (see the bounded-residue note atop this file), so this axis is exercisable via the directory route, not via Parse() rejecting content."*
5. **`cmd/internal_cmds.go`, lines ~573-577** -- `spawnCanSpawnSwarmCmd`'s comment justifying its pre-`Parse()` `os.Stat` check: *"agent.SpawnTree.Parse() swallows a missing file as 'empty' (nil error) exactly like an unreadable one -- it cannot tell the two apart."* Beyond correcting the comment, plan 13 should evaluate whether the manual `os.Stat` pre-check this comment justifies (lines 578-597) is now partially redundant with `Parse()`'s own new error return, and whether simplifying it is in scope.

## Files Created/Modified

- `pkg/agent/spawn_tree.go` -- `ErrSpawnTreeCorrupt` sentinel; three-way `parseFile`/`parseSpawnTreeBytes`/`loadRunStateLocked` contract; write-closure aborts in `RecordSpawn`/`updateStatus`; updated doc comments on `SpawnTree`, `NewSpawnTree`, `Active`
- `pkg/agent/spawn_tree_test.go` -- four new tests: `TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError`, `TestSpawnTreeParseAcceptsEveryShapeTheWriterProduces`, `TestSpawnTreeRefusesToRewriteACorruptLedger`, `TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne`
- `cmd/spawn_ancestor_test.go` -- one comment corrected (no assertion or fixture-byte change)

## Decisions Made

- **The absent-ledger exception is now narrowed to exactly three cases**: file not found, zero bytes, whitespace-only content. Everything else that was previously silently accepted as "empty" is now an error. This is stated explicitly because plan 12's deny logic depends on this exact narrowing being complete -- any fourth silently-tolerated case would reopen the same class of gap.
- **T-173-67 (a well-formed but forged line) is explicitly NOT closed by this plan.** Shape-based classification bounds the specific, demonstrated exploit (garbage/malformed bytes resetting the budget), not forgery in general. This is recorded as accepted residue in the phase's threat register, not implied closed.

## Deviations from Plan

None -- plan executed exactly as written. All acceptance criteria greps, both automated verification commands, the four red-proofs, and the full repository test suite all passed as specified.

## Verification

- `go build ./...` -- exits 0
- `go vet ./pkg/agent` -- exits 0
- `go test ./pkg/agent -count=1` -- passes, including all four new tests
- `go test ./... -count=1 -timeout 900s` -- exits 0 (all 18 packages, including `cmd` at 277-284s)
- All acceptance-criteria greps from both tasks' `<acceptance_criteria>` sections passed
- All four red-proofs performed, confirmed failing by name, and restored (transcripts above)

## Known Stubs

None.

## Threat Flags

None -- this plan closes threats already named in the phase's own threat register (T-173-50, T-173-51, T-173-52, T-173-53, T-173-54, T-173-66, T-173-69) and does not introduce new network endpoints, auth paths, or schema changes at a trust boundary.
