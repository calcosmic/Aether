# Architecture Research: Colony Context Enhancement

**Domain:** Aether Colony System — Session Restoration Architecture
**Researched:** 2026-02-21
**Confidence:** HIGH (based on direct codebase analysis)

---

## Executive Summary

Context restoration for Aether requires a **hybrid approach**: enhance existing components where they already exist, create new specialized components only where gaps exist. The existing architecture already has 80% of what's needed — COLONY_STATE.json, CONTEXT.md, session tracking, and QUEEN.md provide the foundation. The gap is **rich context assembly** — a new component that reads these scattered sources and produces a sub-5-minute "state of the colony" snapshot.

**Key insight:** GSD's `.planning/PROJECT.md` + `STATE.md` + `phases/` structure maps directly to Aether's `COLONY_STATE.json` + `CONTEXT.md` + `plan.phases[]`. The patterns are compatible; only the naming differs.

---

## Existing Architecture Analysis

### Current Data Stores

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        EXISTING CONTEXT LAYER                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────┐ │
│  │  COLONY_STATE.json  │    │     CONTEXT.md      │    │    QUEEN.md     │ │
│  │  ─────────────────  │    │    ────────────     │    │   ──────────    │ │
│  │  • goal             │    │  • System status    │    │  • Philosophies │ │
│  │  • state (READY/etc)│    │  • Session notes    │    │  • Patterns     │ │
│  │  • current_phase    │    │  • Pending work     │    │  • Redirects    │ │
│  │  • plan.phases[]    │    │  • Completed work   │    │  • Stack wisdom │ │
│  │  • memory.learnings │    │  • Active signals   │    │  • Decrees      │ │
│  │  • memory.decisions │    │  • Recent decisions │    │                 │ │
│  │  • memory.instincts │    │  • Next steps       │    │                 │ │
│  │  • events[]         │    │                     │    │                 │ │
│  │  • errors.records   │    │                     │    │                 │ │
│  └─────────────────────┘    └─────────────────────┘    └─────────────────┘ │
│           │                          │                          │          │
│           └──────────────────────────┼──────────────────────────┘          │
│                                      │                                     │
│                           ┌──────────▼──────────┐                         │
│                           │   session.json      │                         │
│                           │  ────────────────   │                         │
│                           │  • colony_goal      │                         │
│                           │  • current_phase    │                         │
│                           │  • last_command     │                         │
│                           │  • suggested_next   │                         │
│                           │  • baseline_commit  │                         │
│                           └─────────────────────┘                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Current Command Flow

```
User runs /ant:resume
        │
        ▼
┌───────────────────┐
│   resume.md       │  ← Already reads COLONY_STATE.json, CONTEXT.md, session.json
│  (exists today)   │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  session-read     │  ← aether-utils.sh subcommand
│  (exists today)   │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  Render Dashboard │  ← Shows goal, phase, next command
│  (exists today)   │
└───────────────────┘
```

### What's Missing for Rich Context

| Need | Current State | Gap |
|------|---------------|-----|
| Decision history with rationale | `memory.decisions[]` exists but is **always empty** | Populate decisions during builds |
| Phase documentation (what happened) | `memory.phase_learnings[]` has entries but sparse | Structured phase summaries |
| Exact resume point | `current_phase` + `state` exist | Need "what was I doing when I stopped" |
| Why choices were made | Not captured | Decision log with rationale |
| Quick context assembly | 3-4 separate files to read | Single "colony snapshot" document |

---

## Recommended Architecture

### Principle: Enhance Existing, Create Minimal New

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    ENHANCED CONTEXT ARCHITECTURE                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  EXISTING (enhance)          NEW (create)          EXISTING (enhance)       │
│  ─────────────────           ────────────          ──────────────────       │
│                                                                             │
│  ┌─────────────────┐        ┌──────────────┐       ┌─────────────────┐     │
│  │ COLONY_STATE    │◄──────►│   NEST.md    │◄─────►│   CONTEXT.md    │     │
│  │  ────────────   │        │  ─────────   │       │  ────────────   │     │
│  │  + decisions[]  │        │  Rich colony │       │  + decisions    │     │
│  │    (populate)   │        │  snapshot    │       │    section      │     │
│  │  + phase_summaries│      │  for resume  │       │  + phase        │     │
│  │    (new field)  │        │              │       │    narratives   │     │
│  └─────────────────┘        └──────────────┘       └─────────────────┘     │
│          ▲                          ▲                                            │
│          │                          │                                            │
│          │                  ┌───────┴───────┐                                   │
│          │                  │  colony-snapshot │  ← NEW: aether-utils.sh        │
│          │                  │  subcommand      │     subcommand                  │
│          │                  └───────┬───────┘                                   │
│          │                          │                                            │
│          └──────────────────────────┘                                            │
│                                                                             │
│  ┌─────────────────┐        ┌──────────────┐       ┌─────────────────┐     │
│  │  session.json   │◄──────►│ resume.md    │◄─────►│   QUEEN.md      │     │
│  │  ────────────   │        │ (enhanced)   │       │  ────────────   │     │
│  │  + last_action  │        │              │       │  (unchanged)    │     │
│  │    (what doing  │        │ Reads        │       │                 │     │
│  │     when stopped)│       │ NEST.md      │       │                 │     │
│  └─────────────────┘        └──────────────┘       └─────────────────┘     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Decisions

