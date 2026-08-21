# Phase 186: Baseline Showdown (Light) - Context

**Gathered:** 2026-08-17
**Status:** Ready for planning
**Source:** PRD Express Path (approved implementation plan of 2026-08-17, "Aether ≥ GSD — Implementation Programme"; owner decisions locked during its planning session)

<domain>
## Phase Boundary

This phase delivers a working Aether-vs-GSD benchmark harness and a committed light baseline: 4 task categories × 3 lanes × 1 run = 12 real runs, with raw evidence preserved and results reproducible via one documented command. It runs BEFORE any code changes elsewhere in the programme — the baseline is what every later phase is measured against.

Out of scope for this phase: the full 3-repeat comparison (that is Phase 192), any change to Aether's runtime code, any change to GSD, and any scoring dimension that requires human judgment.

</domain>

<decisions>
## Implementation Decisions

### Benchmark protocol (locked by owner + approved plan)
- Light tier only: 1 run per cell, 12 runs total. The full 36-run tier is Phase 192's job.
- Three lanes, scored separately: (1) GSD (`/gsd-plan-phase` → `/gsd-execute-phase` → `/gsd-verify-work`); (2) Aether interactive (`/ant-init` → `/ant-plan` → `/ant-build` → `/ant-continue`); (3) Aether autopilot (`/ant-run`). The two Aether lanes are different execution machines and must never be pooled.
- Aether runs in in-repo mode. Worktree mode is disqualified until Phase 187 lands — record this in the results, do not hide it.
- Four task categories: (a) small bug fix with a seeded failing test; (b) brownfield feature touching several files; (c) interrupted execution — SIGKILL the process at first-file-write + 120 seconds, resume with exactly one documented command; (d) fresh-repo full lifecycle (install → init → full cycle → seal for Aether; new-project → phase cycle for GSD).
- Substrate: two small neutral open-source repos, one Go and one TypeScript, pinned at a fixed commit, fresh clone per run. The Aether repo and any repo with Aether or GSD history is disqualified (GSD authored this repo's .planning/ — home-field advantage). The Go/TS split is load-bearing: it exposes ecosystem assumptions.
- Hermetic environment per lane: a clean HOME containing only that lane's system (fresh `aether install` hub, or fresh GSD file-copy install), empty memory, identical minimal global CLAUDE.md, same CLI version, same model.
- Model parity: one pinned model for orchestrator AND workers in both systems. Aether agent frontmatter forced to `inherit` for benchmark runs via configuration, not code edits; if impossible, the deviation is recorded in the results.
- Operator script: every permitted human input enumerated BEFORE runs (approvals, first-option selections, scripted answers). Every actual input logged with timestamps. Any unscripted input counts as an intervention and marks the run non-autonomous.
- Acceptance: one deterministic script per task (exit code 0 = pass), written BEFORE any run, stored in `bench/acceptance/`, executed against a fresh clone of the run's result. Scripts must avoid both systems' verification vocabularies (no reuse of GSD must_haves patterns or Aether criterion bindings).
- Metrics, all script-computed: autonomous success rate; interventions (typed: scripted vs unscripted); hallucinated completion (system claimed success AND acceptance script failed); unnecessary modifications (diff paths vs a per-task allowlist); git cleanliness (orphan branches, tool internals, uncommitted residue — one checker script); resume/recovery success; tokens and model calls measured HARNESS-SIDE from provider usage (never a system's self-report); wall-clock excluding operator-wait time (computable from operator-log timestamps).
- Unscored qualitative appendix: plan quality notes, worker-prompt relevance (via `aether build --print-brief` for Aether, PLAN.md inspection for GSD).

### Sequencing gate (locked)
- FIRST MILESTONE, before any task specs are written: hermetic-profile smoke — a clean HOME boots, authenticates, and completes one trivial task in BOTH systems. This is the harness's biggest practical risk (macOS Keychain auth, `getpwuid` home resolution vs `$HOME`, GSD install-by-copy). Nothing else is built until this passes.
- Stop-check: if the harness is not producing numbers within one week of work, cut harness scope (fewer metrics, same 12 runs). The harness must never become its own framework — it is shell scripts, task specs, and a results table.

### Evidence (locked)
- Everything lives in `bench/` in this repo: `bench/` harness scripts, `bench/tasks/` task specs, `bench/acceptance/` acceptance scripts, `bench/results/<date>/` raw transcripts + operator logs + results table.
- One documented command reproduces a run.
- Acceptance scripts must predate run timestamps (verifiable via git history).
- Additionally committed: a read-only `aether build --print-brief --full` capture from a real mid-project colony (context-composition measurement; the command is verified non-mutating).

### Requirements traceability
- PROOF-01 (remapped): real development tasks in real repositories with interventions counted per task in a committed artifact — satisfied by the 12-run baseline with the operator log.
- PROOF-03 (remapped): tokens measured and recorded, favourable or not — satisfied by harness-side provider-usage measurement (the roadmap's original `aether spend` reference is dead; this is the reconciliation).

### Claude's Discretion
- Which two OSS repos, chosen against these criteria: small (roughly under ~10k LOC), real test suite that runs in under a couple of minutes, buildable offline after one dependency fetch, active enough to be realistic but pinned at one commit, zero Aether/GSD history, licenses permitting benchmark use. Record the choice and the pinned SHAs in `bench/tasks/`.
- Exact task wording per category, provided each task's acceptance script is deterministic and written first.
- Harness implementation details (bash vs small scripts, directory layout inside `bench/`), provided the one-command-reproduces-a-run property holds.
- Which model to pin, defaulting to an inexpensive widely-available model both systems can run; record the exact model ID in results.
- How provider usage is captured harness-side (session transcripts, API usage logs), provided it is the same mechanism for all three lanes.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Programme governance
- `.planning/ROADMAP.md` — "Audit Addendum" section: Phase 186's success criteria and the full phase sequence
- `.planning/HARDENING-PLAN.md` — finish line + 2026-08-17 addendum; governance dials (research off, plan-checker on)
- `/Users/callumcowie/.claude/plans/the-audit-and-adversarial-declarative-bird.md` — the approved implementation plan (frozen outcomes §1, benchmark spec §2, acceptance gate §6)

### Reusable machinery
- `scripts/smoke-daily-driver.sh` — the existing hermetic-HOME pattern (isolated HOME, fresh install) to extend, not reinvent
- `cmd/build_print_brief.go` — the read-only brief-composition printer used for the context capture
- `cmd/spend_session_capture.go` — hook-side usage capture (context for why harness-side measurement is the source of truth: system self-report is Phase 185's job and gets VALIDATED against the harness, never substituted)

</canonical_refs>

<specifics>
## Specific Ideas

- The interrupted-execution kill rule is system-neutral by design: first file write + 120s, SIGKILL, one resume command (`/gsd-resume-work` equivalent for GSD; `/ant-resume` then the runtime's named recovery command for Aether). Same rule, all lanes.
- The git-cleanliness checker is one script run against the substrate repo after each run: counts orphan branches, files matching tool-internal patterns (`.aether/`, `.planning/` beyond what the lane legitimately creates as its working state — define per lane), and uncommitted residue.
- Results table format: one row per run (lane, category, repo, ASR pass/fail, interventions scripted/unscripted, hallucinated-completion flag, unnecessary-mod count, cleanliness score, tokens, calls, wall-clock-net), plus a per-lane summary block. Plain markdown, generated by script.
- The fresh-repo lifecycle category for Aether must also assert the Aether source repo's own `git status` is clean after the run (PROOF-02's check, previewed here, gated in 192).

</specifics>

<deferred>
## Deferred Ideas

- Full 3-repeat tier and the acceptance gate — Phase 192.
- Worktree-mode lane — after Phase 187, reported non-gating in 192.
- Token-guardrail ratification (the provisional 1.5× figure) — owner decision when this phase's data lands, recorded as a dated decision.
- Any remediation of gaps the baseline exposes — at most one hypothesis-first inserted phase, per the programme's conditional-phase rule.

</deferred>

---

*Phase: 186-baseline-showdown-light*
*Context gathered: 2026-08-17 via PRD Express Path (approved implementation plan)*
