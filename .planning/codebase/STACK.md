# Technology Stack

**Analysis Date:** 2026-03-19

## Languages

**Primary:**
- Bash - Core system orchestration and state management (150+ subcommands in `aether-utils.sh`)
- JavaScript/Node.js - CLI orchestration and model routing
- Markdown - Command and agent specifications

**Secondary:**
- XML - Data exchange format for pheromones, wisdom, and registry

## Runtime

**Environment:**
- Node.js >= 16.0.0

**Package Manager:**
- npm - Distributed via `npm install -g aether-colony` or `npx aether-colony install`
- Lockfile: `package-lock.json` (present)

## Frameworks

**Core:**
- Commander.js ^12.1.0 - CLI argument parsing and command routing (`bin/cli.js`)
- No web framework - This is a system toolkit, not a web application

**Testing:**
- AVA ^6.0.0 - JavaScript unit testing framework (490+ passing tests)
- Sinon ^19.0.5 - Test doubles and mocking
- ProxyQuire ^2.1.3 - Module mocking for Node.js

**Build/Dev:**
- Shellcheck - Shell script linting (`npm run lint:shell`)
- Generate-commands.sh - Generates sync checks between Claude and OpenCode commands

## Key Dependencies

**Critical:**
- js-yaml ^4.1.0 - YAML parsing (used in command/agent specs)
- picocolors ^1.1.1 - Terminal color formatting (16+ color palette for agent castes)

**Infrastructure:**
- LiteLLM proxy - Model routing via `ANTHROPIC_BASE_URL` environment variable (http://localhost:4000 default)
- Git - Version control integration (required for archaeology, state sync via git hooks)
- curl - HTTP health checks to LiteLLM proxy endpoint
- jq - JSON processing and validation (state file operations)
- xmlstarlet - XML processing for pheromone/wisdom exchange
- bash - Shell scripting runtime (POSIX-compliant)

## Configuration

**Environment:**
- `ANTHROPIC_BASE_URL` - LiteLLM proxy endpoint (default: http://localhost:4000)
- `ANTHROPIC_AUTH_TOKEN` - Proxy authentication (default: sk-litellm-local)
- `ANTHROPIC_MODEL` - Model assignment for worker spawns
- `HOME` - User home directory (required)
- `AETHER_ROOT` - Project root (auto-detected)

**Build:**
- `package.json` - NPM manifest with test, lint, and install scripts
- `.npmignore` - Excludes local data (`.aether/data/`, `.aether/dreams/`) from package distribution
- `bin/validate-package.sh` - Runs during `npm install` and `prepublishOnly` to verify integrity

**Files:**
- `bin/cli.js` - Entry point for `aether` command (v1.1.11)
- `bin/lib/` - 16 utility modules for state management, model profiles, file locking, telemetry, logging

## Platform Requirements

**Development:**
- Node.js 16+ (runtime)
- Bash 4+ (shell scripting)
- curl (HTTP requests)
- jq (JSON processing)
- git (version control, optional but recommended)
- xmlstarlet or xsltproc (optional, XML processing)
- shellcheck (optional, linting)

**Production (LiteLLM Proxy):**
- LiteLLM proxy server running on localhost:4000
- Environment variables set: ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN, ANTHROPIC_MODEL

**Distribution:**
- npm registry (package published as `aether-colony`)
- Global install location: `~/.npm` or equivalent
- Global commands copied to: `~/.claude/commands/ant/`, `~/.claude/agents/ant/`, `~/.opencode/commands/ant/`

## Distribution & Packaging

**NPM Package Contents:**
- `bin/` - CLI executables
- `.claude/commands/ant/` - 36 slash commands for Claude Code
- `.claude/agents/ant/` - 22 agent definitions
- `.aether/` - Source of truth (utility scripts, templates, exchange modules)
- `.opencode/commands/ant/` - OpenCode command specs
- `.opencode/agents/` - OpenCode worker definitions
- Documentation and README

**Excluded from Package:**
- `.aether/data/` - Runtime state (COLONY_STATE.json, pheromones.json, activity logs)
- `.aether/dreams/` - User notes and session logs
- `.aether/checkpoints/` - Session checkpoints
- `node_modules/` - Dependencies

**Post-Install Setup:**
- `bin/npx-install.js` - Runs via postinstall hook to copy commands/agents to global locations
- `bin/cli.js install --quiet` - Sets up Claude Code and OpenCode integration points
- Hub creation: `~/.aether/` system directory established for multi-colony support

---

*Stack analysis: 2026-03-19*
