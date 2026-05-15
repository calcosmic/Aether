---
phase: "135"
plan: "135-01"
plan_name: "Doc-CLI Alignment — Discovery and Core Test"
subsystem: "Smoke Test / CLI Validation"
tags: ["smoke-test", "cli-alignment", "yaml-parser", "cobra", "host-critical"]
dependency_graph:
  requires: []
  provides: ["DCA-01", "DCA-02"]
  affects: ["platform-health.json", "release-gate"]
tech_stack:
  added: ["pkg/smoke/"]
  patterns: ["YAML auto-discovery", "cobra introspection", "tiered severity"]
key_files:
  created:
    - pkg/smoke/types.go
    - pkg/smoke/yaml_parser.go
    - pkg/smoke/yaml_parser_test.go
    - pkg/smoke/cobra_probe.go
    - pkg/smoke/cobra_probe_test.go
    - pkg/smoke/host_critical_flags.go
    - pkg/smoke/host_critical_flags_test.go
    - cmd/doc_cli_alignment_test.go
  modified: []
decisions:
  - "lifecycle.yaml does not exist in repo, so only 6 host commands detected (not 7)"
  - "Host-critical commands produce blocking severity; non-host produce warnings"
  - "platform-health.json preserves existing keys while adding doc_cli_alignment section"
  - "CrossReferenceWithTSHost scans both test files and host.ts source for flag coverage"
metrics:
  duration: "~25 minutes"
  completed_date: "2026-05-15"
---

# Phase 135 Plan 135-01: Doc-CLI Alignment — Discovery and Core Test Summary

**One-liner:** Auto-discover all 60 YAML command definitions, introspect Cobra flag registrations, and run a tiered alignment smoke test that blocks the release gate on host-critical mismatches.

## What Was Built

### Task 1: YAML Parser Helper (`pkg/smoke/yaml_parser.go`)
- `ParseYAMLCommands(glob)` reads all `.aether/commands/*.yaml` files and extracts command names and flag mentions.
- Scans `wrapper_additions`, `guardrails`, `runtime.command`, `intent_refinement`, `follow_up`, and `codex_orchestration` fields for `--flag-name` patterns.
- Heuristic flag extraction: flags count if they appear as standalone tokens (whitespace-delimited) or inside markdown code blocks.
- `IsHost` detection for 6 host commands: plan, build, continue, oracle, watch, swarm.
- Deduplication within each command's `Flags` slice.
- `go test ./pkg/smoke -run TestParseYAMLCommands` passes with >=60 commands, no duplicates, correct host classification.

### Task 2: Cobra Introspection Helper (`pkg/smoke/cobra_probe.go`)
- `ProbeCobraFlags(root)` walks all subcommands via `root.Commands()`, collects each command's `LocalFlags().VisitAll(...)` to extract flag names.
- Returns a map: command name → sorted list of registered flag names (long form only).
- Excludes the auto-added `help` flag.
- Host commands (those with `DisableFlagParsing: true`) get empty flag lists since no flags are registered in Go.
- `go test ./pkg/smoke -run TestProbeCobraFlags` passes.

### Task 3: Host-Critical Flag Curation (`pkg/smoke/host_critical_flags.go`)
- `HostCriticalFlags` map loaded from the 6 host command YAML files at init time.
- `HostCriticalCombinations` map covers TCV-01 through TCV-04 requirements:
  - plan: `--depth balanced --planning-depth standard`
  - continue: `--verification-depth heavy`
  - watch: `--no-dashboard`
  - swarm: `--no-dashboard`
- `CrossReferenceWithTSHost(yamlFlags)` scans `.aether/ts-host/test/host-flags.test.ts`, `host-integration.test.ts`, and `.aether/ts-host/src/host.ts` for flag references.
- Returns "documented but untested" flags as warnings.
- `go test ./pkg/smoke -run TestHostCriticalFlags` passes.

