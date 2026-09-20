---
phase: 197-one-answer-to-what-next
plan: 07
subsystem: cli
tags: [go, next-action, ci-wiring, ratchet, ast, go-ast, guard-file-inventory]

requires:
  - phase: 197-one-answer-to-what-next
    provides: "resolveNextAction and the availability gate (197-01); renderNextActionCard and applyNextActionToResult (197-02); the four wiring helpers and the seven-surface consistency test (197-04); pause as the canonical Cobra command name (197-05); the six remaining migrated surfaces (197-06) -- all eleven of criterion 2's named commands"
provides:
  - "TestEveryLifecycleCommandEndsWithNextAction -- criterion 2's own named test, driving all eleven lifecycle commands criterion 2 lists, comparing the screen's recommended command against the machine-readable envelope's from real command executions, checking resolution against the live command tree, and checking platform-form-on-screen / runtime-form-in-envelope"
  - "TestNextActionNeverHardcoded -- criterion 3's own named ratchet, an AST scanner over the cmd package finding every string literal naming an aether/ant command used as next-step advice (funnel-function argument, result-key assignment, or return from a *NextStep/*NextAction-suffixed function), frozen at a measured baseline of 116 unique sites"
  - "Both guards registered in wiringGateGuardFiles and the named CI wiring step's -run filter, alongside a pre-existing, previously-unregistered third ratchet (colony_state_atomicity_ratchet_test.go) TestEveryGuardFileIsInTheWiringInventory's derived shape found and this plan registered"
  - "TestEveryGuardFileIsInTheWiringInventory -- derives the expected guard-file inventory from a stated three-part shape (a reviewed -update- regeneration flag, a testdata/ reference, the word \"baseline\" in the file's own source) rather than a hand-kept list, closing the one remaining hole: a guard file never added to the inventory at all"
affects: []

actuals:
  tokens: 28137
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Coverage-by-agreement: for each of the eleven named commands, drive the real command once per output mode and compare the screen's shown command against the envelope's next_command field directly, rather than independently re-resolving and trusting the two happen to agree"
    - "Scope rule as data, not prose: the hardcode ratchet's three detection rules (funnel-call argument, result-key assignment, suffix-named-function return) are held in package-level maps/slices the scanner reads, matching the plan's own instruction"
    - "Derive the guard-file inventory from a stated file shape (regeneration flag + testdata/ reference + \"baseline\" mention) rather than trusting a human to remember to register a new ratchet"

key-files:
  created:
    - cmd/lifecycle_next_action_coverage_test.go
    - cmd/next_action_hardcode_ratchet_test.go
    - cmd/testdata/next_action_hardcode.json
    - cmd/testdata/next_action_hardcode_baseline.json
  modified:
    - cmd/lifecycle_card_endgame_test.go
    - cmd/lifecycle_card_session_test.go
    - cmd/ci_wiring_gate_test.go
    - cmd/colony_state_atomicity_ratchet_test.go
    - cmd/subcommand_reachability_ratchet_test.go
    - .github/workflows/ci.yml

key-decisions:
  - "Screen-vs-envelope agreement is checked by extracting the command shown on screen (commandInClosing) and comparing it directly against the envelope's next_command string, never by independently re-resolving an answer and hoping the two happen to match -- this is what 'the machine-readable version carries the identical information' means per the plan's own interfaces note."
  - "For the three commands whose own outcome is an override a bare re-resolve cannot reproduce (update's repair report, recover's own scan findings, status's in-flight-workers/guided-action facts), the coverage test drives the real command twice -- once per output mode, over the identical saved fixture -- rather than reconstructing a nextAction struct from JSON-decoded output, which would silently fail the type assertion nextActionFromResult relies on."
  - "The hardcode ratchet's scope rule is three data-declared checks (funnel function calls, result-key assignments, suffix-named-function returns), calibrated by running the scanner against the real tree BEFORE writing either data file and reading its output: 116 unique sites (133 raw), spot-checked file by file, well inside the ~200 pre-phase ceiling and none a false positive."
  - "TestEveryGuardFileIsInTheWiringInventory's derived shape (regeneration flag + testdata/ reference + the word 'baseline') was chosen specifically because it excludes cmd/audit_catalog_test.go's ordinary -update-golden mechanism (no baseline pairing, no 'baseline' mention) while including the genuine shrink-only-ratchet class -- verified empirically before writing the real test, not assumed."
  - "The derivation's first run named cmd/colony_state_atomicity_ratchet_test.go as a real, unregistered match. Registered it (adding the file, its 3 tests, and a third exempted regeneration-flag identifier) rather than recording a deliberate omission, per the plan's own instruction not to narrow the shape rule to dodge an inconvenient file."

