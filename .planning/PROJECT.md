# Aether: Claude-Native Queen Ant Colony

## What This Is

Aether is a **standalone multi-agent system** that applies ant colony intelligence to autonomous agent orchestration, built natively for Claude Code. Worker Ants spawn other Worker Ants through bio-inspired pheromone signaling. The Queen (user) provides high-level intention via pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK), and the colony self-organizes to complete tasks through emergent intelligence.

This is a **hybrid prompt+code system** — commands like `/ant:init "Build a REST API"` work directly in Claude Code as skill prompts. Prompts handle reasoning and orchestration; a thin shell utility layer (`aether-utils.sh`, 369 lines, 18 subcommands) handles deterministic operations (pheromone math, state validation, spawn enforcement, memory management, error tracking, learning promotion) that LLMs get wrong.

**What makes it different:**

Autonomous agent spawning is not unique to Aether — systems like AutoGen (ADAS/Meta Agent Search), AutoAgents, and OpenAI's Agents SDK support dynamic agent creation. What Aether does differently is the coordination model:

1. **Stigmergic Communication** — Pheromone signaling with exponential decay, caste sensitivity profiles, and combination effects (not direct commands or message passing)
2. **Caste Architecture** — Six Worker Ant types with specialist watcher modes, each with different pheromone sensitivities
3. **Bayesian Spawn Tracking** — Spawn outcomes tracked per caste with alpha/beta updates, so the colony learns which specialists succeed
4. **Phased Autonomy** — Structure at boundaries, pure emergence within phases
5. **Colony Memory** — Error tracking, phase learnings, and event awareness that persists across sessions
6. **Hybrid Determinism** — Shell utilities for math/validation/enforcement, prompts for reasoning/orchestration
7. **Claude Code Native** — Entire system is markdown skill prompts + thin shell layer, not a Python/Node framework

## Core Value

**Stigmergic Emergence**: Worker Ants detect capability gaps and spawn specialists through pheromone-guided coordination; pure emergence within structured phases; Queen provides signals not commands.

If this works, everything else follows. If this fails, nothing else matters. The system has been run end-to-end on a real project (filmstrip packaging, 2026-02-04) — colony self-organization works. All 32 field notes have been addressed in v4.4.

## Requirements

### Validated

*(Shipped in v1 — 2026-02-02)*

- ✓ **Claude-Native Command System** — commands as Claude Code skill prompts — v1
- ✓ **Pheromone Signal System** — INIT, FOCUS, REDIRECT, FEEDBACK with time-based decay (1h, 6h, 24h) — v1
- ✓ **Six Worker Ant Castes** — Colonizer, Route-setter, Builder, Watcher, Scout, Architect — v1
- ✓ **Autonomous Agent Spawning** — Recursive spawning via Task tool with full spec propagation — v1
- ✓ **Phase-Based Execution** — Structure at boundaries, emergence within — v1
- ✓ **Basic State Persistence** — JSON file storage with atomic writes and file locking — v1

*(Shipped in v2 — 2026-02-02)*

- ✓ **Event Polling Integration** — Worker Ants check events at execution boundaries — v2
- ✓ **Visual Process Indicators** — Emoji status, step progress, pheromone bars — v2
- ✓ **E2E Test Guide** — 94 verification checks across 6 workflows — v2

*(Shipped in v3-rebuild — 2026-02-03)*

- ✓ **Claude-Native Command Execution** — Commands use Read/Write tools directly, not bash/jq — v3
- ✓ **Clean State Schema** — Minimal JSON: COLONY_STATE, pheromones, PROJECT_PLAN — v3
- ✓ **Phase Lead Emergence Model** — One ant spawned per phase, self-organizes everything — v3
- ✓ **Recursive Spec Propagation** — Spawned ants get full spec + pheromones at any depth — v3

*(Shipped in v4.0 — 2026-02-03)*

- ✓ **Utility Layer** — `aether-utils.sh` wrapper script with 18 subcommands for deterministic operations — v4.0
- ✓ **Pheromone Math Engine** — Decay calculation, signal combination, effective strength computation in shell — v4.0
- ✓ **State Validator** — Schema validation for all JSON state files, prevents field drift and corruption — v4.0
- ✓ **Memory Operations** — Token counting, memory compression, eviction logic in shell — v4.0
- ✓ **Error Tracker** — Pattern counting, category aggregation, deduplication in shell — v4.0
- ✓ **Audit Fix: All 11 issues** — File-lock sourcing, state field consistency, race conditions, jq error handling, state backups, pheromone schema, state integrity, worker status casing, expired pheromone cleanup, colony mode documentation — v4.0
- ✓ **Command Integration** — Core command prompts delegate to aether-utils.sh for deterministic operations — v4.0

