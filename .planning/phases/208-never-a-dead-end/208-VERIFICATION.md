---
phase: 208-never-a-dead-end
verified: 2026-09-23T13:10:00Z
status: gaps_found
score: 4/5 roadmap success criteria verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Every refusal that stays names the one command that gets past it, enforced by a test, and the journey runs each printed command (ROADMAP Success Criterion 2, second clause)."
    status: partial
    reason: >
      The refusal contract itself is solid and proven (refusalRegistryProblems rejects empty/placeholder
      next-commands, TestBehaviourMatchesTheRefusalTable drives 19/27 rows at their real call sites, CR-01's
      placeholder fix is landed and tested). The machinery that runs a printed refusal's next command for real
      (journeyPrintedRefusals / journeyRunPrintedNextCommands, 208-07) is present, wired into journeyDriveStep,
      and unit-tested against synthetic transcripts (TestPrintedRefusalExtractorFindsARealRefusal,
      TestMenuFormNextCommandMapsBackToTheRuntimeCommand). But the phase's own binding proof requirement —
      "the proof is a real run in a real chat" (208-CONTEXT.md) — was attempted once (208-08,
      208-JOURNEY-RUN.md) and the single trial never reached the point of running any printed next command
      live: it stopped at step 2 of 14 ("survey") when the chat asked the owner for permission rather than
      running the refusal's own named `--force-resurvey` itself, and journeyDriveStep only calls
      journeyRunPrintedNextCommands after a step's own on-disk fact check passes (cmd/journey_live_test.go:381-399)
      — which this step's fact check did not. 208-JOURNEY-RUN.md says this outright under "Not proven by this
      run": "The generated_at-recovery code path (208-01, 208-07) was never exercised live." WINDOWS.md row 53
      is explicitly left open for this reason, and a new row 55 was opened for the distinct ask-vs-act product
      question this exposed. This is not a hidden gap — the project's own SUMMARY/JOURNEY-RUN/WINDOWS records
      are honest about it — but the roadmap success criterion, read literally, is not yet met by observed
      behaviour.
    artifacts:
      - path: ".planning/phases/208-never-a-dead-end/208-JOURNEY-RUN.md"
        issue: "The one real walk stopped before reaching any code path that runs a printed next command."
      - path: ".planning/WINDOWS.md"
        issue: "Row 53 stays open; row 55 records a new, distinct open question (ask vs. act) neither this phase nor Phase 209 closes."
    missing:
      - "A live journey run whose transcript actually reaches a step where a refusal's printed next command is executed for real, proving journeyRunPrintedNextCommands end-to-end rather than only against a synthetic fixture."
      - "An owner decision on WINDOWS row 55 (should a ProtectsWork=true stop's NextCommand be auto-run by an unattended -p chain, or always deferred to an interactive owner) before that live proof can be attempted again."
human_verification:
  - test: "Decide WINDOWS.md row 55 — should the driving assistant auto-run a ProtectsWork=true refusal's NextCommand when no owner is present to ask (an automated `-p` chain), or should it always defer to an interactive owner as it did in this run?"
    expected: "An owner ruling recorded (e.g. in a future phase's CONTEXT.md or WINDOWS.md itself), and .aether/commands/colonize.yaml's guidance updated to say which."
    why_human: "This is a product judgement call about how much autonomy an unattended verification chain should have, not something a test can decide."
---

# Phase 208: Never a Dead End — Verification Report

