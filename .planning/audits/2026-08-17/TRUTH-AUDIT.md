# Aether Truth Audit — 2026-08-17

*Adversarial audit of Aether vs. GSD as a behavioural benchmark. Branch `oracle-reinstate`, ~v1.0.58. Seven parallel investigations (see `raw/01`–`raw/07`), all claims backed by file:line evidence. No files were modified during the audit.*

> **CORRECTION NOTICE — superseded in part by `HOSTILE-REVIEW.md` (same day).**
> Retracted: "v1.0.56 shipped outside planning" (boundary artifact — that release wrote 10 decision records and roadmap rows; only v1.0.57's single day bypassed planning); "STATE.md is fiction" (stale, not fiction — ROADMAP.md is the maintained ledger); the empty-status fail-open (all ingestion paths are fail-closed first); and the `brief_path` defect (graceful fallback; prose-rot only). Reversed: the GSD "thin orchestrator" and "bounded context" verdicts in §3 — GSD's orchestrator prose is 5× *longer* than Aether's, and GSD plans carry denser executor context than Aether briefs. The defect list, liveness findings, proof gap, and final recommendation all survived hostile re-derivation.

## 1. Executive Verdict

**The "built but never switched on" hypothesis is out of date — in a good way.** It was true when v1.25's researchers found it, and v1.25/v1.26 substantially fixed it: consolidation, hive wisdom, pheromone housekeeping, worker handoffs, evidence-bound acceptance criteria, and Queen depth/caste gating all have live lifecycle callers *and* named wiring tests today. The residue that remains dark (observation intake, curation cadence, five config files without readers) is peripheral, not the golden path.

**The actual dominant failure mode is that Aether has never been proven on real work — and its own process knows it.** The only real-world usage records committed to the repo are failures (CalVault: 256,292 tokens, 3 workers, 12.5 minutes, 0 of 110 files copied; the 2026-08-16 Obsidian init that produced "A unknown project" and walked away; hostile review later added a third: the M4L guardrail-evasion record). Phase 179 "Proof" has survived two rescopes as "the finish line" and has never started.

**Why GSD feels more reliable:** not stronger enforcement — GSD has *weaker* enforcement (pure prompt discipline plus an SDK the LLM must choose to call; its files are saturated with issue-numbered scar tissue from every time discipline failed). GSD feels reliable because (a) it is exercised constantly — this very repo runs on it — so its failure modes got found and patched; (b) its recovery unit is one task, and all progress is re-derivable from disk+git; (c) its golden path asks the user almost nothing. **Aether has the stronger skeleton (Go finalizers, runtime-run tests, SHA-hashed evidence, atomic state) and the weaker mileage.**

**Prescription: measure, fix a short defect list, and re-anchor process. Not a rebuild.**

## 2. Current Aether Golden Path

There are **two parallel execution stacks**: interactive `/ant-build` (Go manifest → LLM spawns subagents → Go finalizer) and `aether run` autopilot (Go spawns headless provider CLI subprocesses, skipping Queen judgement, ceremonies, and checkpoints; `run.md` is 18 lines). Users get materially different builds by entry point.

| Hop | Status | Evidence |
|---|---|---|
| Command → Orchestrator | FRAGILE | 189–358 lines of prose per wrapper; only outcomes checked |
| State read | SOLID | Plan-only manifests; finalizers reject hand-mutated state (state-hash + SHA attempt binding) |
| Context construction | MIXED | Composed in Go (codex_build.go:3080-3110); "verbatim" delivery is prose; brief_path never produced on the interactive path |
| Worker selection | SOLID | Waves, safety floors, depth caps in Go; LLM proposals re-validated |
| Dispatch / execution | FRAGILE | Nothing proves a Task agent ran; fabrication passes if files exist |
| Result collection | SOLID | Packet contract: identity, terminal statuses, claim-paths-exist, non-empty handoffs, phantom rejection, hash binding, freshness (codex_build_finalize.go:464-1072) |
| Verification | SOLID | Continue runs build/lint/tests itself via sh -c (codex_continue.go:1548, 3914); claims re-checked; no --force on continue |
| State mutation | SOLID | Atomic with supersession guard (:915-958) |
| Recovery/advance | MIXED | In-repo: durable attempt journal, deterministic --force redispatch, tested. Worktree mode: broken (§4) |

Cost of one interactive /ant-build (4 workers, 2 waves): ~30–35 orchestrator tool calls. The architecture in one line: it stopped trying to make the LLM obey and built a customs checkpoint the LLM must pass through — the checkpoint is real; the territory before it is trust.

## 3. GSD vs Aether

*(§3 verdicts on thin orchestrators, bounded context, fresh-context equivalence, and "GSD greps" were revised by the hostile review — see the correction notice and HOSTILE-REVIEW.md §B. The table below records the original calls with corrections inline.)*

| Property | Verdict (post-correction) |
|---|---|
| Thin orchestrators | REVERSED: GSD's prose is 5× longer; the real Aether gap is byte-relay delivery (Queen re-emits 6–22KB briefs; GSD passes paths) |
| Fresh-context execution | NOT equivalent: GSD leaner on the accumulation dimension |
| Bounded context | REVERSED on information density: GSD plans carry interfaces/literals/line numbers; Aether briefs carry a 2.2K excerpt + pointers. Aether guarantees the cost floor; GSD's best plans deliver answers |
| Persistent file-based state | Aether BETTER (atomic mutations, SHA-bound attempts, runtime-written evidence) |
| Planning discipline | GSD better: 14-dimension different-agent plan checker; Aether's plan wave has no reviewer caste and self-graded confidence |
| Atomic task progress | GSD better: per-task commits (instructed + partially gated) vs Aether's zero per-task machinery, one commit per phase |
| Explicit completion criteria | Aether BETTER (v1 plans): hash-checked artifacts + actually-executed checks |
| Verification / UAT | Split: Aether's edge is determinism (Go executes; GSD's verifier also executes but as prompted agent). Human UAT: GSD has three layered mechanisms; Aether has none |
| Interruption / resume | In-repo equivalent-or-better (tested); worktree mode worse than nothing (resume GC destroys unmerged work) |
| Clear golden path | Aether unreliable: two stacks, doc drift |

