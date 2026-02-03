# v3.0 Research: What Was Lost and How to Restore It

**Project:** Aether v3.0 — Restore the Soul
**Researched:** 2026-02-03
**Confidence:** HIGH (based on git history analysis of actual deleted code)

## Executive Summary

The v3-rebuild deleted **~30,710 lines** across 73 files to achieve a clean Claude-native architecture (12 commands, 6 worker specs, 3 JSON state files, 2 utility scripts). This was the right decision — the Python runtime, bash event bus, and 600+ line worker specs were incompatible with the Claude Code skill prompt model. But the rebuild removed **capabilities**, not just code. This document catalogs exactly what was lost and maps each loss to the 9 active v3.0 requirements.

**The constraint:** v3.0 must restore these capabilities using Claude-native patterns (Read/Write/Task tools, JSON state, skill prompts) — NOT by resurrecting Python or bash scripts.

---

## Inventory of What Was Deleted

### Python Subsystems (22 files, ~12,976 lines)

| File | Lines | Capabilities Lost |
|------|-------|----|
| `worker_ants.py` | 2,058 | Core worker runtime, spawn execution, caste registry |
| `interactive_commands.py` | 1,275 | Interactive REPL command handlers |
| `cli.py` | 826 | CLI argument parsing, command routing |
| `queen_ant_system.py` | 817 | Queen orchestration, phase coordination |
| `state_machine.py` | 806 | Phase transitions (IDLE→READY→EXECUTING→COMPLETE), guard conditions |
| `phase_engine.py` | 802 | Phase execution, task allocation, progress tracking |
| `pheromone_system.py` | 740 | Signal decay math, caste sensitivity calculations, combination effects |
| `voting_verification.py` | 749 | 4-watcher voting, weighted aggregation, Critical veto, belief calibration |
| `semantic_layer.py` | 738 | Natural language → structured pheromone signal translation |
| `meta_learner.py` | 718 | Alpha/beta Bayesian updates, sample size weighting, specialist recommendations |
| `visualization.py` | 692 | Box-drawing headers, progress bars, agent dashboards, pheromone bars |
| `error_prevention.py` | 686 | 15 error categories, severity levels, error ledger, pattern flagging, prevention rules |
| `short_term_memory.py` | 567 | DAST compression (2.5x ratio), session storage, retrieval |
| `long_term_memory.py` | 538 | Persistent patterns, associative links, knowledge extraction |
| `working_memory.py` | 505 | 200k token budget, eviction policy, relevance scoring |
| `triple_layer_memory.py` | 428 | Working→Short-term→Long-term orchestration, automatic compression triggers |
| `outcome_tracker.py` | 355 | Spawn success/failure tracking, outcome-based learning |
| `repl.py` | 649 | Interactive session (replaced by Claude Code commands — intentional) |
| `demo.py` | 254 | Demo script (not needed) |
| `memory_demo.py` | 285 | Memory demo (not needed) |
| `memory/__init__.py` | 29 | Module init (not needed) |
| `__main__.py` | 11 | Entry point (not needed) |

### Bash Utility Scripts (12 files, ~4,095 lines)

| File | Lines | Capabilities Lost |
|------|-------|----|
| `event-bus.sh` | 890 | Pub/sub event bus, topic subscriptions, delivery tracking, async polling |
| `state-machine.sh` | 527 | Colony state transitions with guard conditions |
| `spawn-decision.sh` | 485 | Multi-factor spawn decision logic, capability gap detection |
| `spawn-tracker.sh` | 335 | Active spawn registry, depth tracking, quota enforcement |
| `checkpoint.sh` | 329 | State snapshots, rollback capability |
| `memory-search.sh` | 280 | Cross-layer memory search with relevance scoring |
| `memory-ops.sh` | 269 | Add/retrieve/compress memory operations |
| `weight-calculator.sh` | 221 | Bayesian weight calculation for watcher reliability |
| `circuit-breaker.sh` | 198 | Cascade failure prevention, cooldown periods |
| `bayesian-confidence.sh` | 195 | Alpha/beta confidence intervals for spawn decisions |
| `issue-deduper.sh` | 187 | Deduplication of reported issues across watchers |
| `vote-aggregator.sh` | 179 | Weighted vote aggregation with Critical veto |

