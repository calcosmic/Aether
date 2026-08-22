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
  - "Operator-verified readability: a human read real, pasted spawn-tree-active output and confirmed it names who called whom, what each helper is doing, and contains nothing unreadable"
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
  - "Task 3's checkpoint fixture was a manual stand-in, not a genuine concurrent build: three live spawn-tree entries were created with the real, production spawn-log command (Queen -> Mason-42 builder, Queen -> Sentry-7 watcher, Mason-42 -> Tester-3 probe), none completed, then the real spawn-tree-active command from Task 1 was run against them. A single execution agent in this worktree cannot literally dispatch a concurrent multi-agent Aether build; the rendering code path exercised is identical to what a genuine build would produce, only the origin of the underlying spawn records differs. This was disclosed to the operator at checkpoint time and is recorded here so the claim does not silently expand into 'verified against a live build' later."

requirements-completed: [SPAWN-07]

# Metrics
duration: ~50min active work (execution paused between Task 2 and the Task 3 checkpoint pending operator reply)
completed: 2026-08-13
---

# Phase 173 Plan 08: Live, Readable Delegation Tree View Summary

**`spawn-tree-active` now renders the delegation tree as a depth-indented, parent-attributed English list instead of a flat JSON array, while leaving the JSON contract byte-for-byte unchanged; automated tests prove the rendering works mid-run, contains no raw JSON or status tokens, mutates nothing, and preserves every pre-existing JSON key. The operator read real, pasted output and confirmed it is readable with no follow-up questions.**

## Performance

- **Duration:** ~50 min of active work (a real-time pause separated Task 2 from the operator's Task 3 checkpoint reply; that pause is not counted as work time)
- **Tasks:** 3 completed
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- `spawn-tree-active`'s visual output is built by a new pure function, `renderSpawnTreeActiveVisual`, called from a branch added ahead of the JSON-building code inside `spawnTreeActiveCmd`'s `RunE` -- the JSON path itself was not touched, so the branch is provably additive, not a refactor of shared logic. Verified directly: `AETHER_OUTPUT_MODE=json aether spawn-tree-active` still returns the exact same `active`/`count` shape as before this plan.
- Indentation is two spaces per level of each entry's own recorded `Depth` (D-05's authority), and ordering is depth-ascending then spawn-time-ascending, so two calls over an unchanged tree produce byte-identical output -- proven directly by `TestSpawnTreeActiveMutatesNothing`, which hashes every file in the test store before and after two runs and additionally compares the two runs' stdout byte for byte.
- Every line names the caste in English via the repo's single caste-identity source (`casteIdentity`, composing `casteEmoji` + a colour-wrapped `casteLabel`, both from `cmd/codex_visuals.go`) -- no second caste map, emoji table or colour map was added to `cmd/spawn.go`, confirmed by `grep -nE 'casteEmojiMap|casteLabelMap|casteColorMap' cmd/spawn.go` returning nothing.
- A helper whose parent has already completed (dropped out of the active set) is still attributed to that parent by name, not silently reparented to the root or dropped -- the T-173-44 mitigation the threat register calls for.
- The header ("This run has used N of the M helpers it is allowed to spawn") reuses `spawnTreeBudgetState()` from plan 05's `cmd/spawn_budget.go`, so this view and that command's own warning line can never disagree; a budget-state read failure skips the header only, never the whole command.
- Four named tests in the new `cmd/spawn_tree_view_test.go` prove the four load-bearing claims by execution, not by inspection: `TestSpawnTreeActiveRendersIndentedByDepthMidRun` (indentation and attribution while every entry is still live -- no `spawn-complete` call on any of them), `TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON` (an invariant test: absence of `{`, `}`, `":`, and the raw status tokens `spawned`/`active`/`running`/`completed` as standalone words, presence of the English caste label), `TestSpawnTreeActiveMutatesNothing` (file-hash and output byte-identity across two runs), and `TestSpawnTreeActiveKeepsItsJSONContract` (every pre-existing JSON key still present).
- Both D-22 red-proofs performed and quoted below; both restored with a verified zero-diff `git diff --stat cmd/spawn.go`.
- The operator read real, verbatim `spawn-tree-active` output (see Task 3 below) and replied "approved" with no readability issues raised -- the one requirement no automated test can check.

## Task Commits

Each task was committed atomically:

1. **Task 1: Render the tree indented by depth, in English, without touching anything** - `f1b5c68d` (feat)
2. **Task 2: Red-proofs for indentation, attribution, mid-run safety and inspection purity** - `cedaa19e` (test)
3. **(interim) Partial summary before the Task 3 checkpoint** - `d05f3e45` (docs)
4. **Task 3: Confirm the tree reads as English to a non-technical operator** - no file changes (the task's own scope is verification only); operator approval recorded below

**Plan metadata:** _pending_ (docs: complete plan -- added by the orchestrator after all worktree agents in this wave merge)

## Files Created/Modified

- `cmd/spawn.go` - `spawnTreeActiveCmd`'s `RunE` gained a visual-mode branch (calling the new `renderSpawnTreeActiveVisual`) ahead of its unchanged JSON-building code; added `renderSpawnTreeActiveVisual(active []agent.SpawnEntry) string` and `spawnTreeActiveParentPhrase(parent string, byName map[string]agent.SpawnEntry) string`; added `"sort"` to the import block
- `cmd/spawn_tree_view_test.go` (NEW) - four named tests for SPAWN-07 (see Accomplishments), plus shared test helpers `findLineContaining`, `leadingSpaces`, `firstLineWithIndent`, `containsStandaloneWord`, `hashAllFilesForTest`, `diffHashes`, and `suppressFirstRunBanner`

## Task 3: Operator Verification

**What was shown:** three live helper entries were recorded with the real, production `spawn-log` command (the same code path a genuine build dispatch uses) -- Queen sent a builder (`Mason-42`, "Implement the CSV export endpoint") and a watcher (`Sentry-7`, "Run the test suite and confirm nothing regressed"); the builder in turn sent a probe (`Tester-3`, "Check coverage on the new export code path"). None of the three were marked complete. `AETHER_FORCE_COLOR=1 aether spawn-tree-active` was then run against that live tree and its exact output was pasted to the operator.

**Disclosed limitation:** a single execution agent running inside this worktree cannot literally dispatch a concurrent multi-agent Aether build (the plan's literal instruction). The stand-in above uses the exact same production `spawn-log` and `spawn-tree-active` commands a real build would use; only the origin of the three spawn records (manually issued rather than emitted by concurrent live workers) differs from the plan's literal wording. This was stated to the operator before they answered, and is recorded here as a key decision so it is not later read as "verified against a live build."

**Output shown to the operator:**

```
This run has used 0 of the 20 helpers it is allowed to spawn.

  Builder Mason-42 is still working -- Implement the CSV export endpoint, sent here by Queen
  Watcher Sentry-7 is still working -- Run the test suite and confirm nothing regressed, sent here by Queen
    Probe Tester-3 is still working -- Check coverage on the new export code path, sent here by Builder Mason-42

In one round, the Queen sends at most a handful of helpers; across the whole run, no more than 20 helpers may ever spawn.
```

(In a real terminal, "Builder", "Watcher" and "Probe" render in colour via ANSI escape codes, shown here as plain text.)

**Operator's reply:** "approved" -- read the output and raised no readability issues. All three questions (who called whom, what each is doing, anything unreadable on screen) were answered affirmatively/negatively as required for approval.

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

## Verification

- `go build ./cmd/aether` succeeds; `go vet ./...` clean across the whole repo.
- `go test ./cmd -run 'TestSpawnTreeActive|TestSpawn' -count=1 -v` -- all pass.
- `go test ./... -count=1 -timeout 900s` -- full suite passes (exit code 0, no `FAIL`/`panic` lines).
- `AETHER_OUTPUT_MODE=json aether spawn-tree-active` keys unchanged from before this plan.
- `grep -nE 'casteEmojiMap|casteLabelMap|casteColorMap' cmd/spawn.go` returns nothing.
- Operator read real, pasted `spawn-tree-active` output and replied "approved" with no readability issues raised.

## Issues Encountered

None beyond the deviation documented above.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

- No blockers for other 173-* plans. This plan touched only `cmd/spawn.go` and the new `cmd/spawn_tree_view_test.go`, both scoped to this plan's `files_modified`.
- Plan 10 (final wiring/marker cleanup, per prior-plan summaries) has no new markers or stubs from this plan to clean up -- this plan introduced no `CONTRACT-STUB` markers.

## Known Stubs

None. This plan's own scope (the rendering and its tests) is fully implemented, tested, and operator-verified. See the Task 3 section above for the one disclosed, bounded limitation (a manual spawn-log stand-in rather than a genuine concurrent build) -- not a stub in the shipped code, but a limitation of how the checkpoint itself was exercised.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-13*

## Self-Check: PASSED

- FOUND: cmd/spawn.go
- FOUND: cmd/spawn_tree_view_test.go
- FOUND: .planning/phases/173-delegation-guard/173-08-SUMMARY.md
- FOUND commit f1b5c68d (Task 1)
- FOUND commit cedaa19e (Task 2)
- FOUND commit d05f3e45 (interim partial summary)
