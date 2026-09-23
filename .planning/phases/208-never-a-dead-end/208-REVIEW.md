---
phase: 208-never-a-dead-end
reviewed: 2026-09-23T09:49:48Z
depth: standard
files_reviewed: 37
files_reviewed_list:
  - .aether/commands/midden-review.yaml
  - .aether/commands/report.yaml
  - .claude/commands/ant/midden-review.md
  - .claude/commands/ant/report.md
  - cmd/codex_build_finalize.go
  - cmd/codex_build.go
  - cmd/codex_colonize_finalize.go
  - cmd/codex_colonize.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_continue.go
  - cmd/codex_plan_finalize.go
  - cmd/codex_visuals.go
  - cmd/codex_workflow_cmds.go
  - cmd/command_guide.go
  - cmd/criterion_evidence.go
  - cmd/criterion_owner_confirmation.go
  - cmd/deterministic_floor.go
  - cmd/entomb_cmd.go
  - cmd/eval_gates.go
  - cmd/helpers.go
  - cmd/init_cmd.go
  - cmd/journey.go
  - cmd/memory_feed_continue.go
  - cmd/memory_feed.go
  - cmd/oracle_promote.go
  - cmd/phase_progress_from_disk.go
  - cmd/phase_skip.go
  - cmd/planning_state.go
  - cmd/refusal_log.go
  - cmd/refusal_register.go
  - cmd/refusal.go
  - cmd/report_cmd.go
  - cmd/root.go
  - cmd/session_flow_cmds.go
  - cmd/status.go
  - cmd/ux_friendly_errors.go
  - cmd/wrapper_command_names.go
  - Makefile
  - pkg/colony/sanitize.go
findings:
  critical: 3
  warning: 5
  info: 0
  total: 8
status: issues_found
---

# Phase 208: Code Review Report

**Reviewed:** 2026-09-23T09:49:48Z
**Depth:** standard
**Files Reviewed:** 38 (listed above; `.aether/commands/*.yaml` and `.claude/commands/ant/*.md` pairs counted separately)
**Status:** issues_found

## Summary

This phase built a real, checked-in refusal contract (`refuse()`/`refusal{}`/`renderRefusal`), a shrink-only untyped-floor ratchet, an `aether report` bundle, per-task criterion verdicts with directory-as-evidence detection, recovery tasks written back onto a blocked phase, and a journey harness that extracts and re-runs a refusal's own printed next command. The mechanism itself is sound and the individual conversions from bare `fmt.Errorf` to `refuse(...)` are, one by one, correct and well-targeted.

Three defects undercut the phase's own central promise ("every refusal carries a real way out, and it is proven by actually running it"):

1. Two registered refusal rows print a next command that is not a real, runnable command at all (a literal `<command>` placeholder never gets substituted).
2. The recovery-task/real-task boundary is decided by a plain string prefix on free-text `Goal`, which a genuinely human- or planner-authored task can collide with, silently removing it from the exact consistency checks (`validateCurrentPlanningState`, `phaseTaskIDSet`) that exist to catch a false record of what was built.
3. `aether report` is a brand-new command whose entire purpose is packaging local failure/refusal data for the owner to hand to a third party, and the sanitization it inherits was built to catch prompt-injection and shell-injection shapes, not literal secret values, so a credential embedded in captured build/test output can ride along into the shared bundle.

Also found: one code path whose refusal is never actually provable by the journey harness despite the doc comments claiming otherwise, a silent failure to honor `aether report --output`, an asymmetric hook-log exclusion list, a symlink-unaware directory containment check next to a symlink-aware one doing the identical job, and a printed-refusal extractor that trusts assistant prose rather than only genuine tool output.

## Critical Issues

### CR-01: Two refusal rows print a placeholder, not a runnable command

**File:** `cmd/refusal_register.go:237-254`
**Issue:** The `invalid-timeout-value` row (Disposition `warn`) and the `missing-required-flag` row (Disposition `stop`) both set:
```go
NextCommand: "aether <command> --help",
```
Nothing in `refuse()` (`cmd/refusal.go:42-68`), `renderRefusal` (`cmd/refusal.go:134-169`), or `renderWarning` (`cmd/refusal.go:98-122`) ever substitutes `<command>` with the command that actually failed — `grep -n '"<command>"' cmd/*.go` outside the register itself returns nothing. `renderRefusal` prints `Next: \`aether <command> --help\`` verbatim, and that is the exact anchored shape (`^Next: \`...\``) `journeyPrintedRefusalNextLineRe` (`cmd/journey.go:289`) treats as a proven way out and `journeyRunPrintedNextCommands` (`cmd/journey_live_test.go:414`) actually executes as a subprocess. Run for real, `aether <command> --help` fails with an unknown-command error, which `journeyNextCommandFailureReason` (`cmd/journey.go:401`) explicitly classifies as a real failure ("reported an unknown command").

