---
phase: 208-never-a-dead-end
plan: 06
subsystem: cli
tags: [go, refusal, error-handling, cli, ast, ratchet]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refuse(), refusalRegistry, renderRefusal, the two exit lanes) every row and driver in this plan builds on"
  - phase: 208-02
    provides: "The cd-prefix verification-command parsing that this plan's verification-command-not-understood driver exercises"
  - phase: 208-05
    provides: "criterion-artifact-is-a-directory / criterion-binding-unsatisfiable rows and the directoryBindingTestPhase fixture this plan's driver reuses"
provides:
  - "refusalDeclaredFiles + the classification rule (ProtectsWork/Disposition/Reason on refusalRow) -- the one written-down rule every row's Reason is justified against"
  - "An AST enumerator (cmd/refusal_enumerate_test.go) that measures, rather than guesses, how many rule-applying error returns in the ten declared lifecycle files are not yet typed -- 411 measured, recorded as a two-sided shrink-only ratchet"
  - "25 of those 411 sites converted to typed refuse(...)/warnAndCarryOn(...) calls (14 new registry rows, one reused across colonize/continue/build's duplicate 'no plan yet' dead ends) -- floor now 386"
  - "warnAndCarryOn(r refusal) + renderWarning -- the 'noticed, not stopped' half of the contract, proven live on the three timeout-flag/env-var refusals"
  - "TestBehaviourMatchesTheRefusalTable -- drives 19 of 27 registry rows at their real call sites and fails if a row's Disposition is flipped without the code changing"
affects: [208-07, 208-08]

# Actuals (#2632)
actuals:
  tokens: 18870
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-sided shrink-only ratchet stored as a JSON fixture (cmd/testdata/refusals/untyped-floor.json), not a Go constant -- the count moves every time a later plan converts a site, and a JSON file is what a human reviewer re-measures against, not a re-typed literal"
    - "Push/pop function-name stack keyed off ast.Inspect's own nil-after-children contract, so a refusal site's enclosing function name survives nested closures without a second AST pass"
    - "warnAndCarryOn / renderWarning mirror refuse / renderRefusal's exact shape (same banner/divider helpers, same log append) but head as 'noticed' and return nothing -- the call site keeps running instead of returning an error"
    - "One refusal id, several call sites: no-project-plan and no-active-phase-to-continue each convert the identical duplicated dead end across codex_continue.go, codex_continue_finalize.go and codex_build_finalize.go"
    - "Table-driven behavioural proof with a declared, reason-carrying no-driver exception list (refusalRowsWithoutDrivers) that must stay shorter than the driven set -- mirrors the fixture-bank guard-index precedent, applied to behaviour instead of coverage"

key-files:
  created:
    - cmd/refusal_enumerate_test.go
    - cmd/refusal_behaviour_test.go
    - cmd/testdata/refusals/untyped-floor.json
  modified:
    - cmd/refusal_register.go
    - cmd/refusal.go
    - cmd/codex_colonize.go
    - cmd/codex_colonize_finalize.go
    - cmd/codex_colonize_test.go
    - cmd/codex_continue.go
    - cmd/codex_continue_finalize.go
    - cmd/codex_build_finalize.go
    - cmd/codex_plan_finalize.go
    - cmd/codex_workflow_cmds.go
    - cmd/criterion_evidence.go
    - cmd/entomb_cmd.go
    - cmd/init_cmd.go
    - cmd/session_flow_cmds.go
    - cmd/deterministic_floor.go
    - cmd/finality_parity_test.go

