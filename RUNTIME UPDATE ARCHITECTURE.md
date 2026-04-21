# Aether Architecture - How It Works

> **Distribution note:** The Go `aether` binary is the only runtime. A thin npm bootstrap package now exists at `npm/` for `npx --yes aether-colony@latest`, but it only downloads and hands off to the published Go release.

Version rule:
- `.aether/version.json` is the source-checkout release version file.
- `npm/package.json` must use the exact same version as `.aether/version.json`.
- The public npm `latest` tag should point at the same stable release version as GitHub Releases.

## The Core Concept

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   cmd/                 ← Go source code (primary)               │
│   ├── main.go         CLI entry point                           │
│   └── *.go            80+ subcommands                           │
│                                                                  │
│   pkg/                 ← Shared Go packages                     │
│   ├── agent/          Agent pool, spawn tree                    │
│   ├── downloader/     Binary download + extraction              │
│   ├── memory/         Learning pipeline, instincts              │
│   └── storage/        JSON store, file locking                  │
│                                                                  │
│   .aether/             ← SOURCE OF TRUTH (companion files)      │
│   ├── workers.md       Worker definitions                       │
│   ├── rules/           Rules files (e.g. aether-colony.md)      │
│   ├── skills/          colony/ (10) + domain/ (18)              │
│   ├── templates/       Colony state, pheromones, etc.           │
│   ├── docs/            Distributed documentation                │
│   ├── agents-claude/   Agent definitions (packaging mirror)     │
│   └── utils/           Runtime utilities                        │
│                                                                  │
│   .aether/data/        ← LOCAL ONLY (gitignored, never sync'd)  │
│   .aether/dreams/      ← LOCAL ONLY (gitignored, never sync'd)  │
│                                                                  │
│   .claude/commands/ant/ ← 50 slash commands (Claude Code)       │
│   .claude/agents/ant/   ← 25 agent definitions (Claude Code)    │
│   .claude/rules/        ← Rules loaded by Claude Code           │
│                                                                  │
│   .opencode/commands/ant/ ← 50 slash commands (OpenCode)        │
│   .opencode/agents/       ← Agent definitions (OpenCode)        │
│   .codex/agents/          ← 25 agent definitions (Codex CLI)    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## The Distribution Flow

```
┌──────────────────┐     aether install      ┌──────────────────────┐
│   Aether Repo    │ ──────────────────────> │   ~/.aether/ (HUB)   │
│                  │                          │                      │
│ .aether/*  ──────┤   copies companion      │ system/              │
│ .claude/   ──────┤   files + agents +      │ ├── workers.md       │
│ .opencode/ ──────┤   commands to hub       │ ├── rules/           │
│                  │                          │ ├── skills/          │
│                  │                          │ ├── templates/       │
│                  │                          │ ├── agents-claude/   │
│                  │                          │ ├── commands/        │
│                  │                          │ └── docs/            │
└──────────────────┘                          └──────────┬───────────┘
                                                         │
                           aether update / aether setup   │
                                                         │
                                              ┌──────────▼───────────┐
                                              │  any-repo/            │
                                              │  ├── .aether/         │
                                              │  ├── .claude/         │
                                              │  └── .opencode/       │
                                              │                      │
                                              │  .aether/data/        │
                                              │  (never touched)      │
                                              └──────────────────────┘
```

## What Goes Where

| Category | Source (Aether repo) | Hub (`~/.aether/system/`) | Target repos |
|----------|---------------------|--------------------------|--------------|
| System files | `.aether/*` | `system/*` | `.aether/*` |
| Rules | `.aether/rules/` | `system/rules/` | `.claude/rules/` |
| Commands (Claude) | `.claude/commands/ant/` | `system/commands/claude/` | `.claude/commands/ant/` |
| Agents (Claude) | `.claude/agents/ant/` | `system/agents-claude/` | `.claude/agents/ant/` |
| Commands (OpenCode) | `.opencode/commands/ant/` | `system/commands/opencode/` | `.opencode/commands/ant/` |
| Agents (OpenCode) | `.opencode/agents/` | `system/agents/` | `.opencode/agents/` |
| Agents (Codex) | `.codex/agents/` | `system/codex/` | `.codex/agents/` |
| Skills (Codex) | `.aether/skills-codex/` | `system/skills-codex/` | `.codex/skills/aether/` |
| Skills | `.aether/skills/` | `system/skills/` | `.aether/skills/` |
| Templates | `.aether/templates/` | `system/templates/` | `.aether/templates/` |
| Docs | `.aether/docs/` | `system/docs/` | `.aether/docs/` |

Published bootstrap path:
- `npx --yes aether-colony@latest` downloads the published Go release, installs it locally, and runs `aether install`.
- The npm package is not a second runtime and should always trail a published Go release, never lead it.

**Never distributed (local only):**
- `.aether/data/` — colony state, pheromones, midden
- `.aether/dreams/` — dream journal
- `.aether/oracle/` — research artifacts
- `.aether/checkpoints/` — session checkpoints
- `.aether/locks/` — file locks

## The Three Commands

### `aether install` (in Aether repo)

Pushes from the Aether repo to the hub. This is what you run after editing source files.

**What it does:**
1. Syncs slash commands to `~/.claude/commands/ant/` and `~/.opencode/command/`
2. Syncs agent definitions to `~/.claude/agents/ant/` and `~/.opencode/agent/`
3. Calls `setupInstallHub()` to create `~/.aether/` with `registry.json` and `version.json`
4. Syncs Codex agents to `~/.codex/agents/` and Codex skills to `~/.codex/skills/aether/`
5. When run from an Aether source checkout, rebuilds the shared local `aether` binary unless `--skip-build-binary` is used
6. Optionally downloads the Go binary from GitHub Releases (`--download-binary`)

**Sync pairs (repo → home directory):**

| Source | Destination | Label |
|--------|-------------|-------|
| `.claude/commands/ant/` | `~/.claude/commands/ant/` | Commands (claude) |
| `.claude/agents/ant/` | `~/.claude/agents/ant/` | Agents (claude) |
| `.opencode/commands/ant/` | `~/.opencode/command/` | Commands (opencode) |
| `.opencode/agents/` | `~/.opencode/agent/` | Agents (opencode) |
| `.codex/agents/` | `~/.codex/agents/` | Agents (codex) |
| `.aether/skills-codex/` | `~/.codex/skills/aether/` | Skills (codex) |

Note: The hub's `system/` directory (companion files) is populated by copying `.aether/` contents. The `install` command also cleans up stale files in the destination that no longer exist in the source.

Runtime rule:
- Unreleased Go runtime fixes propagate across repos on the same machine through `aether install --package-dir <Aether checkout>`, because that is the step that refreshes the shared binary.

### `aether update` (in any repo)

Pulls from the hub to the local repo. This is what you run in other repos to get updates.

**What it does:**
1. Checks hub version against local version
2. Syncs all companion files from `~/.aether/system/` to the local repo
3. Preserves local data (data/, dreams/ are protected)
4. Optionally downloads a new binary (`--download-binary`)

Runtime rule:
- Plain `aether update` does not rebuild or publish the local Go runtime. It only syncs repo companion files.
- `aether update --download-binary` can fetch a published release binary, but it cannot pull an unreleased local source change.

**Sync pairs (hub system/ → local repo):**

| Source (relative to hub system/) | Destination (relative to .aether/) | Label |
|----------------------------------|-------------------------------------|-------|
| `.` | `.` | System files |
| `commands/claude` | `../.claude/commands/ant` | Commands (claude) |
| `commands/opencode` | `../.opencode/commands/ant` | Commands (opencode) |
| `agents` | `../.opencode/agents` | Agents (opencode) |
| `agents-claude` | `../.claude/agents/ant` | Agents (claude) |
| `rules` | `../.claude/rules` | Rules (claude) |

**Modes:**
- **Normal (default):** Only copies new files. Existing files are never overwritten.
- **Force (`--force`):** Overwrites modified files and removes stale ones. Protected directories (data/, dreams/) are never touched.
- **Dry-run (`--dry-run`):** Shows what would change without making changes.

### `aether setup` (in any repo)

Initial setup from hub. Same sync pairs as `update` but never overwrites existing files — local always takes precedence. Also creates required directories (data/, checkpoints/, locks/) and a .gitignore.

## Protected Paths

These are never modified by update/setup:

| Path | Reason |
|------|--------|
| `.aether/data/` | Colony state (COLONY_STATE.json, pheromones, midden) |
| `.aether/dreams/` | Dream journal entries |
| `.aether/checkpoints/` | Session checkpoints |
| `.aether/locks/` | File locks |
| `QUEEN.md` | Hub-level wisdom (never overwritten by update) |
| `CROWNED-ANTHILL.md` | Colony seal marker |

## Simple Rules

| Rule | Explanation |
|------|-------------|
| **Edit `.aether/` system files** | Source of truth in the Aether repo |
| **Edit `.aether/rules/`** | Rules source — syncs to `.claude/rules/` via update |
| **Edit `.claude/commands/ant/`** | Slash commands for Claude Code |
| **Edit `.claude/agents/ant/`** | Agent definitions for Claude Code |
| **Edit `.opencode/` equivalents** | OpenCode commands and agents |
| **`.aether/data/` is safe** | Colony state is never touched by updates |
| **In other repos, don't edit `.aether/` system files** | Working copies get overwritten by `aether update --force` |

## Quick Reference

```bash
# First-time published install without Go:
npx --yes aether-colony@latest

# You changed files in the Aether repo:
aether install                          # Push to hub (~/.aether/system/) and, from source, rebuild the shared binary

# You want updates in another repo:
aether update                           # Pull companion files from hub (safe — new files only)
aether update --force                   # Overwrite modified + remove stale
aether update --dry-run                 # Preview what would change

# First-time setup in a new repo:
aether setup                            # Copy from hub, create directories

# Or use the slash command:
/ant:update                             # Same as aether update

# Binary management:
aether install --download-binary        # Install + fetch latest binary
aether update --download-binary         # Update + fetch latest binary
```

## File Counts

| What | Source Location | Distributed To |
|------|----------------|----------------|
| 50 slash commands (Claude) | `.claude/commands/ant/` | `.claude/commands/ant/` |
| 50 slash commands (OpenCode) | `.opencode/commands/ant/` | `.opencode/commands/ant/` |
| 25 Claude agent definitions | `.claude/agents/ant/` | `.claude/agents/ant/` |
| 25 Codex agent definitions | `.codex/agents/` | `.codex/agents/` |
| 29 source skills (11 colony + 18 domain) | `.aether/skills/` | `.aether/skills/` |
| 29 Codex mirror skills | `.aether/skills-codex/` | `.codex/skills/aether/` |
| 12 templates | `.aether/templates/` | `.aether/templates/` |
| 1 rules file | `.aether/rules/` | `.claude/rules/` |

## Agent Parity Model

Agent definitions have three locations:

1. **`.claude/agents/ant/`** — Canonical (what Claude Code reads)
2. **`.aether/agents-claude/`** — Byte-identical packaging mirror (used by install to populate hub)
3. **`.opencode/agents/`** — Structural parity (same filenames/count, OpenCode format)

When editing agents, always update `.claude/agents/ant/` first, then mirror to `.aether/agents-claude/`.
