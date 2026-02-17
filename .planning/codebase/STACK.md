# Technology Stack

**Analysis Date:** 2026-02-17

## Languages

**Primary:**
- JavaScript (Node.js) - CLI tool implementation, package management
- Bash - Shell scripts for worker orchestration, colony management

**Secondary:**
- XML - Data exchange format (pheromones, registry, worker definitions)
- YAML - Configuration files (model-profiles.yaml, COLONY_STATE.json)

## Runtime

**Environment:**
- Node.js >= 16.0.0 (from `package.json` engines field)
- Bash 4.0+ (modern bash features used)

**Package Manager:**
- npm (Node.js)
- Lockfile: `package-lock.json` (present)

## Frameworks

**Core:**
- Commander.js ^12.1.0 - CLI argument parsing
- js-yaml ^4.1.0 - YAML parsing
- picocolors ^1.1.1 - Terminal color output

**Testing:**
- AVA ^6.0.0 - Unit test runner
- Sinon ^19.0.5 - Mocking/stubbing
- Proxyquire ^2.1.3 - Module mocking

**Build/Dev:**
- npm scripts - Build and test automation
- shellcheck - Shell script linting

## Key Dependencies

**Critical:**
- `commander` ^12.1.0 - CLI framework for `bin/cli.js`
- `js-yaml` ^4.1.0 - Parse model-profiles.yaml and COLONY_STATE.json
- `picocolors` ^1.1.1 - Colored terminal output in logger

**Development:**
- `ava` ^6.0.0 - Test runner for `tests/unit/`
- `sinon` ^19.0.5 - Test spies and stubs
- `proxyquire` ^2.1.3 - Dependency injection for testing

## Shell Script Dependencies

**Required:**
- `git` - Version control integration (checked at runtime in `.aether/aether-utils.sh:52`)
- `jq` - JSON processing (checked at runtime in `.aether/aether-utils.sh:55`)

**Optional (gracefully disabled if missing):**
- `xmllint` / `xmlstarlet` - XML validation (`.aether/utils/xml-utils.sh:49-53`)
- `xsltproc` - XSLT processing (`.aether/utils/xml-utils.sh:57`)
- `xml2json` - XML to JSON conversion (`.aether/utils/xml-utils.sh:61`)
- `md5sum` / `md5` - Checksums (`.aether/utils/xml-utils.sh:1440-1477`)
- `fswatch` / `inotifywait` - File system monitoring (`.aether/utils/watch-spawn-tree.sh:239-243`)
- `sync` - File synchronization (`.aether/utils/atomic-write)

## Configuration

**Environment:**
- Environment variables: `.sh:87`HOME`, `USERPROFILE`, `NO_COLOR`, `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`, `ANTHROPIC_BASE_URL`, `WORKER_NAME`, `CASTE`
- Config files:
  - `.aether/model-profiles.yaml` - Model routing configuration
  - `.aether/data/COLONY_STATE.json` - Colony state
  - `.aether/data/constraints.json` - Constraint definitions

**Build:**
- `package.json` - npm configuration with preinstall/postinstall hooks
- `bin/sync-to-runtime.sh` - Syncs `.aether/` to `runtime/` on npm install

## Platform Requirements

**Development:**
- Node.js >= 16.0.0
- npm
- git
- jq (for JSON processing)
- Bash-compatible shell

**Production:**
- Node.js >= 16.0.0 (runtime only)
- npm for installation (or direct binary distribution)
- Target machines: macOS, Linux (bash required)

---

*Stack analysis: 2026-02-17*
