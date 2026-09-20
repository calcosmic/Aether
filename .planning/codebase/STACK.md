---
last_mapped_commit: 92252d01
---

# Technology Stack

**Analysis Date:** 2026-08-22

## Languages

**Primary:**
- Go 1.26.5 - Core runtime (`cmd/`, `pkg/`); handles all state, CLI, orchestration, verification

**Secondary:**
- TypeScript 5.9.3 - TypeScript host coordinator (`pkg/codex/`, `.aether/ts-host/src/`); wave dispatch, manifestation, provider adaptation
- TypeScript 5.9.3 - Ceremony narrator (`pkg/codex/`, `.aether/ts/narrator.ts`); SSE formatting for platform output
- Node.js 18+ (narrator), 20+ (ts-host) - Runtime for TypeScript tooling and npm bootstrap

## Runtime

**Environment:**
- Go 1.26.5 binary (`aether`) - Single-platform-agnostic executable for Linux, macOS, Windows; distributed via GitHub Releases and npm
- Node.js >= 18 for narrator, >= 20 for ts-host coordinator

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- npm 10+ (`npm/package.json`, `.aether/ts-host/package.json`, `.aether/ts/package.json`)
- Lockfiles: `go.sum`, `.aether/ts-host/package-lock.json`, `.aether/ts/package-lock.json`, `npm/package-lock.json`

## Frameworks

**Core:**
- Cobra 1.10.2 - CLI command framework and subcommand routing
- Custom Aether colony orchestration (`pkg/agent/`, `pkg/codex/`, `pkg/colony/`) - 27 worker agent caste system with wave-based dispatch

**Supporting:**
- Gorilla WebSocket 1.5.3 - WebSocket support (infrastructure, not actively used for external calls)
- go-pretty 6.7.8 - Terminal table formatting for output display

**Build/Dev:**
- goreleaser 2.x - Cross-platform binary packaging and release automation (`.goreleaser.yml`)
- TypeScript 5.9.3 - Type checking and compilation (`tsconfig.json`, `tsconfig.build.json`)
- Make - Local development build orchestration (`Makefile`)

## Key Dependencies

**Critical:**
- `github.com/anthropics/anthropic-sdk-go` v1.29.0 - Anthropic Claude API client for LLM inference (`pkg/llm/`); enables worker reasoning and tool use
- `modernc.org/sqlite` v1.50.0 - Embedded database for learning/instinct storage (`pkg/learn/sqlite_store.go`); stores experiences with trust scoring and lifecycle metadata
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing for configuration, pheromones, and template loading
- `github.com/tidwall/sjson` v1.2.5 - JSON modification without full unmarshaling (for atomic updates)
- `github.com/tidwall/gjson` v1.18.0 - JSON querying and extraction

**Infrastructure:**
- `github.com/spf13/cobra` v1.10.2 - CLI framework and subcommand routing
- `github.com/spf13/pflag` v1.0.9 - Flag parsing
- `github.com/schollz/progressbar` v3.19.0 - Download progress display
- `github.com/invopop/jsonschema` v0.14.0 - JSON schema generation and validation (`pkg/codex/`)
- `github.com/santhosh-tekuri/jsonschema` v6.0.2 - JSON schema validation
- `github.com/BurntSushi/toml` v1.5.0 - TOML parsing (Codex agent definitions)
- `golang.org/x/sync` v0.20.0 - Concurrency primitives (WaitGroup, Mutex)
- `golang.org/x/sys` v0.42.0 - System-level OS operations
- `golang.org/x/text` v0.27.0 - Unicode text handling
- `github.com/google/uuid` v1.6.0 - UUID generation for identifiers

**Display/Formatting:**
- `github.com/jedib0t/go-pretty` v6.7.8 - Table and formatted output rendering

## Configuration

**Environment:**
- `ANTHROPIC_API_KEY` - Anthropic Claude API authentication (required for `pkg/llm/client.go`; defaults to SDK's built-in env resolution)
- `AETHER_WORKER_PLATFORM` - Worker dispatch platform override (`claude`, `opencode`, `codex`); affects `pkg/codex/platform_dispatch.go`
- `AETHER_RELEASE_VERSION` - Override version during goreleaser builds (`.goreleaser.yml`)
- `AETHER_RELEASE_BASE_URL` - Override GitHub Releases URL for binary downloads (used by npm bootstrap, defaults to `https://github.com/calcosmic/Aether/releases/download/`)

**Build:**
- `.goreleaser.yml` - Release pipeline; builds Linux/macOS/Windows (amd64, arm64)
- `.aether/version.json` - Source-of-truth version for releases and npm parity
- `Makefile` - Local build targets: `make build`, `make test`, `make lint`, `make install`
- `tsconfig.json`, `tsconfig.build.json` - TypeScript compilation settings (ts-host, narrator)

## Platform Requirements

**Development:**
- Go 1.26.5+
- Node.js 18+ (narrator), 20+ (ts-host)
- macOS, Linux, or Windows (tested on all three via goreleaser)
- Standard Unix tools for scripting (bash, grep, sed for `.github/workflows/`)

**Production:**
- Single `aether` binary (self-contained, no runtime dependencies)
- Optional: Node.js 18+ if using npm bootstrap (`npx --yes aether-colony@latest`)
- GitHub Releases or npm registry for binary distribution
- ANTHROPIC_API_KEY environment variable for LLM calls

**Storage:**
- Local filesystem for JSON-based colony state (`.aether/data/`)
- Optional: SQLite database for learning/memory (`pkg/learn/sqlite_store.go`)
- File locking via platform-specific mechanisms (`pkg/storage/lock_*.go`)

---

*Stack analysis: 2026-08-22*
