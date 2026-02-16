# Phase 9: Caste Model Assignment - Research

**Researched:** 2026-02-14
**Domain:** Model routing, LiteLLM proxy integration, CLI command patterns
**Confidence:** HIGH

## Summary

This research investigates the current state of model routing in the Aether colony system and what needs to be built to enable users to view, verify, and configure AI model assignments per worker caste.

**Key Finding:** The model routing infrastructure exists but the execution path is unverified. The `model-profiles.yaml` defines caste-to-model mappings, `aether-utils.sh` has `model-profile` commands, and `build.md` documents the intended flow. However, there's a critical gap: workers may all be using default models because the environment variable propagation through Task tool spawns isn't verified.

**Primary recommendation:** Build verification-first: create the `aether caste-models` CLI commands and `/ant:verify-castes` slash command to surface actual model usage, then fix any gaps in the routing pipeline.

## Current State Analysis

### What Exists

| Component | Location | Status | Notes |
|-----------|----------|--------|-------|
| Model profiles config | `.aether/model-profiles.yaml` | Complete | Defines 10 castes → 3 models (glm-5, kimi-k2.5, minimax-2.5) |
| Model profile commands | `.aether/aether-utils.sh` (lines 1561-1630) | Complete | `model-profile get/list/verify`, `model-get`, `model-list` |
| Spawn helper | `.aether/utils/spawn-with-model.sh` | Complete | Sets ANTHROPIC_MODEL before spawning |
| Verification library | `bin/lib/model-verify.js` | Partial | `createVerificationReport()` exists, used by `aether verify-models` |
| CLI command | `bin/cli.js` | Partial | `aether verify-models` exists (lines 1527-1573) |
| Build command | `.claude/commands/ant/build.md` | Documented | Lines 319-360 describe intended model assignment flow |
| Worker docs | `.aether/workers.md` | Complete | Lines 53-117 document model-aware spawning |

### What's Missing

| Requirement | ID | Gap |
|-------------|-----|-----|
| View model assignments | MOD-01 | No `aether caste-models list` command |
| Override model for caste | MOD-02 | No `aether caste-models set` command |
| Proxy health verification | MOD-03 | Health check exists but not integrated into spawn flow |
| Provider routing info | MOD-04 | No command to show which provider handles which model |
| Log actual model used | MOD-05 | Spawn log doesn't record resolved model |
| Surface Dreams in status | QUICK-01 | `/ant:status` doesn't read `.aether/dreams/` |
| Auto-load context | QUICK-02 | Commands don't recognize nestmates (sibling projects) |
| Verify castes command | QUICK-03 | No `/ant:verify-castes` slash command |

### Critical Finding: Unverified Execution Path

The Dream journal (2026-02-14-0238.md, Dream 3) identified:

> "The model profiles are ready. The LiteLLM proxy endpoint is defined. But the actual routing — the moment when a Builder ant is spawned and told 'use kimi-k2.5 for this task' — that's documentation and configuration, not verified execution."

**Evidence:**
1. `build.md` documents setting `ANTHROPIC_MODEL` before spawning (lines 337-339, 349-356)
2. `spawn-with-model.sh` sets the environment variables (lines 32-34)
3. But the Task tool inherits environment from the parent Claude Code process
4. No verification exists that spawned workers actually receive the correct model

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|--------------|---------|---------|--------------|
| LiteLLM Proxy | latest | Model routing gateway | Required for multi-model routing |
| Commander.js | ^11.x | CLI framework | Already used in `bin/cli.js` |
| js-yaml | ^4.x | YAML parsing | For reading `model-profiles.yaml` |
| node-fetch | ^3.x | HTTP requests | For proxy health checks |

### Existing Utilities
| Utility | Location | Purpose |
|---------|----------|---------|
| `aether-utils.sh model-profile get <caste>` | `.aether/aether-utils.sh:1562` | Get model for caste |
| `aether-utils.sh model-profile list` | `.aether/aether-utils.sh:1582` | List all assignments |
| `aether-utils.sh model-profile verify` | `.aether/aether-utils.sh:1598` | Check proxy health + caste count |
| `createVerificationReport()` | `bin/lib/model-verify.js:221` | Full verification report |

