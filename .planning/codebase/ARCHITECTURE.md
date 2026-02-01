# Architecture

**Analysis Date:** 2025-02-01

## Pattern Overview

**Overall:** Queen Ant Colony with Phased Autonomy and State Machine Orchestration

**Key Characteristics:**
- Multi-agent system where Worker Ants self-organize within phases
- User (Queen) provides high-level intention via pheromone signals, not commands
- Autonomous spawning - agents detect capability gaps and spawn specialists
- State machine orchestration with checkpointing for production reliability
- Triple-layer memory system (Working → Short-term → Long-term)
- Pheromone-based communication with semantic understanding
- Pure emergence within structured phases

## Layers

**User Interface Layer (Queen):**
- Purpose: User provides intention and receives status updates
- Location: `.aether/queen_ant_system.py` (QueenAntSystem), `.aether/cli.py`
- Contains: Command interface, status methods, feedback mechanisms
- Depends on: Phase Engine, Pheromone Layer
- Used by: Claude Code commands, CLI, REPL

**Pheromone Communication Layer:**
- Purpose: Signal-based communication (not message passing)
- Location: `.aether/pheromone_system.py`, `.aether/semantic_layer.py`
- Contains: Pheromone types (INIT, FOCUS, REDIRECT, FEEDBACK), signal emission, semantic understanding
- Depends on: sentence-transformers (optional for semantic features)
- Used by: All Worker Ants, Queen, Phase Engine

**State Machine Orchestration Layer:**
- Purpose: Production-grade state transitions with checkpointing
- Location: `.aether/state_machine.py`
- Contains: AetherStateMachine, state transitions, checkpointing, recovery
- Depends on: Colony, Worker Ants
- Used by: Phase Engine (when enabled)

**Phase Execution Layer:**
- Purpose: Manage project phases with emergence inside, checkpoints at boundaries
- Location: `.aether/phase_engine.py`
- Contains: PhaseEngine, Phase dataclass, task management, Queen check-ins
- Depends on: Colony, Pheromone Layer, State Machine (optional)
- Used by: Queen Ant System

**Worker Ant Layer:**
- Purpose: Six specialist castes that execute tasks and spawn subagents
- Location: `.aether/worker_ants.py`
- Contains: MapperAnt, PlannerAnt, ExecutorAnt, VerifierAnt, ResearcherAnt, SynthesizerAnt, Colony
- Depends on: Pheromone Layer, Error Ledger, Memory Layer, Meta-Learner
- Used by: Phase Engine, State Machine

**Memory Layer:**
- Purpose: Triple-layer hierarchical memory (human-like cognition)
- Location: `.aether/memory/triple_layer_memory.py`, `.aether/memory/working_memory.py`, `.aether/memory/short_term_memory.py`, `.aether/memory/long_term_memory.py`
- Contains: WorkingMemory (200k tokens), ShortTermMemory (10 sessions), LongTermMemory (persistent)
- Depends on: None (standalone)
- Used by: All Worker Ants, Queen

**Learning Layer:**
- Purpose: Meta-learning and outcome tracking for autonomous spawning improvement
- Location: `.aether/memory/meta_learner.py`, `.aether/memory/outcome_tracker.py`
- Contains: MetaLearner (spawn recommendations), OutcomeTracker (testing outcomes)
- Depends on: Memory Layer
- Used by: Worker Ants (especially Executor, Verifier)

**Error Prevention Layer:**
- Purpose: Track errors, detect patterns, prevent recurring issues
- Location: `.aether/error_prevention.py`
- Contains: ErrorLedger, ErrorRecord, ErrorPattern
- Depends on: None
- Used by: All Worker Ants

## Data Flow

**Project Initialization Flow:**

1. User emits INIT pheromone with goal
2. QueenAntSystem.init() triggers PhaseEngine.initiate_project()
3. Pheromone emitted via PheromoneLayer
4. MapperAnt explores codebase (responds to INIT pheromone)
5. PlannerAnt creates phase structure (responds to INIT pheromone)
6. Queen reviews plan at phase boundary

**Phase Execution Flow (with State Machine):**

1. PhaseEngine.execute_phase() starts
2. StateMachinePhaseEngine wraps execution with state transitions:
   - IDLE → ANALYZING (Mapper explores)
   - ANALYZING → PLANNING (tasks defined)
   - PLANNING → EXECUTING (colony works with emergence)
   - EXECUTING → VERIFYING (Verifier checks)
   - VERIFYING → COMPLETED (or back to EXECUTING if verification fails)
3. Checkpoint saved before/after each transition
4. At phase boundary: memory compressed, Queen check-in

**Autonomous Spawning Flow:**

1. WorkerAnt receives Task
2. WorkerAnt.delegate_or_handle() decides
3. detect_capability_gap() analyzes task requirements vs own capabilities
4. Meta-learner recommends specialist type (if available)
5. spawn_specialist_autonomously() creates Subagent with InheritedContext
6. Subagent executes task
7. record_task_outcome() records result for learning

**Pheromone Communication Flow:**

