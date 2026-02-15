# Aether Pheromone System - Complete Design

**Status:** Design Phase
**Last Updated:** 2026-02-15
**Context:** Consolidation of all pheromone-related design discussions

---

## 1. Core Philosophy

**Pheromones are the Queen's Eternal Memory made manifest.**

Workers don't read files—they follow scent trails left by previous colonies, stronger where many ants have walked. The system learns as it works. The Queen guides without interrupting.

**Key Principles:**
- Emergence over explicit commands
- No new commands needed—integrated into existing flow
- Pheromones are invisible substrate, visible only when relevant
- Auto-deposition at critical moments with user signaling
- Mid-work injection without stopping workers

---

## 2. Pheromone Taxonomy

| Type | Source | Decay | Purpose | Example |
|------|--------|-------|---------|---------|
| **FOCUS** | `/ant:focus`, planning | 30 days | What to pay attention to | FOCUS[authentication] |
| **REDIRECT** | `/ant:redirect`, errors | 60 days | Patterns to avoid | REDIRECT[regex-parsing] |
| **PHILOSOPHY** | Queen's Will, Eternal | Never | Core beliefs | PHILOSOPHY[emergence-over-orchestration] |
| **STACK** | `/ant:colonize`, detection | When stack changes | Tech constraints | STACK[nodejs-bash-jq] |
| **PATTERN** | Milestones, workers | 90 days | Discovered best practices | PATTERN[bash-for-file-ops] |
| **DECREE** | User override | Never | Explicit commands | DECREE[no-force-push] |

---

## 3. Eternal Memory Architecture

### Global Eternal Memory
```
~/.aether/eternal/
├── queen-will.md              # Core philosophies (never changes without decree)
├── pheromones.json            # Active eternal pheromones
├── stack-profile/             # Tech fingerprints from all colonies
│   ├── nodejs.md
│   ├── python.md
│   └── ...
├── patterns/                  # Validated patterns across colonies
│   ├── emergence.md
│   ├── minimal-change.md
│   └── ...
└── lineage/                   # Inheritable from specific colonies
    └── chamber-{id}/
        ├── decisions.md
        ├── lessons.md
        └── pheromones.json
```

### Per-Colony Pheromones
```
.aether/data/
├── pheromones.json           # Active trails for this colony
├── pheromone-queue.json      # Queued for worker pickup
└── midden/                   # Expired/faded pheromones
    └── pheromones.json
```

### Pheromone JSON Structure
```json
{
  "trails": [
    {
      "id": "phem_abc123",
      "type": "FOCUS",
      "substance": "authentication",
      "strength": 0.9,
      "source": "user:focus",
      "deposited_at": "2026-02-15T10:30:00Z",
      "decay": "30d",
      "decays_at": "2026-03-17T10:30:00Z"
    }
  ]
}
```

---

## 4. Auto-Injection at Critical Points

| Moment | Auto-Pheromone | User Signal | Purpose |
|--------|---------------|-------------|---------|
| `/ant:init` completes | PHILOSOPHY[emergence], PHILOSOPHY[minimal-change] | "🐜 Queen's scent laid: Emergence" | Foundation |
| First `/ant:plan` | PHILOSOPHY[minimal-planning] | "🐜 Trail: Plan just enough" | Planning bias |
| Worker error | REDIRECT[error-pattern] | "🐜 Warning: {pattern}" | Learn from failure |
| Phase complete | PATTERN[what-worked] | "🐜 Success: {pattern}" | Reinforce |
| `/ant:seal` | PHILOSOPHY[milestone] + PATTERN[conventions] | "🐜 Wisdom archived" | Lineage |
| `/ant:swarm` fix | PATTERN[fix] + REDIRECT[failed] | "🐜 Immunity: {pattern}" | Bug prevention |
| User override | DECREE[reason] | "🐜 Decree recorded" | Authority |

### User Signaling Format
```
═══════════════════════════════════════════════════════
  🐜 PHEROMONE DEPOSITED
═══════════════════════════════════════════════════════

  Type:      PATTERN
  Substance: prefer-bash-over-node-for-file-ops
  Strength:  0.7
  Source:    milestone:phase-3-complete
  Why:       Workers used bash 5x more than node

  This trail guides future workers.
  View all: /ant:status

═══════════════════════════════════════════════════════
```

---

## 5. Mid-Work Injection System

### The Challenge
User wants to inject pheromone while workers are active, without interrupting flow.

