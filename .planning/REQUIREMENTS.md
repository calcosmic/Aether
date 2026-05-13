# Requirements: Aether v1.17

**Defined:** 2026-05-13
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.17 Requirements: Classic Restoration

### TS Host Foundation (TS)

- [ ] **TS-01:** TS host dispatches real platform workers (Claude Code, OpenCode, Codex) instead of simulated delays
- [ ] **TS-02:** TS host executes workers within a wave in parallel (concurrent Agent tool spawns)
- [ ] **TS-03:** TS host handles worker errors with retry logic, timeout, and graceful fallback
- [ ] **TS-04:** Event bridge reads Go ceremony events from JSONL stream and tails live events
- [ ] **TS-05:** Node engine bumped to >=20 for chokidar/log-update compatibility
- [ ] **TS-06:** Boundary contract enforced — TS host never writes to `.aether/data/`

### Ceremony & Visuals (CER)

- [ ] **CER-01:** Ceremony banners and art restored to command wrappers (editable markdown, not compiled Go)
- [ ] **CER-02:** Shared ceremony config in YAML (caste emoji/color/label maps, naming conventions)
- [ ] **CER-03:** Go ceremony rendering code replaced by event emission (Go emits, wrappers render from templates)
- [ ] **CER-04:** Crowned Anthill seal ASCII art in editable template
- [ ] **CER-05:** Worker spawn notifications with caste identity frames
- [ ] **CER-06:** Build summary and closeout rituals with template frames
- [ ] **CER-07:** Seal ceremony with Sage, Chronicler, wisdom review, and commit suggestion steps

### Swarm Dashboard (SW)

- [ ] **SW-01:** Live terminal dashboard with animated spinners per active worker
- [ ] **SW-02:** Per-ant progress bars with excavation status phrases
- [ ] **SW-03:** Tool usage counters per worker (reads/writes/fetches/commands)
- [ ] **SW-04:** Chamber activity map showing which project areas have active workers
- [ ] **SW-05:** Elapsed time and token consumption per worker
- [ ] **SW-06:** Auto-refresh via chokidar watching JSONL event file

### Queen Orchestration (ORC)

- [ ] **ORC-01:** Queen selects workflow patterns (SPBV, Investigate-Fix, Refactor, Compliance, Documentation Sprint) based on phase name/content
- [ ] **ORC-02:** Builder-Probe Lock restored — builders return `code_written`, only Probe upgrades to `completed`
- [ ] **ORC-03:** Tiered escalation chain (worker retry → parent reassignment → Queen reassignment → user escalation)
- [ ] **ORC-04:** Intra-build midden threshold checks with auto-REDIRECT pheromone emission
- [ ] **ORC-05:** Phase mode awareness (discovery/prototype/production/maintenance) mapping to verification depth
- [ ] **ORC-06:** Ambassador conditional spawn for integration tasks

### Oracle Enhancement (ORA)

- [ ] **ORA-01:** Phase-aware prompt directives (survey/investigate/synthesize/verify) injected into worker briefs
- [ ] **ORA-02:** Diminishing returns detection with novelty delta tracking
- [ ] **ORA-03:** Template-specific synthesis sections (tech-eval, architecture-review, bug-investigation)

### Parity & Verification (PAR)

- [ ] **PAR-01:** Golden workflow tests comparing ceremony and behavior against v5.4 Classic baseline
- [ ] **PAR-02:** Ceremony snapshot tests (banners, spawn plans, seal rituals match templates)
- [ ] **PAR-03:** Cross-platform smoke tests (Claude Code, OpenCode, Codex)
- [ ] **PAR-04:** State safety tests — all writes go through Go finalizers

## Out of Scope

| Feature | Reason |
|---------|--------|
| Real-time web dashboard | Out of scope per PROJECT.md |
| Cross-colony ledger sharing | Findings go stale across repos |
| Durable execution engine for Oracle | Only justified if Oracle becomes long-running and resumable |
| Full TUI framework adoption | Ink/Blessed are too heavy; custom composite is sufficient |
| Workflow graph engine (LangGraph-style) | Aether's patterns are LLM-driven, not deterministic graphs |
| Multi-agent Oracle crew | Oracle is single-agent with phase transitions |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| TS-01 | Phase 114 | Pending |
| TS-02 | Phase 114 | Pending |
| TS-03 | Phase 114 | Pending |
| TS-04 | Phase 112 | Pending |
| TS-05 | Phase 112 | Pending |
| TS-06 | Phase 112 | Pending |
| CER-01 | Phase 113 | Pending |
| CER-02 | Phase 112 | Pending |
| CER-03 | Phase 113 | Pending |
| CER-04 | Phase 113 | Pending |
| CER-05 | Phase 113 | Pending |
| CER-06 | Phase 113 | Pending |
| CER-07 | Phase 118 | Pending |
| SW-01 | Phase 115 | Complete |
| SW-02 | Phase 115 | Complete |
| SW-03 | Phase 115 | Pending |
| SW-04 | Phase 115 | Complete |
| SW-05 | Phase 115 | Pending |
| SW-06 | Phase 115 | Pending |
| ORC-01 | Phase 116 | Complete |
| ORC-02 | Phase 116 | Complete |
| ORC-03 | Phase 116 | Complete |
| ORC-04 | Phase 116 | Complete |
| ORC-05 | Phase 116 | Complete |
| ORC-06 | Phase 116 | Complete |
| ORA-01 | Phase 117 | Pending |
| ORA-02 | Phase 117 | Pending |
| ORA-03 | Phase 117 | Pending |
| PAR-01 | Phase 118 | Pending |
| PAR-02 | Phase 118 | Pending |
| PAR-03 | Phase 118 | Pending |
| PAR-04 | Phase 118 | Pending |

---

## Prior Requirements

See `.planning/MILESTONES.md` for validated requirements from v1.0 through v1.16.
