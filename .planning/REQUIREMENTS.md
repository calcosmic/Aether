# Requirements: Aether v4.0 Hybrid Foundation

**Defined:** 2026-02-03
**Core Value:** Autonomous Emergence — Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v4.0 Requirements

Requirements for adding a thin shell utility layer and fixing all audit-identified issues. The system becomes hybrid: prompts for reasoning, shell for deterministic operations.

### Utility Layer

- [x] **UTIL-01**: `aether-utils.sh` exists as a single entry point with subcommand dispatch (e.g., `aether-utils pheromone-decay 0.9 3600`)
- [x] **UTIL-02**: Utility script sources `file-lock.sh` and `atomic-write.sh` for shared infrastructure
- [x] **UTIL-03**: All subcommands output JSON to stdout for prompt consumption
- [x] **UTIL-04**: All subcommands return non-zero exit code on error with JSON error message

### Pheromone Math

- [x] **PHER-01**: `pheromone-decay <strength> <elapsed_seconds> <half_life>` computes current strength using exponential decay formula
- [x] **PHER-02**: `pheromone-effective <sensitivity> <strength>` computes effective signal (sensitivity × strength)
- [x] **PHER-03**: `pheromone-batch` reads pheromones.json and outputs all signals with current computed strengths
- [x] **PHER-04**: `pheromone-cleanup` removes expired signals (strength < 0.05) from pheromones.json
- [x] **PHER-05**: `pheromone-combine <signal1_strength> <signal2_strength>` computes combination effect for conflicting signals

### State Validation

- [x] **VALID-01**: `validate-state colony` validates COLONY_STATE.json against expected schema (required fields, types)
- [x] **VALID-02**: `validate-state pheromones` validates pheromones.json (signal array structure, required fields per signal)
- [x] **VALID-03**: `validate-state errors` validates errors.json (error record structure, valid categories, severity levels)
- [x] **VALID-04**: `validate-state memory` validates memory.json (phase_learnings, decisions, patterns arrays)
- [x] **VALID-05**: `validate-state events` validates events.json (event record structure, valid types)
- [x] **VALID-06**: `validate-state all` runs all validators and reports aggregate pass/fail

### Memory Operations

- [x] **MEM-01**: `memory-token-count` approximates token count of memory.json (word count × 1.3)
- [x] **MEM-02**: `memory-compress` removes oldest entries when token count exceeds threshold (default 10000)
- [x] **MEM-03**: `memory-search <keyword>` finds memory entries matching keyword across all arrays

### Error Tracking

- [x] **ERR-01**: `error-add <category> <severity> <description>` appends error with timestamp and auto-increment ID
- [x] **ERR-02**: `error-pattern-check` detects categories with 3+ occurrences and outputs flagged patterns
- [x] **ERR-03**: `error-summary` outputs counts by category and severity
- [x] **ERR-04**: `error-dedup` removes duplicate errors (same category + description within 60 seconds)

### Audit Fixes — Critical

- [x] **FIX-01**: atomic-write.sh sources file-lock.sh so acquire_lock/release_lock are available
- [x] **FIX-02**: COLONY_STATE.json uses single canonical path for goal (`.goal`) and current_phase (`.current_phase`)
- [x] **FIX-03**: All commands read/write using canonical field paths consistently

### Audit Fixes — High Priority

- [x] **FIX-04**: Temp files use unique suffixes (PID + timestamp) to prevent race conditions
- [x] **FIX-05**: All jq operations check exit code and report errors instead of silently failing
- [x] **FIX-06**: State file backups created before critical updates (rotate last 3)
- [x] **FIX-07**: Pheromone schema uses consistent field names between creation and reads
- [x] **FIX-08**: State files validated on load (validate-state called before operations)

### Audit Fixes — Medium Priority

- [x] **FIX-09**: Worker ant status uses consistent casing (lowercase: "ready", "active", "error", "idle")
- [x] **FIX-10**: Expired pheromones cleaned up automatically (pheromone-cleanup called during reads)
- [x] **FIX-11**: Colony mode documented in init.md and ant.md help text

### Command Integration

- [ ] **INT-01**: status.md calls `aether-utils pheromone-batch` for decay bar rendering instead of Claude computing decay
- [ ] **INT-02**: build.md calls `aether-utils error-add` when logging errors
- [ ] **INT-03**: continue.md calls `aether-utils pheromone-cleanup` at phase boundaries
- [ ] **INT-04**: Worker specs document `aether-utils pheromone-effective` for computing signal response
- [ ] **INT-05**: init.md calls `aether-utils validate-state all` after state file creation

## v4.x Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Utilities

- **ADV-01**: `spawn-recommend` — Bayesian analysis of spawn history to recommend caste for capability gap
- **ADV-02**: `context-budget` — Track token usage across conversation and warn at thresholds
- **ADV-03**: `state-diff` — Show what changed in state files since last checkpoint
- **ADV-04**: `pheromone-simulate` — Project pheromone strengths forward in time

## Out of Scope

| Feature | Reason |
|---------|--------|
| Node.js utilities | Shell keeps zero external dependencies |
| Python utilities | Shell keeps zero external dependencies |
| Rewriting commands as scripts | Commands stay as prompts; scripts are called helpers |
| New commands | Utility layer accessed via Bash tool from existing commands |
| GUI/web dashboard | CLI-only, Claude Code native |
| Persistent daemon processes | Against Claude-native architecture |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| UTIL-01 | Phase 19 | Complete |
| UTIL-02 | Phase 19 | Complete |
| UTIL-03 | Phase 19 | Complete |
| UTIL-04 | Phase 19 | Complete |
| FIX-01 | Phase 19 | Complete |
| FIX-02 | Phase 19 | Complete |
| FIX-03 | Phase 19 | Complete |
| FIX-04 | Phase 19 | Complete |
| FIX-05 | Phase 19 | Complete |
| FIX-06 | Phase 19 | Complete |
| FIX-07 | Phase 19 | Complete |
| FIX-08 | Phase 19 | Complete |
| FIX-09 | Phase 19 | Complete |
| FIX-10 | Phase 19 | Complete |
| FIX-11 | Phase 19 | Complete |
| PHER-01 | Phase 20 | Complete |
| PHER-02 | Phase 20 | Complete |
| PHER-03 | Phase 20 | Complete |
| PHER-04 | Phase 20 | Complete |
| PHER-05 | Phase 20 | Complete |
| VALID-01 | Phase 20 | Complete |
| VALID-02 | Phase 20 | Complete |
| VALID-03 | Phase 20 | Complete |
| VALID-04 | Phase 20 | Complete |
| VALID-05 | Phase 20 | Complete |
| VALID-06 | Phase 20 | Complete |
| MEM-01 | Phase 20 | Complete |
| MEM-02 | Phase 20 | Complete |
| MEM-03 | Phase 20 | Complete |
| ERR-01 | Phase 20 | Complete |
| ERR-02 | Phase 20 | Complete |
| ERR-03 | Phase 20 | Complete |
| ERR-04 | Phase 20 | Complete |
| INT-01 | Phase 21 | Pending |
| INT-02 | Phase 21 | Pending |
| INT-03 | Phase 21 | Pending |
| INT-04 | Phase 21 | Pending |
| INT-05 | Phase 21 | Pending |

**Coverage:**
- v4.0 requirements: 38 total
- Mapped to phases: 38
- Unmapped: 0

---
*Requirements defined: 2026-02-03*
*Last updated: 2026-02-03 after Phase 20 completion*
