# Aether Architecture - How It Works

## The Core Concept

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   .aether/             ← SOURCE OF TRUTH for system files       │
│   ├── workers.md       (edit here)                              │
│   ├── aether-utils.sh                                           │
│   ├── utils/                                                    │
│   └── docs/                                                     │
│                                                                  │
│   runtime/             ← STAGING (auto-populated, don't edit)   │
│                                                                  │
│   .claude/commands/ant/ ← Slash commands (Claude Code)          │
│   .opencode/commands/ant/ ← Slash commands (OpenCode)           │
│   .opencode/agents/     ← Agent definitions                     │
│                                                                  │
│   .aether/data/        ← LOCAL ONLY (colony state, never sync)  │
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
        RT["runtime/ (staging)"]
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

    AE -->|"bin/sync-to-runtime.sh"| RT
    RT -->|"npm install -g ."| HS
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

    subgraph staging ["STAGING (auto-generated)"]
        S1["runtime/*"]
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
Syncs .aether/ to runtime/, then pushes to hub

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant Aether as Aether repo
    participant RT as runtime/ (staging)
    participant Hub as ~/.aether/

    Dev->>Aether: Edit .aether/workers.md
    Dev->>Aether: git commit
    Dev->>RT: npm install -g . (preinstall: sync-to-runtime.sh)
    Note over RT: .aether/ system files copied to runtime/
    RT->>Hub: postinstall: cli.js install
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
    Hub-->>Repo: v3.1.7
    Note over Repo: Newer version available!
    Repo->>Hub: Pull system files
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
| **Don't edit `runtime/` directly** | It's auto-populated from `.aether/` on publish |
| **`.aether/data/` is safe** | Colony state is never touched by updates |
| **In other repos, don't edit `.aether/`** | Working copies get overwritten by `aether update` |

## The Sync Script

`bin/sync-to-runtime.sh` copies allowlisted system files from `.aether/` to `runtime/`.

- Runs automatically as npm `preinstall` hook
- Uses the same allowlist as `update-transaction.js`
- Only copies changed files (hash comparison)
- Never deletes extras in `runtime/` (templates, signatures, etc.)

```bash
# Manual run (normally automatic)
bash bin/sync-to-runtime.sh

# Reverse sync (seed .aether/ from runtime/, one-time use)
bash bin/sync-to-runtime.sh --reverse
```

## The Visualizations Exception

```
.aether/visualizations/ → DISTRIBUTED
.aether/dreams/         → NOT distributed
.aether/data/           → NOT distributed (local state)
```

Why? Visualizations are ASCII art assets needed by the `/ant:maturity` command, so they need to be distributed with the package.

## Quick Reference

```bash
# You changed system files in .aether/:
npm install -g .          # Auto-sync + push to hub

# You want updates in another repo:
/ant:update               # Pull from hub

# CLI equivalent:
aether update             # Same as /ant:update
aether update --force     # Stash changes and update
```
