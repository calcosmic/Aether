# External Integrations

**Analysis Date:** 2026-04-01

## LLM Providers

**Anthropic (Claude Code):**
- Primary target platform for colony orchestration
- Integration: 24 agent definitions in `.claude/agents/ant/*.md` define worker personas
- 45 slash commands in `.claude/commands/ant/*.md` define user-facing commands
- Agent definitions reference `prompt_section`, `pheromone_protocol`, and `tools` configurations
- Colony-prime assembles context and injects into worker prompts via markdown templates

**OpenCode:**
- Secondary target platform
- 24 agent definitions in `.opencode/agents/*.md` (structural parity with Claude)
- 45 commands in `.opencode/commands/ant/*.md`
- Config: `.opencode/opencode.json` (schema reference only)
- OpenCode-specific rules in `.opencode/OPENCODE.md`

**LLM Abstraction (Go skeleton):**
- `pkg/llm/llm.go` -- package docstring declares "Anthropic SDK integration for colony worker interactions"
- Not yet implemented (stub only)

## Data Storage

**Colony State (local filesystem):**
- Primary: `.aether/data/COLONY_STATE.json` -- full colony state (goal, phases, tasks, memory, signals, errors)
- Backup: `.aether/data/COLONY_STATE.json.bak`
- JSON format, 2-space indentation, trailing newline
- Go types in `pkg/colony/colony.go` provide exact round-trip compatibility

**Pheromone Signals (local filesystem):**
- `.aether/data/pheromones.json` -- active FOCUS/REDIRECT/FEEDBACK signals
- Content deduplication via SHA-256 hashing
- Prompt injection sanitization (XML tags, angle brackets, shell patterns blocked)
- 500-character content cap

**Learning Observations (local filesystem):**
- `.aether/data/learning-observations.json` -- captured observations
- Rotating backups: `.learning-observations.json.bak.{1,2,3}`

**Event Bus (local filesystem):**
- JSONL format (`.aether/data/` event files)
- Pub/sub with TTL-based cleanup
- Go implementation in `pkg/events/events.go` (stub)
- Go storage layer supports `AppendJSONL`/`ReadJSONL` in `pkg/storage/storage.go`

**Hive Brain (cross-colony, user-level):**
- `~/.aether/hive/wisdom.json` -- 200-entry cap, LRU eviction
- Hub-level file locking prevents concurrent write corruption
- Domain-scoped retrieval for cross-colony wisdom

**Eternal Memory (legacy fallback):**
- `~/.aether/eternal/memory.json` -- high-value signals from expired pheromones

**Midden (failure tracking):**
- `.aether/data/midden/midden.json` -- failure records with acknowledgment tracking

**Session Data:**
- `.aether/data/session.json` -- current session state
- `.aether/data/pending-decisions.json` -- queued decisions

**Other State Files:**
- `.aether/data/constraints.json` -- focus areas and constraints (legacy)
- `.aether/data/queen-wisdom.json` -- cached wisdom
- `.aether/data/last-build-claims.json` -- build verification state
- `.aether/data/last-build-result.json` -- build outcome
- `.aether/data/phase-research/` -- per-phase research data
- `.aether/data/survey/` -- territory survey results
- `.aether/data/watch/` -- live monitoring state
- `.aether/data/spawn-tree.txt` -- worker spawn tree
- `.aether/data/rolling-summary.log` -- rolling session summary
- `.aether/data/activity.log` -- activity tracking
- `.aether/data/errors.log` -- error log

**Colony Registry (user-level):**
- `~/.aether/registry.json` -- tracks all repos using Aether with domain tags and active status

**Go Storage Abstraction:**
- `pkg/storage/storage.go` -- atomic JSON writes (temp file + rename), per-path RWMutex, JSONL append
- Mirrors bash patterns from `atomic-write.sh` and `file-lock.sh`
- Designed for exact compatibility with existing `.aether/data/` file formats

## File Storage

**Local filesystem only.** No cloud storage integrations.
- Colony data in `.aether/data/`
- Session notes in `.aether/dreams/`
- Oracle research in `.aether/oracle/`
- Archived colonies in `.aether/chambers/`

## Caching

**Skills Index Cache:**
- `skill-index` builds a cached skills index for performance
- `skill-cache-rebuild` forces index rebuild

