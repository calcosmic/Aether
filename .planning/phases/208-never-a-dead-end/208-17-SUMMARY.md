---
phase: 208-never-a-dead-end
plan: 17
subsystem: colonize-territory-lifecycle
tags: [go, ast-guard, mutation-testing, colonize, territory-snapshot]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 01-16, especially 14)
    provides: "bindTransactionalTerritoryPublication (cmd/codex_colonize.go),
      the transactional finalize lane and its relaxed
      lighter-than-full-team tolerance (cmd/codex_colonize_finalize.go), and
      the two structural one-decision AST guards this plan widens/proves
      (TestSavedMapPublicationHasOneBuilder,
      cmd/colonize_snapshot_refresh_test.go)"
provides:
  - "TestSavedMapPublicationHasOneBuilder now catches a composite struct
    literal setting codexColonizeManifest.PublicationMode, not only an
    assignment statement -- the exact shape 208-REVIEW-GAP3.md's CR-01 used
    to defeat the original guard -- and states honestly, in its own doc
    comment, the four ways it still cannot see a second writer (WR-04)."
  - "TestALighterSurveyTeamStillPublishesTheSavedMap
    (cmd/colonize_snapshot_refresh_test.go): the missing executable proof
    for 208-14-PLAN.md's D4 must-have and 208-REVIEW-GAP3.md's WR-02 --
    drives a real light-depth colonize (two surveyors, four of seven
    required documents) through the plan-only manifest, completion packet
    and finalize path, and proves runTransactionalColonizeFinalize's
    relaxed branch still succeeds and publishes a valid saved map."
  - "Two new test helpers: lightDepthColonizeState (a light-verification-depth
    colony.ColonyState fixture) and readColonizeActivityLogDetails (reads
    .aether/data/activity.log and returns the details strings for a given
    command)."
affects: [colonize, territory-lifecycle, survey]

