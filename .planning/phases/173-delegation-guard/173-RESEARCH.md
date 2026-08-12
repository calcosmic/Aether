# Phase 173: Delegation Guard - Research

**Researched:** 2026-08-12
**Domain:** Go CLI runtime guard (recursion/budget enforcement) + Claude Code `PreToolUse` hook integration
**Confidence:** MEDIUM-HIGH (the Go-side chokepoints are read directly from source and are HIGH
confidence; the Claude Code hook/subagent-identity mechanism for SPAWN-04 is MEDIUM confidence —
verified against current official docs but with a real, named correlation gap that the planner
must decide how to bound)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01: Depth cap is two levels of delegation.** The coordinator's own workers may each call
  one round of helpers; those helpers may call nobody. A spawn attempt at the third level is
  refused.
- **D-02: Whole-run helper budget is 20.** Fixed constant. No config key for it this phase.
- **D-03: The 20 counts every helper in the run, including the coordinator's own workers.**
- **D-04: The wave cap and the tree budget are two visibly different quantities.** Budget
  consumed in wave 1 is NOT restored in wave 2 (already fixed by ROADMAP criterion 3 — locked).
- **D-05: The coordinator is depth 0. Its own workers are depth 1. Their helpers are depth 2.
  The guard refuses at depth 3.** The written ruling SPAWN-06 asks for.
- **D-06: `.aether/workers.md:46` and `.aether/workers.md:328` hardcode `--depth 0` for
  children, which is wrong under D-05.** Correcting `workers.md` is in scope.
- **D-07: Under D-05, criterion 2's expected number is fixed.** A 3-deep tree must report depth
  2 from `spawn-tree-depth` (root=0, workers=1, helpers=2). A test must assert the manifest
  worker's recorded depth equals 1.

### Claude's Discretion (strong defaults, adjustable if implementation reveals a genuine conflict — any adjustment must be surfaced, not silently taken)

- **D-08: Refuse the offending spawn only; let everything already running finish.** No run
  abort.
- **D-09: A refused spawn must leave no trace in the spawn tree.** No entry, no budget consumed.
- **D-10: The refusal surfaces in the run's own output**, naming the turned-away helper, its
  would-be parent, and the reason (depth vs. budget vs. ancestor-cycle).
- **D-11: Hitting the whole-run ceiling (D-02) is additionally written to the midden.** A
  routine depth refusal is NOT.
- **D-12: `aether spawn-tree-active` is the on-demand view**, indented by depth with parent
  attribution, must work while a run is in progress, read-only, must not interfere with the run.
- **D-13: Passive awareness without a dashboard.** At 75% of budget (15/20 helpers), a single
  line appears in the run's ordinary output. No live-refreshing dashboard this phase.
- **D-14: Non-technical readability is a hard requirement of the rendered tree.** Caste names,
  worker names, task summaries read as English; no raw JSON/internal identifiers as the primary
  view.
- **D-15: Both an automatic reaper and a manual command.**
- **D-16: The automatic reaper is deliberately conservative.** Generous inactivity threshold;
  must never reap a helper showing activity.
- **D-17: The reaper's threshold is configurable, not hardcoded.** The one place a config key is
  warranted this phase.
- **D-18: Reaping releases budget back to the run.**

### Locked (record, do not re-litigate)

- **D-19: Every delegation guard denies when it cannot resolve what it needs.**
  `spawn-can-spawn-swarm` (`cmd/internal_cmds.go:549-558`) currently returns
  `can_spawn: true, remaining_budget: 5` when `COLONY_STATE.json` cannot be loaded — must be
  inverted.
- **D-20: The `PreToolUse` `Agent|Task` hook denies a spawn whose requester depth cannot be
  resolved.** New matcher plus a deny path on existing hook infrastructure.
- **D-21: Write bounded claims, never absolutes.** If a criterion cannot be stated as a bounded,
  provable claim, name the residue explicitly.
- **D-22: Proof is by execution, never by reading.** Every criterion needs a red-proof.

### Deferred Ideas (OUT OF SCOPE)

- Making the whole-run budget configurable (D-02 explicitly declined it).
- A live-refreshing delegation dashboard (D-13 ships a threshold line instead).
- Reaching `.aether/ts-host/src/spawn-orchestrator.ts` (unreachable, out of scope).
- Phase 172's CR-06 residue (the ~18 uninspected CI steps before the release gate) — tracked as
  Phase 172.1, not this phase.
