---
phase: 135
phase_name: Doc-CLI Alignment Smoke Test
gathered: "2026-05-15"
status: Ready for planning
---

# Phase 135: Doc-CLI Alignment Smoke Test — Context

## Phase Boundary

Executable YAML smoke test that prevents the docs and CLI from disagreeing. The test reads command definitions from the canonical YAML source files (`.aether/commands/*.yaml`) and verifies each documented flag is accepted by the Go CLI. This is the final v1.20 milestone phase — it closes the "truthfulness at runtime" loop by ensuring what the docs claim matches what the binary does.

## Decisions

### Test Scope: Critical path with auto-discovery
- **Auto-discover** all commands and flags from `.aether/commands/*.yaml` — the canonical source chain
- Validate **every** command has its documented flags (existence check — fast, ~1s)
- Curated **host-critical subset** (plan, build, continue, oracle, watch, swarm, lifecycle) tested with realistic flag combinations — these are user-facing and must be truthful
- Non-host commands get flag existence checks only, not full combination testing
- Rationale: We already have `--help` smoke tests for all commands. This phase adds flag validation without re-testing help text. Host commands are the user contract; everything else is secondary.

### Failure Severity: Tiered
- **Host-critical commands** (the 7 `aether host` subcommands): mismatch = **blocking** — test fails, release is blocked
- **All other commands**: mismatch = **warning** — test logs the mismatch but does not fail the suite
- Rationale: Matches the v1.20 milestone goal of making the host contract honest. Host commands are the user-facing surface; other commands can drift without breaking user trust. Warnings still surface the issue for fixing.

### Where It Runs: Release gate
- Smoke test runs in the **release gate** (`aether publish` or goreleaser pipeline), not on every PR
- Rationale: The test suite is already large (2900+ tests, ~2.5min). Adding YAML parsing + CLI flag validation to every PR would slow daily iteration. Running at release gate catches drift before users see it. Can be promoted to PR-level later if drift becomes frequent.

### Auto-Discovery: Yes, from YAML source
- Test generator reads `.aether/commands/*.yaml` to build the test matrix automatically
- No separate "test YAML" to maintain — the command YAML **is** the test specification
- If a new command is added to `.aether/commands/`, the next test run automatically includes it
- Rationale: The YAML source chain is already canonical. Maintaining a parallel test YAML would guarantee drift between the two. Auto-discovery keeps them in lockstep.

### Output Format
- Results written to `platform-health.json` (existing pattern from `cmd/smoke_test.go`)
- Include: `flag_mismatches[]` with command, flag, expected, actual
- Include: `host_critical_failures[]` for blocking issues
- Dashboard consumer (`computeWarnings`) already reads this file — extend it to show doc-CLI alignment status

## Threat Model

| Threat | Mitigation |
|--------|-----------|
| YAML parse errors block release | Parse errors are logged as warnings, not failures — only flag mismatch failures are blocking |
| New command without YAML source | Commands without YAML are skipped with a warning; they don't break the test |
| Test takes too long at release gate | Existence checks are fast (~1s for 80+ commands); only host-critical subset gets full combination testing |
| YAML auto-discovery misses wrapper-only flags | Wrappers may add flags not in the Go runtime. Log these as non-blocking warnings. |

## Canonical Refs

- `.aether/commands/*.yaml` — canonical command source definitions
- `cmd/smoke_test.go` — existing smoke test pattern
- `.planning/ROADMAP.md` — Phase 135 goal and requirements (DCA-01, DCA-02, DCA-03)
- `.planning/REQUIREMENTS.md` — requirement traceability
- `.aether/docs/publish-update-runbook.md` — release gate integration

## Deferred Ideas

- PR-level fast check: run only host-critical subset on every PR (separate phase, v1.21)
- Interactive flag explorer: generate a web UI showing which flags are tested vs not (separate phase, v1.21)

## Pre-requisites from Prior Phases

- Phase 130 (Host Surface Completeness): ensures all 7 host subcommands exist and execute
- Phase 131 (Test Coverage): ensures host commands have realistic flag combinations to test
- Phase 132 (Wrapper Ownership Decision): clarifies which flags are Go-owned vs wrapper-added
