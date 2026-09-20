---
phase: 203-biological-runtime
plan: "06"
subsystem: recruitment
tags: [spawn-admission, biological-runtime, security-finding, atomic-write]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-03's full BIO-01 recruitmentIntent wire shape (Permission, Workspace, CostSlots, IntentID, AttemptID) and 203-04's chooseRecruitmentAdapter, both consumed by this plan's extended admission input"
provides:
  - "spawnCanSpawnDecision extended with five new ordered checks (parent, permission, path, cost, duplicate) after the existing depth/budget/ancestor-cycle triad, gated by a declared per-origin applicability table so spawn-log/spawn-can-spawn are never subject to fields they never populate"
  - "amendRecruitmentManifest -- an atomic, upsert-by-IntentID recruitment/manifest.json amendment written before any child process starts, denying launch on a write failure"
  - "recruitmentDepthOverride -- D-11's --max-depth per-run raise, recorded onto the resulting spawn-tree entry"
  - "TestOneAdmissionAuthority / TestEveryChildDispatchPassesAdmission / TestEveryAdmissionReasonIsReachable -- structural guards against a second admission gate ever reappearing"
  - "A documented, unresolved security finding: the pre-existing coordinator-sentinel name match (spawnParentIsRoot) is not corroborated by anything the caller cannot assert, and this plan's own new parent-authority check inherits that exemption rather than closing it"
affects: [203-07, 203-09, 203-14]

# Actuals (#2632)
actuals:
  tokens: 19300
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Declared per-origin applicability table (recruitmentAdmissionChecks) rather than inferring which checks apply from which fields happen to be populated -- a recruit-origin request missing a field an applicable check needs is a denial, never a silent skip"
    - "Root-entrypoint call-graph scan (not per-node) for 'every child dispatch passes admission' -- checking every node would falsely flag dispatchRecruitment, which never calls admission itself by design; its caller (recruitCmd) does"
    - "Live-emitter cross-check (recruitmentAdmissionReasonEmitters) instead of pure AST literal-scanning for reason-string reachability, because several new reasons are emitted via named constants (recruitmentReasonParent, ...), not bare string literals, so an AST literal scan alone would miss them"

key-files:
  created:
    - cmd/recruitment_admission.go
    - cmd/recruitment_admission_test.go
  modified:
    - cmd/spawn.go
    - cmd/recruitment.go

key-decisions:
  - "Permission check denies EVERY recruitment of a repository-read-only caste unconditionally, not only when 'asked to write' in some narrower sense -- dispatchRecruitment has no read-only dispatch mode, so every recruited child gets real write access to its workspace regardless of caste. BIO-02's 'asked to write' distinction collapses to 'always' for this codebase today; documented as an unclassified planner assumption in recruitmentPermissionReason's own doc comment."
  - "Duplicate-intent 'same subtree' is read as the requester's own ancestor lineage (spawnAncestorChain) plus the requester itself, and 'pending' means a recruitment/intents.json entry with no Decision attached yet -- a genuine concurrent request, not one already resolved either way."
  - "D-11's depth-raise record is written onto the SPAWN-TREE entry's Task field as a bracketed suffix ('[depth-raise:max-depth=N]'), not a new struct field -- pkg/agent/spawn_tree.go's SpawnEntry shape is outside this plan's file ownership this wave, and the ledger's fixed 7-field pipe format has no room for a new column without touching that file."
  - "The manifest's 'admitted'->'dispatched' state transition is written immediately BEFORE calling dispatchRecruitment, not strictly after the child process starts as BIO-04's literal wording asks -- cmd/recruitment_dispatch.go's dispatchRecruitment is a synchronous call outside this plan's file ownership and cannot be instrumented mid-execution. Documented as a considered boundary in amendRecruitmentManifest's own doc comment, not a silent reinterpretation."
  - "TestOneAdmissionAuthority uses an explicit, reviewed allowlist of functions permitted to return an admission-shaped result, rather than a fully automatic call-graph inference -- validateRecruitmentIntent and chooseRecruitmentAdapter legitimately share the same result SHAPE for genuinely different concerns (intent validation, adapter choice), and a shape-only AST rule cannot distinguish 'a second admission gate' from 'an unrelated function that happens to reuse a convenient struct'. Adding a name to the allowlist is the explicit, reviewable act this design forces."
  - "DECLINED (see Security Finding below): did not close the coordinator-sentinel (spawnParentIsRoot) name-match bypass a live security review found in cmd/spawn.go/cmd/recruitment.go during this plan's execution. Closing it properly needs a new coordinator-identity corroboration mechanism that does not exist anywhere in this codebase yet, touching pkg/agent/spawn_tree.go and multiple run-entrypoint files outside this plan's declared ownership -- an architectural addition (Rule 4), not an extension of the one chokepoint with a check that calls an existing, already-tested helper (the pattern every one of this plan's five actual dimensions follows). My own new parent-authority check inherits the SAME sentinel exemption every existing caller of spawnParentIsRoot already has (spawnAncestorCycleReason, deriveSpawnDepth, validateRecruitmentIntent) -- it does not add a new hole, and does not make anything worse than it already was since phase 173."

