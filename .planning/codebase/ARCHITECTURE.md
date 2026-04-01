# Architecture

**Analysis Date:** 2026-04-01

## Pattern Overview

**Overall:** Dual-architecture system undergoing shell-to-Go conversion. The production system is a monolithic shell-based CLI dispatcher with domain modules, orchestrated by LLM agent definitions (Markdown prompt specs). The Go rewrite is in early scaffolding with type definitions and a storage layer ported first.

**Key Characteristics:**
- Shell architecture uses a single dispatcher (`aether-utils.sh`) that sources ~42 module files providing ~130+ subcommands
- Commands are orchestrated by LLM slash commands (Markdown files in `.claude/commands/ant/`) that invoke shell subcommands via Bash tool
- 24 agent definitions (Markdown files in `.claude/agents/ant/`) define worker personas and behaviors injected into LLM context
- All shell subcommands output JSON to stdout, errors as JSON to stderr, with deterministic behavior
- Go architecture follows standard Go layout (`cmd/`, `pkg/`, `internal/`) with one-to-one package mapping from shell modules
- JSON file-based persistence (`COLONY_STATE.json`, `pheromones.json`, etc.) -- no database

## Layers

### Shell Architecture (Production)

**Dispatcher Layer:**
- Purpose: Single entry point routing all subcommands via `case "$cmd" in` dispatch
- Location: `.aether/aether-utils.sh`
- Contains: Main `case` statement (~5,640 lines), startup initialization, error constants, feature detection, cleanup traps
- Depends on: All domain modules (sourced at startup), `jq`, `bash` 4+
- Used by: All LLM slash commands via `bash .aether/aether-utils.sh <subcommand> [args]`

**Domain Modules (9 core):**
- Purpose: Business logic for colony operations, extracted from the monolith in Phase 13
- Location: `.aether/utils/`
- Contains: Domain-specific functions grouped by concern
- Depends on: Infrastructure modules, `jq`, shell builtins
- Used by: Dispatcher (sourced at startup)
- Key modules:
  - `.aether/utils/state-api.sh` -- Colony state read/write/mutate/migrate (~17,267 bytes, 7 functions)
  - `.aether/utils/pheromone.sh` -- Pheromone signal CRUD, display, export, decay (~139,004 bytes, largest module)
  - `.aether/utils/learning.sh` -- Wisdom pipeline: observe, promote, capture (~79,701 bytes)
  - `.aether/utils/queen.sh` -- QUEEN.md wisdom read/write/promote (~75,025 bytes)
  - `.aether/utils/swarm.sh` -- Parallel worker spawning and management (~41,241 bytes)
  - `.aether/utils/flag.sh` -- Colony flag creation and management (~10,823 bytes)
  - `.aether/utils/spawn.sh` -- Worker spawn lifecycle management (~10,674 bytes)
  - `.aether/utils/session.sh` -- Session freshness and recovery (~32,827 bytes)
  - `.aether/utils/suggest.sh` -- Auto-suggestion analysis and approval (~28,489 bytes)

**Structural Learning Stack:**
- Purpose: Memory consolidation pipeline with trust scoring, graph relationships, curation
- Location: `.aether/utils/`
- Contains: Trust scoring, event bus, instinct store, graph layer, consolidation
- Depends on: Domain modules, `jq`, `bash`
- Used by: Wisdom pipeline, seal lifecycle, curation ants
- Key modules:
  - `.aether/utils/trust-scoring.sh` -- Weighted trust scoring (40/35/25, 7 tiers)
  - `.aether/utils/event-bus.sh` -- JSONL pub/sub event bus with TTL
  - `.aether/utils/instinct-store.sh` -- Instinct CRUD with provenance tracking
  - `.aether/utils/graph.sh` -- jq-based graph for instinct relationships
  - `.aether/utils/consolidation.sh` -- Phase-end memory consolidation
  - `.aether/utils/consolidation-seal.sh` -- Full seal-time consolidation

**Curation Ants (8 + orchestrator):**
- Purpose: Automated curation pipeline for instinct quality management
- Location: `.aether/utils/curation-ants/`
- Contains: Specialized ants for archiving, quality evaluation, broadcasting, cleanup, indexing, healing, recording, corruption detection
- Depends on: Structural learning stack modules
- Used by: `curation-run` subcommand, seal lifecycle
- Modules: `orchestrator.sh`, `archivist.sh`, `critic.sh`, `herald.sh`, `janitor.sh`, `librarian.sh`, `nurse.sh`, `scribe.sh`, `sentinel.sh`

