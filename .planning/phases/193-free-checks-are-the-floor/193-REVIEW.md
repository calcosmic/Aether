---
phase: 193-free-checks-are-the-floor
reviewed: 2026-08-22T00:00:00Z
depth: standard
files_reviewed: 33
files_reviewed_list:
  - cmd/blackbox_harness_test.go
  - cmd/blackbox_zero_reviewer_test.go
  - cmd/build_attempt.go
  - cmd/check_fix_attempt.go
  - cmd/codex_build_finalize_test.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_test.go
  - cmd/codex_build.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_continue_plan.go
  - cmd/codex_continue_test.go
  - cmd/codex_continue.go
  - cmd/codex_visuals_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/continue_criterion_evidence_finalize_test.go
  - cmd/continue_daily_driver_test.go
  - cmd/criterion_evidence.go
  - cmd/criterion_owner_confirmation.go
  - cmd/deterministic_floor_test.go
  - cmd/deterministic_floor.go
  - cmd/floor_fix_attempt_test.go
  - cmd/floor_reviewer_free_gate_test.go
  - cmd/floor_unskippable_test.go
  - cmd/phase_verified_once_test.go
  - cmd/queen_judgement_test.go
  - cmd/queen_orchestration_regression_test.go
  - cmd/review_depth_test.go
  - cmd/seal_ceremony_test.go
  - cmd/seal_final_review.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/golden_build.txt
  - cmd/verification_scope_test.go
  - cmd/verification_scope.go
findings:
  critical: 2
  warning: 2
  info: 1
  total: 5
status: issues_found
---

# Phase 193: Code Review Report

**Reviewed:** 2026-08-22T00:00:00Z
**Depth:** standard
**Files Reviewed:** 33
**Status:** issues_found

## Summary

Phase 193 rewires `aether continue` so the program's own free checks (build,
types, lint, tests, claimed-files-exist, and per-criterion evidence) are the
only thing that can make a phase pass, while a dispatched reviewer can still
add a block on top. I traced this rule through both continue lanes
(`runCodexContinueVerification` in `cmd/codex_continue.go` and
`runCodexContinueVerificationSnapshot` in `cmd/codex_continue_plan.go`, both
now sharing `runDeterministicFloor` in `cmd/deterministic_floor.go`) and
confirmed the core invariant holds: `checksPassed` is seeded from the floor
alone, and the only place a reviewer verdict touches it afterward is a branch
that can set it from `true` to `false`, never the reverse. The single bounded
automatic fix attempt (`cmd/check_fix_attempt.go`) is correctly capped at one
per phase/check, dispatches only a builder, and re-runs the floor exactly
once. The "reconcile is not a bypass" change to
`continueTasksSupportAdvancement` is correctly gated on the phase-wide
deterministic floor (`task.Verified == verification.ChecksPassed`), not on a
per-task shortcut.

However, this phase also introduces two new command-execution surfaces that
take attacker- or LLM-influenced free text and feed it into a real shell,
without sanitization: `reRunBuilderReportedEvidence` now actually executes a
builder's self-reported `commands_run` strings (previously this data was
inert, display-only text), and `ownerConfirmationCommand` builds a "copy this
exact command" string for the (explicitly non-technical, per this repo's own
CLAUDE.md) project owner by interpolating a phase's success-criterion text
into a double-quoted shell command with Go's `%q`, which does not neutralize
shell metacharacters like `$(...)`, backticks, or `$VAR`. Both are new files/
new code paths added by this phase, not pre-existing behavior. I also found
two lower-severity issues: a manifest-forwarding bug that silently makes the
fix-attempt's task-implication logic read the wrong claims file, and a raw
snake_case status token leaking into otherwise plain-English gate text.

## Critical Issues

### CR-01: Builder-reported shell commands are executed with no sanitization or allowlist

**File:** `cmd/criterion_evidence.go:908-934` (via `cmd/codex_continue.go:4040-4052`)

