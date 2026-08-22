---
phase: 190-lean-non-duplicated-delivery
plan: 02
subsystem: infra
tags: [typescript, go, worker-dispatch, hive-wisdom, colony-prime, ts-host]

# Dependency graph
requires:
  - phase: 189-complete-worker-contract
    provides: codex.HandoffFieldsSummary, the schema-placement decisions this plan does not undo
provides:
  - Single-channel hive wisdom delivery -- Go's colony-prime context capsule is now the ONLY
    source of "## HIVE WISDOM (Cross-Colony Patterns)" content reaching a worker prompt
  - internalWorkerDispatchRequest with no HiveSection field; joinInternalWorkerSections deleted
  - .aether/ts-host/src/hive-injector.ts and its dedicated test deleted outright
  - Four dispatched-command runners (build, plan, continue, dry-run) that no longer compute or
    attach a TS-side hive_section
  - Regression-lock tests (Go + TS) proving hive_section's absence via output content, not names
affects: [191-dead-wood-sweep]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One canonical helper, all readers converge on it -- resolveCodexWorkerContext() remains
       hive wisdom's sole channel; the TS-side duplicate is removed, not reconciled"
    - "Delete the now-trivial/now-orphaned helper outright -- joinInternalWorkerSections deleted
       once its only remaining argument made it a no-op TrimSpace wrapper"
    - "Clean dist/ rebuild after any TS source deletion -- tsc does not prune orphaned compiled
       output for deleted source files; confirmed empirically, not just asserted"

key-files:
  created: []
  modified:
    - cmd/internal_worker_adapter.go
    - cmd/internal_worker_adapter_test.go
    - .aether/ts-host/src/host.ts
    - .aether/ts-host/src/worker-dispatch.ts
    - .aether/ts-host/src/types.ts
    - .aether/ts-host/test/host-integration.test.ts
    - .aether/ts-host/test/worker-dispatch.test.ts
    - .aether/ts-host/test/milestone-audit.test.ts
    - .aether/ts-host/dist/host.js, host.d.ts, types.d.ts, worker-dispatch.js

key-decisions:
  - "Fixed all four TS call sites (dry-run shared runner, build/plan/continue real dispatch) in
     one sweep, not just the two named in the ROADMAP's parenthetical -- same defect, same fix"
  - "Removed hive-injector.ts and its test outright rather than leaving inert dead code, matching
     189's established delete-the-orphaned-helper precedent"
  - "Regression-lock tests assert output/wire-content absence (no hive_section key, no HIVE WISDOM
     string in stdout), not function-name absence, so a renamed reinjection still trips them"

requirements-completed: []

# Metrics
duration: ~35min
completed: 2026-08-20
---

# Phase 190 Plan 02: Remove TS-Host Hive Double-Injection Summary

**Deleted the TypeScript host's independent hive-wisdom computation (`hive-injector.ts`) and its
four attachment sites, so every worker sees "HIVE WISDOM (Cross-Colony Patterns)" exactly once --
via Go's colony-prime context capsule, not twice via a second TS-computed `hive_section`.**

## Performance

- **Duration:** ~35 min (approximate -- first commit 09:03:56 UTC, last commit 09:20:43 UTC, plus
  setup/proof-capture time before the first commit)
- **Started:** ~2026-08-20T09:00:00Z
- **Completed:** 2026-08-20T09:29:13Z
- **Tasks:** 3/3 completed (Task 1 as RED/GREEN TDD pair)
- **Files modified:** 13 in the plan's own `files_modified` list, plus 2 deviation files
  (`.aether/ts-host/test/milestone-audit.test.ts`, `deferred-items.md`) -- see Deviations below

## Accomplishments

- Removed the `HiveSection` field from the Go wire boundary (`internalWorkerDispatchRequest`) and
  deleted `joinInternalWorkerSections`, the now-single-argument no-op wrapper it left behind
