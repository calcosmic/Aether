# Requirements — v1.26 Intelligent Orchestration

**Milestone goal:** The colony reads the work, sends only the workers that work needs, hands each one what it needs to know, and can prove what it cost — so a non-technical operator gets good results on an inexpensive model without tuning anything.

**Supersedes v1.25 "Switch It On."** v1.25 reached 24% (7 of 29 phases). Its still-live intent — cheap-model capability, context reaching workers, phase-fit caste selection, real-world proof — is absorbed below. Its remaining categories (SEE 14, TYPED 8, RECLAIM 9, LOCK 4) are **not** in this scope and move to Future Requirements.

**Evidence standard:** every requirement below traces to a finding verified by reading or *running* current code during v1.26 research. Where a claim could not be verified, it is marked and scoped as a decision rather than a build.

---

## The organising finding

Four independent researchers converged on the same thing: **most of this milestone already exists and was never wired to a caller.**

| Capability | What exists today | Reaches |
|---|---|---|
| Recursive delegation policy | Full engine in `.aether/ts-host/src/spawn-orchestrator.ts` | nobody — build path forbids the TS hop |
| Depth guard | `spawn-can-spawn` (`cmd/spawn.go:169`) | nobody — takes `--depth`, ignores it, always returns true |
| Depth recording | `workers.md:328` hardcodes `--depth 0` | `spawn-tree-depth` structurally always returns 0 |
| Documented guard invocation | `workers.md:292` says run `--enforce` | executing it errors: flag never registered |
| Agent roster | `colony/agents/*.yaml`, 27 castes | zero Go/TS readers |
| Skill lifecycle | 9 commands | 8 have zero references outside their own definition |
| Selection rationale | composed in `caste_relevance.go:168`, carried to manifest | never rendered to a human |
| Token spend | `pkg/codex/usage.go` parses it at dispatch | never persisted; wrapper path has no usage field |

**This is why WIRE-01 lands first.** A ratchet written after the capabilities gets shaped to whatever shipped.

---

## Wiring Proof (WIRE)

- [x] **WIRE-01**: A test fails when a registered subcommand has no caller outside its own definition, seeded with today's known orphans in an allowlist that may only shrink
- [x] **WIRE-02**: The documented invocation in `.aether/workers.md` matches a flag that exists — running the documented command succeeds rather than erroring on `--enforce`
- [x] **WIRE-03**: A test fails when a `.aether/*.md` instruction names a CLI flag the binary does not register

## Delegation Guard (SPAWN)

Enforcement lands before capability: parent/depth linkage is recorded at spawn time and cannot be retrofitted.

- [ ] **SPAWN-01**: `spawn-can-spawn` refuses at a configured depth — a specific test proves it returns `can_spawn: false`, which it has never done
- [ ] **SPAWN-02**: A spawned child records its true depth; `spawn-tree-depth` returns a non-zero value for a nested tree
- [ ] **SPAWN-03**: A whole-tree worker budget stops a run that stays within the depth limit — depth alone is not a budget, and a 2-deep 8-wide tree is 72 workers
- [ ] **SPAWN-04**: A `PreToolUse` hook lets the Go runtime deny a spawn before the platform executes it, and denies when it cannot resolve the requester's depth (fail closed)
- [ ] **SPAWN-05**: The runtime detects ancestor-chain repetition (a caste spawning itself on the same task) and refuses — the real hazard, since a spawn tree cannot contain a cycle
- [ ] **SPAWN-06**: *(Decision)* A written decision records what depth 0 means, resolving the inconsistency present in the repo today
- [ ] **SPAWN-07**: The operator can see a delegation tree as it grows — a runaway subtree is visible while running, not discovered on the bill
- [ ] **SPAWN-08**: An abandoned child worker is reaped and its budget released, and the operator has a command to see and clear orphans — an orphan wastes money once and then progressively locks the colony out of spawning as the tree budget fills with ghosts

## Spend Ledger (SPEND)

- [ ] **SPEND-01**: Worker token usage persists beyond the process and is readable after a run
- [ ] **SPEND-02**: The wrapper path reports token usage — today `codexExternalBuildWorkerResult` has no usage field, so the path an operator actually runs measures nothing
- [ ] **SPEND-03**: A total includes cache-read and cache-creation tokens — the counts are disjoint, and deriving `input + output` undercounted a real run by 186x
- [ ] **SPEND-04**: A run that produced no provider figure appears in the ledger tagged as an estimate, and cannot be read as a measurement
- [ ] **SPEND-05**: Nested child spend rolls up to its parent, so a delegating worker's true cost is visible — parent attribution per row and a no-double-count invariant; the full self/subtree dual-column view is deferred until worker delegation exists (Phase 177)
- [ ] **SPEND-06**: `aether spend` reports per-worker tokens and tool calls for the current run, mutating nothing — dollars never headline and no model-price table is ever built; USD appears only where the provider itself reported it, labelled as hypothetical API price *(reworded 2026-08-13: the owner is on subscription billing — tokens are the unit that draws their limits)*
- [ ] **SPEND-07**: No figure in the spend report derives from a character budget, and documentation stops naming a character budget a "Token Budget" — a dashboard fed by `colonyPrimeBudgetChars` can go green while real cost is unchanged, which is how the 186x undercount stayed invisible
- [ ] **SPEND-08**: *(added 2026-08-13)* The normal end of a build or continue prints one plain-English token line sourced from the ledger ("This phase used ~N tokens (measured)") — the primary surface for a non-technical operator who does not run inspection commands