patterns-established:
  - "A guard file's own regeneration flag is exempted from the runtime-escape-hatch scan by a STATED LIST of reviewed identifiers (exemptedRegenerationFlagIdentifiers), not a single hardcoded string -- widening the list is itself a reviewed action, and the exemptedFlagLineCount assertion tracks the list's own length so a fourth flag can never hide behind it."

requirements-completed: [NEXT-02, NEXT-03]

coverage:
  - id: D1
    description: "One named test (TestEveryLifecycleCommandEndsWithNextAction) drives all eleven lifecycle commands criterion 2 names, with a sub-test per command, and has been demonstrated failing both when the card is removed from one command (names it) and when the driven set shrinks below eleven (anti-vacuity floor)."
    requirement: NEXT-02
    verification:
      - kind: unit
        ref: "cmd/lifecycle_next_action_coverage_test.go#TestEveryLifecycleCommandEndsWithNextAction"
        status: pass
      - kind: manual_procedural
        ref: "card-removal demonstration: renderPauseVisual's renderLifecycleClosing call temporarily replaced with a hand-written renderNextUp call, test FAILED naming \"pausing\", reverted; anti-vacuity demonstration: lifecycleCoverageCases temporarily shrunk to 10 entries, test FAILED naming the shortfall, reverted"
        status: pass
    human_judgment: false
  - id: D2
    description: "The count of places still hand-typing a command name is a measured, frozen number (116 unique sites, 133 raw) that can only fall, demonstrated failing three ways: a real planted violation in a clean file, a synthetic-directory planted violation, and a one-for-one swap."
    requirement: NEXT-03
    verification:
      - kind: unit
        ref: "cmd/next_action_hardcode_ratchet_test.go#TestNextActionNeverHardcoded"
        status: pass
      - kind: unit
        ref: "cmd/next_action_hardcode_ratchet_test.go#TestNextActionHardcodeDetectsAPlantedViolation"
        status: pass
      - kind: unit
        ref: "cmd/next_action_hardcode_ratchet_test.go#TestNextActionHardcodeSwapFails"
        status: pass
      - kind: manual_procedural
        ref: "real-tree demonstration: a throwaway cmd/zz_demo_planted_violation.go file with a hand-typed renderNextUp(...) literal added, TestNextActionNeverHardcoded FAILED naming the file and the exact literal, file removed"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both guards run in CI, registered in the same inventory the repository's other guards are held to account by -- the named CI step's -run filter names every top-level test in both files, in both directions, and the escape-hatch scan covers both and finds no skip path."
    requirement: NEXT-03
    verification:
      - kind: integration
        ref: "cmd/ci_wiring_gate_test.go#TestWiringGateStepRunsEveryWiringTest"
        status: pass
      - kind: integration
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestWiringGuardsHaveNoRuntimeEscapeHatch"
        status: pass
    human_judgment: false
  - id: D4
    description: "A guard file left out of the inventory is now caught by a test rather than by whoever reviews the change: TestEveryGuardFileIsInTheWiringInventory derives the expected inventory from a stated file shape, fails by name on a missing file (demonstrated), fails on a vacuous derivation (demonstrated), and its first real run against the pre-197-07 tree found and this plan closed a genuine, pre-existing gap (colony_state_atomicity_ratchet_test.go)."
    requirement: NEXT-03
    verification:
      - kind: unit
        ref: "cmd/ci_wiring_gate_test.go#TestEveryGuardFileIsInTheWiringInventory"
        status: pass
      - kind: manual_procedural
        ref: "missing-file demonstration: next_action_hardcode_ratchet_test.go temporarily removed from wiringGateGuardFiles, test FAILED naming it, reverted; anti-vacuity demonstration: wiringGuardFileShape temporarily forced to return false, test FAILED naming the broken derivation, reverted"
        status: pass
    human_judgment: false

duration: 145 min
completed: 2026-08-29
status: complete
---

# Phase 197 Plan 07: Criterion 2's Coverage Test and Criterion 3's Hardcode Ratchet Summary

**One named test drives every one of the eleven lifecycle commands criterion 2 lists and compares screen against envelope directly; an AST scanner freezes today's real count of hand-typed command advice at 116 sites and can only shrink; both are registered in CI's guard-file inventory, which a new derived-shape test now protects from ever having an unregistered guard again -- and its first run caught a genuine, three-plan-old gap.**

