# Domain Pitfalls: Per-Caste Model Routing

**Domain:** Adding per-caste GLM-5/GLM-5-turbo model routing via opus/sonnet model slots in Aether colony orchestration
**Researched:** 2026-03-27
**Confidence:** HIGH -- grounded in direct codebase inspection (184 test assertions, spawn mechanism, model-profiles.yaml, settings files, archived routing system, GSD model profile precedent)

---

## Critical Pitfalls

### Pitfall 1: The Task Tool Model Parameter Must Actually Be Used -- Or Routing Is Fiction

**What goes wrong:**
The v1 model routing was archived because Claude Code's Task tool did not support environment variable passing. The archive README states this explicitly (`.aether/archive/model-routing/README.md` line 28): "Claude Code's Task tool does not support environment variable passing to spawned subagents." The current system at v2 still has `spawn-with-model.sh` marked as "non-functional" (archive line 89).

The GSD system solved this by using the `model` parameter directly on Task calls: `Task(subagent_type="gsd-executor", model="{executor_model}")`. This works because Claude Code's Task tool accepts `"opus"`, `"sonnet"`, `"haiku"`, and `"inherit"` as model parameters. The question for v2.3 is whether the same mechanism can be applied to Aether's builder/watcher/scout spawns.

**Why it happens:**
The build-wave playbook (`build-wave.md` lines 288-290) currently spawns workers with `subagent_type="aether-builder"` but does NOT pass a `model` parameter. It relies entirely on the parent session's environment (`ANTHROPIC_MODEL` env var) which, as the archive confirms, is NOT inherited by Task-spawned subagents. This means all current spawns use whatever model the parent Queen session uses -- not the caste-specific model.

**How to avoid:**
1. Do NOT attempt to use `ANTHROPIC_MODEL` environment variable passing. The archive proves this does not work.
2. Follow the GSD pattern exactly: resolve the model slot (`"opus"` or `"sonnet"`) before each Task call and pass it as the `model` parameter.
3. The mapping is: reasoning castes (Prime, Oracle, Archaeologist, Route-Setter, Architect) get `"opus"` slot; execution castes (Builder, Watcher, Scout, Chaos, Colonizer) get `"sonnet"` slot.
4. Both slots are then mapped to GLM models via the LiteLLM proxy (opus -> glm-5, sonnet -> glm-5-turbo) in the proxy config -- NOT in settings.json.

**Warning signs:**
- Workers are spawned but their output quality is identical regardless of caste (all using same model)
- The spawn-tree.txt model column shows the same model for all castes after implementation
- Claude-only mode works fine but GLM proxy mode shows no difference between castes

**Phase to address:** Core implementation phase -- this is the foundational mechanism, must be proven working before anything else

**Evidence:** `.aether/archive/model-routing/README.md` lines 25-44, `.claude/get-shit-done/references/model-profile-resolution.md` lines 17-24, `build-wave.md` line 290

---

### Pitfall 2: Config Swap Workflow Breakage -- The Two-File Dance Gets Fragile

**What goes wrong:**
The user switches between Claude API mode and GLM proxy mode by swapping settings files. Currently this works because the model name in settings.json is irrelevant -- the only thing that matters is `ANTHROPIC_BASE_URL` (either pointing to Anthropic or to the LiteLLM proxy). All castes use the same model, so there is no per-caste config to get out of sync.

With per-caste routing, the system needs to know which mode is active to decide whether to pass `model: "opus"` or `model: "sonnet"` to Task calls. If the detection logic is wrong, workers get routed to the wrong model.

**Specific failure mode:**
The user is in Claude API mode (settings.json points to Anthropic). They run `/ant:build`. The build playbook resolves castes and passes `model: "opus"` for Prime. Claude Code interprets `"opus"` as `claude-opus-4` and routes to Anthropic's opus model. This actually works correctly for Claude mode -- it is the GLM proxy mode that requires the proxy-level mapping.

But if the user has GLM proxy settings active (settings.json.3model), `ANTHROPIC_BASE_URL` points to `localhost:4000`. Claude Code sends the `"opus"` request to the LiteLLM proxy, which then maps it to glm-5. This also works -- BUT only if the proxy config has the opus-to-glm-5 mapping.

