# Codebase Structure

**Analysis Date:** 2026-02-17

## Directory Layout

```
/Users/callumcowie/repos/Aether/
├── .aether/                    # SOURCE OF TRUTH - system files
│   ├── aether-utils.sh         # Core utility functions (~3,700 lines)
│   ├── workers.md              # Worker/caste definitions
│   ├── data/                   # LOCAL - colony state
│   ├── utils/                  # Helper scripts
│   ├── exchange/               # XML exchange utilities
│   ├── commands/               # Command definitions
│   ├── docs/                   # Distributed documentation
│   ├── chambers/               # Archived colonies
│   ├── checkpoints/            # Session recovery points
│   └── dreams/                 # LOCAL - session notes
├── bin/                        # CLI entry point
│   ├── cli.js                  # Main CLI (~79KB)
│   ├── lib/                    # JavaScript modules
│   └── sync-to-runtime.sh      # Sync script
├── .claude/                    # Claude Code integration
│   ├── commands/ant/            # Slash commands
│   ├── hooks/                  # Claude hooks
│   └── agents/                 # Agent definitions
├── .opencode/                  # OpenCode integration
│   ├── commands/ant/            # Slash commands
│   └── agents/                 # Agent definitions
├── runtime/                    # STAGING - auto-generated (DO NOT EDIT)
├── src/                        # Minimal source
├── tests/                      # Test suites
└── docs/                       # Development documentation
```

## Directory Purposes

**`.aether/` (Source of Truth):**
- Purpose: Primary location for all system files
- Contains: workers.md, aether-utils.sh, utils/, docs/
- Key files: `workers.md`, `aether-utils.sh`, `data/COLONY_STATE.json`, `data/pheromones.json`
- **NEVER edit runtime/ directly - edit .aether/ and run sync**

**`bin/` (CLI):**
- Purpose: Node.js CLI entry point
- Contains: cli.js (main), lib/ (modules)
- Key files: `cli.js`, `lib/errors.js`, `lib/update-transaction.js`, `lib/state-sync.js`

**`.claude/commands/ant/` (Claude Commands):**
- Purpose: Slash command definitions for Claude Code
- Contains: Markdown files for each command
- Key files: `build.md`, `colonize.md`, `swarm.md`, `oracle.md`

**`.opencode/commands/ant/` (OpenCode Commands):**
- Purpose: Slash command definitions for OpenCode
- Contains: Markdown files for each command (duplicated from .claude/)
- Key files: `build.md`, `colonize.md`, `swarm.md`, `oracle.md`

**`.aether/utils/` (Shell Utilities):**
- Purpose: Modular helper scripts
- Contains: atomic-write.sh, chamber-utils.sh, file-lock.sh, xml-utils.sh, spawn-tree.sh
- Key files: `xml-utils.sh` (87KB, largest), `chamber-utils.sh`, `spawn-tree.sh`

**`.aether/data/` (LOCAL - Never Sync):**
- Purpose: Persistent colony state
- Contains: COLONY_STATE.json, pheromones.json, activity.log, locks/
- **This directory is LOCAL - never synced to hub**

**`.aether/exchange/` (XML Exchange):**
- Purpose: Structured data import/export
- Contains: pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh
- Used for: External system integration

## Key File Locations

**Entry Points:**
- `/Users/callumcowie/repos/Aether/bin/cli.js` - Main CLI entry
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - Shell utility entry

**Configuration:**
- `/Users/callumcowie/repos/Aether/package.json` - npm package config
- `/Users/callumcowie/repos/Aether/.aether/model-profiles.yaml` - Model routing config

**Core Logic:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - ~3,700 lines of bash
- `/Users/callumcowie/repos/Aether/bin/cli.js` - ~79KB of JavaScript
- `/Users/callumcowie/repos/Aether/.aether/workers.md` - Worker definitions

**State:**
- `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json` - Current colony state
- `/Users/callumcowie/repos/Aether/.aether/data/pheromones.json` - Pheromone signals
- `/Users/callumcowie/repos/Aether/.aether/data/checkpoint-allowlist.json` - Safe files for git stash

**Testing:**
- `/Users/callumcowie/repos/Aether/tests/bash/` - Shell script tests
- `/Users/callumcowie/repos/Aether/tests/unit/` - JavaScript unit tests

## Naming Conventions

**Files:**
- Shell scripts: `kebab-case.sh` (e.g., `file-lock.sh`, `spawn-tree.sh`)
- JavaScript modules: `camelCase.js` (e.g., `errors.js`, `update-transaction.js`)
- Markdown docs: `kebab-case.md` (e.g., `workers.md`, `pheromones.md`)
- Slash commands: `kebab-case.md` (e.g., `build.md`, `swarm.md`)

**Directories:**
- General: `kebab-case/` (e.g., `utils/`, `commands/`)
- Data: `snake_case/` (e.g., `checkpoints/`, `chambers/`)
- Configuration: `lowercase/` (e.g., `data/`, `docs/`)

**Functions (aether-utils.sh):**
- Pattern: `verb_noun` (e.g., `read_colony_state`, `spawn_log`, `activity_log`)
- Case: lowercase with underscores

## Where to Add New Code

**New Feature:**
- Primary code: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (if shell) or `/Users/callumcowie/repos/Aether/bin/cli.js` (if Node.js)
- Tests: `/Users/callumcowie/repos/Aether/tests/bash/` or `/Users/callumcowie/repos/Aether/tests/unit/`

**New Command:**
- Implementation: `/Users/callumcowie/repos/Aether/.claude/commands/ant/<name>.md` AND `/Users/callumcowie/repos/Aether/.opencode/commands/ant/<name>.md`
- Must keep both in sync (use `npm run lint:sync` to verify)

**New Utility:**
- Shell: `/Users/callumcowie/repos/Aether/.aether/utils/<name>.sh`
- JavaScript: `/Users/callumcowie/repos/Aether/bin/lib/<name>.js`

**Documentation:**
- System docs (distributed): `/Users/callumcowie/repos/Aether/.aether/docs/<name>.md`
- Dev docs (local only): `/Users/callumcowie/repos/Aether/docs/<name>.md`

## Special Directories

**`.aether/data/`:**
- Purpose: Colony state and pheromones
- Generated: Yes (at runtime)
- Committed: No (in .gitignore)

**`.aether/checkpoints/`:**
- Purpose: Session recovery via git stash
- Generated: Yes (on checkpoint commands)
- Committed: No

**`.aether/chambers/`:**
- Purpose: Archived colonies
- Generated: On seal/entomb commands
- Committed: Optional

**`runtime/`:**
- Purpose: Staging for npm package
- Generated: Yes (via sync-to-runtime.sh)
- Committed: No (auto-generated from .aether/)

**`.planning/codebase/`:**
- Purpose: Architecture documentation
- Generated: By GSD mapping commands
- Committed: Yes (documentation)

---

*Structure analysis: 2026-02-17*
