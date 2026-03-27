# Feature Research: Aether v2.3 Per-Caste Model Routing

**Domain:** Multi-agent AI development orchestration -- per-worker model selection
**Researched:** 2026-03-27
**Confidence:** HIGH (proven reference implementation in GSD, codebase analysis, YAML config already exists)

---

## Context

Aether currently runs all 22 worker agents on the same model -- whatever the parent Claude Code session uses. The `model-profiles.yaml` config and `model-profiles.js` library were built for name-based routing (glm-5, glm-5-turbo) but never integrated into the actual worker spawn path because `workers.md` incorrectly claims the Claude Code Task tool "does not support" per-worker model selection.

This is wrong. GSD's `resolveModelInternal()` proves the Task tool accepts a `model` parameter with values `"inherit"`, `"sonnet"`, or `"haiku"`. GSD uses this to route 11 agent types across 3 profiles (quality/balanced/budget) -- a working production pattern.

The v2.3 milestone needs to: (1) slot-based routing in Aether, (2) reasoning castes on opus slot, execution castes on sonnet slot, (3) dual-mode support (Claude native + GLM via LiteLLM proxy).

---

## Table Stakes

Missing any of these = per-caste routing does not actually work.

| # | Feature | Why Expected | Complexity | Dependencies |
|---|---------|--------------|------------|--------------|
| T1 | **Slot-based caste-to-model mapping** | Users must be able to say "prime uses opus, builder uses sonnet" without caring what model names those slots resolve to. Slots abstract away the Claude vs GLM distinction. | LOW | model-profiles.yaml update |
| T2 | **Worker spawn passes `model` parameter** | The core mechanism. When colony-prime spawns a worker via Task, it must pass the resolved slot as the `model` parameter. Without this, routing config is decorative. | MEDIUM | T1, build-wave.md playbook, continue playbooks |
| T3 | **Fix workers.md incorrect claim** | Lines 57-90 claim per-caste routing "cannot function due to Claude Code Task tool limitations." This blocks all downstream work -- developers will see it and assume the feature is impossible. | LOW | None |
| T4 | **Dual-mode slot resolution** | In Claude mode, opus=claude-opus, sonnet=claude-sonnet. In GLM mode, opus=glm-5, sonnet=glm-5-turbo. The slot names stay the same; only the resolution changes based on environment. | MEDIUM | T1, LiteLLM proxy |
| T5 | **CLI override for per-spawn model** | Users must be able to force a specific slot for a single build wave: `/ant:build 3 --model sonnet`. Already partially exists in build playbooks but validates against model names instead of slots. | LOW | T1 |

### Table Stakes Detail

**T1: Slot-based caste-to-model mapping**

The current `model-profiles.yaml` maps castes to model NAMES:
```yaml
worker_models:
  prime: glm-5-turbo
  builder: glm-5-turbo
```

This must change to map castes to SLOTS:
```yaml
worker_models:
  prime: opus        # reasoning caste
  builder: sonnet    # execution caste
```

And add a slot resolution section:
```yaml
slot_models:
  claude:
    opus: inherit      # uses parent session's opus
    sonnet: sonnet
    haiku: haiku
  glm:
    opus: glm-5
    sonnet: glm-5-turbo
    haiku: glm-4.5-air
```

The key insight from GSD: opus-tier agents should resolve to `"inherit"` (not `"opus"`) to avoid organization policy version conflicts. `"inherit"` causes the worker to use whatever opus version the user's session is configured with, preventing silent fallbacks to Sonnet.

Confidence: HIGH -- GSD has this exact pattern in production.

**T2: Worker spawn passes `model` parameter**

Currently, all 22 Aether agents have `model: inherit` in their frontmatter, and the build playbooks spawn workers without a `model` parameter:
```
Task(prompt="...", subagent_type="aether-builder", description="...")
```

Must become:
```
Task(prompt="...", subagent_type="aether-builder", model="sonnet", description="...")
```

The playbooks that need updating: `build-wave.md`, `build-full.md`, `continue-verify.md`, `build-prep.md`, and any command that spawns workers via Task.

The resolution happens at orchestration time (not at spawn time). Colony-prime reads the profile, resolves each worker's slot, and passes the slot string to the Task call -- exactly like GSD's `resolveModelInternal()`.

Confidence: HIGH -- GSD's `resolveModelInternal()` + `MODEL_PROFILES` table is the proven reference.

