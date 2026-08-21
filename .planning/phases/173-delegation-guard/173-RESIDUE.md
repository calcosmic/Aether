# Phase 173 Residue — What Delegation Guard proves, and what it does not

This document exists because CLAUDE.md's Definition of Done requires a runnable command behind
every claim, and this repo's own audits (`.planning/v1.14-MILESTONE-AUDIT.md:406`, the learning
pipeline "restored" four times) show what happens when a phase's own summary asserts more than its
tests actually establish. Every line below is either a bounded claim with a command that proves it,
or a residue naming exactly what a reader might wrongly infer and what would be needed to close it.

## What this phase proves

Each line names the command (a `go test -run` filter) that proves it. All commands run from the
repo root against package `./cmd` unless stated otherwise.

**Success criteria** (`.planning/ROADMAP.md` § Phase 173):

1. `aether spawn-can-spawn --depth 99` denies, and a spawn one level past the cap exits non-zero
   and writes no spawn-tree entry —
   `go test ./cmd -run 'TestSpawnCanSpawnDeniesPastDepthCap|TestSpawnLogRefusesPastCapAndWritesNoEntry'`.
2. `aether spawn-tree-depth` reports depth 2 for a three-level tree with every caller passing
   `--depth 0`, because depth is derived from the parent's own recorded entry —
   `go test ./cmd -run 'TestSpawnTreeDepthReportsTwoForAThreeLevelTree|TestSpawnLogDerivesDepthFromRecordedParent|TestWorkersMdStatesOneDepthConvention'`.
3. The whole-run budget of 20 refuses the 21st helper regardless of wave shape, consumption is not
   restored between waves, and neither corrupting the spawn ledger (the file that records which
   helpers were started) nor deleting the run record (the file that says which batch of work is
   currently running) can reset the count to zero —
   `go test ./cmd -run 'TestSpawnTreeBudgetRefusesTheTwentyFirstHelper|TestSpawnTreeBudgetIsNotRestoredBetweenWaves|TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities|TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget|TestErasingTheRunRecordDoesNotResetTheWholeRunBudget|TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn'`.
4. **Yes, with one bound named plainly.**
   `go test ./cmd -run 'TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs|TestDelegationGuardTableCoversEveryGuardCommand'`
   proves every delegation guard (every check that can refuse to let a helper be spawned) denies
   with a non-empty reason against every input it actually reads, including a spawn ledger that is
   present but tampered with, and a run record that is present but blocked (see residue 7 below for
   the exact bounded claim and the (guard, axis) pairs proven). This closes the gap
   `173-VERIFICATION.md` found on 2026-08-13 by actually defeating the budget, not merely by
   reasoning about it. The one bound that remains, named as residue 10 below: a helper record that
   is shaped correctly but simply omits real helpers is not detected by anything in this phase,
   because every check here verifies shape, not truth.
5. A spawn whose (caste, normalised task) already appears in its own ancestor chain is refused,
   naming the ancestor —
   `go test ./cmd -run 'TestSpawnCanSpawnDeniesAncestorCycle|TestSpawnAncestorCheckAllowsDifferentTaskSameCaste|TestSpawnAncestorCheckAllowsSameTaskDifferentCaste'`.
6. `aether spawn-tree-active` renders the tree indented by depth, with parent attribution, while a
   run is in progress, and mutates nothing —
   `go test ./cmd -run 'TestSpawnTreeActiveRendersIndentedByDepthMidRun|TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON|TestSpawnTreeActiveMutatesNothing|TestSpawnTreeActiveKeepsItsJSONContract'`.

**Requirements** (`.planning/REQUIREMENTS.md`):

- **SPAWN-01** (refuse at a configured depth) —
  `go test ./cmd -run 'TestSpawnCanSpawnDeniesPastDepthCap|TestSpawnCanSpawnAllowsWithinCap'`.
- **SPAWN-02** (a spawned child records its true depth) —
  `go test ./cmd -run 'TestSpawnLogDerivesDepthFromRecordedParent|TestSpawnTreeDepthReportsTwoForAThreeLevelTree'`.
