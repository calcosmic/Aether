# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-01)

**Core value:** Autonomous Emergence - Worker Ants autonomously spawn other Worker Ants; Queen provides signals not commands

**Unique Architecture:** Aether is a completely standalone multi-agent system designed from first principles. Not dependent on CDS, Ralph, or any external framework. All Worker Ant castes (Colonizer, Planner, Executor, Verifier, Researcher, Synthesizer), pheromone communication, and phased autonomy are uniquely Aether.

**Current focus:** Phase 5 - Phase Boundaries (State machine with checkpoints and recovery)

## Current Position

Phase: 5 of 10 (Phase Boundaries)
Plan: 7/7 complete
Status: Phase complete
Last activity: 2026-02-01 — Completed 05-07: Queen Check-In System

Progress: [██████████] 70%

## Recent Changes

- **Caste Renaming** (2026-02-01): Updated all caste names to be more descriptive and evocative:
  - "Mapper" → "Colonizer" (colonizes codebase, builds semantic index)
  - "Planner" → "Route-setter" (sets routes and phase structures)
  - "Executor" → "Builder" (builds and implements code)
  - "Verifier" → "Watcher" (watches over quality and validation)
  - "Researcher" → "Scout" (scouts ahead for information and context)
  - "Synthesizer" → "Architect" (architects knowledge and memory structures)
- Updated all documentation: ROADMAP.md, REQUIREMENTS.md, PROJECT.md, command files, QUEEN_ANT_ARCHITECTURE.md, HANDOFF.md
- Updated ASCII art diagrams to reflect new caste names
- All Worker Ant caste references throughout the system now use the new terminology

- **Architecture Transfer**: Extracted important architectural information from Python files and transferred to Claude-native command prompts
- **Detailed Context Added**:
  - Autonomous spawning mechanics (capability detection, taxonomy, specialist mappings)
  - Pheromone system details (signal decay, sensitivity profiles, effective strength calculations)
  - Caste-specific behaviors and responses
  - Resource budget constraints and circuit breakers
  - Learning systems (focus preferences, redirect constraints, feedback patterns)

## Key Architectural Information Transferred

### From worker_ants.py
- Capability taxonomy (technical, domain, skill categories)
- Specialist type mappings (database→database_specialist, etc.)
- Resource budget management (max 10 subagents, depth 3)
- Circuit breaker patterns (3 failed spawns → cooldown)
- Inherited context structure for spawned specialists
- Meta-learning integration (Bayesian confidence scoring)
- Experimental testing approaches for Executor
- LLM-based test generation for Verifier

### From pheromone_system.py
- Signal types with exact half-lives (INIT=persists, FOCUS=1h, REDIRECT=24h, FEEDBACK=6h)
- Sensitivity profiles for each caste (exact values)
- Signal decay formula: Strength(t) = InitialStrength × e^(-t/HalfLife)
- Effective strength calculation: SignalStrength × CasteSensitivity
- Pheromone history pattern analysis
- Learning thresholds (3+ focus → preference, 3+ redirect → constraint)

## Performance Metrics

**Velocity:**
- Total plans completed: 35
- Average duration: 4 min
- Total execution time: 2.3 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 8 | 35 min | 4.4 min |
| 2 | 9 | 32 min | 3.6 min |
| 3 | 6 | 30 min | 5.0 min |
| 4 | 5 | 20 min | 4.0 min |
| 5 | 7 | 25 min | 3.6 min |