## Architecture Patterns

### CLI Command Pattern (from `bin/cli.js`)

```javascript
// Standard command structure
program
  .command('caste-models')
  .description('Manage caste-to-model assignments')
  .addCommand(
    program.createCommand('list')
      .description('List current model assignments')
      .action(wrapCommand(async () => {
        // Implementation
      }))
  )
  .addCommand(
    program.createCommand('set')
      .description('Set model for a caste')
      .argument('<caste>', 'caste name (builder, watcher, etc.)')
      .argument('<model>', 'model alias (glm-5, kimi-k2.5, minimax-2.5)')
      .action(wrapCommand(async (caste, model) => {
        // Implementation
      }))
  );
```

### Model Profile YAML Structure

```yaml
# .aether/model-profiles.yaml
worker_models:
  prime: glm-5           # Long-horizon coordination
  builder: kimi-k2.5     # Code generation
  watcher: kimi-k2.5     # Validation
  oracle: minimax-2.5    # Research

model_metadata:
  glm-5:
    provider: "z_ai"
    capabilities: [planning, coordination, long_context]
  kimi-k2.5:
    provider: "kimi"
    capabilities: [coding, multimodal, agent_swarm]
  minimax-2.5:
    provider: "minimax"
    capabilities: [system_design, browse, search]

proxy:
  endpoint: "http://localhost:4000"
  health_check: "http://localhost:4000/health"
```

### Slash Command Pattern (from `.claude/commands/ant/`)

```markdown
---
name: ant:verify-castes
description: "Verify model routing is working for all castes"
---

You are the **Queen**. Verify that model routing is active.

### Step 1: Check Proxy Health
Run: `curl -s http://localhost:4000/health`

### Step 2: Verify Each Caste
For each caste in [prime, builder, watcher, oracle, scout]:
1. Get assigned model: `bash .aether/aether-utils.sh model-profile get {caste}`
2. Verify model is not default
3. Log result

### Step 3: Test Spawn
Spawn a test worker with explicit model and verify it receives it.
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML parsing | Custom regex | `js-yaml` library | Handles edge cases, comments, nested structures |
| Proxy health check | `curl` shell calls | `node-fetch` with timeout | Better error handling, JSON parsing |
| Model validation | Hardcoded list | Read from `model-profiles.yaml` | Single source of truth |
| Config updates | String manipulation | Read → Modify → Write pattern | Prevents corruption |

## Common Pitfalls

### Pitfall 1: Environment Variable Propagation
**What goes wrong:** Setting `ANTHROPIC_MODEL` in the parent process doesn't guarantee spawned workers receive it.

**Why it happens:** The Task tool inherits environment from the Claude Code process, but if the parent was started without the variables, children won't have them either.

**How to avoid:**
- Always verify environment in spawned workers
- Log actual model used at spawn time
- Provide fallback to default with warning

**Warning signs:** All workers report using the same model regardless of caste.

### Pitfall 2: YAML Parsing Fragility
**What goes wrong:** Simple regex parsing of YAML breaks on comments, multi-line values, or nested structures.

**Why it happens:** Current `aether-utils.sh` uses awk for YAML parsing (lines 1576-1591), which is fragile.

**How to avoid:** Use proper YAML library (js-yaml) for any non-trivial parsing.

### Pitfall 3: Proxy Health False Positives
**What goes wrong:** Proxy health endpoint returns 200 but routing doesn't work.

**Why it happens:** Health check only verifies proxy is running, not that it can reach upstream providers.

**How to avoid:**
- Test actual model routing with a small request
- Check provider-specific endpoints
- Log latency and errors