- **SPAWN-03** (whole-tree budget bounds what depth alone cannot) —
  `go test ./cmd -run 'TestSpawnTreeBudgetRefusesTheTwentyFirstHelper|TestSpawnTreeBudgetIsNotRestoredBetweenWaves|TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities'`.
- **SPAWN-04** (PreToolUse hook denies before the platform acts; fails closed on an unresolved
  requester) —
  `go test ./cmd -run 'TestHookPreToolUseDeniesUnresolvedRequesterDepth|TestHookPreToolUseAllowsMainSessionDispatch|TestHookPreToolUseAllowsFirstTierWorkerDispatch|TestHookPreToolUseDeniesPastTheDepthCap'`.
  Bounded by the two dispatch levels `173-HOOK-FINDINGS.md` actually captured — see residue 3.
- **SPAWN-05** (ancestor-chain repetition detected and refused) —
  `go test ./cmd -run 'TestSpawnCanSpawnDeniesAncestorCycle|TestSpawnAncestorCheckFailsClosedOnUnreadableTree'`.
- **SPAWN-06** *(decision, not build)* — no command proves a decision. The record is
  `173-CONTEXT.md` D-05 ("coordinator depth 0, its workers depth 1, their helpers depth 2, the
  guard refuses at depth 3"); its code-side consequence — that `.aether/workers.md` states this one
  convention rather than the two contradictory ones it stated before this phase — is proven by
  `go test ./cmd -run TestWorkersMdStatesOneDepthConvention`.
- **SPAWN-07** (operator watches the tree grow) —
  `go test ./cmd -run 'TestSpawnTreeActiveRendersIndentedByDepthMidRun|TestSpawnTreeActiveMutatesNothing'`.
- **SPAWN-08** (abandoned child reaped, budget released, operator can see/clear orphans) —
  `go test ./cmd -run 'TestSpawnReapReleasesBudgetForStaleEntryOnly|TestSpawnReapNeverTouchesAnEntryInsideTheThreshold|TestSpawnReapRespectsRefreshedActivity|TestSpawnReapThresholdIsConfigurable|TestSpawnOrphansListingMutatesNothing'`.

## What this phase does NOT prove

### 1. `spawn-can-spawn` without `--name` is advisory, not authoritative

**The bounded claim:** `spawn-can-spawn --depth N` (no `--name`) reports on a depth the caller
states about itself. **The wrong inference:** that a positive `can_spawn:true` answer from this
form means the requester's depth was verified. **What would close it:** nothing needs to close it —
this is by design (`spawnDecisionInput.DepthIsAuthoritative`). Only `spawn-log`'s call, and
`spawn-can-spawn --name` when the name resolves, are authoritative, because a caller can lie to the
checker but cannot avoid the recorder. Named in `cmd/spawn.go`'s own doc comment on
`spawnDecisionInput`.

### 2. `--parent` is caller-supplied; deriving depth from it closes one lie, not two

**The bounded claim:** a spawn must name a *recorded* ancestor and cannot invent one out of thin
air — `deriveSpawnDepth` denies an unresolvable parent. **The wrong inference:** that this closes
every way a caller could misrepresent its position in the tree. **What would close it:** identity
verification of the caller itself, which nothing in this phase attempts. A caller can still name a
real, but shallower, ancestor as its own parent, understating its own depth relative to its actual
dispatch chain. Recorded as open residue in `173-02-SUMMARY.md` (T-173-07): *"a caller naming a real
but shallower ancestor to gain depth remains open."*

### 3. No bridge from the platform's own subagent identifier to Aether's spawn-tree names

**The bounded claim, quoted from `173-HOOK-FINDINGS.md`'s dated verdict (captured 2026-08-13):**

> VERDICT: HOOK_FIRES_IN_SUBAGENT ... There is no mapping from Claude Code's own agent_id (e.g.
> "ae93ff782863d564f") to Aether's spawn-tree AgentName -- 173-HOOK-FINDINGS.md recorded which
> fields the platform supplies, not an identity bridge. Every rule below is therefore a heuristic
> over agent_id/agent_type's presence/absence and value, not a lookup into Aether's own spawn
> records.

