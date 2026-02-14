# Domain Pitfalls: Model Routing & Colony Lifecycle

**Domain:** AI agent orchestration, multi-model routing, colony lifecycle management
**Researched:** 2026-02-14
**Confidence:** HIGH (based on Aether's real-world issues, codebase analysis, and dream session findings)

## Overview

Aether Colony System uses LiteLLM proxy for model routing, assigning different AI models to different worker castes based on task complexity. The system also manages colony lifecycle across sessions through pause/resume mechanisms. This document catalogs pitfalls specific to these two domains based on Aether's actual architecture and the concerns raised in dream sessions.

---

## Critical Pitfalls for Model Routing

### 1. Model Routing Configuration Without Verification (CRITICAL)

**What goes wrong:**
The `model-profiles.yaml` defines sophisticated routing rules (caste-to-model mappings, task-based keyword routing), but the actual routing may not be happening. The dream session noted: "model routing isn't actually happening." Workers may all be using the default model regardless of configuration.

**Why it happens:**
- Configuration exists in YAML but execution path isn't verified
- Environment variables (`ANTHROPIC_MODEL`) may not be set before worker spawn
- LiteLLM proxy may be down or misconfigured, causing fallback to default
- Task tool inherits parent environment, but parent may not have set model-specific vars
- No runtime verification that the correct model is actually being used

**How to avoid:**
1. **Verify at spawn time:** Log the actual model being used when each worker starts
2. **Health check before spawn:** Verify LiteLLM proxy is healthy before spawning workers
3. **Model assertion in prompts:** Include "You are running on {model}" in worker prompts for confirmation
4. **Post-spawn verification:** Have workers report back which model they used
5. **Fail closed:** If proxy is down, fail the spawn rather than silently falling back

**Warning signs:**
- All workers report similar performance characteristics regardless of caste
- No variation in response time between "fast" and "slow" models
- Workers don't acknowledge their assigned model in responses
- `activity.log` shows no model assignment entries
- LiteLLM proxy logs show all requests going to same backend

**Phase to address:**
- v3.1 Open Chambers — Model Routing Verification

---

### 2. Proxy Authentication Failures Silently Defaulting (CRITICAL)

**What goes wrong:**
The LiteLLM proxy returns 401 "Authentication Error, No api key passed in" but the system continues as if routing is working. Workers silently fall back to direct API calls without the user's knowledge.

**Why it happens:**
- Proxy health check only checks if port is listening, not if auth is working
- `curl http://localhost:4000/health` returns healthy even with auth failures
- Claude Code has built-in fallback to direct API when proxy fails
- No verification that routed requests actually go through proxy

**How to avoid:**
1. **Test auth explicitly:** Send a test request with auth token and verify routing
2. **Monitor proxy logs:** Check that requests appear in LiteLLM logs with correct model
3. **Reject on auth failure:** Don't spawn workers if proxy auth fails
4. **User notification:** Warn user immediately if proxy routing unavailable
5. **Fallback audit:** Log when fallback to direct API occurs

**Warning signs:**
- Proxy health check passes but requests don't appear in proxy logs
- API usage shows direct provider calls instead of proxy routing
- Different cost/latency patterns than expected for routed models
- 401 errors in proxy logs with continued operation

**Phase to address:**
- v3.1 Open Chambers — Proxy Health Verification

---

### 3. Caste-Model Mismatch in Worker Spawns (HIGH)

**What goes wrong:**
A worker is spawned with one caste (e.g., "architect") but receives a model assigned to a different caste (e.g., "kimi-k2.5" instead of "glm-5"). The task complexity doesn't match the model capabilities.

**Why it happens:**
- `model-profile get` returns wrong model due to YAML parsing issues
- Caste name normalization problems ("architect" vs "Architect")
- Default fallback kicks in when specific caste not found
- Race condition in reading model-profiles.yaml during parallel spawns

**How to avoid:**
1. **Validate at spawn:** Compare requested caste vs returned model, warn on mismatch
2. **Case-insensitive matching:** Normalize caste names before lookup
3. **Strict mode:** Error if caste not found instead of defaulting
4. **Profile caching:** Load model profiles once at start, not per-spawn
5. **Audit trail:** Log both requested caste and assigned model for every spawn

**Warning signs:**
- Architects (should use glm-5) completing tasks unusually fast
- Builders (should use kimi-k2.5) struggling with complex reasoning
- Model assignment logs don't match caste in spawn logs
- Inconsistent model responses for same caste across different spawns

**Phase to address:**
- v3.1 Open Chambers — Model Assignment Validation

---

### 4. Environment Variable Inheritance Failures (HIGH)

**What goes wrong:**
The parent Claude Code process sets `ANTHROPIC_MODEL` for a specific caste, but spawned workers don't inherit it correctly. All workers end up using the parent's model or default.

**Why it happens:**
- Task tool environment inheritance is undocumented/unclear
- Shell exports in one command don't persist to next command
- Workers spawned via different mechanisms (direct vs script) get different environments
- Environment cleared between command invocations

**How to avoid:**
1. **Explicit env passing:** Pass environment variables directly in Task tool calls
2. **Verify inheritance:** Have workers echo back their environment on start
3. **Single-command spawn:** Set env and spawn in same shell invocation
4. **Avoid shell exports:** Don't rely on `export` persisting across tool calls
5. **Document behavior:** Record exactly how env inheritance works in Claude Code

**Warning signs:**
- Workers report different models than what was set before spawn
- Environment-sensitive behavior inconsistent across workers
- Model assignment logged but not reflected in worker responses
- Spawns via script behave differently than direct Task spawns

**Phase to address:**
- v3.1 Open Chambers — Environment Inheritance Testing

---

### 5. Task-Based Routing Never Triggered (MEDIUM)

**What goes wrong:**
The `model-profiles.yaml` includes sophisticated task-based routing hints (keywords like "design" → glm-5, "validate" → minimax-2.5), but this logic is never executed. All routing uses caste-based assignment only.

**Why it happens:**
- Task routing logic not implemented in spawn path
- Keyword detection requires parsing task descriptions, which is skipped
- Caste assignment happens before task analysis
- Performance optimization skips expensive keyword matching

**How to avoid:**
1. **Implement task routing:** Add keyword analysis before model selection
2. **Pre-compute at planning:** Store recommended model in task metadata during planning
3. **Override capability:** Allow explicit model override in task definition
4. **Measure benefit:** A/B test task routing vs caste-only to verify value
5. **Document limitation:** If not implementing, remove from config to avoid confusion

**Warning signs:**
- "design" tasks assigned to kimi-k2.5 instead of glm-5
- No code references to `task_routing` section of YAML
- Keyword-based rules in config but never mentioned in spawn logic
- Task complexity doesn't match model capabilities

**Phase to address:**
- v3.1 Open Chambers — Task Routing Implementation (or config cleanup)

---

## Critical Pitfalls for Colony Lifecycle

### 6. Pause/Resume Loses Model Context (CRITICAL)

**What goes wrong:**
When pausing and resuming a colony session, the model assignments from the previous session are lost. Workers resume with different models than they started with, breaking task continuity.

**Why it happens:**
- `COLONY_STATE.json` tracks phase and goal but not per-worker model assignments
- Model profile may have changed between pause and resume
- Workers spawned on resume get current model assignments, not historical ones
- No persistence of which model handled which task

**How to avoid:**
1. **Persist model assignments:** Store `model_used` in task metadata in COLONY_STATE.json
2. **Version profiles:** Track which version of model-profiles.yaml was active
3. **Resume with same models:** When resuming, use same models as original spawn
4. **Migration handling:** Document when model changes are acceptable vs breaking
5. **Handoff includes models:** Include active model assignments in HANDOFF.md

**Warning signs:**
- Resumed workers behave differently than before pause
- Task outputs change style/quality after resume
- Model assignment logs differ between original and resumed sessions
- Workers reference different capabilities after resume

**Phase to address:**
- v3.1 Open Chambers — Lifecycle State Persistence

---

### 7. Archive/Reset Destroys User Data (CRITICAL)

**What goes wrong:**
Colony archive or reset operations inadvertently delete or lose user data stored in colony memory (learnings, decisions, instincts). The checkpoint allowlist protects system files but may miss user-generated colony knowledge.

**Why it happens:**
- `memory.phase_learnings` and `memory.decisions` treated as ephemeral
- Archive operation only preserves system state, not user knowledge
- Reset clears all state without distinguishing user vs system data
- No clear boundary between "colony configuration" and "user work"

**How to avoid:**
1. **Separate user data:** Store user learnings/decisions in distinct location from system state
2. **Archive includes memory:** Ensure memory objects are preserved in archives
3. **Export before reset:** Offer to export learnings/instincts before reset
4. **Versioned memory:** Keep history of memory changes for recovery
5. **Clear documentation:** Explicitly state what is/isn't preserved

**Warning signs:**
- User-defined instincts disappear after reset
- Validated learnings from previous sessions gone
- Colony "forgets" user preferences and patterns
- Memory section empty after archive restore

**Phase to address:**
- v3.1 Open Chambers — Data Preservation Boundaries

---

### 8. Multiple Model Providers with Different Latencies (HIGH)

**What goes wrong:**
Workers using different models (Z.AI, Moonshot, MiniMax) have wildly different response times. Parallel tasks complete at different rates, causing coordination issues and timeouts.

**Why it happens:**
- No latency awareness in task scheduling
- Timeout values assume fastest model (kimi-k2.5)
- Synchronization points wait for slowest model (glm-5)
- No prioritization of fast models for time-sensitive tasks

**How to avoid:**
1. **Model-aware timeouts:** Set timeouts based on assigned model's typical latency
2. **Latency tracking:** Record and use historical latency per provider
3. **Fast-path for critical:** Use fast models for blocking coordination tasks
4. **Async for slow:** Use background tasks for slow models when possible
5. **Graceful degradation:** Continue without slow model results if timeout

**Warning signs:**
- Parallel tasks complete minutes apart
- Timeouts on glm-5 tasks that succeed on retry
- Watcher verification waiting disproportionately long
- User frustration with "stuck" phases

**Phase to address:**
- v3.1 Open Chambers — Latency-Aware Scheduling

---

### 9. Backward Compatibility Breaks Existing Colonies (HIGH)

**What goes wrong:**
Changes to model routing or colony state format break existing colonies. Old COLONY_STATE.json files don't work with new code, or model profiles change incompatibly.

**Why it happens:**
- State format changes without migration path
- Model names change in profiles (e.g., "glm-5" → "glm-5-latest")
- New required fields added to state without defaults
- Old colonies expect models that no longer exist

**How to avoid:**
1. **Version state format:** Include version field, migrate on load
2. **Graceful degradation:** Handle missing fields with sensible defaults
3. **Model aliases:** Support old model names as aliases to new ones
4. **Compatibility tests:** Test with old state files before release
5. **Deprecation warnings:** Notify users of breaking changes before applying

**Warning signs:**
- State validation fails after update
- "Model not found" errors for previously working colonies
- Missing field errors in COLONY_STATE.json
- Users report colonies "broken" after system update

**Phase to address:**
- v3.1 Open Chambers — State Migration System

---

### 10. Session Boundaries Lose In-Flight Work (MEDIUM)

**What goes wrong:**
When a session ends (crash, timeout, user closes Claude), workers that were spawned but not yet completed lose their results. The colony state shows tasks "in progress" but no workers are actually running.

**Why it happens:**
- Spawned workers are ephemeral — they don't persist across sessions
- No mechanism to resume or recover in-flight worker tasks
- State tracks "spawned" but not "completed" accurately
- Orphaned tasks remain in "in_progress" state forever

**How to avoid:**
1. **Checkpoint worker state:** Workers periodically report progress to state
2. **Timeout detection:** Mark tasks as failed if no progress for N minutes
3. **Resume capability:** Allow re-spawning workers for orphaned tasks
4. **Idempotent tasks:** Design tasks to be safely re-run
5. **Clear failure marking:** Distinguish "never started" from "started but lost"

**Warning signs:**
- Tasks stuck in "in_progress" for hours
- Spawn tree shows workers that never completed
- Colony state inconsistent with actual activity
- Repeated work because previous attempt lost

**Phase to address:**
- v3.1 Open Chambers — Worker Recovery Mechanism

---

## Moderate Pitfalls

### 11. Model Profile Drift Between Documentation and Reality

**What goes wrong:**
`workers.md` documents model assignments that differ from `model-profiles.yaml`. The documentation says one thing, the config says another, and the code does something else.

**Why it happens:**
- Multiple sources of truth for model assignments
- Documentation updated without changing config
- Config changed without updating documentation
- Code has hardcoded fallbacks that override config

**How to avoid:**
1. **Single source of truth:** Generate documentation from config
2. **Validation tests:** Assert workers.md matches model-profiles.yaml
3. **Config-driven:** No hardcoded models, everything from YAML
4. **Consistency CI:** Block PRs that change one without the other

**Warning signs:**
- Documentation says builder uses X, logs show Y
- model-profiles.yaml has different assignments than workers.md
- Code references models not in config
- Inconsistency between scout and builder assignments

**Phase to address:**
- v3.1 Open Chambers — Configuration Consistency

---

### 12. LiteLLM Proxy Single Point of Failure

**What goes wrong:**
All model routing depends on LiteLLM proxy at localhost:4000. If proxy crashes or is unreachable, entire colony stops working.

**Why it happens:**
- No fallback mechanism if proxy unavailable
- No health check before attempting to use proxy
- Workers can't function without proxy even if direct API available
- No local queue for retry when proxy recovers

**How to avoid:**
1. **Health check with fallback:** If proxy down, use direct API with warning
2. **Proxy redundancy:** Support multiple proxy endpoints
3. **Graceful degradation:** Queue requests, retry when proxy recovers
4. **Standalone mode:** Allow colony operation without proxy (single model)

**Warning signs:**
- All spawns fail when proxy down
- No way to use colony without running proxy separately
- Proxy crashes cause cascading colony failures
- Users confused by proxy requirement

**Phase to address:**
- v3.1 Open Chambers — Proxy Resilience

---

### 13. Colony State Bloat Over Long Sessions

**What goes wrong:**
Over many phases and sessions, COLONY_STATE.json grows unbounded with events, learnings, and history. Performance degrades and state operations become slow.

**Why it happens:**
- Events array grows without pruning
- All historical learnings retained forever
- No archiving of old phase data
- JSON parse/load time increases with file size

**How to avoid:**
1. **Event pruning:** Keep only last N events (already implemented: 100)
2. **Learning archival:** Move old learnings to archive file
3. **Phase compression:** Archive completed phase details
4. **Lazy loading:** Don't load full history unless needed
5. **Size monitoring:** Alert when state file exceeds threshold

**Warning signs:**
- State load takes noticeable time
- COLONY_STATE.json exceeds 1MB
- Memory usage grows with colony age
- Operations slow down over time

**Phase to address:**
- v3.1 Open Chambers — State Optimization

---

## Minor Pitfalls

### 14. Model Name Typos in Configuration

**What goes wrong:**
Typos in model-profiles.yaml (e.g., "kimik2.5" instead of "kimi-k2.5") cause lookup failures and fallback to default.

**How to avoid:**
- Validation script that checks all model names against known list
- YAML schema validation before accepting config changes

### 15. Context Window Mismatch

**What goes wrong:**
Tasks requiring large context (200K+ tokens) assigned to models with smaller windows, causing truncation or errors.

**How to avoid:**
- Check task estimated tokens against model context window
- Warn when assigning large-context task to small-context model

### 16. Cost Tracking Missing

**What goes wrong:**
No visibility into actual API costs per model, making optimization impossible.

**How to avoid:**
- Log token usage per request
- Aggregate costs by model/caste in activity log

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip proxy verification | Faster spawn | Wrong model usage undetected | Never — always verify routing |
| Use default model on error | Spawns always work | Lost optimization, wrong capabilities | Only with explicit user notification |
| Don't persist model in state | Smaller state files | Resume with wrong model | Never — model is critical context |
| Hardcode model fallbacks | Simpler error handling | Config changes don't apply | Never — use config exclusively |
| Ignore latency differences | Simpler scheduling | Poor user experience, timeouts | Only for single-model setups |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| LiteLLM Proxy | Only checking port, not auth | Test actual request with auth token |
| Claude Code Task tool | Assuming env inheritance | Explicitly pass environment variables |
| Model profiles YAML | Editing without validation | Run validation script before commit |
| COLONY_STATE.json | Adding fields without defaults | Always provide fallback for old states |
| Pause/Resume | Forgetting model context | Include model assignments in handoff |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Synchronous proxy health check | Slow spawn when proxy laggy | Async health with timeout | > 10 workers spawning |
| Loading model profiles per spawn | Spawn latency increases | Cache profiles at startup | > 5 parallel spawns |
| Full state write on every event | I/O bottleneck | Batch events, periodic flush | > 100 events/session |
| No model latency tracking | Timeouts on slow models | Track and adapt timeouts | Using glm-5 for first time |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Logging auth tokens | Token exposure in logs | Redact tokens in log output |
| Storing API keys in state | Key exposure in COLONY_STATE.json | Keep keys in env, never persist |
| Model proxy without auth | Unauthorized API usage | Require auth for all proxy requests |
| Including model costs in logs | Cost information leakage | Aggregate costs, don't log per-request |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Silent fallback to default model | User thinks routing works | Explicit notification: "Using default model, proxy unavailable" |
| No visibility into which model used | Can't verify optimization | Include model name in worker output headers |
| Proxy errors buried in logs | User doesn't know routing failed | Surface proxy issues in command output |
| Resume without context | User confused about colony state | Show handoff summary on resume |

---

## "Looks Done But Isn't" Checklist

- [ ] **Model routing:** Verify in LiteLLM logs that requests go to different backends — check that glm-5, kimi-k2.5, and minimax-2.5 all receive traffic
- [ ] **Proxy auth:** Confirm 401 errors don't result in silent fallback — test with invalid token
- [ ] **Environment inheritance:** Spawn a test worker that echoes `ANTHROPIC_MODEL` — verify it matches assignment
- [ ] **Pause/resume:** Pause colony, resume in new session, verify workers use same models
- [ ] **State migration:** Test with v2.0 state file — verify auto-upgrade works
- [ ] **Data preservation:** Reset colony, verify user learnings still available

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Model routing not working | LOW | Check proxy health, restart proxy, verify env vars, re-spawn workers |
| Wrong model assigned | LOW | Update COLONY_STATE.json with correct model, re-spawn affected workers |
| Proxy auth failure | LOW | Check auth token, verify proxy config, restart proxy with correct keys |
| State corruption | MEDIUM | Restore from backup, re-initialize if needed, re-apply user learnings |
| Lost user data after reset | HIGH | Check archive, restore from git history, manual reconstruction |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|------------|
| Model routing without verification | v3.1 Model Routing Verification | LiteLLM logs show traffic to multiple backends; workers report model used |
| Proxy auth silent fallback | v3.1 Proxy Health Verification | Auth failure stops spawn; user sees warning |
| Caste-model mismatch | v3.1 Model Assignment Validation | Spawn logs match caste; workers acknowledge correct model |
| Environment inheritance failure | v3.1 Environment Testing | Workers echo back expected environment variables |
| Pause/resume loses model context | v3.1 Lifecycle State Persistence | Resume uses same models as original session |
| Archive destroys user data | v3.1 Data Preservation | Reset preserves learnings/instincts; user data in separate location |
| Latency coordination issues | v3.1 Latency-Aware Scheduling | Timeouts appropriate per model; no stuck phases |
| Backward compatibility break | v3.1 State Migration | Old state files load successfully; auto-migration works |
| Session boundary data loss | v3.1 Worker Recovery | Orphaned tasks detected and re-spawned |
| Configuration drift | v3.1 Configuration Consistency | CI validates docs match config |
| Proxy single point of failure | v3.1 Proxy Resilience | Colony works (degraded) when proxy down |
| State bloat | v3.1 State Optimization | State file size bounded; load time constant |

---

## Sources

- Aether dream session: `/Users/callumcowie/repos/Aether/.aether/dreams/2026-02-14-0238.md` — "model routing isn't actually happening"
- Model profiles: `/Users/callumcowie/repos/Aether/.aether/model-profiles.yaml`
- Worker definitions: `/Users/callumcowie/repos/Aether/.aether/workers.md`
- Spawn implementation: `/Users/callumcowie/repos/Aether/.aether/utils/spawn-with-model.sh`
- Build command: `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md`
- State structure: `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json`
- Utility layer: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (model-profile command)
- Pause/resume: `/Users/callumcowie/repos/Aether/.claude/commands/ant/pause-colony.md`, `/Users/callumcowie/repos/Aether/.claude/commands/ant/resume-colony.md`
- Colony state tests: `/Users/callumcowie/repos/Aether/tests/unit/colony-state.test.js`
- LiteLLM proxy health check: Direct test (returned 401 auth error)

---

**Confidence Assessment:**

| Area | Level | Reason |
|------|-------|--------|
| Model routing gaps | HIGH | Dream session explicitly identified "model routing isn't actually happening" |
| Proxy auth issues | HIGH | Direct test showed 401 error, yet system continues |
| Environment inheritance | MEDIUM | Documented in code but behavior not verified |
| Lifecycle persistence | HIGH | State structure analysis shows gaps |
| Latency coordination | MEDIUM | Inferred from architecture, not yet observed |
| Backward compatibility | HIGH | State version field exists but migration unclear |

---

*This research informs v3.1 roadmap by flagging which model routing and lifecycle phases need deeper investigation.*
*Researched for: v3.1 Open Chambers milestone*