**Infrastructure Modules:**
- Purpose: Cross-cutting concerns used by all domain modules
- Location: `.aether/utils/`
- Contains: File locking, atomic writes, error handling, hive brain, midden, skills
- Key modules:
  - `.aether/utils/file-lock.sh` -- Per-path file locking with stale detection
  - `.aether/utils/atomic-write.sh` -- Temp file + rename atomic writes with JSON validation
  - `.aether/utils/error-handler.sh` -- Structured JSON error output with error codes
  - `.aether/utils/hive.sh` -- Cross-colony wisdom storage and retrieval
  - `.aether/utils/midden.sh` -- Failure tracking and review
  - `.aether/utils/skills.sh` -- Skill indexing, matching, and injection

**XML Layer:**
- Purpose: XML-based data exchange and transformation
- Location: `.aether/utils/xml-*.sh`, `.aether/exchange/`, `.aether/schemas/`
- Contains: Core XML ops, query, compose, convert, utils; exchange modules for pheromones/wisdom; XSD schemas
- Key modules:
  - `.aether/utils/xml-core.sh` -- XML parsing primitives
  - `.aether/utils/xml-query.sh` -- XPath-like querying
  - `.aether/utils/xml-compose.sh` -- XML assembly
  - `.aether/utils/xml-convert.sh` -- Format conversion
  - `.aether/exchange/pheromone-xml.sh` -- Pheromone XML export/import
  - `.aether/exchange/wisdom-xml.sh` -- Wisdom XML export/import
  - `.aether/schemas/` -- XSD schemas (pheromone.xsd, queen-wisdom.xsd, colony-registry.xsd, prompt.xsd, worker-priming.xsd, aether-types.xsd)

**LLM Orchestration Layer:**
- Purpose: LLM agent definitions and slash commands that drive the colony
- Location: `.claude/commands/ant/`, `.claude/agents/ant/`
- Contains: 45 slash command specs (Markdown), 24 agent definitions (Markdown with frontmatter)
- Depends on: Shell dispatcher (invoked via Bash tool), `colony-prime` subcommand
- Used by: Claude Code runtime directly
- Key agents: Builder, Watcher, Queen, Scout, Route-Setter, Architect (see CLAUDE.md for full list)
- Key commands: `init.md`, `build.md`, `continue.md`, `run.md`, `seal.md`, `plan.md`

**Node.js CLI Layer:**
- Purpose: npm package distribution, installation, update management
- Location: `bin/cli.js`, `bin/lib/`
- Contains: Commander-based CLI for `aether` command, installation/setup logic, state sync, update transactions
- Depends on: `commander`, `js-yaml`, `picocolors`
- Used by: `npm install -g .`, `aether update`, `aether install`

### Go Architecture (Scaffolding)

**Entry Point:**
- Purpose: CLI binary entry point
- Location: `cmd/aether/main.go`
- Contains: Empty `main()` -- placeholder for CLI wiring
- Status: Stub

**Core Types Package:**
- Purpose: Colony state type definitions matching COLONY_STATE.json schema
- Location: `pkg/colony/colony.go`
- Contains: `ColonyState`, `Plan`, `Phase`, `Task`, `Memory`, `Instinct`, `Signal`, `Graveyard`, state constants
- Depends on: `time`, `fmt`
- Used by: All Go packages
- Design: All types use pointer fields for nullable JSON values; designed for exact round-trip compatibility with shell JSON files

**State Machine Package:**
- Purpose: Colony lifecycle state transitions and phase advancement
- Location: `pkg/colony/state_machine.go`
- Contains: `Transition()`, `AdvancePhase()`, `legalTransitions` map
- Depends on: `pkg/colony/colony.go`
- Used by: Future orchestration logic
- States: READY -> EXECUTING -> BUILT -> READY (cycle), or -> COMPLETED (terminal)

**Storage Package:**
- Purpose: Atomic JSON/JSONL file operations with concurrent safety
- Location: `pkg/storage/storage.go`
- Contains: `Store` type with `AtomicWrite`, `SaveJSON`, `LoadJSON`, `AppendJSONL`, `ReadJSONL`
- Depends on: `encoding/json`, `os`, `sync`
- Used by: All Go packages that need file I/O
- Design: Per-path `sync.RWMutex` via `sync.Map`, temp-file + `os.Rename` pattern, JSON validation on write for `.json` files, 2-space indentation matching shell output
- Ported from: `.aether/utils/atomic-write.sh` and `.aether/utils/file-lock.sh`