### Pitfall 4: Model Alias Mismatches
**What goes wrong:** Profile uses `kimi-k2.5` but proxy expects `kimi/k2.5` or vice versa.

**Why it happens:** LiteLLM proxy uses provider/model format, but profiles use shorthand aliases.

**How to avoid:**
- Document alias → provider mapping clearly
- Validate aliases against proxy config
- Provide clear error messages on mismatch

## Code Examples

### Reading Model Profiles (Node.js)

```javascript
const yaml = require('js-yaml');
const fs = require('fs');

function loadModelProfiles(repoPath) {
  const profilePath = path.join(repoPath, '.aether', 'model-profiles.yaml');
  const content = fs.readFileSync(profilePath, 'utf8');
  return yaml.load(content);
}

function getModelForCaste(profiles, caste) {
  return profiles.worker_models?.[caste] || 'kimi-k2.5';
}
```

### Proxy Health Check

```javascript
async function checkProxyHealth(endpoint = 'http://localhost:4000') {
  try {
    const response = await fetch(`${endpoint}/health`, {
      signal: AbortSignal.timeout(5000)
    });
    return {
      healthy: response.ok,
      status: response.status,
      latency: Date.now() - startTime
    };
  } catch (error) {
    return {
      healthy: false,
      error: error.message
    };
  }
}
```

### Logging Model Usage

```javascript
// In spawn flow
const model = getModelForCaste(caste);
console.log(`[${timestamp}] Spawning ${antName} (${caste}) with model: ${model}`);

// Log to activity log
await exec(`bash .aether/aether-utils.sh activity-log "MODEL" "Queen" "${antName} (${caste}): ${model}"`);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single default model | Caste-based routing | v3.0 | Different castes can use optimal models |
| Manual env var setting | `spawn-with-model.sh` helper | v3.0 | Centralized model assignment |
| No verification | `aether verify-models` command | v3.0 | Can check if routing is configured |
| Hardcoded model list | `model-profiles.yaml` | v3.0 | User-configurable assignments |

**Deprecated/outdated:**
- Direct `ANTHROPIC_MODEL` setting without profile lookup
- Hardcoded model assignments in command files

## Implementation Approach

### MOD-01: View model assignments (`aether caste-models list`)

**File:** `bin/cli.js` (add new command)
**Approach:**
1. Read `.aether/model-profiles.yaml` using js-yaml
2. Display table: Caste | Model | Provider | Capabilities
3. Show proxy status indicator

**Dependencies:** Add `js-yaml` to package.json

### MOD-02: Override model (`aether caste-models set <caste>=<model>`)

**File:** `bin/cli.js` (add to caste-models command)
**Approach:**
1. Validate caste exists in known list
2. Validate model exists in model_metadata
3. Read YAML → modify → write back
4. Log change to activity log

**Edge cases:**
- Invalid caste name → error with valid options
- Invalid model → error with valid models
- YAML write failure → backup and restore

### MOD-03: Verify proxy health before spawning

**File:** `.aether/aether-utils.sh` (enhance spawn-log)
**Approach:**
1. Before spawn, check `curl -s http://localhost:4000/health`
2. If unhealthy, log warning and fall back to default
3. Add `--verify-proxy` flag to spawn commands

### MOD-04: Show provider routing info

**File:** `bin/cli.js` (add to caste-models list)
**Approach:**
1. Query LiteLLM proxy `/models` endpoint
2. Map model aliases to provider routes
3. Display routing table

### MOD-05: Log actual model used

**File:** `.aether/aether-utils.sh` (modify spawn-log)
**Approach:**
1. Add `model` parameter to spawn-log command
2. Record in spawn-tree.txt format: `timestamp|parent|caste|child|task|model|status`
3. Update spawn-tree visualization to show model

### QUICK-01: Surface Dreams in `/ant:status`

**File:** `.claude/commands/ant/status.md`
**Approach:**
1. List `.aether/dreams/` directory
2. Show most recent dream (by filename timestamp)
3. Display count of total dreams
4. Add line: `Dreams: N recorded (latest: YYYY-MM-DD)`

