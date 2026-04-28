# Technology Stack

**Project:** Aether v1.11 -- Self-Hosting Cleanup, Smart Init Restoration, Platform Hardening, UX Improvements
**Researched:** 2026-04-28
**Confidence:** HIGH (all findings from direct source code inspection)

## Executive Summary

v1.11 requires zero new external Go dependencies. All four feature areas -- self-hosting cleanup, Smart Init restoration, platform hardening, and UX improvements -- build on patterns that already exist in the codebase. The Go runtime (`cmd/`) already has `init-research` with 10 pheromone suggestion patterns, governance detection, charter generation, and git history analysis. The publish/update pipeline already handles companion file sync across 3 platforms with stale-publish detection. The gap is integration: `suggest-analyze` exists only as a documented-but-unimplemented command, and the init ceremony in the wrappers calls `init-research` but the Go runtime `init` command does not invoke it directly.

## Recommended Stack

### Core Framework (No Changes)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.26.1 | Runtime language | Existing `go.mod` target. No version change needed. |
| `github.com/spf13/cobra` | v1.10.2 | CLI subcommand registration | All new commands (`aether audit`, `aether suggest-analyze`, `aether suggest-approve`) register via `rootCmd.AddCommand()` in `init()`. |
| `pkg/storage.Store` | existing | File-locked JSON persistence | All state mutations use `SaveJSON`/`LoadJSON`/`AtomicWrite` with cross-process file locking. |
| `encoding/json` | stdlib | Marshal/unmarshal | Standard pattern across every command. |
| `os` / `path/filepath` / `io/fs` | stdlib | Directory walking, file operations | Already used by `init_research.go` for the same directory scanning patterns needed for audit. |

### No New Dependencies

| Category | Existing Coverage | New Dependency Needed |
|----------|-------------------|----------------------|
| Directory scanning | `init_research.go` `filepath.WalkDir` with skip lists | No |
| File fingerprinting | `pheromone_write.go` `sha256Sum` content hashing | No |
| CLI output | `helpers.go` `outputOK`/`outputError`/`outputWorkflow` | No |
| Visual rendering | `codex_visuals.go` ANSI caste colors, stage markers | No |
| JSON structured output | All commands use `map[string]interface{}` payloads | No |
| File locking | `pkg/storage/store.go` `FileLocker` | No |

### Feature 1: Self-Hosting Cleanup (`aether audit`)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| New command: `cmd/audit.go` | new | Scan repo for stale self-hosting artifacts | New cobra subcommand following `cmd/autofix.go` pattern (~100-150 lines). Walks known artifact locations, reports findings as structured JSON. |
| `filepath.WalkDir` | stdlib | Recursive directory scanning | Same pattern as `init_research.go` lines 495-530. Extended skip list to avoid `node_modules`, `.git`, etc. |
| `os.Stat` + size checks | stdlib | Detect stale/oversized artifacts | Compare file sizes and ages against thresholds (e.g., chambers > 6 months, oracle archives, stale build data). |
| `os.RemoveAll` (with confirmation) | stdlib | Remove identified artifacts | Follows `autofix.go` checkpoint-then-remove pattern. Audit reports, `aether audit --clean` removes with backup. |
| `cmd/autofix.go` checkpoint pattern | existing | Safety net before destructive cleanup | Already has `autofix-checkpoint` / `autofix-rollback` for COLONY_STATE.json. Audit cleanup can use same pattern for any files it removes. |

**Artifact locations to scan (from direct inspection):**