**Planned Packages (stubs only):**
- `pkg/agent/` -- Worker pool management (package doc: "spawn, lifecycle, task distribution")
- `pkg/events/` -- JSONL event bus (package doc: "publishing, subscription, TTL-based cleanup")
- `pkg/graph/` -- Knowledge graph layer (package doc: "instinct relationships, dependency tracking, graph-based queries")
- `pkg/llm/` -- LLM provider client abstraction (package doc: "Anthropic SDK integration for colony worker interactions")
- `pkg/memory/` -- Wisdom pipeline (package doc: "trust scoring, observation capture, instinct promotion, memory consolidation")
- `internal/config/` -- Configuration loading and validation
- `internal/testing/` -- Shared test helpers and fixtures

## Data Flow

### Colony Lifecycle Flow:

1. User runs `/ant:init "goal"` (Claude Code slash command)
2. `init.md` command spec instructs LLM to invoke shell subcommands:
   - `bash .aether/aether-utils.sh colony-init` -- creates `COLONY_STATE.json` from template
   - `bash .aether/aether-utils.sh queen-init` -- creates `QUEEN.md`
   - `bash .aether/aether-utils.sh session-init` -- creates `session.json`
3. User runs `/ant:plan` to generate phases
4. `plan.md` invokes `bash .aether/aether-utils.sh plan-generate` and writes phases to `COLONY_STATE.json`
5. User runs `/ant:build N` to execute a phase
6. `build.md` orchestrator loads split playbooks from `.aether/docs/command-playbooks/build-*.md`
7. Playbooks instruct LLM to spawn worker agents (Builder, Watcher, Scout) via Task tool
8. Each agent gets context assembled by `colony-prime` subcommand (wisdom, pheromones, skills, learnings)
9. Workers invoke shell subcommands to read/write state, log activity, emit pheromones
10. User runs `/ant:continue` to verify, extract learnings, advance phase
11. `continue.md` orchestrator loads playbooks from `.aether/docs/command-playbooks/continue-*.md`
12. Learnings flow through wisdom pipeline: observe -> trust-score -> instinct -> QUEEN.md -> hive

### State Mutation Flow:

1. Shell subcommand calls `_state_mutate` (from `state-api.sh`)
2. `_state_mutate` reads `COLONY_STATE.json`, applies `jq` transformation, validates result
3. Writes back via `atomic_write` (temp file + rename + JSON validation)
4. File locking via `acquire_lock`/`release_lock` prevents concurrent corruption
5. Activity logged via `activity-log` subcommand

### Wisdom Pipeline Flow:

1. `memory-capture "learning"` records observation to `learning-observations.json`
2. `trust-score-compute` assigns weighted score (recency 40%, source 35%, evidence 25%)
3. `event-bus-publish` publishes scored event to JSONL event bus
4. After threshold (2 observations), `learning-promote-auto` triggers instinct creation
5. `instinct-create` stores in `instinct-store.sh` + `COLONY_STATE.json`
6. `queen-promote` writes high-confidence patterns to QUEEN.md
7. `colony-prime prompt_section` injects wisdom + instincts into worker prompts
8. At seal, `hive-promote` abstracts instincts to cross-colony wisdom

## Key Abstractions

**Colony State:**
- Purpose: Single source of truth for colony lifecycle state
- Examples: `pkg/colony/colony.go` (Go types), `.aether/data/COLONY_STATE.json` (runtime JSON)
- Pattern: JSON file with nested structure: state, plan (phases -> tasks), memory (learnings, decisions, instincts), errors, signals, graveyards

**Pheromone Signals:**
- Purpose: User-colony communication channel for guiding worker behavior
- Examples: `.aether/data/pheromones.json`, `.aether/utils/pheromone.sh`
- Pattern: FOCUS (attract), REDIRECT (repel, hard constraint), FEEDBACK (calibrate). Each signal has TTL, strength decay, content deduplication via SHA-256 hash

**Agent Definitions:**
- Purpose: LLM worker persona and behavior specifications
- Examples: `.claude/agents/ant/aether-builder.md`, `.claude/agents/ant/aether-queen.md`
- Pattern: Markdown files with YAML frontmatter (model slot, description) and structured sections (role, instructions, pheromone_protocol, tools)

**Skills:**
- Purpose: Reusable behavior modules injected into worker context
- Examples: `.aether/skills/colony/build-discipline/SKILL.md`, `.aether/skills/domain/golang/SKILL.md`
- Pattern: Each skill has frontmatter (name, category, detect patterns, roles) and Markdown content. Matched by `skill-match` based on worker role + pheromones + codebase detection. Top 3 colony + top 3 domain injected per worker with 8K character budget

