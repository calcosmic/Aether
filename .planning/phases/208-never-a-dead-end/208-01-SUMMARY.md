---
phase: 208-never-a-dead-end
plan: 01
subsystem: cli
tags: [go, cobra, refusal, error-handling, cli, ux]

# Dependency graph
requires: []
provides:
  - "A typed `refusal` error (cmd/refusal.go) that always carries the one command that gets past it, rendered identically on both exit lanes"
  - "The one checked-in refusal table, `refusalRegistry` (cmd/refusal_register.go), holding 10 rows (3 colonize-finalize freshness refusals + 7 folded-in legacy hinted-error rows)"
  - "A local refusal log (`.aether/data/refusals.jsonl`) every refusal appends to, skipped for the four commands proven to write nothing"
  - "`aether report` / `/ant-report` — one command that writes a sendable markdown bundle (version, project status, recent refusals, recent failures, open item counts)"
  - "`aether colonize-finalize` recovers a missing `generated_at` from Aether's own receipt of the last `aether colonize --plan-only` run, and refuses by name (never a bare string) when it cannot"
affects: [208-02, 208-03, 208-04, 208-05, 208-06]

# Actuals (#2632)
actuals:
  tokens: 18879
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Typed refusal error (ID/What/Why/NextCommand/ProtectsWork/ExtraSteps) implementing `error`, looked up by id from one checked-in, id-sorted table"
    - "Pattern-match specificity resolved by longest-pattern-wins rather than table position, decoupling id-sort order from legacy substring-match precedence"
    - "Clock seam (`reportBundleNow`) for deterministic filename-collision testing, matching the existing `planCandidateNow` precedent"

key-files:
  created:
    - cmd/refusal.go
    - cmd/refusal_register.go
    - cmd/refusal_log.go
    - cmd/refusal_test.go
    - cmd/refusal_log_test.go
    - cmd/report_cmd.go
    - cmd/report_cmd_test.go
    - .aether/commands/report.yaml
    - .claude/commands/ant/report.md
    - .opencode/commands/ant/report.md
    - .planning/phases/208-never-a-dead-end/deferred-items.md
  modified:
    - cmd/root.go
    - cmd/helpers.go
    - cmd/codex_colonize.go
    - cmd/codex_colonize_finalize.go
    - cmd/ux_friendly_errors.go
    - cmd/ux_friendly_errors_test.go
    - cmd/wrapper_command_names.go
    - cmd/command_guide.go
    - cmd/testdata/regression_snapshot.json
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json

key-decisions:
  - "refusalRegistry is checked in by hand in ascending id order (not sorted at runtime), so TestRefusalRegisterIsSortedAndUnique proves the real committed order, not a sort call that would pass regardless of what is committed."
  - "friendlyErrorForPattern resolves ties by longest-pattern-wins rather than table position: alphabetical id order and legacy most-specific-first substring precedence are two genuinely different orderings for the same table, and only the pattern-length rule satisfies both (proven by TestFriendlyErrorSpecificityIsPatternLengthNotTableOrder against the invalid-charter-json vs. corrupted-colony-data conflict the old hand-curated order relied on)."
  - "The colonize-finalize recovery receipt matches on (transaction_id, root) rather than trusting a caller-supplied generated_at directly -- it only ever recovers a value Aether itself already wrote."
  - "outputRefusal and renderRefusalToExitWriter (factored out of ExitWithError's os.Exit call) are the two, and only two, call sites of appendRefusalToLog -- never one at each refusal site -- so the log can never miss a lane or double-count one."

requirements-completed: [UED-10, UED-15]

