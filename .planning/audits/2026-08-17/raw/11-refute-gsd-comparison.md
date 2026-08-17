# Raw report 11 — Hostile re-audit of the GSD comparison verdicts (measured)

*Verbatim output of the "Refute GSD comparison verdicts" investigation, 2026-08-17.*

## Verdict 1 — "Aether OVERENGINEERED on thin orchestrators; GSD budgets 10-15%": REVERSED

| File | Lines | Bytes |
|---|---|---|
| GSD workflows/execute-phase.md | 1,799 | 84,934 |
| + required_reading (agent-contracts, context-budget, gates) | 234 | — |
| + step files (per-plan-worktree-gate, post-merge-gate, codebase-drift-gate) | 291 | — |
| **GSD execute orchestrator total** | **~2,324** | — |
| Aether build.md (playbooks not runtime-loaded since v1.25 — the whole load) | **358** | 18,910 |

GSD's execute orchestrator prose is **5.0x** Aether's by lines, 4.5x by bytes. Elsewhere: plan-phase.md 1,721 vs plan.md 213 (8.1x); gsd-planner.md 1,286 + gsd-plan-checker.md 978 vs Route-Setter's ~120 lines. Tool-call estimate for a 3-plan GSD phase: roughly 40-60 tool calls plus 4-5 agent spawns, vs Aether's ~30-35. The "10-15% context" figure is a target stated inside GSD's own file, not a property; the same file mandates re-emitting a ~100-line bash guard block into every executor prompt.

