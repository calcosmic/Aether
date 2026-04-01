# Codebase Concerns

**Analysis Date:** 2026-04-01

## Shell-to-Go Migration Concerns

### Go skeleton is nearly empty -- only 2 of 8 packages have real code
- Issue: Six of eight Go packages are empty stubs (package declaration only, no types or functions)
- Files: `pkg/agent/agent.go`, `pkg/events/events.go`, `pkg/graph/graph.go`, `pkg/llm/llm.go`, `pkg/memory/memory.go`, `internal/config/config.go`, `internal/testing/testing.go`
- Impact: The Go rewrite has not begun in earnest. These packages represent the core domains: agent management, event bus, knowledge graph, LLM integration, memory/wisdom pipeline, and configuration.
- Fix approach: Prioritize porting `pkg/storage/storage.go` (already complete) and `pkg/colony/colony.go` (types + state machine done) patterns into the remaining packages. The shell implementations in `.aether/utils/` serve as behavioral specifications.

### Go CLI entry point is a no-op
- Issue: `cmd/aether/main.go` contains `func main() {}` -- nothing happens when the binary runs
- Files: `cmd/aether/main.go`
- Impact: The Go binary cannot replace `bin/cli.js` or `.aether/aether-utils.sh` in any capacity
- Fix approach: Wire cobra/urfave CLI, implement subcommand dispatch, port critical paths first (init, state-read, state-write, pheromone-write)

### Go test for real colony state is brittle
- Issue: `TestRoundTripRealColonyState` hardcodes expectations (`current_phase: 2`, `version: "3.0"`) against the live `COLONY_STATE.json`. The test currently fails because `current_phase` is 4, not 2.
- Files: `pkg/colony/colony_test.go:119-154`
- Impact: Test breaks every time colony state changes. Creates noise in CI.
- Fix approach: Make the test assertion-free for structural fields (just verify parse/unmarshal round-trip succeeds) or use a fixture file instead of the live data file.

### Three version numbering schemes coexist
- Issue: `package.json` says `5.3.2`, `CLAUDE.md` says `v2.8.0` (header) and `v2.7.0` (table), `COLONY_STATE.json` says `"3.0"`, and `CHANGELOG.md` starts at `[5.3.0]`. Four different version numbers in four files.
- Files: `package.json`, `CLAUDE.md`, `.aether/data/COLONY_STATE.json`, `CHANGELOG.md`
- Impact: Impossible to determine actual release version. Documentation accuracy is undermined. The Go module (`go.mod`) has no version at all.
- Fix approach: Establish single source of truth. `package.json` is the canonical npm version. `COLONY_STATE.json` version should track colony lifecycle stages (seal/entomb counter), not product version. `CLAUDE.md` should read from `package.json`.

---

## Shell Script Fragility

### 30,000+ lines of shell across 57 scripts with 130+ subcommands
- Issue: The entire colony runtime is implemented in bash with 57 shell scripts totaling 30,114 lines. The main dispatcher (`aether-utils.sh`) alone is 5,642 lines with 336 case arms. This is the core motivation for the Go rewrite.
- Files: `.aether/aether-utils.sh` (5,642 lines), `.aether/utils/pheromone.sh` (3,297), `.aether/utils/learning.sh` (2,000), `.aether/utils/queen.sh` (1,708)
- Impact: Hard to reason about correctness. No compiler. Error handling depends on `set -euo pipefail` and manual `trap` setup. New contributors face a steep learning curve.
- Fix approach: Incremental Go porting. Start with the storage layer (`storage.go` is done), then state API, then pheromone system.

### 1,207 error suppressions (`2>/dev/null`) across shell scripts
- Issue: Over 1,200 instances of `2>/dev/null` error suppression. While many are annotated with `SUPPRESS:OK` comments (added in Phase 10 error triage), the sheer volume makes it impossible to audit for silently swallowed errors.
- Files: All `.aether/utils/*.sh` (worst: `aether-utils.sh` at 207, `scan.sh` at ~76, `suggest.sh` at ~54, `queen.sh` at ~60, `session.sh` at ~41)
- Impact: Real errors can be hidden among intentional suppressions. Debugging production issues requires reading through hundreds of suppression sites.
- Fix approach: Go's explicit error handling eliminates this class of problem entirely. During migration, ensure every shell `2>/dev/null` maps to an explicit Go error check or `os.IsNotExist` guard.

