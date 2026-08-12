---
phase: 172-wiring-proof
verified: 2026-08-12T14:00:00Z
status: gaps_found
score: 2/4 roadmap success criteria fully verified (criteria 2 and 3); criterion 1's originally-reported regression is closed but a distinct, independently-confirmed defect in the same shrink-only mechanism remains; criterion 4's step- and workflow-level bypasses from the prior round are closed, but three new, independently-confirmed job-level/trigger-level bypasses remain
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "2/4 (previous run: criteria 2 and 3 verified; criterion 1 regressed to failed; criterion 4 failed)"
  gaps_closed:
    - "Success criterion 1's specific reported regression — the bare-leaf-name caller-evidence collision that let a call to `aether host colonize` silently vouch for the unrelated top-level `aether colonize` (and the same for `aether ceremony closeout` / `aether closeout`) — is closed by 172-09. Caller evidence, orphan computation, and every allowlist consumer are now keyed by the resolved cobra `CommandPath()`. Independently re-confirmed below by construction: `aether colonize` and `aether closeout` are now present in the live allowlist tagged `path-collision-revealed`/`RECLAIM`, and removing both entries from `cmd/testdata/orphan_allowlist.json` and re-running `TestNoRegisteredSubcommandIsUnreferenced` makes it fail, naming both commands by their full path — then restoring the file makes it pass again."
    - "Success criterion 4's five previously-reproduced step-scoped text-check bypasses (`; true`, `|| exit 0`, an appended second `-run`, `| cat`, a shell-commented-out run line) and the workflow-level gaps identified in the same round (a commented-out gate step, `if:`/`continue-on-error` on either gate step, a disabled `pull_request:`/`push:` trigger, a job-level `if: false`) are closed by 172-10 (an execution-based harness that runs the release gate's own command, read live from ci.yml, against a passing and a deliberately-failing probe module) and 172-11 (structural, indentation-bounded step-block anchoring plus `TestReleaseGateWorkflowActuallyRuns`). This is not narration: the orchestrator's own direct mutation of the live `.github/workflows/ci.yml` (recorded in the task context, not re-derived here) confirmed the guard suite goes red for all seven of: `; true`, `|| exit 0`, two genuinely novel mutations no plan enumerated (`if ! CMD; then true; fi` and `(CMD) ; echo done`), a commented-out `Run Go tests` step, `continue-on-error: true` at the STEP level, and removal of the `pull_request:` trigger. The execution harness generalizing to two unenumerated mutations is real evidence it is checking a property, not matching a blocklist."
  gaps_remaining:
    - "None of the previously-open items survive verbatim, but the underlying category — a check that establishes only a piece of the intended property while a nearby, structurally similar route remains open — recurs in both criteria this round, in a different specific location than before. See the two new gaps below."
  regressions: []
