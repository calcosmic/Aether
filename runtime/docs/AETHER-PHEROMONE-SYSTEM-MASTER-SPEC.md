# Aether Pheromone & Multi-Colony System
## Master Implementation Specification

**Version:** 1.1
**Status:** Design Complete → Implementation Ready
**Created:** 2026-02-15
**Updated:** 2026-02-15
**Purpose:** Cross-agent implementation guide for the complete Aether pheromone memory and multi-colony architecture

---

## CONSOLIDATION NOTE

This document consolidates all previous pheromone and multi-colony design notes:
- `PHEROMONE-SYSTEM-DESIGN.md` → Core philosophy, taxonomy, phases
- `PHEROMONE-INJECTION.md` → Injection timing, queue system, UX flows
- `PHEROMONE-INTEGRATION.md` → Command integration patterns
- `MULTI-COLONY-ARCHITECTURE.md` → Registry, lineage, switching

**This is the single source of truth. Previous documents are archived references.**

---

## Document Purpose

This is the **single source of truth** for implementing the Aether Pheromone System and Multi-Colony Architecture. Multiple agents will work from this document across different phases. Each section includes:
- Clear specifications
- Implementation details
- Agent assignment markers
- Dependencies
- Verification criteria

**For Agents Reading This:**
- Look for `AGENT ASSIGNMENT` markers to find your work
- Check `DEPENDENCIES` before starting
- Verify against `ACCEPTANCE CRITERIA` before marking complete
- Update `STATUS` as you progress

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Core Philosophy](#2-core-philosophy)
3. [Pheromone Taxonomy](#3-pheromone-taxonomy)
   - 3.1 Type Reference
   - 3.2 Strength & Decay
   - 3.3 JSON Schema
   - 3.4 Injection Timing & Critical Points
   - 3.5 Mid-Work Injection System
   - 3.6 Worker Spawn-Time Priming
   - 3.7 Biological Patterns
4. [Multi-Colony Architecture](#4-multi-colony-architecture)
   - 4.1 Directory Structure
   - 4.2 Colony Registry
   - 4.3 Colony Lifecycle
   - 4.4 Lineage & Inheritance
   - 4.5 Concurrent Colony Execution
5. [Phase 1: Foundation](#5-phase-1-foundation)
6. [Phase 2: Integration](#6-phase-2-integration)
7. [Phase 3: Learning](#7-phase-3-learning)
8. [Phase 4: Colonize](#8-phase-4-colonize)
9. [Phase 5: Lineage](#9-phase-5-lineage)
10. [Command Specifications](#10-command-specifications)
   - 10.1 Command Matrix
   - 10.2 Detailed Specs
   - 10.3 Complete UX Flows
11. [File Specifications](#11-file-specifications)
12. [Testing Strategy](#12-testing-strategy)
13. [Appendices](#13-appendices)
14. [Advanced Multi-Colony Concepts](#14-advanced-multi-colony-concepts)
   - 14.1 Colony Networks & Swarms
   - 14.2 Colony Templates
   - 14.3 Colony Forking
   - 14.4 Colony Merging
15. [Hook System](#15-hook-system)
   - 15.1 Event Hooks
   - 15.2 Hook Configuration
   - 15.3 Webhook Integration
16. [Plugin Architecture](#16-plugin-architecture)
   - 16.1 Plugin System Design
   - 16.2 Plugin Hooks
   - 16.3 Plugin Commands
17. [Custom Agent Design](#17-custom-agent-design)
   - 17.1 Agent Definition Framework
   - 17.2 Agent Inheritance
   - 17.3 Agent Swarming
18. [Best Practices & Patterns](#18-best-practices--patterns)
   - 18.1 Anti-Patterns
   - 18.2 Best Practices
   - 18.3 Colony Sizing Guidelines
19. [Emergent Behaviors Philosophy](#19-emergent-behaviors-philosophy)
   - 19.1 From Control to Emergence
   - 19.2 Stigmergy in Code
   - 19.3 Trophallaxis
   - 19.4 Midden Pile
   - 19.5 Quorum Sensing
20. [Future Possibilities](#20-future-possibilities)
   - 20.1 AI-Native Features
   - 20.2 Integration Ideas
   - 20.3 Visualization Enhancements
   - 20.4 Advanced Agent Concepts
21. [Implementation Roadmap (Extended)](#21-implementation-roadmap-extended)
22. [Quick Start Recipes](#22-quick-start-recipes)
   - Appendix A: Quick Reference
   - Appendix B: Migration Guide
   - Appendix C: Troubleshooting
   - Appendix D: Glossary

---

## 1. Executive Summary

### 1.1 What We're Building

The Aether Pheromone System transforms the colony from a single-session tool into a **learning organism** that:
- Remembers patterns across colonies
- Guides workers through invisible scent trails
- Evolves through biological metaphors (stimulus-driven, trophallaxis, midden)
- Supports multiple coordinated colonies per repository

### 1.2 System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    AETHER ECOSYSTEM                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   Colony A   │◄──►│ Eternal Mem  │◄──►│   Colony B   │      │
│  │ (Bug Fixes)  │    │  (~/.aether) │    │(Pheromone    │      │
│  │              │    │              │    │ Foundation)  │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
│         ▲                                    ▲                   │
│         │         ┌──────────────┐           │                   │
│         └────────►│  Pheromone   │◄──────────┘                   │
│                   │    Trails    │                               │
│                   │              │                               │
│                   │ • FOCUS      │                               │
│                   │ • REDIRECT   │                               │
│                   │ • PATTERN    │                               │
│                   │ • PHILOSOPHY │                               │
│                   │ • STACK      │                               │
│                   │ • DECREE     │                               │
│                   └──────────────┘                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 Five Implementation Phases

| Phase | Name | Duration | Key Deliverable | Status |
|-------|------|----------|-----------------|--------|
| 1 | Foundation | 1-2 days | Eternal memory structure, basic auto-deposition | 🟡 PLANNED |
| 2 | Integration | 2-3 days | Queue system, command enhancements, visibility | 🟡 PLANNED |
| 3 | Learning | 3-4 days | Pattern detection, milestone wisdom extraction | 🟡 PLANNED |
| 4 | Colonize | 2-3 days | Deep codebase analysis, stack profiles | 🟡 PLANNED |
| 5 | Lineage | 2-3 days | Cross-colony inheritance, mentor system | 🟡 PLANNED |

**Total Estimated Time:** 10-15 days with parallel work

### 1.4 Agent Coordination

```
AGENT COORDINATION MAP
======================

Phase 1 (Foundation)
├── Agent A: Eternal directory structure
├── Agent B: Pheromone JSON schema
└── Agent C: Auto-deposition in /ant:init

Phase 2 (Integration) - STARTS AFTER Phase 1 complete
├── Agent D: Queue system
├── Agent E: Command enhancements
└── Agent F: Status visibility

Phase 3 (Learning) - STARTS AFTER Phase 2 complete
├── Agent G: Pattern detection
└── Agent H: Milestone extraction

Phase 4 (Colonize) - CAN PARALLEL with Phase 3
└── Agent I: /ant:colonize command

Phase 5 (Lineage) - STARTS AFTER Phase 3 & 4 complete
└── Agent J: Cross-colony system
```

---

## 2. Core Philosophy

### 2.1 The Metaphor

Real ant colonies communicate through **pheromones**—chemical trails that:
- Persist in the environment
- Fade over time (decay)
- Guide behavior without central control
- Create emergent intelligence

Our system replicates this:
- **FOCUS** pheromones attract attention to important areas
- **REDIRECT** pheromones warn away from anti-patterns
- **PATTERN** pheromones mark validated best practices
- **PHILOSOPHY** pheromones encode core beliefs (never decay)
- **STACK** pheromones capture tech constraints
- **DECREE** pheromones record explicit commands

### 2.2 Design Principles

1. **Emergence over Orchestration**
   - Don't tell workers what to do—let them follow scent trails
   - System learns from behavior, not explicit programming

2. **Invisibility**
   - Pheromones operate in the background
   - Visible only when relevant or requested
   - No new commands for basic operation

3. **Decay & Renewal**
   - Old patterns fade (configurable decay)
   - Validated patterns strengthen
   - Failed patterns go to midden (archive)

4. **Cross-Colony Memory**
   - What Colony A learns, Colony B can smell
   - Eternal memory persists beyond any single colony
   - Lineage creates inheritance chains

5. **Mid-Work Injection**
   - Pheromones can be added while workers are active
   - Queue system prevents interruption
   - Workers pick up new scents at natural breakpoints

### 2.3 User Experience

**First Colony:**
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

**Mid-Work Pheromone:**
```
🐜 FOCUS TRAIL LAID
═══════════════════════════════════════════════════════

Queued for active workers.
Next checkpoint: workers will detect.

Queued: FOCUS[error-handling, 0.9]
Active: FOCUS[auth, 0.9] REDIRECT[regex, 0.7]
═══════════════════════════════════════════════════════
```

**Milestone Completion:**
```
🐜 WISDOM ARCHIVED
═══════════════════════════════════════════════════════

Patterns validated:
  • PATTERN[prefer-joi-over-zod] (validated 5x)
  • PATTERN[bash-for-file-ops] (used 12x)
  • REDIRECT[avoid-sync-fs] (failed once)

Preserved in eternal memory for future colonies.
═══════════════════════════════════════════════════════
```

---

## 3. Pheromone Taxonomy

### 3.1 Complete Type Reference

| Type | Source | Decay | Strength Range | Purpose | Example |
|------|--------|-------|----------------|---------|---------|
| **FOCUS** | `/ant:focus`, planning | 30 days | 0.1 - 1.0 | What to pay attention to | FOCUS[authentication] |
| **REDIRECT** | `/ant:redirect`, errors | 60 days | 0.1 - 1.0 | Patterns to avoid | REDIRECT[regex-parsing] |
| **PHILOSOPHY** | Queen's Will, Eternal | Never | 0.5 - 1.0 | Core beliefs | PHILOSOPHY[emergence-over-orchestration] |
| **STACK** | `/ant:colonize`, detection | When stack changes | 0.7 - 1.0 | Tech constraints | STACK[nodejs-bash-jq] |
| **PATTERN** | Milestones, workers | 90 days | 0.3 - 1.0 | Discovered best practices | PATTERN[bash-for-file-ops] |
| **DECREE** | User override | Never | 1.0 | Explicit commands | DECREE[no-force-push] |

### 3.2 Strength Interpretation

```
0.9 - 1.0 : Overwhelming scent - workers will strongly follow
0.7 - 0.8 : Strong trail - workers will likely follow
0.5 - 0.6 : Moderate scent - workers will consider
0.3 - 0.4 : Faint trail - workers may notice
0.1 - 0.2 : Barely perceptible - background influence
```

### 3.3 Decay Mechanics

```python
# Pseudocode for decay calculation
def calculate_current_strength(pheromone):
    age = now - pheromone.deposited_at
    decay_rate = pheromone.decay  # e.g., "30d"

    # Linear decay to 0
    decayed = pheromone.strength * (1 - (age / decay_rate))
    return max(0, decayed)

# Daily decay job (runs on /ant:status or /ant:init)
def apply_decay():
    for pheromone in pheromones:
        current = calculate_current_strength(pheromone)
        if current < 0.1:
            move_to_midden(pheromone)  # Archive, don't delete
```

### 3.4 Pheromone JSON Schema

```json
{
  "$schema": "pheromone-v1.0",
  "trails": [
    {
      "id": "phem_{uuid}",
      "type": "FOCUS|REDIRECT|PHILOSOPHY|STACK|PATTERN|DECREE",
      "substance": "string-identifier",
      "strength": 0.0-1.0,
      "source": {
        "type": "user:focus|user:redirect|milestone|worker:error|worker:success|decree",
        "colony_id": "colony-uuid",
        "phase": 3,
        "command": "/ant:focus"
      },
      "deposited_at": "ISO-8601",
      "decay": "30d|60d|90d|never",
      "decays_at": "ISO-8601",
      "context": {
        "description": "Human-readable explanation",
        "evidence": ["list", "of", "supporting", "facts"],
        "applies_to": ["file patterns", "or", "domains"]
      }
    }
  ],
  "metadata": {
    "version": "1.0",
    "last_decay_check": "ISO-8601",
    "total_deposited": 47,
    "active_count": 12
  }
}
```

---

### 3.5 Detailed Injection Timing & Critical Points

**Pheromones are deposited at 7 critical moments:**

| Moment | Auto-Pheromone | Signal to User | Purpose |
|--------|---------------|----------------|---------|
| `/ant:init` completes | PHILOSOPHY[emergence-over-orchestration] | "🐜 Queen's scent laid: Emergence" | Foundation |
| First `/ant:plan` starts | PHILOSOPHY[minimal-planning-maximum-doing] | "🐜 Trail: Plan just enough, build soon" | Planning bias |
| Worker encounters error | REDIRECT[error-pattern] | "🐜 Warning pheromone deposited: {pattern}" | Learn from failure |
| Phase completes | PATTERN[what-worked] | "🐜 Success trail: {pattern} (strength: {n})" | Reinforce success |
| `/ant:seal` invoked | PHILOSOPHY[maturity-{milestone}] + PATTERN[sealed-conventions] | "🐜 Colony wisdom archived to eternal memory" | Lineage |
| `/ant:swarm` fixes bug | PATTERN[fix-strategy] + REDIRECT[what-failed] | "🐜 Swarm left trail: {solution-type}" | Bug immunity |
| User overrides worker | DECREE[override-reason] | "🐜 Decree recorded: {reason}" | Authority |

**Signal Batching Rules:**
- Batch notifications every 30 seconds if multiple deposited
- Priority: DECREE > REDIRECT > FOCUS > PATTERN > PHILOSOPHY
- Show immediately if strength >= 0.8

**User Signaling Format:**
```
═══════════════════════════════════════════════════════
  🐜 PHEROMONE DEPOSITED
═══════════════════════════════════════════════════════

  Type:     PATTERN
  Substance: prefer-bash-over-node-for-file-ops
  Strength:  0.7
  Source:    milestone:phase-3-complete
  Why:       Workers used bash 5x more than node for file ops

  This trail guides future workers.
  To see all active trails: /ant:sniff

═══════════════════════════════════════════════════════
```

---

### 3.6 Mid-Work Injection System (Queue + Checkpoints)

**The Challenge:** User wants to inject pheromone while workers are active, without interrupting flow.

**Solution: Pheromone Queue + Checkpoint Polling**

```
User: /ant:emit FOCUS "error-handling" (while workers building)

System:
1. Deposit pheromone immediately to .aether/data/pheromone-queue.json
2. Display: "🐜 Pheromone queued - workers will detect at next checkpoint"
3. Worker flow continues uninterrupted

Workers:
- Poll for queue at natural breakpoints
- Pick up without breaking context
- Log: "[Worker] New scent: FOCUS[error-handling]"
```

**Queue Structure:**
```json
{
  "version": "1.0",
  "queue": [
    {
      "id": "phem_queued_123",
      "type": "FOCUS",
      "substance": "error-handling",
      "strength": 0.9,
      "deposited_at": "2026-02-15T10:30:00Z",
      "deposited_by": "user:queen",
      "status": "queued|picked_up|applied",
      "picked_up_by": null,
      "picked_up_at": null
    }
  ]
}
```

**Worker Checkpoint Protocol:**
Natural breakpoints for pheromone detection:
- After completing a task
- Before spawning a sub-worker
- After tool use (Read/Edit/Bash)
- Every 5 minutes of continuous work

**Worker Prompt Addition:**
```markdown
Check for new pheromones at each checkpoint:
```bash
bash .aether/aether-utils.sh pheromone-poll "{worker_name}"
```

If new scents detected, announce:
"🐜 [Worker-Name] New scent detected: FOCUS[error-handling]"
```

---

### 3.7 Worker Spawn-Time Priming

Every worker, at spawn:

1. **Inhales the eternal** - Reads `~/.aether/eternal/queen-will.md`
2. **Checks active trails** - Loads `.aether/data/pheromones.json`
3. **Smells the stack** - Loads relevant stack-profile
4. **Polls queue** - Checks for queued pheromones
5. **Begins work** - Carries context, may deposit new trails

**Example spawn log:**
```
[Prime Worker] Spawned for Phase 7
[Prime Worker] Inhaling Queen's Will... (emergence, minimal-change)
[Prime Worker] Trails: FOCUS[security,0.9] REDIRECT[force-push,0.8]
[Prime Worker] Stack: nodejs-bash-jq (matches eternal profile)
[Prime Worker] Ready to coordinate
```

**Worker Context Header (condensed):**
```
[Pheromones: FOCUS(auth,0.9) REDIRECT(regex,0.7) PATTERN(bash-files,0.8)]
[Stack: nodejs-bash-jq]
[Queen's Will: emergence, minimal-change]
```

---

### 3.8 Biological Patterns to Implement

Real ant colonies exhibit behaviors we replicate in software:

#### Stimulus-Driven Worker Assignment
Workers check pheromone gradients before spawning specialists:
- High REDIRECT[regex-parsing] → spawn Parser Specialist
- High FOCUS[performance] → spawn Watcher before Builder
- PATTERN[error-recovery] detected → Chaos ant joins automatically

```
Queen spawning workers for Phase 3:

Checks pheromone gradients:
  REDIRECT[regex-parsing]: 0.8 (HIGH)
  → Spawn Parser Specialist alongside Builder

  FOCUS[performance]: 0.9 (HIGH)
  → Spawn Watcher before Builder

  No error patterns detected
  → Chaos ant not needed yet
```

#### Trophallaxis (Food Sharing)
Senior workers pass "digested learnings" to new workers:
- `/ant:mentor` - Explicitly share chamber wisdom to new colony
- Auto-mentoring: New colonies inherit from sealed chambers
- Workers in same colony share patterns via activity log

```
[Colony A - Sealed]
     ↓ (extract pheromones)
[PATTERN[auth-pattern]] → Eternal Memory
     ↓ (inherit)
[Colony B - New]
[Worker] Inhaling: PATTERN[auth-pattern] from Colony A
```

#### Midden Piles (Waste Management)
Failed patterns go to `.aether/data/midden/`:
- Not deleted, just marked as "don't go here"
- REDIRECT pheromones point to midden
- Historical record of what didn't work

```
Failed Pattern Archive (Midden)
═══════════════════════════════════════════════════════

REDIRECT[sync-fs-ops] (strength: 0.9)
  Why: Caused race conditions in 3 colonies
  First failed: 2026-01-15
  Last attempted: 2026-02-10

REDIRECT[nested-callbacks] (strength: 0.7)
  Why: Created unreadable code
  Recommendation: Use promises/async-await

═══════════════════════════════════════════════════════
```

#### Trail Following
Workers follow strongest pheromone trails:
- FOCUS trails attract attention
- REDIRECT trails create avoidance zones
- PATTERN trails create preferred paths

```
Worker navigating codebase:

[Auth Module]
   ↑ FOCUS[authentication, 0.9] - Strong trail
   → Enter module

[Database Layer]
   ⚠️ REDIRECT[sync-queries, 0.8] - Avoid zone
   → Route around, use async

[Utils]
   ↗ PATTERN[bash-for-file-ops, 0.8] - Preferred path
   → Use bash instead of node
```

---

## 4. Multi-Colony Architecture

### 4.1 Directory Structure

```
Repository Root
├── .aether/
│   ├── colonies/                          # NEW: All colony states
│   │   ├── registry.json                  # Master colony index
│   │   ├── colony-{id}-{slug}/            # Individual colony
│   │   │   ├── COLONY_STATE.json
│   │   │   ├── activity.log
│   │   │   ├── constraints.json           # Colony-local pheromones
│   │   │   ├── pheromones.json            # Colony-specific trails
│   │   │   ├── completion-report.md
│   │   │   └── events/
│   │   │       └── {timestamp}-{type}.json
│   │   └── colony-{id}-{slug}/            # Another colony
│   │       └── ...
│   ├── data/                              # Legacy (symlink to active)
│   │   └── COLONY_STATE.json ─────────────┐
│   ├── eternal/                           # NEW: Cross-colony memory
│   │   ├── queen-will.md                  # Core philosophies
│   │   ├── pheromones.json                # Active eternal pheromones
│   │   ├── stack-profile/                 # Tech fingerprints
│   │   │   ├── nodejs.md
│   │   │   └── python.md
│   │   ├── patterns/                      # Validated patterns
│   │   │   └── emergence.md
│   │   └── lineage/                       # Inheritable chambers
│   │       └── colony-{id}/
│   │           ├── decisions.md
│   │           ├── lessons.md
│   │           └── pheromones.json
│   └── ...                                # Other existing files
└── ...
```

### 4.2 Colony Registry Format

```json
{
  "version": "1.0",
  "repository": "/absolute/path/to/repo",
  "current_colony": "20260215-pheromone-foundation-x2m4",
  "last_updated": "ISO-8601",
  "colonies": [
    {
      "id": "20260215-fix-loop-bugs-a7k9",
      "slug": "fix-loop-bugs",
      "goal": "v1.1 Bug Fixes & Update System Repair",
      "status": "sealed",
      "mound": "aether-v1.1",
      "created_at": "2026-02-13T20:40:00Z",
      "sealed_at": "2026-02-14T02:39:00Z",
      "phases_completed": 7,
      "lineage": {
        "parent": null,
        "children": ["20260215-pheromone-foundation-x2m4"],
        "siblings": []
      },
      "pheromones": {
        "extracted": 12,
        "philosophy": ["emergence", "minimal-change"],
        "patterns": ["hash-comparison", "sync-verification"]
      },
      "path": ".aether/colonies/20260215-fix-loop-bugs-a7k9/"
    },
    {
      "id": "20260215-pheromone-foundation-x2m4",
      "slug": "pheromone-foundation",
      "goal": "Phase 1: Foundation — Eternal memory structure",
      "status": "active",
      "mound": "pheromone-system",
      "created_at": "2026-02-15T10:30:00Z",
      "sealed_at": null,
      "phases_completed": 0,
      "lineage": {
        "parent": "20260215-fix-loop-bugs-a7k9",
        "children": [],
        "siblings": []
      },
      "pheromones": {
        "inherited": ["emergence", "minimal-change"]
      },
      "path": ".aether/colonies/20260215-pheromone-foundation-x2m4/"
    }
  ],
  "mounds": [
    {
      "name": "aether-v1.1",
      "description": "Bug fixes and update system repair",
      "colonies": ["20260215-fix-loop-bugs-a7k9"]
    },
    {
      "name": "pheromone-system",
      "description": "Complete pheromone memory implementation",
      "colonies": [
        "20260215-pheromone-foundation-x2m4",
        "(future: integration)",
        "(future: learning)",
        "(future: colonize)",
        "(future: lineage)"
      ]
    }
  ]
}
```

### 4.3 Colony Lifecycle

```
COLONY LIFECYCLE
================

1. PLAN (/ant:lay-eggs)
   User defines master goal spanning multiple colonies
   Creates MOUND.md with colony specifications
   Status: PLANNED

2. INITIALIZE (/ant:init)
   Archive current colony (if active)
   Create new colony directory
   Inherit from parent (if specified)
   Link as active
   Status: ACTIVE

3. EXECUTE (/ant:plan, /ant:build)
   Normal colony operations
   Deposit pheromones
   Status: ACTIVE

4. COMPLETE (/ant:continue, /ant:seal)
   Extract patterns → pheromones
   Generate completion report
   Move to sealed state
   Status: SEALED

5. ARCHIVE (/ant:entomb)
   Full archival
   Extract wisdom for lineage
   Preserve in eternal memory
   Status: ENTOMBED
```

### 4.4 Lineage & Inheritance

```
LINEAGE EXAMPLE
===============

                    ┌─────────────────┐
                    │   Colony A      │
                    │  (Bug Fixes)    │
                    │    SEALED       │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
     ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
     │   Colony B   │ │   Colony C   │ │   Colony D   │
     │  Foundation  │ │  Integration │ │    Learning  │
     │    ACTIVE    │ │   PLANNED    │ │   PLANNED    │
     └──────────────┘ └──────────────┘ └──────────────┘

Inheritance Flow:
1. Colony A seals → extracts pheromones
2. Colony B, C, D initialized
3. Each inhales Colony A's pheromones
4. Colony B adds new patterns
5. Colony C inherits from A + B
```

---

### 4.5 Concurrent Colony Execution (CRITICAL)

**The Problem with Single Colony State:**
- Current system: One `COLONY_STATE.json` per repo
- Cannot work on multiple features simultaneously
- No isolation between different work streams

**Solution: Multiple Active Colonies with Fast Switching**

```
CONCURRENT COLONY MODEL
=======================

Repository: MyProject

Colony A (Auth Refactor)        Colony B (Bug Fixes)
├─ State: ACTIVE                 ├─ State: ACTIVE
├─ Phase: 2/5                    ├─ Phase: 1/3
├─ Workers: 3 running            ├─ Workers: 0 (paused)
└─ Files: src/auth/*             └─ Files: src/utils/*

Colony C (Performance)          Colony D (Docs)
├─ State: SEALED                 ├─ State: PLANNED
├─ Completion: 100%              ├─ Ready to start
└─ Pheromones: extracted         └─ Parent: Colony A
```

**Switching Between Colonies:**

```bash
# Current colony
/ant:status
> Active Colony: Colony A (Auth Refactor)
> Phase: 2/5

# Switch to Colony B
/ant:tunnels --switch colony-B-bugfixes
> Switched to Colony B
> Resumed from paused state

# Work on Colony B
/ant:build
...

# Switch back
/ant:tunnels --switch colony-A-auth
> Switched to Colony A
> Resumed Phase 2
```

**Isolation Strategy:**

1. **State Isolation**
   - Each colony has own `COLONY_STATE.json`
   - Each colony has own `activity.log`
   - Each colony has own `pheromones.json`

2. **File Change Isolation**
   - Colonies track different file sets
   - Git branches can be per-colony (optional)
   - Changes isolated until colony completes

3. **Worker Isolation**
   - Workers spawned in one colony don't affect others
   - Colony switch pauses active workers (gracefully)
   - Resume picks up where left off

**Implementation:**

```json
// Registry tracks multiple ACTIVE colonies
{
  "current_colony": "colony-A-auth",
  "active_colonies": [
    {
      "id": "colony-A-auth",
      "status": "active",
      "last_active": "2026-02-15T10:00:00Z",
      "workers_running": 3
    },
    {
      "id": "colony-B-bugfixes",
      "status": "paused",
      "last_active": "2026-02-15T09:30:00Z",
      "workers_running": 0
    }
  ]
}
```

**Command: `/ant:tunnels --switch <colony-id>`**

```markdown
## Switch Process:

1. Save current colony state
   - Set status: paused
   - Record timestamp
   - Preserve worker context

2. Update symlink to new colony
   ```bash
   rm .aether/data/COLONY_STATE.json
   ln -s ../colonies/{new-id}/COLONY_STATE.json .aether/data/
   ```

3. Load new colony state
   - Set status: active
   - Display resume context

4. Output:
   ```
   🕳️🐜🕳️ COLONY SWITCH
   ═══════════════════════════════════════════════════════

   From: Colony A (Auth Refactor) - Paused
   To:   Colony B (Bug Fixes) - Resumed

   Colony B Status:
   📍 Phase: 1/3
   📋 Tasks: 2 completed, 1 in progress
   ⏸️  Resumed from: 2026-02-15 09:30

   Active pheromones:
   🎯 FOCUS: error-handling

   ═══════════════════════════════════════════════════════
   ```
```

**Benefits:**
- Work on multiple features simultaneously
- Context switch without losing state
- Isolate experimental work
- Parallel development tracks
- Each colony learns independently
- Cross-colony learning via eternal memory

---

## 5. Phase 1: Foundation

**STATUS:** 🟡 PLANNED
**DURATION:** 1-2 days
**AGENTS:** 3 parallel tracks
**DELIVERABLE:** Working eternal memory with auto-deposition

### 5.1 Overview

Build the infrastructure for eternal memory and basic pheromone auto-deposition. This is the bedrock everything else builds on.

### 5.2 Agent Assignments

#### AGENT A: Eternal Directory Structure
**TASK:** Create `~/.aether/eternal/` and all subdirectories

**Files to Create:**
```
~/.aether/
├── eternal/
│   ├── queen-will.md
│   ├── pheromones.json
│   ├── stack-profile/
│   ├── patterns/
│   └── lineage/
```

**Content Specifications:**

`queen-will.md`:
```markdown
# Queen's Will - Core Philosophies

## Emergence Over Orchestration
Workers follow scent trails, not explicit commands.
Let the colony learn. Let patterns emerge.
Trust the pheromones.

## Minimal Change
Prefer small, focused changes.
No refactoring "while you're there".
Change only what advances the goal.

## Documentation Through Action
Don't write documentation that will rot.
Write pheromones that guide future workers.
Let the code speak, but scent the path.

## Biological Fidelity
We are ants, not robots.
We forget (decay).
We follow trails (pheromones).
We learn from failure (midden).
We inherit wisdom (lineage).
```

`pheromones.json`:
```json
{
  "version": "1.0",
  "last_updated": "ISO-8601",
  "trails": [],
  "metadata": {
    "total_deposited": 0,
    "active_count": 0
  }
}
```

**ACCEPTANCE CRITERIA:**
- [ ] Directory structure exists
- [ ] Files have correct initial content
- [ ] Permissions allow read/write
- [ ] Validation passes

**DEPENDENCIES:** None

---

#### AGENT B: Pheromone JSON Schema
**TASK:** Define and validate pheromone data structures

**Files:**
- `.aether/schemas/pheromone-v1.0.json` (JSON Schema)
- `.aether/lib/pheromone-utils.sh` (validation functions)

**Functions to Implement:**
```bash
# pheromone-utils.sh

pheromone_validate() {
    # Validate JSON against schema
    # Return: {"valid": true} or {"valid": false, "errors": []}
}

pheromone_create() {
    # Create new pheromone
    # Args: type, substance, strength, source, decay
    # Return: pheromone object
}

pheromone_decay() {
    # Calculate current strength
    # Args: pheromone, current_time
    # Return: decayed_strength
}

pheromone_purge_weak() {
    # Move < 0.1 strength to midden
    # Args: pheromones.json path
    # Return: count moved
}

pheromone_merge() {
    # Merge two pheromone sources
    # Args: existing, new
    # Logic: strengthen if same substance
    # Return: merged pheromone
}
```

**ACCEPTANCE CRITERIA:**
- [ ] JSON Schema validates all 6 pheromone types
- [ ] All utility functions work correctly
- [ ] Decay calculation accurate
- [ ] Tested with sample data

**DEPENDENCIES:** Agent A (directory structure)

---

#### AGENT C: Auto-Deposition in /ant:init
**TASK:** Modify init command to auto-deposit foundation pheromones

**File:** `.claude/commands/ant/init.md`

**Modifications:**

Add to Step 3 (Write Colony State):
```markdown
### Step 3.5: Deposit Foundation Pheromones

After writing COLONY_STATE.json, deposit initial pheromones:

```bash
# Create foundation pheromones
bash .aether/aether-utils.sh pheromone-deposit \
  "PHILOSOPHY" \
  "emergence-over-orchestration" \
  1.0 \
  "queen:init" \
  "never"

bash .aether/aether-utils.sh pheromone-deposit \
  "PHILOSOPHY" \
  "minimal-change" \
  1.0 \
  "queen:init" \
  "never"

# Display to user
```
🐜 Queen's scent established:
   • PHILOSOPHY[emergence-over-orchestration]
   • PHILOSOPHY[minimal-change]

These guide all workers.
```
```

**ACCEPTANCE CRITERIA:**
- [ ] /ant:init deposits 2 PHILOSOPHY pheromones
- [ ] User sees confirmation message
- [ ] Pheromones appear in eternal/pheromones.json
- [ ] Works on first colony (no parent)
- [ ] Works on subsequent colonies (with parent)

**DEPENDENCIES:** Agent A (structure), Agent B (utils)

---

### 5.3 Phase 1 Integration Testing

```bash
# Test script for Phase 1

echo "=== Phase 1: Foundation Tests ==="

# Test 1: Directory structure
ls -la ~/.aether/eternal/ || exit 1

# Test 2: Schema validation
bash .aether/lib/pheromone-utils.sh validate < test-pheromone.json || exit 1

# Test 3: Auto-deposition
/ant:init "Test Colony"
grep -q "emergence" ~/.aether/eternal/pheromones.json || exit 1

echo "=== Phase 1: ALL TESTS PASS ==="
```

---

## 6. Phase 2: Integration

**STATUS:** 🟡 PLANNED
**DURATION:** 2-3 days
**AGENTS:** 3 parallel tracks
**DELIVERABLE:** Queue system, enhanced commands, visibility

### 6.1 Overview

Connect pheromones to the existing command system. Workers can receive pheromones mid-work. Users can see active trails.

### 6.2 Agent Assignments

#### AGENT D: Pheromone Queue System
**TASK:** Implement mid-work injection without interrupting workers

**Files:**
- `.aether/lib/pheromone-queue.sh`
- `.aether/data/pheromone-queue.json` (per colony)

**Data Structure:**
```json
{
  "version": "1.0",
  "queue": [
    {
      "id": "phem_queued_123",
      "type": "FOCUS",
      "substance": "error-handling",
      "strength": 0.9,
      "deposited_at": "ISO-8601",
      "status": "queued|picked_up|applied",
      "picked_up_by": "worker-name",
      "picked_up_at": "ISO-8601"
    }
  ]
}
```

**Functions:**
```bash
pheromone_queue_add() {
    # Add pheromone to queue
    # Args: type, substance, strength
}

pheromone_queue_poll() {
    # Check for queued pheromones
    # Args: worker_name
    # Return: array of pending pheromones
    # Side effect: marks as picked_up
}

pheromone_queue_apply() {
    # Mark as applied after worker uses it
    # Args: pheromone_id, worker_name
}

pheromone_queue_cleanup() {
    # Remove old applied entries
    # Run periodically
}
```

**Worker Integration:**
Add to all worker prompts (build.md, swarm.md, etc.):
```markdown
Check for new pheromones:
```bash
bash .aether/aether-utils.sh pheromone-poll "{worker_name}"
```

If new scents detected, announce:
"🐜 [Worker-Name] New scent: FOCUS[error-handling]"
```

**ACCEPTANCE CRITERIA:**
- [ ] Can queue pheromone while workers active
- [ ] Workers detect at natural breakpoints
- [ ] No interruption to work flow
- [ ] Queue properly cleaned up

**DEPENDENCIES:** Phase 1 complete

---

#### AGENT E: Command Enhancements
**TASK:** Enhance /ant:focus, /ant:redirect, add pheromone visibility

**Files Modified:**
- `.claude/commands/ant/focus.md`
- `.claude/commands/ant/redirect.md`
- `.claude/commands/ant/status.md`

**/ant:focus Enhancement:**
```markdown
Old: Just add to constraints.json
New:
1. If workers active: queue pheromone
2. If no workers: deposit immediately
3. Display pheromone confirmation
4. Update constraints.json (legacy)
```

**New Output:**
```
🐜 FOCUS TRAIL LAID
═══════════════════════════════════════════════════════

Type: FOCUS
Substance: authentication
Strength: 0.9
Decay: 30 days

Status: queued for active workers
Active workers will detect at next checkpoint.

All active trails:
  🎯 FOCUS: authentication (0.9)
  🎯 FOCUS: error-handling (0.7)
  ⚠️ REDIRECT: regex-parsing (0.8)
═══════════════════════════════════════════════
```

**/ant/status Enhancement:**
Add section:
```markdown
### Pheromone Status Display

```
Active Pheromone Trails:
  🎯 FOCUS: authentication (0.9) [12d remaining]
  🎯 FOCUS: error-handling (0.7) [8d remaining]
  ⚠️  REDIRECT: regex-parsing (0.8) [45d remaining]
  📚 PATTERN: bash-for-file-ops (0.8) [82d remaining]
  💭 PHILOSOPHY: emergence (1.0) [eternal]

Queued for workers: 1
Recent deposits: 3 (last 24h)
```
```

**ACCEPTANCE CRITERIA:**
- [ ] /ant:focus queues or deposits correctly
- [ ] /ant:redirect queues or deposits correctly
- [ ] /ant:status shows pheromones
- [ ] Display matches specification

**DEPENDENCIES:** Agent D (queue system)

---

#### AGENT F: Worker Spawn-Time Priming
**TASK:** Workers inhale pheromones at spawn

**Files Modified:**
- `.aether/workers.md` (update spawn protocol)
- `.claude/commands/ant/build.md` (worker prompts)
- `.claude/commands/ant/swarm.md` (scout prompts)

**Worker Prompt Addition:**
```markdown
**Spawn-Time Priming:**

You are {Name}, a {emoji} {Caste} Ant.

Inhale the colony's scent:
```bash
# Read eternal memory
bash .aether/aether-utils.sh pheromone-inhale
```

This gives you:
- Queen's Will (core philosophies)
- Active FOCUS trails (what to prioritize)
- REDIRECT trails (what to avoid)
- Validated PATTERNs (best practices)

Carry this context in your work.
```

**ACCEPTANCE CRITERIA:**
- [ ] Workers read eternal memory at spawn
- [ ] Workers receive active pheromones
- [ ] Output shows what was inhaled
- [ ] Context affects worker behavior

**DEPENDENCIES:** Phase 1 complete

---

## 7. Phase 3: Learning

**STATUS:** 🟡 PLANNED
**DURATION:** 3-4 days
**AGENTS:** 2 parallel tracks
**DELIVERABLE:** Pattern detection, milestone wisdom extraction

### 7.1 Overview

The colony learns from its own behavior. Successful patterns become PHEROMONEs. Failed attempts go to midden.

### 7.2 Agent Assignments

#### AGENT G: Pattern Detection
**TASK:** Detect patterns in worker behavior and deposit as PATTERN pheromones

**Files:**
- `.aether/lib/pattern-detector.sh`
- `.aether/lib/learning-engine.sh`

**Detection Rules:**
```bash
# Pattern detection triggers

# Rule 1: Tool preference
if [ $bash_count -gt 5 ] && [ $node_count -lt 2 ]; then
    deposit_pattern "bash-for-file-ops" 0.7
fi

# Rule 2: Validation success
if [ $validation_passed -eq 3 ]; then
    strengthen_pattern "prefer-joi-over-zod" 0.1
fi

# Rule 3: Error recovery
if [ $error_occurred ] && [ $recovery_successful ]; then
    deposit_pattern "error-recovery-strategy" 0.6
fi

# Rule 4: File organization
if [ $files_in_correct_dirs -gt 0.8 ]; then
    deposit_pattern "directory-conventions" 0.7
fi
```

**Auto-Deposit at Milestones:**
```bash
# After phase completion
pattern_detector_analyze_phase() {
    phase_activity_log=$1

    # Count tool usage
    # Identify repeated patterns
    # Calculate confidence
    # Deposit PATTERN pheromones
}
```

**ACCEPTANCE CRITERIA:**
- [ ] Detects tool preferences
- [ ] Detects validation patterns
- [ ] Deposits with appropriate strength
- [ ] Triggers at phase completion

**DEPENDENCIES:** Phase 2 complete

---

#### AGENT H: Milestone Wisdom Extraction
**TASK:** Extract and archive wisdom when sealing colony

**File:** `.claude/commands/ant/seal.md` (enhanced)

**Enhancement:**
```markdown
### Step X: Extract Pheromones

Before sealing, analyze colony history:

```bash
# Extract patterns
bash .aether/lib/pattern-detector.sh extract \
  --colony-id {id} \
  --output .aether/data/extracted-pheromones.json

# Deposit to eternal memory
for pheromone in extracted; do
    bash .aether/aether-utils.sh pheromone-deposit \
      "PATTERN" \
      "$pheromone.substance" \
      $pheromone.strength \
      "milestone:phase-$phase" \
      "90d"
done

# Display
```
🐜 Wisdom archived: N patterns extracted
   • PATTERN[...] (validated X times)
   ...
```
```

**ACCEPTANCE CRITERIA:**
- [ ] Analyzes entire colony activity
- [ ] Extracts meaningful patterns
- [ ] Deposits to eternal memory
- [ ] Shows user what was learned

**DEPENDENCIES:** Agent G (pattern detection)

---

## 8. Phase 4: Colonize

**STATUS:** 🟡 PLANNED
**DURATION:** 2-3 days
**AGENTS:** 1 focused track
**DELIVERABLE:** Deep codebase analysis command

### 8.1 Overview

Create the `/ant:colonize` command for deep codebase mapping that deposits STACK and PATTERN pheromones.

### 8.2 Agent Assignments

#### AGENT I: /ant:colonize Command
**TASK:** Implement deep codebase analysis

**New File:** `.claude/commands/ant/colonize.md`

**Full Command Specification:**
```markdown
---
name: ant:colonize
description: "🔍🐜🗺️ Deep codebase analysis - scouts map codebase and deposit pheromone trails"
---

## Overview

Send scouts to deeply analyze the codebase:
- Map directory structure
- Identify tech stack
- Extract conventions
- Find concerns (TODO/FIXME)
- Deposit STACK and PATTERN pheromones

## Scouts Deployed

1. **Structure Scout** - Directory mapping
2. **Stack Scout** - Technology detection
3. **Convention Scout** - Pattern analysis
4. **Concern Scout** - Issue mining

## Output Files

- `.aether/colony/stack.md`
- `.aether/colony/conventions.md`
- `.aether/colony/concerns.md`
- Updated `~/.aether/eternal/stack-profile/{detected}.md`
- Deposited pheromones

## Usage

```
/ant:colonize          # Full analysis
/ant:colonize --quick  # Surface scan only
/ant:colonize --deep   # Include git history
```
```

**Implementation Steps:**
1. Deploy 4 scouts in parallel
2. Collect findings
3. Write analysis files
4. Detect stack (nodejs, python, rust, etc.)
5. Update stack-profile
6. Deposit STACK pheromone
7. Deposit discovered PATTERNs

**ACCEPTANCE CRITERIA:**
- [ ] 4 scouts deployed correctly
- [ ] All output files created
- [ ] Stack detection accurate
- [ ] Pheromones deposited
- [ ] Integrates with eternal memory

**DEPENDENCIES:** Phase 2 complete (can parallel with Phase 3)

---

## 9. Phase 5: Lineage

**STATUS:** 🟡 PLANNED
**DURATION:** 2-3 days
**AGENTS:** 1 focused track
**DELIVERABLE:** Cross-colony inheritance system

### 9.1 Overview

Enable colonies to inherit from prior colonies. Create the multi-colony architecture.

### 9.2 Agent Assignments

#### AGENT J: Multi-Colony & Lineage System
**TASK:** Implement full multi-colony architecture

**Files:**
- `.claude/commands/ant/lay-eggs.md` (NEW)
- `.claude/commands/ant/tunnels.md` (ENHANCED)
- `.claude/commands/ant/init.md` (ENHANCED - lineage)
- `.claude/commands/ant/entomb.md` (ENHANCED)
- `.aether/lib/colony-registry.sh`

**Implementation:**

1. **Registry System**
   - Create `.aether/colonies/registry.json`
   - Implement CRUD operations
   - Handle active colony symlink

2. **Colony Directory Migration**
   - Move existing colony to `colonies/{id}/`
   - Create symlink
   - Preserve backward compatibility

3. **Lineage Tracking**
   - Parent-child relationships
   - Inheritance at init
   - Pheromone copying

4. **Command Enhancements**
   - `/ant:lay-eggs` - Create master plan
   - `/ant:tunnels --switch` - Colony switching
   - `/ant:entomb` - Proper archival

**ACCEPTANCE CRITERIA:**
- [ ] Multiple colonies per repo
- [ ] Colony switching works
- [ ] Inheritance extracts pheromones
- [ ] Registry maintained correctly
- [ ] Backward compatibility preserved

**DEPENDENCIES:** Phase 3 & 4 complete

---

## 10. Command Specifications

### 10.1 Command Matrix

| Command | Modified | Purpose | Phase |
|---------|----------|---------|-------|
| `/ant:init` | ✅ Enhanced | Initialize with pheromone deposition | 1 |
| `/ant:focus` | ✅ Enhanced | Deposit FOCUS pheromone | 2 |
| `/ant:redirect` | ✅ Enhanced | Deposit REDIRECT pheromone | 2 |
| `/ant:status` | ✅ Enhanced | Show pheromone trails | 2 |
| `/ant:seal` | ✅ Enhanced | Extract wisdom on completion | 3 |
| `/ant:colonize` | 🆕 New | Deep analysis, deposit STACK | 4 |
| `/ant:lay-eggs` | 🆕 New | Multi-colony master plan | 5 |
| `/ant:tunnels` | ✅ Enhanced | Colony switching | 5 |
| `/ant:entomb` | ✅ Enhanced | Proper archival | 5 |

### 10.2 Detailed Command Specs

[Each command gets 1-2 page detailed specification here including:
- Full prompt text
- All steps
- Error handling
- Output format
- Testing procedures]

---

### 10.3 Complete UX Flows

**User Experience Scenarios**

#### Scenario 1: First Colony Experience (Onboarding)

```
User: /ant:init "Build auth system"

System:
[Standard init flow...]

🐜 FIRST COLONY DETECTED
═══════════════════════════════════════════════════════

Your Queen's scent is being established.

Auto-deposited pheromones:
  ✓ PHILOSOPHY[emergence-over-orchestration]
  ✓ PHILOSOPHY[minimal-change]

These guide all future workers in this colony.

💡 Tip: You can adjust worker behavior mid-flight:
   /ant:focus "security" --strength 0.9

═══════════════════════════════════════════════════════

🌱🐜🆕🐜🌱 ═══════════════════════════════════════════════════
   A E T H E R   C O L O N Y
═══════════════════════════════════════════════════ 🌱🐜🆕🐜🌱

👑 Queen has set the colony's intention

   "Build auth system"

🏠 Colony Status: READY
📋 Session: session_1234567890_abc123

🐜 The colony awaits your command:

   /ant:plan      📋 Generate project plan
   /ant:colonize  🗺️  Analyze existing codebase first
   /ant:watch     👁️  Set up live visibility

💾 State persisted — safe to /clear, then run /ant:plan
```

---

#### Scenario 2: Mid-Work Pheromone Injection

```
[Workers actively building Phase 2...]

User: /ant:focus "error-handling"

System:
🐜 FOCUSTRAIL LAID
═══════════════════════════════════════════════════════

Type: FOCUS
Substance: error-handling
Strength: 0.9
Status: queued for active workers

Current workers: 3 active
Next checkpoint: within 60 seconds

═══════════════════════════════════════════════════════

[30 seconds later, Prime Worker completes task]

Prime Worker: "Checkpoint reached - checking for new scents..."
Prime Worker: "New scent detected: FOCUS[error-handling]"
Prime Worker: "Adjusting priorities..."
Prime Worker: [Continues work with new focus on error handling]
```

---

#### Scenario 3: Pattern Learning

```
[Worker completed 3 similar tasks]

System:
💡 PATTERN DETECTED
═══════════════════════════════════════════════════════

Workers consistently used: "check-file-exists-before-write"

Suggested pheromone:
  PATTERN[check-exists-before-write]
  Confidence: 0.8 (based on 3 repetitions)

Deposit this for future colonies? [Y/n/help]

═══════════════════════════════════════════════════════

User: Y

System:
🐜 PHEROMONE DEPOSITED
═══════════════════════════════════════════════════════

Type: PATTERN
Substance: check-exists-before-write
Strength: 0.8
Source: worker:pattern-detection
Why: Detected 3 times in this phase

This trail will guide future workers.
View all: /ant:status
═══════════════════════════════════════════════════════
```

---

#### Scenario 4: Colony Completion & Wisdom Extraction

```
User: /ant:seal

System:
🏺 Colony Sealed - Crowned Anthill
═══════════════════════════════════════════════════════

3 phases completed, 47 tasks done

🐜 Wisdom extracted and archived:

PATTERNS (validated):
   • PATTERN[prefer-joi-over-zod] (validated 5x)
   • PATTERN[bash-for-file-ops] (used 12x)
   • PATTERN[auth-middleware-pattern] (validated 3x)

REDIRECTS (learned from failures):
   • REDIRECT[avoid-sync-fs] (failed once, fixed)
   • REDIRECT[no-deep-nesting] (caused issues)

PHILOSOPHIES (milestone):
   • PHILOSOPHY[auth-system-completed]

Preserved in eternal memory for future colonies.
═══════════════════════════════════════════════════════

Next:
  /ant:lay-eggs "Next feature"  Start new colony with lineage
  /ant:tunnels                    View all colonies
```

---

#### Scenario 5: Swarm Bug Fix & Immunity

```
User: /ant:swarm "Auth middleware failing intermittently"

System:
[Swarm deploys 4 scouts...]
[Scouts investigate...]
[Fix applied...]

🔥 Swarm Resolved
═══════════════════════════════════════════════════════

Applied: Null check with early return
Files modified: src/auth/middleware.ts

🐜 Immunity deposited: REDIRECT[unchecked-null-access]
   Future workers will guard against this pattern.

Pattern archived in midden for reference.
═══════════════════════════════════════════════════════
```

---

#### Scenario 6: Multi-Colony Switching

```
User: /ant:tunnels

System:
🕳️🐜🕳️ ═══════════════════════════════════════════════
        C O L O N Y   T U N N E L S
═══════════════════════════════════════════════ 🕳️🐜🕳️

Active Colonies:
  #  ID                                    Goal
  1  20260215-auth-system-x2m4            Build auth system ← ACTIVE
  2  20260215-bugfixes-a7k9               Fix critical bugs (paused)
  3  20260215-performance-tuning-p9q1     Optimize queries (sealed)

Commands:
  /ant:tunnels --switch 2     Resume work on bug fixes
  /ant:tunnels --archive      Seal current colony
  /ant:tunnels --compare 1 2  See differences

═══════════════════════════════════════════════════════

User: /ant:tunnels --switch 2

System:
🕳️🐜🕳️ COLONY SWITCH
═══════════════════════════════════════════════════════

From: Colony 1 (Auth System) - Paused
To:   Colony 2 (Bug Fixes) - Resumed

Colony 2 Status:
📍 Phase: 1/3
📋 Tasks: 2 completed, 1 in progress
⏸️  Resumed from: 2026-02-15 09:30

Active pheromones:
🎯 FOCUS: error-handling

Ready to continue work.
═══════════════════════════════════════════════════════
```

---

#### Scenario 7: Viewing Active Trails (/ant:sniff)

```
User: /ant:sniff

System:
🐜 ACTIVE PHEROMONE TRAILS
═══════════════════════════════════════════════════════

ETERNAL (never decay):
  PHILOSOPHY[emergence-over-orchestration]  ████████░░ 0.8
  PHILOSOPHY[minimal-change]                ████████░░ 0.8
  DECREE[no-force-push]                     █████████░ 0.9

FOCUS (decay: 30 days):
  FOCUS[authentication-flow]                █████████░ 0.9  (12 days left)
  FOCUS[performance-optimization]           ██████░░░░ 0.6  (5 days left)

REDIRECT (decay: 60 days):
  REDIRECT[regex-parsing]                   ███████░░░ 0.7  (45 days left)

PATTERN (decay: 90 days):
  PATTERN[bash-for-file-ops]                ████████░░ 0.8  (78 days left)

Queued for pickup:
  FOCUS[error-handling]                     █████████░ 0.9  (waiting)

═══════════════════════════════════════════════════════
To emit: /ant:focus "<substance>" [--strength N]
```

---

## 11. File Specifications

### 11.1 Complete File Inventory

**NEW FILES (27 total):**
```
~/.aether/eternal/
├── queen-will.md
├── pheromones.json
├── stack-profile/
│   ├── nodejs.md
│   ├── python.md
│   ├── rust.md
│   └── go.md
├── patterns/
│   └── (emergent)
└── lineage/
    └── colony-{id}/
        ├── decisions.md
        ├── lessons.md
        └── pheromones.json

.aether/
├── colonies/
│   ├── registry.json
│   └── colony-{id}/
│       ├── COLONY_STATE.json
│       ├── activity.log
│       ├── constraints.json
│       ├── pheromones.json
│       ├── completion-report.md
│       └── events/
├── schemas/
│   └── pheromone-v1.0.json
├── lib/
│   ├── pheromone-utils.sh
│   ├── pheromone-queue.sh
│   ├── pattern-detector.sh
│   ├── learning-engine.sh
│   └── colony-registry.sh
└── colony/ (from /ant:colonize)
    ├── stack.md
    ├── conventions.md
    └── concerns.md
```

**MODIFIED FILES (8 total):**
```
.claude/commands/ant/
├── init.md
├── focus.md
├── redirect.md
├── status.md
├── seal.md
├── tunnels.md
├── entomb.md
└── colonize.md (new)
└── lay-eggs.md (new)
```

---

## 12. Testing Strategy

### 12.1 Test Categories

1. **Unit Tests** - Individual functions
2. **Integration Tests** - Command workflows
3. **End-to-End Tests** - Full colony lifecycle
4. **Multi-Agent Tests** - Parallel coordination

### 12.2 Test Scenarios

**Scenario 1: First Colony**
```
1. /ant:init "Test Goal"
2. Verify PHILOSOPHY pheromones deposited
3. Verify workers inhale at spawn
4. Complete phase
5. Verify PATTERN extraction
```

**Scenario 2: Mid-Work Pheromone**
```
1. Start /ant:build
2. While workers active: /ant:focus "security"
3. Verify queued
4. Verify workers detect
5. Verify behavior change
```

**Scenario 3: Lineage**
```
1. Seal Colony A
2. /ant:init "Colony B" (should inherit)
3. Verify pheromones copied
4. Verify workers get parent context
```

---

## 13. Appendices

### Appendix A: Quick Reference

**Pheromone Types:** FOCUS, REDIRECT, PHILOSOPHY, STACK, PATTERN, DECREE

**Decay Periods:** 30d, 60d, 90d, never

**Key Commands:** init, focus, redirect, status, seal, colonize, lay-eggs, tunnels

**File Locations:** ~/.aether/eternal/, .aether/colonies/, .aether/data/

### Appendix B: Migration Guide

**From Single Colony:**
1. Create registry
2. Move current state to colonies/
3. Create symlink
4. Update all commands

### Appendix C: Troubleshooting

**Issue:** Pheromones not decaying
**Fix:** Check decay job running

**Issue:** Workers not detecting queue
**Fix:** Verify checkpoint polling

**Issue:** Colony switch failed
**Fix:** Check symlink, registry integrity

---

## Document Control

**Version:** 1.0
**Last Updated:** 2026-02-15
**Author:** Aether Design Team
**Reviewers:** (awaiting)
**Status:** Ready for Implementation

**Change Log:**
- v1.0 (2026-02-15): Initial comprehensive specification

---

## 14. Advanced Multi-Colony Concepts

### 14.1 Colony Networks & Swarms

**Concept:** Multiple colonies can form a "swarm"—coordinated colonies working on different aspects of the same large goal.

```
SWARM: "Rebuild Platform"
├── Colony A: "Auth System Rewrite"      (active)
├── Colony B: "Database Migration"       (active)
├── Colony C: "API Modernization"        (planning)
└── Colony D: "Frontend Refactor"        (sealed)

Shared Mound: platform-v2.0
```

**Swarm Behavior:**
- Colonies share eternal memory
- Cross-colony pheromone alerts (e.g., Colony A warns Colony B about auth breaking change)
- Swarm-wide status via `/ant:swarm --network`

**Implementation:**
```json
// .aether/colonies/swarm-config.json
{
  "swarm_id": "platform-v2-rebuild",
  "mound": "platform-v2.0",
  "colonies": [
    "20260215-auth-rewrite-x1a2",
    "20260215-db-migration-b3c4",
    "20260215-api-modern-d5e6"
  ],
  "shared_pheromones": true,
  "cross_alerts": true
}
```

### 14.2 Colony Templates

**Concept:** Pre-configured colony blueprints for common project types.

```
Template: "web-api"
├── PHILOSOPHY: rest-over-rpc
├── PHILOSOPHY: test-driven-development
├── FOCUS: validation
├── STACK: nodejs-express
└── CASTE_ASSIGNMENTS:
    ├── Builder: 3 ants
    ├── Watcher: 1 ant
    └── Scout: 1 ant

Template: "react-frontend"
├── PHILOSOPHY: component-first
├── PHILOSOPHY: minimal-state
├── FOCUS: accessibility
├── STACK: react-typescript
└── CASTE_ASSIGNMENTS:
    ├── Builder: 2 ants
    ├── Watcher: 2 ants
    └── Chaos: 1 ant
```

**Usage:**
```bash
/ant:init --template web-api "Build user API"
```

### 14.3 Colony Forking

**Concept:** Create a child colony that diverges from parent to explore alternative approaches.

```
Colony A (main approach)
└── Fork: Colony A-experimental
    └── Goal: "Same auth system but with JWT instead of sessions"
    └── Inherits: All patterns from Colony A
    └── Diverges: On auth implementation
    └── Outcome: Compare, merge best approach
```

**Use Cases:**
- A/B testing implementation approaches
- Experimental refactoring
- Parallel exploration of solutions

### 14.4 Colony Merging

**Concept:** When two colonies working on related features finish, their learnings merge.

```
Colony A: "User authentication"
Colony B: "Admin authentication"

Merge creates:
├── Unified auth patterns
├── Combined pheromones (strength averaged)
├── Cross-cutting concerns identified
└── New PHILOSOPHY: auth-consistency
```

---

## 15. Hook System

### 15.1 Event Hooks

**Concept:** User-defined scripts that run at key colony lifecycle moments.

```
.aether/hooks/
├── pre-init.sh          # Before colony initialization
├── post-init.sh         # After colony initialization
├── pre-phase.sh         # Before each phase starts
├── post-phase.sh        # After each phase completes
├── on-error.sh          # When errors occur
├── on-pheromone.sh      # When pheromones deposited
└── pre-seal.sh          # Before colony sealed
```

**Example Hook (post-phase.sh):**
```bash
#!/bin/bash
# Send notification when phase completes
PHASE=$1
STATUS=$2

if [ "$STATUS" = "completed" ]; then
    echo "✅ Phase $PHASE complete" | notify-send "Aether Colony"
fi

# Auto-commit on phase complete
if [ "$STATUS" = "completed" ]; then
    git add .
    git commit -m "Phase $PHASE complete"
fi
```

### 15.2 Hook Configuration

```json
// .aether/hooks/config.json
{
  "enabled": {
    "pre-init": true,
    "post-phase": true,
    "on-error": true,
    "on-pheromone": false
  },
  "timeout_seconds": 30,
  "fail_on_error": false
}
```

### 15.3 Webhook Integration

**Concept:** HTTP webhooks for external system integration.

```json
// .aeter/webhooks.json
{
  "webhooks": [
    {
      "url": "https://api.slack.com/webhooks/colony",
      "events": ["colony.initialized", "phase.completed", "colony.sealed"],
      "headers": {
        "Authorization": "Bearer ${SLACK_TOKEN}"
      }
    },
    {
      "url": "https://api.github.com/repos/user/repo/statuses/${SHA}",
      "events": ["phase.completed"],
      "condition": "phase_number >= 3"
    }
  ]
}
```

---

## 16. Plugin Architecture

### 16.1 Plugin System Design

**Concept:** Third-party extensions that add new capabilities to the colony.

```
.aether/plugins/
├── plugin-coverage/           # Code coverage integration
│   ├── manifest.json
│   ├── hooks/
│   └── commands/
├── plugin-deploy/             # Deployment automation
│   ├── manifest.json
│   └── lib/
└── plugin-metrics/            # Custom metrics collection
    ├── manifest.json
    └── collectors/
```

**Plugin Manifest:**
```json
{
  "name": "coverage",
  "version": "1.0.0",
  "description": "Track test coverage across colonies",
  "author": "user",
  "hooks": ["post-phase", "pre-seal"],
  "commands": ["coverage:report", "coverage:compare"],
  "dependencies": {
    "aether": ">=3.0.0",
    "tools": ["nyc", "c8"]
  },
  "config": {
    "threshold": 80,
    "report_format": "html"
  }
}
```

### 16.2 Plugin Hooks

Plugins can register for colony events:

```javascript
// plugin-coverage/hooks/post-phase.js
module.exports = async (context) => {
  const { phase, colony, results } = context;

  // Run coverage check
  const coverage = await runCoverageCheck();

  // Deposit as PATTERN if threshold met
  if (coverage.percent >= context.config.threshold) {
    await colony.depositPheromone({
      type: 'PATTERN',
      substance: 'coverage-maintained',
      strength: coverage.percent / 100,
      evidence: [`Phase ${phase.number}: ${coverage.percent}% coverage`]
    });
  }

  // Store metrics
  await colony.storeMetric('coverage', phase.number, coverage);
};
```

### 16.3 Plugin Commands

Plugins can add new slash commands:

```yaml
# plugin-coverage/commands/report.md
---
name: coverage:report
description: Generate coverage report for current colony
---

Generate test coverage report:

1. Run coverage tool
2. Compare to colony baseline
3. Display trend
4. Suggest improvements
```

---

## 17. Custom Agent Design

### 17.1 Agent Definition Framework

**Concept:** Users can define custom agent types beyond built-in castes.

```yaml
# .aether/agents/security-auditor.yaml
name: security-auditor
emoji: 🔒
color: red
description: Specialized agent for security reviews

capabilities:
  - static-analysis
  - dependency-scanning
  - pattern-matching
  - owasp-checks

triggers:
  - on: pheromone
    type: FOCUS
    substance: security
  - on: command
    name: /ant:audit --security

workflow:
  1: Scan dependencies for vulnerabilities
  2: Check for hardcoded secrets
  3: Validate input sanitization
  4: Review authentication flows
  5: Generate security report

output_format: |
  🔒 {name} Security Report
  =========================
  Critical: {critical_count}
  Warning: {warning_count}
  Info: {info_count}

  Findings:
  {findings}
```

### 17.2 Agent Inheritance

Agents can extend base castes:

```yaml
# .aether/agents/react-specialist.yaml
extends: builder
name: react-specialist
emoji: ⚛️

specializations:
  - react-hooks
  - component-patterns
  - state-management
  - performance-optimization

pheromone_sensitivity:
  FOCUS[react]: 1.5x      # Stronger response
  FOCUS[vue]: 0.0x        # Ignore
  REDIRECT[class-components]: 2.0x  # Strong avoidance
```

### 17.3 Agent Swarming

Multiple agents of same type coordinating:

```
Security Audit Swarm:
├── 🔒 Security-1  Dependency scanning
├── 🔒 Security-2  Static analysis
├── 🔒 Security-3  Secret detection
└── 🔒 Security-4  OWASP validation

Results aggregated by Queen
```

---

## 18. Best Practices & Patterns

### 18.1 Anti-Patterns (REDIRECT these)

```
REDIRECT[colony-too-broad]
  Why: Colonies with goals like "Fix everything" fail
  Better: Specific, bounded goals

REDIRECT[skip-validation]
  Why: Assuming Phase N is correct without verification
  Better: Always validate before seal

REDIRECT[ignore-pheromones]
  Why: Workers not checking eternal memory
  Better: Spawn-time priming mandatory

REDIRECT[over-engineer]
  Why: Complex solutions for simple problems
  Better: Minimal viable change

REDIRECT[no-checkpoints]
  Why: Long phases without save points
  Better: Checkpoint every 30 mins
```

### 18.2 Best Practices (PATTERN these)

```
PATTERN[small-phases]
  Description: Keep phases under 2 hours of work
  Rationale: Easier to validate, less risk

PATTERN[pheromone-rich]
  Description: Deposit pheromones liberally
  Rationale: More guidance for future workers

PATTERN[validate-early]
  Description: Verify assumptions in Phase 1
  Rationale: Catch issues before they compound

PATTERN[document-decisions]
  Description: Record why, not just what
  Rationale: Future colonies understand context

PATTERN[colony-focus]
  Description: One colony per major concern
  Rationale: Clear boundaries, clean inheritance
```

### 18.3 Colony Sizing Guidelines

```
Micro Colony (< 1 day):
├── Goal: Single feature or bug fix
├── Phases: 1-2
├── Workers: 2-3
└── Use case: Quick iterations

Standard Colony (1-3 days):
├── Goal: Feature set or refactoring
├── Phases: 3-7
├── Workers: 3-5
└── Use case: Most development work

Mega Colony (1-2 weeks):
├── Goal: Major initiative
├── Phases: 8-15
├── Workers: 5-8
└── Use case: Version releases, rewrites

Swarm (multiple colonies):
├── Goal: Cross-cutting changes
├── Colonies: 3-6
├── Coordination: Shared mound
└── Use case: Platform rebuilds
```

---

## 19. Emergent Behaviors Philosophy

### 19.1 From Control to Emergence

**Traditional Approach:**
```
User → Detailed Instructions → AI → Output
(Explicit control)
```

**Emergent Approach:**
```
User → Intent + Pheromones → Colony → Self-Organization → Output
(Guided emergence)
```

### 19.2 Stigmergy in Code

Stigmergy: Communication through environmental modification.

```
Real ants:
Ant A drops pheromone → Environment changed →
Ant B detects pheromone → Behavior modified

Code ants:
Worker A deposits FOCUS[testing] → Eternal memory changed →
Worker B detects at spawn → Prioritizes test coverage
```

### 19.3 Trophallaxis (Food Sharing)

**Concept:** Knowledge transfer between workers.

```
Worker A (Builder) learns pattern:
  "Use bash for file operations"
  ↓
Deposits PATTERN pheromone
  ↓
Worker B (Scout) inhales at spawn:
  "I smell that bash is preferred"
  ↓
Worker B uses bash instead of node
  ↓
Pattern reinforced
```

### 19.4 Midden Pile (Waste Management)

Failed experiments aren't deleted—they're archived.

```
Colony attempts approach X → Fails →
Deposits to midden (with reason) →
Future colonies smell midden →
Avoid approach X
```

### 19.5 Quorum Sensing

**Concept:** Colony-wide decision making.

```
Multiple workers detect same issue:
├── Worker A flags concern
├── Worker B confirms concern
└── Worker C confirms concern

Quorum reached → Colony behavior shifts
→ REDIRECT deposited automatically
```

---

## 20. Future Possibilities

### 20.1 AI-Native Features

**Conceptual ideas not yet designed:**

1. **Predictive Pheromones**
   - System predicts what patterns you'll need
   - Pre-deposits based on code analysis

2. **Auto-Colonization**
   - AI detects codebase needs
   - Auto-spawns appropriate colonies

3. **Pheromone Evolution**
   - Patterns mutate and compete
   - Successful patterns reproduce
   - Unsuccessful patterns die out

4. **Cross-Repository Learning**
   - Patterns from Repo A apply to Repo B
   - Global pattern database
   - Community-contributed pheromones

### 20.2 Integration Ideas

**Potential external integrations:**

- **GitHub Actions:** CI/CD colony coordination
- **Linear/Jira:** Issue-driven colony spawning
- **Sentry:** Error-driven pattern detection
- **Figma:** Design-to-code colony pipelines
- **Notion:** Documentation-aware colonies

### 20.3 Visualization Enhancements

**Future visual concepts:**

- **3D Ant Hill:** VR/3D colony visualization
- **Pheromone Heatmap:** Visual trail intensity
- **Colony Timeline:** Gantt-style phase view
- **Pattern Network:** Graph of pattern relationships
- **Swarm View:** Multi-colony coordination map

### 20.4 Advanced Agent Concepts

**Experimental agent types:**

- **Alate (Winged Ant):** Explores new tech/stacks
- **Super-Major:** Heavy-lifter for complex refactors
- **Replete:** Storage specialist for large migrations
- **Nurse:** Onboards new agents, transfers knowledge
- **Soldier:** Guards against regressions

---

## 21. Implementation Roadmap (Extended)

### Phase 6: Ecosystem (Future)
- Plugin marketplace
- Community pheromone sharing
- Template repository
- Hook library

### Phase 7: Intelligence (Future)
- ML-based pattern detection
- Predictive colony spawning
- Auto-pheromone optimization
- Cross-repo learning

### Phase 8: Scale (Future)
- Distributed colonies
- Cloud-based eternal memory
- Team coordination
- Organization-wide patterns

---

## 22. Quick Start Recipes

### Recipe 1: Bug Fix Colony
```bash
# Initialize focused colony
/ant:init "Fix login redirect bug"

# Deposit context
/ant:focus "authentication"
/ant:focus "redirect-handling"

# Build with tight feedback
/ant:build --castes builder,watcher

# Seal when done
/ant:seal
```

### Recipe 2: Feature Colony
```bash
# Initialize with plan
/ant:init "Add password reset feature"

# Analyze existing auth
/ant:colonize --focus auth

# Lay eggs for multi-phase
/ant:lay-eggs "Password reset with email, token, UI"

# Build each phase
/ant:build --phase 1
/ant:build --phase 2

# Final seal
/ant:seal
```

### Recipe 3: Refactoring Swarm
```bash
# Create swarm config
echo '{"swarm_id": "backend-refactor", "colonies": []}' > .aether/swarm.json

# Colony A: Database layer
/ant:init --swarm backend-refactor "Refactor DB layer"

# Colony B: Service layer (parallel)
/ant:init --swarm backend-refactor "Refactor service layer"

# Monitor swarm
/ant:swarm --network

# Merge when complete
/ant:tunnels --merge A B
```

---

## Appendix D: Glossary

**Alate:** Winged ant capable of founding new colonies
**Castes:** Specialized worker types (Builder, Watcher, etc.)
**Chamber:** Archived colony state
**Colony:** Execution context with goal and phases
**Decay:** Pheromone strength reduction over time
**Eternal Memory:** Cross-colony persistent storage (~/.aether/eternal/)
**Focus:** Pheromone attracting attention to area
**Graveyard:** Failed attempt archive
**Lineage:** Parent-child colony relationships
**Midden:** Waste/archive pile for failed approaches
**Mound:** Master roadmap spanning multiple colonies
**Pheromone:** Persistent signal guiding behavior
**Queen:** User/colony initiator
**Redirect:** Pheromone warning away from anti-pattern
**Seal:** Complete and archive colony
**Stigmergy:** Indirect coordination through environment
**Trophallaxis:** Food/knowledge sharing between ants
**Worker:** AI agent performing tasks

---

## Document Control

**Version:** 1.1
**Last Updated:** 2026-02-15
**Author:** Aether Design Team
**Reviewers:** (awaiting)
**Status:** Comprehensive Specification Ready

**Change Log:**
- v1.1 (2026-02-15): Added Sections 14-22, hooks, plugins, custom agents, best practices
- v1.0 (2026-02-15): Initial comprehensive specification

---

*The colony remembers. The colony learns. The colony evolves.*
