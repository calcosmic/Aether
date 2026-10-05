---
phase: 208-never-a-dead-end
round: gap-closure-2 (plan 208-11)
reviewed: 2026-09-23T17:05:00Z
depth: standard
diff_base: 700ac32e
files_reviewed: 4
files_reviewed_list:
  - cmd/refusal_self_recovery.go
  - cmd/refusal_self_recovery_test.go
  - cmd/codex_colonize.go
  - cmd/refusal_log.go
findings:
  critical: 2
  warning: 8
  info: 0
  total: 10
status: issues_found
---

# Phase 208 (plan 11): Code Review Report — second gap-closure round

**Reviewed:** 2026-09-23
**Depth:** standard (plus targeted mutation testing)
**Files Reviewed:** 4
**Status:** issues_found

## Summary

The behaviour this round set out to build is genuinely there and genuinely wired. Both colonize
sites call one shared function; the recovery sets `opts.ForceResurvey` *before* the manifest is
built, so `manifest.ForceResurvey` is true and the sibling existing-survey refusal inside
`runCodexColonizeFinalize` (cmd/codex_colonize_finalize.go:370) provably cannot fire afterwards;
the notice goes through `emitVisualProgress`, not a bare stdout write; the additive `Recovered`
field needs no schema bump and old records read correctly. All six new tests pass on a clean
checkout (`go test ./cmd/ -run '<the six>'` → ok, 2.7s).

The problem is the proof, not the behaviour. I built a throwaway worktree and mutated the
implementation, and two of the plan's own `must_haves` turn out to have no failing command behind
them — the exact "false certificate with a green suite" failure CLAUDE.md names as this
repository's signature. Specifically:

- Deleting **three of the four gates** inside `attemptRefusalSelfRecovery` — the opt-in table
  lookup, the `ProtectsWork` check, and the non-empty `NextCommand` check — leaves every test in
  `cmd/refusal_self_recovery_test.go` green, and leaves the whole
  `Refus|Refuse|Unattended|Colonize|Journey` family green too (the single FAIL in that run,
  `TestCodexNativeCancellationRefusalEvidence`, reproduces identically on the unmutated checkout
  and is unrelated known-red).
- Adding a second, competing self-recovery decision to `cmd/helpers.go`'s `outputRefusal` — the
  most natural second home for one — passes `TestSelfRecoveryHasOneDecision` unchanged.

Two mutations *were* caught, and cleanly: removing the `sessionHasNoOneToAsk()` gate fails both
attended subtests, and swapping `emitVisualProgress` for `fmt.Fprint(stdout, …)` fails both
machine-readable-output assertions. So the attended constraint and the JSON-integrity constraint
are honestly locked. The opt-in constraint and the one-decision constraint are not.

The remaining findings are owner-facing honesty and latent-trap issues in the new notice.

No source file was modified by this review. The mutation worktree was created under the scratchpad
and removed; `git status` is back to its pre-review state (`M .aether/CONTEXT.md` only, pre-existing).

## Critical Issues

### CR-01: Three of the four gates in the one decision function have no test that can fail

**File:** `cmd/refusal_self_recovery.go:68-86`, `cmd/refusal_self_recovery_test.go:420-474`

**Issue:** `attemptRefusalSelfRecovery` refuses to recover unless four conditions hold. Only the
first (`sessionHasNoOneToAsk`) is locked by a test that fails when it is removed. I deleted the
other three in a worktree:

```go
// mutated body — opt-in table gate, ProtectsWork gate and NextCommand gate all removed
if !sessionHasNoOneToAsk() {
    return false
}
reason := refusalSelfRecoveryTable[r.ID]   // any id now recovers, with an empty reason
next := strings.TrimSpace(r.NextCommand)
emitVisualProgress(...)
appendRecoveredRefusalToLog(r)
return true
```

