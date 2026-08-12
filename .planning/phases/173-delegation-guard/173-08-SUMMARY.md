---
phase: 173-delegation-guard
plan: 08
subsystem: infra
tags: [spawn-guard, cli-ux, cobra, go, non-technical-readability]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "173-05's spawnTreeBudgetState()/spawnTreeBudgetWarning() reused here for the header/footer so the two views of budget consumption never disagree"
provides:
  - "spawn-tree-active renders a depth-indented, parent-attributed, English view of the live delegation tree, reusing the repo's single caste-identity source (casteIdentity/casteLabel/casteEmoji in cmd/codex_visuals.go)"
  - "The pre-existing JSON payload (active/count, and each entry's name/parent/caste/task/depth/status/spawned_at) is unchanged"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Visual rendering is a pure, read-only function (renderSpawnTreeActiveVisual) taking the already-fetched []agent.SpawnEntry, branched on shouldRenderVisualOutput(stdout) before the JSON-building code -- the JSON path is untouched, not merely equivalent"
    - "Indentation is driven directly by each entry's own recorded Depth field (two spaces per level), not by a tree walk this command would have to re-derive -- depth-ascending then spawn-time-ascending order guarantees byte-identical output across repeated calls over an unchanged tree"

key-files:
  created:
    - cmd/spawn_tree_view_test.go
  modified:
    - cmd/spawn.go

key-decisions:
  - "A parent no longer in the active set (already completed) is still named by its plain recorded string rather than looked up for its own caste identity, since Active() does not return completed entries and there is nothing further to look up -- this is the deliberate T-173-44 mitigation: losing a subtree's attribution because its parent finished first is exactly the runaway blind spot this plan closes."
  - "The header (budget consumption) is skipped entirely if spawnTreeBudgetState() errors, but the footer (the two-limits sentence) always renders, falling back to a number-free sentence on the same error -- an inspection command must not fail outright over a reporting concern."

requirements-completed: []
requirements-in-progress: [SPAWN-07]

# Metrics
duration: in progress
completed: pending
---

# Phase 173 Plan 08: Live, Readable Delegation Tree View Summary (IN PROGRESS -- paused at Task 3 checkpoint)

**`spawn-tree-active` now renders the delegation tree as a depth-indented, parent-attributed English list instead of a flat JSON array, while leaving the JSON contract byte-for-byte unchanged; automated tests prove the rendering works mid-run, contains no raw JSON or status tokens, mutates nothing, and preserves every pre-existing JSON key. This plan is paused at Task 3's mandatory human-verify checkpoint pending operator confirmation that the rendering reads as English.**

## Status

Tasks 1 and 2 are complete and committed. Task 3 (`checkpoint:human-verify`, blocking) is
awaiting the operator's reply. This summary will be completed and re-committed once the
operator responds.

## Task Commits (so far)

1. **Task 1: Render the tree indented by depth, in English, without touching anything** - `f1b5c68d` (feat)
2. **Task 2: Red-proofs for indentation, attribution, mid-run safety and inspection purity** - `cedaa19e` (test)
3. **Task 3: Confirm the tree reads as English to a non-technical operator** - PAUSED, awaiting operator reply

## Accomplishments (Tasks 1-2)

- `spawn-tree-active`'s visual output is built by a new pure function, `renderSpawnTreeActiveVisual`, called from a branch added ahead of the JSON-building code inside `spawnTreeActiveCmd`'s `RunE` -- the JSON path itself was not touched, so the branch is provably additive, not a refactor of shared logic. Verified directly: `AETHER_OUTPUT_MODE=json aether spawn-tree-active` still returns the exact same `active`/`count` shape as before this plan.
- Indentation is two spaces per level of each entry's own recorded `Depth` (D-05's authority), and ordering is depth-ascending then spawn-time-ascending, so two calls over an unchanged tree produce byte-identical output -- proven directly by `TestSpawnTreeActiveMutatesNothing`, which hashes every file in the test store before and after two runs and additionally compares the two runs' stdout byte for byte.
- Every line names the caste in English via the repo's single caste-identity source (`casteIdentity`, composing `casteEmoji` + a colour-wrapped `casteLabel`, both from `cmd/codex_visuals.go`) -- no second caste map, emoji table or colour map was added to `cmd/spawn.go`, confirmed by `grep -nE 'casteEmojiMap|casteLabelMap|casteColorMap' cmd/spawn.go` returning nothing.
- A helper whose parent has already completed (dropped out of the active set) is still attributed to that parent by name, not silently reparented to the root or dropped -- the T-173-44 mitigation the threat register calls for.
- The header ("This run has used N of the M helpers it is allowed to spawn") reuses `spawnTreeBudgetState()` from plan 05's `cmd/spawn_budget.go`, so this view and that command's own warning line can never disagree; a budget-state read failure skips the header only, never the whole command.
- Four named tests in the new `cmd/spawn_tree_view_test.go` prove the four load-bearing claims by execution, not by inspection: `TestSpawnTreeActiveRendersIndentedByDepthMidRun` (indentation and attribution while every entry is still live -- no `spawn-complete` call on any of them), `TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON` (an invariant test: absence of `{`, `}`, `":`, and the raw status tokens `spawned`/`active`/`running`/`completed` as standalone words, presence of the English caste label), `TestSpawnTreeActiveMutatesNothing` (file-hash and output byte-identity across two runs), and `TestSpawnTreeActiveKeepsItsJSONContract` (every pre-existing JSON key still present).
- Both D-22 red-proofs performed and quoted below; both restored with a verified zero-diff `git diff --stat cmd/spawn.go`.

