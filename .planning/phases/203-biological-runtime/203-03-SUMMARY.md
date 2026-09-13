---
phase: 203-biological-runtime
plan: "03"
subsystem: recruitment
tags: [recruitment-intent, validation, biological-runtime, durable-recording]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-02's governed recruitment tracer (recruitCmd, recruitmentIntent's minimal wire shape, spawnCanSpawnDecision admission chokepoint) that this plan grows to BIO-01's full field set"
provides:
  - "recruitmentIntent extended to BIO-01's full field set: parent/attempt identity, capability, evidence, resolved permission, urgency, declared scope, cost limits (slots + seconds, never money)"
  - "validateRecruitmentIntent -- a by-name validator returning one of eleven fixed reason classes with a human Detail sentence, fail-closed on every unreadable state"
  - "recordRecruitmentIntent / recordRecruitmentDecision -- durable, idempotent recruitment/intents.json recording BEFORE any admission decision is taken"
  - "aether recruit's full twelve-flag CLI surface (--capability, --attempt, --evidence, --urgency, --declared-path, --cost-slots, --cost-seconds added to the tracer's five)"
affects: [203-04, 203-06, 203-07, 203-08, 203-09]

# Actuals (#2632)
actuals:
  tokens: 14700
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Refuse by name across a fixed, named reason-class vocabulary (mirroring spawnDecisionResult's convention) -- never a bare boolean"
    - "Fail-closed on every unreadable state, including a subtle wrapped-error case: os.IsNotExist does not unwrap fmt.Errorf(\"...: %w\", err) chains, so evidence resolution checks store.FileExists first rather than trusting loadWorkerHandoffRecords' own not-exist branch"
    - "validate -> record (unconditionally, before any decision) -> decide -> attach the decision to the already-durable record -- so a refusal is exactly as recoverable as an admission"
    - "Idempotent create (recordRecruitmentIntent) kept structurally separate from idempotent update (recordRecruitmentDecision), each with its own single-writer, first-write-wins discipline"

key-files:
  created:
    - cmd/recruitment_intent_test.go
  modified:
    - cmd/recruitment_intent.go
    - cmd/recruitment.go

key-decisions:
  - "IntentID is generated as the same value as AttemptID (one timestamp-based ID assigned to both fields), rather than two independently-generated identifiers -- the plan names them as conceptually distinct (parent/attempt identity vs. this intent's own durable identifier) but gives no reason they must differ numerically, and duplicating one ID avoids an unnecessary second generator and a correlation problem between recruitmentResult's existing IntentID/RecruitmentID usage and the new durable record's key."
  - "Capability was made optional rather than required, even though BIO-01 lists it alongside Caste as a paired field -- 203-02's existing TestRecruitmentTracerEndToEnd calls `aether recruit` without any capability-equivalent flag, and requiring it would have broken that already-passing regression test. Capability is still sanitised via colony.SanitizeSignalContent and has its own reason class (`capability`) so unsafe content is refused; an empty value simply means the caste's own definition is specific enough."
  - "Evidence resolution checks store.FileExists(workerHandoffsPath) before calling the existing loadWorkerHandoffRecords -- that function's own `os.IsNotExist(err)` branch never actually fires through store.ReadFile's wrapped errors (fmt.Errorf(\"...: %w\", err) is not unwrapped by the pre-errors.Is os.IsNotExist), so calling it directly against a genuinely absent worker-handoffs.json would fail-closed-refuse every evidence-carrying recruitment on a fresh colony. This is a workaround inside my own validator, not a fix to codex_dispatch_contract.go (out of this plan's file ownership) -- flagged as a Rule 3 blocking-issue fix, not a deviation from the plan's design."
  - "recordRecruitmentIntent and recordRecruitmentDecision are two separate functions rather than one that both creates and updates -- recordRecruitmentIntent is a pure, idempotent create (a second call with the same IntentID, even carrying a different payload, leaves the file byte-identical, per the plan's own acceptance criterion); recordRecruitmentDecision is what actually attaches the decision afterward, and its own first-decision-wins rule keeps that step idempotent too. Neither function alone implements the plan's literal 'validate, record, then decide, then update the record with the decision' sentence -- recruitCmd's own flow does, by calling both in sequence."

requirements-completed: [BIO-01]

coverage:
  - id: D1
    description: "recruitmentIntent carries BIO-01's full field set and validateRecruitmentIntent refuses by name across all eleven fixed reason classes (schema, parent, caste, capability, objective, reason, evidence, permission, urgency, scope, cost), with cost expressed only as helper slots and seconds -- never a money field"
    requirement: "BIO-01"
    verification:
      - kind: unit
        ref: "cmd/recruitment_intent_test.go#TestRecruitmentIntentValidation (17 subtests, one per reason class plus the sanitizer's accept/reject boundary)"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^TestRecruitmentIntentValidation$' -count=1 && ! grep -nEi 'usd|dollar|price|currency' cmd/recruitment_intent.go"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every recruitment intent is durably recorded under recruitment/intents.json, keyed by IntentID, BEFORE any admission decision is taken; a refusal is recorded with its reason exactly as durably as an admission; an unrecordable ask refuses rather than proceeding unrecorded; the ledger is bounded to recruitmentIntentRetention entries"
    requirement: "BIO-01"
    verification:
      - kind: unit
        ref: "cmd/recruitment_intent_test.go#TestRecruitmentIntentRecordCreatesAndIsIdempotent"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_intent_test.go#TestRecruitmentIntentRecordRefusalIsAsDurableAsAnAdmission"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_intent_test.go#TestRecruitmentIntentRecordRetentionPrunesTheOldestEntry"
        status: pass
      - kind: unit
        ref: "cmd/recruitment_intent_test.go#TestRecruitmentIntentRecordUnwritableStoreRefusesTheCommand"
        status: pass
    human_judgment: false
  - id: D3
    description: "aether recruit exposes all twelve BIO-01 fields as CLI flags, its --help carries a plain-English carry-on-alone sentence, a refusal's JSON envelope carries reason/detail matching spawn-can-spawn's vocabulary, and validation refuses (e.g. an unlisted --urgency, --cost-slots 0) BEFORE the admission gate ever runs"
    requirement: "BIO-01"
    verification:
      - kind: unit
        ref: "cmd/recruitment_intent_test.go#TestRecruitCommandFlags (4 subtests)"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^TestRecruitCommandFlags$' -count=1 && go build ./cmd"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 03: Full BIO-01 Recruitment Intent, Validator, and Durable Record Summary

**Grew the recruitment tracer's minimal intent into BIO-01's complete typed wire shape -- an eleven-reason-class validator that refuses by name, worker-authored text sanitised before storage, and a durable `recruitment/intents.json` record written before any admission decision is taken.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-09-13 (immediately following 203-02)
- **Completed:** 2026-09-13T12:47:45+02:00
- **Tasks:** 3 completed
- **Files modified:** 3 (2 modified, 1 created)

## Accomplishments

- Extended `recruitmentIntent` (`cmd/recruitment_intent.go`) with every remaining BIO-01 field -- `IntentID`, `ParentAttemptID`, `ParentRunID`, `Capability`, `Evidence`, `Permission` (a resolved `codex.PermissionProfile`, never caller-trusted), `Urgency`, `DeclaredPaths`, `CostSlots`, `CostSeconds` -- while leaving the tracer's original nine fields byte-for-byte unchanged, confirmed by a source scan finding no `usd`/`dollar`/`price`/`currency` field anywhere on the type.
- Added `validateRecruitmentIntent`, a by-name validator returning one of eleven fixed reason classes (`schema`, `parent`, `caste`, `capability`, `objective`, `reason`, `evidence`, `permission`, `urgency`, `scope`, `cost`) with a human `Detail` sentence naming the offending value -- every branch fails closed on an unreadable state (a store error resolving recorded evidence denies, never silently allows).
- Wired `colony.SanitizeSignalContent` into `Objective`/`Reason`/`Capability` validation and `codex.PermissionProfileForCaste` into permission resolution, so worker-authored text and a caller-declared permission are never trusted raw.
- Added `recordRecruitmentIntent` (idempotent create, keyed on `IntentID`, byte-identical file on replay) and `recordRecruitmentDecision` (attaches the first-and-final decision afterward) writing through `store.UpdateJSONAtomically` to the new `recruitment/intents.json`, pruned to `recruitmentIntentRetention` (200) entries.
- Rewired `recruitCmd` to validate, record the intent unconditionally, THEN decide through the existing `spawnCanSpawnDecision` chokepoint (still the single call site -- no second admission authority), THEN attach the decision to the already-durable record; an unrecordable ask refuses with reason `scope` rather than proceeding unrecorded.
- Exposed all twelve BIO-01 fields on `aether recruit`'s CLI (`--capability`, `--attempt`, `--evidence`, `--urgency`, `--declared-path`, `--cost-slots`, `--cost-seconds` added to the tracer's five), plus a plain-English `Long` help description stating a refusal is not an error.

