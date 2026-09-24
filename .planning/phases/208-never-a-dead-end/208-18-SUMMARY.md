---
phase: 208-never-a-dead-end
plan: 18
subsystem: refusal-self-recovery
tags: [go, ast-guard, mutation-testing, refusal, self-recovery, unattended-session]

# Dependency graph
requires:
  - phase: 208-never-a-dead-end (plans 01-17, especially 09, 11, 13)
    provides: "sessionHasNoOneToAsk (cmd/unattended_session.go), the refusal
      contract (cmd/refusal.go), and the one-decision self-recovery gate
      this plan narrows (attemptRefusalSelfRecovery,
      renderRefusalSelfRecoveryNotice, TestSelfRecoveryHasOneDecision, all
      in cmd/refusal_self_recovery.go / cmd/refusal_self_recovery_test.go)"
provides:
  - "TestSelfRecoveryHasOneDecision now excuses five specific, already-
    reviewed declarations (file::function keys in
    mayHoldASelfRecoveryDecision) instead of three whole files, so a rival
    self-recovery decision planted anywhere else inside cmd/refusal.go or
    cmd/unattended_session.go is caught by name -- closing CR-02 from
    208-REVIEW-GAP3.md, the third consecutive round this exact defect class
    (a guard's blind spot moving rather than closing) had survived."
  - "The same guard's walk now skips a 'worktrees' directory at any depth,
    matching its sibling in cmd/colonize_snapshot_refresh_test.go, so it no
    longer parses the leftover linked worktree already checked out inside
    this repository (WR-01)."
  - "refusalSelfRecoveryTable rows now carry two owner-facing fields --
    Reason and Action -- instead of one string. The self-recovery notice's
    closing sentence is composed from the row's own Action rather than a
    fixed phrase baked into the renderer, so a future second row cannot
    silently misdescribe what it is doing (WR-03). A row missing either
    field is refused by refusalSelfRecoveryContractProblems."
  - "TestEveryRecoveryRowSuppliesItsOwnActionWording: proves every real
    row's Reason/Action are both present, distinct, and reach the rendered
    screen, and proves -- against a locally built second row -- that the
    real colonize row's Action cannot leak into a different row's notice."
affects: [refusal-self-recovery, colonize, unattended-session]

# Actuals (#2632)
actuals:
  tokens: 6741
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "A structural excuse list keyed by '<repo-relative file>::<enclosing
      declaration name>' rather than by file alone, so excusing a reviewed
      place never re-excuses everything else in the same file. Built by
      iterating file.Decls, taking a *ast.FuncDecl's own Name.Name or a
      *ast.GenDecl spec's own first declared name as the enclosing
      identifier, then walking only that declaration's subtree for the
      identifiers of interest."
    - "An excuse list proves it cannot rot silently: after the walk, every
      key in the allow-list that was never observed fails the test by name
      as a stale excuse, so the list can only shrink by the place genuinely
      going away, never sit unnoticed once code moves out from under it."
    - "An owner-facing notice's per-row wording lives on the row itself
      (two named fields, both printed verbatim) rather than partly on the
      row and partly hardcoded in the renderer -- so a second row cannot
      silently inherit the first row's description of what it does."
    - "Mutation proofs for AST/behavioural guards are applied and reverted
      inside a disposable git worktree created under the session
      scratchpad at a path whose name contains \"Aether\", never in the
      working checkout -- edited test/source files are copied in from the
      checkout, runtime mutations are planted and reverted only inside the
      worktree by re-copying the checkout's own file, and the worktree is
      removed with `git worktree remove --force`."

key-files:
  created: []
  modified:
    - cmd/refusal_self_recovery.go
    - cmd/refusal_self_recovery_test.go

key-decisions:
  - "mayHoldASelfRecoveryDecision's five keys are exactly the ones the plan
    named: cmd/unattended_session.go::sessionHasNoOneToAsk (the fact's own
    definition), cmd/refusal.go::Error and cmd/refusal.go::renderRefusal
    (the two places that append the shared guidance sentence), and
    cmd/refusal_self_recovery.go::refusalSelfRecoveryTable and
    ::attemptRefusalSelfRecovery (the table's own declaration and the one
    decision). No entry was added or removed beyond that list."
  - "The walk attributes a package-level var/const/type declaration's own
    identifier occurrence to itself (a ValueSpec's Names[0] read by
    ast.Inspect over the spec node) -- this is what makes
    refusalSelfRecoveryTable's own declaration count as a read at its own
    name, exactly as the plan specifies, rather than needing a special
    case."
  - "The colonize row's Action field is the exact clause the renderer used
    to hardcode, moved word for word: 'rebuilding the map of your code'.
    Verified character-for-character below."
  - "TestEveryRecoveryRowSuppliesItsOwnActionWording's second part builds a
    locally-constructed second row (Action: 'arranging a saved copy of your
    work before carrying on') and asserts the real colonize row's own
    Action -- read from the real table at assertion time, never typed as a
    literal -- does not appear in a notice rendered for the different row.
    This is the exact falsifiability proof WR-03 asked for."

patterns-established:
  - "An excuse/allow list guard states, in its own doc comment, both what it
    still cannot catch (inherited from prior rounds) and why keying by
    place rather than file is what closes the specific defect class this
    round fixed -- so a future reader does not have to re-derive the
    reasoning from git history."

requirements-completed: [UED-10]

coverage:
  - id: D1
    description: "TestSelfRecoveryHasOneDecision excuses five specific, already-reviewed declarations (not three whole files), fails naming both file and enclosing declaration on any other read, fails if a listed excuse goes stale, and skips a leftover linked worktree the way its sibling guard already does."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestSelfRecoveryHasOneDecision"
        status: pass
    human_judgment: false
  - id: D2
    description: "The self-recovery notice's closing sentence is composed from the table row's own Action field, not a fixed phrase in the renderer; a row missing its own Reason or Action is refused by the contract checker; what the owner reads today is unchanged."
    requirement: "UED-10"
    verification:
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestEveryRecoveryRowSuppliesItsOwnActionWording"
        status: pass
      - kind: unit
        ref: "cmd/refusal_self_recovery_test.go#TestOnlyASafeRefusalCanRecoverItself"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-09-24
status: complete
---

# Phase 208 Plan 18: Close the Fourth Gap-Closure Round's Self-Recovery Findings Summary

**Narrowed the self-recovery one-decision guard's excuse list from three whole files to the five specific already-reviewed places a rival decision could hide in, added the missing skip for a leftover linked worktree, and moved the self-recovery notice's hardcoded closing clause onto the table row that fires it -- each claim proved by planting the exact mutation it is named for in a disposable worktree, watching the guard go red, then restoring it.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 2 completed
- **Files modified:** 2 (`cmd/refusal_self_recovery.go`, `cmd/refusal_self_recovery_test.go`)

## What this closes, in plain English

When nobody is present to answer a question Aether would otherwise stop and ask, one function in
this program (`attemptRefusalSelfRecovery`) is allowed to decide to carry on by itself instead of
waiting. There is exactly one test in this repository whose whole job is to make sure that decision
is never made a second time, somewhere else in the program, by accident — because two different
places quietly deciding the same thing, possibly differently, is exactly the kind of bug this whole
phase ("Never a Dead End") exists to prevent.

Two rounds ago, a code review found that a rival decision could be planted in a file the test never
looked at, and slip through unnoticed. The fix at the time was to make the test skip three whole
files it considered "already reviewed" rather than catching a rival decision anywhere in the
program. This round's own review (four rounds deep now) found the flaw in that fix: skipping a
whole file means a rival decision hidden inside a *different function* in one of those same three
files also slips through unnoticed — the blind spot moved, it did not close. Task 1 fixes this
properly: instead of "these three files are exempt," the test now says "exactly these five specific,
already-read functions are exempt" — and if a rival decision shows up anywhere else in those same
files, the test names both the file and the exact function it hid in. The test also now ignores a
leftover, stale copy of this repository that happens to be checked out inside itself (this actually
exists in this checkout right now, at `.claude/worktrees/agent-af06650f4d83f8d05`) — without that
fix, the test could have been reading old code from that stale copy instead of the real thing.

Separately, the screen Aether shows when it carries on without you ends with a sentence describing
what it's doing — today, always "rebuilding the map of your code." That sentence used to be typed
directly into the code that draws the screen, so if a second kind of self-recovery were ever added
later, the screen would keep saying "rebuilding the map of your code" even if the new recovery did
something completely different. Task 2 fixes this by moving that sentence onto the specific entry
in Aether's own table of allowed recoveries, so each entry has to supply its own accurate
description — and a new entry that forgets to supply one is refused outright, rather than silently
inheriting the old sentence. What you read on screen today has not changed by a single character;
it is only now impossible for a second entry to lie about what it is doing.

Both fixes were proven the hard way: I deliberately planted the exact bugs each guard is supposed to
catch, watched the guard correctly notice and fail, then reverted everything and confirmed it passed
again — all inside a disposable, throwaway copy of the code so the real project was never at risk.

## Task Commits

1. **Task 1: The one-decision check excuses five reviewed places, not three whole files — and stops reading leftover copies of this repository** - `069ac083` (test)
2. **Task 2: The closing sentence on the screen comes from the row that fired, not from a fixed phrase** - `0f853ebe` (feat)

## Files Created/Modified

- `cmd/refusal_self_recovery_test.go` — replaced the file-keyed `mayReadTheIsAnyoneHereFact` with
  `mayHoldASelfRecoveryDecision` (keyed `"<file>::<enclosing declaration>"`, five entries);
  rewrote `TestSelfRecoveryHasOneDecision`'s walk to attribute every read of
  `sessionHasNoOneToAsk`/`refusalSelfRecoveryTable` to its enclosing `*ast.FuncDecl` or
  `*ast.GenDecl` spec via a new helper, `recordSelfRecoveryReads`; added the stale-excuse check;
  added `"worktrees"` to the walk's `SkipDir` list; updated `refusalSelfRecoveryContractProblems`
  to the new two-field row type and to report a blank `Reason`/`Action`; extended
  `TestOnlyASafeRefusalCanRecoverItself` with a locally-built blank-action case; updated
  `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` and `TestSelfRecoveryNoticeSpeaksPlainEnglish`
  for the new call shape; added `TestEveryRecoveryRowSuppliesItsOwnActionWording`.
- `cmd/refusal_self_recovery.go` — added the `refusalSelfRecoveryRow` struct (`Reason`, `Action`);
  changed `refusalSelfRecoveryTable` to `map[string]refusalSelfRecoveryRow`; changed
  `attemptRefusalSelfRecovery` to read `entry.Reason`/`entry.Action` from the table and pass them
  through; changed `renderRefusalSelfRecoveryNotice`'s signature to accept an `action` parameter and
  compose the closing line from it instead of a hardcoded clause.

## Decisions Made

See `key-decisions` in the frontmatter for the exact allow-list keys chosen, how a package-level
declaration's own identifier is attributed to itself, the word-for-word move of the colonize row's
`Action`, and the shape of the future-second-row falsifiability proof.

## Deviations from Plan

None — plan executed exactly as written. Both tasks matched their acceptance criteria without
requiring any Rule 1-4 auto-fixes or architectural questions. One net-new addition beyond the plan's
literal wording: a `sort` import and `sort.Strings` calls on the offender/stale-excuse slices in
`TestSelfRecoveryHasOneDecision`, so a multi-entry failure message has a deterministic order across
runs — a minor robustness addition, not a weakened or changed assertion.

## Mutation Proofs (verbatim observed output)

All mutations were planted and reverted inside disposable git worktrees created under the session
scratchpad at paths containing "Aether" (`Aether-mutation-proof-t18-1`, `Aether-mutation-proof-t18-2`),
never in this checkout, and removed afterward with `git worktree remove --force`. `git status
--short` in the owner's checkout showed only this plan's two files changing throughout (plus the
pre-existing, untouched `.aether/CONTEXT.md` edit from another session).

### Task 1 — CR-02 (rival decision planted inside an allow-listed file)

Planted at the end of the worktree's `cmd/refusal.go`:

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

`go build ./...` succeeded. `go test ./cmd -run TestSelfRecoveryHasOneDecision -count=1 -v` then
**FAILED**, verbatim:

```
refusal_self_recovery_test.go:495: a self-recovery decision or the is-anyone-here fact may only be read at the five reviewed places named in mayHoldASelfRecoveryDecision; found a read outside that list: [cmd/refusal.go::roguesSecondDecisionInsideAllowlistedFile (reads the is-anyone-here fact)]
--- FAIL: TestSelfRecoveryHasOneDecision (0.30s)
```

With the function removed, the same command **PASSED**:
`--- PASS: TestSelfRecoveryHasOneDecision (3.29s)`.

### Task 1 — no regression on the shape the previous round already closed

Planted inside `outputRefusal` in the worktree's `cmd/helpers.go`:

```go
func outputRefusal(r refusal) {
	markRenderedCommandError(1)
	appendRefusalToLog(r)
	if sessionHasNoOneToAsk() && refusalSelfRecoveryTable != nil {
		_, listed := refusalSelfRecoveryTable[r.ID]
		if listed {
			appendRecoveredRefusalToLog(r)
		}
	}
	...
```

`go build ./...` succeeded. The same test **FAILED**, verbatim, naming both file and function:

```
refusal_self_recovery_test.go:495: a self-recovery decision or the is-anyone-here fact may only be read at the five reviewed places named in mayHoldASelfRecoveryDecision; found a read outside that list: [cmd/helpers.go::outputRefusal (names refusalSelfRecoveryTable; reads the is-anyone-here fact)]
--- FAIL: TestSelfRecoveryHasOneDecision (0.27s)
```

With the mutation removed, it **PASSED**: `--- PASS: TestSelfRecoveryHasOneDecision (0.24s)`.

### Task 1 — the excuse list can only shrink honestly

Removed the `sessionHasNoOneToAsk()` condition from `renderRefusal` in the worktree's
`cmd/refusal.go` (changed `if r.ProtectsWork && next != "" && sessionHasNoOneToAsk() {` to
`if r.ProtectsWork && next != "" {`). `go build ./...` succeeded. The same test **FAILED**,
verbatim, naming the now-unmatched excused place:

```
refusal_self_recovery_test.go:506: mayHoldASelfRecoveryDecision names a place that matches nothing in the current source -- an excused place that matches nothing is a stale excuse, and the list may only shrink by the place genuinely going away: [cmd/refusal.go::renderRefusal]
--- FAIL: TestSelfRecoveryHasOneDecision (1.61s)
```

With the condition restored, it **PASSED**: `--- PASS: TestSelfRecoveryHasOneDecision (0.97s)`.

### Task 1 — WR-01 (the leftover-worktree skip is load-bearing, not decorative)

Created, inside the disposable worktree only, `.claude/worktrees/stale-copy/cmd/stale_copy_second_decision.go`:

```go
package cmd

func staleCopySecondDecision() bool {
	return sessionHasNoOneToAsk()
}
```

With `"worktrees"` present in the walk's `SkipDir` list, `go build ./cmd/...` succeeded and
`go test ./cmd -run TestSelfRecoveryHasOneDecision -count=1 -v` **PASSED**:
`--- PASS: TestSelfRecoveryHasOneDecision (0.36s)`.

With `"worktrees"` temporarily removed from that `SkipDir` list, the same command **FAILED**,
verbatim, naming the planted stale-copy path:

```
refusal_self_recovery_test.go:495: a self-recovery decision or the is-anyone-here fact may only be read at the five reviewed places named in mayHoldASelfRecoveryDecision; found a read outside that list: [.claude/worktrees/stale-copy/cmd/stale_copy_second_decision.go::staleCopySecondDecision (reads the is-anyone-here fact)]
--- FAIL: TestSelfRecoveryHasOneDecision (0.26s)
```

The skip was restored, the planted tree deleted, and the final state **PASSED** again:
`--- PASS: TestSelfRecoveryHasOneDecision (0.25s)`.

### Task 2 — WR-03 (the falsifiability proof: fixed phrase reintroduced)

Restored the old hardcoded clause in the worktree's `renderRefusalSelfRecoveryNotice`, ignoring the
new `action` parameter:

```go
b.WriteString(voiceLine("next", "Aether is going ahead and rebuilding the map of your code instead of stopping to ask (`"+next+"`)."))
```

`go build ./cmd/...` succeeded. `go test ./cmd -run TestEveryRecoveryRowSuppliesItsOwnActionWording
-count=1 -v` then **FAILED**, verbatim, on the second-row assertion (the rendered notice used the
locally-built row's `Reason` but not its `Action`):

```
refusal_self_recovery_test.go:867: a notice rendered with a locally built row's own Action does not contain it:
    ━━ 🙅 C A R R Y I N G   O N   W I T H O U T   Y O U ━━
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    ⛔ A territory survey already exists for this project.
    ❓ No one is here to answer, so Aether is going ahead on its own: nobody is here to answer, and this recovery is safe for Aether to carry out on its own
    ➡️ Aether is going ahead and rebuilding the map of your code instead of stopping to ask (`aether colonize --force-resurvey`).
--- FAIL: TestEveryRecoveryRowSuppliesItsOwnActionWording (0.00s)
```

With the parameter honoured again, it **PASSED**:
`--- PASS: TestEveryRecoveryRowSuppliesItsOwnActionWording (0.00s)`.

### Task 2 — the future-row gate (blank Action on the real colonize row)

Blanked the real colonize row's `Action` field in the worktree's `cmd/refusal_self_recovery.go`
(`Action: "rebuilding the map of your code",` → `Action: "   ",`). `go build ./cmd/...` succeeded.
`go test ./cmd -run TestOnlyASafeRefusalCanRecoverItself -count=1 -v` then **FAILED**, verbatim,
naming the row id:

```
refusal_self_recovery_test.go:563: the real refusalSelfRecoveryTable violates its own contract: [colonize-existing-survey-found: has no action]
--- FAIL: TestOnlyASafeRefusalCanRecoverItself (0.00s)
```

With the field restored, it **PASSED**: `--- PASS: TestOnlyASafeRefusalCanRecoverItself (0.00s)`.

## What the owner reads on screen — unchanged

The colonize row's `Action` value moved word for word out of the renderer and into the table:

- **Before (hardcoded in the renderer):** `"rebuilding the map of your code"`
- **After (the row's own `Action` field):** `"rebuilding the map of your code"`

They match character-for-character; the composed sentence
(`"Aether is going ahead and " + action + " instead of stopping to ask (`" + next + "`)."`) is
unchanged, confirmed by `TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened` and
`TestSelfRecoveryNoticeSpeaksPlainEnglish` passing unmodified.

**In plain English:** the screen Aether shows you when it carries on without you always ends with a
sentence naming exactly what it is doing. That sentence is no longer typed into the drawing code
itself — it now lives on the specific entry in Aether's own table, so a future second kind of
recovery cannot accidentally keep printing this one's description; it has to supply its own, or the
program refuses to accept the entry at all.

## What stays deliberately out of scope, carried forward

Per this plan's own prohibitions, two findings from the second gap-closure round remain open and were
not touched:

- **The decision records success before confirming it** (208-REVIEW-GAP2.md WR-01): by the time
  `attemptRefusalSelfRecovery` returns `true`, it has already announced the recovery and recorded it
  as done — but the actual recovery (the caller setting `ForceResurvey = true`) happens afterward,
  held together by a doc-comment instruction rather than a return value the function itself enforces.
  Fixing this is a real design change (having the function carry out the recovery itself, e.g. via a
  callback) and is explicitly not this plan's to make.
- **A future silently-classified command could recover invisibly** (208-REVIEW-GAP2.md WR-05): the
  notice is only shown when `shouldRenderVisualOutput` AND `streamingAllowedForCurrentCommand` both
  allow it; the second gate silences every command the ceremony taxonomy classifies quiet (every
  `*-finalize` name, and any unrecognised command). Today's one row (`colonize`) is safe because
  `colonize` is not quiet-classified, but nothing yet stops a future row from being added for a quiet
  command, which would let Aether replace work and log it as recovered while printing nothing to a
  person sitting there. This remains honestly open.

WINDOWS row 53 (the practice project has never run past step 2 of 14, per D-06) also stays untouched
— this round does not touch it and must not re-describe it as satisfied.

## Verification Performed

| Check | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./cmd/... ./pkg/...` | clean |
| `go test ./cmd -run 'Refusal\|Refuse\|Unattended\|SelfRecovery' -count=1 -timeout 8m` | 2 pre-existing known-red failures only: `TestCodexNativeCancellationRefusalReplay`, `TestCodexNativeCancellationRefusalEvidence` (WINDOWS row 56, not regressions, not this plan's to fix) |
| `go test ./cmd -run 'TestBehaviourMatchesTheRefusalTable\|TestEveryRefusalRowNamesANextCommand\|TestRefusalRegisterIsSortedAndUnique\|TestEveryRefusalSiteIsRegisteredOrCounted\|TestUntypedRefusalFloorOnlyShrinks\|TestColonizeWrapperCarriesTheActWhenAloneRule' -count=1 -timeout 8m` | PASS — the refusal contract and the wrapper fence are intact, no row moved |
| `git status --short` after each mutation proof and after both worktree removals | only `cmd/refusal_self_recovery.go` and `cmd/refusal_self_recovery_test.go` changed in the owner's checkout |
| `git worktree list` at the end | names only the pre-existing worktrees (`Aether-release-1.0.86`, `.claude/worktrees/agent-af06650f4d83f8d05`) |

No paid walk was run and nothing was spent (no `make eval-gate-journey`, no `claude -p`, no
`AETHER_JOURNEY_TRIALS`), per this plan's prohibitions and D-07. No runtime (non-test) source file
outside this plan's two named files was modified in the owner's checkout — every mutation lived only
inside a disposable worktree and was reverted before removal.

## Issues Encountered

One operational note: during the WR-01 mutation cleanup, `git checkout -- <file>` inside the
disposable worktree reverted an uncommitted (copied-in, not yet committed to that worktree) test-file
edit all the way back to the worktree's checked-out HEAD, briefly discarding this plan's own Task 1
edit inside that worktree. Caught immediately by re-checking the file for the expected `"worktrees"`
skip entry; fixed by re-copying the edited file from the owner's checkout before continuing. No effect
on the owner's checkout or on any committed state — the mistake and its correction happened entirely
inside the disposable worktree, before anything was committed.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

This plan closes CR-02 and WR-01 from `208-REVIEW-GAP3.md` (Task 1) and WR-03 (Task 2), the three
findings this round assigned to it. Per `208-CONTEXT.md`'s fourth gap-closure round (D-07), the
remaining findings from that same review — WR-02 (executable coverage for the lighter-survey-team
relaxation) and WR-04 (an honest "what this can't catch" note on the one-builder guard) — were
already closed by plan 208-17. WINDOWS row 53 (the practice project has never run past step 2 of 14)
remains honestly open and is not this round's to close.

---
*Phase: 208-never-a-dead-end*
*Completed: 2026-09-24*