**The real breakage:** If the proxy config does NOT have the model alias mapping (i.e., the proxy does not know that `opus` should route to `glm-5`), the request fails with "model not found" from the proxy. The proxy silently drops the request or returns an error that Claude Code surfaces as a generic "model error."

**Why it happens:**
The current LiteLLM proxy configuration is external to Aether -- it is a separate litellm config.yaml that the user manages. The proxy config and Aether's model-profiles.yaml are two separate systems that need to agree on model aliases. Per-caste routing adds a third layer: settings.json, model-profiles.yaml, and the proxy config must all be aligned.

**How to avoid:**
1. Aether should NOT try to detect which mode is active. Instead, Aether should always pass the model slot parameter (`"opus"` or `"sonnet"`) to Task calls. In Claude API mode, Claude Code maps `"opus"` to `claude-opus-4` natively. In proxy mode, the LiteLLM proxy maps it to whatever the proxy config says.
2. The proxy config mapping (opus -> glm-5, sonnet -> glm-5-turbo) is the user's responsibility. Aether's model-profiles.yaml should document this mapping but NOT try to enforce it.
3. Add a `model-profile verify` step that checks proxy health AND validates that the expected model aliases exist in the proxy. This already partially exists in aether-utils.sh line 2655-2664.
4. Do NOT add a "detect active mode" mechanism. Detection is fragile and breaks when the user has non-standard settings. Instead, document the two config files clearly and let the user manage the swap.

**Warning signs:**
- `/ant:build` spawns workers that fail with "model not found" from the proxy
- Workers complete but produce output inconsistent with their assigned model (e.g., a builder using opus-level reasoning instead of turbo-level speed)
- Config swap works in one direction (Claude -> GLM) but not the other

**Phase to address:** Implementation phase -- the model slot mechanism must be designed before config swap integration

**Evidence:** `verify-castes.md` lines 46-52, `model-profiles.yaml` line 92-95, `model-verify.js` lines 109-121

---

### Pitfall 3: GLM-5 Looping Despite Proxy Constraints -- The Constraint Escape

**What goes wrong:**
GLM-5 is known to loop in agent workflows. The LiteLLM proxy mitigates this with tight generation constraints (temperature 0.4, top_p 0.85, max_tokens 2500). These constraints work when the proxy is correctly configured. But there are at least four conditions where GLM-5 can escape these constraints:

1. **Temperature override in settings.json:** If the user's Claude Code settings include `temperature` or `top_p` parameters, they may override the proxy constraints. Claude Code sends these parameters in the API request, and if the LiteLLM proxy is configured to pass them through (rather than enforce its own), GLM-5 runs with the user's values.

2. **Proxy restart:** If the LiteLLM proxy is restarted and the config file is not loaded (e.g., the user runs `litellm` without `--config`), GLM-5 runs with default constraints (temperature 1.0, top_p 1.0, max_tokens unlimited).

3. **Nested agent calls:** GLM-5 spawned as Prime may spawn sub-workers (builders). If the sub-worker spawn does not also apply constraints, the sub-worker runs GLM-5 without constraints. The proxy constraints apply to the request, but GLM-5's own agent tendency to loop is amplified when it spawns further agents.

4. **System prompt too long:** The colony-prime prompt assembly injects queen wisdom, pheromone signals, skills, and research context. If the total prompt exceeds GLM-5's effective reasoning window (not its token limit, but the point where long-context attention degrades), GLM-5 may lose track of termination conditions and loop.

**Why it happens:**
The proxy constraints are a safety net, but they are not enforced at the application level. Aether relies on the LiteLLM proxy to constrain GLM-5, but Aether has no visibility into whether those constraints are actually in effect. The `model-profile verify` command checks proxy health but does not verify that specific model parameters (temperature, top_p, max_tokens) are set.

