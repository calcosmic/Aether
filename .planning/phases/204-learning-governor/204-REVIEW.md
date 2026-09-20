---
phase: 204-learning-governor
reviewed: 2026-09-15T00:00:00Z
depth: standard
files_reviewed: 22
files_reviewed_list:
  - cmd/codex_build.go
  - cmd/codex_continue.go
  - cmd/codex_continue_finalize.go
  - cmd/codex_visuals.go
  - cmd/command_guide.go
  - cmd/consolidation_lifecycle.go
  - cmd/episode_ledger.go
  - cmd/eval_gates.go
  - cmd/forced_reviewer_waiver.go
  - cmd/handoff_decisions_cmd.go
  - cmd/improvement_cmds.go
  - cmd/improvement_pass.go
  - cmd/improvement_report.go
  - cmd/learning_validator.go
  - cmd/live_events.go
  - cmd/memory_schema.go
  - cmd/plan_revision.go
  - cmd/recovery_orchestrator.go
  - cmd/rollback.go
  - cmd/shadow_cmds.go
  - cmd/source_proposal.go
  - cmd/swarm_cmd.go
  - cmd/wrapper_command_names.go
findings:
  critical: 4
  warning: 3
  info: 0
  total: 7
status: issues_found
---

# Phase 204: Code Review Report (Gap-Closure Plans 204-12..16)

**Reviewed:** 2026-09-15
**Depth:** standard
**Files Reviewed:** 22 (one file in the config list, `cmd/wrapper_command_names.go`, carries a one-line change; `cmd/codex_build_finalize.go` was read as cross-file context though not itself in the diff)
**Status:** issues_found

## Summary

This review covers the diff between `4eb7a9ed` and `HEAD` for the 23 files
Phase 204's five gap-closure plans (204-12 through 204-16) touched. Most of
the diff is careful, well-documented plumbing: new episode-ledger fields get
real writers, the improvement pass and hypothesis-validator are genuinely
non-blocking, and the "compared" vocabulary keeps the closing card honest.

However, four of the changes have a real correctness problem, and three of
those trace directly back to this same phase's own stated goal: closing gaps
where "machinery exists but was never switched on." In two cases the new
wiring reintroduces exactly that failure mode under a different name — the
automatic promotion pass and the recovery episode boundary can never behave
as documented in the paths that matter most in production (the primary
interactive build-finalize path, and the promotion feature that is this
plan's headline deliverable). A third gap (swarm's delegate/external
finalize lane) was left out of the very episode-ledger-parity sweep that
fixed the identical gap for build and continue in this same diff. A fourth
is an operational risk: this diff is the first thing that ever makes a
previously-dead, git-branch-mutating code path fire automatically and
unattended.

## Critical Issues

### CR-01: The automatic hypothesis-to-validated promoter can never promote anything in production

**File:** `cmd/learning_validator.go:115-141` (`promoteHelpfulHypotheses`,
`learningEntryHasHelpfulApplication`), cross-referenced against
`cmd/application_evidence.go:571,647,777,811,917` and
`cmd/instinct_application.go:69,75`

**Issue:** `promoteHelpfulHypotheses` is 204-16's headline deliverable
(WINDOWS.md entry 44): every `learn.Entry` still in hypothesis status is
promoted to `learn.StatusValidated` when `learningEntryHasHelpfulApplication(entry.ID)`
finds a `GuidanceApplications` record for that same `entry.ID` in the
`guidanceApplicationStateHelpful` state.

The bug is an ID-space mismatch. `entry.ID` comes from `learn.Entry` records
in the *learning* store (`learn.NewColonyStore`), populated exclusively by
the hand-run `learning-propose` command (`cmd/learning_cmds.go:497-511`),
which either leaves `ID` empty (auto-generated inside `learn.ColonyStore.Add`)
or is otherwise never derived from anything in `instincts.json`.

