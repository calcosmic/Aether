---
phase: 203-biological-runtime
plan: "11"
subsystem: pheromones
tags: [pheromone-influence, action-history, ast-guards, go]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-08's shared tick-to-approve queue (colony.PendingSuggestion, enqueuePendingNote/approvePendingNote/editPendingNote/rejectPendingNote, findPendingNote) and its first Action/ActionAt record; 203-05's one resolver (resolveEffectivePheromones) and provenance/quarantine model"
provides:
  - "appendInfluenceHistory/readInfluenceHistory (cmd/pheromone_influence.go): the one append-only writer of pheromones-history.json, structurally guarded (AST) against rewriting or removing an individual entry"
  - "reinforceNote/deferNote/expireNote/revokeNote/appealNote: the five BIO-08 verbs that did not previously exist, each refusing an unknown note by name and recording a history entry"
  - "pheromoneInfluenceActorAllowed: the one function enforcing that revoke and appeal are owner-only while reinforce/defer/expire may also be performed by the runtime or a learning pass"
  - "colony.PheromoneSignal.DeferredUntil/RevokedAt and the resolver's new \"deferred\"/\"revoked\" exclusion reasons"
  - "pheromoneDisplayCmd's --reinforce/--defer/--expire/--revoke/--appeal/--reason/--actor/--actor-name/--dry-run flags -- the owner-facing surface for the five new actions"
affects: ["203-13 (outcome-weighted strength tuning against this immutable history)", "any future BIO-08 work extending accept/edit/reject into the same history file"]

