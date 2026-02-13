# Research Summary: Aether Colony System

**Project:** Aether Colony System
**Synthesized:** 2026-02-13
**Purpose:** Inform roadmap creation from parallel research outputs

---

## Executive Summary

The Aether Colony System is a mature CLI-based multi-agent orchestration framework that coordinates AI agents (workers) using a colony metaphor with specialized castes, pheromone signals, and hierarchical spawning. The system has strong table stakes implementation (goal planning, worker spawning, verification gates, state persistence, git integration) and compelling differentiators (colony metaphor, instincts, cross-session memory, chaos testing).

The recommended approach is to **build on existing strengths** while addressing critical pitfalls: race conditions in shared state, data loss from overly broad checkpoints, and update system version awareness. The architecture uses a proven Queen-Worker hierarchy where the Queen (user) provides goals and constraints, but workers autonomously spawn sub-workers within depth limits. Key risks include concurrent state access corruption, user data loss during checkpoints, and command drift between Claude Code and OpenCode platforms.

---

## Key Findings

### From STACK.md

| Component | Recommendation | Rationale |
|-----------|---------------|-----------|
| CLI Parsing | commander ^11.0.0 | Best-in-class; auto-help, subcommands, type coercion |
| Shell Linting | ShellCheck (existing) | Keep + add to CI, enable strict mode |
| State Validation | JSON Schema (optional) | Only if corruption becomes issue |
| Error Handling | Structured handlers | Centralized error handler pattern |
| Testing | AVA + Bash integration | Industry standard |
| Logging | picocolors | Faster, fewer deps than chalk |
| File Locking | flock (existing) | Already implemented correctly |
| Atomic Writes | mv (existing) | Already implemented correctly |

### From FEATURES.md

**Table Stakes (All Implemented):**
- Project/Goal Initialization
- Task Planning
- Worker Spawning
- Verification (6-phase gates)
- State Persistence (JSON)
- Git Integration (checkpoints, stashes)
- Progress Visibility

**Differentiators (Implements 10+):**
- Colony Metaphor (8+ castes)
- Pheromone Signals (Focus, Redirect, Feedback)
- Nested Spawning (depth 1-3)
- Cross-Session Memory (completion-report.md)
- Instincts System (confidence-scored)
- Graveyard Tracking
- Chaos/Resilience Testing
- Archaeologist (git excavation)
- Dream/Interpret Cycle
- Swarm Command

**Anti-Features to Avoid:**
- Single-agent execution
- No verification
- Ephemeral state
- No git safety
- Micro-management
- Memory-less operation

### From ARCHITECTURE.md

**Component Boundaries:**
- Queen Layer: Orchestration only (init, plan, build, continue)
- Constraints Layer: Declarative focus/avoid rules
- Worker Layer: Execution with depth limits
- Utility Layer: aether-utils.sh as single entry point

**Key Patterns:**
1. Declarative Constraints Over Directives
2. Workers Spawn Workers (depth 1: 4 max, depth 2: 2 max, depth 3: 0)
3. State File as Source of Truth
4. Single Entry Point for Deterministic Ops
5. Depth-Based Behavior Limits

**Build Order:**
1. Core Infrastructure (state, utils, locking)
2. Queen Commands (init, status, constraints)
3. Worker System (spawn protocol, depth enforcement)
4. Execution Pipeline (plan, build, continue)
5. Advanced Features (swarm, council, watch)

### From PITFALLS.md

**Critical Pitfalls:**
1. **Race Conditions in Shared State** - Multiple workers corrupt COLONY_STATE.json
2. **Data Loss from Overly Broad Checkpoints** - git stash wipes user files
3. **Update Without Version Awareness** - No version tracking in local copies
4. **Background Task Results vs Visual Ordering** - Summary appears before agent banners

**Moderate Pitfalls:**
- State isolation confusion (system vs user data)
- Command duplication without sync (Claude Code vs OpenCode)
- Async state updates after context clear
- Magic string allowlists in code

**Minor Pitfalls:**
- No package-lock.json
- Hash computation on every sync
- Event timestamp ordering
- No input validation on file paths

---

## Implications for Roadmap

### Suggested Phase Structure

**Phase 1: Infrastructure Hardening**
- Rationale: Critical pitfalls (race conditions, data loss) must be fixed before any new features
- Delivers: File locking + atomic writes on all state operations, allowlist-based checkpoint system, version tracking
- Pitfalls to avoid: #1 (race conditions), #2 (data loss), #3 (version awareness)

**Phase 2: Command Consolidation**
- Rationale: Command duplication is a maintenance burden; single source needed before expansion
- Delivers: YAML command definitions, generator script, CI verification
- Pitfalls to avoid: #6 (command duplication), #8 (magic strings)

**Phase 3: State & Context Restoration**
- Rationale: Cross-session memory is a core differentiator; must work reliably
- Delivers: Context loading on every command, state validation, timestamp ordering
- Pitfalls to avoid: #7 (async state updates), #11 (timestamp ordering)

**Phase 4: Verification & Polish**
- Rationale: Core workflow complete; focus on UX and robustness
- Delivers: Foreground Task calls for verification, colored output, comprehensive testing
- Pitfalls to avoid: #4 (visual ordering)

**Phase 5: Feature Expansion**
- Rationale: Foundation solid; add differentiating features
- Delivers: New caste specializations, enhanced swarm, better visualization
- Keep: Colony metaphor, instincts, memory systems

### Research Flags

| Phase | Needs Research | Standard Patterns |
|-------|----------------|-------------------|
| Phase 1: Infrastructure | LOW | File locking, atomic writes well-documented |
| Phase 2: Commands | MEDIUM | Generator patterns exist, need design |
| Phase 3: State | MEDIUM | Cross-session patterns unique to Aether |
| Phase 4: Verification | LOW | Standard verification gates |
| Phase 5: Expansion | HIGH | New caste design needs experimentation |

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Commander, ShellCheck, AVA are industry standards |
| Features | HIGH | Table stakes verified in codebase; differentiators unique to Aether |
| Architecture | HIGH | Based on existing working implementation |
| Pitfalls | HIGH | Based on Aether's real bug history |

**Gaps Identified:**
- Limited ecosystem comparison (Aether is unique in colony model)
- Web search unavailable during research
- E2E testing strategy not fully defined

---

## Sources

- STACK.md: Commander.js, ShellCheck, oclif documentation
- FEATURES.md: Aether codebase analysis (commands, state, workers.md)
- ARCHITECTURE.md: QUEEN_ANT_ARCHITECTURE.md, aether-utils.sh
- PITFALLS.md: TO-DOs.md, CONCERNS.md, progress.md (real bugs)