### Specialist Watcher Specs (4 files, ~2,400 lines)

| File | Lines | Capabilities Lost |
|------|-------|----|
| `security-watcher.md` | ~600 | OWASP Top 10 scanning, injection detection, credential exposure |
| `performance-watcher.md` | ~600 | O(n^2) detection, N+1 queries, memory leaks, blocking I/O |
| `quality-watcher.md` | ~600 | Code smells, cyclomatic complexity, naming violations |
| `test-coverage-watcher.md` | ~600 | Missing tests, weak assertions, edge case coverage |

### Deleted Commands (6 files, ~2,228 lines)

| File | Lines | What It Did | Folded Into |
|------|-------|-------------|-------------|
| `execute.md` | 560 | Direct phase execution (separate from build) | `build.md` |
| `errors.md` | 501 | Error inspection and pattern viewing | `status.md` (planned) |
| `adjust.md` | 345 | Colony behavior adjustment | `focus.md`/`redirect.md`/`feedback.md` |
| `review.md` | 314 | Code review at phase boundary | `continue.md` (planned) |
| `memory.md` | 272 | Memory inspection and search | `status.md` (planned) |
| `recover.md` | 236 | Crash recovery | `resume-colony.md` (planned) |

### Worker Spec Reductions (6 files, from ~4,195 to ~556 lines)

| File | Before | After | Lost Content |
|------|--------|-------|---|
| `watcher-ant.md` | 865 | 103 | Event bus polling, 4 specialist modes, severity rubrics, weighted voting, capability gap detection |
| `architect-ant.md` | 804 | 85 | DAST compression algorithms, associative link creation, pattern extraction workflow, memory layer orchestration |
| `scout-ant.md` | 661 | 96 | Research synthesis workflow, source verification, confidence scoring, multi-source triangulation |
| `route-setter-ant.md` | 661 | 101 | Phase dependency analysis, task complexity estimation, resource allocation |
| `builder-ant.md` | 621 | 89 | Code quality patterns, test-driven workflow, implementation verification |
| `colonizer-ant.md` | 583 | 82 | Semantic indexing workflow, pattern detection algorithms, dependency graph construction |

### Command Reductions (12 files, from ~3,739 to ~1,225 lines)

| File | Before | After | Lost Content |
|------|--------|-------|---|
| `status.md` | 456 | 88 | Colony health dashboard, worker activity grouping, pheromone decay bars, error summary, memory usage |
| `build.md` | 446 | 166 | Step progress tracking, visual indicators, multi-wave orchestration |
| `init.md` | 382 | 108 | 7-step visual progress, comprehensive state initialization |
| `phase.md` | 365 | 71 | Detailed task breakdown, dependency visualization |
| `pause-colony.md` | 342 | 85 | Full state serialization with pheromone snapshots |
| `resume-colony.md` | 342 | 65 | State restoration with validation |
| `redirect.md` | 239 | 75 | Colony response visualization |
| `continue.md` | 227 | 68 | Phase review, learning extraction |
| `focus.md` | 217 | 73 | Colony response visualization |
| `ant.md` | 205 | 67 | Detailed workflow guide |
| `feedback.md` | 187 | 76 | Colony response visualization |
| `colonize.md` | 190 | 126 | (Minimal reduction — mostly preserved) |

---

## Mapping Lost Capabilities to v3.0 Requirements

### Requirement 1: Rich Visual Identity
**What was lost:**
- `visualization.py` (692 lines): Box-drawing headers (`╔══╗`, `║  ║`, `╚══╝`), progress bars (`[████████░░]`), agent dashboards, pheromone strength bars
- `status.md` dropped from 456→88 lines: Lost grouped worker display, pheromone decay bars, colony health metrics
- `init.md` dropped from 382→108 lines: Lost 7-step visual progress (`[✓] Step 1/7`, `[→] Step 2/7`)
- `build.md` dropped from 446→166 lines: Lost step progress during phase execution