`GuidanceApplications[].GuidanceID`, on the other hand, is written by
exactly one real writer function
(`recordGuidanceApplicationState`, `cmd/application_evidence.go:541`), and
**every non-test call site passes an Instinct ID, never a learn.Entry ID**:

- `cmd/instinct_application.go:69,75` — `recordGuidanceApplicationState(inst.ID, ...)`
- `cmd/application_evidence.go:571` (`guidanceClaim{GuidanceID: inst.ID}` feeding into `claim.GuidanceID` used at lines 857/862)
- `cmd/application_evidence.go:777` (`guidanceContradiction{GuidanceID: inst.ID}` feeding into `contradiction.GuidanceID` used at lines 869/873)
- `cmd/application_evidence.go:917` — reads back `rec.GuidanceID`, itself always an Instinct ID by the chain above

Instincts (`colony.InstinctEntry`, `instincts.json`) and learn entries
(`learn.Entry`, the separate learning-propose/-validate store) are two
disjoint pipelines with two disjoint ID generators (`pkg/memory/promote.go:153`
mints `InstinctEntry.ID` from an `Observation`, never from a `learn.Entry`).
No production code path ever writes a `GuidanceApplications` record keyed by
a `learn.Entry.ID`. Consequently `learningEntryHasHelpfulApplication` will
return `false` for every real hypothesis, every single check, forever —
`promoteHelpfulHypotheses` runs (per `Ran: true`) but `Promoted` can never be
non-empty in production.

The test suite passes only because `cmd/learning_validator_test.go`'s
`seedHypothesisEntry` hand-types the learn entry's `ID` and then calls
`mustRecordGuidanceState(t, id, ...)`
(`cmd/application_evidence_test.go:513-518`), which calls
`recordGuidanceApplicationState` *directly* with that same id as
`guidanceID` — bypassing every real production call site and constructing a
`GuidanceApplications` record in a shape the runtime itself cannot produce.
This is exactly the "fixture built in a shape the runtime cannot produce"
failure mode this repo's own CLAUDE.md calls out by name ("a false
certificate... how this project has repeatedly shipped a broken feature
with a green suite").

**Fix:** Either (a) route real corroboration checks and promotion through a
shared identifier space — e.g. have `promoteHelpfulHypotheses` look up
guidance applications for the Instinct that a `learn.Entry` was promoted
from (via `entry.ParentID`, or a lineage link recorded at promotion time),
or (b) have the learning-propose/-validate pipeline itself write/read
`GuidanceApplications` using `learn.Entry.ID`, with a real production call
site doing so. Whichever is chosen, add or fix a test that drives the
scenario through the actual production writer chain (`recordInstinctDeliveries`
→ `recordGuidanceApplicationState`) rather than calling
`recordGuidanceApplicationState` directly with a hand-picked ID, so the test
cannot pass unless the real runtime can produce the fixture.

---

### CR-02: Recovery-episode collision silently drops durable close records on the primary interactive build-finalize path

