---
phase: 06-autonomous-emergence
plan: 01
subsystem: autonomous-spawning
tags: [capability-detection, specialist-mapping, spawn-scoring, bash, jq, worker-ants]

# Dependency graph
requires:
  - phase: 05-phase-boundaries
    provides: state machine, pheromone system, worker ant castes
provides:
  - Capability gap detection logic via spawn-decision.sh
  - Multi-factor spawn scoring formula (threshold: 0.6)
  - Specialist type mapping with semantic fallback
  - Worker ant prompts with capability assessment workflow
affects: [06-02-resource-constraints, 06-03-spawn-protocol, 06-04-meta-learning]

# Tech tracking
tech-stack:
  added: [spawn-decision.sh utility]
  patterns:
    - Multi-factor spawn scoring: gap_score (40%), priority (20%), load (15%), budget (15%), resources (10%)
    - Capability taxonomy: technical domains, frameworks, skills
    - Specialist mapping: keyword lookup with semantic analysis fallback
    - Spawn decision threshold: 0.6 (spawn if score >= threshold)

key-files:
  created:
    - .aether/utils/spawn-decision.sh - Capability gap detection and specialist selection functions
  modified:
    - .aether/workers/builder-ant.md - Added Capability Gap Detection section
    - .aether/workers/colonizer-ant.md - Added Capability Gap Detection section
    - .aether/workers/route-setter-ant.md - Added Capability Gap Detection section
    - .aether/workers/scout-ant.md - Added Capability Gap Detection section
    - .aether/workers/watcher-ant.md - Added Capability Gap Detection section
    - .aether/workers/architect-ant.md - Added Capability Gap Detection section

key-decisions:
  - "Multi-factor scoring weights: gap_score (40%), priority (20%), load (15%), budget (15%), resources (10%)"
  - "Spawn threshold: 0.6 (balances autonomous action with resource conservation)"
  - "Specialist mapping uses direct keyword lookup first, semantic analysis as fallback"
  - "Capability gap detection section placed BEFORE existing Autonomous Spawning section in all workers"

patterns-established:
  - "5-step capability assessment: Extract requirements → Compare to own capabilities → Identify gaps → Calculate spawn score → Map to specialist"
  - "All 6 Worker Ants follow identical spawn decision workflow with caste-specific capabilities"
  - "Functions output JSON for programmatic use in spawn decision logic"

# Metrics
duration: 4min
completed: 2026-02-01
---

# Phase 6 Plan 1: Capability Gap Detection with Worker Spawns Workers Summary

**Multi-factor spawn scoring (threshold 0.6) with specialist type mapping using semantic analysis fallback, enabling Worker Ants to autonomously detect capability gaps and select appropriate specialist castes**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-01T18:31:07Z
- **Completed:** 2026-02-01T18:35:03Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Created spawn-decision.sh with 5 functions for capability analysis and specialist selection
- Added "Capability Gap Detection" section to all 6 Worker Ant prompts with caste-specific capabilities
- Implemented multi-factor spawn scoring formula (gap_score × 0.40 + priority × 0.20 + load × 0.15 + budget_remaining × 0.15 + resources × 0.10)
- Established specialist type mapping using worker_ants.json with semantic fallback for novel capabilities

## Task Commits

Each task was committed atomically:

1. **Task 1: Create spawn-decision.sh with capability gap detection functions** - `2e4fd08` (feat)
2. **Task 2: Add capability gap detection to all 6 Worker Ant prompts** - `33b1cbf` (feat)

**Plan metadata:** (pending final metadata commit)

## Files Created/Modified

- `.aether/utils/spawn-decision.sh` - Capability gap detection and specialist selection logic (analyze_task_requirements, compare_capabilities, detect_capability_gaps, calculate_spawn_score, map_gap_to_specialist)
- `.aether/workers/builder-ant.md` - Added Capability Gap Detection section with builder capabilities (code_implementation, command_execution, file_operations, testing_setup, build_automation)
- `.aether/workers/colonizer-ant.md` - Added Capability Gap Detection section with colonizer capabilities (codebase_analysis, semantic_indexing, pattern_detection, dependency_mapping, architecture_understanding)
- `.aether/workers/route-setter-ant.md` - Added Capability Gap Detection section with route_setter capabilities (phase_planning, task_breakdown, dependency_analysis, resource_allocation, route_optimization)
- `.aether/workers/scout-ant.md` - Added Capability Gap Detection section with scout capabilities (information_gathering, documentation_search, context_retrieval, external_research, domain_knowledge)
- `.aether/workers/watcher-ant.md` - Added Capability Gap Detection section with watcher capabilities (validation, testing, quality_checks, security_review, performance_analysis)
- `.aether/workers/architect-ant.md` - Added Capability Gap Detection section with architect capabilities (memory_compression, pattern_extraction, knowledge_synthesis, associative_linking, long_term_storage)