**What to restore (Claude-native):**
- Box-drawing output templates in command prompts (just markdown formatting instructions)
- Step progress indicators (`[✓]/[→]/[ ]`) in multi-step commands
- Pheromone strength bars computed from decay formula
- Worker activity grouping with emoji status indicators
- No Python needed — commands already compute pheromone decay; just need output formatting instructions

**Source of truth for restoration:**
- `git show HEAD:.claude/commands/ant/status.md` (456-line version has full visual dashboard)
- `git show HEAD:.aether/visualization.py` (box-drawing patterns)

---

### Requirement 2: Specialist Watcher Modes
**What was lost:**
- 4 separate spec files (~2,400 lines total): security-watcher.md, performance-watcher.md, quality-watcher.md, test-coverage-watcher.md
- Each had: severity rubrics (Critical/High/Medium/Low), specific detection patterns, event bus subscriptions, weighted voting profiles

**What to restore (Claude-native):**
- Specialist modes as sections within `watcher-ant.md` (not separate files — per PROJECT.md constraint: "No New Commands")
- Each mode: activation trigger, focus areas, severity rubric, output format
- Mode selection based on pheromone context (e.g., `/ant:focus "security"` activates security mode)
- Watcher spec grows from 103 to ~200 lines with 4 embedded modes

**Source of truth for restoration:**
- `git show HEAD:.aether/workers/security-watcher.md` etc. (extract detection patterns and severity rubrics)
- Keep the mode concept but make it pheromone-activated, not command-activated

---

### Requirement 3: Deep Worker Specs
**What was lost:**
- Worker specs went from ~600-800 lines to ~85-103 lines (87% reduction)
- Lost: pheromone calculation examples showing how sensitivity × strength = effective signal
- Lost: combination effects (what happens when FOCUS + REDIRECT are both active)
- Lost: feedback interpretation (how to adjust behavior based on FEEDBACK pheromone content)
- Lost: event awareness (checking events.json at startup for colony context)
- Lost: detailed spawning scenarios with full Task tool prompt examples

**What to restore (Claude-native):**
- Pheromone math section: `effective_signal = sensitivity[type] × current_strength` with worked examples
- Combination effects section: priority rules when conflicting signals active
- Feedback interpretation section: how to parse FEEDBACK content and adjust workflow
- Event awareness section: Read events.json at startup, react to recent events
- Spawning depth examples: full Task tool prompt showing recursive spec propagation
- Target: ~200 lines per spec (from ~90 now)

**Source of truth for restoration:**
- `git show HEAD:.aether/workers/builder-ant.md` (621-line version)
- `.aether/QUEEN_ANT_ARCHITECTURE.md` (pheromone math, combination rules)
- `pheromone_system.py` (sensitivity profiles already captured in current specs, but math examples lost)

---

### Requirement 4: Error Tracking System
**What was lost:**
- `error_prevention.py` (686 lines): ErrorCategory enum (15 categories), ErrorSeverity levels, ErrorRecord dataclass, ErrorLedger with JSON persistence, pattern flagging after 3 occurrences, prevention rule creation
- `errors.md` command (501 lines): Error inspection, pattern viewing, severity filtering
- Error lifecycle: Error occurs → Logged → Root cause analyzed → Prevention created → Constraint enforced

**What to restore (Claude-native):**
- `errors.json` in `.aether/data/`: Array of error records `{id, category, severity, description, root_cause, phase, timestamp, recurrence_count}`
- Error logging integrated into `build.md` (when phase encounters errors)
- Error display integrated into `status.md` (show recent errors, flagged patterns)
- Pattern flagging: when same category hits 3 occurrences, flag for prevention
- No separate command — per constraint, fold into existing commands

**Source of truth for restoration:**
- `git show HEAD:.aether/error_prevention.py` (ErrorCategory enum, ErrorRecord schema)

---

### Requirement 5: Colony Memory
**What was lost:**
- Triple-layer memory system (5 Python files, ~2,543 lines total):
  - Working memory (200k token budget, eviction policy)
  - Short-term memory (10 sessions, DAST 2.5x compression)
  - Long-term memory (persistent patterns, associative links)
  - Orchestrator (automatic compression triggers at phase boundaries)
  - Meta-learner (Bayesian updates, specialist recommendations)