coverage:
  - id: D1
    description: "A refusal carries its next command as a typed field, rendered identically (same next-command line) on both the plain-text exit lane and the drawn-screen lane"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_test.go#TestRefusalCarriesItsNextCommandOnBothLanes"
        status: pass
      - kind: unit
        ref: "cmd/refusal_test.go#TestRefusalScreenTellsTheReaderToReportIt"
        status: pass
    human_judgment: false
  - id: D2
    description: "One checked-in, id-sorted refusal table (refusalRegistry) holds every refusal row in the program, including the seven rows folded in from the old hinted-error map"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_register.go#TestRefusalRegisterIsSortedAndUnique"
        status: pass
      - kind: unit
        ref: "cmd/ux_friendly_errors_test.go#TestFriendlyErrorsReadTheOneRefusalTable"
        status: pass
    human_judgment: false
  - id: D3
    description: "aether colonize-finalize recovers a missing generated_at from Aether's own receipt, and refuses by name (naming the one command that gets past it) when it cannot"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_test.go#TestColonizeFinalizeRecoversAMissingTimestamp"
        status: pass
      - kind: unit
        ref: "cmd/refusal_test.go#TestColonizeFinalizeRefusalNamesTheWayPast"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every refusal the owner sees is recorded to a local log, skipped for the four commands already proven to write nothing, and never able to change a rendered screen or exit code if the append fails"
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_log_test.go#TestRefusalLogAppendsAndNeverBlocks"
        status: pass
      - kind: unit
        ref: "cmd/refusal_log_test.go#TestRefusalLogIsSkippedForHookCommands"
        status: pass
    human_judgment: false
  - id: D5
    description: "aether report / /ant-report writes one sendable markdown bundle -- version, project status, recent refusals, recent failures, open item counts -- from a real project or an empty folder, with unique filenames and valid UTF-8"
    requirement: "UED-15"
    verification:
      - kind: unit
        ref: "cmd/report_cmd_test.go#TestReportBundleNamesWhatTheOwnerNeedsToSend"
        status: pass
      - kind: unit
        ref: "cmd/report_cmd_test.go#TestReportBundleFilenamesNeverCollide"
        status: pass
      - kind: unit
        ref: "cmd/report_cmd_test.go#TestReportBundleWithoutAProject"
        status: pass
      - kind: unit
        ref: "cmd/report_cmd_test.go#TestReportBundleIsValidUTF8ForEveryRecord"
        status: pass
      - kind: unit
        ref: "cmd/report_cmd_test.go#TestReportBundleOrdersRecordsNewestFirst"
        status: pass
      - kind: unit
        ref: "cmd/report_cmd_test.go#TestReportBundleCarriesNoEnvironmentSecrets"
        status: pass
    human_judgment: false
  - id: D6
    description: "/ant-report exists on both maintained platforms (Claude Code, OpenCode), byte-identical, and aether report is reachable (no new orphan)"
    requirement: "UED-15"
    verification:
      - kind: unit
        ref: "cmd/command_parity_test.go#TestClaudeOpenCodeCommandParity"
        status: pass
      - kind: unit
        ref: "cmd/hint_translation_test.go#TestWrapperCommandNamesMatchCanonicalCorpus"
        status: pass
    human_judgment: false

duration: 26min
completed: 2026-09-22
status: complete
---

# Phase 208 Plan 01: The Refusal Contract Summary

**A refusal is now a typed error carrying the one command that gets past it, rendered identically on both exit lanes, logged locally, and routed to a one-command `aether report` bundle -- proven end to end on the colonize-finalize dead end Phase 207's live journey actually hit.**

For the owner, in plain English: before this, when Aether's own colonize-finalize step hit a missing timestamp, it printed a bare error with no way forward, and the assistant chat just narrated a guess instead of telling you what to actually type. Now that exact case is fixed two ways -- Aether first tries to recover the missing information from its own records and quietly carries on, and only if it truly can't, it tells you the one real command to run next, in the same words whether it's printed as plain text or drawn as a screen. Every one of those moments is now written to a small local log, and a new command, `aether report` (or `/ant-report` in Claude Code and OpenCode), bundles that log plus your project's status into one file you can hand to whoever maintains Aether -- instead of asking a chat to patch Aether's own program to work around it.

## Performance

- **Duration:** 26 min
- **Started:** 2026-09-22T17:49:00Z (approx, first file read)
- **Completed:** 2026-09-22T18:15:48Z
- **Tasks:** 3
- **Files modified:** 22 (11 created, 11 modified)

