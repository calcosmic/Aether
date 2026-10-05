---
phase: 207-messy-practice-project-gate
plan: 02
subsystem: testing
tags: [journey-harness, fixture-builder, trap-construction, specification-writers, survey-staleness, go-test]

# Dependency graph
requires:
  - phase: 207-messy-practice-project-gate
    provides: "207-01 -- scripts/build-messy-practice-project.sh's preamble/isolation pattern, cmd/journey_traps.go's manifest reader, cmd/testdata/journey/traps.json's one existing trap (leftover-junk-data), the journey eval gate -- all extended (never replaced) by this plan"
provides:
  - "cmd/testdata/journey/traps.json expanded to all nine declared traps (shortcut-to-a-folder, shortcut-loop, case-only-archive-collision, nested-project, long-folder-names, out-of-date-code-map, specification-corrected-mid-planning, leftover-junk-data, unsaved-changes), each carrying a machine-readable assertion.kind + path(s) descriptor both the builder script and the Go test read"
  - "scripts/build-messy-practice-project.sh builds every declared trap, idempotently, refusing a foreign or undeclared destination by name"
  - "cmd/journey_seed.go -- two hidden, internal-only commands (journey-seed-stale-survey, journey-seed-superseded-plan) that call the runtime's own writers (classifySurveyFreshness's snapshot fields, createSpecificationDraft/approveSpecification/reviseSpecification) so two self-authenticating traps are derived the way the runtime derives them, never hand-typed"
  - "cmd/messy_practice_project_test.go -- one named, genuinely failable check per declared trap (TestMessyPracticeProjectHasEveryTrap) plus five supporting tests covering manifest/script agreement, declared-order failure reporting, and the builder's own edge behaviour"
affects: [207-03-messy-practice-project-gate, 207-04-messy-practice-project-gate, 207-05-messy-practice-project-gate, 207-06-messy-practice-project-gate]

# Actuals (#2632)
actuals:
  tokens: 16990
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "self-authenticating fixture construction: when a trap's on-disk shape is validated by the runtime itself (a digest that must match its own recorded content, an approval receipt hash that must trace to a real approved revision), the builder calls a hidden, undocumented aether subcommand that invokes the runtime's own writer functions in-process, rather than hand-typing JSON that would either be rejected or silently fail to trip the real classifier -- CLAUDE.md's Definition of Done (\"derive fixture values the way the runtime derives them\")"
    - "assertion.kind vs trap id: cmd/journey_traps.go's journeyTrapAssertion carries a Kind field distinct from the trap's own id -- id names WHAT the trap is, Kind names HOW to check it (one of nine generic checker functions) -- so cmd/messy_practice_project_test.go never needs to switch on, and therefore never needs to re-type, any of the nine declared trap ids (verified: grep -c for every one of the nine ids against the test file returns 0)"
    - "pure (repo, trap) -> error checkers, not *testing.T methods: Go's t.Run always propagates a subtest failure to its parent, so TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder (which must deliberately produce two failures and still report itself green) calls the same checker functions directly rather than through t.Run"
    - "staleness via genuine commit count, never a fabricated digest mismatch: the out-of-date-code-map trap pins a real, valid snapshot to the repository's actual HEAD, then the builder creates 27 real commits afterward -- classifySurveyFreshness's own commit-count expiry check (surveyStaleCommitThreshold=25) does the rest"

key-files:
  created:
    - cmd/journey_seed.go
    - cmd/messy_practice_project_test.go
  modified:
    - cmd/testdata/journey/traps.json
    - cmd/journey_traps.go
    - scripts/build-messy-practice-project.sh

