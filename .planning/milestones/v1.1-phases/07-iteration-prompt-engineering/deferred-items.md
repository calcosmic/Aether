# Deferred Items - Phase 07

## Pre-existing Test Failure

- **File:** tests/unit/context-continuity.test.js:165
- **Test:** `pheromone-prime --compact respects max signal limit`
- **Issue:** Assertion `out.result.prompt_section.includes('COMPACT SIGNALS')` returns false
- **Discovered during:** 07-02 plan execution (npm test)
- **Not caused by:** Phase 07 changes (oracle phase transitions or tests)
