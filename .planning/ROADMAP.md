# Roadmap: Aether

## Milestones

- ✅ **v1.3 Maintenance & Pheromone Integration** — Phases 1-8 (shipped 2026-03-19)
- ✅ **v2.1 Production Hardening** — Phases 9-16 (shipped 2026-03-24)
- ✅ **v2.2 Living Wisdom** — Phases 17-20 (shipped 2026-03-25)
- ✅ **v2.3 Per-Caste Model Routing** — Phases 21-24 (shipped 2026-03-27)
- ✅ **v2.4 Living Wisdom** — Phases 25-28 (shipped 2026-03-27)
- 📋 **v2.5 Smart Init** — Phases 29-32 (planned)

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
<summary>v2.4 Living Wisdom (Phases 25-28) — SHIPPED 2026-03-27</summary>

- [x] **Phase 25: Agent Definitions (Oracle + Architect)** — completed 2026-03-27
- [x] **Phase 26: Wisdom Pipeline Wiring** — completed 2026-03-27
- [x] **Phase 27: Deterministic Fallback + Dedup** — completed 2026-03-27
- [x] **Phase 28: Integration Validation** — completed 2026-03-27

</details>

### 📋 v2.5 Smart Init (Planned)

**Milestone Goal:** Make `/ant:init` an intelligent first step -- research the repo, generate a structured colony prompt, show it for approval, and manage the Queen file as a living colony charter.

- [x] **Phase 29: Repo Scanning Module** — SCAN-01, SCAN-02, SCAN-03 (completed 2026-03-27)
- [x] **Phase 30: Charter Management** — CHARTER-01, CHARTER-02, CHARTER-03 (completed 2026-03-27)
- [x] **Phase 31: Init.md Smart Init Rewrite** — PROMPT-01, PROMPT-02, PROMPT-03 (completed 2026-03-27)
- [x] **Phase 32: Intelligence Enhancements** — INTEL-01, INTEL-02, INTEL-03 (completed 2026-03-27)

## Phase Details

### Phase 29: Repo Scanning Module
**Goal**: System scans repo key files, directory structure, and git history before initialization, producing structured research data in under 2 seconds
**Depends on**: Nothing (foundation for all smart init features)
**Requirements**: SCAN-01, SCAN-02, SCAN-03
**Success Criteria** (what must be TRUE):
  1. Running `aether init-research` in a repo outputs JSON containing tech stack detection, directory structure summary, git history summary, and prior colony detection
  2. The scan completes in under 2 seconds on a medium-sized repo (hundreds of files)
  3. When no territory survey exists or the survey is stale, the output includes a suggestion to run `/ant:colonize` with the reason
  4. The output includes a repo complexity estimate (small/medium/large) derived from file count, directory depth, and dependency count
**Plans:** 3/3 plans complete

Plans:
- [ ] 29-01-PLAN.md — Create scan.sh module skeleton with stub functions and dispatch wiring
- [ ] 29-02-PLAN.md — Implement all six scan functions (tech stack, directory, git, survey, colonies, complexity)
- [ ] 29-03-PLAN.md — Add bash integration tests and verify 2-second performance target

---

### Phase 30: Charter Management
**Goal**: Colony charter content (intent, vision, governance, goals) populates QUEEN.md through existing v2 sections without creating new headers
**Depends on**: Nothing (independent of scan module, uses existing queen.sh patterns)
**Requirements**: CHARTER-01, CHARTER-02, CHARTER-03
**Success Criteria** (what must be TRUE):
  1. Calling the charter write function with intent and vision content writes tagged `[charter]` entries to the QUEEN.md `## User Preferences` section
  2. Calling the charter write function with governance rules and goals writes tagged `[charter]` entries to the QUEEN.md `## Codebase Patterns` section
  3. Running the charter write function on a colony that already has charter content updates entries in-place without removing existing wisdom, instincts, learnings, pheromones, or phase progress
  4. No new `## ` headers are created in QUEEN.md -- all 7+ downstream consumers continue to parse correctly after charter writes
**Plans:** 2/2 plans complete

Plans:
- [ ] 30-01-PLAN.md — Implement colony-name helper and charter-write function with dispatch wiring
- [ ] 30-02-PLAN.md — Add integration tests for charter management (first init, re-init safety, no new headers)

---

