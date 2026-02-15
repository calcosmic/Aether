# Codebase Structure

**Analysis Date:** 2026-02-13

## Directory Layout

```
Aether/
├── bin/                    # CLI entry point
│   ├── cli.js              # Node.js CLI (install, update, version)
│   └── generate-commands.sh # Command sync verification tool
├── .claude/                # Claude Code configuration
│   └── commands/ant/       # Claude Code slash commands (28 files)
├── .opencode/              # OpenCode configuration
│   ├── commands/ant/       # OpenCode slash commands (28 files, mirror of .claude)
│   └── agents/             # OpenCode agent definitions (4 agents)
├── runtime/                # Distribution runtime files
│   ├── aether-utils.sh     # Core utility layer (59 subcommands)
│   ├── workers.md          # Worker caste specifications
│   ├── utils/              # Utility scripts (atomic-write, file-lock, etc.)
│   └── docs/               # Reference documentation
├── .aether/                # Repo-local runtime (copied from runtime/)
│   ├── aether-utils.sh     # Local copy of utilities
│   ├── data/               # Per-project state
│   └── oracle/             # Oracle research system
├── tests/                  # Test suites
│   └── e2e/                # End-to-end tests
├── test/                   # Additional tests
│   └── *.test.js           # Jest tests
├── src/                    # Source templates
│   └── commands/           # Command generation templates
├── .planning/              # Planning documents
│   └── codebase/           # Codebase analysis docs (this file)
├── package.json            # npm package definition
├── README.md               # User documentation
├── CHANGELOG.md            # Version history
└── TO-DOS.md               # Project TODOs
```

## Directory Purposes

**bin/:**
- Purpose: Executable entry points for CLI
- Contains: JavaScript and shell scripts
- Key files: `cli.js` (main CLI), `generate-commands.sh` (sync checker)

**.claude/commands/ant/:**
- Purpose: Claude Code slash command definitions
- Contains: Markdown files with command instructions
- Key files: `init.md`, `build.md`, `plan.md`, `continue.md`, `status.md`

**.opencode/commands/ant/:**
- Purpose: OpenCode slash command definitions (mirror of .claude)
- Contains: Identical markdown files to .claude/commands/ant/
- Key files: Same as .claude (maintained in sync)

**.opencode/agents/:**
- Purpose: OpenCode agent personas
- Contains: Agent definition markdown files
- Key files: `aether-queen.md`, `aether-builder.md`, `aether-scout.md`, `aether-watcher.md`

**runtime/:**
- Purpose: Distribution files copied to ~/.aether/system/ and .aether/
- Contains: Shell utilities, worker specs, discipline docs
- Key files: `aether-utils.sh`, `workers.md`, `QUEEN_ANT_ARCHITECTURE.md`

**runtime/utils/:**
- Purpose: Low-level utility scripts
- Contains: Shell scripts for atomic operations
- Key files: `atomic-write.sh`, `file-lock.sh`, `colorize-log.sh`, `watch-spawn-tree.sh`

**runtime/docs/:**
- Purpose: Reference documentation for colony concepts
- Contains: Markdown documentation files
- Key files: `constraints.md`, `pheromones.md`, `pathogen-schema.md`, `progressive-disclosure.md`

**.aether/:**
- Purpose: Repo-local runtime (bootstrapped from runtime/)
- Contains: Same structure as runtime/, plus data/
- Key files: `aether-utils.sh`, `workers.md`, `data/*`

**.aether/data/:**
- Purpose: Per-project colony state
- Contains: JSON state files, logs, archives
- Key files: `COLONY_STATE.json`, `constraints.json`, `flags.json`, `activity.log`, `spawn-tree.txt`

**tests/e2e/:**
- Purpose: End-to-end test suite
- Contains: Shell test scripts
- Key files: `run-all.sh`, `test-install.sh`, `test-update.sh`

## Key File Locations

