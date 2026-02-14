# Phase 11: Foraging Specialization - Research

**Researched:** 2026-02-14
**Domain:** Task-based model routing, keyword detection, performance telemetry, CLI overrides
**Confidence:** HIGH

## Summary

This research investigates how to implement intelligent task-based model routing that goes beyond caste-based assignment. The system will analyze task descriptions for keywords ("design" → glm-5, "implement" → kimi), track model performance telemetry, and support per-command model overrides via `--model` flag.

**Key Finding:** The infrastructure from Phase 9 provides a solid foundation. The `model-profiles.yaml` already contains a `task_routing` section with keyword mappings that has never been implemented. The spawn logging system (Phase 9 Plan 4) already tracks models per spawn. What's needed: (1) keyword matching logic, (2) telemetry storage and analysis, (3) CLI argument parsing for `--model` override.

**Primary recommendation:** Extend the existing `model-profiles.js` library with task analysis functions, add telemetry storage to `.aether/data/telemetry.json`, and integrate routing decisions into the spawn flow.

## Current State Analysis

### What Exists (from Phase 9)

| Component | Location | Status | Notes |
|-----------|----------|--------|-------|
| Task routing config | `.aether/model-profiles.yaml` lines 67-96 | Unimplemented | `task_routing.complexity_indicators` defines keywords → model mappings |
| Model profiles library | `bin/lib/model-profiles.js` | Complete | Caste-based routing fully functional |
| Spawn logging | `bin/lib/spawn-logger.js` | Complete | Records model per spawn in spawn-tree.txt |
| CLI commands | `bin/cli.js` | Complete | `caste-models` commands work, override system functional |
| Activity logging | `bin/lib/logger.js` | Complete | Structured logging with activity types |

### Task Routing Configuration (Already Defined)

```yaml
# From .aether/model-profiles.yaml lines 67-96
task_routing:
  default_model: kimi-k2.5
  complexity_indicators:
    complex:
      keywords:
        - design
        - architecture
        - plan
        - coordinate
        - synthesize
        - strategize
        - optimize
      model: glm-5
    simple:
      keywords:
        - implement
        - code
        - refactor
        - write
        - create
      model: kimi-k2.5
    validate:
      keywords:
        - test
        - validate
        - verify
        - check
        - review
        - audit
      model: minimax-2.5
```

### What's Missing for Phase 11

| Requirement | ID | Gap |
|-------------|-----|-----|
| Keyword pattern matching | MOD-06 | No function to analyze task descriptions against keyword lists |
| Task-based model selection | MOD-06 | Spawn flow doesn't check task content before selecting model |
| Performance telemetry tracking | MOD-07 | No storage or tracking of success rates per model |
| Per-command model override | MOD-08 | No `--model` CLI flag parsing in spawn commands |
| Telemetry query/analytics | MOD-07 | No commands to view or analyze model performance |

## Standard Stack

### Core (Already in Use)
| Library | Version | Purpose | Status |
|---------|---------|---------|--------|
| js-yaml | ^4.x | YAML parsing | Already used |
| Commander.js | ^11.x | CLI framework | Already used |
| Node.js fs | built-in | File I/O | Already used |

### No New Dependencies Required
The implementation can use existing stack. Optional enhancements:
- `minimatch` or `micromatch` - If glob pattern matching needed (not recommended for keywords)

## Architecture Patterns

### Pattern 1: Keyword Matching Strategy

**What:** Match task descriptions against keyword lists to determine complexity

**Options Considered:**

| Approach | Pros | Cons | Recommendation |
|----------|------|------|----------------|
| Exact word match | Fast, predictable | Misses variations ("designs", "designing") | Not recommended |
| Substring match | Catches variations | False positives ("design" matches "redesign") | **Use this** |
| Regex word boundaries | Precise control | Complex to maintain | Overkill |
| Case-insensitive | User-friendly | Slightly slower | **Always use** |