gaps:
  - truth: "Success criterion 1 — the companion assertion fails when the allowlist gains an entry it did not have in the committed baseline; the list may only shrink"
    status: failed
    reason: >
      Independently confirmed by construction (not inherited from 172-REVIEW.md's
      CR-04 prose): registered a genuinely new, real, permanently-orphaned command,
      `aether colony-depth setup` (a throwaway child added to the existing,
      already-called `colony-depth` parent, via the same `rootCmd.AddCommand`
      technique the ratchet's own self-tests use, so it resolves for real through
      `rootCmd.Find`, not as a fictional string), added one matching entry for it to
      BOTH `cmd/testdata/orphan_allowlist.json` and
      `cmd/testdata/orphan_allowlist_baseline.json` in the same operation (as a real
      committer adding a new feature commit would do), and re-ran the full four-test
      chain this criterion's shrink-only guarantee depends on:
      `TestOrphanAllowlistOnlyShrinks` PASSED (live is a subset of baseline — both
      files were edited together), `TestOrphanAllowlistIsPathKeyed` PASSED (the
      command genuinely resolves), `TestPathMigrationDidNotWidenTolerance` PASSED,
      and — critically — `TestNoRegisteredSubcommandIsUnreferenced`, the actual
      ratchet named in the criterion's own wording, also PASSED. Nothing went red.
      The reason: `TestPathMigrationDidNotWidenTolerance`
      (`cmd/subcommand_reachability_ratchet_test.go:1433-1460`) anchors the
      migration's shrink-only guarantee by comparing each baseline entry's LEAF name
      (`e.Name[strings.LastIndex(e.Name," ")+1:]`) against the frozen pre-migration
      snapshot's leaf-name set — not the full path. The frozen snapshot contains 14
      completely generic single-word leaves (`archive`, `export`, `get`, `import`,
      `integrity`, `lifecycle`, `proof`, `reconcile`, `registry`, `serve`, `set`,
      `setup`, `versions`, `wisdom` — independently confirmed present via direct
      inspection of `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json`).
      Any brand-new command whose leaf matches one of these 14 words launders past
      the check that exists specifically to prevent the allowlist from widening. This
      directly falsifies both the roadmap's own criterion 1 wording ("it may only
      shrink") and `.aether/docs/orphan-allowlist-policy.md`'s unqualified claim that
      "the automated check that runs on every change fails immediately, by name" —
      under CLAUDE.md's own Definition of Done ("A documentation claim about runtime
      behaviour must be testable or removed"), that sentence is false today, not
      hypothetically.
    artifacts:
      - path: "cmd/subcommand_reachability_ratchet_test.go"
        issue: "TestPathMigrationDidNotWidenTolerance (lines 1433-1460) compares baseline entries to the frozen pre-migration snapshot by bare leaf name, not by full resolved path, so any new command sharing a leaf with one of the 278 pre-migration entries passes as 'carried forward' even when it is a genuinely new, unrelated command added in the same commit as its own allowlist entry."
    missing:
      - "Replace the leaf-based comparison with an explicit accounting: every baseline entry must be either (a) the exact single path a pre-migration leaf entry expanded to 1:1, (b) a member of a small, in-source, reviewed pathMigrationExpansion map for the 4 known collapsed leaves (get/set/registry/wisdom, each expanding to a fixed, named set of real paths), or (c) a member of the already-reviewed pathCollisionRevealedOrphans set. Anything else fails by full path name, including a same-leaf newcomer."
      - "Reconfirm with the same reproduction after the fix: register a new orphan sharing a tolerated leaf, add matching entries to both files, and require TestPathMigrationDidNotWidenTolerance (or its replacement) to fail naming it."
  - truth: "Success criterion 1 (supporting property) — the frozen pre-migration snapshot that every shrink-only claim is anchored to cannot be silently edited"
    status: failed
    reason: >
      Independently confirmed by construction: appended one fabricated entry
      (`{"name": "brandnewleaf", ...}`) to
      `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` — the file
      `.aether/docs/orphan-allowlist-policy.md` and the code's own doc comments
      describe as "frozen, never-edited-again" and "byte-identical" — then re-ran
      the complete named CI wiring step's 30-test `-run` filter, copied verbatim
      from `.github/workflows/ci.yml:100`, against the live tree. All 30 tests
      PASSED, including every orphan-ratchet and path-migration guard. The file is
      referenced in exactly three places in the whole repository (two
      `loadOrphanAllowlist` calls and one string in `guardedAllowlistFiles`) and by
      no hash check, no immutability test, and no `git`-blob assertion. Combined
      with the prior gap, this means the entire "the allowlist can only shrink"
      property currently rests on a mutable, unverified file plus human review at
      commit-review time — the exact standard CLAUDE.md's Definition of Done exists
      to replace ("Not a commit. Not a checked box in a summary."). The file was
      restored byte-for-byte immediately after the reproduction; `git status
      --porcelain` was confirmed empty before and after.
    artifacts:
      - path: "cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json"
        issue: "No test asserts this file's content hash or otherwise detects a hand edit. Any addition to it silently raises the tolerance ceiling for every future entry sharing that leaf (compounding the previous gap) or, once the previous gap is fixed, for every future entry with that exact path."
    missing:
      - "Pin the file's content (e.g. a SHA-256 constant declared in Go source, matching the pattern the review suggested) and add a test that fails naming the file when the hash no longer matches, wired into the same named CI step."
  - truth: "Success criterion 4 — the ratchet and the flag test run in the same CI command the release gate already runs, verified by deleting a caller and observing the gate go red, not by reading the workflow file"
    status: failed
    reason: >
      Genuine, substantial progress this round: the step-scoped and workflow-shape
      bypasses identified in the prior verification and the prior review round are
      closed (see gaps_closed above). What remains, per the orchestrator's own
      direct mutation of the live `.github/workflows/ci.yml` (authoritative for this
      report — not re-derived, per this task's instructions) is three bypasses none
      of the current guards check for, because they operate one YAML level higher
      than every check 172-10/172-11 added:
      (1) `continue-on-error: true` set at the JOB level (`jobs.go.continue-on-error`,
      not inside any step). `stepCanFailTheBuild`
      (`cmd/ci_wiring_gate_test.go:333-348`) only scans the text INSIDE a step's
      indentation-bounded block for `continue-on-error`; a job-level key sits above
      every step block and is invisible to it. Independently confirmed by direct
      source inspection: `stepBlock`'s bounds never include text before the first
      step marker, and `TestReleaseGateWorkflowActuallyRuns` checks only a job-level
      `if:`, never `continue-on-error` or `env:`.
      (2) `env: {GOFLAGS: "-run=TestNothingZZZ"}` at the job level. The run line
      stays byte-identical to the pinned command (`blanketGateRunCommand`'s exact
      equality check still passes), nothing reads the job's `env:` block, and the
      execution harness (`runGateCommand`) inherits the LOCAL process environment
      rather than the workflow's declared one — so neither the text check nor the
      execution check sees it. CI would run zero tests and still report success.
      (3) `paths-ignore: ['**']` nested under a trigger in the `on:` block.
      `TestReleaseGateWorkflowActuallyRuns`'s check
      (`cmd/ci_wiring_gate_test.go:1027-1031`) is `strings.Contains(onBlock,
      "pull_request:")` and the same for `"push:"` — both substrings remain present
      alongside a `paths-ignore` filter that excludes every path, so the assertion
      passes while the workflow never actually fires for a real change.
      This is the third round of attempts at this criterion, and the honest
      assessment is that the underlying property — "the gate cannot be silently
      narrowed" — is closer to true than at any prior round (four distinct classes
      of bypass are now closed: step-text mutation, step-conditional, workflow-step
      absence, and trigger/job-`if:`-level disabling) but is still not fully true,
      because the checks so far have been added one YAML scope at a time
      (step-text, then step-conditional, then job-`if:`/trigger-presence) rather
      than as a complete enumeration of "every YAML key between the workflow root
      and the command text that can disable or redirect execution."
    artifacts:
      - path: "cmd/ci_wiring_gate_test.go"
        issue: "stepCanFailTheBuild only scans within a step's own indentation-bounded block, never the enclosing job's top-level keys; TestReleaseGateWorkflowActuallyRuns checks job-level if: but not job-level continue-on-error or env:; the on:-block check is a substring presence test for trigger names, not an absence test for paths-ignore/paths/branches-ignore filters that could exclude everything."
    missing:
      - "Extend TestReleaseGateWorkflowActuallyRuns (or a sibling test) to reject a job-level continue-on-error key anywhere between `jobs: go:` and the first step marker."
      - "Either assert the job carries no job-level env: key that could override GOFLAGS/GOTEST-affecting variables, or make the execution harness invoke the command through the same environment composition the real workflow would use (a deliberately narrower, riskier option given this file's own escape-hatch scanner forbids os.Environ) — the job-level assertion is the safer fix."
      - "Reject any paths-ignore, paths, branches-ignore, or branches key nested under pull_request:/push: in the on: block, or explicitly assert the trigger fires unconditionally for the whole repository — a presence-of-trigger-name check is not a fires-for-every-change check."
