---
phase: 208-never-a-dead-end
plan: 02
subsystem: cli
tags: [go, continue, verification, criteria, refusal]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refuse(), refusalRegistry, renderRefusal) both fixes in this plan route their new blocking message through."
provides:
  - "A per-task Verified flag (taskVerifiedFromCriteria) computed from that task's own bound, enforced criteria -- a criterion bound to one task no longer marks a sibling task unverified"
  - "One named blocking-issue line per failing criterion (causalCriterionSummary), never a duplicate generic 'no implementation evidence' line for the same cause"
  - "A `cd <dir> &&`-prefixed verification-commands line (splitVerificationCommandDirectoryPrefix) is parsed, classified, and actually run in that directory, instead of being silently dropped"
  - "A labelled verification-commands line whose command still can't be classified is collected (codexVerificationCommands.Unreadable) and surfaced as a named refusal (verification-command-not-understood) instead of a silent 'no command configured'"
affects: [208-03, 208-04, 208-05, 208-06, 208-07, 208-08]

# Actuals (#2632)
actuals:
  tokens: 9760
  tasks: 2
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-task verdict decoupled from the phase-wide checksPassed flag via a small map (criterionVerdictsByTask) computed once outside the per-task loop"
    - "A wrapper function (runVerificationStepInDir) around an unchanged existing function (runVerificationStep) adds new behaviour without touching any of that function's ~10 existing call sites/tests"
    - "codexVerificationCommands.Unreadable collects a whole original markdown line by value so the eventual refusal message can quote it verbatim"

key-files:
  created:
    - cmd/continue_task_verdict_test.go
    - cmd/verification_command_prefix_test.go
  modified:
    - cmd/codex_continue.go
    - cmd/deterministic_floor.go
    - cmd/refusal_register.go
    - cmd/boundary_double_dispatch_test.go

key-decisions:
  - "taskVerifiedFromCriteria takes the phase-wide checksPassed flag and per-task criterion verdicts as given (not re-derived from deterministic_floor.go, which this plan deliberately leaves unchanged) -- the per-task/gate-dedup fix is scoped to assessCodexContinue's own downstream classification and blocking-issue construction, exactly as the plan's action text specifies; deterministic_floor.go's own criteria.Enforced/criteria.Passed fold (which still flips the phase-wide flag false on ANY enforced criterion, task-bound or not) is untouched and keeps gating phase advancement."
  - "runVerificationStepInDir wraps the existing runVerificationStep rather than adding a directory parameter to its signature -- runVerificationStep has ~10 existing call sites across cmd/verify_out_of_band.go and multiple test files that this plan's files_modified list does not include; the wrapper gets directory support with zero risk to any of them."
  - "parseLabeledVerificationCommand now bounds the text before the first colon to 2 words / 20 chars before treating it as a label attempt. Without this, a Windows path with a drive letter ('go build -o C:\\Users\\...') was misread as a labelled 'build:' line with an unreadable command, discarding the command a later unlabelled-fallback branch would otherwise have correctly extracted -- caught by the pre-existing TestExtractVerificationCommandsPreservesMidLineBackslash regression guard."
  - "mergeCodexVerificationCommands now accumulates Unreadable across every source file merged in (AGENTS.md, CLAUDE.md, .codex/CODEX.md, .opencode/OPENCODE.md, codebase.md) rather than first-source-wins like Build/Type/Lint/Test -- an unreadable line in a later-merged file must still be named even if an earlier file already resolved a different check."
  - "WINDOWS.md rows 50 and 51 (the two field reports this plan closes) marked fixed via `gsd-tools windows fixed`."

requirements-completed: [UED-10, UED-12]

