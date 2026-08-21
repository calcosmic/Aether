# External Integrations

**Analysis Date:** 2026-08-01

## APIs & External Services

**LLM Services:**
- Anthropic Claude - Core AI provider
  - SDK/Client: github.com/anthropics/anthropic-sdk-go v1.29.0 (`pkg/llm/client.go`)
  - Auth: ANTHROPIC_API_KEY (environment variable, required)
  - Models supported: claude-sonnet-4-20250514 (default), configurable via `WithModel()` option
  - Features: Message streaming, tool use, configurable token limits

**Platform APIs:**
- Codex CLI - Internal platform integration (optional)
  - Auth: CODEX_API_KEY (environment variable, optional)
  - Detection: CODEX_CLI environment flag (`cmd/codex_visuals.go`)

**Binary Distribution:**
- GitHub Releases - Downloads and updates
  - Endpoint: https://github.com/calcosmic/Aether/releases/download/
  - Supported platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64
  - Retry logic: maxRetries=3, maxRedirects=5, defaultDownloadTimeout=60s (`pkg/downloader/downloader.go`)
  - Verification: SHA-256 checksum validation

## Data Storage

**Databases:**
- SQLite 3 (embedded)
  - Connection: File-based, typically at `.aether/data/colony.db`
  - Client: Standard Go database/sql with modernc.org/sqlite driver
  - Initialization: WAL mode enabled for concurrent access
  - Stores: Colony state, skills, instincts, curation data, observations
  - Location in code: `pkg/learn/sqlite_store.go`, `pkg/learn/sqlite_schema.go`

**File Storage:**
- Local filesystem only
  - JSON files with atomic write operations (`pkg/storage/storage.go`)
  - JSONL event bus persistence (`pkg/events/bus.go`)
  - File locking for cross-process safety (`pkg/storage/lock_unix.go`, `pkg/storage/lock_windows.go`)
  - Locations: `.aether/data/` directory tree (COLONY_STATE.json, pheromones.json, instincts.json, etc.)

**Caching:**
- In-memory only
  - LRU-based wisdom cache (200 entries max) at `~/.aether/hive/wisdom.json`
  - Event bus with TTL-based pruning (configurable retention)
  - No external cache service (Redis, Memcached) used

## Authentication & Identity

**Auth Provider:**
- Custom - API key based
  - Implementation: Environment variable (ANTHROPIC_API_KEY) passed to Anthropic SDK
  - Per-request: Included in all Claude API calls via SDK
  - No OAuth, JWT, or session management

## Monitoring & Observability

**Error Tracking:**
- None - All errors are logged to stderr or captured in CLI output
- No external error tracking service (Sentry, Rollbar) integrated

**Logs:**
- Stdout/stderr - Standard streams
- JSONL event bus - Structured events with TTL stored locally (`pkg/events/bus.go`)
- No remote logging service integration

## CI/CD & Deployment

**Hosting:**
- GitHub - Source repository (github.com/calcosmic/Aether)
- GitHub Releases - Binary distribution
- Local hub directory - Cross-colony sharing and updates

**CI Pipeline:**
- GitHub Actions - Implied by .github directory structure
- Binary build: goreleaser v2 (`.goreleaser.yml`)
  - Build hook: `go mod tidy`, git diff validation, `TestDocCLIAlignment` test
  - Snapshots use AETHER_RELEASE_VERSION override or git describe

## Environment Configuration

**Required env vars:**
- ANTHROPIC_API_KEY - Claude API authentication (failure without it: `llm.NewClient()` returns error in `pkg/llm/client.go`)

**Optional env vars:**
- CODEX_CLI - Set if running on Codex platform
- CODEX_API_KEY - Codex platform authentication
- AETHER_RELEASE_VERSION - Override version at build time (used by goreleaser snapshot)

**Secrets location:**
- Environment variables only
- `.env` files: Not committed, user-configured per machine
- No vault integration (HashiCorp Vault, AWS Secrets Manager)

## Webhooks & Callbacks

**Incoming:**
- None detected - Aether is command-driven, not event-driven from external sources

**Outgoing:**
- None - No outbound webhooks or callbacks to external systems

## Network & Connectivity

**HTTP Client:**
- Go standard library net/http (`pkg/downloader/downloader.go`)
- Retry logic for GitHub release downloads (maxRetries=3)
- Redirect handling (maxRedirects=5)
- Timeout: defaultDownloadTimeout=60s

**WebSocket:**
- github.com/gorilla/websocket v1.5.3 - Used for streaming responses (`pkg/llm/streaming.go`)
- Streaming support with event handling

## Multi-Tenant & Cross-Colony

**Hub (Local):**
- `~/.aether/` - User-level shared hub for all colonies on same machine
- Contains: System skills, commands, agents, wisdom, eternal memory
- File-based distribution via `aether publish` and `aether update`

**Registry:**
- `~/.aether/registry/` - Tracks all local colonies with domain tags
- File-based, not a networked service

**Hive Brain:**
- `~/.aether/hive/wisdom.json` - Cross-colony wisdom with LRU eviction (200 entries max)
- Domain-scoped retrieval for colony-specific relevance
- No central server; all colonies on same machine share the hub directory

## Platform Wrappers

**Claude Code:**
- Commands: 60 markdown files in `.claude/commands/ant/`
- Agents: 27 definitions in `.claude/agents/ant/`
- No API calls; wrappers orchestrate Go runtime

**OpenCode:**
- Commands: Mirrored from `.claude/` into `.opencode/commands/ant/`
- Dependencies: @opencode-ai/plugin v1.1.63, @kilocode/plugin v7.2.22
- Agents: Separate definitions in `.opencode/agents/`

**Codex:**
- Agents: TOML definitions in `.codex/agents/`
- Native CLI support with no wrapper indirection
- CODEX.md rules file for Codex-specific behavior

---

*Integration audit: 2026-08-01*
