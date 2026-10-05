---
phase: 207-messy-practice-project-gate
verified: 2026-09-22T14:37:44Z
status: passed
score: 6/6 must-haves verified (roadmap success criteria) — 0 behavior_unverified
behavior_unverified: 0
overrides_applied: 0
re_verification: No — initial verification
---

# Phase 207: A Messy Practice Project Is the Release Gate — Verification Report

**Phase Goal:** Dead ends are found by an automated journey through a deliberately messy project before the owner finds them.
**Verified:** 2026-09-22T14:37:44Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP success criteria, the authoritative contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A committed script builds the practice project with every trap named in the milestone section. | ✓ VERIFIED | `scripts/build-messy-practice-project.sh` exists, executable. `cmd/testdata/journey/traps.json` declares exactly the 9 roadmap-named traps (`shortcut-to-a-folder`, `shortcut-loop`, `case-only-archive-collision`, `nested-project`, `long-folder-names`, `out-of-date-code-map`, `specification-corrected-mid-planning`, `leftover-junk-data`, `unsaved-changes`). `TestMessyPracticeProjectHasEveryTrap`, `TestMessyPracticeProjectTrapListAndScriptAgree`, `TestMessyPracticeProjectBuilderRunsCleanTwice` (runs the real script twice, asserts identical marker trap_ids both times), `TestMessyPracticeProjectBuilderRefusesAForeignDirectory` (refuses a directory with an unrelated file, leaves it untouched) all pass — this directly exercises the backstop-tagged "refuse foreign / build fresh / re-verify own output" truth, not just presence. |
| 2 | The whole journey runs through a real chat with hooks and menu commands loaded; asserts on files/commands never wording; three trials; failures sorted into flaky/real; money and turn caps. | ✓ VERIFIED | `cmd/journey_live_test.go`'s `TestJourney` drives all 14 steps via `claude -p --resume`, never `--dangerously-skip-permissions`-style flags that would skip hooks; assertions resolve to `<command-name>` tags and Bash `tool_use` entries in the real on-disk transcript (`journeyFindSessionTranscript`), never chat prose. `journeyMinimumTrials = 3` (`cmd/journey.go`). `classifyJourneyFailure` sorts flaky vs. real, proven directly by `TestJourneyFailureClassifierIsSmallAndExplicit` (asserts `"unknown flag: --frobnicate"`, `"not authenticated"` classify as real; `"429 rate limit exceeded"`, `"OVERLOADED"` classify as transient). Every chat call carries `--max-turns`, a wall-clock timeout, and `--max-budget-usd` when supported (`journeyCapsForStep`, `journeyClaudeSupportsBudgetFlag`), recorded in the report (`journeyCaps` struct). The real three-trial run (`207-JOURNEY-RUN.md`) shows all these mechanisms firing for real, not just compiling. |
| 3 | With each of the 2026-09-21 fixes reverted in turn, the journey fails at that step. **Amended by D-01: only 5 of 6 have a landed fix; the 6th check is built, honestly red, and named for Phase 208.** | ✓ VERIFIED | `cmd/testdata/journey/fix-reverts.json` holds exactly 5 entries (confirmed by direct read: `write-allowlist`, `archive-name-case`, `helper-fingerprint`, `superseded-specification`, `pause-follows-shortcuts`). `cmd/testdata/journey/expected-red.json` holds exactly 1 standing case naming Phase 208/UED-13 as what closes it. `207-JOURNEY-RUN.md` documents a real five-for-five `make prove-journey-fix-reverts` run (real session ids, real cost, real wall clock) — all five reverts caught at the declared step. `TestFixRevertTableNamesFiveLandedFixes`, `TestPhaseProvesFiveFixesNotSix`, `TestSixthBlockerCheckIsStillRed`, `TestExpectedRedRegisterOnlyShrinks` all pass. `grep -ci "all six" CLAUDE.md` returns 0. ROADMAP.md's own success criterion 3 carries the D-01 amendment verbatim. |
| 4 | The journey is the gate for every release from here on. | ✓ VERIFIED | `.aether/docs/publish-update-runbook.md` Preflight section names `make eval-gate-journey` as a required pre-publish step. `cmd/testdata/eval-gates/gates.json` registers a `journey` gate (`budget_seconds: 1100`, measured, not provisional — text names the measuring run). `Makefile`'s `eval-gate-journey` target timeout (1500s) matches the gate's declared budget with headroom (WR-01 fix confirmed landed). CLAUDE.md documents the mechanism with named tests for every claim. |

