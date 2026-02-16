# Feature Landscape: Model Routing & Colony Lifecycle (v3.1 Open Chambers)

**Domain:** AI agent orchestration framework with worker caste system
**Researched:** 2026-02-14
**Confidence:** HIGH (based on existing codebase analysis, workers.md, model-profiles.yaml, and colony state structure)

## Executive Summary

This research maps the feature landscape for v3.1 "Open Chambers" milestone, focusing on two core capabilities:
1. **Intelligent Model Routing** - Assigning optimal LLM models per worker caste based on task characteristics
2. **Colony Lifecycle Management** - Archive/foundation commands with ant-themed milestone progression

The Aether Colony System already has foundational infrastructure for both: model profiles exist in `.aether/model-profiles.yaml`, and the milestone system is partially implemented with six stages from "First Mound" to "Crowned Anthill". This research identifies what features are needed to make these systems production-ready.

---

## Table Stakes (Must Have)

Features users expect for model routing and lifecycle management to feel complete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Model Verification Command** | Users need to verify which models are assigned to which castes before spawning workers | Low | `/ant:models` command to display current model assignments from model-profiles.yaml |
| **Model Override per Command** | Users may want to force a specific model for a specific task | Low | `--model` flag on `/ant:build`, `/ant:swarm`, etc. that overrides caste default |
| **Archive Command** | Colony lifecycle requires archiving completed work before starting fresh | Medium | `/ant:archive` - copies COLONY_STATE.json, activity.log, spawn-tree.txt to `.aether/data/archive/{timestamp}/` |
| **Foundation Command** | Starting a new colony after archiving | Low | `/ant:foundation` - equivalent to `/ant:init` but with ant-themed messaging, clears/renames old state |
| **Milestone Detection** | Colony should auto-detect which milestone it's at based on state | Medium | Logic exists in status.md (lines 109-112) but needs implementation - detect based on phases completed, tests passing, etc. |
| **Proxy Health Check** | Model routing depends on LiteLLM proxy; users need visibility | Low | Already in build.md Step 0.6 - verify proxy at `http://localhost:4000/health` |
| **Fallback Model Behavior** | When proxy is down or model unavailable, need sensible default | Low | Already documented in workers.md (line 98-100) - default to kimi-k2.5 |

---

## Differentiators (Competitive Advantage)

Features that set Aether apart from generic agent frameworks.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Task-Based Model Routing** | Route to different models based on task keywords, not just caste | Medium | model-profiles.yaml has `task_routing` section (lines 71-82) with complexity indicators - implement keyword matching |
| **Caste Personality System** | Each caste has unique communication style and emoji identity | Low | Already in workers.md (lines 15-27, 402-413) - enhances UX but not strictly required |
| **Named Ant Generation** | Workers get unique names (e.g., "Hammer-42", "Vigil-17") for tracking | Low | Already implemented via `generate-ant-name` utility in aether-utils.sh |
| **Model Performance Telemetry** | Track which models perform best for which castes/tasks | Medium | Extend COLONY_STATE.json `memory` section to track model success rates per caste |
| **Intelligent Model Selection** | Auto-select model based on task complexity analysis | High | Analyze task description for complexity keywords, route to glm-5 for complex vs kimi-k2.5 for simple |
| **Milestone Progress Visualization** | Visual representation of colony maturity progression | Low | ASCII art or progress bar showing journey from First Mound to Crowned Anthill |
| **Colony History Timeline** | View archived colonies with goals, milestones, outcomes | Medium | Index archive directory, display summary of past colonies |
| **Cross-Colony Learning** | Inherit instincts from previous colonies via archive | Medium | Already partially implemented in init.md Step 2.5 - reads completion-report.md for instincts |

---

## Anti-Features (Deliberately NOT Building)

