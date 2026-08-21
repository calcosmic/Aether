# Build Check-In and Spawn Economy — Design

**Date:** 2026-08-21
**Owner decisions recorded:** builds always pause for approval; worker questions reach the owner between stages; all three workstreams ship in this round.
**Relationship to Priority Implementation Spec v3:** implements v3's marginal-value spawn gate (§2.8), dispatch visibility (§11), and reviewer-economics direction (§5) as a narrow slice. Explicitly excludes automatic model routing (owner decision stands: rejected) and creates no parallel registries or state stores.

## Problem

1. A build renders its full spawn plan (`aether ceremony spawn-plan`) and then spawns on the next wrapper line. There is no pre-spawn approval anywhere; the only `AskUserQuestion` in `build.md` is post-finalize.
2. Worker handoffs carry a mandatory `open_decisions` field ("choices the next worker or Queen must make"). It renders only into the next worker's prompt (`cmd/codex_dispatch_contract.go:825`). Zero user-facing render sites exist — human-answerable questions are handed to sibling agents instead of the human.
3. Five confirmed spawn-waste sites (see Workstream C).

## Workstream A — Build team check-in (pre-spawn approval)

**Insertion point:** `.claude/commands/ant/build.md`, between the Runtime Spawn Ceremony stage and Worker Spawning (today ~lines 239→241). The Guided Boundary Gate stays ahead of it (same ordering rationale as `plan.md`: a discuss redirect must never throw away decisions already made).

**Flow:**
1. Runtime renders the check-in card (new renderer, see below) from the already-fetched manifest: one line per worker — caste identity in house style, plain-English job, one-line reason (`spawn_budget.selected_reasons`), and a `REQUIRED` / `OPTIONAL` marking. Also lists pruned castes with `pruned_reasons` ("not sent: …") so the owner sees what was already saved.
2. Wrapper asks via `AskUserQuestion` (single question): **Proceed** / **Trim optional workers** (multi-select follow-up listing only the optional tail) / **Redirect first** (routes to `aether discuss`, then demands a fresh manifest — reuse the existing Guided Boundary re-entry contract).
3. On trim: re-fetch `aether build <n> --plan-only --castes <kept> --caste-reason "owner check-in trim"` and spawn from the new manifest. This reuses the exact loop the Queen's Team Decision stage already uses; the caste list simply comes from the owner.
4. Record the trim as a behavior observation through an existing runtime verb (no wrapper-invented state; candidate: `memory-capture` or the suggest pipeline — executor picks the one that feeds the profiler) so repeated trims can become learned defaults later.

**Hard constraints:**
- Required castes are presented as fixed and are not offered for trimming. Go re-adds them regardless (`queenApplyJudgement` restores required castes; required castes bypass the budget), so offering the choice would be silently overridden — the UI must not lie.
- Wrapper must not mutate state; all rendering and decision recording go through runtime commands (wrapper-runtime contract).
- Autopilot (`/ant-run`) and the host-driven lane are untouched — they do not run the interactive wrapper. Smart pause conditions remain their control surface.
- `--yes` style bypass: `/ant-build <n> --no-checkin` (wrapper-level flag) skips the pause for owners who want the old behavior. Default is pause.

**Runtime additions:**
- `aether ceremony team-checkin --manifest-file <f>`: renders the card. Runtime owns presentation (UX architecture rule); wrappers print output verbatim. Derives REQUIRED from `queen_execution_policy.spawn_budget.required_castes`; everything else is OPTIONAL.
- No manifest schema change expected — `caste_decision`, `spawn_budget.selected_reasons`, `pruned_reasons`, `caste_roster.produces` already carry everything.

**Platforms:** `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` edited byte-identically (parity tests enforce). Codex is runtime-native and has no interactive ask: the runtime card still prints (visibility without pause) — documented platform difference, consistent with existing Codex policy.

## Workstream B — Worker questions routed to the owner between stages

**Source of truth:** `open_decisions` in stored worker handoffs (`.aether/data/handoffs/worker-handoffs.json`).

**Runtime addition:** `aether handoff-decisions --pending [--json]` — lists unanswered open decisions from the current attempt's handoffs in plain-English form, deduplicated, newest first, each with a stable id. Wrappers must not parse handoff JSON themselves.

**Flow (build wrapper):** after each execution wave's workers return and before dispatching the next wave, the wrapper runs `handoff-decisions --pending`. If any exist: `AskUserQuestion` (plain English, one question per decision, max 4 per pause; overflow defers to the next boundary). Each answer is recorded via the pending-decisions machinery as a **resolved clarification** — which the context capsule already renders as `## CLARIFIED INTENT` into every subsequent worker prompt (`cmd/colony_prime_context.go:870-917`). No new injection path.

