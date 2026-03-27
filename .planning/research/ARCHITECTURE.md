# Architecture Research: Per-Caste Model Routing

**Domain:** Multi-agent model routing within Claude Code Task tool
**Researched:** 2026-03-27
**Confidence:** HIGH (based on direct codebase analysis of all relevant files + GSD proven pattern)

---

## Executive Summary

Per-caste model routing has been attempted once before (v1, archived 2026-02-15) and failed due to Claude Code's Task tool not passing environment variables to spawned subagents. Since then, two things have changed that make this feasible now:

1. **Claude Code's Task/Agent tool now supports a `model` parameter** that can be set to `inherit`, `sonnet`, `opus`, or `haiku`. This is proven working in the GSD system (`.claude/get-shit-done/workflows/new-project.md` line 585: `model="{researcher_model}"`).

2. **The `ANTHROPIC_DEFAULT_*_MODEL` environment variables** in `~/.claude/settings.json` map Claude Code's model slots to arbitrary model names via the LiteLLM proxy. This is already configured (line 6-8 of `~/.claude/settings.json`).

The mechanism is: **Agent frontmatter `model: inherit`** (currently set on all 22 agents) controls which model slot a worker uses. By changing specific agent frontmatter from `model: inherit` to `model: opus` or `model: sonnet`, and configuring the environment variable mapping in `~/.claude/settings.json`, each caste can be routed to a different underlying model through the LiteLLM proxy.

**The key insight:** The routing does NOT happen through environment variables or the model-profiles.yaml library at spawn time. It happens through Claude Code's own model slot system, which the LiteLLM proxy then maps to actual model endpoints. The model-profiles.yaml infrastructure is still useful for documentation, CLI commands, and task-based routing, but the actual model selection is the `model:` frontmatter field on agent definitions.

---

## Question 1: Current Flow from Colony-Prime Decision to Spawn

### What Actually Happens Today (All Workers Use Same Model)

```
1. User runs: /ant:build 1
2. build.md (Queen) parses $ARGUMENTS, validates state, loads context
3. build-wave.md Step 5.1: Queen reads task list, groups by dependencies
4. For each Wave 1 task:
   a. Queen generates ant name via: bash .aether/aether-utils.sh generate-ant-name "builder"
   b. Queen matches skills via: bash .aether/aether-utils.sh skill-match "builder" "..."
   c. Queen constructs worker prompt (with prompt_section, skill_section, archaeology_context, etc.)
   d. Queen calls Task tool:
      Task(prompt="...", subagent_type="aether-builder", description="...")
5. Claude Code looks up aether-builder agent definition
   -> frontmatter says: model: inherit
   -> "inherit" means: use the parent session's model (glm-5-turbo via LiteLLM proxy)
6. Worker runs with glm-5-turbo regardless of caste
```

**Critical detail:** The `subagent_type` parameter maps to an agent definition in `.claude/agents/ant/`. The `model:` field in that agent's frontmatter determines which model slot the worker uses. Currently ALL 22 agents have `model: inherit`, so every worker inherits the parent session's model.

### Where Model Selection Could Happen

There are two independent mechanisms, both potentially useful:

**Mechanism A: Agent frontmatter `model:` field (PROVEN WORKING)**
- Location: `.claude/agents/ant/aether-*.md` frontmatter, line 6
- Values: `inherit`, `sonnet`, `opus`, `haiku`
- How it works: Claude Code reads this field when spawning via Task tool, selects the corresponding model
- Proven in: GSD system uses `model="{researcher_model}"` in Task tool calls

**Mechanism B: Task tool `model` parameter (PROVEN WORKING in GSD)**
- Location: The Task() call itself in build-wave.md
- Syntax: `Task(prompt="...", subagent_type="...", model="opus", description="...")`
- How it works: Overrides the agent definition's model field for that specific spawn
- Proven in: GSD new-project.md spawns researchers with `model="{researcher_model}"`