**How to avoid:**
1. Add GLM-5 loop detection at the application level. The spawn-tree.txt already records timestamps. If a worker's spawn-to-completion time exceeds a threshold (e.g., 10 minutes for a single task), log a warning. If completion is not recorded within 20 minutes, flag as likely looped.
2. Document that reasoning castes (Prime, Oracle) MUST have the LiteLLM proxy running with constraints. If `model-profile verify` detects the proxy is down, warn specifically that GLM-5 castes will loop without constraints.
3. For the Prime caste specifically: consider whether Prime should ALWAYS use glm-5-turbo despite being a "reasoning" caste. Prime is the orchestrator -- its job is coordination, not deep reasoning. If Prime loops, the entire build hangs.
4. Ensure the build-wave playbook includes max_turns or equivalent termination instructions in the worker prompt for GLM-5 castes. The current builder prompt does not include explicit termination conditions (build-wave.md lines 326-421).

**Warning signs:**
- A build hangs indefinitely with no worker completions logged to spawn-tree.txt
- Workers complete but their output is repetitive or circular (GLM-5 re-explaining the same thing)
- The LiteLLM proxy logs show the same model name in rapid succession (proxy constraint max_tokens is too high, allowing very long responses that look like looping)

**Phase to address:** Implementation phase (loop detection) and testing phase (verify proxy constraints are effective)

**Evidence:** `model-profiles.yaml` lines 16-25 (GLM-5 metadata explicitly warns about "tight constraints for agent stability"), `.aether/archive/model-routing/README.md` line 89, spawn-tree.txt timestamp format

---

### Pitfall 4: 184 Hardcoded Model Names In Tests Will Break On Any YAML Change

**What goes wrong:**
There are 184 occurrences of hardcoded model names (`glm-5-turbo`, `glm-5`, `glm-4.5-air`) across 6 test files. The current model-profiles.yaml has all castes mapped to `glm-5-turbo`. If per-caste routing changes some castes to `glm-5` or adds new model names, every test that asserts on the current model name will fail.

Worse: the test files use their own `createMockProfiles()` helper functions that duplicate the model names. These are not reading from the YAML file -- they are hardcoded separately. Changing the YAML does not automatically update the mock profiles in tests.

The integration test in `model-profiles.test.js` line 422 (`integration: load actual YAML and verify all castes`) reads the actual YAML file but still asserts specific model names: `t.is(modelProfiles.getModelForCaste(profiles, 'builder'), 'glm-5-turbo')`. When builder changes to a different model, this test breaks.

**Why it happens:**
The tests were written when all castes used `glm-5-turbo`. The mocks were copy-pasted from the YAML and never updated. There is no test helper that reads the YAML to generate mock profiles dynamically -- each test file has its own copy of the profile data.

**How to avoid:**
1. Before changing model-profiles.yaml, create a centralized test helper that reads the actual YAML and generates mock profiles. This way, changing the YAML automatically changes the test expectations.
2. When changing per-caste assignments, update tests in the SAME commit. Never change the YAML and defer test updates.
3. Add a "model profile consistency" test that reads the YAML, reads the test mock profiles, and asserts they are in sync. This catches the case where YAML is updated but test mocks are not.
4. The `createMockProfiles()` helper in each test file should be extracted to a shared test utility (`tests/helpers/mock-profiles.js`) so it only needs to be updated in one place.

**Warning signs:**
- `npm test` fails with "expected 'glm-5' but got 'glm-5-turbo'" or similar assertion errors
- Tests pass in one test file but fail in another (different mocks are out of sync in different ways)
- The integration test that reads actual YAML fails while unit tests that use mocks pass (or vice versa)

**Phase to address:** FIRST phase -- before changing any model assignments, refactor test infrastructure to be YAML-driven

**Evidence:** grep of tests/ directory: 184 occurrences across 6 files (model-profiles.test.js: 32, model-profiles-overrides.test.js: 20, model-profiles-task-routing.test.js: 49, cli-override.test.js: 19, cli-telemetry.test.js: 15, telemetry.test.js: 49)

---

## Moderate Pitfalls

### Pitfall 5: Silent Wrong-Model Spawn -- The Build Playbook Does Not Log Model Selection

**What goes wrong:**
The build-wave playbook (`build-wave.md` line 290) spawns workers with `subagent_type="aether-builder"` but does not log which model slot was selected. The `spawn-log` call (`spawn.sh` line 16) accepts an optional `model` parameter, but the build-wave playbook does not pass it. The spawn-tree.txt entry records the model as `"default"` for all workers.

If per-caste routing is implemented but the model selection is not logged, there is no audit trail for debugging. A user reports "my builder seems slower than usual" and there is no way to verify which model the builder actually used.

