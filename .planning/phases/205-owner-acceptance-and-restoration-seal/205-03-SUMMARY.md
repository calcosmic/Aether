---
phase: 205-owner-acceptance-and-restoration-seal
plan: 03
subsystem: lifecycle-closeout
tags: [entomb, seal, seal-finalize, resume, lifecycle-transaction, json-output]

# Dependency graph
requires:
  - phase: 199-front-door-and-classic-contract
    provides: the pause/resume/seal/entomb lifecycle transaction machinery this plan patches (session_flow_cmds.go, entomb_cmd.go, seal_confirmation.go)
provides:
  - Finish-and-archive (`aether entomb`) completes on a colony whose hand-off note is absent, by synthesizing a minimal stand-in from already-loaded colony state and seal outcome
  - `aether resume` writes a fresh hand-off note in the same lifecycle transaction it removes the stale one from, so an entomb run immediately after never sees an empty required slot
  - `aether seal-finalize`'s machine-readable stdout is pure JSON (no stray prose), while the direct `aether seal` command's documented mixed prose+JSON contract is preserved
affects: [205-owner-acceptance-walkthrough-plans, seal-finalize-wrapper-contract]

# Actuals (#2632)
actuals:
  tokens: 6704
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fallback-on-genuinely-missing-required-source, scoped to one logical path, marked with RecoveryProvenanceReconstructed rather than a new vocabulary"
    - "Resume replaces (never merely removes) a required-by-later-command file within the same lifecycle transaction, since the transaction coordinator refuses two declarations for one path"
    - "Machine-mode output suppression scoped to a command's own quiet classification (isExplicitlyQuietCommand/currentStreamingCommand), not blanket shouldRenderVisualOutput, to avoid disturbing a documented mixed-output contract shared by another caller of the same gate"

key-files:
  created:
    - cmd/lifecycle_closeout_contract_test.go
  modified:
    - cmd/entomb_cmd.go
    - cmd/entomb_manifest.go
    - cmd/session_flow_cmds.go
    - cmd/seal_confirmation.go
    - cmd/pause_resume_199_test.go
    - cmd/session_flow_cmds_test.go

key-decisions:
  - "The archive fallback covers exactly the tombstone_input (hand-off note) path; every other required source (COLONY_STATE.json, seal/outcome.json, etc.) still fails hard on a genuinely missing file, proven by a dedicated test that deletes COLONY_STATE.json instead."
  - "Resume's HANDOFF.md removal was replaced by a write of fresh content in the same DeclareRemoval slot, not paired with a second declaration, because lifecycleTransaction.declare() refuses a duplicate target path — the single write IS the pairing."
  - "seal-finalize's JSON-purity fix is scoped to isExplicitlyQuietCommand(currentStreamingCommand) (the existing '-finalize commands are quiet' classification) combined with shouldRenderVisualOutput, not shouldRenderVisualOutput alone, after discovering the direct interactive `aether seal` command shares the same gate and has a documented, tested contract of mixing prose with its JSON envelope (cmd/seal_confirmation_test.go's executeSealForConfirmationTest/lastJSONLine)."
  - "Two pre-existing tests (TestPauseResume199Confirmed, TestResumeColonyRestoresSessionAndClearsHandoff) that pinned the old 'resume deletes HANDOFF.md, nothing replaces it' behavior were updated to assert the new, intentional behavior instead of being left red."

patterns-established:
  - "A synthesized/fallback source is marked on the manifest struct itself (entombArchiveSource.Synthesized) and carried through to the archived colony.ArchiveEntry.Provenance as Reconstructed vs Confirmed — reusing the existing RecoveryProvenance vocabulary rather than inventing a parallel one."

requirements-completed: [PROOF-05]