key-decisions:
  - "Two of the nine traps (out-of-date-code-map, specification-corrected-mid-planning) cannot be honestly constructed by a hand-typed JSON literal: territory-snapshot.json's snapshot_id is a SHA-256 of its own other fields (computeTerritorySnapshotID), and a parked planning run's stage-state.json requires a real approval-receipt hash tracing to an actually-approved specification revision (validatePlanningStageAuthority / planningStageSpecificationBinding.validate). cmd/journey_seed.go adds two Hidden:true cobra commands that call the exact runtime writer functions (classifySurveyFreshness's own snapshot fields; createSpecificationDraft/approveSpecification/reviseSpecification, the same functions `aether spec`/`aether discuss` call) so both traps are self-consistent by construction. Verified against a genuine two-fixture experiment (a scratch _test.go, not committed) that classifySurveyFreshness reports Stale and planningRunIsSuperseded reports true against the real code paths before writing the permanent checkers."
  - "The case-only-archive-collision trap is built as .aether/HANDOFF.md + .aether/data/handoff.md in different directories, per the plan's own instruction and RESEARCH.md Pitfall 4 -- never two same-directory siblings, which a case-insensitive volume (the macOS default) would silently alias."
  - "aether init/aether update --force write no project .gitignore -- .aether/ (and everything a trap seeds under .aether/data/) is untracked by construction, since this script never runs `git add -A`/`git add .` past the very first pre-init commit. This meant six of the nine traps need zero git operations at all; only the long-folder-names file, the out-of-date-code-map churn commits, and the (pre-existing) initial project commit ever call `git add`, and each stages an explicit, named path -- never a blanket add."
  - "journeyTrapAssertion gained a Kind field (a vocabulary distinct from the trap's own id) specifically so cmd/messy_practice_project_test.go can dispatch to one of nine generic checker functions without ever re-typing a declared trap id -- enforced by the acceptance criterion's own grep check and generalized to all nine ids, not just the one the plan named."
  - "TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder's per-trap checks are plain (repo, trap) -> error functions, not t.Run subtests, because a real subtest failure always propagates to its parent in Go -- the ordering test must deliberately produce two failures and still report green."
  - "UED-07 is shared with plans 207-01 and 207-06 (per this session's own instructions); `gsd-tools.cjs query requirements.ready-ids` reports it `blocked`, not `ready`, so this plan's close-out does NOT mark UED-07 complete in REQUIREMENTS.md -- that is 207-06's (or whichever plan the tool judges last) job."

coverage:
  - id: D1
    description: "The committed builder script constructs every one of the nine traps the roadmap names, each in a form that trips the real runtime code path it targets (not a plausible-looking guess)"
    requirement: "UED-07"
    verification:
      - kind: integration
        ref: "scripts/build-messy-practice-project.sh run twice against the same destination (bash -c reproduction of the plan's own <verify> block, adjusted for the destination/repo subdirectory layout Plan 01 already established) -- exits 0 both times, marker names all nine ids after each run"
        status: pass
      - kind: unit
        ref: "cmd/messy_practice_project_test.go#TestMessyPracticeProjectBuilderRunsCleanTwice"
        status: pass
      - kind: unit
        ref: "cmd/messy_practice_project_test.go#TestMessyPracticeProjectBuilderRefusesAForeignDirectory"
        status: pass
      - kind: unit
        ref: "cmd/messy_practice_project_test.go#TestMessyPracticeProjectBuilderRequiresADestination"
        status: pass
    human_judgment: false
  - id: D2
    description: "A test asserts each of the nine traps exists, one named check per trap, on a filesystem/git fact or the exact runtime function the trap exercises -- never on the builder script's own printed output"
    requirement: "UED-07"
    verification:
      - kind: unit
        ref: "cmd/messy_practice_project_test.go#TestMessyPracticeProjectHasEveryTrap (nine subtests, one per declared id, in declared file order)"
        status: pass
      - kind: unit
        ref: "cmd/messy_practice_project_test.go#TestMessyPracticeProjectTrapListAndScriptAgree"
        status: pass
    human_judgment: false
  - id: D3
    description: "Trap failures are reported in the declared file order of the trap list, never map-iteration order"
    requirement: "UED-07"
    verification:
      - kind: unit
        ref: "cmd/messy_practice_project_test.go#TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every one of the nine traps' own check is genuinely failable -- removing one trap from a built project makes exactly that check fail, and no other"
    verification:
      - kind: manual_procedural
        ref: "hand-run scratch test (not committed) cloning the shared built project and breaking each of the nine traps in isolation in turn, asserting exactly that trap's checker fails and all eight others still pass -- caught and fixed a `git add -A` breaker bug (fixed to `git add README.md`) that would otherwise have silently swept up and committed two unrelated traps"
        status: pass
    human_judgment: true
    rationale: "The permanent, committed test suite proves this property for exactly two traps (index 0 and index 4, via TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder) by design -- generalizing it to all nine as a permanent, always-run test would mean cloning and rebuilding the shared fixture nine more times per run, working against this plan's own \"keep the whole file's run time inside fast feedback\" instruction. The broader property was verified by hand this session (see key-decisions) rather than left as an unverified assumption; a human (or a future session) re-running that scratch check is the honest way to re-confirm it without paying its cost on every test run."