- `2026-08-01-finalize-reconcile-task-evidence-gate.md` (unrelated area, `cmd/continue`).
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` (different subsystem; adjacent in spirit
  to D-17 but do not silently fold in — surface if a real shared mechanism is found).

**Nothing gains the ability to delegate in this phase.**
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SPAWN-01 | `spawn-can-spawn` refuses at a configured depth, proven by a test asserting `can_spawn: false` | §"The chokepoint", §"Signature widening" — `spawnCanSpawnDecision` var-func seam at `cmd/spawn.go:176`; test pattern already exists at `cmd/spawn_enforce_test.go` |
| SPAWN-02 | A spawned child records its true depth; `spawn-tree-depth` returns non-zero for a nested tree | §"The chokepoint" — derive at `spawn-log` via parent lookup (reuse `spawn-get-depth`'s lookup logic, `cmd/internal_cmds.go:493-535`), ignore caller `--depth` |
| SPAWN-03 | A whole-tree worker budget stops a run that stays within depth limit | §"Two budgets, not one" — run identity already exists (`agent.SpawnRun`, `BeginRun`/`EndRun`, `EntriesForRun`); count entries in the current run window |
| SPAWN-04 | `PreToolUse` hook denies before platform executes, denies when it cannot resolve requester depth | §"The PreToolUse hook" — genuine correlation gap between Claude Code's `agent_id`/`agent_type` and Aether's spawn-tree `AgentName`; two implementation tiers proposed |
| SPAWN-05 | Ancestor-chain repetition (caste + normalised task) refused, ancestor named | §"Ancestor-chain cycle detection" — new pure function over `SpawnTree.Parse()`, no existing helper |
| SPAWN-06 | *(Decision)* written ruling on what depth 0 means | Already resolved by CONTEXT.md D-05/D-06/D-07 — carry forward verbatim, no additional research needed |
| SPAWN-07 | Operator watches the tree grow, indented by depth, parent attribution, live-safe | §"Rendering the tree" — extend `spawn-tree-active` (`cmd/spawn.go:248-294`); locking already safe (`pkg/storage` `RLock`/`Lock`) |
| SPAWN-08 | Abandoned child reaped, budget released, operator command to see/clear orphans | §"Orphan reaping" — no heartbeat signal exists today, only spawn `Timestamp`; reaper is necessarily a wall-clock heuristic |

</phase_requirements>

## Summary

Nearly every mechanism SPAWN-01..05,07,08 needs already exists in some form — this confirms
v1.26's organising finding applies to Phase 173 too. The single real chokepoint is
`cmd/spawn.go:176`'s `spawnCanSpawnDecision` var-func, already swappable and already exercised
by `--enforce`'s deny path. `spawn-log` (`cmd/spawn.go:14-80`) is where depth must actually be
derived — today it stores whatever `--depth` integer the caller passes, and every caller in
`.aether/workers.md` passes the literal string `0`. A parallel, currently-orphaned command,
`spawn-get-depth` (`cmd/internal_cmds.go:493-535`), already contains the exact "look up a named
ant's recorded depth by scanning `spawn-tree.txt`" logic `spawn-log` needs to reuse for parent
lookup. Run identity for the whole-tree budget already exists and is already wired into every
lifecycle command (`agent.SpawnRun`, `BeginRun`/`EndRun`, called from build/continue/colonize/
plan/swarm) — the tree budget does not need new run-tracking infrastructure, only a count of
spawn-tree entries whose activity timestamp falls in the current run's window
(`SpawnTree.EntriesForRun`/`filterEntriesForRun`, already implemented). File locking for
concurrent-safe reads while a run writes already exists (`pkg/storage.Store.ReadFile` takes an
`RLock`; `UpdateFile`/`AtomicWrite` take an exclusive `Lock`).

The one genuinely under-specified piece is SPAWN-04's hook. Official Claude Code docs (fetched
2026-08-12) confirm `PreToolUse` now fires inside subagent contexts and the JSON envelope
carries `agent_id`/`agent_type` for the calling subagent — but there is no field, and no
existing Aether mechanism, that maps Claude Code's ephemeral `agent_id` to Aether's own
persistent spawn-tree `AgentName` ("Hammer-42"). A hook that denies whenever any `agent_id` is
present would over-deny (it would block the legitimate depth-1→depth-2 helper spawn D-01
explicitly allows), so the hook needs a way to distinguish "I am a depth-1 worker" from "I am a
depth-2 helper" using only what Claude Code actually provides. §"The PreToolUse hook" below
proposes a bounded, testable heuristic (`agent_type` against the known first-tier caste names vs.
the `general-purpose` fallback workers.md already documents helpers use) and names its residue
explicitly, per D-21.

Also load-bearing for planning: six spawn-related subcommands
(`spawn-can-spawn`, `spawn-can-spawn-swarm`, `spawn-efficiency`, `spawn-get-depth`,
`spawn-tree-active`, `spawn-tree-depth`) are currently listed in `cmd/testdata/orphan_allowlist.json`
— tolerated orphans under Phase 172's ratchet. Wiring any of them into a real caller this phase
(the hook calling `spawn-can-spawn`'s decision function directly, `spawn-log` reusing
`spawn-get-depth`'s lookup) should be followed by regenerating and shrinking that allowlist
(`--update-orphan-allowlist`) so the wiring proof stays honest. And `cmd/spawn_enforce_test.go`
is already one of five files tracked by `TestWiringGateStepRunsEveryWiringTest` — any new test
function added to that specific file is automatically required (and automatically verified) to
appear in `.github/workflows/ci.yml`'s named `-run` filter. Putting Phase 173's guard tests in
that file, rather than a new file, gets the CI-wiring requirement enforced for free.

**Primary recommendation:** Widen `spawnCanSpawnDecision` to a single struct-based decision
function that reads depth (from a parent lookup, not the caller), whole-run budget (from
`EntriesForRun`), and ancestor-chain (from a new pure walk over `SpawnTree.Parse()`) — one
chokepoint, three checks, tested by substitution exactly as `spawnCanSpawnCmd` already supports.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Depth derivation at spawn time | Go runtime (`cmd/spawn.go` `spawn-log`) | — | The chokepoint an LLM cannot route around; must live in the recording call itself, not in caller-supplied input |
| Whole-run budget accounting | Go runtime (`pkg/agent.SpawnTree` + `cmd/spawn.go`) | — | Needs the same file the depth check reads (`spawn-tree.txt`) and the same run-window logic already built for `Active()` |
| Ancestor-cycle detection | Go runtime (new pure function over `SpawnTree.Parse()`) | — | Pure computation over already-persisted data; no new storage needed |
| Pre-spawn deny before the platform acts | Claude Code `PreToolUse` hook (`cmd/hook_cmds.go`) → Go runtime | — | This is the only chokepoint that can stop the Task tool call itself, before any Aether command runs; everything else is post-hoc (spawn-log/spawn-can-spawn run inside the child's own turn) |
| Tree visualisation | Go runtime (`cmd/spawn.go` `spawn-tree-active`) rendering | Wrapper markdown (narration only) | D-14 requires English rendering; the runtime already owns visual rendering per CLAUDE.md's UX Architecture (`cmd/codex_visuals.go` owns caste identity, not wrapper markdown) |
| Reaping (auto + manual) | Go runtime (`cmd/spawn.go`/new `spawn-reap*` commands) | — | Mutates `spawn-tree.txt`; must go through the same locked `Store` API as every other spawn-tree write |
| Refusal surfacing in run output | Wrapper markdown (`build.md`/`continue.md` narration) + Go runtime (structured refusal reason in JSON envelope) | — | Runtime computes the reason; wrapper is responsible for surfacing it in the ceremony narration the operator sees (matches the existing Wrapper-Runtime Contract) |

## Standard Stack

This phase adds no new external dependencies. Everything is standard-library Go
(`encoding/json`, `strconv`, `strings`, `time`) plus the existing internal packages already
imported by `cmd/spawn.go`: `github.com/calcosmic/Aether/pkg/agent`,
`github.com/calcosmic/Aether/pkg/events`, `github.com/calcosmic/Aether/pkg/storage` (transitively,
via `store`), `github.com/calcosmic/Aether/pkg/colony` (for `ColonyState`, used by
`spawn-can-spawn-swarm` and the hook's redirect logic), and `github.com/spf13/cobra`.

**Installation:** none required.

## Architecture Patterns

### System Architecture Diagram

```
 Claude Code session (main, depth 0, or a subagent, depth >= 1)
        │
        │  about to call the Task tool (spawn a child)
        ▼
 PreToolUse hook  ──────────────────────────────────────────────┐
 (cmd/hook_cmds.go: hookPreToolUseCmd, new matcher "Task")       │
        │                                                        │
        │  can this caller's requester-depth be resolved?        │
        │    - agent_id absent  → depth 0 (main session)         │
        │    - agent_id present → look up via agent_type         │
        │      heuristic (§"The PreToolUse hook")                │
        ▼                                                        │
   resolved? ── no ──► emitHookBlock("requester depth unresolved")│
        │ yes                                                    │
        ▼                                                        │
   within cap (D-01)? ── no ──► emitHookBlock(reason)             │
        │ yes                                                    │
        ▼                                                        │
   Task tool executes, spawns the child subagent                 │
        │                                                        │
        ▼                                                        │
   child's own turn begins; per .aether/workers.md protocol:      │
     1. `aether spawn-can-spawn {depth} --enforce`  ◄─────────────┘
        (cmd/spawn.go: spawnCanSpawnCmd → spawnCanSpawnDecision)
        checks: depth cap (D-01) + whole-run budget (D-02/D-03)
                + ancestor-chain (SPAWN-05)
        │
        ▼  allowed
     2. `aether spawn-log --parent <name> --caste .. --name <child> --task ..`
        (cmd/spawn.go: spawnLogCmd)
        DEPTH IS DERIVED HERE: look up <name> in spawn-tree.txt,
        recorded depth = parent.Depth + 1  (SPAWN-02 — the real chokepoint,
        caller-supplied --depth is ignored)
        │
        ▼
     spawn-tree.txt (pkg/agent.SpawnTree, via pkg/storage.Store —
                      locked read/write, safe for concurrent access)
        │
        ├──► `aether spawn-tree-active` (SPAWN-07, read-only, RLock)
        │      renders indented-by-depth, parent-attributed, English tree
        │
        ├──► `aether spawn-reap` / auto-reaper (SPAWN-08)
        │      scans for entries with Status="spawned"/"active" whose
        │      Timestamp exceeds the configurable threshold, marks
        │      "abandoned", releases budget
        │
        └──► midden.json (D-11: only whole-run-ceiling refusals logged here)
