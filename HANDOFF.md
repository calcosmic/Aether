# Aether v3.1 Recovery & Release - Context Handoff

**Created:** 2026-02-15
**Status:** v3.1.6 committed and ready for npm publish
**Next Agent:** Read this file first

---

## What Was Accomplished

### 1. Diagnosed the Destructive Update Loop

**The Problem (now fixed):**
- User was editing `.aether/` (working copy) directly
- npm install copied stale `runtime/` to hub
- `aether update` copied stale hub back to `.aether/` → **destroyed work**

**The Fix:** `.aether/` is now the source of truth in the Aether repo. A sync script (`bin/sync-to-runtime.sh`) auto-copies system files to `runtime/` before packaging.

**The Architecture:**
```
.aether/ (source of truth - edit here)
    │
    │  bin/sync-to-runtime.sh (auto on npm install)
    ▼
runtime/ (staging - auto-populated)
    │
    │  npm install -g .
    ▼
~/.aether/ (THE HUB - central distribution)
    │
    │  aether update (in any repo)
    ▼
./.aether/ (working copy - gets overwritten in OTHER repos)
```

**Key Rule:** In the Aether repo, edit `.aether/` system files naturally. In other repos, don't edit `.aether/` — it gets overwritten by `aether update`.

### 2. Multi-Agent Research (5 agents launched)

Researched:
- v3.1 features (Phases 9-12: Caste Model, Entombment, Foraging, Visualization)
- Uncommitted changes analysis
- runtime/ vs .aether/ sync status
- New slash commands (maturity, verify-castes)
- Future features (Pheromone System, Aether 2.0)

### 3. Discovered CHANGELOG vs Reality Gap

**Critical Finding:**
- CHANGELOG documented v3.1.2-3.1.5 as released
- But the actual code changes were **never committed**
- npm package 3.1.5 had broken code (`subagent_type="general"`)
- Fix (`subagent_type="general-purpose"`) was only in uncommitted files

### 4. Committed Everything (v3.1.6)

**Commit `3a5b81c`:** 49 files, +9678 lines
- Agent type fix: `general` → `general-purpose`
- Swarm display integration (13 calls in build.md)
- Archaeologist visualization
- Nested spawn visualization
- New commands: `/ant:maturity`, `/ant:verify-castes`
- ASCII art anthill stages (6 files)
- Caste colors and emoji definitions
- Documentation: Pheromone system spec, v2.0 roadmap

**Commit `8b3180c`:** Version bump to 3.1.6

### 5. Updated Hub

Ran `npm install -g .` - hub now at v3.1.6

---

## Current State

| Component | Version | Status |
|-----------|---------|--------|
| Repo (main branch) | v3.1.6 | ✅ Committed |
| Hub (`~/.aether/`) | v3.1.6 | ✅ Synced |
| Global commands (`~/.claude/commands/ant/`) | v3.1.6 | ✅ Synced |
| npm registry | 3.1.5 | ❌ BROKEN (old version) |

---

## Next Steps (In Order)

### 1. Test in a Repo
```bash
# In any repo with Aether:
aether update

# Then test:
/ant:build 1
```

**Verify:**
- Ants show with emojis (🔨👁️🎲🔍🏺)
- Chaos Ant spawns in Step 5.4.2
- Swarm display updates in real-time
- `/ant:maturity` shows ASCII anthill

### 2. Publish to npm (when ready)
```bash
npm publish
```

### 3. Future Work (from docs research)

**Priority Order:**
1. Visual Output Spec (low effort, immediate UX improvement)
2. Pheromone System Phases 1-2 (3-5 days, foundation)
3. F1 Model Routing (1 week)
4. F7 Cost Optimization (depends on F1)
5. F9 Visual Observatory

**Key Docs:**
- `.aether/docs/AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md`
- `.aether/docs/AETHER-2.0-IMPLEMENTATION-PLAN.md`
- `.aether/docs/VISUAL-OUTPUT-SPEC.md`

---

## Key Files to Know

| File | Purpose |
|------|---------|
| `CLAUDE.md` (repo root) | Development rules, architecture |
| `.aether/` (system files) | Source of truth — auto-syncs to `runtime/` on publish |
| `.claude/commands/ant/` | Slash commands (Claude Code) |
| `.opencode/commands/ant/` | Slash commands (OpenCode) |
| `~/.aether/` | Hub - distributes to all repos |
| `.aether/RECOVERY-PLAN.md` | Full recovery documentation |

---

## Architecture Summary

```
Aether Repo (this repo)
├── .aether/ (SOURCE OF TRUTH for system files)
│   ├── workers.md, aether-utils.sh, utils/, docs/
│   │        │
│   │        │  bin/sync-to-runtime.sh (auto on npm install)
│   │        ▼
├── runtime/ (STAGING — auto-populated)
├── .claude/commands/ant/ ──────────────────────┤──→ npm package
├── .opencode/ ─────────────────────────────────┤
│                                               ▼
│                                         ~/.aether/ (THE HUB)
│                                         ├── system/      ← runtime/
│                                         ├── commands/    ← slash commands
│                                         └── visualizations/
│                                               │
│  aether update (in ANY repo)  ◄───────────────┘
│
▼
any-repo/.aether/ (WORKING COPY)
├── workers.md, aether-utils.sh  ← from hub
└── data/                        ← LOCAL (never touched)
```

**Development workflow:**
1. Edit `.aether/` system files or `.claude/commands/ant/` for commands
2. Commit changes
3. Run `npm install -g .` — auto-syncs `.aether/` → `runtime/`, then pushes to hub
4. Hub distributes to all repos via `aether update`

---

## Visual Features in v3.1

- **Swarm display:** Real-time ant visualization with caste colors
- **Caste emojis:** 🔨 Builder, 👁️ Watcher, 🎲 Chaos, 🔍 Scout, 🏺 Archaeologist, 👑 Prime
- **ASCII anthills:** 6 milestone stages in `.aether/visualizations/anthill-stages/`
- **Tool counters:** 📖 Read, 🔍 Grep, ✏️ Edit, ⚡ Bash
- **Progress bars:** Excavation animation during builds

---

## Commands Reference

```bash
# Update hub from local source
npm install -g .

# Update a repo from hub
aether update

# Force update (stashes uncommitted changes)
aether update --force

# Verify caste model assignments
aether caste-models list

# Run all tests
npm test

# Verify command sync between Claude and OpenCode
npm run lint:sync
```

---

## Git Status (Clean)

```
8b3180c chore: bump version to 3.1.6
3a5b81c release: v3.1 Open Chambers - complete visual system integration
61b4165 sync: runtime/ updated from working .aether/
```

---

**End of Handoff** - Good luck with the next session!