duration: 130min
completed: 2026-09-22
status: complete
---

# Phase 207 Plan 02: Every Declared Trap, One Named Check Each Summary

**`scripts/build-messy-practice-project.sh` now builds all nine roadmap-declared traps (idempotently, refusing a foreign destination by name); two of the nine are self-authenticating and built via new hidden `aether journey-seed-*` commands that call the runtime's own specification/survey writers rather than hand-typed JSON; `cmd/messy_practice_project_test.go` proves each trap exists with a genuinely failable, declared-order-respecting check.**

## Performance

- **Duration:** 130 min
- **Started:** 2026-09-22
- **Completed:** 2026-09-22
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `cmd/testdata/journey/traps.json` expanded from one entry to the full nine declared traps, each with `journey_steps`, `blocker`, `note`, and a machine-readable `assertion.kind`/`path`/`second_path`/`must` descriptor -- the one list both the builder script and the Go test read.
- `scripts/build-messy-practice-project.sh` implements all nine trap constructions: three plain symlink/nested-git-repo primitives, two files-in-different-directories for the case-only collision, one deep committed source file for the long-lineage cap, and three traps whose construction routes through the real `aether` binary (`journey-seed-stale-survey`, `journey-seed-superseded-plan`, and the extended `leftover-junk-data` merge-by-id seeding). Runs clean twice against the same destination and refuses a foreign directory by name.
- `cmd/journey_seed.go`: two `Hidden: true` cobra commands calling the runtime's own writers (`classifySurveyFreshness`'s snapshot fields for a stale territory map; `createSpecificationDraft`/`approveSpecification`/`reviseSpecification` -- the same functions `aether spec`/`aether discuss` call -- for a planning run parked against a specification the owner then corrects). Neither trap could be built as a hand-typed JSON literal: both formats are self-authenticating (a SHA-256 the runtime recomputes and compares, an approval-receipt hash tracing to a real approved revision).
- `cmd/messy_practice_project_test.go`: `TestMessyPracticeProjectHasEveryTrap` (nine named, declared-order subtests, each dispatching through an `assertion.kind`-keyed checker map -- never a re-typed trap id, verified by `grep -c` returning 0 for every one of the nine ids), `TestMessyPracticeProjectTrapListAndScriptAgree` (static manifest/script agreement), `TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder` (two broken traps fail in declared, not map, order), and three builder-edge tests (`RunsCleanTwice`, `RefusesAForeignDirectory`, `RequiresADestination`), the latter two skipped under `-short`.
- Whole-file run time: ~24s (`go test -run "TestMessyPracticeProject" ./cmd`), well inside the plan's own "usable without the full suite" requirement.

## Task Commits

1. **Task 1: Build every trap the roadmap names** - `d28e9f4e` (feat)
2. **Task 2: One named check per trap, and the builder's own edge behaviour** - `05631cc2` (test)

_No separate plan-metadata commit yet -- STATE.md/ROADMAP.md/REQUIREMENTS.md are updated and committed after this file is written, per the executor's atomic close-out order._

