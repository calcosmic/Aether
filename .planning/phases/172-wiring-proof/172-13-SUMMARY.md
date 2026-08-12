---
phase: 172-wiring-proof
plan: "13"
subsystem: ci-wiring-guards
tags: [orphan-allowlist, ci-gate, ratchet, gap-closure]
dependency-graph:
  requires: ["172-12"]
  provides:
    - "pathMigrationExpansion / pathMigrationToleratedPaths / pathMigrationWideningProblems (full-path shrink-only accounting)"
    - "preMigrationSnapshotSHA256 / TestPreMigrationSnapshotIsFrozen (frozen anchor integrity pin)"
  affects:
    - "cmd/subcommand_reachability_ratchet_test.go"
    - "cmd/ci_wiring_gate_test.go"
    - ".aether/docs/orphan-allowlist-policy.md"
    - ".github/workflows/ci.yml"
tech-stack:
  added: []
  patterns:
    - "full-path accounting instead of bare-leaf-name comparison for a shrink-only ratchet"
    - "SHA-256 pin in Go source as an integrity anchor for a frozen JSON fixture"
key-files:
  created: []
  modified:
    - "cmd/subcommand_reachability_ratchet_test.go"
    - "cmd/ci_wiring_gate_test.go"
    - ".aether/docs/orphan-allowlist-policy.md"
    - ".github/workflows/ci.yml"
decisions:
  - "Compare full command paths, not bare last words, when checking whether the allowlist widened across the 172-09 path-key migration (CR-04)."
  - "Pin the frozen pre-migration snapshot's SHA-256 in Go source rather than trusting the JSON file's contents at face value (CR-05)."
  - "Fold WR-06 (policy doc description of this exact mechanism) into this plan rather than opening a separate one, per the plan's explicit scope note."
metrics:
  duration: "~1.5h"
  completed: "2026-08-12"
---

# Phase 172 Plan 13: Full-path allowlist accounting and frozen-anchor integrity pin Summary

Closed CR-04 and CR-05 — the two round-4 defects that made "the allowlist may only
shrink" false: a bare-last-word comparison let a brand-new unwired command slip past
every guard by sharing a common word (`get`, `setup`, ...) with an older tolerated
command, and the frozen migration anchor had no integrity check, so a hand-edited line
passed all 30 named guard tests.

## What changed

**Task 1 — CR-04, full-path accounting (commit `f73fa7c7`).** Replaced the bare-leaf
`preLeaves` membership check inside `TestPathMigrationDidNotWidenTolerance` with:

- `pathMigrationExpansion` — a frozen `map[string][]string` naming the exact 6
  non-trivial leaves (`get`, `set`, `registry`, `wisdom`, `archive`, `lifecycle`) and
  the full command paths each legitimately became.
- `preMigrationSnapshotEntryCount = 278` and `pathMigrationExpandedPathCount = 284` —
  two pinned constants that make widening the expansion map a reviewable two-place diff.
- `pathMigrationToleratedPaths(pre) (map[string]bool, error)` — expands the 278 frozen
  leaves into the 284 tolerated full paths, erroring on a wrong entry count, a duplicate
  leaf, a dead expansion key, or a wrong resulting set size (anti-vacuity guards).
- `pathMigrationWideningProblems(baseline, tolerated) []string` — pure function
  reporting every baseline entry not in `tolerated` or `pathCollisionRevealedOrphans`.
- Rewrote `TestPathMigrationDidNotWidenTolerance` to use these helpers, comparing full
  paths instead of bare leaves. The bare-leaf `preLeaves` variable no longer exists in
  the file's non-comment source (`grep -c` confirms 0).

**Task 2 — by-construction proof (commit `9077694f`).** Added
`TestPathMigrationRejectsASameLeafNewcomer`: 9 rejection rows (including the review's
own `aether newthing get` example and the verifier's live `aether colony-depth setup`
reproduction), 15 acceptance rows (every legitimate expansion path plus two
`pathCollisionRevealedOrphans` members), and a generality loop asserting all 278
frozen leaves reject a same-leaf newcomer at once. Wired into the CI `-run`
alternation. Then reproduced the verifier's exact counterexample live: registered
`aether colony-depth setup` as a real cobra child via a throwaway `_test.go` file,
added matching entries to both JSON allowlists, and confirmed
`TestPathMigrationDidNotWidenTolerance` now fails naming it — before this change all
four tests in that chain passed.

