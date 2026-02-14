# Architecture Research: Model Routing & Colony Lifecycle

**Domain:** Aether Colony System - Multi-model LLM orchestration with colony lifecycle management
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

This research addresses how two new v3.1 capabilities integrate with the existing Aether Colony System architecture:

1. **Model Routing Per Caste**: Intelligent routing of worker castes to optimal LLM models (glm-5 for prime/architect, kimi-k2.5 for builder, etc.) through the LiteLLM proxy
2. **Colony Lifecycle Management**: Ant-themed milestone commands (`/ant:archive`, `/ant:foundation`) for colony archive/reset operations with auto-detection of colony maturity level

The architecture must extend the existing four-layer system (Queen, Constraints, Worker, Utility) while maintaining compatibility with the current state management patterns and worker spawning mechanisms.

## Current Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        QUEEN LAYER                           │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │  init   │  │  plan   │  │  build  │  │continue │        │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘        │
│       │            │            │            │              │
├───────┴────────────┴────────────┴────────────┴──────────────┤
│                      CONSTRAINTS LAYER                       │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              focus/avoid rules (pheromones)          │    │
│  └─────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                        WORKER LAYER                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Builder  │  │ Watcher  │  │  Scout   │  │  Oracle  │    │
│  │  🔨      │  │  👁️      │  │  🔍      │  │  🔮      │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│                     MODEL ROUTING LAYER                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │model-profiles│  │  spawn-with  │  │   LiteLLM    │      │
│  │   .yaml      │  │   -model.sh  │  │   Proxy      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
├─────────────────────────────────────────────────────────────┤
│                        UTILITY LAYER                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │aether-   │  │  file-   │  │ atomic-  │  │  state-  │    │
│  │utils.sh  │  │  lock.sh │  │ write.sh │  │ loader.sh│    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│                        DATA LAYER                            │
│  ┌──────────────┐ ┌──────────────┐ ┌────────────────────┐  │
│  │COLONY_STATE  │ │ spawn-tree   │ │   model-profile    │  │
│  │   .json      │ │   .txt       │ │   .json (new)      │  │
│  └──────────────┘ └──────────────┘ └────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Current Implementation |
|-----------|----------------|------------------------|
| Queen Layer | Orchestration, phase management, worker spawning | `.opencode/agents/aether-queen.md` |
| Constraints Layer | Declarative focus/avoid rules | `.aether/data/constraints.json` |
| Worker Layer | Task execution with depth limits | Spawned via `task` tool, tracked in `spawn-tree.txt` |
| Model Routing Layer | Caste-to-model mapping, proxy routing | `.aether/model-profiles.yaml`, `spawn-with-model.sh` |
| Utility Layer | Deterministic operations, state management | `.aether/aether-utils.sh` |
| Data Layer | Persistent state, activity logs, model configs | `.aether/data/` directory |

## Model Routing Architecture

### Current State Analysis

The model routing infrastructure exists but has a documentation-to-execution gap:

**What Exists:**
- `model-profiles.yaml` - Comprehensive caste-to-model mappings
- `spawn-with-model.sh` - Environment setup script
- `aether-utils.sh model-profile` commands - Profile querying
- COLONY_STATE.json tracks `model_profile.active_profile`

**The Gap (identified in dream analysis):**
- Configuration exists but actual routing verification is incomplete
- ANTHROPIC_MODEL may not be consistently set before Task spawns
- Task-based keyword routing ("design" → glm-5) is aspirational, not verified

### Recommended Architecture

