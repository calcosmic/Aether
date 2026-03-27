# Roadmap: Aether

## Milestones

- ✅ **v1.3 Maintenance & Pheromone Integration** — Phases 1-8 (shipped 2026-03-19)
- ✅ **v2.1 Production Hardening** — Phases 9-16 (shipped 2026-03-24)
- ✅ **v2.2 Living Wisdom** — Phases 17-20 (shipped 2026-03-25)
- ✅ **v2.3 Per-Caste Model Routing** — Phases 21-24 (shipped 2026-03-27)
- 🔄 **v2.4 Living Wisdom** — Phases 25-28 (in progress)

## Phases

<details>
<summary>v1.3 Maintenance & Pheromone Integration (Phases 1-8) — SHIPPED 2026-03-19</summary>

- [x] Phase 1: Data Purge (2/2 plans) — completed 2026-03-19
- [x] Phase 2: Command Audit & Data Tooling (2/2 plans) — completed 2026-03-19
- [x] Phase 3: Pheromone Signal Plumbing (3/3 plans) — completed 2026-03-19
- [x] Phase 4: Pheromone Worker Integration (2/2 plans) — completed 2026-03-19
- [x] Phase 5: Learning Pipeline Validation (2/2 plans) — completed 2026-03-19
- [x] Phase 6: XML Exchange Activation (2/2 plans) — completed 2026-03-19
- [x] Phase 7: Fresh Install Hardening (2/2 plans) — completed 2026-03-19
- [x] Phase 8: Documentation Update (2/2 plans) — completed 2026-03-19

</details>

<details>
<summary>v2.1 Production Hardening (Phases 9-16) — SHIPPED 2026-03-24</summary>

- [x] Phase 9: Quick Wins (2/2 plans) — completed 2026-03-24
- [x] Phase 10: Error Triage (3/3 plans) — completed 2026-03-24
- [x] Phase 11: Dead Code Deprecation (2/2 plans) — completed 2026-03-24
- [x] Phase 12: State API & Verification (3/3 plans) — completed 2026-03-24
- [x] Phase 13: Monolith Modularization (9/9 plans) — completed 2026-03-24
- [x] Phase 14: Planning Depth (2/2 plans) — completed 2026-03-24
- [x] Phase 15: Documentation Accuracy (3/3 plans) — completed 2026-03-24
- [x] Phase 16: Ship (2/2 plans) — completed 2026-03-24

</details>

<details>
<summary>v2.2 Living Wisdom (Phases 17-20) — SHIPPED 2026-03-25</summary>

- [x] Phase 17: Local Wisdom Accumulation — completed 2026-03-24
- [x] Phase 18: Local Wisdom Injection — completed 2026-03-25
- [x] Phase 19: Cross-Colony Hive — completed 2026-03-25
- [x] Phase 20: Hub Wisdom Layer — completed 2026-03-25

</details>

<details>
<summary>v2.3 Per-Caste Model Routing (Phases 21-24) — SHIPPED 2026-03-27</summary>

- [x] Phase 21: Test Infrastructure Refactor (3 plans) — completed 2026-03-27
- [x] Phase 22: Config Foundation & Core Routing (3 plans) — completed 2026-03-27
- [x] Phase 23: Tooling & Overrides (2 plans) — completed 2026-03-27
- [x] Phase 24: Safety & Verification (2 plans) — completed 2026-03-27

</details>

<details>
<summary>v2.4 Living Wisdom (Phases 25-28) — IN PROGRESS</summary>

- [ ] **Phase 25: Agent Definitions (Oracle + Architect)** — AGNT-01, AGNT-02, AGNT-03, AGNT-04, AGNT-05
- [ ] **Phase 26: Wisdom Pipeline Wiring** — PIPE-01, PIPE-02, PIPE-04
- [ ] **Phase 27: Deterministic Fallback + Dedup** — PIPE-03, VAL-02
- [ ] **Phase 28: Integration Validation** — VAL-01

</details>

## Phase Details

