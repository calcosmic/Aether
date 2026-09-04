---
phase: 199-front-door-and-classic-contract
plan: "16"
subsystem: lifecycle-archive
tags: [go, entomb, sha256, archive-manifest, lifecycle-transaction, replay, wrappers]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Digest-backed multi-root lifecycle transactions, rollback, and exactly-once replay
  - phase: 199-front-door-and-classic-contract
    plan: "15"
    provides: Typed verified or forced-incomplete SealOutcome with retained reviewable state
provides:
  - Deterministic content-addressed archive manifests over every archived, cleared, or retained source
  - Fail-closed cross-reference verification for state, report, XML, closure evidence, forced truth, rollback, and receipts
  - Owner-confirmed stage, verify, publish, then clear transaction with deterministic crash replay
  - Thin optional entomb wrappers with Go runtime ownership across Claude and OpenCode
affects: [plan-199-23, plan-199-27, plan-199-29, plan-199-32, plan-199-33, entomb, seal, lifecycle-recovery]

tech-stack:
  added: []
  patterns: [content-addressed archive manifest, immutable preflight snapshot, publish-before-clear transaction, deterministic replay identity]

key-files:
  created:
    - cmd/entomb_manifest.go
    - cmd/entomb_manifest_199_test.go
    - cmd/entomb_transaction_199_test.go
  modified:
    - cmd/entomb_cmd.go
    - .aether/commands/entomb.yaml
    - .claude/commands/ant-entomb.md
    - .claude/commands/ant/entomb.md
    - .opencode/commands/ant/entomb.md

key-decisions:
  - "Derive one stable entomb transaction and chamber identity from the verified seal outcome so interruption resumes instead of creating duplicate archives."
  - "Treat an archive as valid only when every enumerated source byte, digest, kind, path, and closure cross-reference verifies before any publication or clear."
  - "Keep invocation and owner confirmation separate from seal; platform wrappers call the Go runtime exactly once and never reproduce archive or clearing logic."

patterns-established:
  - "Archive proof: canonical JSON plus SHA-256 byte digests and explicit source/archive path pairs, never directory existence."
  - "Destructive lifecycle boundary: immutable preflight, exact owner confirmation, staged proof, one recoverable transaction, verified replay."

requirements-completed: [CEC-08, LIFE-05]

duration: 71min
completed: 2026-09-04
---

# Phase 199 Plan 16: Verified Entomb Transaction Summary

**Entomb now content-addresses the complete sealed record, verifies every byte and closure reference, publishes one chamber and tombstone, and only then clears active state through a replay-safe transaction.**

## Performance

- **Duration:** 71 minutes
- **Started:** 2026-09-04T13:00:32Z
- **Completed:** 2026-09-04T14:11:17Z
- **Tasks:** 3/3
- **Files changed:** 8 production/test/wrapper files

## Accomplishments

- Added a deterministic `ArchiveManifest` builder and verifier over an explicit source set, with relative-path containment, symlink rejection, required-kind coverage, SHA-256 byte digests, and one canonical manifest digest.
- Bound active state, Crowned report, archive XML, findings, learnings, signals, owner checkpoints, rollback, retained memory, tombstone input, seal outcome, and transaction receipt to one closure identity and exact normal/forced disposition.
- Replaced the sequential entomb flow with a read-only preflight, exact owner confirmation, five visible stages, and one journaled transaction that publishes chamber/tombstone targets before deleting active artifacts or writing reset state.
- Made retry deterministic: a partial commit resumes the same transaction, and a completed invocation returns the same chamber, manifest digest, and receipt without duplicating effects.
- Synchronized canonical, Claude flat/nested, and OpenCode wrappers around the exact description `Archive and clear the sealed colony.` while forbidding wrapper-owned copying, parsing, verification, or state clearing.

## Task Commits

Each task was committed atomically:

1. **Task 1: Build a content-addressed archive manifest**
   - `cd0bb64e` — test(199-16): add failing archive manifest contract (RED)
   - `5e34cc4b` — feat(199-16): add verified archive manifest (GREEN)
2. **Task 2: Stage, verify, publish, then clear transactionally**
   - `3ec4f7c7` — test(199-16): add failing entomb transaction contract (RED)
   - `c8dc92ac` — feat(199-16): make entomb a verified transaction (GREEN)