This is exactly the class of defect this whole phase exists to close (WINDOWS.md row 53: "a refusal that names no way forward"), and it directly contradicts this project's own Definition of Done ("A requirement is satisfied only when a command exists that someone can run"). It also breaks the plain-English contract in CLAUDE.md ("give me the exact thing to type... not a description of it") — a non-technical owner who copy-pastes `aether <command> --help` gets a shell error, not help.

**Fix:** Either give each row its own concrete `NextCommand` (e.g. `aether continue --worker-timeout <duration>` isn't right either since the row doesn't know the invoking command — simplest fix is to drop the fabricated flag name and point at a real, always-valid command, e.g. `aether status` or `aether --help`), or thread the actual cobra command path into `refuse()` and substitute it before rendering. Whatever is chosen, add a table-driven check (extending `refusalRegistryProblems`) that fails any row whose `NextCommand` contains a literal `<` or `>` character.

### CR-02: Recovery-task recognition by string prefix can misclassify a genuine task

**File:** `cmd/codex_continue.go:3150-3165` (`recoveryTaskGoalGeneratedPrefix`, `taskGoalLooksLikeRecoveryTask`), consumed at `cmd/codex_continue.go:3178-3206` (`tasksExcludingRecovery`, `planPhasesExcludingRecoveryTasks`), `cmd/codex_build.go:3620-3639` (`phaseTaskIDSet`), and `cmd/planning_state.go:209-238` (`validateCurrentPlanningState`)
**Issue:** A task is recognised as a system-generated recovery task purely by whether its free-text `Goal` begins with the literal substring `"Finish task "`:
```go
func recoveryTaskGoalGeneratedPrefix(goal string) string {
	goal = strings.TrimSpace(goal)
	if !strings.HasPrefix(goal, "Finish task ") {
		return ""
	}
	...
}
```
There is no dedicated field (a `Kind`/`IsRecovery` marker, a reserved ID prefix, anything structural) distinguishing an auto-appended recovery task from an ordinary human- or route-setter-authored task. A perfectly plausible real task Goal — "Finish task queue implementation", "Finish task list UI", "Finish task management for the sprint" — collides with this shape.

The consequence is not cosmetic: `tasksExcludingRecovery` is what `validateCurrentPlanningState`'s `reflect.DeepEqual` comparison against the accepted plan revision runs on (`cmd/planning_state.go:223`), and what `phaseTaskIDSet` runs on for the build-manifest task-set check (`cmd/codex_build.go:3628`). Both exist specifically to catch a phase whose live task list has silently diverged from what was actually planned or dispatched — ground (b) territory in this project's own refusal classification rubric ("record a completion, verification or advancement that is not true"). A colliding real task is silently dropped from both comparisons: a build manifest that never dispatched it, or an accepted plan that never proposed it, would no longer be flagged as a mismatch for that task. The same collision also feeds `alreadyRecoveryTaskIDs` (`cmd/codex_continue.go:3236-3241`), so a genuinely blocked real task with a colliding Goal would never get its own recovery task appended on a later failed check, because it already "looks like" one.

**Fix:** Give a recovery task a real structural marker instead of relying on its rendered prose — e.g. a `colony.Task.Origin` (or similarly named) enum field set to `"recovery"` at creation and never derived from `Goal` text. The doc comment at `recoveryTaskSemanticID` explains why `SemanticID` specifically was avoided (it trips the `legacy_unbound` planning-authority guard) — a new, narrowly-scoped field sidesteps that guard entirely while giving `taskGoalLooksLikeRecoveryTask`'s three call sites something that cannot collide with human prose.

### CR-03: `aether report` bundles unredacted failure output for external sharing

