# Codebase Structure

**Analysis Date:** 2026-03-19

## Directory Layout

```
Aether/
├── bin/                          # Node.js CLI and orchestration
│   ├── cli.js                    # Main entry point (89KB, handles install, hub setup)
│   ├── lib/                      # State and command management
│   │   ├── state-sync.js         # COLONY_STATE synchronization
│   │   ├── state-guard.js        # State mutation guards and validation
│   │   ├── file-lock.js          # Distributed file locking primitives
│   │   ├── update-transaction.js # Hub sync and package updates
│   │   ├── init.js               # Colony initialization logic
│   │   ├── spawn-logger.js       # Worker spawn tracking
│   │   ├── telemetry.js          # Event/telemetry collection
│   │   ├── model-profiles.js     # LiteLLM routing configuration
│   │   └── [11 other modules]
│   ├── generate-commands.sh      # Sync commands between .claude and .opencode
│   ├── validate-package.sh       # Pre-publish validation
│   └── npx-install.js            # NPX installer helper
│
├── .aether/                      # Source of truth (distributed via npm)
│   ├── aether-utils.sh           # Core utility layer (10,249 lines, 150+ subcommands)
│   ├── utils/                    # Shared bash utilities
│   │   ├── file-lock.sh          # Advisory file locking with PID tracking
│   │   ├── atomic-write.sh       # Atomic file writes (temp + mv)
│   │   ├── state-loader.sh       # Load and validate COLONY_STATE.json
│   │   ├── chamber-utils.sh      # Archive/chamber operations
│   │   ├── spawn-tree.sh         # Worker spawn tree visualization
│   │   ├── xml-core.sh           # XML parsing helpers
│   │   ├── xml-compose.sh        # XInclude composition
│   │   ├── xml-query.sh          # XPath/XQuery operations
│   │   ├── semantic-cli.sh       # Command dispatch and routing
│   │   └── [8 other utilities]
│   │
│   ├── exchange/                 # XML export/import for inter-colony knowledge
│   │   ├── pheromone-xml.sh      # Signal export/import with namespaces
│   │   ├── wisdom-xml.sh         # Philosophy promotion pipeline
│   │   ├── registry-xml.sh       # Colony lineage and ancestry
│   │   └── [sample XML files]
│   │
│   ├── schemas/                  # XSD validation for XML exchange
│   │   ├── pheromone.xsd         # Signal structure validation
│   │   ├── queen-wisdom.xsd      # Philosophy format
│   │   ├── colony-registry.xsd   # Lineage schema
│   │   ├── prompt.xsd            # Worker prompt templates
│   │   └── [7 other schemas]
│   │
│   ├── templates/                # Bootstrap and initialization templates
│   │   ├── colony-state.template.json
│   │   ├── constraints.template.json
│   │   ├── pheromones.template.json
│   │   ├── QUEEN.md.template
│   │   └── [8 other templates]
│   │
│   ├── docs/                     # Distributed documentation
│   │   ├── command-playbooks/    # Split orchestration stages
│   │   │   ├── build-prep.md     # Phase 1: Input validation
│   │   │   ├── build-context.md  # Phase 2: State loading
│   │   │   ├── build-wave.md     # Phase 3: Worker spawning
│   │   │   ├── build-verify.md   # Phase 4: Output verification
│   │   │   ├── build-complete.md # Phase 5: Learning synthesis
│   │   │   └── [4 continue playbooks]
│   │   ├── caste-system.md       # Worker roles and castes
│   │   ├── error-codes.md        # Error code reference
│   │   ├── pheromones.md         # Signal system guide
│   │   └── [7 other guides]
│   │
│   ├── agents-claude/            # Claude agent definitions (packaging mirror)
│   │   ├── aether-builder.md
│   │   ├── aether-watcher.md
│   │   ├── aether-queen.md
│   │   └── [22 agents total, byte-identical to .claude/agents/ant/]
│   │
│   ├── data/                     # LOCAL ONLY (excluded from .npmignore)
│   │   ├── COLONY_STATE.json     # Primary: colony state, phases, events, learnings
│   │   ├── pheromones.json       # Signals (FOCUS/REDIRECT/FEEDBACK)
│   │   ├── constraints.json      # Focus areas and hard constraints
│   │   ├── session.json          # Current session metadata
│   │   ├── midden/               # Failure records (category, severity, root cause)
│   │   ├── survey/               # Codebase survey results
│   │   ├── backups/              # Automatic state backups
│   │   └── [other runtime data]
│   │
│   ├── dreams/                   # LOCAL ONLY (never distributed)
│   │   └── *.md                  # User session notes and observations
│   │
│   ├── oracle/                   # LOCAL ONLY (deep research state)
│   │   ├── prompts/              # RALF loop prompts and results
│   │   ├── archive/              # Prior oracle runs
│   │   └── [research artifacts]
│   │
│   ├── chambers/                 # LOCAL ONLY (archived colonies)
│   │   └── v1-1-*/               # Sealed colony snapshots
│   │
│   ├── checkpoints/              # LOCAL ONLY (session recovery)
│   │   └── *.json                # Checkpoint state files
│   │
│   ├── locks/                    # LOCAL ONLY (advisory lock files)
│   │   └── [PID-based lock files]
│   │
│   ├── CONTEXT.md                # Session memory (what's in progress)
│   ├── QUEEN.md                  # Queen's wisdom and philosophies
│   ├── manifest.json             # Package metadata and version
│   └── registry.json             # Colony registry (for archaeology)
│
├── .claude/                      # Claude Code integration
│   ├── commands/ant/             # Slash commands for Claude Code
│   │   ├── init.md               # /ant:init command
│   │   ├── build.md              # /ant:build orchestrator
│   │   ├── continue.md           # /ant:continue orchestrator
│   │   ├── plan.md               # /ant:plan phase generation
│   │   ├── status.md             # /ant:status colony dashboard
│   │   ├── pheromones.md         # /ant:pheromones signal management
│   │   ├── focus.md, redirect.md, feedback.md  # Signal commands
│   │   ├── oracle.md             # /ant:oracle deep research
│   │   ├── swarm.md              # /ant:swarm bug investigation
│   │   ├── seal.md, entomb.md    # /ant:seal, /ant:entomb lifecycle
│   │   └── [28 other commands]
│   │
│   ├── agents/ant/               # Agent definitions for Claude Code
│   │   ├── aether-builder.md     # Implementation specialist
│   │   ├── aether-watcher.md     # Quality verification
│   │   ├── aether-queen.md       # Orchestrator
│   │   ├── aether-scout.md       # Research and discovery
│   │   ├── aether-oracle.md      # Deep analysis (RALF loop)
│   │   └── [22 agents total]
│   │
│   ├── rules/                    # Development guidelines
│   │   └── aether-colony.md      # System rules (consolidated)
│   │
│   └── hooks/                    # Git hooks
│
├── .opencode/                    # OpenCode integration (mirrors Claude setup)
│   ├── commands/ant/             # 36 slash commands (sync'd with .claude/)
│   ├── agents/                   # 22 agent definitions (structural parity)
│   └── opencode.json             # OpenCode config
│
├── .planning/                    # Planning artifacts (LOCAL ONLY)
│   ├── codebase/                 # Codebase analysis documents
│   │   ├── ARCHITECTURE.md       # Architecture analysis
│   │   ├── STRUCTURE.md          # This file
│   │   ├── STACK.md              # Technology stack (tech focus)
│   │   ├── INTEGRATIONS.md       # External integrations (tech focus)
│   │   ├── CONVENTIONS.md        # Coding standards (quality focus)
│   │   ├── TESTING.md            # Testing patterns (quality focus)
│   │   └── CONCERNS.md           # Technical debt (concerns focus)
│   ├── phases/                   # Phase plans and execution results
│   ├── config.json               # Planning configuration
│   └── [other planning artifacts]
│
├── tests/                        # Test suites
│   ├── unit/                     # Unit tests (35+ tests)
│   │   ├── colony-state.test.js  # State schema and validation
│   │   ├── state-sync.test.js    # Synchronization logic
│   │   ├── state-guard.test.js   # Mutation guards
│   │   ├── file-lock.test.js     # Locking primitives
│   │   ├── instinct-confidence.test.js  # Confidence calibration
│   │   ├── oracle-*.test.js      # Oracle state and routing
│   │   ├── model-profiles.test.js # LiteLLM routing
│   │   └── [27 other unit tests]
│   │
│   ├── integration/              # Integration tests (8+ tests)
│   │   ├── learning-pipeline.test.js
│   │   ├── instinct-pipeline.test.js
│   │   ├── wisdom-promotion.test.js
│   │   ├── pheromone-auto-emission.test.js
│   │   └── [4 other integration tests]
│   │
│   ├── bash/                     # Bash integration tests
│   │   ├── test-aether-utils.sh
│   │   ├── test-lock-lifecycle.sh
│   │   ├── test-xml-roundtrip.sh
│   │   └── test-generate-commands.sh
│   │
│   ├── e2e/                      # End-to-end tests (if present)
│   └── unit/helpers/
│       └── mock-fs.js            # File system mocking
│
├── runtime/                      # LEGACY (v3.x) — eliminated in v4.0
│
├── package.json                  # npm configuration (Node 16+)
├── package-lock.json             # Dependency lock
├── CLAUDE.md                     # Project development guide
├── README.md                     # User documentation
├── CHANGELOG.md                  # Version history (auto-collected)
├── RUNTIME UPDATE ARCHITECTURE.md # Hub distribution explanation
├── repo-structure.md             # (legacy)
├── TO-DOS.md                     # Known work items
└── LICENSE                       # MIT
```

