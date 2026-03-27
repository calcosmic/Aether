# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-27)

**Core value:** Reliably interpret user requests, decompose into work, verify outputs, and ship correct work with minimal back-and-forth.
**Current focus:** Phase 23 -- Tooling Overrides (v2.3)

## Current Position

Phase: 23 of 24 (Tooling Overrides)
Plan: 01 of 2
Status: Complete
Last activity: 2026-03-27 -- 23-01 complete (slot resolution and validation functions)

Progress: [======    ] 50%

## Performance Metrics

**Velocity (from v2.1):**
- Total plans completed: 49
- Average duration: 5min
- Total execution time: 4.0 hours

**Recent Trend:**
- v2.1 completed 8 phases, 39 plans in ~3 hours
- v2.2 completed 4 phases, 5 plans
- v2.3: 4 phases planned, Phase 21 complete, Phase 22 complete (3/3 plans), Phase 23 in progress (1/2 plans: slot functions)

*Updated after each plan completion*

## Accumulated Context

### Decisions

- [v2.3]: Phase 1 must complete before any model-profiles.yaml changes -- 184 hardcoded model names in tests will break otherwise
- [v2.3]: Use Approach A (agent frontmatter) for MVP routing -- simpler than Task tool model param, zero playbook changes needed
- [v2.3]: Aether routes by slot name (opus/sonnet), never by actual model name -- keeps dual-mode support clean
- [v2.3]: Specialist analysis castes on opus (Tracker, Auditor, Gatekeeper, Measurer) -- reasoning depth improves accuracy on bounded analysis tasks
- [v2.2]: Focus exclusively on wisdom systems -- ceremony/verification improvements deferred to v2.3
- [v2.3/21-01]: No caching in mock-profiles helper -- each function call reads fresh YAML so tests break intentionally if YAML changes
- [v2.3/21-01]: buildMockProfiles uses spread merge for workerModels/modelMetadata, full replacement for taskRouting
- [v2.3/21-02]: Soft-gate regression test logs violations without failing -- avoids false positives from legitimate test inputs
- [v2.3/21-02]: Module-level YAML-derived constants (BUILDER_MODEL, ALT_MODEL) at file top avoid repeated helper calls
- [v2.3/21-03]: Loop-based provider verification in integration tests -- automatically covers new models added to YAML
- [v2.3/22-01]: keeper placed on inherit tier (3 inherit castes: chronicler, includer, keeper) -- CONTEXT.md listed only 2, needs update
- [v2.3/22-01]: Slot-based worker_models: castes store slot names (opus/sonnet/inherit), model_slots section provides resolution table
- [v2.3/22-02]: Agent frontmatter model: field activates Claude Code native routing -- 8 opus, 11 sonnet, 3 inherit across 22 agents
- [v2.3/22-03]: OpenCode verify-castes mirror updated alongside Claude Code version for sync policy parity
- [v2.3/23-01]: getModelSlotForCaste returns DEFAULT_SLOT ('inherit') for missing castes -- silent fallback, no console warnings
- [v2.3/23-01]: validateSlot uses {valid, error} return pattern for centralized slot-name validation

### Pending Todos

None yet.

### Blockers/Concerns

- GLM-5 constraint passing through subagent spawning is unverified -- if Claude Code does not forward temperature/top_p/max_tokens, reasoning castes may loop on GLM-5
- Task tool `model` parameter vs frontmatter precedence untested -- if Task param does not override frontmatter, Approach B cannot coexist with Approach A
- Oracle and Architect castes lack dedicated agent files -- their work runs through Queen or direct CLI, which may use the wrong model

## Session Continuity

Last session: 2026-03-27
Stopped at: Completed 23-01 (slot resolution and validation functions)
Resume file: None
