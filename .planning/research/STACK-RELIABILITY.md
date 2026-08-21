# Command Surface Inventory and Test Coverage Research

**Domain:** Aether CLI command surface (reliability restoration)
**Researched:** 2026-05-20
**Confidence:** HIGH

## Executive Summary

Aether's command surface is large: **389 unique Cobra-registered commands** (including subcommands), served through three layers -- the Go runtime CLI (`cmd/`), 60 YAML wrapper definitions per platform (Claude, OpenCode), and 27 Codex TOML agents. The Go test suite has **5,075 test functions** across `cmd/` (2,591) and `pkg/` (867 in 84 files), plus 17 additional packages -- all passing.

The critical reliability finding: **70 Go source files (35% of `cmd/`) have no dedicated test file**. These include high-impact paths (eventbus.go, queen.go, instinct.go, spawn.go, shelf_cmd.go, midden_cmds.go, autopilot.go) and infrastructure (changelog.go, completion.go, recipes.go). Meanwhile, the YAML wrapper layer has **5 pure-wrapper commands** with no Go runtime backing (archaeology, chaos, dream, interpret, organize), and **5 lifecycle wrappers** are now delegated through the TypeScript host rather than calling the Go CLI directly (build, colonize, continue, plan, seal).

The downstream roadmapper needs: a classified command inventory with test-status annotations, identification of the 70 untested source files ranked by user-facing severity, and recognition that the YAML-to-runtime mapping has architectural drift from the v1.16 hybrid introduction.

## Audit-Catalog Findings

### Command Surface Size

| Layer | Count | Source |
|-------|-------|--------|
| Cobra-registered CLI commands | 389 | `aether audit-catalog --json` (live, HIGH confidence) |
| YAML wrapper definitions | 60 | `.aether/commands/*.yaml` (file system, HIGH confidence) |
| Claude Code wrappers | 60 | `.claude/commands/ant/*.md` (file system, HIGH confidence) |
| OpenCode wrappers | 60 | `.opencode/commands/ant/*.md` (file system, HIGH confidence) |
| Codex TOML agents | 27 | CLAUDE.md stated count (MEDIUM -- not re-verified) |
| Go source files in `cmd/` | 201 | File system (131 with tests, 70 without) |
| Go test files in `cmd/` | 219 | File system |
| Go test functions in `cmd/` | 2,591 | grep across test files |
| Go test functions in `pkg/` | 867 | grep across 84 test files |
| Total test functions (all packages) | 5,075 | `go test -v -count=1` live run |
| Passing Go packages | 18/18 | `go test ./...` live run |

### Command Group Structure

The 389 Cobra commands include 9 command groups with subcommands:

| Parent Command | Subcommands | Purpose |
|----------------|-------------|---------|
| `ceremony` | 4 (closeout, spawn-plan, wave-start, worker-complete) | Visual rendering for lifecycle events |
| `colony-depth` | 2 (get, set) | Colony depth management |
| `council` | 6 (advocate, budget-check, challenger, deliberate, history, sage) | Multi-perspective deliberation |
| `export` | 4 (archive, pheromones, registry, wisdom) | XML export targets |
| `host` | 9 (build, colonize, continue, lifecycle, oracle, plan, seal, swarm, watch) | TypeScript host delegation |
| `import` | 3 (pheromones, registry, wisdom) | XML import targets |
| `parallel-mode` | 2 (get, set) | Parallel execution mode |
| `plan-granularity` | 2 (get, set) | Planning granularity |
| `porter` | 1 (check) | Delivery readiness |

The remaining 347 commands are standalone (no parent, no subcommands).

### Output Modes

- **269 commands** have CLI flags
- **120 commands** are flagless (simple pass-through commands)
- **15 commands** have explicit JSON output mode (`--json` flag detected by audit-catalog heuristic)
- Most commands report "unknown" output mode (audit-catalog uses flag-based heuristic detection, not behavioral analysis)

## Command Classification

### Classification Definitions