*(Shipped in v4.1 — 2026-02-03)*

- ✓ **Orphan audit** — 4 dead subcommands removed, 4 wired to consumers — v4.1
- ✓ **Inline formula elimination** — All inline decay formulas replaced with aether-utils.sh calls — v4.1
- ✓ **Spawn limit enforcement** — spawn-check subcommand + gates in all 6 worker specs — v4.1
- ✓ **Pheromone quality enforcement** — pheromone-validate subcommand + gate in continue.md — v4.1
- ✓ **Spec compliance enforcement** — Post-action validation checklist in all worker specs — v4.1

*(Shipped in v4.2 — 2026-02-03)*

- ✓ **Per-caste pheromone computation** — Caste sensitivity applied to signal display — v4.2
- ✓ **Watcher execution verification** — Watchers actually run code, not just read it — v4.2
- ✓ **Build output & delegation log** — Queen displays what workers did — v4.2
- ✓ **Worker progress output** — All 6 castes emit structured progress markers — v4.2
- ✓ **Learning extraction flow** — continue.md prompted after each phase — v4.2

*(Shipped in v4.3 — 2026-02-04)*

- ✓ **Activity log file** — Workers write progress to `.aether/data/activity.log` during execution for real-time visibility — v4.3
- ✓ **Incremental Queen updates** — Queen displays worker results between spawns instead of waiting for entire Phase Lead return — v4.3
- ✓ **Automatic learning extraction** — build.md Step 7 extracts phase learnings automatically instead of requiring separate `/ant:continue` call — v4.3

*(Shipped in v4.4 — 2026-02-05)*

- ✓ **Pheromone decay fix** — Three-guard defensive decay (clamp, cutoff, cap) across all pheromone subcommands — v4.4
- ✓ **Activity log append** — cp + >> pattern preserves cross-phase history — v4.4
- ✓ **Error phase attribution** — Optional 4th arg to error-add with regex validation — v4.4
- ✓ **Decision logging** — Two logging points in build.md (strategic + quality) with phase field — v4.4
- ✓ **Same-file conflict prevention** — Two-layer defense: prompt rule + Queen backup validation — v4.4
- ✓ **Safe-to-clear prompting** — build.md, continue.md, colonize.md all end with persistence confirmation — v4.4
- ✓ **Auto-continue mode** — `/ant:continue --all` with Task tool delegation and quality-gated halt — v4.4
- ✓ **Pheromone-first flow** — colonize.md suggests pheromone injections after analysis — v4.4
- ✓ **Multi-ant colonization** — 3 colonizers (Structure/Patterns/Stack) with synthesis and disagreement flagging — v4.4
- ✓ **Adaptive complexity modes** — LIGHTWEIGHT/STANDARD/FULL set at colonization, consumed by build — v4.4
- ✓ **Calibrated watcher scoring** — 5-dimension weighted rubric with anchors and chain-of-thought — v4.4
- ✓ **Aggressive wave parallelism** — DEFAULT-PARALLEL rule with mode-aware limits and auto-approval — v4.4
- ✓ **Advisory reviewer** — Auto-spawns after waves, CRITICAL-only rebuild, reuses watcher-ant.md — v4.4
- ✓ **Auto debugger** — Spawns on retry failure, reuses builder-ant.md with PATCH constraints — v4.4
- ✓ **Pheromone recommendations** — Natural language recs after builds based on outcomes — v4.4
- ✓ **ANSI-colored output** — Caste-specific colors in build and colonize commands — v4.4
- ✓ **Tech debt report** — Generated at project completion from activity log + errors.json — v4.4
- ✓ **Two-tier learning** — Project-local (memory.json) + global (~/.aether/learnings.json) with manual promotion — v4.4
- ✓ **Spawn tree engine** — Queen-mediated delegation with depth-2 limit, all 6 worker specs updated — v4.4
- ✓ **Codebase hygiene** — /ant:organize spawns architect-ant for report-only analysis — v4.4
- ✓ **Pheromone user guide** — When/why to use each signal with 9 scenarios and sensitivity matrix — v4.4