## Task Commits

Each task was committed atomically:

1. **Task 1: Complete the versioned RecruitmentIntent wire shape** - `7da3430d` (feat)
2. **Task 2: Record every intent durably before any decision is taken** - `be779b40` (feat)
3. **Task 3: Expose every intent field on the public command** - `8321888c` (feat)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `cmd/recruitment_intent.go` - `recruitmentIntent` full field set, `validateRecruitmentIntent`, `sanitizedRecruitmentIntentCopy`, `recordRecruitmentIntent`, `recordRecruitmentDecision`, `trimRecruitmentIntents`, the eleven `recruitmentReason*` constants, `recruitmentUrgencies`, and the four bound constants
- `cmd/recruitment.go` - `recruitCmd` rewired to validate/record/decide/attach-decision, all twelve flags, coordinator-sentinel and current-run resolution for the parent
- `cmd/recruitment_intent_test.go` - `TestRecruitmentIntentValidation` (17 subtests), `TestRecruitmentIntentRecordCreatesAndIsIdempotent`, `TestRecruitmentIntentRecordRefusalIsAsDurableAsAnAdmission`, `TestRecruitmentIntentRecordRetentionPrunesTheOldestEntry`, `TestRecruitmentIntentRecordUnwritableStoreRefusesTheCommand`, `TestRecruitCommandFlags` (4 subtests)

