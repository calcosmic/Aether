# Technology Stack

**Analysis Date:** 2026-04-01

## Languages

**Primary:**
- Bash 3.2+ - Core colony runtime: dispatcher (`aether-utils.sh`, 5,642 lines), 50+ utility modules in `.aether/utils/`, 92 bash test files, agent definitions (markdown), slash command specs (markdown)
- Go 1.26.1 - Emerging rewrite target: module skeleton with types and storage layer
- JavaScript (Node.js v25.8.2) - CLI tooling and npm distribution: `bin/cli.js` (2,223 lines), `bin/lib/` (14 modules, 5,443 lines total), test framework glue

**Secondary:**
- Markdown - Agent definitions (24 files), slash commands (45 Claude + 45 OpenCode), skills definitions (28), templates (12), documentation
- JSON - Colony state, pheromones, observations, instincts, config files
- JSONL - Event bus persistence, learning observations
- XML - Cross-colony signal exchange (`.aether/exchange/`)
- jq - JSON processing in bash (referenced throughout utils)
- XSLT - Queen-to-markdown transformation (`queen-to-md.xsl`)

## Runtime

**Environment:**
- macOS (darwin/arm64) -- primary development target
- Bash 3.2+ (macOS default) -- core runtime for shell layer
- Node.js >= 16.0.0 (current: v25.8.2) -- npm CLI tooling
- Go 1.26.1 -- skeleton module, not yet functional

**Package Managers:**
- npm -- primary package manager for `aether-colony` distribution
  - Lockfile: `package-lock.json` present
- Go modules -- `go.mod` with single dependency (no external Go deps yet)

## Frameworks

**Core:**
- No external application framework. The system is self-contained: bash scripts orchestrated by a dispatcher pattern, distributed as an npm package.

**Testing:**
- AVA ^6.0.0 -- JavaScript unit test runner for `bin/lib/` modules
  - Config: inline in `package.json` (`ava.files`, `ava.timeout`)
  - Runs: `tests/unit/**/*.test.js`
- Bash test harness -- custom `bats`-style test framework
  - 92 bash test files in `tests/bash/`
  - Shared helpers in `tests/bash/test-helpers.sh`
  - Runner: `bash tests/bash/test-aether-utils.sh` (dispatches sub-tests)
- Go testing -- standard `testing` package
  - `golang_test.go` (root) -- compilation smoke test
  - `pkg/colony/colony_test.go` -- state machine and JSON round-trip
  - `pkg/storage/storage_test.go` -- atomic file operations

**Build/Dev:**
- npm scripts -- primary build orchestration
- `shellcheck` -- bash linting (error severity)
- `bin/generate-commands.sh` -- command/agent sync verification between Claude and OpenCode
- `bin/validate-package.sh` -- pre-publish validation

## Key Dependencies

**Critical (npm):**
- `commander` ^12.1.0 -- CLI argument parsing for `bin/cli.js`
- `js-yaml` ^4.1.0 -- YAML parsing (used in config/skills)
- `picocolors` ^1.1.1 -- terminal color output

**Critical (bash):**
- `jq` -- JSON processing (assumed available on system)
- `sha256sum` / `shasum` -- content hashing (pheromone dedup, trust scoring)
- `git` -- version control integration, archaeology, worktree management
- `tmux` -- live monitoring (`/ant:watch`)

**Dev (npm):**
- `ava` ^6.0.0 -- test runner
- `proxyquire` ^2.1.3 -- module mocking for tests
- `sinon` ^19.0.5 -- spies/stubs/mocks for tests

**No Go external dependencies yet.** The `go.mod` declares only the local module.

## Configuration

**Environment:**
- `HOME` -- hub directory location (`~/.aether/`)
- `AETHER_ROOT` -- optional override for colony root directory
- `DATA_DIR` -- optional override for data directory
- `npm` `engines` field requires `node >= 16.0.0`

**Build:**
- `package.json` -- npm package definition, scripts, dependencies
- `.npmignore` -- controls what gets published (excludes `.aether/data/`, `.aether/dreams/`, `.planning/`, logs, dev files)
- `bin/validate-package.sh` -- pre-publish checklist (required files, excluded directories)
- `bin/generate-commands.sh` -- verifies Claude/OpenCode command parity

**Go:**
- `go.mod` -- module declaration (`github.com/aether-colony/aether`, go 1.26.1)
- No `go.sum` yet (no external dependencies)

## Platform Requirements

**Development:**
- macOS or Linux (bash, jq, sha256sum, git)
- Node.js >= 16.0.0
- Go 1.26.1 (for Go skeleton work)
- `shellcheck` (recommended for bash linting)
- `tmux` (for `/ant:watch`)

**Production (npm distribution):**
- `npm install -g aether-colony` installs globally
- CLI entry: `aether` command (via `bin/cli.js`)
- NPX entry: `npx aether-colony` (via `bin/npx-entry.js`)
- Files installed: `bin/`, `.claude/commands/ant/`, `.claude/agents/ant/`, `.opencode/commands/ant/`, `.opencode/agents/`, `.aether/`, `README.md`, `LICENSE`, `CHANGELOG.md`

**Distribution targets:**
- Claude Code -- `.claude/commands/ant/*.md` (45 slash commands), `.claude/agents/ant/*.md` (24 agents)
- OpenCode -- `.opencode/commands/ant/*.md` (45 commands), `.opencode/agents/*.md` (24 agents)
- Both share `.aether/` runtime (shell scripts, skills, templates, docs)

## Codebase Size

| Component | Files | Approx Lines |
|-----------|-------|-------------|
| Shell dispatcher | 1 | 5,642 |
| Shell utils | 50 | ~20,900 |
| Shell curation ants | 9 | ~2,000 |
| Bash tests | 92 | ~40,900 |
| Node.js CLI + lib | 15 | ~7,700 |
| Node.js unit tests | 41 | ~14,500 |
| Node.js integration tests | 23 | ~11,800 |
| Node.js e2e tests | ~30 | ~5,000 |
| Go source | 11 | ~370 (types + storage only) |
| Go tests | 2 | ~1,170 |
| Markdown (agents, commands, skills, docs) | ~140+ | -- |
| Templates (JSON, jq, md) | 12 | -- |

---

*Stack analysis: 2026-04-01*
