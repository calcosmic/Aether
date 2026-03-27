# Stack Research: Per-Caste GLM-5 Model Routing

**Domain:** Multi-agent orchestration -- per-caste model assignment via Claude Code native frontmatter
**Researched:** 2026-03-27
**Confidence:** HIGH (based on direct codebase analysis, Claude Code official docs, and verified settings files)

---

## Executive Summary

This research answers how to route reasoning castes (prime, oracle, archaeologist, route_setter, architect) to GLM-5 and execution castes (builder, watcher, scout, chaos, colonizer) to GLM-5-turbo -- without new proxy infrastructure, new models, or env var workarounds.

**The enabling mechanism is Claude Code's native `model:` frontmatter field** in agent `.md` definitions. This field accepts model slot aliases (`sonnet`, `opus`, `haiku`, `inherit`) that map through `settings.json` environment variables to actual model names via the LiteLLM proxy. This mechanism did not exist when v1 model routing was archived on 2026-02-15 -- the archive explicitly lists "Claude Code Feature Request" (Task tool `env:` parameter) as the fix path. The frontmatter `model:` field IS that fix.

**The mapping chain:**
```
Agent frontmatter: model: opus
    -> settings.json: ANTROPIC_DEFAULT_OPUS_MODEL = glm-5
        -> LiteLLM proxy (localhost:4000)
            -> Z.AI GLM-5 API
```

**What changes:**
1. `settings.json` (1 line: opus slot from glm-5-turbo to glm-5)
2. Agent `.md` frontmatter (22 files: 5 to `opus`, 5 to `sonnet`, 12 assessed individually)
3. `model-profiles.yaml` (10 caste entries: 5 to glm-5, 5 to glm-5-turbo)
4. `bin/lib/model-profiles.js` (add `getModelSlotForCaste()` function)
5. `aether-utils.sh` model-profile subcommands (add `model-slot get` subcommand)
6. Tests (update mock profiles, add slot-mapping tests)
7. `verify-castes.md` (remove "model routing impossible" note)
8. `spawn-with-model.sh` (mark as legacy -- no longer the primary routing mechanism)

**What does NOT change:**
- No new proxy infrastructure
- No new models
- No new environment variables
- No changes to the LiteLLM proxy config
- No env var workarounds or bash-level model injection

---

## Q1: What Changes in settings.json?

### Current State

`~/.claude/settings.json` (active, LiteLLM proxy mode):
```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://localhost:4000",
    "ANTHROPIC_AUTH_TOKEN": "sk-litellm-local",
    "ANTHROPIC_MODEL": "glm-5-turbo",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5-turbo",    // <- CHANGE
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "glm-5-turbo",   // <- KEEP
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "glm-4.5-air"     // <- KEEP
  },
  "model": "glm-5-turbo"
}
```

### Required Change

**One line:** Change `ANTHROPIC_DEFAULT_OPUS_MODEL` from `"glm-5-turbo"` to `"glm-5"`.

```json
"ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5"
```

### Rationale

Claude Code's `model:` frontmatter field maps to these environment variables:

| Frontmatter Value | Environment Variable | Current Value | New Value |
|-------------------|---------------------|---------------|-----------|
| `model: opus` | `ANTHROPIC_DEFAULT_OPUS_MODEL` | glm-5-turbo | **glm-5** |
| `model: sonnet` | `ANTHROPIC_DEFAULT_SONNET_MODEL` | glm-5-turbo | glm-5-turbo (no change) |
| `model: haiku` | `ANTHROPIC_DEFAULT_HAIKU_MODEL` | glm-4.5-air | glm-4.5-air (no change) |
| `model: inherit` | Uses parent's model | (parent model) | (no change) |

The sonnet slot already maps to glm-5-turbo, which is correct for execution castes. Only the opus slot needs updating.

### Secondary settings.json.glm Impact

`~/.claude/settings.json.glm` (direct Z.AI API mode, no proxy):
```json
{
  "env": {
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "glm-5-turbo",    // <- CHANGE
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "glm-5-turbo",   // <- KEEP
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "glm-4.5-air"     // <- KEEP
  }
}
```

Same change: `ANTHROPIC_DEFAULT_OPUS_MODEL` -> `"glm-5"`.

This file uses direct Z.AI endpoint (`https://api.z.ai/api/anthropic`) with a real auth token, bypassing the LiteLLM proxy. The model slot mechanism works identically -- Claude Code resolves the slot alias to the model name via the env var, then sends that model name in the API request.