key-decisions:
  - "The floor is a JSON fixture (untyped 411 -> 386), not a Go const, because the plan's own two-sided-ratchet precedent (cmd/eval_gates.go's seedBankUnguardedFloor) is a constant that gets hand-edited in the same reviewed change; a moving cross-plan count reviewed by a human is better served by a diffable JSON file with its own required 'reason' field."
  - "The wrapping-vs-non-wrapping fmt.Errorf distinction is a pure go/ast heuristic (no %w verb, no argument that looks like an error by name/selector/Error()-call), deliberately matching cmd/rendered_fields_invariant_test.go's stated 'without needing go/types or a compiled build' discipline rather than pulling in go/packages for one scanner."
  - "Task 2's conversion set (25 sites) prioritised: every WINDOWS.md-49-53 site (already covered by 208-01/208-05's own rows), then real dead ends reachable from the fourteen journey steps' own commands, picked for a mix of pause/resume integrity, seal timeout flags, entomb archive verification, and the colonize/plan/build/continue precondition checks -- not an exhaustive pass, and the SUMMARY says so rather than claiming completeness."
  - "invalid-timeout-value (the three --worker-timeout/--verification-timeout/AETHER_CONTINUE_VERIFICATION_TIMEOUT <= 0 sites) is the one Disposition=warn conversion: a safe, already-existing built-in default exists for all three, so the call carries on with that default after warnAndCarryOn renders the notice, rather than stopping the command."
  - "Discovered while driving colonize-existing-survey-found for real: codex_colonize.go carries its own, unconverted copy of the same 'existing survey' dead end the direct (non --plan-only) 'aether colonize' dispatch lane actually hits -- distinct from codex_colonize_finalize.go's copy this plan's own Task 2 had already converted. Split into two rows (colonize-existing-survey-found / colonize-finalize-existing-survey-found) because the two lanes' real recovery commands differ (--force-resurvey vs --plan-only --force-resurvey); converting the wrong one to look fixed while leaving the real dead end live would have been exactly the failure mode this phase exists to close."
  - "Renamed two Task 2 rows: pauseColonyInMutationSession (the /ant-pause path) had been misnamed resume-colony-state-unconfirmed / resume-colony-state-not-runnable. Corrected to pause-colony-state-unconfirmed / pause-colony-state-not-runnable with Why text that actually describes pausing, not resuming."

requirements-completed: [UED-11]

coverage:
  - id: D1
    description: "One checked-in, id-sorted refusal table with a written classification rule; every row (27) carries a non-empty Reason and a Disposition of exactly stop or warn, and no row pairs ProtectsWork=true with Disposition=warn"
    requirement: "UED-11"
    verification:
      - kind: unit
        ref: "cmd/refusal_test.go#TestRefusalRegisterIsSortedAndUnique"
        status: pass
      - kind: unit
        ref: "cmd/refusal_enumerate_test.go#TestEveryRowCarriesAClassificationReason"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every rule-applying error return in the ten declared lifecycle files is either a registered refuse(...) call or counted on a two-sided, shrink-only floor (411 measured, never guessed); the anti-vacuity guard fails a broken enumeration rather than passing vacuously"
    requirement: "UED-11"
    verification:
      - kind: unit
        ref: "cmd/refusal_enumerate_test.go#TestEveryRefusalSiteIsRegisteredOrCounted"
        status: pass
      - kind: unit
        ref: "cmd/refusal_enumerate_test.go#TestUntypedRefusalFloorOnlyShrinks"
        status: pass
      - kind: unit
        ref: "cmd/refusal_enumerate_test.go#TestRefusalEnumerationCanFail"
        status: pass
    human_judgment: false
  - id: D3
    description: "25 sites converted: a refusal that does not protect against losing work (invalid-timeout-value) now warns and carries on with a safe default, proven at its own site; every other converted site stays a stop, also proven at its own site"
    requirement: "UED-11"
    verification:
      - kind: unit
        ref: "cmd/refusal_behaviour_test.go#TestBehaviourMatchesTheRefusalTable"
        status: pass
      - kind: unit
        ref: "cmd/refusal_behaviour_test.go#TestNoRefusalBothWarnsAndStops"
        status: pass
    human_judgment: false
  - id: D4
    description: "The checked-in table is not a document: flipping a row's Disposition without changing the program fails TestBehaviourMatchesTheRefusalTable by name (mutation-proven, see Mutation Proofs below); the same for planting a second stop+warn use of one id"
    requirement: "UED-11"
    verification:
      - kind: unit
        ref: "cmd/refusal_behaviour_test.go#TestBehaviourMatchesTheRefusalTable"
        status: pass
    human_judgment: true
    rationale: "The mutation itself (temporarily flipping a Disposition, planting a dual-use id, planting a non-wrapping error, editing the floor upward) was performed manually during this plan's own verification and reverted before committing -- recorded in the Mutation Proofs section below rather than as a standing, always-run test (a permanent 'break the code on purpose' test would itself be the bug it is proving against)."

