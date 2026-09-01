---
phase: 197-one-answer-to-what-next
plan: 03
subsystem: cli
tags: [go, hooks, session-start, next-action, reachability-ratchet, docs]

requires:
  - phase: 197-one-answer-to-what-next
    provides: "resolveNextAction and the availability gate (197-01); renderNextActionCard and nextActionInputForState (197-02)"
provides:
  - "hook-session-start — the runtime-owned greeting printed when a chat opens, resumes, or is carried on after a clear"
  - "nextCommandForHookState collapsed onto the one resolver, so all five deciders are now adapters"
  - "The reachability scanner reads .claude/settings.json, so a registered hook counts as a caller"
  - "TestSessionGreetingIsNotDelegatedToTheAssistant — the shape-matching guard that stops the removed instruction returning"
  - "TestColonyRulesCopiesStayIdentical — the first check that the two colony rules copies match"
affects: [197-04 closing renderers, 197-06 closing renderers, 197-07 cross-command envelope assertion]

actuals:
  tokens: 10600
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Registration-as-caller: a hook settings file parsed as data is D-01 caller evidence, so wiring a hook SHRINKS the tolerated-orphan list instead of growing it"
    - "Shape-matched document guard: three idea groups held as data, required within a 12-line window, so a paraphrase is caught as surely as a copy-paste"
    - "Guard-on-the-guard with a paraphrase fixture, proving the check matches shape rather than sentences"

key-files:
  created:
    - cmd/hook_session_start_test.go
    - cmd/session_start_doc_removal_test.go
  modified:
    - cmd/hook_cmds.go
    - cmd/subcommand_reachability_ratchet_test.go
    - cmd/next_action_one_decider_test.go
    - cmd/testdata/orphan_allowlist.json
    - .claude/settings.json
    - CLAUDE.md
    - AGENTS.md
    - .claude/rules/aether-colony.md
    - .aether/rules/aether-colony.md

key-decisions:
  - "The reachability scanner's shipped-hook corpus gained .claude/settings.json, so the three existing hook commands stopped being tolerated orphans and the frozen baseline was never touched"
  - "The instruction was in FOUR documents, not the plan's six — README.md and docs/phase3-section-walkthrough.md show example output of a command the owner runs, which is not an instruction to the assistant; both were left unmodified and both are still scanned by the guard"
  - "The regeneration flag wipes hand-written disposition reasons, so the 17 surviving reviewed reasons were restored after regenerating; the SET is exactly the scanner's output"
  - "hook-session-start was NOT added to knownEnrichmentSubcommands: it never reaches the documented corpus, and a classification for a command the audit cannot see is dead configuration"
  - "The three command goldens needed no refresh — a hidden command does not reach them, proved by regenerating all three with the repository's own flag and getting a byte-identical result"

patterns-established:
  - "Wiring a hook is a ratchet improvement, not a ratchet cost: read the registration file and the allowlist shrinks"
  - "A document guard matches the instruction's shape held as data, and carries its own proof that it can fail"

requirements-completed: [NEXT-04]

