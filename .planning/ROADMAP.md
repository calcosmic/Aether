# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped 2026-04-21)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-38 (completed 2026-04-23, product v1.0.20)
- **v1.6 Release Pipeline Integrity** - Phases 39-46 (completed 2026-04-24)
- **v1.7 Planning Pipeline Recovery** - Phases 47-48 (completed 2026-04-24)
- **v1.8 Colony Recovery** - Phases 49-51 (shipped 2026-04-25)
- **v1.9 Review Persistence** - Phases 52-56 (shipped 2026-04-26)
- **v1.10 Colony Polish** - Phases 57-69 (shipped 2026-04-28)
- **v1.11 Aether Unification** - Phases 70-79 (shipped 2026-04-30)
- **v1.12 Safe Colony** - Phases 80-87 (shipped 2026-05-01)
- **v1.13 Recovery Hardening & Hive Learning** - Phases 88-92 (shipped 2026-05-03)
- **v1.14 Queen Authority** - Phases 93-99 (shipped 2026-05-04)
- **v1.15 Framework Coherence, Efficiency, and Ship Readiness** - Phases 100-105 (shipped 2026-05-08)
- **v1.16 Hybrid Runtime Boundary and Orchestration Recovery** - Phases 106-111 (shipped 2026-05-13)
- **v1.17 Classic Restoration** - Phases 112-118 (shipped 2026-05-14) — [Archive](milestones/v1.17-ROADMAP.md)
- **v1.18 Hybrid Runtime Parity & Release Gate** - Phases 119-123 (shipped 2026-05-14) — [Archive](milestones/v1.18-ROADMAP.md)
- **v1.19 TypeScript Host Cutover + Oracle Confidence Recovery** - Phases 124-129 (shipped 2026-05-15) — [Archive](milestones/v1.19-ROADMAP.md)
- **v1.20 Host Contract Hardening and Wrapper Reality Check** - Phases 130-135 (shipped 2026-05-15) — [Archive](milestones/v1.20-ROADMAP.md)
- **v1.21 Live Colony** - Phases 136-140 (shipped 2026-05-18) — [Archive](milestones/v1.21-ROADMAP.md)
- **v1.22 Grounded Planning + Ceremony Restore** - Phases 141-144 (in progress)

## Phases

### Phase 141: Survey Noise Filter + Source Anchors

**Goal:** Fix the root cause of generic plans — survey trusts dependency/cache paths instead of repo-owned source files.

**Requirements:**
- GROUND-01: Unify 5 divergent skip lists into one canonical `ScanFilter` in `pkg/codegraph/scan_filter.go`
- GROUND-02: Extend filter to cover `.venv`, `__pycache__`, `site-packages`, `.pytest_cache`, `.tox`, `.mypy_cache`, `node_modules/.cache`, `.gradle`, `.cargo/registry`
- GROUND-03: Extract source anchors from cleaned survey output (top 50 non-dependency source files, depth-first)
- GROUND-04: Write source anchors to survey output so colony-prime and planners can use them
- GROUND-05: Regression test: M4L fixture with `.venv` noise produces clean survey with zero `.venv` references

**Plans:** 2/2 plans complete

Plans:
- [x] 141-01-PLAN.md — Create canonical ScanFilter, migrate 5 call sites, add regression tests
- [x] 141-02-PLAN.md — Extract source anchors, wire through survey output to planner context

**Why first:** Everything else depends on clean survey data. Grounding, anchors, and ceremony all assume the survey is right. Currently it's wrong.

**Risk:** Low. The skip lists already exist — this unifies and extends them. Zero new dependencies.

---

### Phase 142: Grounding Gate + Decision Binding

**Goal:** Make plans specific to the repo and bind discuss decisions into planning as hard constraints.

**Requirements:**
- GROUND-06: Plan-grounding validation gate — scan plan tasks for concrete file/path references; emit soft warning when anchors exist but plan has zero file refs
- GROUND-07: Gate is soft (warning, not rejection) — research/architecture phases may legitimately lack file targets
- GROUND-08: Discuss decision binding — auto-emit REDIRECT pheromones when `HardConstraint: true` decisions are resolved
- GROUND-09: Colony-prime injects resolved decisions into planner worker context
- GROUND-10: Decision conflict detection — warn when resolved decisions contradict each other (e.g., "use PostgreSQL" + "keep serverless")