duration: ~105min (estimate; exact start time not captured before the first file read)
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 06: Measure, Then Convert, Then Prove Summary

**Measured the real size of the "does every refusal name a next step" problem (411 untyped dead ends across ten files), converted 25 of them to the typed refusal contract -- one of them a genuine warn-and-continue -- and wrote a test that fails if the table and the program's real behaviour ever disagree; driving it for real caught two live bugs along the way (a duplicate unconverted dead end, and a refusal whose next command never actually reached the screen) and fixed both.**

For the owner, in plain English: before this, nobody actually knew how many places in Aether could still leave you at a dead end with no way forward -- it was a guess. This plan counted them for real (411, in the ten files that matter for the fourteen everyday steps), fixed 25 of the real ones you'd actually hit -- pausing or resuming with confusing internal state, a project with no plan yet, an archive that doesn't check out, a timeout you typed wrong, a survey that already exists -- and, for three of those (a mistyped timeout), taught Aether to just use its own safe default and keep going instead of stopping you at all. It also built a permanent check that keeps this honest going forward: if anyone (including a future me) ever changes what the table says a refusal does without actually changing the code to match, that check fails by name. Building that check caught two real bugs on the spot: one place where `aether colonize` had a second, unfixed copy of a dead end I'd already fixed elsewhere, and one place where the "here's what to run next" text was being built but never actually printed to the screen. Both are fixed now, not just found.

## Performance

- **Duration:** ~105 min (estimate)
- **Completed:** 2026-09-22T19:20:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 19 (3 created, 16 modified)

## Accomplishments

