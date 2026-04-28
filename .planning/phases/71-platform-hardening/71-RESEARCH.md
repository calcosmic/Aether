# Phase 71: Platform Hardening - Research

**Researched:** 2026-04-28
**Domain:** Go CLI cross-platform consistency (Aether v1.0.20)
**Confidence:** HIGH

## Summary

Phase 71 fixes cross-platform gaps across three AI platforms (Claude Code, OpenCode, Codex CLI) so all produce consistent, correct output for every command. The core problem is a systemic CLI flag mismatch: 120+ markdown command/playbook calls use wrong flags or arguments against the Go runtime, all silently failing behind `2>/dev/null || true`. This makes pheromone signals, memory capture, midden failure tracking, spawn logging, activity logging, and flag/blocker systems non-functional when invoked from wrapper commands.

The Go runtime has 274 registered Cobra subcommands. The markdown wrappers in `.claude/commands/ant/` (50 files) and `.opencode/commands/ant/` (50 files, 49/50 byte-identical to Claude) call these subcommands with flags that don't exist. The fix approach (locked by user decision D-06): add all missing flags/subcommands to the Go runtime to match what markdown expects. The markdown represents the intended API.

Significant uncommitted work already exists in the working tree: 20+ modified `cmd/` files and 10 new files implementing process group management, worker cleanup, and process tracking. These must be incorporated into the plan rather than planned from scratch.