### Task 4: Alignment Smoke Test (`cmd/doc_cli_alignment_test.go`)
- `TestDocCLIAlignment` runs the full alignment check:
  1. Calls `smoke.ParseYAMLCommands(".aether/commands/*.yaml")`
  2. Calls `smoke.ProbeCobraFlags(rootCmd)`
  3. For each non-host YAML command: checks each flag against cobra probe map. Missing = `warning` mismatch.
  4. For each host-critical command: checks each flag. Missing = `blocking` mismatch. Also calls `CrossReferenceWithTSHost` and logs "documented but untested" flags as additional warnings.
  5. Writes `platform-health.json` preserving existing `failed_commands` and `flag_mismatches` keys while adding `doc_cli_alignment` object with `commands_checked`, `host_critical_checked`, `host_critical_failures`, and `warnings`.
  6. If any blocking mismatches exist, calls `t.Fatalf`.
- `TestDocCLIAlignmentBlockingMismatch` validates the severity logic directly.
- `go test ./cmd -run TestDocCLIAlignment` passes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed host command count from 7 to 6**
- **Found during:** Task 1 (acceptance criteria verification)
- **Issue:** The plan expected exactly 7 host commands (plan, build, continue, oracle, watch, swarm, lifecycle), but `lifecycle.yaml` does not exist in `.aether/commands/`.
- **Fix:** Updated `hostCommandNames` map and test expectations to 6 host commands. `ant-lifecycle` is not present in the YAML source.
- **Files modified:** `pkg/smoke/yaml_parser.go`, `pkg/smoke/yaml_parser_test.go`
- **Commit:** f0e8aa93

**2. [Rule 3 - Blocking Issue] Fixed `cobra.Flag` → `pflag.Flag` type mismatch**
- **Found during:** Task 2 (compilation)
- **Issue:** `cobra_probe.go` used `cobra.Flag` in the `VisitAll` callback, but Cobra's `LocalFlags()` returns `*pflag.FlagSet` which uses `*pflag.Flag`.
- **Fix:** Changed type to `*pflag.Flag` and added the `github.com/spf13/pflag` import.
- **Files modified:** `pkg/smoke/cobra_probe.go`
- **Commit:** 073ca8e4

## Known Stubs

| File | Line | Description | Reason |
|------|------|-------------|--------|
| `pkg/smoke/host_critical_flags.go` | 38 | `HostCriticalCombinations` hard-codes TCV combinations | These are requirement-derived test cases, not auto-discoverable from source. Future work could parse test files to auto-populate. |
| `pkg/smoke/host_critical_flags.go` | 140 | `CrossReferenceWithTSHost` returns warnings for flags in YAML but not in TS tests | This is intentional behavior — "documented but untested" flags are warnings, not failures. |

## Threat Flags

No new security-relevant surface introduced. The smoke test is read-only (parses YAML, introspects flags, writes a health JSON file to a test temp directory). No network endpoints, auth paths, or schema changes at trust boundaries.

## Self-Check: PASSED

- [x] `pkg/smoke/types.go` exists
- [x] `pkg/smoke/yaml_parser.go` exists
- [x] `pkg/smoke/yaml_parser_test.go` exists
- [x] `pkg/smoke/cobra_probe.go` exists
- [x] `pkg/smoke/cobra_probe_test.go` exists
- [x] `pkg/smoke/host_critical_flags.go` exists
- [x] `pkg/smoke/host_critical_flags_test.go` exists
- [x] `cmd/doc_cli_alignment_test.go` exists
- [x] Commit f0e8aa93 exists (`feat(135-01): add YAML parser...`)
- [x] Commit 073ca8e4 exists (`feat(135-01): add cobra introspection probe...`)
- [x] Commit af1d4962 exists (`feat(135-01): add host-critical flag curation...`)
- [x] Commit 27041526 exists (`feat(135-01): add doc-CLI alignment smoke test`)
- [x] `go test ./pkg/smoke/...` passes
- [x] `go test ./cmd -run TestDocCLIAlignment` passes
- [x] `go vet ./pkg/smoke/... ./cmd/...` clean
- [x] `go build ./cmd/aether` succeeds