- `cmd/refusal_register.go`: the classification rule ("ProtectsWork is true when carrying on would (a) lose work, (b) record a false completion, or (c) spend money/reach outside the program") is written down once as `refusalRow`'s own doc comment; every one of 27 rows now carries a `Disposition` (`stop`/`warn`) and a one-line `Reason` justified against that rule. `refusalDeclaredFiles` names the ten lifecycle files behind the fourteen everyday steps as the plan's honest, out-loud scope.
- `cmd/refusal_enumerate_test.go`: a pure-`go/ast` enumerator (no `go/types`, matching this package's own stated discipline) walks the declared files and finds every rule-applying, non-wrapping `fmt.Errorf`/`errors.New`/`refuse(...)` return. Measured 411 untyped sites on the tree as shipped; recorded as a two-sided ratchet in `cmd/testdata/refusals/untyped-floor.json` (fails if the real count exceeds the floor, and fails if the floor sits above the real count -- it must equal reality exactly). Anti-vacuity: fails loudly if any declared file, or the whole enumeration, comes back empty.
- 25 of those 411 sites converted to typed `refuse(...)`/`warnAndCarryOn(...)` calls across `codex_colonize.go`, `codex_colonize_finalize.go`, `codex_continue.go`, `codex_continue_finalize.go`, `codex_build_finalize.go`, `codex_plan_finalize.go`, `codex_workflow_cmds.go`, `criterion_evidence.go`, `entomb_cmd.go`, `init_cmd.go`, `session_flow_cmds.go` -- floor now 386. 14 new registry rows, two reused ids (`no-project-plan`, `no-active-phase-to-continue`) each closing the identical duplicated dead end across three files.
- `cmd/refusal.go`: `warnAndCarryOn(r refusal)` + `renderWarning` -- the "noticed, not stopped" half of the contract. Proven on `invalid-timeout-value`: a zero/negative `--worker-timeout`/`--verification-timeout`/`AETHER_CONTINUE_VERIFICATION_TIMEOUT` now prints a notice and falls back to Aether's own built-in default instead of refusing the command outright.
- `cmd/refusal_behaviour_test.go`: `TestBehaviourMatchesTheRefusalTable` drives 19 of 27 rows at their real call sites (14 direct `refuse(...)` sites, 5 legacy pattern-matched rows driven through `renderVisualError`, the one real production caller of `friendlyErrorForPattern`) and asserts stop-rows produce a typed error naming their own id and next command with no continued work, warn-rows produce no error plus a rendered next command plus continued work. The remaining 8 rows are declared in `refusalRowsWithoutDrivers` with a written, honest reason each (heavy fixture cost, or -- for `criterion-binding-unsatisfiable` -- no `refuse(...)` call site at all), and that list is itself asserted shorter than the driven set. `TestNoRefusalBothWarnsAndStops` reuses the Task 1 AST walk to prove no id both stops and warns anywhere in the declared files.
- **Two live bugs found and fixed while driving the table for real** (not found by inspection -- found because the test tried to observe them and couldn't): `codex_colonize.go` had its own unconverted copy of the "existing survey" dead end (the one the direct `aether colonize` dispatch lane actually hits); and `deterministic_floor.go`'s unreadable-verification-command refusal built its `NextCommand` into the `refusal` struct but never actually printed it to the screen. Both fixed; see Deviations.

## Task Commits

1. **Task 1: Enumerate every refusal site and record what is not yet typed** - `5224a020` (feat)
2. **Task 2: Classify and convert -- a refusal that does not protect against losing work warns and carries on** - `cb62c471` (feat)
3. **Task 3: A test fails when the program's behaviour and the table disagree** - `9fd1ba41` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/refusal_enumerate_test.go` - AST enumerator, the two-sided floor ratchet, anti-vacuity, classification-reason gate
- `cmd/refusal_behaviour_test.go` - table-driven behavioural proof + the no-dual-use scan
- `cmd/testdata/refusals/untyped-floor.json` - the measured, shrink-only untyped count (411 -> 386) with its required reason
- `cmd/refusal_register.go` - classification rule doc comment, `Disposition`/`Reason` fields, `refusalDeclaredFiles`, 14 new rows, 2 renamed rows
- `cmd/refusal.go` - `warnAndCarryOn`, `renderWarning`
- `cmd/codex_colonize.go` - the direct-dispatch "existing survey" site converted (found via Task 3's driver, not the plan's own declared scope)
- `cmd/codex_colonize_finalize.go` - "existing survey" and "flag --completion-file" sites converted
- `cmd/codex_colonize_test.go` - `TestColonizeRequiresForceResurveyWhenSurveyExists` assertion updated to the refusal's own text
- `cmd/codex_continue.go` / `cmd/codex_continue_finalize.go` / `cmd/codex_build_finalize.go` - "no project plan" / "no active phase" / "flag --completion-file" sites converted (shared ids across files)
- `cmd/codex_plan_finalize.go` - "empty phase list" and "flag --completion-file" sites converted
- `cmd/codex_workflow_cmds.go` - the three timeout sites converted to `warnAndCarryOn`
- `cmd/criterion_evidence.go` - "missing success criteria" site converted
- `cmd/entomb_cmd.go` - two archive-integrity sites converted
- `cmd/init_cmd.go` - "charter field too long" site converted
- `cmd/session_flow_cmds.go` - four pause/resume integrity sites converted
- `cmd/deterministic_floor.go` - the unreadable-verification-command refusal's `NextCommand` now actually reaches the printed Summary text (Deviations)
- `cmd/finality_parity_test.go` - `TestFinalityParity_LoadExternalContinueCompletion_RejectsEmptyPath` assertion updated to the refusal's own text

## Decisions Made

See `key-decisions` in the frontmatter above -- summarised: JSON-fixture floor over a Go const; pure-AST wrapping heuristic (no `go/types`); a deliberately non-exhaustive, honestly-reported 25-site conversion set; one warn conversion where a safe default genuinely exists; and two correction decisions (the colonize direct-lane split, the pause/resume renaming) made while driving the table for real in Task 3.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `verification-command-not-understood`'s `ProtectsWork` was misclassified `false`**
- **Found during:** Task 1, filling in `Disposition`/`Reason` for the 13 pre-existing rows
- **Issue:** Continuing past an unreadable verification-commands line would silently skip one of a phase's own declared checks and still record the phase as checked -- ground (b), a false verification -- but the pre-existing row (from 208-01/208-02) carried `ProtectsWork: false`.
- **Fix:** Corrected to `true`, with `Disposition` staying `stop` (consistent -- `ProtectsWork: true` requires `stop`).
- **Files modified:** cmd/refusal_register.go
- **Verification:** `TestEveryRowCarriesAClassificationReason` passes; nothing reads `ProtectsWork` for control flow (confirmed by grep), so no behaviour changed, only the recorded classification.
- **Committed in:** 5224a020

**2. [Rule 1 - Bug] Two Task 2 rows misclassified as "resume" when the site actually guards "pause"**
- **Found during:** Task 3, `read_first` for the pause/resume drivers
- **Issue:** `pauseColonyInMutationSession` (line 367, session_flow_cmds.go) is called only from the `/ant-pause` path (confirmed: its one caller). Task 2 had registered its two checks as `resume-colony-state-unconfirmed` / `resume-colony-state-not-runnable`, describing them as guarding a resume.
- **Fix:** Renamed to `pause-colony-state-unconfirmed` / `pause-colony-state-not-runnable`, rewrote `Why`/`Reason` to describe pausing, updated the one call site.
- **Files modified:** cmd/refusal_register.go, cmd/session_flow_cmds.go
- **Verification:** `go build`/`go vet` clean; `TestPauseResume*` and `TestEveryRefusalSiteIsRegisteredOrCounted` (count unchanged -- a rename, not a new/removed site) both pass.
- **Committed in:** 9fd1ba41

**3. [Rule 1 - Bug] `codex_colonize.go` carried its own, unconverted copy of the "existing survey" dead end**
- **Found during:** Task 3, driving `colonize-existing-survey-found` for real via the existing `TestColonizeRequiresForceResurveyWhenSurveyExists` fixture pattern
- **Issue:** The real `aether colonize` (direct, non-`--plan-only`) dispatch lane hits `codex_colonize.go`'s own copy of the "existing territory survey found" check at two sites (lines 143, 354) -- entirely separate from `codex_colonize_finalize.go`'s copy this plan's Task 2 had already converted. This file was never in the plan's declared scope, so nothing would have caught it without actually driving the behaviour. The pre-existing test `TestColonizeRequiresForceResurveyWhenSurveyExists` was, in fact, failing on the unconverted bare-error text at the moment this was found (an earlier verification pass of mine had used a `-run` regex, `TestCodexColonize.*`, that does not match this test's real name -- a real verification gap, also corrected).
- **Fix:** Split into two registry rows since the two lanes' correct recovery commands differ (`aether colonize --force-resurvey` for the direct lane vs. `aether colonize --plan-only --force-resurvey` for the plan-only/finalize lane): `colonize-existing-survey-found` (direct) and `colonize-finalize-existing-survey-found` (plan-only). Converted both `codex_colonize.go` sites to `refuse("colonize-existing-survey-found")`.
- **Files modified:** cmd/codex_colonize.go, cmd/refusal_register.go, cmd/codex_colonize_finalize.go (id rename at its one call site), cmd/codex_colonize_test.go (assertion updated)
- **Verification:** `TestColonizeRequiresForceResurveyWhenSurveyExists` now passes for real (confirmed failing before the fix, passing after); `TestBehaviourMatchesTheRefusalTable/colonize-existing-survey-found` passes.
- **Committed in:** 9fd1ba41

**4. [Rule 1 - Bug] `verification-command-not-understood`'s `NextCommand` never reached the printed screen**
- **Found during:** Task 3, driving `verification-command-not-understood` via `runDeterministicFloor` (reusing 208-02's own fixture pattern)
- **Issue:** `applyUnreadableVerificationCommandRefusals` (cmd/deterministic_floor.go) built `steps[i].Summary` from `r.What + " " + r.Why` only -- `r.NextCommand` ("aether patrol") was computed by `refuse(...)` but never appended, so the owner's screen showed what went wrong and why, but not the one command that gets past it. Exactly the class of gap this whole phase exists to close, found by the test genuinely trying to observe it.
- **Fix:** Appended `" Next: \`<command>\`."` to the Summary, matching `renderRefusal`'s own shape.
- **Files modified:** cmd/deterministic_floor.go
- **Verification:** `TestBehaviourMatchesTheRefusalTable/verification-command-not-understood` passes; pre-existing `TestUnreadableVerificationLineIsNamedNotDropped` / `TestNoVerificationSectionStillWarnsQuietly` still pass (neither asserted on the old, incomplete Summary text).
- **Committed in:** 9fd1ba41

**5. [Rule 3 - Blocking] Two pre-existing tests asserted on the old bare error text at sites this plan converted**
- **Found during:** Task 2 verification
- **Issue:** `TestColonizeRequiresForceResurveyWhenSurveyExists` (codex_colonize_test.go) and `TestFinalityParity_LoadExternalContinueCompletion_RejectsEmptyPath` (finality_parity_test.go) asserted `strings.Contains(err.Error(), "<old bare text>")`. A typed `refusal`'s `Error()` is `"<What> — next: <NextCommand>"`, which does not contain the old wording.
- **Fix:** Updated both assertions to the refusal's own `What`/`NextCommand` text (and, for the second, additionally asserted `errors.As` resolves the exact row id) -- never weakened to a substring that would also match nothing, per the plan's own explicit acceptance criterion.
- **Files modified:** cmd/codex_colonize_test.go, cmd/finality_parity_test.go
- **Verification:** Both pass; see also deviation 3 above (the colonize one surfaced a real, separate bug on the way).
- **Committed in:** cb62c471 (finality_parity_test.go), 9fd1ba41 (codex_colonize_test.go, after deviation 3's fix made the real assertion possible)

---

**Total deviations:** 5 auto-fixed (4 Rule 1 bug fixes -- one classification correction, one naming correction, two genuine behavioural bugs found by driving the table for real; 1 Rule 3 test-assertion update explicitly sanctioned by the plan). **Impact on plan:** All within the plan's own stated scope and explicit allowances (Task 2's acceptance criteria pre-authorize updating a stale test assertion; Task 3's whole purpose is finding exactly this class of gap). No architectural changes; no scope creep beyond the two `codex_colonize.go` sites, which are the same dead end this plan's own Task 2 objective already targets, just discovered at a second call site.

## Issues Encountered

**Pre-existing, unrelated known-red found during verification:** `TestResumeOnAnArchivedProjectSaysThereIsNothingToResume` (cmd/entomb_archived_shell_test.go) fails on both its `fresh` and `older-version` subtests with a planning-migration bug ("revisions[0].plan_hash does not match its phase snapshot") that blocks seal from reaching `COMPLETED` in that fixture. Confirmed pre-existing and unrelated: ran identically at commit `2fda84b7` (immediately before this plan's first commit) in a disposable `git worktree`. Recorded as WINDOWS.md entry 54; not fixed here (out of scope -- nothing this plan touches is on the call path).

**A gap in my own verification, found and corrected:** an earlier `-run 'TestCodexColonize|...'` pass I ran to verify Task 2 did not actually match `TestColonizeRequiresForceResurveyWhenSurveyExists` (the real test name lacks "Codex"), so it silently never ran and I reported "all pass" while that test was, in fact, red. Found only because Task 3's driver tried to observe the same behaviour directly and could not. Corrected: the test now runs (confirmed both failing-before and passing-after) and is included in this plan's final verification sweep.

**Full unscoped `go test ./cmd -count=1 -timeout 90m` was not run** per this session's explicit harness instruction ("Never run the whole `./cmd` package suite... the orchestrator runs the full gate after the wave"), which overrides the plan's own `<verification>` text requesting it. In its place: every named test in the plan's `<verification>` block plus a wide regression sweep across every file this plan touched (`TestRefusal*`, `TestEveryRefusal*`, `TestBehaviourMatchesTheRefusalTable`, `TestNoRefusalBothWarnsAndStops`, `TestUntypedRefusalFloorOnlyShrinks`, `TestColonize*`, `TestCodexColonize*`, `TestCodexPlanFinalize*`, `TestCodexBuildFinalize*`, `TestCodexContinueFinalize*`, `TestEntomb*`, `TestPauseResume*`, `TestFinalityParity*`, `TestFriendlyError*`, `TestInitCmd*`, `TestSessionFlow*`, `TestCriterionEvidence*`, `TestDeterministicFloor*`, `TestCodexWorkflow*`, `TestResolveWorkerTimeout*`, `TestResolveContinueVerificationTimeout*`) -- all pass. `go build ./...` and `go vet ./cmd/` both clean. The orchestrator's post-wave full gate is the authoritative `discovered=N executed=N` check for this plan.

## Mutation Proofs

Performed manually during verification (temporary edit -> confirm the named test fails -> revert -> confirm the test passes again). None of the mutations were committed.

**1. Task 1 -- a non-wrapping `fmt.Errorf` planted in a declared file:** temporarily added a trivial function with a bare `fmt.Errorf("planted mutation probe")` return to `cmd/init_cmd.go`. Result: `TestEveryRefusalSiteIsRegisteredOrCounted` failed ("untyped refusal site count 412 exceeds the recorded floor 411"). Reverted with `git checkout -- cmd/init_cmd.go`; test passes again.

**2. Task 1 -- floor edited upward past the real count:** temporarily set `untyped-floor.json`'s count to 500 (well above the real 411 at that point). Result: `TestUntypedRefusalFloorOnlyShrinks` failed ("recorded ... count (500) is inflated above the real untyped count (411)"). Reverted; test passes again.

**3. Task 1 -- an empty `Reason`:** temporarily blanked the `Reason` field on the `no-colony-initialized` row. Result: `TestEveryRowCarriesAClassificationReason` failed by name (`refusal row "no-colony-initialized" has an empty Reason`). Reverted; test passes again.

**4. Task 3 -- a row's `Disposition` flipped without changing the code:** temporarily changed `invalid-timeout-value`'s `Disposition` from `warn` to `stop` (the real call sites still call `warnAndCarryOn`, unchanged). Result: `TestBehaviourMatchesTheRefusalTable/invalid-timeout-value` failed ("is Disposition=stop but the work after its site ran anyway"). Reverted; test passes again.

**5. Task 3 -- one id both stopping and warning:** temporarily added a function to `cmd/init_cmd.go` that both `return refuse("charter-field-too-long")` and `warnAndCarryOn(refuse("charter-field-too-long"))`. Result: `TestNoRefusalBothWarnsAndStops` failed by name (`refusal id "charter-field-too-long" is used both inside a return (stop) and as a warnAndCarryOn(...) argument (warn)`). Reverted with `git checkout -- cmd/init_cmd.go`; test passes again.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The refusal contract, its floor ratchet, and its behavioural proof are all in place; 386 untyped sites remain honestly counted (down from 411), and 27 registry rows are proven consistent with the real program. Phase 208 plans 07 and 08 (per the phase's affects list) can convert further sites against this same contract, or extend `refusalRowsWithoutDrivers`/add new drivers as their own scope requires, without re-deriving any of this machinery.

No blockers. One pre-existing, unrelated known-red (WINDOWS.md entry 54) is tracked, not a gate on this plan.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*

## Self-Check: PASSED

- All 3 files listed under key-files.created verified present on disk (`[ -f ]`).
- All 3 task commit hashes (5224a020, cb62c471, 9fd1ba41) verified present in `git log --oneline --all`.
- Re-ran acceptance-criteria tests: `TestEveryRefusalSiteIsRegisteredOrCounted`, `TestUntypedRefusalFloorOnlyShrinks`, `TestRefusalEnumerationCanFail`, `TestEveryRowCarriesAClassificationReason`, `TestBehaviourMatchesTheRefusalTable`, `TestNoRefusalBothWarnsAndStops`, `TestEveryRefusalRowNamesANextCommand`, `TestRefusalRegisterIsSortedAndUnique`, `TestFriendlyErrorsReadTheOneRefusalTable` all pass.
- Re-ran plan-level `<verification>` (scoped, per harness instruction against the full unscoped suite -- see Issues Encountered): `go build ./cmd/aether` and `go vet ./cmd` both clean; the named `go test -run 'TestRefusal|TestEveryRefusal|TestBehaviourMatchesTheRefusalTable|TestNoRefusalBothWarnsAndStops|TestUntypedRefusalFloorOnlyShrinks|TestCodexColonize|TestCodexPlanFinalize|TestCodexBuildFinalize|TestCodexContinueFinalize|TestEntomb'` block passes, plus the wider regression sweep listed in Issues Encountered.
- All 5 mutation proofs re-confirmed and reverted before committing.
