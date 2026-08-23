---
gsd_state_version: 1.0
milestone: v1.27
milestone_name: The Queen Decides, the Program Checks
current_phase: 194
current_phase_name: The Queen Decides the Team
status: executing
stopped_at: Completed 194-02-PLAN.md
last_updated: "2026-08-23T10:03:51.959Z"
last_activity: 2026-08-23
last_activity_desc: Phase 194 execution started
state_head: 58ae46ed5c103a60b937f5e252776382a4af1310
progress:
  total_phases: 7
  completed_phases: 1
  total_plans: 14
  completed_plans: 7
  percent: 14
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-22)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.
**Current focus:** Phase 194 — The Queen Decides the Team
**Previous milestone:** v1.26 Intelligent Orchestration — SHIPPED 2026-08-22 (override close; 185, 186-07 and 192 carried into v1.27)
**Product version:** v1.0.63 (binary is authoritative; published 2026-08-21)
**Governing backlog:** priority spec v3, ratified 2026-08-21 (D1), order amended 2026-08-22 (D12) — `.planning/research/priority-spec-v3-backlog.md`

## Current Position

Phase: 194 (The Queen Decides the Team) — EXECUTING
Plan: 3 of 9
Status: Ready to execute
Last activity: 2026-08-23 — Phase 194 execution started

## Performance Metrics

**Velocity:**

- Total plans completed: 69 (v1.26, all Phase 172)
- Average duration: — (v1.25 phases averaged 6-11 plans each)
- Total execution time: 0 hours

**By Phase:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 172 P05 | 25min | 2 tasks | 4 files |
| Phase 173 P10 | 70min | 4 tasks | 5 files |
| Phase 193 P01 | 50min | 2 tasks | 7 files |
| Phase 193 P02 | 62min | 3 tasks | 13 files |
| Phase 193 P03 | 23min | 2 tasks | 2 files |
| Phase 193 P04 | 70min | 3 tasks | 12 files |
| Phase 193 P05 | 43min | 3 tasks | 7 files |
| Phase 194 P01 | 75min | 2 tasks | 9 files |
| Phase 194 P02 | 130min | 2 tasks | 22 files |

## Accumulated Context

### Roadmap Evolution