## Files Created/Modified

- `cmd/journey_seed.go` - two hidden, internal-only fixture-construction commands (`journey-seed-stale-survey`, `journey-seed-superseded-plan`)
- `cmd/journey_traps.go` - `journeyTrapAssertion` gains `Kind`/`Path`/`SecondPath`; `journeyTrap` gains `JourneySteps`/`Note`; new `journeyTrapByID`
- `cmd/testdata/journey/traps.json` - all nine declared traps
- `scripts/build-messy-practice-project.sh` - nine `apply_trap_*` functions, extended argument/verification sections
- `cmd/messy_practice_project_test.go` - the trap-assertion test suite

## Decisions Made

See `key-decisions` in the frontmatter above -- the self-authenticating-fixture decision (two hidden commands calling real runtime writers), the different-directory case-collision construction, the `.gitignore`-free project layout's effect on which traps need any git operations at all, the `Kind`-vs-id vocabulary split, the pure-function checker design for the ordering test, and the UED-07 shared-requirement deferral.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The plan's own Task 1 `<verify>` block checks the wrong directory level**
- **Found during:** Task 1, running the plan's exact `<verify>` command
- **Issue:** The plan's `<verify>` block tests `$D/shortcut-to-src` and runs `git -C "$D" status` -- but Plan 01 already established (and this plan correctly continues) that the practice project lives at `$D/repo`, not `$D` itself (`$D` also holds `.journey-bin/`, `.journey-hub/`, and the marker file). Run verbatim, the block fails (exit 1) even though the builder script is correct.
- **Fix:** None needed to the implementation. Verified the same facts at the correct path (`$D/repo/shortcut-to-src`, `git -C "$D/repo" status`), which pass cleanly. Documented here rather than silently substituting a different command without explanation.
- **Files modified:** none (verification-only finding)
- **Verification:** `bash -c '...; REPO="$D/repo"; test -L "$REPO/shortcut-to-src"; git -C "$REPO" status --porcelain | grep -q .'` -> `VERIFY_OK`, exit 0
- **Committed in:** n/a (no code change required)

**2. [Rule 1 - Bug] `ln -s` line count fell short of the acceptance criterion**
- **Found during:** Task 1, checking `grep -cE '^[^#]*ln -s' scripts/build-messy-practice-project.sh >= 3`
- **Issue:** The shortcut-loop trap's two `ln -sf` calls were written on one `&&`-joined line, so `grep -c` (which counts matching LINES, not occurrences) returned 2, not 3.
- **Fix:** Split the two `ln -sf` calls onto separate lines.
- **Files modified:** scripts/build-messy-practice-project.sh
- **Verification:** `grep -cE '^[^#]*ln -s' scripts/build-messy-practice-project.sh` -> 3
- **Committed in:** d28e9f4e (Task 1 commit)

**3. [Rule 1 - Bug] `TestMessyPracticeProjectTrapListAndScriptAgree`'s `esac` search assumed no indentation**
- **Found during:** Task 2, first run of the full test suite
- **Issue:** `strings.Index(script[start:], "\nesac")` requires `esac` at column 0; the script indents `esac` by two spaces, so the test failed to find the case block's end and reported a false "never closed" error.
- **Fix:** Replaced the literal search with `regexp.MustCompile(`(?m)^\s*esac\s*$`)`, tolerant of the real indentation.
- **Files modified:** cmd/messy_practice_project_test.go
- **Verification:** `go test -run TestMessyPracticeProjectTrapListAndScriptAgree ./cmd -v` -> PASS
- **Committed in:** 05631cc2 (Task 2 commit)