**Recommended Implementation:**
```javascript
// Source: Pattern from existing codebase (aether-utils.sh learning-inject)
function matchTaskKeywords(taskDescription, keywordList) {
  const normalized = taskDescription.toLowerCase();
  return keywordList.some(keyword => normalized.includes(keyword.toLowerCase()));
}
```

### Pattern 2: Model Selection Precedence

**Priority Order (highest to lowest):**

1. **Explicit `--model` flag** (MOD-08) - User override for this command only
2. **User override from `user_overrides`** (Phase 9) - Persistent caste override
3. **Task-based routing** (MOD-06) - Keywords in task description
4. **Caste default** (Phase 9) - `worker_models` in YAML
5. **Global fallback** - `kimi-k2.5`

```javascript
// Source: Extension of existing getEffectiveModel() pattern
function selectModelForTask(profiles, caste, taskDescription, cliOverride = null) {
  // 1. CLI override takes highest precedence
  if (cliOverride && validateModel(profiles, cliOverride).valid) {
    return { model: cliOverride, source: 'cli-override' };
  }

  // 2. Check user_overrides (existing logic)
  const userOverride = profiles.user_overrides?.[caste];
  if (userOverride) {
    return { model: userOverride, source: 'user-override' };
  }

  // 3. Task-based routing (NEW)
  if (taskDescription && profiles.task_routing) {
    const taskModel = getModelForTask(profiles.task_routing, taskDescription);
    if (taskModel) {
      return { model: taskModel, source: 'task-routing' };
    }
  }

  // 4. Caste default (existing logic)
  const casteModel = profiles.worker_models?.[caste];
  if (casteModel) {
    return { model: casteModel, source: 'caste-default' };
  }

  // 5. Fallback
  return { model: DEFAULT_MODEL, source: 'fallback' };
}
```

### Pattern 3: Telemetry Storage

**Storage Location:** `.aether/data/telemetry.json`

**Schema:**
```json
{
  "version": "1.0",
  "last_updated": "2026-02-14T17:30:00Z",
  "models": {
    "kimi-k2.5": {
      "total_spawns": 150,
      "successful_completions": 142,
      "failed_completions": 5,
      "blocked": 3,
      "success_rate": 0.947,
      "by_caste": {
        "builder": { "spawns": 80, "success": 76, "failures": 4 },
        "watcher": { "spawns": 70, "success": 66, "failures": 1, "blocked": 3 }
      },
      "by_task_type": {
        "implement": { "spawns": 60, "success": 58 },
        "code": { "spawns": 40, "success": 38 }
      }
    },
    "glm-5": {
      "total_spawns": 45,
      "successful_completions": 43,
      "failed_completions": 2,
      "success_rate": 0.956
    }
  },
  "routing_decisions": [
    {
      "timestamp": "2026-02-14T17:30:00Z",
      "task": "Design authentication system",
      "caste": "architect",
      "selected_model": "glm-5",
      "source": "task-routing",
      "keywords_matched": ["design"]
    }
  ]
}
```

**Why JSON:**
- Human-readable for debugging
- Easy to query with `jq` or Node.js
- Atomic writes using temp file + rename pattern
- Append-only for routing decisions (rotated at 1000 entries)

### Pattern 4: CLI Argument Parsing for `--model`

**Integration Points:**

The `--model` flag needs to be parsed in commands that spawn workers:

1. **Direct CLI commands** (e.g., `aether spawn-worker` if added)
2. **Slash commands** that spawn (e.g., `/ant:build`)

**Commander.js Pattern:**
```javascript
// From existing bin/cli.js patterns
program
  .command('spawn-worker')
  .description('Spawn a worker ant')
  .option('-m, --model <model>', 'Override model for this spawn')
  .option('-c, --caste <caste>', 'Worker caste')
  .action(wrapCommand(async (options) => {
    const model = options.model; // CLI override
    const caste = options.caste;
    // Pass model to spawn logic
  }));
```

