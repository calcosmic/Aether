---
phase: 191-dead-wood
plan: 01
subsystem: infra
tags: [dead-code, go-test, ratchet, colony-config, roster-01, roster-02]

# Dependency graph
requires:
  - phase: 191-dead-wood (context)
    provides: 191-CONTEXT.md's Criterion 1 zero-readership reconnaissance, 191-PATTERNS.md's ratchet house style
provides:
  - 36 deleted zero-reader colony/ config files (27 colony/agents/*.yaml, 9 colony/phases/*.yaml)
  - cmd/colony_zero_reader_config_ratchet_test.go, a proven reappearance ratchet for those 36 paths
  - A recorded, evidenced finding that 3 of the ROADMAP's named 39 files (colony/policies/model-routing.yaml, autopilot.yaml, memory-rules.yaml) have a real, current reader and were NOT deleted
affects: [191-07 (final verification plan), any future ruling on the 3 held-back policy files, ROSTER-01, ROSTER-02]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Tier-1 existence-check ratchet (os.Stat against a hardcoded path list) reusing repoRootForCommandSourceTest()"]

key-files:
  created: [cmd/colony_zero_reader_config_ratchet_test.go]
  modified: []

key-decisions:
  - "Deleted 36 of the 39 ROADMAP-named files, not 39 -- 3 policy files retained after fresh re-verification found a real, currently-passing test reader the planning pass missed"
  - "Ratchet test scoped to exactly the 36 files actually deleted (27+9), not the plan's literal 39, to avoid a self-contradicting test suite"
  - "Did not modify cmd/policy_schema_test.go to reconcile this -- outside this plan's files_modified and outside this plan's authority to rule on"
  - "Did not mark ROSTER-01/ROSTER-02 complete -- ROSTER-02 explicitly names model-routing.yaml, which was not deleted"

patterns-established:
  - "Tier-1 zero-reader ratchets should re-verify readership as their very first step, not trust planning-time evidence, and must have an explicit built-in contingency for 'delete N of M, not M' when re-verification disagrees with the plan"

requirements-completed: [ROSTER-01]

# Metrics
duration: 21min
completed: 2026-08-21
---

# Phase 191 Plan 01: Delete Zero-Reader Colony Configs Summary

**Deleted 36 of the 39 ROADMAP-named zero-reader `colony/` config files (all 27 `colony/agents/*.yaml` caste definitions and all 9 `colony/phases/*.yaml` phase definitions) with a proven reappearance ratchet; held back the 3 named policy files after finding a real, currently-passing Go test the planning pass missed.**

## Performance

- **Duration:** ~21 min
- **Started:** 2026-08-21T00:53:06Z (base commit `916db8fa`)
- **Completed:** 2026-08-21T01:14:26Z
- **Tasks:** 2/2 executed (both adjusted from their literal 39-file scope to a verified 36-file scope)
- **Files modified:** 37 (36 deletions + 1 new test file)

## Accomplishments
- Deleted all 27 `colony/agents/*.yaml` caste definitions and all 9 `colony/phases/*.yaml` phase definitions -- built in Phase 154 for the dead `control-ts/` control plane (deleted Phase 160), confirmed zero-reader across Go source (including string-built paths), the TS host, wrapper markdown on all three platforms, skills, and the publish/install manifest
- `colony/agents/` and `colony/phases/` no longer exist as directories (git prunes now-empty parents after `git rm`)
- Wrote and fail-then-pass proved `cmd/colony_zero_reader_config_ratchet_test.go`, which fails loudly and by name if any of the 36 deleted paths reappears
- Found and recorded a genuine planning gap: `cmd/policy_schema_test.go` (introduced Phase 160, weeks before this phase's own planning pass) is a real, current reader of all 3 target policy files. This finding required narrowing this plan's scope from the ROADMAP's literal 39-file list to the 36 files independently re-verified as safe, rather than executing the plan as literally written
- Corrected a secondary, minor planning inaccuracy: `cmd/install_cmd.go` does reference `"colony"` (4 times, not zero as 191-CONTEXT.md's D-13 stated) -- but the sync is filtered to exactly `oracle-phase-directives.yaml` via `isOraclePhaseDirectivesFile`, so the conclusion (none of this plan's target files are shipped) still holds
- Full `go build ./...`, `go vet ./...`, and `go test ./cmd/... -count=1` (5000+ tests, 380s) all pass with zero regressions, both before and after the deletion

## Task Commits

Each task was committed atomically:

1. **Task 1: Reconfirm zero-readership and delete the files** - `64531540` (feat) -- deleted 36 of the 39 named files; 3 held back with the reader finding documented in the commit message
2. **Task 2: Write the reappearance ratchet with fail-then-pass proof** - `0108f9ac` (test) -- ratchet scoped to the 36 actually-deleted paths, fail-then-pass proven against 4 recreated files (2 per group)

**Plan metadata:** this SUMMARY's own commit (to follow, `docs(191-01): complete plan`)

## Files Created/Modified
- `cmd/colony_zero_reader_config_ratchet_test.go` - New. `TestZeroReaderColonyConfigsDoNotReappear` (36-path existence-check ratchet) + `TestZeroReaderColonyConfigsListIsComplete` (guards the list's 27+9 shape)
- `colony/agents/*.yaml` (27 files) - Deleted
- `colony/phases/*.yaml` (9 files) - Deleted

## Decisions Made

- **Adjusted scope from 39 to 36 files, per the plan's own explicit contingency.** The plan's Task 1 `<action>` states verbatim: *"If ANY of the 39 files now shows a reader that did not exist during planning, stop, do not delete that specific file, and record the finding in the SUMMARY instead of proceeding past it."* Fresh re-verification (required by both the plan and the orchestrating agent's `<discipline>` instructions -- "the planner's evidence is your map, not your proof") found exactly this condition for 3 files. This is documented as a deviation below, not silently absorbed.
- **Did not touch `cmd/policy_schema_test.go`.** It is outside this plan's `files_modified`, five sibling executors are working disjoint files in this same phase, and reconciling a test that also covers `dispatch-contract.yaml`, `safety-gates.yaml`, `pheromone-lifecycle.yaml`, `signal-rules.yaml`, and `skill-creation.yaml` (files this plan has no authority over) is a decision bigger than "delete 3 more files."
- **Did not run `requirements.mark-complete` for ROSTER-01 or ROSTER-02.** ROSTER-02's exact text ("no file remains that claims to define agents and defines nothing -- including `colony/policies/model-routing.yaml`") is not satisfied: `model-routing.yaml` still exists, for a documented, sufficient reason, but the requirement's literal text is still unmet. Marking it complete would be exactly the "checked box that isn't true" failure mode `./CLAUDE.md`'s Definition of Done exists to prevent. ROSTER-01 (`colony/agents/*.yaml` specifically) is fully delivered and is listed in `requirements-completed` above.

## Deviations from Plan

### Auto-fixed / Scope-Adjusting Issues

**1. [Rule 3 - Blocking, pre-authorized by the plan's own contingency] 3 of the 39 named files were not deleted: a real, current reader exists**

- **Found during:** Task 1, the mandatory fresh six-surface re-verification (immediately before deleting anything, per 191-PATTERNS.md and the plan's own `<action>`)
- **Issue:** `cmd/policy_schema_test.go`'s `TestPolicySchemaRequiredFields` reads `colony/policies/model-routing.yaml`, `colony/policies/memory-rules.yaml`, and `colony/policies/autopilot.yaml` live from disk via `os.ReadFile(filepath.Join("..", "colony", "policies", c.file))` and calls `t.Fatalf` if any is missing. This file was introduced in commit `ecaddb7a` ("Release v1.0.43 -- Phase 160 Fail Loudly", 2026-07-28) -- **weeks before** 191-CONTEXT.md's planning pass (dated 2026-08-21). It is not a race with a sibling executor; it is a real reader the planning grep sweep did not catch (the sweep matched `policyPath(...)` call sites and specific literal path fragments, but this test builds its path from a `filepath.Join` + a table-driven filename, and reads the file for schema validation, not for a production `policyPath()`-style load).
- **Why this matters beyond "one test would fail":** ROSTER-02's exact requirement text names `colony/policies/model-routing.yaml` specifically. This is not a minor implementation detail -- it means the requirement this plan was assigned to deliver "in full" cannot be honestly claimed complete by this plan alone.
- **Fix:** Did not delete `colony/policies/model-routing.yaml`, `colony/policies/autopilot.yaml`, or `colony/policies/memory-rules.yaml`. Deleted only the 36 files independently re-verified as genuinely zero-reader (27 `colony/agents/*.yaml` + 9 `colony/phases/*.yaml`, neither group referenced anywhere in `cmd/policy_schema_test.go` or any other reader found). Scoped `cmd/colony_zero_reader_config_ratchet_test.go` to exactly those 36 paths, with a header comment explaining the exclusion and pointing at this SUMMARY.
- **Files affected:** `colony/policies/model-routing.yaml`, `colony/policies/autopilot.yaml`, `colony/policies/memory-rules.yaml` (all three: not modified, not deleted, confirmed byte-unchanged -- `git status --short colony/policies/` showed zero diff after Task 1)
- **Verification:** `go test ./cmd/... -count=1` (5000+ tests, 380s) passes with zero regressions in the post-deletion state, including `TestPolicySchemaRequiredFields` itself, which still reads all three held-back files successfully. Had they been deleted, this exact test would have failed with `colony/policies/model-routing.yaml: read failed: ...`.
- **Committed in:** `64531540` (Task 1 commit; the commit message documents this finding directly)
- **Follow-up needed:** A future ruling must either (a) delete `cmd/policy_schema_test.go`'s coverage of these 3 files in the *same* change that deletes the files themselves, or (b) explicitly rule that this schema test is a legitimate reader and the files stay, closing ROSTER-02 as "wire," not "delete." This plan does not have the authority or the file scope to make that call unilaterally.

**2. [Minor factual correction, no action needed] `cmd/install_cmd.go` does reference `"colony"` -- 191-CONTEXT.md's D-13 evidence was imprecise**

- **Found during:** Task 1's re-verification of the publish/install manifest surface
- **Issue:** 191-CONTEXT.md's D-13 states *"`cmd/publish_cmd.go` and `cmd/install_cmd.go` contain zero references to `colony`."* `cmd/publish_cmd.go` is correctly zero (re-confirmed). `cmd/install_cmd.go` has 4 references, including a real sync step: `filepath.Join(packageDir, "colony", "policies")` to `filepath.Join(systemDir, "colony", "policies")`.
- **Why it doesn't change the outcome:** That sync step passes `include: isOraclePhaseDirectivesFile`, which matches only `filepath.Base(relPath) == "oracle-phase-directives.yaml"` (confirmed by reading `cmd/platform_sync.go:362-364`). None of this plan's 36 deleted files, nor the 3 held-back policy files, are named `oracle-phase-directives.yaml`. The install step ships exactly one file -- the one this phase's criterion 5 already protects -- so the underlying conclusion (this plan's targets are not shipped anywhere) still holds.
- **Fix:** None required. Recorded here as a correction to the planning record, matching this phase's own established discipline (see 191-CONTEXT.md's own D-07/D-09 corrections) of recording an inaccuracy rather than silently trusting or silently ignoring it.
- **Files affected:** None (no code change)
- **Verification:** Direct read of `cmd/install_cmd.go:790-837` and `cmd/platform_sync.go:358-364`
- **Committed in:** N/A (no code change; documented here only)

---

**Total deviations:** 1 scope-adjusting (pre-authorized by the plan's own contingency), 1 factual correction (no code change)
**Impact on plan:** ROSTER-01 fully delivered. ROSTER-02 NOT fully delivered -- 3 of its named files remain, for a documented, verified, sufficient reason. No scope creep in either direction: nothing was deleted that shouldn't have been, and nothing outside this plan's `files_modified` was touched to force the full 39.

## Issues Encountered

- A duplicate/parallel Bash sandbox rejected two multi-statement compound commands ("too complex to verify... stays inside the worktree") during setup and during commit-safety checks. Resolved by breaking each into individual single-purpose commands; no functional impact.
- An initial `find colony/agents colony/phases -type f | wc -l` returned `2` immediately after `git rm`, before settling to the correct `0` once git finished pruning the now-empty parent directories. Re-ran and confirmed `0`; not a real discrepancy, a transient mid-cleanup read.
- During the fail-then-pass proof, `rm` (not `git rm`) doesn't prune empty parent directories the way `git rm` does. After deleting the 4 recreated proof files, `colony/agents/` and `colony/phases/` were left behind as empty directories (invisible to `git status`, since git never tracked them as empty). Removed with `rmdir` before the final commit so the working tree matches the committed state exactly, not just according to git's diff.

## User Setup Required

None - no external service configuration required.

## Proof (Definition of Done)

Per `./CLAUDE.md`'s Definition of Done ("a command exists that someone can run, and that command fails when the requirement is unmet"):

**(a) Files gone:**
```
$ git status --porcelain colony/policies/
(empty -- zero diff, all 7 remaining policy files including oracle-phase-directives.yaml byte-unchanged)
$ find colony/agents colony/phases -type f 2>/dev/null | wc -l
0
$ test -f colony/policies/model-routing.yaml && echo PRESENT || echo ABSENT
PRESENT   (deliberately retained -- see Deviations #1)
```

**(b) The ratchet fails loudly when a file is restored, then passes again once deleted -- demonstrated, not assumed:**

Recreated 4 files (2 per actually-deleted group, exceeding the plan's "3 files minimum, one per group" bar) from their pre-deletion git content (`git show 916db8fa...:colony/agents/ambassador.yaml`, etc.): `colony/agents/ambassador.yaml`, `colony/agents/builder.yaml`, `colony/phases/build.yaml`, `colony/phases/init.yaml`.

RED (ratchet correctly fails, naming all 4 individually in one run -- not just the first):
```
=== RUN   TestZeroReaderColonyConfigsDoNotReappear
    colony_zero_reader_config_ratchet_test.go:130: colony/agents/ambassador.yaml has reappeared -- this file was ruled zero-reader and deleted in Phase 191 Plan 01 (ROADMAP criterion 1, ROSTER-01/ROSTER-02). Either it has a genuine new reader ... or it should be deleted again
    colony_zero_reader_config_ratchet_test.go:130: colony/agents/builder.yaml has reappeared -- ...
    colony_zero_reader_config_ratchet_test.go:130: colony/phases/build.yaml has reappeared -- ...
    colony_zero_reader_config_ratchet_test.go:130: colony/phases/init.yaml has reappeared -- ...
--- FAIL: TestZeroReaderColonyConfigsDoNotReappear (0.00s)
FAIL
```

Deleted the 4 recreated files again (plus the scratch `.tmp` extraction files and the now-empty `colony/agents/`/`colony/phases/` directories `rm` left behind), confirmed `git status --short colony/` returned empty (byte-clean, matching the committed state exactly).

GREEN (clean pass restored):
```
=== RUN   TestZeroReaderColonyConfigsDoNotReappear
--- PASS: TestZeroReaderColonyConfigsDoNotReappear (0.00s)
=== RUN   TestZeroReaderColonyConfigsListIsComplete
--- PASS: TestZeroReaderColonyConfigsListIsComplete (0.00s)
PASS
ok  	github.com/calcosmic/Aether/cmd	0.911s
```

**(c) The named ceremonies still pass:**
```
$ go build ./...        # clean, both before and after deletion
$ go vet ./...           # clean, both before and after deletion
$ go test ./cmd/... -count=1
ok  	github.com/calcosmic/Aether/cmd	380.222s
?   	github.com/calcosmic/Aether/cmd/aether	[no test files]
```
5000+ tests, zero regressions, including `TestPolicySchemaRequiredFields` (proof the 3 held-back files are correctly still readable) and the two new ratchet tests.

**The ratchet fails loudly if it finds nothing to check:** `TestZeroReaderColonyConfigsListIsComplete` asserts the hardcoded path list has exactly 36 entries in the exact 27 (`colony/agents`) + 9 (`colony/phases`) shape -- an empty or truncated list is a `t.Fatalf`/`t.Errorf`, never a silent vacuous pass.

## Next Phase Readiness

- Plans 191-02 through 191-06 (disjoint files, criteria 2-4) are unaffected by this plan's scope adjustment -- no shared files, no collision.
- **191-07 (final verification, Wave 2) needs to know:** ROSTER-01 is fully delivered; ROSTER-02 is not. The `session-verify-fresh`-class correction discipline this phase established (191-CONTEXT.md D-07) applies here too -- this is a second instance of "the ROADMAP's premise didn't survive contact with the live repository," and should be surfaced to the owner as its own finding, not silently absorbed into a claimed-complete criterion 1.
- **Follow-up ruling required** (not this plan's to make): whether `colony/policies/model-routing.yaml`, `autopilot.yaml`, and `memory-rules.yaml` should ultimately be deleted (requiring `cmd/policy_schema_test.go` to be updated in the same change) or kept and formally ruled "wire" for ROSTER-02's purposes.
- Two new Go test functions (`TestZeroReaderColonyConfigsDoNotReappear`, `TestZeroReaderColonyConfigsListIsComplete`) are permanent, proven guards against the 36 deleted files reappearing silently.

---
*Phase: 191-dead-wood*
*Completed: 2026-08-21*

## Self-Check: PASSED

- FOUND: `cmd/colony_zero_reader_config_ratchet_test.go`
- FOUND: `.planning/phases/191-dead-wood/191-01-SUMMARY.md`
- FOUND: commit `64531540` (Task 1)
- FOUND: commit `0108f9ac` (Task 2)
