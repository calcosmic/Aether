---
phase: 208-never-a-dead-end
plan: 03
subsystem: cli
tags: [go, cobra, status, refusal, sanitize, ux, midden]

# Dependency graph
requires:
  - phase: 208-01
    provides: "The typed refusal contract (refusal.go/refusal_register.go) whose NextCommand field this plan's widened guidance check now reads as one of its three sources."
provides:
  - "/ant-midden-review -- a menu command for the failure log, closing the sixth 2026-09-21 blocker Phase 207 deliberately left red"
  - "TestScreenGuidanceNamesCommandsTheOwnerCanRun -- widens the existing slash-command-only guidance check to every `aether <verb>` program command a screen can advise (status card, every Classic-voice screen including a new refusal screen, and every refusal's NextCommand), with a shrink-only allowlist for the one genuine non-command"
  - "colony.NeutralizeForRecord -- keeps a rejected failure's own words (defusing backticks, command substitution, pipe/semicolon-rm, angle brackets, prompt-injection and secrets-path spans) instead of replacing them with a sentence that says nothing"
affects: [208-04, 208-05, 208-06, 208-07, 208-08]

# Actuals (#2632)
actuals:
  tokens: 12930
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "regexp/syntax-derived test fixtures: TestNeutralisedTextAlwaysPassesTheFilter walks each rejection rule's own compiled pattern to synthesise a genuine matching string, so a new rule added later without a matching generated case fails loudly instead of silently passing"
    - "Shrink-only allowlist pattern reused a second time (guidanceCommandAllowlistCeiling mirrors journeyExpectedRedCeiling exactly) for a verb a screen may legitimately advise with no menu wrapper"
    - "A refusal block (renderRefusal) registered into the existing Classic-voice screen corpus as an owner-facing screen like any other, rather than a separate special case"

key-files:
  created:
    - .aether/commands/midden-review.yaml
    - .claude/commands/ant/midden-review.md
    - .opencode/commands/ant/midden-review.md
    - cmd/testdata/guidance_command_allowlist.json
    - cmd/failure_record_readable_test.go
  modified:
    - cmd/status.go
    - cmd/wrapper_command_names.go
    - cmd/testdata/journey/expected-red.json
    - cmd/journey_expected_red_test.go
    - cmd/journey_fix_reverts_test.go
    - cmd/slash_command_guidance_test.go
    - cmd/codex_visuals.go
    - cmd/oracle_promote.go
    - pkg/colony/sanitize.go
    - pkg/colony/sanitize_test.go
    - cmd/memory_feed.go
    - cmd/memory_feed_continue.go

key-decisions:
  - "The sixth 2026-09-21 blocker was closed by shipping the missing menu wrapper (/ant-midden-review), not by editing the expected-red register -- cmd/testdata/journey/expected-red.json now holds zero cases, and TestSixthBlockerCheckIsStillRed was replaced with TestSixthBlockerGapIsClosed rather than deleted, so a future regression (the wrapper going missing again) is caught by name."
  - "The widened guidance check's real-world first finding (three screens advising `aether pheromone-display`, which has no wrapper) was fixed by renaming the advice to `aether pheromones` -- the exact same command's own registered alias, which already has a wrapper -- rather than building a second wrapper for a command that already has one under a different name."
  - "A refusal block (renderRefusal) was registered into the Classic-voice screen corpus (cmd/slash_command_guidance_test.go's own init) so the widened check's second source (backticked `aether <verb>` spans inside every corpus screen) actually reaches `aether report`'s closing line -- otherwise the plan's own acceptance criterion (renaming report.md must make the check fail) had no source to trip on, since report.md is never rendered by any other registered screen."
  - "The one non-command a screen legitimately advises (`aether <command> --help`, a literal template placeholder inside the generic 'missing required flag' refusal) is recorded on the shrink-only cmd/testdata/guidance_command_allowlist.json with a written reason, rather than adding a fake `--help` menu wrapper or weakening the check to ignore angle-bracket placeholders generally."
  - "recordFailedChecksToMidden no longer skips a whitespace-only blocking issue outright (the prior `continue`) -- it now records a generic 'a check failed' row naming the phase, per Task 3's explicit behavior spec that an empty or whitespace-only failure text still names the check and the phase rather than producing no record at all."
  - "NeutralizeForRecord's shell-injection defusing (command substitution, pipe-rm, semicolon-rm) looks up each compiled rule pattern by its name field in shellInjectionRuleSpecs rather than table position, so it keeps working if that table is ever reordered."