## 4. Five Invariants Audit

1. **Context correctness — PRESENT BUT UNRELIABLE.** The 2025 scaffolding failure (76.6% of prompt) is fixed and test-locked. Four holes: handoff schema never shown to Claude-path workers (Queen fabricates what the finalizer demands); Claude-path continue reviewers get a bare brief (skill/capsule computed then dropped); merged dispatches keep only the first task's criteria; zero intra-build knowledge flow. Capsule not task-aware; total-context proportion guard floors at 5% (nearly vacuous).
2. **State correctness — LARGELY HOLDS.** Per-phase durable evidence (verification.json with actual commands + output, SHA-256 artifact snapshots); state-mutate with typed transitions and test-running guards; PreToolUse hook blocks direct writes (Bash-redirect bypass); resume round-trips exact state.
3. **Execution truth — HOLDS AT PHASE GRANULARITY.** Runtime-executed tests, compile-time hard blocks, no force on continue, atomic gated advance, phantom rejection, hash-frozen artifacts. Residue: claim authorship never verified; reviewer content honor-system; enumerated bypasses (--skip-watchers, env-class downgrades, unresolvable-command skip on unbound plans, seal --force — honestly recorded).
4. **Recovery — HOLDS IN-REPO; BROKEN IN WORKTREE MODE.** gcOrphanedWorktrees force-deletes Allocated/InProgress/Orphaned worktrees + branches from resume/init/continue with no dirty/unmerged check (codex_build_worktree.go:1048); merge gate hard-codes `go test ./...` (:1124) — worktree builds cannot finish in non-Go repos; both untested. Midden failure log severed at the filesystem (writers midden.json; readers midden/midden.json).
5. **Adaptive intelligence — PARTIAL, HONESTLY SO.** Live and tested: depth selection, safety floors, spawn budgets, hybrid caste judgement. Not live: model routing (frontmatter is the only authority; YAML orphan), the 27-caste YAML roster, research heuristics beyond typed mode, learning intake (starved; ≥3-applications bar structurally unreachable; curation once per colony). No Go execution decision reads instincts or wisdom — learned behavior is prompt text only.

## 5. Built But Not Switched On