deferred: []
human_verification: []
---

# Phase 172: Wiring Proof Verification Report

**Phase Goal:** A capability added by this milestone cannot ship without a caller. The orphan
ratchet exists, runs in CI, and blocks — before any of the capabilities it constrains are
built, so it is shaped by the standard rather than by whatever shipped.

**Verified:** 2026-08-12T14:00:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plans 172-09 (wave 7), 172-10 (wave 8), and 172-11
(wave 9), and after independently checking a code review (172-REVIEW.md, re-review, 5 critical
findings) completed against the post-gap-closure state.

## Goal Achievement

### Observable Truths (ROADMAP.md § Phase 172 Success Criteria, verbatim)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Registering a new cobra subcommand with no caller makes `TestNoRegisteredSubcommandIsUnreferenced` fail, naming the command; allowlist seeded from real scanner output (now 293 path-keyed entries); shrink-only against a committed baseline | ✗ FAILED | The specific regression reported last round (`aether colonize`/`aether closeout` bare-leaf collision) is genuinely closed — re-confirmed by removing both entries from the live allowlist and observing the test fail naming both, then restoring it and observing it pass. But a distinct, independently-constructed defect in the same shrink-only mechanism is confirmed live: a real, newly-registered orphan command sharing a generic leaf name (`aether colony-depth setup`) added to both allowlist files in one operation passes every relevant guard, including the ratchet itself. See gaps. |
| 2 | `aether spawn-can-spawn 5 --enforce` exits 0 | ✓ VERIFIED | Re-ran `go run ./cmd/aether spawn-can-spawn 5 --enforce` directly: `{"ok":true,"result":{"can_spawn":true,"depth":5}}`, exit 0. |
| 3 | A test enumerates every `aether …` invocation in `.aether/*.md` and fails naming file/line/flag when an instruction names a flag the binary does not register | ✓ VERIFIED | Re-ran `TestCLIFlagAudit` (141 files across 3 corpora, 126 subcommands audited), `TestCommandCallsMatchCobraContracts` (988 documented invocations across 6 corpora), `TestAuditedCorpusHasNoGluedFenceMarkers` (215 files scanned) — all pass. No regression from the prior round's closed fence-parsing gap. |
| 4 | The ratchet and the flag test run in the same CI command the release gate already runs — verified by deleting a caller and observing the gate go red, not by reading the workflow file | ✗ FAILED | Substantial, real progress this round: the prior round's five step-text bypasses and the workflow-shape gaps found alongside them (commented-out step, step-level `if:`/`continue-on-error`, disabled trigger, job-level `if:`) are closed by 172-10's execution-based harness and 172-11's structural checks — confirmed by the orchestrator's direct mutation of the live workflow catching all seven tested variants, including two genuinely novel mutations no plan enumerated. But three new bypasses one YAML scope higher than any current check remain live today, also confirmed by the orchestrator's direct mutation of the live workflow: job-level `continue-on-error`, job-level `env: GOFLAGS`, and `paths-ignore` under a trigger. See gaps. |

