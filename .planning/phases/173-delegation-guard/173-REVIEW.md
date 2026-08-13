---
phase: 173-delegation-guard
reviewed: 2026-08-13T17:12:33Z
depth: standard
review_type: update
files_reviewed: 8
files_reviewed_list:
  - pkg/agent/spawn_tree.go
  - pkg/agent/spawn_tree_test.go
  - cmd/spawn_ancestor_test.go
  - cmd/spawn_budget.go
  - cmd/spawn_budget_test.go
  - cmd/spawn_failclosed_test.go
  - cmd/internal_cmds.go
  - .github/workflows/ci.yml
findings:
  critical: 0
  warning: 8
  info: 6
  total: 14
status: issues_found
---

# Phase 173: Code Review Report (Update — post gap-closure)

**Reviewed:** 2026-08-13T17:12:33Z
**Depth:** standard
**Files Reviewed:** 8 (gap-closure scope) + prior findings re-verified against current code
**Status:** issues_found

## Summary

This is an update review after plans 173-11..13 closed two budget-reset routes:
(1) a corrupt spawn ledger (`spawn-tree.txt`) that used to be silently treated
as empty, and (2) an erased run record (`spawn-runs.json`) that used to reset
the whole-run helper budget to zero. The gap-closure work is genuinely strong.
`spawn_tree.go` now draws a clean three-way distinction on both files it owns
(absent → empty, unreadable-otherwise → error, present-but-unparseable →
error wrapping `ErrSpawnTreeCorrupt`); `RecordSpawn`/`UpdateStatus` refuse to
rewrite a corrupt ledger and leave the tampered bytes on disk as evidence;
`spawnTreeBudgetState` establishes ledger integrity via `st.Parse()` *before*
resolving the run window and counts the whole ledger when no run resolves
against a non-empty ledger. The new tests are red-proofs with negative
controls, byte-identical no-trace assertions, and canary strings guarding
against task-text leakage. `go build`, `go vet`, `go test ./pkg/agent`, and the
`cmd` gap-closure/guard suite were all run during this review and pass, and the
CI wiring gate (`TestWiringGateStepRunsEveryWiringTest`) confirms the three new
`spawn_budget_test.go` tests are pinned into the named CI step, not merely
carried by the blanket `go test ./...` run.

**CR-01 is resolved.** The worker-writable capture switch
(`~/.aether/hook-capture-path`) was removed the same day; capture is now gated
solely by the `AETHER_HOOK_CAPTURE_FILE` environment variable, which a worker's
Write tool cannot set for the hook process, and the boundary is locked by
`TestHookCaptureHasNoFileBasedSwitch`.

**One new warning (WR-08).** The whole-ledger safety net the gap-closure added
for the "no run resolves" case does **not** cover a run that resolves but has
already ended (or whose window otherwise excludes the live ledger entries). In
that case the budget is scoped to an empty/stale window and reports zero
consumed against a full ledger — a third route to the exact outcome plans
173-11..13 set out to eliminate, reachable by editing (not deleting)
`spawn-runs.json`. Reproduced live during this review.

**Prior warnings WR-01..WR-07 and info IN-01..IN-06 remain open** — each was
re-checked against current code and is unchanged by the gap-closure (they touch
`workers.md`, the patrol wrappers, the hook heuristic, the ancestor walk, and
the advisory-command midden write, none of which plans 173-11..13 modified).

## Warnings

### WR-08 (NEW): The whole-ledger budget safety net is bypassed when the current run has ended — a full budget reset via ordinary `spawn-runs.json` editing

**File:** `cmd/spawn_budget.go:109-134` (`spawnTreeBudgetState`), together with
`pkg/agent/spawn_tree.go:196-211` (`CurrentRun`) and `:583-615`
(`filterEntriesForRun`)
**Issue:** The gap-closure's Gap B defense counts the whole ledger **only** on
the `!ok` branch (no run resolves). But `CurrentRun()` returns the run named by
`current_run_id` regardless of whether that run has ended, so a run with
`status: completed` and a past `ended_at` still resolves `ok=true`. The budget
then counts only `EntriesForRun(run.ID)`, whose window is `[StartedAt,
EndedAt]`. Any live entries recorded outside that window — for example after
the run ended — are silently excluded, and the whole-ledger fallback never
fires. A full ledger can therefore report `budget_consumed: 0`.