**Task 3 — CR-05, integrity pin (commit `6de0b159`).** Added `preMigrationSnapshotSHA256`
(the measured hash, `873cad5a...45e0e`) and `TestPreMigrationSnapshotIsFrozen`, which
reads the frozen anchor file, hashes it, and fails naming both hashes on any mismatch.
Wired into CI. Updated `.aether/docs/orphan-allowlist-policy.md`'s last-word rule
description and the false "278 carried forward unchanged" arithmetic to the real
274 + 10 = 284, 284 + 9 = 293 accounting. Updated `ci_wiring_gate_test.go`'s
anti-vacuity floor comment/message from "measured 32" to "measured 34" (the two new
tests this plan added).

## Re-measured arithmetic (STOP-rule required check)

Independently re-measured against the three checked-in JSON files before writing any
code, using both a standalone Python cross-check and the Go tests themselves:

| Quantity | Plan's stated figure | Measured figure | Match |
|---|---|---|---|
| Frozen pre-migration snapshot entries | 278 | 278 | yes |
| Baseline entries | 293 | 293 | yes |
| Tolerated path set (272 trivial + 2 relocated + 10 collapsed) | 284 | 284 | yes |
| Reviewed path-collision-revealed set | 9 | 9 | yes |
| 284 + 9 | 293 | 293 | yes |
| SHA-256 of the frozen snapshot | `873cad5a20e472af7050e69042be9faf8527a2c3e1d34907762b37e6a5845e0e` | same | yes |

Every figure matched the plan's interfaces block exactly. No STOP was triggered.

## Live `aether colony-depth setup` red proof (the plan's single most important criterion)

Registered `aether colony-depth setup` as a real cobra child of the existing
`colony-depth` parent via a throwaway `cmd/zzz_cr04_live_repro_test.go` (`init()` +
`colonyDepthCmd.AddCommand`), then added a matching entry to both
`cmd/testdata/orphan_allowlist.json` and `cmd/testdata/orphan_allowlist_baseline.json`
(293 → 294 entries each).

```
=== RUN   TestNoRegisteredSubcommandIsUnreferenced
    enumerated 406 registered commands, found 294 orphans
--- PASS: TestNoRegisteredSubcommandIsUnreferenced (0.32s)
=== RUN   TestOrphanAllowlistOnlyShrinks
--- PASS: TestOrphanAllowlistOnlyShrinks (0.00s)
=== RUN   TestOrphanAllowlistIsPathKeyed
--- PASS: TestOrphanAllowlistIsPathKeyed (0.00s)
=== RUN   TestPathMigrationDidNotWidenTolerance
    tolerated path set size = 284 (want 284); baseline size = 294
    1 command(s) were added to the tolerated list without ever having been tolerated
    before, by full path: aether colony-depth setup
--- FAIL: TestPathMigrationDidNotWidenTolerance (0.00s)
```

`git status --porcelain` before restore:
```
 M .github/workflows/ci.yml
 M cmd/subcommand_reachability_ratchet_test.go
 M cmd/testdata/orphan_allowlist.json
 M cmd/testdata/orphan_allowlist_baseline.json
?? cmd/zzz_cr04_live_repro_test.go
```

**Harness-integrity proof**: temporarily added `"aether colony-depth setup": true` to
`pathCollisionRevealedOrphans` and re-ran — `TestPathMigrationDidNotWidenTolerance`
PASSED (`tolerated path set size = 284 ... baseline size = 294`), proving the
rejection is accounting-driven, not an unrelated failure. Reverted the exemption and
confirmed the reproduction went red again with the identical message above.

Deleted the throwaway file, restored both JSON files byte-for-byte (verified `diff`
identical to a pre-mutation backup), re-ran the four-test chain — all PASS
(`enumerated 405 registered commands, found 293 orphans`).

