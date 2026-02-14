# Model Routing Implementation Archive

**Archived:** 2026-02-15
**Reason:** Claude Code Task tool limitation prevents environment variable inheritance
**Status:** Complete implementation, non-functional due to platform constraints

---

## What Was Built

A comprehensive model routing system that maps Aether colony castes to specific AI models through a LiteLLM proxy. The infrastructure is fully functional; the execution layer is blocked by Claude Code limitations.

### Components Archived

1. **Configuration System** - `model-profiles.yaml` with caste-to-model mappings
2. **Library Code** - `bin/lib/model-profiles.js` for model selection logic
3. **CLI Integration** - `aether caste-models list/set` commands
4. **Shell Utilities** - `aether-utils.sh` model-profile subcommands
5. **Documentation** - Workers.md, command references, research notes
6. **Helper Scripts** - `spawn-with-model.sh` (non-functional)

---

## Why It Doesn't Work

### The Core Problem

Claude Code's Task tool **does not support environment variable passing** to spawned subagents.

**Design Assumption:**
```
Parent sets ANTHROPIC_MODEL=glm-5
→ Task tool spawns worker
→ Worker inherits ANTHROPIC_MODEL
→ LiteLLM proxy routes to glm-5
```

**Reality:**
```
Parent sets ANTHROPIC_MODEL=glm-5
→ Task tool spawns worker with FRESH environment
→ Worker uses default model (not glm-5)
→ All castes use same model regardless of assignment
```

### Evidence

From `.aether/workers.md` (lines 118-120):
> "Claude Code's Task tool doesn't support explicit environment variable passing, so proxy routing relies on parent shell inheritance."

From `TO-DOS.md` (lines 46-52):
> "Phase 9 built all the infrastructure for model routing, but we haven't proven spawned workers actually use different models."

From `TO-DOS.md` (lines 68-70):
> "Task tool doesn't inherit environment (Claude Code limitation)"

---

## What Would Be Needed To Fix This

### Option 1: Claude Code Feature Request
Task tool accepting `env:` parameter:
```javascript
Task({
  prompt: "...",
  env: { ANTHROPIC_MODEL: "glm-5" }  // Does not exist
})
```

### Option 2: Alternative Architecture
Spawn separate Claude Code processes with different environment variables.

### Option 3: Proxy-Level Routing
Route based on prompt content/prefix rather than environment variable.

### Option 4: Accept Limitation
Document that all workers use the same model as the parent session.

---

## File Inventory

| File | Purpose | Status |
|------|---------|--------|
| `model-profiles.yaml` | Caste → model mappings | Functional |
| `bin/lib/model-profiles.js` | Model selection library | Functional |
| `bin/cli.js` | CLI commands | Functional |
| `aether-utils.sh` | Shell utilities | Functional |
| `spawn-with-model.sh` | Helper script | Non-functional |
| `workers.md` | Documentation | Outdated |
| `build.md` | Command reference | Outdated |

---

## How To Use This Archive

If Claude Code adds Task tool environment variable support:

1. Restore `model-profiles.yaml` to `.aether/`
2. Restore `bin/lib/model-profiles.js` to `bin/lib/`
3. Update CLI to call `spawn-with-model.sh` before Task tool
4. Update documentation to reflect working model routing

---

## Related Research

See `.planning/research/PITFALLS.md` (lines 46-72) for proxy authentication issues that compound the routing problem.

---

*This archive preserves the complete model routing implementation for future use when platform limitations are resolved.*