**Why it happens:**
The spawn-log function was designed before model routing existed. The model parameter is optional (defaults to `"default"`). The build-wave playbook was written for the single-model era and never updated to pass model information.

**How to avoid:**
1. When implementing per-caste routing, update the build-wave playbook to pass the resolved model slot to `spawn-log`. The call should look like: `bash .aether/aether-utils.sh spawn-log "Queen" "builder" "{ant_name}" "{task}" "{model_slot}"`.
2. The spawn-tree.txt format already supports a model field (line 27: `$ts_full|$parent_id|$child_caste|$child_name|$task_summary|$model|$status`). Verify the build-wave playbook populates this field.
3. Add a post-build verification step that reads spawn-tree.txt and checks that each caste got the expected model slot. If a caste got the wrong model, log a warning to the activity log.

**Warning signs:**
- spawn-tree.txt shows `"default"` for all workers after per-caste routing is implemented
- Users cannot determine which model was used for a specific worker
- Debugging a "wrong model" issue requires re-running the entire build

**Phase to address:** Implementation phase -- model logging should be part of the core routing mechanism

**Evidence:** `spawn.sh` line 16-17 (model parameter), `spawn.sh` line 27 (spawn-tree format with model field), `build-wave.md` line 318 (spawn-log call without model parameter)

---

### Pitfall 6: Colony Lifecycle Edge Cases -- Init, Seal, Entomb May Skip Model Resolution

**What goes wrong:**
The build-wave playbook is the primary spawn mechanism, but Aether has other spawn contexts:

1. **`/ant:init`** initializes a colony. If it spawns workers for colonize/survey, those workers bypass the build-wave playbook and may not get model routing.
2. **`/ant:seal`** marks a colony complete. It may spawn workers for cleanup or archiving.
3. **`/ant:entomb`** archives a completed colony.
4. **`/ant:swarm`** spawns parallel scouts for bug investigation.
5. **`/ant:oracle`** runs deep research with the RALF loop.
6. **`/ant:chaos`** runs resilience testing.

These commands may spawn workers directly (via Task tool) without going through the build-wave playbook's model resolution logic. If model routing is only implemented in build-wave.md, workers spawned by these other commands get whatever model the parent session uses.

**Why it happens:**
The build-wave playbook is the most complex spawn mechanism. Other commands have simpler spawn logic that was not designed with model routing in mind. Each command is a separate markdown file interpreted by Claude Code, so there is no shared "spawn with model" function they all call.

**How to avoid:**
1. Create a shared "model resolution" snippet that can be included in any spawn command. This snippet should: (a) determine the caste, (b) look up the model slot from model-profiles.yaml, (c) pass the model parameter to the Task call.
2. Audit every command that spawns workers and update it to include the model resolution snippet. Commands to check: init.md, seal.md, entomb.md, swarm.md, oracle.md, chaos.md.
3. For the initial implementation, it is acceptable to only route models in build-wave.md and have other commands use the default. But document this limitation clearly.
4. The spawn-log call should always include the model parameter (Pitfall 5), even if the model is "default" for commands that do not yet support routing. This creates an audit trail for future implementation.

**Warning signs:**
- Workers spawned by `/ant:swarm` use a different model than workers spawned by `/ant:build`
- Oracle research quality is inconsistent (sometimes uses glm-5, sometimes glm-5-turbo)
- Seal/entomb cleanup workers fail if they need capabilities not available in the default model

**Phase to address:** Second phase -- after build-wave routing is proven, extend to other spawn commands

**Evidence:** CLAUDE.md lists 44 commands, build-wave.md is one of 9 playbooks in `.aether/docs/command-playbooks/`

---

### Pitfall 7: User Settings.json Missing Expected Model Slot Mappings

**What goes wrong:**
The PROJECT.md scope says "Map opus slot to glm-5, sonnet slot to glm-5-turbo." This mapping must exist somewhere for the proxy to route correctly. If the user's settings.json does not have model mappings that align with Aether's expectations (or if the user has custom model aliases), the routing fails silently.