### Phase 31: Init.md Smart Init Rewrite
**Goal**: `/ant:init` generates a structured colony initialization prompt from research data, displays it for user approval, and creates colony files only after approval
**Depends on**: Phase 29 (scan data), Phase 30 (charter write functions)
**Requirements**: PROMPT-01, PROMPT-02, PROMPT-03
**Success Criteria** (what must be TRUE):
  1. When a user runs `/ant:init "build a REST API"`, the system displays a structured approval prompt containing charter, pheromone, and context sections assembled from scan research data
  2. The user can edit any section of the displayed prompt (charter text, pheromone signals, context notes) and approve or reject in a single interaction
  3. After approval, the system creates colony files with the approved charter written to QUEEN.md using existing sections (no new headers)
  4. Running `/ant:init` on an already-initialized colony updates the charter content without resetting colony state, wisdom, instincts, learnings, pheromones, or phase progress
**Plans:** 2/2 plans complete

Plans:
- [ ] 31-01-PLAN.md — Rewrite init.md with scan-assemble-approve-create flow (Claude + OpenCode mirrors)
- [ ] 31-02-PLAN.md — Integration tests for smart init flow components

---

### Phase 32: Intelligence Enhancements
**Goal**: Init prompt enriched with prior colony context, research-derived pheromone suggestions, and inferred governance from codebase patterns
**Depends on**: Phase 29 (scan data), Phase 31 (init.md rewrite must exist to inject intelligence into)
**Requirements**: INTEL-01, INTEL-02, INTEL-03
**Success Criteria** (what must be TRUE):
  1. When prior colonies exist in chambers/tunnels, the approval prompt includes a "Prior Context" section summarizing completion reports and existing QUEEN.md charter content
  2. The approval prompt includes suggested FOCUS and REDIRECT pheromone signals derived from research findings (e.g., test config present suggests FOCUS on testing, security issues suggest REDIRECT patterns)
  3. When a CONTRIBUTING.md, test config, or similar governance files are detected, the approval prompt includes inferred governance suggestions (e.g., "TDD required" from test config, contribution rules from CONTRIBUTING.md)
**Plans:** 3/3 plans complete

Plans:
- [ ] 32-01-PLAN.md — Add intelligence sub-scan functions to scan.sh (colony context, pheromone suggestions, governance inference)
- [ ] 32-02-PLAN.md — Update init.md (Claude + OpenCode) to consume intelligence data and enrich approval prompt
- [ ] 32-03-PLAN.md — Integration tests for intelligence sub-scan functions

---

## Progress

**Execution Order:**
Phases execute in numeric order: 29 -> 30 -> 31 -> 32

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
| 25. Agent Definitions (Oracle + Architect) | v2.4 | Complete | Complete | 2026-03-27 |
| 26. Wisdom Pipeline Wiring | v2.4 | Complete | Complete | 2026-03-27 |
| 27. Deterministic Fallback + Dedup | v2.4 | Complete | Complete | 2026-03-27 |
| 28. Integration Validation | v2.4 | Complete | Complete | 2026-03-27 |
| 29. Repo Scanning Module | v2.5 | Complete    | 2026-03-27 | - |
| 30. Charter Management | v2.5 | Complete    | 2026-03-27 | - |
| 31. Init.md Smart Init Rewrite | v2.5 | Complete    | 2026-03-27 | - |
| 32. Intelligence Enhancements | v2.5 | Complete    | 2026-03-27 | - |

## Coverage Matrix

| Requirement | Phase | Status |
|-------------|-------|--------|
| SCAN-01 (Structured research scan <2s) | 29 | Pending |
| SCAN-02 (Colonize suggestion when stale) | 29 | Pending |
| SCAN-03 (Repo complexity estimation) | 29 | Pending |
| PROMPT-01 (Deterministic prompt generation) | 31 | Pending |
| PROMPT-02 (Display prompt for review) | 31 | Pending |
| PROMPT-03 (User can edit before approve) | 31 | Pending |
| CHARTER-01 (First init populates User Preferences) | 30 | Pending |
| CHARTER-02 (First init populates Codebase Patterns) | 30 | Pending |
| CHARTER-03 (Re-init updates without resetting) | 30 | Pending |
| INTEL-01 (Inherit prior colony context) | 32 | Pending |
| INTEL-02 (Suggest pheromones from research) | 32 | Pending |
| INTEL-03 (Infer governance from patterns) | 32 | Pending |

**Coverage: 12/12 requirements mapped (100%)**
