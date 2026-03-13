# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-13)

**Core value:** The oracle produces research you can act on -- verified, iteratively deepened, structured for the topic.
**Current focus:** Phase 9 complete -- Source Tracking and Trust Layer

## Current Position

Milestone: v1.1 Oracle Deep Research
Phase: 9 of 11 (Source Tracking and Trust Layer)
Plan: 2 of 2 in current phase (COMPLETE)
Status: Phase Complete
Last activity: 2026-03-13 -- Plan 09-02 complete (trust scoring tests: 10 Ava + 9 bash assertions)

Progress: [#######---] 70%

## Performance Metrics

**v1.0 Velocity (reference):**
- Total plans completed: 11
- Average duration: 3.3min
- Total execution time: 0.61 hours

**v1.1:**
- Total plans completed: 8
- Average duration: 3.9min
- Total execution time: 0.52 hours

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
- Convergence composite score: gap_resolution*40% + coverage*30% + (low_novelty?100:0)*30% with integer arithmetic
- Convergence requires composite >= 85 AND 2 consecutive low-novelty iterations
- Diminishing returns uses 3-iteration rolling window with phase-adjusted thresholds (investigate: 0, others: 1)
- Every exit path triggers synthesis pass -- max-iter changed from exit 1 to synthesis + exit 0
- ORACLE_CONVERGENCE_THRESHOLD and ORACLE_DR_WINDOW env vars for empirical tuning
- Test oracle.sh convergence functions using same sed extraction + isolation pattern from Phase 7
- Multi-function sed extraction needed when function A depends on function B (update_convergence_metrics + compute_convergence)
- Flag unsourced findings rather than reject them -- trust_summary.no_source makes the gap visible without losing research
- Source tracking is a prompt+schema problem -- AI records sources, oracle.sh counts them structurally
- plan.json v1.1 bump is safe -- validate-oracle-state checks version type not value
- Used jq -n for JSON construction in bash tests instead of heredocs to avoid special character issues with nested objects

### Pending Todos

None.

### Blockers/Concerns

- Verify `--json-schema` Claude CLI flag availability early in Phase 7 -- fallback to prompt-based JSON enforcement if unavailable
- Convergence threshold numbers need empirical tuning in Phase 8 -- start with research recommendations, iterate
- Colony integration API (Phase 11) needs deliberate design session before implementation

## Session Continuity

Last session: 2026-03-13
Stopped at: Completed 09-02-PLAN.md (Phase 9 complete)
Resume file: .planning/phases/09-source-tracking-and-trust-layer/09-02-SUMMARY.md