- Deleted `.aether/ts-host/src/hive-injector.ts` (its three exports had no callers outside
  itself and `host.ts`, confirmed by grep before deletion) and its dedicated test file
- Removed `hive_section` computation and attachment from all four dispatched-command runners in
  `host.ts` -- the shared dry-run runner (covers build/plan/continue `--dry-run`),
  `dispatchBuildWave`/`runDispatchedBuildCommand` (incl. its per-iteration re-attach),
  `runDispatchedPlanCommand`, `runDispatchedContinueCommand`
- Removed `hive_section` from `worker-dispatch.ts`'s `dispatchRealWorker` and
  `GoWorkerDispatchRequest`, and from `types.ts`'s `BuildDispatch` interface
- Cleanly rebuilt `.aether/ts-host/dist/`: `hive-injector.js`/`.d.ts` deleted,
  `host.js`/`.d.ts`, `types.d.ts`, `worker-dispatch.js` regenerated -- proven by a non-empty
  `git status --porcelain` diff, not merely asserted
- Rewrote the TS test suite to lock the absence, not just delete the old presence-proving tests --
  regression-lock tests for all four runners plus the Go-side wire boundary

## Task Commits

Each task was committed atomically (Task 1 used RED/GREEN TDD per its `tdd="true"` flag):

1. **Task 1 RED: add failing test for hive_section removal at Go boundary** - `550a21ad` (test)
2. **Task 1 GREEN: remove HiveSection from the Go wire boundary** - `abc84794` (feat)
3. **Task 2: remove TS host's independent hive computation and rebuild dist** - `6ce7c1c9` (feat)
4. **Task 3: lock hive_section's absence across all four dispatch runners** - `012f78c3` (test)

**Plan metadata:** (this commit, following SUMMARY creation)

## Files Created/Modified

- `cmd/internal_worker_adapter.go` - Removed `HiveSection` field and `joinInternalWorkerSections`;
  `skillSection := strings.TrimSpace(request.SkillSection)` directly
- `cmd/internal_worker_adapter_test.go` - Removed the old `hive_section` fixture key; added
  `TestInternalWorkerAdapterIgnoresHiveSectionAndUsesSkillSectionDirectly`
- `.aether/ts-host/src/hive-injector.ts` - Deleted (readHiveWisdom, resolveDomainTags,
  formatHiveWisdomSection -- no remaining callers)
- `.aether/ts-host/src/host.ts` - Deleted `emitHiveSummary`/`prepareHiveSection`; removed the
  hive-injector import; removed hive attachment from all 4 runners; removed `hive_section` from
  `CeremonyDispatchLike`, `HostInjectedDispatchFields`, `toWorkerDispatches`
- `.aether/ts-host/src/worker-dispatch.ts` - Removed the `hive_section` conditional copy in
  `dispatchRealWorker` and the field from `GoWorkerDispatchRequest`
- `.aether/ts-host/src/types.ts` - Removed `hive_section` from `BuildDispatch`
- `.aether/ts-host/dist/` - `hive-injector.js`/`.d.ts` deleted; `host.js`/`.d.ts`, `types.d.ts`,
  `worker-dispatch.js` freshly rebuilt (see Proof of Fix below)
- `.aether/ts-host/test/hive-injector.test.ts` - Deleted entirely (D-12)
- `.aether/ts-host/test/host-integration.test.ts` - Four tests rewritten as absence-proving
  regression locks (build, plan, continue, dry-run); "hive-read failure does not block dispatch"
  and the entire "cross-colony wisdom benefit (HIVE-05)" describe block removed outright (no
  remaining subject); the "keeps the Go-issued manifest immutable" test's hive assertion inverted
  to an absence check; "cross-phase integration (hive + spawn + iteration)" renamed to
  "cross-phase integration (spawn + iteration)" with hive-specific setup/assertions stripped
