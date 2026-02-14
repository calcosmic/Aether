# Requirements: Aether Colony System v1.1

**Defined:** 2026-02-14
**Core Value:** Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

## v1.1 Requirements

Critical bug fixes and reliability improvements for the v1.0 infrastructure.

### Data Safety

- [x] **SAFE-01**: Git checkpoint system only captures Aether-managed files (never user data)
- [x] **SAFE-02**: Explicit allowlist for checkpoint files: `.aether/*.md`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `.opencode/agents/`, `runtime/`, `bin/cli.js`
- [x] **SAFE-03**: User data explicitly excluded: `TO-DOs.md`, `.aether/data/`, `.aether/dreams/`, `.aether/oracle/`
- [x] **SAFE-04**: Checkpoint metadata includes file hashes for integrity verification

### Build Reliability

- [ ] **BUILD-01**: Remove `run_in_background: true` from build.md worker spawns (Steps 5.1, 5.4, 5.4.2)
- [ ] **BUILD-02**: Output timing fixed — summary displays after all agent notifications complete
- [ ] **BUILD-03**: Foreground Task calls with blocking TaskOutput collection

### State Management

- [ ] **STATE-01**: Phase advancement requires fresh verification evidence (Iron Law enforcement)
- [ ] **STATE-02**: Idempotency check prevents re-building already-completed phases
- [ ] **STATE-03**: State lock acquired during phase transitions (prevents concurrent modification)
- [ ] **STATE-04**: Phase transition audit trail in COLONY_STATE.json events

### Update System

- [ ] **UPDATE-01**: Update command uses safe checkpoint before file sync
- [ ] **UPDATE-02**: Two-phase commit: backup → sync → verify → update version
- [ ] **UPDATE-03**: Automatic rollback on sync failure
- [ ] **UPDATE-04**: Stash recovery commands displayed prominently on failure
- [ ] **UPDATE-05**: Better error handling for dirty repos, network failures, partial updates

### Testing Infrastructure

- [x] **TEST-01**: package-lock.json committed for deterministic builds
- [x] **TEST-02**: Unit tests for `syncDirWithCleanup` function
- [x] **TEST-03**: Unit tests for `hashFileSync` function
- [x] **TEST-04**: Unit tests for `generateManifest` function
- [x] **TEST-05**: Mock filesystem using sinon + proxyquire
- [x] **TEST-06**: Idempotency property tests for sync operations

## v2 Requirements (Deferred)

### Future Improvements

- **NOTIFY-01**: Version-aware update notifications (non-blocking check at command start)
- **RECOVER-01**: Checkpoint recovery tracking with auto-suggest
- **METRICS-01**: Update success/failure metrics

## Out of Scope

| Feature | Reason |
|---------|--------|
| New worker castes | Out of scope for bug fix release — defer to v1.2 |
| Enhanced visualization | Feature addition, not a bug fix — defer to v1.2 |
| Real-time monitoring improvements | Feature addition, not a bug fix — defer to v1.2 |
| Cross-repo collaboration | Feature addition, not a bug fix — defer to v1.2 |
| Web UI | CLI-first approach — target v2+ |
| Cloud deployment | Local-first design — target v2+ |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SAFE-01 | 6 | Complete |
| SAFE-02 | 6 | Complete |
| SAFE-03 | 6 | Complete |
| SAFE-04 | 6 | Complete |
| TEST-01 | 6 | Complete |
| TEST-02 | 6 | Complete |
| TEST-03 | 6 | Complete |
| TEST-04 | 6 | Complete |
| TEST-05 | 6 | Complete |
| TEST-06 | 6 | Complete |
| STATE-01 | 7 | Pending |
| STATE-02 | 7 | Pending |
| STATE-03 | 7 | Pending |
| STATE-04 | 7 | Pending |
| UPDATE-01 | 7 | Pending |
| UPDATE-02 | 7 | Pending |
| UPDATE-03 | 7 | Pending |
| UPDATE-04 | 7 | Pending |
| UPDATE-05 | 7 | Pending |
| BUILD-01 | 8 | Pending |
| BUILD-02 | 8 | Pending |
| BUILD-03 | 8 | Pending |

**Coverage:**
- v1.1 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-14*
*Last updated: 2026-02-14 after roadmap creation*