- v1.26 roadmap created 2026-08-08 from 35 requirements across 7 categories (WIRE, SPAWN, SPEND, SEEN, ROSTER, SKILL, PROOF), continuing phase numbering from v1.25's Phase 171
- Research suggested 7 phases; the roadmap ships 8. The single deviation is splitting ROSTER into **176 Roster Reader** (ROSTER-01..02, the shipped 27 YAMLs) and **177 Operator-Authored Agents** (ROSTER-03..08, the user path), so the reader must demonstrably change dispatch output before the user-extension path is planned
- Research's suggested sixth phase ("Delegation Delivery") is **not** in this milestone. The SPAWN requirements cover the *guard* only; nothing gains the ability to delegate recursively in v1.26. Delivery (child result plumbing, follow-on-wave channel, parent-direct spawn) carries a research flag and belongs to a later milestone

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
- [Phase 193]: The deterministic floor (`runDeterministicFloor`, one body for both continue lanes) is the only source of a pass; a reviewer verdict can only add a block. Locked by `TestDeterministicFloorIsTheOnlySourceOfAPass` and `TestBothContinueLanesApplyTheSameFloor`
- [Phase 193]: Build-side reviewer dispatch is gated on the Queen's explicit proposal, not the required-caste floor; the Watcher is verified once (`TestPhaseVerifiedOnce`). Probe/Auditor/Gatekeeper double-dispatch is recorded in `.planning/WINDOWS.md` #1 for Phase 194
- [Phase 193]: Builder-reported commands are re-run only when they match a fixed allowlist of build/test runners, contain no shell metacharacters, and run via argv — never `sh -c` (code-review fix CR-01); owner-facing commands are shell-quoted (CR-02)
- [Phase 193]: An unprovable criterion becomes `needs_owner_confirmation`: the phase advances, seal blocks until `aether decision-answer` records the answer; a failing free check with no reviewer sent gets exactly one automatic builder fix attempt, append-only in the attempt journal, then blocks with one command
- [Phase 193]: Deterministic floor promoted to the primary verification anchor; a dispatched reviewer verdict can only ADD a block, never supply the pass — runDeterministicFloor uses the build-time watcher (already resolved), never a live continue-time dispatch -- avoids re-running shell verification twice for a reviewer brief and guarantees a reviewer cannot supply a pass. Both continue lanes now share this one function.
- [Phase 193]: syntheticCriterionRequirements default checks drop watcher (D-06) — Unbound criteria now get claims plus the matching free check, never a reviewer caste by default; a dispatched reviewer that fails still blocks separately.
- [Phase 193]: [Phase 193 P02] Build's verification-stage watcher dispatch now gates on the Queen's explicit proposal (queenAskedFor/queenJudgement.Proposed), not the effective caste set that always includes watcher via the required-caste floor -- the required-caste floor itself is untouched (Phase 194's territory).
- [Phase 193]: [Phase 193 P02] Discovered (flagged, not fixed): probe, auditor and gatekeeper still legitimately double-dispatch on production/security phases with no explicit Queen proposal, because build and continue independently derive the same required caste from the same phase content. Recorded in .planning/WINDOWS.md for Phase 194 (moves the required-caste floor).
- [Phase 193]: 193-03: no implementation change needed in cmd/codex_continue.go -- runDeterministicFloor (193-01) already runs unconditionally before every reviewer-skip path is consulted, so all ten FLOOR-01 skip-path rows pass against pre-existing code — Plan explicitly allows this outcome; tests kept as regression guards
- [Phase 193]: 193-03: --skip-watchers help text corrected to state plainly that only AI reviewer workers are skipped and the program's own checks (build, types, lint, tests) always run; TestNoContinueFlagClaimsToSkipAChecked guards every continueCmd flag going forward — D-01 (193-CONTEXT.md); plain-English mandate (CLAUDE.md)
- [Phase 193]: FLOOR-03 closed: reconcile-task counted as evidence on both continue lanes, builder evidence re-run by the program (D-04), unprovable criteria marked needs_owner_confirmation and seal blocks until answered (D-05) — continueTasksSupportAdvancement's H-04 branch now requires only task.Verified for a reconciled task; reRunBuilderReportedEvidence re-executes a builder's reported commands itself; owner_confirmation_pending gate + checkSealBlockers extension route through the existing decision-answer and force-override paths
- [Phase 193]: 193-04: Tasks 2 (builder evidence re-run) and 3 (owner confirmation) landed in one commit, not two -- both edit the same per-check evaluation loop in evaluatePhaseCriterionEvidence and could not be cleanly split by git hunk — Task 1 (reconciliation) was cleanly separable via git add -p and is its own commit
- [Phase 193]: Verification tests are scoped to the Go packages a phase's changed files touched (D-07), falling back to the full run whenever that scope cannot be honestly derived or on the plan's final phase; only the tests command is ever scoped. — Narrowing a compiler or linter changes what it can see, so build/types/lint always run as configured; only the tests command has a safe scoped runner (go test ./dir/...).
- [Phase 193]: A failing free check with no reviewer dispatched draws exactly one automatic builder fix attempt (D-02/D-03), recorded as a brand-new append-only entry in the build attempt journal; a second automatic attempt never happens and continue blocks with one exact re-run command if the re-run still fails. — The fix attempt must never overwrite the original result and must be visibly countable on the team card and cost line; a new attempt record is the same append-only discipline the out-of-band verification record already established.
- [Phase 194]: [Phase 194] Plan 1: forced-reviewer Rationale carries a fixed "this touches ..." prefix so tests (and later the check-in card) can tell a signal-forced dispatch apart from the untouched legacy keyword/mode scoring engine sharing the same caste name -- without this, auditor's presence cannot distinguish "forced by a named signal" from "the pre-existing production-mode floor also drew it".
- [Phase 194]: Build floor shrinks to queenBuildSafetyRequiredCastes returning exactly [builder] on non-discovery phases (nil on discovery); queenBuildSafetyReviewRequired and queenPhaseHasSecuritySignal deleted outright. — Ruling D11 (2026-08-22) supersedes the mode/position/keyword-inferred floor. A reviewer is now forced only by a named risk signal at the continue step, never at build.
- [Phase 194]: D-13 landed: heavy-depth continue's probe requirement is gated on queenPhaseProducesTestableCode, so --heavy no longer forces a coverage caste onto a documentation-only phase. — The plan's own action text named this test for rewrite (TestHeavyContinueKeepsProbe), which required the underlying isAlwaysRequired code change, not just a fixture edit.

### Pending Todos

- [x] Get user approval on the rescope — given 2026-08-14 ("go ahead, cut them all")
- [x] Plan Phase 172: Wiring Proof — shipped
- [x] Plan Phase 173: Delegation Guard — shipped
- [x] Plan Phase 174: Spend Ledger — closed partial at plan 2, rest cut
- [x] Phases 180–184: hardening fixes — shipped 2026-08-14/15 (v1.0.54)
- [x] Phase 175: Orchestration Visibility — shipped 2026-08-16 (outside the phase system; SHIP-PROGRESS Item 3)
- [x] Truth audit + hostile review + implementation programme approved — 2026-08-17
- [~] Phase 186: rescoped 2026-08-18 — plans 01-06 shipped; plan 07 cut from twelve runs to one non-gating smoke run, to be fired when convenient
- [x] Phase 193: Free Checks Are the Floor — executed and verified 2026-08-22 (v1.27)
- [x] Discuss + plan Phase 194: The Queen Decides the Team — planned 2026-08-23 (9 plans, 9 waves)
- [ ] Execute Phase 194: The Queen Decides the Team ← **next**
- [ ] Plan Phase 187: Crash-Safe Worktrees & Ecosystem Neutrality
- [ ] Plan Phase 188: One Truth for Failures and Advances (independent — may run in parallel with 187)
- [ ] Plan Phase 189: Complete Worker Contract (≥2 plans; independent — may run in parallel with 187/188)
- [ ] Plan Phase 190: Lean, Non-Duplicated Delivery (after 189)
- [ ] Plan Phase 191: Dead Wood (independent — may run in parallel with 187-190)
- [ ] Plan Phase 185: One Honest Cost Line (after 190)
- [ ] Plan Phase 192: Final Showdown
- ~~Plan Phases 176, 177, 178~~ — cut 2026-08-14; ~~Phase 179~~ — remapped 2026-08-17 into 186+192

### Blockers/Concerns

- ⚠️ [Phase 193] Open in the defect ledger (`.planning/WINDOWS.md` #1): Probe/Auditor/Gatekeeper can still be sent at both the build and continue boundaries on production/security phases with no explicit Queen ask, because each flow derives "always required" independently. Phase 194 retires that rule; `/gsd-ship` blocks while the entry is open
- ℹ️ [Phase 193] GSD tooling: `phase.complete` advanced STATE to backlog entry 172.1 (the next unchecked roadmap box) — corrected by hand to Phase 194; expect the same after each v1.27 phase while 172.1 sits in the backlog list
- ~~Roadmap is unapproved~~ — resolved 2026-08-14. This line was stale: it said not to start Phase 172 until sign-off, and 172, 173 and part of 174 were then executed anyway. Recorded because a gate that gets ignored without anyone noticing is the failure mode this project keeps rediscovering.
- **Phases 176, 177 and 178 are cut, not deferred.** If a future session finds Phase 173's delegation guards bounding a capability nobody has, that is expected and accepted — Phase 177 was the grant, and it was cut deliberately. Do not "restore" it.
- **Phase 173 and Phase 178 carry research flags from the research phase.** Platform nested-spawn behaviour is MEDIUM confidence and both vendors broke it within the last quarter (Claude Code strips the Agent tool from some subagent types; OpenCode has an open "subagents can infinitely recurse, no max depth" defect). The skill supply-chain threat surface is actively evolving. Re-verify vendor docs and open issues at planning time for both
- **OpenCode hook parity is unverified.** No confirmed equivalent to Claude Code's `PreToolUse` deny gate was found. If SPAWN-04's guard must hold on OpenCode, Go-side depth enforcement becomes mandatory rather than defence-in-depth — verify before planning that requirement
- **Codex has no native subagent nesting.** Delegation on the Codex lane must be *off* and reported honestly, never emulated. Emulation would create a second divergent orchestration path — the exact failure v1.24 and v1.25 spent two milestones unwinding
- Two gaps surfaced during roadmapping that have **no supporting requirement** and were deliberately not added to scope (see Coverage Notes in the roadmap return): the char-budget-vs-spend naming guardrail (Pitfall 8), and orphaned-child reaping with an operator command (Pitfall 5). Both are recorded as phase guardrails; if either is to be enforced, it needs a requirement and a user decision
- **`pkg/trace/cost.go` holds a stale model-price table that contradicts Phase 174's D-02 ("no model-price table is ever built"), and it repeats the 186x undercount shape** — `CalculateCost` multiplies only input and output tokens and ignores cache tokens entirely, against hardcoded rates for retired model names. It is called from `pkg/agent/pool.go:192`; no live caller was found under `cmd/` for any of the three dispatch platforms, so it is probably dead, but that was not proven exhaustively. **Deliberately OUT OF SCOPE for Phase 174** (surfaced by 174-RESEARCH.md Open Question 1, resolved there as owner residue). Phase 174 prevents a second instance — the new spend surface carries greps that fail if a price table or any token-to-dollar arithmetic appears — but the existing one needs an owner ruling: delete it as dead code, or fix its arithmetic. Plan 174-09's SUMMARY must restate this so it does not evaporate with the phase.

- `gopkg.in/yaml.v3` is archived and author-declared unmaintained. v1.26 makes YAML the format a non-technical user hand-writes, which changes the risk profile. The migration to `go.yaml.in/yaml/v3` is a mechanical import swap and is deliberately unbundled — it gets its own plan, not a rider on the roster work
- [Phase 193] Probe/auditor/gatekeeper double-dispatch (build AND continue independently require the same caste with no explicit Queen proposal) is unresolved -- flagged for Phase 194, tracked in .planning/WINDOWS.md (kind: unmet-truth). Not a blocker to sealing 193 (out of this plan's declared scope), but Phase 194 planning must account for it.

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

Acknowledged at the v1.26 close (2026-08-22). Each is carried in `.planning/research/v1.27-milestone-brief.md` or is an archived-phase leftover; acknowledging suppresses the audit line only until the artifact changes again.

| Category | Item | Status | Deferred At | Milestone |
|----------|------|--------|-------------|-----------|
| uat_gaps | 173/173-HUMAN-UAT.md | partial | 2026-08-22 | v1.26 |
| uat_gaps | 72/72-HUMAN-UAT.md | partial | 2026-08-22 | v1.26 |
| uat_gaps | 76/76-HUMAN-UAT.md | partial | 2026-08-22 | v1.26 |
| verification_gaps | 173/173-VERIFICATION.md | human_needed | 2026-08-22 | v1.26 |
| verification_gaps | 71/71-VERIFICATION.md | gaps_found | 2026-08-22 | v1.26 |
| verification_gaps | 72/72-VERIFICATION.md | human_needed | 2026-08-22 | v1.26 |
| verification_gaps | 76/76-VERIFICATION.md | human_needed | 2026-08-22 | v1.26 |
| todos | 2026-08-01-finalize-reconcile-task-evidence-gate.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-01-ts-host-preflight-hardcoded-timeout.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-20-spec-builder-feature.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-completion-packet-cannot-express-bundled-work.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-continue-checker-captures-wrong-field.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-finalize-covered-tasks-DISPROVEN.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-no-reentry-path-for-out-of-band-work.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-stage-finalize-deadlock-FIXED.md | (presence-only) | 2026-08-22 | v1.26 |
| todos | 2026-08-21-weight-classes-pipeline-fits-the-task.md | (presence-only) | 2026-08-22 | v1.26 |
| deferred_items | 172/deferred-items.md: 172-00 (Task 1)  - **`.aether/docs/command-playbooks/continue-advance.md` — malf | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 172/deferred-items.md: 172-07 (both tasks)  - **`pkg/codex` `TestCodexReadOnlyProfileSelectsReadOnlySan | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 188/deferred-items.md: 2. `cmd/colony_prime_audit_test.go` and `cmd/medic_repair_test.go` seed the nest | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 190/deferred-items.md: D-190-01-A: Pheromone signals and prior-worker handoffs render twice in the plan | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 190/deferred-items.md: Stale ceremony-adapter snapshot fixtures (pre-existing, unrelated to hive_sectio | acknowledged | 2026-08-22 | v1.26 |
| deferred_items | 191.1/deferred-items.md: 191.1-01: `TestPackedNPMReleaseCandidateContract` fails on a pre-existing versio | acknowledged | 2026-08-22 | v1.26 |

## Session Continuity

Last session: 2026-08-23T10:03:44.383Z
Stopped at: Completed 194-02-PLAN.md
Resume file: None
