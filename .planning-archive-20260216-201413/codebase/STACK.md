# Technology Stack

**Analysis Date:** 2026-02-13

## Languages

**Primary:**
- **JavaScript (Node.js)** - Core CLI implementation in `bin/cli.js`
- **Bash/Shell** - Utility layer in `.aether/aether-utils.sh` (source of truth), auto-synced to `runtime/`

**Secondary:**
- **Markdown** - Command definitions, agent specs, and documentation
- **JSON** - State management, configuration, and data storage

## Runtime

**Environment:**
- **Node.js** >= 16.0.0 (specified in `package.json` engines)
- No specific version lockfile - works across Node 16+

**Package Manager:**
- **npm** - Primary package manager
- Lockfile: Not present (no package-lock.json, intentionally lightweight)
- Alternative: Works with bun (`.opencode/bun.lock` present for OpenCode plugin)

## Frameworks

**Core:**
- None - This is a CLI tool with no web framework dependencies

**CLI Framework:**
- Custom Node.js CLI in `bin/cli.js` - handles install, update, version, uninstall commands

**Testing:**
- Custom Node.js test runner (`test/*.test.js`)
- Bash E2E tests (`tests/e2e/*.sh`)
- No external test framework (jest, mocha, etc.)

**Build/Dev:**
- No build step - JavaScript and Bash run directly
- ShellCheck for shell script linting (`npm run lint:shell`)

## Key Dependencies

**Critical:**
- **jq** (external) - JSON processing throughout `aether-utils.sh`
- **shellcheck** (external) - Shell script linting via `npm run lint:shell`

**OpenCode Integration:**
- **@opencode-ai/plugin** 1.1.63 - OpenCode plugin support (`.opencode/package.json`)

**Node.js Built-ins (used in `bin/cli.js`):**
- `fs` - File system operations
- `path` - Path manipulation
- `crypto` - SHA-256 hashing for file sync
- `child_process` - Git operations

## Configuration

**Environment:**
- `HOME` - Required for installation paths (Claude Code commands, hub directory)
- No `.env` files - configuration is file-based

**Build:**
- `package.json` - npm package definition, scripts, metadata
- `.npmignore` - Package distribution exclusions
- `.gitignore` - Repository exclusions (runtime state, sessions, etc.)

**OpenCode:**
- `.opencode/opencode.json` - OpenCode configuration schema
- `.opencode/package.json` - OpenCode plugin dependencies

## Platform Requirements

**Development:**
- macOS or Linux (uses bash, jq, standard Unix tools)
- Node.js >= 16.0.0
- jq (`brew install jq` on macOS)
- shellcheck (for linting: `brew install shellcheck`)
- git (for repo detection and update safety checks)

**Production:**
- Any system with Node.js >= 16.0.0 and bash
- Works with Claude Code CLI and/or OpenCode CLI
- Platform-agnostic file operations

## Project Structure

**Distribution:**
- Published as `aether-colony` npm package
- Bin entry: `aether` -> `./bin/cli.js`
- Postinstall hook: `node bin/cli.js install --quiet`

**Installed Locations:**
- Claude Code commands: `~/.claude/commands/ant/`
- Distribution hub: `~/.aether/`
- Repo-local state: `<repo>/.aether/data/`

---

*Stack analysis: 2026-02-13*