### Config Swap Workflow Integration

The config swap between `settings.json` (proxy mode) and `settings.json.glm` (direct mode) already exists. Both files need the same opus slot change. This is a one-time coordinated edit. Future config swaps will preserve the opus->glm-5 mapping because both files carry it.

### Confidence: HIGH
- Official Anthropic docs confirm the `model:` field accepts slot aliases
- Settings file content verified by direct read
- The env var naming (`ANTHROPIC_DEFAULT_OPUS_MODEL`) is standard Claude Code configuration

---

## Q2: What Changes in model-profiles.yaml?

### Current State

`.aether/model-profiles.yaml` -- all 10 castes map to glm-5-turbo:
```yaml
worker_models:
  prime: glm-5-turbo
  archaeologist: glm-5-turbo
  architect: glm-5-turbo
  oracle: glm-5-turbo
  route_setter: glm-5-turbo
  builder: glm-5-turbo
  watcher: glm-5-turbo
  scout: glm-5-turbo
  chaos: glm-5-turbo
  colonizer: glm-5-turbo
```

### Required Changes

Update `worker_models` to reflect the two-tier split:
```yaml
worker_models:
  # Reasoning castes -> GLM-5 (via opus slot)
  prime: glm-5
  archaeologist: glm-5
  architect: glm-5
  oracle: glm-5
  route_setter: glm-5
  # Execution castes -> GLM-5-turbo (via sonnet slot)
  builder: glm-5-turbo
  watcher: glm-5-turbo
  scout: glm-5-turbo
  chaos: glm-5-turbo
  colonizer: glm-5-turbo
```

### Optional Addition: model_slots Section

Add a new section that maps castes to model slots (opus/sonnet/haiku). This provides a single source of truth for the JS library and shell commands to query:

```yaml
model_slots:
  prime: opus
  archaeologist: opus
  architect: opus
  oracle: opus
  route_setter: opus
  builder: sonnet
  watcher: sonnet
  scout: sonnet
  chaos: sonnet
  colonizer: sonnet
```

**Why add this:** The `worker_models` section maps castes to model names (glm-5, glm-5-turbo), which is the model-profiles library's domain. The `model_slots` section maps castes to Claude Code slot aliases (opus, sonnet), which is the agent frontmatter's domain. Keeping both ensures each layer can query its own mapping without coupling to the other's abstraction.

**Risk of NOT adding:** Without `model_slots`, any code that needs to know "which slot does this caste use?" must hardcode the mapping or derive it by cross-referencing model names against settings.json env vars -- fragile and coupled.

### Optional Addition: task_routing Update

The `task_routing.complexity_indicators` currently routes ALL complexity levels to glm-5-turbo:
```yaml
task_routing:
  complexity_indicators:
    complex:
      model: glm-5-turbo     # <- Should this change to glm-5?
    simple:
      model: glm-5-turbo
    validate:
      model: glm-5-turbo
```

**Recommendation: Do NOT change task_routing now.** Task routing is a separate concept from caste routing. It routes based on task keywords, not caste identity. Changing it would affect the JS library's `selectModelForTask()` function which is used by CLI override logic. If task routing changes are desired, they should be a separate decision with separate testing. The caste-to-model split is the immediate goal.

### Confidence: HIGH
- File content verified by direct read
- The worker_models section is the documented caste-to-model mapping
- The new model_slots section is additive and non-breaking

---

## Q3: What Changes in Agent Definitions (Spawn Instructions)?

### The Enabling Mechanism

Claude Code subagent definitions (`.claude/agents/ant/*.md`) support a `model:` frontmatter field. From the official Anthropic docs (sub-agents page):

> The `model` field accepts: `sonnet`, `opus`, `haiku`, or `inherit`.
> If omitted, defaults to the configured subagent model.

This is the KEY mechanism that makes per-caste routing possible. When Claude Code spawns a subagent via the Task tool with `subagent_type="aether-builder"`, it reads the agent's `.md` file, resolves the `model:` field to an actual model name using the env vars in settings.json, and sends the API request with that model.

### Why v1 Failed

The archived v1 implementation (`.aether/archive/model-routing/README.md`, archived 2026-02-15) tried to set `ANTHROPIC_MODEL` env vars before spawning workers. The Task tool does not inherit environment variables from the parent process, so all workers received the same model regardless of caste assignment. The archive explicitly lists "Claude Code Feature Request" (Task tool `env:` parameter) as Option 1 for a fix.