`go test ./cmd/ -run 'TestNoOneHereMeansAetherRefreshesTheMapItself|TestOldShapedRefusalLogRecordStillReadsAsNotRecovered|TestAttendedColonizeStillStopsAndAsks|TestUnattendedDirectColonizeRefreshesTheMapItself|TestSelfRecoveryHasOneDecision|TestOnlyASafeRefusalCanRecoverItself'`
→ **ok, 1.475s**. The wider `-run 'Refus|Refuse|Unattended|Colonize|Journey'` run produced one
failure, and that same failure reproduces on the unmutated tree, so nothing in the suite catches
this.

In that mutated build, **any** stop refusal anywhere in the program self-recovers in an unattended
session, silently, with an empty explanatory sentence — including rows that were deliberately kept
out of the table (`colonize-finalize-existing-survey-found`, whose own doc comment says the runtime
*cannot* perform its recovery) and every `ProtectsWork=false` warn-class row.

The reason no test notices is that `TestOnlyASafeRefusalCanRecoverItself` never calls the runtime
function at all. It calls `refusalSelfRecoveryContractProblems`, a checker **defined in the test
file** that nothing in the runtime ever consults. It proves the *table's contents* are sane; it
cannot prove the *gate* still reads the table. Its "prove the guard bites" block mutates the local
checker, not the decision.

This is precisely the Definition-of-Done rule in CLAUDE.md — "a requirement is satisfied only when
a command exists that someone can run, and that command fails when the requirement is unmet" — and
precisely the must_have *"Only a refusal that explicitly declares Aether can carry out its own
recovery is ever recovered from."*

**Fix:** add direct unit coverage of the decision function itself — cheap, no fixture required,
and every value derived from the real registry rather than typed:

```go
func TestOnlyADeclaredRefusalIsEverRecoveredFrom(t *testing.T) {
	saveGlobals(t)
	bindCommandTestRepository(t)
	t.Setenv(unattendedEnvVar, "1")
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	// A registered stop row that protects work and names a command, but is
	// deliberately NOT in the opt-in table -- derived from the registry, never
	// hand-built, so it cannot drift into a shape the runtime can't produce.
	var notDeclared refusal
	for _, row := range refusalRegistry {
		if _, listed := refusalSelfRecoveryTable[row.ID]; listed {
			continue
		}
		if row.Disposition == "stop" && row.ProtectsWork && strings.TrimSpace(row.NextCommand) != "" {
			notDeclared = refuse(row.ID)
			break
		}
	}
	if notDeclared.ID == "" {
		t.Fatal("no undeclared work-protecting stop row exists to drive this guard")
	}

	before := len(refusalLogEntries(200))
	if attemptRefusalSelfRecovery(notDeclared) {
		t.Fatalf("%s is not in the opt-in table and must never be recovered from", notDeclared.ID)
	}
	if got := captureStdoutBuffer(t).String(); got != "" {
		t.Fatalf("a refused self-recovery must print nothing, got:\n%s", got)
	}
	if after := len(refusalLogEntries(200)); after != before {
		t.Fatalf("a refused self-recovery must record nothing: %d -> %d", before, after)
	}

	// The same id, stripped of each remaining precondition in turn.
	declared := refuse("colonize-existing-survey-found")
	noWork := declared
	noWork.ProtectsWork = false
	if attemptRefusalSelfRecovery(noWork) {
		t.Fatal("a refusal that does not protect work must never be recovered from")
	}
	noCommand := declared
	noCommand.NextCommand = "   "
	if attemptRefusalSelfRecovery(noCommand) {
		t.Fatal("a refusal naming no command must never be recovered from")
	}
}
```

Each of those three assertions fails under the mutation above; none does today.

### CR-02: `TestSelfRecoveryHasOneDecision` does not catch a realistic second decision

**File:** `cmd/refusal_self_recovery_test.go:322-410`

