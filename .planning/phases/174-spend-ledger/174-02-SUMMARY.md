---
phase: 174-spend-ledger
plan: 02
subsystem: infra
tags: [go, hooks, jsonschema, claude-code, security-boundary]

# Dependency graph
requires:
  - phase: 173-delegation-guard
    provides: claudeHookInput's hook payload shape and 173-HOOK-FINDINGS.md's empirically confirmed field names, plus the PreToolUse spawn-deny wiring this plan hangs its own fail-soft capture off of
provides:
  - "claudeHookInput.TranscriptPath, decoded from the platform's own PreToolUse payload"
  - "cmd/spend_session_capture.go: recordSpendSessionFromHook (fail-soft write), validateSpendTranscriptPath (path-bounded to $HOME/.claude/projects via filepath.Rel), loadSpendSessionRecord (read side for plan 174-04)"
  - "A locked, named refusal of any wrapper-submitted usage figure on the completion packet, with a proven-failable guard"
affects: [174-04, 174-spend-report, spend-ledger]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fail-soft capture matching captureRawHookPayload's shape: no return value, every error path silent, called unconditionally before any branch that could early-return"
    - "Path containment decided by filepath.Rel with a leading-\"..\" segment check, never a lexical prefix match, mirroring validateAndNormalizeClaimPathToRoot"
    - "A struct that must never carry a field is proven by JSON-bytes fixtures (not struct literals) plus a temporary-add/remove negative check, not by inspection alone"

key-files:
  created:
    - cmd/spend_session_capture.go
    - cmd/spend_session_capture_test.go
    - cmd/spend_packet_guard_test.go
  modified:
    - cmd/hook_cmds.go

key-decisions:
  - "spend/session.json is written by the Go runtime only and deliberately excluded from sanctionedDataWritePrefixes, so a worker's Write tool still cannot aim at the ledger directory"
  - "recordSpendSessionFromHook is called before toolName resolution in hookPreToolUseCmd's RunE, so it runs on every hook fire regardless of which branch the guard later takes and cannot be skipped by an early return"
  - "Idempotency is decided by (TranscriptPath, SessionID) equality against the existing record, not by hashing the whole file, so repeated dispatches in one session leave the file untouched"

patterns-established:
  - "Pattern: a validator that mirrors an existing repo-authoritative path-boundary function (validateAndNormalizeClaimPathToRoot) structurally, so containment logic doesn't get re-invented per subsystem"

requirements-completed: [SPEND-02, SPEND-04]

# Metrics
duration: ~20min
completed: 2026-08-14
---

# Phase 174 Plan 02: Session Transcript Capture and Usage-Assertion Refusal Summary

**The Go runtime now records the Claude Code platform's own transcript-path payload to a path-bounded ledger file on every dispatch, and a completion packet asserting its own token count is rejected by name with a guard proven able to fail.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-14
- **Tasks:** 2
- **Files modified:** 4 (1 modified, 3 created)

## Accomplishments
- `claudeHookInput` now decodes `transcript_path` — the one hook field 173-HOOK-FINDINGS.md confirmed by real capture before this plan needed it, not after
- `recordSpendSessionFromHook` durably records the transcript path, session id, cwd and UTC capture time to `.aether/data/spend/session.json`, refuses anything outside `$HOME/.claude/projects` (null bytes, relative paths, sibling-directory-name collisions all rejected via `filepath.Rel` containment, not string prefixing), and is idempotent across repeated dispatches in the same session
- A capture failure (proved with a read-only store directory, driven through the real `hook-pre-tool-use` CLI surface) leaves the hook's allow/deny answer completely unchanged
- `codexExternalBuildCompletion`'s reflected JSON Schema has no `usage` property anywhere in the document, and a submitted packet carrying one is rejected with a named `additionalProperties` violation — confirmed to be a real guard by temporarily adding a `Usage` field to `codexExternalBuildWorkerResult`, watching both new tests fail, then removing it and watching them pass again