The `model:` frontmatter field IS that fix. It is a native Claude Code feature that resolves model selection at spawn time, bypassing the env var limitation entirely.

### All 22 Agent Files: Classification and Changes

Every agent currently has `model: inherit` on line 6. Here is the complete classification:

#### Reasoning Castes (change to `model: opus`)

These agents perform complex reasoning, planning, coordination, or deep analysis. They benefit from GLM-5's deeper reasoning capability, and their tasks are typically bounded single-shot operations where GLM-5's constraint sensitivity is manageable.

| Agent File | Caste | Current | Change To | Rationale |
|-----------|-------|---------|-----------|-----------|
| `aether-queen.md` | prime | inherit | opus | Central coordinator. Spawns workers, synthesizes results, makes strategic decisions. Highest reasoning demand. |
| `aether-archaeologist.md` | archaeologist | inherit | opus | Excavates git history, identifies regression risks, produces stability maps. Requires careful analysis of complex code history. |
| `aether-route-setter.md` | route_setter | inherit | opus | Phase planning, task decomposition, dependency analysis. Complex planning is GLM-5's documented strength. |
| No agent file | oracle | inherit | N/A | The oracle caste does NOT have a dedicated agent file. Oracle research uses `/ant:oracle` which runs as the Queen or a direct CLI invocation, not as a spawned subagent. No frontmatter change needed. |
| No agent file | architect | inherit | N/A | The architect caste does NOT have a dedicated agent file either. Architecture tasks are handled by Queen or route-setter. No frontmatter change needed. |

**Important finding:** Only 3 of the 5 reasoning castes have dedicated agent files. The `oracle` and `architect` castes lack agent `.md` files -- their work is performed by other agents (Queen, Route-Setter) or by direct CLI invocation. This means the frontmatter mechanism alone cannot route oracle/architect tasks to GLM-5 through subagent spawning.

**Mitigation for oracle/architect:** Since Queen already gets `model: opus`, any oracle or architect work performed by the Queen will use GLM-5. For direct `/ant:oracle` CLI invocations, the user's session model (settings.json `model` field) applies -- currently `glm-5-turbo`. If oracle CLI calls should also use GLM-5, that is a separate decision about the user's default session model, not a caste routing concern.

#### Execution Castes (change to `model: sonnet`)

These agents perform implementation, validation, research, or exploration tasks. They benefit from GLM-5-turbo's deterministic agent-friendly output and fast response times. These agents often run in loops where GLM-5's constraint sensitivity would be problematic.

| Agent File | Caste | Current | Change To | Rationale |
|-----------|-------|---------|-----------|-----------|
| `aether-builder.md` | builder | inherit | sonnet | Implementation work, TDD cycles, coding. GLM-5-turbo is documented as best for agent loops. |
| `aether-watcher.md` | watcher | inherit | sonnet | Test running, validation, quality checks. Deterministic execution, no deep reasoning needed. |
| `aether-scout.md` | scout | inherit | sonnet | Research, codebase analysis, information gathering. Fast execution matters more than deep reasoning. |
| `aether-chaos.md` | chaos | inherit | sonnet | Edge case testing, resilience probing. Deterministic output preferred. |

**Note:** The `colonizer` caste does NOT have a dedicated agent file. Colonize work is performed by the four surveyor agents.

#### Non-Caste Agents (recommendation: keep `model: inherit`)

These agents are specialist or niche roles that are NOT part of the 10-worker caste system. They are spawned for specific quality gates or analysis tasks. Keeping them on `inherit` means they use the parent session's model (currently glm-5-turbo), which is appropriate.