**Slash Command Pattern:**
```markdown
---
name: ant:build
description: "Build a phase"
---

You are the **Queen**. Parse arguments for flags:

1. Extract phase number from $ARGUMENTS
2. Check for `--model <model>` flag in remaining arguments
3. If found, pass model override to spawn logic
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Keyword stemming | Porter stemmer or NLP library | Simple substring matching | Overkill for this use case, adds complexity |
| Complex pattern matching | Regex engine | Substring + word boundaries | Keywords are simple, regex adds maintenance burden |
| Time-series database | InfluxDB, Prometheus | JSON file with rotation | Telemetry is small-scale, file-based is sufficient |
| Statistical analysis | Custom algorithms | Simple success rate calculation | Don't need complex stats for basic telemetry |
| CLI argument parsing | Custom parser | Commander.js | Already in codebase, well-tested |

**Key insight:** The task routing is an enhancement, not a replacement. Simple substring matching on keywords is sufficient - we don't need NLP or ML for this use case.

## Common Pitfalls

### Pitfall 1: Keyword Collision

**What goes wrong:** Task "Redesign the authentication system" matches both "design" (complex → glm-5) and simple caste default (builder → kimi-k2.5), causing inconsistent routing.

**Why it happens:** Substring matching is greedy - "redesign" contains "design".

**How to avoid:**
- Use word boundary detection: `/\bdesign\b/i` instead of simple substring
- OR accept that substring matching is "good enough" and document the limitation
- Log which keywords were matched for debugging

**Warning signs:** Same task description routes to different models on different runs.

### Pitfall 2: Telemetry File Growth

**What goes wrong:** `telemetry.json` grows unbounded, causing slow reads and potential corruption.

**Why it happens:** Routing decisions are appended indefinitely.

**How to avoid:**
- Rotate routing_decisions array at 1000 entries (keep last N)
- Aggregate old data into summary statistics before rotation
- Use streaming writes for new decisions (append to file, don't rewrite entire JSON)

**Warning signs:** File size > 1MB, slow CLI commands, JSON parse errors.

### Pitfall 3: Override Precedence Confusion

**What goes wrong:** Users set `--model glm-5` but worker uses kimi-k2.5 because user_overrides takes precedence, or vice versa.

**Why it happens:** Unclear precedence rules or bugs in precedence logic.

**How to avoid:**
- Document precedence clearly in help text
- Log the source of model selection ("Using glm-5 from CLI override")
- Provide `aether caste-models why <caste>` command to explain routing decision

**Warning signs:** User complaints about overrides not working, inconsistent behavior.

### Pitfall 4: Telemetry Without Context

**What goes wrong:** Telemetry shows glm-5 has 50% success rate, but it's being used for the hardest tasks. Conclusion: glm-5 is worse, when actually it's handling harder work.

**Why it happens:** Raw success rates don't account for task complexity.

**How to avoid:**
- Track task type/complexity in telemetry
- Compare success rates within same task type
- Use caste as proxy for complexity (architect tasks are harder than builder)

**Warning signs:** Misleading analytics, wrong model selection based on skewed data.

## Code Examples

### Task Analysis Function

```javascript
// Source: New function for model-profiles.js
/**
 * Analyze task description and return appropriate model based on keywords
 * @param {object} taskRouting - task_routing section from profiles
 * @param {string} taskDescription - Task description to analyze
 * @returns {string|null} Model name or null if no match
 */
function getModelForTask(taskRouting, taskDescription) {
  if (!taskRouting || !taskDescription) {
    return null;
  }

  const normalizedTask = taskDescription.toLowerCase();

  // Check each complexity indicator
  for (const [complexity, config] of Object.entries(taskRouting.complexity_indicators || {})) {
    const keywords = config.keywords || [];
    const hasMatch = keywords.some(keyword =>
      normalizedTask.includes(keyword.toLowerCase())
    );

    if (hasMatch) {
      return config.model;
    }
  }

  // No keyword match - return default
  return taskRouting.default_model || null;
}
```

### Telemetry Recording

```javascript
// Source: New telemetry.js module
const fs = require('fs');
const path = require('path');