- `.aether/ts-host/test/worker-dispatch.test.ts` - Rewrote the hive_section presence assertion as
  an absence assertion on the Go-bound request object
- `.aether/ts-host/test/milestone-audit.test.ts` - **Deviation** (Rule 3, see below)
- `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md` - **New** (deviation log)

## Decisions Made

- Fixed all four TS call sites in one sweep (the shared dry-run runner covers three workflows by
  itself; build/plan/continue each have their own real-dispatch runner) rather than only the two
  named in the ROADMAP's parenthetical ("build path and continue dry-run") -- the plan/build
  dry-run and plan's real dispatch runner carry the byte-for-byte identical defect (190-CONTEXT.md
  D-09)
- Deleted `joinInternalWorkerSections` outright rather than leaving it as a harmless-looking
  single-argument wrapper (190-CONTEXT.md D-10, mirrors 189's own precedent)
- Left `.aether/ts-host/src/prompt-assembler.ts`'s unrelated `hiveSection` config field untouched
  -- confirmed zero non-test callers, not part of the double-injection defect, explicitly deferred
  to Phase 191 per 190-CONTEXT.md's own scope boundary
- Regression-lock tests assert on OUTPUT CONTENT (no `hive_section` key in the Go-bound request or
  the printed dry-run JSON; `hive-read` never called) rather than on function names existing or
  not, so a differently-named reinjection mechanism would still trip them

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed `milestone-audit.test.ts`'s v1.21 coverage ledger referencing the deleted `hive-injector.test.ts`**
- **Found during:** Task 3 (running the full `npm test` suite after deleting `hive-injector.test.ts`)
- **Issue:** A historical v1.21-requirements coverage ledger (`REQUIREMENT_COVERAGE_MAP` in
  `milestone-audit.test.ts`) listed `hive-injector.test.ts` as the covering file for requirements
  HIVE-01 through HIVE-07. Deleting that file (explicitly instructed by this plan's D-12) made
  the ledger's own "coverage map test files exist on disk" test fail -- a direct, mechanical
  consequence of the plan's own mandated deletion, not scope creep.
- **Fix:** Updated all seven `HIVE-*` entries to reference `host-integration.test.ts` (the one
  surviving file with any relationship to hive-wisdom testing), with an explanatory comment
  block stating the TS-side mechanism these entries originally named was intentionally retired in
  Phase 190 as a correctness fix, not silently re-pointed as if the original claim still held.
- **Files modified:** `.aether/ts-host/test/milestone-audit.test.ts`
- **Verification:** `npm test --prefix .aether/ts-host` -- the "coverage map test files exist on
  disk" test now passes
- **Committed in:** `012f78c3` (Task 3 commit)

**2. [Rule 1 - same-category defect] Fixed two additional hive-presence assertions the plan's own citation list didn't name**
- **Found during:** Task 3 (a repo-wide grep for `hive` across `host-integration.test.ts`, per
  the plan's own read_first instruction to verify the citation list before trusting it)
- **Issue:** The plan named six `it(...)` blocks to rewrite/remove. A broader grep found two more:
  the "keeps the Go-issued manifest immutable while enriching worker briefs" test (one hive
  assertion mixed into an otherwise-unrelated manifest-immutability test) and the entire
  "cross-colony wisdom benefit (HIVE-05)" describe block (a `describe`, not an `it`, so it fell
  outside a literal `it(...)` grep) -- both asserted TS-computed `hive_section` presence and would
  have failed post-fix if left untouched.
- **Fix:** Inverted the manifest-immutability test's one hive assertion to an absence check,
  keeping its real subject (immutability) intact. Removed the "cross-colony wisdom benefit" describe
  block outright -- its subject (colony-A wisdom text surviving into a TS-computed hive_section)
  no longer exists to test, and forcing a replacement would duplicate the four regression-lock
  tests already added elsewhere in the same file.
- **Files modified:** `.aether/ts-host/test/host-integration.test.ts`
- **Verification:** `npm test --prefix .aether/ts-host` passes; `grep -n "hive_section"` in this
  file now shows only the intentional absence-proving sites
- **Committed in:** `012f78c3` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 same-category defect extension)
**Impact on plan:** Both were direct, mechanical consequences of the plan's own mandated
deletions (D-12's `hive-injector.test.ts` removal), not independent scope creep. No architectural
changes, no new files beyond a deviation log.

## Issues Encountered

- **Transient full-suite hang, not a regression.** `go test ./cmd/... -count=1` hung and timed out
  (600s) when run in the background WHILE `.aether/ts-host`'s `npm ci`/`npm run build`/`npm test`
  were also running concurrently on the same machine. Re-running `go test ./cmd/ -count=1` alone,
  in the foreground, completed cleanly in 387s with no failures -- confirmed a resource-contention
  flake from concurrent heavy subprocess/HTTP activity, not a real defect. No code changes were
  needed; documented here rather than silently ignored.
- **3 pre-existing, unrelated TS test failures.** `ceremony-snapshots.test.ts` has 3 failing
  snapshot comparisons (caste-identity emoji + model-tag rendering, e.g. `🔨🐜 Builder [sonnet]`
  vs. the stale fixture's `🔨 Builder`). Confirmed via `git log` that the Go-side visual source
  (`cmd/codex_visuals.go`, last touched in Phase 187's `a81cf6aa`) postdates the snapshot fixtures
  (last touched in `9f40bf48`), and both predate this plan's base commit (`b60a0075`). Nothing in
  this plan's diff touches ceremony rendering. Logged to
  `.planning/phases/190-lean-non-duplicated-delivery/deferred-items.md`, not fixed (out of scope).

## Proof of Fix (fail-then-pass, per CLAUDE.md's Definition of Done)

### BEFORE: double delivery, demonstrated and quoted

**Go collision point** (`cmd/internal_worker_adapter.go:404`, pre-fix): a throwaway probe
constructed a request with `ContextCapsule` already containing a hive section (simulating Go's own
colony-prime capsule) AND a `HiveSection` (simulating the TS host's independently-computed value),
ran it through `internalWorkerConfig` + `codex.AssembleHostedPrompt`, and counted occurrences of
`"## HIVE WISDOM (Cross-Colony Patterns)"` in the assembled prompt:

```
hive header occurrence count in assembled prompt = 2
## Colony State
some context
## HIVE WISDOM (Cross-Colony Patterns)
(go, 0.90) capsule-channel-marker
### Skill: worker-priming
some skill guidance
## HIVE WISDOM (Cross-Colony Patterns)
(go, 0.90) ts-channel-marker
probe task brief
```

**TS attachment layer** (live, executed, pre-fix `npm test`): the actual dry-run stdout JSON for
`aether host build --dry-run` contained a TS-computed `hive_section` key on every dispatch:

```json
"dispatches": [
  {
    "name": "Builder-01", "caste": "builder", "task": "Build",
    "hive_section": "## HIVE WISDOM (Cross-Colony Patterns)\n\n(go, 0.95) Use table-driven tests"
  },
  ...
]
```

...and all 9 tests in the "hive wisdom injection" + "cross-phase integration" describe blocks
passed while asserting `hive_section` PRESENCE on build, plan, continue, and dry-run alike --
covering all four traced call sites.

### AFTER: exactly one, on all four call sites plus the wire boundary

**Go collision point** (same probe, post-fix, `HiveSection` field removed from the request struct
entirely): occurrence count is 1, sourced solely from `ContextCapsule`:

```
hive header occurrence count in assembled prompt = 1
## Colony State
some context
## HIVE WISDOM (Cross-Colony Patterns)
(go, 0.90) capsule-channel-marker
### Skill: worker-priming
some skill guidance
probe task brief
```

**TS side** (`npm test --prefix .aether/ts-host`, filtered to the 5 new regression-lock tests):

```
✔ build runner does not call hive-read and never attaches hive_section (regression lock for Phase 190, was HIVE-04)
✔ plan runner does not call hive-read and never attaches hive_section (regression lock for Phase 190)
✔ continue runner does not call hive-read and never attaches hive_section (regression lock for Phase 190)
✔ dry-run does not call hive-read and the printed manifest JSON never contains a hive_section key (regression lock for Phase 190)
✔ real dispatch delegates the full worker context to the Go adapter boundary (asserts no hive_section key on the Go-bound request)
ℹ tests 5
ℹ pass 5
ℹ fail 0
```

The dry-run test captures the LITERAL stdout bytes a real `aether host build --dry-run` invocation
prints (not a re-derived approximation) and asserts the substring `"hive_section"` never appears --
an output-content invariant that a renamed reinjection mechanism would still trip.

### Regression lock

`TestInternalWorkerAdapterIgnoresHiveSectionAndUsesSkillSectionDirectly` (Go) and the 5 TS tests
above are permanent, committed regression locks. All assert on WIRE CONTENT (JSON keys present in
a request/response, substrings in printed output) rather than on function or variable names
existing -- a future re-introduction of hive computation under any name, attached to any field,
would still be caught as long as it changes the observable request/output shape.

## Next Phase Readiness

- Criterion 3 of Phase 190's ROADMAP goal is fully satisfied: TS-host hive double-injection is
  removed end-to-end, at all four traced call sites, with a rebuilt `dist/` and regression locks
  on both sides of the Go/TS boundary.
- `.aether/ts-host/src/prompt-assembler.ts`'s unrelated, unused `hiveSection` config field remains
  as a Phase 191 "Dead Wood" candidate (confirmed zero non-test callers; not touched here, per
  190-CONTEXT.md's explicit scope boundary).
- Two out-of-scope, pre-existing issues logged to `deferred-items.md` for future attention: stale
  `ceremony-snapshots.test.ts` fixtures (needs `AETHER_UPDATE_SNAPSHOTS=1`), and several inert
  `hive-read` mock branches left in unrelated test fixtures (cosmetic only).
- No blockers for Plan 190-01 (disjoint files: `cmd/codex_build.go`, `cmd/build_print_brief.go`,
  wrapper docs) or for merging both plans back to the phase branch.

---
*Phase: 190-lean-non-duplicated-delivery*
*Completed: 2026-08-20*

## Self-Check: PASSED

**Files verified present:**
- FOUND: cmd/internal_worker_adapter.go
- FOUND: cmd/internal_worker_adapter_test.go
- FOUND: .aether/ts-host/src/host.ts
- FOUND: .aether/ts-host/src/worker-dispatch.ts
- FOUND: .aether/ts-host/src/types.ts
- FOUND: .aether/ts-host/test/host-integration.test.ts
- FOUND: .aether/ts-host/test/worker-dispatch.test.ts
- FOUND: .aether/ts-host/test/milestone-audit.test.ts
- FOUND: .aether/ts-host/dist/host.js, host.d.ts, types.d.ts, worker-dispatch.js
- FOUND: .planning/phases/190-lean-non-duplicated-delivery/deferred-items.md

**Files verified absent (intentional deletions):**
- CONFIRMED ABSENT: .aether/ts-host/src/hive-injector.ts
- CONFIRMED ABSENT: .aether/ts-host/test/hive-injector.test.ts
- CONFIRMED ABSENT: .aether/ts-host/dist/hive-injector.js
- CONFIRMED ABSENT: .aether/ts-host/dist/hive-injector.d.ts

**Commits verified in git log:**
- FOUND: 550a21ad (test: RED)
- FOUND: abc84794 (feat: GREEN)
- FOUND: 6ce7c1c9 (feat: Task 2)
- FOUND: 012f78c3 (test: Task 3)

No missing items.