**Mechanism C: model-profiles.yaml + environment variables (PREVIOUSLY FAILED)**
- Location: `.aether/model-profiles.yaml`, read by `bin/lib/model-profiles.js`
- Problem: Claude Code Task tool does NOT pass environment variables to spawned workers
- Status: Infrastructure exists and works (CLI commands, library, validation) but cannot drive actual model routing
- Archived at: `.aether/archive/model-routing/README.md`

### Recommendation: Use Mechanism B (Task tool `model` parameter)

This is the most flexible approach because:
1. **Caste-based routing without modifying agent definitions** -- the build-wave playbook can look up the caste's model slot and pass it in the Task call
2. **No sync burden** -- changing model assignments doesn't require editing 22 agent files
3. **CLI override still works** -- `--model` flag in `/ant:build` can override everything
4. **GSD already proves this works** -- the exact pattern is in production

---

## Question 2: Where model-profiles.yaml Gets Read During a Build

### Code Paths That Read model-profiles.yaml

| Entry Point | Code Path | When Called | Purpose |
|-------------|-----------|------------|---------|
| `/ant:build --model X` | build-prep.md Step 1 -> `model-profile validate "$cli_model_override"` | Build start | Validates CLI override model name |
| `aether caste-models list` | `bin/cli.js` line 1979 -> `loadModelProfiles()` | Manual CLI | Displays caste-to-model table |
| `aether caste-models set builder=glm-5` | `bin/cli.js` line 2076 -> `setModelOverride()` | Manual CLI | Sets user override in YAML |
| `aether verify-models` | `bin/cli.js` line 2155 -> verification logic | Manual CLI | Checks routing configuration |
| `spawn-with-model.sh` | `.aether/utils/spawn-with-model.sh` line 23 -> `model-profile get "$CASTE"` | **NOT USED** | Legacy helper, non-functional |
| `model-profile get <caste>` | `aether-utils.sh` line 2616 -> awk on YAML | Shell calls | Returns model for caste (JSON) |
| `model-profile select <caste> <task>` | `aether-utils.sh` line 2668 -> Node.js inline script | Shell calls | Full precedence chain selection |

### Critical Finding: model-profiles.yaml Is NOT Read During Worker Spawn

The build-wave.md playbook (Step 5.1) does NOT call `model-profile get` or `model-profile select` when spawning workers. It only uses:
1. `generate-ant-name` -- naming the worker
2. `skill-match` / `skill-inject` -- loading skills
3. `spawn-log` -- logging the spawn
4. `colony-prime --compact` -- loading context

**The model-profiles.yaml infrastructure is completely disconnected from the actual spawn path.** This is why the previous attempt failed -- the infrastructure was built but never wired into the spawn mechanism.

### What Needs to Change

The build-wave.md playbook needs to:
1. Determine the correct model slot for each worker's caste (from model-profiles.yaml or a lookup table)
2. Pass `model="{resolved_slot}"` in the Task tool call

The model-profiles.yaml can serve as the source of truth for which caste maps to which model slot, but the shell subcommand that reads it (`model-profile get`) must be called during the build prep phase, and the result must be threaded into the Task tool call.

---

## Question 3: How spawn-with-model.sh Determines the Model

### Script Analysis (`.aether/utils/spawn-with-model.sh`)

```bash
# Line 23: Get model for this caste
model_info=$(bash "$AETHER_ROOT/.aether/aether-utils.sh" model-profile get "$CASTE" 2>/dev/null || echo '{"ok":true,"result":{"model":"glm-5-turbo"}}')
model=$(echo "$model_info" | jq -r '.result.model // "glm-5-turbo"')

# Line 34: Set environment variable
export ANTHROPIC_MODEL="$model"

# Line 50: Start Claude Code with the environment set
claude --cwd "$PROJECT_ROOT"
```

**How it works:**
1. Calls `model-profile get <caste>` to get the model name from YAML
2. Sets `ANTHROPIC_MODEL` environment variable
3. Starts Claude Code CLI as a subprocess, inheriting the environment