**Issue:** The guard flags a file only if it *names `refusalSelfRecoveryTable`*, or if it calls
**both** `sessionHasNoOneToAsk` **and** `refuse(...)` in the same file. Its doc comment admits one
honest limitation (a helper that hard-codes `true`) but not the one that actually matters: the
files that *handle* refusals do not call `refuse(...)` — they receive an already-built `refusal`.
`cmd/helpers.go`'s `outputRefusal` and `cmd/root.go`'s exit lane are both in that shape.

I planted a second decision in the most natural place and ran the guard:

```go
// cmd/helpers.go
func secondSelfRecoveryDecision(r refusal) bool {
	if !sessionHasNoOneToAsk() {
		return false
	}
	return r.ProtectsWork && strings.TrimSpace(r.NextCommand) != ""
}

func outputRefusal(r refusal) {
	if secondSelfRecoveryDecision(r) {
		appendRecoveredRefusalToLog(r)
		return
	}
	markRenderedCommandError(1)
	appendRefusalToLog(r)
	...
```

`go test ./cmd/ -run TestSelfRecoveryHasOneDecision -v` → **PASS (0.28s)**. A second copy that
swallows *every* work-protecting refusal in an unattended session sails straight through the guard
whose must_have reads *"a second copy anywhere in the module fails a named test."*

**Fix:** widen the "acts on a refusal" predicate beyond `refuse(`. The honest signal for "this file
decides whether to stop" is any file that reads the is-anyone-here fact at all, outside the two
files allowed to:

```go
// allowed to name sessionHasNoOneToAsk:
//   cmd/unattended_session.go   (defines it)
//   cmd/refusal.go              (the two render/Error lanes, 208-09)
//   cmd/refusal_self_recovery.go (the one decision)
var mayReadTheIsAnyoneHereFact = map[string]bool{
	"cmd/unattended_session.go":    true,
	"cmd/refusal.go":               true,
	"cmd/refusal_self_recovery.go": true,
}
...
if callsSessionHasNoOneToAsk && !mayReadTheIsAnyoneHereFact[rel] {
	reasons = append(reasons, "reads the is-anyone-here fact outside the three files allowed to")
}
```

That list is short, closed, and shrinking-only in spirit — and it fails by name on the planted
decision above. Whatever shape is chosen, the guard is not finished until a planted second copy
in `outputRefusal` turns it red.

## Warnings

### WR-01: The one decision announces and records the recovery but never performs it

**File:** `cmd/refusal_self_recovery.go:60-86`

