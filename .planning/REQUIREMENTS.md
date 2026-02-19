# Requirements: Aether

**Defined:** 2026-02-18
**Core Value:** Context preservation, clear workflow guidance, self-improving colony

## v1.2 Requirements

Hardening & reliability. Fix every documented bug, clean up the distribution chain, leave a bulletproof foundation.

### Distribution Chain

- [ ] **DIST-01**: update-transaction.js reads from hub/system/ not hub root (fix line 909 + verifyIntegrity + detectPartialUpdate)
- [ ] **DIST-02**: EXCLUDE_DIRS covers commands, agents, rules inside hub/system/
- [ ] **DIST-03**: Dead duplicates removed — .aether/agents/ and .aether/commands/ deleted
- [ ] **DIST-04**: caste-system.md added to sync allowlist (reaches target repos)
- [ ] **DIST-05**: Phantom planning.md removed from sync allowlists
- [ ] **DIST-06**: Old 2.x npm versions deprecated on registry

### Lock Safety

- [ ] **LOCK-01**: No lock deadlocks on jq failure in flag operations (BUG-002, BUG-005, BUG-011)
- [ ] **LOCK-02**: Trap-based lock cleanup fires on all exit paths (EXIT, TERM, INT)
- [ ] **LOCK-03**: Race condition in atomic-write backup creation fixed (BUG-003)
- [ ] **LOCK-04**: context-update uses file locking to prevent concurrent corruption (GAP-009)

### Error Handling

- [ ] **ERR-01**: json_err fallback handles error codes correctly — two-argument form works even when error-handler.sh fails to load (ISSUE-006)
- [ ] **ERR-02**: All json_err calls use E_* constants — zero hardcoded strings remaining (BUG-004, 007, 008, 009, 010, 012)
- [ ] **ERR-03**: Error code standards documented for contributors (GAP-007)
- [ ] **ERR-04**: Error path test coverage for lock and flag operations (GAP-008)

### Architecture Gaps

- [ ] **ARCH-01**: queen-init resolves templates via hub path, not hardcoded runtime/ (ISSUE-004)
- [ ] **ARCH-02**: State files validated against schema version on load (GAP-001)
- [ ] **ARCH-03**: Spawn-tree entries cleaned up on session end (GAP-002)
- [ ] **ARCH-04**: Failed Task spawns have retry logic (GAP-003)
- [ ] **ARCH-05**: queen-* commands documented (GAP-004, GAP-006)
- [ ] **ARCH-06**: queen-read validates JSON output before returning (GAP-005)
- [ ] **ARCH-07**: model-get/model-list have exec error handling (ISSUE-002)
- [ ] **ARCH-08**: Help command lists all available commands including queen-* (ISSUE-003)
- [ ] **ARCH-09**: Feature detection doesn't race with error handler loading (ISSUE-007)
- [ ] **ARCH-10**: Temp files cleaned up via exit trap (cleanup_temp_files wired to trap)

## v1.3 Requirements (Deferred)

Architecture simplification + new features. Requires design spike first.

### Hub Architecture
- **HUB-01**: Remove runtime/ staging — package .aether/ directly via package.json files
- **HUB-02**: Hub has clean directory structure (system/, commands/, agents/, wisdom/, chambers/, preferences/)
- **HUB-03**: Single allowlist in package.json files field (sync-to-runtime.sh eliminated)

### Colony Architecture
- **COL-01**: Per-repo .aether/ only contains local state (no system file copies)
- **COL-02**: Slash commands reference global system files via shim pattern
- **COL-03**: Colony init creates only local state directories

### Queen System
- **QUEEN-01**: Global queen.md + colony queen.md stack together in worker context
- **QUEEN-02**: Queen files use CLAUDE.md-compatible patterns

### Wisdom Flow
- **WISD-01**: Local pheromones can be promoted to eternal wisdom at hub
- **WISD-02**: Eternal wisdom available to all colonies automatically
- **WISD-03**: Chambers archive to hub (browsable from any repo)

### Platform Alignment
- **PLAT-01**: OpenCode mirror has all v1.1 visual features backported

## Out of Scope

| Feature | Reason |
|---------|--------|
| Model-per-caste routing verification | Config exists but unverified; not a bug, just unproven |
| ANSI color codes | Renders as garbage in Claude Code |
| Animated spinners | Not supported in Claude Code chat |
| Multi-ant parallel execution | Needs design discussion first |
| YAML command generator | Working manually, not broken |
| Pheromone evolution | Feature exists but unused — v2 |
| Worker quality scores | Future research |
| Colony sleep / self-driving | Future vision |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| ERR-01 | Phase 14 | Pending |
| ARCH-01 | Phase 14 | Pending |
| DIST-01 | Phase 15 | Pending |
| DIST-02 | Phase 15 | Pending |
| DIST-03 | Phase 15 | Pending |
| DIST-04 | Phase 15 | Pending |
| DIST-05 | Phase 15 | Pending |
| DIST-06 | Phase 15 | Pending |
| LOCK-01 | Phase 16 | Pending |
| LOCK-02 | Phase 16 | Pending |
| LOCK-03 | Phase 16 | Pending |
| LOCK-04 | Phase 16 | Pending |
| ERR-02 | Phase 17, Phase 19 (gap closure) | Pending |
| ERR-03 | Phase 17, Phase 19 (gap closure) | Pending |
| ERR-04 | Phase 17 | Pending |
| ARCH-02 | Phase 18 | Pending |
| ARCH-03 | Phase 18 | Pending |
| ARCH-04 | Phase 18 | Pending |
| ARCH-05 | Phase 18 | Pending |
| ARCH-06 | Phase 18 | Pending |
| ARCH-07 | Phase 18 | Pending |
| ARCH-08 | Phase 18 | Pending |
| ARCH-09 | Phase 18 | Pending |
| ARCH-10 | Phase 18 | Pending |

**Coverage:**
- v1.2 requirements: 24 total
- Mapped to phases: 24
- Unmapped: 0

---
*Requirements defined: 2026-02-18*
*Last updated: 2026-02-19 — ERR-02/ERR-03 gap closure via Phase 19*