**File:** `cmd/report_cmd.go:118-225`, `cmd/memory_feed_continue.go:137-169`, `pkg/colony/sanitize.go`
**Issue:** `aether report` is new in this phase and its entire purpose, stated in its own banner text and in `.aether/commands/report.yaml`, is producing a file "to send to whoever maintains Aether." `reportBundleFailuresSection` (`cmd/report_cmd.go:187-210`) lists up to 20 recent midden entries verbatim, and those entries are populated (`recordFailedChecksToMidden`, `cmd/memory_feed_continue.go:137-169`) from the last line of real build/type/lint/test tool output (`failureSummaryForStep`, `cmd/codex_continue.go:5535`) run through `colony.SanitizeSignalContent` / `colony.NeutralizeForRecord`.

Both of those sanitizers (`pkg/colony/sanitize.go`, `pkg/colony/prompt_integrity.go`) are built to catch XML structural tags, prompt-injection phrasing, shell-injection shapes (backticks, `$()`, pipe/semicolon-`rm` chains), and *paths* that look like secrets files (`secretsPathRules` — e.g. `~/.aws/credentials`, `.env`). None of these rules match a literal secret *value* — an API key, bearer token, or password that a failing build or test genuinely printed to stdout/stderr (e.g. a test that hit a real external service and echoed its auth header on failure) passes through untouched and is a candidate for `aether report`'s bundle. This is a new externally-facing distribution channel for data that previously stayed local to `.aether/data/` (gitignored); the project's own CLAUDE.md is explicit that a secret must never be written into "anything that gets committed" — the same principle applies once a purpose-built "send this to someone else" command exists.

Note that the refusal log itself (`refusalLogEntry`, `cmd/refusal_log.go:19-31`) is safe here: `refuse()` only ever writes dynamic call-site detail into `Why`, and `refusalLogEntry` never persists `Why` at all — only the static, hand-written `What`. The exposure is entirely in the midden/failure-log section.

**Fix:** Add a dedicated secret-value redaction pass (common token shapes: `sk-...`, `AKIA[0-9A-Z]{16}`, `ghp_...`, `Bearer <token>`, generic `[A-Za-z0-9+/]{32,}={0,2}` base64-ish runs, `Authorization:`/`X-Api-Key:` header lines) to `reportBundleFailuresSection` and/or `NeutralizeForRecord` itself, and consider warning the owner in the bundle's own "What this is" section that raw tool output may contain secrets and should be scanned before sending.

## Warnings

### WR-01: `verification-command-not-understood`'s "Next" command is never proven and never shown to the owner

**File:** `cmd/deterministic_floor.go:186-195`
**Issue:** `applyUnreadableVerificationCommandRefusals` builds a step summary by concatenating `r.What`, `r.Why`, and the next command into one sentence:
```go
steps[i].Summary = strings.TrimSpace(strings.TrimSpace(r.What) + " " + strings.TrimSpace(r.Why) + fmt.Sprintf(" Next: `%s`.", strings.TrimSpace(r.NextCommand)))
```
This never goes through `renderRefusal`; `Summary` is instead one of many strings joined with `"; "` in `cmd/codex_continue.go:2011` (`strings.Join(verification.BlockingIssues, "; ")`) or embedded mid-sentence elsewhere. Because `"Next: \`aether patrol\`."` is never on its own line starting exactly with `Next:`, `journeyPrintedRefusalNextLineRe` (anchored `^Next: `) can never find it, so this refusal's way out can never be proven by the journey harness the way every other refusal's is. Separately, the owner-facing "what to run next" for a blocked continue comes from `continueNextCommandForBlocked` (`cmd/codex_continue.go:3344`), which has its own independent fallback logic (reconcile/redispatch/`aether status`) and never reads this row's `NextCommand` at all — so `aether patrol` is not actually what the owner sees next for this specific refusal, despite the row naming it.

**Fix:** Route this refusal through the same `outputRefusal`/`renderRefusal` (or at minimum through `warnAndCarryOn`) path every other typed refusal uses instead of hand-formatting `Summary`, and make `continueNextCommandForBlocked` recognise this specific blocker and surface its registered `NextCommand`.

### WR-02: `aether report --output` failures are silently swallowed

