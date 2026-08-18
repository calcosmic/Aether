---
phase: 187-crash-safe-worktrees-ecosystem-neutrality
plan: 03
subsystem: infra
tags: [git, worktree, safety, cli, go]

# Dependency graph
requires:
  - phase: 187-01
    provides: "worktreeDestructionSafety guard, preserveWorktreeWork preservation routine, describeWorktreePreservation/reportWorktreePreservation reporting, isDestructiveWorktreeAction/worktreeDestructionCategory classification"
provides:
  - "gcOrphanedWorktrees rewritten as preserve-and-report — never calls removeGitWorktree"
  - "resume/continue/init report worktree preservation in plain English on every occurrence"
  - "worktree-reap: the only named, operator-invoked destruction command left in the codebase"
  - "Real-git fail-then-pass tests proving criterion 1 (crash-then-resume never destroys the work being resumed)"
affects: [187-04, worktree-lifecycle, recover, subcommand-reachability-ratchets]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Preserve-and-report replacing implicit destroy-on-cleanup inside recovery paths (resume/continue/init)"
    - "Destruction gated behind an explicit, named, --force-only operator command with a lifecycle-caller ratchet enforcing it stays that way"
    - "orphan_allowlist.json/_baseline.json deliberate-exemption pattern for a command required to have zero callers by design"

key-files:
  created:
    - cmd/worktree_reap.go
    - cmd/worktree_crash_safety_test.go
  modified:
    - cmd/codex_build_worktree.go
    - cmd/codex_continue.go
    - cmd/session_flow_cmds.go
    - cmd/codex_visuals.go
    - cmd/init_cmd.go
    - cmd/session_flow_cmds_test.go
    - cmd/subcommand_reachability_ratchet_test.go
    - cmd/testdata/command_catalog.json
    - cmd/testdata/orphan_allowlist.json
    - cmd/testdata/orphan_allowlist_baseline.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/regression_snapshot.json

key-decisions:
  - "gcOrphanedWorktrees keeps its no-phase-filter behavior deliberately (CONTEXT.md) — the report names each entry's phase instead of hiding older worktrees, since the missing filter is no longer dangerous once nothing is destroyed"
  - "worktree-reap's --force-alone branch (unmerged/dirty, no --include-unmerged) does not stash — the worktree is not being touched in that branch, so stashing would needlessly disturb a workspace being left alone; stashing only happens on the destroy path (--force --include-unmerged)"
  - "worktree-reap is a permanent, by-design registered-but-uncalled command: added to testdata/orphan_allowlist.json, testdata/orphan_allowlist_baseline.json, and pathCollisionRevealedOrphans together in one change, following the existing CR-04 reviewed-exemption precedent, because T-187-13 requires it have no caller anywhere except a human typing it"
  - "init_cmd.go's preservation message says 'aether recover' rather than naming worktree-reap directly, since TestWorktreeReapHasNoLifecycleCaller fails the build if that literal string appears in any of the six named lifecycle files"

requirements-completed: []

# Metrics
duration: 95min
completed: 2026-08-18
---

# Phase 187 Plan 03: Crash-Safe Worktrees & Ecosystem Neutrality Summary

**Rewrote `gcOrphanedWorktrees` from silent force-delete to preserve-and-report, added the `worktree-reap` operator-only destruction command, and proved the fix with real-git fail-then-pass tests that a crash between dispatch and finalize survives resume.**

## Performance

- **Duration:** 95 min
- **Started:** 2026-08-18T15:50:17Z (worktree branch base corrected from a stale commit before Task 1)
- **Completed:** 2026-08-18T17:01:29Z
- **Tasks:** 4
- **Files modified:** 13 (2 new, 11 modified)