**Issue:** `reRunBuilderReportedEvidence` (new in this phase) reads every
persisted worker handoff for the current phase and, for each string in
`record.CommandsRun` (`pkg/codex.WorkerHandoff.CommandsRun`, populated
entirely from a builder worker's own self-reported completion packet), calls:

```go
step := runVerificationStep(ctx, root, "builder-reported", false, command, timeout)
```

which bottoms out in `runShellCommandContext` (`cmd/codex_continue.go:4040`):

```go
cmd = exec.CommandContext(ctx, "sh", "-c", command)
```

Before this phase, `CommandsRun` was purely informational — every other
caller (`appendHandoffList(&b, "Commands run", record.CommandsRun)` in
`cmd/codex_dispatch_contract.go:823`) only renders it as markdown text. Phase
193 is the first place in the codebase that actually shells out to a string a
worker wrote into its own handoff, and it runs automatically, with no operator
confirmation, whenever a phase's "claims" evidence check fails and no
reviewer was dispatched — which is the normal, default path for most phases
under this design (`evaluatePhaseCriterionEvidence`, lines 431-546).

A builder worker (an LLM agent operating on task content that may itself
include content from files, URLs, issue text, or other data the agent read
during the phase) can populate `CommandsRun` with an arbitrary string. Because
it is later replayed through `sh -c` on the machine running `aether continue`
— which, per this repo's own model, is the *owner's own machine*, not a
sandboxed CI runner — a compromised or prompt-injected builder handoff
becomes a route to arbitrary code execution the moment `/ant-continue` is
next run for that phase, with no allowlist, no confirmation prompt, and no
restriction to the kind of build/test commands this mechanism is meant to
verify.

**Fix:** Do not execute arbitrary reported strings. Either (a) restrict
`reRunBuilderReportedEvidence` to re-running only commands that exactly match
(or are a strict, safelisted subset of) the project's own already-resolved
verification commands (`resolveCodexVerificationCommands`) rather than
anything a worker chooses to report, or (b) require an explicit
`--allow-reported-command-rerun` opt-in (with the raw command shown to the
operator before it runs) so this new execution surface is never silently on
by default:

```go
func reRunBuilderReportedEvidence(ctx context.Context, root string, phase colony.Phase, timeout time.Duration) builderEvidenceResult {
    resolved := resolveCodexVerificationCommands(root)
    allowed := map[string]bool{
        strings.TrimSpace(resolved.Build): true,
        strings.TrimSpace(resolved.Type):  true,
        strings.TrimSpace(resolved.Lint):  true,
        strings.TrimSpace(resolved.Test):  true,
    }
    // ... only re-run `command` when allowed[command] is true; otherwise
    // record it as Unresolvable rather than executing it.
}
```

### CR-02: Shell-metacharacter injection in the "exact command" shown to the (non-technical) owner

**File:** `cmd/criterion_owner_confirmation.go:25-50`

**Issue:** `ownerConfirmationCommand` builds the literal string the owner is
told to copy and run to resolve an unprovable criterion:

```go
func ownerConfirmationCommand(phaseID int, taskID, criterion string) string {
	return fmt.Sprintf("aether decision-answer --question %q --answer \"confirmed\" --phase %d",
		ownerConfirmationQuestionText(phaseID, taskID, criterion), phaseID)
}
```

`criterion` is a phase or task's success-criterion text — free-form English
authored by the planning LLM, which may itself have been influenced by
external content (a linked issue, a spec document, scraped text) during
`/ant-plan`. `%q` produces a **Go-syntax** double-quoted string, not a
shell-safe one: it escapes `"` and `\`, but leaves `$`, backticks, and other
shell metacharacters untouched. This string is then embedded inside the
surrounding `"..."` of the generated shell command, and surfaced verbatim as:

- `RecoveryOptions` on the `owner_confirmation_pending` gate
  (`cmd/codex_continue.go:354`)
- the seal blocker `Description`
  (`cmd/criterion_owner_confirmation.go:110`)

Per this repo's own CLAUDE.md, the person reading this text is explicitly
non-technical and is expected to copy-paste the shown command into a
terminal. If a criterion's text contains `$(...)`, a backtick pair, or even a
plain `$SOMEVAR`, the resulting "exact command to run" either (a) silently
corrupts the recorded confirmation (variable expansion changes what
`--question` actually receives, so `ownerConfirmationAnswered`'s exact-text
match then never resolves the criterion), or (b), for `$(...)`/backticks,
executes attacker-chosen shell code on the owner's machine the moment they
run the suggested command.

**Fix:** Never build a "commands to paste into a shell" string by
interpolating untrusted text with `%q` (Go quoting) — quote for the shell
instead, or better, avoid embedding the criterion text in the command at all
(reference it by a stable ID instead, and print the human-readable criterion
text separately, outside the code block the owner is told to run):

```go
func ownerConfirmationCommand(phaseID int, taskID, criterion string) string {
	id := ownerConfirmationID(phaseID, taskID, criterion) // stable, opaque, shell-safe
	return fmt.Sprintf("aether decision-answer --confirmation-id %s --answer confirmed", id)
}
```

If the question text must stay in the command, shell-quote it explicitly
(wrap in single quotes and escape embedded single quotes as `'\''`) rather
than relying on Go's `%q`.

## Warnings

### WR-01: `planCheckFixAttempt` reads claims from the wrong (default) path, ignoring the caller's real manifest

**File:** `cmd/check_fix_attempt.go:179-205` (specifically line 195)

**Issue:** `applyAutomaticCheckFixAttempt` (`cmd/check_fix_attempt.go:247`)
receives the real `manifest codexContinueManifest` for this continue run and
uses it correctly when re-running the floor (`runDeterministicFloor(ctx,
root, phase, manifest, ...)`, line 280). But `planCheckFixAttempt`, which
builds the failure index used to compute `ImplicatedTaskIDs` for the fix
builder's brief and for the eventual `check_fix_attempt` gate's recovery
command, discards that manifest entirely:

```go
func planCheckFixAttempt(state colony.ColonyState, phase colony.Phase, floor deterministicFloorResult, reviewerDispatched bool) (checkFixAttemptRecord, bool) {
	...
	claims := loadRawBuildClaimsForScope(codexContinueManifest{})
	index := buildCheckFailureIndex(*failing, claims, phase)
	...
}
```

`loadRawBuildClaimsForScope` (`cmd/verification_scope.go:113-126`) only
honors a non-default `ClaimsPath` when `manifest.Present` is true; passing a
zero-value `codexContinueManifest{}` always forces the default
`last-build-claims.json`, even when this continue run's real manifest names a
different claims file (as the external/wrapper lane's completion packets can).
The result is silent: on a build where `ClaimsPath` differs from the default,
`implicatedTaskIDsFromExcerpts` will read stale or missing claims, producing
an empty or wrong `ImplicatedTaskIDs`, which in turn makes
`buildTargetedRedispatchCommand` fall back to a full-phase
`buildForceRedispatchCommand` instead of the precise per-task command the
`check_fix_attempt` gate is documented to offer. No test in
`floor_fix_attempt_test.go` exercises a non-default `ClaimsPath`, so this gap
is untested as well as unused in the current call.

**Fix:** Thread the real manifest through:

```go
func planCheckFixAttempt(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, floor deterministicFloorResult, reviewerDispatched bool) (checkFixAttemptRecord, bool) {
	...
	claims := loadRawBuildClaimsForScope(manifest)
	...
}
```
and update the one call site in `applyAutomaticCheckFixAttempt` to pass its
own `manifest` parameter through.

### WR-02: Raw snake_case status token leaks into user-facing gate text

**File:** `cmd/codex_continue.go:3317`

**Issue:** The `check_fix_attempt` gate's detail line is built as:

```go
Detail: fmt.Sprintf("one automatic fix attempt ran for the %s check (%s)", fix.Check, fix.Outcome),
```

`fix.Outcome` is one of the two internal enum values set in
`cmd/check_fix_attempt.go:281-284` (`"fixed"` or `"still_failing"`). This
produces reader-facing text such as "one automatic fix attempt ran for the
tests check (still_failing)". CLAUDE.md's communication rule (and its
"READ THIS BEFORE YOU WRITE ANYTHING TO THE OWNER" banner) requires every
piece of text this system writes to the project owner to be plain English
with no internal code-style tokens; `still_failing` with an underscore, shown
mid-sentence in parentheses, is exactly the kind of leak that rule exists to
catch.

**Fix:**

```go
outcomeText := "fixed it"
if fix.Outcome == "still_failing" {
	outcomeText = "the check is still failing"
}
Detail: fmt.Sprintf("one automatic fix attempt ran for the %s check — %s", fix.Check, outcomeText),
```

## Info

### IN-01: A changed file at the repository root labels the whole suite as "targeted"

**File:** `cmd/verification_scope.go:152-172` (`goPackagePathsForChangedFiles`)

**Issue:** When a phase's changed files include a `.go` file directly at the
repository root (this repo has one: `embedded_assets.go`), `filepath.Dir`
returns `"."`, and the function emits the pattern `"./..."` — the exact same
pattern the "full" mode already uses. `deriveVerificationScope` then reports
`Mode: "targeted"`, `PackageCount: 1`, and a reason like "targeted to the 1
package(s) this phase's changed files touched", even though the resulting
test command is byte-identical to a full run. This is confirmed as tested,
intentional behavior (`TestScopedCommandNeverBroadensTheRun`'s "root package"
case explicitly asserts this is safe because it is a subset of, not broader
than, the full pattern), so it is not a coverage-safety bug — the "never
broaden or miss the changed package" invariant genuinely holds. It is,
however, a misleading label: the plain-English closing card this scope
system exists to feed (per the file's own doc comment, "targeted: 3
packages") would say something like "targeted: 1 package" for a change that
in fact ran every test in the project.

**Fix:** When `packages == ["./..."]`, report `Mode: verificationScopeFull`
with a reason naming the root-level file, rather than `verificationScopeTargeted`
with a misleading package count — the derived test command is identical
either way, so this is a labeling-only fix.

---

_Reviewed: 2026-08-22T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