```

### Recommended Project Structure

No new packages. All changes land in existing files plus targeted new files inside `cmd/`:

```
cmd/
├── spawn.go              # spawnCanSpawnDecision widened; spawn-log depth derivation;
│                          # spawn-tree-active rendering upgrade
├── spawn_enforce_test.go # ALREADY tracked by the CI wiring-gate ratchet — put every
│                          # new SPAWN-01..05 guard test here, not a new file
├── spawn_budget.go        # NEW — whole-run budget accounting (D-02/D-03/D-04), reusing
│                          # SpawnTree.EntriesForRun / CurrentRun
├── spawn_ancestor.go       # NEW — ancestor-chain walk + task normalisation (SPAWN-05)
├── spawn_reap.go           # NEW — reaper (SPAWN-08): spawn-reap-scan / spawn-reap / config
├── internal_cmds.go        # spawn-can-spawn-swarm: invert the fail-open default (D-19)
├── hook_cmds.go             # hookPreToolUseCmd: new "Task" matcher + deny path (SPAWN-04)
└── hook_cmds_test.go        # extend with Task-matcher deny/allow cases
pkg/agent/
└── spawn_tree.go            # add "abandoned" to normalizeSpawnStatus/IsTerminalSpawnStatus
                              # if the reaper needs a distinct terminal status (see §"Orphan reaping")