coverage:
  - id: D1
    description: "Archiving a finished project succeeds when the hand-off note file is absent, synthesizing a minimal stand-in from already-loaded colony state and seal outcome, and a genuinely missing required source (COLONY_STATE.json) still fails the archive by name."
    requirement: "PROOF-05"
    verification:
      - kind: unit
        ref: "cmd/lifecycle_closeout_contract_test.go#TestEntombSynthesizedInputIsMarkedAsSynthesized"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_closeout_contract_test.go#TestEntombStillFailsOnAGenuinelyMissingSource"
        status: pass
    human_judgment: false
  - id: D2
    description: "A return-to-work run (resume) followed by a seal and an archive, with no other command in between, completes without a missing-source error, because resume now leaves a fresh hand-off note behind instead of an empty slot."
    requirement: "PROOF-05"
    verification:
      - kind: integration
        ref: "cmd/lifecycle_closeout_contract_test.go#TestResumeThenSealThenEntombCompletes"
        status: pass
      - kind: unit
        ref: "cmd/pause_resume_199_test.go#TestPauseResume199Confirmed"
        status: pass
      - kind: unit
        ref: "cmd/session_flow_cmds_test.go#TestResumeColonyRestoresSessionAndClearsHandoff"
        status: pass
    human_judgment: false
  - id: D3
    description: "seal-finalize's machine-readable standard output parses as a single JSON document (both the proceed and the do-not-proceed branch), while the direct `aether seal` command's human-mode card and question, and its documented mixed prose+JSON contract, are unchanged."
    requirement: "PROOF-05"
    verification:
      - kind: unit
        ref: "cmd/lifecycle_closeout_contract_test.go#TestSealFinalizeJSONStdoutIsPureJSON"
        status: pass
      - kind: unit
        ref: "cmd/lifecycle_closeout_contract_test.go#TestSealFinalizeHumanOutputIsUnchanged"
        status: pass
      - kind: unit
        ref: "cmd/seal_confirmation_test.go#TestSealIssueWarning"
        status: pass
      - kind: unit
        ref: "cmd/seal_confirmation_test.go#TestSealAlwaysAsksBeforeFinishing"
        status: pass
    human_judgment: false
  - id: D4
    description: "Regression scope check across the entomb, session-flow and seal-confirmation packages: every scoped test passes except one pre-existing, unrelated failure confirmed present identically before this plan's changes."
    verification:
      - kind: integration
        ref: "go test ./cmd/ -run 'Entomb|Seal|Resume|SessionFlow|Lifecycle'"
        status: pass
    human_judgment: true
    rationale: "The scoped run's one remaining failure (TestEveryLifecycleCommandEndsWithNextAction) was independently reproduced against the pre-change source in the identical scoped combination and passes in isolation — classified as pre-existing test-order flakiness, not a regression from this plan. A human should confirm this classification is acceptable before treating the phase's broader gate as clean."

# Metrics
duration: ~65min (approximate — PLAN_START_TIME was not captured at session start; based on commit and investigation span)
completed: 2026-09-15
status: complete
---

# Phase 205 Plan 03: Finish-and-Archive Completion Fixes Summary

**Entomb synthesizes a minimal hand-off stand-in when the note is genuinely absent, resume now writes a fresh note in the same transaction it removes the stale one from, and seal-finalize's machine-readable output is pure JSON while the direct `aether seal` command's mixed prose+JSON contract is preserved.**

## Performance

- **Duration:** ~65 min (approximate)
- **Tasks:** 3 completed (Task 1: tracer/tdd, Task 2: auto/tdd, Task 3: auto verification-only)
- **Files modified:** 6 modified, 1 created (7 total)
- **Commits:** 4

## Accomplishments

- `aether entomb`'s archive preflight no longer hard-fails when `.aether/HANDOFF.md` is missing: `addRequiredOrSynthesizedTombstoneInput` builds a minimal stand-in from the colony state and seal outcome already loaded in the same preflight scope (goal, phase count/completion state, seal disposition), and marks the source `Synthesized` so the archive never presents invented content as retrieved content. Every other required source (state, seal outcome, findings, learnings, etc.) still fails hard on a genuinely missing file, unchanged.
- `aether resume` no longer leaves an empty slot: its lifecycle transaction now declares a *write* of a fresh, minimal hand-off note where it previously declared only a *removal* — the transaction coordinator refuses two declarations on one path, so the write itself is the "pairing" the plan called for.
- `aether seal-finalize`'s machine-readable stdout (`AETHER_OUTPUT_MODE=json`) is now pure JSON on both the "proceed" and "do not proceed" branches — no preflight card, no confirmation question ahead of the envelope. The direct, interactive `aether seal` command (which shares the same confirmation gate) keeps its existing, documented, tested behavior of mixing prose with its final JSON line.
- Reproduced the exact field-report sequence end-to-end: pause → resume → (simulated) seal → archive preflight, asserting the whole chain completes with no missing-source error.

## Task Commits

