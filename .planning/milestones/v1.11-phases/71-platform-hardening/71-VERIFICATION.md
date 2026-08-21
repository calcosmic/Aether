---
phase: 71-platform-hardening
verified: 2026-04-28T17:30:00Z
status: gaps_found
score: 7/12 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Uncommitted process tracking and worker cleanup code is committed and tested"
    status: failed
    reason: "The 71-01 files (process_tracker.go, process_group_unix.go, codex_worker_cleanup.go, etc.) exist in the working tree but are NOT committed on the current branch (codex/fix-opencode-subagent-dispatch). The 71-01 commit (cdb008fb) is on main but is NOT an ancestor of the current branch HEAD. Git status shows all 71-01 artifact files as modified/untracked."
    artifacts:
      - path: "pkg/codex/process_tracker.go"
        issue: "Untracked file in working tree, not committed on current branch"
      - path: "pkg/codex/process_group_unix.go"
        issue: "Untracked file in working tree, not committed on current branch"
      - path: "cmd/codex_worker_cleanup.go"
        issue: "Untracked file in working tree, not committed on current branch"
      - path: "cmd/verification_process_group_unix.go"
        issue: "Untracked file in working tree, not committed on current branch"
    missing:
      - "Commit the 71-01 foundation files on the current branch (cherry-pick cdb008fb or merge main)"
  - truth: "state-mutate --verify-only and --revert flags are functional"
    status: failed
    reason: "Both flags are registered (lines 860-861 in state_cmds.go) but the RunE function (lines 26-53) never reads or handles them. No code checks for 'verify-only' or 'revert' flag values. The flags are dead -- setting them has no effect on command behavior."
    artifacts:
      - path: "cmd/state_cmds.go"
        issue: "Flags registered at lines 860-861 but RunE (lines 26-53) has no verify-only/revert handling"
    missing:
      - "Add verifyOnly, _ := cmd.Flags().GetBool('verify-only') handling in stateMutateCmd RunE"
      - "Add revert, _ := cmd.Flags().GetString('revert') handling in stateMutateCmd RunE"
  - truth: "All 25 agent types produce valid dispatch manifests via codexBuildManifest"
    status: failed
    reason: "Plan 71-01 specified creating a test that iterates over all 25 caste names calling codexAgentFileForCaste and codexAgentNameForCaste. Only 1 agent type (ambassador) is tested in codex_build_test.go line 414. No test validates all 25 castes produce correct TOML filenames."
    artifacts:
      - path: "cmd/codex_build_test.go"
        issue: "Only tests codexAgentNameForCaste for ambassador caste (line 414), not all 25"
    missing:
      - "Create test that iterates all 25 castes and validates codexAgentFileForCaste returns non-empty .toml filename"
  - truth: "suggest-approve subcommand produces real data (not hardcoded empty)"
    status: partial
    reason: "suggest-approve returns hardcoded empty suggestions array ([]interface{}{}). Command is callable and registered (PLAT-04 satisfied) but does not produce real data. This is a compatibility stub."
    artifacts:
      - path: "cmd/compatibility_cmds.go"
        issue: "suggestApproveCmd RunE returns hardcoded empty suggestions"
    missing:
      - "Wire suggest-approve to read actual pending suggestions from colony state"
  - truth: "chamber-compare subcommand produces real data (not hardcoded empty)"
    status: partial
    reason: "chamber-compare returns hardcoded empty matches/diffs arrays. Command is callable and registered (PLAT-04 satisfied) but does not perform actual comparison."
    artifacts:
      - path: "cmd/chamber.go"
        issue: "chamberCompareCmd RunE returns hardcoded empty matches and diffs"
    missing:
      - "Wire chamber-compare to read chamber archive and compare against current state"
  - truth: "PLAT-05: All 50 commands produce correct output on all 3 platforms"
    status: partial
    reason: "Smoke test validates all 318 Go subcommands respond to --help (VERIFIED). However, PLAT-05 requires verifying correct output on all 3 platforms (Claude Code, OpenCode, Codex CLI). Go runtime validation covers the shared backend but platform-specific output rendering cannot be verified programmatically."
    artifacts:
      - path: "cmd/smoke_test.go"
        issue: "Tests Go subcommands only; does not test Claude Code slash commands or OpenCode commands"
    missing:
      - "Human verification needed: test 50 slash commands on each platform for correct output"