Coverage was empirically observed at exactly two dispatch levels in one real nested-dispatch
capture: the coordinator's own dispatch (no `agent_id`, depth 0) and a subagent's dispatch of its
own helper (`agent_id` present, `agent_type` carrying the *requester's* own dispatched type). No
third-level (helper-of-helper) dispatch was captured. **The wrong inference:** that the hook
performs identity resolution against Aether's own records. **What would close it:** a real identity
bridge between the platform's `agent_id` and Aether's `AgentName`, which does not exist today.
`spawn-log`'s `deriveSpawnDepth` remains the authoritative enforcement; the hook is a
before-the-fact deterrent layered in front of it, not a replacement.

### 4. Run membership is a wall-clock activity window, not a transactional link

**The bounded claim:** the whole-run budget counts "spawns whose activity timestamp falls in the
current run's window" (`pkg/agent/spawn_tree.go`'s `EntriesForRun`, filtering by `SpawnRun`'s
recorded time span). **The wrong inference:** that each spawn is transactionally tagged with the
run ID that created it. **What would close it:** a run identifier written onto each spawn-tree
entry at record time instead of inferred after the fact from timestamps. Today, two runs starting
within the same second, or clock skew between processes, can misattribute a spawn to the wrong
run's budget count. Not built in this phase.

### 5. Ancestor-cycle matching is textual after normalisation, not semantic

**The bounded claim:** the guard refuses "the same caste on a task that normalises to the same
string" — `normalizeSpawnTask` applies exactly four rules (lowercase, whitespace-run collapse,
trim, one trailing full stop) and nothing else, proven exhaustively by
`TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop`. **The wrong inference:** that
reworded but identical work is caught. **What would close it:** semantic task comparison (embedding
similarity, paraphrase detection), deliberately out of scope — `173-06-SUMMARY.md` names this as
T-173-31: "the guard matches text, not meaning," and states that every additional normalisation
rule widens the set of distinct tasks that collide, turning a cycle guard into a false-positive
generator.

### 6. There is no liveness signal; the reaper detects elapsed time, not abandonment

**The bounded claim:** `agent.SpawnStatusAbandoned` means "a configured elapsed time (default 120
minutes, `colony.ColonyState.SpawnReapThresholdMinutes`) passed with no completion reported."
**The wrong inference:** that a reaped entry is confirmed dead or stuck. **What would close it:** a
genuine liveness signal (a heartbeat, a process check) — this system has none. `173-09-SUMMARY.md`
states this explicitly: *"It does not, and cannot, claim the helper is confirmed dead — this system
has no liveness signal, and every comment, output string, and test name in this plan was written to
avoid the word 'detects' applied to abandonment."* The threshold is generous specifically because
no better signal exists.

### 7. Fail-closed is proven per-guard, against the inputs each guard actually reads

**The bounded claim the tests carry:** each delegation guard denies when the inputs it actually
reads cannot be resolved. `spawn-can-spawn-swarm` is the **only** guard that reads
`COLONY_STATE.json`; `spawn-can-spawn` and `spawn-log` depend on `spawn-tree.txt` and
`spawn-runs.json`; `hook-pre-tool-use` reads no file at all and fails closed on an unresolvable
requester identity in its own stdin payload. **The wrong inference to head off:** "every guard
denies **because** colony state is unreadable" — three of the four guards never read it, and any
denial they produce in that scenario comes from their own missing spawn-tree entries instead, not
from `COLONY_STATE.json` at all.

The exact (guard, axis) pairs `TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs` proves,
all exercisable (no axis in this table was declared non-exercisable). "The spawn ledger" below
means `spawn-tree.txt`, the file that records which helpers were started; "the run record" means
`spawn-runs.json`, the file that says which batch of work is currently running; a "guard" is a
check that can refuse to let a helper be spawned:

| Guard | Axis | Fault injected | Deny path reached |
|---|---|---|---|
| `spawn-can-spawn` | `run-state-unreadable` | invalid JSON in the run record | `spawnTreeBudgetReason` |
| `spawn-can-spawn` | `ledger-corrupt` | one line of non-pipe-format text written over the spawn ledger | `spawnTreeBudgetReason` |
| `spawn-can-spawn` | `requester-not-recorded` | the named agent is genuinely absent from a spawn ledger that reads cleanly | `spawnAncestorCycleReason` |
| `spawn-log` | `run-state-unreadable` | invalid JSON in the run record | `spawnTreeBudgetReason` |
| `spawn-log` | `ledger-corrupt` | one line of non-pipe-format text written over the spawn ledger | `spawnTreeBudgetReason` |
| `spawn-log` | `run-state-obstructed` | a directory blocking the run record's own path | `spawnTreeBudgetReason` |
| `spawn-log` | `parent-not-recorded` | `--parent` names a corrupted, unresolvable ledger line | `deriveSpawnDepth` |
| `spawn-can-spawn-swarm` | `colony-state-unreadable` | invalid JSON in `COLONY_STATE.json` | direct `LoadJSON` error branch |
| `spawn-can-spawn-swarm` | `ledger-corrupt` | one line of non-pipe-format text written over the spawn ledger | the existing parser-error deny branch already inside this guard's own code |
| `spawn-can-spawn-swarm` | `spawn-tree-unreadable` | a directory at the spawn ledger's own path | `os.Stat`/`os.IsDir` guard ahead of `Parse()` |
| `hook-pre-tool-use` | `requester-identity-unresolvable` | `agent_id` present, `agent_type` "general-purpose" | `hookSpawnDenyReason` |

The four new rows — `ledger-corrupt` on all three guards that read the spawn ledger, and
`run-state-obstructed` on `spawn-log` — are this gap closure's addition (plan 173-13), proving the
exact gap named in the corrected criterion 4 above. The `requester-not-recorded` row's fault
description also changed from an earlier draft of this table: plan 173-12 re-pointed that axis at a
spawn ledger that is well-formed but simply does not mention the named agent, because a genuinely
corrupted ledger is now caught by the budget check first (the `ledger-corrupt` row immediately
above it), before the ancestor check this row proves is ever reached.

`TestDelegationGuardTableCoversEveryGuardCommand` fails if a guard later starts reading colony
state (it AST-parses `cmd/spawn.go`, `cmd/spawn_budget.go` and `cmd/spawn_ancestor.go` for a
`COLONY_STATE.json` string literal), if a new registered `spawn-can-spawn*`/`spawn-log`/
`hook-pre-tool-use` command is added without a table entry, or if any guard's axis list decays to
zero exercisable entries — so this residue line cannot go stale silently; the same test that would
catch the drift also names exactly what needs to widen here.

**ROADMAP success criterion 4, closed by building, dated 2026-08-13.** This criterion used to be
left at its original absolute wording on purpose, until real evidence existed to judge it against.
That evidence now exists, and the criterion was closed by building a fix, not by softening the
words to fit whatever got built — the direction `172-STOP-RULE.md` requires, and the opposite of
narrowing a claim in advance of evidence. What happened, in order:

1. Checking this phase's own work (`173-VERIFICATION.md`, run 2026-08-13) actually broke the
   whole-run helper budget — the limit of 20 helpers a single run is allowed to create. Overwriting
   the spawn ledger with one line of garbage text made the count reset to zero, and the next helper
   spawned as if the limit had never been reached.
2. While planning the fix, a second way to reach the identical outcome was found — one that needs
   no tampering with the spawn ledger at all. Simply deleting the run record, while leaving a
   perfectly valid, full spawn ledger in place, made the budget check unable to tell which helpers
   belonged to the current run, and it fell back to reporting zero used.
3. Three plans closed both routes. Plan 173-11 taught the file-reading code the difference between
   "this file has never been written" (fine, allow) and "this file exists but is broken or
   unreadable" (not fine, refuse). Plan 173-12 made the helper-limit check itself refuse to answer
   when either file could not be trusted, instead of quietly assuming a fresh, empty run. Plan
   173-13 (this plan) proved every guard that reads either file now refuses correctly on both
   routes, and corrected five comments that used to say this could not happen.