1. Queen/user emits pheromone signal via PheromoneCommands
2. PheromoneLayer stores signal with strength and decay
3. Semantic layer (optional) adds vector embedding
4. WorkerAnts detect_pheromones() based on SensitivityProfile
5. Effective strength = signal strength × ant sensitivity
6. If above threshold, ant responds via respond_to_signal()

**Memory Compression Flow:**

1. Phase completes
2. PhaseEngine._compress_phase_memory() triggered
3. WorkingMemory flushed (all items extracted)
4. SessionSummary created with DAST compression (2.5x ratio)
5. Compressed session stored in ShortTermMemory (max 10 sessions)
6. Patterns extracted for LongTermMemory
7. LRU eviction if short-term exceeds limit

## Key Abstractions

**PheromoneType (Enum):**
- Purpose: Types of signals from Queen
- Examples: INIT, FOCUS, REDIRECT, FEEDBACK
- Pattern: Enum with strength and half-life decay

**WorkerAnt (Base Class):**
- Purpose: Base for all six castes
- Examples: MapperAnt, PlannerAnt, ExecutorAnt, VerifierAnt, ResearcherAnt, SynthesizerAnt
- Pattern: Class with caste, capabilities, sensitivity, spawns attributes; override respond_to_signal()

**SensitivityProfile:**
- Purpose: Determines how each ant caste responds to pheromones
- Examples: SENSITIVITY_PROFILES dict in pheromone_system.py
- Pattern: Dataclass with init/focus/redirect/feedback floats; effective_strength = signal × sensitivity

**Subagent (Autonomous Spawning):**
- Purpose: Dynamically spawned specialist with inherited context
- Examples: database_specialist, frontend_specialist, test_specialist
- Pattern: Dataclass with name, purpose, parent, inherited_context, capabilities, depth

**Capability Gap Detection:**
- Purpose: Determine when to spawn specialists
- Examples: CAPABILITY_TAXONOMY, SPECIALIST_MAPPING in worker_ants.py
- Pattern: analyze_task_requirements() returns set of required capabilities; gaps = required - own_capabilities

**Phase (Dataclass):**
- Purpose: Structured phase with tasks, milestones, Queen feedback
- Examples: Created by PlannerAnt.decompose_goal()
- Pattern: id, name, description, tasks (list of Task), status, milestones, queen_approval

**SystemState (Enum - State Machine):**
- Purpose: Explicit orchestration states
- Examples: IDLE, ANALYZING, PLANNING, EXECUTING, VERIFYING, COMPLETED, FAILED
- Pattern: State transitions triggered by Events; checkpointed before/after each transition

**ContextItem (Memory):**
- Purpose: Item in working memory
- Examples: All additions to WorkingMemory
- Pattern: item_id, content, metadata, timestamp, access_count, tokens

## Entry Points

**Python Module Entry Point:**
- Location: `.aether/__main__.py`
- Triggers: `python -m aether`
- Responsibilities: Delegates to CLI or demo

**CLI Entry Point:**
- Location: `.aether/cli.py`, main() function
- Triggers: `python -m aether.cli <command>`
- Responsibilities: Argument parsing, command execution

**Queen Ant System Entry Point:**
- Location: `.aether/queen_ant_system.py`, QueenAntSystem class
- Triggers: await system.start()
- Responsibilities: Initialize colony, pheromone layer, phase engine, memory layer

**REPL Entry Point:**
- Location: `.aether/repl.py`, AetherREPL class
- Triggers: `python -m aether.cli repl`
- Responsibilities: Interactive shell for real-time colony control

## Error Handling

**Strategy:** Pattern-based error prevention with learning

**Patterns:**
- ErrorLedger logs all errors with full context (symptom, type, location, root cause)
- ErrorPattern aggregates similar errors (3+ occurrences = flagged)
- Systematic fixes deployed for recurring patterns
- Errors learned in LongTermMemory for prevention

**Error Categories:**
- Technical: SYNTAX, IMPORT, RUNTIME, TYPE
- Agent-specific: SPAWNING, CAPABILITY, PHASE, VERIFICATION
- External: API, NETWORK, FILE
- Quality: LOGIC, PERFORMANCE, SECURITY

**Severity Levels:**
- CRITICAL: Blocks progress, must fix immediately
- HIGH: Significant impact, should fix soon
- MEDIUM: Moderate impact, fix when convenient
- LOW: Minor impact, note for future
- INFO: Informational, no action needed

## Cross-Cutting Concerns

**Logging:** Stored in ErrorLedger, printed to console, no external logging framework

**Validation:** Task dependencies validated before execution, signal strength thresholds checked

**Authentication:** Not applicable (local system)

**State Persistence:** Checkpoints saved to `.aether/checkpoints/`, errors to `.aether/errors/`, memory to `.aether/memory/`

**Semantic Understanding:** Optional sentence-transformers integration for 10-100x bandwidth reduction in pheromone communication

**Meta-Learning:** Spawn outcomes tracked to improve specialist selection over time

---

*Architecture analysis: 2025-02-01*