coverage:
  - id: D1
    description: "A task's Verified field, and the outcome classifyContinueTaskAssessment derives from it, comes from that task's own bound criteria -- a criterion bound to one task never marks a sibling task unverified, and a phase-level criterion still blocks the phase without falsely marking any individual task implemented_unverified"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/continue_task_verdict_test.go#TestOneFailingCriterionMarksOnlyItsOwnTask"
        status: pass
      - kind: unit
        ref: "cmd/continue_task_verdict_test.go#TestPhaseLevelCriterionDoesNotUnverifyEveryTask"
        status: pass
    human_judgment: false
  - id: D2
    description: "One failing criterion produces exactly one blocking-issue line naming it, never a second, generic line for the same cause"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/continue_task_verdict_test.go#TestFailingCriterionProducesOneNamedFailureNotTwo"
        status: pass
    human_judgment: false
  - id: D3
    description: "A `cd <dir> &&`/`cd <dir>;` prefix (quoted or bare) on a verification-commands line is recognised, the remainder is classified and stored, and the command actually runs in that directory; an absolute or path-escaping prefix is refused by name instead of run"
    requirement: "UED-12"
    verification:
      - kind: unit
        ref: "cmd/verification_command_prefix_test.go#TestVerificationCommandPrefixCheckCanFail"
        status: pass
      - kind: unit
        ref: "cmd/verification_command_prefix_test.go#TestVerificationCommandWithADirectoryPrefixRuns"
        status: pass
    human_judgment: false
  - id: D4
    description: "A labelled verification-commands line whose command still can't be classified is named on screen as a refusal (never silently reported as no command configured), while a project with genuinely nothing configured is unaffected"
    requirement: "UED-12"
    verification:
      - kind: unit
        ref: "cmd/verification_command_prefix_test.go#TestUnreadableVerificationLineIsNamedNotDropped"
        status: pass
      - kind: unit
        ref: "cmd/verification_command_prefix_test.go#TestNoVerificationSectionStillWarnsQuietly"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 02: Per-Task Criterion Verdicts and cd-Prefixed Verification Commands Summary

**Closes two real field reports (WINDOWS.md rows 50 and 51) from an external project running Aether 1.0.88: a failing check now unverifies only the task whose own criterion failed and blames it once, and a verification command that starts with `cd <dir> &&` now actually runs instead of being silently dropped.**

For the owner, in plain English: when someone else's project checked its work and one requirement failed, Aether used to mark every task on that check as failed for the same one reason -- so it looked like two things were broken when only one was, and Aether's automatic repair chased a problem that did not exist. Now only the task whose own requirement failed is marked failed; every other task is left alone. Separately, if a project's own instructions told Aether to run a check from inside a subfolder (`cd server && ...`), Aether used to just give up and say no check was configured at all, even though one was right there. Now it runs the check in that subfolder, and if a check's instructions are worded in a way Aether genuinely can't understand, it says so plainly instead of pretending nothing was set up.

## Performance

- **Duration:** ~45 min
- **Started:** 2026-09-22 (approx, first file read)
- **Completed:** 2026-09-22T18:47:13+02:00
- **Tasks:** 2
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments
- `criterionVerdictsByTask` / `taskVerifiedFromCriteria` (`cmd/codex_continue.go`): a task's Verified flag now comes from that task's own bound, enforced criteria combined with the phase-wide checksPassed flag -- a criterion bound to one task no longer unverifies its siblings, and an unbound (phase-level) criterion still blocks the phase without falsely blaming any one task.
- `causalCriterionSummary` plus a de-duplicated, first-seen-order `BlockingIssues` (`uniqueStringsPreserveOrder`): one failing criterion now produces exactly one named blocking-issue line, never a second generic "no implementation evidence" line for the identical cause.
- `splitVerificationCommandDirectoryPrefix` + `runVerificationStepInDir` (`cmd/codex_continue.go`): a `cd <dir> &&` or `cd <dir>;` prefix (quoted or bare) on a verification-commands line is recognised, the remainder is classified normally, and the check runs in that directory -- an absolute or `../`-escaping path is refused by name instead of run or silently ignored.
- `codexVerificationCommands.Unreadable` + `applyUnreadableVerificationCommandRefusals` (`cmd/deterministic_floor.go`): a labelled line whose command still can't be classified is named on screen through a new refusal row (`verification-command-not-understood`, `aether patrol`), replacing the old silent "no command configured" outcome for that specific case -- a project with genuinely nothing configured is untouched.
- `mergeCodexVerificationCommands` now carries the new per-kind `*Dir` fields and accumulates `Unreadable` across every merged source file (AGENTS.md, CLAUDE.md, `.codex/CODEX.md`, `.opencode/OPENCODE.md`, codebase.md), so a later file's unreadable line is never dropped just because an earlier file already resolved a different check.

## Task Commits

Each task ran RED-then-GREEN (TDD):

