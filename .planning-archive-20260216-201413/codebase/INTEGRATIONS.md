# External Integrations

**Analysis Date:** 2026-02-13

## APIs & External Services

**None** - Aether is a local CLI tool with no external API calls or cloud services.

## AI Editor Integrations

**Claude Code (Anthropic):**
- Command directory: `~/.claude/commands/ant/`
- 28 slash commands in `.claude/commands/ant/`
- Invoked via `/ant:<command>` syntax
- Installation: `aether install` copies commands to `~/.claude/commands/ant/`

**OpenCode (SST):**
- Plugin: `@opencode-ai/plugin` 1.1.63
- Command directory: `.opencode/commands/ant/` (mirrors Claude commands)
- Agent definitions: `.opencode/agents/` (4 specialized agents)
  - `aether-queen.md` - Orchestrator agent
  - `aether-builder.md` - Implementation agent
  - `aether-scout.md` - Research agent
  - `aether-watcher.md` - Verification agent
- Configuration: `.opencode/opencode.json`

## Data Storage

**Databases:**
- None - All data is file-based JSON

**File Storage:**
- **Local filesystem only**
- Colony state: `.aether/data/COLONY_STATE.json`
- Constraints: `.aether/data/constraints.json`
- Flags: `.aether/data/flags.json`
- Activity log: `.aether/data/activity.log`
- Spawn tree: `.aether/data/spawn-tree.txt`
- Backups: `.aether/data/backups/`
- Registry: `~/.aether/registry.json` (tracks installed repos)

**Caching:**
- None - No caching layer

## Authentication & Identity

**Auth Provider:**
- None - Aether is a local tool

**Implementation:**
- Relies on Claude Code / OpenCode for any authentication
- Git used for repo detection and dirty state checking

## Monitoring & Observability

**Error Tracking:**
- Built-in error ledger in `COLONY_STATE.json` (`.errors.records`)
- Error pattern detection via `aether-utils.sh error-pattern-check`
- Flag system for blockers: `.aether/data/flags.json`

**Logs:**
- Activity log: `.aether/data/activity.log`
- JSON-based structured logging via `aether-utils.sh activity-log`
- Spawn tracking: `.aether/data/spawn-tree.txt`

## CI/CD & Deployment

**Hosting:**
- npm registry: `aether-colony` package
- GitHub: `https://github.com/calcosmic/Aether.git`

**CI Pipeline:**
- None detected - no `.github/workflows/` or CI config files
- Linting available via `npm run lint`

**Distribution:**
- npm publish (manual)
- Postinstall hook runs `aether install --quiet`

## Environment Configuration

**Required env vars:**
- `HOME` - Required for installation paths (checked in `bin/cli.js`)

**Optional env vars:**
- None - configuration is file-based

**Secrets location:**
- None - No secrets or credentials required

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Git Integration

**Safety Checks:**
- Dirty file detection before updates (`git status --porcelain`)
- Stash capability for forced updates (`git stash push`)
- Repo detection (`git rev-parse --git-dir`)

**Implementation:**
- `bin/cli.js` functions: `isGitRepo()`, `getGitDirtyFiles()`, `gitStashFiles()`
- Target directories checked: `.aether/`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `.opencode/agents/`

## File Synchronization

**Hub Distribution Model:**
- Central hub: `~/.aether/`
- Source files in package `runtime/` -> hub `~/.aether/system/`
- Commands synced to hub, then to registered repos
- SHA-256 hash comparison prevents unnecessary writes

**Sync Functions (in `bin/cli.js`):**
- `syncDirWithCleanup()` - Full directory sync with orphan removal
- `syncSystemFilesWithCleanup()` - Allowlisted system file sync
- `generateManifest()` - Track installed files with hashes

---

*Integration audit: 2026-02-13*
