---
phase: 204-learning-governor
plan: 12
subsystem: learning-governor
tags: [shadow-comparison, canary-promotion, cobra-cli, go]

# Dependency graph
requires:
  - phase: 204-learning-governor
    provides: "pkg/shadow (204-08), cmd/promotion_gate.go + cmd/rollback.go (204-09), cmd/improvement_report.go (204-10), and 204-14's guard-count changes to cmd/testdata/fixture-bank/v1/bank.json (read at run time, never snapshotted)"
provides:
  - "A real per-fixture classifier (shadowClassifyAgainstBank) behind shadowEvaluator's FrozenEvaluator.Run seam, replacing the always-pass placeholder -- closes SC4b"
  - "runAutomaticImprovementPass, called once from runPhaseEndConsolidation, driving a declared candidate through comparison, gate admission, canary start, and (once its bound is reached) completion/rollback -- closes half of SC5b (the production caller)"
  - "One plain-English closing-card line (renderImprovementPassBeat) naming what the automatic pass did, or nothing at all when it did nothing"
  - "aether improve (inspection by default; --declare/--compare for the mutating forms), plus registration of aether shadow-declare and aether shadow-compare (WINDOWS.md entries 39-40, now fixed)"
  - "/ant-improve wrapper triplet (.aether/commands/improve.yaml, .claude/commands/ant/improve.md, .opencode/commands/ant/improve.md)"
affects: [204-VERIFICATION.md SC4b, SC5b]

# Actuals (#2632)
actuals:
  tokens: 19104
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Run-time fixture resolution (shadowGraderResolveExampleFixtures): tests resolve a guarded and an unguarded example fixture from the real committed bank at run time rather than by ID, so 204-14's guard-count edits (17 guarded / 30 unguarded of 47, as observed when this plan ran) can never invalidate them."
    - "One card-worthy event per candidate outcome (D-03): the automatic pass's own Compared/Admitted counters record internal steps, but only a terminal outcome (refused, started, completed, rolled_back) becomes a rendered closing-card line -- never three lines for one candidate's one pass through the pipeline."

key-files:
  created:
    - cmd/improvement_pass.go
    - cmd/improvement_pass_test.go
    - cmd/shadow_grader_test.go
    - cmd/improvement_cmds.go
    - .aether/commands/improve.yaml
    - .claude/commands/ant/improve.md
    - .opencode/commands/ant/improve.md
  modified:
    - cmd/shadow_cmds.go
    - cmd/consolidation_lifecycle.go
    - cmd/codex_visuals.go
    - cmd/command_guide.go
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/command_catalog.json

key-decisions:
  - "The word-overlap classifier (shadowFixtureSubjectWords/shadowTextNamesFixtureSubject) is a simple, run-time-derived significant-word match against a fixture's own Title+Invariant text, with a small generic stop-word list -- never a per-fixture table. This satisfies the plan's ban on hard-coding fixture IDs or guard counts while still producing a real, differentiated classifier."
  - "The automatic pass emits exactly ONE card-worthy event per candidate per pass (refused, started, completed, or rolled_back) -- comparison and admission are recorded only in the summary's own Compared/Admitted counters, never as their own closing-card line. This resolves an internal tension between the plan's D-03 wording (\"one line per event\") and its Test 6 wording (\"renders ONE ... closing-card line\") in favour of the latter, which is the literal acceptance check."
  - "improvementPassScopePaths hard-codes .aether/data/instincts.json for the project-knowledge scope and .aether/data/COLONY_STATE.json for the routing scope -- the only two files startCanary's checkpoint may ever cover for an admitted candidate, chosen because an EMPTY scope-paths list checkpoints the entire working tree (saveRepairCheckpoint's own documented behaviour), which is exactly the unbounded blast radius the two-scope boundary exists to prevent. No test in this plan's own acceptance criteria pins these two paths; a future plan could reasonably widen or specialise them further."
  - "improvementPassPhaseGatesPassed reads the phase's own gate outcome from its latest durable build attempt's free-check report (cmd/build_attempt.go), defaulting to \"passed\" when no evidence exists -- because runAutomaticImprovementPass's one production call site (runPhaseEndConsolidation) is itself only ever reached after the current phase's own checks have already passed. The rollback branch this feeds is therefore real and tested via the pure helper, but not exercised end-to-end through the production call site in this plan (no acceptance criterion required it)."
  - "cmd/shadow_cmds.go's stale 'NOT registered yet' comment above its own init() was deliberately left unedited, per the plan's explicit instruction not to touch that file's init() or its comment block unless gofmt/vet required it."

