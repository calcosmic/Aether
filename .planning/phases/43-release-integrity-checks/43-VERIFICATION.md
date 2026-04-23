---
phase: 43-release-integrity-checks
verified: 2026-04-23T18:15:00Z
status: gaps_found
score: 5/6 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Medic flags incomplete stable/dev publishes and prints exact recovery command (Roadmap SC 2, REL-02)"
    status: failed
    reason: "Plan 43-02 (medic integration + tests) was created but never executed. scanIntegrity() does not exist in medic_scanner.go. aether medic --deep does not include integrity findings."
    artifacts:
      - path: "cmd/medic_scanner.go"
        issue: "No scanIntegrity() function; no integrity category in deep scan results"
      - path: "cmd/integrity_cmd_test.go"
        issue: "File does not exist; zero test coverage for integrity command"
    missing:
      - "Add scanIntegrity() function to cmd/medic_scanner.go"
      - "Wire scanIntegrity() into performHealthScan when opts.Deep is true"
      - "Create cmd/integrity_cmd_test.go with unit and E2E tests"
      - "Add TestMedicDeepIncludesIntegrity test"
---

# Phase 43: Release Integrity Checks and Diagnostics Verification Report

**Phase Goal:** Implement the `aether integrity` CLI command that validates the full release pipeline chain with visual and JSON output, auto-detecting source vs consumer repo context.
**Verified:** 2026-04-23T18:15:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1   | `aether integrity` exists as a first-class Cobra command with `--json`, `--channel`, `--source` flags | VERIFIED | `--help` shows all 3 flags; registered via `rootCmd.AddCommand(integrityCmd)` in init() at line 41 |
| 2   | Source repo context runs 5 checks; consumer repo context runs 4 checks | VERIFIED | Source context: Source version, Binary version, Hub version, Hub companion files, Downstream simulation (5). Consumer context from /tmp: Binary version, Hub version, Hub companion files, Downstream simulation (4). Code lines 90-105. |
| 3   | Visual output shows pass/fail per check, summary, and recovery commands | VERIFIED | Banner via renderBanner, checkmark/cross per check, `-- Summary --` with pass count, `Recovery Commands` section. Function `buildIntegrityVisual()` lines 146-183. |
| 4   | JSON output produces structured results with check list, versions, and recovery commands | VERIFIED | Valid JSON with context, channel, checks[] (name/status/message/details/recovery_command), overall, recovery_commands[]. Tested with `--json` flag. |
| 5   | Exit codes: 0 = all pass, 1 = any check fails, 2 = command error | VERIFIED | Exit 0: all checks pass. Exit 1: any check fails (confirmed via actual run). Exit 2: hub not installed (os.Exit(2) at lines 78, 81). |
| 6   | Medic flags incomplete stable/dev publishes and prints exact recovery command (Roadmap SC 2, REL-02) | FAILED | Plan 43-02 was created but never executed. `scanIntegrity()` does not exist in `cmd/medic_scanner.go`. `aether medic --deep` does not include integrity-category findings. |

