# Codebase Structure

**Analysis Date:** 2026-08-01

## Directory Layout

```
/Users/callumcowie/repos/Aether/
├── cmd/                               # Go CLI commands (500+ files)
│   ├── aether/main.go                # Binary entry point
│   ├── root.go                       # Cobra root command setup
│   ├── build_*.go                    # Build orchestration (10+ files)
│   ├── continue_*.go                 # Continue flow (5+ files)
│   ├── pheromone_*.go                # Pheromone management (10+ files)
│   ├── build_attempt*.go             # Build attempt tracking
│   ├── seal.go, entomb_cmd.go        # Lifecycle commands
│   ├── plan_*.go                     # Planning commands
│   ├── status*.go, history.go        # Status & monitoring
│   ├── learning.go, instinct.go      # Learning pipeline
│   ├── session*.go                   # Session management
│   ├── *_cmds.go                     # Command groupings
│   ├── helpers.go, ctxhelper.go      # Helpers & utilities
│   └── testdata/                     # Test fixtures
│
├── pkg/                              # Shared Go packages
│   ├── agent/                        # Worker pool & caste system
│   │   ├── agent.go                 # Agent interface, Caste enum
│   │   ├── pool.go                  # Agent spawning
│   │   ├── spawn_tree.go            # Execution tree tracking
│   │   ├── registry.go              # Agent registry
│   │   ├── stream_manager.go        # Output multiplexing
│   │   └── task_*.go                # Task routing & graph
│   │
│   ├── colony/                       # State management
│   │   └── colony.go                # ColonyState struct & types
│   │
│   ├── storage/                      # Atomic file operations
│   │   ├── storage.go               # Store interface
│   │   ├── lock.go                  # Cross-process locking
│   │   └── paths.go                 # Path resolution
│   │
│   ├── events/                       # Event bus
│   │   ├── event.go                 # Event type, TTL
│   │   ├── bus.go                   # Pub/sub implementation
│   │   └── ceremony.go              # Lifecycle events
│   │
│   ├── learn/                        # Learning pipeline
│   │   ├── learn.go                 # LearnStore interface
│   │   ├── colony_store.go          # Repo-local storage
│   │   ├── hive_store.go            # Cross-colony wisdom
│   │   ├── curator.go               # Learning promotion
│   │   └── classify.go              # Evidence classification
│   │
│   ├── codex/                        # TS host integration
│   │   ├── dispatch.go              # Work dispatch
│   │   ├── execution_binding.go     # Execution tracking
│   │   ├── handoff.go               # Worker handoff
│   │   └── platform_*.go            # Platform-specific
│   │
│   ├── graph/                        # Knowledge graph
│   │   └── doc.go                   # Graph types
│   │
│   ├── llm/                          # LLM streaming
│   │   └── stream.go                # Stream handlers
│   │
│   ├── terminal/                     # Output formatting
│   │   └── output.go                # Terminal utilities
│   │
│   ├── trace/                        # Command tracing
│   │   └── tracer.go                # Execution recorder
│   │
│   ├── cache/                        # Caching layer
│   ├── memory/                       # Memory management
│   ├── exchange/                     # Signal exchange
│   └── downloader/                   # Binary downloads
│
├── .aether/                          # System files (source of truth)
│   ├── data/                         # LOCAL - Colony state
│   │   ├── COLONY_STATE.json        # Current state
│   │   ├── event-bus.jsonl          # Event log
│   │   ├── pheromones.json          # Active signals
│   │   ├── session.json             # Session metadata
│   │   └── build/                   # Build attempts
│   │       └── phase-N/attempts/    # Phase-specific attempts
│   │
│   ├── docs/                         # Distributed documentation
│   ├── skills/                       # Reusable behaviors
│   │   ├── colony/                  # Colony skills (55 items)
│   │   └── domain/                  # Domain skills (31 items)
│   │
│   ├── templates/                    # Project templates
│   ├── schemas/                      # JSON validation
│   ├── utils/                        # Helper scripts
│   ├── commands/                     # Command definitions
│   ├── exchange/                     # XML signal modules
│   ├── archive/                      # Historical chambers
│   ├── checkpoints/                  # Recovery points
│   ├── locks/                        # File locks
│   └── version.json                 # Version info
│
├── .claude/                          # Claude Code platform
│   ├── agents/ant/                  # Agent definitions (Markdown)
│   ├── commands/ant/                # Slash commands (60 items)
│   │   ├── init.md, plan.md         # Lifecycle
│   │   ├── build.md, continue.md    # Execution
│   │   └── ...                      # 60+ commands
│   │
│   ├── rules/                        # Platform rules
│   └── settings*.json               # Configuration
│
├── .opencode/                        # OpenCode platform
│   ├── agents/                      # Agent definitions (Markdown)
│   ├── commands/ant/                # Bash script commands
│   ├── OPENCODE.md                 # Rules
│   └── package.json                # NPM distribution
│
├── .codex/                           # Codex platform
│   ├── agents/                      # Agent definitions (TOML)
│   └── CODEX.md                    # Rules
│
├── docs/                             # Repository documentation
├── skills/                           # Repo-local custom skills
├── .github/                          # GitHub workflows
├── go.mod, go.sum                   # Go module dependencies
├── CLAUDE.md                         # Development guide
└── CHANGELOG.md                      # Changelog

```