requirements-completed: [BIO-02]

coverage:
  - id: D1
    description: "Five new admission dimensions (parent authority, permission, path containment, cost, duplicate intent) added to spawnCanSpawnDecision in the declared order, each denying by name with a human Detail sentence, each failing closed on unreadable state"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestRecruitmentAdmissionParent* (5 subtests), #TestRecruitmentAdmissionPermission* (3), #TestRecruitmentAdmissionPath* (3), #TestRecruitmentAdmissionCost* (3), #TestRecruitmentAdmissionDuplicate* (4)"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestRecruitmentAdmissionExistingChecksUnchanged"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestRecruitmentAdmission|TestSpawnDecision|TestSpawnAncestor|TestSpawnBudget|TestSpawnFailClosed)' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "A declared applicability table decides which checks apply to which caller (spawn-log/spawn-can-spawn get none of the five; only a real recruitment gets all five) -- a check can never be silently skipped for a caller missing the field it needs"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestRecruitmentAdmissionChecksCoverEveryOrigin, #TestRecruitmentAdmissionChecksOnlyApplyToRecruit"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-11's --max-depth per-run raise honoured on aether recruit; the default cap of two is unchanged; every use is recorded onto the resulting spawn-tree entry so a raise is always visible after the fact"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestRecruitmentAdmissionDepthOverrideDefaultStaysAtTwo, #TestRecruitmentAdmissionDepthOverrideRaisesAboveDefault, #TestRecruitmentAdmissionDepthOverrideRecordsOnTheLedger"
        status: pass
    human_judgment: false
  - id: D4
    description: "An admitted recruitment's manifest amendment is written atomically before any process starts; a failed write denies launch (reason unresolved), naming the write error, with no spawn-tree entry or result created"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestRecruitmentManifestAmendmentDeniesLaunchOnWriteFailure, #TestRecruitmentManifestAmendmentAtomicWithSpawnTreeEntry"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^TestRecruitmentManifestAmendment' -count=1"
        status: pass
    human_judgment: false
  - id: D5
    description: "The manifest amendment's State transitions from admitted to dispatched, readable at both points; an amendment written under an unrecognised schema version is reported as unknown rather than fabricated"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestRecruitmentManifestAmendmentStateTransitionsAdmittedToDispatched, #TestRecruitmentManifestAmendmentUnknownSchemaVersion"
        status: pass
    human_judgment: false
  - id: D6
    description: "A second admission authority, an un-admission-gated child launch, or an unreachable/orphaned admission reason all fail a named structural test -- proven live by mutation-testing each guard (added a second gate, removed the admission call, added an undeclared reason; each guard failed and named the exact offender; all three reverted, working tree clean)"
    requirement: "BIO-02"
    verification:
      - kind: unit
        ref: "cmd/recruitment_admission_test.go#TestOneAdmissionAuthority, #TestEveryChildDispatchPassesAdmission, #TestEveryAdmissionReasonIsReachable"
        status: pass
      - kind: other
        ref: "Live mutation of each guard, confirmed failing by name, then reverted (git diff empty after each revert) -- see Deviations/self-check below"
        status: pass
    human_judgment: false
  - id: D7
    description: "The coordinator-sentinel name-match authentication gap found during this plan's own execution (a caller can claim --parent Queen to bypass the depth cap regardless of its real recorded depth) is documented, scoped, and explicitly left open rather than silently fixed or silently ignored"
    verification: []
    human_judgment: true
    rationale: "This is a genuine security finding, but closing it needs a new coordinator-identity corroboration mechanism that does not exist in this codebase and would touch pkg/agent/spawn_tree.go plus multiple run-entrypoint files outside this plan's declared file ownership -- an architectural decision for the owner/a dedicated follow-up plan, not something a single plan's admission-gate extension should absorb unreviewed. See the Security Finding section below and .planning/WINDOWS.md."

