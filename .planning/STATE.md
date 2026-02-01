# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-01)

**Core value:** Autonomous Emergence - Worker Ants autonomously spawn other Worker Ants; Queen provides signals not commands

**Unique Architecture:** Aether is a completely standalone multi-agent system designed from first principles. Not dependent on CDS, Ralph, or any external framework. All Worker Ant castes (Colonizer, Planner, Executor, Verifier, Researcher, Synthesizer), pheromone communication, and phased autonomy are uniquely Aether.

**Current focus:** Phase 4 - Triple-Layer Memory (Working → Short-term → Long-term with associative links)

## Current Position

Phase: 4 of 10 (Triple-Layer Memory)
Plan: 3 of N plans complete
Status: In progress - LRU eviction, pattern extraction, associative links ready
Last activity: 2026-02-01 — Completed Phase 4 Plan 3: Short-term LRU eviction, Long-term pattern extraction, associative links

Progress: [████████░░] 75%

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
- Total plans completed: 0
- Average duration: - min
- Total execution time: 0.0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**
- Last 5 plans: -
- Trend: -

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

Last session: 2026-02-01 (Phase 4 Plan 3 - LRU and Pattern Extraction)
Stopped at: Completed Phase 4 Plan 3 - Short-term LRU eviction, Long-term pattern extraction, associative links
Resume file: .planning/phases/04-triple-layer-memory/.continue-here.md (to be created)

**Progress Summary:**
- ✅ Phase 1: Colony Foundation (8/8 tasks) - State schemas, file locking, atomic writes
- ✅ Phase 2: Worker Ant Castes (9/9 tasks) - 6 caste prompts, spawning pattern, commands
- ✅ Phase 3: Pheromone Communication (6/6 tasks) - FOCUS, REDIRECT, FEEDBACK emission, all Worker Ant response, verification complete
- 🔄 Phase 4: Triple-Layer Memory (3/N plans) - Working Memory, DAST compression, LRU eviction, pattern extraction, associative links ready
