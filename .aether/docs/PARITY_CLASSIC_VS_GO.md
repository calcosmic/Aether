# Classic Parity Checklist: Classic v5.4.0 vs Current Go Runtime

> **Version:** v1.24
> **Last Updated:** 2026-09-01 (Phase 198.3 autopilot contract)
> **Classic Baseline:** v5.4.0
> **Applies to:** Phases 152-159

---

## Purpose and Scope

This document is the golden reference for comparing Classic Aether v5.4.0 behaviour against the current Go runtime. It exists so every downstream planner and implementer knows what "correct" looks like.

For dummies: This is the scorecard. Classic v5.4.0 is the opponent we are trying to match or beat. Each row says: "here is what Classic did, here is what Go does now, and here is whether they agree."

---

## How to Use This Doc

Each row uses five columns:

| Column | Meaning |
|--------|---------|
| **Classic Behaviour** | What v5.4.0 did |
| **Go Equivalent** | What the current Go runtime does |
| **Status** | MATCH / GAP / DEGRADED / INTENTIONALLY_CHANGED |
| **Verification Method** | How we prove it |
| **Decision** | Classic / Go / Hybrid / PENDING |

**Decision rule:** When Classic and Go disagree, the user decides per item (D-06). There is no blanket "Classic wins" or "Go wins." Each behaviour is evaluated on its own merit.

### Counting Verifiable Items

To determine whether the parity checklist meets the "at least 50% verifiable" threshold (TEST-07), use the following counting method:

| Status | Counts as Verifiable? | Reason |
|--------|----------------------|--------|
| **MATCH** | Yes | Behaviour matches Classic; can be verified against golden tests |
| **DEGRADED** | Yes | Behaviour is observable but differs from Classic; still verifiable |
| **GAP** | No | Behaviour is missing; nothing to verify |
| **INTENTIONALLY_CHANGED** | No | Behaviour was deliberately altered; not a parity target |

**Threshold:** At least 50% of total checklist items (excluding Known Gaps) must be verifiable.

---

## Parity Checklist

