# Audit Under Fire — Hostile Review of the Truth Audit — 2026-08-17

*Five refutation investigations (see `raw/08`–`raw/12`) re-derived every major conclusion of `TRUTH-AUDIT.md` from source, tasked to disprove it. Read this document first; where it conflicts with the truth audit, this one wins. This document's Frozen Target (§F) is the source of truth for the implementation programme (phases 186–192).*

**Outcome in one paragraph:** the defect list and the proof gap survived hostile re-derivation almost intact (six of nine bugs confirmed twice over, one worse than stated); the process narrative was materially wrong about v1.0.56 and is retracted; four of six GSD-comparison verdicts fell — two reversed against GSD's favor, two against Aether's — changing what "simplify the orchestrator" should mean. New findings the first audit missed: downstream repos run on hardcoded fallbacks while the "live" policy YAMLs sit unread; the learning pipeline's two halves are unbridged by dead code; resume's worktree GC races recover's evidence scanner with no enforced order. **Verdict: GO — with measurement, not code, as the first step.**

## A. Surviving Conclusions

| Conclusion | Confidence | Result |
|---|---|---|
| Worktree GC destroys unmerged work | HIGH | Re-derived twice; targets Allocated/InProgress/Orphaned with --force + branch -D, no dirty/unmerged check; runs before recover can classify the same worktrees as critical evidence; deletes worktrees the reconciler deliberately preserved |
| go test hard-coded in merge gate | HIGH | Confirmed, fatal at finalize, duplicated in the manual recovery command; resolveTestCommand() never used on either worktree path |
| Midden path split | HIGH | Six writers flat path, six readers dir path, zero writers to the read path; status shows counts while colony-prime/autopilot/immune/memory-health see zero forever |
| Handoff contract unreachable by workers | HIGH | Worsened: schema IS in build.md — addressed to the Queen, who is forbidden to relay it ("Nothing else, nothing invented"); subprocess path synthesizes missing handoffs in code, Claude path leaves it to the Queen uninstructed |
| Heavy-continue reviewers get bare brief | HIGH | Refined: skill_section shipped and never mentioned; capsule and pheromones have no fields at all on the external dispatch struct |
| Merged dispatches drop later tasks' criteria | HIGH | Worsened: every covered task is marked complete by a worker never shown its criteria. Narrowed: only serial single-worker chains merge |
| Legacy attempt-binding hatch | MEDIUM | .Legacy never inspected; finalize mints a fresh attempt for unbound manifests; packet semantics/provenance/freshness/gates still apply — only the tamper ratchet is bypassed |
| Phase 185 unstarted; 179 never run; finish line self-inconsistent | HIGH | Verified; 179's criterion names `aether spend`, abolished by 185's own rescope. "Nonexistent" corrected to "fully specified, never started" |
| No provider-backed automated E2E; no committed downstream success | HIGH | Verified (FakeInvoker, --synthetic, httptest, zero CI provider refs); a THIRD adverse record surfaced (M4L guardrail evasion). Provider-backed runs exist only on Aether itself, uncommitted |
| GSD better at plan verification | HIGH | Strengthened: no reviewer caste; loop confidence self-graded by the Route-Setter; only evidence-hash binding restrains it |
| GSD better at atomic task progress | HIGH | Aether has zero per-task commit machinery; GSD's instructed with partial gating — against zero, the verdict holds |
| Execution truth runtime-enforced at phase granularity | HIGH | Strengthened — the claimed fail-open was fail-closed |
| Zero-reader configs dead; learning intake starved | HIGH | All five families re-confirmed; the documented bridge (newLearningValidator) has no caller; the promotion-gating application counter has no automatic writer; workers.md:825 documents a capture that does not exist |
| pkg/trace/cost.go dead | HIGH | Doubly dead — its sole caller lives in a pool nothing constructs |
| Benchmark must come first | HIGH | Every branch of the review strengthened it: the comparison verdicts that fell, fell for lack of measurement |

## B. Retracted or Weakened Conclusions