# Actuals (#2632)
actuals:
  tokens: 16033
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reused the AST-based structural-ratchet pattern 203-05/203-08 established: a narrow, name-scoped detector (assignsIntoInfluenceHistorySliceElement, scoped to a field literally named \"Notes\") rather than a generic double-index-assignment scan, which produced false positives against unrelated pre-existing functions the first time it was written broadly."
    - "Extended the existing pheromoneJSONWriterAllowlist ratchet from this plan's own test file's init() (cmd/pheromone_influence_test.go), rather than editing cmd/pheromone_resolver_test.go which declares it -- the same extension contract 203-08 already established for clearSignalQuarantine."
    - "Centralised the pheromones.json write for all four signal-mutating actions (reinforce/defer/expire/revoke) through one new function, pheromoneInfluenceSaveSignals, so the writer ratchet needed exactly one new named entry instead of four."

key-files:
  created:
    - cmd/pheromone_influence.go
    - cmd/pheromone_influence_test.go
  modified:
    - pkg/colony/pheromones.go
    - cmd/pheromone_resolver.go
    - cmd/pheromone_mgmt.go

key-decisions:
  - "accept/edit/reject (the three actions that landed with 203-08's approval surface) are NOT rewired to write into pheromones-history.json in this plan. cmd/pheromone_approval.go and cmd/suggest_approve.go were explicitly out of scope for this wave (owned by the already-merged 203-08, per this plan's own dispatch constraints) -- 'call into it, do not edit it.' TestEveryDeclaredActionIsReachable proves those three actions are reachable via their existing implementation functions (approvePendingNote/editPendingNote/rejectPendingNote) and existing suggest-approve flags (--approve/--edit/--dismiss), satisfying the must_haves truth that all eight actions are real, reachable, and owner-facing -- but their recorded evidence remains 203-08's own PendingSuggestion.Action/ActionAt scalar, not this plan's new append-only file. A future plan wiring accept/edit/reject into pheromones-history.json would need to touch cmd/pheromone_approval.go, which this plan deliberately did not."
  - "The five new actions operate on two different underlying objects depending on which BIO-08 verb: reinforce/defer/expire/revoke act on a stored colony.PheromoneSignal (identified by ID in pheromones.json), matching their described effect on the resolver and on Strength/ReinforcementCount; appeal acts on a colony.PendingSuggestion (identified by ID in the pending-suggestion queue) whose Action is already \"rejected\", since only a queued item -- never a live signal -- can be in a rejected state. Both identifier spaces use the same sig_<timestamp>_<random> format (both generated via generateSignalID()), so a single note-identifier flag works for all five without the CLI needing to know which kind of object it names."
  - "appealNote reads the queued item via findPendingNote (cmd/pheromone_approval.go, same package) rather than duplicating that lookup -- calling into 203-08's behaviour without editing its file, per this plan's dispatch constraints."
  - "A defer window is enforced purely by the resolver's own time comparison (pheromoneSignalDeferred) against the stored DeferredUntil timestamp -- nothing clears the field once the window passes, mirroring how ExpiresAt-based expiry already works without any command needing to run. A revoked note's RevokedAt is, symmetrically, never cleared by anything in this runtime -- only a fresh, separately recorded owner action could ever change a revoked note going forward, and nothing in this plan builds an \"un-revoke\" path."
  - "An action's reason text that fails colony.SanitizeSignalContent (oversized, XML-tag, or prompt-injection content) is dropped to an empty string rather than blocking the action -- a runtime or learning actor's reason is machine-authored text replayed to the owner, and the fact that the action happened must still be recorded even when its stated reason is unsafe to store verbatim."

requirements-completed: []

coverage:
  - id: D1
    description: "An append-only influence history (pheromones-history.json) records every action taken on a note, with the actor and a before/after summary on every entry; the only exported mutation is an append, verified structurally via an AST guard that fails by name against a deliberately introduced slice-element rewrite; pruning removes only whole extinguished notes' histories past a named retention bound, never a live note's entries."
    requirement: "BIO-08"
    verification:
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestInfluenceHistoryAppendOnly"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestInfluenceHistoryActor"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestInfluenceHistoryRetention"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestInfluenceHistoryAppendOnly|TestInfluenceHistoryActor|TestInfluenceHistoryRetention)$' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "The five previously-missing BIO-08 actions (reinforce/defer/expire/revoke/appeal) are real: each refuses an unknown note by name, reinforce raises strength to the ceiling and increments the reinforcement count, defer/revoke change what the resolver returns (deferred notes return to effect automatically once their window passes; revoked notes stay out permanently), expire sets the expiry to now, appeal records against a rejection without reversing it, actor permissions are enforced by one function (owner-only for revoke/appeal), and --dry-run performs zero writes, proven by a write counter."
    requirement: "BIO-08"
    verification:
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestInfluenceActions"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestDeferredNoteReturns"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestRevokedNoteStaysOut"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestInfluenceActorPermissions"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestInfluenceActions|TestDeferredNoteReturns|TestRevokedNoteStaysOut|TestInfluenceActorPermissions)$' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "The list of declared influence actions and the list of actions the program can actually perform are the same list, proven by three tests each manually verified this session to fail by name against a deliberately introduced violation (a ninth declared action with no implementation; a string literal passed where a declared action constant is required; the owner-only check removed) before being restored to the passing state."
    requirement: "BIO-08"
    verification:
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestEveryDeclaredActionIsReachable"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestEveryRecordedActionIsDeclared"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_influence_test.go#TestInfluenceSurfaceIsOwnerFacing"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestEveryDeclaredActionIsReachable|TestEveryRecordedActionIsDeclared|TestInfluenceSurfaceIsOwnerFacing)$' -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 65min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 11: An Append-Only History and the Five Missing Pheromone Actions Summary

**Eight declared pheromone-note actions are now real and reachable; the five that did not exist (reinforce/defer/expire/revoke/appeal) are built against a new, structurally append-only `pheromones-history.json`, with owner-only enforcement for revoke and appeal in one function.**

## Performance