Specifically: Claude Code's Task tool accepts `"opus"`, `"sonnet"`, `"haiku"` as model parameters. These are Claude Code's own aliases. In Claude API mode, `"opus"` maps to `claude-opus-4`. In proxy mode, the LiteLLM proxy needs to know that `"opus"` maps to `glm-5`. If the user's proxy config maps `"opus"` to something else (or does not map it at all), the routing is wrong.

**Why it happens:**
Aether's model-profiles.yaml defines the Aether-level mapping (caste -> model name). The LiteLLM proxy config defines the proxy-level mapping (model alias -> actual model). The user's settings.json defines the Claude Code-level connection. These are three separate config files managed by different parts of the system. They can be independently correct but mutually inconsistent.

**How to avoid:**
1. In `model-profiles.yaml`, use the Claude Code model slot names (`"opus"`, `"sonnet"`, `"haiku"`) as the model identifiers, NOT GLM-specific names. The current YAML uses `glm-5-turbo` directly. Changing to use slot names means the YAML is mode-agnostic: it says "builder uses sonnet" rather than "builder uses glm-5-turbo."
2. The LiteLLM proxy config is where the actual GLM mapping lives: `opus -> glm-5`, `sonnet -> glm-5-turbo`. This mapping is the user's responsibility. Aether should document the expected mapping but not try to configure the proxy.
3. Add a `model-profile verify` check that tests the model slot by making a tiny request to the proxy and checking the response. This verifies end-to-end that "opus" actually routes to the expected model.
4. The `DEFAULT_MODEL` constant in `model-profiles.js` (line 17) should be `"sonnet"` (the safe default), not `"glm-5-turbo"` (a GLM-specific name).

**Warning signs:**
- `model-profile verify` passes (proxy is healthy) but workers get the wrong model
- Users report that "all workers are using opus" when they should be split
- The proxy logs show model names that do not match expectations

**Phase to address:** Implementation phase -- the model naming convention must be decided before writing any code

**Evidence:** `model-profiles.js` line 17 (`DEFAULT_MODEL = 'glm-5-turbo'`), `model-profiles.yaml` lines 4-14 (GLM-specific model names), `verify-castes.md` lines 55-57 (lists GLM model names as "available models via LiteLLM proxy")

---

### Pitfall 8: The Bash YAML Parser and Node.js Library Can Return Different Results

**What goes wrong:**
Aether has TWO model profile parsers:
1. **Bash awk parser** in aether-utils.sh (lines 2626-2628) -- used by `model-profile get` and `model-profile list`
2. **Node.js yaml.load parser** in model-profiles.js (lines 36-57) -- used by `model-profile select` and `model-profile validate`

The bash parser uses a simple awk pattern (`/^  '$caste':/{print $2; exit}`) that extracts the second field from lines matching the caste name. The Node.js parser uses `js-yaml` which does full YAML parsing including environment variable substitution, multiline strings, comments, and anchors.

These parsers can diverge on:
- Comments after values (bash parser ignores everything after field 2, Node.js strips comments properly)
- Environment variable substitution (Node.js handles `${VAR:-default}`, bash parser does not)
- YAML anchors/aliases (Node.js resolves them, bash parser returns the literal `*anchor`)
- User overrides section (Node.js reads `user_overrides` and applies them in `getEffectiveModel`, bash parser only reads `worker_models`)

**Why it happens:**
The bash parser was written first (quick awk for simple YAML). The Node.js library was added later for more complex operations (task routing, override management). Neither was replaced by the other. The bash parser is still used for `model-profile get` because it avoids the Node.js startup overhead.

**How to avoid:**
1. When implementing per-caste routing, use ONLY the Node.js library for model resolution in the build-wave playbook. The bash `model-profile get` command is used by other commands and should be updated to delegate to the Node.js library (like `model-profile select` already does).
2. If the bash parser must be kept (for performance reasons), add a test that runs both parsers against the same YAML and asserts identical output. This catches divergence early.
3. The `model-profile get` bash implementation does NOT check user_overrides (line 2628-2631). The Node.js `getEffectiveModel` does check them (lines 321-339). If per-caste routing uses `getEffectiveModel` but logging/display uses `model-profile get`, the model shown to the user may differ from the model actually used.