**Issue:** By the time `attemptRefusalSelfRecovery` returns `true` it has already told the owner
*"Aether ran `X` on your behalf and is carrying on"* and written a durable `recovered: true`
record — but it has not run anything. The actual recovery is the caller's `opts.ForceResurvey =
true`, held together by a doc-comment instruction: *"A caller that receives true from this function
should fall through into the recovery path unconditionally."* This repository's own Phase 208
lesson is that a wrapper instruction is not a mechanism. A third call site that returns early, or
errors, after a `true` leaves the owner told a false thing and the log carrying a false record,
with no test anywhere that would notice.

**Fix:** make the action part of the contract rather than an instruction — e.g. have the function
take the recovery itself and only announce/record once it has run:

```go
func attemptRefusalSelfRecovery(r refusal, carryOut func() error) bool
```

so the announcement and the log record are emitted after `carryOut()` succeeds, and a caller
physically cannot receive `true` without the recovery having happened.

### WR-02: The notice tells the owner Aether ran a command it did not run

**File:** `cmd/refusal_self_recovery.go:103`; call site `cmd/codex_colonize.go:363-374`

**Issue:** The line is `"Aether ran \`" + next + "\` on your behalf and is carrying on."`, where
`next` is the registry row's `NextCommand` — `aether colonize --force-resurvey`. On the plan-only
lane Aether did not run that command; it continued the *current* `--plan-only` invocation with the
force flag set, and at that point it has not re-surveyed anything at all — it has handed a host a
manifest to dispatch surveyors from. The owner is told a re-survey happened when the re-survey has
not yet started.

**Fix:** state what actually happened rather than naming a command that was not run, e.g.
`voiceLine("next", "Aether is going ahead and rebuilding the map of your code instead of stopping to ask.")`,
keeping the command name (if wanted) as evidence rather than as the claim.

### WR-03: The owner-facing notice prints internal bookkeeping and repo jargon

**File:** `cmd/refusal_self_recovery.go:22-27, 101`

**Issue:** The table's reason string is rendered verbatim into the owner's screen, and it reads:

> No one is here to answer, so Aether is doing this itself: nobody is present to ask, the refusal's
> own next command (aether colonize --force-resurvey) is safe for Aether to run itself, and the
> alternative -- printing an instruction and hoping an unattended chat follows it -- is a rehearsal
> that can never finish **(D-03, 208-CONTEXT.md)**.

CLAUDE.md's standing rule is explicit: *"Finding IDs (CR-06, WR-04) and criterion numbers are
internal bookkeeping — say what the problem is instead of naming its code."* A planning-decision
ID and a `.planning/` filename on the owner's screen is exactly that. "the refusal's own next
command", "an unattended chat" and "a rehearsal" are also repo vocabulary, untranslated.

Because this new screen is not registered in the voice corpus (`classic_voice_corpus_test.go`),
none of `TestVoicedScreensSpeakPlainEnglish`, `TestVoicedScreensCarryNoRawStateToken` or the
density gate ever look at it.

**Fix:** split the map value in two — a developer-facing rationale kept as a Go comment next to the
row (where the `D-03` citation belongs), and a short plain-English owner sentence that is what
`renderRefusalSelfRecoveryNotice` prints. Then register the rendered notice in the shared voice
corpus so the existing plain-English and raw-token checks cover it.

### WR-04: The attended-behaviour proof depends on an ambient environment variable being absent

**File:** `cmd/refusal_self_recovery_test.go:212-215, 232-234`

**Issue:** Both attended subtests rely on `AETHER_UNATTENDED` simply not being set
("Deliberately not setting unattendedEnvVar -- the ordinary case"), while every other subtest in
the file pins it with `t.Setenv`. Run the suite from any shell or harness that exports
`AETHER_UNATTENDED=1` — which is exactly what the journey gate's unattended walks do — and the
*only* proof of D-03's attended constraint turns red for an environmental reason, indistinguishable
from a real regression. The attended guarantee should not be conditional on the tester's shell.

**Fix:** pin it explicitly in both subtests:

```go
t.Setenv(unattendedEnvVar, "") // a person is present -- pinned, not inherited
```

(`sessionHasNoOneToAsk` accepts only the exact value `"1"`, so an empty value is the attended case.)

### WR-05: "silent in machine-output mode" is only half the gate — a future row could recover invisibly

**File:** `cmd/refusal_self_recovery.go:56-63`

**Issue:** The doc comment presents `emitVisualProgress` as gated on machine-output mode alone.
It is gated on two things (`cmd/codex_visuals.go:623-626`): `shouldRenderVisualOutput(stdout)` **and**
`streamingAllowedForCurrentCommand()`. The second silences every command the ceremony taxonomy
classifies quiet — every `*-finalize` name, `ceremony`, `version`, `spawn-log`, and **every command
the taxonomy does not recognise at all** (`classifyCommandCeremonyLevel`'s `default`). `colonize` is
worker-theatre so today's single row is safe, but the moment a second id is added for a quiet or
unlisted command, Aether will replace an owner's work, write a `recovered: true` record, and print
absolutely nothing — in visual mode, to a person sitting there. Nothing in the file or the tests
names this.

**Fix:** either say so in the doc comment and in the table's "nothing else goes in this map without
its own owner ruling" note, or make the contract enforceable — assert in
`refusalSelfRecoveryContractProblems` that every listed id's owning command is not quiet-classified,
so a future row on a silent command is refused by name at test time.

### WR-06: The declared key_link is only proven at its first hop

**File:** `cmd/refusal_self_recovery_test.go:96-170`

**Issue:** The plan's key_link is *"opts.ForceResurvey -> manifest.ForceResurvey -> the sibling
existing-survey check in runCodexColonizeFinalize does not fire -> the territory snapshot is
rewritten."* The tests stop at `result.force_resurvey == true` on the plan-only lane. Nothing drives
plan-only self-recovery onward through `colonize-finalize`. Reading the code, the chain does hold
(`cmd/codex_colonize_finalize.go:370` requires `!manifest.ForceResurvey`), so this is a proof gap
rather than a defect — but it is the exact segment WINDOWS entry 53 lives in, and it is the segment
three paid journey walks have died on.

**Fix:** extend the plan-only subtest one step: feed the recovered manifest into
`runCodexColonizeFinalize` with a synthetic completion packet and assert the sibling refusal does
not fire.

### WR-07: The visual-mode assertion does not anchor on the notice itself

**File:** `cmd/refusal_self_recovery_test.go:164-168`

**Issue:** The only assertion is `strings.Contains(rendered, translateHintCommandsForPlatform(row.NextCommand, "claude"))`.
It never checks that the notice's own banner ("Carrying On Without You") or its
"no one is here" line is present, so the test passes on any colonize output that happens to mention
that command string anywhere. It happens not to today, which is luck rather than design.

**Fix:** also assert the banner heading and the no-one-is-here line, both read from the renderer
rather than re-typed:

```go
if !strings.Contains(rendered, "Carrying On Without You") {
	t.Fatalf("the self-recovery notice itself never appeared:\n%s", rendered)
}
```

### WR-08: The decision trusts the caller's refusal fields instead of the registry row

**File:** `cmd/refusal_self_recovery.go:76-82`

**Issue:** Having matched `r.ID` against the table, the gate then reads `r.ProtectsWork` and
`r.NextCommand` **off the caller-supplied struct**, not off `refusalForID(r.ID)`. The build-time
contract checker (`refusalSelfRecoveryContractProblems`) checks the registry row; the runtime checks
whatever the caller handed over. The two can disagree, and the command the owner is told Aether ran
is taken from the caller's field. CLAUDE.md's full-rigour column names "anything a worker or a
wrapper can influence" as untrusted input; the same discipline applies here, cheaply.

**Fix:** re-read the authoritative fields from the registry once the id is known:

```go
row, ok := refusalForID(r.ID)
if !ok || !row.ProtectsWork || strings.TrimSpace(row.NextCommand) == "" {
	return false
}
next := strings.TrimSpace(row.NextCommand)
```

This also closes the gap between the runtime gate and the contract checker, so one test genuinely
covers both.

---

## Verification performed

| Check | Result |
|---|---|
| Six new tests, clean checkout | pass (2.7s) |
| Mutation: remove opt-in table + ProtectsWork + NextCommand gates | **not caught** — all six pass (1.5s); wider `Refus\|Colonize\|Unattended\|Journey` run also clean apart from pre-existing known-red `TestCodexNativeCancellationRefusalEvidence` |
| Mutation: remove `sessionHasNoOneToAsk()` gate | caught — both `TestAttendedColonizeStillStopsAndAsks` subtests fail |
| Mutation: `emitVisualProgress` → `fmt.Fprint(stdout, …)` | caught — both machine-readable-output assertions fail |
| Probe: second decision planted in `outputRefusal` (`cmd/helpers.go`) | **not caught** — `TestSelfRecoveryHasOneDecision` passes |
| `git status` after review | unchanged (`M .aether/CONTEXT.md`, pre-existing); mutation worktree removed |

_Reviewed: 2026-09-23_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard + mutation testing_
