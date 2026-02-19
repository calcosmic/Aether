# Aether Architecture - How It Works (v4.0)

> **Historical note:** Prior to v4.0, a `runtime/` staging directory was used as an intermediary between `.aether/` and the npm package. This was removed in v4.0 to eliminate maintenance burden and the destructive update loop risk. `.aether/` is now packaged directly.

## The Core Concept

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   .aether/             ← SOURCE OF TRUTH (packaged directly)    │
│   ├── workers.md       (edit here)                              │
│   ├── aether-utils.sh                                           │
│   ├── utils/                                                    │
│   └── docs/                                                     │
│                                                                  │
│   .aether/data/        ← LOCAL ONLY (excluded by .npmignore)    │
│   .aether/dreams/      ← LOCAL ONLY (excluded by .npmignore)    │
│                                                                  │
│   .claude/commands/ant/ ← Slash commands (Claude Code)          │
│   .opencode/commands/ant/ ← Slash commands (OpenCode)           │
│   .opencode/agents/     ← Agent definitions                     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## The Distribution Flow

```mermaid
flowchart TB
    subgraph AetherRepo["Aether Repo (source)"]
        AE[".aether/ system files"]
        CC[".claude/commands/ant/"]
        OC[".opencode/"]
        VIS[".aether/visualizations/"]
    end

    subgraph Hub["~/.aether/ (THE HUB)"]
        HS["system/"]
        HC["commands/claude/"]
        HO["commands/opencode/"]
        HA["agents/"]
        HV["visualizations/"]
    end

    subgraph AnyRepo["any-repo/.aether/"]
        RS["system files"]
        RC["commands"]
        RV["visualizations"]
        RD["data/ (local only)"]
    end

    AE -->|"npm install -g . (direct)"| HS
    CC -->|"npm install -g ."| HC
    OC -->|"npm install -g ."| HO
    VIS -->|"npm install -g ."| HV

    HS -->|"aether update"| RS
    HC -->|"aether update"| RC
    HV -->|"aether update"| RV
```

## What Goes Where

```mermaid
flowchart LR
    subgraph distributed ["DISTRIBUTED (via npm)"]
        D1[".aether/ system files"]
        D5[".claude/commands/ant/*"]
        D6[".opencode/commands/ant/*"]
        D7[".opencode/agents/*"]
        D8[".aether/visualizations/*"]
    end

    subgraph local ["LOCAL ONLY (never distributed)"]
        L1[".aether/dreams/*"]
        L2[".aether/data/*"]
        L3[".planning/*"]
        L4[".aether/oracle/*"]
    end
```

## The Update Commands

### `npm install -g .` (in Aether repo)
Validates `.aether/` via `bin/validate-package.sh`, then pushes directly to hub

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant Aether as Aether repo
    participant Hub as ~/.aether/

    Dev->>Aether: Edit .aether/workers.md
    Dev->>Aether: git commit
    Dev->>Aether: npm install -g . (prepublishOnly: validate-package.sh)
    Note over Aether: Validates required files exist, private dirs guarded
    Aether->>Hub: postinstall: cli.js setupHub() reads .aether/ directly
    Note over Hub: Hub now has new version
    Note over Dev: Other repos can now aether update
```

### `aether update` (in any repo)
Pulls latest from hub into that repo's `.aether/`

```mermaid
sequenceDiagram
    participant User
    participant Repo as any-repo/.aether/
    participant Hub as ~/.aether/

    User->>Repo: aether update
    Repo->>Hub: Check version.json
    Hub-->>Repo: v4.0.0
    Note over Repo: Newer version available!
    Repo->>Hub: Pull system files (syncAetherToRepo — exclude-based)
    Repo->>Hub: Pull commands
    Repo->>Hub: Pull visualizations
    Note over Repo: data/ is NEVER touched
```

## Simple Rules

| Rule | Explanation |
|------|-------------|
| **Edit `.aether/` system files** | Source of truth in the Aether repo |
| **Edit `.claude/commands/ant/`** | Slash commands for Claude Code |
| **Edit `.opencode/agents/`** | Agent definitions |
| **`.aether/data/` is safe** | Colony state is never touched by updates |
| **In other repos, don't edit `.aether/`** | Working copies get overwritten by `aether update` |

## The Validation Script

`bin/validate-package.sh` runs before packaging to verify the `.aether/` directory is ready.

- Runs automatically as npm `prepublishOnly` hook
- Checks required files exist (workers.md, aether-utils.sh, etc.)
- Guards against private data exposure (verifies .aether/.npmignore covers data/, dreams/, oracle/, etc.)
- Supports `--dry-run` mode for pre-commit checks

```bash
# Manual run (normally automatic via npm install -g .)
bash bin/validate-package.sh

# Dry-run mode
bash bin/validate-package.sh --dry-run

# Verify what npm would actually package
npm pack --dry-run
```

## The Hub Sync (setupHub)

`bin/cli.js setupHub()` distributes content from the installed package to the hub (`~/.aether/`):

- **System files:** Walks `.aether/` directly using exclude-based approach (`HUB_EXCLUDE_DIRS`) — no allowlist
- **Claude commands:** Copies `.claude/commands/ant/` → `~/.aether/commands/claude/`
- **OpenCode commands + agents:** Copies from `.opencode/` directories
- **Rules:** Dedicated step copies `.claude/rules/` → `~/.aether/commands/rules/`

## The Target Repo Sync (syncAetherToRepo)

`bin/lib/update-transaction.js syncAetherToRepo()` distributes from hub to target repos:

- Exclude-based approach (`EXCLUDE_DIRS`) — no allowlist
- Excludes: data/, dreams/, oracle/, archive/, chambers/, locks/
- Only system files are distributed; local data is never touched

## The Visualizations

```
.aether/visualizations/ → DISTRIBUTED
.aether/dreams/         → NOT distributed (excluded by .npmignore)
.aether/data/           → NOT distributed (excluded by .npmignore)
```

Why? Visualizations are ASCII art assets needed by the `/ant:maturity` command, so they need to be distributed with the package.

## Quick Reference

```bash
# You changed system files in .aether/:
npm install -g .          # Validate + push to hub

# You want updates in another repo:
/ant:update               # Pull from hub

# CLI equivalent:
aether update             # Same as /ant:update
aether update --force     # Stash changes and update
```