## Task Commits

Each task was committed atomically:

1. **Task 1: Declare and durably capture the platform-delivered transcript path** - `388ae20b` (feat)
2. **Task 2: Lock the refusal of a wrapper-asserted token figure** - `359c33e7` (test)

**Plan metadata:** (this commit, docs)

## Files Created/Modified
- `cmd/hook_cmds.go` - Added `TranscriptPath` to `claudeHookInput`; calls `recordSpendSessionFromHook(input)` unconditionally in `hookPreToolUseCmd`'s RunE, right after `captureRawHookPayload`
- `cmd/spend_session_capture.go` - New: `spendSessionRecord`, `validateSpendTranscriptPath`, `recordSpendSessionFromHook`, `loadSpendSessionRecord`
- `cmd/spend_session_capture_test.go` - New: 4 named tests covering every `<behavior>` bullet (record, idempotency, path refusal including a sibling-directory-collision subtest, and hook-decision isolation under a read-only store)
- `cmd/spend_packet_guard_test.go` - New: `TestCompletionPacketRefusesAssertedWorkerUsage` and `TestCompletionPacketSchemaHasNoUsageProperty`

## Decisions Made
- Mirrored `validateAndNormalizeClaimPathToRoot`'s structural shape for `validateSpendTranscriptPath` (null-byte check, absolute-path requirement, `filepath.Clean`, `EvalSymlinks` on both root and candidate with a fallback to the un-evaluated path, `filepath.Rel` containment) rather than inventing a second path-boundary idiom in the same codebase
- Used raw JSON-bytes fixtures (not Go struct literals) for the packet-guard test, matching `cmd/completion_packet_submitted_bytes_test.go`'s established pattern — a struct literal cannot express a field the struct doesn't declare, which is exactly the case being tested
- Chose a literal `"usage"` byte-search across the marshaled schema document for `TestCompletionPacketSchemaHasNoUsageProperty` rather than a recursive property-key walker — every invopop/jsonschema property name appears as a literal JSON object key, so the simpler check is exactly as strong and far less likely to have its own bug

## Deviations from Plan

None - plan executed exactly as written. The plan's own acceptance criteria required a negative check on the packet guard (temporarily add a `Usage` field, confirm both tests fail, then remove it and confirm they pass) — this was performed manually against a backed-up copy of `cmd/codex_build_finalize.go` and is recorded here as the plan required, not committed to history.

**Negative check performed and recorded:**
1. Added `Usage codex.WorkerUsage \`json:"usage,omitempty"\`` to `codexExternalBuildWorkerResult` in `cmd/codex_build_finalize.go`.
2. Ran `go test ./cmd/... -run 'TestCompletionPacketRefusesAssertedWorkerUsage|TestCompletionPacketSchemaHasNoUsageProperty' -count=1 -v` — both tests failed as expected (`TestCompletionPacketRefusesAssertedWorkerUsage` found no violation because the field is now legitimate; `TestCompletionPacketSchemaHasNoUsageProperty` found the literal `"usage"` property in the reflected schema).
3. Restored `cmd/codex_build_finalize.go` from the pre-edit backup (`diff` confirmed byte-identical to the committed version), rebuilt, and reran the same tests — both passed again.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 174-04's `loadSpendSessionRecord` read-side helper is in place and ready to be consumed by the transcript-parsing plan; the wrapper-path token figure it will produce is tagged non-provider-grade by construction, matching D-06
- `.aether/data/spend/session.json` exists as the durable, path-bounded per-run artifact the honesty argument in D-06 depends on
- No blockers. `go test ./... -count=1`, `go vet ./...`, and `go build ./cmd/aether` all pass across the whole repository, and `.aether/schemas/completion-packet.schema.json` remains byte-identical to its pre-plan state (this plan adds no field to any reflected struct)

---
*Phase: 174-spend-ledger*
*Completed: 2026-08-14*