Each task was committed as a RED/GREEN pair (2 commits per task, per each task's `tdd="true"`):

1. **Task 1 (RED): add failing tests for entomb hand-off synthesis** - `c3f1264d` (test)
2. **Task 1 (GREEN): synthesize entomb hand-off input, resume leaves a fresh note** - `53afd2a4` (feat)
3. **Task 2 (RED): add failing tests for pure JSON seal-finalize output** - `5c13a642` (test)
4. **Task 2 (GREEN): keep seal-finalize's machine-readable output pure JSON** - `f2b1528f` (feat)

Task 3 (regression scope check) made no code changes — see "Deviations from Plan" and "Issues Encountered" below for what it found.

## Files Created/Modified

- `cmd/lifecycle_closeout_contract_test.go` (new) - `TestResumeThenSealThenEntombCompletes`, `TestEntombStillFailsOnAGenuinelyMissingSource`, `TestEntombSynthesizedInputIsMarkedAsSynthesized`, `TestSealFinalizeJSONStdoutIsPureJSON`, `TestSealFinalizeHumanOutputIsUnchanged`
- `cmd/entomb_cmd.go` - `addRequiredOrSynthesizedTombstoneInput` (new helper), `buildSynthesizedEntombTombstoneInput` (new stand-in builder), `entombSourceDigest` updated to treat a synthesized source's baseline as "missing"
- `cmd/entomb_manifest.go` - `entombArchiveSource.Synthesized` field; `buildEntombArchiveManifest` sets `colony.ArchiveEntry.Provenance` to `Reconstructed` vs `Confirmed` accordingly
- `cmd/session_flow_cmds.go` - `resumeColonyAt`'s transaction declares a write of a fresh `buildHandoffDocument(...)` note instead of a bare removal
- `cmd/seal_confirmation.go` - `sealPreflightGateShouldWriteHumanOutput` gates the two human writes in `runSealPreflightConfirmationGate` on `isExplicitlyQuietCommand(currentStreamingCommand)` + `shouldRenderVisualOutput(stdout)`
- `cmd/pause_resume_199_test.go` - `TestPauseResume199Confirmed` updated to assert the resume-written note exists and carries real content, not that the file was removed
- `cmd/session_flow_cmds_test.go` - `TestResumeColonyRestoresSessionAndClearsHandoff` updated the same way

## Decisions Made

- The synthesized fallback is scoped to exactly one logical path (`tombstone_input`) inside a dedicated helper (`addRequiredOrSynthesizedTombstoneInput`) rather than adding a general "optional" flag to `addFile`'s existing required-source loop — every other required source keeps failing hard by name, proven by `TestEntombStillFailsOnAGenuinelyMissingSource` deleting `COLONY_STATE.json` instead.
- Resume's fix is a single `DeclareWrite` replacing the prior `DeclareRemoval`, not two separate declarations, because `lifecycleTransaction.declare()` rejects a duplicate target path — this is the correct way to "pair a removal with a write" on the same path within one transaction.
- The seal-finalize JSON-purity fix reuses `isExplicitlyQuietCommand`/`currentStreamingCommand` — existing infrastructure already used elsewhere in this file's package to classify "-finalize commands are quiet" — rather than inventing a new flag or environment variable, satisfying the acceptance criterion directly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Scoped the seal-finalize JSON-purity fix to quiet ("-finalize") commands only, after an initial broader fix broke a documented, tested contract**
- **Found during:** Task 2, while running the plan's Task 3 regression scope check against my first implementation
- **Issue:** My first pass gated `runSealPreflightConfirmationGate`'s two human writes on `shouldRenderVisualOutput(stdout)` alone. That broke `TestSealIssueWarning` and `TestSealAlwaysAsksBeforeFinishing` — both of which exercise the DIRECT, interactive `aether seal` command (not `seal-finalize`), which shares the same confirmation gate and has a documented, deliberately-tested contract of mixing prose with its final JSON line on stdout (`cmd/seal_confirmation_test.go`'s `executeSealForConfirmationTest`/`lastJSONLine` doc comments say so explicitly). The test suite's own `TestMain` sets `AETHER_OUTPUT_MODE=json` globally for every test by default, which is why this was only surfaced by actually running the tests, not by reading the code.
- **Fix:** Added `sealPreflightGateShouldWriteHumanOutput`, which only suppresses the two human writes when the current top-level command is classified "quiet" (`isExplicitlyQuietCommand`, the same "-finalize commands are quiet" rule `cmd/codex_visuals.go` already uses) AND machine-readable output was requested. The direct `aether seal` path (not a "-finalize" command) is unaffected in any mode.
- **Files modified:** `cmd/seal_confirmation.go`, `cmd/lifecycle_closeout_contract_test.go` (tests set `currentStreamingCommand = "seal-finalize"` to simulate the real invocation context)
- **Verification:** `TestSealIssueWarning`, `TestSealAlwaysAsksBeforeFinishing` (all 5 subtests), `TestSealFinalizeJSONStdoutIsPureJSON`, `TestSealFinalizeHumanOutputIsUnchanged` all pass together
- **Committed in:** `f2b1528f` (Task 2 GREEN commit)