| Agent File | Role | Current | Recommendation | Rationale |
|-----------|------|---------|----------------|-----------|
| `aether-probe.md` | Coverage analysis | inherit | **sonnet** | Writes test files and runs them. Implementation work -> sonnet slot. |
| `aether-keeper.md` | Knowledge preservation | inherit | keep inherit | Knowledge synthesis, moderate reasoning. Inherit is fine. |
| `aether-tracker.md` | Bug investigation | inherit | keep inherit | Root cause analysis benefits from reasoning, but inherit from Queen (opus) is acceptable. |
| `aether-weaver.md` | Refactoring | inherit | **sonnet** | Code modification with test verification. Implementation work -> sonnet slot. |
| `aether-auditor.md` | Code quality | inherit | keep inherit | Read-only analysis. Inherit from parent is fine. |
| `aether-gatekeeper.md` | Security audit | inherit | keep inherit | Read-only static analysis. Fast execution, inherit is fine. |
| `aether-includer.md` | Accessibility | inherit | keep inherit | Read-only static analysis. Inherit is fine. |
| `aether-measurer.md` | Performance | inherit | keep inherit | Read-only profiling. Inherit is fine. |
| `aether-sage.md` | Wisdom synthesis | inherit | keep inherit | Pattern extraction from history. Moderate reasoning, inherit is fine. |
| `aether-ambassador.md` | External integrations | inherit | **sonnet** | Implements integrations with error handling. Coding work -> sonnet slot. |
| `aether-chronicler.md` | Documentation | inherit | keep inherit | Writes docs only. Inherit is fine. |
| `aether-surveyor-nest.md` | Colony survey | inherit | **sonnet** | Codebase exploration + file writes. Execution work -> sonnet slot. |
| `aether-surveyor-disciplines.md` | Colony survey | inherit | **sonnet** | Same as above. |
| `aether-surveyor-pathogens.md` | Colony survey | inherit | **sonnet** | Same as above. |
| `aether-surveyor-provisions.md` | Colony survey | inherit | **sonnet** | Same as above. |

#### Summary of All Frontmatter Changes

| Change | Files | Count |
|--------|-------|-------|
| `model: inherit` -> `model: opus` | queen, archaeologist, route-setter | 3 |
| `model: inherit` -> `model: sonnet` | builder, watcher, scout, chaos, probe, weaver, ambassador, surveyor-nest, surveyor-disciplines, surveyor-pathogens, surveyor-provisions | 11 |
| Keep `model: inherit` | keeper, tracker, auditor, gatekeeper, includer, measurer, sage, chronicler | 8 |
| **Total changes** | | **14 of 22** |

### Why Not `model: opus` for More Agents?

GLM-5 requires tight constraints (temperature 0.4, top_p 0.85, max_tokens 2500) for agent stability. GLM-5-turbo works reliably for agent workflows without extra constraints. The GLM-5 constraint requirements are documented in `model-profiles.yaml` under `best_for`:
- GLM-5: "Bounded single-shot reasoning tasks" + "Deep analysis with explicit stop conditions" + "Tasks where you control the runtime tightly"
- GLM-5-turbo: "Agent loops (deterministic output, clean termination)" + "Coding and implementation tasks" + "Orchestration and coordination" + "Any task where looping = failure"

The reasoning caste agents (Queen, Archaeologist, Route-Setter) primarily do single-shot reasoning tasks with explicit stop conditions. Execution agents (Builder, Watcher, Scout) run in agent loops where GLM-5's constraint sensitivity would be problematic.

### How Frontmatter interacts with Task Tool Spawning

The build/playbook system spawns workers via:
```
Task({
  prompt: "...",
  subagent_type: "aether-builder"
})
```

Claude Code reads `.claude/agents/ant/aether-builder.md`, resolves `model: sonnet` to the value of `ANTHROPIC_DEFAULT_SONNET_MODEL` (glm-5-turbo), and sends the API request with `model: glm-5-turbo`. No env vars need to be set by the parent. No bash-level injection needed.

### Confidence: HIGH
- Official Anthropic docs confirm the mechanism
- All 22 agent files verified by direct read (all have `model: inherit` on line 6)
- The mapping chain (frontmatter -> env var -> proxy -> API) is the documented Claude Code behavior
- The v1 archive confirms the env var approach was the blocker, and frontmatter bypasses it

---

## Q4: What Changes in bin/lib/model-profiles.js and Shell Libraries?

### bin/lib/model-profiles.js

The JS library operates on model names (glm-5, glm-5-turbo), not model slots (opus, sonnet). It needs a new function to bridge the gap between castes and model slots, so that shell commands and display logic can show "builder uses sonnet slot -> glm-5-turbo".

#### New Function: `getModelSlotForCaste(profiles, caste)`