duration: ~90min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 06: BIO-02 Admission Gate Extension Summary

**Five new admission dimensions (parent authority, permission, path containment, cost, duplicate intent) now live inside the one `spawnCanSpawnDecision` chokepoint behind a declared per-origin applicability table, the admitted child's manifest is written atomically before any process starts, D-11's `--max-depth` raise is honoured and permanently visible in the spawn-tree ledger, and three structural guards -- each proven live by mutation -- lock the chokepoint against ever forking a second one.**

## Performance

- **Duration:** ~90 min
- **Started:** 2026-09-13 (immediately following 203-03/203-04 in this wave)
- **Completed:** 2026-09-13
- **Tasks:** 3 completed
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- Extended `spawnDecisionInput` (`cmd/spawn.go`) with `Origin`, `Permission`, `Workspace`, `CostSlots`, `IntentID`, `AttemptID`, and added `spawnDecisionOrigin` (`spawn-log` / `spawn-can-spawn` / `recruit`) with a `spawnDecisionOrigins()` completeness accessor -- `spawnLogCmd` and `spawnCanSpawnCmd` now explicitly declare their own origin too, so the new fields are never populated by accident on a caller they were never meant for.
- Extended `spawnCanSpawnDecision` with five new ordered checks after the existing depth/budget/ancestor-cycle triad: parent authority, permission, path containment, cost, duplicate intent -- each denying by name (`parent`/`permission`/`path`/`cost`/`duplicate`) with a human `Detail` sentence, each failing closed on unreadable state, gated per-origin by a new `recruitmentAdmissionChecks` table so spawn-log/spawn-can-spawn (which never populate the new fields) are subject to none of the five.
- New `cmd/recruitment_admission.go` holds the five check implementations, each calling an existing, already-tested substrate rather than reimplementing it: `codex.PermissionProfileForCaste` (permission), `validateSpendContainedPath` (path, symlink-aware), `spawnTreeBudgetState` (cost, D-12 -- no second counter, no arithmetic on a budget total), and `spawnAncestorChain` (duplicate, reusing the exact chain-walk `spawnAncestorCycleReason` already uses so the two rules cannot drift apart).
- `amendRecruitmentManifest` upserts `recruitment/manifest.json` by `IntentID` through `store.UpdateJSONAtomically`; `recruitCmd` writes the `admitted` record before recording the spawn-tree entry and before any process starts, and a write failure refuses the recruitment (reason `unresolved`, naming the error) with no spawn-tree entry or result ever created. The record transitions to `dispatched` immediately before `dispatchRecruitment` runs.
- `recruitmentDepthOverride` resolves D-11's `--max-depth` per-run raise; the default cap of two is unchanged, and every use is written onto the resulting spawn-tree entry's `Task` field as a `[depth-raise:max-depth=N]` marker, so a raised cap is always visible in the append-only ledger after the fact.
- Three structural guards lock the single admission authority, each proven live by mutation-testing (added, confirmed failing, reverted): `TestOneAdmissionAuthority` (an explicit allowlist of every function permitted to return an admission-shaped result), `TestEveryChildDispatchPassesAdmission` (a root-entrypoint call-graph scan over `recruitment.go`/`recruitment_dispatch.go`), and `TestEveryAdmissionReasonIsReachable` (a live-emitter cross-check of the full declared reason vocabulary, in both directions).

## Task Commits

Each task followed RED (failing test) then GREEN (implementation):

1. **Task 1: Add parent authority, permission, path, cost and duplicate to the one chokepoint**
   - `df1d1e75` (test) -- failing tests for the five new dimensions
   - `f8ce0a9b` (feat) -- `spawnDecisionOrigin`, the five check functions, `recruitmentAdmissionChecks`, `recruitmentDepthOverride`, and the `--max-depth` flag
