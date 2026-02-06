# Aether: Claude-Native Queen Ant Colony

## What This Is

Aether is a **standalone multi-agent system** that applies ant colony intelligence to autonomous agent orchestration, built natively for Claude Code. Worker Ants spawn other Worker Ants through bio-inspired pheromone signaling. The Queen (user) provides high-level intention via pheromone signals (INIT, FOCUS, REDIRECT, FEEDBACK), and the colony self-organizes to complete tasks through emergent intelligence.

This is a **hybrid prompt+code system** — commands like `/ant:init "Build a REST API"` work directly in Claude Code as skill prompts. Prompts handle reasoning and orchestration; a thin shell utility layer (`aether-utils.sh`, 87 lines, 5 subcommands) handles deterministic operations (state validation, error tracking, activity logging) that LLMs get wrong.

**What makes it different:**

Autonomous agent spawning is not unique to Aether — systems like AutoGen (ADAS/Meta Agent Search), AutoAgents, and OpenAI's Agents SDK support dynamic agent creation. What Aether does differently is the coordination model:

1. **Stigmergic Communication** — Pheromone signaling with TTL-based expiration and priority levels (not direct commands or message passing)
2. **Caste Architecture** — Six Worker Ant types with specialist watcher modes, consolidated in single workers.md
3. **Output-as-State** — SUMMARY.md existence signals completion; state survives context boundaries
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

*(Shipped in v5.0 — 2026-02-05)*

- ✓ **Path Migration** — All 14 commands and 6 worker specs reference `~/.aether/` for global resources, `.aether/data/` for per-project state — v5.0
- ✓ **DATA_DIR Fix** — `aether-utils.sh` uses `$PWD/.aether/data` instead of `$SCRIPT_DIR/data` — v5.0
- ✓ **NPM Package** — `package.json`, `bin/cli.js` with install/version/uninstall, `.npmignore` — v5.0
- ✓ **Postinstall** — `npm install -g` auto-copies commands to `~/.claude/commands/ant/` and runtime to `~/.aether/` — v5.0
- ✓ **Cosmetic Fixes** — continue.md learnings_extracted guard, pheromones.md source field, validate-state mode fields — v5.0
- ✓ **Documentation** — README.md with install/uninstall instructions, file structure updated for global/local split — v5.0

*(Shipped in v5.1 — 2026-02-06)*

- ✓ **State Consolidation** — 6 JSON files merged into single COLONY_STATE.json (103 refs across 15 commands) — v5.1
- ✓ **State Update Timing** — State updates at start-of-next-command; state survives context boundaries — v5.1
- ✓ **TTL Pheromones** — Simple expires_at timestamps replace decay math; priority levels replace sensitivity matrices — v5.1
- ✓ **Worker Spec Collapse** — 6 worker specs (1,866 lines) merged into single workers.md (171 lines, 91% reduction) — v5.1
- ✓ **Command Shrinking** — build.md 1,080→414 lines (62%), continue.md 534→111 lines (79%) — v5.1
- ✓ **Utility Reduction** — aether-utils.sh 372→87 lines (77% reduction) — v5.1
- ✓ **Output-as-State** — SUMMARY.md existence = phase complete; passive detection by continue.md — v5.1

### Active

*(No active requirements — ready for next milestone)*

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
- ~~**NPM packaging/distribution**~~ — Shipped in v5.0
- ~~**Deployment model for external repos**~~ — Shipped in v5.0 (global install + per-project state)

## Context

### Current State (post v5.0 — 2026-02-06)

**What exists:**
- 14 commands as Claude Code skill prompts (~7,400 lines total)
- 6 worker ant specs (1,866 lines combined)
- 6 state files that must stay consistent
- `aether-utils.sh` (369 lines, 18 subcommands)
- NPM package for global installation

**Real-world test results (M4L-AnalogWave, 2026-02-05):**
- Phase 1 completed successfully — all 5 tasks produced output
- **State fell out of sync** — PROJECT_PLAN.json still showed "pending" despite completed work
- **~70% of context spent on framework overhead** — ceremony-to-work ratio extreme
- **State updates at END of commands get dropped** at context boundaries
- Postmortem verdict: "Core ideas sound, implementation over-engineered by ~4x"

**What to preserve (per postmortem):**
- Parallel task decomposition (Wave 1 → Wave 2)
- Structured phase output (`.planning/phase-N/`)
- Worker role concept (6 castes)
- Signal/constraint concept (simplified)
- Quality gates
- Pause/resume capability
- Phase-based planning

**Anti-patterns to eliminate:**
- Critical state writes at end of long operations
- Distributed state across multiple files
- Mathematical models (Bayesian, exponential) where simple rules suffice
- Prescribing exact output formatting in commands
- Shell round-trips for trivial operations
- Tracking metrics before enough data exists

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
- **Shell Utilities Only** — Utility layer uses bash+jq only, stays thin (<100 lines total after v5.1)
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

*Last updated: 2026-02-06 after v5.1 milestone start*