## Directory Purposes

**bin/:**
- Purpose: Node.js orchestration and installation logic
- Contains: CLI entry point, state management modules, command generation
- Key files: `cli.js` (main), `lib/*` (state logic)
- Non-editable: Auto-generated by `generate-commands.sh`

**.aether/:**
- Purpose: Source of truth for system files, distributed via npm
- Contains: Utilities, templates, schemas, documentation
- Key files: `aether-utils.sh` (150+ subcommands), `templates/` (bootstrap)
- Rules: Edit here; changes propagate via `npm install -g .` → hub → `aether update`

**.aether/data/:**
- Purpose: Runtime state — LOCAL ONLY, never distributed
- Contains: COLONY_STATE.json, pheromones.json, session state
- Generated: By `/ant:init` and updated by every command
- Excluded: From `.npmignore` to prevent distribution

**.aether/dreams/:**
- Purpose: User notes and observations — LOCAL ONLY
- Contains: Session notes, decision logs, philosophy observations
- Generated: By user via `/ant:dream`, edited manually
- Excluded: From `.npmignore`

**.aether/oracle/:**
- Purpose: Deep research state — LOCAL ONLY
- Contains: RALF loop prompts, research results, archive
- Generated: By `/ant:oracle` command
- Excluded: From `.npmignore`

