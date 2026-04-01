# Codebase Structure

**Analysis Date:** 2026-04-01

## Directory Layout

```
Aether/                          # Project root (npm package + Go module)
├── .aether/                     # Shell runtime (source of truth, packaged with npm)
│   ├── aether-utils.sh          # Main dispatcher (~5,642 lines, ~130+ subcommands)
│   ├── workers.md               # Worker role definitions, spawn protocol
│   ├── utils/                   # Shell modules (42 scripts)
│   │   ├── Domain modules (9):  # flag.sh, spawn.sh, session.sh, suggest.sh,
│   │   │                        # queen.sh, swarm.sh, learning.sh, pheromone.sh, state-api.sh
│   │   ├── Learning stack (6):  # trust-scoring.sh, event-bus.sh, instinct-store.sh,
│   │   │                        # graph.sh, consolidation.sh, consolidation-seal.sh
│   │   ├── Infrastructure (6):  # file-lock.sh, atomic-write.sh, error-handler.sh,
│   │   │                        # hive.sh, midden.sh, skills.sh
│   │   ├── XML (5):             # xml-core.sh, xml-query.sh, xml-compose.sh,
│   │   │                        # xml-convert.sh, xml-utils.sh
│   │   ├── Other (16):          # scan.sh, immune.sh, council.sh, clash-detect.sh,
│   │   │                        # worktree.sh, spawn-tree.sh, swarm-display.sh,
│   │   │                        # emoji-audit.sh, colorize-log.sh, semantic-cli.sh,
│   │   │                        # state-loader.sh, merge-driver-lockfile.sh,
│   │   │                        # chamber-utils.sh, chamber-compare.sh, watch-spawn-tree.sh,
│   │   │                        # oracle.sh (in utils/oracle/)
│   │   └── curation-ants/       # 8 ants + orchestrator
│   ├── skills/                  # Reusable behavior modules
│   │   ├── colony/              # 10 colony skills (build-discipline, error-presentation, etc.)
│   │   └── domain/              # 18 domain skills (golang, react, python, etc.)
│   ├── templates/               # 12 templates (colony-state, pheromones, QUEEN.md, etc.)
│   ├── docs/                    # Distributed documentation
│   │   ├── command-playbooks/   # 9 split playbooks (build-prep/context/wave/verify/complete,
│   │   │                        #   continue-verify/gates/advance/finalize)
│   │   ├── disciplines/         # 7 discipline docs
│   │   ├── plans/               # Historical plan docs
│   │   └── archive/             # Deprecated docs
│   ├── exchange/                # XML exchange modules
│   │   ├── pheromone-xml.sh     # Pheromone XML export/import
│   │   ├── wisdom-xml.sh        # Wisdom XML export/import
│   │   ├── registry-xml.sh      # Registry XML export/import
│   │   └── *.xml                # Generated XML artifacts
│   ├── schemas/                 # XSD schemas (6 files)
│   ├── agents-claude/           # Claude agent mirror for packaging (24 files, byte-identical)
│   ├── agents/                  # Condensed agent definitions for OpenCode (24 files)
│   ├── data/                    # LOCAL ONLY (excluded by .npmignore)
│   │   ├── COLONY_STATE.json    # Colony state (version, plan, memory, errors, signals)
│   │   ├── pheromones.json      # Active pheromone signals
│   │   ├── session.json         # Session tracking
│   │   ├── learning-observations.json # Captured learnings
│   │   ├── activity.log         # Activity log
│   │   ├── constraints.json     # Legacy constraints
│   │   ├── colonies/            # Per-colony data (midden, spawn-tree-archive, swarm-archive)
│   │   ├── midden/              # Failure records
│   │   ├── backups/             # State backups
│   │   └── survey/              # Territory survey results
│   ├── dreams/                  # LOCAL ONLY (session notes, never distributed)
│   ├── oracle/                  # LOCAL ONLY (deep research artifacts)
│   ├── chambers/                # Archived completed colonies
│   ├── locks/                   # File lock state
│   ├── temp/                    # Temp files (PID-based orphan cleanup)
│   ├── checkpoints/             # Session checkpoints
│   └── rules/                   # Colony rules
│
├── .claude/                     # Claude Code integration
│   ├── commands/ant/            # 45 slash commands (Markdown)
│   ├── commands/gsd/            # GSD planning commands
│   ├── agents/ant/              # 24 agent definitions (canonical)
│   ├── rules/                   # Development rules (aether-colony.md)
│   └── hooks/                   # Claude Code hooks
│
├── .opencode/                   # OpenCode integration
│   ├── commands/ant/            # 45 slash commands (structural parity with Claude)
│   └── agents/                  # 24 agent definitions (structural parity)
│
├── cmd/aether/                  # Go CLI entry point
│   └── main.go                  # Empty main() placeholder
│
├── internal/                    # Go internal packages (not importable by external code)
│   ├── config/config.go         # Configuration loading (stub)
│   └── testing/testing.go       # Shared test helpers (stub)
│
├── pkg/                         # Go public packages
│   ├── colony/                  # Core colony types + state machine
│   │   ├── colony.go            # ColonyState, Plan, Phase, Task, Memory, Instinct, etc.
│   │   └── state_machine.go     # Transition(), AdvancePhase()
│   ├── storage/                 # Atomic JSON/JSONL file operations
│   │   └── storage.go           # Store: AtomicWrite, SaveJSON, LoadJSON, AppendJSONL, ReadJSONL
│   ├── agent/agent.go           # Worker pool (stub)
│   ├── events/events.go         # Event bus (stub)
│   ├── graph/graph.go           # Knowledge graph (stub)
│   ├── llm/llm.go               # LLM client abstraction (stub)
│   └── memory/memory.go         # Wisdom pipeline (stub)
│
├── bin/                         # Node.js CLI
│   ├── cli.js                   # Main CLI (Commander-based, ~79KB)
│   ├── npx-entry.js             # npx entry point
│   ├── npx-install.js           # npx installer
│   ├── generate-commands.js     # Command parity generator
│   ├── generate-commands.sh     # Shell command generator
│   ├── validate-package.sh      # Pre-publish validation
│   ├── sync-to-runtime.sh       # Runtime sync
│   └── lib/                     # CLI library modules (14 files)
│       ├── init.js              # Colony initialization
│       ├── interactive-setup.js # Interactive configuration
│       ├── state-guard.js       # State validation/protection
│       ├── state-sync.js        # Multi-repo state sync
│       ├── update-transaction.js # Atomic update transactions
│       ├── file-lock.js         # Node.js file locking
│       ├── spawn-logger.js      # Spawn tree logging
│       ├── logger.js            # Logging utilities
│       ├── errors.js            # Error types
│       ├── colors.js            # Terminal colors
│       ├── banner.js            # CLI banner
│       ├── caste-colors.js      # Caste-specific colors
│       ├── event-types.js       # Event type definitions
│       └── nestmate-loader.js   # Agent loading
│
├── tests/                       # Test suites
│   ├── bash/                    # 92 bash test files (subcommand-level tests)
│   ├── unit/                    # 41 JS unit test files (AVA framework)
│   ├── integration/             # 25 integration test files
│   ├── e2e/                     # 26 end-to-end test files
│   └── unit/helpers/            # Test helper utilities
│
├── test/                        # Additional test artifacts
│
├── docs/                        # Project documentation
│   ├── badges/                  # Status badges
│   ├── correlations/            # Correlation analysis
│   ├── plans/                   # Historical plans
│   └── specs/                   # Specifications
│
├── site/                        # Website files (separate repo eventually)
├── .planning/                   # Active planning phases
│   ├── codebase/                # Codebase mapping documents (this file lives here)
│   ├── milestones/              # Milestone phase definitions
│   └── research/                # Research documents
├── .planning-archive/           # Archived planning
│   ├── codebase/                # Archived codebase maps
│   ├── milestones/              # Archived milestone definitions
│   ├── phases/                  # Archived phase plans (Phases 09-44)
│   └── research/                # Archived research
│
├── .github/workflows/           # CI/CD
├── runtime/                     # Legacy runtime directory
├── src/commands/                # Legacy command source (README only)
│
├── go.mod                       # Go module definition (github.com/aether-colony/aether)
├── package.json                 # npm package (aether-colony v5.3.2)
├── CLAUDE.md                    # Claude Code project instructions
├── CHANGELOG.md                 # Automated changelog
├── README.md                    # Project README
├── LICENSE                      # MIT
├── DISCLAIMER.md                # Usage disclaimer
├── golang_test.go               # Go module structure smoke test
└── RUNTIME UPDATE ARCHITECTURE.md # Distribution architecture doc
```