**Entry Points:**
- `bin/cli.js`: npm CLI entry point (install, update, version)
- `.claude/commands/ant/init.md`: Colony initialization command
- `.claude/commands/ant/build.md`: Phase execution command
- `.claude/commands/ant/plan.md`: Planning command

**Configuration:**
- `package.json`: npm package config, scripts, dependencies
- `.gitignore`: Git ignore patterns
- `.npmignore`: npm package files whitelist

**Core Logic:**
- `.aether/aether-utils.sh`: 59 subcommands for colony operations (source of truth)
- `.aether/workers.md`: Worker caste definitions and spawn protocol (source of truth)
- `runtime/` contains auto-populated copies for npm packaging

**Testing:**
- `tests/e2e/run-all.sh`: E2E test runner
- `tests/e2e/test-install.sh`: Installation tests
- `tests/e2e/test-update.sh`: Update system tests

**State Files:**
- `.aether/data/COLONY_STATE.json`: Colony state (goal, phase, memory, errors)
- `.aether/data/constraints.json`: Focus areas and avoid patterns
- `.aether/data/flags.json`: Blockers, issues, notes

## Naming Conventions

**Files:**
- Shell scripts: `kebab-case.sh` (e.g., `aether-utils.sh`, `atomic-write.sh`)
- Markdown docs: `UPPERCASE.md` or `lowercase.md` depending on context
- JSON state: `UPPERCASE.json` (e.g., `COLONY_STATE.json`) or `lowercase.json` (e.g., `flags.json`)
- Commands: `lowercase.md` (e.g., `init.md`, `build.md`)

**Directories:**
- Runtime: `lowercase` (e.g., `runtime`, `utils`, `docs`)
- State: `lowercase` (e.g., `data`, `archive`)
- Config: `lowercase` (e.g., `commands`, `agents`)

## Where to Add New Code

**New Command:**
- Primary code: `.claude/commands/ant/{command-name}.md`
- Mirror copy: `.opencode/commands/ant/{command-name}.md`
- Run sync check: `npm run lint:sync` or `./bin/generate-commands.sh check`

**New Worker Caste:**
- Definition: `.aether/workers.md` (add to Worker Roles section) - source of truth
- Emoji: `.aether/aether-utils.sh` (add to `get_caste_emoji` function) - source of truth
- Name generation: `.aether/aether-utils.sh` (add to `generate-ant-name` case) - source of truth
- Note: The sync script auto-populates runtime/ from .aether/ during npm install

**New Utility Subcommand:**
- Implementation: `.aether/aether-utils.sh` (add to case statement) - source of truth
- The sync script auto-populates `runtime/aether-utils.sh` during npm install
- Help output: `.aether/aether-utils.sh` (add to help command list)

**New State File:**
- Schema: Define in appropriate command (init.md, etc.)
- Location: `.aether/data/{filename}.json`
- Validation: Add check to `validate-state` subcommand

**New Test:**
- E2E test: `tests/e2e/test-{feature}.sh`
- Unit test: `test/{feature}.test.js`

**Utilities:**
- Shared helpers: `runtime/utils/{utility-name}.sh`
- Must source in `aether-utils.sh` if needed

## Special Directories

**.aether/data/archive/:**
- Purpose: Archived session state
- Generated: Yes (when sessions complete or pause)
- Committed: No (runtime state)

**.aether/dreams/:**
- Purpose: Dream session output files
- Generated: Yes (when /ant:dream runs)
- Committed: No (runtime output)

**.aether/oracle/archive/:**
- Purpose: Archived oracle research sessions
- Generated: Yes (when oracle runs)
- Committed: No (runtime output)

**.opencode/node_modules/:**
- Purpose: OpenCode dependencies (zod)
- Generated: Yes (npm install in .opencode/)
- Committed: No (dependencies)

**~/.aether/:**
- Purpose: Global distribution hub (outside repo)
- Contains: `system/`, `commands/`, `agents/`, `registry.json`, `version.json`
- Generated: Yes (by `aether install`)
- Committed: No (user-specific)

---

*Structure analysis: 2026-02-13*
