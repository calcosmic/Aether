# External Integrations

**Analysis Date:** 2026-03-19

## APIs & External Services

**LLM Model Routing:**
- LiteLLM Proxy - Routes API calls to configured LLM providers
  - SDK/Client: Direct HTTP via curl and fetch in Node.js
  - Endpoint: `http://localhost:4000` (configurable via `ANTHROPIC_BASE_URL`)
  - Auth: Token-based via `ANTHROPIC_AUTH_TOKEN` (default: `sk-litellm-local`)
  - Health check: `GET /health` endpoint polled by `bin/lib/proxy-health.js`
  - Model list: `GET /models` endpoint (OpenAI format)
  - Purpose: Abstracts away underlying LLM provider (Anthropic Claude, other models)

**Claude Code / OpenCode Integration:**
- Claude Code and OpenCode are spawned as external processes
  - Method: `claude --cwd <project_root>` command-line invocation
  - Context: Spawned with environment variables for model routing
  - Task delivery: Via inline prompts in agent specifications and command files
  - No direct API calls - communication via environment and file system

## Data Storage

**Databases:**
- Not applicable - No database connections (relational, document, or otherwise)

**File Storage:**
- Local filesystem only
  - Primary: `.aether/data/` - Runtime state and colony memory
  - State files: `COLONY_STATE.json`, `pheromones.json`, `constraints.json`, `learning-observations.json`
  - Logs: `activity.log`, `spawn-tree.txt`, `flags.json`
  - Failure tracking: `.aether/data/midden/` - Structured failure records
  - Backups: `.aether/data/backups/` - Incremental state snapshots

**Caching:**
- Not applicable - No external caching service used
- In-memory: Command execution state cached during CLI invocation
- File-system locks: `file-lock.sh` provides distributed locking primitives (`/tmp/.aether-*.lock`)

## Authentication & Identity

**Auth Provider:**
- Custom implementation
  - No OAuth, SAML, or third-party auth service
  - Authentication is implicit: Whoever has access to the file system and can run `aether` commands
  - Token authentication for LiteLLM proxy uses simple static token (`sk-litellm-local` by default, configurable)
  - No user accounts or role-based access control - Colony operates as single logical actor

**Authorization:**
- File system permissions only
  - `.aether/data/` directory permissions determine who can access colony state
  - Atomic writes and file locking prevent concurrent modification race conditions

## Monitoring & Observability

**Error Tracking:**
- In-process only - No external error tracking service
  - Errors logged to `.aether/data/activity.log` (JSON lines format)
  - Error context: `bin/lib/errors.js` defines structured error types (AetherError, RepoError, GitError, etc.)
  - Failure records: `.aether/data/midden/midden.json` tracks failures for colony learning

**Logs:**
- File-based logging via `bin/lib/logger.js`
  - Activity log: `.aether/data/activity.log` (all operations: SPAWN, COMPLETE, ACTIVITY)
  - Spawn tree: `.aether/data/spawn-tree.txt` (hierarchical worker spawn visualization)
  - Telemetry: `.aether/data/view-state.json` (model performance, spawn metrics)
  - Rotation: Not implemented - logs grow indefinitely until manually archived

**Metrics:**
- In-memory collection via `bin/lib/telemetry.js`
  - Spawn count, model usage, caste distribution
  - Timing metrics: Task duration, phase execution time
  - Memory state: Event count, learnings count, instincts count (displayed in `/ant:status`)

## CI/CD & Deployment

**Hosting:**
- Distributed package via npm registry
  - Package: `aether-colony` on npmjs.com
  - Installation: `npm install -g aether-colony` or `npx aether-colony install`
  - No server hosting - Client-side toolkit for local Claude Code environments

**Package Distribution:**
- npm postinstall script: `bin/npx-install.js`
  - Copies slash commands to `~/.claude/commands/ant/`
  - Copies agent definitions to `~/.claude/agents/ant/`
  - Copies OpenCode commands to `~/.opencode/commands/ant/`
  - Creates hub directory: `~/.aether/`