## Performance

- **Duration:** 145 min
- **Tasks:** 3
- **Files created:** 4
- **Files modified:** 6

## Accomplishments

- **`TestEveryLifecycleCommandEndsWithNextAction`** is the single place criterion 2 is now checked. It drives all eleven named commands -- reusing the exact fixtures the four per-wave test files (197-04/197-05/197-06) already proved drive the real thing, never a second hand-typed copy -- and for each: confirms the screen ends with the shared card, compares the screen's recommended command directly against the machine-readable envelope's `next_command` field, checks that command resolves against the live command tree, and checks platform-correct rendering on Claude/OpenCode/Codex while the envelope stays in runtime form.
- **`TestNextActionNeverHardcoded`** is criterion 3's ratchet: an AST scanner over the cmd package (excluding the resolver and test files) that finds every string literal naming an `aether <verb>` or `/ant-<verb>` command used as next-step advice -- passed to the shared `renderNextUp` funnel, assigned to a result key that carries the next step, or returned from a function whose name says it decides the next step. Run against the real tree before either data file was written, its output (116 unique sites, 133 raw) was read and spot-checked file by file before freezing -- comfortably inside the "roughly two hundred" pre-phase ceiling the plan named as a sanity bound, and every sampled entry a genuine hand-typed advice site, not a false positive.
- **Both guards registered where the repository holds its other guards to account.** `wiringGateGuardFiles` and the named CI step's `-run` filter both name every top-level test in both new files; the escape-hatch scan's single hardcoded exemption became a stated list of reviewed identifiers so a second (and, as it turned out, third) regeneration flag could be added without weakening the check.
- **`TestEveryGuardFileIsInTheWiringInventory`** closes the last hole the plan's `<interfaces>` section named: a guard file that is never added to the inventory at all is invisible to every other check. It derives the expected inventory from a stated three-part shape (a reviewed `-update-...` regeneration flag, a `testdata/` reference, and the word "baseline" somewhere in the file's own source) -- verified empirically to include the genuine shrink-only-ratchet class while excluding an ordinary golden-file test (`cmd/audit_catalog_test.go`'s `-update-golden`, which has no baseline pairing and never mentions "baseline").
- **The derivation's first real run found a genuine, three-phase-old gap.** `cmd/colony_state_atomicity_ratchet_test.go` (Phase 188's COLONY_STATE.json atomicity ratchet) matched the shape and was never in `wiringGateGuardFiles`. Registered rather than recorded as a deliberate omission, per the plan's own instruction not to narrow the shape rule to dodge an inconvenient file.

## Task Commits

1. **Task 1: criterion 2's named coverage test** -- `ec5c8dc0` (test)
2. **Task 2: criterion 3's hardcode ratchet, measured at 116 sites** -- `7fec3144` (test)
3. **Task 3: register both guards in CI, close the inventory hole** -- `97ce3abf` (feat)

### Recorded demonstration evidence

**Task 1 -- card removed from pausing:**
```
--- FAIL: TestEveryLifecycleCommandEndsWithNextAction/pausing
    pausing does not end with the shared card at all
```
**Task 1 -- driven set shrunk to 10:**
```
--- FAIL: TestEveryLifecycleCommandEndsWithNextAction
    this coverage test drives 10 lifecycle commands; criterion 2 names 11:
    [starting discussing planning building continuing pausing resuming
     sealing/finishing updating recovering checking status]
```

**Task 2 -- real planted violation** (a throwaway `cmd/zz_demo_planted_violation.go` calling `renderNextUp("Run `aether ratchet-selftest-demo` now.")`):
```
--- FAIL: TestNextActionNeverHardcoded
    1 new hand-typed command-advice site(s) found that are not in
    testdata/next_action_hardcode_baseline.json:
      cmd/zz_demo_planted_violation.go:renderDemoPlantedViolation
      "Run `aether ratchet-selftest-demo` now."
```

**Task 3 -- guard file removed from the inventory:**
```
--- FAIL: TestEveryGuardFileIsInTheWiringInventory
    1 file(s) match the shrink-only baseline ratchet shape ... but are not
    in wiringGateGuardFiles:
      next_action_hardcode_ratchet_test.go
```
**Task 3 -- derivation forced vacuous:**
```
--- FAIL: TestEveryGuardFileIsInTheWiringInventory
    wiringGuardFileShape matched zero files under cmd/ -- the shape rule is
    broken (or has been narrowed into uselessness), not that every
    shrink-only baseline ratchet vanished from the repository
```
All four demonstrations were reverted immediately after capture; `git status --short cmd/` was clean before each task's real commit.

