# Aether Retrospective

## Milestone: v1.11 — Aether Unification

**Shipped:** 2026-04-30
**Phases:** 10 | **Plans:** 18

### What Was Built

- Self-hosting cleanup removing stale agents, duplicate commands, orphaned files
- Platform hardening across Claude Code, OpenCode, and Codex CLI
- Smart Init charter ceremony with Go-native approval flow and ANSI rendering
- Rich init-research with tech stack analysis, governance, and pheromone suggestions
- Suggest-analyze for automatic pheromone suggestions during builds
- Intelligence core with Bayesian confidence scoring and circuit breaker events
- Ceremony data surfacing with research display and event bus wiring
- UX improvements including onboarding, feedback, and ceremony polish
- Platform test coverage expansion and documentation/validation hygiene

### What Worked

- Documentation-only phases (79) for closing audit gaps — fast, low-risk
- TDD discipline maintained across most implementation phases
- GSD executor agents with worktree isolation — parallel execution with no conflicts
- Incremental verification with post-merge test gates catching integration issues

### What Was Inefficient

- Some phases had pre-existing test failures that complicated verification
- Worktree merge-back required manual cleanup of stale worktrees from prior sessions
- Audit items from Phase 71 (partial test coverage) carried through as tech debt
- Multiple phases had human verification items that remain pending

### Patterns Established

- Documentation/validation phases as gap-closure strategy
- Ceremony event bus as the standard routing mechanism for init intelligence
- JSON round-trip pattern for type conversion from interface{} to typed structs

### Key Lessons

- Always clean up stale worktrees before starting new worktree-based execution
- Phase audits should be run mid-milestone, not just at the end, to catch gaps early
- Human verification items accumulate across phases — need periodic UAT sweeps
- Stub implementations should be tracked explicitly to prevent them becoming permanent

### Cost Observations

- Model mix: primarily sonnet for execution, opus for planning
- Sessions: ~12 development sessions over 75 days
- Notable: Documentation phases take minutes; code phases take 10-15 minutes each

---

## Milestone: v1.26 — Intelligent Orchestration (rescoped to hardening)

**Shipped:** 2026-08-22 (override close)
**Phases:** 15 of 19 | **Plans:** 66 archived

### What Was Built

- Wiring ratchet: no feature without a caller; release gate proven by execution (172)
- Fail-closed delegation guards with a whole-run ceiling and tamper-refusing ledgers (173)
- Token measurement with stated arithmetic; the 186× undercount fixed (174, plans 1-2)
- Hardening H1-H5: internals stop leaking into other projects; task-first skill scoring; castes with nothing to do refused; the cap caps (180-184)
- Crash-safe worktrees; one truth for failures and advances; complete non-duplicated briefs; dead wood deleted with reappearance ratchets (187-191)
- Field hardening from three downstream repos in one day; honest out-of-band re-entry (191.1)
- Team check-in, owner decision routing, honest no-change results, the fresh-eyes review gate (2026-08-21, outside GSD)

### What Worked

- Execution-based proofs over text checks: every criterion that "reads ci.yml and runs it" survived review; every substring check did not
- Reproduce-the-field-failure-first: 191.1's four fixes each began as a failing test built from a downstream transcript
- Rescoping by removal (2026-08-14): cutting four phases and shipping five "remove things" items moved the owner-felt cost more than any addition
- Owner rulings recorded as decision files (D1-D12) — later sessions could find the authority without re-litigating

### What Was Inefficient

- The milestone was rescoped twice (2026-08-13 reshape, 2026-08-14 cut) and extended once (2026-08-17 addendum); the roadmap checklist never listed 180-192, so the GSD archive step found only 3 phases at close and the record had to be repaired by hand
- 19 of 22 acknowledged artifacts at close were archived-phase leftovers (v1.11) and bookkeeping drift, not live gaps — earlier closes had not run the audit
- Work landed outside the phase system (175, 180-184, the 2026-08-21 sessions) was real but invisible to GSD's progress counters until reconciled
- The one benchmark run that would have measured token cost after hardening (186-07) was never fired — so the milestone ends with no post-hardening measurement

### Patterns Established

- Ratchets over rules: allowlists that can only shrink, with a committed baseline
- "Definition of Done = a command that fails when the requirement is unmet" applied to every phase criterion
- Decision records in `.planning/decisions/` as the authority trail; the owner's words quoted verbatim

### Key Lessons

- A floor written in code ("Watcher always", "production ⇒ auditor") quietly defeats the judgement the system was built to exercise; measured 2026-08-22: a 1-task bug fix → 8 workers
- Verifying the same files two or three times is the largest token cost, not any single specialist
- Keep the roadmap checklist in sync when phases are inserted by addendum, or the close tooling cannot see them

### Cost Observations

- Model mix: opus for planning/verification, sonnet for execution; the three `inherit` roles (chronicler, keeper, includer) silently follow the parent
- Sessions: ~25 over 15 days; three overnight multi-phase runs
- Notable: post-hardening token cost unmeasured — v1.27 feature 4 (cost line) and feature 7 (benchmark) close that gap

---

## Cross-Milestone Trends

| Milestone | Phases | Plans | Duration | Key Theme |
|-----------|--------|-------|----------|------------|
| v1.26 | 15/19 | 66 | 15 days | Hardening: remove until a small job costs a small amount |
| v1.11 | 10 | 18 | 75 days | Unification and intelligence |
| v1.10 | 14 | 34 | ~10 days | Colony polish |
| v1.9 | 5 | 8 | ~2 days | Review persistence |
| v1.8 | 3 | 6 | ~1 day | Colony recovery |
| v1.7 | 2 | 2 | <1 day | Pipeline recovery |
| v1.6 | 8 | 13 | ~2 days | Release integrity |