**Why it doesn't work for colony builds:**
- This script spawns a **separate Claude Code process**, not a Task tool subagent
- Colony workers are spawned via Claude Code's Task tool, not as separate processes
- The Task tool does not inherit environment variables from the parent shell
- This script was designed for an architecture where each worker runs in its own Claude Code instance

**Verdict:** This script is a vestige of the failed v1 approach. It should be kept for reference but is not useful for the new per-caste routing mechanism.

---

## Question 4: How build.md and continue.md Pass the Model to the Agent Tool

### Current State: They Don't

**build.md / build-full.md:** The build playbooks spawn workers using:
```
Task tool with subagent_type="aether-builder", description="..."
```

There is NO `model` parameter in any of the Task tool calls across the entire build system. The model is entirely determined by the agent definition's `model: inherit` frontmatter.

**continue.md / continue-full.md:** The continue playbooks do NOT spawn any workers (they verify and advance state). No Task tool calls at all.

**Other commands that spawn workers:**
| Command | Spawns | Model Parameter |
|---------|--------|-----------------|
| `/ant:build` | Builders, Watcher, Chaos, Archaeologist, Ambassador, Measurer | None |
| `/ant:swarm` | 4 Scouts via Task tool | None |
| `/ant:colonize` | 4 Surveyors via Task tool | None |
| `/ant:seal` | Sage, Chronicler via Task tool | None |
| `/ant:patrol` | Watcher via Task tool | None |
| `/ant:organize` | Keeper via Task tool | None |
| `/gsd:new-project` | Researchers, Roadmapper, Synthesizer | **YES** (`model="{researcher_model}"`) |

### What Needs to Change

Every command that uses the Task tool needs to include the `model` parameter, resolved from the caste's model slot. The build system is the highest priority since it spawns the most workers (6-8 per phase).

---

## Question 5: How the Model Parameter Actually Reaches the Agent Tool

### The Proven Mechanism (from GSD)

The GSD system demonstrates the exact pattern that works:

**Step 1: Resolve model at workflow start**
```bash
# .claude/get-shit-done/workflows/new-project.md, line 49
INIT=$(node ./.claude/get-shit-done/bin/gsd-tools.cjs init new-project)
# Returns JSON with: researcher_model, synthesizer_model, roadmapper_model
```

**Step 2: Use model in Task tool call**
```javascript
// .claude/get-shit-done/workflows/new-project.md, line 585
Task(prompt="...", subagent_type="general-purpose", model="{researcher_model}", description="Stack research")
```

**Step 3: Claude Code resolves the model slot**
- `model="inherit"` -> Uses parent session's model
- `model="sonnet"` -> Uses Claude's sonnet model (or `ANTHROPIC_DEFAULT_SONNET_MODEL` if set)
- `model="opus"` -> Uses Claude's opus model (or `ANTHROPIC_DEFAULT_OPUS_MODEL` if set)
- `model="haiku"` -> Uses Claude's haiku model (or `ANTHROPIC_DEFAULT_HAIKU_MODEL` if set)

**Step 4: LiteLLM proxy maps to actual model**
The `~/.claude/settings.json` environment variables tell Claude Code what model name to send to the API:
```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://localhost:4000",
    "ANTHROPIC_AUTH_TOKEN": "sk-litellm-local",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5-turbo",    // Currently all same
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "glm-5-turbo",   // Currently all same
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "glm-4.5-air"    // Different!
  }
}
```

The LiteLLM proxy at `localhost:4000` receives the model name (e.g., `glm-5-turbo`) and routes to the appropriate provider backend.

### The Complete Chain