**Warning signs:**
- `model-profile get builder` returns `glm-5-turbo` but the actual spawn uses `glm-5` (because of a user override)
- `model-profile list` shows different values than what the build-wave playbook resolves
- Tests pass for the Node.js library but fail for the bash parser (or vice versa)

**Phase to address:** Implementation phase -- standardize on one parser before routing logic depends on consistent results

**Evidence:** aether-utils.sh lines 2626-2631 (bash awk parser), model-profiles.js lines 36-57 (Node.js yaml.load), model-profiles.js lines 321-339 (getEffectiveModel with overrides)

---

## Minor Pitfalls

### Pitfall 9: Archived Model Routing Files Create Confusion

**What goes wrong:**
`.aether/archive/model-routing/` contains a complete model routing implementation that was archived because it did not work (Task tool limitation). The archive includes model-profiles.yaml with a different caste-to-model mapping (using minimax-2.5 and kimi-k2.5), a model-profiles.js file, and a detailed README explaining why it failed.

Developers (or the LLM itself) may reference the archived implementation as a "how to do it" guide, not realizing it was specifically archived because it does not work. The archived YAML has different model names and a different architecture (3 models: glm-5, minimax-2.5, kimi-k2.5 vs. current 2 models: glm-5, glm-5-turbo).

**How to avoid:**
1. Before starting implementation, add a prominent note to the archived files: "ARCHIVED -- DO NOT REFERENCE FOR IMPLEMENTATION. See v2.3 milestone plan for current approach."
2. Ensure the new implementation uses the GSD pattern (model parameter on Task call) rather than the archived approach (environment variable passing).
3. Consider deleting the archive entirely if it causes confusion. The README explains the problem well enough that the archive files are not needed.

**Phase to address:** Pre-implementation cleanup

**Evidence:** `.aether/archive/model-routing/README.md`, `.aether/archive/model-routing/model-profiles.yaml` (different model names)

---

### Pitfall 10: Model Metadata Section Becomes Stale After Per-Caste Changes

**What goes wrong:**
model-profiles.yaml has a `model_metadata` section (lines 15-61) with capabilities, context windows, speed, and cost tiers for each model. This metadata is used by `getModelMetadata()` and `getAllAssignments()` in model-profiles.js. If new model names are added or models are renamed, the metadata section must be updated in sync.

Currently, the metadata says glm-5 has `context_window: 200000` and glm-5-turbo has `context_window: 200000`. If the per-caste routing adds a third model (e.g., glm-4.5-air for validation tasks), the metadata section must include it. If it does not, `validateModel()` will reject the new model name because it is not in `model_metadata`.

**How to avoid:**
1. Any model name change must update both `worker_models` and `model_metadata` sections.
2. Add a validation test that checks: for every model in `worker_models`, there is a corresponding entry in `model_metadata`. This catches missing metadata entries immediately.
3. The `validateModel()` function (model-profiles.js line 131) checks `model_metadata` keys. If a caste is assigned a model that is not in `model_metadata`, validation fails. This is actually good behavior -- but it will surprise developers who add a model to `worker_models` and forget `model_metadata`.

**Phase to address:** Implementation phase -- update metadata when changing model assignments

**Evidence:** model-profiles.yaml lines 15-61, model-profiles.js lines 131-140

---

### Pitfall 11: The "inherit" Pattern From GSD Does Not Apply Here

**What goes wrong:**
GSD uses `"inherit"` for opus-tier agents so they use the parent session's model, avoiding org policy conflicts. The PROJECT.md says "Map opus slot -> glm-5." If someone copies the GSD pattern and uses `"inherit"` for reasoning castes, those castes will use the parent Queen's model -- which defeats the purpose of per-caste routing.

The GSD pattern makes sense for GSD because GSD agents are spawned by an orchestrator that may be running on any model. But in Aether, the Queen is always running on the same model as the workers (because of the single-session constraint). There is no "inherit" benefit -- the Queen IS the session model.

**How to avoid:**
1. Use `"opus"` for reasoning castes, not `"inherit"`. The GSD reasoning for `"inherit"` (avoiding org policy conflicts with specific opus versions) does not apply when routing through a LiteLLM proxy that maps `"opus"` to `glm-5`.
2. Document why Aether uses `"opus"` instead of `"inherit"`, so future developers do not copy the GSD pattern without understanding the difference.