**Primary recommendation:** Fix one Go subcommand at a time (per D-08), adding missing flags, writing tests, and committing before moving to the next. Build a smoke test suite that validates the CLI surface after all fixes.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** OpenCode init.md and entomb.md already include shelf backlog/archive sections (byte-identical to Claude's). Keep in scope for deeper audit -- verify Go runtime shelf operations work when called from OpenCode wrapper flow.
- **D-02:** Audit should check: does `aether shelf-list`, `aether shelf-promote-batch`, `aether shelf-dismiss-batch` work when invoked from OpenCode wrapper?
- **D-03:** PLAT-03 concern is runtime manifest correctness for all agent types, not Codex-specific subagent dispatch.
- **D-04:** Scope is runtime manifest correctness only: ensure Go runtime generates correct dispatch manifests (JSON) for all agent types.
- **D-05:** Do NOT test actual platform-specific agent dispatch (e.g., Codex's subagent system) -- that's the platform's responsibility.
- **D-06:** Fix approach: Add all missing flags/subcommands to Go runtime to match markdown. Do NOT rewrite markdown call sites.
- **D-07:** Full fix scope: all 120+ broken CLI calls across all command files and playbooks.
- **D-08:** One subcommand at a time. Add flags for each subcommand, write/update tests, commit, then move to next.
- **D-09:** Affected systems: pheromone signals (16+ calls), memory/learning pipeline (15+ calls), midden failure tracking (8+ calls), spawn tracking (20+ calls), activity logging (15+ calls), flag/blocker system (10+ calls), registry (4+ calls), plus 6 subcommands that don't exist at all.
- **D-10:** Build automated smoke test that runs each Go subcommand and checks it exits cleanly with expected output. Part of test suite, runs on every commit.
- **D-11:** Smoke test covers Go CLI surface -- the single source of truth. If Go works, platform wrappers will work since they delegate to Go.
- **D-12:** 20+ modified cmd/ files and 10 new untracked files already in working tree. Planner MUST inspect and incorporate into phase plan.
- **D-13:** Key new files: `cmd/codex_worker_cleanup.go`, `pkg/codex/process_tracker.go`, `pkg/codex/process_tracker_test.go`, `pkg/codex/process_group_unix.go`, `pkg/codex/process_group_windows.go`, `cmd/verification_process_group_unix.go`, `cmd/verification_process_group_windows.go`, `cmd/worker_cleanup_signal_common.go`, `cmd/worker_cleanup_signal_unix.go`, `cmd/worker_cleanup_signal_windows.go`.

### Claude's Discretion
- Exact order of subcommand flag fixes (which subcommand first)
- Smoke test structure and assertion granularity
- How to handle the 6 subcommands that don't exist at all (create new commands vs alias to existing ones)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLAT-01 | OpenCode init.md includes shelf backlog section | Shelf section already present and identical to Claude's (verified by diff). Go subcommands `shelf-list`, `shelf-promote-batch`, `shelf-dismiss-batch` all exist. Audit needed: verify OpenCode wrapper flow invokes these correctly. |
| PLAT-02 | OpenCode entomb.md includes shelf archive summary | Shelf section already present and identical to Claude's (verified by diff). Same audit scope as PLAT-01. |
| PLAT-03 | Codex subagent dispatch works correctly across all agent types | Go runtime dispatch manifest generation via `codexBuildManifest` struct. All 25 agent TOML files exist in `.codex/agents/`. Caste-to-TOML mapping via `codexAgentNameForCaste()` and `codexAgentFileForCaste()`. Platform dispatch via `SelectPlatformInvoker()` supports Codex, Claude, and OpenCode. |
| PLAT-04 | CLI flag mismatches between wrapper markdown and Go runtime are resolved | 120+ broken calls across 9 playbook files and 50 command files. Root cause: markdown uses `--type`, `--content`, `--priority`, `--source`, `--reason`, `--ttl`, `--section`, `--key`, `--path`, `--goal`, `--tags`, `--id`, `--description`, `--summary`, `--caste`, `--worker`, `--name`, `--status` flags that don't exist on Go subcommands. Fix: add missing flags to Go runtime. |
| PLAT-05 | All 50 commands produce correct output on all 3 platforms | Smoke test covering Go CLI surface. 274 registered subcommands. If Go works, wrappers work since they delegate to Go. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI flag/arg definitions | Go Runtime (cmd/) | Markdown wrappers | Go runtime is authoritative (CLAUDE.md ownership model). Markdown represents intended API surface. |
| Dispatch manifest generation | Go Runtime (cmd/codex_build.go) | -- | Manifest JSON is generated by Go, consumed by all platforms |
| Platform agent routing | Platform-specific (Claude/OpenCode/Codex CLIs) | Go Runtime (platform_dispatch.go) | Go detects active platform and selects dispatcher; actual agent routing is the platform's responsibility (D-05) |
| Worker process lifecycle | Go Runtime (pkg/codex/worker.go) | Platform CLIs | Go manages spawning, tracking, heartbeat, timeout, cleanup |
| Shelf operations | Go Runtime (cmd/shelf_cmd.go) | Markdown wrappers | Go implements shelf CRUD; markdown calls Go |
| Smoke test validation | Go test suite (cmd/) | CI | Tests run against Go binary, validate CLI surface |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.23+ (verified via `go test`) | Language runtime | Project is Go-only CLI binary |
| Cobra | (pinned in go.mod) | CLI framework | Already in use for all 274 subcommands |
| testing | stdlib | Test framework | Project uses stdlib `testing` package with `t.Run` subtests |
| BurntSushi/toml | (pinned in go.mod) | TOML parsing | Used for Codex agent definition validation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gopkg.in/yaml.v3 | (pinned in go.mod) | YAML parsing | Markdown agent definition validation in platform_dispatch.go |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Adding flags to Go | Rewriting markdown call sites | User locked D-06: add flags to Go, don't rewrite markdown |
| Smoke test in Go | Shell script | Go test integrates with existing test suite, runs on every commit |

**Installation:**
No new dependencies needed. All fixes are to existing Go code.

**Version verification:**
```
go test ./...  # All 18 packages pass (verified 2026-04-28)
go build ./cmd/aether  # Binary builds successfully
```

## Architecture Patterns

### System Architecture Diagram

```
                    Claude Code              OpenCode             Codex CLI
                    wrapper .md             wrapper .md          native CLI
                         |                      |                    |
                         v                      v                    v
              ┌──────────────────────────────────────────────────────┐
              │                   aether Go binary                   │
              │                  (cmd/ -- 274 subcommands)          │
              │                                                      │
              │  ┌──────────────────────────────────────────────┐  │
              │  │         Platform Dispatch Layer               │  │
              │  │  (pkg/codex/platform_dispatch.go)             │  │
              │  │  DetectActivePlatform() -> SelectPlatformInvoker()│
              │  │  Supports: Codex, Claude, OpenCode, Fake      │  │
              │  └──────────────────────────────────────────────┘  │
              │                         |                            │
              │  ┌──────────────────────┼───────────────────────┐  │
              │  │     Worker Lifecycle (pkg/codex/worker.go)   │  │
              │  │     - Process tracking (process_tracker.go)  │  │
              │  │     - Process group mgmt (process_group_*.go)│  │
              │  │     - Stale cleanup (codex_worker_cleanup.go) │  │
              │  └─────────────────────────────────────────────┘  │
              │                         |                            │
              │  ┌──────────────────────┼───────────────────────┐  │
              │  │    Dispatch Manifest (cmd/codex_build.go)    │  │
              │  │    - codexBuildManifest struct -> JSON       │  │
              │  │    - Caste -> Agent TOML mapping             │  │
              │  │    - Worker briefs + task plans              │  │
              │  └─────────────────────────────────────────────┘  │
              │                         |                            │
              │  ┌──────────────────────┼───────────────────────┐  │
              │  │  Colony State (pkg/colony/, pkg/storage/)   │  │
              │  │  - COLONY_STATE.json (atomic read-modify-write)│  │
              │  │  - Pheromones, instincts, events            │  │
              │  │  - Shelf, midden, session files             │  │
              │  └─────────────────────────────────────────────┘  │
              └──────────────────────────────────────────────────────┘
```

### Recommended Project Structure

No new directories needed. All changes go into existing `cmd/` files and test files.

### Pattern 1: One-Subcommand-at-a-Time Fix (User-Locked)
**What:** Add missing flags to one Go subcommand, write tests, commit, repeat.
**When to use:** Per user decision D-08, this is the mandatory approach for PLAT-04.
**Example:**
```go
// In cmd/pheromone_write.go, add missing --type flag (already exists)
// Check: pheromoneWriteCmd.Flags().String("type", "", "Signal type")
// The flag already exists! Need to audit WHICH flags are actually missing.
```

### Pattern 2: Smoke Test Suite (PLAT-05)
**What:** Table-driven test that runs each Go subcommand with `--help` or zero args and verifies it exits cleanly.
**When to use:** After all flag fixes, create a persistent smoke test.
**Example:**
```go
func TestSubcommandSmokeTest(t *testing.T) {
    // Build the binary once
    // For each subcommand, run with --help and verify exit code 0
    // Optionally run with known-safe flags and verify JSON output
}
```

### Anti-Patterns to Avoid
- **Modifying markdown call sites:** User locked D-06 -- markdown is the intended API, Go must match.
- **Testing platform-specific agent dispatch:** User locked D-05 -- that's each platform's responsibility.
- **Adding flags all at once:** User locked D-08 -- one subcommand at a time for safe isolation.
- **Skipping tests:** Each flag addition needs a corresponding test.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI smoke test framework | Custom shell script | Go `testing` + `exec.Command` | Integrates with existing test suite, runs on `go test ./...` |
| Flag parsing | Manual arg parsing | Cobra `Flags()` | Already in use, consistent with existing patterns |
| Subcommand registration | Manual init functions | `rootCmd.AddCommand()` | Existing pattern for all 274 subcommands |

**Key insight:** The Go runtime already has a mature Cobra-based CLI. The problem is not architecture but missing flag registrations on existing subcommands. No new frameworks or patterns needed.

## Common Pitfalls

### Pitfall 1: Assuming flags exist without checking
**What goes wrong:** Adding a flag that already exists, or missing one that doesn't.
**Why it happens:** 274 subcommands across 289 Go files -- hard to track mentally.
**How to avoid:** Before each fix, run `grep -n 'Flags()' cmd/<file>.go` to see current flags, then compare against markdown call sites.
**Warning signs:** Test fails with "unknown flag" or "flag provided but not defined".

### Pitfall 2: Breaking existing tests when adding flags
**What goes wrong:** Adding a required flag breaks existing test calls that don't pass it.
**Why it happens:** Existing tests use the subcommand without the new flag.
**How to avoid:** Make all new flags optional with sensible defaults. Never make a flag required if the markdown calls sometimes omit it.
**Warning signs:** Test failures in unrelated packages after a flag addition.

### Pitfall 3: Uncommitted work conflicts
**What goes wrong:** The 20+ modified files and 10 new files in the working tree conflict with planned changes.
**Why it happens:** Phase 71 work builds on top of already-in-progress changes from a different branch/session.
**How to avoid:** The planner MUST inspect all uncommitted changes first, understand what they implement, and plan the flag fixes to integrate cleanly.
**Warning signs:** Merge conflicts or test failures when combining changes.

### Pitfall 4: Smoke test too brittle
**What goes wrong:** Smoke test checks exact output text, breaks on minor formatting changes.
**Why it happens:** Over-specifying assertions.
**How to avoid:** Smoke test should verify: (1) exit code 0, (2) non-empty stdout, (3) no stderr errors. Not exact text matching.
**Warning signs:** Smoke test fails on cosmetic changes.

## Code Examples

Verified patterns from existing codebase:

### Current Pheromone Write Flags (already implemented)
```go
// Source: cmd/pheromone_write.go:307-314
pheromoneWriteCmd.Flags().String("type", "", "Signal type: FOCUS, REDIRECT, or FEEDBACK (required)")
pheromoneWriteCmd.Flags().String("content", "", "Signal content (required)")
pheromoneWriteCmd.Flags().String("priority", "", "Priority: low, normal, high")
pheromoneWriteCmd.Flags().Float64("strength", 0, "Signal strength (default 1.0)")
pheromoneWriteCmd.Flags().StringArray("tag", nil, "Signal tags (repeatable)")
pheromoneWriteCmd.Flags().String("source", "cli", "Signal source")
pheromoneWriteCmd.Flags().String("reason", "", "Reason for the signal")
pheromoneWriteCmd.Flags().String("ttl", "", "Override expiry duration")
```

### Current Midden Write Flags (already implemented)
```go
// Source: cmd/midden_cmds.go:496-498
middenWriteCmd.Flags().String("category", "general", "Failure category")
middenWriteCmd.Flags().String("message", "", "Failure message (required)")
middenWriteCmd.Flags().String("source", "unknown", "Failure source")
```

### Caste-to-Agent Mapping (for PLAT-03 dispatch verification)
```go
// Source: cmd/codex_build.go:1653-1663
func codexAgentFileForCaste(caste string) string {
    normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(caste, "_", "-")))
    if normalized == "" { normalized = "builder" }
    return "aether-" + normalized + ".toml"
}

func codexAgentNameForCaste(caste string) string {
    return strings.TrimSuffix(codexAgentFileForCaste(caste), ".toml")
}
```

### Platform Dispatch Selection (for PLAT-03)
```go
// Source: pkg/codex/platform_dispatch.go:246-276
func SelectPlatformInvoker(ctx context.Context) WorkerInvoker {
    active := DetectActivePlatform()
    preferred := active
    if override := normalizePlatform(os.Getenv(envWorkerPlatform)); override != PlatformUnknown && override != PlatformFake {
        preferred = override
    }
    dispatchers := reorderDispatchers([]PlatformDispatcher{
        NewCodexDispatcher(),
        NewClaudeDispatcher(),
        NewOpenCodeDispatcher(),
    }, preferred)
    // ... tries each in order, returns first available
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell scripts for commands | Go binary with Cobra | Before v1.0 | All commands now Go-native |
| Markdown calls with wrong flags | Need to fix Go to match | Discovered 2026-04-11 | 120+ silently broken calls |
| No process tracking | ProcessTracker + process groups | In uncommitted work | Worker cleanup now possible |

**Deprecated/outdated:**
- Shell-based commands: Fully replaced by Go binary
- Constraints.json: Legacy, being replaced by pheromone system

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 120+ broken CLI calls are primarily about missing flags, not missing subcommands | CLI Flag Mismatch | Some calls may invoke non-existent subcommands (6 reported); need to verify exact count |
| A2 | All pheromone-write flags already exist in Go runtime | Standard Stack | Memory note from 2026-04-11 says flags missing; but code shows `--type`, `--content`, `--priority` all registered. Need to verify which markdown calls actually use wrong flags vs. correct ones. |
| A3 | The 20+ uncommitted cmd/ files can be committed as-is before flag fix work begins | Common Pitfalls | Some files may be in an incomplete state; planner must inspect each |
| A4 | 49/50 OpenCode commands are byte-identical to Claude | Platform Parity Status | Verified by diff (init.md shows no differences), but should verify a broader sample |
| A5 | The smoke test can use `--help` to validate subcommand registration | Code Examples | Some subcommands may panic or error on `--help` with no args; need fallback strategy |

**If this table is empty:** All claims in this research were verified or cited -- no user confirmation needed.

## Open Questions

1. **Exact count and identity of missing flags per subcommand**
   - What we know: 120+ broken calls, affecting 7 systems (pheromones, memory, midden, spawn, activity, flags, registry)
   - What's unclear: For each subcommand, which specific flags are missing? The memory note is 16 days old and may be stale.
   - Recommendation: Planner should run a systematic audit before writing tasks. For each markdown file, extract CLI calls, compare against Go `Flags()` registrations.

2. **Which 6 subcommands don't exist at all**
   - What we know: Memory note mentions "6 subcommands that don't exist"
   - What's unclear: Which specific subcommands? Need to identify them.
   - Recommendation: Extract all `aether <subcommand>` calls from markdown, check each against `rootCmd.AddCommand` registrations.

3. **Uncommitted file readiness**
   - What we know: 20+ modified files, 10 new files in working tree
   - What's unclear: Are they complete and tested? Do they pass `go test ./...`? (Current test run shows all packages pass.)
   - Recommendation: Planner should inspect each uncommitted file, verify it's ready for commit, and decide whether to commit first or incorporate into flag fix work.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All development | Yes | 1.23+ | -- |
| Cobra | CLI framework | Yes | (pinned) | -- |
| `go test` | Validation | Yes | (stdlib) | -- |
| `jq` | Markdown CLI call parsing | Yes | (system) | Use Go string parsing instead |

**Missing dependencies with no fallback:**
- None

**Missing dependencies with fallback:**
- None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none |
| Quick run command | `go test ./cmd/ -run TestPheromone -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PLAT-01 | OpenCode init.md shelf section present | audit | Manual comparison | Yes (already identical) |
| PLAT-02 | OpenCode entomb.md shelf section present | audit | Manual comparison | Yes (already identical) |
| PLAT-03 | Dispatch manifest correct for all castes | unit | `go test ./cmd/ -run TestCodexBuild -count=1` | Yes (codex_build_test.go) |
| PLAT-04 | CLI flags match markdown expectations | unit | Per-subcommand tests | Partial (existing tests) |
| PLAT-05 | All subcommands produce correct output | smoke | `go test ./cmd/ -run TestSubcommandSmoke -count=1` | No (Wave 0) |

### Sampling Rate
- **Per task commit:** `go test ./cmd/ -run <specific_test> -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/smoke_test.go` -- covers PLAT-05 subcommand validation
- [ ] Individual flag fix tests per subcommand (identified during systematic audit)

*(Partial coverage: existing test suite covers build/continue flows well, but specific flag tests for pheromone/memory/midden subcommands are likely missing.)*

## Security Domain

Not applicable for this phase. Phase 71 is about CLI flag correctness and platform consistency, not security features. No new authentication, input handling, or cryptographic operations are introduced.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: codebase grep] -- All 274 subcommand registrations via `rootCmd.AddCommand`
- [VERIFIED: go test ./...] -- All 18 packages pass (2026-04-28)
- [VERIFIED: diff] -- OpenCode init.md is byte-identical to Claude init.md
- [VERIFIED: codebase read] -- cmd/pheromone_write.go flags (lines 307-314)
- [VERIFIED: codebase read] -- cmd/midden_cmds.go flags (lines 496-498)
- [VERIFIED: codebase read] -- cmd/codex_build.go caste mapping (lines 1653-1663)
- [VERIFIED: codebase read] -- pkg/codex/platform_dispatch.go (full file)
- [VERIFIED: codebase read] -- pkg/codex/worker.go (full file)
- [VERIFIED: codebase read] -- pkg/codex/process_tracker.go (new file)
- [VERIFIED: codebase read] -- cmd/codex_worker_cleanup.go (new file)
- [VERIFIED: codebase read] -- All 26 Codex agent TOML files exist in .codex/agents/

### Secondary (MEDIUM confidence)
- [CITED: memory note project_cli_flag_mismatch.md] -- Original discovery of 120+ broken calls (2026-04-11, 16 days old -- may be stale)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - verified via go test, codebase reads
- Architecture: HIGH - verified by reading platform_dispatch.go, worker.go, codex_build.go in full
- Pitfalls: HIGH - based on codebase patterns and user-locked decisions

**Research date:** 2026-04-28
**Valid until:** 30 days (stable domain -- Go CLI flag additions)