- **Duration:** 65 min (approximate)
- **Started:** 2026-09-13T12:55:00Z (approx.)
- **Completed:** 2026-09-13T14:00:00Z
- **Tasks:** 3 completed
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `cmd/pheromone_influence.go` (new): `appendInfluenceHistory`/`readInfluenceHistory` are the one exported mutation and its reader for `pheromones-history.json`, written through `store.UpdateJSONAtomically`. The history is structurally append-only (an AST guard fails by name against any slice-element rewrite), records the actor kind/name and a before/after summary on every entry, refuses an entry naming a note that does not exist, and prunes only whole extinguished notes' histories past a named retention bound (200), never a live note's entries however large.
- The three declared actor kinds (owner/runtime/learning) and the closed set of all eight declared influence action names are now data, not prose -- `pheromoneActors()` and `pheromoneInfluenceActionNames()` are the single accessors every guard test derives from.
- `reinforceNote`, `deferNote`, `expireNote`, `revokeNote`, and `appealNote` are real: reinforcing raises strength to the ceiling and increments the reinforcement count; deferring sets a recorded defer-until time the resolver honours automatically; expiring sets the expiry to now; revoking takes a note out of effect permanently with no runtime path back; appealing records against a rejection without reversing it. `pheromoneInfluenceActorAllowed` is the single function enforcing that revoke and appeal are owner-only.
- `colony.PheromoneSignal` gained `DeferredUntil`/`RevokedAt` (pointer-backed, omitempty), and `resolveEffectivePheromones` gained two new exclusion reasons (`deferred`, `revoked`) that behave symmetrically with the existing expiry check: a deferred note returns to effect on its own once its window passes, a revoked note never returns.
- The five new actions are exposed as flags on the existing pheromone management command (`pheromoneDisplayCmd`, aliased `pheromones`) rather than five new commands: `--reinforce`, `--defer` (with `--defer-until`), `--expire`, `--revoke`, `--appeal`, plus `--reason`, `--actor`, `--actor-name`, and `--dry-run`.
- Three Task 3 tests prove the declared/implemented/recorded lists agree, each manually verified this session to fail by name against a deliberately introduced violation, then restored: adding a ninth declared action with no surface entry, passing a raw string literal to `appendInfluenceHistory`, and removing the owner-only check.

## Task Commits

1. **Tasks 1-3 (feat):** `b57565a6` -- `cmd/pheromone_influence.go`, `cmd/pheromone_influence_test.go`, `pkg/colony/pheromones.go`, `cmd/pheromone_resolver.go`, `cmd/pheromone_mgmt.go`.

**Plan metadata:** commit pending (this SUMMARY)

**Note on task/commit mapping:** The two new files' content spans all three tasks together (the append-only spine, the five new actions, and the reachability/declaration proofs) and could not be cleanly split into separate commits by git hunk without fabricating intermediate compilable states -- the same precedent this phase's own 203-05 and 203-08 already recorded for an identical reason.

## Files Created/Modified

- `cmd/pheromone_influence.go` -- `pheromoneInfluenceEntry`, `appendInfluenceHistory`, `readInfluenceHistory`, `pheromoneInfluenceActions`, `pheromoneInfluenceActionNames`, `pheromoneActorOwner`/`pheromoneActorRuntime`/`pheromoneActorLearning`, `pheromoneActors`, `reinforceNote`, `deferNote`, `expireNote`, `revokeNote`, `appealNote`, `pheromoneInfluenceActorAllowed`, `pheromoneInfluenceSaveSignals`, `pheromoneNoteExists`, `pruneExtinguishedInfluenceHistories`
- `cmd/pheromone_influence_test.go` -- 10 test functions (Task 1-3 verify commands) plus AST helpers and seeding utilities
- `pkg/colony/pheromones.go` -- `PheromoneSignal.DeferredUntil`/`RevokedAt`
- `cmd/pheromone_resolver.go` -- `pheromoneExcludedDeferred`/`pheromoneExcludedRevoked`, `pheromoneSignalDeferred`, `pheromoneSignalRevoked`, resolver ordering extended
- `cmd/pheromone_mgmt.go` -- `runPheromoneInfluenceFlags`, `renderPheromoneInfluenceResult`, new flags on `pheromoneDisplayCmd`

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential: accept/edit/reject are proven reachable via their existing (203-08) implementations rather than rewired into this plan's new history file, because `cmd/pheromone_approval.go` was explicitly out of scope for this wave.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Edited two files outside the plan's declared `files_modified`**
- **Found during:** Task 2
- **Issue:** The plan's frontmatter `files_modified` lists only `cmd/pheromone_influence.go`, `cmd/pheromone_write.go`, `cmd/pheromone_mgmt.go`, `cmd/pheromone_influence_test.go` -- but Task 2's own action text explicitly requires adding `DeferredUntil`/`RevokedAt` fields to `colony.PheromoneSignal` (`pkg/colony/pheromones.go`) and extending `resolveEffectivePheromones` (`cmd/pheromone_resolver.go`) to honour them. Neither file is owned by a parallel sibling plan this wave (203-06 owns `cmd/recruitment_admission.go`/`cmd/spawn.go`/`cmd/recruitment.go`; 203-10 owns `cmd/trophallaxis.go`/`cmd/immune.go`/`cmd/codex_dispatch_contract.go`).
- **Fix:** Edited both files as required by the acceptance criteria, following the same precedent 203-05 and 203-08 already recorded in this same phase for an identical mismatch.
- **Files modified:** `pkg/colony/pheromones.go`, `cmd/pheromone_resolver.go`
- **Verification:** `TestDeferredNoteReturns`, `TestRevokedNoteStaysOut` pass; `TestOneEffectivePheromonePredicate`/`TestEveryBriefReaderUsesTheResolver` (pre-existing guards) still pass, confirming no second predicate was introduced.
- **Committed in:** `b57565a6`

