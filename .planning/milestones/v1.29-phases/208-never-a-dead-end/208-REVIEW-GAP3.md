---
phase: 208-never-a-dead-end
round: gap-closure-3 (plans 208-13, 208-14, 208-15, 208-16)
reviewed: 2026-09-24T00:00:00Z
depth: standard
diff_base: 0cc5fff0
files_reviewed: 7
files_reviewed_list:
  - cmd/refusal_self_recovery.go
  - cmd/refusal_self_recovery_test.go
  - cmd/codex_colonize.go
  - cmd/codex_colonize_finalize.go
  - cmd/codex_workflow_cmds.go
  - cmd/colonize_snapshot_refresh_test.go
  - cmd/testdata/refusals/untyped-floor.json
findings:
  critical: 2
  warning: 4
  info: 0
  total: 6
status: issues_found
---

# Phase 208 (plans 13, 14, 15, 16): Code Review Report — third gap-closure round

**Reviewed:** 2026-09-24
**Depth:** standard (plus targeted mutation testing)
**Files Reviewed:** 7
**Status:** issues_found

## Summary

This round set out to fix two things: make the second gap-closure review's two CRITICAL findings
(208-REVIEW-GAP2.md CR-01, CR-02, independently reproduced by 208-VERIFICATION.md) actually catch
the mutations they were named for, and diagnose/fix the fourth proximate cause of WINDOWS row 53
(the saved territory map's revision staying stale after a forced resurvey through colonize's own
plan-only path). Both of the round's stated goals are genuinely, verifiably done: I independently
reproduced every mutation this round's own summaries claim to have fixed, in a disposable git
worktree, and all of them now fail the right test for the right reason. GAP2's CR-01, CR-02, WR-02,
WR-03, WR-04, WR-07 and WR-08 are all confirmed fixed below with fresh, independent proof — not
trusted from the SUMMARY files' own claims.

The problem is that fixing GAP2's two guards introduced two new guards this round (one explicit,
`TestSavedMapPublicationHasOneBuilder`; one widened, `TestSelfRecoveryHasOneDecision`), and both of
those new/widened guards have their own mutation-provable blind spots, in the exact same failure
class CLAUDE.md's Definition of Done calls out by name — a guard whose whole job is to catch a
second, independent decision or writer, that a second, independent decision or writer can still
evade. I built a disposable worktree, planted each mutation, and both guards passed unchanged when
they should have failed. Neither is a contrived shape: one is the single most idiomatic way to
construct a Go struct (a composite literal instead of a field assignment), and the other is placing
a second decision inside one of the three files the fix's own allow-list now exempts wholesale,
rather than a fourth file.

Two further warnings concern proof gaps and a coupling landmine, not guard defeats: the
`runTransactionalColonizeFinalize` relaxation's own stated justification (a lighter-than-full
surveyor team still succeeds) has zero executable test coverage on the actual changed code path,
and the notice renderer's hardcoded "rebuilding the map of your code" clause silently reintroduces
WR-02's exact failure class the moment a second table entry is ever added.

No source file was modified by this review. Both mutation worktrees were created under disposable
scratch paths and removed with `git worktree remove --force`; `git status` in the main checkout is
unchanged from before the review (`M .aether/CONTEXT.md` only, pre-existing).

## Critical Issues

### CR-01: `TestSavedMapPublicationHasOneBuilder`'s AST walk only catches assignment statements, not composite literals

**File:** `cmd/colonize_snapshot_refresh_test.go:319-332`