2. **Task 2: Amend the manifest atomically before dispatch, and deny when that write fails**
   - `ce0b02e2` (test) -- failing tests for the atomic manifest amendment
   - `d70cdee6` (feat) -- `amendRecruitmentManifest`, `recruitmentManifestEntryByIntentID`, wired into `recruitCmd`
3. **Task 3: Lock the single admission authority**
   - `d4e9de18` (test) -- `TestOneAdmissionAuthority`, `TestEveryChildDispatchPassesAdmission`, `TestEveryAdmissionReasonIsReachable` (no separate feat commit -- the deliverable IS the test; production code already satisfied it)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/spawn.go` - `spawnDecisionOrigin`/`spawnOriginSpawnLog`/`spawnOriginSpawnCanSpawn`/`spawnOriginRecruit`/`spawnDecisionOrigins`, extended `spawnDecisionInput` (six new fields), extended `spawnCanSpawnDecision` (five new ordered checks), `spawnLogCmd`/`spawnCanSpawnCmd` now declare their own `Origin`
- `cmd/recruitment_admission.go` (new) - `recruitmentReasonPath`/`recruitmentReasonDuplicate`/`recruitmentReasonUnresolved`, `recruitmentAdmissionReasons`, `recruitmentAdmissionChecks`, `recruitmentCheckApplies`, `recruitmentParentAuthorityReason`, `recruitmentPermissionReason`, `recruitmentPathReason`, `recruitmentCostReason`, `recruitmentDuplicateReason`, `recruitmentDepthOverride`, `recruitmentManifestRecord`/`recruitmentManifestFile`, `amendRecruitmentManifest`, `recruitmentManifestEntryByIntentID`, `errRecruitmentManifestUnknownSchema`
- `cmd/recruitment.go` - `--max-depth` flag, depth-override wiring in `recruitCmd`'s admission decision, the depth-raise ledger marker, the manifest-amendment write-then-deny-on-failure wiring, the `admitted`->`dispatched` transition
- `cmd/recruitment_admission_test.go` (new) - all tests named in Task Commits above, plus `recruitmentAdmissionSeedParent`/`recruitmentAdmissionFillBudget` fixtures and `recruitmentAdmissionReasonEmitters`/`recruitmentAdmissionAuthorityAllowlist`/`recruitmentBuildCallGraph` supporting the three Task 3 guards

## Decisions Made

See `key-decisions` in the frontmatter for the full list. In short: a read-only caste is refused unconditionally (BIO-02's "asked to write" collapses to "always" given this codebase's dispatch mechanism); "same subtree" for duplicates means the requester's own ancestor lineage; the depth-raise record lives in the spawn-tree entry's existing `Task` field (no new column, since that struct is outside this plan's ownership); the manifest's `dispatched` transition is written just before dispatch rather than strictly after process start (a documented, unavoidable boundary given file ownership); and `TestOneAdmissionAuthority` uses a reviewed allowlist rather than pure shape inference, because two existing functions legitimately share the same result shape for unrelated concerns.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test fixture mistook "not a regular file" for "unreadable" when proving fail-closed behaviour**
- **Found during:** Task 1, writing `TestRecruitmentAdmissionDuplicateFailsClosedOnUnreadableIntents`
- **Issue:** `store.FileExists` reports `exists=false, err=nil` for a directory sitting at the target path (it only checks `info.Mode().IsRegular()`) -- a directory placed directly at `recruitment/intents.json` therefore reads as "does not exist" (a routine allow), not a genuine I/O fault, so the fail-closed fixture as first written proved nothing.
- **Fix:** The fixture instead blocks the PARENT `recruitment` directory with a regular file, so `os.Stat(".../recruitment/intents.json")` fails with a real `ENOTDIR` stat error.
- **Files modified:** `cmd/recruitment_admission_test.go` (within this plan's own file list)
- **Verification:** Test passes and names the ledger as unreadable; re-confirmed the original (wrong) fixture silently passed as an "allow", which would have hidden a real fail-open regression.
- **Committed in:** `f8ce0a9b` (Task 1 feat commit, alongside the fixture fix)

**2. [Rule 1 - Bug] Byte-identical budget assertion double-counted an unrelated fixture's entry via the whole-ledger fallback**
- **Found during:** Task 1, writing `TestRecruitmentAdmissionExistingChecksUnchanged`
- **Issue:** Filling exactly `spawnTreeBudgetMax` filler entries in the SAME store already used by the preceding ancestor-cycle subtest (which had seeded one extra entry, `B1`) produced 21 live entries against a cap of 20 with no run ever begun -- `spawnTreeBudgetState`'s Gap-B whole-ledger fallback then reported "21 of 20... counted across the entire ledger because no run is recorded", not the plain "in this run" sentence the assertion expected.
- **Fix:** The budget subtest uses its own isolated fresh store with an explicit `BeginRun` window, so it exercises the ordinary per-run counting path rather than the (also real, separately covered) whole-ledger fallback.
- **Files modified:** `cmd/recruitment_admission_test.go`
- **Verification:** Assertion passes against the exact byte-identical sentence spawn_budget.go already produces.
- **Committed in:** `f8ce0a9b` (Task 1 feat commit)

**3. [Rule 1 - Bug] Manifest-write-failure fixture accidentally blocked the sibling intents-file write too**
- **Found during:** Task 2, writing `TestRecruitmentManifestAmendmentDeniesLaunchOnWriteFailure`
- **Issue:** Blocking the whole `recruitment/` directory with a regular file (this file's own established fault-injection style) also blocks `recruitment/intents.json`'s write, which runs EARLIER in `recruitCmd` than the manifest amendment -- the command failed at `recordRecruitmentIntent` with reason `scope`, never reaching `amendRecruitmentManifest` at all, so the test proved the wrong failure.
- **Fix:** The fixture instead places a directory at `recruitment/manifest.json`'s own leaf path only, leaving `recruitment/` itself a real directory so the sibling `intents.json` write still succeeds.
- **Files modified:** `cmd/recruitment_admission_test.go`
- **Verification:** Test now asserts reason `unresolved` (the manifest-specific refusal) rather than `scope` (the intent-recording refusal); confirmed the corrected fixture reaches `amendRecruitmentManifest` by checking the failing path before the fix.
- **Committed in:** `d70cdee6` (Task 2 feat commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 -- test fixtures that would have proven the wrong thing or nothing at all, found and corrected before any commit landed with a false-positive guard).
**Impact on plan:** All three were necessary for the fail-closed and atomicity guarantees to be GENUINELY proven rather than accidentally vacuous. No production-code scope creep; every fix stayed inside this plan's own test file.

## Security Finding (not fixed -- see Decisions/D7 above)

A live security review during this plan's execution found a fail-open authorization gap in files this plan owns:

- **What:** `spawnParentIsRoot` (`cmd/spawn.go`) matches an unauthenticated, caller-supplied `--parent` name against the fixed sentinel list `{"Queen", "Prime-1", "Swarm"}` with no corroborating evidence. A match grants depth 0 with `DepthIsAuthoritative = true` and skips the spawn-tree lookup entirely. Any caller -- including an already-deeply-recruited helper -- can pass `--parent Queen` to `aether recruit` or `aether spawn-can-spawn --name Queen` and have the depth check evaluate its claimed identity instead of its real recorded one, defeating `spawnMaxDelegationDepth`.
- **What is NOT broken:** every other part of the path is correctly fail-closed. A non-sentinel parent's depth is always derived from its own recorded spawn-tree entry (`latestSpawnEntryByName`), never a flag; `validateRecruitmentIntent` explicitly refuses a non-sentinel parent whose depth is not authoritative. This gap is inherited from phase 173 (`deriveSpawnDepth`/`spawnAncestorChain` share the identical sentinel exemption), not introduced by this plan.
- **Why not fixed here:** closing it properly needs a new coordinator-identity corroboration mechanism -- "something the caller cannot assert" -- and no such mechanism exists anywhere in this codebase today. `agent.SpawnRun` (`pkg/agent/spawn_tree.go`) records only `{ID, Command, StartedAt, EndedAt, Status}`, no coordinator identity field, and that file is outside this plan's declared ownership this wave; the real run-begin call sites (`beginRuntimeSpawnRun`'s callers) live in files also outside this plan's ownership. This is a genuine architectural addition (Rule 4), unlike this plan's five actual dimensions, each of which extends the one chokepoint by calling an EXISTING, already-tested helper.
- **My own new parent-authority check inherits, and does not worsen, this exemption**: `recruitmentParentAuthorityReason` allows the sentinel case unconditionally, identically to every other caller of `spawnParentIsRoot` in this codebase. It closes the gap BIO-02's must_haves actually name (a NAMED, non-sentinel parent resolving to a recorded, non-terminal entry) without silently absorbing an unplanned, cross-cutting architectural change.
- **Recorded in `.planning/WINDOWS.md`** by the reviewer; tracked for a dedicated follow-up plan or owner decision.

## Threat Flags

| Flag | File | Description |
|------|------|--------------|
| threat_flag: fail-open-authorization | cmd/spawn.go, cmd/recruitment.go | The coordinator-sentinel name match (`spawnParentIsRoot`) is not corroborated by anything the caller cannot assert -- see Security Finding above. Pre-existing since phase 173; this plan's new parent-authority check does not close it and does not worsen it. |

## Known Stubs

None. Every dimension this plan's must_haves name is implemented, wired through the one chokepoint, and tested; no placeholder or empty-value stub was introduced. The D-11 depth-raise record living in the spawn-tree entry's existing `Task` field (rather than a new dedicated column) is a deliberate design choice within this plan's file-ownership boundary, not an incomplete stub -- it is fully functional and tested (`TestRecruitmentAdmissionDepthOverrideRecordsOnTheLedger`).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- BIO-02's admission gate is complete: eight ordered checks (three pre-existing plus five new), one declared applicability table, one atomic manifest amendment, D-11's raise honoured and always visible in the ledger, and three structural guards against a second gate -- all live-proven, not merely typed correctly.
- `recruitment/manifest.json` is a new, additive data file with `SchemaVersion`/`State` fields ready for 203-07's `RecruitmentResult` recovery work to read (an `admitted`-state entry with no matching terminal result is exactly the crash signature 203-07's recovery plan is expected to classify).
- No blockers for 203-07, 203-09, or 203-14: this plan touched only its declared files (`cmd/spawn.go`, `cmd/recruitment.go`, `cmd/recruitment_admission.go`, `cmd/recruitment_admission_test.go`); every existing caller of `spawnDecisionInput`/`spawnCanSpawnDecision` that never sets the new `Origin` field continues to get zero new checks, confirmed by the full `TestSpawn*`/`TestRecruitment*` regression suite passing unmodified.
- **Open item carried forward, not a blocker:** the coordinator-sentinel authentication gap (Security Finding above) remains open. It does not block 203-07/203-09/203-14 (none of them depend on sentinel corroboration), but should be triaged by the owner or a dedicated follow-up plan -- it predates this phase and is broader than BIO-02's own scope.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/spawn.go`
- FOUND: `cmd/recruitment_admission.go`
- FOUND: `cmd/recruitment.go`
- FOUND: `cmd/recruitment_admission_test.go`
- FOUND: commit `df1d1e75` (test: five new admission dimensions) in `git log --oneline`
- FOUND: commit `f8ce0a9b` (feat: parent, permission, path, cost, duplicate checks) in `git log --oneline`
- FOUND: commit `ce0b02e2` (test: atomic manifest amendment) in `git log --oneline`
- FOUND: commit `d70cdee6` (feat: amend the manifest atomically before dispatch) in `git log --oneline`
- FOUND: commit `d4e9de18` (test: lock the single admission authority) in `git log --oneline`
- Re-ran plan-level `<verification>`: `go build ./... && go vet ./cmd ./pkg/... && go test ./cmd -run '^(TestRecruitment|TestSpawn|TestOneAdmissionAuthority|TestEveryChildDispatchPassesAdmission|TestEveryAdmissionReasonIsReachable|TestColonyStateWriteAllowlistOnlyShrinks|TestNextActionNeverHardcoded|TestPlatformParityGolden)' -count=1` -- PASS
- Re-ran each task's own literal `<verify>` command exactly as written in 203-06-PLAN.md -- all three PASS
- Mutation-tested all three Task 3 guards live: added a second admission-shaped function (`TestOneAdmissionAuthority` failed, named it), removed the admission call from `recruitCmd` (`TestEveryChildDispatchPassesAdmission` failed, named `recruitCmd`), added an undeclared reason string (`TestEveryAdmissionReasonIsReachable` failed, named it) -- all three reverted, `git status --short` and `git diff --stat` empty after each revert
- Confirmed `git status --short` clean at plan completion (no untracked or uncommitted changes)
