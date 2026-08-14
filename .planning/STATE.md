---
gsd_state_version: 1.0
milestone: v1.26
milestone_name: Intelligent Orchestration (rescoped to hardening 2026-08-14)
status: executing
stopped_at: Phase 174 closed partial; phases 176-178 cut; hardening phases 180-185 added
last_updated: "2026-08-14T00:00:00.000Z"
last_activity: 2026-08-14 -- roadmap rescoped by owner decision; Phase 180 is next
progress:
  total_phases: 11
  completed_phases: 3
  total_plans: 29
  completed_plans: 29
  percent: 27
  note: >
    Counts cover milestone v1.26 only (phases 172, 173, 174, 180-185, 175, 179).
    174 counts as complete because it is closed partial and will not be resumed;
    2 of its 9 plans shipped, the rest were cut. The previous values in this
    block were self-contradictory (2 of 38 phases at 100%; 82 plans completed
    out of a total of 36) and were reporting on a phase set that no longer
    exists.
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-08)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Phase 180 — keep Aether's own internals out of other people's projects
**Milestone:** v1.26 — rescoped 2026-08-14 from "Intelligent Orchestration" to hardening. The governing document is now `.planning/HARDENING-PLAN.md`; it overrides ROADMAP.md for anything numbered 175 and above.
**Previous milestone:** v1.25 Switch It On — SUPERSEDED at 24% (7 of 29 phases)
**Product version:** v1.0.53 *(corrected — this file said v1.0.50, PROJECT.md says v1.0.42, the binary reports 1.0.53; the binary is authoritative)*

## Current Position

Phase: 180 — Aether Stays Out Of Other People's Projects
Plan: not yet planned
Status: ready to plan
Progress: 3 of 11 phases in this milestone
Last activity: 2026-08-14 -- roadmap rescoped by owner decision

**Order from here:** 180 → 181 → 182 → 183 → 184 → 185 → 175 → 179.
Each of 180-184 is independently completable and independently verifiable, so
losing context part-way costs one phase, not the sequence.

**The finish line (from HARDENING-PLAN.md):** a small job costs one or two
workers and finishes in minutes; one screen says who was dispatched, why, and
what it cost; Aether never sends its own internals into someone else's project.
When those are true, building stops.

## Performance Metrics

**Velocity:**

- Total plans completed: 20 (v1.26, all Phase 172)
- Average duration: — (v1.25 phases averaged 6-11 plans each)
- Total execution time: 0 hours

**By Phase:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 172 P05 | 25min | 2 tasks | 4 files |
| Phase 173 P10 | 70min | 4 tasks | 5 files |

## Accumulated Context

### Roadmap Evolution

- v1.26 roadmap created 2026-08-08 from 35 requirements across 7 categories (WIRE, SPAWN, SPEND, SEEN, ROSTER, SKILL, PROOF), continuing phase numbering from v1.25's Phase 171
- Research suggested 7 phases; the roadmap ships 8. The single deviation is splitting ROSTER into **176 Roster Reader** (ROSTER-01..02, the shipped 27 YAMLs) and **177 Operator-Authored Agents** (ROSTER-03..08, the user path), so the reader must demonstrably change dispatch output before the user-extension path is planned
- Research's suggested "Phase 6 — Delegation Delivery" is **not** in this milestone. The SPAWN requirements cover the *guard* only; nothing gains the ability to delegate recursively in v1.26. Delivery (child result plumbing, follow-on-wave channel, parent-direct spawn) carries a research flag and belongs to a later milestone

### Decisions