### Phase 25: Agent Definitions (Oracle + Architect)

**Requirements:** AGNT-01, AGNT-02, AGNT-03, AGNT-04, AGNT-05
**Research needed:** No — all 22 existing agents follow identical structure
**Depends on:** Nothing (purely additive)

Create dedicated agent definition files for Oracle and Architect castes, filling the two documented gaps in the agent roster. Both get opus model slot routing for reasoning depth. Oracle is spawnable by Queen during builds (not just via /ant:oracle command). Architect has a design-create mode for writing architecture docs.

**Plans:** 2 plans

Plans:
- [ ] 25-01-PLAN.md — Create Oracle + Architect agent definitions and mirrors (6 new files)
- [ ] 25-02-PLAN.md — Wire agents into build flow and update documentation (5 files modified)

**Success criteria:**
1. `aether-oracle.md` exists in `.claude/agents/ant/` with `model: opus` frontmatter and proper role/execution_flow/pheromone_protocol sections
2. `aether-architect.md` exists in `.claude/agents/ant/` with `model: opus` frontmatter, design-create mode, and distinct role from Keeper and Route-Setter
3. Both agent files are mirrored to `.opencode/agents/` and `.aether/agents-claude/` with structural parity
4. Queen's build-wave playbook references Oracle as a spawnable worker caste (not only slash-command-triggered)
5. Agent count in workers.md and CLAUDE.md updated from 22 to 24

---

### Phase 26: Wisdom Pipeline Wiring

**Requirements:** PIPE-01, PIPE-02, PIPE-04
**Research needed:** Moderate — continue-advance.md is 434 lines; insertion points must be verified
**Depends on:** Phase 25 (agents must exist before pipeline references them)

Wire the existing wisdom functions into the continue-advance flow so that wisdom accumulates automatically during colony work. Add Step 2.6 to call `queen-write-learnings` after learning extraction, and Step 3d to call `hive-promote` after instinct promotion. Both steps are non-blocking (failures logged but never stop the continue flow). Add visible feedback so users see when wisdom is written.

**Success criteria:**
1. After running `/ant:continue`, the QUEEN.md `## Build Learnings` section contains new entries from the completed phase
2. After running `/ant:continue` with high-confidence instincts (>= 0.8), `~/.aether/hive/wisdom.json` receives new entries
3. Continue output displays a wisdom summary line (e.g., "3 learnings recorded, 1 instinct promoted to hive")
4. Pipeline steps are non-blocking — if queen-write-learnings or hive-promote fail, continue completes normally with a logged warning

---

### Phase 27: Deterministic Fallback + Dedup

**Requirements:** PIPE-03, VAL-02
**Research needed:** Moderate — git-diff-based extraction quality is unvalidated
**Depends on:** Phase 26 (pipeline must be wired before fallback can push data through it)

Add a deterministic fallback for builder learning extraction. When AI agents skip learning output, extract learnings from git diff + test results. Also add content normalization to instinct deduplication so semantically similar instincts consolidate (not just SHA-256 exact match).

**Success criteria:**
1. When a builder produces synthesis JSON without `learning.patterns_observed`, the fallback extracts at least one learning from git diff and writes it to COLONY_STATE
2. Creating an instinct similar to an existing one (same topic, different wording) consolidates into a single entry with incremented confidence, not a duplicate
3. Content normalization handles common variations: whitespace, casing, punctuation, synonym substitution at the word level
4. The fallback path is testable with a mock git diff producing deterministic learnings

---

### Phase 28: Integration Validation

**Requirements:** VAL-01
**Research needed:** No — follows established colony lifecycle test patterns
**Depends on:** Phases 25, 26, 27 (validates the full chain)

Write an end-to-end integration test that verifies the complete wisdom flow: build produces work, continue extracts learnings, QUEEN.md gets populated, hive brain receives promoted instincts. Also update documentation to reflect the 24-agent roster and wired wisdom pipeline.