## Directory Purposes

**cmd/**
- Purpose: CLI command implementations using Cobra framework
- Contains: 500+ Go files with command handlers, arguments, output formatting
- Key files: `root.go` (Cobra setup), `build_attempt.go` (build tracking), `pheromone_mgmt.go` (signal system)
- Invocation: `aether <command>` routes to handler in this directory

**pkg/agent/**
- Purpose: Worker pool management and caste-based dispatch
- Contains: Agent interface, Pool (spawn/execute), StreamManager (output multiplexing)
- Key file: `agent.go` (Caste enum: builder, watcher, scout, oracle, etc.)
- Used by: Build/continue orchestration, autopilot, agent spawning

**pkg/colony/**
- Purpose: Struct definitions for colony state and lifecycle
- Contains: ColonyState (phases, tasks, instincts), Phase/Task/Workfree status constants
- Key file: `colony.go` (round-trip JSON compatibility with shell)
- Used by: All state read/write operations, phase transitions

**pkg/storage/**
- Purpose: Atomic file operations with cross-process locking
- Contains: Store (reads/writes with temp+rename), FileLocker (Unix/Windows)
- Key methods: `AtomicWrite()`, `UpdateFile()` (read-modify-write with lock)
- Used by: All JSON mutations in cmd/ and pkg/

**pkg/events/**
- Purpose: Event bus with pub/sub, topic matching, TTL expiry
- Contains: Event type, Bus (publish/query), topic matching with wildcards
- Persistence: JSONL append-only log with file locking
- Used by: Learning pipeline triggers, agent activation, observation recording

**pkg/learn/**
- Purpose: Learning pipeline from observations to cross-colony wisdom
- Contains: Entry/Evidence types, LearnStore interface, Curator promotion logic
- Implementations: ColonyStore (repo-local JSON), HiveStore (cross-colony SQLite)
- Used by: Build completion callback, seal promotion, hive-read for context injection

**pkg/codex/**
- Purpose: Integration with TypeScript host (Claude Code, OpenCode) for worker dispatch
- Contains: Dispatch (send work), ExecutionBinding (track runs), Handoff (context transfer)
- Used by: Build command → worker spawn, continue verification

**.aether/data/**
- Purpose: Runtime state storage (never distributed)
- Contains: COLONY_STATE.json (mutable), event-bus.jsonl (append-only), session.json
- Locking: FileLocker in `.aether/locks/` prevents concurrent corruption
- Lifecycle: Initialized at `/ant-init`, preserved across `/ant-resume`, archived at `/ant-seal`

**.aether/docs/**
- Purpose: Distributed documentation (system guide, API reference)
- Included in: Hub publish via `aether publish`
- Read by: Platform wrappers, user reference

**.claude/commands/ant/**
- Purpose: 60 slash commands for Claude Code platform
- Files: One `.md` per command (e.g., `build.md`, `continue.md`)
- Distribution: Published to hub via `aether publish` → `~/.aether/system/claude/commands/`

**.opencode/agents/** & **.codex/agents/**
- Purpose: Platform-specific agent definitions
- Format: Markdown (.md) for OpenCode, TOML (.toml) for Codex
- Distribution: Published to hub like Claude commands

**cmd/testdata/**
- Purpose: Test fixtures and mock data
- Contains: Sample state files, expected outputs, test scenarios

## Key File Locations

**Entry Points:**
- `cmd/aether/main.go` — Binary startup (calls cmd.Execute())
- `cmd/root.go` — Cobra root command, store/tracer initialization

**Configuration:**
- `CLAUDE.md` — Development guide (architecture, workflow, tools)
- `.aether/version.json` — Version tracking
- `go.mod` — Go module manifest

**Core Logic:**
- `pkg/colony/colony.go` — State structures
- `pkg/agent/agent.go` — Caste definition and interfaces
- `pkg/storage/storage.go` — Atomic operations
- `cmd/build_attempt.go` — Build tracking

**Testing:**
- `cmd/*_test.go` — Command tests (co-located)
- `pkg/*/\*_test.go` — Package tests (co-located)

## Naming Conventions

**Files:**
- Commands: `<feature>_cmd.go` or `<feature>_cmds.go` (e.g., `build_flow_cmds.go`)
- Core logic: `<feature>.go` (e.g., `colony.go`, `instinct.go`)
- Tests: `<feature>_test.go` (co-located with source)
- Helpers: `helpers.go`, `ctxhelper.go` (general utilities)
- Entry: `main.go`, `root.go`

**Directories:**
- Packages: `<lowercase_name>/` (e.g., `pkg/agent/`)
- Platforms: `.claude/`, `.opencode/`, `.codex/`
- System: `.aether/`

**State Files (in .aether/data/):**
- Main state: `COLONY_STATE.json`
- Events: `event-bus.jsonl`
- Signals: `pheromones.json`
- Sessions: `session.json`
- Builds: `build/phase-{N}/attempts/{id}.json`

## Where to Add New Code

**New Command:**
1. Create `cmd/my_feature_cmd.go` with `myFeatureCmd` Cobra struct
2. Implement `RunE` function with `func(cmd *cobra.Command, args []string) error`
3. Register in init: `rootCmd.AddCommand(myFeatureCmd)`
4. Create `cmd/my_feature_cmd_test.go` with tests

**New Agent/Worker Caste:**
1. Add to `pkg/agent/agent.go` Caste enum (CasteMyRole)
2. Create agent implementation file: `cmd/my_agent.go` or `pkg/agent/my_agent.go`
3. Implement Agent interface: Name(), Caste(), Triggers(), Execute()
4. Register in agent registry (in cmd/root.go init or pkg/agent/registry.go)
5. Add `.claude/agents/ant/my-agent.md` for Claude wrapper
6. Add `.opencode/agents/my-agent.md` for OpenCode wrapper
7. Add `.codex/agents/my-agent.toml` for Codex wrapper

**New State Field:**
1. Add to `pkg/colony/ColonyState` struct
2. Update JSON schema in `.aether/schemas/` if needed
3. Create migration in cmd if loading old state files
4. Update `cmd/state_load.go` to handle new field

**New Package:**
1. Create `pkg/mypackage/` directory
2. Add core file: `mypackage.go` with package comment
3. Add types/interfaces
4. Create `mypackage_test.go` with tests
5. Import from cmd/ or other packages via `"github.com/calcosmic/Aether/pkg/mypackage"`

**Platform-Specific Code:**
- Claude Code: Add to `.claude/` (agents, commands, rules)
- OpenCode: Add to `.opencode/` (agents, commands, config)
- Codex: Add to `.codex/` (agents, rules)
- Shared: Add to `.aether/` (docs, skills, templates)

## Special Directories

**`.aether/data/`**
- Generated at runtime, never committed
- Contains current colony state and session data
- Backed up on seal, archived in `.aether/archive/`

**`cmd/testdata/`**
- Test fixtures and mock data
- Always committed
- Used by `*_test.go` files for expected output validation

**`.planning/codebase/`**
- Auto-generated by GSD mapper
- Contains: ARCHITECTURE.md, STRUCTURE.md, CONVENTIONS.md, TESTING.md, STACK.md, INTEGRATIONS.md, CONCERNS.md
- Used by GSD executor to follow patterns and conventions

---

*Structure analysis: 2026-08-01*
