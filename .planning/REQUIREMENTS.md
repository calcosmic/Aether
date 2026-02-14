# Requirements: Aether Colony System v3.1 Open Chambers

**Defined:** 2026-02-14
**Core Value:** Autonomous multi-agent orchestration that scales from single-user development to team collaboration, with pheromone-based constraints guiding agent behavior.

---

## v3.1 Requirements

### Model Routing & Configuration

**Table stakes:**
- [ ] **MOD-01**: User can view current model assignments per caste (`/ant:caste-models` or `aether caste-models list`)
- [ ] **MOD-02**: User can override model for specific caste (`aether caste-models set builder=claude-sonnet`)
- [ ] **MOD-03**: System verifies LiteLLM proxy health before spawning workers
- [ ] **MOD-04**: Model verification shows which provider each model routes to
- [ ] **MOD-05**: System logs actual model used per worker spawn (not just configured)

**Differentiators:**
- [ ] **MOD-06**: Task-based routing — keyword detection ("design", "architecture" → glm-5, "implement" → kimi)
- [ ] **MOD-07**: Model performance telemetry — track success rates per model/caste
- [ ] **MOD-08**: Model override per command (`--model` flag for one-time override)

---

### Colony Lifecycle (Ant-themed CDS)

**Table stakes:**
- [ ] **LIFE-01**: `/ant:entomb` — Entomb current colony (preserve in `.aether/chambers/`), seal with pheromone trails (metadata), reset COLONY_STATE
- [ ] **LIFE-02**: `/ant:lay-eggs` — Lay first eggs of new colony ("First Eggs" milestone — beginning of metamorphosis), spawn fresh or from entombed chamber
- [ ] **LIFE-03**: Milestone auto-detection — Compute maturity from state (First Mound → Open Chambers → Brood Stable → Ventilated Nest → Sealed Chambers → Crowned Anthill)
- [ ] **LIFE-04**: `/ant:tunnels` — Explore tunnels (browse archived colonies) with summary view
- [ ] **LIFE-05**: Entombment includes pheromone manifest (manifest.json with date, goal, phases completed, learnings preserved)

**Differentiators:**
- [ ] **LIFE-06**: ASCII art anthill visualization showing colony maturity journey
- [ ] **LIFE-07**: Chamber comparison — compare pheromone trails across entombed colonies

---

### Immersive Visualization

**Table stakes:**
- [ ] **VIZ-01**: Real-time foraging display — show ants currently working with caste emoji
- [ ] **VIZ-02**: Collapsible tunnel view — expand/collapse to see nested agent spawns
- [ ] **VIZ-03**: Tool usage stats — count of Read/Grep/Edit/Bash per ant
- [ ] **VIZ-04**: Trophallaxis metrics (token usage) — show resources consumed per ant/task
- [ ] **VIZ-05**: Timing information — duration per ant, elapsed time, ETA

**Differentiators:**
- [ ] **VIZ-06**: Ant-themed presentation — "3 foragers excavating...", pheromone trail metaphor for activity log
- [ ] **VIZ-07**: Chamber activity map — show which nest zones (Fungus Garden, Nursery, Refuse Pile) have active ants
- [ ] **VIZ-08**: Live excavation progress bars for long-running operations
- [ ] **VIZ-09**: Color + caste emoji together — distinct color per caste (Builder=blue, Watcher=green, Scout=yellow, Chaos=red, Prime=purple) AND emojis (🔨🐜, 👁️🐜, 🔍🐜, 🎲🐜)

---

### Quick Wins

- [ ] **QUICK-01**: Surface Dreams in `/ant:status` — show recent dream count and last dream time
- [ ] **QUICK-02**: Auto-Load Context — commands automatically recognize nestmates (read TO-DOs and colony state)
- [ ] **QUICK-03**: `/ant:verify-castes` command — verify model routing per caste

---

## v4.0 Requirements (Deferred)

### Future Improvements
- **INTELLIGENT-01**: AI-driven model selection based on task complexity analysis
- **CROSS-01**: Cross-colony analytics and comparison
- **CLOUD-01**: Cloud-based routing (violates local-first — deferred indefinitely)

---

## Out of Scope

| Feature | Reason |
|---------|--------|
| Web UI | CLI-first approach, target v4.0+ |
| Real-time collaboration | Single developer focus, target v4.0+ |
| Cost tracking per model | Out of scope for v3.1 |
| Per-request model switching | Breaks context consistency |

---

## Ant Terminology Guide

| Concept | Ant Term | Caste/Behavior |
|---------|----------|----------------|
| Archive | **Entomb** | Undertaker (preserves completed/disused) |
| New Milestone | **Lay Eggs** / **First Eggs** | Queen/Metamorphosis (new colony beginning) |
| History | **Tunnels** / **Explore Tunnels** | Forager/Scout (paths through colony) |
| Metadata | **Pheromone Trails** | Trail marking (persistent signals) |
| Complete | **Seal Chamber** | Mason (interface freeze) |
| Status | **Nest Status** | Queen/Colony overview |
| Activity Log | **Foraging Trails** | Forager behavior |
| Progress | **Excavation** | Mason nest building |
| Resources/Tokens | **Trophallaxis** | Social fluid exchange |

---

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| MOD-01 | 9 | Complete |
| MOD-02 | 9 | Complete |
| MOD-03 | 9 | Complete |
| MOD-04 | 9 | Complete |
| MOD-05 | 9 | Complete |
| MOD-06 | 11 | Pending |
| MOD-07 | 11 | Pending |
| MOD-08 | 11 | Pending |
| LIFE-01 | 10 | Pending |
| LIFE-02 | 10 | Pending |
| LIFE-03 | 10 | Pending |
| LIFE-04 | 10 | Pending |
| LIFE-05 | 10 | Pending |
| LIFE-06 | 12 | Pending |
| LIFE-07 | 12 | Pending |
| VIZ-01 | 12 | Pending |
| VIZ-02 | 12 | Pending |
| VIZ-03 | 12 | Pending |
| VIZ-04 | 12 | Pending |
| VIZ-05 | 12 | Pending |
| VIZ-06 | 12 | Pending |
| VIZ-07 | 12 | Pending |
| VIZ-08 | 12 | Pending |
| VIZ-09 | 12 | Pending |
| QUICK-01 | 9 | Complete |
| QUICK-02 | 9 | Complete |
| QUICK-03 | 9 | Complete |

**Coverage:**
- v3.1 requirements: 27 total
- Mapped to phases: 27
- Unmapped: 0 ✓

---

*Requirements defined: 2026-02-14*
*Last updated: 2026-02-14 after ant terminology refinement*
