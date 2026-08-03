---
phase: 160-fail-loudly
verified: 2026-08-03T00:00:00Z
status: passed
score: 13/13 requirements verified
overrides_applied: 0
verification_mode: goal-backward, execution-first
---

# Phase 160: Fail Loudly — Verification Report

**Phase Goal:** Every playbook and wrapper CLI call either succeeds or visibly fails — no call silently swallowed by a `/dev/null` redirect — and the drift-detection test catches positional-argument drift, not only `--flag` drift. Seven confirmed-broken calls fixed; `control-ts/` deleted standalone; `.aether/ts-host/` and `.aether/ts/` kept and untouched.

**Verified:** 2026-08-03
**Status:** passed
**Re-verification:** No — initial verification (the v1.25 mid-milestone audit found no VERIFICATION.md existed)

**Method:** Every requirement below was checked by executing commands and tests in this session, not by reading SUMMARY claims. Where a check is weaker than full execution, that is stated explicitly in the evidence column.

## Environment Gates (all executed this session)

| Gate | Command | Result |
|---|---|---|
| Build | `go build ./cmd/aether` | OK |
| cmd suite | `go test ./cmd -count=1` | ok (267s) |
| codex suite | `go test ./pkg/codex -count=1` | ok (19s) |

## Requirements Coverage