**.aether/chambers/:**
- Purpose: Sealed colony archives — LOCAL ONLY
- Contains: Complete snapshots of finished colonies (v1-1-bug-fixes-*, etc.)
- Generated: By `/ant:seal` and `/ant:entomb`
- Excluded: From `.npmignore`

**.claude/commands/ant/ & .opencode/commands/ant/:**
- Purpose: Slash command definitions (user-facing interfaces)
- Contains: 36 command definitions in Markdown
- Synced: By `bin/generate-commands.sh check` (linted) and `generate-commands.sh write` (propagated)
- Pattern: Front matter (name, description), markdown instructions, playbook references

**.claude/agents/ant/ & .opencode/agents/:**
- Purpose: Worker agent definitions
- Contains: 22 caste definitions with role, constraints, tools
- Pattern: YAML front matter (name, tools, model), markdown instructions
- Canonical: `.claude/agents/ant/` is source; `.aether/agents-claude/` is packaging mirror

**.planning/:**
- Purpose: Project-specific planning — LOCAL ONLY
- Contains: Codebase analysis (ARCHITECTURE.md, STRUCTURE.md), phase plans, config
- Generated: By `/gsd:map-codebase` and `/gsd:plan-phase` commands
- Excluded: From `.npmignore`

**tests/:**
- Purpose: Comprehensive test coverage
- Pattern: Co-located with source (unit tests) or centralized by type (integration, bash)
- Framework: AVA (JavaScript), custom bash runners
- Target: 80%+ coverage for new code

## Key File Locations

**Entry Points:**
- `bin/cli.js`: Node CLI entry (89KB, handles install/hub setup)
- `.claude/commands/ant/init.md`: `/ant:init` command definition
- `.claude/commands/ant/build.md`: `/ant:build` orchestrator
- `.aether/aether-utils.sh`: Bash utility entry point (10,249 lines)

**Configuration:**
- `package.json`: NPM metadata, scripts, dependencies
- `.npmignore`: Excludes data/, dreams/, oracle/, chambers/, locks/ from distribution
- `.aether/manifest.json`: Package version and metadata
- `.aether/model-profiles.yaml`: LiteLLM model routing

**Core Logic:**
- `bin/lib/state-sync.js`: State synchronization and validation
- `bin/lib/state-guard.js`: State mutation guards
- `bin/lib/init.js`: Colony initialization
- `.aether/utils/file-lock.sh`: Advisory file locking (PID-based)
- `.aether/utils/atomic-write.sh`: Safe file writes (temp + mv)

