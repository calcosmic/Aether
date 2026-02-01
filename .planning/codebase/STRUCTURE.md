# Codebase Structure

**Analysis Date:** 2025-02-01

## Directory Layout

```
[project-root]/
├── .aether/                    # Core Aether system (Python)
│   ├── memory/                 # Triple-layer memory system
│   │   ├── working_memory.py   # 200k token working memory
│   │   ├── short_term_memory.py # 10-session compressed memory
│   │   ├── long_term_memory.py # Persistent knowledge patterns
│   │   ├── triple_layer_memory.py # Memory orchestrator
│   │   ├── meta_learner.py     # Meta-learning for spawning
│   │   └── outcome_tracker.py  # Testing outcome tracking
│   ├── .aether/               # Runtime data (gitignored)
│   │   ├── checkpoints/        # State machine checkpoints
│   │   ├── errors/            # Error ledger persistence
│   │   └── data/              # Runtime data
│   ├── checkpoints/           # Additional checkpoint storage
│   ├── errors/               # Additional error storage
│   └── data/                 # Additional data storage
├── .claude/                   # Claude Code integration
│   └── commands/              # Claude-native prompts
│       └── ant/               # /ant:* commands
├── .planning/                 # Planning documents
│   ├── codebase/              # Codebase analysis docs
│   └── phases/                # Phase plans
├── .ralph/                    # Ralph research (legacy)
├── README.md                  # Main documentation
└── LICENSE                    # MIT License
```

## Directory Purposes

**`.aether/` - Core System:**
- Purpose: All Python code for the Aether Queen Ant Colony system
- Contains: Worker ants, pheromone system, state machine, memory, learning
- Key files: `queen_ant_system.py`, `worker_ants.py`, `pheromone_system.py`, `phase_engine.py`, `state_machine.py`

**`.aether/memory/` - Memory System:**
- Purpose: Triple-layer hierarchical memory implementation
- Contains: Working, short-term, and long-term memory layers; meta-learning and outcome tracking
- Key files: `triple_layer_memory.py`, `working_memory.py`, `short_term_memory.py`, `long_term_memory.py`, `meta_learner.py`, `outcome_tracker.py`

**`.aether/.aether/`, `.aether/checkpoints/`, `.aether/errors/`, `.aether/data/` - Runtime Data:**
- Purpose: Runtime data storage (gitignored)
- Contains: Checkpoints, error logs, persistent data
- Generated: Yes
- Committed: No

**`.claude/commands/` - Claude Integration:**
- Purpose: Claude Code command definitions for /ant:* commands
- Contains: Prompt files that define slash commands
- Key files: Various .md or .prompt files defining /ant:init, /ant:plan, etc.

**`.planning/` - Planning Documents:**
- Purpose: Project planning and codebase analysis
- Contains: This file (STRUCTURE.md), ARCHITECTURE.md, phase plans
- Key files: `codebase/ARCHITECTURE.md`, `codebase/STRUCTURE.md`, `phases/*`

## Key File Locations

**Entry Points:**
- `.aether/__main__.py`: Python module entry point
- `.aether/cli.py`: Command-line interface
- `.aether/repl.py`: Interactive REPL
- `.aether/queen_ant_system.py`: Main system class

**Core Architecture:**
- `.aether/worker_ants.py`: Six Worker Ant castes with autonomous spawning
- `.aether/pheromone_system.py`: Pheromone signal system
- `.aether/state_machine.py`: State machine orchestration
- `.aether/phase_engine.py`: Phase execution with emergence

**Configuration:**
- `.aether/semantic_layer.py`: Semantic communication configuration
- No config file (configuration is programmatic)

**Core Logic:**
- `.aether/queen_ant_system.py`: Main system orchestration
- `.aether/phase_engine.py`: Phase lifecycle management
- `.aether/worker_ants.py`: Agent behaviors and spawning
- `.aether/memory/triple_layer_memory.py`: Memory orchestration

**Testing:**
- No dedicated test directory (testing done via demo functions)
- Test generation: `.aether/worker_ants.py` (VerifierAnt.generate_test)