**File:** `cmd/report_cmd.go:41-52`
**Issue:**
```go
if mkErr := os.MkdirAll(target, 0755); mkErr == nil {
    dest := filepath.Join(target, filepath.Base(path))
    if renameErr := os.Rename(path, dest); renameErr == nil {
        path = dest
    }
}
```
If `MkdirAll` or `Rename` fails — e.g. a cross-device rename (common when `--output` points at a different filesystem/mounted volume than `.aether/reports/`), a permissions error, or a non-existent parent — the function falls straight through with no error and no warning. The command still reports success and prints the *default* location as if `--output` had been honored, with nothing telling the owner their explicit flag was silently ignored.

**Fix:** Surface a warning line (or refuse outright) when `--output` cannot be honored, naming the real path where the file actually landed.

### WR-03: Refusal-log hook exclusion is a fixed enumeration, not systematic hook coverage

**File:** `cmd/refusal_log.go:46-51`
**Issue:** `refusalLogExcludedCommands` hand-lists exactly `hook-stop`, `hook-post-tool-use`, `hook-session-start`, and `status-line` — the four commands with an existing, tested "never mutates" guarantee. `hook-pre-tool-use` and `hook-pre-compact` (`cmd/hook_cmds.go:58`, `cmd/hook_cmds.go:675`) are not in this list and currently call no `refuse()`/`outputRefusal`/`warnAndCarryOn` path, so today this is latent rather than active. But nothing structurally prevents a future refusal being wired into either of those two hook commands, at which point `appendRefusalToLog` would begin writing to `refusals.jsonl` during a hook invocation with no test catching the new mutation the way `TestRefusalLogIsSkippedForHookCommands` catches the four named ones.

**Fix:** Either derive the exclusion from a `strings.HasPrefix(currentStreamingCommand, "hook-")` check (broadened deliberately, with `status-line` kept as an explicit extra), or add a structural test that fails when a new `hook-*` command is registered without a corresponding entry in `refusalLogExcludedCommands` (or an explicit, reviewed opt-in).

### WR-04: Verification working-directory containment check doesn't resolve symlinks

**File:** `cmd/codex_continue.go:4067-4122` (`splitVerificationCommandDirectoryPrefix`), `cmd/codex_continue.go:4178-4204` (`runVerificationStepInDir`)
**Issue:** Both functions guard against a `cd`-prefixed verification-command directory escaping the project root using only string/path-cleaning checks (`filepath.Clean`, rejecting a leading `..`). Neither resolves symlinks. `criterionArtifactBindingIsDirectory` in the same phase (`cmd/criterion_evidence.go`) does exactly this kind of containment check correctly, using `filepath.EvalSymlinks` before deciding. A directory name that is itself a symlink pointing outside the project root (e.g. a `vendor` or `shared` symlink some monorepo layouts use) would pass `runVerificationStepInDir`'s containment check as a clean, in-tree relative path, yet the shell would actually execute in a location outside the project root once it follows the symlink.

**Fix:** Resolve the target with `filepath.EvalSymlinks` (falling back to the unresolved path only if resolution fails because the directory doesn't exist yet) before the containment check, matching `criterionArtifactBindingIsDirectory`'s existing pattern.

### WR-05: Printed-refusal extraction trusts assistant prose, not only genuine tool output

**File:** `cmd/journey.go:299-331` (`journeyPrintedRefusals`)
**Issue:** The regex-based extractor looks for the `^Next: \`...\`` shape in two places: an assistant's own `text` content block, *or* a `tool_result` block. The assistant-text branch means any text the chat model itself produces — including a hallucinated or copy-pasted line that happens to match `Next: \`aether ...\`` — is treated identically to a real, verbatim block of Aether's own rendered output, and `journeyRunPrintedNextCommands` (`cmd/journey_live_test.go:414-469`) will genuinely execute it as `exec.CommandContext(ctx, "aether", fields[1:]...)`. Execution is contained (args-array, not a shell string, and gated to `fields[0] == "aether"`), and this only runs inside the disposable practice-project sandbox the journey harness builds, so the practical blast radius today is low — but the design does not actually verify the matched text came from Aether's own tool output at all, which is a weaker guarantee than the surrounding code's comments ("a real `renderRefusal` block") imply.

**Fix:** Prefer restricting the extraction to `tool_result` blocks only (the actual captured stdout/stderr of a Bash call), or at minimum require the assistant-text match to be immediately preceded by a fenced code block boundary consistent with the wrapper commands' own "show it in a fenced text block... unchanged" instruction, so a model's own free-form prose can't masquerade as a genuine refusal screen.

---

_Reviewed: 2026-09-23T09:49:48Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