## Accomplishments
- Rewrote `gcOrphanedWorktrees` (`cmd/codex_build_worktree.go`) so it never calls `removeGitWorktree`: every entry is checked with `worktreeDestructionSafety` first, dirty/unmerged/undeterminable work is stashed and kept via `preserveWorktreeWork`, and even clean-and-merged worktrees are kept (marked `Orphaned`) rather than deleted — destruction moved entirely to an explicit operator command
- Made `resume`, `continue`, and `init` report what was preserved in plain English on every occurrence: fixed a previously-captured-but-never-rendered error in the resume dashboard, corrected a "non-blocking" comment on a call that was actually synchronous with its error discarded, and rewrote `init`'s stderr warning
- Closed a compounding hazard in `init`: its unconditional `os.RemoveAll` of the worktrees directory would have silently undone every preservation the rewritten `gcOrphanedWorktrees` just made — now gated on `preserved == 0`
- Added `worktree-reap`, the only remaining caller of `removeGitWorktree` in the codebase: read-only by default (mutates nothing, verified byte-identical `COLONY_STATE.json`), `--force` still refuses to destroy unmerged/dirty work, and `--include-unmerged` (which requires `--force`) stashes before destroying so consent-based destruction still cannot lose bytes
- Wrote 9 real-git-fixture tests (`git init` + `git worktree add`, never a nonexistent path) covering the crash-survives-resume path, the cross-phase non-destruction path, a source-level ratchet against `removeGitWorktree` reappearing inside `gcOrphanedWorktrees`, a lifecycle-caller ratchet for `worktree-reap`, and the four `worktree-reap` CLI behaviors
- Performed and recorded the fail-then-pass demonstration ROADMAP criterion 1 explicitly requires (see below)

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite gcOrphanedWorktrees as preserve-and-report** - `bf1085db` (feat)
2. **Task 2: Make resume, continue and init report preservation in plain language** - `a81cf6aa` (feat)
3. **Task 3: Add the named operator-invoked destruction command** - `9fcfde53` (feat)
4. **Task 4: Fail-then-pass crash-and-resume tests** - `95868ea4` (test)

**Plan metadata:** (this commit, made by the orchestrator after wave merge)

_Note: Task 4's commit also carries the fixes discovered while writing the tests (see Deviations) and the golden/allowlist file updates the new command requires — these were verified together as one coherent, fully-green change rather than split across commits that would each leave the suite red._

## Files Created/Modified
- `cmd/codex_build_worktree.go` - `gcOrphanedWorktrees` rewritten; signature changed to `(cleaned, preserved int, err error)`; doc comment corrected to no longer claim it "cleans up" anything
- `cmd/codex_continue.go` - renamed `gcOrphaned` to `gcPreserved`, captured and surfaced the previously-discarded error, corrected the "non-blocking" comment
- `cmd/session_flow_cmds.go` - renamed the resume dashboard key to `worktrees_preserved`, surfaced `worktree_gc_error`
- `cmd/codex_visuals.go` - `renderResumeVisual` now reads `worktrees_preserved`/`worktree_gc_error` instead of the old `worktree_gc` key, with wording that avoids "orphan"/"GC"/"cleaned up"
- `cmd/init_cmd.go` - preservation-aware stderr reporting; `os.RemoveAll` of the worktrees directory gated on `preserved == 0`
- `cmd/session_flow_cmds_test.go` - updated two tests that asserted the old `worktree_gc` key and "cleaned up"/"could not be cleaned" wording
- `cmd/worktree_reap.go` (new) - the `worktree-reap` cobra command: `--force`, `--include-unmerged`, `--branch`, `--json` flags; read-only by default
- `cmd/worktree_crash_safety_test.go` (new) - 9 tests proving criterion 1 end to end
- `cmd/subcommand_reachability_ratchet_test.go` - documented `worktree-reap` as a deliberate reviewed exemption in `pathCollisionRevealedOrphans`
- `cmd/testdata/{command_catalog,parity_snapshot,regression_snapshot}.json` - golden files refreshed for the new command (401 commands, was 400)
- `cmd/testdata/orphan_allowlist{,_baseline}.json` - `aether worktree-reap` added to both, reason `deliberately-operator-invoked-only`, owner_phase `187`

## Decisions Made
- Followed the plan's exact evaluation order in `gcOrphanedWorktrees`: guard first, preserve-and-report if unsafe, preserve-and-report-as-removable if safe-but-merged, drop only if the path is genuinely gone
- Kept the missing phase filter deliberately per CONTEXT.md — the report now names each entry's phase so a user resuming phase 5 who sees a phase 2 worktree mentioned understands what is meant, rather than adding a filter that would hide it
- Chose to route `worktree-reap`'s human-facing stdout/stderr text through `visualFprint`/`visualFprintf`/`visualFprintln` rather than raw `fmt.Fprint*`, matching `TestHumanFacingOutputGoesThroughWriteVisualOutput`'s discipline that all human-facing text funnel through `writeVisualOutput` for platform command-name translation
- Resolved the tension between Task 2 (which asked `init_cmd.go` to name `worktree-reap` directly) and Task 4's `TestWorktreeReapHasNoLifecycleCaller` (which fails if that literal string appears in `init_cmd.go`) by pointing the user at `aether recover` instead — `gcOrphanedWorktrees` itself already names `worktree-reap` per-entry via `reportWorktreePreservation` from a file outside the ratchet's six-file scope

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] worktree-reap stashed dirty work even when only reporting, not destroying, it**