1. **Task 1a: failing tests for per-task criterion verdicts** - `49c1a0d5` (test)
2. **Task 1b: compute a task's verdict from its own bound criteria** - `b593af58` (feat)
3. **Task 2a: failing tests for cd-prefixed verification commands** - `371a988e` (test)
4. **Task 2b: read a cd-prefixed verification command instead of dropping it** - `e6a48c46` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `cmd/continue_task_verdict_test.go` - Task 1's tests (two-task fixture, per-task verdict, one-named-failure)
- `cmd/verification_command_prefix_test.go` - Task 2's tests (split-prefix guard, real directory-aware run, unreadable-line refusal, quiet-when-unconfigured)
- `cmd/codex_continue.go` - `criterionVerdictsByTask`, `taskVerifiedFromCriteria`, `causalCriterionSummary`, `uniqueStringsPreserveOrder`, `splitVerificationCommandDirectoryPrefix`, `runVerificationStepInDir`; `codexVerificationCommands` gains `BuildDir/TypeDir/LintDir/TestDir/Unreadable`; `codexVerificationStep` gains `WorkingDir`; `parseLabeledVerificationCommand`/`parseVerificationCommandTableLine`/`looksLikeVerificationCommand`/`detectVerificationCommandKind`/`setVerificationCommand`/`extractVerificationCommands`/`mergeCodexVerificationCommands` all updated
- `cmd/deterministic_floor.go` - `applyUnreadableVerificationCommandRefusals`, `unreadableLineForKind`, `verificationCommandLineKind`; the four `runVerificationStep` calls switched to `runVerificationStepInDir`
- `cmd/refusal_register.go` - new `verification-command-not-understood` row
- `cmd/boundary_double_dispatch_test.go` - one pre-existing assertion switched from `!=` to `reflect.DeepEqual` (see Deviations)

## Decisions Made
See `key-decisions` in the frontmatter above: the per-task fix is scoped to `assessCodexContinue`'s downstream classification, not the floor's own aggregate criteria fold; `runVerificationStepInDir` wraps rather than modifies `runVerificationStep`'s signature; the labelled-line-length bound on `parseLabeledVerificationCommand`; and `mergeCodexVerificationCommands`'s accumulate-not-overwrite behaviour for `Unreadable`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Adding `Unreadable []string` made `codexVerificationCommands` non-comparable with `!=`**
- **Found during:** Task 2, `go vet ./cmd/`
- **Issue:** `cmd/boundary_double_dispatch_test.go` compared two `codexVerificationCommands` values with `!=`, which is a compile error once the struct carries a slice field.
- **Fix:** Switched that one assertion to `reflect.DeepEqual`, added the `reflect` import.
- **Files modified:** cmd/boundary_double_dispatch_test.go
- **Verification:** `go vet ./cmd/` clean; the test still passes and still fails if `resolveCodexVerificationCommands` becomes unstable across calls (unchanged assertion intent).
- **Committed in:** e6a48c46

**2. [Rule 1 - Bug] Label-detection false positive on a Windows path containing a colon**
- **Found during:** Task 2, first full test run (`TestExtractVerificationCommandsPreservesMidLineBackslash` regressed)
- **Issue:** Making `parseLabeledVerificationCommand` preserve its recognised `kind` even when the command is unreadable (needed to collect genuinely unreadable lines) meant `go build -o C:\Users\test\aether.exe ./cmd/aether` was misread as a labelled `build:` line (the colon after the Windows drive letter `C` was mistaken for a label separator), discarding the command a later unlabelled-fallback branch would otherwise have correctly extracted.
- **Fix:** Added `verificationCommandLabelMaxChars`/`verificationCommandLabelMaxWords` (20 chars / 2 words) bounding how long the text before the first colon may be before it's treated as a label attempt at all -- a genuine label ("tests", "Build Command") is always short; a full command line with flags never is.
- **Files modified:** cmd/codex_continue.go
- **Verification:** `TestExtractVerificationCommandsPreservesMidLineBackslash` passes again; `TestUnreadableVerificationLineIsNamedNotDropped` still passes (a genuinely short, unreadable label is still caught).
- **Committed in:** e6a48c46

**3. [Rule 1 - Bug] `mergeCodexVerificationCommands` silently dropped the new fields**
- **Found during:** Task 2, `TestVerificationCommandWithADirectoryPrefixRuns`/`TestUnreadableVerificationLineIsNamedNotDropped` initially failing with empty `commands.Test`/`commands.Unreadable`
- **Issue:** `resolveCodexVerificationCommands` merges each source file's parsed `codexVerificationCommands` into one accumulator via `mergeCodexVerificationCommands`, which only copied `Build/Type/Lint/Test` -- the new `*Dir` and `Unreadable` fields were silently discarded on every merge.
- **Fix:** Extended `mergeCodexVerificationCommands` to carry the per-kind directory alongside its command (first-source-wins, matching the command itself) and to accumulate `Unreadable` across every source (never overwritten).
- **Files modified:** cmd/codex_continue.go
- **Verification:** Both tests pass; `TestResolveCodexVerificationCommandsReadsCODEXAndOPENCODEDocs` (multi-file merge) still passes unchanged.
- **Committed in:** e6a48c46