## Accomplishments
- `cmd/refusal.go` / `cmd/refusal_register.go`: a typed `refusal` error with a required `NextCommand` field, looked up from one checked-in, id-sorted table (`refusalRegistry`), rendered through one shared `renderRefusal` on both the plain-text exit lane (`ExitWithError`) and the drawn-screen lane (`outputRefusal`).
- `cmd/codex_colonize_finalize.go` / `cmd/codex_colonize.go`: the real refusal this slice proves end to end. `aether colonize --plan-only` now writes a small receipt of its own manifest's `generated_at`; `aether colonize-finalize` recovers a missing timestamp from that receipt and carries on, and only refuses (naming `aether colonize --plan-only --force-resurvey`) when no matching receipt exists.
- `cmd/refusal_log.go`: every rendered refusal is appended to `.aether/data/refusals.jsonl`, skipped for the four commands (`hook-stop`, `hook-post-tool-use`, `hook-session-start`, `status-line`) already proven to write nothing, and never able to change the rendered screen or exit code if the append itself fails.
- Folded the seven-row legacy hinted-error map (`errorPatternMap`) into `refusalRegistry` -- there is now exactly one refusal/hinted-error table in the program, enforced by a structural AST scan (`TestFriendlyErrorsReadTheOneRefusalTable`).
- `cmd/report_cmd.go` + `.aether/commands/report.yaml` + `.claude/commands/ant/report.md` + `.opencode/commands/ant/report.md`: `aether report` / `/ant-report` writes a paste-ready markdown bundle (version, project goal/phase/next-command, last 20 refusals, last 20 failures, open blocker/issue/note counts) to `.aether/reports/`, with collision-proof filenames, valid-UTF-8 output, and no environment-variable leakage -- and behaves correctly with no project set up at all.

## Task Commits

Each task was committed atomically (Task 2 and Task 3 as TDD test-then-feat pairs):

1. **Task 1: One refusal, end to end** - `cebe676d` (feat) — typed refusal + register + both exit lanes + colonize-finalize recovery/refusal
2. **Task 2a: Tests for the one-table fold and local log** - `fadbd8db` (test)
2. **Task 2b: Fold hinted-error table + local refusal log** - `381fe03c` (feat)
3. **Task 3a: Tests for the report bundle command** - `cc8e33b6` (test)
3. **Task 3b: aether report command** - `bd4f3ce6` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `cmd/refusal.go` - typed `refusal` error, `refuse()`, `renderRefusal()`
- `cmd/refusal_register.go` - `refusalRow`, the checked-in `refusalRegistry` (10 rows), `refusalForID`, `refusalRowForPattern`, `refusalRegistryProblems`
- `cmd/refusal_log.go` - `refusalLogEntry`, `appendRefusalToLog`, `refusalLogEntries`, `refusalLogSkippedCommands`
- `cmd/report_cmd.go` - `aether report` command, `runReportBundle`, the six bundle sections
- `cmd/root.go` - `ExitWithError` refusal branch (`renderRefusalToExitWriter`, factored out of `os.Exit` for testability)
- `cmd/helpers.go` - `outputRefusal`, the drawn-screen lane's refusal entry point
- `cmd/codex_colonize.go` - writes `colonize-manifest-receipt.json` after building the plan-only manifest
- `cmd/codex_colonize_finalize.go` - freshness refusals converted to `refuse()`, receipt-based recovery, RunE wiring
- `cmd/ux_friendly_errors.go` - rewritten as thin adapters over `refusalRegistry` (no second table)
- `cmd/wrapper_command_names.go` / `cmd/command_guide.go` - `report` registered as a literal wrapper command
- `cmd/testdata/regression_snapshot.json` / `command_catalog.json` / `parity_snapshot.json` - refreshed goldens for the new command
- `.aether/commands/report.yaml`, `.claude/commands/ant/report.md`, `.opencode/commands/ant/report.md` - the `/ant-report` wrapper triplet

