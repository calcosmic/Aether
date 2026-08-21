# Phase 173: Delegation Guard - Context

**Gathered:** 2026-08-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Recursive delegation is bounded by the runtime at the one chokepoint an LLM cannot route
around — the spawn-recording call. Depth is derived from the parent's recorded entry rather
than asserted by the caller, a whole-tree budget bounds what depth alone cannot, every guard
fails closed, and the operator can watch the tree while it grows.

**Nothing gains the ability to delegate in this phase.** Enforcement lands before capability
because parent/depth linkage is recorded at spawn time and cannot be retrofitted to past runs.

Requirements: SPAWN-01 … SPAWN-08 (SPAWN-06 is a decision to record, not an implementation
to plan).

**Out of scope:** granting any new delegation capability; the ts-host recursion policy engine
at `.aether/ts-host/src/spawn-orchestrator.ts` (unreachable today — reaching it is a later
phase's problem, this phase must not depend on it); anything from Phase 172's CR-06 residue.

</domain>

<decisions>
## Implementation Decisions

### The limits themselves (user-decided)

- **D-01: Depth cap is two levels of delegation.** The coordinator's own workers may each
  call one round of helpers; those helpers may call nobody. A spawn attempt at the third
  level is refused. Three levels was offered and rejected as too hard to reason about;
  "nobody spawns at all" was offered and rejected as removing a capability the repo already
  instructs workers to use (`.aether/workers.md:328`).
- **D-02: Whole-run helper budget is 20.** Chosen over ~40 and over "make it adjustable".
  The user explicitly rejected making it adjustable at this stage — *a limit you can raise
  under pressure is a limit that gets raised.* Plan for a fixed constant; do not add a config
  key for it in this phase.
- **D-03: The 20 counts every helper in the run, including the coordinator's own workers.**
  Chosen over counting only helpers-of-helpers. Rationale accepted verbatim: a limit you have
  to do arithmetic on to understand is a limit that gets misread. So a wave of 8 workers has
  already consumed 8 of the 20.
- **D-04: The wave cap and the tree budget are two visibly different quantities.** D-01 and
  D-02 are not one number doing double duty. Depth alone is not a budget: 8-wide × 2-deep is
  73 workers while never breaking the depth rule. Budget consumed in wave 1 is NOT restored
  in wave 2 (already fixed by ROADMAP criterion 3 — treat as locked, not open).

### Depth counting convention — SPAWN-06 (Claude's discretion, flagged to user, not objected)

- **D-05: The coordinator is depth 0. Its own workers are depth 1. Their helpers are depth 2.
  The guard refuses at depth 3.** This is the written ruling SPAWN-06 asks for.
- **D-06: The repo's current instructions are internally contradictory and one of them is
  wrong.** `.claude/commands/ant/build.md:269` sends `--depth 1` for the coordinator's own
  manifest workers (correct under D-05). `.aether/workers.md:46` and `.aether/workers.md:328`
  send `--depth 0` for workers' *children* (wrong under D-05 — children are deeper than
  parents, not shallower). Correcting `workers.md` is in scope for this phase.
- **D-07: Under D-05, criterion 2's expected number is fixed.** A 3-deep tree must report
  depth 2 from `spawn-tree-depth` when depth is derived rather than asserted (root=0,
  workers=1, helpers=2). A test must assert the manifest worker's recorded depth equals 1.

### What happens at the limit (Claude's discretion — user paused, delegated these)

- **D-08: Refuse the offending spawn only; let everything already running finish.** Do NOT
  abort the whole run. Rationale: preserves usable work and matches ROADMAP criterion 1's own
  wording — *"a scripted spawn one level past the cap exits non-zero **and writes no
  spawn-tree entry**"* — which describes a per-spawn refusal, not a run abort. Advisory-only
  ("warn and let it through") was explicitly rejected: an advisory limit does not stop a
  runaway, which is the entire point of the phase.
- **D-09: A refused spawn must leave no trace in the spawn tree.** No entry, no budget
  consumed. A refusal that still books budget would let repeated refusals exhaust the run.