### Active

No active milestone. Run `/cds:new-milestone` to start next version.

### Out of Scope

- **Python CLI/REPL interfaces** — Replaced by Claude-native prompt commands
- **Large bash systems** — v2's 879-line event-bus.sh was too complex; utilities stay thin (<300 lines total)
- **Node.js/Python utility layer** — Shell keeps zero external dependencies
- **Separate /ant:errors command** — Error display integrated into /ant:status
- **Separate /ant:review command** — Review integrated into /ant:continue
- **Separate /ant:memory command** — Memory state shown in /ant:status
- **Separate /ant:adjust command** — Use /ant:focus, /ant:redirect, /ant:feedback directly
- **Separate /ant:recover command** — Recovery integrated into /ant:resume-colony
- **External vector databases** — Using Claude's native semantic understanding
- **Predefined workflows** — Defeats emergence; use phased autonomy instead
- **Code for reasoning/orchestration** — Prompts handle decisions; code handles math
- **GUI/web dashboard** — CLI-only, Claude Code native
- **Persistent daemon processes** — Against Claude-native architecture
- **NPM packaging/distribution** — Deferred until core system stabilizes (field note 16)
- **Deployment model for external repos** — Deferred until core system stabilizes (field note 6)

## Context

### Current State (post v4.4 — 2026-02-05)

**What exists (working):**
- 14 commands as Claude Code skill prompts (init, plan, build, status, phase, continue, focus, redirect, feedback, pause-colony, resume-colony, colonize, organize, ant)
- 6 worker ant specs (~250-560 lines each) with pheromone math, spawning scenarios, event awareness, progress output, mandatory activity logging, SPAWN REQUEST format, scoring rubric (watcher)
- 6 state files: COLONY_STATE.json (with mode + spawn_tree), pheromones.json, PROJECT_PLAN.json, errors.json (with phase attribution), memory.json (with phase-attributed decisions), events.json
- 1 global state file: ~/.aether/learnings.json (cross-project learning promotion)
- `aether-utils.sh` utility wrapper with 18 subcommands (pheromone math, state validation, spawn enforcement, memory ops, error tracking, activity logging, learning promote/inject)
- 2 infrastructure scripts: atomic-write.sh, file-lock.sh
- 1 documentation file: .aether/docs/pheromones.md (user guide with 9 scenarios)
- ANSI-colored build output with caste-specific colors
- Queen-driven execution with advisory reviewer + auto-debugger post-wave
- Auto-continue mode (/ant:continue --all) with quality-gated halt
- Adaptive complexity (LIGHTWEIGHT/STANDARD/FULL) set at colonization
- Multi-colonizer synthesis (3 lenses: Structure/Patterns/Stack)
- Two-tier learning (project-local + global with manual promotion)
- Queen-mediated spawn tree engine (depth-2 limit)
- Tech debt report at project completion
- Codebase hygiene scanning (/ant:organize, report-only)

**Known issues:**
- continue.md Step 6 writes learnings_extracted event unconditionally even when Step 4 was skipped (cosmetic)
- continue.md Step 8 display template lacks skip-case guidance (LLM handles contextually)
- pheromones.md documents global learning FEEDBACK source as 'auto:colonize' but implementation uses 'global:inject' (documentation mismatch)
- validate-state does not check mode/mode_set_at/mode_indicators fields (optional with fallback)
- aether-utils.sh at 369/370 line capacity

**Real-world test results (2026-02-04):**
- Tested on filmstrip packaging project — 5 phases, 21 tasks, 100% completion rate
- All 32 field notes addressed in v4.4 milestone (23 actionable + 9 informational)
- Learning propagation worked cross-phase
- REDIRECT pheromone respected across all phases

### Background