**2. [Rule 1 - Bug] Updated two pre-existing tests that pinned the old "resume deletes the hand-off note" behavior**
- **Found during:** Task 1, running the plan's Task 3 regression scope check
- **Issue:** `TestPauseResume199Confirmed` (`cmd/pause_resume_199_test.go`) and `TestResumeColonyRestoresSessionAndClearsHandoff` (`cmd/session_flow_cmds_test.go`) both asserted `.aether/HANDOFF.md` is absent after resume — the exact bug this plan fixes.
- **Fix:** Updated both assertions to require the file exist and carry real hand-off content (`# Colony Handoff`), matching the new, intentional behavior.
- **Files modified:** `cmd/pause_resume_199_test.go`, `cmd/session_flow_cmds_test.go`
- **Verification:** Both tests pass
- **Committed in:** `53afd2a4` (Task 1 GREEN commit)

---

**Total deviations:** 2 auto-fixed (1 scope-narrowing bug fix on my own initial implementation, 1 pre-existing-test update)
**Impact on plan:** Both were necessary to land a correct fix without silently breaking existing, deliberately-tested behavior. No scope creep — both stayed inside the plan's named files plus the two directly-affected pre-existing test files.

## Issues Encountered

- **`TestEveryLifecycleCommandEndsWithNextAction`** failed inside the Task 3 scoped run (`go test ./cmd/ -run 'Entomb|Seal|Resume|SessionFlow|Lifecycle'`), both before and after this plan's changes, but passes in isolation. I reproduced this by reverting all of this plan's source changes and re-running the identical scoped pattern — it failed identically against the pre-change source, proving it is pre-existing test-order/pollution flakiness specific to this scoped combination, not a regression introduced here. It is NOT in the repo's static 2026-09-14 known-red baseline list (17 entries), so it is recorded here rather than silently treated as baseline, and logged to `.planning/phases/205-owner-acceptance-and-restoration-seal/deferred-items.md`. Not fixed — out of this plan's scope (Task 3 directs classifying, not fixing every unrelated finding, and this one's root cause is in `cmd/lifecycle_next_action_coverage_test.go`, a file this plan never touches). **Consequence for Task 3's own acceptance criterion:** the scoped command's actual exit code is 1 (this one failure), not the literally-worded 0 — the criterion is satisfied in substance (every failure classified, zero regressions from this plan) but not in literal exit code, because of this pre-existing, independently-reproduced defect.
- **`writeVisualOutput` does not gate on output mode for writing at all** — only for command-name translation (confirmed by reading the function and by the `TestSealIssueWarning`/`TestSealAlwaysAsksBeforeFinishing` investigation above). This is worth flagging for anyone extending machine-mode output purity elsewhere in this codebase: every other `visualFprint`/`writeVisualOutput` call site in `cmd/` that has NOT been explicitly gated by its caller (the way `outputWorkflow` and now `runSealPreflightConfirmationGate` do) will ALSO write human text regardless of `AETHER_OUTPUT_MODE`. Not fixed here — out of this plan's scope, which only names the seal-finalize confirmation gate.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Journey 1 (seal → archive → init) from the phase's owner walk-through plan can now run without hitting the field-reported entomb/resume defect.
- `AETHER_OUTPUT_MODE=json aether seal-finalize` is safe for a wrapper to parse as pure JSON.
- Flag for a later phase or plan (not blocking this one): the broader "which `visualFprint` call sites leak human text under machine mode" question above is unresolved outside this one gate.

---
*Phase: 205-owner-acceptance-and-restoration-seal*
*Completed: 2026-09-15*

## Self-Check: PASSED

- `cmd/lifecycle_closeout_contract_test.go` exists on disk: confirmed
- `cmd/entomb_cmd.go`, `cmd/entomb_manifest.go`, `cmd/session_flow_cmds.go`, `cmd/seal_confirmation.go`, `cmd/pause_resume_199_test.go`, `cmd/session_flow_cmds_test.go` modified on disk: confirmed
- All 4 commit hashes (`c3f1264d`, `53afd2a4`, `5c13a642`, `f2b1528f`) present in `git log --oneline`: confirmed
- `go build ./cmd/...`, `go build ./cmd/aether`, `go vet ./cmd/...`: all exit 0
- `go test ./cmd/ -run 'TestResumeThenSealThenEntombCompletes|TestEntombStillFailsOnAGenuinelyMissingSource|TestEntombSynthesizedInputIsMarkedAsSynthesized|TestSealFinalizeJSONStdoutIsPureJSON|TestSealFinalizeHumanOutputIsUnchanged' -count=1 -timeout 90m`: exit 0
- `go test ./cmd/ -run 'Entomb|Seal|Resume|SessionFlow|Lifecycle' -count=1 -timeout 90m`: exit code 1 (380 passed, 1 failed) — the sole failure is `TestEveryLifecycleCommandEndsWithNextAction`, independently reproduced as identical against the pre-change source in the same scoped combination and passing in isolation; see "Issues Encountered" and `deferred-items.md`