| Class | Definition | Count (est.) |
|-------|-----------|-------------|
| **Public lifecycle** | Core colony workflow commands users invoke directly | ~20 |
| **Public utility** | Standalone tools for inspection or management | ~40 |
| **Internal runtime** | Subcommands called by wrappers, agents, or other commands | ~200 |
| **Alias** | Convenience redirect to another command | ~15 |
| **Subcommand** | Child of a parent command group | ~30 |
| **Pure wrapper** | YAML-defined with no Go runtime command (agent-only) | 5 |
| **TS host-delegated** | Lifecycle commands routed through TypeScript host | 5 |
| **Deprecated/placeholder** | Stubs or no-ops | ~3 |

### Public Lifecycle Commands (Priority 1 for Reliability)

These are the commands users invoke in the standard colony workflow:

| Command | Runtime Path | YAML Wrapper | Has Test File | Notes |
|---------|-------------|-------------|---------------|-------|
| `init` | Direct CLI | ant-init | `init_cmd_test.go` | Guided ritual, ceremony contract |
| `colonize` | TS host | ant-colonize | `codex_colonize_test.go` | Delegated since v1.16 |
| `plan` | TS host | ant-plan | `codex_plan_test.go` | Delegated since v1.16 |
| `build` | TS host | ant-build | `codex_build_test.go` | Delegated since v1.16 |
| `continue` | TS host | ant-continue | `codex_continue_test.go` | Delegated since v1.16 |
| `run` | Direct CLI | ant-run | `autopilot.go` **UNTESTED** | Autopilot wrapper |
| `swarm` | Direct CLI | ant-swarm | `swarm_cmd_test.go` | Bug destroyer |
| `seal` | TS host | ant-seal | `seal_wrapper_ceremony_test.go` | Delegated since v1.16 |
| `entomb` | Direct CLI | ant-entomb | `entomb_cmd_test.go` | Archive sealed colony |
| `oracle` | Direct CLI | ant-oracle | `oracle_loop_test.go` | RALF research loop |
| `recover` | Direct CLI | N/A | `recover_test.go` | Stuck colony rescue |
| `lay-eggs` | Direct CLI | ant-lay-eggs | `setup_cmd_test.go` | First-time setup |
| `update` | Direct CLI | ant-update | `update_cmd_test.go` | Hub sync |
| `publish` | Direct CLI | N/A | `publish_cmd_test.go` | Hub publish |
| `install` | Direct CLI | N/A | `install_cmd_test.go` | Platform install |
| `integrity` | Direct CLI | N/A | `integrity_cmd_test.go` | Pipeline validation |

**Gap:** 15/16 have test files, but `run` (autopilot) maps to `autopilot.go` which is in the untested source files list. The `run` command itself may be tested indirectly through other test files, but `autopilot.go` has no dedicated test.

### Public Utility Commands (Priority 2)

| Command | Has Test File | Gap Severity |
|---------|---------------|-------------|
| `status` | `status_test.go` | Covered |
| `history` | `history_test.go` | Covered |
| `phase` | `phase_test.go` | Covered |
| `focus` | `pheromone_write_test.go` | Covered |
| `redirect` | `pheromone_write_test.go` | Covered |
| `feedback` | `pheromone_write_test.go` | Covered |
| `pheromone-display` | `pheromone_mgmt_test.go` | Covered (display helpers untested) |
| `medic` | `medic_cmd_test.go` + 5 others | Covered |
| `discuss` | `discuss_test.go` | Covered |
| `preferences` | None found | **HIGH -- user preferences can be lost** |
| `profile-read` | `profile_test.go` | Covered |
| `resume-colony` | None found | **HIGH -- session recovery** |
| `resume-dashboard` | None found | **MEDIUM** |
| `pause-colony` | None found | **HIGH -- session pause data can be lost** |
| `maturity` | None found | **LOW -- display-only** |
| `tunnels` | None found | **LOW -- read-only** |
| `watch` | None found | **LOW -- compatibility alias** |
| `verify-castes` | None found | **MEDIUM -- validation** |
| `patrol-check` | `patrol_check_test.go` | Covered |
| `quick` | None found | **MEDIUM** |
| `data-clean` | None found | **MEDIUM -- destructive operation** |
| `insert-phase` | None found | **MEDIUM -- state mutation** |
| `shelf-*` (6 cmds) | `shelf_test.go` | **Covered but `shelf_cmd.go` itself untested** |
| `reference-*` (3 cmds) | `references_test.go` | Covered |
| `bump-version` | None found | **MEDIUM -- release pipeline** |
| `help` | `root_test.go` | Covered |
| `flags` | `flags_test.go` | **Covered but `flag_cmds.go` itself untested** |
| `migrate-state` | `internal_cmds_test.go` | Covered |
| `pheromones` (wrapper) | `pheromone_mgmt_test.go` | Covered |

