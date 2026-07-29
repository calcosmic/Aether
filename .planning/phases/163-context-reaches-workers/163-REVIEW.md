---
phase: 163-context-reaches-workers
reviewed: 2026-07-29T00:00:00Z
depth: standard
files_reviewed: 41
files_reviewed_list:
  - .aether/references/contracts/protected-local-state-contract.md
  - .aether/rules/aether-colony.md
  - .claude/commands/ant/build.md
  - .claude/rules/aether-colony.md
  - .opencode/commands/ant/build.md
  - .opencode/OPENCODE.md
  - cmd/build_print_brief_test.go
  - cmd/build_print_brief.go
  - cmd/build_wrapper_ceremony_test.go
  - cmd/ceremony_cmd_test.go
  - cmd/ceremony_cmd.go
  - cmd/charter_gate_test.go
  - cmd/codex_build_finalize_test.go
  - cmd/codex_build_finalize.go
  - cmd/codex_build_manifest_context_test.go
  - cmd/codex_build_test.go
  - cmd/codex_build.go
  - cmd/codex_continue.go
  - cmd/codex_workflow_cmds.go
  - cmd/colony_prime_charter_test.go
  - cmd/colony_prime_context.go
  - cmd/context_budget_test.go
  - cmd/context_weighting.go
  - cmd/gate_test.go
  - cmd/gate.go
  - cmd/helpers.go
  - cmd/hook_cmds_test.go
  - cmd/hook_cmds.go
  - cmd/internal_worker_adapter_test.go
  - cmd/permission_profile_integration_test.go
  - cmd/safety_invariant_test.go
  - cmd/suggest_analyze_test.go
  - cmd/suggest_analyze.go
  - cmd/survey_staleness_test.go
  - cmd/survey_staleness.go
  - cmd/testdata/command_catalog.json
  - cmd/testdata/regression_snapshot.json
  - pkg/codex/permission_profile_test.go
  - pkg/codex/permission_profile.go
  - pkg/codex/worker_test.go
  - pkg/codex/worker.go
findings:
  critical: 1
  warning: 6
  info: 4
  total: 11
status: issues_found
---

# Phase 163: Code Review Report

**Reviewed:** 2026-07-29
**Depth:** standard
**Files Reviewed:** 41
**Status:** issues_found

## Summary

Phase 163 wired colony-prime context, the colony charter, the charter compliance
gate, survey staleness notices, the suggest-analyze pipeline, sanctioned
`.aether/data` scratch subpaths, and a checklist-first `--print-brief` inspector
into the build/continue runtime. The implementation is broadly high quality:
producers have live callers, gates copy the established two-check honesty shape,
prompt-injection controls are inherited via `colonyPrimeSection` rather than
string concatenation, the hook allowlist uses slash-delimited segments with
negative tests, and the four documentation surfaces are pinned by a test
(`TestSanctionedScratchDirsDocumented`). Docs and code agree.