coverage:
  - id: D1
    description: "Opening a chat in a project that already has a colony shows a where-you-left-off card automatically, produced by the same logic every command uses, naming the current phase and the exact command to run next."
    requirement: NEXT-04
    verification:
      - kind: integration
        ref: "cmd/hook_session_start_test.go#TestSessionStartCardReflectsState/a_project_part-way_through_a_build_names_the_phase_and_what_to_run_next"
        status: pass
      - kind: integration
        ref: "cmd/hook_session_start_test.go#TestSessionStartCardReflectsState/a_finished_project_is_told_what_signing_off_means"
        status: pass
      - kind: integration
        ref: "cmd/hook_session_start_test.go#TestSessionStartCardReflectsState/the_command_named_is_the_one_every_other_surface_names"
        status: pass
    human_judgment: false
  - id: D2
    description: "A project with no colony gets nothing at all — a repository that has never used Aether is not greeted, and neither is a leftover state file carrying no goal."
    requirement: NEXT-04
    verification:
      - kind: integration
        ref: "cmd/hook_session_start_test.go#TestSessionStartCardIsSilentWithoutAColony/an_empty_project_is_not_greeted"
        status: pass
      - kind: integration
        ref: "cmd/hook_session_start_test.go#TestSessionStartCardIsSilentWithoutAColony/a_leftover_state_file_with_no_goal_is_not_a_project"
        status: pass
    human_judgment: false
  - id: D3
    description: "The greeting writes nothing. Running it twice over an unchanged project leaves every file under the project's data directory byte-identical and produces the same card."
    requirement: NEXT-04
    verification:
      - kind: integration
        ref: "cmd/hook_session_start_test.go#TestSessionStartHookDoesNotMutate"
        status: pass
    human_judgment: false
  - id: D4
    description: "The greeting decides nothing of its own, and the fifth rival decider is gone: nextCommandForHookState returns the one resolver's answer, and the agreement invariant now drives all five entry points."
    requirement: NEXT-04
    verification:
      - kind: unit
        ref: "cmd/next_action_one_decider_test.go#TestEveryDeciderAgreesOnTheNextCommand"
        status: pass
      - kind: unit
        ref: "cmd/next_action_one_decider_test.go#TestNoSurvivingDeciderSpellsItsOwnCommand"
        status: pass
      - kind: unit
        ref: "cmd/hook_session_start_test.go#TestSessionStartHookChoosesNoCommandOfItsOwn"
        status: pass
    human_judgment: false
  - id: D5
    description: "The hook appears on startup, on resume and after clearing the conversation, and the registration is read out of the shipped settings file rather than asserted about in prose."
    requirement: NEXT-04
    verification:
      - kind: unit
        ref: "cmd/hook_session_start_test.go#TestSessionStartHookIsRegistered"
        status: pass
    human_judgment: false
  - id: D6
    description: "The new hook is genuinely reachable, and the tolerated-orphan list shrank rather than grew: the reachability scanner reads the shipped hook settings file, the three pre-existing hook commands left the list, and the frozen baseline is unchanged."
    requirement: NEXT-04
    verification:
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestNoRegisteredSubcommandIsUnreferenced"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestOrphanAllowlistOnlyShrinks"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestOrphanAllowlistIsPathKeyed"
        status: pass
      - kind: other
        ref: "git diff --exit-code cmd/testdata/orphan_allowlist_baseline.json"
        status: pass
    human_judgment: false
  - id: D7
    description: "No shipped document asks the assistant to check the saved session file at the start of a conversation and report what it found, and a test fails if a reworded version returns to any of the six documents."
    requirement: NEXT-04
    verification:
      - kind: unit
        ref: "cmd/session_start_doc_removal_test.go#TestSessionGreetingIsNotDelegatedToTheAssistant"
        status: pass
      - kind: unit
        ref: "cmd/session_start_doc_removal_test.go#TestSessionGreetingGuardCanFail"
        status: pass
      - kind: manual_procedural
        ref: "plant-back demonstration: the instruction was appended to README.md, the guard FAILED naming README.md:1267, the edit was reverted"
        status: pass
    human_judgment: false
  - id: D8
    description: "The two copies of the colony rules file stay byte-identical, which nothing previously enforced."
    verification:
      - kind: unit
        ref: "cmd/session_start_doc_removal_test.go#TestColonyRulesCopiesStayIdentical"
        status: pass
      - kind: other
        ref: "diff .claude/rules/aether-colony.md .aether/rules/aether-colony.md"
        status: pass
    human_judgment: false
  - id: D9
    description: "The greeting's wording reads as plain English for someone who has never opened a file in this repository."
    verification: []
    human_judgment: true
    rationale: "The card is 197-02's, already covered by TestNextActionCardSpeaksPlainEnglish for vocabulary. Whether the greeting actually lands as the first sentence of a non-technical owner's session is a judgement no test asserts."

duration: 33 min
completed: 2026-08-28
status: complete
---

# Phase 197 Plan 03: The Greeting the Program Owns Summary

**A hidden session-start hook renders the shared "what next" card when a chat opens, resumes or is carried on after a clear — and teaching the reachability scanner to read `.claude/settings.json` made the tolerated-orphan list shrink by four instead of growing by one.**