**Score:** 2/4 fully verified (criteria 2, 3); 2 failed (criterion 1 — original regression closed, new defect in the same mechanism confirmed; criterion 4 — four bypass classes closed, three new ones confirmed one scope higher).

### Independent Reproductions Performed In This Verification

All reproductions below were constructed directly against the live source in this repository —
not read from 172-REVIEW.md's prose — using the same technique the codebase's own self-tests
use (`rootCmd.AddCommand` for a real, resolvable synthetic fixture). Every file touched was
restored byte-for-byte immediately afterward; `git status --porcelain` was confirmed empty
before and after each reproduction and at the end of this verification.

| # | Claim | Method | Result |
|---|---|---|---|
| 1 | CR-04 (shrink-only guarantee is launderable) | Registered a real cobra command `aether colony-depth setup` (child of the existing, already-called `colony-depth` parent) via package `init()` in a throwaway `_test.go` file; added a matching entry to both `orphan_allowlist.json` and `orphan_allowlist_baseline.json`; ran `TestOrphanAllowlistOnlyShrinks`, `TestOrphanAllowlistIsPathKeyed`, `TestPathMigrationDidNotWidenTolerance`, `TestNoRegisteredSubcommandIsUnreferenced` | All four PASSED — confirmed. (A first attempt using a brand-new fictional parent `aether newthing get` correctly failed on the bare parent `aether newthing`, which is itself enumerated as a registered command with no caller — that variant does not demonstrate the laundering path; the corrected reproduction using an existing, already-vouched-for parent isolates the actual defect.) |
| 2 | CR-05 (frozen pre-migration snapshot has no integrity pin) | Appended one fabricated entry to `orphan_allowlist_baseline_pre_path_migration.json`; ran the full named CI wiring step's 30-test `-run` filter copied verbatim from `.github/workflows/ci.yml:100` | All 30 tests PASSED — confirmed. Nothing detects the edit. |
| 3 | Criterion 1's specific prior regression is closed | Removed `aether colonize`/`aether closeout` from the live allowlist; ran `TestNoRegisteredSubcommandIsUnreferenced` | FAILED, naming both by full path, as expected. Restored the file; re-ran; PASSED. |
| 4 | Criterion 2 still holds | `go run ./cmd/aether spawn-can-spawn 5 --enforce` | Exit 0, `{"ok":true,...}` — confirmed. |
| 5 | CR-01/CR-02/CR-03 (job-level and trigger-level bypasses) | Not re-mutated in this verification (already independently confirmed live by direct mutation of the real `.github/workflows/ci.yml`, per the orchestrator's findings supplied with this task) | Corroborated by direct source inspection: `stepCanFailTheBuild` (`cmd/ci_wiring_gate_test.go:333-348`) scans only within a step's own indentation-bounded block, never the job's top-level keys; `TestReleaseGateWorkflowActuallyRuns` checks job-level `if:` only, never `continue-on-error` or `env:`; the `on:`-block check is `strings.Contains(onBlock, "pull_request:")`/`"push:"`, a presence test with no check for `paths-ignore`/`paths`/`branches-ignore` filters. The source confirms exactly the mechanism the orchestrator's live mutations exploited. |

### Full Named CI Guard-Test Set (re-run against current tree, unmutated)

```
go test ./cmd -run '<30-name filter copied verbatim from ci.yml:100>' -count=1 -timeout 900s -v
```

All 30 tests pass in ~7.1s. `go build ./...` and `go vet ./cmd/...` are clean. This confirms the
gap-closure summaries did not fabricate green output — the guard suite is genuinely green on the
unmutated tree. The findings in this report are about what the green suite does *not* yet check,
demonstrated by construction, not about the suite lying regarding what it does check.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| WIRE-01 | 172-00, 172-02, 172-04, 172-05, 172-07, 172-08, 172-09 | A test fails when a registered subcommand has no caller outside its own definition, seeded with today's known orphans in a shrink-only allowlist | ✗ BLOCKED | The prior blocking defect (colonize/closeout bare-leaf collision) is fixed. A new, independently-confirmed defect (CR-04/CR-05: the shrink-only guarantee is launderable by a same-leaf newcomer, and its anchor file is unpinned) blocks this requirement today. REQUIREMENTS.md's checklist prose marks WIRE-01 `[x]` (line 32) while its traceability table still marks it `Pending` (line 139) — unchanged since the prior verification round; the traceability table remains the more accurate of the two. |
| WIRE-02 | 172-01, 172-05, 172-07, 172-10, 172-11 | The documented `.aether/workers.md` invocation matches a real flag; running it succeeds rather than erroring on `--enforce` | ✓ SATISFIED | Confirmed by direct execution above. |
| WIRE-03 | 172-00, 172-03, 172-04, 172-05, 172-06, 172-07, 172-08 | A test fails when a `.aether/*.md` instruction names a CLI flag the binary does not register | ✓ SATISFIED | Confirmed via re-run of the full named guard-test set and the (still closed) fence-parsing gap. |

No orphaned requirements: WIRE-01/02/03 are the only IDs REQUIREMENTS.md maps to Phase 172, and
all three appear in at least one plan's `requirements` field (172-09/10/11 all declare all
three, reflecting that the gap-closure plans touch shared guard infrastructure).

