# Phase 88: Recovery Foundation - Context

**Gathered:** 2026-05-01
**Status:** Ready for planning

## Phase Boundary

Build provenance validation prevents phantom advancement (builds claiming success but producing nothing). Gate failures become structured, recoverable data with a clear unblock path. A privacy gate blocks secrets from entering learning data. This is the trust foundation that all later v1.13 phases build on.

## Implementation Decisions

### Build Provenance
- **D-01:** Metadata-only validation at build-complete — check worker result JSON has `status=success` AND `files_modified > 0`. No filesystem checks (avoids worktree false negatives).
- **D-02:** Continue provenance tracing uses manifest lookup — reads stored worker results from the build manifest to verify claims reference valid runs. Does not re-check filesystem.
- **D-03:** Missing or stale provenance at continue causes rejection and halt — clear error message pointing to `/ant-continue` or `/ant-unblock`. No warn-and-allow.

### Gate Failure UX
- **D-04:** Gate failures use JSON + wrapper rendering — Go runtime outputs structured JSON (extended `gateCheck` struct with `fix_hint` and `recovery_options` fields). Wrapper markdown renders formatted messages on Claude/OpenCode. Codex gets JSON directly.
- **D-05:** Multiple gate failures shown as aggregated summary — all failures in one block with per-gate status, then a single recovery choice. Not sequential per-gate.
- **D-06:** Forbidden strings replaced with structured recovery messages — "CRITICAL: Do NOT proceed" and "The phase will NOT advance" removed. Replaced with: what failed, why, how to fix, two recovery options (manual fix + `/ant-continue`, or `/ant-unblock`).
- **D-07:** Extend existing `gateCheck` struct — add optional `fix_hint string` and `recovery_options []string` fields with `omitempty`. New `gateCheckResult` struct for the richer gate-results.json format. Backward compatible.

### Privacy Gate
- **D-08:** Standard secret patterns — block writes containing API keys (`sk-*`, `key-*`, `token-*`, `bearer`), private keys (`BEGIN RSA/EC PRIVATE KEY`), passwords (`password=`, `passwd=`), and common env file patterns (`.env`, `.env.local`, `credentials.json`). Built on existing `pkg/colony/sanitize.go` patterns.
- **D-09:** Home directory paths are redacted, not blocked — absolute paths starting with `/Users/`, `/home/`, or `~` are scrubbed from content before storage. The write proceeds with paths removed.
- **D-10:** Secrets trigger block + log — when a secret pattern is detected, the entire write is rejected and the matched pattern is logged. No partial writes or redaction of secrets themselves.
- **D-11:** Privacy scanner extends existing security infrastructure — add privacy-scan logic to `cmd/security_cmds.go` and pattern matching alongside existing `pkg/colony/sanitize.go`. No new package.

### Unblock Command
- **D-12:** Phase 88 `/ant-unblock` is info + manual path — reads `gate-results.json`, renders gate failure summary, offers two options: (1) try `/ant-continue` again after manual fixes, or (2) view specific fix hints. No Fixer dispatch (that's Phase 89).
- **D-13:** New `cmd/unblock_cmd.go` — dedicated cobra command file, follows existing command pattern (like `recover_cmds.go`).
- **D-14:** Gate results persist in per-phase `gate-results.json` in `.aether/data/` — contains array of gate results with: gate name, status (`passed`/`failed`/`skipped`/`not-reached`), detail, fix_hint, recovery_options, timestamp, retry_count. Scoped per phase (e.g., `gate-results-88.json`).
- **D-15:** Flags Gate and Watcher Veto always re-run — all other gates check `gate-results.json` and skip if previously passed/skipped. These two check live state so they must always re-evaluate.

### Claude's Discretion
- Exact regex patterns for secret detection (build on existing sanitize.go patterns, extend as needed)
- Gate result JSON schema field naming and nesting depth
- Error message wording for build provenance rejection
- How gate-results.json filenames are scoped (per-phase naming convention)

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — Full v1.13 requirements with traceability (SAFE-01/02/03/04, GATE-01/02/03/04/05, LOOP-01, PRIV-01/02)
- `.planning/ROADMAP.md` § Phase 88 — Success criteria and goal definition

### Research
- `.planning/research/SUMMARY.md` — v1.13 research synthesis with architecture approach and critical pitfalls (pitfall 1: provenance false negatives in worktree mode)

### Existing Code
- `cmd/codex_build_finalize.go` — Build completion flow where SAFE-01/02 validation inserts
- `cmd/codex_continue_finalize.go` — Continue flow where SAFE-03/04 provenance tracing inserts
- `cmd/gate.go` — Existing `gateCheck`/`gateResult` structs and gate-check command (extended for D-04/D-07)
- `cmd/security_cmds.go` — Existing antipattern scanning (extended for D-11 privacy scanner)
- `pkg/colony/sanitize.go` — Existing prompt injection and sanitization patterns (foundation for D-08 secret patterns)

## Existing Code Insights

### Reusable Assets
- `gateCheck` struct in `cmd/gate.go` — extend with `fix_hint` and `recovery_options` fields (D-07)
- `pkg/colony/sanitize.go` — regex pattern matching for injection detection; reuse pattern infrastructure for secret scanning (D-08)
- `cmd/security_cmds.go` `checkAntipattern` function — existing file content scanning; extend for privacy patterns (D-11)
- `cmd/recover_cmds.go` — recovery command pattern to follow for `/ant-unblock` (D-13)
- `codexExternalBuildWorkerResult` struct in `cmd/codex_build_finalize.go` — has `Status`, `FilesModified`, `Name` fields needed for provenance checks (D-01)

### Established Patterns
- OutputWorkflow pattern: Go runtime returns structured JSON, wrapper markdown renders it, Codex gets JSON directly (D-04)
- All new struct fields use `omitempty` for backward compatibility
- Per-phase file naming: `.aether/data/` directory, JSON format
- Cobra command pattern: dedicated `cmd/` files, `outputOK()`/`outputError()` for responses

### Integration Points
- Build finalize: between worker result aggregation and manifest write — insert provenance validation (SAFE-01/02)
- Continue finalize: before gate run — insert provenance tracing against stored manifest (SAFE-03/04)
- Gate check: extend `gateCheck` return to include recovery info (GATE-01/02)
- Gate retry: check `gate-results.json` before running gates, skip passed/skipped except Flags + Watcher Veto (GATE-04/05)
- Circuit breaker: gate retry counts feed into existing v1.12 circuit breaker (LOOP-01)
- Privacy scan: intercept learning writes before storage (PRIV-01/02) — hooks into future Phase 90 learning pipeline

## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 88-Recovery Foundation*
*Context gathered: 2026-05-01*