```
Task tool call: model="opus"
        |
        v
Claude Code resolves "opus" to environment variable:
  ANTHROPIC_DEFAULT_OPUS_MODEL = "glm-5"  (configured in settings.json)
        |
        v
Claude Code sends API request:
  POST http://localhost:4000/v1/messages
  model: "glm-5"
  (via ANTHROPIC_BASE_URL and ANTHROPIC_AUTH_TOKEN)
        |
        v
LiteLLM proxy receives "glm-5":
  Routes to Z.AI GLM-5 provider backend
        |
        v
Worker runs on GLM-5 (deep reasoning model)
```

### Is It In the Prompt Text or a Function Call?

**It is a function parameter, not prompt text.** The `model` parameter is a first-class argument to Claude Code's Task/Agent tool. It is NOT included in the prompt string. This is why the previous approach failed -- they tried to pass it through environment variables and prompt text, but the Task tool has its own `model` parameter.

---

## Integration Architecture

### New vs Modified Components

| Component | Status | Change Required |
|-----------|--------|-----------------|
| `~/.claude/settings.json` | MODIFY | Change `ANTHROPIC_DEFAULT_OPUS_MODEL` from `glm-5-turbo` to `glm-5` |
| `.claude/agents/ant/aether-*.md` (22 files) | MODIFY | Change `model: inherit` to `model: opus` or `model: sonnet` per caste **OR** leave as-is and use Task tool param |
| `.aether/model-profiles.yaml` | MODIFY | Add model_slot field to caste entries |
| `.aether/docs/command-playbooks/build-wave.md` | MODIFY | Add model resolution + pass `model=` param in Task calls |
| `.aether/docs/command-playbooks/build-full.md` | MODIFY | Same model resolution in full build path |
| `.claude/commands/ant/swarm.md` | MODIFY | Add model param to Scout spawns |
| `.claude/commands/ant/colonize.md` | MODIFY | Add model param to Surveyor spawns |
| `.claude/commands/ant/seal.md` | MODIFY | Add model param to Sage/Chronicler spawns |
| `.claude/commands/ant/patrol.md` | MODIFY | Add model param to Watcher spawn |
| `.claude/commands/ant/organize.md` | MODIFY | Add model param to Keeper spawn |
| `bin/lib/model-profiles.js` | MODIFY | Add `getModelSlot()` function for slot resolution |
| `.aether/aether-utils.sh` | MODIFY | Add `model-slot get <caste>` subcommand |
| `.aether/utils/spawn-with-model.sh` | NO CHANGE | Legacy, keep for reference |
| `bin/cli.js` | MODIFY | Update `caste-models list` to show slot mapping |

### Recommended Caste-to-Slot Mapping

| Caste | Model Slot | Actual Model (via proxy) | Rationale |
|-------|-----------|-------------------------|-----------|
| queen (prime) | opus | glm-5 | Strategic coordination, deep reasoning |
| archaeologist | sonnet | glm-5-turbo | Git history parsing is straightforward |
| architect | opus | glm-5 | System design requires deep reasoning |
| oracle | opus | glm-5 | Deep research benefits from full reasoning |
| route_setter | opus | glm-5 | Task decomposition is complex planning |
| builder | sonnet | glm-5-turbo | Fast, deterministic implementation |
| watcher | sonnet | glm-5-turbo | Verification is systematic, not creative |
| scout | sonnet | glm-5-turbo | Research is execution-focused |
| chaos | sonnet | glm-5-turbo | Edge case testing is methodical |
| colonizer | sonnet | glm-5-turbo | Environment setup is straightforward |
| ambassador | opus | glm-5 | Integration design requires reasoning |
| keeper | sonnet | glm-5-turbo | Documentation is systematic |
| tracker | sonnet | glm-5-turbo | Bug investigation is methodical |
| sage | opus | glm-5 | Wisdom synthesis requires reasoning |
| chaos | sonnet | glm-5-turbo | Resilience testing is methodical |
| weaver | sonnet | glm-5-turbo | Refactoring follows patterns |
| gatekeeper | sonnet | glm-5-turbo | Security scanning is systematic |
| includer | sonnet | glm-5-turbo | Accessibility checks are methodical |
| measurer | sonnet | glm-5-turbo | Performance analysis follows patterns |
| chronicler | sonnet | glm-5-turbo | Documentation generation is straightforward |
| probe | sonnet | glm-5-turbo | Coverage analysis is systematic |
| auditor | sonnet | glm-5-turbo | Quality analysis is methodical |
| surveyor-* | sonnet | glm-5-turbo | Codebase analysis is systematic |