**4. [Rule 1 - Bug] Nine literal trap ids appeared in doc comments, failing the "never re-typed" acceptance criterion**
- **Found during:** Task 2, running `grep -c 'shortcut-to-a-folder' cmd/messy_practice_project_test.go` (acceptance criterion requires 0; first draft returned 1, and the same issue existed for all nine ids in prose comments)
- **Issue:** Doc comments above each checker function named its trap by its literal declared id (e.g. "-- the shortcut-to-a-folder trap."), which technically satisfies "never SWITCH on a re-typed id" but not the letter of the grep check, and risks exactly the drift the check exists to catch (a comment silently going stale if an id is renamed).
- **Fix:** Reworded every such comment to a paraphrase (e.g. "the folder-shortcut trap", "the stale-survey-snapshot trap") that never repeats the literal declared id string.
- **Files modified:** cmd/messy_practice_project_test.go
- **Verification:** looped `grep -c -- "$id" cmd/messy_practice_project_test.go` over all nine declared ids -> 0 for every one
- **Committed in:** 05631cc2 (Task 2 commit)

**5. [Rule 1 - Bug] The declared-order test's trap-breaking helper used `git add -A`, silently corrupting unrelated traps**
- **Found during:** post-Task-2 manual verification (a scratch, uncommitted test exercising all nine break-one-trap-in-isolation scenarios, run to confirm the "must be able to fail" property CLAUDE.md requires)
- **Issue:** `breakJourneyTrap`'s `dirty_worktree` case ran `git add -A && git commit` to force a clean tree -- which also staged and committed the folder-shortcut symlink and the nested-git-repo directory, silently breaking those two UNRELATED traps as a side effect. This never surfaced in the permanent, committed `TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder` test because it only ever breaks trap indices 0 and 4 (never index 8, the dirty-worktree trap), but the broader manual check caught it before it could become a false-confidence gap.
- **Fix:** Changed the breaker to `git add README.md` (the one modified tracked file the trap actually cares about), never a blanket `-A`.
- **Files modified:** cmd/messy_practice_project_test.go
- **Verification:** re-ran the scratch all-nine-break-in-isolation check -- all nine traps individually failable, no cross-contamination
- **Committed in:** 05631cc2 (Task 2 commit)

---

**Total deviations:** 5 auto-fixed (2 verification-methodology findings requiring no code change, 3 Rule 1 bugs in the newly-written test/script code itself)
**Impact on plan:** All five were necessary to make the plan's own stated acceptance criteria genuinely true rather than superficially green. No scope creep -- deviation 4 in particular generalizes an acceptance criterion the plan stated for one id (`shortcut-to-a-folder`) to all nine, which is what the criterion's own stated purpose ("the ids come from loadJourneyTraps(), never re-typed in the test") actually requires.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The full nine-trap fixture and its assertion suite are ready for 207-03+ to drive the actual multi-turn `claude -p` journey against.
- **Load-bearing for later plans:** the practice project always lives at `<destination>/repo`, never at `<destination>` itself (Plan 01's own layout, continued here) -- any later plan's own verification commands must account for this subdirectory, exactly as this plan had to correct for in Deviation 1.
- **Load-bearing for later plans:** `aether init`/`aether update --force` write no project `.gitignore` -- `.aether/` is untracked by construction in the practice project, not because of any `.gitignore` rule. A later plan constructing a new trap under `.aether/data/` needs no git operations at all; one touching the project's own real source tree does, and must stage an explicit, named path, never `-A`/`.`.
- **Load-bearing for later plans:** two hidden commands now exist specifically for fixture construction (`journey-seed-stale-survey`, `journey-seed-superseded-plan`) -- if a future trap similarly needs a self-authenticating on-disk shape, extend this file rather than hand-typing JSON that the runtime's own validators would reject or silently fail to trip.
- UED-07 is NOT marked complete in REQUIREMENTS.md by this plan -- `gsd-tools.cjs query requirements.ready-ids` reports it `blocked` (shared with 207-01 and 207-06). Whichever plan the tool judges ready should mark it.
- No blockers.

---
*Phase: 207-messy-practice-project-gate*
*Completed: 2026-09-22*

## Self-Check: PASSED

All 5 created/modified files verified present with `[ -f ]`; both task commit hashes (`d28e9f4e`, `05631cc2`) verified present in `git log --oneline --all`.