requirements-completed: []
# LEARN-06 and LEARN-07 are both declared by multiple sibling plans in this
# phase (204-08/204-09 among them). `requirements.ready-ids` reported both
# BLOCKED (0/2 ready) because not every declaring plan has produced a
# SUMMARY.md yet -- correctly deferred to whichever plan finishes last.

coverage:
  - id: D1
    description: "shadowEvaluator's run function is a real per-fixture classifier (shadowClassifyAgainstBank) distinguishing a beneficial candidate from a harmful/overfit one, reachable through the real shadowEvaluator() production entrypoint -- closes SC4b"
    requirement: LEARN-06
    verification:
      - kind: unit
        ref: "cmd/shadow_grader_test.go#TestShadowGraderDistinguishesBeneficialFromHarmful"
        status: pass
      - kind: unit
        ref: "cmd/shadow_grader_test.go#TestOverfitCandidateIsRefusedAtTheGate"
        status: pass
      - kind: unit
        ref: "cmd/shadow_grader_test.go#TestEvaluatorDigestIsStableAndChanged"
        status: pass
    human_judgment: false
  - id: D2
    description: "runAutomaticImprovementPass drives a declared, beneficial, promotable candidate through comparison, gate admission and canary start, called from a single production site reached by both check lanes -- closes the production-caller half of SC5b"
    requirement: LEARN-07
    verification:
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestAutomaticImprovementPassTracerEndToEnd"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestAutomaticImprovementPassIsReachedFromBothCheckLanes"
        status: pass
    human_judgment: false
  - id: D3
    description: "The automatic pass never promotes outside the two canary-promotable scopes, carries no actor/bypass parameter, and refuses an unrecognized scope -- retained-authority boundary structurally preserved"
    requirement: LEARN-07
    verification:
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestAutomaticPassNeverPromotesOutsideTheTwoScopes"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestAutomaticPassCarriesNoBypassParameter"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestUnrecognizedScopeIsRefusedAndWritesNothing"
        status: pass
    human_judgment: false
  - id: D4
    description: "The closing card names what the automatic pass did in plain English, or nothing at all when it did nothing; a comparison/admission/canary failure never blocks the check; a colony with no declared candidate costs nothing"
    requirement: LEARN-07
    verification:
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestImprovementPassClosingLineIsPlainEnglish"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestCheckWithNoDeclaredCandidateCostsNothing"
        status: pass
      - kind: unit
        ref: "cmd/improvement_pass_test.go#TestImprovementPassFailureNeverBlocksTheCheck"
        status: pass
    human_judgment: false
  - id: D5
    description: "aether improve (inspection, --declare, --compare), aether shadow-declare and aether shadow-compare are registered, reachable in the real binary, and improve's default form mutates nothing on disk"
    requirement: LEARN-06
    verification:
      - kind: unit
        ref: "cmd/parity_test.go#TestPlatformParityGolden"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestNoRegisteredSubcommandIsUnreferenced"
        status: pass
      - kind: manual_procedural
        ref: "go run ./cmd/aether shadow-declare --help / shadow-compare --help / improve --help (each exits 0); checksum of a scratch colony's .aether/data before and after `aether improve` is identical"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-14
status: complete
---

# Phase 204 Plan 12: Real shadow grader and the automatic canary pass Summary