### Heavy jq dependency -- hundreds of invocations per script
- Issue: The shell scripts depend critically on `jq` for JSON manipulation. The top consumers: `scan.sh` (~76 jq calls), `suggest.sh` (~54), `queen.sh` (~60), `pheromone.sh` (~40), `swarm.sh` (~35), `skills.sh` (~43), `session.sh` (~41), `state-api.sh` (~28). If `jq` is missing or produces unexpected output, subcommands fail silently or produce malformed JSON.
- Files: All `.aether/utils/*.sh`
- Impact: External dependency that must be present on every developer's machine. jq version differences can cause subtle parsing failures. No validation that jq output is valid before passing it to downstream commands.
- Fix approach: Go's `encoding/json` eliminates jq dependency entirely. The `storage.go` package already handles JSON marshaling/unmarshaling natively.

### Only 13 of 57 shell scripts have strict mode
- Issue: Only the main dispatcher (`aether-utils.sh`) and 12 other scripts use `set -e` or `set -euo pipefail`. The remaining 44 utility scripts are sourced into the dispatcher and rely on its strict mode, but if any are run standalone (e.g., during testing or debugging), errors go undetected.
- Files: `.aether/utils/*.sh` (most files)
- Impact: Standalone execution of utility scripts can silently fail or produce partial results
- Fix approach: Go enforces error checking at compile time and runtime. Each package is independently safe.

### Fallback atomic_write bypasses JSON validation
- Issue: The fallback `atomic_write` in `aether-utils.sh` (lines 104-112) writes content via `echo > temp; mv temp target` without any JSON validation. This fallback activates when `atomic-write.sh` fails to source. Corrupted JSON could be written to COLONY_STATE.json without detection.
- Files: `.aether/aether-utils.sh:104-112`
- Impact: Data corruption if the fallback path is triggered during a partial installation
- Fix approach: The Go `storage.go` implementation validates JSON on every write for `.json` files. This is already solved.

### Lock ownership is caller-managed, not library-managed
- Issue: `atomic-write.sh` documents that callers must manage lock release via EXIT traps (BUG-006 in known-issues.md). If a caller forgets to set up a trap, or the trap fails, locks remain held indefinitely.
- Files: `.aether/utils/atomic-write.sh`, `.aether/utils/file-lock.sh` (313 lines of lock management)
- Impact: Deadlocks requiring manual lock file deletion (`rm .aether/data/*.lock`)
- Fix approach: Go's `defer` pattern and `storage.go`'s built-in mutex management eliminate this. The `Store` type uses per-path `sync.RWMutex` with automatic release.

### Placeholder build summary line
- Issue: Line 952 in `aether-utils.sh` contains `# What didn't (placeholder - would come from midden)` indicating an incomplete implementation in the build summary generation.
- Files: `.aether/aether-utils.sh:952`
- Impact: Build summaries are missing failure information from the midden system
- Fix approach: Implement midden integration in the build summary, or port to Go where the data flow can be properly wired.

---

## Compatibility Risks

### npm CLI (bin/cli.js) duplicates shell logic in JavaScript
- Issue: `bin/cli.js` (2,223 lines) reimplements colony initialization, state synchronization, and file operations in JavaScript using Node.js. `bin/lib/` contains 5,443 lines across 14 JS files including `update-transaction.js` (1,709 lines), `file-lock.js` (695 lines), `state-guard.js` (602 lines), and `state-sync.js` (516 lines). These must stay in sync with the shell implementations.
- Files: `bin/cli.js`, `bin/lib/*.js`
- Impact: Three implementations of the same logic (shell, Node.js CLI, and soon Go). Any behavioral change must be replicated in all three. Inconsistencies between implementations are a real risk.
- Fix approach: Go binary should replace both `bin/cli.js` and `.aether/aether-utils.sh`. Until migration is complete, `bin/cli.js` should delegate to shell scripts rather than reimplementing logic.

### 12 deprecated subcommands still shipped
- Issue: 12 subcommands emit deprecation warnings (`_deprecation_warning`) but remain functional and shipped in the package. They are marked for removal in "v3.0" but no migration timeline exists.
- Files: `.aether/aether-utils.sh` (search for `_deprecation_warning`)
- Impact: Dead code increases package size, adds attack surface, and confuses users. The "v3.0" removal target is ambiguous given the version numbering chaos.
- Fix approach: Set a firm removal date. Add migration guide for each deprecated command. Remove in the Go rewrite.