## Directory Purposes

**`.aether/` (Shell Runtime):**
- Purpose: Complete colony system runtime -- all shell scripts, templates, data, and documentation
- Contains: Shell dispatcher, 42 utility modules, 28 skills, 12 templates, XML schemas, data files
- Key files: `.aether/aether-utils.sh` (dispatcher), `.aether/utils/` (modules), `.aether/data/` (state)
- Packaged: Yes, via npm `files` field in `package.json`

**`.claude/` (Claude Code Integration):**
- Purpose: Claude Code slash commands and agent definitions
- Contains: 45 command specs, 24 agent definitions, project rules, hooks
- Key files: `.claude/commands/ant/*.md`, `.claude/agents/ant/*.md`, `.claude/rules/aether-colony.md`
- Not packaged with npm (repo-level only)

**`.opencode/` (OpenCode Integration):**
- Purpose: OpenCode slash commands and agent definitions
- Contains: 45 command specs, 24 agent definitions
- Key files: `.opencode/commands/ant/*.md`, `.opencode/agents/*.md`
- Packaged: Yes, via npm `files` field

**`cmd/` (Go Entry Point):**
- Purpose: Go CLI binary entry point
- Contains: Single `main.go` file (empty stub)
- Key files: `cmd/aether/main.go`

**`pkg/` (Go Public Packages):**
- Purpose: Importable Go packages for colony operations
- Contains: 7 packages (colony, storage, agent, events, graph, llm, memory)
- Key files: `pkg/colony/colony.go`, `pkg/storage/storage.go`
- Implementation status: `colony` and `storage` are fully implemented; rest are stubs with package docs