Features that seem good but create problems in this domain.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Per-Request Model Switching** | Too much overhead, breaks context continuity | Stick with caste-level assignment; castes exist precisely to group similar work |
| **Automatic Model Fallback Chain** | Complex retry logic hides real problems | Simple fallback to kimi-k2.5 with warning; let user fix proxy if needed |
| **Cloud-Based Model Routing** | Violates local-first, repo-local state principle | Keep routing local via LiteLLM proxy; user controls their proxy config |
| **Automatic Colony Archival** | User should consciously decide when work is complete | Require explicit `/ant:seal` or `/ant:archive` command |
| **Multiple Active Colonies** | Complexity without clear benefit | One active colony per repo; archive before starting new |
| **Model Cost Tracking** | Adds complexity, not core value | Defer to LiteLLM proxy's built-in tracking; Aether focuses on orchestration |
| **Real-Time Model Swapping** | Workers are spawned with a model; switching mid-task breaks context | Model is set at spawn time via environment variable |
| **Global Colony Registry** | Privacy concerns, unnecessary complexity | Keep colony data repo-local; optional registry-add is already implemented |

---

## Feature Dependencies

```
Model Verification (/ant:models)
    ├──requires──> Model Profiles YAML parsing
    │                   └──requires──> aether-utils.sh helper
    │
    └──enhances──> Model Override (--model flag)

Archive Command (/ant:archive)
    ├──requires──> COLONY_STATE.json exists
    ├──requires──> Archive directory structure
    │                   └──requires──> .aether/data/archive/ creation
    └──enhances──> Colony History Timeline

Foundation Command (/ant:foundation)
    ├──requires──> Archive Command (optional but recommended)
    ├──requires──> State reset capability
    └──conflicts──> Active EXECUTING state

Milestone Detection
    ├──requires──> Phase completion tracking
    ├──requires──> Test status checking
    ├──requires──> Build/lint status checking
    └──enhances──> Milestone Progress Visualization

Task-Based Routing
    ├──requires──> Task description analysis
    ├──requires──> Keyword matching logic
    └──enhances──> Model Verification (show routing rules)
```

### Dependency Notes

- **Archive requires completed phases:** Should warn if archiving with incomplete phases (already in seal.md logic)
- **Foundation should suggest archive:** If existing colony state detected, prompt user to archive first
- **Milestone detection requires multiple signals:** Phase completion alone is insufficient; need test status, build status
- **Task routing is enhancement, not replacement:** Caste-based routing remains default; task keywords are override

---

## Colony Milestone System

The six-stage milestone progression (already defined in seal.md and status.md):

| Milestone | Trigger | Description | Visual Indicator |
|-----------|---------|-------------|------------------|
| **First Mound** | Phase 1 complete | First runnable output | Single mound emoji |
| **Open Chambers** | 2+ phases complete | Feature work underway | Multiple chambers |
| **Brood Stable** | Tests consistently green | Quality baseline achieved | Stable structure |
| **Ventilated Nest** | Build + lint clean | Performance acceptable | Air flow metaphor |
| **Sealed Chambers** | All phases complete | Interfaces frozen | Sealed appearance |
| **Crowned Anthill** | User confirms via `/ant:seal` | Release-ready | Crowned, majestic |

### Milestone Auto-Detection Logic

```
IF all_phases_completed:
    IF user_confirmed_seal:
        milestone = "Crowned Anthill"
    ELSE:
        milestone = "Sealed Chambers"
ELIF build_clean AND lint_clean:
    milestone = "Ventilated Nest"
ELIF tests_consistently_green:
    milestone = "Brood Stable"
ELIF phases_completed >= 2:
    milestone = "Open Chambers"
ELIF phases_completed >= 1:
    milestone = "First Mound"
ELSE:
    milestone = "New Colony"
```

---

## Worker Caste Model Assignments

Current assignments from model-profiles.yaml:

| Caste | Model | Purpose | Context |
|-------|-------|---------|---------|
| prime | glm-5 | Long-horizon coordination | 200K context, strategic planning |
| archaeologist | glm-5 | Historical pattern analysis | Long timeframe analysis |
| architect | glm-5 | Pattern synthesis, documentation | Complex reasoning |
| oracle | minimax-2.5 | Research, foresight, browse/search | 76.3% BrowseComp |
| route_setter | kimi-k2.5 | Task decomposition, planning | 256K context, structured output |
| builder | kimi-k2.5 | Code generation, refactoring | 76.8% SWE-Bench |
| watcher | kimi-k2.5 | Validation, testing | Multimodal capable |
| scout | kimi-k2.5 | Research exploration | Parallel sub-agents |
| chaos | kimi-k2.5 | Edge case probing | Resilience testing |
| colonizer | kimi-k2.5 | Environment setup | Visual coding |