```
Model Routing Flow:
┌─────────────────────────────────────────────────────────────┐
│                     SPAWN REQUEST                            │
│              (Queen or Worker spawns sub-worker)            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  CASTE DETERMINATION                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  1. Explicit caste parameter from spawn request     │    │
│  │  2. Task keyword analysis (if no explicit caste)    │    │
│  │  3. Default to "builder" for implementation tasks   │    │
│  └─────────────────────────────────────────────────────┘    │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                MODEL PROFILE RESOLUTION                      │
│                                                              │
│  Query: bash aether-utils.sh model-profile get <caste>      │
│                                                              │
│  Returns: { model: "kimi-k2.5", provider: "kimi", ... }     │
│                                                              │
│  Fallback chain:                                            │
│    1. Profile-specific caste mapping                        │
│    2. Default profile caste mapping                         │
│    3. Global default: "kimi-k2.5"                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              ENVIRONMENT VARIABLE INJECTION                  │
│                                                              │
│  ANTHROPIC_BASE_URL=http://localhost:4000                   │
│  ANTHROPIC_AUTH_TOKEN=sk-litellm-local                      │
│  ANTHROPIC_MODEL=<resolved-model>                           │
│                                                              │
│  These are inherited by Claude Code via Task tool           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  LITELLM PROXY ROUTING                       │
│                                                              │
│  localhost:4000 ──► model alias resolution ──► Provider     │
│                                                              │
│  glm-5      ──► Z_AI API (744B MoE, 200K context)          │
│  kimi-k2.5  ──► Kimi API (1T MoE, 256K context)            │
│  minimax-2.5 ──► MiniMax API (fast, browse-capable)        │
└─────────────────────────────────────────────────────────────┘
```

### Integration Points

| Integration | How | File/Component |
|-------------|-----|----------------|
| Caste detection | Parse task description for keywords | `build.md` spawn logic |
| Model resolution | `model-profile get <caste>` command | `aether-utils.sh` |
| Environment setup | Export ANTHROPIC_MODEL before spawn | `spawn-with-model.sh` |
| Proxy health | `curl localhost:4000/health` | `build.md` Step 0.6 |
| Verification | `aether verify-models` CLI command | `bin/cli.js` |

### Data Flow Changes

**Current (v3.0 - partial implementation):**
```bash
# In build.md - sets model but doesn't verify
model=$(bash aether-utils.sh model-profile get "$caste" | jq -r '.result.model')
export ANTHROPIC_MODEL="$model"
# ... spawn worker
```

**Recommended (v3.1 - verified routing):**
```bash
# 1. Resolve model with fallback chain
model_info=$(bash aether-utils.sh model-profile get "$caste")
model=$(echo "$model_info" | jq -r '.result.model // "kimi-k2.5"')

# 2. Verify proxy health
if curl -s http://localhost:4000/health | grep -q "healthy"; then
    routing_status="active"
else
    routing_status="degraded (using default)"
fi

# 3. Log routing decision
bash aether-utils.sh activity-log "MODEL" "Queen" "$caste → $model ($routing_status)"

# 4. Export for Task tool inheritance
export ANTHROPIC_BASE_URL="http://localhost:4000"
export ANTHROPIC_AUTH_TOKEN="sk-litellm-local"
export ANTHROPIC_MODEL="$model"

# 5. Spawn with model context in prompt
# (Include MODEL CONTEXT section in worker prompt)
```

## Colony Lifecycle Architecture

### Concept: Ant-Themed Milestones

Replace generic CDS milestone commands with colony-appropriate terminology:

| CDS Term | Aether Term | Meaning |
|----------|-------------|---------|
| `/cds:milestone-archive` | `/ant:archive` | Archive colony history, reset to fresh |
| `/cds:milestone-foundation` | `/ant:foundation` | Start completely fresh colony |
| Milestone detection | Colony maturity detection | Auto-detect based on state/completion |

### Colony Maturity Levels

```
Colony Lifecycle States:

┌─────────────────────────────────────────────────────────────┐
│                    🥚 EGG (Initial)                          │
│  No COLONY_STATE.json exists                                 │
│  Commands: /ant:init to hatch                                │
└────────────────────────┬────────────────────────────────────┘
                         │ /ant:init
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              🐜 LARVA (Goal Set, No Plan)                    │
│  COLONY_STATE.json exists, goal set, no phases               │
│  Commands: /ant:plan to develop                              │
└────────────────────────┬────────────────────────────────────┘
                         │ /ant:plan
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           🏗️ FIRST MOUND (Planning Complete)                 │
│  Phases defined, ready to build                              │
│  Commands: /ant:build 1 to begin construction                │
└────────────────────────┬────────────────────────────────────┘
                         │ Complete phases
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              🌿 OPEN CHAMBERS (In Progress)                  │
│  Some phases completed, colony active                        │
│  Commands: /ant:continue, /ant:build N                       │
└────────────────────────┬────────────────────────────────────┘
                         │ Complete all phases
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              🏛️ BROOD STABLE (Completed)                     │
│  All phases completed, goal achieved                         │
│  Commands: /ant:archive to preserve, /ant:foundation to reset│
└─────────────────────────────────────────────────────────────┘
```