## Files Created/Modified (Tasks 1-2)

- `cmd/spawn.go` - `spawnTreeActiveCmd`'s `RunE` gained a visual-mode branch (calling the new `renderSpawnTreeActiveVisual`) ahead of its unchanged JSON-building code; added `renderSpawnTreeActiveVisual(active []agent.SpawnEntry) string` and `spawnTreeActiveParentPhrase(parent string, byName map[string]agent.SpawnEntry) string`; added `"sort"` to the import block
- `cmd/spawn_tree_view_test.go` (NEW) - four named tests for SPAWN-07 (see Accomplishments), plus shared test helpers `findLineContaining`, `leadingSpaces`, `firstLineWithIndent`, `containsStandaloneWord`, `hashAllFilesForTest`, `diffHashes`, and `suppressFirstRunBanner`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The pre-existing first-run welcome banner (`cmd/ux_firstrun.go`) fired inside every visual-mode test, corrupting both the JSON parsing of setup calls and the mutation-purity assertion**
- **Found during:** Task 2, first run of the new test file
- **Issue:** `checkAndEmitFirstRun` (called from `root.go`'s `PersistentPreRun` on every command) prints a welcome banner and writes a `.welcomed` marker file the first time any command runs in visual mode against a data directory with no `COLONY_STATE.json` and no existing marker. Setting `AETHER_OUTPUT_MODE=visual` for a whole test (needed to force the rendering path in a non-terminal test buffer) caused this banner to print ahead of the setup `spawn-log` calls' JSON output too, breaking `parseEnvelope`, and -- more seriously for `TestSpawnTreeActiveMutatesNothing` -- caused `checkAndEmitFirstRun` to write the `.welcomed` marker file as a side effect of the very first command in the test, which is a real mutation this plan's own purity test is designed to catch, but not one caused by `spawn-tree-active`'s own rendering.
- **Fix:** Added a `suppressFirstRunBanner` test helper that pre-writes the `.welcomed` marker file (matching the pattern `cmd/ux_firstrun_test.go` already uses) immediately after each visual-mode test's store is created, before any command runs. This is test setup, not a change to the command under test.
- **Files modified:** cmd/spawn_tree_view_test.go
- **Verification:** All four new tests pass; `TestSpawnTreeActiveMutatesNothing`'s file-hash comparison holds with the marker file pre-seeded (present, unchanged) rather than appearing mid-test
- **Committed in:** `cedaa19e` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3, blocking issue in test setup)
**Impact on plan:** No file outside `files_modified` (`cmd/spawn.go`, `cmd/spawn_tree_view_test.go`) was touched. The fix is confined to the new test file's own setup.

## Red-Proof Evidence (D-22)

**Red-proof 1 -- changed the indentation width from two spaces to zero (`strings.Repeat("  ", e.Depth)` to `strings.Repeat("", e.Depth)`):**

```
=== RUN   TestSpawnTreeActiveRendersIndentedByDepthMidRun
    spawn_tree_view_test.go:166: H1's indentation (0 spaces) is not exactly two more than W1's (0 spaces):
        This run has used 2 of the 20 helpers it is allowed to spawn.

        Builder W1 is still working -- fix the login form, sent here by Queen
        Watcher H1 is still working -- check the fix, sent here by Builder W1

        In one round, the Queen sends at most a handful of helpers; across the whole run, no more than 20 helpers may ever spawn.
--- FAIL: TestSpawnTreeActiveRendersIndentedByDepthMidRun (0.01s)
```

Flipped to red as required. Restored the two-space repeat; `git diff --stat cmd/spawn.go` showed no output; re-ran and all four tests passed green.

**Red-proof 2 -- replaced the render body with a raw `json.Marshal(active)` dump:**

```
=== RUN   TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON
    spawn_tree_view_test.go:229: visual output contains raw JSON punctuation "{":
        [{"Timestamp":"2026-08-12T21:56:39Z","ActivityTimestamp":"2026-08-12T21:56:39Z","ParentName":"Queen","Caste":"builder","AgentName":"W1","Task":"fix the login form","Depth":1,"Status":"spawned","Summary":""}]
--- FAIL: TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON (0.00s)
```

Flipped to red as required. Restored the real rendering body; `git diff --stat cmd/spawn.go` showed no output; re-ran and all four tests passed green.

## Verification (Tasks 1-2)

- `go build ./cmd/aether` succeeds; `go vet ./...` clean across the whole repo.
- `go test ./cmd -run 'TestSpawnTreeActive|TestSpawn' -count=1 -v` -- all pass.
- `go test ./... -count=1 -timeout 900s` -- full suite passes (exit code 0, no `FAIL`/`panic` lines).
- `AETHER_OUTPUT_MODE=json aether spawn-tree-active` keys unchanged from before this plan.
- `grep -nE 'casteEmojiMap|casteLabelMap|casteColorMap' cmd/spawn.go` returns nothing.

## User Setup Required

None so far.

## Known Stubs

None. Tasks 1-2's own scope (the rendering and its tests) is fully implemented and tested.

## Next Steps

Task 3 is a blocking `checkpoint:human-verify` gate: the operator must read real, pasted
`spawn-tree-active` output and confirm it reads as English (who called whom, what each helper
is doing, nothing unreadable on screen). This plan is paused there. A continuation agent will
resume once the operator replies, complete this summary with the final duration/task count, and
run the state-update/final-commit steps.

---
*Phase: 173-delegation-guard*
*Status: PAUSED at Task 3 (checkpoint:human-verify)*
