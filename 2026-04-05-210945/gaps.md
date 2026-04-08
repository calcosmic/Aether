# Knowledge Gaps

## Open Questions
- **Q2 (35%):** Worktree merge-back gap — infrastructure exists for creation/cleanup but merge-back step is dead code. Placement identified (continue-advance Step 2.0.4) but implementation details needed: exact git merge commands, conflict resolution strategy per file type, rollback on failure, interaction with existing midden/pheromone steps.
- **Q3 (40%):** Publish flow — Go sources NOT in npm package (confirmed via files whitelist). No Go compilation in prepublishOnly. Go binary is local-only. Still unknown: should Go binary be distributed? Is separate distribution planned? What about go.sum integrity?
- **Q4 (72%):** Push/publish sequence — No version bump needed yet (Go incomplete), no go mod tidy needed, CI has no Go steps, registry has 46 repos but only shell changes propagate via npm. Remaining: confirm PR vs direct-commit workflow preference, plan CI Go step addition before Phase 51.
- **Q5 (0%):** Other repos affected — registry contents now mapped (46 repos), need to assess stale colony state and update requirements.
- **Q6 (0%):** Go transition roadmap — Phase 50 at 1/6, Phase 51 not started, 14/20 phases complete in v5.4 milestone. Need to assess next priorities.

## Contradictions
- continue-advance.md Step 2.0.5 (Pheromone Merge-Back) and Step 2.0.6 (Midden Collection) reference `$last_merged_branch` and `$last_merge_sha` variables that no existing step ever populates. These steps are dead code paths.
- Go source is tracked in git but excluded from npm distribution — the Go CLI exists as a local-only tool with no distribution mechanism. This may be intentional (early development) or an oversight.
- REDIRECT signal says "all changes go through PRs" but all Go work (Phases 45-50) has been committed directly to main with no PR workflow. This is a process gap.
- CI pipeline (ci.yml) has NO Go steps but Go code is being actively developed and merged. Go compile errors will silently pass CI.

## Last Updated
Iteration 4 -- 2026-04-02T10:45:00Z
