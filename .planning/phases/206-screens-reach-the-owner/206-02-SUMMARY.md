---
phase: 206-screens-reach-the-owner
plan: 02
subsystem: cli-hooks
tags: [go, cobra, claude-code-statusline, reachability-ratchet, real-chat-proof]

requires:
  - phase: 206-screens-reach-the-owner (plan 01)
    provides: "the direct screen-delivery route (aether hook-post-tool-use, directScreenDelivery) this plan proves in a real chat and judges alongside"
provides:
  - "aether status-line: a hidden cobra command, redrawn on every Claude Code turn, sourced entirely from resolveNextAction (no rival decision), silent without a project, and proven never to write anything"
  - "scripts/proof-screens-reach-the-owner.sh: builds Aether from source, installs into an isolated hub, runs the real init/update --force install path in a scratch project, then drives one capped real claude -p chat proving the direct route delivers every banner line as an informational message distinct from the Bash tool's own echoed stdout"
  - ".planning/phases/206-screens-reach-the-owner/206-OWNER-VERDICT.md: the owner's recorded verdict (keep-on) after looking at both surfaces on his own screen"
  - "hookSettingsCommandArgs (cmd/subcommand_reachability_ratchet_test.go) extended to recognize the statusLine top-level settings key as a genuine caller, not only hooks"
affects: [207-messy-practice-project]

actuals:
  tokens: 10400
  tasks: 4
  commits: 4

tech-stack:
  added: []
  patterns:
    - "A redrawn-every-turn surface (status line) reuses the existing read-only loader (loadNextActionInput) rather than the greeting's richer loader (loadNextActionInputForGreeting), because the richer loader's extra hive/preference reads cost something a per-second redraw should not pay"
    - "A command's display spelling always routes through the one existing translator (translateHintCommandsForPlatform) at render time, rather than trusting a projection field computed for a different, hardcoded platform earlier in the pipeline"
    - "A real-chat proof script discriminates 'delivered as an informational message' from 'merely echoed back as the tool's own stdout' by excluding tool_result content blocks from the delivery-evidence text before checking for banner lines, so the gate cannot pass merely because the Bash command produced the right output"

key-files:
  created:
    - cmd/status_line.go
    - cmd/status_line_test.go
    - scripts/proof-screens-reach-the-owner.sh
    - .planning/phases/206-screens-reach-the-owner/206-OWNER-VERDICT.md
    - .planning/phases/206-screens-reach-the-owner/deferred-items.md
  modified:
    - cmd/hook_cmds.go
    - .claude/settings.json
    - cmd/hook_session_start_test.go
    - cmd/subcommand_reachability_ratchet_test.go
    - CLAUDE.md

key-decisions:
  - "statusLineText always resolves the command through translateHintCommandsForPlatform, never trusting projection.NextAction.DisplayCommand's raw value, because gateLifecycleProjection (cmd/next_action.go) resets DisplayCommand to the untranslated 'aether <verb>' form on every ordinary single-command answer -- following the plan's literal 'use DisplayCommand when non-empty' wording verbatim would have printed 'aether continue' instead of '/ant-continue' on Claude, failing both the plan's own <behavior> spec ('shown in the form the owner can actually type in this platform') and its acceptance criteria (a forward-slash-ant-prefixed command)."
  - "hookSettingsCommandArgs (the reachability ratchet's JSON hook-settings decoder) extended to also credit a project's statusLine.command the same way it credits a hooks entry, because it previously only understood the 'hooks' key and would have reported aether status-line as a permanent false orphan otherwise."
  - "The proof script's delivery gate excludes every string leaf found inside a tool_result content block before checking for banner-line text, because the Bash tool's own echoed stdout will always contain the banner lines regardless of whether the direct-delivery route works at all -- a naive whole-stream substring search could never fail."
  - "Owner reviewed both surfaces (the delivered screen and the status line) on his own screen 2026-09-22 and answered keep-on, verbatim, with no accompanying sentence: the direct route stays on by default; the Stop-hook backstop from Phase 205 remains, unchanged, as the safety net underneath."

patterns-established:
  - "A settings-shape reachability gap (a new top-level key the scanner doesn't parse) is fixed in the scanner itself, in the same commit that introduces the new key, rather than allowlisted -- the allowlist is shrink-only and widening it to hide a true finding is exactly the failure TestOrphanAllowlistOnlyShrinks exists to prevent."

