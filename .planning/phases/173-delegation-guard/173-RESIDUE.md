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
3. The whole-run budget of 20 refuses the 21st helper regardless of wave shape, and consumption is
   not restored between waves —
   `go test ./cmd -run 'TestSpawnTreeBudgetRefusesTheTwentyFirstHelper|TestSpawnTreeBudgetIsNotRestoredBetweenWaves|TestSpawnTreeBudgetAndWaveCapAreSeparateQuantities'`.
4. **Partially, with a named bound.** `go test ./cmd -run TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs`
   proves every delegation guard denies with a non-empty reason against the inputs it actually
   reads. It does NOT prove the criterion's literal framing that all four guards deny *because*
   both `COLONY_STATE.json` and the spawn tree are unreadable — three of the four guards never read
   `COLONY_STATE.json` at all (see residue 7 below for the exact bounded claim and the (guard,
   axis) pairs proven). Per this plan's own instruction, criterion 4 is left unnarrowed and
   whether this evidence satisfies it as written is verification's call, not this document's.
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
all exercisable (no axis in this table was declared non-exercisable):

| Guard | Axis | Fault injected | Deny path reached |
|---|---|---|---|
| `spawn-can-spawn` | `run-state-unreadable` | invalid JSON in `spawn-runs.json` | `spawnTreeBudgetReason` |
| `spawn-can-spawn` | `requester-not-recorded` | `--name` resolves to nothing (corrupted tree line) | `spawnAncestorCycleReason` |
| `spawn-log` | `run-state-unreadable` | invalid JSON in `spawn-runs.json` | `spawnTreeBudgetReason` |
| `spawn-log` | `parent-not-recorded` | `--parent` names a corrupted, unresolvable tree line | `deriveSpawnDepth` |
| `spawn-can-spawn-swarm` | `colony-state-unreadable` | invalid JSON in `COLONY_STATE.json` | direct `LoadJSON` error branch |
| `spawn-can-spawn-swarm` | `spawn-tree-unreadable` | a directory at `spawn-tree.txt`'s path | `os.Stat`/`os.IsDir` guard ahead of `Parse()` |
| `hook-pre-tool-use` | `requester-identity-unresolvable` | `agent_id` present, `agent_type` "general-purpose" | `hookSpawnDenyReason` |

`TestDelegationGuardTableCoversEveryGuardCommand` fails if a guard later starts reading colony
state (it AST-parses `cmd/spawn.go`, `cmd/spawn_budget.go` and `cmd/spawn_ancestor.go` for a
`COLONY_STATE.json` string literal), if a new registered `spawn-can-spawn*`/`spawn-log`/
`hook-pre-tool-use` command is added without a table entry, or if any guard's axis list decays to
zero exercisable entries — so this residue line cannot go stale silently; the same test that would
catch the drift also names exactly what needs to widen here.

**ROADMAP success criterion 4 is deliberately left at its original absolute wording for this
phase, to be judged against evidence at verification time rather than pre-emptively narrowed; if
the evidence then shows it unprovable, narrowing is the correct response at that point, and this
residue is the record of why.**

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

## Stop rule

This phase applies `172-STOP-RULE.md`'s reasoning under the same condition it applied there: **the
rule fires after verification has run and produced evidence, never before.** Phase 172 ran four
completed build-verify rounds before its stop rule fired on the fifth attempted round, and every
narrowing it recorded (ROADMAP line 671, criteria 1 and 4) happened *after* a verifier found a real,
demonstrated gap.

If verification of this phase finds a NEW class of bypass not named in the nine residues above, the
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