**`internal/` (Go Internal Packages):**
- Purpose: Go packages not importable by external consumers
- Contains: `config` and `testing` (both stubs)
- Key files: `internal/config/config.go`, `internal/testing/testing.go`

**`bin/` (Node.js CLI):**
- Purpose: npm package CLI for installation, updates, and distribution
- Contains: Commander-based CLI, validation scripts, library modules
- Key files: `bin/cli.js` (main CLI), `bin/lib/` (14 library modules)

**`tests/` (Test Suites):**
- Purpose: Comprehensive testing across all layers
- Contains: Bash tests (92 files), JS unit tests (41 files), integration tests (25 files), e2e tests (26 files)
- Key files: `tests/bash/test-aether-utils.sh` (main bash test runner, ~73KB)

**`.planning/` (Active Planning):**
- Purpose: Current phase planning, milestones, research
- Contains: Phase definitions, milestone plans, codebase mapping documents
- Key files: `.planning/phases/`, `.planning/milestones/`

## Key File Locations

**Entry Points:**
- `.aether/aether-utils.sh`: Shell dispatcher -- all subcommands route through here
- `cmd/aether/main.go`: Go CLI entry point (stub)
- `bin/cli.js`: Node.js CLI for npm distribution
- `.claude/commands/ant/init.md`: Colony initialization command
- `.claude/commands/ant/build.md`: Build orchestrator (loads playbooks)
- `.claude/commands/ant/continue.md`: Continue orchestrator (loads playbooks)
- `.claude/commands/ant/run.md`: Autopilot command

**Configuration:**
- `package.json`: npm package config (version, scripts, dependencies)
- `go.mod`: Go module config (module path, Go version)
- `.gitignore`: Git ignore rules
- `.npmignore`: npm packaging exclusions
- `.gitattributes`: Git attributes