Aether is based on **383,000+ words of research** across 25 documents by Ralph (research agent), covering:
- Multi-agent orchestration patterns
- Semantic communication protocols (AINP, SACP)
- Context engines and memory architecture
- Autonomous spawning research (other systems like AutoGen ADAS, AutoAgents, OpenAI Agents SDK have dynamic agent creation; Aether's contribution is the stigmergic coordination model)
- Verification and quality systems

## Constraints

- **Hybrid Architecture** — Prompts for reasoning/orchestration, shell scripts for deterministic operations
- **JSON State Persistence** — All state in `.aether/data/*.json` files
- **Task Tool for Spawning** — Autonomous spawning uses Claude's Task tool with full spec injection
- **Standalone Architecture** — Aether is its own system, not dependent on CDS or any framework
- **No External Dependencies** — No vector DBs, no embedding services, no Node.js, no Python
- **Shell Utilities Only** — Utility layer uses bash+jq only, stays thin (<400 lines total)
- **Command Enrichment Over Proliferation** — Functionality enriched in existing commands; new commands only when requirements demand (14 commands as of v4.4)
- **Novel Coordination** — Stigmergic pheromone model is Aether's differentiator (spawning concept exists elsewhere; coordination approach is novel)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Claude-native vs Python | Commands work directly in Claude, not separate tool | ✓ Good — 12 commands working |
| Read/Write tools vs bash/jq | Claude manipulates JSON directly, more reliable than shell scripts | ✓ Good — v3 rebuild proved this works |
| Phase Lead emergence | One ant spawned per phase, self-organizes | ✓ Good — true emergence achieved |
| Consolidate commands (19 → 12) | Fewer, richer commands over many thin ones | ✓ Good — cleaner UX |
| Specialist modes vs separate specs | Watcher specializations inside watcher-ant.md, not 4 separate files | ✓ Good — v3.0 |
| JSON state for infrastructure | errors.json, memory.json, events.json as state files | ✓ Good — v3.0 |
| Enrich existing commands vs add new | Fold review/errors/memory into status/continue rather than new commands | ✓ Good — v3.0 |
| Hybrid prompt+code | Prompts for reasoning, shell scripts for deterministic math/validation | ✓ Good — v4.0, 18 subcommands at 241 lines |
| Single wrapper script | aether-utils.sh with subcommands vs separate scripts | ✓ Good — v4.0, clean dispatch pattern |
| Pheromone-based communication | Stigmergic signals (vs direct commands in AutoGen/CrewAI) provide different coordination affordances | ✓ Good — 4 signal types with decay working |
| Standalone system | Aether is its own framework, zero dependencies | ✓ Confirmed |
| Pattern flagging stays LLM responsibility | error-add records, LLM analyzes patterns in context | ✓ Good — v4.0, clear boundary |
| validate-state after init | Catch schema errors immediately after state creation | ✓ Good — v4.0, prevents silent corruption |

| Activity log for live visibility | Workers write to file during execution; Queen reads between spawns | ✓ Good — v4.3, 3 subcommands + all worker specs |
| Auto-learning in build Step 7 | Prevent learning loss from forgotten /ant:continue calls | ✓ Good — v4.3, auto-extraction + duplicate detection |
| Phase Lead as planner only | Separate planning from execution for visibility | ✓ Good — v4.3, Queen drives worker spawns |
| Event-based flag coordination | events.json auto_learnings_extracted for cross-command state | ✓ Good — v4.3, phase-specific matching |
| Three-guard pheromone decay | clamp elapsed>=0, skip >10 half-lives, cap at initial strength | ✓ Good — v4.4, consistent across 3 subcommands |
| Queen-mediated spawn delegation | Workers signal via SPAWN REQUEST blocks, Queen fulfills between waves | ✓ Good — v4.4, avoids platform limitations on subagent Task tool access |
| Caste reuse pattern | Reviewer=watcher, debugger=builder, archivist=architect — no new castes | ✓ Good — v4.4, keeps 6-caste architecture clean |
| Two-tier learning with manual promotion | Project-local + global (~/.aether/learnings.json), 50-entry cap | ✓ Good — v4.4, prevents stale cross-project knowledge |
| Adaptive complexity modes | LIGHTWEIGHT/STANDARD/FULL set at colonization, consumed at 5 build points | ✓ Good — v4.4, scales overhead to project size |
| Auto-continue via Task delegation | Build delegated via Task tool prompt reading build.md, not inlined | ✓ Good — v4.4, avoids prompt duplication |
| ANSI codes only in bash printf | Colors never in LLM text, always in `bash -c 'printf ...'` | ✓ Good — v4.4, reliable terminal rendering |

---

*Last updated: 2026-02-05 after v4.4 milestone completion*