**Phase Goal:** The program warns and carries on unless work could be lost, and every refusal that remains says how to get past it.
**Verified:** 2026-09-23T13:10:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Every refusal is listed and sorted; one that does not protect against losing work is a warning that carries on. | ✓ VERIFIED | `cmd/refusal_register.go` holds one checked-in, id-sorted `refusalRegistry` (27 rows). `refusalRegistryProblems` fails on an unsorted/duplicate/empty-next-command row (`TestRefusalRegisterIsSortedAndUnique`, `TestEveryRefusalRowNamesANextCommand` — ran and pass). `warnAndCarryOn`/`renderWarning` exist as the "noticed, not stopped" half of the contract and are proven live on the three timeout-flag refusals (`TestBehaviourMatchesTheRefusalTable/invalid-timeout-value` — ran and passes; `TestNoRefusalBothWarnsAndStops` present). The 411 untyped call sites found across the ten declared lifecycle files are not silently ignored — they are counted on a shrink-only floor (`cmd/testdata/refusals/untyped-floor.json`, 411→386), enforced by `TestUntypedRefusalFloorOnlyShrinks` and `TestEveryRefusalSiteIsRegisteredOrCounted` (both ran and pass). "Listed and sorted" is satisfied for every registered row; the remainder is honestly counted, not hidden, matching the plan's own must-have wording. |
| 2 | Every refusal that stays names the one command that gets past it, enforced by a test, and the journey runs each printed command. | ✗ PARTIAL — see gap | The "names the one command, enforced by a test" clause is verified: CR-01's placeholder fix landed (`5bbe27a2`), `refusalRegistryProblems` now rejects any row whose `NextCommand` still carries `<`/`>` (`TestRefusalCheckCatchesAnUnsubstitutedPlaceholder` — ran and passes), and `grep '"<command>"'` over `cmd/refusal_register.go` returns nothing. The "journey runs each printed command" clause is built and unit-tested (`journeyPrintedRefusals`, `journeyRunPrintedNextCommands`, `TestPrintedRefusalExtractorFindsARealRefusal`, `TestMenuFormNextCommandMapsBackToTheRuntimeCommand` — all ran and pass) but was never exercised by a passing live journey step; the one real run (208-08) stopped before reaching that code path. See gap below. |
| 3 | A failed check adds tasks and carries on; status is worked out from what is on disk and the less-finished record is believed. | ✓ VERIFIED | `cmd/phase_progress_from_disk.go`'s `resolvePhaseProgressFromDisk` is wired into `cmd/status.go:1063` (`renderDashboard`) and believes the less-finished of the stored phase and the durable check record. `cmd/failed_check_carries_on_test.go`'s `TestFailedCheckAddsTheUnfinishedWorkAsTasks`, `TestFailedCheckAddsTheSameTasksOnlyOnce`, `TestFailedCheckNeverAdvancesOrVerifies`, `TestBlockedCheckAlwaysNamesANextCommand` all ran and pass. Real-world weight: this mechanism caused (and had fixed) a genuine regression in the plan_hash/build-manifest task-set comparison during this very phase (WINDOWS row 54, fixed `122ff01a`) — proof it was exercised for real, not just in isolated unit tests. |
| 4 | No screen advises a command without a menu version or uses an unexplained invented word; the failure log has a menu command; a failure the safety filter rejects is still recorded in readable words. | ✓ VERIFIED | `/ant-midden-review` menu command exists on both Claude and OpenCode (`.claude/commands/ant/midden-review.md`, `.opencode/commands/ant/midden-review.md`, `.aether/commands/midden-review.yaml`). `TestSixthBlockerGapIsClosed` and `TestScreenGuidanceNamesCommandsTheOwnerCanRun` ran and pass (widened check over status card, every Classic-voice screen, and every refusal's `NextCommand`, with a shrink-only reasoned allowlist for the one genuine non-command placeholder). `colony.NeutralizeForRecord`/`RedactSecretValues`-adjacent readability path proven by `TestRejectedFailureStillNamesWhatFailed` and `TestEmptyAndNilFailureText` (both ran and pass). |
| 5 | One command writes a report bundle, and every refusal tells a chat to report it rather than patch Aether. | ✓ VERIFIED | `aether report` / `/ant-report` exists (`cmd/report_cmd.go`, `.claude/commands/ant/report.md`, `.opencode/commands/ant/report.md`, `.aether/commands/report.yaml`). `renderRefusal`'s fixed template (`cmd/refusal.go:166`) reads: "Run `aether report` and send the file it writes to whoever maintains Aether -- please do not edit Aether's own program files to work around this." Secret redaction landed for the bundle (CR-03, `colony.RedactSecretValues`, `TestRedactSecretValuesCatchesCommonTokenShapes` ran and passes) and the `--output` failure-swallowing bug (WR-02) is fixed (`applyReportOutputOverride`). |

**Score:** 4/5 roadmap success criteria fully verified; 1 partially verified (mechanism built and unit-tested, live proof not yet achieved — openly documented by the project itself).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/refusal.go` | Typed refusal contract, both exit lanes | ✓ VERIFIED | Exists, `refuse()`/`renderRefusal`/`warnAndCarryOn` all present |
| `cmd/refusal_register.go` | One checked-in, id-sorted table | ✓ VERIFIED | 27 rows, `refusalRegistryProblems` guard present |
| `cmd/refusal_log.go` | Local refusal log, hook-excluded | ✓ VERIFIED | `refusalLogWriteExcludedForCommand` uses `strings.HasPrefix(command, "hook-")` (WR-03 fix) |
| `cmd/report_cmd.go` | `aether report` bundle writer | ✓ VERIFIED | Present; `applyReportOutputOverride` (WR-02); redaction wired (CR-03) |
| `cmd/phase_progress_from_disk.go` | Less-finished-of-two-records resolver | ✓ VERIFIED | Wired into `cmd/status.go` |
| `cmd/failed_check_carries_on_test.go` | Recovery-task tests | ✓ VERIFIED | 6+ named tests, ran and pass |
| `cmd/criterion_binding_dead_end_test.go` | Directory-artifact refusal tests | ✓ VERIFIED | `TestDirectoryArtifactIsRefusedByNameAtBuildTime`, `TestUnsatisfiableCriterionRoutesToTheOwnerAnswer` ran and pass |
| `cmd/refusal_enumerate_test.go` / `cmd/testdata/refusals/untyped-floor.json` | Shrink-only floor ratchet | ✓ VERIFIED | Floor recorded at 386, tests ran and pass |
| `cmd/refusal_printed_test.go` | Printed-refusal extractor tests | ✓ VERIFIED | Ran and pass; live exercise not proven (see gap) |
| `.claude/commands/ant/midden-review.md`, `.opencode/...`, `.aether/commands/midden-review.yaml` | Menu command for failure log | ✓ VERIFIED | All three present |
| `pkg/colony/secret_redaction.go` | `RedactSecretValues` | ✓ VERIFIED | Present, tested |
| `colony.Task.Origin` / `TaskOriginRecovery` | Structural recovery-task marker (CR-02) | ✓ VERIFIED | `pkg/colony/colony.go:835-840`, consumed at `cmd/codex_continue.go:3178,3282` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/root.go` `ExitWithError` | `renderRefusal` | plain-text exit lane | ✓ WIRED | Confirmed present in refusal.go/root.go |
| `cmd/codex_continue.go` `assessCodexContinue` | `classifyContinueTaskAssessment` | per-task verdict | ✓ WIRED | `TestOneFailingCriterionMarksOnlyItsOwnTask` etc. (208-02, unit-verified) |
| `cmd/status.go` `renderDashboard` | `resolvePhaseProgressFromDisk` | less-finished record shown | ✓ WIRED | `cmd/status.go:1063` |
| `cmd/status.go` guidance | `.claude/commands/ant/midden-review.md` | menu command exists | ✓ WIRED | `TestScreenGuidanceNamesCommandsTheOwnerCanRun` ran and passes |
| `cmd/journey.go` `journeyPrintedRefusals` | `cmd/journey_live_test.go` `journeyDriveStep` | run printed command after fact check passes | ✓ WIRED (mechanism) / ✗ NOT LIVE-EXERCISED | Code path confirmed at `cmd/journey_live_test.go:381-399`; gated on the step's on-disk fact check passing first — that gate was never cleared in the one real run |
| `cmd/criterion_evidence.go` `validatePhaseCriterionEvidenceAgainstDisk` | build/build-finalize/continue/verify | directory-artifact refusal at one chokepoint | ✓ WIRED | `TestDirectoryArtifactIsRefusedByNameAtBuildTime` ran and passes |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Refusal register integrity | `go test ./cmd -run '^(TestEveryRefusalRowNamesANextCommand\|TestRefusalRegisterIsSortedAndUnique\|TestRefusalCheckCatchesAnUnsubstitutedPlaceholder\|TestBehaviourMatchesTheRefusalTable\|TestEveryRefusalSiteIsRegisteredOrCounted\|TestUntypedRefusalFloorOnlyShrinks)$' -v` | All 6 groups PASS (27/27 behaviour-table subtests pass) | ✓ PASS |
| Printed-refusal extraction | `go test ./cmd -run '^(TestPrintedRefusalExtractorFindsARealRefusal\|TestPrintedRefusalExtractorIgnoresAssistantOwnText)$' -v` | PASS | ✓ PASS |
| Recovery-task write-back | `go test ./cmd -run '^(TestFailedCheckAddsTheUnfinishedWorkAsTasks\|TestFailedCheckNeverAdvancesOrVerifies\|TestHumanTaskWithRecoveryLikeGoalIsNeverExcluded)$' -v` | PASS | ✓ PASS |
| Directory-artifact refusal | `go test ./cmd -run '^(TestDirectoryArtifactIsRefusedByNameAtBuildTime\|TestUnsatisfiableCriterionRoutesToTheOwnerAnswer)$' -v` | PASS | ✓ PASS |
| Menu-command guidance / midden-review | `go test ./cmd -run '^(TestSixthBlockerGapIsClosed\|TestScreenGuidanceNamesCommandsTheOwnerCanRun\|TestExpectedRedRegisterIsEmptyAndTheFiveRevertsStand)$' -v` | PASS | ✓ PASS |
| Readable failure records | `go test ./cmd -run '^(TestRejectedFailureStillNamesWhatFailed\|TestEmptyAndNilFailureText)$' -v` | PASS | ✓ PASS |
| Secret redaction | `go test ./pkg/colony -run '^(TestRedactSecretValuesCatchesCommonTokenShapes\|TestRedactSecretValuesLeavesOrdinaryTextAlone\|TestRedactSecretValuesEmptyInput)$' -v` | PASS | ✓ PASS |
| Build / vet | `go build ./...`, `go vet ./cmd/ ./pkg/...` | clean | ✓ PASS |
| Full `go test ./...` | Already running in background (orchestrator-started); not duplicated per instructions | not re-run | ? SKIP (see note) |

Note: a full, unscoped `go test ./cmd -count=1 -timeout 90m` was already run once by the 208-08 executor itself (per its SUMMARY and WINDOWS row 56) and found 30 pre-existing failures unrelated to this phase's declared files, cross-referenced against the known-red baseline. That evidence, plus this verifier's own scoped runs above (all passing), is treated as sufficient; the background full suite triggered by the orchestrator was not duplicated or waited on per the verification instructions.

### Probe Execution

Not applicable — no `scripts/*/tests/probe-*.sh` declared or referenced by this phase's plans/summaries. The equivalent proof artifact for this phase is the messy-practice-project journey (`make eval-gate-journey`), whose one real, owner-approved trial is recorded in `208-JOURNEY-RUN.md` and discussed above (Success Criterion 2 gap).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| UED-10 | 208-01, 02, 05, 07, 08 | Every refusal carries the one command that gets past it and says whether it protects against losing work | ⚠️ PARTIAL | Code/tests solid; live journey proof not achieved (see gap) |
| UED-11 | 208-06 | A refusal that does not protect against losing work is a warning that carries on | ✓ SATISFIED | `TestNoRefusalBothWarnsAndStops`, `TestBehaviourMatchesTheRefusalTable` |
| UED-12 | 208-02, 04 | A failed check adds tasks and carries on | ✓ SATISFIED | `cmd/failed_check_carries_on_test.go` suite passes |
| UED-13 | 208-03 | No screen advises a command with no menu version; no unexplained invented word; failure log has a menu command | ✓ SATISFIED | `TestScreenGuidanceNamesCommandsTheOwnerCanRun`, `TestSixthBlockerGapIsClosed` |
| UED-14 | 208-03 | A failure whose text the safety filter rejects is still recorded in readable words | ✓ SATISFIED | `TestRejectedFailureStillNamesWhatFailed` |
| UED-15 | 208-01 | One command writes a report bundle; every refusal tells a chat to report it rather than patch Aether | ✓ SATISFIED | `aether report` exists; `renderRefusal` template text confirmed |

No orphaned requirements found — REQUIREMENTS.md lists exactly UED-10..15 for Phase 208, and all six are claimed across the eight plans' frontmatter.

REQUIREMENTS.md itself currently shows all six as `[x]` checked. This verification confirms five are fully backed by passing, ran-live evidence; UED-10 is backed by solid code and unit-test evidence but its own declared proof method (the journey, run for real) has not yet cleared the specific step needed to exercise the printed-next-command-execution behaviour live.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/placeholder/stub patterns found in this phase's key modified files (`cmd/refusal.go`, `cmd/refusal_register.go`, `cmd/refusal_log.go`, `cmd/report_cmd.go`, `cmd/phase_progress_from_disk.go`, `cmd/criterion_evidence.go`, `cmd/criterion_owner_confirmation.go`, `cmd/journey.go`, `pkg/colony/secret_redaction.go`, `pkg/colony/colony.go`, `cmd/codex_continue.go`, `cmd/deterministic_floor.go`).

The eight-finding code review (208-REVIEW.md: 3 critical, 5 warning) was independently re-checked against the current HEAD (`c6096b82`) rather than trusted from its own "fixed" claims:

| ID | Claimed fix | Independently confirmed |
|----|-------------|--------------------------|
| CR-01 | Placeholder `<command>` removed; structural guard added | ✓ Confirmed — `grep '"<command>"'` returns nothing outside a code comment; `refusalRegistryProblems` rejects `<`/`>` |
| CR-02 | `colony.Task.Origin`/`TaskOriginRecovery` structural marker | ✓ Confirmed — field and constant exist, consumed at the three call sites named |
| CR-03 | `colony.RedactSecretValues` on report bundle | ✓ Confirmed — function exists, wired into `report_cmd.go:250`, tests pass |
| WR-01 | Partial — `continueNextCommandForBlocked` surfaces the registered `NextCommand` | ✓ Confirmed partial fix present; the SUMMARY/REVIEW's own admission that full routing through `renderRefusal` is deferred is accurate and not overstated |
| WR-02 | `applyReportOutputOverride` warns on failed `--output` | ✓ Confirmed |
| WR-03 | `hook-*` prefix exclusion, not enumeration | ✓ Confirmed |
| WR-04 | `EvalSymlinks` before containment check | ✓ Confirmed |
| WR-05 | Extraction restricted to `tool_result` blocks | ✓ Confirmed |

No blockers found in the review-fix layer.

### Human Verification Required

1. **WINDOWS.md row 55 — ask-vs-act product decision.**
   Test: none — this is a product judgement call, not a code check.
   Expected: An owner ruling on whether an unattended `-p` chain should auto-run a `ProtectsWork: true` stop refusal's `NextCommand`, or always defer to an interactive owner (as the real chat did in this run). `.aether/commands/colonize.yaml`'s own "follow the runtime recovery guidance" instruction is currently silent on this.
   Why human: This is exactly the kind of scope-changing, business-rule decision CLAUDE.md reserves for the owner ("Ask me about... Anything where guessing wrong would materially change what gets built").

### Gaps Summary

Seven of eight plans (208-01 through 208-07) landed clean, well-tested code that closes every field-reported dead end named in WINDOWS.md rows 49-52 and builds the full refusal/report/recovery-task/menu-guidance machinery the phase goal calls for — all independently re-run and confirmed passing by this verification, not merely trusted from SUMMARY.md claims. The code review's 8 findings were also independently re-checked against the current commit and all are genuinely fixed (7 fully, 1 explicitly and honestly partial).

The one open item is plan 208-08's own declared must-have: a live journey run whose survey step passes, proving the "journey runs each printed command" half of ROADMAP Success Criterion 2. The one owner-approved real-money trial did not get that far — it stopped at step 2 of 14 when the driving assistant chose to ask the owner rather than auto-run the refusal's own named recovery command, a scenario an unattended `-p` chain cannot resolve on its own. This is not a code defect this phase introduced (the underlying refusal predates Phase 208 per git blame) and it is not hidden — 208-JOURNEY-RUN.md, WINDOWS row 53 (left open) and row 55 (new) document it plainly, exactly as CLAUDE.md's Definition of Done requires ("a documentation claim about runtime behaviour must be testable or removed" — here it is honestly left untested rather than claimed).

Per this project's own standard ("do not mark it verified" for an unmet must-have), this verification reports the phase as **gaps_found** rather than passed, scoped to this single, well-understood item. Everything else in the phase is solid.

---

_Verified: 2026-09-23T13:10:00Z_
_Verifier: Claude (gsd-verifier)_