This is the same failure class as the two routes plans 173-11..13 closed, and
it is reachable by editing an ordinary writable file (`spawn-runs.json`) rather
than deleting it. The `!ok` deletion route was hardened to count the whole
ledger; the `ok=true, run ended, window excludes the ledger` route was not.
Reproduced live during this review against a freshly built binary:

```
spawn-runs.json: current_run_id "build-1", status completed,
                 started_at 2h ago, ended_at 1h ago
spawn-tree.txt:  25 live "spawned" entries, all timestamped "now"

$ aether spawn-log --parent Queen --caste builder --name P26 --task "..." --depth 0
{"ok":true,"result":{"budget_consumed":0,"budget_max":20,...,"recorded":true}}
exit=0
```

25 live helpers in a valid ledger, whole-run cap of 20, and the 26th spawn is
allowed with `budget_consumed:0`. The `spawnReapStaleEntries` call at
`beginRuntimeSpawnRun` masks this in ordinary steady-state operation (old
ghosts get reaped when the next lifecycle run begins), but it does not close
the window a direct `spawn-log`/`spawn-can-spawn` call hits between an ended run
and the next `beginRuntimeSpawnRun`, and it does not stop an operator or
compromised worker from editing `spawn-runs.json`'s timestamps to free the
budget deliberately. The code comment at `cmd/spawn_budget.go:86-89` reasons
that "deleting spawn-runs.json can only make the budget stricter" — true for
deletion, but the sibling case (a valid file naming an ended/stale run) is not
covered by that reasoning and has no test.
**Fix:** In the `ok=true` branch, treat an ended or window-excluded run as
untrusted and fall back to the whole-ledger count. Concretely: if the resolved
run's status is not active, or if any live (`IsLiveSpawnStatus`) entry falls
outside `[StartedAt, EndedAt]`, count the whole ledger's non-abandoned live
entries as a floor rather than trusting the window — a legitimately active run
always contains its own spawns in-window, so this only tightens the anomalous
case. Add a red-proof mirroring
`TestErasingTheRunRecordDoesNotResetTheWholeRunBudget` but with
`spawn-runs.json` left present and valid, naming an ended run whose window
predates a full live ledger. At minimum, if a fix is deferred, extend the
residual-bound note (T-173-67) to explicitly concede that well-formed
`spawn-runs.json` window manipulation also defeats the count, so the gap is
documented rather than silently implied closed.

### WR-01: workers.md still ships the old three-level spawn convention inside the child-prompt template (carried forward — OPEN)

**File:** `.aether/workers.md:364-369`
**Issue:** Re-verified unchanged. The Step 4 child-prompt "SPAWN CAPABILITY"
block still reads `{if depth < 3: "You MAY spawn sub-workers..."}` and
`Spawn limits: Depth 1→4, Depth 2→2, Depth 3→0`, contradicting the same file's
"there is no depth 3" statement (workers.md:272-273) and the two-level guard
this phase enforces. `TestWorkersMdStatesOneDepthConvention` still does not
match this prose block, so the contradiction ships green.
**Fix:** Rewrite the block to the two-level convention (depth 1 may spawn
helpers; depth 2 completes inline) and delete the `Depth 1→4, Depth 2→2, Depth
3→0` line. Extend `TestWorkersMdStatesOneDepthConvention` to fail on
`"depth < 3"`, `"Depth 3"`, and any `Depth 2→N` allowance where N > 0.

### WR-02: workers.md documents a spawn-can-spawn return shape the runtime does not produce (carried forward — OPEN)

**File:** `.aether/workers.md:353-354`
**Issue:** Re-verified unchanged. Step 1 still documents
`# Returns: {"can_spawn": true/false, "depth": N, "max_spawns": N, "current_total": N}`.
The actual command returns `can_spawn`, `depth`, `authoritative` (+
`reason`/`detail` on deny) — see `cmd/spawn.go:368-376`. `max_spawns` and
`current_total` do not exist; a worker parsing them gets null.
**Fix:** Replace with the real shape and have
`TestSpawnCanSpawnAcceptsDocumentedInvocation` assert the documented return
keys are a subset of the actual result keys.

### WR-03: spawn-can-spawn (an inspection command) mutates midden.json when the budget is exhausted (carried forward — OPEN)

