# Phase 41: Midden Collection - Context

**Gathered:** 2026-03-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Failure records from merged branches are collected into main's midden with idempotency and cross-PR pattern detection. Four new subcommands: midden-collect, midden-handle-revert, midden-cross-pr-analysis, and midden-prune. This phase builds on Phase 40's pheromone propagation pattern (branch-local data flows to main via worktree path).

</domain>

<decisions>
## Implementation Decisions

### API Design
- **D-01:** Smart wrapper API — user-facing command accepts `--branch <name> --merge-sha <sha>` and internally resolves the worktree path. The ROADMAP success criteria use this simpler form. Internally, the design doc's `--worktree-path` is used when the wrapper can't auto-resolve.
- **D-02:** Update ROADMAP success criteria to reflect the smart wrapper API — `midden-collect --branch <branch> --merge-sha <sha>` is the user-facing contract.

### Cross-PR Auto-Emit
- **D-03:** Cross-PR systemic patterns auto-emit REDIRECT pheromones to the hub. No manual intervention needed — like a smoke detector. Uses the design doc's tiered thresholds: 2+ PRs with 3+ entries = systemic, 3+ PRs with 5+ entries = critical.

### Wiring Points
- **D-04:** Wire midden-collect into `/ant:continue` and `/ant:run`. These are the existing workflows where merges happen. No git hook setup needed.
- **D-05:** Wire midden-cross-pr-analysis into `/ant:continue` and `/ant:run` as well (runs after collection).
- **D-06:** Do NOT wire into `/ant:status` for this phase — defer to a future phase if needed.

### Pruning
- **D-07:** Include pruning commands (`midden-prune --stale-merges`, `midden-prune --reverted --age 30`) in this phase's scope. Build all four subcommands together so the system is complete from day one.

### Claude's Discretion
- Exact worktree path resolution logic (git worktree list, fallback patterns)
- Cross-PR score formula tuning (design doc provides defaults)
- Output format details (JSON structure, human-readable summaries)
- Whether to add a `--dry-run` flag to midden-collect (design doc mentions it)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design Specification
- `.aether/docs/midden-collection-design.md` — Complete design: 3 subcommands, idempotency, revert handling, cross-PR analysis, edge cases, retention policy, integration points

### Existing Implementation
- `.aether/utils/midden.sh` — Current midden functions: `_midden_write`, `_midden_recent_failures`, `_midden_review`, `_midden_acknowledge`, `_midden_ingest_errors`, `_midden_search`, `_midden_tag`
- `.aether/aether-utils.sh` line 34 — Sources midden.sh
- `.aether/aether-utils.sh` lines 1308-1312 — Current midden subcommand dispatch
- `.aether/aether-utils.sh` line 1707 — Auto-REDIRECT threshold (>= 3 same-category entries)

### Integration Points
- `.aether/utils/spawn.sh` — `_worktree_create` function (worktree path resolution)
- `.aether/docs/command-playbooks/continue-verify.md` — Continue flow (where midden-collect should be wired)
- `.aether/docs/command-playbooks/continue-advance.md` — Continue advance flow (where cross-pr-analysis should run)
- `.aether/docs/command-playbooks/build-verify.md` — Build/Run verification (where midden-collect should be wired for /ant:run)

### State Contract
- `.aether/docs/state-contract-design.md` — Branch-local state rules, .aether/data/ gitignore behavior

### Pheromone Propagation (Phase 40 pattern)
- `.aether/utils/pheromone.sh` — REDIRECT emission pattern (cross-PR auto-emit follows same approach)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `_midden_write`: Core write function with category, source, message. Uses file locking and atomic writes.
- `_midden_tag`: Tag entries by ID — directly reusable for revert tagging (`reverted:<sha>`)
- `_midden_search`: Search entries by keyword with category/source filters — basis for cross-PR queries
- `_midden_ingest_errors`: Existing ingestion pattern (read source file, create entries, move to .ingested) — precedent for midden-collect
- `acquire_lock`/`release_lock`: File locking for concurrent access
- `atomic_write`: Safe file updates via atomic-write.sh

### Established Patterns
- Subcommand dispatch pattern in aether-utils.sh: `midden-*) _midden_subcommand "$@" ;;`
- Entry schema: `{id, timestamp, category, source, message, reviewed}` with optional `tags[]`, `acknowledged`, `acknowledged_at`
- Midden file location: `$COLONY_DATA_DIR/midden/midden.json` (branch-local, gitignored)
- Test pattern: isolated temp repos with `setup_test_repo()` helper (from pheromone tests)

### Integration Points
- `/ant:continue` flow reads branch context and runs post-merge steps
- `/ant:run` is the autopilot that chains build-verify-advance
- Worktree paths available via `git worktree list` or stored in spawn state

</code_context>

<specifics>
## Specific Ideas

- The design doc is comprehensive and well-verified against the codebase. Planning should focus on implementation, not design decisions.
- The smart wrapper API means the planner needs to implement worktree path resolution as part of midden-collect.
- Pruning is included in scope — all four subcommands should be built together.

</specifics>

<deferred>
## Deferred Ideas

- `/ant:status` integration for cross-PR analysis — defer to a future phase
- Closed-without-merge worktree tracking — the design doc says this is a no-op, which is fine
- Re-revert handling (revert of a revert) — the design doc covers this but it's edge-casey

</deferred>

---
*Phase: 41-midden-collection*
*Context gathered: 2026-03-31*