| Classic Behaviour | Go Equivalent | Status | Verification Method | Decision |
|-------------------|---------------|--------|---------------------|----------|
| **Init ceremony** — Charter approval flow, repo scanning, governance detection, rich init-research (tech stack, directory analysis, 10 pheromone patterns) | Go `init_cmd.go` + `init_research.go` restores charter and research; pheromone suggestions via `suggest-analyze` | MATCH | Golden test: init produces charter, research, and pheromone suggestions | Hybrid |
| **Queen loads** — Colony-prime context assembly, pheromone injection, skill matching, token budget trimming | Go `colony_prime_context.go` assembles context; pheromones injected via `pheromone_loader.go`; skills matched via `skill-match` | MATCH | Golden test: build worker receives full context with pheromones and skills | Go |
| **Worker spawn** — Platform dispatch (Claude/OpenCode/Codex), spawn-log/spawn-complete, caste identity (emoji + ANSI + deterministic name) | Go `spawn.go` + platform dispatch; caste identity in `cmd/codex_visuals.go`; spawn tracking in `spawn_track.go` | DEGRADED | Golden test: worker spawn produces visible caste identity; manual check: Codex gets runtime-native visuals, Claude/OpenCode get wrapper framing | Hybrid |
| **Planner** — Scout planning, route-setter phase breakdown, oracle loop integration | Go `plan` + `phase` commands; scout survey in `survey_cmds.go`; route-setter logic in `codex_plan.go` | MATCH | Golden test: plan produces manifest with phases and task breakdown | Go |
| **Oracle loop** — RALF research, confidence iteration, research planning, diminishing returns detection | Go `oracle_loop.go` with confidence evaluation and research formulation; template-specific synthesis | MATCH | Golden test: oracle produces research plan and confidence score; 34 Oracle tests pass | Go |
| **Scout** — Codebase survey, territory mapping, source anchor extraction | Go `survey_cmds.go` + `codex_colonize.go`; source anchors wired through planner context | MATCH | Golden test: colonize produces survey with anchors | Go |
| **Build wave** — Manifest generation, worker brief assembly, parallel execution, worktree allocation | Go `build_flow_cmds.go` + `codex_build.go` + `codex_build_worktree.go`; manifest via `aether host build` | MATCH | Golden test: build produces manifest, spawns workers, merges worktrees | Hybrid |
| **Watcher** — Verification depth (light/standard/heavy), quality gates, test coverage analysis | Go verification depth in build/continue; quality gates in `gate.go`; probe coverage in test suite | MATCH | Golden test: continue at standard depth spawns watcher + probe; heavy adds auditor + gatekeeper | Go |
| **Gatekeeper** — Security scan, antipattern check (~6 patterns), blocker creation | Go `security_cmds.go` + `check-antipattern`; runs after verification | MATCH | Test: gatekeeper detects exposed secrets and debug artifacts | Go |
| **Probe** — Coverage analysis, test gap identification, coverage percentage reporting | Go test suite provides coverage; probe caste identified in visuals | GAP | Manual checklist: coverage gaps are identified and reported as part of continue review | PENDING |
| **Memory** — Learning pipeline, instinct promotion, hive store, trust scoring | Go `learning.go` + `instinct.go` + `hive.go`; trust scoring in `trust.go` | MATCH | Golden test: continue promotes learnings to instincts; high-confidence instincts reach hive | Go |
| **Lessons** — Phase learnings, hypothesis tracking, validated/disproven status | Go `learning_cmds.go` with hypothesis/validate/disprove lifecycle | MATCH | Golden test: continue extracts hypotheses and tracks evidence | Go |
| **Skills** — Skill index, skill match, skill inject, custom skill creation | Go `skills.go` + `skill-index` / `skill-match` / `skill-inject` subcommands | MATCH | Test: skill-match scores skills by worker role and pheromones | Go |
| **Events** — Event bus, NDJSON stream, event TTL, pub/sub | Go `eventbus.go` emits events; TTL cleanup in maintenance | DEGRADED | Golden test: events have type, timestamp, payload; manual: NDJSON stream is readable by TS host | Hybrid |
| **Recovery** — Stuck-state detection (7 classes), auto-repair, resume with full context | Go `recovery_snapshot.go` + `autofix.go`; 7 stuck-state classes detected; resume restores context | MATCH | E2E test: `aether recover` detects and fixes safe issues | Go |
| **Cleanup** — Data-clean, midden review, session sync, stale file removal | Go `data-clean` command + midden review in `midden_cmds.go`; session freshness checks | MATCH | Test: data-clean removes test artifacts; midden-review groups failures by category | Go |

### Summary

| Metric | Value |
|--------|-------|
| Total checklist items | 15 |
| Verifiable items (MATCH + DEGRADED) | 12 |
| Non-verifiable items (GAP + INTENTIONALLY_CHANGED) | 3 |
| Percentage verifiable | **80%** |
| Threshold | ≥50% — **PASS** |

---

## Verification Methods

### Automated: Golden, Snapshot, and Integration Tests

The 9 flagship workflows have automated anchors that prove or characterize
their relationship to the v5.4 baseline:

1. **build** — `TestGoldenBuildCeremony`
2. **continue** — `TestGoldenContinueCeremony`
3. **plan** — `TestGoldenPlanManifest`
4. **colonize** — `TestGoldenColonizeSurvey`
5. **autopilot** — joined-up proof: `TestOvernightRunCompletesSixPhases`; typed policy anchors: `TestAutopilotTriggerCatalogue`, `TestAutopilotDispositionMatrix`, and `TestRunDryRunUsesCanonicalTriggerCatalogue`
6. **seal** — `TestGoldenSealCeremony`
7. **entomb** — `TestGoldenEntombArchive`
8. **swarm** — `TestGoldenSwarmDashboard`
9. **oracle** — `TestGoldenOracleConfidenceLoop`