- **Found during:** Task 4, while writing `TestWorktreeReapForceKeepsUnmergedWork`
- **Issue:** In the `--force` (no `--include-unmerged`) branch, the initial implementation called `preserveWorktreeWork`, which runs `git stash`, even though the worktree in that branch is not being destroyed at all — it is simply left alone. This needlessly disturbed an untouched worktree's working tree.
- **Fix:** Removed the stash call from that branch; a worktree that is not being destroyed has nothing that needs preserving.
- **Files modified:** `cmd/worktree_reap.go`
- **Verification:** `TestWorktreeReapForceKeepsUnmergedWork` passes, asserting the dirty file is readable directly in the working tree (not via stash pop) after `--force` alone.
- **Committed in:** `95868ea4` (Task 4 commit)

**2. [Rule 3 - Blocking] Task 2's required init_cmd.go wording directly violated Task 4's own lifecycle-caller ratchet**

- **Found during:** Task 4, running the full suite after adding `TestWorktreeReapHasNoLifecycleCaller`
- **Issue:** Task 2's instructions said to name `worktree-reap` in `init_cmd.go`'s stderr messages. Task 4's own new ratchet test fails the build if the literal string `"worktree-reap"` appears in `init_cmd.go` (one of the six named lifecycle files) — a real conflict between two tasks in the same plan.
- **Fix:** Reworded `init_cmd.go`'s two messages to point at `aether recover` instead. This loses no information the user needs: `gcOrphanedWorktrees` itself already names `worktree-reap` per clean-and-merged entry via `reportWorktreePreservation`, called from `cmd/codex_build_worktree.go`, which is outside the ratchet's six-file scope.
- **Files modified:** `cmd/init_cmd.go`
- **Verification:** `TestWorktreeReapHasNoLifecycleCaller` passes; `init`'s reporting behavior (count preserved, warn on check failure) is unchanged.
- **Committed in:** `95868ea4` (Task 4 commit)

**3. [Rule 1 - Bug] Test fixtures used absolute paths for `WorktreeEntry.Path`, silently masking the fail-then-pass demonstration**

- **Found during:** Task 4's fail-then-pass demonstration — `TestResumingOnePhaseDoesNotDestroyAnotherPhasesWorktree` unexpectedly passed against the reverted (destructive) code
- **Issue:** Production code and every existing fixture in `cmd/worktree_test.go` store `WorktreeEntry.Path` relative to root and join it with root at use time (`filepath.Join(root, entry.Path)`). My first draft of the new fixtures passed already-absolute paths, so the mutated code's `filepath.Join(root, entry.Path)` double-prefixed root, pointed at a nonexistent directory, and silently no-op'd — making the test pass regardless of whether the guard was present. This would have shipped a fail-then-pass test that could never actually fail.
- **Fix:** Added `addCrashSafetyWorktree` returning both the absolute path (for filesystem/git assertions) and the root-relative path (for `WorktreeEntry.Path`), and used the relative path consistently across all fixtures in the new test file.
- **Files modified:** `cmd/worktree_crash_safety_test.go`
- **Verification:** Re-ran the fail-then-pass demonstration after the fix — all three target tests genuinely failed against the mutated code this time, confirming the earlier pass had been a fixture bug, not a real result. See "Fail-Then-Pass Demonstration" below.
- **Committed in:** `95868ea4` (Task 4 commit; the bug was caught and fixed before commit, so no separate fix commit exists)

### Golden-File / Allowlist Updates (not code deviations, but required for `go test ./cmd/` to pass)

