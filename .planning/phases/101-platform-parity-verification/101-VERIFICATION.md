---
phase: 101-platform-parity-verification
verified: 2026-05-07T19:30:00Z
status: gaps_found
score: 7/9 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Parity tests verify all five surfaces agree on command names AND flags (ROADMAP SC #1)"
    status: partial
    reason: "Tests verify command names across all 5 surfaces but do NOT verify flags. D-07 explicitly scoped to names only. ROADMAP SC #1 requires 'command names and flags'."
    artifacts:
      - path: "cmd/parity_test.go"
        issue: "Golden file and all 4 test functions check names only; no flag extraction or comparison logic exists"
      - path: "cmd/testdata/parity_snapshot.json"
        issue: "Contains only name arrays, no flag data"
    missing:
      - "Flag extraction per surface and flag-name comparison across surfaces"
  - truth: "Three known parity gaps are closed (ROADMAP SC #2)"
    status: partial
    reason: "Only 1 of 3 gaps is closed (command-guide alignment verified). Wrapper contract field verification and Codex coverage gaps are documented in KNOWN-GAPS.md but NOT closed. Gaps deferred to Phase 105 per D-06."
    artifacts:
      - path: ".planning/phases/101-platform-parity-verification/KNOWN-GAPS.md"
        issue: "W-01 and I-01 document gaps rather than closing them; ROADMAP says gaps should be 'closed' in this phase"
    missing:
      - "Wrapper contract field verification across all 60 wrappers (e.g., runtime_command, guardrails, output_mode)"
      - "Codex TOML coverage improvement or explicit acceptance decision"
deferred:
  - truth: "Wrapper contract field verification gap"
    addressed_in: "Phase 104 and Phase 105"
    evidence: "Phase 104 SC: 'Structural snapshot tests freeze verified command contracts'; Phase 105 SC: 'All source-check and wrapper-contract checks pass'"
  - truth: "Codex TOML coverage gap (33 commands without agents)"
    addressed_in: "Phase 105"
    evidence: "Phase 105 SC: 'All findings from Phases 100-104 are either resolved or explicitly documented as accepted tech debt'"
  - truth: "Flag parity verification across surfaces"
    addressed_in: "Phase 105"
    evidence: "Phase 105 SC: 'All findings from Phases 100-104 are either resolved or explicitly documented as accepted tech debt'"
---

# Phase 101: Platform Parity Verification -- Verification Report

**Phase Goal:** Go runtime, YAML definitions, Claude wrappers, OpenCode wrappers, and Codex command-guide all agree on command names and what they do
**Verified:** 2026-05-07T19:30:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | All 5 surfaces produce consistent command name lists after alias resolution | VERIFIED | TestPlatformParityGolden passes; golden file shows 60 YAML = 60 Claude = 60 OpenCode = 60 guide; all alias-resolved names present in 377-command runtime catalog |
| 2 | Phantom commands (in wrappers/YAML but not in runtime) are detected and reported | VERIFIED | TestNoPhantomCommands checks YAML, Claude, OpenCode, and guide surfaces against runtime with alias resolution and exclusion sets; passes with 0 phantoms |
| 3 | Prompt-only commands and Cobra builtins are excluded from phantom detection | VERIFIED | promptOnlyCommands (5 entries) and cobraBuiltinCommands (1 entry) skip runtime check; archaeology, chaos, dream, interpret, organize, help correctly excluded |
| 4 | Golden file captures command names from all 5 surfaces for CI drift detection | VERIFIED | parity_snapshot.json contains 6 arrays: yaml_catalog (60), claude_wrapper_catalog (60), opencode_wrapper_catalog (60), command_guide_catalog (60), runtime_catalog_names (377), runtime_resolved_names (12) |
| 5 | Tests pass with current codebase state (known gaps recorded per D-06) | VERIFIED | All 4 parity tests pass; full cmd test suite (77s) passes with 0 failures |
| 6 | Every known parity gap is classified by severity (Critical/Warning/Info) | VERIFIED | KNOWN-GAPS.md has 3 severity sections with 0 Critical, 1 Warning, 1 Info |
| 7 | Gap counts per tier are documented | VERIFIED | KNOWN-GAPS.md summary table: Critical 0, Warning 1, Info 1 |
| 8 | No fix suggestions are included (per D-02) | VERIFIED | grep for fix/suggest/recommend/should in KNOWN-GAPS.md returns no matches |
| 9 | Parity tests verify all five surfaces agree on command names AND flags (ROADMAP SC #1) | FAILED | Tests verify names only; no flag extraction or comparison logic. D-07 scoped to names. ROADMAP SC #1 explicitly requires "command names and flags" |

**Score:** 8/9 truths verified (1 partial failure)

### ROADMAP Success Criteria Assessment

| # | Success Criterion | Status | Evidence |
|---|-------------------|--------|----------|
| SC-1 | Parity tests pass verifying all five surfaces agree on command names and flags | PARTIAL | Names verified across all 5 surfaces; flags NOT verified. D-07 scoped to names only. |
| SC-2 | Three known parity gaps are closed: command-guide alignment, wrapper contract fields, Codex coverage | PARTIAL | 1 of 3 closed (command-guide alignment: 60=60). 2 gaps documented but not closed. Deferred to Phase 105 per D-06. |
| SC-3 | No platform wrapper describes behavior the Go runtime does not support (automated check) | VERIFIED | TestNoPhantomCommands provides automated phantom detection; 0 phantoms found. |

### Deferred Items

Items not yet met but explicitly addressed in later milestone phases.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Wrapper contract field verification | Phase 104, 105 | Phase 104 SC: "Structural snapshot tests freeze verified command contracts"; Phase 105 SC: "All source-check and wrapper-contract checks pass" |
| 2 | Codex TOML coverage gap (33 commands without agents) | Phase 105 | Phase 105 SC: "All findings resolved or documented as accepted tech debt" |
| 3 | Flag parity verification across surfaces | Phase 105 | Phase 105 SC: "All findings resolved or documented as accepted tech debt" |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/parity_test.go` | Combined 5-surface parity test with golden file | VERIFIED | 317 lines, 4 test functions (TestPlatformParityGolden, TestNoPhantomCommands, TestAllYamlHaveWrappersAndGuide, TestAliasResolutionCompleteness), 12-entry alias map, 2 exclusion sets |
| `cmd/testdata/parity_snapshot.json` | Frozen parity snapshot for CI regression | VERIFIED | Valid JSON with 6 arrays covering all 5 surfaces plus resolved names |
| `.planning/phases/101-platform-parity-verification/KNOWN-GAPS.md` | Severity-classified parity gap report | VERIFIED | 53 lines, 3 severity sections, summary table, scope section |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| cmd/parity_test.go | cmd/audit_catalog.go | buildAuditCatalog(rootCmd) | WIRED | Called in extractRuntimeNames() (line 93), TestNoPhantomCommands (line 182), TestAliasResolutionCompleteness (line 301) |
| cmd/parity_test.go | cmd/command_guide.go | commandGuideCatalog() | WIRED | Called in extractCommandGuideNames() (line 82), TestAllYamlHaveWrappersAndGuide (line 252) |
| cmd/parity_test.go | .aether/commands/*.yaml | yamlCommandNamesForGuideTest(t) | WIRED | extractYAMLNames() delegates to yamlCommandNamesForGuideTest from command_guide_test.go (line 58) |
| cmd/parity_test.go | .claude/commands/ant/*.md | extractWrapperNames(t, repoRoot, subdir) | WIRED | ReadDir + .md filter in extractWrapperNames (lines 64-78) |
| cmd/parity_test.go | .opencode/commands/ant/*.md | extractWrapperNames(t, repoRoot, subdir) | WIRED | Same function, different subdir |
| KNOWN-GAPS.md | Phase 105 | "For Phase 105 remediation" | WIRED | Header references Phase 105 as remediation target |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| cmd/parity_test.go | runtimeNames | buildAuditCatalog(rootCmd) | Yes -- 377 Cobra commands | FLOWING |
| cmd/parity_test.go | guideNames | commandGuideCatalog() | Yes -- 60 guide entries | FLOWING |
| cmd/parity_test.go | yamlNames | yamlCommandNamesForGuideTest(t) | Yes -- 60 YAML files read from filesystem | FLOWING |
| cmd/parity_test.go | claudeNames/opencodeNames | extractWrapperNames() | Yes -- 60 .md files read from filesystem | FLOWING |
| parity_snapshot.json | All 5 surfaces | TestPlatformParityGolden | Yes -- 6 arrays populated from live extraction | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 4 parity tests pass | `go test ./cmd/ -run "TestPlatformParity\|TestNoPhantom\|TestAllYaml\|TestAliasResolution" -count=1 -timeout 30s -v` | PASS (0.661s) | PASS |
| Full cmd test suite passes | `go test ./cmd/ -count=1 -timeout 120s` | PASS (77.571s) | PASS |
| Golden file valid JSON with all surfaces | `python3 -c "import json,sys; d=json.load(open('cmd/testdata/parity_snapshot.json')); print(len(d['yaml_catalog']), len(d['claude_wrapper_catalog']))"` | "60 60" | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| PLAT-01 | 101-01, 101-02 | Go runtime, YAML, Claude, OpenCode, Codex agree on command names, flags, and behavior descriptions | PARTIAL | Names verified across all 5 surfaces with 0 gaps. Flags and descriptions not checked (D-07 scope decision). |
| PLAT-02 | 101-01, 101-02 | Existing parity tests extended to close 3 known gaps (command-guide alignment, wrapper contract fields, Codex coverage) | PARTIAL | New combined test extends coverage. Command-guide alignment closed (60=60). Wrapper contract field verification NOT done. Codex coverage gap documented but NOT closed. |
| PLAT-03 | 101-01, 101-02 | No platform wrapper describes behavior the runtime does not support | VERIFIED | TestNoPhantomCommands provides automated phantom detection. 0 phantoms found across all surfaces. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | - | - | - | No anti-patterns detected in parity_test.go or KNOWN-GAPS.md |

### Human Verification Required

No items require human verification. All checks are automated (Go tests and filesystem analysis).

### Gaps Summary

Two gaps were identified where the ROADMAP success criteria promise more than the phase delivered:

1. **ROADMAP SC #1 requires "command names and flags" but tests only verify names.** Decision D-07 (made during planning) explicitly scoped the golden test to command names only, excluding flags and descriptions. This was a deliberate scope reduction. The golden file and all 4 test functions check name parity only. To fully satisfy the ROADMAP contract, flag extraction per surface and cross-surface flag comparison would need to be added.

2. **ROADMAP SC #2 says "three known parity gaps are closed" but only 1 of 3 is closed.** Command-guide alignment with YAML is verified (60=60 perfect match). However, wrapper contract field verification (checking that all 60 wrappers contain the correct runtime_command, guardrails, and output_mode fields) was not implemented. Codex coverage gap (33 commands without TOML agents) was documented in KNOWN-GAPS.md but not closed. Decision D-06 explicitly deferred gap resolution to Phase 105.

Both gaps result from deliberate planning decisions (D-07 and D-06) that narrowed scope. The phase delivers strong value: automated 5-surface name parity verification, phantom detection, golden file CI regression protection, and a classified gap report. The ROADMAP success criteria language is broader than what the phase scope covered.

---

_Verified: 2026-05-07T19:30:00Z_
_Verifier: Claude (gsd-verifier)_
