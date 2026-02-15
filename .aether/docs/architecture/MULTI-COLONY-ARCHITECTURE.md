# Multi-Colony Architecture Design

**Status:** Design Phase
**Goal:** Enable multiple coordinated colonies per repo with shared eternal memory

---

## 1. Core Concept

Instead of one COLONY_STATE.json per repo, each "colony" is a distinct execution context with:
- Unique colony ID
- Isolated state and history
- Shared eternal memory layer (`~/.aether/eternal/`)
- Lineage relationships to other colonies

```
Repository
├── .aether/
│   ├── colonies/                    # NEW: Multiple colony states
│   │   ├── colony-001-bugfixes/     # Archived colony
│   │   │   ├── COLONY_STATE.json
│   │   │   ├── activity.log
│   │   │   └── completion-report.md
│   │   ├── colony-002-pheromone-foundation/  # Active colony
│   │   │   ├── COLONY_STATE.json
│   │   │   └── ...
│   │   └── colony-003-pheromone-integration/ # Future colony
│   │       └── ...
│   ├── data/                        # Legacy (migrate to colonies/)
│   │   └── COLONY_STATE.json        # Current colony (symlink to active)
│   └── eternal/                     # NEW: Cross-colony memory
│       └── (from pheromone design)
```

---

## 2. Colony Identity System

### Colony ID Format
```
{timestamp}-{slug}-{random}

Examples:
- 20260215-fix-loop-bugs-a7k9
- 20260215-pheromone-phase1-x2m4
- 20260216-pheromone-phase2-p9q1
```

### Colony State Location
```
.aether/colonies/{colony-id}/
├── COLONY_STATE.json       # Colony configuration
├── activity.log            # Worker activity
├── constraints.json        # Active pheromones
├── pheromones.json         # Colony-specific trails
├── completion-report.md    # Final report (when sealed)
└── events/                 # Event history
    └── {timestamp}-{type}.json
```

### Active Colony Symlink
```
.aether/data/COLONY_STATE.json -> ../colonies/{active-id}/COLONY_STATE.json
```

---

## 3. Command Modifications

### `/ant:lay-eggs` - NEW Command
**Purpose:** Create master plan spanning multiple colonies

```
/ant:lay-eggs "Implement Complete Pheromone System"

Creates:
- .aether/MOUND.md (master roadmap)
- Colony specifications for each phase
- Lineage relationships

Output:
🥚🐜🥚 ═══════════════════════════════════════════════
        M O U N D   C R E A T E D
═══════════════════════════════════════════════ 🥚🐜🥚

Master Goal: Implement Complete Pheromone System

Planned Colonies:
  1. Phase 1: Foundation — Eternal memory structure
  2. Phase 2: Integration — Queue system & commands
  3. Phase 3: Learning — Pattern detection
  4. Phase 4: Colonize — Deep codebase analysis
  5. Phase 5: Lineage — Cross-colony inheritance

Each colony will share pheromone trails via eternal memory.

Start first colony:
  /ant:init "Phase 1: Foundation"
```

### `/ant:init` - Modified
**Changes:**
1. Archive current colony (if exists and not already archived)
2. Create new colony directory
3. Link `data/COLONY_STATE.json` to new colony
4. Inherit from prior colonies via pheromones
5. Register in colony registry

**New Output:**
```
🌱🐜🆕🐜🌱 ═══════════════════════════════════════════════
   A E T H E R   C O L O N Y   C R E A T E D
═══════════════════════════════════════════════ 🌱🐜🆕🐜🌙

Colony ID: 20260215-pheromone-foundation-x2m4
Goal: Phase 1: Foundation — Eternal memory structure

📚 Lineage:
   Parent: 20260215-fix-loop-bugs-a7k9
   Inherited: 1 instinct, 5 learnings

🐜 Active Colonies in Repo: 2
   Current: Phase 1: Foundation
   Sealed: Bug Fixes & Update Repair

Next: /ant:plan to generate phase waves
```

### `/ant:tunnels` - Enhanced
**Current:** Browse archived colonies
**Enhanced:** Full colony management

```
/ant:tunnels              # List all colonies
/ant:tunnels --switch 1   # Switch to colony #1
/ant:tunnels --compare 1 2 # Compare two colonies
/ant:tunnels --archive    # Archive current colony
```