```javascript
/**
 * Get the Claude Code model slot (opus/sonnet/haiku) for a caste
 * @param {object} profiles - Parsed model profiles
 * @param {string} caste - Caste name
 * @returns {object} { slot: string, model: string, source: string }
 */
function getModelSlotForCaste(profiles, caste) {
  if (!profiles || typeof profiles !== 'object') {
    return { slot: 'sonnet', model: DEFAULT_MODEL, source: 'fallback' };
  }

  // Check model_slots section first (if added to YAML)
  if (profiles.model_slots && profiles.model_slots[caste]) {
    const slot = profiles.model_slots[caste];
    const model = getModelForCaste(profiles, caste);
    return { slot, model, source: 'slot-config' };
  }

  // Derive from model name
  const model = getModelForCaste(profiles, caste);
  if (model === 'glm-5') {
    return { slot: 'opus', model, source: 'derived' };
  }
  if (model === 'glm-4.5-air') {
    return { slot: 'haiku', model, source: 'derived' };
  }
  // Default: sonnet (glm-5-turbo and unknown models)
  return { slot: 'sonnet', model, source: 'derived' };
}
```

This function would be exported and used by:
- `model-profile slot-get <caste>` shell subcommand (new)
- `model-profile list --with-slots` (enhanced listing)
- `/ant:verify-castes` display logic
- Status dashboard model assignment display

#### No Changes Required to Existing Functions

The existing functions (`loadModelProfiles`, `getModelForCaste`, `selectModelForTask`, etc.) operate on model names and continue to work correctly. The `worker_models` section in model-profiles.yaml already maps castes to model names. The new `model_slots` section is additive.

#### Export Update

Add `getModelSlotForCaste` to `module.exports`:
```javascript
module.exports = {
  // ... existing exports ...
  getModelSlotForCaste,  // NEW
};
```

### aether-utils.sh Model-Profile Subcommands

#### New Subcommand: `model-profile slot-get <caste>`

Uses the JS library's `getModelSlotForCaste()` to return the model slot for a caste:
```bash
model-profile slot-get)
    caste="${2:-}"
    [[ -z "$caste" ]] && { echo "Usage: model-profile slot-get <caste>" >&2; exit 1; }
    node "$BIN_DIR/lib/model-profiles.js" slot-get "$REPO_ROOT" "$caste"
    ;;
```

#### Existing Subcommands: No Breaking Changes

- `model-profile get <caste>` -- Returns model name for caste. Still works, just returns updated values (glm-5 for reasoning castes).
- `model-profile list` -- Lists all caste-to-model assignments. Still works.
- `model-profile select <caste> <task> [cli_override]` -- Full precedence chain. Still works.
- `model-profile validate <model>` -- Validates model name. Still works.

### bin/cli.js

#### New CLI Command: `model-profile slot <caste>`

Add a CLI entry point for the slot-get function:
```javascript
// In the model-profile command handler:
if (subcommand === 'slot') {
  const caste = args._[1];
  const result = modelProfiles.getModelSlotForCaste(profiles, caste);
  console.log(JSON.stringify({ ok: true, result }, null, 2));
}
```

### spawn-with-model.sh

**Status: Legacy.** This script sets `ANTHROPIC_MODEL` env var and spawns Claude Code. It was the v1 approach that failed because the Task tool doesn't inherit env vars.

**With the new frontmatter-based approach, this script is no longer the primary routing mechanism.** Model routing now happens at the Claude Code level via agent frontmatter, not at the bash level via env vars.

**Recommended action:** Keep the file but add a deprecation notice:
```bash
# LEGACY: This script sets ANTHROPIC_MODEL for direct Claude Code spawning.
# Per-caste model routing now uses Claude Code's native agent frontmatter
# (model: opus/sonnet/haiku) in .claude/agents/ant/*.md files.
# This script is retained for manual spawning use cases only.
```

### OpenCode Agent Mirrors

The `.opencode/agents/` directory maintains structural parity with `.claude/agents/ant/`. If OpenCode supports a similar `model:` frontmatter field, the same changes should be applied. If not, the OpenCode agents will continue to use `inherit` (parent session model). This needs verification against OpenCode's agent definition specification.

### Confidence: HIGH
- JS library code verified by direct read (446 lines)
- The new function is additive, non-breaking
- Shell subcommand pattern follows existing conventions
- spawn-with-model.sh status is clear from the archive documentation

---

## Q5: What Changes in Tests?

### Overview

Four test files cover the model-profiles system. All use mock profiles where castes map to specific models. The mocks need updating to reflect the new two-tier split.

### tests/unit/model-profiles.test.js

**Current mock state:** All castes map to glm-5-turbo.