**Issue:** The structural guard is supposed to fail by name "if a second, independent call site
anywhere in the module sets the lane-selecting field (`codexColonizeManifest.PublicationMode`)
directly instead of going through" `bindTransactionalTerritoryPublication`
(208-14-PLAN.md's must_have D3). Its AST walk only inspects `*ast.AssignStmt` nodes whose LHS is a
`*ast.SelectorExpr` named `PublicationMode`:

```go
ast.Inspect(file, func(n ast.Node) bool {
    assign, ok := n.(*ast.AssignStmt)
    if !ok {
        return true
    }
    for _, lhs := range assign.Lhs {
        sel, ok := lhs.(*ast.SelectorExpr)
        if ok && sel.Sel.Name == "PublicationMode" {
            setsPublicationMode = true
        }
    }
    return true
})
```

A composite struct literal sets the same field through a `*ast.CompositeLit` /
`*ast.KeyValueExpr`, a node shape this walk never visits — and it is the more idiomatic of the two
ways to construct the struct in Go. I proved this in a disposable worktree by adding, to
`cmd/codex_workflow_cmds.go` (a file that is not the sole builder), a function that never assigns
the field at all:

```go
func secondPublicationModeSetterCompositeLit() codexColonizeManifest {
    return codexColonizeManifest{PublicationMode: territoryPublicationTransactional}
}
```

`go build ./...` succeeds and `go test ./cmd -run TestSavedMapPublicationHasOneBuilder -count=1 -v`
→ **PASS** unchanged. This is precisely the "second, independent builder" scenario the must-have
exists to catch, expressed in the single most natural alternative Go syntax for the exact same
effect.

**Fix:** walk `*ast.CompositeLit` nodes too, and flag any `*ast.KeyValueExpr` whose key is an
`*ast.Ident` named `PublicationMode`:

```go
case *ast.CompositeLit:
    for _, elt := range node.Elts {
        if kv, ok := elt.(*ast.KeyValueExpr); ok {
            if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "PublicationMode" {
                setsPublicationMode = true
            }
        }
    }
```

Then re-run the reproduction above against the widened guard and confirm it now fails, before
restoring.

### CR-02: The widened one-decision guard's fix for GAP2's CR-02 exempts three whole files, not just their existing call sites — a second decision can still hide inside one of them

**File:** `cmd/refusal_self_recovery_test.go:342-350, 419-424`

**Issue:** 208-13's fix for GAP2's CR-02 (a second decision in `cmd/helpers.go`'s `outputRefusal`
sailing through the old guard undetected) is a closed allow-list,
`mayReadTheIsAnyoneHereFact = {"cmd/unattended_session.go", "cmd/refusal.go",
"cmd/refusal_self_recovery.go"}`. The walk now skips ALL THREE of those files entirely:

```go
if mayReadTheIsAnyoneHereFact[rel] {
    continue
}
```

`cmd/refusal.go` is on that list for a real, narrow, already-reviewed reason: `Error()` and
`renderRefusal` both call `sessionHasNoOneToAsk()` to decide whether to append the shared guidance
sentence (208-09-PLAN.md). But because the whole file is exempted rather than only those two
call sites, nothing stops a second, competing self-recovery decision from being added anywhere
else inside that same file. I proved this in a disposable worktree by appending, to the end of
`cmd/refusal.go`:

```go
func roguesSecondDecisionInsideAllowlistedFile(r refusal) bool {
    if !sessionHasNoOneToAsk() {
        return false
    }
    if !r.ProtectsWork || strings.TrimSpace(r.NextCommand) == "" {
        return false
    }
    appendRecoveredRefusalToLog(r)
    return true
}
```

`go build ./...` succeeds and `go test ./cmd -run TestSelfRecoveryHasOneDecision -count=1 -v` →
**PASS** unchanged. This is the exact same failure shape GAP2's CR-02 named ("a second, competing
self-recovery decision... discards the refusal path... in an unattended session") — it has simply
moved from a fourth file (`cmd/helpers.go`, now caught) into one of the three files the fix's own
allow-list now exempts wholesale. The widened guard closed the specific reproduction 208-13-PLAN.md
was handed, but not the general property its own must-have states ("a second copy anywhere in the
module fails a named test").

**Fix:** narrow the allow-list from "may not read the fact at all outside these files" to "may
read the fact only at these specific, already-reviewed call sites" — e.g. require the read to occur
inside one of a small, named set of function identifiers (`Error`, `renderRefusal`,
`attemptRefusalSelfRecovery`, `sessionHasNoOneToAsk` itself), rather than exempting the containing
file unconditionally. At minimum, the doc comment's "what this cannot catch" section should say so
honestly, the same way `TestSelfRecoveryHasOneDecision`'s own comment already discloses the
hard-coded-`true` gap; today it does not mention this one at all.

## Warnings

### WR-01: `TestSelfRecoveryHasOneDecision`'s whole-module walk does not skip a stale linked worktree, unlike its sibling guard added in this same round

**File:** `cmd/refusal_self_recovery_test.go:388-394`

**Issue:** The walk's `SkipDir` list is `".git", "vendor", "node_modules", "testdata", ".planning",
"dist"` — no `"worktrees"`. `cmd/colonize_snapshot_refresh_test.go`'s new
`TestSavedMapPublicationHasOneBuilder`, added in this very round, has an identical walk but *does*
skip `"worktrees"`, because 208-14-SUMMARY.md documents hitting exactly this problem: "the first
attempt at `TestSavedMapPublicationHasOneBuilder` failed *before any mutation was planted*, because
the walk reached into a stale, unrelated linked worktree checked out inside this repository."

That stale worktree still exists in this checkout right now
(`.claude/worktrees/agent-af06650f4d83f8d05`, pinned to commit `a8b96efe`, 1321 `.go` files) and
`TestSelfRecoveryHasOneDecision`'s walk does not skip it — confirmed by running the test with the
worktree present (it passes only because that old commit's `cmd/helpers.go` and
`cmd/refusal_self_recovery.go` happen not to trip the guard, which is luck, not design). A future
in-progress linked worktree touching either of the guard's two subject files would be parsed and
could produce a spurious failure naming a path that is not part of the tree anyone is actually
reviewing, or — depending on what that worktree's code looks like — mask a genuine offender behind
noise from an unrelated stale checkout.

**Fix:** add `"worktrees"` to this walk's `SkipDir` list too, matching the fix already applied to
its sibling in the same round and to the pre-existing pattern in
`cmd/build_attempt_external_test.go`.

### WR-02: The transactional-finalize relaxation's own justification (a lighter survey team still succeeds) has no executable proof on the code path it changed

**File:** `cmd/codex_colonize_finalize.go:482-495`; verification claim in 208-14-PLAN.md's D4

**Issue:** 208-14's stated must-have D4 is: "a forced resurvey with a lighter-than-full surveyor
team still succeeds (the transactional lane's fallback now matches the legacy lane's)." Its own
verification method is re-running a list of pre-existing tests
(`TestColonizeFinalizeAllowsFirstTimeWorkerWrittenSurveyArtifacts` among them) plus code reading.

`grep -rn "preservedWorkerArtifacts|runTransactionalColonizeFinalize" cmd/*_test.go` returns
**nothing** — no test file anywhere references either identifier. The two candidates that could
plausibly cover this:

- `TestColonizeFinalizeAllowsFirstTimeWorkerWrittenSurveyArtifacts` drives a manifest built by
  plain `runCodexColonizePlanOnly` with no `ForceResurvey`+`existingSurvey` combination, so
  `PublicationMode` is never set — it exercises only the **legacy** lane's pre-existing tolerance,
  not the transactional branch this plan touched.
- The new `TestColonizeRefreshesTheSavedMapsRevision` (`cmd/colonize_snapshot_refresh_test.go`)
  does exercise the transactional lane, but
  `completeColonizeSnapshotRefreshDispatches` writes real content at every one of every dispatch's
  declared output paths, so `preservedWorkerArtifacts` is always the full count there — the
  relaxed branch (`preservedWorkerArtifacts < len(requiredSurveyMarkdownFiles)`) is never taken.

This is a real, reachable production path, not a hypothetical: `queenSurveyorSpecsForState`'s own
comment states "light = structure + dependency surveyors only" — a light-depth colony's forced
resurvey genuinely dispatches fewer surveyors than the seven required documents, which is exactly
the scenario the relaxation exists for. Nothing currently proves that scenario still succeeds on
the transactional lane, or that `logActivity`'s message fires with the right numbers, or that
`validateTerritoryPublicationMarkdown`'s placeholder check still passes the synthesized fallback
content on this lane specifically.

**Fix:** add a test that drives `runTransactionalColonizeFinalize` (through the real
plan-only-manifest → completion-packet → finalize path, per this round's own established pattern in
`TestColonizeRefreshesTheSavedMapsRevision`) with genuinely fewer dispatches than required
documents, and asserts the finalize succeeds, the activity log carries the expected note, and the
published snapshot's content still passes validation.

### WR-03: The notice's "next" line hardcodes colonize-specific wording that will silently misdescribe a future second table entry

**File:** `cmd/refusal_self_recovery.go:132-142`

**Issue:** `renderRefusalSelfRecoveryNotice`'s reworded "next" line is now a bare literal, not
templated from the table's per-row `reason` the way the "question" line above it is:

```go
b.WriteString(voiceLine("next", "Aether is going ahead and rebuilding the map of your code instead of stopping to ask (`"+next+"`)."))
```

Today this is accurate because the one entry in `refusalSelfRecoveryTable` is exactly a resurvey.
The table's own doc comment says "Nothing else goes in this map without its own owner ruling," but
nothing ties that future ruling to also revisiting this hardcoded clause — a reviewer adding a
second, unrelated opt-in row (say, a different work-protecting stop whose safe recovery is not
about rebuilding a map at all) would see `reason` update correctly on the line above, and this line
keep silently claiming "rebuilding the map of your code" regardless of what the new row's recovery
actually does. That is WR-02's exact failure class (the notice tells the owner something untrue
about what Aether is doing) reintroduced one clause later, and neither
`TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` nor
`TestSelfRecoveryNoticeSpeaksPlainEnglish` would catch it, since both are scoped to today's one row.

**Fix:** parameterize the "next" line the same way the "question" line is — either fold its content
into the table's own reason string, or add a second per-row field for "what Aether is going ahead
and doing" so a future row cannot be added without also supplying its own accurate action clause.

### WR-04: `TestSavedMapPublicationHasOneBuilder` discloses no known limitation, unlike its sibling guard in the same round

**File:** `cmd/colonize_snapshot_refresh_test.go:259-268`

**Issue:** `TestSelfRecoveryHasOneDecision`'s doc comment (in the same round, same package) is
explicit about what it cannot catch: "a second decision expressed without ever naming
refusalSelfRecoveryTable or calling sessionHasNoOneToAsk." `TestSavedMapPublicationHasOneBuilder`'s
doc comment makes no equivalent disclosure — it simply asserts the guard "fails by name if a
second, independent call site anywhere in the module sets the lane-selecting field... directly."
Given CR-01 above shows this claim is not true for a composite-literal write, the missing honesty
disclosure compounds the problem: nothing in the test file itself would have prompted a future
reader to notice the gap the way the sibling guard's own comment models good practice for.

**Fix:** once CR-01 is fixed, keep an honest "what this still cannot catch" note in this guard's
doc comment too, matching the standard the sibling guard in this same round already sets.

## Resolution of the second gap-closure round's own findings

Independently re-verified against current HEAD, by reproducing each original mutation in a
disposable worktree rather than trusting the SUMMARY files' own claims:

| Finding | Status | How verified |
|---|---|---|
| GAP2 CR-01 (three of four gates unenforced) | **FIXED** | Reproduced the exact three-gate deletion from 208-VERIFICATION.md against current HEAD; `TestOnlyADeclaredRefusalIsEverRecoveredFrom` now fails naming the undeclared id. Also independently confirmed the narrower single-gate deletions (opt-in lookup alone; the WR-08 authoritative re-check alone) each fail the same test naming the specific broken case. |
| GAP2 CR-02 (second decision in `outputRefusal` undetected) | **FIXED for the reported shape** — but see new **CR-02** above | Reproduced `secondSelfRecoveryDecision` in `cmd/helpers.go`'s `outputRefusal` verbatim; `TestSelfRecoveryHasOneDecision` now fails naming `cmd/helpers.go`. The widened guard's own new blind spot (a decision placed inside one of the three now-exempted files) is a fresh finding, not a re-opening of the original one. |
| WR-02 (notice claims a command already ran) | **FIXED** | Reproduced the old past-tense wording; `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` fails naming the false claim. See new WR-03 above for a narrower, related residual risk. |
| WR-03 (planning-decision id/filename on screen) | **FIXED** | Reproduced the old table value carrying `(D-03, 208-CONTEXT.md)`; `TestSelfRecoveryNoticeSpeaksPlainEnglish` fails naming the identifier. |
| WR-04 (attended proof depends on unset env var) | **FIXED** | Both `TestAttendedColonizeStillStopsAndAsks` subtests now call `t.Setenv(unattendedEnvVar, "")` explicitly; confirmed by reading the diff — no longer relies on the variable being ambiently absent. |
| WR-07 (visual assertion doesn't anchor on the notice itself) | **FIXED** | Reproduced a renderer returning only the bare command string; the widened assertion now fails naming the missing banner heading. |
| WR-08 (gate trusts caller's fields over the registry) | **FIXED** | `attemptRefusalSelfRecovery` now re-reads `refusalForID(r.ID)` and both the gate and the notice's command use that authoritative row, confirmed by reading the diff and by the passing `TestOnlyADeclaredRefusalIsEverRecoveredFrom` case that forces caller/registry disagreement. |
| WR-06 (key link proven only at its first hop) | **Closed as a side effect of 208-14**, not this round's stated target | `TestColonizeRefreshesTheSavedMapsRevision` now drives the self-recovered, forced-resurvey manifest all the way through the real `colonize-finalize` command and asserts it succeeds without hitting the sibling `colonize-finalize-existing-survey-found` refusal — exactly the chain WR-06 asked to be extended one hop further. |
| WR-01 (decision announces/records without itself performing the recovery) | **Still open, honestly carried forward** | 208-13-SUMMARY.md explicitly scopes this out; not re-litigated here as a new finding. |
| WR-05 (silent recovery on a future quiet-classified command) | **Still open, honestly carried forward** | Same as above; today's one row (`colonize`) remains safe because `colonize` is not quiet-classified. |

## Verification performed

| Check | Result |
|---|---|
| Full targeted test set, clean checkout (`TestOnlyADeclaredRefusalIsEverRecoveredFrom`, `TestSelfRecoveryHasOneDecision`, `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened`, `TestSelfRecoveryNoticeSpeaksPlainEnglish`, `TestNoOneHereMeansAetherRefreshesTheMapItself`, `TestAttendedColonizeStillStopsAndAsks`, `TestUnattendedDirectColonizeRefreshesTheMapItself`, `TestOnlyASafeRefusalCanRecoverItself`, `TestOldShapedRefusalLogRecordStillReadsAsNotRecovered`) | all PASS (3.7s) |
| Mutation: reproduce GAP2 CR-01 (delete opt-in/ProtectsWork/NextCommand gates) | **caught** — `TestOnlyADeclaredRefusalIsEverRecoveredFrom` fails naming the undeclared id |
| Mutation: reproduce GAP2 CR-01 narrower variant (opt-in lookup alone) | **caught** — same test fails, same message |
| Mutation: reproduce GAP2 CR-02 (`secondSelfRecoveryDecision` in `cmd/helpers.go`) | **caught** — `TestSelfRecoveryHasOneDecision` fails naming `cmd/helpers.go` |
| Mutation: WR-02 old wording restored | **caught** — `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` fails |
| Mutation: WR-03 old `D-03` table value restored | **caught** — `TestSelfRecoveryNoticeSpeaksPlainEnglish` fails |
| Mutation: WR-07 renderer reduced to bare command string | **caught** — visual subtest fails naming the missing banner heading |
| Mutation: composite-literal `PublicationMode` write in a third file | **NOT caught** — `TestSavedMapPublicationHasOneBuilder` passes unchanged (confirms new CR-01) |
| Mutation: second self-recovery decision planted inside `cmd/refusal.go` (an allow-listed file) | **NOT caught** — `TestSelfRecoveryHasOneDecision` passes unchanged (confirms new CR-02) |
| `TestEveryRefusalSiteIsRegisteredOrCounted` + `TestUntypedRefusalFloorOnlyShrinks` (untyped-floor.json 386→385) | both PASS — confirms the floor edit is exact and tied to the one removed `fmt.Errorf`, not masking anything else |
| `go build ./...`, `go vet ./cmd/... ./pkg/...` | clean |
| `go test ./cmd -run 'Colonize\|Territory\|Refusal\|Refuse\|Unattended' -count=1 -timeout 20m` | only the two pre-existing, already-catalogued `TestCodexNativeCancellation*` failures (WINDOWS row 56); no new regressions |
| `git status` after review | unchanged (`M .aether/CONTEXT.md`, pre-existing); both mutation worktrees removed |

---

_Reviewed: 2026-09-24_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard + mutation testing_
