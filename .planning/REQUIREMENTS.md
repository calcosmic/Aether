# Requirements: Aether v1.11

**Defined:** 2026-04-28
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.11 Requirements

Requirements for the Aether Unification milestone. Each maps to roadmap phases.

### Self-Hosting Cleanup

Remove artifacts that exist because Aether was used to develop itself.

- [x] **CLEAN-01**: Stale `.aether/agents/` directory removed (26 files that duplicate `agents-claude/`)
- [x] **CLEAN-02**: Tracked chamber files removed from git (241 files in `.aether/chambers/`)
- [x] **CLEAN-03**: Runtime state files removed from git tracking (CONTEXT.md, CROWNED-ANTHILL.md)
- [x] **CLEAN-04**: Chambers directory added to `.aether/.gitignore` to prevent future self-hosting leaks
- [x] **CLEAN-05**: Verify `agents-claude/` is byte-identical to `.claude/agents/ant/` after cleanup

### Platform Hardening

Fix cross-platform gaps and ensure consistent behavior across Claude Code, OpenCode, and Codex CLI.

- [x] **PLAT-01**: OpenCode init.md includes shelf backlog section (v1.10 audit gap)
- [x] **PLAT-02**: OpenCode entomb.md includes shelf archive summary (v1.10 audit gap)
- [x] **PLAT-03**: Codex subagent dispatch works correctly across all agent types
- [x] **PLAT-04**: CLI flag mismatches between wrapper markdown and Go runtime are resolved
- [x] **PLAT-05**: All 50 commands produce correct output on all 3 platforms

### Smart Init Restoration

Re-port the colony charter ceremony and rich init-research from the deleted shell scripts to Go.

- [ ] **INIT-01**: Colony charter ceremony runs during `/ant-init` — scans repo, writes charter, presents for approval
- [ ] **INIT-02**: Charter approval flow with accept/revise/reject options
- [ ] **INIT-03**: Rich init-research produces tech stack analysis (languages, frameworks, build tools)
- [ ] **INIT-04**: Init-research detects directory structure patterns (monorepo, microservices, etc.)
- [ ] **INIT-05**: Init-research identifies governance files (.eslintrc, pyproject.toml, Makefile, etc.)
- [ ] **INIT-06**: Init-research generates pheromone suggestions based on detected patterns
- [ ] **INIT-07**: Init ceremony outputs formatted colony context summary

### Intelligence Features

Restore lost intelligence features from the shell-to-Go migration.

- [ ] **INTEL-01**: Suggest-analyze runs during build (Step 4.2) — automatic pattern detection across codebase
- [ ] **INTEL-02**: Suggest-analyze deduplicates against existing pheromone signals
- [ ] **INTEL-03**: Suggest-approve provides tick-to-approve UI for reviewing suggestions
- [ ] **INTEL-04**: Bayesian confidence scoring restored for wisdom pipeline (40/35/25 weighted, 60-day half-life)
- [ ] **INTEL-05**: Circuit breaker prevents cascade failure across parallel workers

### UX Improvements

Better onboarding, clearer feedback, smoother flows.

- [ ] **UX-01**: First-run experience provides clear guidance for new users
- [ ] **UX-02**: Error messages explain what happened in plain language and suggest next steps
- [ ] **UX-03**: Build and continue ceremonies provide progress feedback during long operations
- [ ] **UX-04**: Status command surfaces actionable information, not just raw state

## v2 Requirements

Deferred to future milestone. Tracked but not in current roadmap.

### Advanced Intelligence

- **INTEL-06**: State machine transitions with explicit validation and pheromone-triggered checkpoints
- **INTEL-07**: Council system for multi-perspective deliberation
- **INTEL-08**: Curation ant pipeline (8-ant orchestrated knowledge curation)
- **INTEL-09**: Consolidation pipeline (phase-end knowledge compression)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full shell script restoration | Port concepts to Go architecture, don't resurrect bash |
| Cross-colony ledger sharing | Findings contain repo-specific paths that go stale |
| Real-time agent sync | YAGNI — agents write during build/continue, not concurrently |
| Web dashboard | CLI-only for now |
| New dependencies | All features use existing Go stdlib + cobra + pkg/storage |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLEAN-01 | Phase 70 | Complete |
| CLEAN-02 | Phase 70 | Complete |
| CLEAN-03 | Phase 70 | Complete |
| CLEAN-04 | Phase 70 | Complete |
| CLEAN-05 | Phase 70 | Complete |
| PLAT-01 | Phase 71 | Complete |
| PLAT-02 | Phase 71 | Complete |
| PLAT-03 | Phase 71 | Complete |
| PLAT-04 | Phase 71 | Complete |
| PLAT-05 | Phase 71 | Complete |
| INIT-01 | Phase 72 | Pending |
| INIT-02 | Phase 72 | Pending |
| INIT-03 | Phase 73 | Pending |
| INIT-04 | Phase 73 | Pending |
| INIT-05 | Phase 73 | Pending |
| INIT-06 | Phase 73 | Pending |
| INIT-07 | Phase 73 | Pending |
| INTEL-01 | Phase 74 | Pending |
| INTEL-02 | Phase 74 | Pending |
| INTEL-03 | Phase 74 | Pending |
| INTEL-04 | Phase 75 | Pending |
| INTEL-05 | Phase 75 | Pending |
| UX-01 | Phase 76 | Pending |
| UX-02 | Phase 76 | Pending |
| UX-03 | Phase 76 | Pending |
| UX-04 | Phase 76 | Pending |

**Coverage:**
- v1 requirements: 26 total
- Mapped to phases: 26
- Unmapped: 0

---
*Requirements defined: 2026-04-28*
*Last updated: 2026-04-28 after roadmap creation*