| Location | Size | Description | Action |
|----------|------|-------------|--------|
| `.aether/chambers/` | 6.3M | 18 entombed colony archives | Report age, offer selective removal |
| `.aether/data/build/` | varies | Build artifacts from old phases | Report, safe to clean if colony not active |
| `.aether/data/backups/` | varies | COLONY_STATE backups | Report age, offer removal |
| `.aether/data/*.bak*` | small | Rotated backup files | Safe to remove |
| `.aether/data/spawn-tree-archive/` | varies | Old spawn tree snapshots | Report, offer removal |
| `.aether/data/learning-observations.json.bak.*` | small | Old observation backups | Safe to remove |
| `.aether/data/session.json.bak*` | small | Session backups | Safe to remove |
| `.aether/data/pr-context-cache.json` | small | Stale context cache | Safe to remove |
| `.aether/data/watch-progress.txt` / `watch-status.txt` | small | Stale watch output | Safe to remove |
| `.aether/dreams/` | small | Local session notes | Never remove (user content) |
| `.aether/oracle/archive/` | 1.6M | Deep research archives | Report, offer selective removal |

### Feature 2: Smart Init Restoration

The Go runtime already has most of the Smart Init intelligence. The gap is that `aether init` does not call `init-research` internally, and `suggest-analyze` / `suggest-approve` commands do not exist.

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `cmd/init_cmd.go` (modify) | existing | Add `--smart` flag that invokes init-research before state creation | Currently `init` creates COLONY_STATE.json directly. With `--smart`, it first runs `initResearchCmd` logic internally, outputs charter for approval, then creates state. |
| `cmd/init_research.go` (modify) | existing | Export scan functions as reusable Go functions (not just CLI command) | `detectGovernance()`, `analyzeGitHistory()`, `generatePheromoneSuggestions()`, `generateCharter()` already exist as internal functions. They need to be callable from `init_cmd.go` without going through the cobra command. Currently they are -- they are package-level functions, not methods on a struct. **No code change needed.** |
| New command: `cmd/suggest.go` | new | `aether suggest-analyze` and `aether suggest-approve` subcommands | ~200-300 lines. `suggest-analyze` reuses `generatePheromoneSuggestions()` from `init_research.go` plus adds build-specific patterns. `suggest-approve` reads suggestions and calls `pheromone-write` for approved ones. |
| `.claude/commands/ant/build-context.md` (modify) | existing | Uncomment suggest-analyze step in build playbook | Currently lines 149-181 deprecate suggest-analyze. Remove deprecation guard and call the real command. |

**Smart Init ceremony flow (Go-side):**

```
aether init --smart "Build feature X"
  |
  +-- Run init-research scan (reuse existing functions)
  |   |-- detectGovernance() -> linters, CI, tests, formatters
  |   |-- analyzeGitHistory() -> commits, contributors, branch
  |   |-- generatePheromoneSuggestions() -> 10 patterns
  |   |-- generateCharter() -> intent, vision, governance, goals
  |   +-- detectPriorColonies() -> archived colony count
  |
  +-- Output charter + suggestions as structured JSON
  |
  +-- Wait for approval (interactive or --auto-approve flag)
  |
  +-- Write approved pheromones via pheromone-write
  |
  +-- Create COLONY_STATE.json (existing init logic)
```

### Feature 3: Platform Hardening

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `cmd/platform_sync.go` (modify) | existing | Add parity validation to `aether publish` | Already has `repoSyncPairs()` defining 9 sync pairs. Add a validation pass that compares command/agent counts across Claude, OpenCode, and Codex after publish. Same counting logic as `checkStalePublish()` in `update_cmd.go` lines 403-472. |
| `cmd/command_parity_test.go` (existing) | existing | Already tests command count parity | Verify this test covers all 3 platforms, not just Claude. Currently tests `expectedClaudeCommandCount = 50` and `expectedOpenCodeCommandCount = 50`. |
| Error wrapping with `fmt.Errorf` | stdlib | Consistent error messages across platforms | Some commands use bare `outputError`, others use `fmt.Errorf`. Standardize on wrapping chain with `%w` for debugging. |
| `cmd/codex_worker_artifacts.go` (existing) | existing | Codex worker cleanup | Already handles Codex-specific worker artifact management. Verify it handles all error cases gracefully. |

**Current parity status (verified by direct inspection):**

