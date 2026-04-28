# Phase 71: Platform Hardening - Context

**Gathered:** 2026-04-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix cross-platform gaps across Claude Code, OpenCode, and Codex CLI so all three platforms produce consistent, correct output for every command. This covers CLI flag mismatches, dispatch manifest correctness, command output verification, and OpenCode parity audit.

Requirements: PLAT-01, PLAT-02, PLAT-03, PLAT-04, PLAT-05.

</domain>

<decisions>
## Implementation Decisions

### PLAT-01/02: OpenCode Shelf Parity (Audit)
- **D-01:** Both OpenCode init.md and entomb.md already include shelf backlog/archive sections and are byte-identical to Claude's (added during Phase 65 idea shelving work). However, keep these requirements in scope for a deeper audit — verify that the Go runtime behind the markdown actually supports shelf operations correctly when called from OpenCode's command surface.
- **D-02:** The audit should check: does `aether shelf-list`, `aether shelf-promote-batch`, and `aether shelf-dismiss-batch` work when invoked from the OpenCode wrapper flow?

### PLAT-03: Platform Dispatch Verification
- **D-03:** PLAT-03's requirement ("Codex subagent dispatch works correctly") is misphrased. The real concern is that dispatch works correctly for whatever AI the user is running in a session — Claude, OpenCode, Codex, or any other LLM client.
- **D-04:** Scope is runtime manifest correctness only: ensure the Go runtime generates correct dispatch manifests (JSON) for all agent types. Each platform handles its own agent routing from the manifest.
- **D-05:** Do NOT test actual platform-specific agent dispatch (e.g., Codex's subagent system) — that's the platform's responsibility, not Aether's.

### PLAT-04: CLI Flag Mismatch Fix
- **D-06:** Fix approach: Add all missing flags/subcommands to the Go runtime to match what the markdown commands expect. The markdown represents the intended API — do NOT rewrite markdown call sites.
- **D-07:** Full fix scope: all 120+ broken CLI calls across all command files and playbooks. After this phase, pheromones, memory, midden, spawn tracking, and activity logging must all be functional.
- **D-08:** Implementation approach: one subcommand at a time. Add flags for each subcommand, write/update tests, commit, then move to the next. Safer than all-at-once.
- **D-09:** Affected systems (from memory note): pheromone signals (16+ calls), memory/learning pipeline (15+ calls), midden failure tracking (8+ calls), spawn tracking (20+ calls), activity logging (15+ calls), flag/blocker system (10+ calls), registry (4+ calls), plus 6 subcommands that don't exist at all.

### PLAT-05: Cross-Platform Testing
- **D-10:** Build an automated smoke test that runs each Go subcommand and checks it exits cleanly and produces expected output. This becomes part of the test suite, runs on every commit.
- **D-11:** The smoke test covers the Go CLI surface — the single source of truth. If Go works correctly, the platform wrappers (Claude/OpenCode) will work correctly since they delegate to Go.

### Existing Uncommitted Work
- **D-12:** There are 20+ modified cmd/ files and 7 new untracked files (process tracker, worker cleanup, process group handling) already in the working tree. The planner MUST inspect these changes, understand what they implement, and incorporate them into the phase plan rather than planning from scratch.
- **D-13:** Key new files to incorporate: `cmd/codex_worker_cleanup.go`, `pkg/codex/process_tracker.go`, `pkg/codex/process_tracker_test.go`, `pkg/codex/process_group_unix.go`, `pkg/codex/process_group_windows.go`, `cmd/verification_process_group_unix.go`, `cmd/verification_process_group_windows.go`, `cmd/worker_cleanup_signal_common.go`, `cmd/worker_cleanup_signal_unix.go`, `cmd/worker_cleanup_signal_windows.go`.

### Claude's Discretion
- Exact order of subcommand flag fixes (which subcommand first)
- Smoke test structure and assertion granularity
- How to handle the 6 subcommands that don't exist at all (create new commands vs alias to existing ones)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` -- PLAT-01 through PLAT-05 define the five platform hardening requirements

### Roadmap
- `.planning/ROADMAP.md` -- Phase 71 goal, success criteria, dependency chain

### Platform Architecture
- `CLAUDE.md` -- Platform policy (primary/secondary), UX architecture, wrapper-runtime contract
- `.aether/docs/wrapper-runtime-ux-contract.md` -- Full contract for how wrappers delegate to Go runtime
- `RUNTIME UPDATE ARCHITECTURE.md` -- Full distribution flow

### CLI Flag Mismatch Analysis
- The systemic CLI flag mismatch was discovered 2026-04-11. Key missing flags: `--worker`, `--source`, `--reason`, `--ttl`, `--section`, `--key`, `--content`, `--path`, `--goal`, `--tags`, `--id`, `--description`, `--summary`, `--caste`. Key missing subcommands: 6 subcommands that don't exist but are called from markdown.

### Command Sources
- `.claude/commands/ant/*.md` -- 50 Claude Code slash commands (primary platform)
- `.opencode/commands/ant/*.md` -- 50 OpenCode slash commands (secondary platform)
- `.codex/CODEX.md` -- Codex commands and rules (Codex platform)

### Playbooks (contain CLI calls)
- `.aether/docs/command-playbooks/build-prep.md`
- `.aether/docs/command-playbooks/build-context.md`
- `.aether/docs/command-playbooks/build-wave.md`
- `.aether/docs/command-playbooks/build-verify.md`
- `.aether/docs/command-playbooks/build-complete.md`
- `.aether/docs/command-playbooks/continue-verify.md`
- `.aether/docs/command-playbooks/continue-gates.md`
- `.aether/docs/command-playbooks/continue-advance.md`
- `.aether/docs/command-playbooks/continue-finalize.md`

### Go Runtime (authoritative CLI surface)
- `cmd/root.go` -- CLI entry point, all subcommands registered here
- `cmd/codex_build.go` -- Build command with dispatch and caste-to-agent mapping
- `cmd/codex_continue.go` -- Continue command with worker lifecycle

### Existing Uncommitted Work (MUST inspect)
- `cmd/codex_worker_cleanup.go` -- New: stale worker cleanup before dispatch
- `pkg/codex/process_tracker.go` -- New: worker process tracking
- `pkg/codex/process_tracker_test.go` -- New: process tracker tests
- `pkg/codex/process_group_unix.go` -- New: Unix process group management
- `pkg/codex/process_group_windows.go` -- New: Windows process group management

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/codex_build.go` -- `codexAgentNameForCaste()` and `codexAgentFileForCaste()` handle caste-to-TOML mapping for Codex dispatch
- `cmd/codex_visuals.go` -- Caste identity system (colors, emojis, names) used across all platforms
- `.aether/commands/*.yaml` -- YAML source definitions that generate identical command wrappers

### Established Patterns
- Commands already achieve parity via YAML source generation (49/50 identical between Claude and OpenCode; only help.md differs by one line)
- Go runtime is authoritative for CLI surface — markdown wrappers delegate to Go
- All broken CLI calls are wrapped in `2>/dev/null || true` making failures invisible

### Integration Points
- `cmd/root.go` -- All CLI subcommands registered here; new subcommands and flags added here
- `.claude/commands/ant/continue.md` -- 11 CLI calls, many with broken flags (highest density of broken calls)
- `.aether/docs/command-playbooks/*.md` -- 9 playbook files with CLI calls that need flag verification

### Platform Parity Status
| Surface | Claude | OpenCode | Codex |
|---------|--------|----------|-------|
| Commands (50) | Primary | 49/50 identical | Runtime-native |
| Agents (25) | Canonical | Mirror | Platform-specific TOML |
| init.md shelf section | Has it | Has it (identical) | N/A |
| entomb.md shelf section | Has it | Has it (identical) | N/A |

</code_context>

<specifics>
## Specific Ideas

- User clarified that PLAT-03 should NOT be about Codex-specific subagent dispatch — it should cover "whatever AI you are running in a session." The fix is ensuring runtime dispatch manifests are correct for all agent types, regardless of which platform consumes them.
- User confirmed full fix scope for CLI flags — all 120+ broken calls fixed, all systems working after this phase.
- User wants one-subcommand-at-a-time approach for adding missing flags to Go — safer, easier to isolate issues.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 71-platform-hardening*
*Context gathered: 2026-04-28*