### OpenCode agent parity maintenance burden
- Issue: Every agent and command must be maintained in three places: `.claude/commands/ant/`, `.claude/agents/ant/`, and `.opencode/commands/ant/` + `.opencode/agents/`. YAML-based generation (`src/commands/_meta/*.yaml`, `bin/generate-commands.js`) exists but only covers 6 of ~45 commands.
- Files: `.claude/commands/ant/` (45 files), `.opencode/commands/ant/` (45 files), `.claude/agents/ant/` (24 files), `.opencode/agents/` (24 files), `src/commands/_meta/*.yaml`
- Impact: High maintenance burden. Any command change requires manual edits in 2-3 places. Parity drift is likely.
- Fix approach: YAML-based generation should cover all commands, not just 6. Or the Go CLI should serve as the single source and generate provider-specific formats.

### COLONY_STATE.json schema has no formal specification
- Issue: The colony state JSON schema is defined implicitly by the Go types in `pkg/colony/colony.go` and the shell scripts' jq usage. There is no JSON Schema, no formal validation layer, and no migration system for schema changes.
- Files: `pkg/colony/colony.go`, `.aether/utils/state-api.sh`
- Impact: Schema changes can break the shell scripts silently. The Go round-trip test (`TestRoundTripRealColonyState`) fails because the real data has drifted from test expectations.
- Fix approach: Define a JSON Schema for COLONY_STATE.json. Add schema validation to `state-api.sh`. Use the Go types as the canonical schema definition.

---

## Known Bugs (from known-issues.md)

### BUG-006: Lock not released on JSON validation failure
- Symptoms: If JSON validation fails in `atomic_write`, the temp file is cleaned but any lock held by the caller is not released
- Files: `.aether/utils/atomic-write.sh`
- Trigger: Write malformed JSON to a locked file
- Workaround: Callers must use trap-based cleanup (documented but not enforced)
- Status: Open

### ISSUE-005: Potential infinite loop in spawn-tree
- Symptoms: Edge case with circular parent chain in spawn tree
- Files: `.aether/utils/spawn.sh` (`spawn-tree-depth`)
- Trigger: Manually corrupted spawn tree data with circular references
- Workaround: Safety limit of 5 depth exists
- Status: Open, low risk

### ISSUE-006: Fallback json_err loses error codes
- Symptoms: If `error-handler.sh` fails to load, the fallback `json_err` in `aether-utils.sh` does not accept error code parameters
- Files: `.aether/aether-utils.sh:67-82`
- Trigger: Corrupted installation where error-handler.sh is missing
- Workaround: None -- error codes silently lost
- Status: Open, low risk

---

## Performance Concerns

### Every subcommand sources 35+ shell scripts
- Issue: The main dispatcher sources 35 utility scripts on every invocation (lines 27-65 of `aether-utils.sh`). This includes all domain modules, XML utilities, curation ants, and infrastructure modules regardless of which subcommand is being called.
- Files: `.aether/aether-utils.sh:27-65`
- Impact: Every `aether-utils.sh` invocation pays the startup cost of parsing 30,000+ lines of bash. For frequently called subcommands (e.g., `state-read`, `pheromone-read`), this adds significant latency.
- Fix approach: Go binary eliminates startup overhead entirely. For interim improvement, consider lazy-sourcing only the modules needed for the requested subcommand.

### jq subprocess overhead per JSON operation
- Issue: Each `jq` invocation spawns a new process. Scripts like `scan.sh` (~76 calls) and `suggest.sh` (~54 calls) spawn dozens of subprocesses per invocation.
- Files: `.aether/utils/scan.sh`, `.aether/utils/suggest.sh`
- Impact: Measurable latency on complex operations. Particularly slow on macOS where process creation is expensive.
- Fix approach: Go's native JSON handling is in-process with zero subprocess overhead.

### Colony state file grows unbounded
- Issue: COLONY_STATE.json accumulates events, learnings, instincts, error records, and graveyard entries without any size limit or pruning mechanism. Over long-lived colonies, this file can grow to hundreds of KB.
- Files: `.aether/data/COLONY_STATE.json`, `.aether/utils/state-api.sh`
- Impact: Every `state-read` and `state-write` must parse the full file. JSON manipulation via jq becomes slower as the file grows.
- Fix approach: Add archival/pruning to the consolidation pipeline. Move old events and completed graveyards to separate files.

---