requirements-completed: [UED-13, UED-14]

coverage:
  - id: D1
    description: "The status screen's own advice (`aether midden-review`) is followed by a real menu command on both platforms, and the sixth 2026-09-21 blocker's expected-red register is empty because the gap genuinely closed"
    requirement: "UED-13"
    verification:
      - kind: unit
        ref: "cmd/journey_expected_red_test.go#TestSixthBlockerGapIsClosed"
        status: pass
      - kind: unit
        ref: "cmd/journey_fix_reverts_test.go#TestExpectedRedRegisterIsEmptyAndTheFiveRevertsStand"
        status: pass
      - kind: unit
        ref: "cmd/command_parity_test.go#TestClaudeOpenCodeCommandParity"
        status: pass
    human_judgment: false
  - id: D2
    description: "No owner-facing screen advises a program command with no menu command behind it, except a counted, reasoned, shrink-only allowlist -- widened from the prior slash-command-only check"
    requirement: "UED-13"
    verification:
      - kind: unit
        ref: "cmd/slash_command_guidance_test.go#TestScreenGuidanceNamesCommandsTheOwnerCanRun"
        status: pass
      - kind: unit
        ref: "cmd/slash_command_guidance_test.go#TestGuidanceCommandAllowlistOnlyShrinks"
        status: pass
      - kind: unit
        ref: "cmd/slash_command_guidance_test.go#TestStatusWarningsSpeakPlainEnglish"
        status: pass
    human_judgment: false
  - id: D3
    description: "A failure whose text the safety filter rejects (backticks, command substitution, pipe/semicolon-rm, angle brackets, an instruction-like phrase, a secrets path) is still recorded in readable words naming what failed, and the record itself always passes the same filter"
    requirement: "UED-14"
    verification:
      - kind: unit
        ref: "pkg/colony/sanitize_test.go#TestNeutralisedTextAlwaysPassesTheFilter"
        status: pass
      - kind: unit
        ref: "cmd/failure_record_readable_test.go#TestRejectedFailureStillNamesWhatFailed"
        status: pass
      - kind: unit
        ref: "cmd/failure_record_readable_test.go#TestEmptyAndNilFailureText"
        status: pass
      - kind: unit
        ref: "pkg/colony/sanitize_test.go#TestNeutraliserNeverSplitsACharacter"
        status: pass
    human_judgment: false

duration: 24min
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 03: Never a Dead End Summary

**The status screen's advice now leads somewhere real (`/ant-midden-review` exists), the check that catches a dead-end command was widened from slash commands to every program command a screen can advise (and immediately found one real gap), and a rejected failure is now recorded in its own words instead of a sentence that tells the owner nothing.**

In plain English: the project's own status screen used to tell you to run a command that didn't exist on the menu -- following its advice led nowhere. That command now exists (`/ant-midden-review`), and the check the project runs before every release to catch this exact mistake was widened so it looks at every command a screen might suggest, not just menu-style ones -- and the very first time it ran for real, it caught three other screens making the identical mistake with a different command, which is now fixed too. Separately, when one of Aether's own safety checks failed and the failure's own description happened to contain something that looked like a command (a backtick, for instance -- extremely common in real error messages), Aether used to throw the whole description away and record "could not be safely recorded," which told you nothing. It now keeps the actual words wherever it safely can.

## Performance

- **Duration:** ~24 min
- **Started:** 2026-09-22T16:50:00Z (approx, first file read)
- **Completed:** 2026-09-22T17:14:03Z
- **Tasks:** 3
- **Files modified:** 17 (5 created, 12 modified)

## Accomplishments