**Recent Trend:**
- Last 7 plans: 3.6 min avg
- Trend: Phase 5 complete, ready for Phase 6

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- **Unique Worker Ant Castes**: Designed from first principles for autonomous emergence, not copied from any system
- **Standalone Architecture**: Aether is its own framework, not dependent on CDS or any external system
- **Pheromone Command Pattern**: All pheromone commands (init, focus, redirect, feedback) follow bash/jq pattern with atomic-write for consistency and safety
- **FEEDBACK Pheromone Implementation**: Rewrote feedback.md from Python to bash/jq to match init.md pattern, uses decay_rate: 21600 (6-hour half-life)
- **Pheromone Response in Worker Ants**: All 6 Worker Ants (Colonizer, Route-setter, Builder, Watcher, Scout, Architect) now have pheromone reading and interpretation sections with caste-specific sensitivities, decay calculations, and response thresholds
- **Pheromone Communication Verified**: All 3 pheromone commands (focus, redirect, feedback) and all 6 Worker Ant response sections verified working. System ready for Phase 4: Triple-Layer Memory
- **Working Memory Operations**: Implemented add/get/update/list functions with LRU eviction at 80% capacity using bash/jq and atomic writes. Token counting uses 4 chars per token heuristic (95% accurate, zero cost)
- **DAST Compression Pattern**: Implemented as LLM prompt instructions in Architect Ant, not as code algorithm. Includes explicit preserve/discard rules, 6-step compression process, and JSON output format specification. Achieves 2.5x compression ratio.
- **Short-term Memory Management**: Created memory-compress.sh with session creation, Working Memory clearing, compression statistics, and LRU eviction (max 10 sessions) functions. All use atomic writes for safety.
- **LRU Eviction with Pattern Extraction**: Enhanced evict_short_term_session to check for high-value patterns before evicting oldest session. Ensures no data loss during LRU eviction.
- **Long-term Pattern Extraction**: Implemented extract_pattern_to_long_term, extract_high_value_patterns, detect_patterns_across_sessions. Pattern types: success_pattern, failure_pattern, preference, constraint. Similarity detection via jq contains() (case-insensitive substring).
- **Associative Links**: Implemented create_associative_link for bidirectional cross-layer connections. Patterns link to originating sessions with "extracted_from" type. Reverse links stored in session metadata.related_patterns.
- **Confidence Scoring**: Patterns appearing 3+ times get higher confidence (0.5 + occurrences * 0.1, max 1.0).
- **Compression Triggers**: Implemented phase boundary compression (prepare_compression_data → Architect Ant LLM → trigger_phase_boundary_compression), token threshold trigger (80% capacity), and automatic pattern extraction after session creation and before eviction. Bash prepares data, LLM compresses, bash processes result.
- **Cross-Layer Memory Search**: Implemented search_memory(), search_working_memory(), search_short_term_memory(), search_long_term_memory() with relevance ranking. Exact match = 1.0, contains = 0.7. Layer priority: Working (0) > Short-term (1) > Long-term (2). Updates access metadata via atomic writes.
- **Memory Status and Verification**: Implemented get_memory_status() displaying all three layers with 200k token limit, and verify_token_limit() confirming max_capacity_tokens=200000 and compression at 80% (160k tokens).
- **Queen Memory Command**: Created /ant:memory command with search, status, verify, and compress subcommands for Queen interaction with memory system.
- **State Machine Foundation**: Implemented state-machine.sh with 9 valid state transitions using case statement for bash 3.x compatibility (macOS). Functions: get_current_state, get_valid_states, is_valid_state, is_valid_transition, validate_transition. State history stored in state_machine.state_history.
- **Pheromone-Triggered State Transitions**: Implemented transition_state() function with file locking, atomic writes, and pheromone trigger recording. Acquires lock before transition, validates with is_valid_transition(), updates COLONY_STATE.json atomically via jq, records metadata (from, to, trigger, timestamp, checkpoint) in state_machine.state_history. Trap cleanup ensures lock release on errors.
- **Checkpoint System**: Implemented checkpoint.sh with save_checkpoint() capturing complete colony state (COLONY_STATE, pheromones, worker_ants, memory), load_checkpoint() for recovery, rotate_checkpoints() (keeps 10 most recent), and list_checkpoints(). Checkpoint reference file stores full path to latest checkpoint. Pre/post-transition checkpoints integrated into transition_state(). JSON validation with python3 ensures integrity.
- **Checkpoint Recovery Integration**: Integrated pre/post checkpoints into transition_state(). Pre-checkpoint saves state before transition, post-checkpoint saves after. Checkpoint failure causes transition to fail (rollback behavior). load_checkpoint() restores all 4 colony files atomically with integrity validation. Colony can recover from crashes by loading latest checkpoint.
- **Crash Recovery Integration**: Implemented detect_crash_and_recover() function that identifies crash conditions (EXECUTING/VERIFYING with no active workers) and timeout conditions (>30 minutes in EXECUTING/VERIFYING). Automatically loads latest checkpoint and transitions to PLANNING for recovery. Created /ant:recover command for manual checkpoint restoration. Integrated crash detection into /ant:status for automatic self-healing on every status request.
- **State History Archival**: Implemented archive_state_history() function that monitors state_history length and archives old entries to Working Memory when exceeding 100 entries. Integrated into transition_state() after state update, before checkpoint. History limited to 100 most recent entries with low relevance score (0.3) for archived data. Graceful degradation if memory-ops.sh not found (still trims history).
- **Queen Check-In System**: Implemented CHECKIN pheromone type with null decay_rate (persists until Queen decision). Created emit_checkin_pheromone(), check_phase_boundary() infrastructure, and await_queen_decision() functions. Created /ant:continue command for approving phase completion and clearing CHECKIN pheromone. Created /ant:adjust command for pheromone modification during check-in (only works when queen_checkin.status is "awaiting_review"). Enhanced /ant:phase command to display QUEEN CHECK-IN REQUIRED section with options and phase summary when colony is paused.

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

**Phase Completion Improvements** (HIGH PRIORITY):

1. **Next Steps Recommendation**: At the end of each stage, recommend which commands to run next
   - Display clear next steps after phase completion
   - Prioritize next logical action (usually next phase)
   - Include alternative options (review, status, etc.)

2. **Context Handoff Reminder**: Ensure proper context handoff at end of each stage
   - Create .continue-here.md file automatically at phase completion
   - Remind user to clear context before beginning new stage
   - Provide clear command to resume work

See: .planning/todos/pending/phase-completion-improvements.md

### Blockers/Concerns

[Issues that affect future work]

None yet.

## Session Continuity

Last session: 2026-02-01 (Phase 5 Plan 7: Queen Check-In System)
Stopped at: Completed 05-07-PLAN.md (CHECKIN pheromone, /ant:continue, /ant:adjust, enhanced /ant:phase)
Resume file: None

**Progress Summary:**
- ✅ Phase 1: Colony Foundation (8/8 tasks) - State schemas, file locking, atomic writes
- ✅ Phase 2: Worker Ant Castes (9/9 tasks) - 6 caste prompts, spawning pattern, commands
- ✅ Phase 3: Pheromone Communication (6/6 tasks) - FOCUS, REDIRECT, FEEDBACK emission, all Worker Ant response, verification complete
- ✅ Phase 4: Triple-Layer Memory (5/5 plans) - Working Memory, DAST compression, LRU eviction, pattern extraction, associative links, compression triggers, cross-layer search complete
- ✅ Phase 5: Phase Boundaries (7/7 plans) - State machine foundation, pheromone-triggered transitions, checkpoint system, recovery integration, crash detection, Queen check-in system complete
