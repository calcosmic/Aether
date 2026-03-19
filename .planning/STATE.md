# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-19)

**Core value:** The pheromone system should be a living system -- auto-emitting signals during builds, carrying context across sessions, and actually changing worker behavior -- not just a storage format that nobody reads.
**Current focus:** Phase 6 (next phase)

## Current Position

Phase: 5 of 8 (Learning Pipeline Validation) -- COMPLETE
Plan: 2 of 2 in current phase
Status: Phase complete
Last activity: 2026-03-19 -- Completed 05-02-PLAN.md

Progress: [███████░░░] 69%

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: 5min
- Total execution time: 0.62 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-data-purge | 1 | 3min | 3min |
| 02-command-audit-data-tooling | 2 | 12min | 6min |
| 03-pheromone-signal-plumbing | 2 | 8min | 4min |
| 04-pheromone-worker-integration | 2 | 7min | 3.5min |
| 05-learning-pipeline-validation | 2 | 7min | 3.5min |

**Recent Trend:**
- Last 5 plans: 4min, 3min, 4min, 3min, 4min
- Trend: stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Clean before integrating -- test data must be purged before pheromone integration can be validated
- [Roadmap]: XML exchange system should be ACTIVATED (wired into commands), not archived
- [Roadmap]: constraints.json is a legacy parallel store; eventual deprecation in favor of pheromones.json
- [01-02]: Force-added gitignored data files to commit purge changes for traceability
- [01-02]: Kept all 16 real worker spawn records in spawn-tree.txt
- [Phase 01]: Kept sig_feedback_001 despite 'Test coverage' text matching broad regex -- it is a real signal from worker_builder, not test data
- [Phase 01]: pheromones.json and constraints.json are gitignored -- cleaned locally but not committable to git
- [02-02]: Placed data-clean subcommand at end of case statement for minimal diff and clear separation
- [02-02]: Used atomic_write for file modifications when available, with direct write fallback
- [Phase 02-01]: Naming inconsistencies (help.md, memory-details.md, resume.md) documented as warnings not fixes -- frontmatter name does not affect slash command invocation
- [Phase 02-01]: Removed broken .aether/planning.md reference from plan.md -- inline rules already provided
- [03-01]: Replaced approx_epoch (365.25 days/year) with to_epoch (365 days/year) for consistency over precision
- [03-01]: Fixed jq // operator treating active:false as null -- used explicit if/elif chain instead
- [03-03]: Fixed same jq // active:false bug in pheromone-prime and context-capsule (discovered by injection chain tests)
- [03-03]: prompt_section groups signals by type (FOCUS, REDIRECT, FEEDBACK) not by strength -- test assertions adapted
- [04-01]: Placed pheromone_protocol after critical_rules, before return_format -- signals are critical but secondary to core rules like TDD
- [04-01]: Kept protocols under 35 lines using principle-based instructions (workers are LLMs, understand intent)
- [04-01]: Pre-existing lint:sync command count mismatch (38 vs 37) logged to deferred-items, not fixed (out of scope)
- [04-02]: Defined 'influence' as signal in prompt_section + agent has pheromone_protocol -- maximum verifiable without live LLM builds
- [04-02]: Reproduced build-wave.md Step 5.2 threshold logic in JS tests for isolation
- [05-01]: Third memory-capture returns already_promoted without re-invoking instinct-create -- confidence stays at 0.75 (correct idempotent behavior)
- [05-01]: Used 4 realistic content strings from actual Aether development phases as non-synthetic test data
- [05-02]: Agent pheromone_protocol uses 'signals' not 'instincts' directly -- signals is the delivery mechanism that includes instincts, so verification accepts either term
- [05-02]: Evidence field stored as array by instinct-create -- assertions join array before substring check

### Pending Todos

None yet.

### Blockers/Concerns

- Research flag: Phase 3 (Pheromone Signal Plumbing) likely needs research-phase during planning due to multiple interacting components across bash, playbooks, and agent definitions
- Risk: aether-utils.sh has 150 subcommands with no module boundaries; schema changes can cascade across 47+ test files

## Session Continuity

Last session: 2026-03-19
Stopped at: Completed 05-02-PLAN.md (Phase 5 complete)
Resume file: None