**Replaced shadowEvaluator's always-pass placeholder with a real per-fixture classifier and wired one automatic pass (runAutomaticImprovementPass) that drives a declared candidate through comparison, gate admission, canary start and completion/rollback from both check lanes, closing SC4b and the production-caller half of SC5b.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-14T21:54:00Z (approximate)
- **Completed:** 2026-09-14T22:49:00Z
- **Tasks:** 3
- **Files modified:** 13 (7 created, 6 modified)

## Accomplishments

- `shadowClassifyAgainstBank` is a real per-fixture classifier: the baseline subject passes a fixture exactly when it carries a guard; a non-baseline subject fails a fixture its own harms name, passes a guarded fixture otherwise, and passes an unguarded fixture only when its own scope+expected-benefit text names that fixture's subject -- all resolved from the fixture's own Title/Invariant text at run time, never a hand-typed per-fixture table.
- `runAutomaticImprovementPass`, called once from `runPhaseEndConsolidation`, drives a declared candidate through `runShadowCompare` → `admitCandidateToCanary` → `startCanary`, and drives a running canary through `completeCanary`/`rollbackCanary` once its bound is reached -- reachable from both `runCodexContinue` and `runCodexContinueFinalize` through that one call site, proven by an AST call-graph guard in the same shape as the credit-ledger's own precedent.
- `renderImprovementPassBeat` renders one plain-English line per card-worthy candidate outcome (refused / started / completed / rolled back), reading `result["improvement_pass"]` dual-typed against both the in-process struct and the JSON-round-tripped snake_case map `attachConsolidationSummary` now also populates -- and renders nothing at all when the pass had nothing to do.
- `aether improve` (inspection by default, `--declare`/`--compare` for the mutating forms) is now a registered, reachable command, along with `aether shadow-declare` and `aether shadow-compare` -- WINDOWS.md entries 39 and 40, both now marked fixed.
- The `/ant-improve` wrapper triplet exists on Claude and OpenCode, byte-identical, and its routing text is what credits all three runtime commands with real caller evidence in the orphan-reachability scan.

## Task Commits

Each task was committed atomically:

1. **Task 1: A declared candidate travels comparison → gate → canary → closing card, automatically, on one check lane** - `3154e5ff` (feat)
2. **Task 2: Prove both check lanes reach the pass, and that it can never widen what may change** - `14bfe8ba` (test)
3. **Task 3: The hand-run improvement surface — a registered command and its wrapper triplet** - `e0ab4f42` (feat)

_Note: Task 1 was declared `type="tracer"`; its own `<verify>` passed before commit and was re-confirmed after commit (auto mode active in this worktree-parallel dispatch context), satisfying the tracer feedback gate before Task 2 began._

## Files Created/Modified

- `cmd/shadow_cmds.go` - `shadowEvaluator` now built from a real classifier; prefix bumped to `shadow-evaluator-v2`
- `cmd/shadow_grader_test.go` - grader behavior tests (beneficial/not-beneficial/overfit/unresolvable, digest stability)
- `cmd/improvement_pass.go` - `runAutomaticImprovementPass`, `improvementPassSummary`, scope-path derivation, gate-outcome read
- `cmd/improvement_pass_test.go` - tracer, closing-line, zero-state, non-blocking, both-lanes-reachable, never-outside-two-scopes, no-bypass-parameter, unrecognized-scope tests
- `cmd/consolidation_lifecycle.go` - `phaseEndConsolidationSummary.ImprovementPass` field; one call to `runAutomaticImprovementPass` at the end of `runPhaseEndConsolidation`; `attachConsolidationSummary` extended with a snake_case `improvement_pass` map
- `cmd/codex_visuals.go` - `renderImprovementPassBeat` and its dual-type event-view reduction, called from `renderContinueVisual`
- `cmd/improvement_cmds.go` - `improveCmd` (inspection/declare/compare) and registration of `improveCmd`/`shadowDeclareCmd`/`shadowCompareCmd`
- `cmd/command_guide.go` - `improve` added to the literal-command list
- `.aether/commands/improve.yaml`, `.claude/commands/ant/improve.md`, `.opencode/commands/ant/improve.md` - the wrapper triplet
- `cmd/testdata/parity_snapshot.json`, `cmd/testdata/command_catalog.json` - updated with exactly the three new commands' real scanner output (hand-inserted after confirming byte-for-byte match against `-update-golden`'s own output, to avoid absorbing unrelated pre-existing drift already present in these golden files)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Grader tests initially built a synthetic evaluator instead of exercising the real production entrypoint**
- **Found during:** Task 1, while performing the FAILS-WHEN-UNGRADED mutation check
- **Issue:** The first draft of `TestShadowGraderDistinguishesBeneficialFromHarmful` and `TestOverfitCandidateIsRefusedAtTheGate` built their own `shadow.NewFrozenEvaluator(...)` wrapping a small synthetic bank, rather than calling `shadowEvaluator()` itself. Restoring `shadowEvaluator`'s run function to the always-pass placeholder therefore did NOT make these tests fail, defeating the plan's own acceptance criterion.
- **Fix:** Rewrote both tests to construct their evaluator via `shadowEvaluator()` (the real production entrypoint, over the real committed bank), while still resolving example fixtures from the real bank at run time. Re-ran the mutation check afterward and confirmed it now fails as required.
- **Files modified:** `cmd/shadow_grader_test.go`
- **Verification:** FAILS-WHEN-UNGRADED mutation re-run and confirmed non-zero (see Verification section below)
- **Committed in:** `3154e5ff` (Task 1 commit)

