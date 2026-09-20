# Phase 193: Free Checks Are the Floor - Pattern Map

**Mapped:** 2026-08-22
**Files analyzed:** 7 (all modified, no new files)
**Analogs found:** 7 / 7 (self-referential — this phase edits existing functions in place; each file is its own best analog)

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/codex_continue.go` | orchestrator / gate-runner (in-process lane) | request-response, deterministic pipeline | itself — `runCodexContinueVerification` (~1560-1700), gate list (~3196-3290) | exact (edit in place) |
| `cmd/codex_continue_finalize.go` | orchestrator (wrapper/external lane) | request-response, deterministic pipeline | `cmd/codex_continue.go`'s gate + verification functions (the two lanes must mirror) | role-match, cross-file mirror |
| `cmd/criterion_evidence.go` | evaluator / evidence binder | CRUD-like evaluation over declarative requirements | itself — `syntheticCriterionRequirements` (~142), `evaluateCriterionCheck` (~600) | exact (edit in place) |
| `cmd/codex_build.go` | orchestrator (build-side dispatch planner) | event-driven (dispatch list construction) | itself — `queenBuildPostWaveDispatches` / watcher dispatch block (~1190-1224) | exact (edit in place) |
| `cmd/codex_build_finalize.go` | orchestrator (build finalize / commit) | request-response, report recording | itself — report/attempt recording functions (~1172-2062) | exact (edit in place) |
| `cmd/verify_out_of_band.go` | service (operator-invoked reconciliation) | request-response, fresh re-verify | itself — `runVerifyOutOfBand` + its re-verify machinery | exact (reuse, don't duplicate) |
| `cmd/build_attempt.go` | model / journal (append-only) | event-driven, append-only log | itself — `beginBuildAttempt` / `transitionBuildAttempt` (~118, ~180) | exact (edit in place) |

All target files already exist and already implement adjacent versions of the exact behavior this phase changes — there is no "new file, find an analog elsewhere" case here. The pattern work is: locate the precise existing block, understand its shape, and either delete/replace a branch or extend a struct/switch consistently with the surrounding style.

## Pattern Assignments

### `cmd/codex_continue.go` — verification pipeline + gate list (in-process lane)

**Analog:** itself, `runCodexContinueVerification` (lines 1560-1700)

**Core deterministic-step pattern** (lines 1566-1573):
```go
verificationTimeout = effectiveContinueVerificationTimeout(verificationTimeout)
commands := resolveCodexVerificationCommands(root)
requiredChecks := requiredVerificationChecks(phase)
steps := []codexVerificationStep{
    runVerificationStep(ctx, root, "build", requiredChecks["build"], commands.Build, verificationTimeout),
    runVerificationStep(ctx, root, "types", requiredChecks["types"], commands.Type, verificationTimeout),
    runVerificationStep(ctx, root, "lint", requiredChecks["lint"], commands.Lint, verificationTimeout),
    runVerificationStep(ctx, root, "tests", requiredChecks["tests"], commands.Test, verificationTimeout),
}
steps = applyExpectedTestFailure(steps, phase)
claims := verifyCodexBuildClaims(root, manifest)
```
`runVerificationStep` signature (line 2946): `func runVerificationStep(ctx context.Context, root, name string, required bool, command string, timeout time.Duration) codexVerificationStep` — this is the exact call shape D-07's scoped/targeted-vs-full run must still produce; a scoped runner just changes what `commands.Build/.Type/.Lint/.Test` resolve to before this loop runs, not the loop shape.

**The watcher-decision branch this phase must collapse** (lines 1596-1622): five branches — `skipWatchers` / `isEnvironmentBlockedWatcher` / host-boundary skip / provider-unavailable auto-skip / consecutive-failure auto-skip — each producing a `codexWatcherVerification{Present: true, Passed: true, Status: "skipped", ...}` literal. D-01/D-08 replace "zero executed checks hands verification to a Watcher" (the comment block at lines 1585-1595 names this explicitly) with "claims + criterion evidence only, no watcher fallback." Copy the literal shape of these skip-status structs when building whatever replaces them (e.g., a "no watcher needed, floor is deterministic" status), but the branch that spawns `runCodexContinueWatcherVerification` on the strength of "nothing else caught this" is the one D-08 removes as the implicit gate.

**Gate declaration pattern** (lines 3196-3247, `gateCheck` literals):
```go
verifCheck := gateCheck{
    Name:   "verification_steps_passed",
    Passed: verification.ChecksPassed,
    Detail: continueVerificationDetail(verification),
}
if !verification.ChecksPassed {
    verifCheck.FixHint = gateRecoveryTemplate("verification_loop")
    verifCheck.RecoveryOptions = []string{
        "Fix manually and run /ant-continue",
        "Run /ant-unblock for guided recovery",
    }
    blockers = append(blockers, verification.BlockingIssues...)
}
checks = append(checks, verifCheck)
```
Every new gate this phase needs (e.g. a `needs_owner_confirmation` surfacing gate for D-05, or a "fix attempt" retry-count gate for D-02/D-03) should follow this exact `gateCheck{Name, Passed, Detail}` + conditional `FixHint`/`RecoveryOptions` + `blockers = append(...)` shape — it's the load-bearing convention every gate in this function uses, including `manifest_present` (3196-3210) and `implementation_evidence` (3231-3247).

**The "always-pass gate is worse than no gate" precedent** (lines 3249-3257, comment on the removed `operational_evidence` gate) is directly cited in CONTEXT.md as the precedent for why `watcher` cannot remain a synthetic default check with no dispatched watcher behind it (D-06) — read this comment before touching `evaluateCriterionCheck("watcher")`.

---

### `cmd/criterion_evidence.go` — synthetic requirement defaults + check evaluation

**Analog:** itself, lines 142-160 and ~600-624

**Synthetic default checks (D-06 target)** (lines 142-160):
```go
func syntheticCriterionRequirements(criteria []string) []colony.CriterionEvidenceRequirement {
    requirements := make([]colony.CriterionEvidenceRequirement, 0, len(criteria))
    for _, criterion := range nonEmptyCriteria(criteria) {
        lower := strings.ToLower(criterion)
        checks := []string{"claims", "watcher"}   // <-- D-06 removes "watcher" here
        switch {
        case strings.Contains(lower, "test") || strings.Contains(lower, "coverage"):
            checks = append(checks, "tests")
        ...
        }
        requirements = append(requirements, colony.CriterionEvidenceRequirement{
            Criterion: criterion,
            Checks:    uniqueSortedStrings(checks),
        })
    }
    return requirements
}
```

**Check evaluation switch (`evaluateCriterionCheck`)** (~line 600):
```go
func evaluateCriterionCheck(check string, steps []codexVerificationStep, claims codexClaimVerification, watcher codexWatcherVerification) (bool, string, string) {
    check = strings.ToLower(strings.TrimSpace(check))
    switch check {
    case "claims":
        if claims.Present && claims.Passed && !claims.Skipped {
            return true, "current-build claims verified", ""
        }
        return false, "", "current-build claims were missing, skipped, or failed verification"
    case "watcher":
        if watcher.Present && watcher.Passed && !strings.EqualFold(strings.TrimSpace(watcher.Status), "skipped") {
            return true, fmt.Sprintf("watcher %s passed", firstNonEmpty(watcher.Worker, "verification")), ""
        }
        return false, "", "an executed Watcher review did not pass"
    default:
        for _, step := range steps {
            if strings.EqualFold(strings.TrimSpace(step.Name), check) {
                if step.Passed && !step.Skipped { ... }
```
D-06 requires `evaluateCriterionCheck("watcher")` to be satisfied by deterministic evidence when no watcher was dispatched, while still blocking when a dispatched watcher failed. This is a targeted change to the `case "watcher":` branch only — add a "no watcher was dispatched, floor already deterministic" success path, keep `watcher.Present && !watcher.Passed` as a hard fail. Follow the existing `(bool, string, string)` return convention exactly (passed, evidence text, blocking-issue text) — every caller (`criterion_evidence.go` ~496-505) treats a non-empty third value as the blocker message appended to `result.BlockingIssues`.

**Where `Deterministic` is tracked per-check** (lines 496-505):
```go
for _, check := range requirement.Checks {
    passed, evidence, issue := evaluateCriterionCheck(check, steps, claimsVerification, watcher)
    if passed {
        result.Evidence = append(result.Evidence, evidence)
        if check != "watcher" {
            evaluation.Deterministic = true
        }
        continue
    }
    result.BlockingIssues = append(result.BlockingIssues, issue)
}
```
This `if check != "watcher"` line (~line 501, cited directly in CONTEXT.md) is the flag that currently treats watcher-satisfied criteria as non-deterministic. Once "watcher" can pass via deterministic evidence with no dispatch, this line's semantics need revisiting alongside the switch case — they are two halves of the same D-06 change and must stay consistent (a criterion should not read `Deterministic: false` when it was, in fact, satisfied by the free checks alone).

---

### `cmd/codex_build.go` — build-side dispatch planner (D-08)

**Analog:** itself, lines ~1190-1224

**The watcher dispatch this phase removes/gates**:
```go
if queenCastes["watcher"] {
    dispatches = append(dispatches, codexBuildDispatch{
        Stage:         "verification",
        ExecutionWave: nextVerificationWave,
        Caste:         "watcher",
        Name:          deterministicAntName("watcher", fmt.Sprintf("phase:%d:watcher", phase.ID)),
        Task:          "Independent verification before advancement" + findingsInjectionForCaste("watcher"),
        Status:        "spawned",
    })
}
```
D-08 says the build-side "verification" stage dispatches no watcher — this block (or the `queenCastes["watcher"]` condition feeding it) is the literal removal target. The surrounding reviewer-wave pattern (probe/postWaveDispatches at lines ~1197-1207) is a decision made per-caste before this call and untouched by this phase (Phase 194's territory) — do not follow this dispatch's `codexBuildDispatch{...}` struct shape as a template for anything new in 193; it's the thing being deleted, not copied. If `TestWatcherIsAlwaysRequiredOnBuild` (in `cmd/queen_judgement_test.go`, alongside `TestQueenCannotDropTheWatcher`) breaks from this change, CONTEXT.md explicitly directs retiring it via the test-deletion ledger citing D11, in this phase.

---

### `cmd/codex_build_finalize.go` — recording the free-check report (D-08)

**Analog:** itself — no exact block yet exists for "record checks as report, not gate"; closest existing shape is the attempt-terminal recording pattern.

**Pattern to extend** (`recordBuildAttemptTerminal`, `cmd/build_attempt.go` line 614, called from `codex_build_finalize.go`): the build finalize path already calls into `build_attempt.go`'s terminal-recording functions after dispatch completion. This phase's job is to make sure whatever free-check results run at build time land in that same terminal record as a *report field*, not as something `codex_build_finalize.go` uses to block advancement — advancement stays `continue`'s job (D-08). Follow the existing terminal-record shape in `build_attempt.go` (see below) rather than inventing a parallel report file.

---

### `cmd/verify_out_of_band.go` — reuse target for D-04's re-verify machinery

**Analog:** itself — already does "fresh verification against current disk state, never fabricate worker receipts" (lines 15-35).

**Doc-comment framing to preserve verbatim in spirit** (lines 15-31):
```go
// This command re-verifies CURRENT disk state, fresh, against
// the phase's own success criteria, and only closes the stuck work when
// that fresh verification genuinely holds -- citing the verification, never
// a synthetic worker dispatch.
```
D-04 explicitly says "fall back to builder evidence re-run by the program... The program re-runs the verification command the builder reports in its handoff (`commands_run`) and confirms its `changed_files` exist." This is the same shape `verify_out_of_band.go` already implements for its own operator-invoked ceremony. CONTEXT.md's directive is "reuse its re-verify machinery," not duplicate it — the machinery to lift is whatever internal function `runVerifyOutOfBand` calls to re-run commands and re-hash artifacts (walk from `runVerifyOutOfBand` down; do not copy the cobra command shell, only the re-verify core). Keep the "never fabricate worker receipts" guard intact — do not let the new evidence-re-run path synthesize a `codexBuildDispatch` or watcher struct the way `codex_build.go`'s old fallback did.

---

### `cmd/build_attempt.go` — append-only attempt journal (D-03's fix attempt)

**Analog:** itself, `beginBuildAttempt` (line 118) and `transitionBuildAttempt` (line 180)

**Status constants to extend or reuse** (lines 18-27):
```go
const (
    buildAttemptPrepared    = "prepared"
    buildAttemptAwaiting    = "awaiting_external"
    buildAttemptDispatching = "dispatching"
    buildAttemptTerminal    = "terminal"
    buildAttemptBuilt       = "built"
    buildAttemptFailed      = "failed"
    buildAttemptInterrupted = "interrupted"
)
```

**Provenance-record pattern to copy exactly for the D-02/D-03 fix attempt** (lines 32-72, `outOfBandVerificationRecord`) — this struct is the direct precedent for "a build attempt carries a record of *why* it exists and what closed it, distinct from a normal worker dispatch." D-03's fix attempt needs an equivalent: a new attempt entry (never overwriting the first builder's result — "append-only" is stated explicitly in CONTEXT.md, spec §2.5/§2.6), carrying its own `reason` field ("fixing failed check: tests"), and counted separately. Model it as a sibling struct/field alongside `outOfBandVerificationRecord`, following the same doc-comment discipline (explain what distinguishes this record from a normal one, cite the named test that will assert append-only-ness).

**Transition function signature to follow** (line 180):
```go
func transitionBuildAttempt(attemptRel, status, summary string, dispatches []codexBuildDispatch, claims *codexBuildClaims, dispatchMode string, transitionErr error) error {
```
The fix-attempt dispatch is "a new attempt in the attempt journal" per D-03 — it should go through `beginBuildAttempt`/`transitionBuildAttempt` like any other attempt, with `dispatches` containing the one builder fix-dispatch and a `summary`/reason string identifying it as a check-driven fix, not a fresh phase attempt.

---

## Shared Patterns

### `gateCheck` struct — every gate in this phase's gate list
**Source:** `cmd/codex_continue.go` lines 3196-3290 (whole gate-building function)
**Apply to:** any new/changed gate in `codex_continue.go` and its mirror in `codex_continue_finalize.go`
```go
check := gateCheck{Name: "<gate_name>", Passed: <bool>, Detail: "<human summary>"}
if !check.Passed {
    check.FixHint = "<what to do>"
    check.RecoveryOptions = []string{
        "Fix manually and run /ant-continue",
        "Run /ant-unblock for guided recovery",
    }
    blockers = append(blockers, check.Detail)
}
checks = append(checks, check)
```

### Two-lane mirroring (in-process vs wrapper)
**Source:** `cmd/codex_continue.go` (in-process) vs `cmd/codex_continue_finalize.go` (wrapper/external lane)
**Apply to:** every rule this phase adds — CONTEXT.md states directly: "every rule above must hold on both lanes (the 2026-08-21 review gate found the in-process lane half-wired)." Any change to gate logic, watcher-skip logic, or criterion evaluation in `codex_continue.go` needs its counterpart checked/updated in `codex_continue_finalize.go`. There is no shared helper enforcing this — it is a manual-parity discipline, the same one that produced the original half-wired defect.

### Append-only, never-overwrite provenance records
**Source:** `cmd/build_attempt.go` `outOfBandVerificationRecord` (lines 32-72) and the attempt journal functions
**Apply to:** the D-03 fix-attempt record — new entries only, cite the source record's own doc comment discipline (explain distinguishing fields, name the future test).

### "An always-pass check is worse than no check"
**Source:** `cmd/codex_continue.go` lines 3249-3257 (comment on the deleted `operational_evidence` gate)
**Apply to:** D-06 (`watcher` check must not silently pass with nothing behind it), and any new "floor" gate this phase adds — every gate must be able to fail.

## No Analog Found

None — all seven target files already contain the exact functions this phase modifies. There is no case here of building a genuinely new file type; the planner should treat every plan item as "locate function X in file Y, read its current branch/switch, replace or extend it consistent with the excerpts above."

## Metadata

**Analog search scope:** `cmd/` (all seven files named in CONTEXT.md's "Code this phase changes"), cross-referenced against `cmd/queen_judgement_test.go` for the named-test convention.
**Files scanned:** `cmd/codex_continue.go`, `cmd/criterion_evidence.go`, `cmd/codex_build.go`, `cmd/codex_build_finalize.go`, `cmd/verify_out_of_band.go`, `cmd/codex_continue_finalize.go`, `cmd/build_attempt.go`, `cmd/queen_judgement_test.go`
**Pattern extraction date:** 2026-08-22