| Requirement | Status | Evidence (executed this session) |
|---|---|---|
| LOUD-01 (survey-load) | ✓ VERIFIED | Requirement satisfied via the "playbooks corrected" arm, done by removal: `./aether survey-load "test-phase"` → `Error: unknown command "survey-load"`. `TestSurveyLoadAbsentAndUncalled` PASS (subtests `NotRegistered` + `NotReferenced` — no stale caller anywhere). Survey context now flows via `resolveSurveySection` (reconnected pre-phase, commit `281dd34a`). |
| LOUD-02 (check-antipattern gate) | ✓ VERIFIED | Executed with a planted secret: `./aether check-antipattern <file>` (positional) AND `--file <file>` both return `clean:false, exposed-secret at line 3`. Gate pipeline: `TestContinueAntiPatternGate*` (4 tests), `TestAntiPatternScanFailureHardBlocks`, and the UAT-found bug pin `TestAntiPatternGateDoesNotCountAbsentFilesAsScanned` (2 subtests) — all PASS. Absent file at CLI level still returns clean by design; the gate level hard-blocks, which is what the test pins. |
| LOUD-03 (five remaining calls) | ✓ VERIFIED | All five exist with real cobra contracts (`--help` executed for `print-next-up`, `verify-claims`, `state-checkpoint`, `generate-progress-bar`, `skill-detect`). `TestCommandCallsMatchCobraContracts` PASS: "audited 882 documented invocations across 5 corpora", 0 violations. |
| LOUD-04 (positional drift detection) | ✓ VERIFIED | `TestAuditDetectsPositionalDrift` PASS — self-test feeds a positional to a registered `cobra.NoArgs` command and asserts the audit flags it. The old blind spot (regex matching only `--flag` tokens) is closed by validating against cobra's own `Args` validators. 160-07 SUMMARY records a verified deliberate-regression proof (mutation → test FAIL → revert). |
| LOUD-05 (verified by execution, not regex) | ✓ VERIFIED (with stated limit) | `TestCommandCallsMatchCobraContracts` resolves all 882 documented invocations against the real cobra command tree using cobra's own `Find` + `ValidateArgs` + real flag sets — machinery execution, not text regex. **Stated limitation (honest):** the audit deliberately stops short of invoking `RunE`, because really running ~882 invocations would mutate colony state, delete data, and publish to the hub. A runtime failure inside `RunE` with a well-formed command line is not caught; this limit is documented in the test file itself (`cmd/command_call_audit_test.go:24-29`). The seven originally-broken calls were additionally confirmed by real execution during the phase, and this session re-executed `survey-load` and both `check-antipattern` forms. |
| LOUD-06 (no load-bearing /dev/null) | ✓ VERIFIED | `TestLiveWrapperStderrSuppressionCount` PASS. Two invariants: (1) total `2>/dev/null` sites across both live wrapper dirs ≤ 8 (all benign: git/grep fallbacks in dream.md, archaeology.md, mirrored per platform); (2) the sharper one — **no line containing the suppression token may also contain `aether`** — so a suppressed aether call can never return. |
| LOUD-07 (docs stop claiming consolidation runs) | ✓ VERIFIED | Grepped all three named files this session: `CLAUDE.md:838` now reads "no lifecycle command invokes either one yet (Phase 162 wires this)"; `AGENTS.md:899` same; `structural-learning-stack.md:206,237` say "not wired to any lifecycle command yet". `TestDocsDoNotClaimConsolidationRunsToday` PASS pins it. |
| LOUD-08 (/ant-unblock exists) | ✓ VERIFIED | D-02 arm taken (wrapper built, guidance stays true): `.aether/commands/unblock.yaml`, `.claude/commands/ant/unblock.md`, `.opencode/commands/ant/unblock.md` all exist (verified by `ls`). `cmd/unblock_cmd.go:126` still says `/ant-unblock --dispatch` — now truthful. `TestSlashCommandGuidancePointsAtRealCommands` PASS. UAT executed `aether unblock --phase 1` with readable output and a loud, useful failure with no colony. |
| LOUD-09 (worker failure debug artifacts) | ✓ VERIFIED | All PASS this session: `TestWriteHostedWorkerOutputDebugRecordsEveryFailureMode` (subtests: timeout, non_zero_exit, provider_error_envelope, parse_failure — with exit code/duration/session id), `...EnforcesRetentionCapOnWrite` (cap), `...RedactsProviderOutput` + `...RedactsArgumentValues` (sanitization), `TestDataCleanWorkerDebug*` (4 tests: dry-run no-mutate, confirm prunes, file cap, missing dir OK), and D-05: `TestWorkerDebugArtifactsResolveToTrackingRoot` + `TestWorkerDebugCallSitesCoverEveryFailureModeAndUseTrackingRoot`. |
| RETIRE-01 (control-ts deleted, standalone first commit) | ✓ VERIFIED (ordering caveat) | `ls control-ts` → No such file or directory. Deletion commit `9077df26` touches **only** `control-ts/` paths (single parent, non-control-ts file list empty) — genuinely standalone. **Caveat:** it was not literally the milestone's first commit (its parent is "docs(160): mark plan 05 complete"; wave-2, plan 06). The ordering that carried the safety weight held: the replacement schema test (`90dd4605`, 160-01) landed before the deletion. `TestRetiredPackagesStayRetired/ControlTSIsGone` PASS pins the absence. |
| RETIRE-02 (.aether/ts-host kept) | ✓ VERIFIED | Directory exists; `go build ./cmd/aether` OK (the `//go:embed` at `embedded_assets.go:13` would break the build if it were gone); `TestRetiredPackagesStayRetired/TSHostAndTSAreKept` + `/EmbedDirectivesStillReferenceKeptTrees` PASS. No phase-160 commit (`grep "(160-"` across all refs) touched the tree; the ts-host diffs inside release squash #74 came from the pre-phase dispatch-fix session (`b2b41486`), recorded retroactively in the ledger. Working tree diff empty. |
| RETIRE-03 (.aether/ts kept) | ✓ VERIFIED | Directory exists; `go test ./cmd -run TestNarrator -count=1` → ok (narrator launcher tests still consume it); embed-directive subtest PASS. No phase-160 commit touched it. |
| RETIRE-04 (deleted-test ledger + schema replacement) | ✓ VERIFIED | `.aether/docs/retired-tests-ledger.md` exists with both required entries: `control-ts/tests/schemas/policy.schema.test.ts` → `recovered-by:cmd/policy_schema_test.go`, and the retroactive `.aether/ts-host/test/playbook-loader.test.ts` → `dead-with-no-replacement` with rationale. `TestPolicySchemaRequiredFields` PASS — and it is a scope improvement: it validates the **live** `colony/policies/*.yaml` files, where the retired TS test read fixture copies that could drift. Replacement landed before the deletion (commit order verified). |