### Lifecycle Command Architecture

```
/ant:archive Flow:
┌─────────────────────────────────────────────────────────────┐
│                     VALIDATION PHASE                         │
├─────────────────────────────────────────────────────────────┤
│  1. Verify colony exists and has history                     │
│  2. Confirm with user (destructive operation)                │
│  3. Check for uncommitted changes (warn if present)          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     ARCHIVE PHASE                            │
├─────────────────────────────────────────────────────────────┤
│  1. Create archive directory: .aether/archive/<timestamp>/   │
│  2. Copy COLONY_STATE.json → archive/                        │
│  3. Copy constraints.json → archive/                         │
│  4. Copy flags.json → archive/                               │
│  5. Copy spawn-tree.txt → archive/                           │
│  6. Copy activity.log → archive/                             │
│  7. Create archive manifest (metadata.json)                  │
│  8. Create summary report (SUMMARY.md)                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     RESET PHASE                              │
├─────────────────────────────────────────────────────────────┤
│  1. Reset COLONY_STATE.json to v3.0 template                 │
│  2. Clear constraints.json (keep structure)                  │
│  3. Clear flags.json (keep structure)                        │
│  4. Truncate spawn-tree.txt                                  │
│  5. Archive (don't delete) activity.log                      │
│  6. Set state to "IDLE", goal to null                        │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     CONFIRMATION PHASE                       │
├─────────────────────────────────────────────────────────────┤
│  Display:                                                    │
│  ✅ Colony archived to .aether/archive/YYYYMMDD-HHMMSS/      │
│  ✅ Colony reset to EGG state                                │
│  Next: /ant:init "new goal" to start fresh colony            │
└─────────────────────────────────────────────────────────────┘
```

```
/ant:foundation Flow:
┌─────────────────────────────────────────────────────────────┐
│                     VALIDATION PHASE                         │
├─────────────────────────────────────────────────────────────┤
│  1. Strong confirmation required (type "FOUNDATION")         │
│  2. Warn: This deletes all colony history (no archive)       │
│  3. Check for active workers (block if building)             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     DESTRUCTION PHASE                        │
├─────────────────────────────────────────────────────────────┤
│  1. Delete COLONY_STATE.json                                 │
│  2. Delete constraints.json                                  │
│  3. Delete flags.json                                        │
│  4. Delete spawn-tree.txt                                    │
│  5. Delete activity.log                                      │
│  6. Remove .aether/data/ directory                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     BOOTSTRAP PHASE                          │
├─────────────────────────────────────────────────────────────┤
│  1. Recreate .aether/data/ directory                         │
│  2. Initialize fresh COLONY_STATE.json (v3.0 template)       │
│  3. Initialize empty constraints.json                        │
│  4. Initialize empty flags.json                              │
│  5. Create fresh activity.log                                │
│  6. Set state to "IDLE"                                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                     CONFIRMATION PHASE                       │
├─────────────────────────────────────────────────────────────┤
│  Display:                                                    │
│  🔥 Colony foundation laid — all history cleared             │
│  🥚 State: EGG (ready for /ant:init)                         │
└─────────────────────────────────────────────────────────────┘
```

### Colony Maturity Detection