pkg/memory observation intake (no automatic feeder) · instinct-apply (no lifecycle caller) · 8 curation ants per-phase (seal-only) · pkg/graph (near-orphan) · behavior profiling (manual-only) · colony/agents/*.yaml (27, zero readers) · model-routing.yaml / autopilot.yaml / memory-rules.yaml / colony/phases/*.yaml (zero readers) · session-verify-fresh (unwired, vacuous) · midden-as-failure-log (path split) · skill/capsule on Claude continue (computed, dropped) · brief_path (advertised, never produced on the interactive path) · token spend surface (plumbing without a command) · Phase 179 Proof (specified, never started).

## 6. Keep / Fix / Wire Up / Simplify / Delete / Benchmark

**KEEP:** Go finalizer chain; runtime-run verification; atomic state + state-mutate + guards; evidence-bound criteria; recover/medic; resume snapshot fence; Queen safety floors + depth caps; E2E/blackbox tests; per-phase auto-commit.
**FIX:** worktree GC data loss; go-test merge gate; midden path split; handoff-contract delivery; skill/capsule drop on continue; merged-dispatch criteria loss; ~~brief_path~~ (downgraded); ~~empty-status default~~ (retracted); deleted-files-as-tests_written mislabel; legacy attempt-binding hatch; build-time boundary gate prose-only.
**WIRE UP:** observation intake from continue's live learning capture; the Phase 185 cost line. Nothing else until measured.
**SIMPLIFY:** brief delivery (paths not bytes — the corrected thin-orchestrator fix); two-stack story; asymmetric manifest fetching; doc drift.
**DELETE:** the five reader-less config YAML families; pkg/trace/cost.go + pool; session-verify-fresh; pkg/graph if no consumer emerges.
**BENCHMARK BEFORE DECIDING:** learning/wisdom outcome effect; curation-per-phase; worktree mode as a whole; adaptive-Queen extensions; ceremony cost (owner values it — price it honestly).

## 7. Roadmap Critique

v1.25/160-171 is not the current roadmap (superseded at 24%; HARDENING-PLAN governs; 180–184 + 175 complete). **185 and 179 are the only planned work left — and they are precisely the measurement.** The suspicion that benchmarking occurs too late is confirmed in the strongest form: it occurs never. Deferred blocks (SEE/TYPED/RECLAIM/LOCK) are symptoms, not causes; cuts (176–178) stand; deferred work should become benchmark acceptance criteria, admitted only when a measured baseline shows its dimension is the constraint.

## 8. Benchmark Design

**Headline: AUTONOMOUS SUCCESS RATE** = tasks completed correctly without human rescue ÷ total. Same repo/commit/model/prompt/criteria/environment; GSD lane + two Aether lanes scored separately; fresh clones; hermetic HOMEs; scripted operator protocol; acceptance = deterministic scripts written before runs, hidden from systems; ten task categories (bug fix, isolated feature, brownfield, refactor, ambiguity, research, interruption, worker failure, verification-failure repair, fresh-repo lifecycle); secondaries all script-computable (interventions, hallucinated completion, unnecessary modifications, git cleanliness, resume/recovery success, harness-side tokens and calls, wall-clock). *(Refined into Benchmark V2 by the hostile review; V2 governs.)*

## 9. Minimum Intervention Plan

1. Benchmark first (= Phase 179 upgraded to comparative form; start with 4 categories). 2. Fix the defect list. 3. Ship the cost line (185). 4. Thin the interactive build path (paths not bytes; doc drift; two-stack honesty). 5. Re-anchor process (work re-enters through planning or a declared light lane; STATE.md maintained). 6. Re-run the benchmark; only what the numbers indict gets a next phase.

## 10. Risks

Benchmark-building becoming the next framework (it is shell scripts and a spreadsheet; >1 week to first number = it became the disease) · another roadmap rewrite · test theatre (acceptance scripts written before runs, hidden from the system) · state split-brain (two stacks, one store) · context bloat regression (5% floor) · deleting the soul (ceremonies stay; price them honestly).

## 11. Final Recommendation

**Option E: declare the hardening plan's tail — 185 and 179 — the entire roadmap; execute 179 as the comparative Aether-vs-GSD benchmark; plus the §6 FIX list; nothing else until the baseline exists.** A–D presuppose v1.25 is the live roadmap, and it is not. One sentence: **Aether's skeleton is now stronger than GSD's; what it lacks is mileage, measurement, and about eleven screws — so stop building, start measuring, and tighten the screws the measurements confirm.**