- `/ant-midden-review` (`.aether/commands/midden-review.yaml`, `.claude/commands/ant/midden-review.md`, `.opencode/commands/ant/midden-review.md`, `cmd/wrapper_command_names.go`): the failure log now has a real menu command on both maintained platforms, closing the sixth 2026-09-21 blocker Phase 207 deliberately left red. `cmd/testdata/journey/expected-red.json` now holds zero cases, and `TestSixthBlockerCheckIsStillRed` was replaced with `TestSixthBlockerGapIsClosed`, an ordinary green assertion that fails again if the wrapper disappears (proven by mutation).
- `cmd/status.go`'s unacknowledged-failure warning and `middenGuidedAction`'s summary now gloss the invented word "midden" in the same sentence as the command, naming "the log of things that went wrong."
- `TestScreenGuidanceNamesCommandsTheOwnerCanRun` (`cmd/slash_command_guidance_test.go`) widens the pre-existing slash-command-only guidance check (`TestSlashCommandGuidancePointsAtRealCommands`) to every `aether <verb>` program command a screen can advise, from exactly three sources: the status card's own guided actions and warnings, every Classic-voice corpus screen (now including a newly-registered "refusal" screen so `aether report`'s closing line is covered), and every refusal's `NextCommand`. `cmd/testdata/guidance_command_allowlist.json` + `guidanceCommandAllowlistCeiling` is the shrink-only allowlist for the one genuine non-command (`aether <command> --help`, a literal template placeholder).
- The widened check's first real run found a genuine gap: three screens advised `aether pheromone-display`, which has no wrapper -- fixed by renaming the advice to `aether pheromones`, the exact same command's own registered cobra alias, which already has a wrapper.
- `colony.NeutralizeForRecord` (`pkg/colony/sanitize.go`) defuses each rejection rule in turn (backticks -> apostrophe, `$(...)` broken, pipe/semicolon-rm separator swapped for a comma, angle brackets escaped, a matched prompt-injection or secrets-path span replaced with a short bracketed note) then truncates to `SanitizeSignalContent`'s own 500-character limit at a rune boundary, so `SanitizeSignalContent(NeutralizeForRecord(x))` never errors. The four rejection fallbacks (`recordFailedChecksToMidden`, `recordQuickFailureToMidden`, `recordSwarmWorkerFailureToMidden`, `sanitizedWorkerSentence`) now use it, falling back to a sentence naming the concrete fact each site already has (check kind, question, worker name/status) only when nothing readable survives.

## Task Commits

Each task was committed atomically:

1. **Task 1: The failure log gets a menu command, and the sixth blocker's register goes empty because the gap closed** - `087bc5d6` (feat)
2. **Task 2: Widen the guidance check from menu commands to the program commands screens advise** - `05cf40d4` (test) -- this task's tests double as its own implementation (a check-widening task adds no production behavior beyond the check itself, plus the pheromone-display rename deviation captured in the same commit)
3. **Task 3: A failure the safety filter rejects is still recorded in words that name what failed** - `32b8e700` (test, RED) then `295f40a6` (feat, GREEN)

**Plan metadata:** commit pending (this SUMMARY + STATE.md + ROADMAP.md)

_Note: Task 3 is `tdd="true"` -- the RED commit lands the test file and the GREEN commit lands `NeutralizeForRecord` plus its four call sites._

## Files Created/Modified