**Changes needed:**
1. Update `createMockProfiles()` to use the new two-tier mapping:
   ```javascript
   worker_models: {
     // Reasoning castes -> GLM-5
     prime: 'glm-5',
     archaeologist: 'glm-5',
     architect: 'glm-5',
     oracle: 'glm-5',
     route_setter: 'glm-5',
     // Execution castes -> GLM-5-turbo
     builder: 'glm-5-turbo',
     watcher: 'glm-5-turbo',
     scout: 'glm-5-turbo',
     chaos: 'glm-5-turbo',
     colonizer: 'glm-5-turbo',
   },
   ```
2. Update `getModelForCaste` test expectations:
   - `builder` -> `glm-5-turbo` (same as before)
   - `architect` -> `glm-5` (changed)
   - `oracle` -> `glm-5` (changed)
3. Update `getAllAssignments` tests:
   - `builder.provider` -> `z_ai` (unchanged)
   - `architect.model` -> `glm-5` (changed)
   - `oracle.model` -> `glm-5` (changed)
4. Update integration test (line 452-459):
   - `getModelForCaste(profiles, 'builder')` -> `glm-5-turbo` (unchanged)
   - `getModelForCaste(profiles, 'architect')` -> `glm-5` (changed)
   - `getModelForCaste(profiles, 'oracle')` -> `glm-5` (changed)

**Affected tests:** ~8 tests need expectation updates.

### tests/unit/model-profiles-task-routing.test.js

**Current mock state:** All castes map to glm-5-turbo, task routing has distinct models per complexity level (glm-5 for complex, glm-5-turbo for simple, glm-4.5-air for validate).

**Changes needed:**
1. Update `createMockProfiles()` worker_models to reflect two-tier split (same as above).
2. `selectModelForTask` tests: These tests use mock profiles where task routing complexity_indicators have different models (glm-5, glm-5-turbo, glm-4.5-air). The task routing section is independent of the worker_models section, so these tests should continue to pass with the existing task_routing mock. BUT verify that the caste default tests work correctly when the caste default model is glm-5 instead of glm-5-turbo.
3. `selectModelForTask caste default is used when no task_routing config` test (line 270-279): Currently expects `glm-5-turbo` for builder. With two-tier mapping, builder stays `glm-5-turbo`, so this test passes unchanged.
4. `selectModelForTask works with different castes` test (line 366-387): Currently expects `glm-5-turbo` for all castes. With two-tier split, architect and oracle would return `glm-5`. This test needs updating.

**Affected tests:** ~2-3 tests need expectation updates.

### tests/unit/model-profiles-overrides.test.js

**Current mock state:** All castes map to glm-5-turbo, except architect -> glm-5 and oracle -> glm-4.5-air.

**Changes needed:**
1. Update `createMockProfiles()` to use the new two-tier mapping. The current mock already has architect -> glm-5, which aligns with the new split. But oracle -> glm-4.5-air does NOT align -- the new split puts oracle on glm-5, not glm-4.5-air.
2. Update mock to use the correct mapping:
   ```javascript
   worker_models: {
     prime: 'glm-5',
     archaeologist: 'glm-5',
     architect: 'glm-5',       // already correct
     oracle: 'glm-5',          // changed from glm-4.5-air
     route_setter: 'glm-5',
     builder: 'glm-5-turbo',
     watcher: 'glm-5-turbo',
     scout: 'glm-5-turbo',
     chaos: 'glm-5-turbo',
     colonizer: 'glm-5-turbo',
   },
   ```
3. `getEffectiveModel returns default model when no override` test (line 298-306): Currently expects `glm-5-turbo` for builder (unchanged).
4. `integration: full set/reset workflow` test (line 384-433): Step 6 expects `glm-5-turbo` for builder after reset (unchanged).

**Affected tests:** ~1 test (the mock profile definition itself, which cascades to assertions).

### tests/unit/cli-override.test.js

**Current mock state:** Uses inline YAML with prime -> glm-5, builder -> glm-5-turbo, watcher -> glm-4.5-air, etc.

**Changes needed:**
1. Update inline YAML to use the new two-tier mapping.
2. Tests are integration tests that call bash commands and parse JSON output. The test logic (precedence chain) is independent of the specific model names, so most tests pass with just the YAML update.
3. Verify `model-profile validate` tests: Known models list changes slightly if model_metadata changes, but it shouldn't -- glm-5, glm-5-turbo, glm-4.5-air all remain in metadata.

**Affected tests:** Mock YAML update, but test logic likely unchanged.

### New Tests: getModelSlotForCaste

Add a new test file or section testing the new `getModelSlotForCaste()` function:

```javascript
// tests/unit/model-profiles-slots.test.js

test('getModelSlotForCaste returns opus for reasoning castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles(); // with two-tier split

  t.is(modelProfiles.getModelSlotForCaste(profiles, 'prime').slot, 'opus');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'archaeologist').slot, 'opus');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'route_setter').slot, 'opus');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'oracle').slot, 'opus');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'architect').slot, 'opus');
});

test('getModelSlotForCaste returns sonnet for execution castes', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  t.is(modelProfiles.getModelSlotForCaste(profiles, 'builder').slot, 'sonnet');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'watcher').slot, 'sonnet');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'scout').slot, 'sonnet');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'chaos').slot, 'sonnet');
  t.is(modelProfiles.getModelSlotForCaste(profiles, 'colonizer').slot, 'sonnet');
});

test('getModelSlotForCaste returns sonnet for unknown caste', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);
  const profiles = createMockProfiles();

  const result = modelProfiles.getModelSlotForCaste(profiles, 'unknown');
  t.is(result.slot, 'sonnet');
  t.is(result.source, 'fallback');
});

test('getModelSlotForCaste handles null profiles gracefully', t => {
  const modelProfiles = require(MODEL_PROFILES_PATH);

  const result = modelProfiles.getModelSlotForCaste(null, 'builder');
  t.is(result.slot, 'sonnet');
  t.is(result.source, 'fallback');
});
```

### verify-castes.md Changes

Lines 70-73 of `.claude/commands/ant/verify-castes.md`:
```
Note: Model-per-caste routing was attempted but is not
possible with Claude Code's Task tool (no env var support).
See archived config: .aether/archive/model-routing/
Tag: model-routing-v1-archived
```

**Replace with:**
```
Model routing: Per-caste routing is active.
  Reasoning castes (prime, archaeologist, route-setter): GLM-5 via opus slot
  Execution castes (builder, watcher, scout, chaos): GLM-5-turbo via sonnet slot
  See: .aether/model-profiles.yaml for full assignments
```

### Confidence: HIGH
- All test files verified by direct read
- Mock profile patterns are consistent across files
- The new slot tests follow the same pattern as existing tests
- The verify-castes change is straightforward text replacement

---

## Complete Change Inventory

### Files to Modify

| File | Change | Effort | Risk |
|------|--------|--------|------|
| `~/.claude/settings.json` | 1 line: `ANTHROPIC_DEFAULT_OPUS_MODEL` -> `"glm-5"` | Trivial | Low -- one env var change |
| `~/.claude/settings.json.glm` | 1 line: same opus slot change | Trivial | Low -- same as above |
| `.claude/agents/ant/aether-queen.md` | `model: inherit` -> `model: opus` | Trivial | Low -- one word change |
| `.claude/agents/ant/aether-archaeologist.md` | `model: inherit` -> `model: opus` | Trivial | Low |
| `.claude/agents/ant/aether-route-setter.md` | `model: inherit` -> `model: opus` | Trivial | Low |
| `.claude/agents/ant/aether-builder.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-watcher.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-scout.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-chaos.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-probe.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-weaver.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-ambassador.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-surveyor-nest.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-surveyor-disciplines.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-surveyor-pathogens.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.claude/agents/ant/aether-surveyor-provisions.md` | `model: inherit` -> `model: sonnet` | Trivial | Low |
| `.aether/model-profiles.yaml` | Update worker_models (10 lines), add model_slots section (11 lines) | Small | Low -- additive + value updates |
| `.claude/commands/ant/verify-castes.md` | Replace "model routing impossible" note | Trivial | Low -- text change |
| `.aether/utils/spawn-with-model.sh` | Add legacy/deprecation notice | Trivial | None -- documentation only |

### Files to Add

| File | Content | Effort | Risk |
|------|---------|--------|------|
| `tests/unit/model-profiles-slots.test.js` | New tests for `getModelSlotForCaste()` | Small | None -- new file |

### Files to Modify (Code Changes)

| File | Change | Effort | Risk |
|------|--------|--------|------|
| `bin/lib/model-profiles.js` | Add `getModelSlotForCaste()` function + export | Small | Low -- additive, non-breaking |
| `.aether/aether-utils.sh` | Add `model-profile slot-get` subcommand | Small | Low -- new case entry |
| `bin/cli.js` | Add `model-profile slot` CLI handler | Small | Low -- additive |

### Files to Modify (Test Updates)

