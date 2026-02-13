# Roadmap: Aether Colony System v1.0

**Milestone:** Infrastructure & Core Reliability
**Created:** 2026-02-13
**Requirements:** 16 v1 requirements mapped across 5 phases

---

## Phase 1: Infrastructure Hardening

**Goal:** Harden core infrastructure to prevent race conditions, data loss, and update failures

**Success Criteria:**
- [x] All state file operations use file locking
- [x] All JSON state updates use atomic writes
- [x] Git checkpoints only stash Aether-managed directories
- [x] Update command compares versions before syncing
- [x] No data loss during concurrent state access
- [x] Signatures.json template exists and works
- [x] Hash comparison prevents unnecessary file writes

**Status:** ✓ Complete (2026-02-13)

**Requirements Covered:**
| Requirement | Description |
|-------------|-------------|
| INFRA-01 | File locking enforced on all state file operations |
| INFRA-02 | Atomic writes use temp file + mv pattern |
| INFRA-03 | Git checkpoints only stash Aether-managed directories |
| INFRA-04 | Update command tracks version and compares before syncing |

**Estimated Duration:** 1-2 sessions
**Dependencies:** None

---

## Phase 2: Testing Foundation

**Goal:** Add comprehensive test coverage for critical paths

**Success Criteria:**
- [x] AVA test framework integrated
- [x] Unit tests for Node.js utilities
- [x] Bash integration tests for aether-utils.sh
- [x] Existing tests pass (sync, user-modification, namespace)
- [x] Oracle bugs fixed (duplicate keys, timestamp ordering)
- [x] Tests verify Oracle bugs are fixed

**Requirements Covered:**
| Requirement | Description |
|-------------|-------------|
| TEST-01 | AVA unit test framework for Node.js utilities |
| TEST-02 | Bash integration tests for aether-utils.sh commands |
| TEST-03 | Existing tests continue to pass |

**Plans:**
- [x] 02-01-PLAN.md — Set up AVA test framework and create unit tests for COLONY_STATE.json validation
- [x] 02-02-PLAN.md — Create Bash integration tests for aether-utils.sh subcommands
- [x] 02-03-PLAN.md — Fix Oracle bugs and add regression tests

**Wave Structure:**
- Wave 1: 02-01, 02-02 (parallel - independent test setup)
- Wave 2: 02-03 (depends on test infrastructure from 02-01)

**Estimated Duration:** 2-3 sessions
**Dependencies:** Phase 1 complete

**Status:** ✓ Complete (2026-02-13)

---

## Phase 3: Error Handling & Recovery

**Goal:** Implement centralized error handling with graceful degradation

**Success Criteria:**
- [ ] Centralized error handler in cli.js with structured errors
- [ ] Error handler in aether-utils.sh provides consistent error JSON
- [ ] Graceful degradation when optional features fail
- [ ] Error logging to activity.log
- [ ] User-friendly error messages
- [ ] Recovery suggestions in error output

**Requirements Covered:**
| Requirement | Description |
|-------------|-------------|
| ERROR-01 | Centralized error handler in cli.js |
| ERROR-02 | Error handler in aether-utils.sh |
| ERROR-03 | Graceful degradation on optional feature failures |

**Estimated Duration:** 1-2 sessions
**Dependencies:** Phase 2 complete

---

## Phase 4: CLI Improvements

**Goal:** Migrate to commander.js with better UX

**Success Criteria:**
- [ ] Argument parsing migrated to commander.js
- [ ] Colored output using picocolors
- [ ] Auto-help works for all commands
- [ ] Subcommand structure implemented
- [ ] Help text clarifies slash commands vs CLI commands
- [ ] Backward compatibility maintained

**Requirements Covered:**
| Requirement | Description |
|-------------|-------------|
| CLI-01 | Migrate argument parsing to commander.js |
| CLI-02 | Add colored output using picocolors |
| CLI-03 | Auto-help for all commands works correctly |

**Estimated Duration:** 2 sessions
**Dependencies:** Phase 3 complete

---

## Phase 5: State & Context Restoration

**Goal:** Ensure reliable cross-session memory and context

**Success Criteria:**
- [ ] Colony state loads on every command invocation
- [ ] Context restoration works after session pause/resume
- [ ] Spawn tree persists correctly across sessions
- [ ] Event timestamps in chronological order
- [ ] No duplicate keys in JSON structures
- [ ] State validation on load

**Requirements Covered:**
| Requirement | Description |
|-------------|-------------|
| STATE-01 | Colony state loads on every command invocation |
| STATE-02 | Context restoration works after session pause/resume |
| STATE-03 | Spawn tree persists correctly across sessions |

**Estimated Duration:** 1-2 sessions
**Dependencies:** Phase 4 complete

---

## Progress Tracking

| Phase | Status | Plans | Progress |
|-------|--------|-------|----------|
| 1 | ✓ | 3/3 | 100% |
| 2 | ✓ | 3/3 | 100% |
| 3 | ○ | 0/1 | 0% |
| 4 | ○ | 0/1 | 0% |
| 5 | ○ | 0/1 | 0% |

**Overall:** 7/16 requirements complete (INFRA-01 through INFRA-04, TEST-01 through TEST-03)

---

## Oracle Bug Fixes (Priority)

### Phase 1 Complete:
| Issue | Severity | Fix Location | Status |
|-------|----------|--------------|--------|
| Missing signatures.json | MEDIUM | runtime/data/signatures.json | ✓ Fixed |
| syncSystemFilesWithCleanup no hash compare | LOW | bin/cli.js:279-317 | ✓ Fixed |
| CLI help unclear on /ant:init | LOW | bin/cli.js:710,614 | ✓ Fixed |

### Phase 2 Complete:
| Issue | Severity | Fix Location | Status |
|-------|----------|--------------|--------|
| Duplicate "status" key in COLONY_STATE.json | LOW | .aether/data/COLONY_STATE.json | ✓ Verified clean |
| Event timestamps out of order | LOW | .aether/data/COLONY_STATE.json | ✓ Verified clean |

---

## Out of Scope for v1.0

| Feature | Reason | Target |
|---------|--------|--------|
| Web UI | CLI-first approach | v2+ |
| Cloud deployment | Local-first design | v2+ |
| OAuth/multi-user auth | Single developer focus | v2+ |
| Mobile support | Desktop CLI tool | v2+ |
| Real-time monitoring | Complexity, not core | v2+ |

---

*Last updated: 2026-02-13*