**Testing:**
- `tests/unit/colony-state.test.js`: State schema validation
- `tests/unit/state-sync.test.js`: Synchronization logic
- `tests/bash/test-aether-utils.sh`: Bash utility integration
- `tests/integration/learning-pipeline.test.js`: End-to-end learning flow

**State Files:**
- `.aether/data/COLONY_STATE.json`: Primary state (goal, phases, events, learnings)
- `.aether/data/pheromones.json`: Active signals (FOCUS/REDIRECT/FEEDBACK)
- `.aether/data/constraints.json`: Hard constraints and focus areas
- `.aether/data/session.json`: Session metadata and recovery info

## Naming Conventions

**Files:**
- Bash utilities: kebab-case with .sh extension (`file-lock.sh`, `atomic-write.sh`)
- Node modules: kebab-case.js (`state-sync.js`, `file-lock.js`)
- Markdown documents: kebab-case.md or PascalCase (CLAUDE.md, COLONY_STATE.json)
- Test files: [name].test.js or test-[name].sh pattern
- Templates: descriptive-name.template.json or similar

**Directories:**
- Logical grouping: lowercase with hyphens (`command-playbooks`, `agents-claude`)
- Private/local: prefixed with dot (`.aether`, `.claude`, `.planning`)
- System: uppercase names for distributed artifacts (CHANGELOG.md, README.md)

**Functions/Commands:**
- Bash functions: snake_case ending in `()` (`json_ok`, `cleanup_locks`)
- CLI subcommands: kebab-case in documentation (`ant:init`, `ant:build`)
- JavaScript exports: camelCase (`setupHub`, `syncAetherToRepo`)

**JSON Fields:**
- snake_case for state fields (`current_phase`, `session_id`, `initialized_at`)
- camelCase for programmatic objects (results, parameters)
- Consistency enforced by state schema validation

## Where to Add New Code

**New Slash Command:**
1. Add `.claude/commands/ant/your-command.md` with front matter and instructions
2. Add `.opencode/commands/ant/your-command.md` (mirror)
3. Run `bash bin/generate-commands.sh write` to sync
4. Add unit test in `tests/unit/cli-[feature].test.js` if logic is complex

**New Agent/Caste:**
1. Add `.claude/agents/ant/aether-your-role.md` with role definition
2. Add `.aether/agents-claude/aether-your-role.md` (packaging mirror)
3. Add `.opencode/agents/aether-your-role.md` (structural parity)
4. Update caste emoji in `aether-utils.sh` (get_caste_emoji function)
5. Add tests for role-specific behavior in `tests/integration/`

**New Utility Function:**
- Bash: Add function to `.aether/utils/[category].sh` or new file in `utils/`
- Node: Add module to `bin/lib/` or extend existing module
- Both: Add corresponding tests in `tests/unit/`

**New Test:**
- Unit test: `tests/unit/[module].test.js` (test single module)
- Integration test: `tests/integration/[feature].test.js` (test multiple components)
- Bash test: `tests/bash/test-[feature].sh` (test shell integration)
- E2E test: `tests/e2e/[scenario].test.js` (test full workflow)

**New Data Schema:**
1. Define XSD in `.aether/schemas/`
2. Add template in `.aether/templates/`
3. Add loader/validator in `bin/lib/state-loader.js` or `.aether/utils/state-loader.sh`
4. Add test in `tests/unit/validate-*.test.js`

**Documentation:**
- User guides: `.aether/docs/` (distributed)
- Development guides: `.claude/rules/` (distributed to hub)
- Session notes: `.aether/dreams/` (LOCAL ONLY)
- Planning docs: `.planning/` (LOCAL ONLY)

## Special Directories

**node_modules/:**
- Purpose: npm dependencies
- Generated: By `npm install`
- Committed: Yes (lock file `package-lock.json`)
- Never edit manually

**.aether/locks/:**
- Purpose: Advisory file locks for concurrent access control
- Generated: By file-lock.sh at runtime
- Committed: No (gitignore)
- Cleaned up automatically on process exit

**.aether/temp/:**
- Purpose: Temporary files during atomic writes
- Generated: By atomic-write.sh and other utilities
- Committed: No (gitignore)
- Cleaned up at startup (orphan detection)

**.aether/checkpoints/:**
- Purpose: Session recovery snapshots
- Generated: By `/ant:pause-colony`
- Committed: No (gitignore)
- Can be manually restored for recovery

**tests/unit/helpers/:**
- Purpose: Test utilities and mocks
- Example: `mock-fs.js` for filesystem mocking
- Pattern: Shared across unit tests
- Testing-only (never deployed)

---

*Structure analysis: 2026-03-19*