- **The organising finding: most of v1.26 already exists and was never wired to a caller.** Four independent researchers converged on this by reading and *running* this repo's code. The recursion policy engine (`.aether/ts-host/src/spawn-orchestrator.ts`), the depth guard (`spawn-can-spawn` ignores its own `--depth`), the 27-caste roster (`colony/agents/*.yaml`, zero readers), the skill lifecycle (8 of 9 commands unreferenced), the selection rationale (composed, carried to the manifest, discarded), and token measurement (parsed, never persisted). The framing is **switch on and prove**, not **design and build**
- **Phase ordering is load-bearing, not stylistic.** WIRE first (a ratchet written after the capabilities gets shaped to whatever shipped) → SPAWN before SPEND (parent/depth linkage is recorded at spawn time and cannot be retrofitted to past runs) → SPEND before ROSTER/SKILL (extensibility changes what workers cost; without a ledger every later claim is unfalsifiable) → ROSTER reader before the user path → SKILL security inside the skill phase → PROOF last
- **ARCHITECTURE vs PITFALLS disagreement resolved in favour of guard-before-ledger.** ARCHITECTURE argued spend must ship first because delegation budgets set without spend data are guesses; PITFALLS argued the guard must ship first because the ledger's subtree roll-up needs linkage recorded at spawn time. Synthesis: the guard enforces with a deliberately conservative default (depth 2, tree total ~20 — the TS host's already-chosen numbers), and the ledger then supplies the evidence to retune those numbers with the measurement recorded
- **Success criteria must be able to fail.** CLAUDE.md's Definition of Done governs. Two corrections from this repo's history shaped them: a criterion reading "fewer than 27 castes loaded" already passed and proved nothing, and a token ledger's unit test asserted the same arithmetic its parser used, so a 186x undercount shipped green. The spend criteria therefore name the provider's documented figures (102,050 / 102,550), not the intent. Where possible criteria assert a proportion or invariant (grand total equals the sum of the `self` column; the ratchet allowlist may only shrink; the depth-cap numbers hold with five user agents installed)
- **Decision-shaped, not build-shaped: SPAWN-06.** What depth 0 means. The repo contradicts itself today (`build.md` hardcodes `--depth 1` for manifest workers, `workers.md` hardcodes `--depth 0` for their children). This is a ruling to record; the number matters less than one convention existing. Once recorded it fixes the expected value in Phase 173's depth-derivation criterion
- **Two further rulings the phases must make explicitly rather than discover:** (a) Phase 176 — whether the roster or `colony/policies/model-routing.yaml` owns model routing; both exist with zero readers, and wiring one without ruling on the other recreates the original condition. (b) Phase 177 — the user-agent namespace must be excluded from `TestCanonicalAgentSourcesRemainAligned` by decision, not by a hand-grown exemption list, or a user adding one agent either breaks CI or gets an agent that silently vanishes on two platforms
- **The TypeScript host is still KEPT** (carried from v1.25). `.aether/ts-host/src/spawn-orchestrator.ts` is a complete, correct ~150-line specification for Phase 173's Go port — it does not need rewriting, it needs a caller
- **Explicitly out of scope, already ruled:** automatic model routing (user decision 2026-07-28, reaffirmed — cheap-model capability means the framework carries the intelligence, not that it picks models), a compressed inter-agent language, intra-wave peer communication, a tokenizer dependency, user-configurable depth flags, an interactive agent-authoring wizard, an agent/skill marketplace, and further compression of the worker brief
- [Phase 172]: Named CI step 'Verify subcommand wiring and CLI flag contracts' added alongside (not replacing) the blanket go test ./... release gate, so a wiring failure reads as a wiring problem
- [Phase 172]: ROADMAP.md Phase 172/178 criteria corrected to the scan's real numbers: 278 seeded orphans, 6 owned by phase 178, replacing the pre-scan assumption of 8
- [Phase 173]: Plan 10: orphan_allowlist.json earned zero removals -- all five spawn-related candidates remain genuine orphans against the live reachability ratchet because .aether/workers.md is not a caller corpus this ratchet scans
- [Phase 173]: Plan 10: ROADMAP criterion 4 left byte-identical -- the cross-guard fail-closed proof establishes a narrower claim than the criterion's literal wording; whether it is satisfied is left to verification, per 173-RESIDUE.md residue 7

### Pending Todos

- [x] Get user approval on the rescope — given 2026-08-14 ("go ahead, cut them all")
- [x] Plan Phase 172: Wiring Proof — shipped
- [x] Plan Phase 173: Delegation Guard — shipped
- [x] Plan Phase 174: Spend Ledger — closed partial at plan 2, rest cut
- [ ] Plan Phase 180: Aether Stays Out Of Other People's Projects ← **next**
- [ ] Plan Phase 181: Reading Material Chosen By The Task
- [ ] Plan Phase 182: Specialists Earn Their Seat
- [ ] Plan Phase 183: The Worker Limit Actually Limits
- [ ] Plan Phase 184: One Worker Owns A Run Of File Work
- [ ] Plan Phase 185: One Honest Cost Line
- [ ] Plan Phase 175: Orchestration Visibility
- [ ] Plan Phase 179: Proof
- ~~Plan Phases 176, 177, 178~~ — cut 2026-08-14

### Blockers/Concerns