```

### Pattern 1: Package-level `var func` decision seam (already established)

**What:** `spawnCanSpawnDecision` is `var spawnCanSpawnDecision = func(depth int) (bool, string) { ... }`
— a function variable, not a plain function, specifically so tests can substitute a fake
decision for the duration of one test case.
**When to use:** Any place the phase adds a new all-or-nothing guard decision that both
production code and tests need to drive independently.
**Example (existing, `cmd/spawn.go:176`):**
```go
// Source: cmd/spawn.go:170-178 (read directly, HIGH confidence)
var spawnCanSpawnDecision = func(depth int) (bool, string) {
	return true, ""
}
```
**Recommendation:** widen the signature rather than adding parallel seams. Something like:
```go
type spawnDecisionInput struct {
	RequesterName string // the would-be parent, looked up for its recorded depth
	Caste         string
	Task          string
}
type spawnDecisionResult struct {
	Allowed bool
	Reason  string // "depth", "budget", "ancestor-cycle", or "" when allowed
	Detail  string // human-readable, D-10
}
var spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult { ... }
```
This is a breaking signature change for the existing `spawnCanSpawnCmd` positional-depth
contract (`.aether/workers.md:292`'s `aether spawn-can-spawn {your_depth} --enforce}` must keep
resolving). Keep the CLI's own positional/`--depth` argument as the depth the caller CLAIMS to
be at for the pre-check (this is a report, matching D-13's `--enforce` semantics, not the
authoritative depth — the authoritative depth is still only ever set at `spawn-log`). Internally,
the CLI command can look up the caller's OWN name (if it has one available — it usually doesn't;
`spawn-can-spawn` today takes no `--name`/`--parent` flag) — this is a genuine gap: **today
`spawn-can-spawn` cannot verify the claimed depth against a recorded entry, because it is never
told who is asking.** See "Open Questions" — recommend adding an optional `--name` flag so the
CLI check can be upgraded to look up the real recorded depth exactly like `spawn-log` will, while
keeping the positional-depth form working for backward compatibility with `workers.md:292`.

### Pattern 2: Reuse `SpawnTree.Parse()` for all "look up an entry" needs

**What:** Every derived-data command (`spawn-get-depth`, `spawn-tree-depth`, `spawn-efficiency`)
independently re-implements "parse `spawn-tree.txt`, look for field `[3] == name`." Two of these
(`spawn-get-depth` in `cmd/internal_cmds.go`, `spawn-tree-depth` in `cmd/spawn.go`) hand-roll
their own pipe-split parsing instead of using `agent.SpawnTree.Parse()`, which already handles
completion-line merging, status normalisation, and malformed-line skipping.
**When to use:** SPAWN-02's depth-derivation logic and SPAWN-05's ancestor walk should both call
`agent.NewSpawnTree(store, "spawn-tree.txt").Parse()` and search `[]SpawnEntry` by `AgentName`,
not re-parse the file with `splitPipe`/`splitLineFields`.
**Example:**
```go
// Source: pkg/agent/spawn_tree.go:563-569 (read directly, HIGH confidence)
func (st *SpawnTree) Parse() ([]SpawnEntry, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	entries, _, err := st.parseFile()
	return entries, err
}
```

### Pattern 3: Run-window filtering already exists for the whole-run budget

**What:** `filterEntriesForRun` (`pkg/agent/spawn_tree.go:490-522`) already restricts a slice of
`SpawnEntry` to those whose activity timestamp falls inside a given `SpawnRun`'s
started/superseded window. `SpawnTree.Active()` already uses this to scope
`spawn-tree-active`'s output to the current run.
**When to use:** The whole-run budget (D-02/D-03) should count `len(EntriesForRun(currentRunID))`
(or equivalently reuse the same filter `Active()` already applies, but over ALL statuses, not
just live ones, since D-03 counts every helper spawned in the run — including already-completed
ones — not just currently-active ones).
**Example:**
```go
// Source: pkg/agent/spawn_tree.go:180-197, 469-480 (read directly, HIGH confidence)
run, ok, err := st.CurrentRun()
entries, err := st.EntriesForRun(run.ID)
consumed := len(entries) // every helper spawned in this run, live or completed
```
**Caveat (verify before relying on this):** `filterEntriesForRun` windows by
`ActivityTimestamp`/`Timestamp`, which is UTC wall-clock, not a strict causal link to the run ID.
A very fast preceding run ending and a new run beginning in the same second, or clock skew
between processes, is a real (if narrow) edge case — the run boundary is time-based, not
transaction-based. Treat as a bounded approximation (D-21), not an absolute guarantee of "exactly
this run's spawns."

### Anti-Patterns to Avoid

- **Do not add a second budget vocabulary.** `cmd/queen_spawn_budget.go`'s `queenSpawnBudget`
  is a pre-dispatch, per-wave WORKER SELECTION budget (decides which castes get dispatched in a
  wave, before any spawn happens). It is NOT an enforcement gate at spawn-record time and must
  not be confused with, or reused as, the SPAWN-03 whole-run tree budget — they answer different
  questions (D-04 explicitly requires them to stay visibly different quantities).
- **Do not trust `--depth` as authoritative anywhere.** The whole point of SPAWN-02 is that the
  caller-supplied depth is decorative from now on. Any new code that reads `cmd.Flags().GetInt("depth")`
  and treats it as ground truth for a security/budget decision reintroduces the exact defect this
  phase closes.
- **Do not re-parse `spawn-tree.txt` by hand.** Three existing commands already do this
  independently (`spawn-get-depth`, `spawn-tree-depth`, `spawn-can-spawn-swarm`'s budget count);
  adding a fourth hand-rolled parser increases the chance of a fourth divergent bug. Consolidate
  onto `agent.SpawnTree.Parse()`.
- **Do not put new guard tests in a new file** unless there is a strong reason to. `cmd/spawn_enforce_test.go`
  is already tracked by `wiringGateGuardFiles` (`cmd/ci_wiring_gate_test.go:87-93`); a new file
  is invisible to that ratchet unless someone remembers to add it to the list by hand.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-process safe read/write of `spawn-tree.txt` while a run is in progress | A new locking scheme for the reaper/budget code | `pkg/storage.Store.ReadFile` (RLock) / `UpdateFile` (Lock) — already used by `SpawnTree.RecordSpawn`/`UpdateStatus` | Already implements atomic write + advisory file locking (`pkg/storage/lock.go`, `lock_unix.go`, `lock_windows.go`) cross-platform |
| "Which run does this spawn belong to" | A new run-tracking file/format | `agent.SpawnTree.BeginRun`/`EndRun`/`CurrentRun`/`EntriesForRun` (`pkg/agent/spawn_tree.go:92-197`), already called from build/continue/colonize/plan/swarm via `beginRuntimeSpawnRun`/`finishRuntimeSpawnRun` (`cmd/spawn_runs.go`) | Already solves supersession of stale runs, run-ID generation, and time-windowed filtering |
| Caste display name / emoji for the readable tree render (D-14) | New string maps in `spawn.go` | `casteLabel(caste)`, `casteEmoji(caste)` (`cmd/codex_visuals.go:3808-3844`) | Single source of truth for caste identity per CLAUDE.md's UX Architecture; a second map would drift |
| "Look up a named ant's recorded depth" | A fourth hand-rolled pipe parser | `agent.SpawnTree.Parse()` + linear search by `AgentName` (already the shape `spawn-get-depth` uses, just not via the shared type) | Handles completion-line status merging and malformed-line skipping already |

**Key insight:** this phase's actual net-new code surface is small — a widened decision struct,
a budget counter, an ancestor walk, a reaper, and a hook matcher/deny path. Everything else is
consolidating three already-existing, slightly-divergent implementations of "read the spawn
tree" onto the one that's already correct (`agent.SpawnTree`).

## Common Pitfalls

### Pitfall 1: Depth-behavior table in `workers.md` contradicts D-01 beyond the two named lines

**What goes wrong:** CONTEXT.md's D-06 names two specific line-level bugs
(`workers.md:46`, `workers.md:328` hardcoding `--depth 0`). But `.aether/workers.md:263-272`
also contains a full "Depth-Based Behavior" table with its OWN independent depth semantics:
```
| Depth | Role              | Can Spawn?      | Max Sub-Spawns | Behavior |
| 0     | Queen             | Yes             | 4              | ...      |
| 1     | Prime Worker/Builder | Yes          | 4              | ...      |
| 2     | Specialist        | Yes (if surprised) | 2           | focused work, spawn only for unexpected complexity |
| 3     | Deep Specialist   | No              | 0              | complete work inline |
```
Under this table, depth-2 workers ("Specialist") ARE allowed to spawn (reaching depth 3, with a
max of 2 sub-spawns) "if surprised" — this directly contradicts D-01 ("those helpers may call
nobody... Three levels was offered and rejected"). There is also a separate "Global Cap: Maximum
10 workers per phase" line that contradicts D-02's 20.
**Why it happens:** `workers.md` accreted instructions over time from different phases/authors;
D-06's diagnosis (two hardcoded `--depth 0` calls) undercounts the actual scope of the
inconsistency.
**How to avoid:** the plan must rewrite this whole table (or delete it in favor of one clear
statement of D-01/D-05), not just the two named `--depth 0` lines. Also correct the stated
"Global Cap: Maximum 10 workers per phase" line, which is neither D-02's 20 nor consistent with
`queenSpawnBudget`'s per-wave numbers (4-8) — it is a third, orphaned number.
**Warning signs:** any test that asserts `workers.md` no longer contains the string `--depth 0`
will pass while this table's depth-2/"Can Spawn: Yes"/"Global Cap: 10" residue survives untouched.
A content-level test (e.g., asserting `workers.md` states depth-2 entries cannot spawn) is needed,
not just a literal-string search.

### Pitfall 2: `spawn-can-spawn` cannot verify the depth it is told

**What goes wrong:** `spawn-can-spawn` takes a bare integer (`--depth`/positional) and, today,
always allows it. Once `spawnCanSpawnDecision` enforces D-01's cap, a caller can still simply
pass a smaller number than its true depth (`aether spawn-can-spawn 1 --enforce` when it is
actually at depth 2) and sail through — because the command has no way to look up the caller's
real recorded depth; it has no `--name`/`--parent` flag today.
**Why it happens:** `.aether/workers.md:292` documents the invocation as a positional integer
the WORKER supplies about itself, with no cross-check against the spawn tree.
**How to avoid:** either (a) accept this as a documented, bounded residue — the REAL enforcement
is `spawn-log`'s parent-lookup depth derivation (SPAWN-02), which cannot be similarly gamed
without also lying about `--parent`, and `spawn-can-spawn --enforce` is a courtesy pre-check, not
the authoritative gate — or (b) add an optional `--name` flag to `spawn-can-spawn` so it can look
itself up in the tree and ignore the caller-supplied depth entirely when a recorded entry exists
(falling back to the caller-supplied depth, or denying, when it doesn't — a real decision the
planner should make explicitly). Recommendation: (b) is stronger and cheap, but flag it as a
scope decision since `workers.md:292`'s documented invocation (positional-only) must keep
resolving unchanged per WIRE-02.
**Warning signs:** a red-team test spawning at a self-declared shallow depth from an actually-deep
context should be part of the plan's verification if (a) is chosen, so the residue is proven and
named rather than assumed.

### Pitfall 3: The `PreToolUse` hook's `agent_id`/`agent_type` do not identify an Aether spawn-tree entry

**What goes wrong:** it is tempting to assume Claude Code's subagent `agent_id` can be looked up
directly against `spawn-tree.txt`'s `AgentName` field ("Hammer-42"). It cannot — they are
different identifier spaces with no existing bridge. A hook implementation that silently assumes
this correlation exists will either never resolve (over-denying everything) or falsely match
(under-denying, defeating SPAWN-04's purpose).
**Why it happens:** Claude Code assigns `agent_id` internally when the Task tool spawns a
subagent; Aether assigns `AgentName` via `generate-ant-name` and records it via `spawn-log`,
called from the PARENT's own turn. Nothing threads one into the other today.
**How to avoid:** see §"The PreToolUse hook" for the two bounded implementation tiers proposed.
Do not silently assume a correlation that hasn't been verified against a real nested Task call.
**Warning signs:** a hook test that only checks `agent_id` presence/absence but never checks a
real recorded example of `agent_type` for a depth-1 vs depth-2 caste is testing a hypothesis, not
the mechanism — run a real nested-delegation build in Wave 0 and capture the actual hook stdin
JSON before locking in a specific field-matching strategy.

### Pitfall 4: No heartbeat signal exists for "still active" — reaping is a wall-clock heuristic only

**What goes wrong:** `SpawnEntry.ActivityTimestamp` is set at spawn time and only updated again
when `UpdateStatus` (completion) is called — there is no periodic "still working" touch while an
entry remains in `spawned`/`active` status. A reaper that reads `ActivityTimestamp` as a
liveness signal will treat a slow-but-alive worker identically to an abandoned one, because
both look the same: an old `Timestamp`, no completion.
**Why it happens:** the spawn-tree format was designed for start/complete bookkeeping, not
continuous liveness reporting.
**How to avoid:** be explicit in the plan that the reaper threshold (D-17) must be set generously
BECAUSE there is no better signal available, not because a shorter threshold was considered and
rejected on its merits. Do not claim the reaper "detects abandonment" — it detects "has exceeded
elapsed wall-clock time since spawn," which is the honest, bounded (D-21) framing.
**Warning signs:** a test that reaps an entry immediately after spawn (0-elapsed-time) would
prove the reaper is trigger-happy; a test that reaps only past the configured threshold, and
never reaps an entry updated (via `UpdateStatusPreserveActivity` or a fresh completion) inside
the threshold window, is the correct proof shape.

### Pitfall 5: `spawn-can-spawn-swarm`'s fail-open defect and `spawn-can-spawn`'s new fail-closed decision must not diverge

**What goes wrong:** `spawn-can-spawn` (general delegation) and `spawn-can-spawn-swarm`
(`/ant-swarm`'s bug-investigation flow) are two SEPARATE commands with separate, currently
divergent logic (`cmd/spawn.go` vs `cmd/internal_cmds.go:540-595`). D-19 names the
`-swarm` variant's fail-open defect specifically. It would be easy to fix `spawn-can-spawn`'s
fail-closed behavior thoroughly while leaving `-swarm`'s defect only partially addressed, or
vice-versa, since they don't share code today.
**Why it happens:** the two commands evolved independently for different flows (general
delegation vs. swarm bug investigation) and never shared a decision function.
**How to avoid:** fix both explicitly. Do not assume fixing one fixes the other. Consider (as a
planning decision, not a mandate) whether `spawn-can-spawn-swarm` should be refactored to call
the same widened `spawnCanSpawnDecision` seam — but note `spawnCanSpawnSwarmCmd` is a genuinely
orphaned command (no real caller found anywhere in `cmd/`, `.aether/`, or `.claude/` outside its
own registration and test) unlike `spawn-can-spawn`, so merging them changes an untested,
uncalled code path's behavior with no downstream flow to observe the change. Fixing the fail-open
default in place, without necessarily merging the two commands, is the narrower, safer scope for
this phase.
**Warning signs:** grep for `remaining_budget` and `can_spawn` across both files after the change
— if `spawn-can-spawn-swarm` still returns `true`/non-zero-budget on a load error, D-19 is not
actually satisfied even if `spawn-can-spawn` is fixed.

## Code Examples

### The chokepoint today (spawn-log, unmodified) — SPAWN-02's starting point
```go
// Source: cmd/spawn.go:14-79 (read directly, HIGH confidence)
var spawnLogCmd = &cobra.Command{
	Use: "spawn-log",
	RunE: func(cmd *cobra.Command, args []string) error {
		parent, _ := cmd.Flags().GetString("parent")
		// ...
		depth, _ := cmd.Flags().GetInt("depth") // <-- caller-supplied, currently trusted

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn(parent, caste, name, task, depth); err != nil { // <-- depth passed through unchanged
			outputError(2, fmt.Sprintf("failed to record spawn: %v", err), nil)
			return nil
		}
		// ...
	},
}
```
**What must change:** look up `parent` via `agent.NewSpawnTree(store, "spawn-tree.txt").Parse()`,
find the most recent entry with `AgentName == parent`, and set `depth = parentEntry.Depth + 1`.
If `parent` resolves to nothing (D-19 fail-closed): either treat the spawning agent as depth 0
(the coordinator/root case, which legitimately has no recorded parent entry) or refuse the spawn
— the plan must decide which parent-name value the coordinator itself uses (likely a sentinel
like `"Prime-1"` or empty string) and special-case exactly that value as depth 0, denying every
other unresolved parent.

### The reusable lookup logic already written (spawn-get-depth) — reuse via SpawnTree.Parse(), don't re-copy the hand-rolled version
```go
// Source: cmd/internal_cmds.go:493-535 (read directly, HIGH confidence) — the SHAPE to reuse,
// but rewritten against agent.SpawnTree.Parse() instead of hand-rolled splitPipe/splitLines,
// per "Don't Hand-Roll" above.
func lookupRecordedDepth(st *agent.SpawnTree, name string) (depth int, found bool) {
	entries, err := st.Parse()
	if err != nil {
		return 0, false
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].AgentName == name {
			return entries[i].Depth, true
		}
	}
	return 0, false
}
```

### Whole-run budget count (SPAWN-03) — built entirely from existing SpawnTree methods
```go
// Pattern derived from pkg/agent/spawn_tree.go:162-197 (CurrentRun/EntriesForRun, read
// directly, HIGH confidence). Composition below is NEW (not yet written).
func spawnsConsumedThisRun(st *agent.SpawnTree) (int, error) {
	run, ok, err := st.CurrentRun()
	if err != nil || !ok {
		return 0, err // D-19: caller must decide fail-closed behavior when run identity is unknown
	}
	entries, err := st.EntriesForRun(run.ID)
	if err != nil {
		return 0, err
	}
	return len(entries), nil // D-03: every helper counted, live or completed
}
```

### D-19's named defect, and the fix shape
```go
// Source: cmd/internal_cmds.go:549-558 (read directly, HIGH confidence) — the CURRENT,
// fail-open behavior that must be inverted.
var state colony.ColonyState
if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
	// No state = no spawns, budget available   <-- WRONG per D-19
	outputOK(map[string]interface{}{
		"can_spawn":        true,
		"remaining_budget": 5,
		"current_spawns":   0,
	})
	return nil
}
```
**Fix shape:** on load error, return `can_spawn: false` with a named reason (e.g.
`"colony state unreadable: cannot verify spawn budget"`), non-zero exit if `--enforce`-style
behavior applies to this command too (today it doesn't take `--enforce`; confirm whether SPAWN-04/
D-19's "exits non-zero" expectation applies to `-swarm` or is scoped to `spawn-can-spawn` only —
ROADMAP criterion 4 names `spawn-can-spawn-swarm` specifically returning `can_spawn: true` as the
defect, so at minimum the JSON payload's `can_spawn` field must flip to `false`).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Caller-asserted `--depth` trusted at `spawn-log` | Depth derived from parent's recorded entry | This phase (SPAWN-02) | Closes the primary "LLM just lies about its own depth" bypass |
| `spawnCanSpawnDecision(depth int)` | Widened to also read tree budget + ancestor chain | This phase | One chokepoint for all three checks, matching D-04's "visibly different quantities, one decision point" framing |
| `PreToolUse` hook matches `Write\|Edit` only | Adds a `Task` (and possibly `Agent`) matcher | This phase (SPAWN-04) | First hook-level (pre-execution) delegation guard, distinct from the CLI-level (post-execution, inside the child's own turn) guard |

**Deprecated/outdated:** `.aether/workers.md`'s "Depth-Based Behavior" table (lines 263-272) and
its "Global Cap: Maximum 10 workers per phase" line are both superseded by D-01/D-02 and must be
rewritten, not merely have their two `--depth 0` lines patched (see Pitfall 1).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Claude Code's `agent_type` field for a subagent hook event echoes the `subagent_type` value the Task tool was invoked with (e.g. `"aether-builder"` vs `"general-purpose"`) | §"The PreToolUse hook" | If `agent_type` instead reflects something else (a Claude-assigned label unrelated to `subagent_type`), the depth-1-vs-depth-2 heuristic proposed there is unusable and the hook needs a different signal or a stored correlation cache |
| A2 | `PreToolUse` fires for tool calls made from inside a subagent session in the currently-shipping Claude Code version, per the official docs page fetched 2026-08-12 | §"The PreToolUse hook" | A closed GitHub issue (#34692) reported the opposite behavior in an earlier version; if the docs page describes an unreleased or partially-rolled-out feature, the hook may not fire for the depth-1→depth-2 case at all, silently reducing SPAWN-04 to "only ever fires for the coordinator's own Task calls" |
| A3 | Depth-1 workers spawned via a named `aether-*` `subagent_type` never legitimately fall back to `subagent_type="general-purpose"` in a way that would misclassify them as depth-2 under the proposed heuristic | Pitfall/§"The PreToolUse hook" | The command-playbooks docs show a documented fallback ("otherwise use general-purpose and inject the ... role") for at least Probe/Weaver/Gatekeeper/Auditor when a named agent definition isn't available — if this fallback is live in the currently-used `build.md`/`continue.md` (not just the retired playbooks), a legitimate depth-1 worker using the fallback would be denied its one permitted helper spawn |
| A4 | The coordinator's own `spawn-log` calls use a consistent, identifiable parent-name sentinel (e.g. always `"Prime-1"`) that the depth-derivation logic can special-case as depth 0 | §"Code Examples" | If the coordinator's self-identifying name varies (per-command, per-session), the depth-0 special case in `spawn-log`'s derivation logic needs a different anchor (e.g. "parent not found in tree AND caller depth is 0 by convention" vs. a fixed string match) |

## Open Questions

> **Status as of planning, 2026-08-12:** all three are resolved. Q1 and Q3 are resolved
> *factually* — a plan implements a specific answer. Q2 is resolved **procedurally, not
> factually**: we now have a defined way to obtain the answer and a defined response to each
> possible answer, but we do not yet know what the answer is. Do not read Q2 as settled
> knowledge about how the hook behaves.

1. **Should `spawn-can-spawn` gain a `--name` flag so its own report can be verified against a
   recorded entry, rather than trusting the caller's self-declared depth outright?**
   - What we know: today it takes a bare integer with no identity attached, so it cannot
     cross-check anything (Pitfall 2). `spawn-log` (the real chokepoint) will not have this
     weakness once SPAWN-02 lands.
   - What's unclear: whether adding `--name` this phase is in scope, or whether the residue
     (self-declared depth going into the pre-check, real depth enforced later at `spawn-log`) is
     an acceptable bounded gap to name and defer.
   - Recommendation: treat as Claude's discretion at planning time; either is defensible, but
     whichever is chosen must be stated as a decision in the plan, not left implicit.
   - **RESOLVED (factually) by plan 04.** `--name` is added as an optional flag on
     `spawn-can-spawn`, registered in `init()` with help text
     `Requester's recorded agent name; when it resolves, the recorded depth overrides --depth`.
     When it resolves via `latestSpawnEntryByName`, `RequesterDepth` comes from the recorded
     entry and `DepthIsAuthoritative` is true; omitting it keeps the documented invocation
     working and stays advisory. The residual advisory gap is carried as residue 1 in
     `173-RESIDUE.md` (plan 10 task 4) and as Known Residue 1 in `173-VALIDATION.md`.