## Performance

- **Duration:** 33 min
- **Started:** 2026-08-28T19:51Z
- **Completed:** 2026-08-28T20:27Z
- **Tasks:** 3
- **Files created:** 2
- **Files modified:** 9

## Accomplishments

- `aether hook-session-start` loads the project, asks `resolveNextAction`, and prints `renderNextActionCard`. It composes nothing of its own, and it is silent when there is no project — a repository with Aether installed but no colony running is not greeted.
- The hook is registered in `.claude/settings.json` for `startup`, `resume` and `clear`, each with a timeout, and the registration is read out of that file as data by `TestSessionStartHookIsRegistered` rather than described in prose.
- `nextCommandForHookState` — the fifth rival decider, deliberately left by 197-02 because this plan owns `cmd/hook_cmds.go` — is now a one-line adapter. `TestEveryDeciderAgreesOnTheNextCommand` and `TestNoSurvivingDeciderSpellsItsOwnCommand` both cover all five entry points, not four.
- The reachability scanner's shipped-hook corpus gained `.claude/settings.json`, parsed into the platform's documented hook shape. That closed a real hole: **every Aether hook was counted as having no caller** while the platform fired it on every session.
- CLAUDE.md, AGENTS.md and both copies of the colony rules file no longer ask the assistant to check the saved session file when a conversation opens. Each now describes the hook doing it — a claim five named tests can prove.

## The allowlist shrink

This is the manoeuvre the plan turned on: adding a hook would normally force an edit to the frozen baseline, which `TestOrphanAllowlistOnlyShrinks` forbids. Widening the scan instead made the list move the right way.

| Measurement | Before | After |
|---|---|---|
| Live tolerated-orphan list (`cmd/testdata/orphan_allowlist.json`) | **272** entries | **268** entries |
| Orphans found by the scanner | 272 (including the new, uncredited `aether hook-session-start`) | 268 |
| Frozen baseline (`orphan_allowlist_baseline.json`) | 284 | **284 — unchanged** |

**Recorded failing run before the widening:**

```
--- FAIL: TestNoRegisteredSubcommandIsUnreferenced
    enumerated 404 registered commands, found 272 orphans
    1 registered subcommand(s) have no caller and are not in testdata/orphan_allowlist.json:
      aether hook-session-start is registered but nothing calls it
      (searched: wrappers, menu specs, hooks, scripts)
```

**The four entries that left the list:**

| Entry | Why it left |
|---|---|
| `aether hook-pre-tool-use` | Now credited: `.claude/settings.json` registers it |
| `aether hook-stop` | Now credited: same |
| `aether hook-pre-compact` | Now credited: same |
| `aether memory-capture` | **Already stale before this plan** — see below |

`aether memory-capture` is not a hook and this plan's widening cannot have credited it. The arithmetic proves it was stale already: the pre-change scan found 272 orphans *including* the new command, so 271 pre-existing orphans sat against a 272-entry list — exactly one entry that was no longer an orphan. It is called from `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md`, so it had a real caller and the list had simply not been regenerated since. The regeneration removed it as a side effect, which is the flag doing its job.

`git diff --exit-code cmd/testdata/orphan_allowlist_baseline.json` is clean. The final live-list diff is **four deletions, zero insertions, zero other changes**.

## Task Commits

1. **Task 1 RED: failing tests for the session-start greeting** — `cdcf9c75` (test)
2. **Task 1 GREEN: the hook, and the fifth decider delegates** — `0012479d` (feat)
3. **Task 2: registration, the widened scan, and the shrunk list** — `b7555368` (feat)
4. **Task 3 RED: the failing removal guard** — `e12798c5` (test)
5. **Task 3 GREEN: the four documents** — `b11d4e19` (docs)

**Recorded RED output for each TDD task:**

- **Task 1 RED** — `unknown command "hook-session-start" for "aether"` at every execution site; `hookSessionStartCmd was not found in cmd/hook_cmds.go`; `.claude/settings.json registers no SessionStart hook at all, so the greeting never runs`.
- **Task 2 RED** — quoted in full in the shrink table above: `aether hook-session-start is registered but nothing calls it`, plus the registration test failing on the absent `SessionStart` block.
- **Task 3 RED** — behavioural, not a build failure. The guard was written to compile and run against the tree as it stood, and named the documents that genuinely carried the instruction:

```
--- FAIL: TestSessionGreetingIsNotDelegatedToTheAssistant
    CLAUDE.md:1103 asks the assistant to check the saved session file when a
      conversation opens and report what it finds.
    AGENTS.md:802 ...
    .claude/rules/aether-colony.md:1 ...
    .aether/rules/aether-colony.md:1 ...
```

Four documents, not six. See the deviations.

## The plant-back demonstration

The plan required proof that the guard fails when the instruction is planted back into any one of the six documents. `README.md` — one of the two that never carried it — was chosen precisely because a guard that only catches text in the files it was written against is not a guard:

```
--- FAIL: TestSessionGreetingIsNotDelegatedToTheAssistant
    README.md:1267 asks the assistant to check the saved session file when a
    conversation opens and report what it finds.
```

Reverted with `git checkout -- README.md`; `git status --short` clean, the guard passes again.

`TestSessionGreetingGuardCanFail` holds the same proof permanently, in two directions: it fires on the original wording **and** on a paraphrase sharing no sentence with it ("When a chat session starts, take a look at the saved session file and… tell the user what you found"), and it stays quiet on legitimate prose that names `session.json` in a state-contract table or describes the runtime doing the greeting.

## Files Created/Modified

- `cmd/hook_cmds.go` — `hookSessionStartCmd`, and `nextCommandForHookState` collapsed to a one-line adapter
- `cmd/hook_session_start_test.go` — the card, silence, no-mutation, no-own-command and registration tests
- `cmd/session_start_doc_removal_test.go` — the shape-matching removal guard, its own can-fail proof, and the rules-copy identity check
- `cmd/subcommand_reachability_ratchet_test.go` — `.claude/settings.json` added to `hookScriptCorpora`, plus `hookSettingsCommandArgs`
- `cmd/next_action_one_decider_test.go` — the fifth decider added to both halves of the agreement invariant
- `cmd/testdata/orphan_allowlist.json` — four entries removed
- `.claude/settings.json` — the `SessionStart` block
- `CLAUDE.md`, `AGENTS.md`, `.claude/rules/aether-colony.md`, `.aether/rules/aether-colony.md` — the request replaced by a description of the hook

## Decisions Made

**1. The scan reads the settings file as data, not as text.**
Grepping it would credit any command name appearing anywhere in the file — a description, a permission rule, a foreign tool's argument. `hookSettingsCommandArgs` decodes the platform's documented shape (`hooks` → event → entries → inner hooks → `command`) and only credits a string the platform will actually execute. The command string itself then goes through the *same* shell-like tokenizer and the *same* command-name shape the script scan already uses, so `aether host colonize` resolves identically here and there rather than through a second parser that can drift. A JSON file that is not a hook settings file — `.claude/package.json`, also in that directory — decodes to an empty map and credits nothing.

**2. The corpus entry is the `.claude` directory, not a single file path.**
`hookScriptCorpora` is a directory-plus-extension list consumed by three different scans (`collectCallerEvidence`, `listCallerCorpusFiles`, `collectSubstitutionCallerNames`). Adding a one-off single-file special case to all three would have created exactly the drift the shared shape prevents.

**3. `hook-session-start` was not added to `knownEnrichmentSubcommands`.**
The plan makes this conditional on the command reaching the documented corpus. It does not: that corpus is the wrapper `.md`/`.yaml` trees plus four named `.aether/*.md` files, and a hidden hook command invoked only from a settings file never appears there. `TestDocumentedSubcommandsAreSeverityClassified` passes untouched, and the three existing hook commands are unclassified for the same reason. Adding an entry for a command the audit cannot see would be dead configuration — and if a future document does name it in backticks, that test will force the decision at the moment it becomes real. For the record, the judgement it should be given then is **enrichment**: a greeting that fails must never halt anything.