**Gap:** 9/27 public utility commands have no test file. Another 3 have test coverage via adjacent files but their source file is untested.

### Internal Runtime Commands (Priority 3)

The remaining ~200 commands are internal subcommands. Key functional groups:

| Group | Example Commands | Test Coverage | Untested Source |
|-------|-----------------|--------------|-----------------|
| State management | state-read, state-write, state-mutate, state-checkpoint | `state_cmds_test.go` present | -- |
| Pheromone subcommands | pheromone-write, pheromone-expire, pheromone-prime | Multiple test files | `pheromones_read.go`, `pheromone_display_helpers.go`, `pheromone_sync.go` |
| Session management | session-init, session-update, session-clear | `session_cmds_test.go` | `session.go`, `session_compat.go` |
| Event bus | event-bus-publish, event-bus-query, event-bus-subscribe | `eventbus_subscribe_test.go`, `eventbus_timeout_test.go` | **`eventbus.go` (6 commands)** |
| Learning pipeline | learning-observe, learning-promote-auto, learning-inject | `learning_cmds_test.go` | -- |
| Instinct management | instinct-create, instinct-read, instinct-archive | `instinct_runtime_test.go` | **`instinct.go` (6 commands)** |
| Spawn tracking | spawn-log, spawn-complete, spawn-tree-active | `codex_worker_cleanup_test.go` | **`spawn.go`, `spawn_runs.go`, `spawn_track.go` (10 commands)** |
| Gate system | gate-check, gate-auto-resolve, should-skip-gate | `gate_test.go` | **`gate_results.go`** |
| Curation ants (9) | curation-archivist, curation-run, curation-sentinel | `curation_cmds_test.go` | -- |
| Hive system | hive-init, hive-store, hive-promote | `hive_runtime_test.go` | **`hive.go`, `hive_search.go`** |
| Queen system | queen-init, queen-promote, queen-write-learnings | Multiple `queen_*_test.go` files | **`queen.go`** |
| Midden (failure) | midden-write, midden-review, midden-search | None dedicated | **`midden_cmds.go` (10 commands)** |
| Swarm subcommands | swarm-display-*, swarm-findings-*, swarm-timing-* | `swarm_cmd_test.go` | `swarm.go` |
| Worktree | worktree-allocate, worktree-merge-back | `worktree_test.go` | -- |
| Export/Import | export archive, import pheromones | `exchange_import_sanitize_test.go` | **`exchange.go`** |
| Autopilot subcommands | autopilot-init, autopilot-status, autopilot-stop | None dedicated | **`autopilot.go` (8 commands)** |
| Council | council-deliberate, council-advocate | None dedicated | **`council.go` (7 commands)** |
| Flag subcommands | flag-add, flag-acknowledge, flag-check-blockers | `flags_test.go` | **`flag_cmds.go` (6 commands)** |
| Shelf subcommands | shelf-add, shelf-dismiss, shelf-promote | `shelf_test.go` | **`shelf_cmd.go` (7 commands)** |

## YAML Wrapper Runtime Mapping

### TS Host-Delegated (5 wrappers -- architectural drift from v1.16)

Since the hybrid architecture introduction, 5 lifecycle wrappers delegate through the TypeScript host rather than calling the Go CLI directly:

| YAML Wrapper | Runtime Path | TS Host Command |
|-------------|-------------|-----------------|
| ant-build | `aether host build --dry-run` + `aether build-finalize` | `host build` |
| ant-colonize | `aether host colonize --dry-run` + `aether colonize-finalize` | `host colonize` |
| ant-continue | `aether host continue --dry-run` + `aether continue-finalize` | `host continue` |
| ant-plan | `aether host plan --dry-run` + `aether plan-finalize` | `host plan` |
| ant-seal | `aether host seal --dry-run` + `aether seal-finalize` | `host seal` |

**Reliability implication:** Longer execution path (wrapper -> TS host -> Go CLI) with more failure points. Tests exist for the Go CLI side and the Codex workflow path, but end-to-end tests through the TS host are needed.

### Pure Wrapper / Agent-Only (5 commands -- no runtime backing)

| YAML Wrapper | Agent Behavior | Risk |
|-------------|---------------|------|
| `ant-archaeology` | Excavates git history | No runtime validation |
| `ant-chaos` | Probes edge cases | Writes back via `aether midden-*` |
| `ant-dream` | Philosophical observation | Writes to `.aether/dreams/` |
| `ant-interpret` | Reads dream sessions | Read-only |
| `ant-organize` | Codebase hygiene report | No output artifact contract |

**Reliability implication:** Cannot be smoke-tested via Go CLI. Classified as low priority -- they don't mutate colony state through the runtime.

## Untested Source Files (The Core Gap)

70 Go source files in `cmd/` lack a dedicated `*_test.go` file. Ranked by user-facing severity:

### HIGH SEVERITY -- Failures cause data loss or silent corruption

| Source File | Untested Commands | Impact of Failure |
|-------------|-------------------|-------------------|
| `eventbus.go` | event-bus-publish, event-bus-query, event-bus-replay, event-bus-subscribe, event-bus-cleanup (6) | Core pub/sub for wisdom pipeline; failures silently drop learning events |
| `queen.go` | queen-init, queen-read, queen-promote, queen-promote-instinct, queen-seed-from-hive, queen-thresholds, queen-write-learnings, queen-migrate (8) | Central colony intelligence; failures corrupt QUEEN.md |
| `instinct.go` | instinct-create, instinct-read, instinct-apply, instinct-archive, instinct-decay-all (5+) | Core learning pipeline; failures lose learned patterns |
| `midden_cmds.go` | midden-write, midden-acknowledge, midden-recent-failures, midden-review, midden-search, midden-tag, midden-prune, midden-collect, midden-cross-pr-analysis, midden-handle-revert (10) | Failure tracking; 10 subcommands with zero test |
| `hive.go` | hive-init, hive-read, hive-store, hive-promote, hive-abstract (5) | Cross-colony wisdom; failures lose shared knowledge |
| `hive_search.go` | hive-search (1) | FTS recall for hive wisdom |
| `spawn.go` | spawn-log, spawn-complete, spawn-can-spawn, spawn-can-spawn-swarm (4) | Worker lifecycle tracking; failures cause phantom/orphan workers |
| `spawn_runs.go` | spawn-tree-active, spawn-tree-depth, spawn-tree-load, spawn-efficiency, spawn-get-depth (5) | Spawn tree visibility |
| `spawn_track.go` | spawn-track (1) | Timeout enforcement |
| `autopilot.go` | run, autopilot-init, autopilot-status, autopilot-stop, autopilot-update, autopilot-check-replan, autopilot-headless-check, autopilot-set-headless (8) | Colony autopilot |
| `shelf_cmd.go` | shelf-add, shelf-dismiss, shelf-dismiss-batch, shelf-list, shelf-promote, shelf-promote-batch, shelf-detect (7) | Idea management; data loss risk |
| `council.go` | council, council-advocate, council-challenger, council-deliberate, council-history, council-sage, council-budget-check (7) | Deliberation system |

**Subtotal HIGH: 12 files, ~67 untested commands**

### MEDIUM SEVERITY -- Failures cause degraded experience or partial state issues