**2. [Rule 1 - Bug] Dual-type event renderer initially only handled the JSON-round-tripped map shape**
- **Found during:** Task 1, writing `TestImprovementPassClosingLineIsPlainEnglish`'s map-shape subtest
- **Issue:** `improvementPassEventViewsFromRaw`'s `map[string]interface{}` case only handled `events` as `[]interface{}` (the shape produced after an actual JSON marshal/unmarshal round-trip). `attachConsolidationSummary` builds the map directly as `[]map[string]interface{}` before any serialization, so the renderer silently produced zero events for the in-process map path.
- **Fix:** Extended the map case to handle both `[]interface{}` and `[]map[string]interface{}`.
- **Files modified:** `cmd/codex_visuals.go`
- **Verification:** `TestImprovementPassClosingLineIsPlainEnglish` (both subtests) passes
- **Committed in:** `3154e5ff` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs found while proving the plan's own mutation/acceptance checks, not scope changes).
**Impact on plan:** Both fixes were necessary for the plan's own acceptance criteria to be genuinely meaningful rather than passing vacuously. No scope creep.

## FAILS-WHEN-UNWIRED / FAILS-WHEN-UNGRADED / FAILS-WHEN-UNREGISTERED (performed, observed, reverted)

**Task 1 — FAILS-WHEN-UNWIRED:** commented out `summary.ImprovementPass = runAutomaticImprovementPass(phaseID)` in `runPhaseEndConsolidation`. Observed:
```
improvement_pass_test.go:81: expected the improvement pass Ran == true, got {Ran:false CandidatesConsidered:0 Compared:0 Admitted:0 Refused:0 CanariesStarted:0 CanariesCompleted:0 CanariesRolledBack:0 Events:[] Failures:[]}
--- FAIL: TestAutomaticImprovementPassTracerEndToEnd (0.01s)
```
Reverted; test passes again.

**Task 1 — FAILS-WHEN-UNGRADED:** restored `shadowEvaluator`'s run function to `shadow.NewResult(true)` unconditionally. Observed:
```
shadow_grader_test.go:116: expected baseline to FAIL an unguarded fixture "fixture-03f8df082186"
shadow_grader_test.go:145: expected candidate to FAIL guarded fixture "fixture-0b946259e159" because its harms name that fixture's own subject
shadow_grader_test.go:167: expected an unresolvable task to fail for the candidate
--- FAIL: TestShadowGraderDistinguishesBeneficialFromHarmful (0.00s)
```
Reverted; test passes again.

