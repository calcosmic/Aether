---
gsd_state_version: 1.0
milestone: v1.26
milestone_name: Intelligent Orchestration (rescoped to hardening 2026-08-14)
status: ready_to_plan
stopped_at: Phase 174 context gathered; roadmap reshaped per owner-approved alignment review
last_updated: "2026-08-21T00:53:05.982Z"
last_activity: 2026-08-21 -- Phase 191 execution started
progress:
  total_phases: 51
  completed_phases: 7
  total_plans: 63
  completed_plans: 114
  percent: 14
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-08)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Phase 191 — dead-wood
**Milestone:** v1.26 — hardening (rescoped 2026-08-14) plus the Audit Addendum (2026-08-17, phases 186-192). Governing documents: `.planning/HARDENING-PLAN.md` (with its 2026-08-17 addendum) and ROADMAP.md's "Audit Addendum" section; the frozen 12-outcome target lives in the approved implementation plan of 2026-08-17.
**Previous milestone:** v1.25 Switch It On — SUPERSEDED at 24% (7 of 29 phases)
**Product version:** v1.0.58 (binary is authoritative)

## Current Position

Phase: 192
Plan: Not started
Status: Ready to plan
Progress: 9 of 16 phases in this milestone
Last activity: 2026-08-21

**Order from here (revised 2026-08-18):** 187, 188, 189→190 and 191 may start
NOW, in any order and in parallel — the 186 rescope removed their only
dependency, and each carries its own fail-able wiring criteria. Then 185, then

192. Phase 186's remaining plan is one smoke run: fire it when convenient, it

gates nothing. 192 is
the full benchmark with the acceptance gate; the milestone completes only when
that gate passes, not when the tasks are done.

**The finish line (HARDENING-PLAN.md + 2026-08-17 addendum):** a small job
costs one or two workers and finishes in minutes; one screen says who was
dispatched, why, and what it cost; Aether never sends its own internals into
someone else's project; the audit fix phases land; and the full benchmark
shows both Aether lanes at least as good as GSD without brute-forcing tokens.
When Phase 192's gate passes, building stops.

**Owner constraints (2026-08-17):** auxiliary commands stay (/ant-oracle,
/ant-dream, chaos, archaeology, swarm, council...); simplification touches
only the main lifecycle path. Both Aether lanes (interactive and /ant-run)
must pass the benchmark gate. Light baseline first, full 3-repeat at the end.

## Performance Metrics

**Velocity:**

- Total plans completed: 51 (v1.26, all Phase 172)
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
- [x] Phases 180–184: hardening fixes — shipped 2026-08-14/15 (v1.0.54)
- [x] Phase 175: Orchestration Visibility — shipped 2026-08-16 (outside the phase system; SHIP-PROGRESS Item 3)
- [x] Truth audit + hostile review + implementation programme approved — 2026-08-17
- [~] Phase 186: rescoped 2026-08-18 — plans 01-06 shipped; plan 07 cut from twelve runs to one non-gating smoke run, to be fired when convenient
- [ ] Plan Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality ← **next**
- [ ] Plan Phase 188: One Truth for Failures and Advances (independent — may run in parallel with 187)
- [ ] Plan Phase 189: Complete Worker Contract (≥2 plans; independent — may run in parallel with 187/188)
- [ ] Plan Phase 190: Lean, Non-Duplicated Delivery (after 189)
- [ ] Plan Phase 191: Dead Wood (independent — may run in parallel with 187-190)
- [ ] Plan Phase 185: One Honest Cost Line (after 190)
- [ ] Plan Phase 192: Final Showdown
- ~~Plan Phases 176, 177, 178~~ — cut 2026-08-14; ~~Phase 179~~ — remapped 2026-08-17 into 186+192

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