## Files Created/Modified

- `cmd/lifecycle_next_action_coverage_test.go` -- `TestEveryLifecycleCommandEndsWithNextAction`, the 11-case table, and the three custom drivers (update/recover/status) needed because their own outcome is an override a bare re-resolve cannot reproduce
- `cmd/next_action_hardcode_ratchet_test.go` -- the AST scanner (`scanNextActionHardcodeSource`), the three data-declared scope rules, `TestNextActionNeverHardcoded`, `TestNextActionHardcodeBaselineMatchesLive`, and the two "prove it fires" self-tests
- `cmd/testdata/next_action_hardcode.json`, `cmd/testdata/next_action_hardcode_baseline.json` -- the measured, frozen, byte-identical pair
- `cmd/ci_wiring_gate_test.go` -- both new files added to `wiringGateGuardFiles`, the anti-vacuity floor re-measured to 50 (from a stale, never-updated "20" pin), `TestEveryGuardFileIsInTheWiringInventory` and its shape rule
- `.github/workflows/ci.yml` -- the named wiring step's `-run` filter extended with all 9 new/newly-registered test names (5 from this plan's own two files, 3 from colony_state_atomicity_ratchet_test.go, 1 for `TestEveryGuardFileIsInTheWiringInventory` itself)
- `cmd/subcommand_reachability_ratchet_test.go` -- the escape-hatch exemption widened to `exemptedRegenerationFlagIdentifiers` (a stated list, now three entries), the `< 11` inventory floor raised to `< 14`
- `cmd/colony_state_atomicity_ratchet_test.go` -- registered in the inventory (deviation, see below); its "cmd/\*.go" phrasing rewritten to "the cmd package's .go files" (deviation, see below)
- `cmd/lifecycle_card_endgame_test.go`, `cmd/lifecycle_card_session_test.go` -- four single-assertion, now-subsumed per-wave tests deleted (see Deviations)

## Decisions Made

See `key-decisions` in the frontmatter. In short: agreement is checked by comparing what's on screen against what's in the envelope directly (never two independent resolves trusted to match); the three commands with real, run-specific overrides are driven twice rather than reconstructed from JSON; the ratchet's scope rule was calibrated against the real tree before freezing anything; and the inventory-derivation shape rule was chosen and verified to separate genuine shrink-only ratchets from an ordinary golden-file test before being trusted.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 -- Blocking] Registering the two new guard files required also registering a genuine, pre-existing unregistered one**