const TELEMETRY_FILE = 'telemetry.json';
const MAX_ROUTING_DECISIONS = 1000;

function recordSpawnTelemetry(repoPath, spawnInfo) {
  const telemetryPath = path.join(repoPath, '.aether', 'data', TELEMETRY_FILE);

  // Load existing or create new
  let telemetry = { version: '1.0', models: {}, routing_decisions: [] };
  if (fs.existsSync(telemetryPath)) {
    try {
      telemetry = JSON.parse(fs.readFileSync(telemetryPath, 'utf8'));
    } catch (e) {
      // Corrupted - start fresh
    }
  }

  // Update model stats
  const model = spawnInfo.model || 'unknown';
  if (!telemetry.models[model]) {
    telemetry.models[model] = {
      total_spawns: 0,
      successful_completions: 0,
      failed_completions: 0,
      blocked: 0,
      by_caste: {},
      by_task_type: {}
    };
  }

  telemetry.models[model].total_spawns++;

  // Record routing decision
  telemetry.routing_decisions.push({
    timestamp: new Date().toISOString(),
    task: spawnInfo.task,
    caste: spawnInfo.caste,
    selected_model: model,
    source: spawnInfo.source // 'cli-override', 'task-routing', 'caste-default', etc.
  });

  // Rotate if needed
  if (telemetry.routing_decisions.length > MAX_ROUTING_DECISIONS) {
    telemetry.routing_decisions = telemetry.routing_decisions.slice(-MAX_ROUTING_DECISIONS);
  }

  // Atomic write
  const tempPath = telemetryPath + '.tmp';
  fs.writeFileSync(tempPath, JSON.stringify(telemetry, null, 2));
  fs.renameSync(tempPath, telemetryPath);
}
```

### CLI Override Parsing

```javascript
// Source: Pattern for build.md and other spawn commands
function parseBuildArguments(args) {
  const parts = args.trim().split(/\s+/);
  const phaseNumber = parts[0];
  const flags = {};

  for (let i = 1; i < parts.length; i++) {
    if (parts[i] === '--model' || parts[i] === '-m') {
      flags.model = parts[i + 1];
      i++; // Skip next as it's the value
    } else if (parts[i] === '--verbose' || parts[i] === '-v') {
      flags.verbose = true;
    }
  }

  return { phaseNumber, flags };
}

// Usage in spawn logic
const { phaseNumber, flags } = parseBuildArguments($ARGUMENTS);
const modelOverride = flags.model || null;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Caste-only routing | Task-aware routing | Phase 11 | Better model utilization based on actual work |
| No telemetry | Performance tracking | Phase 11 | Data-driven model selection improvements |
| No CLI override | `--model` flag support | Phase 11 | User can force specific model when needed |
| Static assignments | Dynamic keyword matching | Phase 11 | Automatic adaptation to task content |

**Deprecated/outdated:**
- Relying solely on caste for model selection (still supported but enhanced)
- Manual model specification via environment variables (use `--model` instead)

## Integration Points

### With Existing Code

| Existing Component | Integration | Change Required |
|-------------------|-------------|-----------------|
| `bin/lib/model-profiles.js` | Add `getModelForTask()` function | Extend - new function |
| `bin/lib/model-profiles.js` | Add `selectModelForTask()` with precedence | Extend - new function |
| `bin/lib/spawn-logger.js` | Record telemetry on spawn | Extend - call telemetry module |
| `bin/cli.js` | Add `telemetry` command | Extend - new command |
| `.claude/commands/ant/build.md` | Parse `--model` flag, pass to spawn | Modify - add flag parsing |
| `.aether/aether-utils.sh` | Add telemetry recording to spawn-log | Modify - add telemetry call |