**Score:** 4/4 roadmap success criteria verified (all four map to the six per-plan must-have clusters below; see per-plan detail for the finer-grained backstop items, all independently checked).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `scripts/build-messy-practice-project.sh` | Idempotent practice-project builder, all 9 traps | ✓ VERIFIED | Exists, executable; offline tests pass |
| `cmd/journey.go` | Transcript/report/verdict library | ✓ VERIFIED | `journeyGateVerdict`, `classifyJourneyFailure`, report schema all present and tested |
| `cmd/journey_traps.go` | Shared trap manifest reader | ✓ VERIFIED | Read by both script (jq) and Go |
| `cmd/journey_live_test.go` | `TestJourney`, live `//go:build journey` harness | ✓ VERIFIED | Builds under `-tags=journey`; `go vet -tags=journey ./cmd/...` clean |
| `cmd/testdata/journey/traps.json` | 9 declared traps | ✓ VERIFIED | Confirmed by direct JSON read |
| `cmd/testdata/eval-gates/gates.json` | `journey` gate entry, measured budget | ✓ VERIFIED | `budget_seconds: 1100`, purpose text names the measuring run |
| `Makefile` | `eval-gate-journey`, `prove-journey-fix-reverts` targets | ✓ VERIFIED | Both present; timeout matches gates.json (WR-01 fixed) |
| `cmd/journey_expected_red.go` / `cmd/testdata/journey/expected-red.json` | Sixth-blocker register | ✓ VERIFIED | Exactly 1 case, derives commands from real guidance code, not a re-typed list |
| `cmd/journey_fix_reverts.go` / `cmd/testdata/journey/fix-reverts.json` / `scripts/prove-journey-catches-the-2026-09-21-fixes.sh` | 5-entry revert table + harness | ✓ VERIFIED | 5 entries confirmed; harness never touches owner's checkout (`TestFixRevertHarnessNeverTouchesTheOwnersCheckout` passes, `git clean` added to forbidden list per WR-03) |
| `cmd/journey_seed.go` | Hidden fixture-construction commands | ✓ VERIFIED | Gated `//go:build journey` (excluded from release binary — confirmed live: `./aether journey-seed-stale-survey --help` → "unknown command"); requires `.journey-practice-project.json` marker (CR-01 fixed) |
| `.planning/phases/207-messy-practice-project-gate/207-JOURNEY-RUN.md` | Real measured run record | ✓ VERIFIED | Present, detailed, honest about the real dead end found |
| `.aether/docs/publish-update-runbook.md`, `CLAUDE.md` | Journey documented as release gate | ✓ VERIFIED | Both updated; CLAUDE.md section names a test for every claim |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/testdata/journey/traps.json` | builder script + Go test | jq / `journeyTraps` reader | ✓ WIRED | `TestMessyPracticeProjectTrapListAndScriptAgree` passes |
| `cmd/eval_gates.go` closed vocabulary | `gates.json` `journey` entry | `Makefile` `eval-gate-journey` | ✓ WIRED | `TestEvalGate*` pass; all three name the gate |
| `loadJourneyExpectedRed` | journey report `expected_red` section | `journeyGateVerdict`'s stale-register refusal | ✓ WIRED | `TestJourneyGateVerdictRefusals/every_expected-red_case_must_be_still-red` passes |
| `cmd/testdata/journey/fix-reverts.json` | `scripts/prove-journey-catches-the-2026-09-21-fixes.sh` + `journey_fix_reverts_test.go` | one table, two readers | ✓ WIRED | `TestFixRevertTableNamesFiveLandedFixes` and the real `make prove-journey-fix-reverts` run (207-JOURNEY-RUN.md) both confirm |
| Measured run | `gates.json` `budget_seconds` | `Makefile` timeout | ✓ WIRED | 1100s in manifest, 1500s Makefile timeout (headroom), text names the measuring run |

### Behavioral Spot-Checks / Probe Execution

Live `TestJourney` / `make eval-gate-journey` / `make prove-journey-fix-reverts` / `claude -p` were **not run** in this verification per explicit instruction (real money spend). Evidence instead comes from:
- Offline test suite: `go test ./cmd -run 'TestJourney|TestMessyPracticeProject|TestExpectedRed|TestSixthBlocker|TestFixRevert|TestEvalGate|TestStatusGuidance|TestPhaseCountsFiveRevertsAndOneStandingRedCase|TestPhaseProvesFiveFixesNotSix' -count=1 -timeout 900s` → **all pass**, zero failures.
- `go build ./cmd/aether` → clean. `go vet ./cmd/...` → clean. `go vet -tags=journey ./cmd/...` → clean.
- `./aether journey-seed-stale-survey --help` → `Error: unknown command` (confirms CR-01 fix live, not just by test).
- `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced` → fails only on the pre-existing, documented, out-of-scope orphan (`aether codex-native-worker context-ack`), exactly as the task instructions predicted.
- `207-JOURNEY-RUN.md`'s recorded real three-trial and five-revert runs, cross-checked against `journeyGateVerdict`'s actual refusal logic (`TestJourneyGateVerdictRefusals` proves the verdict function would indeed refuse a report where a trial declares 14 steps but executes 2 — matching the documented real result).

### Code Review Fixes (207-REVIEW.md) — re-verified independently

| Finding | Fix | Verified how |
|---------|-----|---------------|
| CR-01 (hidden commands could corrupt real projects) | `//go:build journey` + marker-file guard | `go build ./cmd/aether` excludes the command; `./aether journey-seed-stale-survey --help` → unknown command |
| CR-02 (env leak breaks subsequent tests) | `os.LookupEnv`/`os.Unsetenv` + filtered subprocess env | `go test ./cmd -run 'TestSixthBlockerCheckIsStillRed\|TestMessyPracticeProjectHasEveryTrap'` passes together (previously failed deterministically per the review's own reproduction) |
| WR-01 (Makefile budget stale) | Makefile updated to 1500s/1100s-measured wording | `grep -n eval-gate-journey Makefile` shows the corrected comment and timeout |
| WR-02 (stderr classification divergence) | stderr persisted to sibling file, classified from stdout+stderr concatenation | Code inspected directly in `journeyRunStep`/`journeyDriveStep`; matches the described fix exactly |
| WR-03 (`git clean` missing from forbidden list) | Added to test and script comment | `grep -n "git clean"` in both files confirms |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| UED-07 | 207-01, 207-02 | Committed script builds practice project with every roadmap trap | ✓ SATISFIED | All 9 traps present, each with a named, genuinely-failable check; idempotency proven live |
| UED-08 | 207-01, 207-04 | Whole journey through real chat, files/commands not wording, 3 trials, flaky/real, caps | ✓ SATISFIED | `TestJourney` structure, `journeyGateVerdict`, caps, classifier all directly verified; real run recorded |
| UED-09 | 207-03, 207-05 | Journey catches each of the six 2026-09-21 blockers | ✓ SATISFIED (narrowed per owner ruling D-01) | Five of six proven live (five-for-five revert run); sixth is a standing, honestly-red, named case for Phase 208 — exactly as D-01 requires. ROADMAP.md success criterion 3 carries the D-01 amendment verbatim, so the roadmap contract (the authoritative source per this verifier's Step 2a) already reflects the narrowed scope. |

**Orphaned requirements check:** `grep -n "Phase 207" .planning/REQUIREMENTS.md` — none found beyond UED-07/08/09, all three accounted for above. No orphans.

**Minor documentation note (non-blocking):** `.planning/REQUIREMENTS.md`'s one-line UED-09 description still reads "The journey catches each of 2026-09-21's six blockers" without the D-01 amendment inline (the amendment lives in ROADMAP.md's success criterion 3 and in `207-CONTEXT.md`). This is a wording staleness in the terse requirements list, not a functional gap — the authoritative roadmap contract, the phase's own context file, CLAUDE.md, and every test in the codebase are all explicit and consistent that five (not six) are proven. Not treated as a gap because the owner's own ruling (D-01) is the controlling decision and it is faithfully implemented everywhere that matters; flagged here only so REQUIREMENTS.md's wording can be tidied at low cost in a future pass.

### Anti-Patterns Found

None. No `TBD`/`FIXME`/`XXX` markers found in phase-modified files. No stub returns, no hardcoded empty data flowing to assertions. The one deliberate, documented, guarded "hidden command" pattern (`journey_seed.go`) was flagged by code review and independently confirmed fixed above.

### WINDOWS.md Cross-Check

Entry 53 (the out-of-date-code-map dead end found live by all three real trials at the "survey" step) is correctly filed as Phase 208 work, not a Phase 207 gap — the journey's own on-disk fact check caught it exactly as UED-08 intends, and the check was not weakened to get past it. This is the phase's success criterion 2/4 working as designed, not a defect in this phase.

### Human Verification Required

None. All must-haves resolve to direct code/test evidence or to real, recorded live-run transcripts (207-JOURNEY-RUN.md) rather than requiring a fresh paid `claude -p` run to confirm.

### Gaps Summary

No gaps found. The phase goal — "dead ends are found by an automated journey through a deliberately messy project before the owner finds them" — is demonstrably achieved: the journey was run for real, found a genuine dead end (WINDOWS #53), reported it honestly without being weakened, and separately proved (five-for-five, live) that it would have caught each of the five already-fixed 2026-09-21 blockers had their fixes been reverted. The sixth blocker's check is real, wired to the runtime's own guidance code, and honestly red, exactly as owner ruling D-01 specifies — never claimed as caught, never silently skipped. All five code-review findings (2 critical, 3 warning) are independently confirmed fixed in the current codebase, not merely claimed fixed in 207-REVIEW.md's prose.

---

_Verified: 2026-09-22T14:37:44Z_
_Verifier: Claude (gsd-verifier)_