**Hive Brain:**
- Purpose: Cross-colony wisdom sharing at user level (~/.aether/)
- Examples: `~/.aether/hive/wisdom.json`, `.aether/utils/hive.sh`
- Pattern: 200-entry cap with LRU eviction, domain-scoped retrieval, multi-repo confidence boosting

**Atomic Store:**
- Purpose: Safe concurrent JSON file operations
- Examples: `pkg/storage/storage.go` (Go), `.aether/utils/atomic-write.sh` + `.aether/utils/file-lock.sh` (Shell)
- Pattern: Per-path locking (RWMutex in Go, flock-based in shell), temp file + rename for atomicity, JSON validation before commit

## Entry Points

**Shell CLI (production):**
- Location: `.aether/aether-utils.sh`
- Triggers: `bash .aether/aether-utils.sh <subcommand> [args...]`
- Responsibilities: Dispatches ~130+ subcommands across all domain modules. Sourced by all LLM slash commands.

**Node.js CLI (distribution):**
- Location: `bin/cli.js`
- Triggers: `aether <command>` or `npx aether-colony`
- Responsibilities: Package installation, `aether update`, `aether install`, file sync, setup

**Claude Code Slash Commands:**
- Location: `.claude/commands/ant/*.md` (45 files)
- Triggers: `/ant:init`, `/ant:build`, `/ant:continue`, etc.
- Responsibilities: Orchestrate LLM behavior for colony operations. Instruct LLM to invoke shell subcommands.

**OpenCode Slash Commands:**
- Location: `.opencode/commands/ant/*.md` (45 files)
- Triggers: Same command names for OpenCode provider
- Responsibilities: Structural parity with Claude commands, content adapted for OpenCode format

**Go CLI (scaffolding):**
- Location: `cmd/aether/main.go`
- Triggers: `go run ./cmd/aether`
- Responsibilities: Currently empty placeholder. Will become the Go-native CLI.

**npm install hook:**
- Location: `bin/cli.js` (postinstall script)
- Triggers: `npm install -g .`
- Responsibilities: `node bin/cli.js install --quiet` -- copies .aether/ to target repo

## Error Handling

**Shell Strategy:** Structured JSON errors with typed error codes

**Patterns:**
- `json_err()` outputs `{"ok":false,"error":{"code":"E_SOMETHING","message":"..."}}` to stderr, exits 1
- `json_ok()` outputs `{"ok":true,"result":...}` to stdout, exits 0
- Error codes defined as constants: `E_UNKNOWN`, `E_FILE_NOT_FOUND`, `E_JSON_INVALID`, `E_LOCK_FAILED`, `E_LOCK_STALE`, `E_GIT_ERROR`, `E_VALIDATION_FAILED`, etc.
- `set -euo pipefail` in dispatcher ensures unhandled errors propagate
- ERR trap with `error_handler` provides line number and command context
- Feature detection with `feature_disable` for graceful degradation when tools (jq, git) are missing
- `# SUPPRESS:OK` comments mark intentional error suppression patterns

**Go Strategy:** Standard Go error handling with sentinel errors

**Patterns:**
- `fmt.Errorf("%w: ...")` for error wrapping
- `errors.Is()` for sentinel error checking (e.g., `ErrInvalidTransition`)
- `errors.New()` for sentinel values
- No panic/recover in production code

## Cross-Cutting Concerns

**Logging:**
- Shell: `activity-log` subcommand writes to `.aether/data/activity.log` with JSON entries
- Go: Not yet implemented

**Validation:**
- Shell: `jq` validation on all JSON writes (atomic-write.sh), `# SUPPRESS:OK` annotations for intentional suppressions
- Go: `json.Valid()` check in `storage.AtomicWrite` for `.json` files

**Authentication:**
- Not applicable (local CLI tool, no auth)

**Concurrency:**
- Shell: File-based locking via `file-lock.sh` (flock with stale detection), atomic writes via temp+rename
- Go: Per-path `sync.RWMutex` via `sync.Map` in `storage.Store`

**Configuration:**
- Shell: Environment variables (`AETHER_ROOT`, `DATA_DIR`, `TEMP_DIR`), fallback defaults in dispatcher
- Go: `internal/config` package (stub)

**Agent Parity:**
- `.claude/agents/ant/*.md` is canonical
- `.aether/agents-claude/*.md` is byte-identical mirror for npm packaging
- `.opencode/agents/*.md` maintains structural parity (same filenames/count)
- `npm run lint:sync` enforces all three stay in sync

---

*Architecture analysis: 2026-04-01*
