---
phase: 173-delegation-guard
plan: 09
subsystem: infra
tags: [spawn-guard, reaper, budget, cobra, cli, go]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: "173-05's whole-run helper budget (spawnTreeBudgetState/spawnTreeBudgetMax) and its abandoned-status literal placeholder, which this plan replaces with the real constant"
provides:
  - "agent.SpawnStatusAbandoned: a terminal, non-live status a reaper applies to a spawn-tree entry that has gone quiet past a configured elapsed time with no completion reported (D-15..D-18)"
  - "colony.ColonyState.SpawnReapThresholdMinutes: the one configurable knob in this phase, defaulting to 120 minutes when unset (D-17)"
  - "cmd/spawn_reap.go: spawnReapCandidates (read-only scan), spawnReapStaleEntries (the mutation, via UpdateStatusPreserveActivity so a reap never poisons the run-window budget count), and the spawn-orphans command (list, or --clear to reap)"
  - "beginRuntimeSpawnRun now reaps automatically at the start of every run, before any spawn decision consults the budget (D-15's automatic half)"
  - "cmd/spawn_budget.go's consumed-budget count now excludes agent.SpawnStatusAbandoned via the real constant, closing D-18's budget-release half"
affects: [173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A reap/clear command pair follows cmd/spawn.go's existing list-command + mutate-command shape (spawnTreeActiveCmd/spawnCompleteCmd), not a single combined command: spawn-orphans without --clear is read-only, --clear is the mutation, both share the same underlying scan function"
    - "The LATER of two candidate timestamps (spawn Timestamp vs. refreshed ActivityTimestamp) is the only evidence of life this system has, and staleness is judged against it, never the earlier one -- proven by a dedicated red-proof rather than left as an implicit implementation detail"
    - "A dual JSON/English-text output command routes its visual branch through writeVisualOutput and its JSON branch through a raw envelope write exempted in visualWriterExemptions, matching unblock_cmd.go's established pattern rather than inventing a third output shape"

key-files:
  created:
    - cmd/spawn_reap.go
    - cmd/spawn_reap_test.go
  modified:
    - pkg/agent/spawn_tree.go
    - pkg/agent/spawn_tree_test.go
    - pkg/colony/colony.go
    - pkg/colony/colony_test.go
    - cmd/spawn_runs.go
    - cmd/spawn_budget.go
    - .claude/commands/ant/patrol.md
    - .claude/commands/ant-patrol.md
    - .opencode/commands/ant/patrol.md
    - cmd/visual_writer_discipline_test.go
    - cmd/command_call_audit_test.go
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "The reaper scans the WHOLE spawn tree (agent.SpawnTree.Parse(), every entry ever recorded), not just the current run's window. D-15 does not restrict reaping to the active run, and a ghost from an earlier, never-completed run is exactly the kind of thing the operator's spawn-orphans command should be able to see and clear, not something scoped invisible by run-window filtering."
  - "spawn-orphans was wired into the patrol health-check flow (.claude/commands/ant/patrol.md and its OpenCode mirror) rather than given its own new slash command, because SPAWN-08's own framing is a health check (\"is anything stuck?\"), and CLAUDE.md's Definition of Done requires a registered command to have a real caller, not a standalone command nobody invokes yet."
  - "spawn-orphans is classified as D-01 enrichment, not a gate, in cmd/command_call_audit_test.go's severity classification: its failure degrades visibility into ghost helpers only, and no verification result, security scan, or gate outcome depends on it -- the same tier as spawn-log/spawn-complete/spawn-can-spawn."
  - "Test fixtures write stale-timestamped entries directly into spawn-tree.txt's on-disk pipe format rather than adding a clock-injection seam to SpawnTree, per the plan's explicit instruction: RecordSpawn calls time.Now().UTC() directly, and a fixture that writes the file is smaller and does not widen the production surface."
  - "Test files (pkg/agent/spawn_tree_test.go, pkg/colony/colony_test.go) were touched even though the plan's own files_modified frontmatter omitted them; Task 1's own action text explicitly required tests in both packages, and both files sit alongside code files already in scope. Documented here rather than silently expanding scope."

patterns-established:
  - "A guard status that stops counting against a budget (abandoned) is registered exactly where the two existing status classifiers already live (IsTerminalSpawnStatus/IsLiveSpawnStatus in pkg/agent/spawn_tree.go), not as a parallel check elsewhere -- future terminal statuses should follow the same single point of registration."

requirements-completed: [SPAWN-08]

# Metrics
duration: ~40min
completed: 2026-08-13
---

# Phase 173 Plan 09: Reap Abandoned Helpers Summary

**A generous, wall-clock-only reaper (default 120 minutes, configurable via `colony.ColonyState.SpawnReapThresholdMinutes`) marks a quiet spawn-tree entry `abandoned` and gives its whole-run budget slot back, running automatically at the start of every run and on demand via `aether spawn-orphans` / `aether spawn-orphans --clear`.**

## Performance

- **Duration:** ~40 min
- **Tasks:** 3 completed
- **Files modified:** 15 (2 created, 13 modified)

## Accomplishments

- `agent.SpawnStatusAbandoned` is registered as terminal and explicitly not live, with a doc comment stating exactly and only what it means: a configured elapsed time passed with no completion reported. It does not, and cannot, claim the helper is confirmed dead — this system has no liveness signal, and every comment, output string, and test name in this plan was written to avoid the word "detects" applied to abandonment.
- `colony.ColonyState.SpawnReapThresholdMinutes` (a `*int`, `omitempty`) is the one configurable knob in this phase (D-17), defaulting to 120 minutes when unset or invalid — proven to round-trip through JSON and to load `nil` for a state file written before this field existed (`TestSpawnReapThresholdMinutesRoundTrips`, `TestSpawnReapThresholdMinutesNilWhenAbsent`).
- The reaper compares against the LATER of a spawn entry's original `Timestamp` and its refreshed `ActivityTimestamp`, never the earlier — proven directly by `TestSpawnReapRespectsRefreshedActivity`, the test that stops the reaper from killing live work: an entry with a 5-hour-old spawn timestamp but a 10-minute-old activity refresh is spared, because the refresh is the only evidence of life this system has.
- Reaping uses `UpdateStatusPreserveActivity`, never the non-preserving `UpdateStatus` — a reap must not look like fresh activity, which would move the entry back inside the run-window filter and corrupt the whole-run budget count. `cmd/spawn_budget.go` now excludes `agent.SpawnStatusAbandoned` via the real constant (closing D-18), replacing plan 05's literal placeholder and its now-obsolete owner comment.
- The automatic half of D-15 is wired into `beginRuntimeSpawnRun`: every run reaps stale ghosts before any spawn decision consults the budget, and a reaping failure is deliberately ignored so it can never block a run from starting.
- The manual half of D-15 is `aether spawn-orphans`: without `--clear` it lists every entry past the threshold in plain English (caste identity, worker name, elapsed time, who spawned it) and mutates nothing — proven by hashing every file in the store across two consecutive listing runs and asserting byte-identical stdout both times (`TestSpawnOrphansListingMutatesNothing`). With `--clear` it performs the reap and reports the consumed budget before and after, and the same test proves the tree file DID change afterward, so the clear path is not a no-op.
- Both D-22 red-proofs performed and quoted below: comparing against the earlier timestamp instead of the later one flips the refreshed-activity test to red; counting abandoned entries again in the budget flips the budget-release test to red. Both restored with a verified clean diff.

## Task Commits

Each task was committed atomically:

1. **Task 1: Register an abandoned status and a configurable threshold** - `551fd1ec` (feat)
2. **Task 2: The scan, the reap, the automatic call site, the operator command, and the budget release** - `4cc6ae10` (feat)
3. **Task 3: Red-proofs that it reaps the stale one, spares the fresh one, and returns the budget** - `d02aa34e` (test)
4. **Follow-up fix: satisfy wiring/golden ratchets surfaced by the full suite** - `a5d66b62` (fix)

**Plan metadata:** _pending_ (docs: complete plan — added by the orchestrator after all worktree agents in this wave merge)

## Files Created/Modified

- `cmd/spawn_reap.go` (NEW) - `spawnReapThresholdMinutes` (fails toward not acting when colony state is unreadable), `spawnReapEntryLastActivity` (later-of-two-timestamps), `spawnReapCandidates` (read-only scan), `spawnReapStaleEntries` (the mutation), and the `spawn-orphans` command with dual JSON/English-text rendering
- `cmd/spawn_reap_test.go` (NEW) - five named tests plus the raw-pipe-line fixture helpers the plan specified in place of a clock-injection seam
- `pkg/agent/spawn_tree.go` - `SpawnStatusAbandoned` constant; `IsTerminalSpawnStatus` now includes it, `IsLiveSpawnStatus` does not
- `pkg/agent/spawn_tree_test.go` - `TestSpawnStatusAbandonedIsTerminalNotLive`
- `pkg/colony/colony.go` - `SpawnReapThresholdMinutes *int` field beside `ColonyDepth`
- `pkg/colony/colony_test.go` - `TestSpawnReapThresholdMinutesRoundTrips`, `TestSpawnReapThresholdMinutesNilWhenAbsent`
- `cmd/spawn_runs.go` - `beginRuntimeSpawnRun` now calls `spawnReapStaleEntries` after the run begins, error ignored
- `cmd/spawn_budget.go` - `spawnTreeBudgetState` now excludes `agent.SpawnStatusAbandoned`; plan 05's literal constant and owner comment removed
- `.claude/commands/ant/patrol.md` / `.opencode/commands/ant/patrol.md` - added a bullet wiring `aether spawn-orphans` into the patrol health-check flow, giving the new command a real caller
- `.claude/commands/ant-patrol.md` - the flat-mirror copy of the canonical patrol.md change (`aether install`'s installed-consumer shape)
- `cmd/visual_writer_discipline_test.go` - exempted `spawn_reap.go:writeSpawnOrphansEnvelope`'s JSON branch
- `cmd/command_call_audit_test.go` - classified `spawn-orphans` as D-01 enrichment
- `cmd/testdata/command_catalog.json`, `cmd/testdata/parity_snapshot.json`, `cmd/testdata/regression_snapshot.json` - regenerated golden files (400 → 401 commands)

## Decisions Made

- **The reaper scans the whole spawn tree, not just the current run's window.** D-15 doesn't restrict reaping to the active run, and a ghost left over from an earlier, never-completed run is exactly what the operator needs `spawn-orphans` to be able to see and clear.
- **`spawn-orphans` was wired into `/ant-patrol` rather than given its own slash command.** SPAWN-08 is a health-check concern by nature, and giving it a real caller inside the existing health-check flow satisfies `TestNoRegisteredSubcommandIsUnreferenced` without inventing a new user-facing menu entry for what is fundamentally a diagnostic.
- **`spawn-orphans` is D-01 enrichment, not a gate.** Its failure degrades visibility only; no verification, security, or gate outcome depends on it — the same tier as the other `spawn-*` commands.
- **Test files outside the plan's stated `files_modified` were touched** (`pkg/agent/spawn_tree_test.go`, `pkg/colony/colony_test.go`) because Task 1's own action text explicitly required tests in both packages. Named here rather than left implicit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `writeSpawnOrphansEnvelope`'s JSON branch bypassed `writeVisualOutput`, failing `TestHumanFacingOutputGoesThroughWriteVisualOutput`**
- **Found during:** Full-suite verification after Task 3
- **Issue:** The command's JSON envelope write went directly to `stdout` via `fmt.Fprintln`, which the repo's visual-writer discipline ratchet forbids for any new call site not explicitly exempted — JSON output is deliberately never routed through `writeVisualOutput` (which translates command names for the reader's platform), but that exemption has to be declared, not assumed.
- **Fix:** Added `"spawn_reap.go:writeSpawnOrphansEnvelope"` to `visualWriterExemptions` in `cmd/visual_writer_discipline_test.go`, matching the existing `unblock_cmd.go` JSON-branch exemption pattern exactly.
- **Files modified:** cmd/visual_writer_discipline_test.go
- **Verification:** `TestHumanFacingOutputGoesThroughWriteVisualOutput` passes
- **Committed in:** `a5d66b62`

**2. [Rule 1 - Bug] New command `spawn-orphans` had no D-01 severity classification, failing `TestDocumentedSubcommandsAreSeverityClassified`**
- **Found during:** Full-suite verification after Task 3
- **Issue:** Every subcommand documented anywhere in the audited wrapper corpora must be deliberately classified as either gate-worthy (failure halts a run) or enrichment (failure warns, run continues). A new command defaults to neither and fails the test until judged.
- **Fix:** Added `spawn-orphans` to `knownEnrichmentSubcommands` in `cmd/command_call_audit_test.go`, with a comment stating the rationale (health-check visibility only, no gate/verification/security outcome depends on it) alongside the existing `spawn-log`/`spawn-complete`/`spawn-can-spawn` entries.
- **Files modified:** cmd/command_call_audit_test.go
- **Verification:** `TestDocumentedSubcommandsAreSeverityClassified` passes
- **Committed in:** `a5d66b62`

**3. [Rule 1 - Bug] `.claude/commands/ant/patrol.md`'s new bullet was not copied to its flat-mirror consumer shape, failing `TestLifecycleFlatMirrorsMatchCanonical`**
- **Found during:** Full-suite verification after Task 3
- **Issue:** `aether install` writes `.claude/commands/ant/patrol.md` to the flat, installed-consumer path `.claude/commands/ant-patrol.md`; the two must stay byte-identical, and the wrapper edit in Task 2 only touched the canonical source.
- **Fix:** Copied the added bullet into `.claude/commands/ant-patrol.md` so the two files are byte-identical again.
- **Files modified:** .claude/commands/ant-patrol.md
- **Verification:** `TestLifecycleFlatMirrorsMatchCanonical/patrol` passes; `diff` confirms byte-identity
- **Committed in:** `a5d66b62`

**4. [Rule 1 - Bug] Three golden-file tests (`TestAuditCatalogGolden`, `TestPlatformParityGolden`, `TestRegressionSnapshot`) fell out of date after the new command was registered**
- **Found during:** Full-suite verification after Task 3
- **Issue:** Adding a new cobra command changes the total command count and catalog contents these golden files snapshot; all three are designed to fail loudly on any undeclared change and require an explicit `-update-golden` run to accept a legitimate one.
- **Fix:** Ran each affected test with `-update-golden`, inspected the resulting diffs (each was minimal and exactly attributable to `spawn-orphans`'s addition — no unrelated drift), and committed the regenerated golden files.
- **Files modified:** cmd/testdata/command_catalog.json, cmd/testdata/parity_snapshot.json, cmd/testdata/regression_snapshot.json
- **Verification:** `TestAuditCatalogGolden`, `TestPlatformParityGolden`, `TestRegressionSnapshot` all pass; each diff inspected and confirmed minimal
- **Committed in:** `a5d66b62`

---

**Total deviations:** 4 auto-fixed, all Rule 1 (test failures directly caused by this plan's own new command, none discovered until the full `go test ./...` run — the plan's own per-task verification commands only exercised a targeted subset of tests)
**Impact on plan:** All four are one-line, deliberate classification/mirroring/golden-regeneration changes this repo requires for every new registered command. No behavior change, no scope creep outside what Task 2's new command required to be wired correctly.

## Issues Encountered

None beyond the four auto-fixed items above, all caught by the plan's own final `go test ./... -count=1 -timeout 900s` verification step before this summary was written.

## Red-Proof Evidence (D-22)

**Red-proof 1 — compared against the EARLIER of the two timestamps instead of the later:**

```
=== RUN   TestSpawnReapRespectsRefreshedActivity
    spawn_reap_test.go:219: reaped = [StillWorking]: StillWorking was reaped despite its recently-refreshed activity timestamp
--- FAIL: TestSpawnReapRespectsRefreshedActivity (0.00s)
```

Flipped to red as required — proves the reaper would kill a helper whose activity was recently refreshed if it ever regressed to comparing against the earlier timestamp. Restored `spawnReapEntryLastActivity` to compare against the later timestamp; re-ran and all five reap/orphans tests passed green.

**Red-proof 2 — made `spawnTreeBudgetState` count abandoned entries again:**

```
=== RUN   TestSpawnReapReleasesBudgetForStaleEntryOnly
    spawn_reap_test.go:125: Consumed after reaping = 2, want 1 (before - 1)
--- FAIL: TestSpawnReapReleasesBudgetForStaleEntryOnly (0.00s)
```

Flipped to red as required — proves reaping genuinely returns a budget slot rather than merely changing a status string nobody checks. Restored the `agent.SpawnStatusAbandoned` exclusion in `cmd/spawn_budget.go`; `git diff cmd/spawn_budget.go` confirmed against the pre-red-proof commit shows only the intended Task 2 change; re-ran and all tests passed green.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 10 (final wiring/marker cleanup) can now register `cmd/spawn_reap_test.go`'s five test names in `wiringGateGuardFiles` (`.github/workflows/ci.yml`) — this plan deliberately did not, per its own instructions, since the file was not yet tracked there at plan-09 time.
- `agent.SpawnStatusAbandoned` is the authoritative constant plan 05 anticipated by name (`spawnTreeBudgetAbandonedStatus`, now removed); any future phase adding a new terminal status should register it in the same two places (`IsTerminalSpawnStatus`/`IsLiveSpawnStatus`) this plan used.
- No blockers for plan 10 or for phase completion; `go test ./... -count=1 -timeout 900s` passes cleanly across the whole repo as of this plan's final commit.

## Known Stubs

None. `spawnReapCandidates`, `spawnReapStaleEntries`, `spawnReapThresholdMinutes`, and `spawn-orphans` (both the listing and `--clear` paths) are fully implemented, tested, and red-proofed — no placeholder behavior remains in any file this plan touched.

---
*Phase: 173-delegation-guard*
*Completed: 2026-08-13*

## Self-Check: PASSED

- FOUND: cmd/spawn_reap.go
- FOUND: cmd/spawn_reap_test.go
- FOUND: .planning/phases/173-delegation-guard/173-09-SUMMARY.md
- FOUND commit 551fd1ec (Task 1)
- FOUND commit 4cc6ae10 (Task 2)
- FOUND commit d02aa34e (Task 3)
- FOUND commit a5d66b62 (follow-up fix)