**Output:**
```
🕳️🐜🕳️ ═══════════════════════════════════════════════
        C O L O N Y   T U N N E L S
═══════════════════════════════════════════════ 🕳️🐜🕳️

Active Colonies:
  #  ID                                    Goal
  1  20260215-pheromone-foundation-x2m4    Phase 1: Foundation ← ACTIVE
  2  20260215-fix-loop-bugs-a7k9           Bug Fixes (sealed)

Commands:
  /ant:tunnels --switch 2     Resume work on bug fixes
  /ant:tunnels --archive      Seal current colony
  /ant:tunnels --compare 1 2  See differences
```

### `/ant:entomb` - Modified
**Current:** Archive colony to chambers/
**Enhanced:** Proper colony archival with pheromone extraction

```
Process:
1. Extract patterns → PATTERN pheromones
2. Extract wisdom → PHILOSOPHY pheromones
3. Move to .aether/colonies/{id}/
4. Mark as "sealed" in registry
5. Update eternal memory
```

---

## 4. Colony Registry

### Registry File
```json
// .aether/colonies/registry.json
{
  "version": "1.0",
  "current_colony": "20260215-pheromone-foundation-x2m4",
  "colonies": [
    {
      "id": "20260215-fix-loop-bugs-a7k9",
      "goal": "v1.1 Bug Fixes & Update System Repair",
      "status": "sealed",
      "created_at": "2026-02-13T20:40:00Z",
      "sealed_at": "2026-02-14T02:39:00Z",
      "phases_completed": 7,
      "lineage": {
        "parent": null,
        "children": ["20260215-pheromone-foundation-x2m4"]
      },
      "path": ".aether/colonies/20260215-fix-loop-bugs-a7k9/"
    },
    {
      "id": "20260215-pheromone-foundation-x2m4",
      "goal": "Phase 1: Foundation — Eternal memory structure",
      "status": "active",
      "created_at": "2026-02-15T10:30:00Z",
      "sealed_at": null,
      "phases_completed": 0,
      "lineage": {
        "parent": "20260215-fix-loop-bugs-a7k9",
        "children": []
      },
      "path": ".aether/colonies/20260215-pheromone-foundation-x2m4/"
    }
  ]
}
```

---

## 5. Migration Path

### Current State
- Single `COLONY_STATE.json` in `.aether/data/`
- `chambers/` for archived colonies
- No registry

### Migration Steps
1. **Create registry** from existing state
2. **Move current colony** to `colonies/{id}/`
3. **Create symlink** `data/COLONY_STATE.json → colonies/{id}/COLONY_STATE.json`
4. **Move archived** from `chambers/` to `colonies/`

### Backward Compatibility
- Old commands continue working via symlink
- Gradual migration of chamber archives

---

## 6. Pheromone Integration

### Colony Creation Flow
```
User: /ant:init "Phase 2: Integration"

System:
1. Archive current colony → extract pheromones
2. Create new colony state
3. Inherit from parent:
   - Read parent's completion-report.md
   - Extract PHILOSOPHY pheromones
   - Extract PATTERN pheromones
   - Copy to ~/.aether/eternal/
4. Write new colony state
5. Link as active
```

### Worker Spawn Flow
```
Worker spawned in Colony B (child of Colony A)

Worker inhales:
1. Colony B's constraints.json (local pheromones)
2. ~/.aether/eternal/pheromones.json (eternal trails)
3. Parent colony's archived wisdom (if exists)

Worker carries context from lineage.
```

---

## 7. Implementation Phases

### Phase A: Registry System (Prerequisite)
- [ ] Create colony registry format
- [ ] Implement migration from single-state
- [ ] Modify `/ant:init` to use registry
- [ ] Add colony directory structure

### Phase B: Enhanced Commands
- [ ] Create `/ant:lay-eggs` command
- [ ] Enhance `/ant:tunnels` for switching
- [ ] Modify `/ant:entomb` for archival
- [ ] Add colony listing/comparison

### Phase C: Lineage System
- [ ] Track parent-child relationships
- [ ] Implement pheromone inheritance
- [ ] Create cross-colony activity viewer
- [ ] Add completion report extraction

### Phase D: Eternal Memory (Pheromone Foundation)
- [ ] Create `~/.aether/eternal/` structure
- [ ] Implement pheromone auto-deposition
- [ ] Add worker spawn-time priming
- [ ] Build pheromone queue system

---

## 8. Design Decisions

1. **Symlink vs Copy:** Symlink active colony for backward compatibility
2. **Registry Location:** `.aether/colonies/registry.json` (centralized)
3. **Archive Strategy:** Keep full state in `colonies/`, don't compress
4. **Lineage Storage:** Bidirectional (parent knows children, child knows parent)
5. **Pheromone Scope:** Global eternal + colony-local

---

*Colonies multiply. Memory endures. The mound grows.*