| Source File | Untested Commands | Impact |
|-------------|-------------------|--------|
| `flag_cmds.go` | flag-add, flag-acknowledge, flag-auto-resolve, flag-check-blockers, flag-resolve (5+) | Colony flags/blocks not persisted correctly |
| `survey.go` | survey-load, survey-verify (2) | Survey data integrity |
| `changelog.go` | changelog-append, changelog-collect-plan-data (2) | Changelog management |
| `gate_results.go` | gate-results-read, gate-results-write (2) | Gate persistence |
| `session.go` | session-init, session-read, session-update (3+) | Session management |
| `session_compat.go` | Session compatibility | Backward compat |
| `memory_health.go` | memory-metrics (1) | Memory health display |
| `memory_details.go` | memory-details (1) | Memory drill-down |
| `skill_curator.go` | skill-curator-run (1) | Skill lifecycle |
| `skill_lifecycle.go` | skill-archive, skill-recover, skill-pin, skill-promote, skill-diff (5) | Skill lifecycle |
| `dispatch_runtime.go` | Runtime dispatch helpers | Platform dispatch |
| `exchange.go` | export, import (parent) | XML exchange parent |
| `learn_export.go` | learn-export, learn-import (2) | Learning pack portability |
| `swarm.go` | swarm (parent) | Swarm parent logic |
| `state_repair.go` | State repair helpers | Colony recovery support |
| `phase_skip.go` | skip-phase (1) | Emergency phase skip |
| `grave.go` | grave-add, grave-check (2) | Agent grave tracking |
| `immune.go` | immune-auto-scar (1) | Failure pattern auto-detection |
| `lifecycle_helpers.go` | Lifecycle helper functions | Shared across build/continue |
| `codegraph.go` | codebase-scan, codebase-query (2) | Dependency graph |
| `codegraph_context.go` | Codebase context assembly | Dependency context |
| `command_truth.go` | Command truth helpers | Source-of-truth validation |
| `prompt_integrity.go` | Prompt integrity checks | Worker prompt validation |
| `queen_spawn_budget.go` | Spawn budget calculation | Worker economy |
| `pheromone_sync.go` | pheromone-snapshot-inject, pheromone-merge-back (2) | Worktree pheromone sync |
| `pheromone_display_helpers.go` | Pheromone display formatting | Display helpers |
| `pheromones_read.go` | pheromone-read, pheromone-count (2) | Pheromone reading |
| `view_state.go` | view-state-* (6 commands) | View state management |
| `completion.go` | completion (1) | Shell completion |
| `recipes.go` | recipes (1) | Workflow recipes |

**Subtotal MEDIUM: 29 files, ~55 untested commands**

### LOW SEVERITY -- Platform-specific, build artifacts, or trivial

| Source File | Notes |
|-------------|-------|
| `oracle_background_unix.go` | Platform-specific Oracle background |
| `oracle_background_windows.go` | Platform-specific Oracle background |
| `oracle_process_unix.go` | Platform-specific Oracle process |
| `oracle_process_windows.go` | Platform-specific Oracle process |
| `verification_process_group_unix.go` | Platform-specific process group |
| `verification_process_group_windows.go` | Platform-specific process group |
| `worker_cleanup_signal_common.go` | Platform-specific cleanup |
| `worker_cleanup_signal_unix.go` | Platform-specific cleanup |
| `worker_cleanup_signal_windows.go` | Platform-specific cleanup |
| `binary_download.go` | Binary download (E2E tested elsewhere) |
| `build_playbook_context.go` | Build playbook context loading |
| `codex_build_progress.go` | Codex build progress |
| `codex_colonize_finalize.go` | Codex colonize finalization |
| `codex_continue_finalize.go` | Codex continue finalization |
| `codex_continue_plan.go` | Codex continue planning |
| `codex_dispatch_contract.go` | Codex dispatch contract |
| `codex_worker_artifacts.go` | Codex worker artifacts |
| `recover_repair.go` | Tested via `recover_test.go` |
| `recover_scanner.go` | Tested via `recover_test.go` |
| `recover_visuals.go` | Recovery visuals |
| `recovery_snapshot.go` | Recovery snapshot |
| `autofix.go` | Autofix checkpoint/rollback |
| `clash.go` | Worktree clash detection |
| `maintenance.go` | Maintenance helpers |
| `ts_host_artifacts.go` | TS host artifact management |
| `version.go` | Version command (trivial) |
| `context_weighting.go` | Context weighting |