**Runtime Data:**
- `.aether/.aether/checkpoints/`: State machine checkpoints
- `.aether/.aether/errors/`: Error ledger JSON files
- `.aether/memory/long_term.json`: Long-term memory persistence

## Naming Conventions

**Files:**
- Modules: `snake_case.py` (e.g., `worker_ants.py`, `pheromone_system.py`)
- Test files: No dedicated test files (tests are demo functions)
- Data: JSON files with descriptive names (e.g., `long_term.json`, error IDs)

**Directories:**
- Core: `.aether/` (dot prefix for system directory)
- Subsystems: `memory/`, `.aether/` (runtime), `checkpoints/`, `errors/`
- Integration: `.claude/` (Claude Code), `.planning/` (documents)

**Classes:**
- Ant castes: `{Caste}Ant` (e.g., `MapperAnt`, `PlannerAnt`)
- Systems: `{Name}System` (e.g., `QueenAntSystem`)
- Layers: `{Name}Layer` (e.g., `PheromoneLayer`)
- Base classes: `{Name}Ant` (e.g., `WorkerAnt`)

**Functions:**
- Public API: `verb_noun` (e.g., `initiate_project`, `execute_phase`)
- Internal: `_verb_noun` (e.g., `_execute_with_emergence`)
- Factory: `create_{name}` (e.g., `create_colony`, `create_pheromone_layer`)

**Constants:**
- Enums: `PascalCase` (e.g., `PheromoneType`, `SystemState`)
- Module-level: `UPPER_SNAKE_CASE` (e.g., `SENSITIVITY_PROFILES`)

## Where to Add New Code

**New Worker Ant Caste:**
- Implementation: `.aether/worker_ants.py`
- Pattern: Inherit from `WorkerAnt`, define `caste`, `capabilities`, `sensitivity`, `spawns`
- Register in `Colony._init_worker_ants()`

**New Pheromone Type:**
- Implementation: `.aether/pheromone_system.py`
- Pattern: Add enum value to `PheromoneType`, update default half-lives
- Update sensitivity profiles in `SENSITIVITY_PROFILES`

**New Memory Layer:**
- Implementation: `.aether/memory/{layer_name}_memory.py`
- Integration: `.aether/memory/triple_layer_memory.py`
- Pattern: Inherit or follow pattern of existing layers

**New State Machine State:**
- Implementation: `.aether/state_machine.py`
- Pattern: Add to `SystemState` enum, implement transition logic in `_execute_transition()`

**New CLI Command:**
- Implementation: `.aether/cli.py`
- Pattern: Add subparser, handler in `run_command()`

**New Claude Command:**
- Implementation: `.claude/commands/ant/{command}.md`
- Pattern: Follow existing command structure

**Utilities:**
- Shared helpers: `.aether/{utility_name}.py`
- Memory utilities: `.aether/memory/{utility_name}.py`

**Configuration:**
- System settings: `.aether/queen_ant_system.py` (in `__init__`)
- Colony settings: `.aether/worker_ants.py` (factory functions)

## Special Directories

**`.aether/memory/`:**
- Purpose: Triple-layer memory implementation
- Generated: No
- Committed: Yes
- Contains: WorkingMemory, ShortTermMemory, LongTermMemory, TripleLayerMemory, MetaLearner, OutcomeTracker

**`.aether/.aether/`, `.aether/checkpoints/`, `.aether/errors/`, `.aether/data/`:**
- Purpose: Runtime data storage
- Generated: Yes
- Committed: No (gitignored)
- Contains: Checkpoints, error JSON files, persistent state

**`.claude/commands/ant/`:**
- Purpose: Claude Code slash command definitions
- Generated: No
- Committed: Yes
- Contains: Prompt files for /ant:* commands

**`.planning/codebase/`:**
- Purpose: Codebase analysis documents (this file and ARCHITECTURE.md)
- Generated: Yes (by CDS codebase mapper)
- Committed: Yes
- Contains: ARCHITECTURE.md, STRUCTURE.md

**`.planning/phases/`:**
- Purpose: Phase implementation plans
- Generated: Yes (by CDS planner)
- Committed: Yes
- Contains: Phase-specific implementation plans

---

*Structure analysis: 2025-02-01*