Where the length goes: GSD ~600-700 lines are bug-scar armor executed as prose (#2916 branch-base, #2924 HEAD guards, #2070 SUMMARY rescue, #2774 worktree filter, #2410 heartbeats, #3212 stall surveillance, classifyHandoffIfNeeded workaround), ~450 lines of gates, ~200 of state bookkeeping. Aether's 358 are stage framing, the Queen's judgment table, spawn protocol, guardrails — its scar tissue lives in compiled Go with named tests. Honest verdict: **both are prose-heavy; GSD is 5x heavier. GSD spends prose on failure-handling discipline (fragile: LLM-executed bash); Aether spends prose on judgment rules and moved the machinery into tested code.** "Aether uniquely heavy" is unsupportable.

## Verdict 2 — "Atomic task progress, GSD better": SURVIVES (edges softened)

- Aether has NO per-task commit instruction anywhere in the worker path: zero "commit" matches in aether-builder.md; the runtime brief contains none; workers.md mentions commit only inside worktree-isolation injection and Route-Setter's plan template (not delivered to builders).
- Runtime: commitPhaseAdvance (phase_commit.go:125) = ONE commit per phase at durable advance, toggleable off. No per-task hashes anywhere. Default in-repo mode: a phase's work sits uncommitted until /ant-continue advances.
- GSD: per-task commits INSTRUCTED (gsd-executor task_commit_protocol) and PARTIALLY enforced: checkpoint returns carry per-task hash tables continuation agents verify; the MVP+TDD gate hard-fails on a missing RED commit; TDD review greps commit pairs; orchestrator spot-check requires ≥1 commit per plan.
- Softening: the generic spot-check is plan-granular — a squashed plan passes outside TDD mode. "Enforced" overstates GSD; "instructed with partial checks" is accurate. Against Aether's zero, the verdict holds.

## Verdict 3 — "Plan verification, GSD better": SURVIVES, stronger than stated

- Aether's plan wave has NO reviewer caste (plan.md guardrail: "the contract is Scout ... then Route-Setter per iteration"). Nobody adversarially reviews the Route-Setter.
- The loop's confidence is SELF-GRADED: mergePlanConfidence (codex_plan.go:2539) lets the Route-Setter's score override the runtime baseline whenever >0, including Overall. The only anti-gaming check is evidence-hash binding ("planning confidence changed from %d%% to %d%% without new Scout or Route-Setter evidence", codex_plan_finalize.go:661). Stall/target/cap mechanics are real compiled enforcement — but they enforce the LOOP, not plan quality. No requirement-coverage, scope, or task-completeness semantics in plan-finalize.
- GSD's checker: a separate 978-line agent with 14 dimensions, goal-backward, max-3-iteration loop, structured BLOCKER/WARNING output, filesystem fallback. Also LLM judgment — but DIFFERENT-AGENT judgment against enumerated criteria, categorically more than author self-scoring. Counter-jab that partially lands: GSD's loop cap and convergence are prose; Aether's loop mechanics are compiled and test-locked. GSD checks more; Aether enforces what little it checks harder. Net: verdict correct.

## Verdict 4 — "Aether better automated (GSD greps); GSD better human UAT": automated half REFUTED, human half SURVIVES

- "GSD greps" is false: gsd-verifier Step 7b runs behavioral spot-checks — literal `curl -s http://localhost:$PORT/...`, `npm test -- --grep`; Step 7c: "the verifier must run the probe in its own process". Grep-heavy (36 occurrences) and won't start servers, but it executes. Both systems execute; Aether's edge is determinism (Go vs agent-obeyed prose) — an edge, not a category difference.
- The human half is even more one-sided than claimed. Aether has NOTHING: the only "human" matches in cmd/ are "human-readable" comments; autopilot pauses are all machine signals; build.md's post-build question never offers "go try it." GSD has three layered mechanisms: checkpoint:human-verify (90% of checkpoints), verifier human_needed → persisted committed HUMAN-UAT.md surfaced by /gsd-progress and /gsd-audit-uat until resolved, and /gsd-verify-work conversational UAT.

## Verdict 5 — "Fresh-context execution EQUIVALENT": REFUTED — GSD leaner where it counts

Both spawn fresh-context subagents; mechanisms differ materially. Aether's build.md:259 admits "inline JSON briefs of 6-22KB" and instructs the orchestrator to read each brief and re-emit it VERBATIM into each Task prompt, prepend the capsule, then collect full structured results whose 9-field handoff "will be rejected" if empty. An 8-worker heavy build routes 8×6-22KB through the Queen's context twice (read + emit) plus 8 structured results plus ceremony. The Queen is a byte-for-byte relay; brief_path exists precisely because the inline channel truncated, yet bytes still transit the orchestrator. GSD: "Pass paths only — executors read files themselves"; executors return a compact PLAN COMPLETE block; SUMMARY.md goes to disk, orchestrator reads spot-check slices. Caveats short of a rout: GSD stamps ~100 lines of static worktree-guard bash into every spawn prompt, and its orchestrator reads plan objectives and summaries. But "equivalent" is wrong on the accumulation dimension.

## Verdict 6 — "Bounded context EQUIVALENT+ for Aether": REVERSED on information density

Side by side: renderCodexBuildWorkerBrief gives task text, phase objective, constraints/criteria, a codegraph excerpt capped at 2,200 chars, research ≤3,000, survey POINTERS, a one-line expected outcome, a verification-command section. No interface signatures, no exact expected values, no per-task acceptance checks. GSD's real 174-01-PLAN.md (16.4KB) hands the executor: the current interface quoted verbatim (full WorkerUsage struct, signatures), line numbers ("mapInternalWorkerResult near line 474"), named tests to write, literal expected values with their arithmetic ("102050... as 50 + 100000 + 2000, never a call back into the code under test"), per-task verify commands, grep-checkable acceptance, machine-checkable key_links regexes, a threat register. An Aether builder must rediscover the interfaces the GSD executor is handed. "Bounded" guarantees cost, not sufficiency — a 23.7K ceiling on a brief that contains pointers is not "EQUIVALENT+" to a plan that contains answers. Fairness caveats: this compares Aether's runtime-guaranteed FLOOR against one GSD artifact (a strong planner run — pointedly, one planning Aether itself); a lazy gsd-planner can emit vague plans, which checker Dimension 2 exists to catch. The floor-vs-ceiling asymmetry is real, but the verdict as issued is backwards.

## Scoreboard

| # | Original verdict | Ruling |
|---|---|---|
| 1 | Aether overengineered orchestrators | REVERSED — GSD prose 5x longer (1,799+525 vs 358 lines) |
| 2 | GSD better atomic progress | SURVIVES — Aether provably has no per-task commit path |
| 3 | GSD better plan verification | SURVIVES — no reviewer caste, self-graded confidence |
| 4 | Aether better automated / GSD better UAT | SPLIT: automated half REFUTED (GSD executes curl/npm/probes); human half SURVIVES |
| 5 | Fresh-context equivalent | REFUTED — Aether relays 6-22KB briefs byte-for-byte; GSD passes paths |
| 6 | Bounded context equivalent+ Aether | REVERSED — GSD plans carry signatures/literals/line numbers; Aether briefs carry a 2.2K excerpt and pointers |