Adding a genuinely new command (`worktree-reap`) required refreshing three golden files (`command_catalog.json`, `parity_snapshot.json`, `regression_snapshot.json` — command count 400 → 401) via each test's own `-update-golden` flag, and adding `aether worktree-reap` to both `testdata/orphan_allowlist.json` and `testdata/orphan_allowlist_baseline.json` together (plus `pathCollisionRevealedOrphans` in the test file itself) as a deliberate, on-the-record, permanent exemption — the reachability ratchet's own message states this must never be done by editing the baseline alone without review, so the rationale is recorded both as a JSON `reason` field (`deliberately-operator-invoked-only`) and as a code comment at the point the ratchet fires. All diffs were inspected and confirmed additive-only (one new command, nothing else changed).

---

**Total deviations:** 3 auto-fixed (2 bugs, 1 blocking-tension resolution). No architectural changes, no scope creep.
**Impact on plan:** All three were caught and corrected before any task commit landed, so no fix-up commits were needed. The third (absolute vs. relative test paths) is the most consequential: it would have produced a fail-then-pass test that could never fail, silently defeating the exact ROADMAP criterion 1 verification this plan exists to prove.

## Fail-Then-Pass Demonstration

Performed as required by Task 4's acceptance criteria (ROADMAP criterion 1's literal wording):

1. Temporarily replaced the safety-guarded body of `gcOrphanedWorktrees` with the original pre-Phase-187 destructive logic (direct `removeGitWorktree` call, no guard).
2. Ran the three tests the plan names: `TestCrashBetweenDispatchAndFinalizeSurvivesResume`, `TestResumingOnePhaseDoesNotDestroyAnotherPhasesWorktree`, `TestGCNeverCallsRemoveGitWorktree`.
3. **First attempt:** only 2 of 3 failed as expected (`TestResumingOnePhaseDoesNotDestroyAnotherPhasesWorktree` passed against the destructive code — traced to the absolute-path fixture bug documented above).
4. Fixed the fixture bug, re-ran: **all three failed** against the destructive code, confirmed by exact error output:
   - `TestCrashBetweenDispatchAndFinalizeSurvivesResume`: `expected preserved >= 1, got 0`; `expected worktree directory to still exist, stat error: no such file or directory`
   - `TestResumingOnePhaseDoesNotDestroyAnotherPhasesWorktree`: `expected stalled phase-2 worktree directory to still exist, stat error: no such file or directory`
   - `TestGCNeverCallsRemoveGitWorktree`: `gcOrphanedWorktrees's body still contains a call to removeGitWorktree outside of comments`
5. Restored the real fix (verified `git diff` showed zero difference from the committed Task 1 state).
6. Re-ran all 9 named tests in the plan's verification block: **all 9 passed.**
7. Ran the full `go test ./cmd/ -count=1` suite: **passed** (no regressions from either the fix or the golden-file updates).

## Issues Encountered

**Worktree branch base was stale at agent start**, same class of issue as plan 01: `HEAD` was on an old commit (`6577f51c`, several releases behind) instead of the expected wave-1 tracking commit (`a76d75712c...`). Per the documented recovery protocol — HEAD confirmed on the correct `worktree-agent-*` namespace, not a protected ref — the branch was reset to the expected base before any task work began. Expected worktree-agent setup behavior, not a plan defect.

## User Setup Required

None — no external service configuration required. This plan changes internal Go command behavior and adds one new CLI command; nothing requires end-user configuration.

## Next Phase Readiness

Plan 04 (scoped solely to `mergePhaseWorktrees`'s ecosystem-neutral test-command wiring, per the threat model's T-187-13 correction note) is unaffected by and does not depend on this plan's changes beyond the shared `worktree_safety.go` guard from plan 01. No blockers identified.

The one deferred item worth flagging for future phases: `detectOrphanedWorktrees`'s current-phase blind spot (`cmd/codex_build.go`) remains unfixed, as CONTEXT.md explicitly scoped it out of this phase. `gcOrphanedWorktrees`'s no-phase-filter behavior is intentional and unchanged.

## Self-Check: PASSED

- Verified all 5 files created/modified in this plan's Task Commits exist on disk (`cmd/worktree_reap.go`, `cmd/worktree_crash_safety_test.go`, plus the 11 other modified files) — all found.
- Verified all 5 claimed commit hashes (`bf1085db`, `a81cf6aa`, `9fcfde53`, `95868ea4`, `517bcfc0`) appear in `git log --oneline` — all found.

---
*Phase: 187-crash-safe-worktrees-ecosystem-neutrality*
*Completed: 2026-08-18*