**4. The greeting is silent on every failure path, not only on "no project".**
`loadNextActionInput` reports `NoColony` for a missing store, an unreadable state file and a state file carrying no goal alike, so all three are silence. A hook that errors fires before the owner has typed a word; a greeting that fails loudly on a malformed project is worse than one that says nothing.

**5. The wording says what the repo's words mean.**
The replacement text in all four documents avoids "colony", "seal" and "entomb" unqualified — "a folder with no project set up in it", "the one command to run next". The card itself is 197-02's and already covered by its plain-English check.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] The instruction was in four documents, not six**

- **Found during:** Task 3
- **Issue:** The plan's objective names "CLAUDE.md, AGENTS.md, README.md and both copies of the colony rules file", and its behavior block says six documents each ask the assistant to check the session file. They do not. `README.md` and `docs/phase3-section-walkthrough.md` contain the string "Previous colony session detected", but inside a fenced transcript showing what `aether resume` / `/ant-resume` prints when the OWNER runs it. That is example output of a command, not an instruction addressed to the assistant, and removing it would delete an accurate illustration. Confirmed by sweeping every markdown file in the repository for the instruction's own phrasing: `grep -rn "first message" --include="*.md"` returns exactly four documents.
- **Fix:** Removed the instruction from the four that carried it. `README.md` and `docs/phase3-section-walkthrough.md` are unmodified — but both are still in `sessionGreetingDocuments` and scanned on every run, because the guard's job is to stop the request appearing anywhere it plausibly could, not only where it once was. The plant-back demonstration above uses `README.md` for exactly that reason.
- **Verification:** the recorded RED names four documents; the plant-back proof fires on the fifth.
- **Commits:** `e12798c5`, `b11d4e19`

**2. [Rule 2 — Missing critical] The regeneration flag wipes reviewed disposition reasons**

- **Found during:** Task 2
- **Issue:** `writeOrphanAllowlist` derives each entry's `reason` and `owner_phase` from the frozen pre-migration snapshot, defaulting to `"unreviewed-pre-existing"`. Seventeen live entries carried hand-written dispositions added after a previous regeneration — `"dispositioned-2026-08-16: belongs to the uncommitted PR-based worktree workflow…"`, `"superseded-2026-08-16: Go owns the entire Oracle loop…"`, and the reviewed `worktree-reap` note. Running the flag destroyed all seventeen, replacing months of review with a placeholder.
- **Fix:** Regenerated with the flag as the plan requires, then restored `reason` and `owner_phase` from the previous file for every entry that survived. The SET of names is exactly the scanner's output — untouched — and the shrink-only rule is about names. The final diff is four deletions and nothing else.
- **Note for a future plan:** this is a wart in the tool, not in this change. Anyone running `-update-orphan-allowlist` will destroy those reasons again. The fix would be for `writeOrphanAllowlist` to carry forward the existing live file's reason for a surviving name; that is a change to a wiring-guard file and belongs to whichever plan next owns it.
- **Commit:** `b7555368`

**3. [Rule 3 — Blocking] One edit outside the declared files, required by the plan's own acceptance criterion**

- **Found during:** Task 1
- **Issue:** The plan requires "`TestEveryDeciderAgreesOnTheNextCommand` still passes **with the hook decider included in the comparison**". That test lives in `cmd/next_action_one_decider_test.go`, which is 197-02's file and is not in this plan's `files_modified`. Including the fifth decider without editing it is impossible.
- **Fix:** Two additions, both a single map entry: `nextCommandForHookState` in the behavioural comparison, and `"nextCommandForHookState": "hook_cmds.go"` in the structural AST sweep. Nothing existing was changed or weakened. This is the smaller of the two available edits — the alternative was writing a second, rival agreement test in my own file, which is the duplication this whole phase exists to end.
- **Verification:** both tests pass with five deciders; the fixture set and every existing assertion are unchanged.
- **Commit:** `0012479d`

**4. [Rule 1 — Bug] A fixture meant to be "a build that is running" read as "a build that stalled"**