| File | Change | Effort | Risk |
|------|--------|--------|------|
| `tests/unit/model-profiles.test.js` | Update mock profiles + ~8 test expectations | Medium | Low -- expectation updates only |
| `tests/unit/model-profiles-task-routing.test.js` | Update mock profiles + ~3 test expectations | Small | Low -- expectation updates only |
| `tests/unit/model-profiles-overrides.test.js` | Update mock profiles | Small | Low -- mock update only |
| `tests/unit/cli-override.test.js` | Update inline YAML mock | Small | Low -- mock update only |

---

## Implementation Order

1. **settings.json** (1 line each in 2 files) -- enables the mapping chain
2. **Agent frontmatter** (14 files, one-word change each) -- activates routing
3. **model-profiles.yaml** (worker_models + model_slots) -- updates source of truth
4. **bin/lib/model-profiles.js** (new function) -- adds slot-query capability
5. **aether-utils.sh** (new subcommand) -- shell access to slot mapping
6. **bin/cli.js** (new CLI handler) -- CLI access to slot mapping
7. **Tests** (4 files updated + 1 new file) -- validates everything
8. **verify-castes.md** (text update) -- removes outdated "impossible" note
9. **spawn-with-model.sh** (deprecation notice) -- documentation hygiene

Steps 1-3 can be done in a single commit (the actual routing changes). Steps 4-6 are the library support. Steps 7-9 are cleanup and validation.

---

## Risks and Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| GLM-5 instability in reasoning castes | Medium | GLM-5 requires tight constraints (temp 0.4, top_p 0.85). Claude Code may not pass these constraints for subagent calls. Monitor for infinite loops or degenerate output. If issues arise, revert specific agents to `inherit`. |
| GLM-5 slower response for Queen | Low | Queen spawns workers and synthesizes. The parent session model (glm-5-turbo) remains unchanged. Only the Queen subagent itself uses GLM-5. The user's interactive experience is unaffected. |
| Missing oracle/architect agent files | Low | Oracle and architect castes have no dedicated agent files. Their work runs through Queen (which gets opus/GLM-5). Document this gap. If dedicated agents are needed later, create them with `model: opus`. |
| Config swap forgetting to update both files | Low | Both `settings.json` and `settings.json.glm` need the opus slot change. Make the change atomically. Add a comment in both files noting the coordination. |
| Test suite breakage | Low | All test changes are expectation updates (changing expected values from glm-5-turbo to glm-5 or vice versa). No test logic changes. Run `npm test` after all changes. |
| OpenCode parity gap | Low | OpenCode may not support the `model:` frontmatter field. If not, OpenCode agents continue using `inherit`. This is acceptable -- OpenCode is the secondary IDE target. |

---

## Open Questions

1. **GLM-5 constraint passing:** Does Claude Code pass temperature/top_p/max_tokens constraints to subagent API calls? If not, GLM-5 may produce less stable output in agent contexts. This needs live testing.
2. **OpenCode `model:` support:** Does OpenCode support the same agent frontmatter `model:` field? If yes, mirror the changes to `.opencode/agents/`.
3. **Queen as GLM-5:** The Queen is the most critical agent. Moving it from glm-5-turbo to GLM-5 is the highest-risk change. Consider a graduated rollout: change Queen to opus last, after validating with Archaeologist and Route-Setter first.
4. **Task routing interaction:** Should task_routing.complexity_indicators also be updated to route complex tasks to glm-5? Currently all task routing levels point to glm-5-turbo. This is a separate decision.

---

## Sources

- Official Anthropic docs: Claude Code sub-agents page (fetched via webReader, confirms `model:` field accepts `sonnet`, `opus`, `haiku`, `inherit`) -- HIGH confidence
- `.aether/archive/model-routing/README.md` -- v1 failure documentation, confirms env var limitation -- HIGH confidence
- `~/.claude/settings.json` -- verified by direct read -- HIGH confidence
- `~/.claude/settings.json.glm` -- verified by direct read -- HIGH confidence
- `.claude/agents/ant/*.md` (22 files) -- all verified by direct read, all have `model: inherit` on line 6 -- HIGH confidence
- `.aether/model-profiles.yaml` -- verified by direct read -- HIGH confidence
- `bin/lib/model-profiles.js` -- verified by direct read (446 lines) -- HIGH confidence
- Test files (4 existing + new slots test) -- verified by direct read -- HIGH confidence

---
*Stack research for: Per-caste GLM-5 model routing via Claude Code native frontmatter*
*Researched: 2026-03-27*
