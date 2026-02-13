# Feature Landscape: CLI-Based AI Agent Orchestration Frameworks

**Domain:** Agent orchestration for Claude Code / OpenCode
**Researched:** 2026-02-13
**Confidence:** MEDIUM (synthesized from Aether codebase analysis + ecosystem knowledge)

## Executive Summary

CLI-based AI agent orchestration frameworks coordinate multiple AI agents (workers) to accomplish complex development tasks. This research maps the feature landscape for such frameworks, categorizing features as **table stakes** (expected baseline), **differentiators** (unique value), or **anti-features** (common mistakes).

Aether exemplifies a mature system with strong differentiators: colony metaphor, pheromone signals, nested spawning, and cross-session memory. Its table stakes features (phase planning, verification gates, git integration) are well-implemented.

## Feature Categories

### Table Stakes

Features users expect in any agent orchestration framework. Missing these = product feels incomplete or broken.

| Feature | Why Expected | Complexity | Aether Status |
|---------|--------------|------------|----------------|
| **Project/Goal Initialization** | Users need to define what they want to build | Low | Implemented (`/ant:init`) |
| **Task Planning** | Break goals into executable steps | Medium | Implemented (`/ant:plan`) |
| **Worker Spawning** | Actually execute tasks with agents | Medium | Implemented (Builder, Scout, Watcher castes) |
| **Verification** | Confirm work is correct (tests pass, build succeeds) | Medium | Implemented (6-phase verification) |
| **State Persistence** | Survive context resets | Medium | Implemented (JSON state files) |
| **Git Integration** | Checkpoints, rollback capability | Low | Implemented (stashes, commits) |
| **Progress Visibility** | Know what's happening | Low | Implemented (`/ant:status`, spawn tree) |

### Differentiators

Features that set products apart. Not expected, but highly valued when present.

| Feature | Value Proposition | Complexity | Aether Status |
|---------|-------------------|------------|----------------|
| **Colony Metaphor** | Specialized worker castes with distinct roles | High | Implemented (8+ castes) |
| **Pheromone Signals** | Guide behavior without micro-managing | Medium | Implemented (Focus, Redirect, Feedback) |
| **Nested Spawning** | Workers can spawn sub-workers | High | Implemented (depth 1-3) |
| **Cross-Session Memory** | Learn from previous projects | High | Implemented (completion-report.md) |
| **Instincts System** | Pattern-based decision making | High | Implemented (confidence-scored) |
| **Graveyard Tracking** | Remember what failed before | Medium | Implemented (grave markers) |
| **Chaos/Resilience Testing** | Probe for edge cases | Medium | Implemented (Chaos ant) |
| **Archaeologist** | Understand WHY code exists | Medium | Implemented (git history excavation) |
| **Dream/Interpret Cycle** | Philosophical reflection on codebase | Medium | Implemented |
| **Swarm Command** | Multi-angle attack on stubborn bugs | Medium | Implemented (4 scouts in parallel) |

### Anti-Features

Features to explicitly NOT build. Common mistakes in this domain.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Single-agent execution** | Doesn't scale to complex tasks | Parallel worker spawning |
| **No verification** | Code may not actually work | Execution-based verification (not just reading) |
| **Ephemeral state** | Context loss on reset | Persistent JSON/state files |
| **No git safety** | Dangerous to lose work | Checkpoints before phases |
| **Micro-management** | User exhausted by constant decisions | Pheromone signals for guidance |
| **Memory-less** | Repeating mistakes | Instincts + learnings system |
| **No rollback** | Can't recover from bad decisions | Git stash/checkpoint integration |

## Feature Dependencies

```
Project Initialization
    │
    ├──► Planning ───────────────────────┐
    │       │                             │
    │       └──► Task Analysis            │
    │               │                     │
    │               └──► Worker Spawning ─┼──► Verification ──► Phase Completion
    │                       │             │
    │                       └──► Sub-worker Spawning (nested)
    │
    └──► Memory/Instincts (cross-session)
            │
            └──► Graveyard Tracking
                    │
                    └──► Chaos Testing
```

## MVP Recommendation

For a minimal viable orchestration framework, prioritize:

### Phase 1: Table Stakes
1. **Goal initialization** - Set project intention
2. **Task planning** - Break into phases
3. **Worker execution** - Spawn agents to do work
4. **Basic verification** - Confirm tests pass
5. **State persistence** - Survive context resets

### Phase 2: Safety
1. **Git checkpoints** - Rollback capability
2. **Blocker flagging** - Track issues that prevent progress

### Phase 3: Differentiation
1. **Specialized workers** - Different agent types for different needs
2. **Pheromone signals** - Guide without micro-managing
3. **Memory inheritance** - Learn from previous sessions

## Aether Feature Inventory

Aether currently implements the following features:

### Core Workflow (Table Stakes)
- `/ant:init` - Colony initialization with goal
- `/ant:plan` - Phase planning with confidence scoring
- `/ant:build` - Execute phase with worker spawning
- `/ant:continue` - Verification gates + advance phase

### Pheromones (Differentiating)
- `/ant:focus` - Direct attention to areas
- `/ant:redirect` - Avoid specific approaches
- `/ant:feedback` - Teach preferences

### Specialized Castes (Differentiating)
- Builder - Implements code
- Watcher - Verifies work
- Scout - Researches
- Colonizer - Explores codebases
- Architect - Extracts patterns
- Route-Setter - Plans phases
- Archaeologist - Excavates git history
- Chaos - Resilience testing
- Dreamer - Philosophical reflection
- Interpreter - Grounds dreams in evidence

### Memory Systems (Differentiating)
- Instincts - Confidence-scored patterns
- Learnings - Validated knowledge
- Graveyards - Failure markers
- Completion reports - Cross-session inheritance

### Safety & Git
- Git checkpoints before each phase
- Stash-based rollback
- Gate-based commits

### Visibility
- `/ant:status` - Colony overview
- `/ant:watch` - Real-time tmux monitoring
- `/ant:phase` - Phase details
- Spawn tree visualization

### Issue Tracking
- `/ant:flag` - Create blockers/issues/notes
- `/ant:flags` - List and resolve

### Session Management
- `/ant:pause-colony` - Save state for break
- `/ant:resume-colony` - Restore state
- `/ant:migrate-state` - Upgrade old formats

## Research Notes

**Confidence Assessment:**
- Table stakes features: HIGH confidence (well-established patterns)
- Differentiators: MEDIUM confidence (Aether is unique, limited ecosystem comparison)
- Anti-features: HIGH confidence (common failure modes documented)

**Sources:**
- Aether codebase analysis (commands, state, workers.md)
- README.md feature documentation
- COLONY_STATE.json for runtime patterns

**Gaps:**
- Limited published research on CLI agent orchestration frameworks
- Web search unavailable during research (API errors)
- Ecosystem comparison limited due to unique nature of colony model

## Roadmap Implications

When building or extending Aether:

1. **Keep table stakes solid** - Any regression in verification, state persistence, or git integration would be critical
2. **Differentiators are Aether's value** - The colony metaphor, instincts, and memory systems are what make it special
3. **Avoid anti-features** - Single-agent execution or memory-less operation would undermine the core value
4. **Consider adding:**
   - More caste specializations (security reviewer, performance profiler)
   - Enhanced swarm capabilities
   - Better visualization/reporting