- **RETRACTED — "v1.0.56 and v1.0.57 shipped outside the planning system."** Boundary artifact. v1.0.56's development day wrote ten decision records, updated REQUIREMENTS/PROJECT/ROADMAP-Progress, committed the CalVault field report. Only v1.0.57's single feature day (15 commits, zero planning writes) stands.
- **RETRACTED — "STATE.md is fiction."** Stale, not fiction; ROADMAP.md Progress rows + decisions/ are the maintained ledger. The real defect: two position ledgers, one abandoned — a split-brain, not a lie.
- **WEAKENED — "the hardening plan was ignored."** Mostly inverted: 180–184 and 175 were executed under the plan and recorded. Survives: one feature day jumped the declared queue ahead of 185/179 with no in-repo sanction record.
- **REVERSED — "Aether overengineered on thin orchestrators."** Measured: GSD execute-orchestrator prose ~2,324 lines vs Aether's 358 (5×); est. 40–60 tool calls vs ~30–35. GSD's "10–15% context" is a target stated in its own file. GSD spends prose on LLM-executed failure-handling bash; Aether moved scar tissue into compiled, tested Go.
- **REFUTED — "GSD's verifier just greps."** It executes curl/npm/probe commands. Aether's automated edge is *determinism*, not category.
- **REFUTED — "fresh-context execution equivalent."** Aether's Queen relays 6–22KB briefs byte-for-byte and collects full structured results — a byte relay and accumulation point; GSD passes paths. **This — not wrapper line count — is the real thin-orchestrator gap.**
- **REVERSED — "bounded context: Aether equivalent-plus."** GSD plans hand executors interface signatures, line numbers, literal expected values, per-task verify commands; Aether briefs cap a codegraph excerpt at 2,200 chars and point at files. Floor-vs-ceiling caveat noted; the verdict as issued was backwards.
- **RETRACTED — empty-status "fail-open."** Every ingestion path validates terminal status first; the runtime parser defaults to failed.
- **DOWNGRADED — brief_path.** Graceful fallback; prose-rot with an unmitigated truncation risk, not breakage.
- **CORRECTED — "policy YAMLs are live."** review-depth.yaml, dispatch-contract.yaml, colony-prime.md, visuals.md load CWD-relative with silent fallback and are never distributed — live only in the dev checkout; **every downstream repo runs on hardcoded Go defaults.** New finding; the exact "documentation claim about runtime behaviour" failure mode this repo's rules name.
- **CORRECTED** — housekeeping runs on every *advancing* continue; the adverse-datapoint count was two, is three.

## C. Single-Source-of-Truth Map

See `raw/12` for the full table with citations. Five genuine split-brains: (1) three build executors (wrapper LLM / aether run / TS host) over one planner with different finalize routes, retry semantics, caste inputs; (2) phase progression — duplicated continue advance blocks with diverging guards (finalize path lacks the supersession check) + opt-in-guard state-mutate; (3) TS host as second context composer (hive wisdom delivered twice from two scorers); (4) recovery ordering — resume's force-GC destroys what recover classifies as critical evidence, no enforced order (worst); (5) six unledgered retry budgets. Single-authority (no action): state writes (hook-enforced), model routing (agent frontmatter only), caste selection (Go final say), ceremony (Go renderers). Latent: plan-finalize's private freshness copy. Context flow: eager mass ceiling ~23,700 chars/worker; pheromones and handoffs each delivered twice (thrice for pheromones on the subprocess path); biggest irrelevancy risk is the always-on cross-colony wisdom trio with no task-relevance filter (the CalVault class); `aether build --print-brief` is verified read-only and can measure real composition.

## D. Benchmark V2 (governs; absorbed the fairness attacks)