| Component | Action | Rationale |
|-----------|--------|-----------|
| **COLONY_STATE.json** | **ENHANCE** | Add `phase_summaries[]` array, populate `decisions[]` (currently empty) |
| **CONTEXT.md** | **ENHANCE** | Add structured decision log section, phase narrative section |
| **session.json** | **ENHANCE** | Add `last_action` field capturing what was in progress when stopped |
| **NEST.md** | **CREATE** | New "colony snapshot" file — single document for instant context restoration |
| **colony-snapshot** | **CREATE** | New aether-utils.sh subcommand — assembles NEST.md from all sources |
| **resume.md** | **ENHANCE** | Read NEST.md instead of multiple files, render richer dashboard |
| **build.md** | **ENHANCE** | Write decisions to COLONY_STATE.json, update phase summaries |
| **continue.md** | **ENHANCE** | Write phase summary when phase completes |
| **QUEEN.md** | **NO CHANGE** | Already has wisdom structure, no enhancement needed |

---

## NEST.md: The New Colony Snapshot

### Purpose
A single file that contains everything a new session needs to understand the colony in under 5 minutes. Named "NEST" to fit ant metaphor — it's where the colony lives.

### Location
`.aether/NEST.md` — alongside CONTEXT.md, excluded from git (add to .gitignore)

### Structure

```markdown
# NEST — Colony Snapshot

> Generated: 2026-02-21T14:32:00Z
> Colony: {session_id}
> Read this first after /clear

---

## Why This Colony Exists

{goal from COLONY_STATE.json}

{1-2 sentence narrative of what we're building}

---

## Where We Are

**Current Phase:** {current_phase} of {total} — {phase_name}
**Status:** {state}
**Last Action:** {what was happening when session ended}

### Phase Progress
{v} Phase 1: Name — COMPLETED
{~} Phase 2: Name — IN PROGRESS
{ } Phase 3: Name — PENDING

---

## What We've Decided

| When | Decision | Rationale |
|------|----------|-----------|
| 2026-02-21 | Use React not Vue | Team familiarity, larger ecosystem |
| 2026-02-20 | Defer auth to Phase 3 | MVP focus on core features first |

---

## What's Working / What's Not

### Patterns Validated
- {from QUEEN.md patterns}

### Redirects Active
- {from constraints.json}

### Known Issues
- {from errors.records}

---

## Next Action

**Run:** {suggested_next}
**Why:** {reason}

---

## Quick Reference

- **Goal:** {goal}
- **Started:** {initialized_at}
- **Last Updated:** {timestamp}
- **Total Phases:** {count}
- **Completed:** {count}
```

---

## Data Flow: Session Restoration

```
User runs /ant:resume
        │
        ▼
┌─────────────────────────────────────────┐
│  resume.md (enhanced)                   │
│  ────────────────────                   │
│  1. Check for NEST.md                   │
│  2. If fresh: read NEST.md only         │
│  3. If stale/missing:                   │
│     - Call colony-snapshot              │
│     - Generate fresh NEST.md            │
│  4. Render dashboard from NEST.md       │
└─────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────┐
│  colony-snapshot (new subcommand)       │
│  ───────────────────────────────        │
│  Reads:                                 │
│    • COLONY_STATE.json → goal, phase,   │
│      decisions, phase_summaries         │
│    • CONTEXT.md → session notes,        │
│      recent activity                    │
│    • QUEEN.md → patterns, redirects     │
│    • session.json → last action         │
│    • constraints.json → active signals  │
│                                         │
│  Writes:                                │
│    • NEST.md (assembled snapshot)       │
└─────────────────────────────────────────┘
```

---

## Mapping GSD Patterns to Aether

| GSD Pattern | Aether Equivalent | Notes |
|-------------|-------------------|-------|
| `.planning/PROJECT.md` | `COLONY_STATE.json` + `CONTEXT.md` | Split across two files in Aether |
| `.planning/STATE.md` | `session.json` + `NEST.md` (new) | Session state + rich snapshot |
| `.planning/phases/{n}-*/PLAN.md` | `plan.phases[n]` in COLONY_STATE | Array instead of directories |
| `.planning/phases/{n}-*/SUMMARY.md` | `phase_summaries[n]` (new field) | Array of completion summaries |
| Decisions in STATE.md | `memory.decisions[]` | Already exists, needs population |
| Research files | `oracle/` directory | Already exists for research |

