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
- [ ] **SPEND-05**: Nested child spend rolls up to its parent, so a delegating worker's true cost is visible
- [ ] **SPEND-06**: `aether spend` reports per-worker tokens, cost and tool calls for the current run, mutating nothing
- [ ] **SPEND-07**: No figure in the spend report derives from a character budget, and documentation stops naming a character budget a "Token Budget" — a dashboard fed by `colonyPrimeBudgetChars` can go green while real cost is unchanged, which is how the 186x undercount stayed invisible

## Orchestration Visibility (SEEN)

- [ ] **SEEN-01**: The operator can see which workers the Queen chose and why, in plain English, before they spawn
- [ ] **SEEN-02**: When the runtime overrides the Queen's choice — restoring a required caste, trimming to budget — the override and its reason are stated, not applied silently
- [ ] **SEEN-03**: A worker that returned no actionable finding is distinguishable in the run summary from one that did

## Agent Roster (ROSTER)

The reader is proven against the shipped 27 before any user-extension path is built.

- [ ] **ROSTER-01**: `colony/agents/*.yaml` has a working Go reader, verified by editing a shipped caste's YAML and observing the change take effect without a rebuild
- [ ] **ROSTER-02**: `colony/agents` is published and installed to the hub — it is not distributed at all today, so downstream repos see nothing
- [ ] **ROSTER-03**: The operator can add a new agent to a repo with one command, without editing Go
- [ ] **ROSTER-04**: A user-added agent is written to all three platform lanes (Claude markdown, OpenCode markdown, Codex TOML) and a round-trip test proves the translation, so a bad translation fails locally rather than in CI
- [ ] **ROSTER-05**: A user-added agent cannot delegate by default
- [ ] **ROSTER-06**: A user-added agent cannot enter the safety-caste floor, so the light/standard/heavy worker guarantees continue to hold
- [ ] **ROSTER-07**: A malformed user agent produces a diagnostic naming the file and the problem — it does not vanish silently the way a malformed skill does today
- [ ] **ROSTER-08**: A user-added caste renders with a readable identity rather than blank, since the colour/emoji/label maps are hardcoded Go

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
| WIRE-01 | Phase 172 | Wiring Proof | Pending |
| WIRE-02 | Phase 172 | Wiring Proof | Pending |
| WIRE-03 | Phase 172 | Wiring Proof | Pending |
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
| SEEN-01 | Phase 175 | Orchestration Visibility | Pending |
| SEEN-02 | Phase 175 | Orchestration Visibility | Pending |
| SEEN-03 | Phase 175 | Orchestration Visibility | Pending |
| ROSTER-01 | Phase 176 | Roster Reader | Pending |
| ROSTER-02 | Phase 176 | Roster Reader | Pending |
| ROSTER-03 | Phase 177 | Operator-Authored Agents | Pending |
| ROSTER-04 | Phase 177 | Operator-Authored Agents | Pending |
| ROSTER-05 | Phase 177 | Operator-Authored Agents | Pending |
| ROSTER-06 | Phase 177 | Operator-Authored Agents | Pending |
| ROSTER-07 | Phase 177 | Operator-Authored Agents | Pending |
| ROSTER-08 | Phase 177 | Operator-Authored Agents | Pending |
| SKILL-01 | Phase 178 | Skill Authoring Hardening | Pending |
| SKILL-02 | Phase 178 | Skill Authoring Hardening | Pending |
| SKILL-03 | Phase 178 | Skill Authoring Hardening | Pending |
| SKILL-04 | Phase 178 | Skill Authoring Hardening | Pending |
| PROOF-01 | Phase 179 | Proof | Pending |
| PROOF-02 | Phase 179 | Proof | Pending |
| PROOF-03 | Phase 179 | Proof | Pending |
| PROOF-04 | Phase 179 | Proof | Pending |