---

**Total deviations:** 3 auto-fixed (all Rule 1 bugs surfaced by the plan's own test suite, none touching scope or architecture). **Impact on plan:** None -- all three were implementation-detail corrections required for the new fields/logic to actually reach the code paths that consume them.

## Issues Encountered

**Pre-existing, unrelated known-red confirmed during verification (all four independently re-verified against a disposable worktree pinned to the commit immediately before this plan's work, `f7f7f334`, per `.planning/WINDOWS.md` row 12's known-red list from Phase 202):**
- `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptNeverOverwritesTheFirstResult`, `TestNoSecondAutomaticFixAttempt`, `TestFixAttemptIsCountedSeparately` (`cmd/floor_fix_attempt_test.go`) -- all fail identically on the pre-existing commit, unrelated to this plan's files.
- `TestResolveTestCommand_GoProject` (`cmd/gate_test.go`) -- fails because it reads this repository's own real `CLAUDE.md`, whose "Verification Commands" prose now contains the literal substring "go test ./..." inside a warning paragraph; `extractTestCommand` (`cmd/gate.go`) is a separate, cruder parser this plan does not touch. Also confirmed failing identically pre-existing.
- `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput` (`cmd/golden_workflow_test.go`) -- golden-file mismatches from earlier, unrelated visual-voice work; also on WINDOWS.md row 12's known-red list.

None of these touch a file this plan modified (`cmd/codex_continue.go`, `cmd/deterministic_floor.go`, `cmd/refusal_register.go`, `cmd/boundary_double_dispatch_test.go`, or either new test file).

## Known Stubs

None.

## Mutation Proofs

- **Task 1:** with `taskVerifiedFromCriteria` temporarily forced to `return checksPassed` (the old phase-wide-only behaviour, ignoring `byTask` entirely), `TestOneFailingCriterionMarksOnlyItsOwnTask` fails: task 1's own criterion failed but the forced function reports it Verified regardless (observed failure: "expected task 1 (whose own criterion failed) to be Verified=false, got true"). Reverted; test passes again.
- **Task 2:** with `splitVerificationCommandDirectoryPrefix` temporarily forced to `return "", "", false` unconditionally, all four sub-tests of `TestVerificationCommandWithADirectoryPrefixRuns` fail (the directory is never recognised, so the real-run sub-test finds an empty `commands.Test` and the three refusal sub-tests find no `dir` at all). Reverted; test passes again.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

`.planning/WINDOWS.md` rows 50 and 51 marked `fixed`. Plans 208-03 through 208-08 (per this plan's `affects` list) can build on the refusal contract from 208-01 and this plan's `Unreadable`/refusal wiring without re-deriving either.

No blockers.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*

## Self-Check: PASSED

- Both created files verified present on disk (`[ -f ]`): `cmd/continue_task_verdict_test.go`, `cmd/verification_command_prefix_test.go`.
- All 4 task commit hashes (49c1a0d5, b593af58, 371a988e, e6a48c46) verified present in `git log --oneline --all`.
- Re-ran acceptance-criteria tests: `TestOneFailingCriterion*`, `TestPhaseLevelCriterion*`, `TestFailingCriterionProduces*`, `TestVerificationCommand*`, `TestUnreadableVerificationLine*`, `TestAssessCodexContinue*`, `TestNoVerificationSectionStillWarnsQuietly` all pass.
- Re-ran plan-level `<verification>`: `go build ./cmd/aether` and `go vet ./cmd` clean; `go test ./cmd -run 'TestOneFailingCriterion|TestPhaseLevelCriterion|TestFailingCriterionProduces|TestVerificationCommand|TestUnreadableVerificationLine|TestAssessCodexContinue|TestRunDeterministicFloor|TestDeriveVerificationScope' -count=1 -timeout 8m` passes (the last two name patterns match zero tests, per Go's `-run` semantics for an unmatched regex -- no failures).
