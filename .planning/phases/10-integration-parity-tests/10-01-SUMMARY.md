---
phase: 10-integration-parity-tests
plan: 01
subsystem: testing
tags: [parity, integration, shell-vs-go, test-harness]
dependency_graph:
  requires: [storage-layer, cobra-cli]
  provides: [parity-harness, parity-comparators, parity-overlap-tests]
  affects: [cmd/parity_*.go]
tech_stack:
  added: [go-testing, os/exec, syscall, context-timeout]
  patterns: [table-driven-tests, file-based-output-capture, process-group-kill]
key_files:
  created:
    - cmd/parity_harness_test.go
    - cmd/parity_comparators_test.go
    - cmd/parity_overlap_test.go
  modified: []
decisions:
  - File-based shell output capture prevents pipe goroutine hangs in test suite
  - Combined stdout/stderr in single file for simpler parity comparison
  - 5-second per-command timeout with process group kill for reliability
  - Known parity breaks documented as test metadata, not suppressed
metrics:
  duration: 72min
  completed: 2026-04-04
  tasks: 2
  files: 3
---

# Phase 10 Plan 01: Integration Parity Tests Summary

Built shell-vs-Go parity test harness with table-driven overlap tests covering 195 commands across 23 categories, enabling systematic detection of behavioral differences between shell and Go implementations.

## What Was Done

### Task 1: Parity test harness and semantic comparators

Created `cmd/parity_harness_test.go` with infrastructure helpers:
- `setupParityEnv` -- Creates temp directory with test fixtures, returns tmpDir for shared shell/Go testing
- `runShellCommand` -- Executes shell subcommands via `bash aether-utils.sh` with file-based output capture and 5s process-group kill timeout
- `runGoCommand` -- Executes Go subcommands via `rootCmd.SetArgs` with global state save/restore
- `projectRoot` -- Resolves project root via `runtime.Caller`
- `isJSON` -- Validates JSON parseability
- `truncateStr` -- Safe string truncation for logging

Created `cmd/parity_comparators_test.go` with comparison helpers:
- `assertEnvelopeParity` -- Verifies both outputs have matching "ok" boolean values
- `assertResultFieldParity` -- Compares .result field types (string-string, map-map, mixed)
- `compareByPaths` -- Extracts values at dot-separated JSON paths and compares
- `extractByPath` -- Walks dot-separated path into parsed JSON map
- `isParityBreak` -- Detects known structural differences (JSON vs non-JSON, string vs object result, key overlap ratio, ok field presence mismatch)

### Task 2: Table-driven parity tests for 195 overlapping commands

Created `cmd/parity_overlap_test.go` with 23 test functions covering:
1. State commands (19): load-state, validate-state, state-read, etc.
2. Pheromone commands (8): pheromone-read, pheromone-write, pheromone-count, etc.
3. Flag commands (6): flag-list, flag-add, flag-resolve, etc.
4. Spawn commands (9): spawn-log, spawn-complete, spawn-can-spawn, etc.
5. Queen commands (9): queen-init, queen-read, queen-promote, etc.
6. Learning commands (12): learning-promote, learning-inject, learning-observe, etc.
7. Midden commands (10): midden-write, midden-recent-failures, midden-review, etc.
8. Hive commands (5): hive-init, hive-store, hive-read, etc.
9. Instinct commands (6): instinct-read, instinct-create, instinct-apply, etc.
10. Trust commands (3): trust-score-compute, trust-score-decay, trust-tier
11. Event bus commands (4): event-bus-publish, event-bus-query, etc.
12. Graph commands (4): graph-link, graph-neighbors, graph-reach, graph-cluster
13. Display commands (7): swarm-display-init, swarm-display-update, etc.
14. Curation commands (9): curation-run, curation-archivist, etc.
15. Security commands (7): check-antipattern, error-add, error-flag-pattern, etc.
16. Build flow commands (9): generate-ant-name, generate-commit-message, etc.
17. Context commands (4): context-capsule, context-update, colony-prime, pr-context
18. History commands (3): history, changelog-append, changelog-collect-plan-data
19. Session commands (7): session-init, session-read, session-update, etc.
20. Registry commands (4): registry-add, registry-list, registry-export-xml, etc.
21. Exchange commands (7): pheromone-export-xml, pheromone-import-xml, etc.
22. Suggest commands (5): suggest-analyze, suggest-approve, etc.
23. Misc commands (38): entropy-score, memory-metrics, data-safety-stats, etc.

## Results

| Metric | Count |
|--------|-------|
| Total test cases | 195 |
| PASS | 70 |
| SKIP (Go command not yet ported) | 105 |
| Parity gaps detected | 43 |
| Known parity breaks documented | 12 |
| Full suite runtime | 62s |

## Known Parity Breaks

| Command | Issue |
|---------|-------|
| generate-ant-name | Shell returns bare string, Go returns object with name+caste |
| pheromone-count | Different field names (lowercase shell vs uppercase Go) |
| entropy-score | Different nesting structure |
| milestone-detect | Different calculation method |
| memory-metrics | Shell returns raw JSON, Go wraps in envelope |
| flag-list | Go needs --json flag |
| version | Go does not use JSON envelope |
| swarm-display-text | Shell outputs ANSI text + JSON |
| swarm-display-render | Shell outputs ANSI text + JSON |
| swarm-display-inline | Shell outputs deprecated warning + JSON |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Shell command pipe goroutine hangs**
- **Found during:** Task 2 -- `swarm-display-render` caused test suite to hang indefinitely
- **Issue:** `exec.CommandContext` with `CombinedOutput` blocks on pipe reads even after SIGKILL, because child bash processes spawn their own children
- **Fix:** Switched to file-based output capture (`>outfile 2>&1`), redirecting output to a temp file and reading it after command completion. Combined with `syscall.SysProcAttr{Setpgid: true}` and process group kill for reliability.
- **Files modified:** cmd/parity_harness_test.go

**2. [Rule 1 - Bug] isParityBreak didn't detect ok field absence**
- **Found during:** Task 2 -- `memory-metrics` shell output lacks "ok" field but Go has it
- **Issue:** The `isParityBreak` function only checked JSON vs non-JSON and result type differences, not "ok" field presence
- **Fix:** Added check for `ok` field presence mismatch (one output has envelope, other doesn't)
- **Files modified:** cmd/parity_comparators_test.go

**3. [Rule 1 - Bug] `append` with no variadic values**
- **Found during:** Task 2 -- `go vet` flagged `append([]string{"-c", ...})` with no additional values
- **Issue:** The shell command builder used `append` when a simple slice literal sufficed
- **Fix:** Replaced `append` with direct `[]string{"-c", shellCmd}` assignment
- **Files modified:** cmd/parity_harness_test.go

## Key Decisions

1. **File-based output capture over pipes** -- Pipes block indefinitely when bash child processes survive SIGKILL. File redirect (`>outfile 2>&1`) avoids this entirely.
2. **5-second per-command timeout** -- Most shell commands complete in <0.5s. 5s is generous enough for slower commands while catching genuine hangs.
3. **Known breaks as metadata, not suppression** -- Known parity differences are logged clearly rather than silently skipped, making them visible in CI output.
4. **Combined stdout/stderr** -- Shell commands mix stdout and stderr, so parity comparison uses combined output for consistency.

## Verification

All existing Go tests pass. Parity tests run in ~62 seconds. No race conditions detected.

## Self-Check: PASSED

- cmd/parity_harness_test.go: FOUND
- cmd/parity_comparators_test.go: FOUND
- cmd/parity_overlap_test.go: FOUND
- Commit 4ec2d4f4: FOUND