# Actuals (#2632)
actuals:
  tokens: 3329
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "AST structural guards widened to catch both *ast.AssignStmt and
      *ast.CompositeLit/*ast.KeyValueExpr shapes of the same field write,
      never narrowed to a struct type name (a literal's type can be
      inferred from context)."
    - "Mutation proofs for AST/behavioural guards are applied and reverted
      inside a disposable git worktree created under the session
      scratchpad with a path name containing \"Aether\" (this repo's
      TestResolveAetherRoot_GitFallback constraint), never in the working
      checkout -- test-file edits are copied in from the checkout, runtime
      mutations are planted and reverted only inside the worktree, and the
      worktree is removed with `git worktree remove --force`."

key-files:
  created: []
  modified:
    - cmd/colonize_snapshot_refresh_test.go

key-decisions:
  - "The composite-literal AST case is deliberately not narrowed to a
    particular struct type name, because a literal can be written with its
    type inferred from context (inside a slice, a map, or a nested field);
    requiring the type name would reintroduce a hole of exactly the shape
    this widening exists to close."
  - "The new survey-team test derives every number it needs at runtime
    (the light roster from queenSurveyorSpecsForState, the declared-output
    count from the manifest itself, the required count from
    requiredSurveyMarkdownFiles, the revision from `git rev-parse HEAD`)
    rather than typing any of them as literals, and refuses to pass if the
    light roster ever stops being strictly smaller than the required set --
    so it cannot silently degrade into re-proving the full-roster path."

patterns-established:
  - "A structural one-decision/one-builder AST guard's doc comment must
    carry an explicit 'what this still cannot catch' paragraph, stated
    honestly rather than claimed away, matching the standard
    TestSelfRecoveryHasOneDecision already set in this same round."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "The one-builder AST guard (TestSavedMapPublicationHasOneBuilder) now catches a composite struct literal setting PublicationMode, not only an assignment statement, and its doc comment says what it still cannot see."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/colonize_snapshot_refresh_test.go#TestSavedMapPublicationHasOneBuilder"
        status: pass
    human_judgment: false
  - id: D2
    description: "A genuinely lighter-than-full survey team (two surveyors, four of seven required documents) still finishes and publishes the saved map on the transactional finalize lane -- proved by a new test that reaches the relaxed branch through the real command path, where nothing did before."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/colonize_snapshot_refresh_test.go#TestALighterSurveyTeamStillPublishesTheSavedMap"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-24
status: complete
---

# Phase 208 Plan 17: Close the Third Gap-Closure Review's Saved-Map Findings Summary

**Widened the one-builder AST guard to catch struct-literal writes (not just assignments), gave it an honest "what this can't see" note, and added the missing test proving a lighter-than-full survey team still publishes the saved map on the real transactional finalize path -- each claim proved by planting the exact mutation that used to slip through, watching the guard go red, then restoring it.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 2 completed
- **Files modified:** 1 (`cmd/colonize_snapshot_refresh_test.go`)

## What this closes, in plain English

Aether keeps a saved picture of your code (a "territory snapshot") that later commands read to
decide whether a fresh survey is needed. One test in this repo's own test suite exists purely to
make sure only one place in the program is ever allowed to decide which "lane" (fast/transactional
vs. legacy) that saved picture gets rewritten through -- because if a second place could set that
switch too, two different pieces of code could quietly disagree about how the saved picture gets
written, which is exactly the kind of bug this whole phase (208, "Never a Dead End") exists to stop.

A review two rounds ago found that this "only one place can decide" test had a blind spot: it could
see one of the two ordinary ways a Go programmer sets a struct field (`x.Field = value`), but not
the other, more common way (`StructName{Field: value}`). That's like a smoke detector that only
notices smoke coming from one direction. Task 1 fixes that: the check now looks both ways, and its
own comment now honestly lists the (much rarer, currently-unused) tricks that could still slip past
it, rather than implying it catches everything.

Separately, a previous round of work had written down, in a document, that "a smaller survey team
still finishes the job successfully" -- but nobody had ever actually built a test that ran a smaller
survey team through the real command and checked that claim. It was a sentence, not a check. Task 2
builds that check: it runs a genuinely smaller survey (the kind Aether sends when you pick a
"light" survey depth) all the way through the real save-and-publish process and confirms it still
works, still records the right note in Aether's own activity log, and the resulting saved documents
still pass Aether's own "this isn't empty filler text" check.

Both fixes were proven the hard way: I deliberately broke each thing the check was supposed to
catch, watched the check correctly notice and fail, then put everything back and confirmed it
passed again -- all inside a disposable, throwaway copy of the code so the real project was never
at risk.

## Task Commits

1. **Task 1: The one-builder check sees both ways of setting the field, and says what it still cannot see** - `e5b22adf` (test)
2. **Task 2: A smaller survey team still finishes — proved on the path that actually changed** - `37a6be75` (test)

## Files Created/Modified

- `cmd/colonize_snapshot_refresh_test.go` — widened `TestSavedMapPublicationHasOneBuilder`'s AST
  walk to catch composite-literal writes to `PublicationMode` in addition to assignment statements;
  extended its doc comment with an honest "what this still cannot catch" section; added
  `TestALighterSurveyTeamStillPublishesTheSavedMap`, `lightDepthColonizeState`, and
  `readColonizeActivityLogDetails`.

## Decisions Made

- See `key-decisions` in the frontmatter: the composite-literal case is intentionally not narrowed
  to a struct type name, and the new survey-team test derives every number it asserts on at runtime
  rather than typing any literal.

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched their acceptance criteria without
requiring any Rule 1-4 auto-fixes or architectural questions.

## Mutation Proofs (verbatim observed output)

All mutations were planted and reverted inside disposable git worktrees created under the session
scratchpad at paths containing "Aether" (`Aether-mutation-proof-t1`, `Aether-mutation-proof-t2`),
never in this checkout, and removed afterward with `git worktree remove --force`. `git status
--short` in the owner's checkout showed only `cmd/colonize_snapshot_refresh_test.go` changing
throughout (plus the pre-existing, untouched `.aether/CONTEXT.md` and `.planning/STATE.md` edits
from other sessions).

### Task 1 — CR-01 (composite-literal write)

Planted in the worktree's `cmd/codex_workflow_cmds.go` (not the sole builder file):

```go
func secondPublicationModeSetterCompositeLit() codexColonizeManifest {
	return codexColonizeManifest{PublicationMode: territoryPublicationTransactional}
}
```

`go build ./...` succeeded (the mutation is legal Go). `go test ./cmd -run
TestSavedMapPublicationHasOneBuilder -count=1 -v` then **FAILED**, verbatim:

```
colonize_snapshot_refresh_test.go:378: only cmd/codex_colonize.go may set PublicationMode -- the saved map's publication binding must be built in exactly one shared place; found a second builder in: [cmd/codex_workflow_cmds.go]
--- FAIL: TestSavedMapPublicationHasOneBuilder (0.47s)
```

With the function removed, the same command **PASSED**: `--- PASS: TestSavedMapPublicationHasOneBuilder (0.75s)`.

### Task 1 — no regression on the assignment shape 208-14 already covered

Planted in the same file:

```go
func secondPublicationModeSetterAssignment() codexColonizeManifest {
	var m codexColonizeManifest
	m.PublicationMode = territoryPublicationTransactional
	return m
}
```

`go build ./...` succeeded. The same test **FAILED**, verbatim, naming the same file:

```
colonize_snapshot_refresh_test.go:378: only cmd/codex_colonize.go may set PublicationMode -- the saved map's publication binding must be built in exactly one shared place; found a second builder in: [cmd/codex_workflow_cmds.go]
--- FAIL: TestSavedMapPublicationHasOneBuilder (0.36s)
```

With the function removed, it **PASSED**: `--- PASS: TestSavedMapPublicationHasOneBuilder (0.28s)`.

### Task 2 — WR-02 (the falsifiability proof)

Planted in the worktree's `cmd/codex_colonize_finalize.go`, restoring the strict pre-relaxation
behaviour (returning an error instead of calling `logActivity`):

```go
if preservedWorkerArtifacts < len(requiredSurveyMarkdownFiles) {
	return nil, fmt.Errorf("transactional territory refresh published only %d of %d required survey documents from worker claims", preservedWorkerArtifacts, len(requiredSurveyMarkdownFiles))
}
```

`go build ./...` succeeded. `go test ./cmd -run TestALighterSurveyTeamStillPublishesTheSavedMap
-count=1 -v` then **FAILED**, verbatim (the test's own `t.Fatalf`, which surfaced the planted
error's exact text):

```
colonize_snapshot_refresh_test.go:538: colonize-finalize returned error for a lighter-than-full survey team: command failed after rendering error output with code 1; stderr={"ok":false,"error":"transactional territory refresh published only 4 of 7 required survey documents from worker claims","code":1}
--- FAIL: TestALighterSurveyTeamStillPublishesTheSavedMap (0.16s)
```

### Task 2 — the coverage-gap proof (confirms WR-02's "zero executable coverage" finding was accurate)

Under that same planted mutation, `go test ./cmd -run TestColonizeRefreshesTheSavedMapsRevision
-count=1 -v` **PASSED** unchanged:

```
--- PASS: TestColonizeRefreshesTheSavedMapsRevision (2.24s)
```

This confirms the pre-existing test's full roster never takes the relaxed branch — the review's
"zero executable coverage" finding was accurate before this plan, and
`TestALighterSurveyTeamStillPublishesTheSavedMap` is genuinely the first thing that covers it.

With the mutation reverted, both tests **PASSED**:

```
--- PASS: TestColonizeRefreshesTheSavedMapsRevision (2.28s)
--- PASS: TestALighterSurveyTeamStillPublishesTheSavedMap (1.34s)
```

## Verification Performed

| Check | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./cmd/... ./pkg/...` | clean |
| `go test ./cmd -run 'TestSavedMapPublicationHasOneBuilder\|TestALighterSurveyTeamStillPublishesTheSavedMap\|TestColonizeRefreshesTheSavedMapsRevision' -count=1 -timeout 8m` | PASS |
| `go test ./cmd -run 'Colonize\|Territory' -count=1 -timeout 8m` | PASS (no failures at all; the known-red `TestCodexNativeCancellation` entries, WINDOWS row 56, are not matched by this regex and were not encountered) |
| `git status --short` after each mutation proof and after worktree removal | only `cmd/colonize_snapshot_refresh_test.go` changed in the owner's checkout |
| `git worktree list` at the end | names only the pre-existing worktrees (`Aether-release-1.0.86`, `.claude/worktrees/agent-af06650f4d83f8d05`) |

No paid walk was run and nothing was spent (no `make eval-gate-journey`, no `claude -p`, no
`AETHER_JOURNEY_TRIALS`), per this plan's prohibitions and D-07. No runtime (non-test) source file
was modified in the owner's checkout — every mutation lived only inside a disposable worktree and
was reverted before removal.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

This plan closes CR-01, WR-04, and WR-02 from `208-REVIEW-GAP3.md`. Two further findings from that
same review remain open and are explicitly out of scope for this plan (carried forward by design,
per the plan's own prohibitions): CR-02 (the self-recovery one-decision guard's allow-listed files
can still hide a second decision) and WR-03 (the notice's hardcoded "next" line would silently
misdescribe a future second refusal-recovery table row). WR-01 (a stale linked worktree not skipped
by `TestSelfRecoveryHasOneDecision`'s walk) is also still open. Per STATE.md, these are scoped to
this round's sibling plans (208-18/19), not this one.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-24*