| Surface | Claude Code | OpenCode | Codex CLI | Status |
|---------|-------------|----------|-----------|--------|
| Slash commands | 50 | 50 | N/A (uses `aether` CLI) | PARITY |
| Agent definitions | 26 | 26 | 26 | PARITY |
| Build command | 125 lines (identical) | 125 lines (identical) | Go runtime | PARITY |
| Continue command | identical | identical | Go runtime | PARITY |
| Init command | calls `init-research` | calls `init-research` | needs `--smart` flag | GAP (Codex) |
| suggest-analyze | documented but no Go impl | documented but no Go impl | no Go impl | MISSING |
| Error format | `outputError` JSON | `outputError` JSON | Go runtime | CONSISTENT |

### Feature 4: UX Improvements

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `cmd/codex_visuals.go` (modify) | existing | Add ANSI visual output for audit, smart-init, and suggest commands | Already has `casteColorMap`, `casteEmojiMap`, `casteLabelMap`, `casteIdentity()`, stage markers. New commands follow the same `render*Visual()` function pattern. |
| `cmd/colony_prime_context.go` (modify) | existing | Add init-research findings as a colony-prime section | The section system supports arbitrary sections with priority ordering. An `init_context` section at low priority would inject detected governance and pheromone suggestions into worker prompts during the first build. |
| Structured JSON output | existing | All commands use `outputOK`/`outputError`/`outputWorkflow` | New commands must use the same output contract for wrapper compatibility. |

## New Files to Create

| File | Purpose | Estimated Size | Pattern Source |
|------|---------|----------------|----------------|
| `cmd/audit.go` | `aether audit` + `aether audit --clean` commands for stale artifact detection and removal | ~200 lines | `cmd/autofix.go` (checkpoint pattern) + `cmd/init_research.go` (directory walking) |
| `cmd/suggest.go` | `aether suggest-analyze` and `aether suggest-approve` subcommands | ~250 lines | `cmd/midden_cmds.go` (subcommand registration) + `cmd/init_research.go` (suggestion generation) |

## Files to Modify

| File | Change | Risk |
|------|--------|------|
| `cmd/init_cmd.go` | Add `--smart` flag that runs init-research scan before state creation; output charter for approval; write approved pheromones | Medium -- modifies the init hot path, needs careful idempotency handling |
| `cmd/init_research.go` | Add build-specific pheromone patterns to `generatePheromoneSuggestions()` (e.g., test coverage gaps, TODO density, large file detection) | Low -- additive function, no existing callers affected |
| `.claude/commands/ant/build-context.md` | Remove suggest-analyze deprecation guard (lines 149-181), replace with real `aether suggest-analyze` call | Low -- already documented as the intended flow |
| `.opencode/commands/ant/build-context.md` | Same as above | Low |
| `cmd/codex_visuals.go` | Add `renderAuditVisual()`, `renderSmartInitVisual()`, `renderSuggestVisual()` functions | Low -- additive rendering functions, no existing callers affected |
| `cmd/platform_sync.go` | Add post-publish parity validation (command count, agent count across all 3 platforms) | Low -- additive validation pass |
| `cmd/command_parity_test.go` | Extend to cover Codex agent count (26) and Codex skill count (83) | Low -- additive test cases |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Artifact cleanup approach | New `aether audit` command with `--clean` flag | Shell script one-liner (`find .aether -type f -mtime +30 -delete`) | A Go command integrates with the existing checkpoint/rollback safety net, produces structured JSON output for wrapper consumption, and follows the "runtime is authoritative" principle. Shell scripts break the wrapper-runtime contract. |
| Smart Init integration | `--smart` flag on existing `aether init` | Separate `aether smart-init` command | A flag on the existing command avoids duplicating the idempotency logic, sealed-colony detection, worktree cleanup, and session creation that `init_cmd.go` already handles. The flag simply inserts the research-approval step before state creation. |
| suggest-analyze implementation | New `cmd/suggest.go` reusing `generatePheromoneSuggestions()` | LLM-based analysis in wrapper markdown | The original shell `suggest-analyze` was 300 lines of deterministic file pattern detection -- not LLM analysis. The Go port should match this: deterministic checks that produce reproducible suggestions. The 10 patterns already in `init_research.go` plus build-specific additions cover the original shell script's scope. |
| Parity validation | Add to `aether publish` | Separate `aether verify-parity` command | Validation at publish time catches parity breaks before they reach downstream repos. A separate command would require developers to remember to run it manually. |
| Audit scope | Walk known `.aether/` subdirectories | Full repo scan for any Aether-managed file | A targeted scan of known locations is faster, produces actionable results, and avoids false positives from user-created files that happen to be in `.aether/` but are not self-hosting artifacts (e.g., `dreams/`). |