human_verification:
  - test: "Run 10 representative slash commands on Claude Code platform"
    expected: "Each command produces expected output without errors"
    why_human: "Platform-specific rendering and markdown wrapper behavior cannot be tested via Go tests"
  - test: "Run 10 representative slash commands on OpenCode platform"
    expected: "Each command produces expected output matching Claude Code behavior"
    why_human: "OpenCode wrapper parity requires interactive testing"
  - test: "Verify Codex CLI dispatches all 25 agent types without errors"
    expected: "Each agent type spawns correctly with proper TOML configuration"
    why_human: "Codex CLI is a separate binary interface that requires running codex commands"
  - test: "Verify state-mutate --verify-only behavior manually"
    expected: "Command checks guard precondition and reports result without mutating state"
    why_human: "Flag is registered but not wired -- needs manual confirmation of expected vs actual behavior"
---

# Phase 71: Platform Hardening Verification Report

**Phase Goal:** All three platforms (Claude Code, OpenCode, Codex CLI) produce consistent, correct output for every command
**Verified:** 2026-04-28T17:30:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1   | Uncommitted process tracking and worker cleanup code is committed and tested | FAILED | 71-01 files exist in working tree but are NOT committed on current branch. Commit cdb008fb is on main only, not an ancestor of HEAD. |
| 2   | All 25 agent types produce valid dispatch manifests via codexBuildManifest | FAILED | Only 1 of 25 castes tested (ambassador). No test iterates all castes via codexAgentFileForCaste. |
| 3   | Shelf subcommands work correctly from Go CLI | VERIFIED | shelf-list (with --status, --json), shelf-add, shelf-promote-batch, shelf-dismiss-batch all registered with correct flags in shelf_cmd.go and shelf_init.go. |
| 4   | All existing tests pass after committing uncommitted work | VERIFIED | `go test ./... -count=1` passes all 18 packages. |
| 5   | Systematic audit proves which CLI flags are missing | VERIFIED | TestCLIFlagAudit scans 108 unique markdown CLI calls against 316 Go registrations. Test passes with 0 mismatches. |
| 6   | All missing subcommands registered | VERIFIED | suggest-approve (3 refs), versions (2 refs), chamber-compare (3 refs), flag-create alias (1 ref), council parent (6 refs) all registered. |
| 7   | Any remaining flag gaps found by audit are fixed | VERIFIED | 8 flag gaps fixed: state-mutate --verify-only/--revert (registered only), flag-list --phase, midden-cross-pr-analysis --window, pheromone-merge-back --export-file, pheromone-expire --phase-end-only, learning-approve-proposals --deferred/--verbose, continue --synthetic. |
| 8   | Smoke test covers all registered subcommands with --help validation | VERIFIED | TestSubcommandSmokeTest passes all 318 subcommand --help checks. TestNewSubcommandFlags validates 5 new commands with flags. |
| 9   | RESEARCH.md open questions resolved with evidence from audit | VERIFIED | Audit test provides concrete evidence: 108 markdown CLI calls matched against 316 Go registrations, proving flag coverage. |
| 10  | state-mutate --verify-only and --revert flags are functional | FAILED | Flags registered (lines 860-861) but RunE (lines 26-53) never reads them. Dead flags with no behavioral effect. |
| 11  | PLAT-01: OpenCode init.md includes shelf backlog section | VERIFIED | OpenCode init.md lines 53-67 include shelf backlog section matching Claude Code exactly. |
| 12  | PLAT-02: OpenCode entomb.md includes shelf archive summary | VERIFIED | OpenCode entomb.md lines 18-19 include shelf summary line matching Claude Code. |