- **Found during:** Task 1
- **Issue:** The first draft used `BuildStartedAt: fixtureTime(t, "2026-08-01T10:00:00Z")`, copied from 197-01's tests where it feeds the pure resolver directly. Driven through the real loader against the real clock, that date is far past `abandonedBuildThreshold`, so the card correctly recommended `aether build 2 --force` and the test's `aether continue` expectation failed. The fixture was wrong, not the code.
- **Fix:** `runningBuildStartedAt()` derives the timestamp from `abandonedBuildThreshold` itself (half of it, in the past), so a fixture meant to represent a running build cannot silently become a stalled one when that constant changes. CLAUDE.md's rule about fixtures in a shape the runtime really produces, applied to a value rather than a structure.
- **Commit:** `0012479d`

**5. [Rule 2 — Missing critical] Nothing enforced the two rules-file copies matching**

- **Found during:** Task 3
- **Issue:** The plan asked me to refresh the distributed copy's recorded checksum "the way the release tooling does", or to say so in the summary if no check enforces it. There is no `.aether/manifest.json` in this repository, `.aether/references/manifest.json` does not cover the rules file, and the only place the two copies are named together (`cmd/hook_cmds_test.go`'s `TestSanctionedScratchDirsDocumented`) checks each for a substring rather than comparing them. **Nothing enforced it.** An edit to one copy and not the other would have shipped silently — and the `.aether/` copy is the one published to the hub and read by other repositories.
- **Fix:** `TestColonyRulesCopiesStayIdentical` compares the two files byte for byte and says which one downstream repositories actually read.
- **Verification:** `diff .claude/rules/aether-colony.md .aether/rules/aether-colony.md` produces no output; the test passes.
- **Commit:** `e12798c5`

### Deliberate non-changes

- **`cmd/testdata/command_catalog.json`, `parity_snapshot.json`, `regression_snapshot.json`: not refreshed, because they did not change.** The plan warned that Phase 196 added one subcommand and broke four checks at once. That command was visible; this one is `Hidden: true`, and the catalog contains no `hook-` entry at all. Rather than assume, all three were regenerated with the repository's own `-update-golden` flag and `git status` showed no change to any of them. All six named checks pass untouched.
- **`cmd/command_call_audit_test.go`: not modified.** Decision 3 above.
- **`.aether/manifest.json`, `docs/phase3-section-walkthrough.md`, `README.md`: not modified.** The first does not exist; the other two never carried the instruction.
- **`cmd/.claude/settings.json`: not modified.** This tracked file is a stale copy used when the binary is run with the working directory inside `cmd/`; it already lacked the `Agent|Task` entry the root file has gained since. It is not the published source (`install_cmd.go:809` publishes the repo-root `.claude`), it is not what the reachability scan reads, and bringing it into line is an unrelated tidy-up that would have widened this plan's blast radius. Worth a follow-up.

---

**Total deviations:** 5 auto-fixed (2 bugs, 2 missing-critical, 1 blocking file-scope)
**Impact on plan:** No scope creep. One is a correction to the plan's own premise about how many documents carried the instruction; two prevented information loss (reviewed dispositions, and an unenforced file-copy invariant); one is a single-map-entry edit the plan's own acceptance criterion required.

## Issues Encountered

None beyond the deviations above.

## Known Stubs

None. Everything this plan describes is wired and fires: the hook is registered in the file the platform reads, the registration is asserted from that file, and the reachability ratchet — the repository's own anti-orphan check — now counts it as reached.

## Threat Flags

None. This plan adds no network endpoint, no auth path and no schema at a trust boundary. It adds one read-only command that reads project-local state the runtime already reads, and writes nothing — proved by `TestSessionStartHookDoesNotMutate`.

## Defect ledger

Nothing was appended to `.planning/WINDOWS.md`. This plan leaves behind no stub, no skipped test and no unrun verification, and its one out-of-declared-files edit was required by the plan's own acceptance criterion rather than being an unreviewed liberty. The ledger is an append-only JSON array with a running id, plan 197-04 was executing concurrently in its own worktree, and two blind appends would have collided on merge — recording here instead, where the reviewer reads it, was the safer trade. The one item genuinely worth carrying forward is the regeneration flag wiping reviewed disposition reasons (deviation 2), which is a wart in an existing tool rather than a defect introduced here.