**T3: Fix workers.md incorrect claim**

Lines 57-90 of `workers.md` state:
> "Claude Code's Task tool does not support passing environment variables to spawned workers. All workers inherit the parent session's model configuration."

And:
> "A model-per-caste routing system was designed and implemented (archived in .aether/archive/model-routing/) but cannot function due to Claude Code Task tool limitations."

GSD's working implementation disproves both claims. The Task tool accepts a `model` parameter. The archived v1 implementation failed because it used model NAMES (glm-5, minimax-2.5) instead of slots, not because of Task tool limitations.

Confidence: HIGH -- codebase evidence, GSD working implementation.

**T4: Dual-mode slot resolution**

Two operational modes exist:

**Claude mode** (default, no proxy):
- User runs `claude` normally
- opus slot -> `"inherit"` (parent session's opus model)
- sonnet slot -> `"sonnet"` (Claude Sonnet)
- haiku slot -> `"haiku"` (Claude Haiku)
- No proxy, no environment variable changes

**GLM mode** (via LiteLLM proxy):
- User sets `ANTHROPIC_BASE_URL=http://localhost:4000`, `ANTHROPIC_AUTH_TOKEN=sk-litellm-local`
- opus slot -> `"inherit"` (resolves to GLM-5 via proxy model mapping)
- sonnet slot -> `"sonnet"` (resolves to GLM-5-turbo via proxy model mapping)
- haiku slot -> `"haiku"` (resolves to GLM-4.5-air via proxy model mapping)
- LiteLLM proxy config maps slot names to actual GLM models

The beauty of slot-based routing: Aether never needs to know whether the user is in Claude or GLM mode. It passes slot names to the Task tool, and the proxy (or Claude native) handles resolution. The user's environment determines the mode.

Confidence: HIGH -- GSD uses identical slot values; LiteLLM proxy already configured at localhost:4000.

**T5: CLI override for per-spawn model**

Existing build playbooks already have `--model` flag parsing. But they validate against model names. Change to validate against slot names (`opus`, `sonnet`, `haiku`).

Precedence chain (adapted from GSD):
1. CLI override (`--model opus`) -- highest
2. User override (`model-profiles.yaml` `user_overrides` section)
3. Caste default (from `worker_models` table)
4. Fallback: `sonnet` -- lowest

Confidence: HIGH -- GSD `resolveModelInternal()` implements this exact chain.

---

## Differentiators

These make Aether stand out compared to running Claude Code manually.

| # | Feature | Value Proposition | Complexity | Dependencies |
|---|---------|-------------------|------------|--------------|
| D1 | **Task complexity-based auto-routing** | Instead of fixed caste assignments, analyze the task description and route to opus if it contains complexity indicators (design, architecture, strategize) or sonnet if it contains execution indicators (implement, code, refactor). | MEDIUM | T1, task_routing already exists in model-profiles.yaml |
| D2 | **Per-profile caste assignments** | Like GSD's quality/balanced/budget profiles, Aether could offer profiles: `deep` (all castes on opus), `default` (reasoning=opus, execution=sonnet), `fast` (all castes on sonnet). Users pick based on budget/quality needs. | LOW | T1 |
| D3 | **Runtime profile switching** | `/ant:set-profile fast` to switch all workers to sonnet for quick iterations, `/ant:set-profile deep` for critical architecture work. No restart needed. | LOW | D2 |
| D4 | **Model usage tracking and reporting** | Track how many Task calls used opus vs sonnet vs haiku per phase. Display in `/ant:status` and phase summaries. Users can see their token spend profile. | MEDIUM | T2 |

### Differentiators Detail

**D1: Task complexity-based auto-routing**

The `model-profiles.yaml` already has a `task_routing` section with `complexity_indicators`:
```yaml
task_routing:
  complexity_indicators:
    complex:
      keywords: [design, architecture, plan, coordinate, synthesize, strategize, optimize]
      model: glm-5-turbo    # currently -- should be a slot
    simple:
      keywords: [implement, code, refactor, write, create]
      model: glm-5-turbo
```

This currently maps to model names and all point to the same model. With v2.3, it should map to slots and differentiate:
```yaml
task_routing:
  complexity_indicators:
    complex:
      keywords: [design, architecture, plan, coordinate, synthesize, strategize, optimize]
      slot: opus
    simple:
      keywords: [implement, code, refactor, write, create]
      slot: sonnet
```

Task-based routing takes precedence over caste defaults when a match is found, but is overridden by CLI override and user overrides. This matches GSD's approach where task context can shift routing.

Complexity: MEDIUM because it requires the build playbooks to (1) read the task description, (2) check for keyword matches, (3) resolve the slot. The matching logic exists in `model-profiles.js` (`getModelForTask()`), so it is a wiring problem, not a new implementation.

Confidence: HIGH -- matching logic already exists, just needs slot adaptation.

**D2: Per-profile caste assignments**

Currently, `model-profiles.yaml` has a single `profile: default`. Extend to support named profiles:

```yaml
profiles:
  deep:
    prime: opus
    architect: opus
    oracle: opus
    builder: opus
    watcher: opus
    # all castes on opus
  default:
    prime: opus        # reasoning castes
    architect: opus
    oracle: opus
    route_setter: opus
    archaeologist: opus
    builder: sonnet    # execution castes
    watcher: sonnet
    scout: sonnet
    chaos: sonnet
    colonizer: sonnet
  fast:
    prime: sonnet
    # all castes on sonnet
```

The user selects their profile in `model-profiles.yaml` or via CLI. The `default` profile matches the v2.3 milestone target: reasoning=opus, execution=sonnet.

Complexity: LOW -- it is a data structure change, not a code change. `getModelForCaste()` already reads from `worker_models`; add a profile selector.

Confidence: HIGH -- GSD implements this with quality/balanced/budget profiles.

**D3: Runtime profile switching**

Command: `/ant:set-profile <profile_name>`

Writes to `model-profiles.yaml` `user_overrides` or `active_profile` field. No restart needed because model resolution happens at each spawn time, not at colony init.

Complexity: LOW -- existing `setModelOverride()` and `resetModelOverride()` in `model-profiles.js` already handle YAML write operations.

Confidence: HIGH -- pattern exists in GSD (`/gsd:set-profile`).

**D4: Model usage tracking**

After each build wave, log:
```
Phase 3 model usage:
  opus (inherit): 4 spawns (prime, architect, oracle, route_setter)
  sonnet: 6 spawns (builder x2, watcher x2, scout, chaos)
```

Store in COLONY_STATE.json under a `model_usage` field. Display in `/ant:status`.

Complexity: MEDIUM -- requires instrumentation in build playbooks to count spawn calls by slot, plus a new section in status display.

Confidence: MEDIUM -- no reference implementation exists, but it is straightforward data collection.

---

## Anti-Features

Features to explicitly NOT build.

| # | Anti-Feature | Why Avoid | What to Do Instead |
|---|--------------|-----------|-------------------|
| A1 | **Per-worker environment variable injection** | Claude Code's Task tool does not support per-subagent environment variables. The archived v1 implementation tried this and failed. | Use the `model` parameter with slot values. This is the correct mechanism. |
| A2 | **Model name routing (glm-5, glm-5-turbo) in Aether code** | Aether should not know about specific model names. Model names are user configuration (in settings.json or proxy config). Aether routes by slot; the environment resolves slots to names. | Slot-based routing. `model-profiles.yaml` maps castes to slots. Users configure slot-to-name mapping in their environment. |
| A3 | **Automatic model selection based on cost** | Adding a "pick cheapest model that can handle this" heuristic adds complexity and unpredictability. Users should explicitly choose their profile. | Named profiles (deep/default/fast) that users select once. |
| A4 | **Health-check-gated model routing** | Do not make routing conditional on LiteLLM proxy health. If the proxy is down, the user's environment variables handle it -- Aether does not need to detect or react to proxy status. | Existing proxy health check in build playbooks (Step 0.6) is advisory only, not routing-conditional. |
| A5 | **Caste model assignment in agent frontmatter** | The 22 agent `.md` files currently have `model: inherit`. Do not change these to caste-specific values. Agent definitions should stay generic; routing belongs in the orchestration layer. | Keep agents as `model: inherit`. Routing happens in build playbooks when spawning, not in agent definitions. |

### Anti-Features Detail

**A1: Per-worker environment variable injection**

The archived v1 implementation (`.aether/archive/model-routing/`) tried to set per-worker `ANTHROPIC_MODEL=glm-5` via env vars. This does not work because Claude Code's Task tool does not forward environment variables to spawned subagents.

GSD discovered the correct approach: use the Task tool's `model` parameter, which accepts slot values. This is a first-class API feature, not an env var hack.

Confidence: HIGH -- archived v1 failure + GSD working implementation.

**A2: Model name routing in Aether code**

The current `model-profiles.yaml` and `model-profiles.js` route by model name (glm-5, glm-5-turbo). This creates a hard coupling between Aether and specific GLM models, making Claude-native mode impossible without config changes.

The correct design:
- Aether routes by SLOT (opus, sonnet, haiku)
- Claude Code resolves slots to Claude models (native)
- LiteLLM proxy resolves slots to GLM models (when proxy is configured)
- Aether never knows or cares about the actual model name

The `model-profiles.js` library's `DEFAULT_MODEL = 'glm-5-turbo'` and `validateModel()` function that checks against model names must be refactored to work with slots.

Confidence: HIGH -- GSD uses exclusively slot values; never references model names.

**A5: Caste model assignment in agent frontmatter**

Each of the 22 agent `.md` files has a frontmatter `model: inherit` field. This is correct and should stay. The reason: agent definitions are generic templates. The same builder agent might need to run on opus for a complex task and sonnet for a simple one. Routing is the orchestration layer's responsibility, not the agent's.

If agent frontmatter set `model: opus`, then every builder spawn would use opus regardless of profile or task complexity. That removes all flexibility.

Confidence: HIGH -- matches GSD's approach (agent definitions do not set model; orchestrator resolves at spawn time).

---

## Feature Dependencies

```
T1 (slot mapping) ──────────────────────────────────────┐
T2 (spawn model param) ────────────────┬────────────────┤
T3 (fix workers.md) ───────────────────┤                │
T4 (dual-mode resolution) ─────────────┤                │
T5 (CLI override) ─────────────────────┘                │
                                                       │
D1 (task complexity routing) ────── requires T1, T2    │
D2 (per-profile assignments) ────── requires T1         │
D3 (runtime switching) ──────────── requires D2         │
D4 (usage tracking) ──────────────── requires T2        │
```

**Build order:**
1. T3 (fix workers.md) -- zero-risk docs fix, unblocks thinking
2. T1 (slot mapping) -- foundational config change
3. T4 (dual-mode resolution) -- completes the routing layer
4. T2 (spawn model param) -- the core integration, touches playbooks
5. T5 (CLI override) -- extends T2 with user-facing control
6. D2 (per-profile assignments) -- extends T1 with profiles
7. D3 (runtime switching) -- extends D2 with commands
8. D1 (task complexity routing) -- extends T2 with intelligence
9. D4 (usage tracking) -- observability on top of working routing

---

## Behavior Matrix: Claude Mode vs GLM Mode

| Aspect | Claude Mode | GLM Mode |
|--------|-------------|----------|
| Trigger | User runs `claude` normally | User sets `ANTHROPIC_BASE_URL=http://localhost:4000` |
| opus slot resolves to | `inherit` -> Claude Opus (parent session version) | `inherit` -> GLM-5 (via proxy) |
| sonnet slot resolves to | `sonnet` -> Claude Sonnet | `sonnet` -> GLM-5-turbo (via proxy) |
| haiku slot resolves to | `haiku` -> Claude Haiku | `haiku` -> GLM-4.5-air (via proxy) |
| model-profiles.yaml | Maps castes to slots | Same config (slot-based, mode-agnostic) |
| LiteLLM proxy | Not used | Must be running at localhost:4000 |
| Aether routing code | Identical to GLM mode | Identical to Claude mode |
| User action to switch | `claude` (native) | `export` env vars + `claude` |

**Key insight:** Aether's routing code is identical in both modes. The only difference is the user's environment. Aether passes slot names to the Task tool; the environment (Claude native or LiteLLM proxy) resolves slots to actual models.

**Edge case -- proxy down in GLM mode:**
- LiteLLM proxy health check (build-wave Step 0.6) warns user but does not block
- If proxy is down, Task calls will fail at the Claude Code level (connection refused)
- Aether should NOT try to detect this and fallback -- that adds complexity for a scenario the user controls
- User's responsibility to start/stop proxy; Aether's responsibility to warn

**Edge case -- no opus quota in Claude mode:**
- If user's org blocks opus, the `"inherit"` value means the parent session already failed to get opus
- The user's session model would already be sonnet/haiku, so `"inherit"` inherits that
- No special handling needed -- this is exactly why GSD uses `"inherit"` instead of `"opus"`

---

## Caste Classification

Based on the v2.3 milestone target and GSD's quality profile as reference:

### Reasoning Castes (opus slot)

| Caste | Rationale |
|-------|-----------|
| prime | Orchestrates the entire colony, makes decomposition decisions, coordinates workers |
| oracle | Deep research via RALF loop -- complex multi-step reasoning with synthesis |
| archaeologist | Git history analysis requires deep reasoning about intent and context |
| route_setter | Phase planning and goal decomposition -- architecture decisions |
| architect | System design -- highest reasoning requirement |

### Execution Castes (sonnet slot)

| Caste | Rationale |
|-------|-----------|
| builder | Follows explicit plan instructions; implementation, not decision-making |
| watcher | Validation and quality checks; follows criteria, does not design them |
| scout | Research within codebase; structured exploration, not creative reasoning |
| chaos | Edge case generation and resilience testing; pattern-based, not reasoning |
| colonizer | Codebase exploration and territory mapping; read-only structured output |

### Future haiku candidates (for budget profile)

| Caste | Rationale |
|-------|-----------|
| surveyor-nest | Directory listing -- zero reasoning required |
| surveyor-provisions | Dependency listing -- structured extraction |
| probe | Coverage analysis -- pattern matching |

These are lower priority; haiku-tier routing can be added in the budget profile (D2) after v2.3 ships.

---

## Implementation Touch Points

Files that must change for T1-T5 (table stakes):

| File | Change | Complexity |
|------|--------|------------|
| `.aether/model-profiles.yaml` | Castes map to slots not model names; add slot_models section; add profiles section | LOW |
| `.aether/workers.md` lines 57-90 | Remove incorrect "cannot function" claim; document slot-based routing | LOW |
| `bin/lib/model-profiles.js` | Refactor from name-based to slot-based; DEFAULT_MODEL becomes DEFAULT_SLOT; validateModel becomes validateSlot | MEDIUM |
| `.aether/docs/command-playbooks/build-wave.md` | Add model resolution step before worker spawns; pass `model` param to Task calls | MEDIUM |
| `.aether/docs/command-playbooks/build-prep.md` | Resolve model profile at wave start | LOW |
| `.aether/docs/command-playbooks/continue-verify.md` | Pass model param to spawned verification workers | LOW |
| `.claude/agents/ant/*.md` (22 files) | No change needed -- keep `model: inherit` | NONE |
| `.claude/settings.json` | No change needed -- model routing is in model-profiles.yaml, not Claude settings | NONE |

---

## MVP Recommendation

**Ship in v2.3:**
1. T1 -- Slot-based caste mapping (foundational)
2. T3 -- Fix workers.md (unblocks mental model)
3. T4 -- Dual-mode resolution (completes the abstraction)
4. T2 -- Worker spawn passes model param (makes it real)
5. T5 -- CLI override (user control)

**Defer to v2.4:**
- D1 (task complexity routing) -- the keyword matching is brittle; better to ship fixed caste assignments first and refine later
- D2/D3 (profiles + switching) -- nice-to-have once base routing works
- D4 (usage tracking) -- observability comes after functionality

**Explicitly out of scope:**
- A1 through A5 (all anti-features)
- Per-worker environment variables (proven impossible)
- Model name routing in Aether code (architectural violation)

---

## Sources

| Source | Confidence | Evidence |
|--------|------------|----------|
| GSD `resolveModelInternal()` (gsd-tools.cjs:3970-3985) | HIGH | Working production code, proven pattern |
| GSD `MODEL_PROFILES` table (gsd-tools.cjs:128-140) | HIGH | 11 agents, 3 profiles, slot-based |
| GSD `model-profiles.md` reference | HIGH | Documents the `"inherit"` rationale for opus |
| GSD `model-profile-resolution.md` reference | HIGH | Documents Task tool model parameter usage |
| Aether `model-profiles.yaml` (current) | HIGH | Existing config, needs slot adaptation |
| Aether `model-profiles.js` (current) | HIGH | Existing library, needs refactoring |
| Aether `workers.md` lines 57-90 | HIGH | Incorrect claim, needs correction |
| Aether archived model routing (`.aether/archive/model-routing/`) | HIGH | v1 failure -- used model names not slots |
| Claude Code Task tool `model` parameter | HIGH (inferred from GSD working code) | No official docs accessible; GSD proves it works |
| LiteLLM proxy config (localhost:4000) | HIGH | Already configured in model-profiles.yaml |