**Subtotal LOW: 29 files**

## Recommendations for Roadmap

### Phase 1: Classification and Audit (CATALOG-01, CATALOG-02)
- Extend `audit-catalog` with a `--classify` flag that tags each command with its category
- Add classification metadata to YAML source files
- This is a code-level task, not a research task -- the classification logic is documented here

### Phase 2: Critical Untested Commands (TEST-01, TEST-02)
Priority order based on data-loss risk:

1. **eventbus.go** -- 6 commands, backbone of wisdom pipeline. Silent failure = lost learning.
2. **queen.go** -- 8 commands, colony intelligence core. Failure = QUEEN.md corruption.
3. **instinct.go** -- 5+ commands, learning pipeline. Failure = lost learned patterns.
4. **midden_cmds.go** -- 10 commands, entire failure tracking subsystem untested.
5. **hive.go + hive_search.go** -- 6 commands, cross-colony wisdom. Failure = lost shared knowledge.
6. **spawn.go + spawn_runs.go + spawn_track.go** -- 10 commands, worker lifecycle. Failure = phantom workers.
7. **autopilot.go** -- 8 commands, colony autopilot. Failure = silent build loop breakage.
8. **flag_cmds.go** -- 5+ commands, colony flag management.
9. **council.go** -- 7 commands, deliberation system.
10. **shelf_cmd.go** -- 7 commands, idea management.

**Estimated:** 12 source files, ~67 commands need test files. At ~5-8 tests per file, this is ~60-100 new test functions.

### Phase 3: TS Host E2E Tests (WORKFLOW-01 through WORKFLOW-09)
End-to-end tests through the TS host for the 5 host-delegated lifecycle commands. Current coverage is Go CLI-side and Codex workflow-side only.

### Phase 4: Public Utility Test Gaps
Address the 9 untested public utility commands (preferences, resume-colony, pause-colony, data-clean, insert-phase, quick, verify-castes, bump-version, maturity).

## What NOT to Do

| Avoid | Why | Do Instead |
|-------|-----|-------------|
| Test the 29 LOW-severity files first | They are platform-specific, build artifacts, or trivially tested elsewhere | Focus on HIGH-severity files that cause data loss |
| Add test coverage metrics to audit-catalog | Binary coverage metrics obscure the real gap: which commands have no smoke test at all | Track "has at least one smoke test" per command |
| Write integration tests for pure-wrapper commands (chaos, dream, archaeology, interpret, organize) | They have no runtime backing; testing them would test prompt quality, not code reliability | Document them as "agent-only, no runtime contract" and exclude from test coverage requirements |
| Refactor test files to match source file names 1:1 | Some test files legitimately test multiple source files (e.g., `codex_build_test.go` tests build + finalize + worktree) | Add tests where coverage is missing, don't restructure existing coverage |

## Sources

- `aether audit-catalog --json` -- live CLI output, HIGH confidence (2026-05-20)
- `aether --help` -- live CLI output, HIGH confidence (2026-05-20)
- File system scan of `cmd/*.go` (201 files), `cmd/*_test.go` (219 files) -- HIGH confidence
- File system scan of `.aether/commands/*.yaml` (60 files) -- HIGH confidence
- File system scan of `.claude/commands/ant/*.md` (60 files), `.opencode/commands/ant/*.md` (60 files) -- HIGH confidence
- `go test ./cmd/... --count=1` -- live run, all passing (2026-05-20)
- `go test ./... --count=1` -- 5,075 test functions, all 18 packages passing (2026-05-20)
- `cmd/audit_catalog.go` -- source code for audit-catalog implementation, HIGH confidence
- `.aether/commands/*.yaml` -- all 60 YAML files read and classified, HIGH confidence
- CLAUDE.md, PROJECT.md -- project documentation, HIGH confidence

---
*Command surface inventory for: Aether v1.23 Daily Driver Reliability*
*Researched: 2026-05-20*