## Decisions Made

- **Spawn threshold set to 0.6**: Balances autonomous action (spawning when needed) with resource conservation (not over-spawning). Threshold based on multi-factor scoring where gap_score has highest weight (40%) as primary driver.
- **Specialist mapping uses hybrid approach**: Direct keyword lookup from worker_ants.json specialist_mappings.capability_to_caste for known patterns, with semantic analysis as fallback for novel capability gaps.
- **Capability detection section placement**: Inserted BEFORE existing "Autonomous Spawning" section to ensure gap analysis happens before spawning protocol.
- **All workers use identical workflow**: 5-step process (Extract → Compare → Identify → Calculate → Map) ensures consistent spawn decision logic across castes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed atomic-write.sh sourcing path in spawn-decision.sh**
- **Found during:** Task 1 (spawn-decision.sh creation)
- **Issue:** SCRIPT_DIR resolution failed when sourcing atomic-write.sh, causing "no such file or directory" error
- **Fix:** Added AETHER_ROOT environment variable detection with fallback to relative path from current directory
- **Files modified:** .aether/utils/spawn-decision.sh
- **Verification:** All functions load correctly, atomic-write functions available
- **Committed in:** 2e4fd08 (Task 1 commit)

**2. [Rule 1 - Bug] Corrected spawn score calculation verification**
- **Found during:** Task 1 (testing calculate_spawn_score)
- **Issue:** Plan's expected value (0.68) was incorrect. Actual formula result (0.73) is mathematically correct: 0.8×0.40 + 0.9×0.20 + 0.3×0.15 + 0.7×0.15 + 0.8×0.10 = 0.73
- **Fix:** Updated test expectation to match correct calculation
- **Files modified:** None (documentation fix in test verification)
- **Verification:** Formula produces mathematically correct results
- **Committed in:** 2e4fd08 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both auto-fixes necessary for correctness. No scope creep.

## Issues Encountered

- **Shell variable persistence**: Plan start time shell variables didn't persist across tool calls. Workaround: calculated duration at end of execution manually.
- **Bash array declaration conflict**: `declare -a functions` conflicted with autoloaded parameter. Workaround: used simple for loop without explicit array declaration.

## Verification Results

### Overall Verification Checks

1. **spawn-decision.sh exists and works**
   - Script is executable
   - All 5 functions defined: analyze_task_requirements, compare_capabilities, detect_capability_gaps, calculate_spawn_score, map_gap_to_specialist

2. **Worker Ant prompts updated**
   - All 6 workers have capability gap detection section
   - Section content is caste-specific (correct capabilities from worker_ants.json)
   - Multi-factor scoring formula documented with threshold (0.6)
   - Specialist mapping table included

3. **Integration verified**
   - spawn-decision.sh can be sourced in worker ant prompts
   - Capability gap detection flows into existing autonomous spawning section
   - Resource constraints mentioned (will be implemented in Wave 2)

### Function Testing Results

- analyze_task_requirements: Returns JSON array of capabilities from task description
- compare_capabilities: Identifies gaps between required and available capabilities, calculates coverage percentage
- detect_capability_gaps: Returns "spawn" decision when gaps exist or failures occurred
- calculate_spawn_score: Produces mathematically correct multi-factor scores
- map_gap_to_specialist: Maps capability gaps to specialist castes using keyword lookup and semantic fallback

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**Ready for 06-02 (Resource Constraints):**
- spawn-decision.sh provides calculate_spawn_score function that outputs score for resource constraint checking
- Worker Ant prompts have spawn decision threshold (0.6) documented
- Next plan will implement resource budget tracking (max 10 spawns per phase, depth limit 3) and circuit breaker pattern (3 failed spawns → cooldown)

**Ready for 06-03 (Spawn Protocol):**
- Capability gap detection logic in place for spawn triggering
- Specialist type mapping established for subagent type selection
- Next plan will implement actual spawning mechanism with inherited context structure

**Ready for 06-04 (Meta-Learning):**
- spawn-decision.sh has placeholder for pattern recognition in detect_capability_gaps
- Next plan will integrate Bayesian confidence scoring from spawn_history

**No blockers or concerns.**

---
*Phase: 06-autonomous-emergence*
*Completed: 2026-02-01*
