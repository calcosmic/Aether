---
phase: 11-foraging-specialization
plan: 03
type: execute
subsystem: cli
tags: [telemetry, cli, model-performance, visualization]

requires:
  - 11-02 (Telemetry System)

provides:
  - CLI telemetry commands
  - Model performance visualization
  - Data-driven model selection insights

affects:
  - 12 (Colony Visualization - may use telemetry data)

tech-stack:
  added: []
  patterns:
    - Commander.js subcommands
    - Color-coded CLI output
    - Mock-based unit testing with proxyquire

key-files:
  created:
    - test/cli-telemetry.test.js
  modified:
    - bin/cli.js

decisions:
  - Default telemetry command shows summary (no subcommand required)
  - Color thresholds: green >=90%, yellow >=70%, red <70% success rate
  - Performance ranking sorts by success rate descending
  - Model detail view shows caste breakdown

metrics:
  duration: "45 minutes"
  completed: 2026-02-14
---

# Phase 11 Plan 03: Telemetry CLI Commands Summary

## One-liner
Added `aether telemetry` CLI command with summary, model, and performance subcommands for viewing model performance data.

## What Was Built

### CLI Commands

1. **`aether telemetry`** (default) - Shows overall telemetry summary
   - Total spawns and models used
   - Per-model performance with color-coded success rates
   - Recent routing decisions (last 5)

2. **`aether telemetry summary`** - Explicit summary command
   - Same output as default command

3. **`aether telemetry model <name>`** - Detailed model stats
   - Total spawns, success rate, completions/failures/blocked counts
   - Performance breakdown by caste

4. **`aether telemetry performance`** - Ranked performance table
   - Models sorted by success rate (highest first)
   - Columns: Rank, Model, Spawns, Success count, Rate
   - Color-coded rate column

### Color Coding

- **Green**: >= 90% success rate
- **Yellow**: >= 70% success rate
- **Red**: < 70% success rate

### Tests

Created `test/cli-telemetry.test.js` with 15 tests covering:
- Summary command with empty and populated data
- Model command with valid and invalid models
- Performance ranking functionality
- Color coding thresholds
- Edge cases (zero spawns, missing files)

## Deviations from Plan

None - plan executed exactly as written.

## Key Implementation Details

### Default Command Behavior
The telemetry command uses Commander.js's `.action()` on the parent command to provide a default behavior when no subcommand is specified. This matches user expectations from tools like `git status` or `npm ls`.

### Import Structure
```javascript
const {
  getTelemetrySummary,
  getModelPerformance,
} = require('./lib/telemetry');
```

### Output Formatting
- Uses existing color palette from `bin/lib/colors.js`
- Box-drawing characters for table separators (─)
- Fixed-width columns for alignment
- Emoji indicators for visual scanning

## Verification Results

All success criteria met:
- [x] `aether telemetry` shows overall summary
- [x] `aether telemetry model <name>` shows detailed model stats
- [x] `aether telemetry performance` shows ranked performance table
- [x] Output uses colors appropriately (green >90%, yellow 70-90%, red <70%)
- [x] Help text includes telemetry commands
- [x] All 15 tests pass

## Commits

1. `b0416b2` - feat(11-03): add telemetry command to CLI
2. `0b2c312` - test(11-03): add CLI telemetry tests

## Next Phase Readiness

This completes Phase 11 Plan 03. The telemetry CLI commands enable users to:
1. View model performance data collected by the telemetry system
2. Make data-driven decisions about model selection
3. Identify which models perform best for specific castes

This fulfills requirement MOD-07 from REQUIREMENTS.md for viewing telemetry data.