**Task 2 — FAILS-WHEN-UNWIRED (repeat, naming both lanes):** same call site commented out. Observed:
```
improvement_pass_test.go:303: runAutomaticImprovementPass is not transitively reachable from: [runCodexContinue runCodexContinueFinalize] -- both check lanes must reach the automatic improvement pass
--- FAIL: TestAutomaticImprovementPassIsReachedFromBothCheckLanes (0.26s)
```
Reverted; test passes again.

**Task 3 — FAILS-WHEN-UNREGISTERED:** commented out `rootCmd.AddCommand(shadowDeclareCmd)`. Observed:
```
$ go run ./cmd/aether shadow-declare --help
Error: unknown command "shadow-declare" for "aether"
exit status 1
```
Reverted; `go run ./cmd/aether shadow-declare --help` exits 0 again.

## Bank-Independence and Guard-Tally Checks

- Parsed `cmd/testdata/fixture-bank/v1/bank.json`'s 47 fixture IDs and confirmed by direct string search that none of them appears anywhere in `cmd/shadow_grader_test.go` or `cmd/shadow_cmds.go`.
- Confirmed by reading both files that neither contains a numeric literal describing how many bank fixtures are guarded or unguarded; the run-time resolution helper used instead is `shadowGraderResolveExampleFixtures` (test-side) and the classifier's own `shadowFixtureByID`/`fixture.Guard != nil` checks (production-side).
- The bank actually carried **17 guarded / 30 unguarded fixtures (47 total)** when this plan ran — an observation of 204-14's own result, never a figure this plan's code or tests depend on.
- `go test ./cmd -run '^TestShadowGraderDistinguishesBeneficialFromHarmful$' -count=1 -v` shows the fixture-honesty subtest ("fixture_honesty: at least one guarded and one unguarded fixture exist") as PASS.

## Allowlist Ratchet

- `cmd/testdata/orphan_allowlist.json` was **not modified** — its 255 entries were already sufficient because the `/ant-improve` wrapper's routing text (in `.aether/commands/improve.yaml`) names all three runtime commands (`aether improve`, `aether shadow-declare`, `aether shadow-compare`) in backtick-wrapped invocations, which `collectCallerEvidence`'s YAML scan credits directly. Before and after this task: 255 entries, none of the three new command names among them (confirmed by parsing the JSON and comparing entry names, not grepping the file).

## Issues Encountered

None beyond the two auto-fixed deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SC4b is closed: `shadowEvaluator` is a real, differentiated classifier reachable through its own production entrypoint, proven to fail when reverted.
- SC5b's production-caller half is closed: `admitCandidateToCanary` and `startCanary` are reached automatically from both check lanes; `completeCanary`/`rollbackCanary` are reached from the same pass's running-canary loop (exercised directly via `improvementPassCanaryBoundExceeded`/`processRunningImprovementCanary`, though not end-to-end through the production call site in this plan, since the one production call site is only ever reached after the current phase's own gates have already passed — see key-decisions).
- WINDOWS.md entries 39 and 40 are both marked `fixed`.
- LEARN-06 and LEARN-07 remain unchecked in REQUIREMENTS.md by design: `requirements.ready-ids` reported both blocked (0/2 ready) because sibling plans in this phase also declare them and have not all produced a SUMMARY.md yet. The last plan to finish should re-run the mark-complete check.
- Not addressed by this plan (correctly out of scope): SC5c (downstream of SC3a's episode-field wiring) and SC5d (source-proposal caller) remain open per 204-VERIFICATION.md.

## Self-Check: PASSED

- `cmd/improvement_pass.go` — FOUND
- `cmd/improvement_pass_test.go` — FOUND
- `cmd/shadow_grader_test.go` — FOUND
- `cmd/improvement_cmds.go` — FOUND
- `.aether/commands/improve.yaml` — FOUND
- `.claude/commands/ant/improve.md` — FOUND
- `.opencode/commands/ant/improve.md` — FOUND
- Commit `3154e5ff` — FOUND in `git log --oneline --all`
- Commit `14bfe8ba` — FOUND in `git log --oneline --all`
- Commit `e0ab4f42` — FOUND in `git log --oneline --all`

---
*Phase: 204-learning-governor*
*Completed: 2026-09-14*