**What to restore (Claude-native):**
- `memory.json` in `.aether/data/`: `{phase_learnings: [], decisions: [], patterns: []}`
- Phase learnings: extracted at phase boundaries by `continue.md` (key decisions, patterns found, anti-patterns avoided)
- Decision history: logged by commands when significant choices made
- Pattern recognition: Architect ant extracts patterns during memory compression
- Claude's native context IS the working memory — no need to duplicate
- DAST-style compression: Architect ant summarizes verbose learnings into concise entries

**Source of truth for restoration:**
- `git show HEAD:.aether/memory/triple_layer_memory.py` (compression trigger logic)
- `.ralph/MEMORY_ARCHITECTURE_RESEARCH.md` (design rationale)

---

### Requirement 6: Event Awareness
**What was lost:**
- `event-bus.sh` (890 lines): Full pub/sub system with topic subscriptions and delivery tracking
- Event polling sections in all 10 worker specs
- Integration test suite (298 lines)

**What to restore (Claude-native):**
- `events.json` in `.aether/data/`: Array of `{id, type, source, content, timestamp}`
- Event types: `phase_complete`, `error`, `spawn`, `task_complete`, `pheromone_emitted`
- Commands Write events: init.md writes `colony_initialized`, build.md writes `phase_started`/`phase_complete`, error events on failures
- Workers Read events at startup: "Check events.json for what happened since your last execution"
- No delivery tracking needed — events are a log, not a queue. Workers check timestamps.
- Simple and Claude-native: Read the file, filter by timestamp, act on relevant events

**Source of truth for restoration:**
- `git show HEAD:.aether/utils/event-bus.sh` (event schema, topic types)
- v2 research in `.planning/research/ARCHITECTURE.md` (polling patterns)

---

### Requirement 7: Enhanced Status Dashboard
**What was lost:**
- `status.md` dropped from 456→88 lines
- Lost: worker activity grouping (Active/Idle/Error workers shown separately)
- Lost: pheromone strength visualization with decay bars
- Lost: error summary section
- Lost: memory usage section
- Lost: phase progress with visual completion bar

**What to restore (Claude-native):**
- Enrich `status.md` from 88 back to ~200+ lines with:
  - Colony header with box drawing
  - Worker section: grouped by status (active/idle/error) with emoji
  - Pheromone section: each active signal with computed decay bar
  - Phase section: current phase progress with task completion
  - Error section: recent errors, flagged patterns (reads errors.json)
  - Memory section: recent learnings (reads memory.json)
  - Event section: recent colony events (reads events.json)
  - Next actions section: contextual routing

**Source of truth for restoration:**
- `git show HEAD:.claude/commands/ant/status.md` (456-line version with full dashboard)

---

