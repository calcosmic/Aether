---
last_mapped_commit: 92252d01
---

# External Integrations

**Analysis Date:** 2026-08-22

## APIs & External Services

**LLM Provider:**
- Anthropic Claude API - Worker reasoning and tool use for all 27 agent castes
  - SDK/Client: `github.com/anthropics/anthropic-sdk-go` v1.29.0 (`pkg/llm/client.go`, `pkg/llm/tools.go`)
  - Auth: `ANTHROPIC_API_KEY` environment variable (or Anthropic SDK default resolution)
  - Models: claude-sonnet-4-20250514 (default configurable via `WithModel()`)
  - Token limits: 4096 output tokens (configurable via `WithMaxTokens()`)
  - Usage: Worker dispatch, verification, tool calling for build/continue/seal phases

**Platform Dispatchers (No External API):**
- Claude Code - Local agent execution via `.claude/agents/` and `.claude/commands/`
- OpenCode - Local agent execution via `.opencode/agents/` and `.opencode/commands/` (requires local OpenCode server running)
- Codex CLI - Local agent execution via `.codex/agents/` (TOML format, native CLI)
  - Selection via `AETHER_WORKER_PLATFORM` env var; no external API calls

## Data Storage

**Local Filesystem:**
- Colony state: `.aether/data/COLONY_STATE.json` - Atomic JSON operations via `pkg/storage/`
- Pheromone signals: `.aether/data/pheromones.json` - Plain-text signal ledger
- Learning observations: `.aether/data/learning/` - JSONL event bus (per-phase)
- Constraints: `.aether/data/constraints.json` - Focus/redirect directives
- Midden (failure log): `.aether/data/midden/midden.json` - Failure tracking
- Survey data: `.aether/data/survey/` - Territory analysis results
- Session data: `.aether/data/session.json` - Recovery checkpoint
- File locking: `.aether/locks/` - Platform-specific lock files (Unix fcntl, Windows file locking)

**SQLite (Optional, Opt-In):**
- Learning database: `~/.aether/colonies/[colony-id]/memories.db` (WAL mode, single-writer per instance)
  - Stores: observations with evidence, classification, confidence, phase/caste context
  - Implementation: `pkg/learn/sqlite_store.go` (`SQLiteColonyStore`)
  - Migrations: Automatic on first open; uses `database/sql` standard lib
  - Purpose: Persistent cross-session instinct storage with trust scoring

**Hive Brain (Cross-Colony):**
- Shared wisdom: `~/.aether/hive/wisdom.json` - 200-entry LRU cache of generalized instincts across projects
  - Domains: Scoped by project domain tags (e.g., "web", "api", "mobile")
  - Promotion: High-confidence instincts (>= 0.8) auto-promoted at seal time
  - Retrieval: Domain-filtered injection into worker briefs

**Eternal Memory (Legacy Fallback):**
- High-value signals: `~/.aether/eternal/memory.json` - Superseded by Hive Brain, kept for backward compatibility

## Authentication & Identity

**Auth Provider:**
- Custom local auth model - No external identity provider; all identity is local file-based
  - ANTHROPIC_API_KEY - Required for LLM calls (user provides at setup)
  - GitHub token - `GITHUB_TOKEN` used in CI/CD workflows (via `secrets.GITHUB_TOKEN` in Actions) for release publishing

**Secrets Storage:**
- Environment variables only (no credential files committed)
- `.env` files ignored in development; never committed to git

## Monitoring & Observability

**Error Tracking:**
- None - Errors are logged locally to stdout/stderr and recorded in midden.json
- No external error tracking service integration

**Logs:**
- Approach: Structured event bus (`pkg/events/`) with JSON lines format (JSONL) to `.aether/data/events/`
- TTL cleanup: Events auto-expire (24h default, configurable per event type)
- Midden review: `/ant-midden-review` for human-readable failure analysis

**Build Output:**
- Terminal formatting via `pkg/terminal/` (tables, progress bars, colored output)
- Narration: TypeScript narrator (`pkg/codex/`, `.aether/ts/narrator.ts`) formats SSE events for platform display
- Visuals dump: `aether visuals-dump --json` exports caste identities and colors

## CI/CD & Deployment

**Hosting:**
- GitHub Releases (`github.com/calcosmic/Aether/releases/`) - Primary binary distribution
- npm registry (`https://www.npmjs.com/package/aether-colony`) - Bootstrap wrapper distribution

**CI Pipeline:**
- GitHub Actions (`.github/workflows/`)
  - `ci.yml` - Runs on every PR and push to main; tests Go, TypeScript, npm, goreleaser
  - `release.yml` - Manual triggered; builds release binaries, publishes to GitHub + npm
- Test matrix: race detection, linting (go vet), unit tests, type checking (TypeScript), npm audit

**Release Pipeline:**
- Goreleaser 2.x (`.goreleaser.yml`) - Cross-platform builds (Linux, macOS, Windows; amd64, arm64)
- Before hooks: `go mod tidy`, version alignment checks, doc CLI alignment tests
- After: SHA-256 checksums, archive creation, version tagging
- npm sync: `npm/package.json` version must match `.aether/version.json` for stable releases

**Deployment:**
- User's machine: `aether publish --channel stable` publishes to `~/.aether/system/` (hub)
- Target repos: `aether update --force` syncs from hub (no binary re-downloaded unless `--download-binary` flag)

## Environment Configuration

**Required env vars:**
- `ANTHROPIC_API_KEY` - Anthropic API authentication (required at runtime for any LLM worker dispatch)

**Optional env vars:**
- `AETHER_WORKER_PLATFORM` - Worker dispatch target (`claude`, `opencode`, `codex`); auto-detected if unset
- `AETHER_RELEASE_VERSION` - Override version during build (goreleaser only)
- `AETHER_RELEASE_BASE_URL` - Override GitHub Releases download URL (npm bootstrap)
- `AETHER_RELEASE_ACCEPTANCE_DIR` - Test-only; sets staged release directory
- `AETHER_HIVE_POLICY` - Control cross-colony wisdom sharing (`promote` default, `read`, `off`)
- `AETHER_UPDATE_SNAPSHOTS` - Testing flag for snapshot refreshes

**Secrets location:**
- ANTHROPIC_API_KEY: User's shell profile or local `.env` (never committed)
- GITHUB_TOKEN: GitHub Actions secrets (`.github/workflows/` read at runtime)

## Webhooks & Callbacks

**Incoming:**
- None - Aether is purely pull-based (user runs commands, workers execute)

**Outgoing:**
- GitHub Releases API - Publish binaries at release time (`pkg/downloader/` reverse flow used in CI)
- npm registry - Push new versions at stable release time (GitHub Actions workflow step)

---

*Integration audit: 2026-08-22*