### File Locations for New Code

| File | Purpose |
|------|---------|
| `bin/lib/telemetry.js` | Telemetry recording and querying |
| `bin/lib/task-router.js` | Keyword matching and task analysis (optional - could be in model-profiles.js) |

## Open Questions

1. **Keyword Expansion:** Should users be able to add custom keywords via config?
   - What we know: YAML structure supports adding new complexity_indicators
   - What's unclear: Should this be user-editable or system-defined?
   - Recommendation: Start with system-defined, add user config in future phase

2. **Telemetry Retention:** How long to keep detailed routing decisions?
   - What we know: 1000 entries is ~100KB, reasonable size
   - What's unclear: Do we need historical analytics beyond recent decisions?
   - Recommendation: Keep 1000 recent, aggregate older data into model stats

3. **Success Definition:** What constitutes a "successful" spawn?
   - What we know: Spawns can complete, fail, or be blocked
   - What's unclear: Should we track task completion quality or just spawn status?
   - Recommendation: Track spawn status only (completed/failed/blocked), quality is subjective

4. **Task Type Inference:** Should we categorize tasks beyond keyword matching?
   - What we know: Keywords can indicate complexity but not task type
   - What's unclear: Would task type analytics be useful?
   - Recommendation: Start with keywords, add task type if telemetry shows need

## Implementation Approach

### MOD-06: Task-based routing

**Files to modify:**
1. `bin/lib/model-profiles.js` - Add `getModelForTask()` and `selectModelForTask()`
2. `.claude/commands/ant/build.md` - Parse task description, call routing function
3. `.aether/aether-utils.sh` - Add task parameter to spawn-log

**Algorithm:**
1. Normalize task description (lowercase)
2. Check each complexity indicator's keywords
3. Return first matching model or default
4. Log routing decision source

### MOD-07: Model performance telemetry

**Files to create/modify:**
1. `bin/lib/telemetry.js` - New module for telemetry operations
2. `bin/cli.js` - Add `aether telemetry` command with subcommands
3. `bin/lib/spawn-logger.js` - Integrate telemetry recording

**Data flow:**
1. On spawn: record model, caste, task, routing source
2. On completion: update success/failure counts
3. CLI command: read and display aggregated stats

### MOD-08: Per-command model override

**Files to modify:**
1. `.claude/commands/ant/build.md` - Parse `--model` flag from $ARGUMENTS
2. Other spawn commands - Add similar parsing
3. `bin/lib/model-profiles.js` - Accept cliOverride parameter

**Precedence:**
```
--model flag > user_overrides > task_routing > caste_default > fallback
```

## Verification Strategy

1. **Unit tests:**
   - Keyword matching with various task descriptions
   - Precedence logic with all override types
   - Telemetry aggregation math

2. **Integration tests:**
   - Spawn worker with task containing "design" → verify glm-5 selected
   - Spawn with `--model` flag → verify override works
   - Run multiple spawns → verify telemetry accumulates

3. **End-to-end:**
   - `/ant:build 1 --model glm-5` → verify all workers use glm-5
   - Check telemetry after build → verify stats recorded

## Sources

### Primary (HIGH confidence)
- `.aether/model-profiles.yaml` lines 67-96 - task_routing configuration
- `bin/lib/model-profiles.js` - existing routing logic and patterns
- `bin/lib/spawn-logger.js` - spawn tracking with model field
- `bin/cli.js` - CLI command patterns with Commander.js

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` - spawn flow and argument handling
- `.aether/aether-utils.sh` lines 320-338 - spawn-log command with model parameter
- `bin/lib/logger.js` - activity logging patterns

### Tertiary (LOW confidence)
- Research on keyword matching algorithms (general knowledge)
- Telemetry best practices (industry patterns)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in use
- Architecture: HIGH - clear extension of existing patterns
- Pitfalls: MEDIUM - some based on general experience, not all verified

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable patterns)