```javascript
// Algorithm for /ant:status maturity display
function detectColonyMaturity(state) {
  // EGG: No state file or no goal
  if (!state || state.goal === null) {
    return { level: "EGG", emoji: "🥚", description: "Ready to hatch" };
  }

  // LARVA: Goal set but no phases
  if (!state.plan?.phases || state.plan.phases.length === 0) {
    return { level: "LARVA", emoji: "🐜", description: "Developing plan" };
  }

  const totalPhases = state.plan.phases.length;
  const completedPhases = state.plan.phases.filter(p => p.status === "completed").length;

  // FIRST MOUND: Plan exists but no phases completed
  if (completedPhases === 0) {
    return { level: "FIRST_MOUND", emoji: "🏗️", description: "Ready to build" };
  }

  // BROOD STABLE: All phases completed
  if (completedPhases === totalPhases) {
    return { level: "BROOD_STABLE", emoji: "🏛️", description: "Colony complete" };
  }

  // OPEN CHAMBERS: In progress
  return {
    level: "OPEN_CHAMBERS",
    emoji: "🌿",
    description: `${completedPhases}/${totalPhases} chambers complete`
  };
}
```

## Component Boundaries

### What Talks to What

```
┌─────────────────────────────────────────────────────────────┐
│                    COMMUNICATION MATRIX                      │
├──────────────────┬──────────────────┬───────────────────────┤
│ Source           │ Target           │ Method                │
├──────────────────┼──────────────────┼───────────────────────┤
│ Queen (build.md) │ Model Routing    │ bash aether-utils.sh  │
│ Queen (build.md) │ LiteLLM Proxy    │ curl health check     │
│ Worker spawn     │ Environment      │ export ANTHROPIC_*    │
│ CLI (cli.js)     │ Model Verify     │ model-verify.js       │
│ Archive command  │ State files      │ file copy/move        │
│ Foundation cmd   │ Data layer       │ file delete/create    │
│ Status command   │ Maturity detect  │ state analysis        │
└──────────────────┴──────────────────┴───────────────────────┘
```

### Data Flow

```
Model Routing Data Flow:
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Worker     │────▶│model-profile │────▶│  LiteLLM     │
│   Spawn      │     │   get cmd    │     │   Proxy      │
└──────────────┘     └──────────────┘     └──────┬───────┘
       │                                         │
       │                                         ▼
       │                                ┌──────────────┐
       │                                │   Provider   │
       │                                │   API        │
       │                                └──────────────┘
       │
       ▼
┌──────────────┐
│  ANTHROPIC_  │
│  MODEL env   │
│  (inherited  │
│  by Task)    │
└──────────────┘

Lifecycle Data Flow:
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   /ant:      │────▶│   Archive    │────▶│  .aether/    │
│  archive     │     │   Logic      │     │  archive/    │
└──────────────┘     └──────────────┘     └──────────────┘
       │
       ├──────────────────────────────────────────┐
       ▼                                          ▼
┌──────────────┐                         ┌──────────────┐
│  Reset State │                         │  Fresh State │
│  (IDLE, null │                         │  (IDLE, null │
│   goal)      │                         │   goal)      │
└──────────────┘                         └──────────────┘
     Archive path                              Foundation path
```

## Suggested Build Order

Based on dependencies between features:

### Phase 1: Model Routing Verification (Week 1)

1. **Model Verification Command** (`aether verify-models`)
   - No dependencies
   - Provides visibility into current routing state
   - Changes: `bin/cli.js` add verify-models command
   - Tests: Unit tests for model-verify.js

2. **Verified Worker Spawning** (update `build.md`)
   - Depends on: Model verification
   - Ensures ANTHROPIC_MODEL is consistently set
   - Changes: `build.md` spawn logic, add MODEL CONTEXT to prompts
   - Tests: E2E test that verifies model env is set

3. **Task-Based Routing** (keyword detection)
   - Depends on: Verified spawning
   - Implements "design" → glm-5, "validate" → minimax-2.5
   - Changes: `build.md` task analysis logic
   - Tests: Unit tests for keyword detection

### Phase 2: Colony Lifecycle (Week 2)

4. **Archive Command** (`/ant:archive`)
   - No dependencies
   - Safely preserves colony history
   - Changes: New `archive.md` command, archive logic in utils
   - Tests: E2E test for archive → reset → continue flow

5. **Foundation Command** (`/ant:foundation`)
   - No dependencies
   - Complete reset with confirmation
   - Changes: New `foundation.md` command
   - Tests: E2E test for foundation → init flow

