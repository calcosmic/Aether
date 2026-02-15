# Aether Architecture - How It Works

## The Core Concept

```
┌─────────────────────────────────────────────────────────────────┐
│                     AETHER REPO (this repo)                      │
│                                                                  │
│   runtime/              ← SOURCE OF TRUTH for distribution      │
│   ├── workers.md                                             │
│   ├── aether-utils.sh                                        │
│   ├── utils/                                                 │
│   └── docs/              ← Only docs for USERS go here         │
│                                                                  │
│   .claude/commands/ant/ ← Slash commands (Claude Code)         │
│   .opencode/commands/ant/ ← Slash commands (OpenCode)          │
│   .opencode/agents/     ← Agent definitions                    │
│                                                                  │
│   .aether/              ← YOUR LOCAL WORK (not distributed)    │
│   ├── docs/             ← Your personal notes                  │
│   ├── visualizations/   ← ASCII art (gets distributed)         │
│   └── data/             ← Colony state (never touched)         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## The Distribution Flow

```mermaid
flowchart TB
    subgraph AetherRepo["Aether Repo (npm package)"]
        RT["runtime/"]
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

    RT -->|"npm install -g ."| HS
    CC -->|"npm install -g ."| HC
    OC -->|"npm install -g ."| HO
    VIS -->|"npm install -g ."| HV

    HS -->|"aether update"| RS
    HC -->|"aether update"| RC
    HV -->|"aether update"| RV

    style RT fill:#90EE90
    style HS fill:#87CEEB
    style RD fill:#FFB6C1
```

## What Goes Where

```mermaid
flowchart LR
    subgraph "DISTRIBUTED (via npm)"
        D1["runtime/*.md"]
        D2["runtime/*.sh"]
        D3["runtime/utils/*"]
        D4["runtime/docs/*"]
        D5[".claude/commands/ant/*"]
        D6[".opencode/commands/ant/*"]
        D7[".opencode/agents/*"]
        D8[".aether/visualizations/*"]
    end

    subgraph "LOCAL ONLY (never distributed)"
        L1[".aether/docs/*"]
        L2[".aether/data/*"]
        L3[".planning/*"]
    end

    style D1 fill:#90EE90
    style D2 fill:#90EE90
    style D3 fill:#90EE90
    style D4 fill:#90EE90
    style D5 fill:#90EE90
    style D6 fill:#90EE90
    style D7 fill:#90EE90
    style D8 fill:#90EE90
    style L1 fill:#FFB6C1
    style L2 fill:#FFB6C1
    style L3 fill:#FFB6C1
```

## The Update Commands

### `/ant:update` (in any repo)
Pulls latest from hub into that repo's `.aether/`

```mermaid
sequenceDiagram
    participant User
    participant Repo as any-repo/.aether/
    participant Hub as ~/.aether/

    User->>Repo: /ant:update
    Repo->>Hub: Check version.json
    Hub-->>Repo: v3.1.6
    Note over Repo: Newer version available!
    Repo->>Hub: Pull system files
    Repo->>Hub: Pull commands
    Repo->>Hub: Pull visualizations
    Note over Repo: data/ is NEVER touched
```

### `npm install -g .` (in Aether repo)
Pushes to hub, making updates available to all repos

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant Aether as Aether repo
    participant Hub as ~/.aether/

    Dev->>Aether: Edit runtime/workers.md
    Dev->>Aether: git commit
    Dev->>Hub: npm install -g .
    Note over Hub: Hub now has new version
    Note over Dev: Other repos can now /ant:update
```

## Simple Rules

| Rule | Explanation |
|------|-------------|
| **Edit `runtime/`** | Changes that go to ALL users via npm |
| **Edit `.claude/commands/ant/`** | Slash commands for Claude Code |
| **Edit `.aether/visualizations/`** | ASCII art (special case - gets distributed) |
| **NEVER edit `.aether/` system files** | They get overwritten by updates |
| **`.aether/data/` is safe** | Colony state is never touched by updates |
| **`.aether/docs/` is yours** | Personal notes, not distributed |

## The Visualizations Exception

```
.aether/visualizations/ → DISTRIBUTED
.aether/docs/          → NOT distributed
.aether/data/          → NOT distributed (local state)
.aether/*.md           → NOT distributed (working copies)
```

Why? Visualizations are ASCII art assets needed by the `/ant:maturity` command, so they need to be distributed with the package.

## Quick Reference

```bash
# You changed something in Aether repo:
npm install -g .          # Push to hub

# You want updates in another repo:
/ant:update               # Pull from hub

# CLI equivalent:
aether update             # Same as /ant:update
aether update --force     # Stash changes and update
```