### QUICK-02: Auto-load context (nestmates)

**File:** `.claude/commands/ant/init.md` and others
**Approach:**
1. Check for sibling `.aether/` directories
2. If found, load their constraints and instincts
3. Display: `Nestmates found: N related colonies`

### QUICK-03: `/ant:verify-castes` command

**File:** `.claude/commands/ant/verify-castes.md` (new)
**Approach:**
1. Check proxy health
2. For each caste, verify model assignment
3. Spawn test worker for each caste
4. Have worker report which model it sees
5. Display verification report

## File Locations and Modification Points

| File | Line | Change |
|------|------|--------|
| `bin/cli.js` | After 1607 | Add `caste-models` command with subcommands |
| `bin/cli.js` | 25 | Import js-yaml |
| `package.json` | dependencies | Add `js-yaml: ^4.1.0` |
| `.aether/aether-utils.sh` | 319 | Add model parameter to spawn-log |
| `.aether/aether-utils.sh` | 334 | Include model in spawn-tree.txt format |
| `.claude/commands/ant/status.md` | After 132 | Add Dreams section |
| `.claude/commands/ant/verify-castes.md` | New file | Create verification command |

## Dependencies and Prerequisites

### Required
- `js-yaml` package for YAML parsing
- LiteLLM proxy running on localhost:4000
- `.aether/model-profiles.yaml` exists

### Optional but Recommended
- `node-fetch` for proxy API calls (may already be available in Node 18+)
- Write access to `.aether/` for profile updates

## Risks and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| YAML corruption on write | Medium | High | Backup before write, validate after |
| Proxy not running | High | Medium | Graceful fallback to default model |
| Invalid model alias | Low | Medium | Validate against known models |
| Task env propagation fails | Medium | High | Verify in spawned workers, log actual model |
| Breaking existing spawn flow | Low | High | Test with existing build command |

## Verification Strategy

1. **Unit tests:** Parse sample YAML, verify command output
2. **Integration test:** Run `aether caste-models list`, verify output matches YAML
3. **End-to-end:** Run `/ant:verify-castes`, confirm each caste reports correct model
4. **Regression:** Run `/ant:build 1`, verify workers spawn successfully

## Open Questions

1. **Model alias format:** Should we enforce provider/model format or keep shorthand?
   - Current: `kimi-k2.5`
   - LiteLLM expects: `kimi/k2.5`
   - Recommendation: Keep shorthand in profiles, map to LiteLLM format internally

2. **Profile persistence:** Should overrides be per-repo or global?
   - Current: `model-profiles.yaml` is per-repo
   - Option: Add `~/.aether/model-profiles.yaml` for global defaults
   - Recommendation: Keep per-repo for now

3. **Task-based routing:** The YAML has `task_routing` section with keyword detection. Should this be implemented?
   - Current: Documented but not implemented
   - Recommendation: Out of scope for Phase 9, document as Phase 11 prerequisite

## Sources

### Primary (HIGH confidence)
- `.aether/model-profiles.yaml` - caste-to-model mappings
- `.aether/aether-utils.sh` lines 1561-1630 - model-profile commands
- `.aether/utils/spawn-with-model.sh` - model assignment logic
- `bin/cli.js` lines 1527-1573 - existing verify-models command
- `bin/lib/model-verify.js` - verification library

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` lines 319-360 - intended spawn flow
- `.aether/workers.md` lines 53-117 - model-aware spawning documentation
- `.aether/dreams/2026-02-14-0238.md` Dream 3 - critical finding on unverified routing

### Tertiary (LOW confidence)
- LiteLLM proxy documentation (not verified in this research)
- Claude Code Task tool environment inheritance (observed behavior, not documented)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - existing code patterns
- Architecture: HIGH - clear patterns in existing commands
- Pitfalls: MEDIUM - based on observed behavior, not all verified

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable CLI patterns)
