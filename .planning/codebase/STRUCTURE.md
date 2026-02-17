# Codebase Structure

**Analysis Date:** 2026-02-17

## Directory Layout

```
/Users/callumcowie/repos/Aether/
├── .aether/                    # SOURCE OF TRUTH for system files
│   ├── aether-utils.sh         # Main utility shell script
│   ├── workers.md              # Worker definitions and disciplines
│   ├── CONTEXT.md              # Colony memory (generated)
│   ├── utils/                  # Helper shell scripts
│   │   ├── file-lock.sh
│   │   ├── atomic-write.sh
│   │   ├── error-handler.sh
│   │   ├── xml-utils.sh
│   │   └── ...
│   ├── data/                   # LOCAL colony state (never synced)
│   │   ├── COLONY_STATE.json
│   │   ├── pheromones.json
│   │   ├── constraints.json
│   │   ├── flags.json
│   │   ├── activity.log
│   │   └── spawn-tree.txt
│   ├── docs/                   # Distributed documentation
│   ├── commands/               # Distributed command definitions
│   ├── chambers/                # Archived colony states
│   ├── checkpoints/             # Session checkpoints
│   └── dreams/                  # LOCAL dream journal
│
├── runtime/                    # STAGING (auto-populated, DO NOT EDIT)
│   ├── aether-utils.sh         # Copied from .aether/
│   ├── utils/                  # Copied from .aether/
│   └── ...
│
├── bin/                        # CLI entry point
│   ├── cli.js                  # Main CLI (JavaScript)
│   ├── lib/                    # CLI libraries
│   │   ├── errors.js
│   │   ├── logger.js
│   │   ├── init.js
│   │   ├── state-sync.js
│   │   ├── model-profiles.js
│   │   └── ...
│   └── sync-to-runtime.sh      # Sync script
│
├── .claude/                    # Claude Code integration
│   └── commands/ant/           # Slash command definitions (31 commands)
│       ├── init.md
│       ├── build.md
│       ├── plan.md
│       └── ...
│
├── .opencode/                  # OpenCode integration
│   ├── commands/ant/           # Slash commands (mirrored)
│   └── agents/                 # Agent definitions
│
├── tests/                      # Test suite
│   ├── unit/                   # AVA unit tests
│   ├── bash/                   # Shell script tests
│   ├── integration/            # Integration tests
│   └── e2e/                    # End-to-end tests
│
└── package.json                # npm package definition
```

## Directory Purposes

**`.aether/`:**
- Purpose: Source of truth for all distributed system files
- Contains: workers.md, aether-utils.sh, utils/, docs/, commands/
- Key files: `workers.md`, `aether-utils.sh`, `CONTEXT.md`, `coding-standards.md`

**`runtime/`:**
- Purpose: Staging directory for npm package
- Contains: Auto-populated copy of .aether/ files
- Key files: Same as .aether/ but auto-generated

**`.aether/data/`:**
- Purpose: Local colony state (never synced to hub)
- Contains: COLONY_STATE.json, pheromones.json, activity.log, spawn-tree.txt
- Key files: All JSON state files

**`bin/`:**
- Purpose: JavaScript CLI entry point
- Contains: cli.js, lib/, sync-to-runtime.sh
- Key files: `cli.js` (77KB main CLI)

**`.claude/commands/ant/`:**
- Purpose: Claude Code slash commands
- Contains: 31 markdown prompt files
- Key files: `init.md`, `build.md`, `plan.md`, `continue.md`

**`.opencode/commands/ant/`:**
- Purpose: OpenCode slash commands (mirrored from Claude)
- Contains: Same 31 commands

**`tests/`:**
- Purpose: Test suite
- Contains: unit/, bash/, integration/, e2e/
- Key files: test-*.sh scripts, *.test.js files

## Key File Locations

**Entry Points:**
- `bin/cli.js`: Main CLI (aether command)
- `.claude/commands/ant/init.md`: Colonize new project
- `.claude/commands/ant/build.md`: Execute work
- `.claude/commands/ant/plan.md`: Plan work

**Configuration:**
- `package.json`: npm package definition
- `.aether/model-profiles.yaml`: Model routing configuration
- `.aether/registry.json`: Worker registry

**Core Logic:**
- `.aether/aether-utils.sh`: 143KB utility layer
- `bin/cli.js`: 77KB CLI layer
- `bin/lib/*.js`: 15 library modules

**Testing:**
- `tests/unit/`: AVA tests
- `tests/bash/test-aether-utils.sh`: Shell tests

## Naming Conventions

**Files:**
- Shell scripts: `kebab-case.sh` (e.g., `file-lock.sh`, `sync-to-runtime.sh`)
- JavaScript modules: `camelCase.js` (e.g., `errors.js`, `logger.js`)
- Markdown docs: `kebab-case.md` (e.g., `workers.md`, `context.md`)
- Slash commands: `kebab-case.md` (e.g., `init.md`, `build.md`)

**Directories:**
- General: `kebab-case/` (e.g., `bin/`, `tests/`, `utils/`)
- Data categories: `snake_case` (e.g., `chambers/`, `checkpoints/`)

**Functions (Bash):**
- Subcommands: `verb_noun` (e.g., `activity_log`, `spawn_log`)
- Helpers: `_internal_helper` (e.g., `_cmd_context_update`)

**Functions (JavaScript):**
- camelCase (e.g., `loadModelProfiles`, `getEffectiveModel`)

## Where to Add New Code

**New Feature:**
- Primary code: `.aether/aether-utils.sh` (for shell features)
- CLI commands: `bin/cli.js` (for JS features)
- Tests: `tests/unit/` or `tests/bash/`

**New Worker Type:**
- Definition: `.aether/workers.md`
- Caste info: Add to caste table in workers.md

**New Slash Command:**
- Claude Code: `.claude/commands/ant/<name>.md`
- OpenCode: `.opencode/commands/ant/<name>.md` (auto-generated)

**New Utility Function:**
- Shell: `.aether/utils/<category>.sh`
- JavaScript: `bin/lib/<category>.js`

## Special Directories

**`.aether/data/`:**
- Purpose: Colony state storage
- Generated: Yes (by CLI operations)
- Committed: No (never synced to hub)

**`.aether/dreams/`:**
- Purpose: Dream journal for session notes
- Generated: Yes (user-created)
- Committed: No (never synced)

**`.aether/chambers/`:**
- Purpose: Archived colony states
- Generated: Yes (on seal command)
- Committed: Optional (user choice)

**`runtime/`:**
- Purpose: Staging for npm distribution
- Generated: Yes (sync script)
- Committed: Yes (part of package)

**`.planning/`:**
- Purpose: Planning documents
- Generated: Yes (by CDS mapper)
- Committed: No (local planning)

---

*Structure analysis: 2026-02-17*