---

## MVP Definition (v3.1 Open Chambers)

### Launch With (v3.1.0)

Minimum features for "Open Chambers" milestone:

1. **Model Verification** (`/ant:models`) - Display current assignments
2. **Model Override** (`--model` flag) - Force specific model per command
3. **Archive Command** (`/ant:archive`) - Archive + reset colony state
4. **Foundation Command** (`/ant:foundation`) - Start fresh colony
5. **Milestone Auto-Detection** - Compute milestone from state
6. **Proxy Health Integration** - Verify proxy before model-dependent commands

### Add After Validation (v3.1.x)

Once core routing is stable:

1. **Task-Based Routing** - Keyword-based model selection
2. **Model Performance Telemetry** - Track success rates per model/caste
3. **Milestone Visualization** - Visual progress indicator

### Future Consideration (v3.2+)

Defer until routing system is mature:

1. **Intelligent Model Selection** - AI-driven complexity analysis
2. **Colony History Timeline** - Browse archived colonies
3. **Cross-Colony Analytics** - Compare model performance across colonies

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Model verification (/ant:models) | HIGH | Low | P1 |
| Archive command (/ant:archive) | HIGH | Medium | P1 |
| Foundation command (/ant:foundation) | HIGH | Low | P1 |
| Milestone auto-detection | MEDIUM | Medium | P1 |
| Model override (--model) | MEDIUM | Low | P1 |
| Proxy health check | MEDIUM | Low | P1 |
| Task-based routing | MEDIUM | Medium | P2 |
| Model performance telemetry | LOW | Medium | P2 |
| Milestone visualization | LOW | Low | P2 |
| Colony history timeline | LOW | Medium | P3 |
| Intelligent model selection | MEDIUM | High | P3 |

---

## Implementation Notes

### Model Routing Implementation

The routing system is already partially implemented:

1. **Configuration**: `model-profiles.yaml` defines caste-to-model mappings
2. **Environment**: Workers receive `ANTHROPIC_MODEL` env var based on caste
3. **Proxy**: LiteLLM at `localhost:4000` routes to actual providers
4. **Fallback**: Default to kimi-k2.5 if profile missing

What needs to be added:
- CLI command to view current assignments
- `--model` flag override mechanism
- Task keyword analysis for dynamic routing

### Colony Lifecycle Implementation

Lifecycle commands already exist in various forms:

- **Init**: `/ant:init` - Creates fresh colony state
- **Seal**: `/ant:seal` - Archives completed colony (Crowned Anthill)
- **Status**: `/ant:status` - Shows current milestone

What needs to be added:
- `/ant:archive` - Archive without requiring completion
- `/ant:foundation` - Re-init with ant-themed messaging
- Milestone auto-calculation logic

---

## Sources

- `.aether/workers.md` - Worker caste definitions, model assignments, personality system
- `.aether/model-profiles.yaml` - Model metadata, routing configuration
- `.claude/commands/ant/init.md` - Colony initialization logic
- `.claude/commands/ant/seal.md` - Colony sealing/archival logic
- `.claude/commands/ant/status.md` - Milestone display
- `.claude/commands/ant/build.md` - Proxy health check pattern
- `.aether/data/COLONY_STATE.json` - State structure for milestone detection
- `.planning/PROJECT.md` - v3.1 milestone goals

**Confidence Assessment:**

| Area | Level | Reason |
|------|-------|--------|
| Table Stakes | HIGH | Based on existing command patterns and yaml config |
| Differentiators | MEDIUM | Some features exist in config but not implemented |
| Anti-Features | HIGH | Clear from project constraints (local-first, simple) |
| Dependencies | HIGH | Clear from existing command structure |
| Milestone System | HIGH | Already defined in seal.md and status.md |

---
*Feature research for: v3.1 Open Chambers - Model Routing & Colony Lifecycle*
*Researched: 2026-02-14*