`git status --porcelain` after restore:
```
 M .github/workflows/ci.yml
 M cmd/subcommand_reachability_ratchet_test.go
```
(the two intentional, in-progress Task 2 edits — no testdata or throwaway files remain)

## Generality loop

```
generality loop: built 278 newcomer paths from all 278 frozen leaves; 278 reported as
widening problems
```
(`cmd/subcommand_reachability_ratchet_test.go:1717`, logged by
`TestPathMigrationRejectsASameLeafNewcomer/generality_loop_all_frozen_leaves`)

## CR-05 red proof against the full 34-name CI filter

Appended `{"name": "brandnewleaf", "reason": "test", "owner_phase": "test"}` to
`cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` (278 → 279 entries),
then ran the full named `-run` filter copied verbatim from `.github/workflows/ci.yml`
line 100 (34 top-level test names).

Required outcome achieved:
```
=== RUN   TestPreMigrationSnapshotIsFrozen
    testdata/orphan_allowlist_baseline_pre_path_migration.json has SHA-256
    f7a6053d5b6196602d7111448079840015a87fdddef6721fe678d75deb0c9008, want
    873cad5a20e472af7050e69042be9faf8527a2c3e1d34907762b37e6a5845e0e
    (preMigrationSnapshotSHA256)
    This file is the anchor every "the list can only shrink" guarantee rests on and
    must never be edited. If it genuinely must change, that is a reviewed edit to the
    preMigrationSnapshotSHA256 constant in Go source, not a silent JSON edit.
--- FAIL: TestPreMigrationSnapshotIsFrozen (0.00s)
```
(`TestPathMigrationDidNotWidenTolerance` and `TestPathMigrationRejectsASameLeafNewcomer`
also failed as an expected cascade — the mutated 279-entry snapshot trips
`pathMigrationToleratedPaths`'s entry-count anti-vacuity guard.)

`git status --porcelain` before restore:
```
 M .aether/docs/orphan-allowlist-policy.md
 M .github/workflows/ci.yml
 M cmd/ci_wiring_gate_test.go
 M cmd/subcommand_reachability_ratchet_test.go
 M cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json
```

Restored the frozen snapshot byte-for-byte (SHA-256 re-verified to match
`873cad5a...45e0e`), re-ran the full 34-name filter:
```
ok  	github.com/calcosmic/Aether/cmd	6.988s
```

`git status --porcelain` after restore:
```
 M .aether/docs/orphan-allowlist-policy.md
 M .github/workflows/ci.yml
 M cmd/ci_wiring_gate_test.go
 M cmd/subcommand_reachability_ratchet_test.go
```
(only this plan's four intentional files — the testdata mutation is gone)

## Recomputed SHA-256

`873cad5a20e472af7050e69042be9faf8527a2c3e1d34907762b37e6a5845e0e` — matches the
plan's pinned value and the `preMigrationSnapshotSHA256` constant exactly.

## Must-not-regress re-proof (ten mutations, rounds 1-3)

Each mutation applied to `.github/workflows/ci.yml` (or the live allowlist), confirmed
to make a named test FAIL, then reverted with a path-scoped edit; `git status
--porcelain` confirmed clean of unintended files after each and at the end.

| # | Mutation | Caught by |
|---|---|---|
| 1 | `; true` appended to the blanket gate run line | `TestWiringGateStepRunsEveryWiringTest` — "run line must be exactly ... found: go test ./... -count=1 -timeout 900s; true" |
| 2 | `\|\| exit 0` appended | `TestWiringGateStepRunsEveryWiringTest` — same message, `... \|\| exit 0` |
| 3 | `\| cat` appended | `TestWiringGateStepRunsEveryWiringTest` — same message, `... \| cat` |
| 4 | second `-run` filter appended | `TestWiringGateStepRunsEveryWiringTest` — same message, `... -run 'TestNothingMatchesThisAtAll'` |
| 5 | `if ! CMD; then true; fi` wrapping | `TestWiringGateStepRunsEveryWiringTest` — "found: if ! go test ./... -count=1 -timeout 900s; then true; fi" |
| 6 | `(CMD) ; echo done` wrapping | `TestWiringGateStepRunsEveryWiringTest` — "found: (go test ./... -count=1 -timeout 900s) ; echo done" |
| 7 | commented-out gate step | `TestWiringGateStepRunsEveryWiringTest` — "has no step named \"Run Go tests\" — the blanket release gate has been removed or renamed" |
| 8 | step-level `continue-on-error: true` | `TestWiringGateStepRunsEveryWiringTest` — "has continue-on-error set — a step whose failure does not fail the job is not a gate" |
| 9 | removed `pull_request:` trigger | `TestReleaseGateWorkflowActuallyRuns` ("the check that is supposed to run on every change would no longer run") AND `TestReleaseGateWorkflowShapeIsWhitelisted` ("missing the required trigger \"pull_request\"") |
| 10 | removed `aether colonize` / `aether closeout` from the live allowlist | `TestNoRegisteredSubcommandIsUnreferenced` — "2 registered subcommand(s) have no caller ... aether closeout is registered but nothing calls it ... aether colonize is registered but nothing calls it" |

## Anti-vacuity proofs (Task 1)

- Changed `pathMigrationExpandedPathCount` from 284 to 285 →
  `TestPathMigrationDidNotWidenTolerance` FAILED: "tolerated path set has 284 entries,
  want exactly 285 (pathMigrationExpandedPathCount) — the expansion map and the frozen
  snapshot have drifted out of sync." Restored → PASSED.
- Added a 7th key `"zzz-not-in-snapshot"` to `pathMigrationExpansion` →
  `TestPathMigrationDidNotWidenTolerance` FAILED: "pathMigrationExpansion key
  \"zzz-not-in-snapshot\" is absent from the frozen pre-migration snapshot — a dead
  expansion entry would tolerate paths nothing in the frozen snapshot ever earned."
  Restored → PASSED, file byte-identical to a pre-mutation backup.

## Full named CI filter

`go test ./cmd -run '<34-name alternation from .github/workflows/ci.yml line 100>'
-count=1 -timeout 900s -v` — 34 top-level `Test...` names ran (counted via `=== RUN`
lines), exit 0, confirmed twice (after Task 2's addition and again after Task 3's).

## Full test suite

`go vet ./...` and `go build ./...` — both clean, no output.

`go test ./... -count=1 -timeout 900s` — **all packages PASSED, exit 0**, including
`pkg/codex` (25.740s) — the flake documented in
`.planning/phases/172-wiring-proof/deferred-items.md` (172-07) did not occur on this
run. `cmd` package: 285.231s, PASS.

## Which named test now makes the policy document's headline claim true

`.aether/docs/orphan-allowlist-policy.md`'s headline claim ("Both lists can only get
shorter... the automated check that runs on every change fails immediately, by name")
is made true across the 172-09 key-format migration by `TestPathMigrationDidNotWidenTolerance`
(comparing full paths, not last words, per this plan's Task 1 rewrite), backed by
`TestPreMigrationSnapshotIsFrozen` (Task 3 — the anchor that comparison rests on cannot
be silently edited) and proven by construction for every one of the 278 tolerated last
words by `TestPathMigrationRejectsASameLeafNewcomer` (Task 2).

## Deviations from Plan

None — plan executed exactly as written. All three tasks' acceptance criteria were
met without needing to adjust any measured constant (the STOP rule's re-measurement
check found zero discrepancies).

## Known Stubs

None. This plan only touches test guard code, a policy document, and CI wiring — no
UI or data-rendering surface.

## Threat Flags

None beyond what this plan's own `<threat_model>` already documents and mitigates
(T-172-65 through T-172-69). No new network endpoint, auth path, file access pattern,
or schema change was introduced.

## Self-Check: PASSED

- FOUND: cmd/subcommand_reachability_ratchet_test.go
- FOUND: cmd/ci_wiring_gate_test.go
- FOUND: .aether/docs/orphan-allowlist-policy.md
- FOUND: .github/workflows/ci.yml
- FOUND: f73fa7c7 (Task 1 commit)
- FOUND: 9077694f (Task 2 commit)
- FOUND: 6de0b159 (Task 3 commit)