## What NOT to Add

| Technology | Why Avoid |
|------------|-----------|
| Any new `go.mod` dependency | All four feature areas are covered by existing patterns. `go mod tidy` should produce no changes. The entire v1.11 scope is additive Go code using stdlib + cobra + existing `pkg/` packages. |
| SQLite or any database | Artifact metadata is trivially small (file paths, sizes, timestamps). JSON files and in-memory maps are sufficient. Adding a database for a one-time audit tool is massive over-engineering. |
| LLM API calls for suggest-analyze | The original shell script was deterministic pattern matching, not LLM analysis. Porting it as Go code that checks file patterns preserves the original behavior and avoids API costs, latency, and non-determinism. |
| Configuration file for audit rules | The artifact locations are well-known and few (11 locations identified). Hard-coding them in `audit.go` with clear constants is simpler, more auditable, and avoids a config file that needs to be kept in sync. |
| Watchdog or cron-based cleanup | `aether audit` is a manual command. Self-hosting cleanup is a one-time event, not an ongoing process. Automated cleanup risks deleting active colony data. Manual execution with `--clean` flag gives the user control. |
| New agent definitions | No new worker castes are needed. Audit, suggest-analyze, and smart-init are CLI operations, not colony worker tasks. |

## Installation

No installation needed. All dependencies already exist in `go.mod`.

```bash
# Verify existing dependencies are sufficient
go mod tidy  # should show no changes
go build ./cmd/aether  # should compile cleanly

# Run existing tests to establish baseline before starting
go test ./... -race
```

## Sources

- `/Users/callumcowie/repos/Aether/go.mod` -- Existing dependency inventory (Go 1.26.1, cobra v1.10.2, no database drivers)
- `/Users/callumcowie/repos/Aether/cmd/init_cmd.go` -- Current `aether init` implementation (222 lines), idempotency checks, sealed-colony detection
- `/Users/callumcowie/repos/Aether/cmd/init_research.go` -- Existing `init-research` command (598 lines) with `detectGovernance()`, `analyzeGitHistory()`, `generatePheromoneSuggestions()`, `generateCharter()`, `detectPriorColonies()`, 10 pheromone patterns, 12 project type detectors, 23 governance detectors
- `/Users/callumcowie/repos/Aether/cmd/autofix.go` -- Checkpoint/rollback pattern for safe state mutation (105 lines)
- `/Users/callumcowie/repos/Aether/cmd/midden_cmds.go` -- CRUD subcommand registration pattern (10 subcommands)
- `/Users/callumcowie/repos/Aether/cmd/update_cmd.go` -- Publish/update pipeline with stale-publish detection (533 lines), `checkStalePublish()` parity checks
- `/Users/callumcowie/repos/Aether/cmd/platform_sync.go` -- `repoSyncPairs()` defining 9 sync pairs across 3 platforms
- `/Users/callumcowie/repos/Aether/cmd/codex_visuals.go` -- ANSI rendering infrastructure for caste identity, stage markers
- `/Users/callumcowie/repos/Aether/cmd/colony_prime_context.go` -- Section-based context injection with priority ordering
- `/Users/callumcowie/repos/Aether/cmd/pheromone_write.go` -- Pheromone write with SHA-256 deduplication
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md` -- Wrapper init command calling `aether init-research` then `aether init`
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-context.md` -- Build playbook with suggest-analyze deprecation guard (lines 149-181)
- `/Users/callumcowie/repos/Aether/pkg/storage/storage.go` -- Store API with `AtomicWrite`, `SaveJSON`, `LoadJSON`, file locking