requirements-completed: [UED-05, UED-06]

coverage:
  - id: D1
    description: "A permanent status line shows the phase, the open task and the one next command, sourced entirely from the shared what-next decision, silent without a project, byte-identical on repeat, safe under concurrent reads, and writes nothing"
    requirement: "UED-06"
    verification:
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineIsRegistered"
        status: pass
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineComesFromTheSharedDecision"
        status: pass
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineIsSilentWithoutAProject"
        status: pass
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineChangesNothingAndRepeatsItself"
        status: pass
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineIsSafeUnderConcurrentReads (also run with -race)"
        status: pass
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineSpeaksPlainEnglish"
        status: pass
      - kind: unit
        ref: "cmd/status_line_test.go#TestStatusLineInstallOnlyWhenTheProjectHasNone"
        status: pass
    human_judgment: false
  - id: D2
    description: "A committed real-chat proof script installs Aether by the real path into a scratch project and proves, in one real claude -p chat, that the direct route delivers the screen as an informational message distinct from the Bash tool's own echoed output, and that the status line prints"
    requirement: "UED-05"
    verification:
      - kind: e2e
        ref: "scripts/proof-screens-reach-the-owner.sh (two real runs, 2026-09-21/22, both exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The owner looked at the delivered screen and the status line on his own screen and judged how the fixed Claude Code prefix reads"
    requirement: "UED-05"
    verification: []
    human_judgment: true
    rationale: "How a fixed, unstylable platform prefix reads on the owner's own screen is exactly the kind of judgment CLAUDE.md's checkpoint protocol reserves for the owner; recorded verbatim in 206-OWNER-VERDICT.md rather than inferred."