Reference: `v1.18-ROADMAP.md` PAR-01 through PAR-07 for the baseline verification that already passed.

The Phase 198.3 autopilot anchor supersedes the former broad pause golden test.
It proves one real six-phase `run` loop, durable visual/runtime/replan morning
work, genuine and normal stop boundaries, phase-scoped provider readiness,
unchanged blocker evidence, honest reporting, and the owner-only seal boundary.
The exact active dispositions are recorded in
`.planning/decisions/autopilot-pause-conditions.md`; historical marker-scanner
behavior is not current parity authority.

### Manual: Checklist Steps

For edge cases and ceremony that automated tests cannot fully capture:

- [ ] Init produces a charter the user can approve or edit
- [ ] Build workers receive caste identity in output (emoji + label + name)
- [ ] Continue at light depth does not overspawn
- [ ] Oracle stops iterating when confidence plateaus (diminishing returns)
- [ ] Swarm dashboard shows animated spinners and chamber activity
- [ ] Seal produces a review summary before finalizing
- [ ] Recovery detects stale session and offers to clean it

---

## Known Gaps

These Classic behaviours are intentionally not restored (per PAR-07, v1.18):

| Gap | Reason | Status |
|-----|--------|--------|
| Raw Bash state mutation | Violates architecture boundary — Go owns state | INTENTIONALLY_CHANGED |
| Visual output parsing as authority | Violates architecture boundary — never parse banners for state | INTENTIONALLY_CHANGED |
| 100+ default agents | Out of scope for v1.24 — 8 core agents maximum | INTENTIONALLY_CHANGED |
| Cross-colony ledger sharing | Findings contain repo-specific paths that go stale | INTENTIONALLY_CHANGED |
| Real-time ledger sync across agents | YAGNI — agents write serially during build/continue | INTENTIONALLY_CHANGED |
| Interactive shell | Deferred to post-v1.24 | INTENTIONALLY_CHANGED |
| Vector backend for memory | File-backed memory sufficient for this milestone | INTENTIONALLY_CHANGED |

---

## Decision Log

| Item | Decision | Rationale | Source |
|------|----------|-----------|--------|
| Go owns state mutation | Go | Safety, atomicity, file locking | v1.16 boundary contract |
| TS owns orchestration | TypeScript | Iteration speed, platform adapters, prompt contracts | v1.16 research |
| Assets own behaviour definition | Markdown/YAML/JSON | Human-editable, version-controlled, distributed | D-01 through D-04 |
| 8 core agents max | Hybrid | Scope control for v1.24; more agents deferred | v1.24 PROJECT.md |
| File-backed memory sufficient | Go | Vector backend deferred; SQLite FTS works | v1.23 STATE.md |
| Golden tests for 9 workflows | Automated | Proves parity without manual ceremony every time | v1.18 PAR-01..PAR-06 |

---

## Cross-References

- **Architecture boundary:** See [ARCHITECTURE_BOUNDARY.md](ARCHITECTURE_BOUNDARY.md) for the three-tier boundary and asset-type matrix.
- **v1.18 milestone:** See `.planning/milestones/v1.18-ROADMAP.md` for the Classic baseline establishment and golden test verification.
- **v1.18 requirements:** See `.planning/milestones/v1.18-REQUIREMENTS.md` for PAR-01 through PAR-07.
- **Requirements:** See `.planning/REQUIREMENTS.md` §Boundary & Parity for BOUNDARY-03.
- **Hybrid strategy:** See `.aether/docs/hybrid-runtime-strategy-research.md` for the research that defined this parity scope.
- **Command parity matrix:** See `.aether/docs/classic-command-parity-matrix.md` for the command-by-command authority map.

---

*Documented for Phase 152. Serves as the golden reference for Phases 153-159.*