3. **Task 3: Synchronize the optional entomb wrapper**
   - `f6472f27` — feat(199-16): synchronize entomb wrappers

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/entomb_manifest.go` — canonical archive manifest construction, SHA-256 digesting, path containment, required-kind checks, and closure cross-reference verification.
- `cmd/entomb_manifest_199_test.go` — deterministic-build, tamper, missing-input, unsafe-path, cross-reference, and forced-marker tests.
- `cmd/entomb_cmd.go` — pure preflight, exact confirmation, staging, transaction assembly, clear ordering, replay loading, tombstone/context output, and result rendering.
- `cmd/entomb_transaction_199_test.go` — zero-write refusal, stage ordering, injected faults, replay, forced truth, success, and wrapper parity proof.
- `.aether/commands/entomb.yaml` — canonical optional archive-and-clear runtime contract.
- `.claude/commands/ant-entomb.md` — generated flat Claude wrapper aligned to the runtime contract.
- `.claude/commands/ant/entomb.md` — generated nested Claude wrapper aligned to the runtime contract.
- `.opencode/commands/ant/entomb.md` — generated OpenCode wrapper aligned to the runtime contract.

## Decisions Made

- Archive and transaction IDs use only the validated seal outcome identity, giving retries a stable destination and preventing duplicate chambers.
- Manifest verification compares live source bytes to staged archive bytes and validates semantic closure references; a file existing at the expected path is never proof by itself.
- Chamber files and the tombstone/context are the publish prefix of one lifecycle transaction. Active-file removals and reset `COLONY_STATE.json` follow them in the same durable intent, so a crash can resume or roll back without an unverifiable half-clear.
- A no-flag invocation is a read-only preview. Only `--confirm` crosses the destructive boundary, and seal never invokes entomb automatically.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Bound the manifest receipt to the archive transaction**
- **Found during:** Task 2 (manifest/transaction integration)
- **Issue:** The standalone Task 1 builder initially inherited the upstream seal receipt reference, but the published chamber must prove the receipt for the entomb transaction that actually wrote and cleared its targets.
- **Fix:** The integrated builder records the deterministic entomb receipt ID, and published-result verification requires the manifest transaction and receipt to match the lifecycle journal receipt exactly.
- **Files modified:** `cmd/entomb_manifest.go`, `cmd/entomb_cmd.go`
- **Verification:** `TestEntombTransaction199Success` and `TestEntombTransaction199ReplayExactlyOnce` compare the chamber manifest, durable receipt, digest, and replay result.
- **Committed in:** `c8dc92ac`

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug)
**Impact on plan:** The correction closes the intended receipt cross-reference without changing the public surface or expanding scope.

## Automated Checks

- PASS — exact Task 1 command: `go test ./cmd -run '^TestEntombManifest199(Build|Deterministic|Tamper|CrossReferences|ForcedMarker)$' -count=1`.
- PASS — exact Task 2 command: `go test ./cmd -run '^TestEntombTransaction199(RefusalZeroWrite|StageOrder|FaultRetainsActive|ReplayExactlyOnce|ForcedMarker|Success)$' -count=1`.
- PASS — exact Task 3 commands: `TestEntombWrapperContract199` and `TestCommandSourceHygiene`.
- PASS — combined 13-test Plan 199-16 selection (`10.948s`) and its race-enabled equivalent (`13.399s`).
- PASS — `go vet ./cmd` and `git diff --check`.
- EXPECTED STAGED MIGRATION — `go test ./cmd -run '^TestEntomb' -count=1` retains 10 old unconfirmed/untyped entomb-fixture failures; `go test ./... -count=1` also retains the already-documented catalog, status, Next Up, archived-path, orphan-ratchet, and suffixed-resume migrations. The exact mismatch and handoff are recorded in `deferred-items.md`.

## TDD Gate Compliance

- Task 1 RED `cd0bb64e` failed because the archive manifest builder/verifier did not exist; GREEN `5e34cc4b` made all five exact groups pass.
- Task 2 RED `3ec4f7c7` failed because the verified transaction and confirmation API did not exist; GREEN `c8dc92ac` made all six exact groups pass.
- Task 3's first wrapper-contract run failed against the old description and missing runtime-ownership language before `f6472f27` synchronized all four surfaces; this task was test-first but not declared as a separate TDD commit pair.

## Known Stubs

None. The plan-owned implementation contains no TODO/FIXME marker, placeholder/coming-soon path, or empty mock data feeding the archive result.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: archive-file-boundary | `cmd/entomb_manifest.go` | Entomb reads and compares an explicitly enumerated repository/data source set; canonical-root containment, relative-path validation, duplicate rejection, regular-file requirements, and symlink refusal constrain the boundary. |
| threat_flag: destructive-lifecycle-transaction | `cmd/entomb_cmd.go` | Owner-confirmed entomb publishes and removes files across repository and lifecycle-data roots; immutable intent, pre-image digests, stage receipts, ordered targets, rollback, and replay verification protect the destructive transition. |

## Issues Encountered

- The old entomb command fixtures call `entomb` as an immediate action and build only a Crowned directory/state shape. They do not pass `--confirm` or create the typed `SealOutcome`, digest-linked findings/learnings/checkpoints/rollback, and receipt that Plan 199-16 must reject when absent. Weakening preflight to satisfy them would violate D-17, so their migration is deferred to the executable-corpus/full-gate plans.
- The repository-wide run continued to show the staged failures already listed in `deferred-items.md`; no focused Plan 199-16 normal, race, vet, source-hygiene, or whitespace gate failed.
- The installed progress handler counted 18 of 34 summaries but combines that fraction with zero completed phases and therefore wrote `0%`; the registered `frontmatter.set` handler restored the accurate plan-count progress to `53%`. The requirements handler could not parse this repository's bold-description checkbox format, but `CEC-08` and `LIFE-05` were already checked complete, so `REQUIREMENTS.md` required no edit.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes were left untouched and unstaged throughout execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 199-23 and 199-27 can consume one typed, optional entomb contract for platform parity and executable journey coverage.
- Plan 199-29 can migrate the superseded command fixtures and enforce the final repository-wide normal/race gates without reopening archive semantics.
- Plan 199-32 can route recovery inspection using the durable transaction and retained-state errors already emitted here.
- No Plan 199-16 blocker remains.

## Self-Check: PASSED

- All eight declared implementation/test/wrapper files and this summary exist.
- RED/GREEN commits `cd0bb64e`, `5e34cc4b`, `3ec4f7c7`, and `c8dc92ac`, plus wrapper commit `f6472f27`, resolve in Git in the documented order.
- Exact normal and race-enabled Plan 199-16 tests pass, summary/deferred evidence is whitespace-clean, and protected pre-existing paths remain unstaged.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