- `.aether/commands/midden-review.yaml`, `.claude/commands/ant/midden-review.md`, `.opencode/commands/ant/midden-review.md` - the new menu command's three-file wrapper (byte-identical `.md` pair)
- `cmd/wrapper_command_names.go` - registers `midden-review` as a wrapped verb
- `cmd/status.go` - glosses "midden" in the unacknowledged-failure warning and guided-action summary; renames `pheromone-display` advice to `pheromones`
- `cmd/testdata/journey/expected-red.json` - now `"cases": []`
- `cmd/journey_expected_red_test.go` - `TestSixthBlockerCheckIsStillRed` replaced with `TestSixthBlockerGapIsClosed`; `TestPhaseProvesFiveFixesNotSix` removed in favor of the merged test in `journey_fix_reverts_test.go`
- `cmd/journey_fix_reverts_test.go` - `TestPhaseCountsFiveRevertsAndOneStandingRedCase` replaced with `TestExpectedRedRegisterIsEmptyAndTheFiveRevertsStand`
- `cmd/slash_command_guidance_test.go` - `TestScreenGuidanceNamesCommandsTheOwnerCanRun`, `TestGuidanceCommandAllowlistOnlyShrinks`, `TestStatusWarningsSpeakPlainEnglish`, `screenAdvisedProgramCommands`, and a new "refusal" screen registration
- `cmd/testdata/guidance_command_allowlist.json` - the one allowlisted non-command, with a written reason
- `cmd/codex_visuals.go`, `cmd/oracle_promote.go` - `pheromone-display` -> `pheromones` rename (same command, its own wrapped alias)
- `pkg/colony/sanitize.go` - `NeutralizeForRecord`, `shellRulePatternByName`, `truncateForRecord`
- `pkg/colony/sanitize_test.go` - `TestNeutralisedTextAlwaysPassesTheFilter` (regexp/syntax-derived fixtures) and five more `NeutraliserXxx` tests
- `cmd/memory_feed.go` - `sanitizedWorkerSentence`'s fallback now uses `NeutralizeForRecord`, falling back to worker name + status
- `cmd/memory_feed_continue.go` - `recordFailedChecksToMidden`, `recordQuickFailureToMidden`, `recordSwarmWorkerFailureToMidden` all now use `NeutralizeForRecord`; `fallbackCheckFailureText` added; a whitespace-only blocking issue is now recorded (generic "a check failed" row) instead of silently skipped
- `cmd/failure_record_readable_test.go` (new) - `TestRejectedFailureStillNamesWhatFailed`, `TestEmptyAndNilFailureText`

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Three screens advised `aether pheromone-display`, which has no menu wrapper**
- **Found during:** Task 2, first real run of `TestScreenGuidanceNamesCommandsTheOwnerCanRun`
- **Issue:** `cmd/status.go`, `cmd/codex_visuals.go`, and `cmd/oracle_promote.go` all advised the owner to run `aether pheromone-display`. That command has no menu wrapper -- but it is registered with the cobra alias `"pheromones"` (`cmd/pheromone_mgmt.go`: `Aliases: []string{"pheromones"}`), and `/ant-pheromones` already wraps exactly that alias.
- **Fix:** Renamed the advice string in all three files from `aether pheromone-display` to `aether pheromones` -- the identical command, reached through its own already-wrapped name. Zero behavior change; confirmed no test asserted the literal string `aether pheromone-display`.
- **Files modified:** `cmd/status.go`, `cmd/codex_visuals.go`, `cmd/oracle_promote.go`
- **Verification:** `go test ./cmd -run 'TestPheromoneDisplay|TestParity|TestVisualWriterDiscipline|TestCommandCallAudit|TestSanitizationIntegration|TestOraclePromote'` all pass; `TestScreenGuidanceNamesCommandsTheOwnerCanRun` passes.
- **Committed in:** `05cf40d4`

