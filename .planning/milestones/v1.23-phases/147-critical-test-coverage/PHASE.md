---
phase: 147
name: Critical Test Coverage
goal: The 12 most critical untested source files have test files proving their data-persistence paths work
milestone: v1.23
requirements: [TEST-01, TEST-02, TEST-03]
---

# Phase 147: Critical Test Coverage

## Success Criteria
1. Test files exist for all HIGH-severity source files (eventbus, queen, instinct, midden, hive, spawn, autopilot, flags, council, shelf)
2. Every public lifecycle command has at least one smoke or fixture test
3. A test matrix documents every command, its test file, and coverage type

## Plans

| Plan | Wave | Depends On | Focus |
|------|------|------------|-------|
| 147-01 | 1 | - | Core data-persistence: eventbus, queen, instinct, midden |
| 147-02 | 1 | - | Wisdom and spawn: hive, spawn |
| 147-03 | 2 | - | Orchestration: autopilot, council |
| 147-04 | 3 | 147-01, 147-02, 147-03 | Test matrix, lifecycle smoke tests, flags/shelf audit |

## Key Risks
- Event bus tests are timing-sensitive
- Queen/hive tests must redirect AETHER_HUB_DIR to avoid modifying ~/.aether/
- Cross-test pollution if saveGlobals/resetRootCmd missed