**Score:** 13/13 requirements verified.

## Observable Truths (goal-backward)

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | No documented aether call violates its real argument contract | ✓ VERIFIED | 882 invocations, 0 violations, cobra's own machinery |
| 2 | The drift test sees positional drift, not just flags | ✓ VERIFIED | `TestAuditDetectsPositionalDrift` PASS + regression proof |
| 3 | A failed aether call in a live wrapper cannot vanish into /dev/null | ✓ VERIFIED | no-`aether`-on-suppression-line invariant PASS |
| 4 | A security gate that cannot run blocks rather than passes | ✓ VERIFIED | hard-block tests incl. absent-file UAT fix, all PASS |
| 5 | Worker failure evidence survives every failure mode and worktree removal | ✓ VERIFIED | 4 failure-mode subtests + tracking-root tests PASS |
| 6 | Docs no longer describe behavior that does not happen | ✓ VERIFIED | grep of 3 files + doc test PASS |
| 7 | control-ts is gone; ts-host and ts are alive and load-bearing | ✓ VERIFIED | ls + go build + embed/narrator tests PASS |

## Behavioral Spot-Checks (executed)

| Behavior | Command | Result | Status |
|---|---|---|---|
| Deleted command fails loudly | `./aether survey-load "test-phase"` | `Error: unknown command` | ✓ PASS |
| Scanner catches planted secret (positional) | `./aether check-antipattern <planted.go>` | `clean:false`, exposed-secret line 3 | ✓ PASS |
| Scanner catches planted secret (flag) | `./aether check-antipattern --file <planted.go>` | identical finding | ✓ PASS |
| Five LOUD-03 commands registered | `./aether <cmd> --help` × 5 | all resolve with real contracts | ✓ PASS |

## Anti-Pattern / Honesty Notes

| Item | Severity | Note |
|---|---|---|
| LOUD-05 stops short of `RunE` | ℹ️ Info | Deliberate, documented in the test file; full execution of 882 invocations would mutate state. Strongest safe check available. Recorded here so the limit stays visible. |
| RETIRE-01 "first commit" ordering | ℹ️ Info | Standalone yes; literally-first no (wave-2). Safety-net ordering (schema test before deletion) held, which is the constraint with teeth. |
| UAT provenance | ℹ️ Info | UAT was orchestrator-executed with captured output, not independent user testing — disclosed in 160-UAT.md itself. Its exploratory probing found and fixed a real gate bug (absent files counted as scanned), now pinned by a passing test. |
| UAT side effect | ℹ️ Info | A probe (`gate-results-write` without `--passed`) mutated COLONY_STATE.json during UAT; reverted and disclosed. Two out-of-scope findings (`--passed` defaulting, `--fixer-mode` unvalidated) recorded in UAT, not fixed — correctly out of scope. |
| REQUIREMENTS.md checkboxes | ⚠️ Warning (bookkeeping) | All 13 LOUD/RETIRE boxes in `.planning/REQUIREMENTS.md` remain unticked despite being verified here. The orchestrator should tick them citing this report. |

## Human Verification Required

None outstanding. The three manual-only items in 160-VALIDATION.md (terminal legibility of a failing call, gate outcome visibility, golden diff review) were exercised during the completed UAT session with captured output, and the regenerated goldens pass the full committed `cmd` suite. No item remains that a human must test before proceeding.

## Gaps Summary

No gaps. Every requirement has at least one command, run in this session, that fails when the requirement is unmet — the phase meets the project's Definition of Done by execution, not by summary claims.

---

_Verified: 2026-08-03_
_Verifier: Claude (gsd-verifier), goal-backward, execution-first_
