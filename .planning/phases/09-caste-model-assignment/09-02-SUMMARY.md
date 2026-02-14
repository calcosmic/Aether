---
phase: 09-caste-model-assignment
plan: 02
subsystem: cli
completed: 2026-02-14
duration: 45m
tags: [cli, model-routing, yaml, overrides]
requires:
  - 09-01
provides:
  - caste-models CLI command
  - user override persistence
  - model assignment viewing
affects:
  - 09-03
  - 09-04
tech-stack:
  added: []
  patterns:
    - CLI subcommands with commander.js
    - YAML persistence with js-yaml
    - ValidationError for user input
key-files:
  created:
    - tests/unit/model-profiles-overrides.test.js
  modified:
    - bin/lib/model-profiles.js
    - bin/cli.js
decisions:
  - id: D09-02-001
    text: Use user_overrides section in model-profiles.yaml for persistence
    rationale: Keeps all model configuration in one file, clear separation from defaults
  - id: D09-02-002
    text: Show (override) indicator in list output for transparency
    rationale: Users need to see which models are overridden vs defaults
  - id: D09-02-003
    text: Include caste emojis in CLI output for visual recognition
    rationale: Matches ant colony metaphor and improves scannability
---

# Phase 9 Plan 2: Caste Models CLI Commands Summary

## One-Liner

CLI commands for viewing and modifying caste-to-model assignments with persistence in model-profiles.yaml user_overrides section.

## What Was Built

### 1. User Override Functions (bin/lib/model-profiles.js)

Added 4 new functions to the model-profiles library:

- **setModelOverride(repoPath, caste, model)** - Sets user override with validation
  - Validates caste exists in worker_models
  - Validates model exists in model_metadata
  - Persists to user_overrides section in YAML
  - Returns {success: true, previous: string|null}

- **resetModelOverride(repoPath, caste)** - Removes user override
  - Validates caste exists
  - Removes from user_overrides section
  - Cleans up empty user_overrides section
  - Returns {success: true, hadOverride: boolean}

- **getEffectiveModel(profiles, caste)** - Returns effective model with source
  - Checks user_overrides first
  - Falls back to worker_models default
  - Final fallback to 'kimi-k2.5'
  - Returns {model: string, source: 'override'|'default'|'fallback'}

- **getUserOverrides(profiles)** - Returns current overrides object
  - Returns empty object if no overrides
  - Returns all overrides if present

### 2. CLI Commands (bin/cli.js)

Added `caste-models` command with 3 subcommands:

- **`aether caste-models list`** - Display table of assignments
  - Shows caste emoji, model, provider, context window, status
  - Indicates overrides with "(override)" suffix
  - Lists active overrides summary at bottom

- **`aether caste-models set <caste=model>`** - Set override
  - Validates caste and model before writing
  - Shows previous value when updating
  - Helpful error messages with valid options

- **`aether caste-models reset <caste>`** - Remove override
  - Validates caste exists
  - Shows confirmation message
  - No-op if no override existed

### 3. Unit Tests (tests/unit/model-profiles-overrides.test.js)

18 comprehensive tests covering:

- setModelOverride: success, update existing, invalid caste, invalid model, creates section
- resetModelOverride: success, no override existed, removes empty section, invalid caste
- getEffectiveModel: override, default, fallback, null profiles
- getUserOverrides: empty, with overrides, null profiles
- Integration: full set/reset workflow

## Commands

```bash
# View current assignments
aether caste-models list

# Set override (persists to YAML)
aether caste-models set builder=glm-5

# Reset to default (removes from YAML)
aether caste-models reset builder

# Get help
aether caste-models --help
```

## Example Output

```
Caste Model Assignments

Caste          Model          Provider   Context  Status
────────────────────────────────────────────────────────────
🏛️ Prime      glm-5          z_ai       200K     ✓
🔨 Builder     glm-5 (override) z_ai       200K     ✓
👁️ Watcher    kimi-k2.5      kimi       256K     ✓
...

Active overrides: 1
  builder: glm-5
```

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| D09-02-001 | Use user_overrides section in model-profiles.yaml | Keeps all model configuration in one file, clear separation from defaults |
| D09-02-002 | Show (override) indicator in list output | Users need to see which models are overridden vs defaults |
| D09-02-003 | Include caste emojis in CLI output | Matches ant colony metaphor and improves scannability |

## Test Results

```
✔ 18 new tests for override functions (all pass)
✔ 28 existing model-profiles tests (all pass)
✔ 46 total model-profiles related tests
```

## Files Changed

| File | Changes |
|------|---------|
| bin/lib/model-profiles.js | +157 lines - 4 new functions |
| bin/cli.js | +180 lines - caste-models command with subcommands |
| tests/unit/model-profiles-overrides.test.js | +433 lines - 18 unit tests |

## Deviations from Plan

None - plan executed exactly as written.

## Next Phase Readiness

- ✅ MOD-01: View model assignments per caste - COMPLETE
- ✅ MOD-02: Override model for specific caste - COMPLETE
- ⏳ MOD-03: Verify LiteLLM proxy health - Next (09-03)
- ⏳ MOD-04: Show provider routing info - Partial (shown in list)
- ⏳ MOD-05: Log actual model used per spawn - Pending (09-04)

## Commits

- `3092165` feat(09-02): add user override functions to model-profiles.js
- `51e7051` feat(09-02): add caste-models CLI command with subcommands
- `fbcffe8` test(09-02): add unit tests for override functions