## Verification Run

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(SessionStart\|SessionGreetingIsNotDelegatedToTheAssistant\|SessionGreetingGuardCanFail\|ColonyRulesCopiesStayIdentical\|EveryDeciderAgreesOnTheNextCommand\|NoSurvivingDeciderSpellsItsOwnCommand\|NoRegisteredSubcommandIsUnreferenced\|OrphanAllowlist\|WiringGuardsHaveNoRuntimeEscapeHatch\|AuditCatalogGolden\|CatalogCompleteness\|PlatformParityGolden\|RegressionSnapshot\|DocumentedSubcommandsAreSeverityClassified\|HumanFacingOutputGoesThroughWriteVisualOutput)' -count=1` | ok (1.7s) |
| `go test ./cmd -run 'Test(Hook\|NextActionCard\|SanctionedScratchDirsDocumented)' -count=1` | ok |
| `go test ./cmd -run 'Test(DeletingACallerMakesTheRatchetNameIt\|CallerEvidence\|PathMigration\|Settings\|Update)' -count=1` | ok — the ratchet's own self-tests and the settings-merge suite |
| `go test ./cmd -run 'Test(Doc\|Docs\|Readme\|README\|Parity\|Hygiene\|Manifest\|VersionSync\|Claude\|Agents)' -count=1` | ok |
| `go test ./cmd -run 'Test(Ceremony\|Closeout\|Recovery\|Resume\|Pause\|Status\|Continue\|Build\|Compat\|Truth\|Visual\|Hint\|NextUp\|Snapshot)' -count=1` | ok (89s) — the downstream callers of the repointed decider |
| `git diff --exit-code cmd/testdata/orphan_allowlist_baseline.json` | clean |
| `diff .claude/rules/aether-colony.md .aether/rules/aether-colony.md` | no output |
| `go build ./...` | clean |
| `go build ./cmd/aether` | clean |
| `go vet ./cmd/` | clean |
| `gofmt -l cmd/ pkg/` | no output |

The full `go test ./cmd` run is the orchestrator's after merge-back, per this executor's instructions.

## User Setup Required

None — no external service configuration required. The hook takes effect in this repository as soon as `.claude/settings.json` is read by a new session; downstream repositories receive it through the existing `aether publish` / `aether update` path, where `mergeClaudeSettings` adds the new `SessionStart` block without disturbing any foreign hook.

## Next Phase Readiness

- **197-04 / 197-06** inherit an unchanged `cmd/codex_visuals.go` — this plan touched none of their files. `renderNextActionCard` now has a second live caller (the hook), so a change to the card is felt in the greeting too.
- **All five deciders are now adapters.** `TestEveryDeciderAgreesOnTheNextCommand` covers every surviving entry point, so any later plan that repoints a closing renderer is measured against a complete invariant rather than four fifths of one.
- **The reachability scan now credits registered hooks**, so a future plan adding a hook adds a caller rather than an orphan.

## Self-Check: PASSED

**Files claimed as created — all present on disk:**

| File | Present |
|---|---|
| `cmd/hook_session_start_test.go` | yes |
| `cmd/session_start_doc_removal_test.go` | yes |
| `.planning/phases/197-one-answer-to-what-next/197-03-SUMMARY.md` | yes |

**Commits claimed — all present in `git log`:** `cdcf9c75`, `0012479d`, `b7555368`, `e12798c5`, `b11d4e19`.

**Nothing changed outside the declared surface.** `git diff --name-only 98195e1a..HEAD` lists exactly eleven files: the six under `cmd/`, `.claude/settings.json`, and the four documents. `cmd/codex_visuals.go` — plan 197-04's file in this wave — does not appear.

**`.planning/STATE.md` and `.planning/ROADMAP.md`:** not modified; they do not appear in that diff. The orchestrator owns them in parallel-executor mode.

**Temporary demonstrations reverted:** the README plant-back and two scratch helper files (`restore_reasons.py`, `allowlist_before.json`) were removed; `git status --short` is clean apart from this summary.

---
*Phase: 197-one-answer-to-what-next*
*Completed: 2026-08-28*
