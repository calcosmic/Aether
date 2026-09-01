---
phase: 198-put-the-thrown-away-data-back-on-screen
reviewed: 2026-08-29T00:00:00Z
depth: standard
files_reviewed: 46
files_reviewed_list:
  - .claude/commands/ant-build.md
  - .claude/commands/ant-plan.md
  - .claude/commands/ant-seal.md
  - .claude/commands/ant/build.md
  - .claude/commands/ant/plan.md
  - .claude/commands/ant/seal.md
  - .opencode/commands/ant/build.md
  - .opencode/commands/ant/plan.md
  - .opencode/commands/ant/seal.md
  - cmd/blackbox_harness_test.go
  - cmd/build_blocker_advisory_test.go
  - cmd/build_blocker_advisory.go
  - cmd/build_wrapper_ceremony_test.go
  - cmd/ceremony_cmd.go
  - cmd/ceremony_emitter_test.go
  - cmd/ceremony_team_checkin.go
  - cmd/closeout_cmd.go
  - cmd/closeout_direct_render.go
  - cmd/codex_build_progress.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_continue.go
  - cmd/codex_visuals.go
  - cmd/codex_workflow_cmds_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/context.go
  - cmd/continue_detail_render_test.go
  - cmd/continue_live_progress_test.go
  - cmd/continue_worker_measurements_test.go
  - cmd/deliberate_drops_lock_test.go
  - cmd/e2e_lifecycle_test.go
  - cmd/golden_workflow_test.go
  - cmd/hive_policy_test.go
  - cmd/plan_wrapper_ceremony_test.go
  - cmd/rendered_fields_invariant_test.go
  - cmd/resume_detail_test.go
  - cmd/seal_ceremony_test.go
  - cmd/seal_confirmation_test.go
  - cmd/seal_confirmation.go
  - cmd/seal_final_review.go
  - cmd/seal_wrapper_ceremony_test.go
  - cmd/state_load_test.go
  - cmd/status.go
  - cmd/testdata/golden_build.txt
  - cmd/testdata/golden_continue.txt
  - cmd/testdata/golden_plan.txt
  - cmd/testdata/rendered_field_allowlist_baseline.json
  - cmd/testdata/rendered_field_allowlist.json
  - cmd/wrapper_path_parity_test.go
findings:
  critical: 2
  warning: 3
  info: 0
  total: 5
status: clean
---

# Phase 198: Code Review Report

**Reviewed:** 2026-08-29
**Depth:** standard
**Files Reviewed:** 46 (diff against e885a989)
**Status:** issues_found

## Summary

This phase restores previously-collected-but-never-shown data (verification detail,
gate detail, requirement evidence, specialist findings, resume phase progress, plan
confidence, a build-start blocker heads-up, and a seal confirmation gate) to the
rendered screens across `plan`, `build`, `continue`, `seal`, and `resume`. The dual-type
rendering pattern (typed struct on the direct path, JSON-round-tripped map on the
chat/closeout path) is applied consistently and is generally well tested, and the
wrapper markdown triplets (`.claude/commands/ant-*.md`, `.claude/commands/ant/*.md`,
`.opencode/commands/ant/*.md`) remain byte-identical as required.

Two BLOCKER-level defects were found, both matching this project's own stated failure
pattern of "the guarantee exists on paper (a comment, a test) but not on the path an
owner actually takes": (1) a new plain-English gate-name translation table only covers
6 of the ~19 real gate keys the runtime emits, so raw internal names leak straight to
the non-technical owner — and the regression test meant to catch exactly this is
self-referential and cannot fail; (2) the new build-start blocker heads-up's
"unanswered orchestrator boundary question" signal is silently dead on the direct
(non-`--plan-only`) build lane, contradicting the code's own comment that both lanes
call the signal logic "identically."

## Critical Issues

### CR-01: Gate names shown to the owner leak raw internal jargon, and the guard test cannot catch it

**File:** `cmd/codex_visuals.go:2698-2712` (gateCheckDisplayNames), confirmed shipped in `cmd/testdata/golden_continue.txt:47` (`✓ anti pattern executed`)

**Issue:**
The new "Gates: N/M passed" detail block (`renderContinueGateDetail` /
`renderGateCheckDetailLines`) translates a gate's internal snake_case key into
plain English via a lookup table:

```go
var gateCheckDisplayNames = map[string]string{
	"manifest_present":           "the build's own plan file is on disk",
	"verification_steps_passed":  "the build/test checks passed",
	"implementation_evidence":    "there's evidence the work was actually done",
	"owner_confirmation_pending": "nothing is waiting on your confirmation",
	"anti_pattern":               "no risky code patterns were found",
	"charter_compliance":         "the project's own rules were followed",
}
```