**Runtime addition:** `aether decision-resolve --id <id> --answer "<text>"` (or reuse of an existing discuss-flow verb if one fits) writing the resolved record into `pending-decisions.json` through the existing store, under lock.

**Also surfaced at:** the existing post-verification checkpoint in `continue.md` (Suggested Steering block) gains an open-decisions section; and the post-build `AskUserQuestion` includes any still-unanswered decisions. An unanswered decision never blocks a build — it persists and resurfaces at the next boundary (matching the non-blocking colony philosophy).

**Worker-side rule (brief text):** worker briefs instruct: when blocked on a preference/product judgement the owner could answer, do not research around it — record it in `open_decisions` and proceed with the least-committal option, noting the assumption. (Brief assembler text change in the runtime, not playbooks.)

## Workstream C — Spawn-economy fixes (the five leaks)

| # | Leak | Fix | Enforcing test (fails when unmet) |
|---|---|---|---|
| C1 | Seal requires Probe (+ its budget bypass) even when the final phase produced no testable code — same gate build/continue already have was missed for seal (`cmd/caste_relevance.go:349-350`) | Gate seal's required probe on `queenPhaseProducesTestableCode`, mirroring `queen_spawn_budget.go:152-154` | `TestSealProbeRequiresTestableCode` |
| C2 | Castes selectable for a build with no build dispatch path (`route_setter`; `scout` when no task matches) consume a budget slot and produce zero dispatches, silently displacing real specialists | Build-flow selection excludes castes that cannot yield at least one dispatch (pre-wave, post-wave, or task-caste path). Invariant-style test preferred over naming individual castes | `TestEveryBuildSelectedCasteProducesADispatch` (invariant: for any selected caste set, dispatch count contribution ≥ 1 per caste or caste not selected) |
| C3 | Swarm unconditionally requires Scout + Archaeologist for every bug, including trivial ones (`cmd/caste_relevance.go:338-342`) | Demote scout + archaeologist from always-required to scored for swarm; tracker + builder + watcher stay required. High-risk/synthetic-phase keyword relevance can still select them | `TestSwarmTrivialBugSkipsHistoryAndResearch` + `TestSwarmKeepsCoreTrio` |
| C4 | Continue's fast path ignores `--castes` (nil-proposal variant, `cmd/codex_continue.go:1428`) and its keyword engine summons low-value castes (keeper base 10 + common words "standard/document/pattern") on incidental hits | (a) Fast path honors `--castes`/`--caste-reason` when provided, through `queenContinueReviewSpecsWithJudgement`; (b) tighten low-value continue castes: remove incidental common-word keywords or raise their effective threshold so two incidental hits no longer clear 30 | `TestContinueFastPathHonoursCasteProposal` + `TestContinueDoesNotSummonKeeperOnIncidentalWords` |
| C5 | Colonize always spawns exactly 4 surveyors; relevance scoring is dead code against a synthetic phase | Honor `--verification-depth light` for colonize: trims to nest + provisions surveyors (structure + dependencies). Default stays 4. No attempt to make scoring live in this round | `TestColonizeLightDepthTrimsSurveyors` |

**Out of scope this round:** making colonize scoring live, re-architecting continue's in-process spawning into a manifest flow, any model routing, oracle iteration budgets.

## Definition-of-Done compliance

Every behavior above lands with a named test that fails when the wiring is absent:
- Wrapper structure: extend `platform_doc_hygiene_test.go`-style assertions — build.md must contain the check-in stage between the spawn ceremony and worker spawning; wave boundary must reference `handoff-decisions`; parity across `.claude`/`.opencode` byte-identical.
- Renderer: `TestTeamCheckinCardShowsReasonAndRequiredMarking` (house style, REQUIRED/OPTIONAL, pruned list).
- Injection: `TestResolvedOpenDecisionReachesNextWorkerPrompt` (end-to-end: handoff open decision → resolve → CLARIFIED INTENT in next brief).
- All five C-fixes carry their table tests.
- `--dry-run`/inspection commands (`handoff-decisions`, `team-checkin`) must not mutate state — add the corresponding no-mutate tests.

## Risks

- **Wrapper drift:** three-way mirroring is manual; parity tests are the guard. Codex path is print-only.
- **Attempt re-entry:** trim re-fetch relies on plan-only attempt auto-supersede; already tested behavior, but the check-in must re-verify session freshness after the pause (owner may walk away mid-question).
- **Question fatigue:** max 4 questions per boundary; dedupe by content hash; never repeat an answered decision.