### Anti-Patterns / Additional Findings (not blocking the 4 roadmap criteria on their own, but load-bearing for the phase's stated purpose)

| File | Finding | Severity | Impact |
|---|---|---|---|
| `cmd/ci_wiring_gate_test.go:376` | The blanket step's `blanketGateRunCommand` exact-equality check is explicitly documented in its own doc comment as "not the proof" — `TestReleaseGateCommandFailsATreeWithAFailingTest` is named as the actual proof. This is good practice (prevents a future reader from mistaking a whitelist for a guarantee) and is noted here as a positive finding, not a gap. | ℹ️ Info | Reduces the risk of a fourth premature "wired" claim resting on the wrong check. |
| `.aether/docs/orphan-allowlist-policy.md:43-46` | States without qualification that the allowlist "can only get shorter... the automated check that runs on every change fails immediately, by name." This is currently false per the CR-04 reproduction above. | 🛑 Blocker (documentation claim about runtime behavior that is not currently testable-true) | Under CLAUDE.md's own Definition of Done, this sentence must be corrected or the underlying check must be fixed before the claim is accurate again. |
| `cmd/ci_wiring_gate_test.go` (job-level scope) | No test inspects `jobs.go`'s own top-level keys (`continue-on-error`, `env`) independent of its steps. | ⚠️ Warning | This is the direct mechanism behind CR-01/CR-02, both confirmed live by direct mutation. |