**CI Pipeline:**
- GitHub Actions (inferred from `.github/workflows/` reference in CLAUDE.md)
- Local development testing:
  - Shell linting: `npm run lint:shell` (shellcheck)
  - Sync validation: `npm run lint:sync` (command/agent parity)
  - JSON validation: `npm run lint:json` (state file format)
  - Unit tests: `npm run test:unit` (AVA, 490+ tests)
  - Bash integration tests: `npm run test:bash` (functional shell tests)

**Release Process:**
- Manual via `npm publish`
  - Runs `prepublishOnly` hook: Executes `bin/validate-package.sh`
  - Validates package integrity before upload to npm registry

## Environment Configuration

**Required env vars:**
- `ANTHROPIC_BASE_URL` - LiteLLM proxy endpoint (default: http://localhost:4000)
- `ANTHROPIC_AUTH_TOKEN` - Proxy auth token (default: sk-litellm-local)
- `ANTHROPIC_MODEL` - LLM model identifier (caste-specific, e.g., claude-3.5-sonnet)
- `HOME` - User home directory (required for hub location)

**Optional env vars:**
- `AETHER_ROOT` - Project root override (auto-detected from `.aether/` location)
- `DATA_DIR` - State directory override (default: `$AETHER_ROOT/.aether/data`)
- `TEMP_DIR` - Temporary directory for atomic writes (default: system temp)

**Secrets location:**
- Not managed by Aether - User responsible for:
  - Setting `ANTHROPIC_AUTH_TOKEN` securely (via shell profile or environment management)
  - Protecting `.aether/data/` directory permissions
  - Securing LiteLLM proxy deployment (authentication, TLS)

## Webhooks & Callbacks

**Incoming:**
- Not applicable - No webhook endpoints

**Outgoing:**
- Git integration (optional)
  - Post-install hook creates git config (inferred from `syncStateFromPlanning` in `bin/lib/state-sync.js`)
  - Reads from `.planning/` directory structure for phase plans (inferred from clone/sync operations)
  - Commits colony state changes if git hooks are configured

**File System Events:**
- Monitoring via `bash .aether/utils/watch-spawn-tree.sh` (optional)
  - Watches `.aether/data/spawn-tree.txt` for real-time spawn visualization
  - Used with tmux (see `/ant:watch` command)

## External Tool Dependencies

**Required (Hard):**
- bash 4+ - Shell scripting execution
- node 16+ - JavaScript runtime
- curl - HTTP health checks to proxy

**Optional (Graceful Degradation):**
- jq - JSON processing (fallback XML parsing without it)
- git - Version control operations and state sync
- xmlstarlet - XML to JSON conversion (fallback to xsltproc or xml2json)
- tmux - Terminal multiplexing for `/ant:watch` live monitoring

**Conditional:**
- claude - Claude Code CLI (required if spawning workers in Claude Code)
- opencode - OpenCode CLI (required if spawning workers in OpenCode)

## Model Routing & LLM Provider Abstraction

**LiteLLM Proxy Integration:**
- Routes requests through configurable proxy (`http://localhost:4000`)
- Supports multiple LLM providers (Anthropic, OpenAI, etc.) via single proxy configuration
- Model profiles: `bin/lib/model-profiles.js` maps castes to specific models
  - Builder, Watcher, Scout, etc. each can have different model assignments
  - Override system: `aether set-model-override <caste> <model>`
  - Model verification: `/ant:verify-models` checks proxy health and model availability

**Execution Flow:**
1. Worker caste spawned with task
2. `spawn-with-model.sh` queries model profile for caste
3. Sets `ANTHROPIC_MODEL` and `ANTHROPIC_BASE_URL` in environment
4. Claude Code/OpenCode spawned with these env vars
5. LLM provider (via proxy) receives request
6. Response routed back to worker

---

*Integration audit: 2026-03-19*