**File:** `cmd/recovery_orchestrator.go:216-249`, `cmd/live_events.go:604-611`
(`currentLiveRecoveryEpisode`), `cmd/episode_ledger.go:257-329`
(`recordEpisodeOutcome`'s "second close" refusal), triggered from
`cmd/codex_build_finalize.go:1694` (`buildExternalBuildRecoveryInstructions`)

**Issue:** The new 204-13 code in `orchestrateRecovery` opens and closes its
own standalone episode boundary only in the fallback case —
`recoveryEpisodeKind == events.EpisodeKindRecovery`, i.e. when
`currentLiveRecoveryEpisode` found no already-open build or continue
episode to attach the decision to. In that fallback,
`currentLiveRecoveryEpisode(phase)` always returns the same fixed ID,
`fmt.Sprintf("recovery-phase-%d", phase)` (`cmd/live_events.go:611`).

`cmd/codex_build_finalize.go`'s `buildExternalBuildRecoveryInstructions`
(the recovery step of the delegate/external build-finalize lane — the
lane the interactive wrapper's documented primary path uses, per this
repo's own CLAUDE.md "Command Playbooks" section) calls `orchestrateRecovery`
once per failed dispatch, inside a `for _, dispatch := range failed` loop
(`cmd/codex_build_finalize.go:1694`). This lane never calls
`setActiveLiveBuildEpisode` (confirmed: that setter is only called from
`cmd/codex_build.go`, `cmd/codex_continue.go`, and
`cmd/codex_continue_finalize.go` — never from `codex_build_finalize.go`),
so `currentLiveBuildEpisode()`/`currentLiveContinueEpisode()` are both
empty during this loop, and every iteration falls into the new
`recoveryOwnsItsOwnEpisode` branch with the **same** `recovery-phase-N` ID.

On the first failed dispatch this opens and closes `recovery-phase-N`
cleanly. On the second (and every subsequent) failed dispatch in the same
phase, `emitColonyLiveEpisodeStarted` re-opens the same ID (harmless), but
the closing `emitColonyLiveEpisodeEnded` hits
`recordEpisodeOutcome`'s WR-02 "second close" refusal
(`cmd/episode_ledger.go:303-312`: "episode ledger refuses a second close for
episode %q: it is already closed") — a `warning:` line to stderr and the
record is silently dropped, not written. A wave with two or more failed
workers in the same phase — an entirely ordinary occurrence — will lose the
durable recovery-episode record for every failure after the first, on the
lane the interactive wrapper actually uses day to day.

This directly undercuts the plan's own stated intent in the code comment
("a standalone decision is still recorded rather than dropped") — for
multiple standalone decisions in one phase, it is exactly dropped.

**Fix:** Give each standalone recovery decision its own unique episode id
when it does not attach to an open build/continue episode (e.g. include the
worker name, task id, or a monotonic counter in the fallback ID:
`recovery-phase-%d-%s`), or fold repeated recovery decisions for the same
phase into one still-open recovery episode across the whole loop, closing
it once after the loop finishes rather than per-iteration. Add a test that
drives `buildExternalBuildRecoveryInstructions` (or `orchestrateRecovery`
directly) with two-or-more failed dispatches in one phase, with no active
build/continue episode, and asserts two independent durable closed records
exist.

---

### CR-03: Swarm's delegate/external finalize lane never records a durable episode at all

**File:** `cmd/swarm_cmd.go:291-335` (`runSwarmDestroy`, wired) vs.
`cmd/swarm_cmd.go:1087-1160` (`runSwarmFinalize`, not wired)

**Issue:** This same diff (204-13/204-15) added episode-ledger open/close
parity for build's native lane (`codex_build.go`) and *both* of continue's
lanes — native (`codex_continue.go`) and delegate
(`codex_continue_finalize.go`, itself newly wired in this diff at
`cmd/codex_continue_finalize.go:100-129`, explicitly because "before this
plan, `runCodexContinueFinalize` opened and closed NO episode at all"). The
same treatment was applied to swarm's native lane, `runSwarmDestroy`
(`cmd/swarm_cmd.go:335` onward: `emitColonyLiveEpisodeStarted`, a deferred
`emitColonyLiveOutcomeRecorded`/`emitColonyLiveEpisodeEndedEventOnly`).

Swarm also has a delegate/external lane — `runSwarmFinalize`
(`cmd/swarm_cmd.go:1087`), reached via `aether swarm-finalize
--completion-file <file>`, the exact same "plan-only wrapper dispatches,
then finalize commits" shape build and continue both have, and named in
`buildSwarmManifest`'s own `FinalizerCommand` field
(`cmd/swarm_cmd.go:886`). `runSwarmFinalize` contains zero calls to
`emitColonyLiveEpisodeStarted`, `emitColonyLiveEpisodeEnded`, or
`emitColonyLiveOutcomeRecorded` — confirmed by grep. Every swarm run
dispatched through the delegate/wrapper path therefore produces no durable
episode-ledger record at all, unlike the native swarm lane and unlike both
build's and continue's delegate lanes.

`TestEveryLifecycleLaneWritesADurableOutcome`
(`cmd/episode_ledger_test.go:539`) does not catch this: it drives exactly
one registered entry point per declared episode kind
(`liveLaneEntryPoints[events.EpisodeKindSwarm] = driveSwarmLiveLane`, which
exercises `runSwarmDestroy` only — `cmd/live_lane_coverage_test.go:447`).
The delegate swarm lane has no registered entry point and no test coverage,
so this gap is invisible to the very suite built to catch "a lane that goes
dark on the episode boundary."

This is the identical class of gap 204-15 explicitly closed for build and
continue's delegate lanes in this same diff — left open for swarm's.

**Fix:** Wire `runSwarmFinalize` with the same
`emitColonyLiveEpisodeStarted`/deferred-close shape `runSwarmDestroy` uses
(reusing `episodeCloseFacts`/`episodeCloseBasics` the same way), and add a
second `liveLaneEntryPoints`-style driver (or extend the existing swarm one)
so `TestEveryLifecycleLaneWritesADurableOutcome`-style coverage exercises
both swarm lanes, the same way continue's two lanes are both provably
covered.

---

### CR-04: The automatic improvement pass's repeated-intervention trigger performs live, unattended `git checkout` on the real working tree, with a documented failure mode that leaves the repo on the wrong branch

**File:** `cmd/improvement_pass.go:361-410` (`triggerRepeatedInterventionProposal`,
new; first production caller of `proposeSourceImprovement`), cross-referenced
against `cmd/source_proposal.go:300-385` (`proposeSourceImprovement`,
pre-existing but previously unreachable)

**Issue:** `triggerRepeatedInterventionProposal` runs unconditionally at the
end of every automatic improvement pass — i.e. at the end of every
`/ant-continue` check, on both check lanes, including inside `/ant-run`
autopilot loops where no one may be watching the terminal. Once any single
declared `episodeInterventionKind` category reaches
`sourceProposalRepeatedInterventionThreshold` (3) distinct episodes, it
calls `proposeSourceImprovement` (`cmd/improvement_pass.go:402`), which is
now reachable from production for the first time (per this file's own doc
comment: "proposeSourceImprovement's first real production caller").

`proposeSourceImprovement` operates on `os.Getwd()` — the actual, live
working directory of the running `aether` process — and, after checking the
tree is clean, does `git checkout -b <branch> <baseCommit>`
(`cmd/source_proposal.go:353`), writes the proposal file, and checks back
out to the original branch (`cmd/source_proposal.go:363`). If that final
`git checkout <originalBranch>` call itself fails, the function returns an
error but **the real repository is left checked out on the newly-created
proposal branch** — there is no retry, and no code path restores the
original branch after that point.

Before this diff, this git-mutating path was dead code (never invoked in
production), so this risk was theoretical. This diff makes it live and
unattended. This repository's own operational history already includes a
documented incident class matching exactly this shape (concurrent sessions
sharing one checkout corrupting state via uncoordinated git operations —
see this project's own `oracle-reinstate` shared-checkout note), and this
new trigger fires with no user confirmation, no `--dry-run` gate, and no
isolation (e.g. a worktree) protecting a concurrent session's working tree
from a mid-check branch switch.

**Fix:** At minimum, run `proposeSourceImprovement`'s git operations inside
an isolated worktree (this project already has worktree infrastructure
elsewhere) rather than the live process cwd, so an automatic, unattended
trigger can never touch the branch a person or another session is actively
using. Failing that, add a recovery/retry path for the "applied but failed
to return to original branch" failure so an automatic pass can never leave
the shared repository on the wrong branch, and surface this specific
failure as loudly as a check failure (not just a `summary.Failures` entry a
person may never read).

## Warnings

### WR-01: Refused improvement candidates are re-compared and re-reported on every single check, forever

**File:** `cmd/improvement_pass.go:314-365` (`processNewImprovementCandidate`)

**Issue:** A declared candidate that is refused by `admitCandidateToCanary`
never gets a canary run recorded (`loadCanaryRun` continues to report
`found: false`). Since `runAutomaticImprovementPass` iterates
`candidateFile.Entries` unconditionally on every phase-end consolidation,
the same refused candidate is re-compared and re-emits an
`improvementPassEventRefused` closing-card line on every subsequent check,
indefinitely, until the candidate is manually withdrawn. This is not
incorrect, but it means a single stale declared candidate produces a
permanent, repeating line on every future check's closing card — a
maintenance/noise concern the design does not appear to have accounted for.

**Fix:** Consider recording a "refused" marker (e.g. a lightweight entry in
the canary-run store, or a refusal timestamp on the candidate record) so a
refused candidate is only re-evaluated when its declaration changes, rather
than on every check forever.

### WR-02: Canary completion/rollback decisions use the current phase's own free-check pass/fail as a proxy for the canary's own health, regardless of which phase or scope the canary actually covers

**File:** `cmd/improvement_pass.go:369-399` (`processRunningImprovementCanary`),
`cmd/improvement_pass.go:284-296` (`improvementPassPhaseGatesPassed`)

**Issue:** `improvementPassPhaseGatesPassed(phaseID)` reads only the
*current* phase's own latest build attempt's free-check report, and — by
its own doc comment — defaults to "passed" whenever no evidence exists,
because its one production call site (`runPhaseEndConsolidation`) is "only
ever reached after the current phase's own checks have already passed."
That reasoning makes `gatesPassed` true in the overwhelming majority of
calls, regardless of whether the specific canary's own scope
(`.aether/data/instincts.json` or `.aether/data/COLONY_STATE.json`) is
actually implicated by the current phase's checks at all. In practice this
means `processRunningImprovementCanary` will almost always choose
"complete" over "roll back" once a canary's bound is reached, since the
signal it consults is largely orthogonal to whether the canary's own change
caused any regression.

**Fix:** If this is deliberate (a genuinely lightweight, best-effort signal
until a scope-specific evaluator exists), say so explicitly in
`processRunningImprovementCanary`'s own doc comment, the way
`shadowEvaluator`'s doc comment names its own placeholder-vs-real-classifier
history — so a future reader does not mistake "gates passed" for "this
canary was graded."

### WR-03: The shadow classifier can never credit a candidate for addressing a fixture whose Title/Invariant text has no words of 5+ letters

**File:** `cmd/shadow_cmds.go:46-76` (`shadowFixtureSubjectWords`,
`shadowTextNamesFixtureSubject`)

**Issue:** `shadowClassifyAgainstBank`'s rule 3 (a non-baseline candidate
passes an unguarded fixture only when its own Scope+ExpectedBenefit text
shares a significant word with the fixture's Title/Invariant) depends
entirely on `shadowFixtureSubjectWords` finding at least one word 5+
letters long that is not in the stop-word list. A fixture whose Title and
Invariant happen to consist only of short/common words (e.g. terse titles)
silently produces zero subject words, meaning `shadowTextNamesFixtureSubject`
returns `false` for every possible candidate text — that fixture becomes
permanently un-addressable by any real candidate, with no error, warning,
or fallback signalling the degraded classification.

**Fix:** Add a minimum-subject-words assertion (or a fallback to full-word
matching without the length filter) so a terse fixture cannot silently
become impossible for a genuine candidate to be credited against, and
consider a startup/test-time check that every fixture in the committed
bank has at least one qualifying subject word.

---

_Reviewed: 2026-09-15_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