**2. [Rule 1 - Bug] The first AST detector for "no slice-element assignment" was too broad and false-positived**
- **Found during:** Task 1 (initial test run)
- **Issue:** A generic double-index-assignment AST scan flagged two unrelated pre-existing functions (`applySpecificationRevisionChange`, `cloneSpecificationBody`) that have nothing to do with the pheromone influence history.
- **Fix:** Narrowed the detector to require the inner index's base be a `SelectorExpr` named exactly `Notes`, the same name-scoped-heuristic discipline `TestNoUngovernedQuarantineClear`'s `looksLikePheromoneSignalReceiver` already established in this codebase for an identical false-positive risk.
- **Files modified:** `cmd/pheromone_influence_test.go`
- **Verification:** Re-ran the full suite named below; no unrelated function is flagged, and the guard still fails by name when a real violation is introduced (verified manually this session, see Self-Check).
- **Committed in:** `b57565a6` (test authored and fixed before the first commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical required by the plan's own acceptance criteria, 1 bug in a test detector caught before commit). **Impact:** No scope creep; both closed genuine gaps between the plan's stated intent and what a naive implementation would have produced. No file owned by a parallel sibling plan (203-06, 203-10) was touched, and neither `cmd/pheromone_approval.go` nor `cmd/suggest_approve.go` (203-08's already-merged files) was edited.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 203-13 can now build outcome-weighted strength tuning against a record that makes it reversible: every action a note has undergone (its own five new verbs) is permanently recorded with an actor, so an owner-set strength change is always distinguishable from a learned one for those five verbs.
- A future plan wanting accept/edit/reject recorded in the SAME `pheromones-history.json` file (rather than 203-08's own `PendingSuggestion.Action`/`ActionAt` scalar) will need to edit `cmd/pheromone_approval.go`, which this plan deliberately left untouched per this wave's file-ownership constraints.
- No blockers for parallel sibling plans 203-06/203-10 in this wave; no file overlap occurred.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/pheromone_influence.go`
- FOUND: `cmd/pheromone_influence_test.go`
- FOUND: commit `b57565a6` in `git log --oneline`
- `go build ./...` -- PASS; `go vet ./...` -- PASS
- Re-ran plan-level task `<verify>` commands: Task 1 `go test ./cmd -run '^(TestInfluenceHistoryAppendOnly|TestInfluenceHistoryActor|TestInfluenceHistoryRetention)$' -count=1` -- PASS; Task 2 `go test ./cmd -run '^(TestInfluenceActions|TestDeferredNoteReturns|TestRevokedNoteStaysOut|TestInfluenceActorPermissions)$' -count=1` -- PASS; Task 3 `go test ./cmd -run '^(TestEveryDeclaredActionIsReachable|TestEveryRecordedActionIsDeclared|TestInfluenceSurfaceIsOwnerFacing)$' -count=1` -- PASS
- All three Task 3 guard tests, plus Task 1's AST slice-element guard, independently verified this session to fail-by-name against a deliberately introduced violation, then restored to the clean/passing state (`git status --short` empty after restoration)
- Regression sweep: `go test ./cmd -run '^(TestColonyStateWriteAllowlistOnlyShrinks|TestOnePheromoneWriterOnly|TestNoUngovernedQuarantineClear|TestOneApprovalSurface|TestEveryProposalEntersTheOneQueue|TestOneEffectivePheromonePredicate|TestEveryBriefReaderUsesTheResolver|TestPlatformParityGolden)$' -count=1` -- PASS; `go test ./cmd -run 'Pheromone|Influence|Approval|Exchange|Resolver|Quarantine|Suggest' -count=1` -- PASS; `go test ./cmd -run 'PhaseEnd|Midden|Hive|Decision|CodexPlan|SignalHousekeeping|Signal' -count=1` -- PASS; `go test ./pkg/colony/... -count=1` -- PASS