**Score:** 9/12 truths verified (3 failed, 0 partial counted in score)

### Deferred Items

No deferred items. None of the identified gaps are covered by later milestone phases (72-76 focus on init charter, rich research, suggest-analyze, intelligence, and UX -- not platform hardening).

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `pkg/codex/process_tracker.go` | Worker process tracking | VERIFIED | 404 lines, all functions: TrackProcess, UntrackProcess, KillProcess, KillAll, DetectStaleWorkers |
| `pkg/codex/process_group_unix.go` | Unix process group management | VERIFIED | 47 lines, setpgid, SIGTERM/SIGKILL, process existence checks |
| `cmd/codex_worker_cleanup.go` | Stale worker cleanup | VERIFIED | 32 lines, calls codex.CleanupStaleWorkers, wired into build/plan/colonize/continue |
| `cmd/state_cmds.go` | state-mutate --verify-only and --revert | STUB | Flags registered (lines 860-861) but NOT handled in RunE (lines 26-53) |
| `cmd/cli_flag_audit_test.go` | Systematic flag audit | VERIFIED | 188 lines, scans markdown, compares against Go registrations, passes |
| `cmd/smoke_test.go` | PLAT-05 smoke test | VERIFIED | 81 lines, validates all 318 subcommands with --help |
| `cmd/compatibility_cmds.go` | Missing subcommand stubs | VERIFIED | suggest-approve and versions registered (stub implementations) |
| `cmd/council.go` | Council parent command | VERIFIED | Parent registered, backward-compatible dual registration |
| `cmd/chamber.go` | chamber-compare subcommand | VERIFIED | Registered (stub implementation) |
| `cmd/flag_cmds.go` | flag-create alias | VERIFIED | Alias registered for flag-add |

### Key Link Verification

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `cmd/codex_worker_cleanup.go` | `pkg/codex/process_tracker.go` | `codex.CleanupStaleWorkers(root)` | WIRED | Called in codex_build.go:898, codex_plan.go:673, codex_colonize.go:437, codex_continue.go:852,1109 |
| `cmd/root.go` | `cmd/codex_worker_cleanup.go` | `setupWorkerCleanupHandler()` in init() | WIRED | root.go:145 calls setupWorkerCleanupHandler, defined in worker_cleanup_signal_unix.go:13 |
| `cmd/cli_flag_audit_test.go` | `cmd/*.go` | Parses markdown CLI calls, checks Go registrations | WIRED | Audit test reads real markdown files and compares against real Cobra registrations |
| `.claude/commands/ant/*.md` | `cmd/compatibility_cmds.go` | `aether suggest-approve/versions` | WIRED | Markdown calls match registered Go subcommands |
| `cmd/state_cmds.go` --verify-only | stateMutateCmd RunE | Flag -> handler | NOT_WIRED | Flag registered but never read in RunE function |
| `cmd/state_cmds.go` --revert | stateMutateCmd RunE | Flag -> handler | NOT_WIRED | Flag registered but never read in RunE function |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `cmd/cli_flag_audit_test.go` | registered (map) | rootCmd.Commands() | FLOWING | Reads real Cobra registrations at test runtime |
| `cmd/cli_flag_audit_test.go` | markdown CLI calls | os.ReadDir + regex | FLOWING | Reads real markdown files, extracts real CLI patterns |
| `cmd/smoke_test.go` | subcommands | rootCmd.Commands() | FLOWING | Iterates real registered commands |
| `cmd/compatibility_cmds.go` suggest-approve | suggestions | hardcoded []interface{}{} | STATIC | Always returns empty array regardless of state |
| `cmd/chamber.go` chamber-compare | matches, diffs | hardcoded []interface{}{} | STATIC | Always returns empty arrays regardless of state |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| CLI flag audit passes | `go test ./cmd/ -run TestCLIFlagAudit -count=1` | PASS (0 mismatches, 108 subcommands found) | PASS |
| Subcommand registration check | `go test ./cmd/ -run TestCLIFlagAuditSubcommandsRegistered -count=1` | PASS (all 5 subcommands + alias) | PASS |
| Smoke test all subcommands | `go test ./cmd/ -run TestSubcommandSmokeTest -count=1` | PASS (318 subcommands) | PASS |
| New subcommand flags | `go test ./cmd/ -run TestNewSubcommandFlags -count=1` | PASS (5 commands) | PASS |
| Full test suite | `go test ./... -count=1` | PASS (18 packages) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PLAT-01 | 71-01 | OpenCode init.md includes shelf backlog section | SATISFIED | OpenCode init.md lines 53-67 match Claude Code |
| PLAT-02 | 71-01 | OpenCode entomb.md includes shelf archive summary | SATISFIED | OpenCode entomb.md lines 18-19 match Claude Code |
| PLAT-03 | 71-01 | Codex subagent dispatch works correctly across all agent types | BLOCKED | ProcessTracker + worker cleanup are substantive but uncommitted on current branch; dispatch test only covers 1/25 castes |
| PLAT-04 | 71-02 | CLI flag mismatches resolved | SATISFIED | Audit test proves 108 markdown calls match Go registrations; 5 missing subcommands registered; 8 flag gaps fixed |
| PLAT-05 | 71-02 | All 50 commands produce correct output on all 3 platforms | NEEDS HUMAN | Go smoke test passes for 318 subcommands; cross-platform output verification requires human testing |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `cmd/state_cmds.go` | 860-861 | Dead flags: --verify-only and --revert registered but never handled | Blocker | Flags silently ignored, misleading users into thinking they work |
| `cmd/compatibility_cmds.go` | suggestApproveCmd | Hardcoded empty suggestions return | Warning | suggest-approve always returns empty, no real data flows |
| `cmd/chamber.go` | chamberCompareCmd | Hardcoded empty matches/diffs return | Warning | chamber-compare always returns empty, no real comparison |