## Test Coverage Gaps

### Go packages have zero tests for 6 of 8 packages
- Issue: Only `pkg/colony` and `pkg/storage` have test files. Six packages (`agent`, `events`, `graph`, `llm`, `memory`, `config`, `testing`) have no tests at all.
- Files: All `pkg/*/` and `internal/*/` directories except `pkg/colony/` and `pkg/storage/`
- Risk: As Go implementation grows, untested packages will accumulate bugs
- Priority: High -- test infrastructure should be established before significant Go code is written

### Shell test for colony round-trip is broken
- Issue: `TestRoundTripRealColonyState` in `pkg/colony/colony_test.go:135` expects `current_phase: 2` but the actual file has `current_phase: 4`. The test fails on `go test ./...`.
- Files: `pkg/colony/colony_test.go:135`
- Risk: CI will show false failures, reducing trust in the test suite
- Priority: Medium -- fix by making assertions dynamic or using a fixture

### Error path test coverage is incomplete
- Issue: `known-issues.md` (GAP-008) documents that error handling paths are not fully tested despite Phase 12 adding state-api tests. The 92 bash test files cover happy paths well but edge cases (malformed JSON, missing files, permission errors, lock contention) are undertested.
- Files: `.aether/docs/known-issues.md` (GAP-008)
- Risk: Error handling regressions go undetected
- Priority: Medium

### No integration tests for shell-to-Go compatibility
- Issue: There are no tests verifying that the Go `storage.go` implementation produces identical output to the shell `atomic-write.sh` when operating on the same files.
- Files: None exist
- Risk: Subtle behavioral differences between shell and Go implementations could cause data corruption during migration
- Priority: High -- needed before any production Go usage

---

## Scaling Limits

### Single-process shell architecture prevents parallelism
- Issue: The entire colony system runs as a single bash process with file-based locking. The TO-DOS.md explicitly marks "Multi-Ant Parallel Execution" as "DO NOT IMPLEMENT" without discussion. File locking (`file-lock.sh`, 313 lines) uses `flock`-style locking with PID-based stale detection.
- Files: `.aether/utils/file-lock.sh`, `TO-DOS.md` (Colony Lifecycle section)
- Current capacity: One colony operation at a time per machine
- Limit: Cannot run parallel builders, watchers, or scouts without risking COLONY_STATE.json corruption
- Scaling path: Go's goroutines + the `storage.go` mutex-based locking enable safe concurrent access. The per-path `sync.RWMutex` in `Store` already supports concurrent readers with exclusive writers.

### File-based IPC limits cross-process communication
- Issue: Colony components communicate through JSON files (COLONY_STATE.json, pheromones.json, events.jsonl). No socket, pipe, or shared memory communication exists.
- Files: `.aether/data/*.json`, `.aether/data/*.jsonl`
- Current capacity: Sufficient for single-user, single-colony workflow
- Limit: High-latency communication for real-time operations. File contention under concurrent access.
- Scaling path: Go enables in-process communication via channels, or external communication via gRPC/HTTP.

---

## Dependencies at Risk

### jq is a hard runtime dependency with no fallback
- Issue: All JSON manipulation in shell scripts depends on `jq`. No graceful degradation exists. If jq is missing, subcommands produce empty/garbled output.
- Files: All `.aether/utils/*.sh` (~800+ jq invocations total)
- Risk: Installation on minimal systems without jq fails silently
- Migration plan: Go eliminates this dependency entirely

### Bash version assumptions
- Issue: Scripts use bash-specific features (arrays, `[[ ]]`, `mapfile`, process substitution, `BASH_SOURCE`) that require bash 4+. macOS ships bash 3.2 by default due to GPL v3 licensing.
- Files: All `.aether/utils/*.sh`
- Risk: Scripts fail on default macOS bash. Users must install bash 5+ via Homebrew.
- Migration plan: Go binary has no bash dependency

### Node.js required for CLI but not for core operations
- Issue: `bin/cli.js` requires Node.js >= 16 for installation and update operations. Core colony operations (via `aether-utils.sh`) only need bash + jq. The Go rewrite should eliminate the Node.js dependency entirely.
- Files: `package.json` (engines: node >= 16.0.0), `bin/cli.js`
- Risk: Adding a runtime dependency for what is fundamentally a shell tool
- Migration plan: Go binary is self-contained with no runtime dependencies beyond the OS

---

*Concerns audit: 2026-04-01*