**Why second:** Depends on Phase 141 for clean anchors. The grounding gate is only useful if the survey is trustworthy.

**Risk:** Low-medium. Grounding regex is simple. Decision conflict detection needs design but scope is small.

**Plans:** 2/2 plans complete

Plans:
- [x] 142-01-PLAN.md — Plan-grounding validation gate + planner brief anchor hint (GROUND-06, GROUND-07)
- [x] 142-02-PLAN.md — Decision conflict detection with FEEDBACK pheromone emission (GROUND-10)


---

### Phase 143: Build + Plan Ceremony Restore

**Goal:** Restore playbook-driven ceremony for build and plan workflows — one conductor, not three.

**Requirements:**
- CEREMONY-01: TS host reads build playbooks (build-prep, build-context, build-wave, build-verify, build-complete) and follows their step sequences
- CEREMONY-02: TS host reads plan playbooks and follows Scout → Route-Setter flow with grounding gate integration
- CEREMONY-03: YAML slimmed to packaging only — remove orchestration procedure from `build.yaml` wrapper_additions.orchestration (the 15-step duplicate)
- CEREMONY-04: One conductor per platform: Claude/OpenCode wrappers + TS host (playbook-driven), Codex (runtime-native command-guide + skills)
- CEREMONY-05: Go ceremony adapter emits events; TS host renders them following playbook timing
- CEREMONY-06: Document-injection pattern for playbook consumption (not step-parsing) — same pattern as existing Go `renderBuildPlaybookContext`

**Plans:** 2/2 plans complete

Plans:
- [x] 143-01-PLAN.md — PlaybookLoader module, plan playbooks, YAML orchestration removal (CEREMONY-01, CEREMONY-02, CEREMONY-03, CEREMONY-06)
- [x] 143-02-PLAN.md — Host integration, conductor unification, ceremony-coupled test updates (CEREMONY-04, CEREMONY-05)

**Why third:** Ceremony restore is medium complexity and touches all 3 platform surfaces. Requires grounding (Phase 142) to be meaningful — pretty ceremony on bad plans is worse than no ceremony.

**Risk:** Medium-high. 465 TS tests are ceremony-coupled and will break. Phase-by-phase test updates needed.

---

### Phase 144: Regression + Execution Path Cleanup

**Goal:** Prove everything works end-to-end. One declared execution path per workflow per platform. Clean up stragglers.

**Requirements:**
- CLEAN-01: M4L regression test: `.venv` fixture produces grounded plan with concrete file references, zero generic tasks
- CLEAN-02: Execution path audit: one declared path per workflow (build, plan, continue, colonize, seal) per platform (Claude/OpenCode, Codex)
- CLEAN-03: Audit ceremony-coupled TS tests — document expected breakage from Phase 143 changes, fix broken tests
- CLEAN-04: YAML packaging-only verification — no orchestration logic remains in YAML wrapper_additions
- CLEAN-05: Codex smoke test: command-guide + skills produce correct behavior without playbook loading
- CLEAN-06: Cross-platform parity: build ceremony output matches between Claude Code and OpenCode

**Plans:** 1/2 plans executed

Plans:
- [x] 144-01-PLAN.md — Fix YAML anchors, Go test failures, CLI flag audit (CLEAN-03, CLEAN-04)
- [ ] 144-02-PLAN.md — M4L regression test, execution path audit, Codex smoke, parity check (CLEAN-01, CLEAN-02, CLEAN-05, CLEAN-06)

**Why last:** Validates all previous phases. The M4L fixture is the proof that the entire pipeline (survey → anchor → grounding → ceremony) works correctly.

**Risk:** Medium. Test fixes from Phase 143 changes are expected but bounded.

## Phase Dependencies

```
141 (Noise Filter + Anchors)
  └── 142 (Grounding Gate + Decisions)
        └── 143 (Ceremony Restore)
              └── 144 (Regression + Cleanup)
```

Strict sequential. Each phase builds on the previous one.

## Success Criteria

- M4L fixture produces a grounded plan with zero `.venv` references
- Plans reference concrete repo files, not generic descriptions
- Discuss decisions appear as REDIRECT pheromones in planning context
- Build ceremony comes from playbooks, not hardcoded TS/Go logic
- One execution path per workflow per platform — no dual conductors
- All 2900+ Go tests and 465+ TS tests green

## Next Milestone

_Planning phases 141-144 defined. Run `/gsd-plan-phase 141` to start._
