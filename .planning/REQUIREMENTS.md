# Requirements: Aether v4.0 Hybrid Foundation

**Defined:** 2026-02-03
**Core Value:** Autonomous Emergence — Worker Ants autonomously spawn Worker Ants; Queen provides signals not commands

## v4.0 Requirements

Requirements for adding a thin shell utility layer and fixing all audit-identified issues. The system becomes hybrid: prompts for reasoning, shell for deterministic operations.

### Utility Layer

- [ ] **UTIL-01**: `aether-utils.sh` exists as a single entry point with subcommand dispatch (e.g., `aether-utils pheromone-decay 0.9 3600`)
- [ ] **UTIL-02**: Utility script sources `file-lock.sh` and `atomic-write.sh` for shared infrastructure
- [ ] **UTIL-03**: All subcommands output JSON to stdout for prompt consumption
- [ ] **UTIL-04**: All subcommands return non-zero exit code on error with JSON error message

### Pheromone Math

- [ ] **PHER-01**: `pheromone-decay <strength> <elapsed_seconds> <half_life>` computes current strength using exponential decay formula
- [ ] **PHER-02**: `pheromone-effective <sensitivity> <strength>` computes effective signal (sensitivity × strength)
- [ ] **PHER-03**: `pheromone-batch` reads pheromones.json and outputs all signals with current computed strengths
- [ ] **PHER-04**: `pheromone-cleanup` removes expired signals (strength < 0.05) from pheromones.json
- [ ] **PHER-05**: `pheromone-combine <signal1_strength> <signal2_strength>` computes combination effect for conflicting signals

### State Validation

- [ ] **VALID-01**: `validate-state colony` validates COLONY_STATE.json against expected schema (required fields, types)
- [ ] **VALID-02**: `validate-state pheromones` validates pheromones.json (signal array structure, required fields per signal)
- [ ] **VALID-03**: `validate-state errors` validates errors.json (error record structure, valid categories, severity levels)
- [ ] **VALID-04**: `validate-state memory` validates memory.json (phase_learnings, decisions, patterns arrays)
- [ ] **VALID-05**: `validate-state events` validates events.json (event record structure, valid types)
- [ ] **VALID-06**: `validate-state all` runs all validators and reports aggregate pass/fail

### Memory Operations

- [ ] **MEM-01**: `memory-token-count` approximates token count of memory.json (word count × 1.3)
- [ ] **MEM-02**: `memory-compress` removes oldest entries when token count exceeds threshold (default 10000)
- [ ] **MEM-03**: `memory-search <keyword>` finds memory entries matching keyword across all arrays

### Error Tracking

- [ ] **ERR-01**: `error-add <category> <severity> <description>` appends error with timestamp and auto-increment ID
- [ ] **ERR-02**: `error-pattern-check` detects categories with 3+ occurrences and outputs flagged patterns
- [ ] **ERR-03**: `error-summary` outputs counts by category and severity
- [ ] **ERR-04**: `error-dedup` removes duplicate errors (same category + description within 60 seconds)

### Audit Fixes — Critical

- [ ] **FIX-01**: atomic-write.sh sources file-lock.sh so acquire_lock/release_lock are available
- [ ] **FIX-02**: COLONY_STATE.json uses single canonical path for goal (`.goal`) and current_phase (`.current_phase`)
- [ ] **FIX-03**: All commands read/write using canonical field paths consistently

### Audit Fixes — High Priority

- [ ] **FIX-04**: Temp files use unique suffixes (PID + timestamp) to prevent race conditions
- [ ] **FIX-05**: All jq operations check exit code and report errors instead of silently failing
- [ ] **FIX-06**: State file backups created before critical updates (rotate last 3)
- [ ] **FIX-07**: Pheromone schema uses consistent field names between creation and reads
- [ ] **FIX-08**: State files validated on load (validate-state called before operations)

### Audit Fixes — Medium Priority

- [ ] **FIX-09**: Worker ant status uses consistent casing (lowercase: "ready", "active", "error", "idle")
- [ ] **FIX-10**: Expired pheromones cleaned up automatically (pheromone-cleanup called during reads)
- [ ] **FIX-11**: Colony mode documented in init.md and ant.md help text

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
| UTIL-01 | Phase 19 | Pending |
| UTIL-02 | Phase 19 | Pending |
| UTIL-03 | Phase 19 | Pending |
| UTIL-04 | Phase 19 | Pending |
| FIX-01 | Phase 19 | Pending |
| FIX-02 | Phase 19 | Pending |
| FIX-03 | Phase 19 | Pending |
| FIX-04 | Phase 19 | Pending |
| FIX-05 | Phase 19 | Pending |
| FIX-06 | Phase 19 | Pending |
| FIX-07 | Phase 19 | Pending |
| FIX-08 | Phase 19 | Pending |
| FIX-09 | Phase 19 | Pending |
| FIX-10 | Phase 19 | Pending |
| FIX-11 | Phase 19 | Pending |
| PHER-01 | Phase 20 | Pending |
| PHER-02 | Phase 20 | Pending |
| PHER-03 | Phase 20 | Pending |
| PHER-04 | Phase 20 | Pending |
| PHER-05 | Phase 20 | Pending |
| VALID-01 | Phase 20 | Pending |
| VALID-02 | Phase 20 | Pending |
| VALID-03 | Phase 20 | Pending |
| VALID-04 | Phase 20 | Pending |
| VALID-05 | Phase 20 | Pending |
| VALID-06 | Phase 20 | Pending |
| MEM-01 | Phase 20 | Pending |
| MEM-02 | Phase 20 | Pending |
| MEM-03 | Phase 20 | Pending |
| ERR-01 | Phase 20 | Pending |
| ERR-02 | Phase 20 | Pending |
| ERR-03 | Phase 20 | Pending |
| ERR-04 | Phase 20 | Pending |
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
*Last updated: 2026-02-03 after initial definition*