Verification performed: `go build ./...` and `go vet ./cmd ./pkg/codex` are
clean. The phase's new tests pass. However, `go test ./cmd` **fails at HEAD in
this repository**: `TestSuggestAnalyze_NonBlockingOnError` escapes its sandbox,
runs a real persisting `suggest-analyze` against the developer's repo, and
fails whenever an active colony exists (CR-01). Reproduced at the diff base
with identical environment, so the escape is pre-existing brittleness — but the
phase edited this exact file, its refactor claim ("extraction changed no
observable output") was validated only in clean worktrees, and the phase's new
closeout feature makes the resulting state pollution user-visible.

**Disclosure:** running the test suite during this review exercised the
defective test, which wrote ~136 junk pending suggestions and
`last_analyze_commit` into the real `.aether/data/COLONY_STATE.json` (local
only — the file is gitignored). This mutation is itself the proof of CR-01.
The state should be cleaned (`aether suggest-approve --dismiss-all` or
`/ant-data-clean`) — the new "Suggestions From This Build" closeout block will
otherwise display all 136 to the user at the next build.

## Critical Issues

### CR-01: `TestSuggestAnalyze_NonBlockingOnError` escapes its sandbox — suite fails at HEAD and pollutes the real colony state

**File:** `cmd/suggest_analyze_test.go:239-265` (interaction with `cmd/root.go:179-194`)
**Issue:** The test sets `store = nil` to force the no-store error path, then
calls `rootCmd.Execute()`. `rootCmd.PersistentPreRunE` unconditionally
re-initializes `store` from `storage.ResolveDataDir` (there is no
`COLONY_DATA_DIR` isolation in this test, unlike `newTestStore`). Consequences:

1. The nil-store branch under test is never reached through the CLI; the test
   only passed historically because CI/worktree environments had no
   `.aether/data`.
2. On any machine with an active colony (this one), the test runs a **real,
   persisting** `suggest-analyze --target .` against the developer's repo:
   `expected 0 suggestions on error, got 136`. `go test ./cmd` fails at HEAD.
   Reproduced identically at the diff base (`34e6f10b`) with the same state
   present, confirming the escape predates the phase — but the phase's
   validation claim "CLI output byte-identical, all tests pass" was never true
   in the dogfooding environment this project runs in.
3. The run writes ~136 pending suggestions plus `last_analyze_commit` into the
   real `.aether/data/COLONY_STATE.json` — a path every project document in
   this phase declares protected. Phase 163's new closeout block
   (`renderPendingSuggestionsBlock`) then surfaces that junk to the user.

**Fix:**
```go
func TestSuggestAnalyze_NonBlockingOnError(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	// Isolate the data dir so PersistentPreRunE cannot resolve the real
	// repo colony; point COLONY_DATA_DIR at an unwritable/empty location,
	// or test the nil-store contract on runSuggestAnalyze directly:
	store = nil
	if _, err := runSuggestAnalyze(".", false); err == nil {
		t.Fatal("expected hard error when no store is initialized")
	}
}
```
If the CLI envelope behavior must stay covered, use `newTestStore(t)` (which
sets `COLONY_DATA_DIR`) with a deliberately corrupt/absent `COLONY_STATE.json`
to exercise the non-blocking empty-result path instead. Separately, clean the
polluted local state (`aether suggest-approve --dismiss-all` or
`/ant-data-clean`).

## Warnings

### WR-01: `runSuggestAnalyze` persists colony state via non-atomic read-modify-write

**File:** `cmd/suggest_analyze.go:194-201`
**Issue:** The persist branch marshals the `cs` snapshot loaded at function
entry and overwrites the whole file with `store.AtomicWrite("COLONY_STATE.json", ...)`.
Between load and write, the function shells out to git and walks the entire
repo tree (TODO scan, large-file scan, test-gap scan) — a window of seconds on
large repos. Any state change landing in that window is silently clobbered.
The codebase already has the correct primitive (`store.UpdateJSONAtomically`,
used by `gateResultsWrite` at `cmd/gate.go:1225`), and the project's own
history records full-state reconstruction as its worst corruption class. The
pattern is pre-existing, but this phase gave the function its first live
caller in the critical path (`collectPendingSuggestions` runs immediately
after the build lifecycle commit in `runCodexBuildFinalize`), so exposure is
now per-build rather than manual-only.
**Fix:** Move the merge of `PendingSuggestions`/`LastAnalyzeCommit` inside a
`store.UpdateJSONAtomically("COLONY_STATE.json", ...)` mutation callback so
the read and write are a single guarded operation.

### WR-02: `total` / `pending_suggestion_count` semantics are inconsistent across branches

**File:** `cmd/suggest_analyze.go:99-107, 150-209`; `cmd/codex_build_finalize.go:461-471`
**Issue:** On a full analysis run, `total` counts only NEW sanitized
suggestions — existing pending suggestions carried forward by the merge are
excluded. On the below-threshold skip path, `total` counts ALL stored pending
suggestions **including dismissed ones** (`pendingSuggestionsToMap` does not
filter `Dismissed`). `collectPendingSuggestions` forwards this as
`pending_suggestion_count` and gates the `pending_suggestions_next: aether
suggest-approve` hint on it. Concrete misbehaviors: (a) a build whose analysis
finds nothing new reports `pending_suggestion_count: 0` and no approve hint
even when older suggestions still await review; (b) a below-threshold build
whose only stored suggestions are all dismissed reports a positive count and
tells the user to run `suggest-approve` with nothing actionable.
**Fix:** Define one meaning (active pending suggestions after merge), filter
with the same `Dismissed` predicate `filterActiveSuggestions` uses, and return
that from both branches.

### WR-03: Budget ceiling sums the non-compact capsule constant while the delivered capsule is compact

**File:** `cmd/build_print_brief.go:256-262`; `cmd/colony_prime_context.go:962-963`
**Issue:** `assembledContextBudgetCeilingChars()` adds `colonyPrimeBudgetChars`
(8000), but every capsule this phase actually delivers —
`resolveCodexWorkerContext()` in the plan-only manifest, the hosted path, and
the checklist itself — is built with `buildColonyPrimeOutput(true)` under
`colonyPrimeCompactBudgetChars` (4000). The D-03 growth guard
(`TestAssembledContextStaysUnderBudgetCeiling`) and the checklist's
`TOTAL x/y z%` line are therefore ~4000 chars looser than the real delivery
budget, and the per-section assertion `len(capsule) <= colonyPrimeBudgetChars`
cannot bind until the capsule exceeds double its actual cap. A guard that
cannot trip until reality has drifted 2x is the drift the constants were named
to prevent.
**Fix:** Sum `colonyPrimeCompactBudgetChars` in the ceiling (or make
`resolveCodexWorkerContext` take the budget it is measured against), and
tighten the individual capsule assertion to the same constant.

### WR-04: `--print-brief --full` does not print "exactly what the worker receives"

**File:** `cmd/build_print_brief.go:87-95, 373-382`
**Issue:** The checklist's TOTAL correctly counts capsule + brief + skill
section — the wrapper contract (`.claude/commands/ant/build.md:97`) prompts
workers with `context_capsule + brief + skill_section`. But `--full` prints
only `brief` and computes its composition table over `brief` alone: the
manifest-level capsule (including the charter, the very thing this phase
delivers) and `dispatch.SkillSection` are silently absent from the "raw
assembled prompt." The function's own doc comment ("the inspector must show
exactly what a wrapper-spawned worker receives") and the flag help text are
not satisfied in --full mode, and composition percentages use a smaller
denominator than the checklist's TOTAL, so the two modes disagree about the
same prompt.
**Fix:** In the `options.Full` branch, print the capsule once (clearly marked
manifest-level), then the brief, then the skill section, and feed
capsule+brief+skills into the composition denominator — or rename the output
to state it is the brief only.

### WR-05: Scout elevation to `workspace_write` is prose-enforced on two of three platforms; lexical hook check is symlink-bypassable

**File:** `pkg/codex/permission_profile.go:52-95`; `cmd/hook_cmds.go:217-251, 316-334`
**Issue:** Removing `scout` from `repositoryReadOnlyCastes` converts a
sandbox-enforced read-only boundary into `workspace_write` plus a behavioral
restriction string ("write phase research artifacts under
.aether/data/phase-research only"). Mechanical enforcement exists only on
Claude Code via the `PreToolUse` allowlist; `.opencode/OPENCODE.md` explicitly
concedes "conduct, not a sandbox" for OpenCode, and Codex native has no
equivalent hook. A scout can now write arbitrary project source on those
platforms with nothing but prose in the way. Additionally,
`normalizeHookPath` is purely lexical (`filepath.Clean`, no
`filepath.EvalSymlinks`), so a symlink created inside a sanctioned scratch dir
(e.g. `.aether/data/planning/link -> ../COLONY_STATE.json`) makes a Write to
the "allowed" path land on protected state. This is a deliberate D-05 design
trade, but the asymmetry and the symlink gap are unstated in the contract doc,
which presents the allowlist as the enforcement mechanism.
**Fix:** Resolve symlinks in `normalizeHookPath` before matching
(`filepath.EvalSymlinks` on the deepest existing ancestor), and add the
platform-enforcement asymmetry to
`.aether/references/contracts/protected-local-state-contract.md` as a named
residual risk.

### WR-06: Checklist charter detection breaks under section-template header overrides

**File:** `cmd/build_print_brief.go:353`; `cmd/prompt_template_loader.go:111-117`
**Issue:** The Charter row is located by the hardcoded string
`"## Charter -- Binding Rules"`, but the charter section header is emitted via
`writeSectionHeader("charter", ...)`, which prefers a colony's
`SectionTemplates["charter"].Header` override when present. A colony using a
custom charter header gets a checklist that reports the charter ABSENT while
it is in fact delivered — a false negative from the one tool built to answer
"did this context arrive." The phase's own manifest test
(`cmd/codex_build_manifest_context_test.go:93-95`) explicitly refuses
hardcoded-heading probes for exactly this reason; the production checklist
should hold itself to the same standard. The same fragility applies to the
`"STALE MAP WARNING"` / `"never been surveyed"` literals if
`surveyStalenessNotice` wording ever changes (though those are at least
defined in the same repo).
**Fix:** Resolve the charter heading through the same template path the
producer uses (e.g. a shared `charterSectionHeading()` helper called by both
`buildColonyPrimeOutput` and `renderBriefChecklist`), or match on the ledger
(`colonyPrimeOutput.Ledger.Included` contains a `charter` entry) instead of
string-searching the assembled text.

## Info

### IN-01: `locateChecklistSection` matches headings anywhere in the text, not at line starts

**File:** `cmd/build_print_brief.go:284-302`
**Issue:** `strings.Index(text, heading)` reports "present" for prose that
merely quotes a heading (research or handoff content containing the literal
`## Phase Research`), and `"### X"` contains `"## X"` as a substring, so a
sub-heading can satisfy a top-level probe. Sizes are then measured from the
false match.
**Fix:** Anchor matches to line starts: check `strings.HasPrefix(text, heading)`
or search for `"\n" + heading`.

### IN-02: `countChangedFiles` misses the singular "1 file changed" summary line

**File:** `cmd/suggest_analyze.go:234-251`
**Issue:** The filter skips lines containing `"files changed"`, but a
single-file diff's summary reads `"1 file changed, ..."` (singular), so the
count comes back as 2 for a 1-file change. Harmless at `changeThreshold = 5`,
and pre-existing, but the off-by-one is real.
**Fix:** Also skip lines containing `" file changed"`, or use
`git diff --name-only | wc -l` semantics instead of parsing `--stat`.

### IN-03: `--print-brief` help says "mutates nothing" but the read path writes cache files

**File:** `cmd/codex_workflow_cmds.go:1167`; `cmd/colony_prime_context.go:326, 367-368`
**Issue:** The capsule/skill resolution invoked per dispatch writes
`reviews/_summary_cache.json` (`buildPriorReviewsSection`) and deletes stale
session-cache files (`sc.ClearStale`). The phase itself acknowledged this by
teaching `TestPlanOnlyUnchanged` to tolerate `.cache_*` side files. These are
cache-tier, not colony-state, mutations — but "Reads state; mutates nothing"
in the flag help and the "calls only pure readers" doc comment overstate the
guarantee that the phase's own dry-run history says must be exact.
**Fix:** Soften the wording to "never mutates colony state (may refresh local
read caches)".

### IN-04: "Suggestions From This Build" heading over-claims

**File:** `cmd/ceremony_cmd.go:241-245, 585-592`
**Issue:** The closeout block renders every active pending suggestion in
colony state, including ones accumulated by earlier builds or manual
`suggest-analyze` runs — not only suggestions from the build being closed out.
With CR-01's pollution present, that is 136 entries under a heading claiming
they came from this build.
**Fix:** Either scope the block to suggestions whose `CreatedAt` postdates the
build's `startedAt`, or rename the stage marker to "Pending Suggestions".

---

_Reviewed: 2026-07-29_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