**Summary:** 6 castes use `opus` (glm-5), 16 castes use `sonnet` (glm-5-turbo).

### Settings.json Configuration Change

```json
{
  "env": {
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5",         // Was glm-5-turbo
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "glm-5-turbo", // Unchanged
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "glm-4.5-air"   // Unchanged
  }
}
```

This single change in `~/.claude/settings.json` is the **minimum viable configuration**. Combined with changing agent frontmatter from `model: inherit` to `model: opus` or `model: sonnet`, this routes all workers through the proxy to the correct model.

---

## Two Approaches Compared

### Approach A: Agent Frontmatter (Static)

Change `model: inherit` to `model: opus` or `model: sonnet` in each agent's frontmatter.

**Pros:**
- Zero changes to build playbooks (build-wave.md, swarm.md, etc.)
- Automatic -- every spawn uses the correct model without playbook logic
- Simple, declarative

**Cons:**
- All 22 agent files need updating (including mirrors in `.aether/agents-claude/` and `.opencode/agents/`)
- Cannot be overridden per-build without also changing agent definitions
- Cannot do task-based routing (different tasks for same caste using different models)
- Sync burden: `.claude/agents/ant/*.md` must match `.aether/agents-claude/*.md` (packaging mirror)

### Approach B: Task Tool Parameter (Dynamic) -- RECOMMENDED

Pass `model="{slot}"` in each Task tool call, resolved from model-profiles.yaml at build time.

**Pros:**
- Agent definitions stay at `model: inherit` (no sync burden)
- Can be overridden per-build via `--model` flag (already partially implemented)
- Can do task-based routing (different tasks for same caste using different models)
- model-profiles.yaml becomes the single source of truth
- GSD proves this pattern works

**Cons:**
- Must update every Task tool call in every command that spawns workers (7 commands, ~20 spawn points)
- Build playbooks need a model resolution step before spawning
- More complex -- dynamic resolution at runtime

### Recommendation: Approach B

The dynamic approach is worth the complexity because:
1. It keeps the single-source-of-truth in model-profiles.yaml
2. The `--model` flag already exists in build.md (Step 1) -- the infrastructure for CLI override is partially built
3. The GSD system proves the `model` parameter works in Task tool calls
4. It enables future task-based routing (the `task_routing` section of model-profiles.yaml is already defined)

---

## Data Flow (End-to-End for Build)

### Target Flow

```
1. /ant:build 1
       |
       v
2. build-prep.md Step 1: Parse $ARGUMENTS
   - Extract phase number
   - Extract --model flag (if present) -> cli_model_override
       |
       v
3. build-prep.md Step 1 (new): Resolve model slots
   For each caste that will be spawned:
     bash .aether/aether-utils.sh model-slot get builder
     -> {"ok":true,"result":{"slot":"sonnet","model":"glm-5-turbo","source":"profile"}}
   Store in cross-stage state: builder_model_slot="sonnet"
       |
       v
4. build-context.md Step 0.6: Verify proxy health
   curl -s http://localhost:4000/health
       |
       v
5. build-wave.md Step 5.1: Spawn Wave 1 workers
   For each task:
     Task(
       prompt="...",
       subagent_type="aether-builder",
       model="{builder_model_slot}",          // <-- NEW: resolves to "sonnet"
       description="..."
     )
       |
       v
6. Claude Code resolves "sonnet" slot:
   Reads ANTHROPIC_DEFAULT_SONNET_MODEL from settings.json env
   Value: "glm-5-turbo"
       |
       v
7. Claude Code sends to LiteLLM proxy:
   POST http://localhost:4000/v1/messages
   model: "glm-5-turbo"
       |
       v
8. Builder runs on glm-5-turbo (fast, deterministic)

   Meanwhile, if Oracle was spawned:
   model="opus" -> ANTHROPIC_DEFAULT_OPUS_MODEL -> "glm-5"
   Oracle runs on glm-5 (deep reasoning)
```