- **Found during:** Task 3, running `TestEveryGuardFileIsInTheWiringInventory` for the first time against the pre-change tree, exactly as the plan's `<action>` instructed.
- **Issue:** the derived shape (a reviewed `-update-...` regeneration flag, a `testdata/` reference, the word "baseline" in the file's own source) matched `cmd/colony_state_atomicity_ratchet_test.go` -- Phase 188's COLONY_STATE.json atomicity ratchet -- which had never been added to `wiringGateGuardFiles`. The plan's own `<action>` names this exact scenario and its two honest resolutions.
- **Fix:** registered it (file, its 3 top-level tests, a third exempted regeneration-flag identifier `updateColonyStateWriteAllowlist`), rather than recording a deliberate omission, per the plan's explicit instruction not to narrow the shape rule to dodge an inconvenient file. The CI filter absorbed its 3 tests without difficulty.
- **Files modified:** `cmd/ci_wiring_gate_test.go`, `cmd/subcommand_reachability_ratchet_test.go`, `.github/workflows/ci.yml`.
- **Verification:** `TestWiringGateStepRunsEveryWiringTest`, `TestWiringGuardsHaveNoRuntimeEscapeHatch`, `TestColonyStateWriteAllowlistOnlyShrinks` and its two siblings all pass.
- **Commit:** `97ce3abf`

**2. [Rule 1 -- Bug] `stripGoComments`'s naive block-comment stripper silently swallowed almost an entire guard file**

- **Found during:** Task 2 (own file), then again in Task 3 when registering `colony_state_atomicity_ratchet_test.go`.
- **Issue:** `stripGoComments` (`cmd/subcommand_reachability_ratchet_test.go`) treats any literal `/*` two-character sequence as opening a block comment, with no awareness of string literals. Both my new `next_action_hardcode_ratchet_test.go` and the pre-existing `colony_state_atomicity_ratchet_test.go` write "cmd/\*.go" in doc-comment prose -- a glob pattern, not a real comment -- and neither file contains a literal "\*/" anywhere to close it, so `TestWiringGuardsHaveNoRuntimeEscapeHatch`'s scan of either file collapsed to ~30 bytes of stripped content instead of the real ~20-30KB, silently under-scanning almost the whole file (confirmed by a temporary debug log: `next_action_hardcode_ratchet_test.go` measured `len(raw)=24944 len(stripped)=30`).
- **Fix:** reworded every "cmd/\*.go" mention in both files to "the cmd package's .go files" / "every .go file directly under cmd/" -- a comment-only change, no behavior change. `stripGoComments` itself (a shared helper other, unrelated guard files also use) was left untouched; fixing its general string-literal blindness is a larger, riskier change out of this plan's scope, and every currently-registered guard file was confirmed clean of the trigger pattern except the two this plan touched.
- **Files modified:** `cmd/next_action_hardcode_ratchet_test.go` (declared), `cmd/colony_state_atomicity_ratchet_test.go` (not declared -- required to honestly register it in Task 3).
- **Verification:** `len(stripped)` re-measured at a proportionate size after the fix (confirmed via the same temporary debug log, removed before commit); `TestWiringGuardsHaveNoRuntimeEscapeHatch` passes with both files' real content actually scanned.
- **Commit:** `7fec3144` (own file), `97ce3abf` (colony_state file)

**3. [Rule 3 -- Blocking] Four single-assertion per-wave tests deleted as fully subsumed**

- **Issue:** the plan's own Task 1 instructs: "Keep the per-wave tests if they assert something this one does not; delete them if they are now a subset, and say which in the summary."
- **Fix:** `TestSealEnvelopeMatchesCard` (`cmd/lifecycle_card_endgame_test.go`), `TestUpdateEnvelopeCarriesTheCardsFields` (`cmd/lifecycle_card_session_test.go`), `TestStatusEnvelopeCarriesTheCardsFields` and `TestStatusEndsWithTheCard` (both `cmd/lifecycle_card_endgame_test.go`) each asserted exactly one thing -- command presence/prefix, or marker presence -- over the identical fixture `TestEveryLifecycleCommandEndsWithNextAction`'s corresponding subtest now checks, with nothing extra. Deleted with a one-line pointer left in their place. Every OTHER per-wave test (the situation-specific variants: plan-only, blocked, part-finished, the two update outcomes, the two status overrides, plain-English wording, platform correctness) asserts something the new test does not and was kept unchanged.
- **Files modified:** `cmd/lifecycle_card_endgame_test.go`, `cmd/lifecycle_card_session_test.go`.
- **Verification:** full targeted re-run of all four `lifecycle_card_*_test.go` files plus the new coverage test, all pass; no assertion was weakened, only removed where fully duplicate.
- **Commit:** `ec5c8dc0`

### Scope boundaries observed

- `stripGoComments`'s core algorithm (string-literal blindness in its block-comment detection) was NOT fixed -- only the two files this plan's own work touches were reworded around the trigger. A general fix belongs to a future, dedicated plan if the same pattern surfaces again in an unrelated guard file.
- No change was made to what any command DOES, matching the phase's own "Explicitly out of scope" boundary.
- `.planning/STATE.md`, `.planning/ROADMAP.md` -- untouched, per worktree-mode convention (orchestrator's job).

---

**Total deviations:** 3 (1 blocking-registration, 1 bug fix, 1 blocking-deletion)
**Impact on plan:** No scope creep. All three are direct, minimal, mechanically-required corollaries of doing Task 3's own instructed derivation honestly, or of Task 1's own explicit instruction to delete fully-subsumed tests.

## Issues Encountered

**One transient full-suite failure, not reproduced.** The first `go test ./cmd -count=1` run of this plan's verification failed; the next two consecutive full runs (236s, 244s) both passed cleanly with zero `--- FAIL` lines, and the specific test suspected (`TestBuildDispatchStartsHeartbeatMonitor`, the exact pre-existing full-suite-only flake documented in this phase's `deferred-items.md` from waves 5 and 6) passed reliably in isolation. Consistent with the already-documented, already-deferred flake -- not investigated further, not re-logged (already recorded).

## Known Stubs

None. Every check this plan added is a real, running test with no skip path, no environment switch, and no build constraint (asserted by `TestWiringGuardsHaveNoRuntimeEscapeHatch` covering both new files).

## Threat Flags

None. This plan adds no network endpoint, no auth path, no file-access pattern and no schema at a trust boundary. It adds CI-time static checks over the repository's own Go source.

## Verification Run

| Command | Result |
|---|---|
| `go test ./cmd -run 'TestEveryLifecycleCommandEndsWithNextAction' -count=1` | ok |
| `go test ./cmd -run 'Test(EveryLifecycleCommandEndsWithNextAction\|MigratedLifecycleSurfaces)' -count=1` | ok |
| `go test ./cmd -run 'TestNextActionNeverHardcoded' -count=1` | ok |
| `go test ./cmd -run 'Test(NextActionNeverHardcoded\|OrphanAllowlistOnlyShrinks\|ColonyStateWriteAllowlistOnlyShrinks\|FlagAuditSkipListOnlyShrinks)' -count=1` | ok |
| `go test ./cmd -run 'Test(WiringGateStepRunsEveryWiringTest\|BlanketGateCheckRejectsADecoyStep\|WiringGuardsHaveNoRuntimeEscapeHatch\|AllowlistPolicyNamesEveryGuardedFile)' -count=1` | ok |
| `go test ./cmd -run 'TestEveryGuardFileIsInTheWiringInventory' -count=1` | ok |
| `go test ./cmd -run 'Test(EveryLifecycleCommandEndsWithNextAction\|NextActionNeverHardcoded\|WiringGateStepRunsEveryWiringTest\|WiringGuardsHaveNoRuntimeEscapeHatch\|EveryGuardFileIsInTheWiringInventory\|BlanketGateCheckRejectsADecoyStep\|OrphanAllowlistOnlyShrinks)' -count=1` | ok |
| `git diff --exit-code cmd/testdata/orphan_allowlist_baseline.json` | clean, no diff |
| `go build ./cmd/aether && go vet ./... && gofmt -l cmd` | clean, no output |
| `go test ./cmd -count=1` (full package, three runs) | 1 transient flake (documented pre-existing), then ok (236s), ok (244s) |

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

- **Phase 197 is complete.** All eleven of criterion 2's named commands render from the one resolver (197-04/197-06), the coverage test proves it as one named, running check, and the hardcode ratchet proves the count of remaining hand-typed advice can only fall from today's measured 116.
- **The four NEXT-02-declaring plans (197-02, 197-04, 197-06, 197-07) all finish with this plan.** Per the shared-ID gate, NEXT-02 becomes markable Complete once the orchestrator processes this SUMMARY.
- **The wiring inventory now protects itself.** `TestEveryGuardFileIsInTheWiringInventory` will catch, by name, any future ratchet-shaped guard file that is written but never registered -- the exact gap this plan's own authoring fell into once (with `colony_state_atomicity_ratchet_test.go`) before this test existed to catch it.
- **`stripGoComments`'s string-literal blindness remains a latent, undocumented-until-now property of a shared helper.** No other currently-registered guard file was found to trigger it, but a future guard file that mentions a glob pattern like `X/*.Y` in its own doc comments, with no literal `*/` anywhere later in the file, will silently under-scan itself under `TestWiringGuardsHaveNoRuntimeEscapeHatch`. Worth a dedicated, scoped fix if it resurfaces.

## Self-Check: PASSED

**Files claimed as created -- all present on disk:**

| File | Present |
|---|---|
| `cmd/lifecycle_next_action_coverage_test.go` | yes |
| `cmd/next_action_hardcode_ratchet_test.go` | yes |
| `cmd/testdata/next_action_hardcode.json` | yes |
| `cmd/testdata/next_action_hardcode_baseline.json` | yes |
| `.planning/phases/197-one-answer-to-what-next/197-07-SUMMARY.md` | yes |

**Commits claimed -- all present in `git log`:** `ec5c8dc0`, `7fec3144`, `97ce3abf`, plus this summary commit.

**Nothing changed outside the declared surface plus documented deviations.** `git diff --stat 0ec65bff..HEAD -- cmd/ .github/` lists exactly the 10 files named above (4 created, 6 modified); `.planning/STATE.md` and `.planning/ROADMAP.md` do not appear in it.

**All four temporary demonstrations reverted:** each confirmed via `git status --short cmd/` showing clean (or only the intended, already-tracked files) before its task's real commit.

---
*Phase: 197-one-answer-to-what-next*
*Completed: 2026-08-29*