- ~~Roadmap is unapproved~~ — resolved 2026-08-14. This line was stale: it said not to start Phase 172 until sign-off, and 172, 173 and part of 174 were then executed anyway. Recorded because a gate that gets ignored without anyone noticing is the failure mode this project keeps rediscovering.
- **Phases 176, 177 and 178 are cut, not deferred.** If a future session finds Phase 173's delegation guards bounding a capability nobody has, that is expected and accepted — Phase 177 was the grant, and it was cut deliberately. Do not "restore" it.
- **Phase 173 and Phase 178 carry research flags from the research phase.** Platform nested-spawn behaviour is MEDIUM confidence and both vendors broke it within the last quarter (Claude Code strips the Agent tool from some subagent types; OpenCode has an open "subagents can infinitely recurse, no max depth" defect). The skill supply-chain threat surface is actively evolving. Re-verify vendor docs and open issues at planning time for both
- **OpenCode hook parity is unverified.** No confirmed equivalent to Claude Code's `PreToolUse` deny gate was found. If SPAWN-04's guard must hold on OpenCode, Go-side depth enforcement becomes mandatory rather than defence-in-depth — verify before planning that requirement
- **Codex has no native subagent nesting.** Delegation on the Codex lane must be *off* and reported honestly, never emulated. Emulation would create a second divergent orchestration path — the exact failure v1.24 and v1.25 spent two milestones unwinding
- Two gaps surfaced during roadmapping that have **no supporting requirement** and were deliberately not added to scope (see Coverage Notes in the roadmap return): the char-budget-vs-spend naming guardrail (Pitfall 8), and orphaned-child reaping with an operator command (Pitfall 5). Both are recorded as phase guardrails; if either is to be enforced, it needs a requirement and a user decision
- **`pkg/trace/cost.go` holds a stale model-price table that contradicts Phase 174's D-02 ("no model-price table is ever built"), and it repeats the 186x undercount shape** — `CalculateCost` multiplies only input and output tokens and ignores cache tokens entirely, against hardcoded rates for retired model names. It is called from `pkg/agent/pool.go:192`; no live caller was found under `cmd/` for any of the three dispatch platforms, so it is probably dead, but that was not proven exhaustively. **Deliberately OUT OF SCOPE for Phase 174** (surfaced by 174-RESEARCH.md Open Question 1, resolved there as owner residue). Phase 174 prevents a second instance — the new spend surface carries greps that fail if a price table or any token-to-dollar arithmetic appears — but the existing one needs an owner ruling: delete it as dead code, or fix its arithmetic. Plan 174-09's SUMMARY must restate this so it does not evaporate with the phase.

- `gopkg.in/yaml.v3` is archived and author-declared unmaintained. v1.26 makes YAML the format a non-technical user hand-writes, which changes the risk profile. The migration to `go.yaml.in/yaml/v3` is a mechanical import swap and is deliberately unbundled — it gets its own plan, not a rider on the roster work

## Deferred Items

Carried forward from v1.25 (superseded) and v1.23:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Visibility | SEE (14) — rich terminal visibility beyond SEEN-01..03 | Deferred | v1.26 roadmap |
| Typing | TYPED (8) — required `mode` field, removal of prose inference | Deferred | v1.26 roadmap |
| Reclaim | RECLAIM (9) — wider unreachable-command sweep beyond WIRE-01's ratchet | Deferred | v1.26 roadmap |
| Locking | LOCK (4) — state locking and lifecycle transactions | Deferred | v1.26 roadmap |
| Models | MODEL — automatic model selection | Rejected | user decision 2026-07-28 |
| Delegation | Recursive delegation *delivery* (child result plumbing, follow-on wave, parent-direct spawn) | Deferred | v1.26 roadmap — guard only this milestone |
| Hygiene | `gopkg.in/yaml.v3` → `go.yaml.in/yaml/v3` migration | Tracked, unbundled | v1.26 roadmap |
| Catalog | CATALOG-01, CATALOG-02 | Deferred | v1.25 roadmap |
| Test coverage | TEST-01, TEST-02 | Deferred | v1.25 roadmap |
| Workflow | WORKFLOW-01 … WORKFLOW-09 | Deferred | v1.25 roadmap |
| Runtime | RUNTIME-01, RUNTIME-02 | Deferred | v1.25 roadmap |

## Session Continuity

Last session: 2026-08-13T20:44:38.413Z
Stopped at: Phase 174 context gathered; roadmap reshaped per owner-approved alignment review
Resume file: .planning/phases/174-spend-ledger/174-CONTEXT.md