### New Subcommand: model-slot

```bash
# Get the model slot for a caste (returns Claude Code slot name)
bash .aether/aether-utils.sh model-slot get builder
# -> {"ok":true,"result":{"slot":"sonnet","model":"glm-5-turbo","source":"profile"}}

# Get all caste-to-slot mappings
bash .aether/aether-utils.sh model-slot list
# -> {"ok":true,"result":{"prime":"opus","builder":"sonnet",...}}

# Validate a slot name
bash .aether/aether-utils.sh model-slot validate sonnet
# -> {"ok":true,"result":{"valid":true,"valid_slots":["inherit","sonnet","opus","haiku"]}}
```

The `model-slot` subcommand reads `model-profiles.yaml`, looks up the caste, and returns the corresponding Claude Code model slot name. This is a thin wrapper around the existing `model-profile get` subcommand that adds a mapping layer from model names to slot names.

---

## Spawn Points Requiring Changes

### Build System (Priority 1 -- most spawns)

| File | Spawn Points | Caste(s) | Current Model Param |
|------|-------------|----------|-------------------|
| `build-wave.md` Step 5.1 | Wave 1 builders | builder | None |
| `build-wave.md` Step 5.3 | Wave 2+ builders | builder | None |
| `build-wave.md` Step 5.4 | Watcher | watcher | None |
| `build-wave.md` Step 5.5.1 | Measurer | measurer | None |
| `build-wave.md` Step 5.6 | Chaos | chaos | None |
| `build-wave.md` Step 4.1 | Archaeologist | archaeologist | None |
| `build-wave.md` Step 5.1.1 | Ambassador | ambassador | None |
| `build-full.md` (duplicate) | All above | Same | None |

### Other Commands (Priority 2)

| File | Spawn Points | Caste(s) |
|------|-------------|----------|
| `swarm.md` | 4 scouts | scout |
| `colonize.md` | 4 surveyors | surveyor-nest, surveyor-disciplines, surveyor-pathogens, surveyor-provisions |
| `seal.md` | sage + chronicler | sage, chronicler |
| `patrol.md` | watcher | watcher |
| `organize.md` | keeper | keeper |

### Total: ~20 spawn points across 7 command files

---

## Backward Compatibility

### Claude-Only Workflow (No Proxy)

When running Claude Code directly (no LiteLLM proxy):
- `opus` -> Claude Opus (expensive, deep reasoning)
- `sonnet` -> Claude Sonnet (balanced)
- `haiku` -> Claude Haiku (fast, cheap)

This still works correctly. The model slots have sensible defaults in Claude Code. The routing layer (LiteLLM proxy) is transparent -- if the proxy isn't running, Claude Code uses its native models.

### CLI Override Flag

The `--model` flag in `/ant:build` (build-prep.md Step 1) already validates model names. This needs to be updated to also accept slot names (`opus`, `sonnet`, `haiku`) and override all workers to that slot.

### OpenCode Parity

OpenCode agent definitions in `.opencode/agents/` need the same `model` field treatment. The spawn mechanism may differ (OpenCode has its own agent spawning), so this needs per-platform investigation.

---

## Build Order for Implementation

### Phase 1: Model Slot Resolution Infrastructure
1. Add `model-slot` subcommand to aether-utils.sh (reads model-profiles.yaml, returns slot name)
2. Add `getModelSlot()` to bin/lib/model-profiles.js
3. Add `model_slot` field to model-profiles.yaml caste entries
4. Add `aether model-slots` CLI command
5. Tests for all new functions