**Core Logic (Shell):**
- `.aether/utils/state-api.sh`: Colony state read/write/mutate/migrate
- `.aether/utils/pheromone.sh`: Pheromone signal system (largest module)
- `.aether/utils/learning.sh`: Wisdom pipeline
- `.aether/utils/queen.sh`: QUEEN.md wisdom management
- `.aether/utils/swarm.sh`: Parallel worker management
- `.aether/utils/atomic-write.sh`: Atomic file writes
- `.aether/utils/file-lock.sh`: File locking

**Core Logic (Go):**
- `pkg/colony/colony.go`: All colony state type definitions
- `pkg/colony/state_machine.go`: State transitions and phase advancement
- `pkg/storage/storage.go`: Atomic JSON/JSONL file operations

**Data Files:**
- `.aether/data/COLONY_STATE.json`: Colony state (plan, memory, errors, signals)
- `.aether/data/pheromones.json`: Active pheromone signals
- `.aether/data/session.json`: Session tracking
- `.aether/data/learning-observations.json`: Captured learnings
- `.aether/data/activity.log`: Activity log
- `.aether/data/queen-wisdom.json`: Local wisdom

**Agent Definitions:**
- `.claude/agents/ant/*.md`: 24 canonical Claude agent definitions
- `.aether/agents-claude/*.md`: Byte-identical mirror for packaging
- `.opencode/agents/*.md`: 24 OpenCode agent definitions

**Templates:**
- `.aether/templates/colony-state.template.json`: COLONY_STATE.json template
- `.aether/templates/QUEEN.md.template`: QUEEN.md template
- `.aether/templates/pheromones.template.json`: Pheromones template
- `.aether/templates/session.template.json`: Session template

**Testing:**
- `tests/bash/test-aether-utils.sh`: Main bash test runner (~73KB, runs all subcommand tests)
- `tests/bash/test-*.sh`: 92 individual bash test files
- `tests/unit/*.test.js`: 41 JS unit tests (AVA framework)
- `tests/integration/*.test.js`: 25 integration tests
- `tests/e2e/`: 26 end-to-end test files
- `pkg/colony/colony_test.go`: Go colony type tests
- `pkg/storage/storage_test.go`: Go storage tests
- `golang_test.go`: Go module structure smoke test

## Naming Conventions

**Shell Files:**
- Utility modules: `kebab-case.sh` (e.g., `state-api.sh`, `atomic-write.sh`)
- Curation ants: `kebab-case.sh` in `curation-ants/` subdirectory (e.g., `orchestrator.sh`)
- Test files: `test-<module>.sh` (e.g., `test-state-api.sh`, `test-pheromone-module.sh`)
- Templates: `kebab-case.template.<ext>` (e.g., `colony-state.template.json`)
- Skills: `kebab-case/` directory containing `SKILL.md` (e.g., `build-discipline/SKILL.md`)

**Shell Functions:**
- Private functions: `_underscore_case` prefix (e.g., `_state_read`, `_pheromone_write`)
- Subcommand handlers: `_module_command` pattern (e.g., `_state_mutate`, `_trust_calculate`)
- Public helpers: `lowercase-hyphenated` (e.g., `json_ok`, `json_err`, `atomic_write`)
- Error constants: `UPPER_SNAKE_CASE` with `E_` prefix (e.g., `E_FILE_NOT_FOUND`, `E_JSON_INVALID`)

**Go Files:**
- Package files: `kebab-case.go` (e.g., `state_machine.go`)
- Test files: `<package>_test.go` or `test_<name>.go` (e.g., `colony_test.go`, `golang_test.go`)
- Package names: `lowercase` (e.g., `colony`, `storage`, `events`)

**Go Types:**
- Structs: `PascalCase` (e.g., `ColonyState`, `PhaseLearning`, `ErrorRecord`)
- Interfaces: `PascalCase` (none yet)
- Constants: `PascalCase` for exported (e.g., `StateREADY`, `ErrInvalidTransition`), `camelCase` for unexported
- JSON tags: `snake_case` (e.g., `json:"current_phase"`, `json:"colony_name"`)