**Colony-Prime Context Cache:**
- Token budget system (8K normal, 4K compact) with trim priority ordering

**Spawn Tree Archive:**
- `.aether/data/spawn-tree-archive/` -- historical spawn trees

## Authentication & Identity

**No external auth provider.** The system operates entirely on local filesystem permissions.
- User identity derived from `HOME` environment variable
- Hub directory at `~/.aether/` serves as user-level identity boundary
- No API keys, tokens, or credentials in the codebase
- User preferences stored in `~/.aether/QUEEN.md`

## Cross-Colony Exchange

**XML Signal Exchange:**
- `/ant:export-signals` -- exports pheromone signals to XML
- `/ant:import-signals` -- imports pheromone signals from XML
- Implementation: `.aether/exchange/pheromone-xml.sh`, `wisdom-xml.sh`, `registry-xml.sh`
- XML utilities: `xml-core.sh`, `xml-query.sh`, `xml-compose.sh`, `xml-convert.sh`, `xml-utils.sh`
- XSLT: `queen-to-md.xsl` for Queen wisdom transformation

## CI/CD & Deployment

**npm Publishing:**
- `npm publish` with `prepublishOnly` hook running `validate-package.sh`
- Distributed via npm registry as `aether-colony` package
- `npm install -g .` for local development testing (runs `validate-package.sh` then `setupHub()`)

**Validation Pipeline:**
```bash
npm run lint           # shellcheck + JSON validation + command sync check
npm test               # unit tests (AVA)
npm run test:bash      # bash integration tests
npm run test:intelligence  # intelligence pipeline tests
npm run lint:sync      # verify Claude/OpenCode command and agent parity
```

**Git Integration:**
- Archaeology agent reads git history
- Worktree management via `.aether/utils/worktree.sh`
- Clash detection via `.aether/utils/clash-detect.sh`
- Merge driver for lockfiles: `.aether/utils/merge-driver-lockfile.sh`

**No CI/CD pipeline detected.** No GitHub Actions, Travis CI, or similar configuration files.

## Error Handling

**Structured Error System (Node.js):**
- `bin/lib/errors.js` -- error class hierarchy (AetherError, HubError, RepoError, GitError, ValidationError, FileSystemError, ConfigurationError)
- Structured JSON error output to stderr
- Exit codes mapped to error types via `getExitCode()`

**Structured Error System (Bash):**
- `.aether/utils/error-handler.sh` -- error constants (E_UNKNOWN, E_HUB_NOT_FOUND, E_REPO_NOT_INITIALIZED, etc.)
- `trap` handler in dispatcher for line-level error context
- JSON error messages to stderr

**State Guard (Node.js):**
- `bin/lib/state-guard.js` -- enforces "Iron Law" (phase advancement requires fresh verification)
- Idempotency checks, file locking, structured error codes

## Monitoring & Observability

**Built-in monitoring (no external services):**
- `/ant:status` -- colony dashboard with memory health table
- `/ant:memory-details` -- detailed drill-down view
- `/ant:watch` -- live tmux monitoring of worker processes
- `/ant:history` -- browse colony events
- Activity logging to `.aether/data/activity.log`
- Error logging to `.aether/data/errors.log`
- Spawn logging via `bin/lib/spawn-logger.js`

**No external error tracking, APM, or log aggregation services.**

## Skills System

**Domain Skills (18):**
- django, docker, golang, graphql, html-css, nextjs, nodejs, postgresql, prisma, python, rails, react, rest-api, svelte, tailwind, testing, typescript, vue

**Colony Skills (10):**
- build-discipline, colony-interaction, colony-lifecycle, colony-visuals, context-management, error-presentation, pheromone-protocol, pheromone-visibility, state-safety, worker-priming

**Custom Skills:**
- Users can create skills in `~/.aether/skills/domain/`
- `/ant:skill-create "<topic>"` generates skills via Oracle research

## Environment Configuration

**Required env vars:**
- `HOME` -- hub directory location (mandatory)

**Optional env vars:**
- `AETHER_ROOT` -- override colony root directory
- `DATA_DIR` -- override data directory

**No secrets, API keys, or credential files in the codebase.**

## Webhooks & Callbacks

**None.** The system operates entirely through CLI commands and local file I/O.

---

*Integration audit: 2026-04-01*