## Orchestration Visibility (SEEN)

- [ ] **SEEN-01**: The operator can see which workers the Queen chose and why, in plain English, before they spawn
- [ ] **SEEN-02**: When the runtime overrides the Queen's choice — restoring a required caste, trimming to budget — the override and its reason are stated, not applied silently
- [ ] **SEEN-03**: A worker that returned no actionable finding is distinguishable in the run summary from one that did

## Agent Roster (ROSTER) — reshaped 2026-08-13

ROSTER-01/02 reframed as a wire-or-delete ruling (Phase 176). ROSTER-03..08 (operator-authored agents) **shelved to Future Requirements** by the owner-approved reshape: the owner selects among the 27 existing castes, they do not author new ones.

- [ ] **ROSTER-01**: The `colony/agents/*.yaml` zero-reader contradiction is resolved by an explicit recorded ruling — wired with executable proof, or deleted with the compiled registry documented as authoritative *(reworded 2026-08-13; original: "has a working Go reader")*
- [ ] **ROSTER-02**: After the ruling, no file remains that claims to define agents and defines nothing — including `colony/policies/model-routing.yaml`, ruled in the same breath *(reworded 2026-08-13; original: hub distribution — applies only under a "wire" ruling)*

## Worker Delegation Grant (SPAWN, continued) — added 2026-08-13

Phase 173 built the complete guard system and deliberately granted nothing. These grant the capability, per-caste, under those guards (Phase 177).

- [ ] **SPAWN-09**: An ordinary worker of a granted caste can actually spawn a helper when its task needs one, on the wrapper path the operator runs — today only Queen and Route-Setter carry the dispatch tool, so workers.md's spawn protocol is unreachable for every ordinary worker
- [ ] **SPAWN-10**: A worker-spawned helper is recorded in the spawn tree with correct parent linkage and derived depth, counted by the whole-run budget, and visible under its parent in the live tree — the Phase 173 guards demonstrably bound worker-originated spawns
- [ ] **SPAWN-11**: The grant is per-caste and recorded: a written list states which castes carry the tool and why, a test asserts non-granted castes carry none, and the third delegation level is re-verified refused after the grant makes that path reachable

## Skill Authoring (SKILL)

Security lands in this phase, not a phase later.

- [ ] **SKILL-01**: The unreferenced skill-lifecycle commands either have a caller or are removed — 8 of 9 reach nobody today
- [ ] **SKILL-02**: Skill selection cannot be won by filename — an alphabetical tiebreak currently lets one broad `aaa-*` skill evict three shipped skills from every worker prompt
- [ ] **SKILL-03**: Skill validation runs on the Claude path — `/ant-skill-create` currently hand-writes `SKILL.md` and bypasses `aether skill-create`, so validation would protect Codex users only
- [ ] **SKILL-04**: A skill that fails validation is reported with the file and reason, not silently dropped

## Proof (PROOF)

The milestone's verdict. Not satisfiable by tests.

- [ ] **PROOF-01**: Three real development tasks complete in real repositories on an inexpensive model, with operator interventions counted and recorded per task
- [ ] **PROOF-02**: A full colony lifecycle runs in a downstream repo without Aether modifying its own repository
- [ ] **PROOF-03**: Measured tokens-per-phase before and after this milestone are recorded, including if the result is unfavourable
- [ ] **PROOF-04**: A task interrupted mid-session resumes in a fresh session without the operator re-explaining what was in progress

---

## Future Requirements (deferred, not abandoned)

Carried from v1.25, out of scope for v1.26:

- **SEE** (14) — rich terminal visibility beyond SEEN-01..03
- **TYPED** (8) — required `mode` field and removal of prose-based inference
- **RECLAIM** (9) — the wider unreachable-command sweep beyond WIRE-01's ratchet
- **LOCK** (4) — state locking and lifecycle transactions
- **MODEL** — revised per the 2026-07-28 decision: automatic model selection is rejected; cheap-model capability means the framework carries the intelligence, not that it picks models
- **ROSTER-03..08** (6) — operator-authored agents (one-command creation, three-lane translation, safety exclusions, identity rendering). Shelved 2026-08-13 by the owner-approved reshape: the owner selects among the 27 existing castes and has never asked to author a 28th. Revisit only if that changes

Also tracked, deliberately unbundled:
- `gopkg.in/yaml.v3` is archived and author-declared unmaintained. v1.26 makes YAML the format a non-technical user hand-writes, which changes the risk profile. Migration to `go.yaml.in/yaml/v3` is a mechanical import swap and gets its own plan.