2. **Does the `PreToolUse` hook actually fire for genuinely nested (depth-1 worker spawning a
   depth-2 helper) Task calls in the Claude Code version this repo's users run, and does
   `agent_type` carry the value needed for the proposed heuristic?**
   - What we know: official docs (fetched 2026-08-12) say yes to both; a closed GitHub issue
     says no to the first, from an unspecified earlier version.
   - What's unclear: ground truth on the actual runtime this phase ships against, since this
     research session could not execute a live nested Task call to observe the real hook stdin
     payload.
   - Recommendation: make this a Wave 0 verification step — dispatch one real two-level nested
     spawn in a scratch colony, capture the actual `hook-pre-tool-use` stdin JSON (e.g., have the
     hook log its raw input to a scratch file when an env var is set), and confirm the field
     shape before finalizing the hook's matching logic. This is exactly the kind of empirical
     check the project's Definition of Done culture expects before locking in a mechanism whose
     correctness depends on an external platform's undocumented-until-now behavior.
   - **RESOLVED PROCEDURALLY by plan 01 — not factually.** Plan 01 is this phase's Wave 0
     gate: it dispatches a real two-level nested spawn in a scratch colony, captures the actual
     `hook-pre-tool-use` stdin JSON, and writes a dated `VERDICT:` line to
     `173-HOOK-FINDINGS.md`. Plan 07 task 1 opens by reading that verdict and branching on it:
     `INCONCLUSIVE` stops the plan and reports to the operator rather than implementing;
     `HOOK_DOES_NOT_FIRE_IN_SUBAGENT` implements but bounds every comment and test name to the
     coordinator's own dispatches; `HOOK_FIRES_IN_SUBAGENT` uses the field names the capture
     actually shows, correcting `claudeHookInput`'s struct tags before any logic is written
     against them. So the *process* is settled and the stop condition is explicit. **The
     ground-truth answer is still unknown** and stays unknown until plan 01 runs — which is why
     `wave_0_complete` remains false in `173-VALIDATION.md`, and why the identity-bridge gap is
     carried as residue 3 in `173-RESIDUE.md` regardless of which way the verdict lands.