### Key Difference

GSD uses **file-per-phase** organization (directories with PLAN.md/SUMMARY.md). Aether uses **JSON array** organization (plan.phases[]). For context restoration, Aether's approach is actually more efficient — single file read vs directory traversal.

---

## Build Order (Dependencies)

```
Phase 1: Foundation
├── Enhance COLONY_STATE.json schema
│   └── Add phase_summaries[] field
│   └── Add decisions[] population in build.md
├── Create colony-snapshot subcommand
│   └── Read all sources
│   └── Generate NEST.md
└── Create NEST.md template

Phase 2: Integration
├── Enhance resume.md
│   └── Read NEST.md instead of multiple files
│   └── Render richer dashboard
├── Enhance build.md
│   └── Write decisions during execution
│   └── Update phase summaries on complete
└── Enhance continue.md
    └── Write phase summary when advancing

Phase 3: Polish
├── Enhance CONTEXT.md updates
│   └── Add decision log section
│   └── Add phase narrative section
├── Add session.json enhancements
│   └── Track last_action
└── Backwards compatibility
    └── Migration for existing colonies
```

---

## Migration Path for Existing Colonies

### Existing Colony State (v3.0)
- COLONY_STATE.json exists with goal, phases, learnings
- CONTEXT.md exists with session notes
- QUEEN.md exists with wisdom
- session.json exists with basic tracking

### Migration Strategy
1. **Additive only** — never remove fields
2. **Lazy migration** — new fields appear on next build/continue
3. **colony-snapshot** — works with or without new fields
4. **NEST.md generation** — gracefully handles missing data

### Migration Code Pattern
```bash
# In colony-snapshot subcommand
# If phase_summaries doesn't exist, build from phase_learnings
# If decisions doesn't exist, show "No decisions recorded yet"
# Always produces valid NEST.md regardless of source data completeness
```

---

## Integration Points Summary

| Integration | Type | Description |
|-------------|------|-------------|
| COLONY_STATE.json | **Read/Write** | Source of truth for goal, phase, decisions, summaries |
| CONTEXT.md | **Read** | Session notes, recent activity, next steps |
| QUEEN.md | **Read** | Patterns, redirects, stack wisdom, decrees |
| session.json | **Read/Write** | Last action, resume point tracking |
| constraints.json | **Read** | Active FOCUS/REDIRECT/FEEDBACK signals |
| NEST.md | **Write** | Generated snapshot for instant restoration |
| resume.md | **Enhanced** | Primary consumer of NEST.md |
| build.md | **Enhanced** | Writes decisions, updates phase state |
| continue.md | **Enhanced** | Writes phase summaries on advancement |

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Replace Existing Files
**What:** Creating new files that duplicate COLONY_STATE.json or CONTEXT.md
**Why bad:** Fragments source of truth, causes drift
**Instead:** Enhance existing files, use NEST.md as assembled view only

### Anti-Pattern 2: Break Backwards Compatibility
**What:** Schema changes that break existing colonies
**Why bad:** Users with active colonies can't resume
**Instead:** Additive changes only, lazy migration

### Anti-Pattern 3: Over-Engineer the Snapshot
**What:** Including everything including full file contents
**Why bad:** Violates 5-minute read constraint, information overload
**Instead:** Summaries and pointers, not full content

### Anti-Pattern 4: Ignore the Ant Metaphor
**What:** Using generic names like "context" or "state"
**Why bad:** Breaks thematic consistency
**Instead:** NEST.md (ant home), colony-snapshot (ant action), etc.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Existing architecture | HIGH | Direct codebase analysis, files read and verified |
| GSD pattern mapping | HIGH | Both PROJECT.md and STATE.md examined |
| Integration points | HIGH | All command files read, data flow traced |
| Build order | MEDIUM | Dependencies clear, but implementation order may shift |
| Migration path | HIGH | Additive-only strategy proven in v3.0 |

---

## Open Questions

1. **NEST.md freshness:** Should we regenerate on every command, or only when stale?
   *Recommendation:* Check timestamp in resume.md, regenerate if >1 hour old

2. **Phase summary detail:** How much detail per phase?
   *Recommendation:* 2-3 bullet points: what was built, key decisions, known issues

3. **Decision capture:** When exactly do we write decisions?
   *Recommendation:* When user makes explicit choice (e.g., approves promotion, selects option)

---

*Architecture research for: v4.0 Colony Context Enhancement*
*Researched: 2026-02-21*
*Sources: Direct codebase analysis of .aether/data/COLONY_STATE.json, .aether/CONTEXT.md, .claude/commands/ant/*.md, .aether/aether-utils.sh*