### Human Verification Required

### 1. Cross-platform output verification

**Test:** Run 10 representative slash commands on Claude Code, OpenCode, and Codex CLI
**Expected:** Each command produces correct, consistent output across all 3 platforms
**Why human:** Platform-specific rendering and markdown wrapper behavior cannot be tested via Go tests

### 2. Codex dispatch for all 25 agent types

**Test:** Trigger Codex CLI dispatch for each of the 25 agent types
**Expected:** Each agent type spawns correctly with proper TOML configuration
**Why human:** Codex CLI is a separate binary interface that requires running codex commands

### 3. state-mutate --verify-only behavior

**Test:** Run `aether state-mutate --guard task-complete:1 --verify-only`
**Expected:** Command reports guard check result without mutating state
**Why human:** Flag is registered but not wired to behavior -- need to confirm expected behavior

### Gaps Summary

Three BLOCKER gaps prevent phase goal achievement:

1. **71-01 files uncommitted on current branch**: The process tracking, worker cleanup, and process group files exist in the working tree but are not committed on the `codex/fix-opencode-subagent-dispatch` branch. They were committed on `main` (cdb008fb) but that commit is not an ancestor of the current branch. This means PLAT-03 (Codex subagent dispatch) cannot be verified as complete.

2. **state-mutate --verify-only and --revert flags are dead**: Both flags are registered on the Cobra command but the RunE function never reads them. They have zero behavioral effect. Users calling `aether state-mutate --verify-only --guard task-complete:1` will get the same behavior as without --verify-only -- the mutation will still execute.

3. **Dispatch manifest test covers only 1 of 25 agent types**: Plan 71-01 specified creating a test iterating all 25 castes through codexAgentFileForCaste/codexAgentNameForCaste. Only ambassador is tested. Without this test, there is no automated verification that dispatch manifests work for all agent types.

Additionally, two WARNING-level stubs (suggest-approve and chamber-compare return hardcoded empty data) are noted. These satisfy PLAT-04 (commands are callable) but don't produce real data.

---

_Verified: 2026-04-28T17:30:00Z_
_Verifier: Claude (gsd-verifier)_