6. **Maturity Detection** (update `/ant:status`)
   - Depends on: Archive/Foundation (for state transitions)
   - Auto-detects colony maturity level
   - Changes: `status.md` command, maturity detection logic
   - Tests: Unit tests for detectColonyMaturity function

### Phase 3: Integration (Week 3)

7. **Dreams Integration** (surface in `/ant:status`)
   - Depends on: Maturity detection
   - Shows recent dreams in status output
   - Changes: `status.md` dream reading logic

8. **Auto-Load Context** (HANDOFF.md improvements)
   - Depends on: Archive/Foundation (clean state handling)
   - Automatically loads relevant context on init
   - Changes: `init.md`, `continue.md` context loading

### Dependency Graph

```
                    ┌─────────────────────┐
                    │  Model Verification │
                    │     (Week 1)        │
                    └──────────┬──────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
         ▼                     ▼                     ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│ Verified Spawn  │   │ Task-Based      │   │ Archive Command │
│ (build.md)      │   │ Routing         │   │ (Week 2)        │
└─────────────────┘   └─────────────────┘   └────────┬────────┘
                                                     │
         ┌───────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│ Foundation Cmd  │◀──│ Maturity Detect │──▶│ Dreams in       │
│ (Week 2)        │   │ (Week 2)        │   │ Status (Week 3) │
└─────────────────┘   └────────┬────────┘   └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │ Auto-Load       │
                        │ Context (Week 3)│
                        └─────────────────┘
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Model Routing Without Verification
**What people do:** Set ANTHROPIC_MODEL but never verify it reaches the worker
**Why it's wrong:** Silent fallback to default model defeats purpose of routing
**Do this instead:** Always verify with `aether verify-models` and log routing decisions

### Anti-Pattern 2: Archive Without Manifest
**What people do:** Copy files to archive without metadata
**Why it's wrong:** Can't understand archive contents later, no reproducibility
**Do this instead:** Always create manifest.json with timestamp, goal summary, phase count

### Anti-Pattern 3: Foundation Without Confirmation
**What people do:** Allow destructive reset with simple confirmation
**Why it's wrong:** Accidental data loss, no recovery path
**Do this instead:** Require typed confirmation ("FOUNDATION"), warn about data loss

### Anti-Pattern 4: Hardcoded Model Names
**What people do:** Use "kimi-k2.5" directly in spawn code
**Why it's wrong:** Bypasses profile configuration, hard to change
**Do this instead:** Always query `model-profile get <caste>` for resolution

### Anti-Pattern 5: Lifecycle Commands Without State Checks
**What people do:** Allow archive/foundation while workers are active
**Why it's wrong:** Corrupts running operations, causes errors
**Do this instead:** Check state === "EXECUTING" and block with clear message

## Scalability Considerations

| Concern | At 1 colony | At 10 colonies | At 100 colonies |
|---------|-------------|----------------|-----------------|
| Model profiles | Single YAML file | Profile inheritance | Profile versioning |
| Archive storage | Local .aether/archive/ | Rotate old archives | Archive compression |
| Proxy routing | localhost:4000 | Shared proxy instance | Load-balanced proxy |
| Activity logs | Single file | Log rotation | Structured logging |

### Archive Retention Strategy

```yaml
# Proposed archive retention in model-profiles.yaml
archive_retention:
  max_archives: 10           # Keep last 10 colonies
  max_age_days: 90           # Or 90 days, whichever is less
  compress_after_days: 30    # Gzip archives older than 30 days
  auto_cleanup: true         # Run on /ant:archive
```

## Sources

- `.aether/model-profiles.yaml` - Caste-to-model mappings
- `.aether/workers.md` - Model-aware spawning documentation
- `.aether/utils/spawn-with-model.sh` - Environment setup pattern
- `.claude/commands/ant/build.md` - Current spawn logic
- `bin/cli.js` - CLI command structure
- `.aether/dreams/2026-02-14-0238.md` - Gap analysis (model routing verification)
- `.planning/PROJECT.md` - v3.1 milestone definition
- `.aether/QUEEN_ANT_ARCHITECTURE.md` - Colony architecture principles

---
*Architecture research for: Aether Colony v3.1 Open Chambers*
*Researched: 2026-02-14*