Any key not in this map falls through to `gateCheckDisplayName`'s default branch,
which just does `strings.ReplaceAll(trimmed, "_", " ")`. The runtime's real universe
of gate keys is much larger — `gateClassifications` in `cmd/gate.go` alone names 15
more: `gatekeeper`, `watcher_veto`, `flags`, `tests_pass`, `no_critical_flags`,
`anti_pattern_executed`, `charter_compliance_executed`, `auditor`, `complexity`,
`tdd_evidence`, `verification_loop`, `spawn_gate`, `medic`, `runtime`. Several of
these coincidentally read as plausible English once underscores become spaces
("no critical flags"), but others do not: `anti_pattern_executed` renders as
**"anti pattern executed"** — raw jargon ("anti pattern" is this repo's internal
term, never explained to the owner) glued to an odd verb. This is confirmed shipped
behavior: `cmd/testdata/golden_continue.txt` (a locked golden fixture, i.e. "this is
correct output") contains the line `✓ anti pattern executed` right next to the
correctly-translated `✓ no risky code patterns were found` for the sibling gate.

This is exactly the failure mode CLAUDE.md dedicates its opening section to: "the
internal key never reaches the owner directly," "translate each one inline, every
single time." It is not a hypothetical — it is in the committed golden output.

Worse, the regression test intended to catch this class of bug cannot ever catch
this specific instance. `TestRestoredDetailIsPlainEnglish`
(`cmd/continue_detail_render_test.go:386`) builds its list of "internal names that
must not leak" from `continueGateCheckNames`, which is *derived from
`gateCheckDisplayNames`'s own keys* (`cmd/codex_visuals.go:2723-2730`):

```go
var continueGateCheckNames = func() []string {
	names := make([]string, 0, len(gateCheckDisplayNames))
	for name := range gateCheckDisplayNames {
		names = append(names, name)
	}
	return names
}()
```

So the test only ever asks "does the table's own key leak verbatim?" — which is
tautologically false for every key already in the table, and it never enumerates
`anti_pattern_executed`, `watcher_veto`, `spawn_gate`, etc. at all, because those
never entered the input list in the first place. This is precisely the "false
certificate" pattern CLAUDE.md's Definition of Done calls out: a fixture built from
the shape being tested, not from what the runtime actually produces.

**Fix:**
1. Add the missing translations to `gateCheckDisplayNames` (at minimum
   `anti_pattern_executed`, and ideally all of `gateClassifications`'s keys —
   `gatekeeper`, `watcher_veto`, `flags`, `tests_pass`, `no_critical_flags`,
   `charter_compliance_executed`, `auditor`, `complexity`, `tdd_evidence`,
   `verification_loop`, `spawn_gate`, `medic`, `runtime`).
2. Fix `TestRestoredDetailIsPlainEnglish` to derive its "must not leak verbatim"
   name list from the real runtime source of gate keys (e.g. the keys of
   `gateClassifications` unioned with the four structural keys already in
   `gateCheckDisplayNames`), not from `gateCheckDisplayNames` itself, so a future
   gate name that never got a translation actually fails the test.

## Warnings

### WR-01: Direct build lane's "unanswered orchestrator boundary question" signal is silently dead, contradicting its own "both lanes agree" comment

**File:** `cmd/codex_workflow_cmds.go:319-337`, `cmd/build_blocker_advisory.go:44-49`

**Issue:**
The new D-08/D-09 "build-start blocker heads-up" is supposed to fire an
`unanswered-question` signal whenever `manifest.BoundaryQuestionCount > 0`
(`buildStartBlockerSignals`, `cmd/build_blocker_advisory.go:44`). On the
`--plan-only` lane this works: the manifest passed in comes from
`buildCodexBuildManifest` + `materializeOrchestratorBoundaryQuestions("build", ...)`
(`cmd/codex_build.go:490-497`), which actually computes and sets
`BoundaryQuestionCount`.

On the direct dispatch lane (`aether build <phase>` without `--plan-only`,
exercised in tests via `--synthetic`), the new code builds its own throwaway
manifest instead of reusing anything real:

```go
directBuildManifest := codexBuildManifest{
    Phase:           phaseNum,
    ForcedReviewers: forcedReviewerRecords(queenForcedReviewersForPhase(state.Plan.Phases[phaseNum-1])),
}
blockerAdvisoryDirect := decideBuildBlockerAdvisory(buildStartBlockerSignals(directBuildManifest), noCheckinFlag)
```

`BoundaryQuestionCount` is left at its zero value, so `buildStartBlockerSignals`
always falls through to the `pendingHandoffDecisions` branch for the
"unanswered-question" signal and can never report a real, unresolved orchestrator
boundary question on this lane. This is not merely a missed reuse of a nearby
variable — a few lines above, this same function already loads the real on-disk
`manifest.json` into a local `manifest` variable and then discards everything
except `manifest.Dispatches`:

```go
if manifestPath, ok := result["manifest"].(string); ok && strings.TrimSpace(manifestPath) != "" {
    rel := strings.TrimPrefix(manifestPath, ".aether/data/")
    var manifest codexBuildManifest
    if err := store.LoadJSON(rel, &manifest); err == nil && len(manifest.Dispatches) > 0 {
        dispatches = manifest.Dispatches
    }
}
```

But this doesn't fully solve it either: the direct-dispatch manifest-building code
path (`cmd/codex_build.go:2655`, `planOnly=false`) never calls
`materializeOrchestratorBoundaryQuestions` at all, unlike the plan-only path at
line 490 — so even the on-disk manifest for a direct/synthetic build never carries
a real `BoundaryQuestionCount` today. The comment directly above the new code
claims "the unanswered-question ... signal[] need[s] no manifest fields beyond
Phase," which is not true — `BoundaryQuestionCount` is exactly the manifest field
this signal is defined against (`cmd/build_blocker_advisory.go:45`), and it's
simply never populated on this lane.

This contradicts the surrounding comment's own claim ("both lanes call
buildStartBlockerSignals and decideBuildBlockerAdvisory identically") and CLAUDE.md's
explicit rule: "if the direct path and the chat path can disagree, both get proved."
`TestBothBuildLanesEmitTheHeadsUp` (`cmd/build_blocker_advisory_test.go`) only proves
parity for the forced-reviewer signal on both lanes — the boundary-question case is
never exercised end-to-end on the direct lane, so this gap shipped untested.

**Fix:** Either (a) call `materializeOrchestratorBoundaryQuestions("build", ...)`
(read-only check, not the materializing/creating call) on the direct dispatch lane
before building `directBuildManifest`, mirroring the plan-only path, or (b) if
boundary questions are structurally impossible on this lane for a documented
reason, say so explicitly in the comment instead of asserting parity that doesn't
exist, and add a test that actually exercises a boundary-question fixture on the
direct/`--synthetic` lane (the equivalent of `TestBothBuildLanesEmitTheHeadsUp` but
for `BoundaryQuestionCount`, not just `ForcedReviewers`).

### WR-02: `sealConfirmationInput.Force` is dead code that contradicts its own doc comment

**File:** `cmd/seal_confirmation.go:190-197`

**Issue:** The field is documented as being "carried through for the decision's
own bookkeeping/Why text," but no caller ever sets it:
`runSealConfirmationGate` (`cmd/seal_confirmation.go:301-329`, called from both
`sealCmd`'s `RunE` and `runSealFinalize`) never populates `Force` on the
`sealConfirmationInput` it builds, and `decideSealConfirmation`
(`cmd/seal_confirmation.go:236-271`) never reads `input.Force` anywhere in its
body. The result is a struct field that exists purely as unread/unwritten
scaffolding, and a doc comment promising behavior the code never implements.

**Fix:** Either wire `Force`/`ForceReason` into the `Why` text of
`sealConfirmationDecision` as documented (so a force-sealed-but-still-asked case
says why in the recorded decision trail), or remove the unused field and its
now-inaccurate comment.

### WR-03: Verification-step display name can panic on a non-ASCII check name

**File:** `cmd/codex_continue.go:3343-3350` (`verificationStepDisplayName`)

**Issue:** The fallback branch does `strings.ToUpper(trimmed[:1]) + trimmed[1:]`
on an arbitrary, CLAUDE.md-configured check name. `trimmed[:1]` is a byte slice,
not a rune slice; if a project ever configures (or a future check key is named)
a check whose first character is a multi-byte UTF-8 rune, this panics with an
invalid-UTF-8/index-out-of-range style failure rather than degrading gracefully.
The four known keys today (`build`, `types`, `lint`, `tests`) are all ASCII, so
this is latent rather than currently triggered, but the function's whole purpose
is to safely label arbitrary external input.

**Fix:** Use `[]rune(trimmed)` (or `unicode/utf8`) to take the first rune safely
before upper-casing it, the same defensive pattern this codebase already uses
elsewhere for user-controlled display strings.

---

_Reviewed: 2026-08-29_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