### Phase 2: Settings.json Configuration
1. Update `~/.claude/settings.json` `ANTHROPIC_DEFAULT_OPUS_MODEL` to `glm-5`
2. Verify proxy routes `glm-5` correctly
3. Test that `model="opus"` in Task tool actually reaches GLM-5

### Phase 3: Build System Integration (highest impact)
1. Add model resolution step to build-prep.md
2. Update build-wave.md to pass `model=` in all Task tool calls
3. Update build-full.md (mirror of build-wave.md)
4. Test end-to-end: `/ant:build 1` with different castes using different models
5. Verify with `/ant:verify-castes` (update this command)

### Phase 4: Other Commands
1. Update swarm.md Scout spawns
2. Update colonize.md Surveyor spawns
3. Update seal.md Sage/Chronicler spawns
4. Update patrol.md, organize.md

### Phase 5: Polish
1. Update `/ant:verify-castes` to show actual model routing (remove "archived" message)
2. Update CLAUDE.md documentation
3. Add `--model` flag support for opus/sonnet/haiku slot names
4. OpenCode parity (if applicable)

---

## Pitfalls

### Pitfall 1: Agent Frontmatter vs Task Tool Param Conflict
If both the agent definition says `model: inherit` and the Task call says `model="opus"`, which wins?
- **Assumption:** The Task tool `model` parameter overrides the agent frontmatter. This needs verification.
- **Mitigation:** If Task param doesn't override, fall back to Approach A (changing agent frontmatter).

### Pitfall 2: LiteLLM Proxy Model Name Mismatch
The proxy must recognize the model name sent by Claude Code. If `ANTHROPIC_DEFAULT_OPUS_MODEL` is set to `glm-5`, the proxy must have `glm-5` configured as a model alias.
- **Verification:** Check LiteLLM proxy config (litellm_config.yaml) to confirm model aliases.
- **Mitigation:** The proxy is already running with these models (settings.json lines 70-74 list them).

### Pitfall 3: Build-Full.md vs Build-Wave.md Divergence
There are TWO build paths: the split-playbook path (build.md -> build-wave.md) and the full path (build-full.md). Both contain spawn logic. Changes must be applied to BOTH.
- **Mitigation:** build-full.md is a consolidated copy. Apply the same model param changes to both.

### Pitfall 4: Continue Commands Don't Spawn Workers
The continue playbooks (continue-verify.md, continue-gates.md, etc.) do NOT spawn workers -- they only verify and advance state. No changes needed there. But verify-castes.md should be updated to reflect working routing.

### Pitfall 5: model-profiles.yaml YAML Parsing Fragility
The shell `awk`-based YAML parser in aether-utils.sh (line 2628) is fragile. It parses indentation-based YAML with awk. If the YAML structure changes, parsing breaks silently.
- **Mitigation:** The `select` and `validate` subcommands use Node.js for reliable parsing. Add a `model-slot` subcommand that also uses Node.js.

---

## Sources

- HIGH confidence: Direct codebase analysis of `.claude/agents/ant/*.md` (22 agent definitions, all with `model: inherit`)
- HIGH confidence: Direct analysis of `~/.claude/settings.json` (environment variable mappings)
- HIGH confidence: GSD system proven pattern in `.claude/get-shit-done/workflows/new-project.md` (line 585)
- HIGH confidence: GSD model resolution in `.claude/get-shit-done/bin/gsd-tools.cjs` (MODEL_PROFILES, cmdResolveModel)
- HIGH confidence: Previous failure documented in `.aether/archive/model-routing/README.md`
- HIGH confidence: model-profiles.yaml structure and bin/lib/model-profiles.js logic
- MEDIUM confidence: Task tool `model` parameter behavior (observed working in GSD, not formally documented by Anthropic)
- MEDIUM confidence: LiteLLM proxy model name resolution (proxy config not examined directly)

---

*Architecture research for: Aether v2.3 Per-Caste Model Routing*
*Researched: 2026-03-27*