**Success criteria:**
1. Integration test passes: init -> plan -> build -> continue -> QUEEN.md has learnings -> hive brain has entries
2. All 584+ existing tests remain passing (no regressions)
3. `bin/validate-package.sh` passes (new agent files included in package)
4. CLAUDE.md and workers.md accurately reflect 24 agents and the wisdom pipeline behavior

---

## Progress

**Execution Order:**
Phases execute in numeric order: 25 -> 26 -> 27 -> 28

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Data Purge | v1.3 | 2/2 | Complete | 2026-03-19 |
| 2. Command Audit & Data Tooling | v1.3 | 2/2 | Complete | 2026-03-19 |
| 3. Pheromone Signal Plumbing | v1.3 | 3/3 | Complete | 2026-03-19 |
| 4. Pheromone Worker Integration | v1.3 | 2/2 | Complete | 2026-03-19 |
| 5. Learning Pipeline Validation | v1.3 | 2/2 | Complete | 2026-03-19 |
| 6. XML Exchange Activation | v1.3 | 2/2 | Complete | 2026-03-19 |
| 7. Fresh Install Hardening | v1.3 | 2/2 | Complete | 2026-03-19 |
| 8. Documentation Update | v1.3 | 2/2 | Complete | 2026-03-19 |
| 9. Quick Wins | v2.1 | 2/2 | Complete | 2026-03-24 |
| 10. Error Triage | v2.1 | 3/3 | Complete | 2026-03-24 |
| 11. Dead Code Deprecation | v2.1 | 2/2 | Complete | 2026-03-24 |
| 12. State API & Verification | v2.1 | 3/3 | Complete | 2026-03-24 |
| 13. Monolith Modularization | v2.1 | 9/9 | Complete | 2026-03-24 |
| 14. Planning Depth | v2.1 | 2/2 | Complete | 2026-03-24 |
| 15. Documentation Accuracy | v2.1 | 3/3 | Complete | 2026-03-24 |
| 16. Ship | v2.1 | 2/2 | Complete | 2026-03-24 |
| 17. Local Wisdom Accumulation | v2.2 | Complete | Complete | 2026-03-24 |
| 18. Local Wisdom Injection | v2.2 | Complete | Complete | 2026-03-25 |
| 19. Cross-Colony Hive | v2.2 | Complete | Complete | 2026-03-25 |
| 20. Hub Wisdom Layer | v2.2 | Complete | Complete | 2026-03-25 |
| 21. Test Infrastructure Refactor | v2.3 | Complete | Complete | 2026-03-27 |
| 22. Config Foundation & Core Routing | v2.3 | Complete | Complete | 2026-03-27 |
| 23. Tooling & Overrides | v2.3 | Complete | Complete | 2026-03-27 |
| 24. Safety & Verification | v2.3 | 2/2 | Complete | 2026-03-27 |
| 25. Agent Definitions (Oracle + Architect) | v2.4 | 2 | Planned | — |
| 26. Wisdom Pipeline Wiring | v2.4 | 0 | Pending | — |
| 27. Deterministic Fallback + Dedup | v2.4 | 0 | Pending | — |
| 28. Integration Validation | v2.4 | 0 | Pending | — |

## Coverage Matrix

| Requirement | Phase | Status |
|-------------|-------|--------|
| AGNT-01 (Oracle agent file) | 25 | Pending |
| AGNT-02 (Architect agent file) | 25 | Pending |
| AGNT-03 (Agent mirrors) | 25 | Pending |
| AGNT-04 (Oracle spawnable by Queen) | 25 | Pending |
| AGNT-05 (Architect design-create mode) | 25 | Pending |
| PIPE-01 (queen-write-learnings in continue) | 26 | Pending |
| PIPE-02 (hive-promote in continue) | 26 | Pending |
| PIPE-04 (Visible wisdom feedback) | 26 | Pending |
| PIPE-03 (Deterministic fallback) | 27 | Pending |
| VAL-02 (Content normalization dedup) | 27 | Pending |
| VAL-01 (E2E integration test) | 28 | Pending |

**Coverage: 11/11 requirements mapped (100%)**