duration: ~65min active (Tasks 1, 2 and 4; excludes the blocking-human wait for Task 3's owner decision)
completed: 2026-09-22
status: complete
---

# Phase 206 Plan 02: Status Line, Real-Chat Proof and Owner Verdict Summary

**A permanent Claude Code status line sourced from the one shared what-next decision, a real-chat proof script that catches the difference between "the screen was delivered" and "the Bash command merely produced the right stdout," and the owner's recorded keep-on verdict on both surfaces.**

## Performance

- **Duration:** ~65 min active work (Tasks 1, 2, 4); a blocking-human wait for the owner's Task 3 decision separates Task 2's completion (2026-09-21T22:02Z) from Task 4's start
- **Started:** 2026-09-21T23:35Z (approx, following 206-01)
- **Completed:** 2026-09-22T08:14Z (Task 4)
- **Tasks:** 4 (3 code/doc tasks + 1 owner-decision checkpoint)
- **Files modified:** 10

## Accomplishments

- Built `aether status-line` (new hidden cobra command) and its two pure helpers (`statusLineText`, `statusLineTaskLabel`), registered against a new top-level `statusLine` settings key, sourced entirely from `resolveNextAction` with an AST guard proving it spells no command of its own.
- Discovered and fixed a real gap in the reachability ratchet: `hookSettingsCommandArgs` only recognized the `hooks` settings key, so the new `statusLine` key would have left `aether status-line` a permanent false orphan; extended the same decoder to credit it the same way.
- Built `scripts/proof-screens-reach-the-owner.sh`: installs Aether by the real `init` + `update --force` path into an isolated scratch project and hub, then drives one capped real `claude -p` chat that runs `aether status` and proves the direct-delivery route (206-01) carries every banner line to the owner as an informational message -- with a delivery check specifically designed to fail if the banner text only ever reached the transcript via the Bash tool's own echoed stdout, never through the direct route.
- Ran the proof script for real, twice: both runs passed end to end (1 Bash tool call, 7/7 banner lines proven delivered as an informational message distinct from the tool's own output, no transient retry needed, status line printed correctly).
- Recorded the owner's verdict (`keep-on`, verbatim, on his own screen, 2026-09-22) in `206-OWNER-VERDICT.md` and added one sentence to CLAUDE.md's "The screen reaches the owner directly" section naming the date and the decision; per the plan's own instruction for a `keep-on` answer, no runtime or settings code changed.
- Ran the full test suite once (`go test ./... -count=1 -timeout 90m`, in the background per the coordinator's instruction) and confirmed its own accounting line (`FULL-SUITE FAIL discovered=5908 executed=5908 lanes=59`) shows no silent truncation before classifying any failure.

## Task Commits

Each task was committed atomically:

1. **Task 1: A permanent line saying where the owner is and what to run next** - `4d005cc4` (feat)
2. **Task 2: Prove it in a real chat, in a scratch project, with the hooks really installed** - `3be6b11e` (feat)
3. **Task 3: The owner looks at it on his own screen and decides** - checkpoint, no code commit; the owner's answer (`keep-on`) was collected out of band and relayed for Task 4 to record
4. **Task 4: Record the verdict, apply it, and run the whole suite** - `1ca7ca1e` (docs)

**Plan metadata:** committed alongside this SUMMARY.

## Files Created/Modified

- `cmd/status_line.go` - `statusLineCmd`, `statusLineText`, `statusLineTaskLabel`, `statusLineMaxTaskRunes`
- `cmd/status_line_test.go` - seven tests: registration, shared-decision + AST no-command guard, silence, mutation/repeat, concurrency (race-clean), plain-English, settings-merge ownership
- `cmd/hook_cmds.go` - registers `statusLineCmd` in `init()`
- `.claude/settings.json` - new top-level `statusLine` key (`aether status-line`, padding 0)
- `cmd/hook_session_start_test.go` - `claudeHookSettingsFile` gains a `StatusLine` field so `TestStatusLineIsRegistered` can read the shipped settings as data
- `cmd/subcommand_reachability_ratchet_test.go` - `hookSettingsCommandArgs` credits `statusLine.command` the same way it credits a `hooks` entry
- `scripts/proof-screens-reach-the-owner.sh` - the real-chat proof, executable, with turn and wall-clock caps
- `.planning/phases/206-screens-reach-the-owner/deferred-items.md` - the pre-existing, unrelated `codex-native-worker context-ack` orphan, logged not fixed
- `.planning/phases/206-screens-reach-the-owner/206-OWNER-VERDICT.md` - the owner's recorded verdict
- `CLAUDE.md` - one added sentence recording the owner's review and decision

## Decisions Made

See `key-decisions` in the frontmatter for the full rationale on each. In short: the status line's command always goes through the existing platform translator rather than a projection field that turns out to be pre-reset to the untranslated form; the reachability ratchet learned about the new settings key in the same commit that introduced it; the proof script's pass/fail gate is structurally unable to pass on tool-echo alone; and the owner's `keep-on` answer changes no code, only the record and one CLAUDE.md sentence.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reachability ratchet did not recognize the new `statusLine` settings key**
- **Found during:** Task 1's acceptance-criteria verification loop (`go test ./cmd -run 'TestStatusLine|TestAuditCatalogGolden|TestCatalogCompleteness|TestNoRegisteredSubcommandIsUnreferenced'`)
- **Issue:** `hookSettingsCommandArgs` only decoded the `hooks` top-level key; `aether status-line` reported as an orphan with no caller
- **Fix:** Extended the decoder to also credit `statusLine.command`, factored through a small shared `creditCommandString` closure so the `hooks` path and the new path can't drift
- **Files modified:** `cmd/subcommand_reachability_ratchet_test.go`
- **Verification:** `TestNoRegisteredSubcommandIsUnreferenced` no longer names `aether status-line`; the one remaining name (`aether codex-native-worker context-ack`) is unrelated and pre-existing (see below)
- **Committed in:** `4d005cc4` (Task 1 commit)

**2. [Rule 1 - Bug, judgment correction] `statusLineText`'s command resolution deviates from the plan's literal action-text wording**
- **Found during:** Task 1 implementation, confirmed empirically by tracing `gateLifecycleProjection`
- **Issue:** The plan's action text said to prefer `answer.Projection.NextAction.DisplayCommand` when non-empty, falling back to a translated `RuntimeCommand` only when it was empty. Tracing the actual code shows `gateLifecycleProjection` (`cmd/next_action.go`) resets `DisplayCommand` to the untranslated `RuntimeCommand` value on every ordinary single-command answer (it is computed for a hardcoded `"codex"` platform earlier in `resolveNextAction`, then unconditionally overwritten with the plain form once the command is confirmed available) -- so `DisplayCommand` is essentially always equal to the untranslated `RuntimeCommand`, never empty for a normal answer. Following the literal instruction would have printed `aether continue` on the Claude status line instead of `/ant-continue`, contradicting both the plan's own `<behavior>` line ("shown in the form the owner can actually type in this platform") and its acceptance criterion (a forward-slash-ant-prefixed command).
- **Fix:** Whichever command source is found (`DisplayCommand` or `RuntimeCommand`) is always routed through the existing `translateHintCommandsForPlatform`, matching every other rendered surface in this package (`cardText`, `renderNextUp`) and satisfying the plan's own stated behavior and acceptance criteria.
- **Files modified:** `cmd/status_line.go`
- **Verification:** `go run ./cmd/aether status-line` in this repo prints `Aether · next: /ant-resume`; `TestStatusLineComesFromTheSharedDecision` asserts the rendered command equals `translateHintCommandsForPlatform(answer.Command, platform)`
- **Committed in:** `4d005cc4` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking/scanner gap, 1 bug-avoiding correction to literal plan wording that would have failed the plan's own acceptance criteria)
**Impact on plan:** Both fixes were necessary for the plan's own stated behavior and acceptance criteria to hold. No scope creep -- neither touches anything beyond the status line itself.

### Logged, not fixed (out of scope)

**`aether codex-native-worker context-ack` is a pre-existing, unrelated orphan.** `TestNoRegisteredSubcommandIsUnreferenced` reports it with no caller (wrappers, menu specs, hooks, scripts). It originates from commit `1f905181` ("feat: require source-bound native context reads and acknowledgements"), part of the still-in-progress Phase 204.2 (Codex Native Worker Lifecycle) work `STATE.md` itself documents as incomplete. No file this plan touches names `codex-native-worker` or `context-ack`, and the scanner change this plan made (recognizing `statusLine`) cannot affect resolution of a command with no settings/wrapper/script caller at all. Logged in `.planning/phases/206-screens-reach-the-owner/deferred-items.md` per the executor's scope-boundary rule (fix only what the current task's changes touch); belongs to whichever plan finishes wiring Phase 204.2's native context-acknowledgement path.

## Issues Encountered

- **`--max-turns` does not appear in `claude --help` on the installed CLI (2.1.278), but the flag is accepted and appears to function** (a quick, isolated probe -- `claude -p "say hi" --max-turns 1` -- completed normally rather than erroring on an unrecognized flag). The proof script applies it unconditionally, per the plan's instruction to "always" apply a turn cap; both real runs of the proof script completed within the cap with no observed issue. `--max-budget-usd` is conditional on `claude --help` mentioning it, per the plan's own flagged assumption, and was present and applied on this installed version.
- **The proof script's "delivered message" print was noisy on the first two real runs** (it printed every string leaf in the whole stream, not only the delivered informational message, because the targeted `type=="system"`/`subtype=="informational"` selector did not match this CLI version's actual field name on the second run). Fixed by adding a display-only refinement (filter to lines containing the stable `" says: "` hook-relay prefix) that never touches the underlying pass/fail logic; the refinement was validated directly against both runs' captured output rather than spending a third live `claude -p` invocation, staying within the two-live-run budget the project constraints set.

## User Setup Required

None - no external service configuration required.

## Full Suite Verification (Task 4)

`go test ./... -count=1 -timeout 90m` was run once, in the background as instructed, and its own accounting line was checked before trusting any result:

```
FULL-SUITE FAIL discovered=5908 executed=5908 lanes=59
```

Discovered equals executed -- no lane silently stopped partway through (the exact failure mode this project's own notes warn a truncated run can otherwise hide behind a clean-looking summary).

**30 unique top-level test names failed** (43 lines including subtests) across 20 of 59 lanes. Every one is classified pre-existing, not caused by this plan:

- **17 names match the 2026-09-14 known-red baseline** (recorded before Phase 204 began) plus the **Codex-native evidence set added 2026-09-20**, both pre-dating this plan: `TestBuildStartLegacyHelpersRetired200`, `TestCheapModelWorkerNeedsNoReasonOnTheCard`, `TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount`, `TestCodexNativeCancellationRefusalEvidence`, `TestCodexNativeCancellationRefusalReplay`, `TestCodexNativeEvidenceReceiptSchemaDispatch`, `TestCodexNativeEvidenceRejects`, `TestCodexNativeFourthReviewLegacyReplayInventory`, `TestCodexNativeGapRecovery`, `TestCodexNativePhaseEvidence`, `TestCodexNativeThirdReviewReplayInventory`, `TestContinueCreditsTasksProvenInAnEarlierAttempt`, `TestCurrentVocabulary199`, `TestDefaultOracleInvokerAvoidsOpenCodeInsideOpenCodeAgent`, `TestFailedCheckSendsExactlyOneBuilderFixAttempt`, `TestFixAttemptIsCountedSeparately`, `TestFixAttemptNeverOverwritesTheFirstResult`, `TestGoldenBuildVisualOutput`, `TestGoldenContinueVisualOutput`, `TestGoSourceHintsMatchCobraContracts`, `TestHumanFacingOutputGoesThroughWriteVisualOutput`, `TestNoSecondAutomaticFixAttempt`, `TestPartialRedispatchRecoveryNeverNamesAlreadyProvenWork`, `TestPhase199GateReceipt`, `TestPlanningAdversarial200`, `TestPlanningPublicPaths200`, `TestResolveTestCommand_GoProject` -- all tied to Phase 204.2's still-incomplete native-worker qualification work (`STATE.md`'s own "Qualification" note) or earlier documented pre-Phase-204 known-red entries, none touching anything this plan changed.
- **`TestGateProbe`** is documented child-process test noise (project memory: "TestGateProbe FAIL lines are noise").
- **`TestNoRegisteredSubcommandIsUnreferenced`** fails only on the pre-existing, unrelated `codex-native-worker context-ack` orphan (see "Logged, not fixed" above) -- the `aether status-line` orphan this plan could have introduced is fixed and does not appear.
- **`TestCodexNativeWorkerReceiptValidation`** appears only as a nested/child test name inside `TestCodexNativeEvidenceReceiptSchemaDispatch` / `TestCodexNativeFourthReviewLegacyReplayInventory`'s own subtests (both already on the known-red list above); run alone it passes, confirmed against the pre-phase commit `b7341b75`.

**No `TestDirectRoute*`, `TestBackstop*`, `TestStatusLine*`, or any other test this plan or 206-01 introduced or touched appears in the failure list.** Zero new failures caused by Phase 206.

**Provenance note on this classification:** the structural argument (none of this plan's changed files overlap the failing tests' packages, confirmed by inspecting `git diff --stat` across every commit this plan made) was independently verified here; the specific baseline-list membership and the pre-phase-commit check for `TestCodexNativeWorkerReceiptValidation` were relayed from the orchestrator's own classification pass and are recorded as such rather than re-claimed as this executor's own independent re-run (a from-scratch baseline-worktree comparison was started but stopped, and the worktree removed, once the orchestrator's classification arrived and instructed not to re-run the full suite).

## Next Phase Readiness

- Phase 206 (Screens Reach the Owner) is complete: both plans done, the owner has reviewed and kept the direct route on by default, and the full suite shows zero new failures.
- Ready for `/gsd-verify-work 206` and `/gsd-plan-phase 207` (the messy practice project, the release gate).
- No blockers. The one open, pre-existing item (`aether codex-native-worker context-ack`, Phase 204.2's in-progress orphan) is tracked in `deferred-items.md` and belongs to that phase's own closing work, not to Phase 206 or 207.

---
*Phase: 206-screens-reach-the-owner*
*Completed: 2026-09-22*

## Self-Check: PASSED

- `cmd/status_line.go` — FOUND
- `cmd/status_line_test.go` — FOUND
- `scripts/proof-screens-reach-the-owner.sh` — FOUND
- `.planning/phases/206-screens-reach-the-owner/206-OWNER-VERDICT.md` — FOUND
- `.planning/phases/206-screens-reach-the-owner/deferred-items.md` — FOUND
- `git log --oneline --all | grep 4d005cc4` — FOUND
- `git log --oneline --all | grep 3be6b11e` — FOUND
- `git log --oneline --all | grep 1ca7ca1e` — FOUND
- `go test ./cmd -run 'TestStatusLine' -count=1` — re-ran, all 7 tests PASS
- `go test ./cmd -run 'TestStatusLineIsSafeUnderConcurrentReads' -race -count=1` — PASS
- `bash -n scripts/proof-screens-reach-the-owner.sh` — exit 0
- `.claude/settings.json` `statusLine.command == "aether status-line"` and `PostToolUse` still names `aether hook-post-tool-use` (keep-on, unchanged) — both confirmed
- `git status --porcelain -- .claude/commands .opencode/commands .codex .claude/rules` — empty