## Decisions Made
- Kept `refusalRegistry` hand-sorted in source (not runtime-sorted) so the id-order test proves the real committed order.
- Resolved the id-sort vs. pattern-specificity conflict with longest-pattern-wins rather than reordering the table or keeping a second ordering scheme.
- Matched the colonize-finalize recovery receipt on `(transaction_id, root)` so recovery only ever uses a value Aether itself wrote, never a caller's claim.

## Deviations from Plan

None architecturally — plan executed as written. Two small Rule 1/3-style fill-ins during implementation:

**1. [Rule 3 - Blocking] `permission-denied` legacy row had no `aether`-prefixed next command to promote**
- **Found during:** Task 2 (folding the legacy hinted-error rows)
- **Issue:** The old `permission denied` row's only next step was a non-Aether shell command (`ls -la <path>`), leaving no runnable Aether command to extract for `NextCommand`.
- **Fix:** Used `aether patrol` (the same diagnostic command the two adjacent storage-related rows already point at) as `NextCommand`, moved the original `ls -la` step into `ExtraSteps`.
- **Files modified:** cmd/refusal_register.go
- **Verification:** `TestEveryRefusalRowNamesANextCommand` passes; the row still surfaces the original inspection step.
- **Committed in:** 381fe03c

**2. [Rule 1 - Bug] `TestRefusalCarriesItsNextCommandOnBothLanes`'s first draft compared translated vs. untranslated output**
- **Found during:** Task 1, first test run
- **Issue:** The test set `AETHER_OUTPUT_MODE=visual` only before the visual-lane call, so the two lanes were compared under different rendering environments and the platform command-name translator (unrelated to the refusal contract itself) made them differ.
- **Fix:** Set the same environment before both calls and compare the rendered `Next:` line between the two outputs directly, rather than against a raw untranslated string.
- **Files modified:** cmd/refusal_test.go
- **Verification:** Test passes and still fails if the two lanes genuinely disagree (proven by the original failing run).
- **Committed in:** cebe676d

---

**Total deviations:** 2 auto-fixed (1 blocking gap-fill, 1 test-construction bug). **Impact on plan:** Neither touched scope or architecture; both are implementation-detail corrections needed for correctness.

## Issues Encountered

**Pre-existing, unrelated known-red found during verification:** `TestNoRegisteredSubcommandIsUnreferenced` fails on `aether codex-native-worker context-ack` ("registered but nothing calls it"). Verified via a disposable git worktree pinned to the commit immediately before this plan's Task 3 (`381fe03c`) that this failure predates all of this plan's changes and is unrelated to `aether report` or the refusal work (which introduces no new orphan). Logged in `.planning/phases/208-never-a-dead-end/deferred-items.md`; not fixed here (out of scope for this plan).

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The refusal contract, register, local log, and `aether report` command are all in place and proven on one real refusal (colonize-finalize's freshness check). Plans 208-02 through 208-06 (per the phase's affects list) can now convert their own dead ends to `refuse(...)` calls against this same table and rely on `appendRefusalToLog`/`aether report` without re-deriving any of this machinery.

No blockers. The one open item (`context-ack` orphan) is pre-existing and tracked in `deferred-items.md`, not a gate on this plan or on later 208-0x plans.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-22*

## Self-Check: PASSED

- All 11 files listed under key-files.created verified present on disk (`[ -f ]`).
- All 5 task commit hashes (cebe676d, fadbd8db, 381fe03c, cc8e33b6, bd4f3ce6) verified present in `git log --oneline --all`.
- Re-ran acceptance-criteria tests: `TestRefusal*`, `TestReportBundle*`, `TestFriendlyError*`, `TestColonize*`, `TestClaudeOpenCodeCommandParity`, `TestWrapperCommandNamesMatchCanonicalCorpus` all pass. `TestNoRegisteredSubcommandIsUnreferenced` fails on a pre-existing, unrelated orphan (`context-ack`), confirmed via disposable worktree to predate this plan.
- Re-ran plan-level `<verification>` commands: `go build ./cmd/aether` and `go vet ./cmd` clean; both named `go test` verification blocks pass except the pre-existing known-red above.
