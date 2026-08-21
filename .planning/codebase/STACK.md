# Technology Stack

**Analysis Date:** 2026-08-01

## Languages

**Primary:**
- Go 1.26.1 - Main runtime, CLI, and core system (`cmd/`, `pkg/`)

**Secondary:**
- TypeScript/JavaScript - Wrapper platform support
  - Claude Code commands (`.claude/commands/`)
  - OpenCode commands (`.opencode/commands/`)
  - Bootstrap npm installer (`npm/`)

## Runtime

**Environment:**
- Go 1.26.1 (binary CLI executable)
- Node.js >= 18 (optional, for npm installer)

**Package Manager:**
- Go modules (go.mod/go.sum)
- npm (for platform wrappers and bootstrap)

## Frameworks

**Core:**
- Cobra v1.10.2 - CLI framework for command structure and routing (`cmd/root.go`, `cmd/` subcommands)

**LLM Integration:**
- Anthropic SDK for Go v1.29.0 - Claude API client (`pkg/llm/client.go`)
  - Supports streaming and tool use
  - Configurable model selection and token limits

**Data & Persistence:**
- SQLite 3 (via modernc.org/sqlite v1.50.0) - Structured data storage (`pkg/learn/sqlite_store.go`)
  - Uses WAL mode for concurrent access
  - Stores colony state, skills, instincts, and curation data
- JSON files with atomic operations - State management (`pkg/storage/`)
  - Cross-process safe via file locking (`pkg/storage/lock.go`)

**Event System:**
- Custom in-memory event bus - Pub/sub messaging (`pkg/events/bus.go`)
  - JSONL persistence with TTL-based pruning
  - Pattern-based subscriptions

**Build & Release:**
- goreleaser v2 - Binary cross-compilation and release (`.goreleaser.yml`)
  - Builds for: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64
  - CGO disabled for portability

## Key Dependencies

**Critical:**
- github.com/anthropics/anthropic-sdk-go v1.29.0 - LLM API integration (required for runtime operation)
- modernc.org/sqlite v1.50.0 - Database for colony state, skills, and learning
- github.com/spf13/cobra v1.10.2 - CLI framework (core to command dispatch)

**Data Handling:**
- github.com/tidwall/gjson v1.18.0 - JSON path queries
- github.com/tidwall/sjson v1.2.5 - JSON mutations
- gopkg.in/yaml.v3 v3.0.1 - YAML parsing (agent specs, commands, configs)
- github.com/BurntSushi/toml v1.5.0 - TOML parsing

**Development & Tooling:**
- github.com/gorilla/websocket v1.5.3 - Streaming support
- github.com/jedib0t/go-pretty/v6 v6.7.8 - Terminal tables and formatting
- github.com/schollz/progressbar/v3 v3.19.0 - Progress indicators
- github.com/invopop/jsonschema v0.14.0 - JSON Schema generation
- golang.org/x/sync v0.20.0 - Synchronization primitives

## Configuration

**Environment:**
- ANTHROPIC_API_KEY - Claude API authentication (required)
- CODEX_CLI - Codex platform detection flag (optional)
- CODEX_API_KEY - Codex API authentication (optional)
- AETHER_RELEASE_VERSION - Override version at build time (optional, used by goreleaser)

**Build:**
- `.goreleaser.yml` - Release automation config
- `Makefile` - Development build targets (build, test, lint, clean, install)
- `go.mod` / `go.sum` - Go module versioning

## Platform Requirements

**Development:**
- Go 1.26.1 or later
- CGO is optional (disabled by default in releases for portability)
- Make (for development targets)
- Git (for version detection from tags)

**Production:**
- Linux, macOS, Windows (x86_64 or ARM64)
- No external runtime required (statically compiled binary)
- SQLite (embedded via modernc.org/sqlite)

## Version Strategy

Version resolution priority (in `cmd/root.go`):
1. ldflags Version (set by goreleaser for releases)
2. `.aether/version.json` in repo (for source checkouts)
3. Nearest git tag (for dev builds)
4. Installed hub version
5. Fallback "0.0.0-dev"

---

*Stack analysis: 2026-08-01*