## Decisions Made

- **`IntentID` equals `AttemptID`** (one generated ID assigned to both) rather than two independently-generated identifiers -- see key-decisions above.
- **Capability is optional**, not required, to preserve 203-02's existing regression test that calls `aether recruit` without a capability-equivalent flag. Still sanitised; unsafe content is refused with reason `capability`.
- **Evidence resolution checks `store.FileExists` before calling `loadWorkerHandoffRecords`** -- that existing function's own not-exist branch never fires through `store.ReadFile`'s wrapped errors, so calling it directly against a fresh colony (no handoffs recorded yet) would have fail-closed-refused every evidence-carrying recruitment. See Deviations below.
- **`recordRecruitmentIntent` and `recordRecruitmentDecision` are two separate functions**, not one create-or-update function, so the plan's own "byte-identical on replay" acceptance criterion for `recordRecruitmentIntent` and the "update the record with the decision" requirement can both hold without contradicting each other.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `loadWorkerHandoffRecords`'s not-exist branch never fires through wrapped store errors**
- **Found during:** Task 1 (writing the evidence-resolution subtest for a fresh colony with no handoffs recorded)
- **Issue:** `cmd/codex_dispatch_contract.go`'s `loadWorkerHandoffRecords` treats a missing `handoffs/worker-handoffs.json` as "nothing recorded yet" only when `os.IsNotExist(err)` is true, but `store.ReadFile` always wraps its own error via `fmt.Errorf("...: %w", err)`, and `os.IsNotExist` does not unwrap generic `%w` chains (it only peels `*PathError`/`*LinkError`/`*SyscallError`). Calling `loadWorkerHandoffRecords` directly against a genuinely absent file therefore always returns a non-nil error, which my fail-closed evidence check would have turned into "recorded evidence unreadable" -- refusing every evidence-carrying recruitment on a fresh colony, not just ones with an unresolvable identifier.
- **Fix:** `validateRecruitmentIntent`'s evidence check calls `store.FileExists(workerHandoffsPath)` first. A definite "does not exist" is treated as zero recorded evidence (any claimed identifier then correctly fails to resolve, with reason `evidence` naming it); only a genuine `FileExists`/`loadWorkerHandoffRecords` error still fails closed. `cmd/codex_dispatch_contract.go` itself was not modified -- it is outside this plan's file ownership, and the fix lives entirely inside my own validator.
- **Files modified:** `cmd/recruitment_intent.go` (within this plan's own file list)
- **Verification:** `TestRecruitmentIntentValidation/an_evidence_identifier_that_resolves_to_no_recorded_lifecycle_evidence_is_refused_with_reason_evidence_and_names_it` failed with a generic "recorded evidence unreadable" detail before this fix (naming the wrapped file-not-found error, not the identifier) and passes with it, naming `ghost-evidence-id` specifically.
- **Committed in:** `7da3430d` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking, latent bug in an out-of-scope existing function worked around locally)
**Impact on plan:** Necessary for the evidence check's own correctness on a fresh colony -- no scope creep; the out-of-scope function itself was left untouched and the workaround lives entirely inside this plan's own file.