Disqualified as substrate: the Aether repo and any repo either system has planned (GSD authored this repo's .planning/ — home-field advantage). Subjective metrics (plan quality, context relevance) demoted to unscored appendix. Acceptance: deterministic scripts only, written before runs, stored outside both systems' reach, avoiding both systems' verification vocabularies, run in a fresh clone of the result. Model parity: one pinned model for orchestrator + workers both systems (Aether frontmatter forced to inherit via config). Hermetic HOME per lane (contamination channel: the operator's ~/.claude contains GSD's skills and both systems' memories). Three lanes scored separately: GSD, Aether interactive, Aether /ant-run (different machines; pooling hides whichever is worse); Aether in in-repo mode (worktree disqualified until fixed — recorded, not hidden). Operator script: permitted inputs enumerated before runs, every input logged, unscripted input ⇒ intervention + non-autonomous. Interruption rule: SIGKILL at first-file-write + 120s, same trigger all lanes, one documented resume command. Tier 1 (baseline gate): categories bug-fix / brownfield / interrupt / fresh-repo-lifecycle; Tier 2 only if Tier 1 shows a gap worth explaining. Headline: ASR. Secondaries all script-computable; tokens and calls measured harness-side from provider usage — never a system's self-report (Aether's cost line is validated against this, not substituted). Wall-clock excludes operator-wait.

## E. Change Budget

New functionality (the only new build allowed): benchmark harness + operator script + acceptance scripts. Bug fixes: worktree GC guard + recover ordering; resolveTestCommand in merge gate; midden unification; merged-dispatch criteria; supersession check + shared advancePhase(); atomic-write migration + mandatory current_phase guard. Wiring: handoff contract into the brief's Expected Output; capsule/skill/pheromone fields on continue's external dispatch; Phase 185 cost line from spend/session.json. Simplification: briefs by path not bytes; TS hive double-injection deletion; pheromone/handoff dedup (one home each); plan-finalize shared validator. Deletion: model-routing.yaml, caste/phases/autopilot/memory-rules YAMLs; distribute-or-delete ruling on the four CWD-relative loaders (deletion preferred); pkg/trace/cost.go + pool; session-verify-fresh; dead bridge; false docs. Deferred (hypothesis-first, post-baseline only): observation-intake wiring, plan-reviewer caste, curation-per-phase, adaptive-Queen extensions, semantic dedup.

## F. Frozen Target (source of truth — outcome-level, no architecture named)

1. **Crash safety** — killing the process at any moment never causes on-disk work (including worktrees) to be deleted by a subsequent Aether command; recovery surfaces survivors with a named next command.
2. **Ecosystem neutrality** — every lifecycle command completes, or fails with a named next step, identically in non-Go projects.
3. **One failure ledger** — a failure recorded anywhere is visible to every consumer that claims to read failures.
4. **Complete worker contract** — every worker's prompt contains, for every task it covers: the task, its constraints/acceptance criteria, and the exact output contract the finalizer will enforce — on every platform path.
5. **Reviewer parity** — reviewers receive the same composed context regardless of dispatch path.
6. **No silent configuration** — a config file that exists but cannot load fails loudly or does not exist; downstream behaviour matches dev-checkout behaviour.
7. **Honest cost** — every build/continue ends with one plain-language cost line, measured vs estimated, from a persisted artifact, validated once against harness-side provider usage.
8. **Measured baseline** — committed benchmark artifact for GSD + both Aether lanes on identical tasks; no new subsystem without a hypothesis about which number it moves.
9. **Deterministic resumption** — one command reports position/next step consistent with git; inconsistency reported, never guessed over.
10. **Single advance discipline** — phase advance only through paths enforcing the same verification set; bypass requires explicit flag + recorded reason.
11. **Budgeted iteration** — every retry loop draws from a recorded budget; exhaustion reported with what was tried.
12. **No duplicate delivery** — no context section reaches the same worker twice; composition inspectable read-only.

Plus the owner constraint: the auxiliary command surface (/ant-oracle, /ant-dream, chaos, archaeology, swarm, council, …) is preserved; simplification touches only the main lifecycle path.

## G. Go / No-Go

**GO** — into implementation planning, sequence constrained by the change budget: benchmark first, bug fixes second, wiring third, everything else deferred until measured. The two remaining gaps are measurement gaps, not investigation gaps, and are the first two implementation steps: a real `--print-brief --full` capture (done in Phase 186 plan 04) and benchmark Tier 1. If either measurement contradicts this review, the plan changes before code does.