**Phase to address:** Implementation phase -- model naming decision

**Evidence:** `.claude/get-shit-done/references/model-profiles.md` line 92 (explains inherit rationale), PROJECT.md line 13 (Aether uses opus slot -> glm-5)

---

## Phase-Specific Warnings

| Phase | Likely Pitfall | Mitigation |
|-------|---------------|------------|
| Test infrastructure refactor | Pitfall 4: 184 hardcoded model names | Centralize mock profiles before any YAML changes |
| Core routing implementation | Pitfall 1: Task tool model parameter | Prove with a single Task(spawn, model="opus") call before building full system |
| Core routing implementation | Pitfall 7: Model naming convention | Use Claude Code slot names ("opus"/"sonnet") not GLM names in model-profiles.yaml |
| Core routing implementation | Pitfall 8: Parser divergence | Standardize on Node.js library, deprecate bash awk parser |
| Proxy config documentation | Pitfall 2: Config swap breakage | Document expected proxy mapping clearly, add verify step |
| GLM-5 loop prevention | Pitfall 3: Constraint escape | Add application-level loop detection, document Prime-as-turbo option |
| Build playbook update | Pitfall 5: Missing model logging | Pass model slot to spawn-log in every spawn call |
| Non-build command update | Pitfall 6: Lifecycle edge cases | Audit all spawn commands, update one by one |
| YAML changes | Pitfall 10: Stale metadata | Add validation test for metadata completeness |
| Pre-implementation cleanup | Pitfall 9: Archive confusion | Delete or annotate archived files |

---

## Suggested Implementation Order

Based on pitfall dependencies, the phases should be ordered:

1. **Test infrastructure refactor** -- Centralize 184 hardcoded model name references before any YAML changes (Pitfall 4)
2. **Pre-implementation cleanup** -- Annotate or delete archived routing files (Pitfall 9)
3. **Core routing mechanism** -- Implement model slot resolution + Task tool model parameter (Pitfalls 1, 7, 8)
4. **Build playbook integration** -- Wire routing into build-wave.md, add model logging (Pitfalls 5, 10)
5. **Proxy verification** -- Add end-to-end model verification, document config swap (Pitfall 2)
6. **GLM-5 loop prevention** -- Add application-level loop detection (Pitfall 3)
7. **Lifecycle command audit** -- Extend routing to non-build spawn commands (Pitfall 6)
8. **Documentation update** -- Update verify-castes.md, CLAUDE.md, and all references (Pitfall 11)

**Critical ordering constraint:** Step 1 (test refactor) MUST come before Step 2 (any YAML changes). If YAML changes happen before test infrastructure is updated, every test change becomes a guessing game of "which of the 184 occurrences does this need to match?"

---

## Sources

- `.aether/archive/model-routing/README.md` -- HIGH confidence (direct codebase inspection, explains why v1 routing failed)
- `.aether/model-profiles.yaml` -- HIGH confidence (current configuration, lines 1-96)
- `bin/lib/model-profiles.js` -- HIGH confidence (current library code, 446 lines)
- `bin/lib/model-verify.js` -- HIGH confidence (verification logic, 289 lines)
- `.aether/utils/spawn.sh` -- HIGH confidence (spawn mechanism, 240 lines)
- `.aether/utils/spawn-with-model.sh` -- HIGH confidence (non-functional spawn helper, 57 lines)
- `.aether/docs/command-playbooks/build-wave.md` -- HIGH confidence (build spawn logic, 598 lines)
- `.aether/aether-utils.sh` lines 2610-2760 -- HIGH confidence (model-profile bash commands)
- `.claude/commands/ant/verify-castes.md` -- HIGH confidence (current documentation of model system)
- `.claude/get-shit-done/references/model-profile-resolution.md` -- HIGH confidence (GSD working pattern)
- `.claude/get-shit-done/references/model-profiles.md` -- HIGH confidence (GSD model profile table)
- `.planning/PROJECT.md` -- HIGH confidence (milestone scope definition)
- Test files (184 model name occurrences) -- HIGH confidence (direct grep results)

---
*Pitfalls research for: Aether v2.3 Per-Caste Model Routing*
*Researched: 2026-03-27*