### Requirement 8: Phase Review in Continue
**What was lost:**
- `continue.md` dropped from 227→68 lines
- `review.md` deleted (314 lines): showed what was built, test results, quality assessment
- Lost: phase summary showing tasks completed, files changed, decisions made
- Lost: learning extraction (what worked, what didn't)

**What to restore (Claude-native):**
- Enrich `continue.md` to show phase review BEFORE advancing:
  - Read current phase tasks from PROJECT_PLAN.json
  - Show completion status for each task
  - Show key decisions made (from memory.json)
  - Show errors encountered (from errors.json)
  - Extract and store learnings to memory.json
  - THEN advance phase and clean expired pheromones

**Source of truth for restoration:**
- `git show HEAD:.claude/commands/ant/continue.md` (227-line version)
- `git show HEAD:.claude/commands/ant/review.md` (review logic to fold in)

---

### Requirement 9: Spawn Outcome Tracking
**What was lost:**
- `outcome_tracker.py` (355 lines): Track which specialist spawns succeed/fail
- `spawn-tracker.sh` (335 lines): Active spawn registry, depth tracking
- `bayesian-confidence.sh` (195 lines): Alpha/beta confidence intervals
- `meta_learner.py` (718 lines): Bayesian updates, sample size weighting

**What to restore (Claude-native):**
- Add `spawn_outcomes` field to COLONY_STATE.json: `{caste: {spawns: N, successes: N, alpha: N, beta: N}}`
- When build.md spawns a Phase Lead, record the spawn
- When phase completes (continue.md), record success/failure
- Confidence calculation: `confidence = alpha / (alpha + beta)` — simple enough to compute inline
- Workers check spawn history before spawning: "Colonizer has 0.8 confidence (4 successes, 1 failure)"
- No separate utility script needed — the math is one line

**Source of truth for restoration:**
- `git show HEAD:.aether/memory/outcome_tracker.py` (schema and Bayesian update logic)
- `git show HEAD:.aether/utils/bayesian-confidence.sh` (alpha/beta formulas)

---

## Implementation Approach: Claude-Native Patterns

### Pattern 1: State as JSON
All new capabilities store state in `.aether/data/*.json` files:
- `errors.json` — Error tracking (Req 4)
- `memory.json` — Colony memory (Req 5)
- `events.json` — Event log (Req 6)
- `COLONY_STATE.json` — Extended with spawn_outcomes field (Req 9)

### Pattern 2: Commands as Enrichment
No new commands. Existing 12 commands get richer:
- `status.md` — Grows to show errors, memory, events, full dashboard (Req 1, 7)
- `continue.md` — Shows phase review before advancing (Req 8)
- `build.md` — Logs events and errors during execution (Req 4, 6)
- `init.md` — Creates all JSON state files (Req 4, 5, 6)

### Pattern 3: Worker Specs as Knowledge
Worker specs grow with domain knowledge, not code:
- Pheromone math examples (Req 3)
- Specialist watcher modes in watcher-ant.md (Req 2)
- Event awareness instructions (Req 6)
- Spawn outcome checking (Req 9)

### Pattern 4: Visual Output as Prompt Instructions
Box drawing, progress bars, emoji status — all just output formatting instructions in command prompts:
- "Display the following box-drawing header..."
- "Show pheromone strength as: `[━━━━━━━━░░] 0.8`"
- "Group workers by status: 🟢 Active first, then ⚪ Idle, then 🔴 Error"

---

## Risk Assessment

### What Should NOT Be Restored
- **Python runtime** — Replaced by Claude-native commands. Correct decision.
- **Bash event bus (890 lines)** — Replaced by simple events.json log. Correct decision.
- **Bash spawning infrastructure** — Replaced by Task tool. Correct decision.
- **REPL interface** — Replaced by Claude Code commands. Correct decision.
- **Separate specialist watcher files** — Fold into watcher-ant.md modes. Correct decision.
- **6 deleted commands** — Fold functionality into remaining 12. Correct decision.

### What MUST Be Restored (capabilities, not code)
1. Visual identity makes the system feel alive and professional
2. Specialist watcher modes are core to multi-perspective verification
3. Deep worker specs enable true autonomous behavior
4. Error tracking prevents repeated mistakes
5. Colony memory enables learning across sessions
6. Event awareness enables reactive coordination
7. Status dashboard makes emergence visible
8. Phase review ensures quality gates work
9. Spawn outcome tracking enables meta-learning

### Pitfalls to Avoid
- **Don't over-engineer JSON schemas** — Start minimal, grow as needed
- **Don't add bash scripts** — Everything through Read/Write tools
- **Don't create new commands** — Enrich existing 12
- **Don't exceed ~200 lines per worker spec** — Trim content, not cut it
- **Don't make status.md a monolith** — Keep sections modular in the prompt

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| What was lost | HIGH | Verified via git history, exact line counts |
| What to restore | HIGH | Mapped 1:1 to 9 active requirements |
| How to restore | HIGH | Claude-native patterns proven in v3-rebuild |
| Risk of over-engineering | MEDIUM | Need discipline to keep it simple |

**Overall confidence:** HIGH — Ready for roadmap creation.

---

*Research completed: 2026-02-03*
*Ready for roadmap: yes*
*Recommended next step: `/cds:new-milestone` or create ROADMAP.md for v3.0 phases*
