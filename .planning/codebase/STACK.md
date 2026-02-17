# Technology Stack

**Analysis Date:** 2026-02-17

## Languages

**Primary:**
- JavaScript (Node.js) - CLI implementation, npm package
- Bash - Core utilities, colony orchestration

**Secondary:**
- YAML - Configuration files (`model-profiles.yaml`)
- JSON - Data storage (pheromones, colony state)

## Runtime

**Environment:**
- Node.js >=16.0.0
- Bash (POSIX-compatible, macOS/Linux)

**Package Manager:**
- npm
- Lockfile: `package-lock.json` (present in node_modules)

## Frameworks

**Core:**
- Commander.js ^12.1.0 - CLI argument parsing
- js-yaml ^4.1.0 - YAML configuration parsing
- picocolors ^1.1.1 - Terminal colors

**Testing:**
- AVA ^6.0.0 - JavaScript unit tests
- Sinon ^19.0.5 - Test mocking
- proxyquire ^2.1.3 - Module mocking for tests
- shellcheck - Bash linting

**Build/Dev:**
- npm scripts - Build and test automation
- Git hooks - Pre-commit validation

## Key Dependencies

**Critical:**
- `commander` - CLI framework for `aether` command
- `js-yaml` - Parse `model-profiles.yaml` for caste-model routing
- `picocolors` - Colored terminal output for logs

**Development:**
- `ava` - Test runner for `npm test`
- `sinon` - Mocking framework
- `proxyquire` - Dependency injection for testing

## Required External Tools

**Must be installed:**
- `git` - Version control integration
- `jq` - JSON processing (required for pheromone/state handling)
- `claude` or `opencode` - AI agent CLI (spawned as workers)

**Optional (gracefully degraded):**
- `xmlstarlet` - XML pheromone processing
- `xmllint` - XML validation
- `xsltproc` - XSL transformations
- `fswatch` or `inotifywait` - File watching (swarm display)
- `curl` - HTTP health checks

## Configuration

**Environment:**
- No `.env` file required
- Configuration via YAML files:
  - `.aether/model-profiles.yaml` - Model routing
  - `.aether/data/pheromones.json` - Colony signals
  - `.aether/data/COLONY_STATE.json` - Session state

**Build:**
- `bin/sync-to-runtime.sh` - Syncs `.aether/` to `runtime/`
- `bin/cli.js` - Main CLI entry point

## Platform Requirements

**Development:**
- Node.js >=16.0.0
- npm
- git
- jq
- Claude Code or OpenCode CLI in PATH

**Production:**
- npm package distribution via `npm install -g aether-colony`
- Runtime requires same tools as development

---

*Stack analysis: 2026-02-17*