- **D-10: The refusal surfaces in the run's own output**, naming which helper was turned away,
  its would-be parent, and the reason (depth vs. budget vs. ancestor-cycle). No new place for
  the operator to check.
- **D-11: Hitting the whole-run ceiling (D-02) is additionally written to the midden** (the
  project's record of things that went wrong) so the system can learn the pattern. A routine
  depth refusal is NOT — it is expected behaviour, not a fault. This split was the stated
  recommendation and is deliberate: do not log every refusal to the failure record.

### Watching it while it runs — SPAWN-07 (Claude's discretion)

- **D-12: `aether spawn-tree-active` is the on-demand view**, rendering the tree indented by
  depth with parent attribution, and it must work **while a run is in progress** — that is the
  requirement, not a nice-to-have. It must be read-only and must not interfere with the run.
- **D-13: Passive awareness without a dashboard.** When a run crosses a threshold of its
  budget (suggest 75% of 20 = 15 helpers), a single line appears in the run's ordinary output.
  The operator gets warned without having to babysit a second window. Do not build a
  live-refreshing dashboard in this phase.
- **D-14: Non-technical readability is a hard requirement of the rendered tree.** The operator
  is non-technical. Caste names, worker names and task summaries must read as English; do not
  render raw JSON or internal identifiers as the primary view.

### Abandoned helpers — SPAWN-08 (Claude's discretion)

- **D-15: Both an automatic reaper and a manual command**, as SPAWN-08 requires: an abandoned
  child is reaped and its budget released, AND the operator has a command to see and clear
  orphans.
- **D-16: The automatic reaper is deliberately conservative.** A generous inactivity threshold,
  and it must never reap a helper showing activity. Killing live work is worse than a ghost
  holding budget for a while — the failure modes are not symmetric.
- **D-17: The reaper's threshold is configurable, not hardcoded.** This is the one place a
  config key is warranted (contrast D-02, where the user rejected configurability). Reason:
  the right threshold depends on the model and machine in a way the depth and budget caps do
  not. See the reviewed-but-not-folded todo in `<deferred>` — the repo already has a known
  complaint about a hardcoded, unconfigurable timeout in this general area.
- **D-18: Reaping releases budget back to the run.** Otherwise the tree budget fills with
  ghosts and progressively locks the colony out of spawning — the exact harm SPAWN-08 names.

### Fail-closed behaviour (locked by ROADMAP criterion 4 — record, do not re-litigate)

- **D-19: Every delegation guard denies when it cannot resolve what it needs.** Specifically
  named defect: `spawn-can-spawn-swarm` at `cmd/internal_cmds.go:549-558` currently returns
  `can_spawn: true, remaining_budget: 5` when `COLONY_STATE.json` cannot be loaded. It fails
  in the wrong direction and must be inverted.
- **D-20: The `PreToolUse` `Agent|Task` hook denies a spawn whose requester depth cannot be
  resolved.** The hook mechanism already exists and is wired (see `<code_context>`); this is
  a new matcher plus a deny path, not new infrastructure.

### How this phase's success criteria must be written (carried from Phase 172)

- **D-21: Write bounded claims, never absolutes.** Phase 172 ran four build-verify rounds
  because two of its criteria were written as absolutes over an adversarial surface, which has
  no defined edge. See `.planning/phases/172-wiring-proof/172-STOP-RULE.md`. If a criterion
  here cannot be stated as a bounded, provable claim, name the residue explicitly rather than
  implying coverage.
- **D-22: Proof is by execution, never by reading.** Per `CLAUDE.md`'s Definition of Done: a
  requirement is satisfied only when a command exists that someone can run, and that command
  fails when the requirement is unmet. Every criterion here needs a red-proof — break it,
  watch the named test fail, restore.

### Claude's Discretion

The user selected all four discussion areas, completed "The limits themselves" (D-01 … D-04),
then paused and explicitly delegated the remaining three areas with the instruction to write
up recommendations and proceed to planning. **D-08 … D-18 are therefore Claude's calls, not
the user's**, and the user should be able to push back on any of them once the behaviour is
visible. Planner and executor should treat D-01 … D-07 as locked and D-08 … D-18 as strong
defaults that may be adjusted if implementation reveals a genuine conflict — but any such
adjustment must be surfaced, not silently taken.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` § "Phase 173: Delegation Guard" — the goal, the six success criteria,
  and the SPAWN-06 decision note. Criterion 3's "wave cap 8 / tree budget 20" is the origin of
  D-02.
- `.planning/REQUIREMENTS.md` lines 40-47 — SPAWN-01 … SPAWN-08 verbatim, and lines 142-149 for
  traceability rows that must move from `Pending` to satisfied.
- `.planning/ROADMAP.md` § "v1.26 Intelligent Orchestration" — the milestone's organising
  finding: *most of this already exists and was never wired to a caller.* The framing is
  **switch on and prove**, not **design and build**.

### Governing standards (non-negotiable)
- `CLAUDE.md` § "Definition of Done" — a requirement is satisfied only when a command exists
  that fails when the requirement is unmet. Also its corollaries on `--dry-run` purity and on
  preferring invariant/proportion assertions over section-presence assertions.
- `CLAUDE.md` § "Communication Style" and the repo-vocabulary translation table — the owner is
  non-technical; this governs D-14 and all user-facing output from this phase.
- `.planning/phases/172-wiring-proof/172-STOP-RULE.md` — why absolutes over adversarial
  surfaces do not terminate. Governs D-21.

### The code this phase changes
- `cmd/spawn.go:175-218` — `spawnCanSpawnDecision` and the `spawn-can-spawn` command.
- `cmd/spawn.go:248-294` — `spawn-tree-active` (SPAWN-07's command).
- `cmd/spawn.go:296-315+` — `spawn-tree-depth` (SPAWN-02).
- `cmd/internal_cmds.go:541-595` — `spawn-can-spawn-swarm`, including the fail-open defect at
  lines 549-558 named in D-19.
- `cmd/queen_spawn_budget.go` — the existing per-wave budget (`queenSpawnBudget`, `MaxWorkers`,
  `queenBuildSafetyRequiredCastes`). The wave cap already exists; the tree budget is new.
- `cmd/hook_cmds.go:29` (`hookPreToolUseCmd`) and `cmd/hook_cmds.go:471` (registration), plus
  `.claude/settings.json` § `PreToolUse` — the hook seam for D-20.
- `cmd/spawn_track.go`, `cmd/spawn_runs.go`, `pkg/agent` (`agent.NewSpawnTree`,
  `agent.SpawnEntry`) — where spawn entries are recorded and parsed.

### Instruction files that are wrong today and must be corrected
- `.aether/workers.md:46` and `.aether/workers.md:328` — send `--depth 0` for children. Wrong
  under D-05.
- `.claude/commands/ant/build.md:269` — sends `--depth 1` for manifest workers. Correct under
  D-05; keep.

### Explicitly out of scope but referenced
- `.aether/ts-host/src/spawn-orchestrator.ts` — the recursion policy engine that exists and is
  unreachable. Do NOT make this phase depend on it.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **A designated seam already exists for the entire cap decision.** `cmd/spawn.go:176`:
  ```go
  // Phase 173 to implement the real cap logic.
  var spawnCanSpawnDecision = func(depth int) (bool, string) {
      return true, ""
  }
  ```
  It is a package-level `var` function — swappable, and already testable by substitution. This
  is the one chokepoint. **The signature will need widening** (it takes only `depth`, but D-02
  and SPAWN-05 need tree-budget and ancestor-chain awareness too).
- **The refusal machinery around it already works.** `cmd/spawn.go:200-210` already honours
  `--enforce` and exits non-zero with a reason when `canSpawn` is false. Only the decision is
  a stub. `cmd/spawn_enforce_test.go:202` already asserts a `can_spawn: false` path exists.
- **The per-wave budget already exists** in `cmd/queen_spawn_budget.go`, including a
  safety-required-caste bypass that must not be broken. Reuse its shape for the tree budget
  rather than inventing a second vocabulary.
- **`PreToolUse` hook infrastructure exists and is wired** (`cmd/hook_cmds.go:29`,
  registration at `:471`, `.claude/settings.json`). D-20 adds a matcher and a deny path, not
  a new subsystem.
- **`spawn-tree-active` already renders active entries with parent, caste, task, depth and
  status** (`cmd/spawn.go:265-291`). SPAWN-07 needs indentation-by-depth and in-progress
  usability, not a new command.

### Established Patterns

- Positional-and-flag dual input: `spawn-can-spawn` accepts both `aether spawn-can-spawn 5
  --enforce` (as `.aether/workers.md:292` instructs) and `--depth 5` (as the build playbooks
  send). Positional wins. Preserve this — Phase 172's WIRE-02 exists because that exact
  invocation was broken.
- `outputOK` / `outputError` JSON envelope; `AETHER_OUTPUT_MODE=json`.
- Guard tests are named, and wired into `.github/workflows/ci.yml:100`'s `-run` filter. Any
  new guard test from this phase must be added to that filter or it does not run — Phase 172
  built the ratchet that catches exactly this omission.

### Integration Points

- **`spawn-log` is the recording call and therefore the real chokepoint** — depth must be
  *derived from the parent's recorded entry* at this point, and the caller-supplied `--depth`
  ignored (SPAWN-02). This is what makes the guard un-routearoundable by an LLM: an LLM can
  lie about its depth, but it cannot avoid being recorded.
- `spawn-tree.txt` via `agent.NewSpawnTree(store, "spawn-tree.txt")` is the persisted tree.
- `COLONY_STATE.json` is the state whose unreadability currently triggers the fail-open in
  D-19.

</code_context>

<specifics>
## Specific Ideas

- The user's stated reasoning for rejecting a configurable whole-run budget, worth preserving
  verbatim because it generalises: **"a limit you can raise under pressure is a limit that
  gets raised."**
- The user's stated reasoning for D-03: **"a limit you have to do arithmetic on to understand
  is a limit that gets misread."**
- The 8-wide × 2-deep = 73 workers arithmetic was what made D-04 land. Keep that concrete
  figure in any operator-facing explanation of why there are two numbers.

</specifics>

<deferred>
## Deferred Ideas

- **Making the whole-run budget configurable.** Offered and explicitly declined for this phase
  (D-02). If a real project later needs a different ceiling, that is a separate, deliberate
  change — not a knob shipped now.
- **A live-refreshing delegation dashboard.** D-13 deliberately ships a threshold warning line
  instead. A real-time view is a plausible later phase.
- **Reaching the ts-host recursion policy engine** (`.aether/ts-host/src/spawn-orchestrator.ts`,
  unreachable). Out of scope; belongs with the wider ts-host work.
- **Phase 172's CR-06 residue** — the ~18 CI steps that run before the release gate are
  inspected by nothing. Tracked as Phase 172.1. Not this phase.

### Reviewed Todos (not folded)

Both matched only on generic keyword overlap and neither is about delegation. Reviewed and
deliberately left out of scope by the user ("Neither"):

- **`2026-08-01-finalize-reconcile-task-evidence-gate.md`** — `continue-finalize` does not
  count `--reconcile-task` as recorded reconciliation for the implementation-evidence gate.
  Area: `cmd/continue`. Unrelated to spawn depth or budget.
- **`2026-08-01-ts-host-preflight-hardcoded-timeout.md`** — ts-host preflight timeout is
  hardcoded, not configurable, and runs in the repo cwd. Area: `ts-host`. Adjacent in spirit
  to D-17 (which also concerns a timeout that should be configurable) but a different
  subsystem; if the executor finds the two genuinely share a mechanism, surface it rather than
  silently folding it in.

</deferred>

---

*Phase: 173-Delegation Guard*
*Context gathered: 2026-08-12*