## Issues Encountered

None beyond the deviation above, found and fixed while writing Task 1's own tests, before Task 2 began.

## Threat Flags

None. This plan strictly adds validation, sanitisation, and durable recording in front of the existing (already-flagged, in 203-02's summary) recruitment dispatch surface -- no new network endpoint, auth path, or schema change at a trust boundary.

## Known Stubs

None -- every field, reason class, and CLI flag this plan's must_haves name is implemented and tested; no placeholder or empty-value stub was introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `cmd/recruitment_intent.go` now carries BIO-01's complete, validated, sanitised, durably-recorded intent shape -- plan 203-06's admission gate (BIO-02's full permission/path/cost/duplicate dimensions layered onto `spawnCanSpawnDecision`) has one complete input type to read and no reason to invent a second, per this plan's own success criteria.
- `recruitmentDecisionResult`'s eleven-reason vocabulary is stable and exported-by-convention (fixed strings, not free text) for 203-06, 203-07, and 203-09 to switch on.
- `recordRecruitmentIntent`/`recordRecruitmentDecision`'s `recruitment/intents.json` ledger is a new, additive data file -- no existing reader is affected, and its shape (`Intent` + optional `Decision` + timestamps) is available for 203-09's inline/end-of-run rendering to consume.
- No blockers for 203-04, 203-06, 203-07, or 203-08: this plan touched only `cmd/recruitment_intent.go`, `cmd/recruitment.go`, and `cmd/recruitment_intent_test.go`, all within its declared file-ownership boundary; no field or function signature that `cmd/recruitment_dispatch.go` or `cmd/recruitment_result.go` (owned by sibling plans) consume was renamed or re-signed -- only additive fields were introduced on `recruitmentIntent`, and both files' existing call sites (`intent.Caste`, `intent.Objective`, `intent.ParentName`, `intent.Workspace`) continue to compile and pass unchanged, confirmed by the full 203-02 regression suite (`TestRecruitmentTracerEndToEnd`, `TestRecruitmentDispatchTerminatesWholeProcessGroup`, `TestRecruitLiveEventsGoThroughTheOneBoundary`, `TestPlainBuildPathDoesNotTouchRecruitment`) passing unmodified.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/recruitment_intent.go`
- FOUND: `cmd/recruitment.go`
- FOUND: `cmd/recruitment_intent_test.go`
- FOUND: commit `7da3430d` (feat: grow recruitment intent to BIO-01's full field set) in `git log --oneline`
- FOUND: commit `be779b40` (feat: record every recruitment intent durably before any decision) in `git log --oneline`
- FOUND: commit `8321888c` (feat: expose every recruitment intent field on aether recruit) in `git log --oneline`
- Re-ran plan-level `<verification>`: `go build ./... && go vet ./cmd && go test ./cmd -run '^(TestRecruitmentIntentValidation|TestRecruitmentIntentRecord|TestRecruitCommandFlags|TestRecruitmentTracerEndToEnd|TestRecruitmentDispatchTerminatesWholeProcessGroup|TestRecruitLiveEventsGoThroughTheOneBoundary|TestPlainBuildPathDoesNotTouchRecruitment|TestPlatformParityGolden)$' -count=1` -- PASS
- Re-ran each task's own literal `<verify>` command exactly as written in 203-03-PLAN.md -- all three PASS
- Confirmed no field name or JSON tag in `cmd/recruitment_intent.go` matches `usd|dollar|price|currency` (grep, case-insensitive)
- Confirmed via `git diff --stat` against the pre-plan base that only the three declared files were touched -- no overlap with 203-04, 203-07, or 203-08's owned files