**Markdown Files:**
- Commands: `kebab-case.md` (e.g., `build.md`, `continue.md`, `data-clean.md`)
- Agents: `aether-<role>.md` (e.g., `aether-builder.md`, `aether-queen.md`)
- Documentation: `kebab-case.md` (e.g., `known-issues.md`, `pheromones.md`)

**Directories:**
- Shell modules: `kebab-case` (e.g., `utils/`, `curation-ants/`)
- Go packages: `kebab-case` (e.g., `pkg/colony/`, `internal/config/`)
- Agent types: `kebab-case` (e.g., `ant/`, `gsd/`)

## Where to Add New Code

**New Shell Subcommand:**
- Implementation: Add function to appropriate `.aether/utils/<module>.sh` file
- Registration: Add `case` entry in `.aether/aether-utils.sh` dispatcher (~line 5600+)
- Tests: Add `tests/bash/test-<module>.sh` or extend existing test file
- Documentation: Update `CLAUDE.md` command table if user-facing

**New LLM Slash Command:**
- Claude: `.claude/commands/ant/<name>.md`
- OpenCode: `.opencode/commands/ant/<name>.md`
- Must maintain parity (same filename, adapted content)

**New Agent Definition:**
- Claude: `.claude/agents/ant/aether-<role>.md`
- Mirror: `.aether/agents-claude/aether-<role>.md` (byte-identical copy)
- OpenCode: `.opencode/agents/aether-<role>.md` (structural parity)

**New Go Package:**
- Public: `pkg/<name>/<name>.go`
- Internal: `internal/<name>/<name>.go`
- Tests: `pkg/<name>/<name>_test.go` or `internal/<name>/<name>_test.go`
- Register in `golang_test.go` blank imports

**New Domain Skill:**
- Shipped with Aether: `.aether/skills/domain/<name>/SKILL.md`
- User-created: `~/.aether/skills/domain/<name>/SKILL.md`

**New Template:**
- Location: `.aether/templates/<name>.template.<ext>`

**New Curation Ant:**
- Implementation: `.aether/utils/curation-ants/<name>.sh`
- Registration: Source in `.aether/aether-utils.sh` (line ~55-63), add `case` entry in dispatcher

## Special Directories

**`.aether/data/` (Runtime State):**
- Purpose: Mutable colony state files
- Generated: Yes (by shell subcommands at runtime)
- Committed: Sometimes (COLONY_STATE.json tracked; others in .gitignore)
- NEVER modify programmatically (protected path per CLAUDE.md rules)

**`.aether/dreams/` (Session Notes):**
- Purpose: Dream journal entries and session notes
- Generated: Yes
- Committed: No (excluded by .npmignore)
- NEVER modify programmatically

**`.aether/chambers/` (Archived Colonies):**
- Purpose: Completed colony archives
- Generated: Yes (by `/ant:entomb`)
- Committed: Yes (preserved for history)

**`.aether/agents-claude/` (Agent Mirror):**
- Purpose: Byte-identical copy of `.claude/agents/ant/` for npm packaging
- Generated: No (manual sync)
- Committed: Yes
- Must match `.claude/agents/ant/` exactly (enforced by `npm run lint:sync`)

**`.aether/temp/` (Temp Files):**
- Purpose: Atomic write temp files (PID-based)
- Generated: Yes (by atomic-write.sh)
- Committed: No
- Cleaned up: On exit trap + startup orphan detection

**`.aether/locks/` (File Locks):**
- Purpose: File lock state
- Generated: Yes
- Committed: No

**`node_modules/` (Dependencies):**
- Purpose: npm dependencies
- Generated: Yes
- Committed: No

**`.planning-archive/` (Historical Plans):**
- Purpose: Archived phase plans and milestones
- Generated: Yes
- Committed: Yes (historical record)

---

*Structure analysis: 2026-04-01*