4. The commands that now fail if either route reopens:
   `go test ./cmd -run 'TestCorruptingTheLedgerDoesNotResetTheWholeRunBudget|TestErasingTheRunRecordDoesNotResetTheWholeRunBudget|TestAFreshColonyWithNoLedgerIsStillAllowedToSpawn'`
   together with
   `go test ./pkg/agent -run 'TestSpawnTreeParseTreatsAnAbsentLedgerAsEmptyButACorruptOneAsAnError|TestSpawnTreeRefusesToRewriteACorruptLedger|TestSpawnRunStateTellsAnAbsentRunFileApartFromAnUnreadableOne'`.

Criterion 4 is now proven for both ways this phase found to defeat the whole-run helper limit, with
one bound named plainly rather than left implicit: a helper record that is shaped correctly but
simply lies about what happened is still not caught by anything in this phase. See residue 10
below.

### 8. The refusal reason is proven present in the runtime's output, not in the operator's narration

D-10 (`173-CONTEXT.md`) requires the refusal to surface in the run's own output naming the helper,
its would-be parent, and the reason. Plan 04 delivers exactly that at the Go runtime tier —
`spawnCanSpawnDecision`'s `Detail` field is a human sentence, embedded verbatim in `spawn-log`'s and
`spawn-can-spawn --enforce`'s error output — and the raw CLI error very likely reaches whatever
transcript the wrapper is running inside.

**What is NOT proven:** that this reason renders inside the `/ant-build` or `/ant-continue` ceremony
narration a non-technical operator actually reads. No plan in this phase touches `build.md` or
`continue.md`, deliberately — that is wrapper work and out of scope here. This must be settled by
observation, not asserted: run a build that hits the cap and check whether the reason is visible
without reading JSON. Per CLAUDE.md, the owner of this repo is non-technical, so a refusal they
cannot see is a refusal that does not help them. If observation later shows it is invisible in the
ceremony, that is a wrapper-tier follow-up for a later phase, not a reason to reopen this one.

### 9. `cmd/testdata/orphan_allowlist.json` earned zero removals from this phase

Five spawn-related allowlist entries were evaluated empirically against the reachability ratchet
(`TestNoRegisteredSubcommandIsUnreferenced`) rather than by reading: `spawn-can-spawn`,
`spawn-tree-depth`, `spawn-tree-active`, `spawn-get-depth`, `spawn-can-spawn-swarm`. All five
remained orphans when removed from the allowlist and re-tested. **The bounded claim:** none of them
are called from the corpora this specific ratchet scans — `.claude/commands/ant/`,
`.opencode/commands/ant/`, `.aether/commands/*.yaml`, `.aether/utils/hooks/*.js`, `scripts/*.sh`, or
Go self-invocation inside `cmd/`. **The wrong inference:** that these commands have no caller
anywhere in the repo. `.aether/workers.md` documents `aether spawn-can-spawn {your_depth}
--enforce` as the spawn protocol every worker is instructed to follow — but `workers.md` is prose
documentation, not a wrapper command file, a hook script, or Go source, so it is not one of the
corpora this ratchet was built to trust as caller evidence (the same distinction CLAUDE.md's
Definition of Done draws about `suggest-analyze` being described in a playbook the runtime never
loaded). **What would close it:** either the ratchet's corpus list gaining a documentation-based
evidence class (a real architectural change, out of scope for this plan), or a real wrapper/hook
call site being added for one of these five commands in a later phase. No caller was invented here
to make the number look better, per this plan's own instruction.

### 10. A spawn ledger that is shaped correctly but false is still believed