## Out of Scope (explicit exclusions)

| Excluded | Reason |
|---|---|
| Automatic model routing | Rejected by user decision 2026-07-28 and reaffirmed. Capability, not allocation |
| A compressed inter-agent language (SAP) | Targets the ~5% of spend the framework composes. The README already rejected schema-based agent messages: "one hallucinated bracket crashes the system" |
| Intra-wave peer communication | Would make a worker's context depend on peer completion order, destroying reproducibility. Waves already express independence |
| A tokenizer dependency | Both providers report exact counts including cache reads |
| User-configurable depth flags | Anti-feature per research — vendors that shipped them retreated |
| An interactive agent-authoring wizard | Built and removed by two vendors (Claude Code v2.1.198, Cursor). One command that writes files |
| An agent/skill marketplace | Third-party skill markets now sell security scanning as the headline feature |
| Compressing the worker brief further | ~6k tokens against ~117k spent. Spend the budget better, not smaller |
| Visuals map expansion as a roster | `casteColorMap` et al. are a 35-entry superset including queen and curation ants — not a caste roster |

---

## Traceability

**Coverage: 35/35 v1.26 requirements mapped to exactly one phase. No orphans, no duplicates.**

Phase ordering is load-bearing and argued in `research/SUMMARY.md` and `research/PITFALLS.md`:
WIRE first (the ratchet must precede the capabilities it constrains) → SPAWN before SPEND
(parent/depth linkage is recorded at spawn time and cannot be retrofitted) → SPEND before
ROSTER/SKILL (extensibility changes what workers cost) → ROSTER reader before the user path →
SKILL security inside the skill phase → PROOF last.

| REQ-ID | Phase | Phase Name | Status |
|--------|-------|------------|--------|
| WIRE-01 | Phase 172 | Wiring Proof | Satisfied (2026-08-12) |
| WIRE-02 | Phase 172 | Wiring Proof | Satisfied (2026-08-12) |
| WIRE-03 | Phase 172 | Wiring Proof | Satisfied (2026-08-12) |
| SPAWN-01 | Phase 173 | Delegation Guard | Pending |
| SPAWN-02 | Phase 173 | Delegation Guard | Pending |
| SPAWN-03 | Phase 173 | Delegation Guard | Pending |
| SPAWN-04 | Phase 173 | Delegation Guard | Pending |
| SPAWN-05 | Phase 173 | Delegation Guard | Pending |
| SPAWN-06 | Phase 173 | Delegation Guard *(decision, not build)* | Pending |
| SPAWN-07 | Phase 173 | Delegation Guard | Pending |
| SPAWN-08 | Phase 173 | Delegation Guard | Pending |
| SPEND-01 | Phase 174 | Spend Ledger | Pending |
| SPEND-02 | Phase 174 | Spend Ledger | Pending |
| SPEND-03 | Phase 174 | Spend Ledger | Pending |
| SPEND-04 | Phase 174 | Spend Ledger | Pending |
| SPEND-05 | Phase 174 | Spend Ledger | Pending |
| SPEND-06 | Phase 174 | Spend Ledger | Pending |
| SPEND-07 | Phase 174 | Spend Ledger | Pending |
| SPEND-08 | Phase 174 | Spend Ledger | Pending |
| SEEN-01 | Phase 175 | Orchestration Visibility | Pending |
| SEEN-02 | Phase 175 | Orchestration Visibility | Pending |
| SEEN-03 | Phase 175 | Orchestration Visibility | Pending |
| ROSTER-01 | Phase 176 | Roster Ruling | Pending |
| ROSTER-02 | Phase 176 | Roster Ruling | Pending |
| ROSTER-03 | — | Shelved to Future Requirements (2026-08-13) | Deferred |
| ROSTER-04 | — | Shelved to Future Requirements (2026-08-13) | Deferred |
| ROSTER-05 | — | Shelved to Future Requirements (2026-08-13) | Deferred |
| ROSTER-06 | — | Shelved to Future Requirements (2026-08-13) | Deferred |
| ROSTER-07 | — | Shelved to Future Requirements (2026-08-13) | Deferred |
| ROSTER-08 | — | Shelved to Future Requirements (2026-08-13) | Deferred |
| SPAWN-09 | Phase 177 | Worker Delegation Grant | Pending |
| SPAWN-10 | Phase 177 | Worker Delegation Grant | Pending |
| SPAWN-11 | Phase 177 | Worker Delegation Grant | Pending |
| SKILL-01 | Phase 178 | Skill Authoring Hardening | Pending |
| SKILL-02 | Phase 178 | Skill Authoring Hardening | Pending |
| SKILL-03 | Phase 178 | Skill Authoring Hardening | Pending |
| SKILL-04 | Phase 178 | Skill Authoring Hardening | Pending |
| PROOF-01 | Phase 179 | Proof | Pending |
| PROOF-02 | Phase 179 | Proof | Pending |
| PROOF-03 | Phase 179 | Proof | Pending |
| PROOF-04 | Phase 179 | Proof | Pending |