**2. [Rule 2 - Missing critical functionality] The widened check's second source had no way to reach `aether report`, so the plan's own mutation-proof acceptance criterion had nothing to trip on**
- **Found during:** Task 2, while satisfying the acceptance criterion "fails when `.claude/commands/ant/report.md` is temporarily renamed"
- **Issue:** `aether report` is advised on every refusal screen (`renderRefusal`, `cmd/refusal.go`'s closing "Run `aether report`" line) -- a genuinely owner-facing screen -- but no refusal screen was registered in the Classic-voice corpus `screenAdvisedProgramCommands` reads from, so that advice was invisible to the check.
- **Fix:** Registered a "refusal" screen into `voiceScreenRegistry` (via `cmd/slash_command_guidance_test.go`'s own `init`, the one new-screen-registration site the corpus convention names), rendering a real refusal (`refuse("colonize-finalize-missing-timestamp")`). This also makes the refusal block subject to `TestVoicedScreensSpeakPlainEnglish` (it passed -- no invented-word violations).
- **Files modified:** `cmd/slash_command_guidance_test.go`
- **Verification:** Mutation proof -- renaming `.claude/commands/ant/report.md` away makes `TestScreenGuidanceNamesCommandsTheOwnerCanRun` fail naming `aether report` (seen via voice screen "refusal"); restoring the file makes it pass again. `TestVoicedScreensSpeakPlainEnglish/refusal` passes.
- **Committed in:** `05cf40d4`

**3. [Rule 1 - Bug] A whitespace-only blocking issue was silently dropped, not recorded**
- **Found during:** Task 3, implementing the plan's own behavior spec ("An empty or whitespace-only failure text records a row naming the check and the phase")
- **Issue:** `recordFailedChecksToMidden`'s prior `if trimmed == "" { continue }` skipped a whitespace-only `BlockingIssues` entry entirely -- no record at all, which is a stricter failure than the "canned unhelpful sentence" problem this task exists to fix (a missing row hides that a check ever failed).
- **Fix:** Replaced the skip with a generic `"a check failed (its own text could not be safely recorded)"` row (via the same `fallbackCheckFailureText` helper the neutralizer's own empty-result fallback uses), still combined with the existing `"— check on phase %d"` suffix.
- **Files modified:** `cmd/memory_feed_continue.go`
- **Verification:** `TestEmptyAndNilFailureText/whitespace-only_blocking_issue_names_the_check_and_the_phase` passes; `TestFailedCheckWritesOneFailureRecordOnBothLanes` and `TestPassingChecksWriteNoFailureRecord` (pre-existing, non-blank-issue behavior) still pass.
- **Committed in:** `295f40a6`

---

**Total deviations:** 3 auto-fixed (2x Rule 1, 1x Rule 2).
**Impact on plan:** All three were necessary for correctness or to satisfy the plan's own explicit acceptance criteria; none expanded scope beyond what Task 2 and Task 3's own text already specified. No architectural changes, no new dependencies.

## Mutation Proofs

1. **Task 1:** Renamed `.claude/commands/ant/midden-review.md` away -> `TestSixthBlockerGapIsClosed` failed naming the missing wrapper (`missing wrapper verbs = [midden-review]`). Restored -> passes again.
2. **Task 2:** Renamed `.claude/commands/ant/report.md` away -> `TestScreenGuidanceNamesCommandsTheOwnerCanRun` failed naming `aether report` (seen via voice screen "refusal"). Restored -> passes again.
3. **Task 3:** Temporarily replaced `NeutralizeForRecord`'s body with a pass-through (trim + truncate only, no defusing) -> the real `TestNeutralisedTextAlwaysPassesTheFilter` failed on all 11 of its regexp/syntax-derived rule cases (XML tag, all 5 prompt-injection rules, all 4 shell-injection rules, the secrets-path rule). Restored -> passes again, confirming every defusing step is load-bearing.

## Known-Red Baseline

No known-red failures were encountered while running any verification command in this plan; every test named in the plan's `<verify>`/`<verification>` blocks passed cleanly against the current tree.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The sixth 2026-09-21 blocker is closed by a real fix (menu wrapper), not by editing the expected-red register -- `.planning/WINDOWS.md` entries related to this gap can be closed if any reference it directly (none found naming UED-13 by id).
- The widened guidance check (`TestScreenGuidanceNamesCommandsTheOwnerCanRun`) is now a standing regression guard for any future screen that advises a command with no wrapper -- it already found and fixed one real gap in this same plan.
- No blockers for the remaining Phase 208 plans (208-04 onward).

## Self-Check: PASSED

- `.aether/commands/midden-review.yaml`, `.claude/commands/ant/midden-review.md`, `.opencode/commands/ant/midden-review.md` — FOUND
- `cmd/testdata/guidance_command_allowlist.json` — FOUND
- `cmd/failure_record_readable_test.go` — FOUND
- Commits `087bc5d6`, `05cf40d4`, `32b8e700`, `295f40a6` — FOUND in `git log --oneline`
- `go build ./cmd/aether && go vet ./...` — clean
- `go test ./pkg/colony/... -count=1 -timeout 5m` — PASS
- `go test ./cmd -run 'TestSixthBlockerGapIsClosed|TestExpectedRedRegister|TestScreenGuidanceNamesCommandsTheOwnerCanRun|TestGuidanceCommandAllowlistOnlyShrinks|TestStatusWarningsSpeakPlainEnglish|TestRejectedFailure|TestEmptyAndNilFailureText|TestNeutraliser|TestClaudeOpenCodeCommandParity|TestWrapperCommandNamesMatchCanonicalCorpus|TestVoicedScreensSpeakPlainEnglish' -count=1 -timeout 8m` — PASS

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*