### Gaps Summary

Two of the four roadmap success criteria remain unmet on the current tree, but both are
materially closer to met than in the prior verification round, and both failures are now
narrower and better-characterized than before:

1. **Criterion 1's originally-reported regression (colonize/closeout) is genuinely fixed.**
   Path-keyed caller evidence closes the specific defect this phase's own prior verification
   found. In its place, an independently-constructed reproduction confirms a different defect
   in the same shrink-only mechanism: `TestPathMigrationDidNotWidenTolerance` compares by bare
   leaf name rather than full path, so a brand-new orphan sharing one of 14 generic tolerated
   leaves (`get`, `set`, `setup`, `export`, etc.) passes every guard, including the ratchet
   itself, when its allowlist entry is added in the same commit — which is exactly how a real
   contributor would add both a feature and its (temporary or permanent) allowlist entry
   together. Compounding this, the frozen snapshot file the whole migration's shrink-only claim
   is anchored to has no integrity pin and can be hand-edited with nothing noticing.

2. **Criterion 4 has closed four distinct bypass classes since the last verification round**
   (step-text mutation via the new execution-based harness; step-level `if:`/`continue-on-error`;
   workflow-step absence; and disabled triggers/job-level `if:`, via new structural checks) —
   this is real, substantive engineering, not narration, and the execution-based harness in
   particular is evidence-backed as generalizing beyond its enumerated mutation list (it caught
   two mutations no plan wrote down). What remains is three bypasses one YAML scope higher than
   any current check operates at: job-level `continue-on-error`, job-level `env:` overrides, and
   `paths-ignore` filters under a trigger. This is the third round of work on this criterion;
   an honest read is that each round has correctly closed what it targeted but has not yet
   enumerated the full set of YAML scopes between the workflow root and the command text that
   can disable or redirect execution.

**Is the execution-based harness (172-10) the right foundation to keep building on?** Yes, for
what it can see. `gateCommandDiscriminates` makes no assumption about how a mutation is spelled
— it runs the real command against a passing and a failing probe and checks exit codes — which
is why it caught two mutations nobody enumerated. It is not, however, a substitute for asserting
that the command is *reached at all* by the workflow; job-level `continue-on-error`, job-level
`env:` overrides, and trigger-path filtering all prevent the command from running in a form the
harness would ever see, because the harness only runs when a human (or CI) invokes `go test ./cmd
-run TestReleaseGateCommandFailsATreeWithAFailingTest` directly — it does not simulate GitHub
Actions' own job/trigger evaluation. The correct next step is the same direction 172-11 already
took (structural, workflow-shape assertions in `TestReleaseGateWorkflowActuallyRuns`), extended
to cover the job's own top-level keys and the trigger's path/branch filters — not a fourth
attempt at a smarter text check on the step's run line, and not a replacement for the execution
harness. Both mechanisms are complementary and both should be kept.

---

_Verified: 2026-08-12T14:00:00Z_
_Verifier: Claude (gsd-verifier)_