**File:** `cmd/spawn_budget.go:169-170` (`spawnTreeBudgetReason` →
`spawnTreeBudgetCeilingToMidden`), reached from `cmd/spawn.go:306` on every
`spawnCanSpawnDecision` call, including the advisory `spawn-can-spawn` path
(`cmd/spawn.go:357`, no `--enforce`)
**Issue:** Re-verified unchanged. `spawnTreeBudgetCeilingToMidden` still fires
from inside `spawnTreeBudgetReason`, which runs on every decision — so an
advisory `spawn-can-spawn` report at the ceiling still appends a midden entry.
This is the same "inspection command must not mutate state" defect class
CLAUDE.md documents for the `consolidation-*-dry-run` bug, and a worker polling
`spawn-can-spawn` at the ceiling floods midden.json.
`TestBudgetCeilingWritesMiddenButDepthRefusalDoesNot` only exercises the
spawn-log path, so the advisory path stays untested.
**Fix:** Pass an `attempting bool` through `spawnDecisionInput` and write midden
only on a genuine refused spawn attempt (spawn-log's deny branch), not on the
advisory report. Add a purity test: `spawn-can-spawn` at the ceiling changes no
file hashes.

### WR-04: the hook's agent_type heuristic fails both ways (carried forward — OPEN)

**File:** `cmd/hook_cmds.go:177-201` (`hookSpawnDenyReason`,
`hookAetherAgentTypePrefix` at :131)
**Issue:** Re-verified unchanged. Requester depth still keys entirely on the
`agent_type` prefix (`aether-*` → depth 1). (a) Under-block: a first-tier
worker that dispatches its helper with any registered `aether-*` type makes the
helper's subsequent dispatch classify as depth 1, allowing a third-level
platform dispatch (spawn-log then refuses to record it, but the subagent still
runs). (b) Over-block: every non-`aether-*` subagent in the repo that
legitimately dispatches its own subagent is denied with "cannot resolve who is
asking to delegate", because the hook is installed repo-wide for all Agent/Task
calls.
**Fix:** Treat an `aether-*` agent_type as depth 1 only when the dispatch target
(`tool_input.subagent_type`) is `general-purpose` or non-aether; deny (or
consult a session-scoped depth marker) when an aether-typed requester
dispatches another aether-typed agent. Scope the deny to sessions with an
active colony so non-Aether tooling is not collateral. At minimum, document the
bypass and add a test pinning current behavior for an aether-*-typed requester.

### WR-05: spawn-log's guard decision and RecordSpawn are not atomic (carried forward — OPEN)

**File:** `cmd/spawn.go:94-118`
**Issue:** Re-verified unchanged. `deriveSpawnDepth`, `spawnCanSpawnDecision`
(budget + ancestor checks), and `st.RecordSpawn` each re-read the ledger
independently; no lock spans check-then-record. Two parallel processes that
both observe `Consumed == 19` both pass the budget check and both record,
yielding 21+ entries — the budget raised by race rather than by config. The
gap-closure moved the *integrity* check earlier but did not close this
check-then-act window.
**Fix:** Perform the budget/ancestor decision inside the same
`store.UpdateFile` critical section that writes the entry (compute from the
`existing` bytes and return an error to abort the write), or hold the store's
file lock across decision + record. A test spawning N concurrent spawn-log
invocations at `Consumed == Max-1` asserting exactly one succeeds would pin it.

### WR-06: patrol wrapper markdown edited while its declared YAML source was not (carried forward — OPEN)

**File:** `.aether/commands/patrol.yaml` (still no spawn-orphans step) vs
`.claude/commands/ant/patrol.md:12` (and the ant-patrol/opencode mirrors)
**Issue:** Re-verified unchanged. The wrappers still carry the "also run
`aether spawn-orphans`" step and the "Aether-managed: runtime spec at
`.aether/commands/patrol.yaml`. Synced by aether update." header, but
`patrol.yaml` says nothing about spawn-orphans. Any regeneration/sync from the
YAML spec will silently drop the only user-facing surfacing of the reaper this
phase built.
**Fix:** Add the spawn-orphans guidance to `patrol.yaml` so spec and wrappers
agree, or update the wrapper header if wrappers are now canonical for patrol.

### WR-07: spawnAncestorChain silently truncates on a dangling mid-chain parent (carried forward — OPEN)

**File:** `cmd/spawn_ancestor.go:67-77`
**Issue:** Re-verified unchanged. The walk still breaks silently when an
ancestor's parent name is not in the tree (`parent, ok := byName[...]; if !ok
... break`). An unresolvable *startName* denies (fail-closed), but an
unresolvable *ancestor* yields a shortened chain and an allow — opposite
policies applied to the same class of unreadable evidence. Note the gap-closure
did tighten one related path: a *corrupt* ledger now makes `st.Parse()` return
an error, so a corrupt mid-chain line makes the *whole* walk deny; but a
well-formed ledger that simply omits a named mid-chain parent still truncates
silently.
**Fix:** Distinguish "reached a root sentinel" (complete chain, allow) from
"parent named but unresolvable" (incomplete chain, return an error so
`spawnAncestorCycleReason` denies with "ancestor chain unreadable"). Add a test
with a well-formed tree missing a middle ancestor.

## Info

### IN-01: "73 workers" arithmetic counts the coordinator as a spawned worker (carried forward — OPEN)

**File:** `.aether/workers.md:280-283`; `cmd/spawn_budget.go:16-19`
**Issue:** Re-verified unchanged. "8 workers wide and 2 levels deep — 73
workers in total" counts 1 coordinator + 8 + 64; the budget rule counts only
the 72 spawned helpers, and the coordinator is never recorded.
**Fix:** Say "72 spawned helpers (73 agents counting the coordinator)".

### IN-02: spawn-orphans --clear reports partial success as pure failure (carried forward — OPEN)

**File:** `cmd/spawn_reap.go:277-281`
**Issue:** Re-verified unchanged. On a partial per-entry failure the `--clear`
path discards the reaped names and reports only "reap encountered an error",
even though entries were mutated and budget released.
**Fix:** On partial failure, still render the reaped list and budget delta
alongside the error.

### IN-03: midden entry IDs can collide, and ceiling refusals grow midden.json unboundedly (carried forward — OPEN)

**File:** `cmd/spawn_budget.go:221`
**Issue:** Re-verified unchanged. `midden_%d_%d` (unix seconds + pid) collides
for two refusals in the same second in one process, and every ceiling refusal
appends a new entry with no dedup (compounded by WR-03).
**Fix:** Add a nanosecond/counter component to the ID; dedup or reinforce a
single ceiling entry as pheromones do.

### IN-04: claudeHookInput.SessionID is decoded but never read (carried forward — OPEN)

**File:** `cmd/hook_cmds.go:34`
**Issue:** Re-verified unchanged. `claudeHookInput.SessionID` is populated from
the payload but no code path reads it (the only `SessionID:` assignment at
`cmd/hook_cmds.go:603` is on a different `colony.SessionFile`).
**Fix:** Drop the field, or note the intended future use where it is declared.

### IN-05: sentinel-coverage test walks gitignored local data (carried forward — OPEN)

**File:** `cmd/spawn_enforce_test.go:695-716`
**Issue:** Re-verified unchanged. `TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent`
still walks all of `.aether` (dirs list includes
`filepath.Join(repoRoot, ".aether")`) without excluding the gitignored,
local-only `.aether/data/`, `.aether/dreams/`, `.aether/oracle/` trees. A
developer's local artifacts containing a `spawn-log --parent "..."` line can
fail the test locally while CI stays green.
**Fix:** Skip `.aether/data/`, `.aether/dreams/`, and `.aether/oracle/` during
the walk.

### IN-06: capture appends a bare newline when stdin is empty (carried forward — OPEN, reduced scope)

**File:** `cmd/hook_cmds.go:354-368`
**Issue:** Re-verified. `captureRawHookPayload` now short-circuits on an empty
path (the CR-01 fix removed the file-based switch), but when
`AETHER_HOOK_CAPTURE_FILE` is set and the raw payload is empty it still opens
the file and writes a lone `"\n"` per invocation. Scope is now operator-only
(the env var cannot be set by a worker), so impact is minor.
**Fix:** `if path == "" || len(raw) == 0 { return }`.

---

_Reviewed: 2026-08-13T17:12:33Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard (update review — gap-closure scope + prior-finding re-verification)_