### Solution: Pheromone Queue + Checkpoint Polling

```
User: /ant:focus "error-handling" (while workers building)

System:
1. Deposit to pheromone-queue.json immediately
2. Display: "🐜 FOCUS queued - workers detect at next checkpoint"
3. Worker flow continues uninterrupted

Workers:
- Poll queue at natural breakpoints
- Pick up without breaking context
- Log: "[Worker] New scent: FOCUS[error-handling]"
```

### Queue Structure
```json
{
  "queue": [
    {
      "id": "phem_queued_123",
      "type": "FOCUS",
      "substance": "error-handling",
      "strength": 0.9,
      "status": "queued",
      "picked_up_by": null,
      "picked_up_at": null
    }
  ]
}
```

### Worker Checkpoint Protocol
Natural breakpoints for pheromone detection:
- After completing a task
- Before spawning a sub-worker
- After tool use (Read/Edit/Bash)
- Every 5 minutes of continuous work

---

## 6. Worker Spawn-Time Priming

Every worker, at spawn:

1. **Inhales the eternal** - Reads `~/.aether/eternal/queen-will.md`
2. **Checks active trails** - Loads `.aether/data/pheromones.json`
3. **Smells the stack** - Loads relevant stack-profile
4. **Polls queue** - Checks for queued pheromones
5. **Begins work** - Carries context, may deposit new trails

**Example spawn log:**
```
[Prime Worker] Spawned for Phase 7
[Prime Worker] Inhaling Queen's Will... (emergence, minimal-change, no-emojis)
[Prime Worker] Trails: FOCUS[security,0.9] REDIRECT[force-push,0.8]
[Prime Worker] Stack: nodejs-bash-jq (matches eternal profile)
[Prime Worker] Ready to coordinate
```

---

## 7. Biological Patterns to Implement

### Stimulus-Driven Worker Assignment
Workers check pheromone gradients before spawning specialists:
- High REDIRECT[regex-parsing] → spawn Parser Specialist
- High FOCUS[performance] → spawn Watcher before Builder
- PATTERN[error-recovery] detected → Chaos ant joins automatically

### Trophallaxis (Food Sharing)
Senior workers pass "digested learnings" to new workers:
- `/ant:mentor` - Explicitly share chamber wisdom to new colony
- Auto-mentoring: New colonies inherit from sealed chambers

### Midden Piles (Waste Management)
Failed patterns go to `.aether/data/midden/`:
- Not deleted, just marked as "don't go here"
- REDIRECT pheromones point to midden
- Historical record of what didn't work

### Trail Following
Workers follow strongest pheromone trails:
- FOCUS trails attract attention
- REDIRECT trails create avoidance zones
- PATTERN trails create preferred paths

---

## 8. Integration with Existing Commands

### `/ant:init` → Foundation
```
🌱 First colony initialized
🐜 Queen's scent established:
   • Emergence over orchestration
   • Minimal changes preferred

These guide all workers. View with /ant:status
```

### `/ant:focus "area"` → FOCUS Pheromone
```
🐜 Focus trail laid: authentication
   Workers will prioritize auth concerns.
   Strength: 0.9 (30-day decay)

Active: FOCUS[auth,0.9] FOCUS[perf,0.6]
```

### `/ant:redirect "pattern"` → REDIRECT Pheromone
```
🐜 Warning trail laid: regex-parsing
   Workers will avoid or carefully consider.
   Strength: 0.8 (60-day decay)
```

### `/ant:status` → Show Pheromones
```
📈🐜🏘️🐜📊 Colony Status
═══════════════════════════════════════════════════════

State: EXECUTING
Phase: 2 of 4

Active Pheromone Trails:
  🎯 FOCUS: authentication (0.9), error-handling (0.7)
  ⚠️  REDIRECT: regex-parsing (0.8)
  📚 PATTERN: bash-for-file-ops (0.8)

[rest of status...]
```

### `/ant:seal` → Wisdom Extraction
```
🏺 Colony Sealed - Crowned Anthill

3 phases, 47 tasks completed

🐜 Wisdom archived:
   • PATTERN[prefer-joi-over-zod] (validated 5x)
   • PATTERN[bash-for-file-ops] (used 12x)
   • REDIRECT[avoid-sync-fs] (failed once)

Preserved in eternal memory.
```

---

## 9. `/ant:colonize` - CDS-Style Codebase Mapping

Deep codebase analysis producing pheromone trails:

```
/ant:colonize → Scouts map codebase → Deposit trails → Update eternal memory

Outputs:
- .aether/colony/stack.md          (emergent detection)
- .aether/colony/conventions.md    (pattern analysis)
- .aether/colony/concerns.md       (TODO/FIXME mining)
- Updates ~/.aether/eternal/stack-profile/{detected}.md
- Deposits STACK pheromones
- Deposits PATTERN pheromones for discovered conventions
```

**Integration:**
- Called automatically during `/ant:init` (basic)
- Can be called explicitly for deep analysis
- Updates at milestones (what actually got used vs planned)

---

## 10. Lineage System

### Chamber Inheritance
When sealing a colony (`/ant:seal`):
1. Extract phase learnings → PATTERN pheromones
2. Archive to `~/.aether/eternal/lineage/{chamber}/`
3. Update stack-profiles if new tech discovered
4. Strengthen successful pattern trails

### Starting New Colony with Lineage
```
/ant:lay-eggs "New goal" --inherit

🐜 Inheriting from chamber-{id}:
   • 12 patterns adopted
   • 3 redirects (avoidances)
   • Stack profile: nodejs-bash-jq

New colony carries ancestral wisdom.
```

---

## 11. Onboarding UX

### First Colony Experience
Auto-deposit foundation pheromones with explanation:
```
🐜 FIRST COLONY DETECTED
═══════════════════════════════════════════════════════

Your Queen's scent is being established.

Auto-deposited:
  ✓ PHILOSOPHY[emergence-over-orchestration]
  ✓ PHILOSOPHY[minimal-change]

These guide all workers.

💡 Tip: Adjust mid-flight: /ant:focus "security"
═══════════════════════════════════════════════════════
```

### Pattern Suggestions (Non-blocking)
```
[Worker completed 3 similar tasks]

💡 Pattern noticed: check-file-exists-before-write
   Will be deposited as PATTERN at phase completion.
```

---

## 12. Implementation Phases

### Phase 1: Foundation
- [ ] Create `~/.aether/eternal/` structure
- [ ] Implement pheromones.json schema
- [ ] Worker spawn-time priming
- [ ] Auto-deposition in `/ant:init`

### Phase 2: Integration
- [ ] Enhance `/ant:focus` with visibility
- [ ] Enhance `/ant:redirect` with visibility
- [ ] Add pheromone section to `/ant:status`
- [ ] Pheromone queue system

### Phase 3: Learning
- [ ] Phase completion pattern detection
- [ ] Milestone wisdom extraction
- [ ] Swarm immunity deposition
- [ ] Midden pile system

### Phase 4: Colonize
- [ ] `/ant:colonize` command
- [ ] Stack-profile updates
- [ ] Convention detection
- [ ] Auto-update at milestones

### Phase 5: Lineage
- [ ] Chamber inheritance
- [ ] `/ant:lay-eggs --inherit`
- [ ] Trophallaxis (`/ant:mentor`)
- [ ] Cross-colony pattern validation

---

## 13. Context Window Management

Workers don't carry full pheromone history:

```
[Worker Context Header]
[Pheromones: FOCUS(auth,0.9) REDIRECT(regex,0.7) PATTERN(bash-files,0.8)]
[Stack: nodejs-bash-jq]
[Queen's Will: emergence, minimal-change]

[Worker continues with work...]
```

Full details available via tool call if needed.

---

## 14. Persistence Model

| Storage | Location | Lifetime | Access |
|---------|----------|----------|--------|
| Eternal | `~/.aether/eternal/` | Forever | All colonies |
| Colony | `.aether/data/pheromones.json` | Colony lifetime | This colony |
| Queue | `.aether/data/pheromone-queue.json` | Until picked up | Active workers |
| Midden | `.aether/data/midden/` | Forever (archived) | Debug/analysis |
| Chamber | `~/.aether/eternal/lineage/{id}/` | Forever | Inheritance |

---

## Design Decisions

1. **No new commands** - Integrate into existing flow
2. **Auto-deposition** - System learns without user action
3. **Visible signals** - User sees what the colony learned
4. **Mid-work injection** - Queue system for continuous work
5. **Biological fidelity** - Real ant behaviors (stimulus-driven, trophallaxis, midden)
6. **Eternal memory** - Cross-colony persistence via ~/.aether/
7. **Emergence-first** - Pheromones guide, don't command

---

*The colony remembers. The colony learns. The colony evolves.*
