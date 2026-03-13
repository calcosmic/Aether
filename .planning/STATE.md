# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-13)

**Core value:** The oracle produces research you can act on -- verified, iteratively deepened, structured for the topic.
**Current focus:** Phase 7 -- Iteration Prompt Engineering

## Current Position

Milestone: v1.1 Oracle Deep Research
Phase: 7 of 11 (Iteration Prompt Engineering)
Plan: 2 of 2 in current phase
Status: Phase Complete
Last activity: 2026-03-13 -- Plan 07-02 complete (phase transition tests + iteration counter tests)

Progress: [##########] 100%

## Performance Metrics

**v1.0 Velocity (reference):**
- Total plans completed: 11
- Average duration: 3.3min
- Total execution time: 0.61 hours

**v1.1:**
- Total plans completed: 4
- Average duration: 4.3min
- Total execution time: 0.29 hours

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.

- Research recommends: state schema first, then iteration prompts, then orchestrator -- strict dependency chain
- Phase 9 (Trust) and Phase 10 (Steering) can be parallel -- both depend on Phase 8, not each other
- Colony integration (Phase 11) deferred last -- requires all other systems stable
- Enum validation in validate-oracle-state uses jq array membership check pattern
- research-plan.md regenerated after every iteration (negligible cost, always-current user view)
- Topic change detection reads state.json directly, wizard passes new topic via ORACLE_NEW_TOPIC env var
- Oracle wizard creates 5 structured files replacing research.json and progress.md
- Archive uses timestamped subdirectories for cleaner session preservation
- Status display reads research-plan.md executive summary instead of progress.md tail
- Phase transitions use structural jq metrics: 25%/60%/80% avg confidence thresholds
- build_oracle_prompt prepends phase heredoc directives to oracle.md content
- Confidence rubric anchored to evidence quality with anti-inflation rule (one blog post = 30% not 70%)
- Iteration and phase managed exclusively by oracle.sh, not the AI prompt
- Test oracle.sh functions by extracting via sed and sourcing in isolation -- avoids set -e and main-loop side effects
- Edge case tests include zero questions, boundary confidence values (exactly 25%), and all-answered scenarios

### Pending Todos

None.

### Blockers/Concerns

- Verify `--json-schema` Claude CLI flag availability early in Phase 7 -- fallback to prompt-based JSON enforcement if unavailable
- Convergence threshold numbers need empirical tuning in Phase 8 -- start with research recommendations, iterate
- Colony integration API (Phase 11) needs deliberate design session before implementation

## Session Continuity

Last session: 2026-03-13
Stopped at: Phase 8 context gathered
Resume file: .planning/phases/08-orchestrator-upgrade/08-CONTEXT.md