**The bounded claim:** every check this phase adds looks at whether a line in the spawn ledger (the
file that records which helpers were started) is in the right *shape* — the right number of fields,
in the right format. None of them check whether the line is *true*. Someone who writes a
hand-crafted line in exactly the right shape — one that never actually corresponded to a real
helper being spawned — passes every check in this phase, because nothing here can tell a genuine
record from a well-forged one. **The wrong inference:** that a spawn ledger passing every guard in
this phase is guaranteed to be an honest record of every helper actually spawned. **What would
close it:** proof against forgery that shape alone cannot give — for example, a record that can
only ever be added to and never rewritten, or a cryptographic signature covering the file's
contents. That is a change to how the record itself is stored, not to any of the checks (guards)
this phase adds, and remains out of scope here. First named in plan 173-11's own summary as
T-173-67; still unclosed by every plan in this phase, including this one.

### 11. The whole-run limit gates two commands, not the coordinator's own dispatch

**The bounded claim:** the 20-helper whole-run limit is only consulted by two commands built to ask
it a question first — the one that records a helper (`spawn-log`) and the one that checks in
advance whether a helper could be recorded (`spawn-can-spawn`). Everywhere else in the program that
starts a wave of helpers writes straight into the spawn ledger without asking the limit check
first — including the coordinator's own build dispatch, in the function named
`recordCodexBuildDispatches`, and the equivalent code in the plan, colonize, and continue paths.
**The wrong inference:** that the 20-helper whole-run limit bounds every way helpers can be created.
It bounds the two commands built to consult it; the other paths' spawns are still *counted* by
anything that reads the spawn ledger afterward (so they still consume the budget for whoever checks
next), but they are never *asked permission* first. **What would close it:** routing every one of
those direct-write call sites through the same permission check `spawn-log` uses — a change to how
the coordinator's own dispatch works, not a guard fix, and deliberately not attempted in this
gap-closure plan.

### 12. A tampered ledger makes the operator's live view quieter, not louder

**The bounded claim:** the live view an operator watches while helpers are running
(`spawn-tree-active`) shows no header, no helper list, and no error the moment the spawn ledger
becomes unreadable — it simply goes quiet, rather than saying the record is broken. This is the
opposite failure from the checks this gap closure fixes: those now correctly *refuse* on a broken
ledger; this view just stops showing anything. **The wrong inference:** that a blank or
empty-looking live view means nothing is currently running. **What would close it:** one
plain-English line added to that view's own code, naming the unreadable file when this happens — a
change to what the operator sees, not to any guard, and out of scope for this plan.

### 13. Nothing here addresses two processes racing near the limit

**The bounded claim:** nothing in this phase, including this gap closure, stops two processes from
checking the whole-run helper limit at nearly the same moment, both seeing "room for one more," and
both recording a helper — pushing the true count one or more past the stated limit of 20. **The
wrong inference:** that 20 is a hard ceiling under every condition. It is a hard ceiling for one
process checking at a time; under genuine concurrency it is a strong deterrent, not a guarantee.
**What would close it:** holding an exclusive lock across the "may I spawn" check and the "record
this spawn" write, so the two steps happen as one action nothing else can interrupt — a different
kind of fix, aimed at a different part of the code, and not attempted here. Still open as WR-05 in
`173-VERIFICATION.md`'s own findings.

## Stop rule

This phase applies `172-STOP-RULE.md`'s reasoning under the same condition it applied there: **the
rule fires after verification has run and produced evidence, never before.** Phase 172 ran four
completed build-verify rounds before its stop rule fired on the fifth attempted round, and every
narrowing it recorded (ROADMAP line 671, criteria 1 and 4) happened *after* a verifier found a real,
demonstrated gap.

If verification of this phase finds a NEW class of bypass not named in the thirteen residues above, the
correct response is to narrow the affected criterion, record the residue here, and mark the phase
complete against the narrowed criterion — not to open another build round chasing an unbounded
absolute. A narrower true claim beats a broader unproven one.

This does **not** authorise narrowing anything in this document, or in the ROADMAP, in advance of
that evidence. Task 4 of this plan corrected exactly one ROADMAP number — criterion 2's expected
depth, fixed by the already-locked decision D-05 — and left criterion 4 byte-identical specifically
because nothing has yet been built or verified that shows it unmeetable. Softening a success
criterion before a build round proves more against it is the exact failure CLAUDE.md's Definition
of Done exists to stop, and it is the opposite direction of travel from how `172-STOP-RULE.md`
actually fired.