**Score:** 5/6 truths verified

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
|---|-------------------|--------|----------|
| 1 | `aether` command validates source version, binary version, hub version, and companion surfaces together | VERIFIED | Source context runs all 5 checks covering these surfaces |
| 2 | Medic flags incomplete stable/dev publishes and prints exact recovery command | FAILED | No medic integration; plan 43-02 not executed |
| 3 | Integrity check is runnable both locally (source repo) and downstream (consumer repo) | VERIFIED | Tested from repo root (source) and /tmp (consumer) |
| 4 | Diagnostic output is human-readable and actionable | VERIFIED | Visual output with clear pass/fail markers and recovery commands |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | ----------- | ------ | ------- |
| `cmd/integrity_cmd.go` | Full integrity command implementation | VERIFIED | 352 lines, all check functions, orchestrator, visual/JSON output, context detection. Builds and runs correctly. |
| `cmd/integrity_cmd_test.go` | Unit and E2E tests for integrity command | MISSING | File does not exist. Zero test coverage for the integrity command. |
| `cmd/medic_scanner.go` (modified) | scanIntegrity() wired into deep scan | MISSING | No scanIntegrity() function. Integrity not included in medic --deep output. |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| integrity_cmd.go | rootCmd | rootCmd.AddCommand(integrityCmd) | WIRED | Line 41 in init() |
| runIntegrity | checkSourceVersion | direct call | WIRED | Line 92 |
| runIntegrity | checkBinaryVersion | direct call | WIRED | Lines 93, 100 |
| runIntegrity | checkHubVersion | direct call | WIRED | Lines 94, 101 |
| runIntegrity | checkHubCompanionFiles | direct call | WIRED | Lines 95, 102 |
| runIntegrity | checkDownstreamSimulation | direct call | WIRED | Lines 96, 103 |
| runIntegrity | checkStalePublish (phase 42) | via checkDownstreamSimulation | WIRED | Line 314 calls checkStalePublish |
| runIntegrity | buildIntegrityVisual | direct call | WIRED | Line 136 |
| runIntegrity | json.MarshalIndent | direct call | WIRED | Line 129 |
| medic_scanner.go | scanIntegrity | N/A | NOT_WIRED | Function does not exist; plan 43-02 not executed |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| integrity_cmd.go | binaryVersion | resolveVersion() | FLOWING | Returns "1.0.20" from embedded version |
| integrity_cmd.go | hubVersion | readHubVersionAtPath(hubDir) | FLOWING | Returns actual hub version from ~/.aether/version.json |
| integrity_cmd.go | sourceVersion | resolveSourceVersion() | FLOWING | Returns version from .aether/version.json in repo root |
| integrity_cmd.go | companion file counts | countEntriesInDir() | FLOWING | Counts actual files in hub system directories |
| integrity_cmd.go | downstream result | checkStalePublish() | FLOWING | Calls phase 42 stale-publish detection with real version data |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Command exists and shows help | `aether integrity --help` | Shows Usage with --json, --channel, --source flags | PASS |
| JSON output is valid JSON | `aether integrity --json` | Valid JSON with all expected fields | PASS |
| Source context runs 5 checks | `aether integrity --json` (from repo root) | 5 checks in output | PASS |
| Consumer context runs 4 checks | `aether integrity --json` (from /tmp) | 4 checks, context=consumer | PASS |
| Exit code 1 on failures | `aether integrity; echo $?` | EXIT_CODE=1 | PASS |
| --source flag forces source context | `aether integrity --json --source` (from /tmp) | context=source, 5 checks | PASS |
| --channel dev works | `aether integrity --json --channel dev` | channel=dev | PASS |
| --channel validation | `aether integrity --channel foobarbaz` | Silently defaults to stable (dead code bug) | FAIL (warning) |
| Medic deep includes integrity | `aether medic --deep` | No integrity category in output | FAIL |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| REL-01 (R062) | 43-PLAN | Integrity check validates source, binary, hub, companion files, and downstream result together | SATISFIED | Source context runs all 5 checks; JSON and visual output confirmed |
| REL-02 (R063) | 43-PLAN-02 | Medic/dedicated diagnostics flag incomplete stable and dev publishes with exact recovery commands | BLOCKED | Plan 43-02 created but never executed; no scanIntegrity() in medic_scanner.go |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| cmd/integrity_cmd.go | 48-49 | Dead code: --channel validation unreachable due to normalizeRuntimeChannel default case | Warning | Invalid channel values silently accepted as stable |

### Human Verification Required

No items requiring human verification. All behaviors are programmatically testable.

### Gaps Summary

The core `aether integrity` command is fully functional with all 5 user-specified must-haves met: the Cobra command exists with all flags, context detection works correctly (5 source / 4 consumer checks), visual and JSON output are complete, and exit codes are correct.

However, the ROADMAP defines a broader goal that includes medic integration: "Single integrity check validates the full chain (source to binary to hub to downstream) **and medic flags incomplete publishes with recovery commands**." Plan 43-02 was created to wire `scanIntegrity()` into `aether medic --deep` and write tests, but it was never executed. This means:

1. **REL-02 is blocked** -- medic does not include integrity findings
2. **Roadmap SC 2 is failed** -- "Medic flags incomplete stable/dev publishes and prints exact recovery command"
3. **Zero test coverage** for the integrity command (no integrity_cmd_test.go)

The companion file count check also reports a discrepancy (`skills/codex/ has 0 files (expected 29)`) which appears to be a real environment issue rather than a code bug -- the expected counts may need updating to match the actual installed file layout.

---

_Verified: 2026-04-23T18:15:00Z_
_Verifier: Claude (gsd-verifier)_