3. **Where should the reaper's configurable threshold (D-17) live?**
   - What we know: no existing colony-level runtime config file/pattern exists for this kind of
     knob; `.planning/config.json` found via `learning.go`/`gate.go` is this repo's OWN GSD
     planning config (for building Aether), not a downstream colony's Aether runtime config.
   - What's unclear: whether to add a new field to `COLONY_STATE.json` (consistent with how
     `ColonyDepth`/verification depth are already stored there) or a small dedicated file (e.g.
     `.aether/data/spawn-reaper-config.json`).
   - Recommendation: a new optional field on `colony.ColonyState` (e.g.
     `SpawnReapThresholdMinutes *int`, `omitempty`, defaulting to a generous value such as 120
     minutes when unset) is the more consistent choice, since it's colony-scoped state exactly
     like the fields it would sit beside.
   - **RESOLVED (factually) by plan 09.** The recommendation was taken: the threshold lives on
     `colony.ColonyState` as `SpawnReapThresholdMinutes`, read by
     `spawnReapThresholdMinutes()` with a generous default when unset. No new config file was
     added. Recorded as settled in `173-VALIDATION.md` § Wave 0 Requirements.

## Environment Availability

No external tools, services, or runtimes beyond the repo's own Go toolchain and Claude Code
itself are required. Go toolchain and `go test`/`go build`/`go vet` availability is already
assumed by every other phase in this milestone and by the CI workflow this research read
directly (`.github/workflows/ci.yml` uses `go-version: "1.26"`).

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All new/changed `cmd/`, `pkg/agent/` code | ✓ (assumed, per CI config) | 1.26 (per `ci.yml`) | — |
| Claude Code `PreToolUse` hook subagent support | SPAWN-04 | Unverified in this session (see Open Question 2) | — | If subagent-context hooks don't fire as documented, the hook mechanism only covers the coordinator's own Task calls; the CLI-level `spawn-can-spawn`/`spawn-log` chokepoints remain the enforcement backstop for the depth-1→depth-2 case regardless |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), existing `cmd` package test suite |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./cmd/... -run 'TestSpawn|TestHookPreToolUse' -count=1 -v` |
| Full suite command | `go test ./... -count=1 -timeout 900s` (matches `ci.yml`'s "Run Go tests" step) |

### Phase Requirements → Test Map

Every row below is a **red-proof**: break the guard, watch the named test fail, restore — per
D-22. All new tests should live in `cmd/spawn_enforce_test.go` (already tracked by
`wiringGateGuardFiles`, see Pattern/Anti-Pattern above) except the hook test, which extends
`cmd/hook_cmds_test.go`.

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SPAWN-01 | `spawn-can-spawn --depth 99 --enforce` returns `can_spawn:false` and a non-zero exit, naming the reason | unit | `go test ./cmd -run 'TestSpawnCanSpawnDeniesPastDepthCap' -count=1 -v` | ❌ Wave 0 — extend `spawn_enforce_test.go` |
| SPAWN-01 | Scripted spawn one level past the cap exits non-zero AND writes no `spawn-tree.txt` entry (D-09) | unit/integration | `go test ./cmd -run 'TestSpawnLogRefusesPastCapWithNoTreeEntry' -count=1 -v` | ❌ Wave 0 |
| SPAWN-02 | A 3-deep nested spawn chain (each level calling `spawn-log --parent <true-parent>`) reports `spawn-tree-depth` == 2 (D-07's fixed number) | unit | `go test ./cmd -run 'TestSpawnTreeDepthDerivedFromParentChain' -count=1 -v` | ❌ Wave 0 |
| SPAWN-02 | A caller-supplied `--depth` value is ignored when a parent entry resolves | unit | `go test ./cmd -run 'TestSpawnLogIgnoresCallerSuppliedDepth' -count=1 -v` | ❌ Wave 0 |
| SPAWN-03 | With wave cap 8 and tree budget 20, worker 21 in a single run is refused even though no single wave exceeded 8; budget consumed in wave 1 is not restored in wave 2 | unit/integration | `go test ./cmd -run 'TestSpawnTreeBudgetNotRestoredBetweenWaves|TestSpawnTreeBudgetRefusesAtTwentyFirst' -count=1 -v` | ❌ Wave 0 |
| SPAWN-04 | `hook-pre-tool-use` denies a `Task` call when the requester's depth cannot be resolved (e.g. `agent_id` present, `agent_type` unresolvable) | unit | `go test ./cmd -run 'TestHookPreToolUseDeniesUnresolvedRequesterDepth'` | ❌ Wave 0 — extend `hook_cmds_test.go` |
| SPAWN-04 | `hook-pre-tool-use` allows a Task call from the main session (no `agent_id`) and from a recognized depth-1 caste `agent_type`; denies from a `general-purpose` `agent_type` | unit | `go test ./cmd -run 'TestHookPreToolUseAllowsMainSessionAndDepthOneWorker|TestHookPreToolUseDeniesGeneralPurposeHelper'` | ❌ Wave 0 |
| SPAWN-05 | An A→B→A chain with the same (caste, normalised task) is refused with the ancestor named | unit | `go test ./cmd -run 'TestSpawnCanSpawnDeniesAncestorCycle'` | ❌ Wave 0 |
| SPAWN-06 | *(decision, not build)* — a test asserting `spawn-log`'s recorded depth for a manifest worker equals 1 doubles as the SPAWN-06 proof (D-07) | unit | covered by the SPAWN-02 tests above | ❌ Wave 0 |
| SPAWN-07 | `spawn-tree-active` renders indented-by-depth output with parent attribution while entries remain in "spawned"/"active" status (i.e. mid-run) | unit | `go test ./cmd -run 'TestSpawnTreeActiveRendersIndentedByDepth'` | ❌ Wave 0 |
| SPAWN-08 | An entry past the configured reap threshold, still "spawned", is reaped: status becomes terminal, budget count decreases, a fresh (in-threshold) entry is untouched | unit | `go test ./cmd -run 'TestSpawnReapReleasesBudgetForStaleEntryOnly'` | ❌ Wave 0 |
| SPAWN-08 | A manual command lists and clears orphans on demand | unit | `go test ./cmd -run 'TestSpawnOrphansListAndClear'` | ❌ Wave 0 |
| ROADMAP criterion 4 (D-19) | With `COLONY_STATE.json` and the spawn tree made unreadable, every delegation guard (`spawn-can-spawn`, `spawn-can-spawn-swarm`, the hook) denies and exits non-zero | integration | `go test ./cmd -run 'TestAllDelegationGuardsFailClosedOnUnreadableState'` | ❌ Wave 0 |
| CI wiring | Every new guard test function is present in `.github/workflows/ci.yml`'s named `-run` filter | ratchet (already exists) | `go test ./cmd -run 'TestWiringGateStepRunsEveryWiringTest'` | ✅ exists (`cmd/ci_wiring_gate_test.go:111`) — enforces the addition automatically IF new tests land in `cmd/spawn_enforce_test.go` |
| Wiring proof | `spawn-can-spawn`, `spawn-get-depth`, `spawn-tree-active`, `spawn-tree-depth` are no longer orphans once wired to real callers | ratchet | regenerate via `go run ./cmd/aether ... --update-orphan-allowlist` (or the test-suite equivalent) then `go test ./cmd -run 'TestOrphanAllowlistOnlyShrinks|TestNoRegisteredSubcommandIsUnreferenced'` | ✅ mechanism exists; requires the plan to actually regenerate + commit the shrunk allowlist |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run 'TestSpawn|TestHookPreToolUse' -count=1 -v`
- **Per wave merge:** `go test ./... -count=1 -timeout 900s` (matches CI's "Run Go tests" step)
- **Phase gate:** full suite green, plus the two ratchets above (`TestWiringGateStepRunsEveryWiringTest`,
  `TestOrphanAllowlistOnlyShrinks`) green, before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `cmd/spawn_enforce_test.go` — add every SPAWN-01/02/03/05 unit test named above
- [ ] `cmd/hook_cmds_test.go` — add SPAWN-04 hook test cases (Task matcher allow/deny)
- [ ] `cmd/spawn_budget_test.go` or inline in `spawn_enforce_test.go` — SPAWN-03 whole-run budget
      tests (needs a fixture that begins a run, records N spawns via `spawn-log`, asserts refusal
      at N+1)
- [ ] `cmd/spawn_reap_test.go` or inline — SPAWN-08 reaper tests (needs a fixture that fakes an
      old `Timestamp` on a `spawned`-status entry without waiting real wall-clock time — likely
      requires a small seam, e.g. an injectable "now" function, since `RecordSpawn` currently
      calls `time.Now().UTC()` directly with no clock injection point)
- [ ] Framework install: none — Go stdlib `testing` already covers this, no new test framework
- [ ] **Empirical verification (Wave 0, before locking the hook's matching logic):** dispatch one
      real two-level nested Task spawn and capture the actual `PreToolUse` stdin JSON to confirm
      A1/A2/A3 in the Assumptions Log, per Open Question 2

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | no | No new authentication surface — this phase does not add user-facing auth |
| V3 Session Management | no | Claude Code's own session/subagent model is out of Aether's control surface; this phase only reads hook-provided session fields, does not manage sessions |
| V4 Access Control | yes | The entire phase IS an access-control mechanism (who may spawn whom, how deep, how many) — see Known Threat Patterns below |
| V5 Input Validation | yes | `--parent`/`--name`/`--task` flags are already sanitized on write (`sanitizeSpawnField` strips `\r`/`\n`/`|`, `pkg/agent/spawn_tree.go:422-427`); no new unvalidated input surface is added, but the new ancestor-chain "normalised task" comparison (SPAWN-05) is a new text-processing surface worth a bounded test |
| V6 Cryptography | no | No cryptographic operations in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| A spawned agent lies about its own depth to bypass the cap | Elevation of Privilege | Depth derived from parent lookup (SPAWN-02), not caller-asserted — closes the direct version of this; residual "lie about who your parent is" surface named in Pitfall 2/Open Question 1 |
| A spawned agent floods the tree budget by spawning many cheap/no-op children before doing real work | Denial of Service (resource exhaustion, against the operator's own spend, not a remote attacker) | Whole-run budget (SPAWN-03), refusal writes no entry so refusals themselves cannot consume budget (D-09) |
| An A→B→A caste/task cycle spins without ever exceeding the depth cap (each hop only 1 level deep, but repeating) | Denial of Service | Ancestor-chain repetition check (SPAWN-05) |
| A guard silently allows when its data source (COLONY_STATE.json, spawn-tree.txt) is unreadable, corrupted, or mid-write | Tampering / Elevation of Privilege | Fail-closed on every guard (D-19); this is a genuine, previously-shipped defect (`spawn-can-spawn-swarm`), not a hypothetical |
| A hook that cannot identify its caller defaults to allow (fail-open) rather than deny | Elevation of Privilege | D-20 mandates deny-on-unresolved; §"The PreToolUse hook" — the correlation gap between Claude's `agent_id` and Aether's `AgentName` makes "cannot resolve" the LIKELY common case for anything beyond the coordinator's own calls unless the heuristic in this research is verified and adopted |

## Sources

### Primary (HIGH confidence)
- `cmd/spawn.go` (read directly) — `spawnLogCmd`, `spawnCanSpawnDecision`, `spawnCanSpawnCmd`,
  `spawnTreeActiveCmd`, `spawnTreeDepthCmd`
- `cmd/spawn_track.go`, `cmd/spawn_runs.go` (read directly)
- `pkg/agent/spawn_tree.go` (read directly, full file) — `SpawnEntry`, `SpawnRun`, `SpawnTree`,
  `BeginRun`/`EndRun`/`CurrentRun`/`EntriesForRun`/`Active`/`Parse`
- `cmd/queen_spawn_budget.go` (read directly, full file) — confirmed per-wave, pre-dispatch
  budget, distinct from the tree budget this phase adds
- `cmd/hook_cmds.go` (read directly, full file) — `hookPreToolUseCmd`, `claudeHookInput`,
  `.claude/settings.json`'s current `PreToolUse` matcher (`Write|Edit` only)
- `cmd/internal_cmds.go:470-595` (read directly) — `spawnGetDepthCmd`, `spawnCanSpawnSwarmCmd`
  including the named D-19 fail-open defect at lines 549-558
- `cmd/spawn_enforce_test.go` (read directly, full file) — existing test pattern for substituting
  `spawnCanSpawnDecision`, and the WIRE-02 invocation-tracking test
- `cmd/ci_wiring_gate_test.go` (read directly, relevant sections) — `wiringGateGuardFiles`
  (confirms `spawn_enforce_test.go` is already tracked), `TestWiringGateStepRunsEveryWiringTest`
- `cmd/subcommand_reachability_ratchet_test.go` (read directly, relevant sections) —
  `TestOrphanAllowlistOnlyShrinks`, allowlist mechanics
- `cmd/testdata/orphan_allowlist.json` (read directly, grepped) — confirms `spawn-can-spawn`,
  `spawn-can-spawn-swarm`, `spawn-efficiency`, `spawn-get-depth`, `spawn-tree-active`,
  `spawn-tree-depth` are currently tolerated orphans
- `.github/workflows/ci.yml` (read directly, full relevant range) — the named `-run` filter step,
  the broad `go test ./...` step
- `pkg/storage/storage.go` (read directly, relevant sections) — `Store.ReadFile` (RLock),
  `Store.UpdateFile`/`AtomicWrite` (Lock)
- `.aether/workers.md` (read directly, relevant sections) — the Spawn Tracking snippet, the Step-
  by-Step Spawn Protocol, and the "Depth-Based Behavior" table (Pitfall 1)
- `.claude/settings.json` (read directly) — current hook matcher configuration
- `.planning/phases/173-delegation-guard/173-CONTEXT.md`, `.planning/REQUIREMENTS.md`,
  `.planning/ROADMAP.md` (read directly) — locked decisions, requirement text, success criteria
- `https://code.claude.com/docs/en/hooks` (fetched 2026-08-12) — confirmed current documented
  behavior: `PreToolUse` fires inside subagent contexts; `agent_id`/`agent_type` fields present

### Secondary (MEDIUM confidence)
- `cmd/codex_visuals.go` (grepped, not fully read) — confirmed `casteLabel`/`casteEmoji` function
  existence and location for D-14 reuse, did not read full implementation

### Tertiary (LOW confidence, flagged for validation)
- GitHub issue `anthropics/claude-code#34692` (fetched 2026-08-12, closed as not planned) —
  reports `PreToolUse`/`PostToolUse` do NOT fire for subagent tool calls; conflicts with the
  current official docs page. Treated as evidence of a past/possibly-still-partial gap, not as
  current ground truth — see Assumption A2 and Open Question 2. This conflict is the single
  most consequential unresolved fact for SPAWN-04's design and should be empirically verified in
  Wave 0 before implementation, not assumed either way.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all findings read directly from source
- Architecture (Go-side chokepoints, budget, ancestor chain, reaper): HIGH — every mechanism
  traced to specific, currently-read source lines
- Architecture (PreToolUse hook / SPAWN-04): MEDIUM — official docs are current and specific, but
  a conflicting historical GitHub issue and the absence of any existing `agent_id`↔`AgentName`
  correlation mean the concrete implementation needs a Wave 0 empirical check before the design
  is locked
- Pitfalls: HIGH — five concrete, source-verified pitfalls, each with a specific test-shape
  recommendation
- Validation architecture: HIGH — reuses an already-verified-working test/ratchet pattern from
  Phase 172; every new test is scoped to a single named requirement

**Research date:** 2026-08-12
**Valid until:** 2026-08-19 (7 days) for the SPAWN-04 hook-behavior claims specifically (fast-
moving external platform behavior, actively disputed between docs and a GitHub issue); 2026-09-11
(30 days) for the Go-side chokepoint findings (stable, internally-controlled code)
