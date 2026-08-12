---
phase: 172-wiring-proof
plan: "09"
subsystem: cmd (orphan reachability ratchet)
tags: [wiring-proof, ci-guard, path-keying, gap-closure]
dependency-graph:
  requires: ["172-08"]
  provides: ["path-keyed orphan ratchet", "GAP B closure"]
  affects: ["cmd/subcommand_reachability_ratchet_test.go", "cmd/testdata/orphan_allowlist.json", ".github/workflows/ci.yml"]
tech-stack:
  added: []
  patterns: ["rootCmd.Find-based caller-evidence resolution", "shrink-only baseline diff across a key-format migration"]
key-files:
  created:
    - cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json
  modified:
    - cmd/subcommand_reachability_ratchet_test.go
    - cmd/cli_flag_audit_test.go
    - cmd/testdata/orphan_allowlist.json
    - cmd/testdata/orphan_allowlist_baseline.json
    - .github/workflows/ci.yml
    - .aether/docs/orphan-allowlist-policy.md
decisions:
  - "Keyed caller evidence and orphan reporting by the resolved cobra CommandPath() instead of the bare leaf name, closing GAP B/CR-04"
  - "Migrated the allowlist/baseline to path keys via a single, deliberate, on-the-record baseline replacement (D-11), with a frozen pre-migration snapshot as the permanent shrink-only reference"
  - "Recorded (not repaired) 9 newly revealed orphans with reason path-collision-revealed / owner_phase RECLAIM, per D-07"
  - "Did not modify .planning/ROADMAP.md — worktree mode reserves that file for the orchestrator; the intended edit is recorded below"
metrics:
  duration: "~2.5 hours"
  completed: 2026-08-12
---

# Phase 172 Plan 09: Path-Keyed Caller Evidence (GAP B Closure) Summary

Closes verification GAP B (172-VERIFICATION.md) / CR-04 (172-REVIEW.md): the orphan
reachability ratchet keyed caller evidence and orphan reporting by bare leaf command
name with no parent path, so a documented call to `aether host colonize` silently
credited the unrelated top-level `aether colonize` as wired, and the same held for
`aether ceremony closeout` vs `aether closeout`. Twenty leaf names exist at more than
one registered path in the live cobra tree, so this hole was latent across the whole
command surface, not just these two commands.

## What changed

**Task 1** — `credit()` (inside `singleFileCallerNames`), `collectGoSelfInvocationCallers`,
and `computeOrphanNames` now resolve every documented invocation through
`rootCmd.Find` and key on the resolved `CommandPath()` (e.g. `"aether host colonize"`)
instead of the bare leaf token. Added the permanent, hermetic
`TestCallerEvidenceIsNotSharedBetweenSameLeafNames`, built on a synthetic
`ratchet-selftest-collide` / `ratchet-selftest-parent` fixture registered and torn down
inside the test body — deliberately NOT pinned to `colonize`/`closeout`, so it stays
meaningful after those two are eventually repaired.

**Task 2** — Moved every remaining bare-name consumer to paths:
`collectYAMLRuntimeCommandNames` renamed to `collectYAMLRuntimeCommandPaths` (resolves
through `rootCmd.Find`); the D-08 skill-lifecycle reason-tagging map is now resolved at
runtime via `resolveSkillLifecyclePaths` rather than hand-written;
`collectSubstitutionCallerNames`, the token-boundary self-check, and the
`TestRatchetDetectsASyntheticOrphan` / `TestDeletingACallerMakesTheRatchetNameIt` /
`TestCallerEvidenceCreditsCommandSubstitution` self-tests all reason in paths now.
Also closed WR-01: `TestWiringGuardsHaveNoRuntimeEscapeHatch` now rejects
`flag.Bool`/`flag.String`/`flag.Int` as a runtime escape hatch, with exactly one
reviewed exemption (the `-update-orphan-allowlist` flag's own declaration line),
asserted to be exactly 1 so a second flag cannot hide behind it and the exemption
cannot be silently deleted without the guard noticing.

**Task 3** — Migrated the allowlist and baseline to path keys without widening
tolerance. Froze the pre-migration baseline byte-for-byte into
`orphan_allowlist_baseline_pre_path_migration.json` (278 entries, never edited again),
regenerated the live/baseline allowlist against the path-keyed scanner (293 entries),
added `TestOrphanAllowlistIsPathKeyed` and `TestPathMigrationDidNotWidenTolerance` to
keep D-10's "shrink-only" property true across the key-format change, added the new
snapshot file to `guardedAllowlistFiles`, added the three new test names to the named
CI wiring step, and rewrote `.aether/docs/orphan-allowlist-policy.md`.

## RED transcripts (Task 1, before/after)

**Before the edit** — the ratchet PASSES and believes both `colonize` and `closeout`
have callers (zero occurrences of "colonize" in the output, and the allowlist
membership probe confirms neither is even in the tolerated-orphan list):

```
=== RUN   TestNoRegisteredSubcommandIsUnreferenced
    subcommand_reachability_ratchet_test.go:908: enumerated 405 registered commands, found 278 orphans
--- PASS: TestNoRegisteredSubcommandIsUnreferenced (0.34s)
PASS
ok  	github.com/calcosmic/Aether/cmd	1.049s
```
```
$ python3 -c "import json;n={e['name'] for e in json.load(open('cmd/testdata/orphan_allowlist.json'))};print('colonize' in n, 'closeout' in n)"
False False
```

**After the edit (Task 1 alone, diagnostic re-run)** — with the non-vacuity YAML gate
(Task 2's job) temporarily downgraded from `t.Fatal` to `t.Log` purely to capture this
transcript before Task 2 landed, then immediately reverted (verified with
`git status --porcelain` / `diff` against the intended end state — clean both times),
the path-keyed scanner correctly names both commands:

```
=== RUN   TestNoRegisteredSubcommandIsUnreferenced
    ...
    subcommand_reachability_ratchet_test.go:989: 293 registered subcommand(s) have no caller and are not in testdata/orphan_allowlist.json:
          ...
          aether closeout is registered but nothing calls it (searched: wrappers, menu specs, hooks, scripts)
          ...
          aether colonize is registered but nothing calls it (searched: wrappers, menu specs, hooks, scripts)
          ...
--- FAIL: TestNoRegisteredSubcommandIsUnreferenced (0.33s)
FAIL
```
`grep -c 'aether colonize is registered'` and `grep -c 'aether closeout is registered'`
both returned 1.

Note on sequencing: taken strictly, Task 1 alone (without Task 2's rename of
`collectYAMLRuntimeCommandNames`) makes the test fail one gate earlier — on the
non-vacuity YAML-evidence assertion, which itself expects paths after Task 1's change
but still received bare names until Task 2 renamed the function — rather than on the
allowlist-membership fragment the plan predicted. The diagnostic run above (with that
gate temporarily downgraded to `t.Log`) proves the deeper, structural claim: the
path-keyed evidence and orphan computation correctly names both commands the moment
they are reached. Task 2 makes that reachable via the normal test run; by the end of
Task 2, `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced` fails only on
allowlist membership, confirmed by reading its output (no YAML non-vacuity error, no
D-08 `t.Fatalf`).

## Mutation / red proofs performed and reverted

All performed via temporary in-place edits, verified to fail as expected, then
reverted and diffed clean against the intended end-of-task state before the task's
commit:

1. **Task 1** — reverting `credit()`'s body to the old bare-name-plus-following-token
   scheme made `TestCallerEvidenceIsNotSharedBetweenSameLeafNames` FAIL (did not credit
   the full child path; reported the child as a false orphan). Restored, PASS confirmed.
2. **Task 2** — inserting `var somethingElse = flag.Bool("skip-wiring", false, "")`
   made `TestWiringGuardsHaveNoRuntimeEscapeHatch` FAIL naming that exact line.
   Restored, PASS confirmed.
3. **Task 2** — renaming the `updateOrphanAllowlist` flag identifier (compile-safe
   rename, both declaration and use site) made the same test FAIL with
   `"expected exactly 1 line exempted ... found 0"`. Restored, PASS confirmed.
4. **Task 3** — appending a fabricated `aether not-a-real-command` entry to the
   baseline made `TestPathMigrationDidNotWidenTolerance` FAIL naming it. Restored,
   PASS confirmed.
5. **Task 3** — rewriting one baseline entry's name to its bare leaf made
   `TestOrphanAllowlistIsPathKeyed` FAIL naming it. Restored, PASS confirmed.

## Newly revealed orphans (Task 3, D-07 disposition: recorded, not repaired)

Regenerating the allowlist against the path-keyed scanner raised the entry count from
278 (pre-migration, leaf-keyed) to 293 (post-migration, path-keyed): 274 carried
forward one-to-one (just renamed to their full path, including the 6 unchanged
skill-lifecycle entries tagged `owner_phase: "178"`), 10 produced by splitting 4
collapsed leaf entries into the real commands they covered (see the correction under
"Pre- and post-migration counts"), and 9 genuinely new discoveries — tagged
`reason: "path-collision-revealed"`, `owner_phase: "RECLAIM"` — each previously
vouched for by an unrelated, same-leaf-name sibling that does have a real caller:

| Newly revealed path | Registered at | Previously (wrongly) credited by |
|---|---|---|
| `aether colonize` | `cmd/codex_workflow_cmds.go:30` | a documented call to `aether host colonize` |
| `aether closeout` | `cmd/ceremony_cmd.go:118` | a documented call to `aether ceremony closeout` |
| `aether host` (bare parent) | `cmd/host_cmd.go:16` | ANY `aether host <subcommand>` call (old scheme credited the parent token bare, regardless of which subcommand followed) |
| `aether host build` | `cmd/host_cmd.go:56` | a documented call to top-level `aether build` (`cmd/codex_workflow_cmds.go:106`) |
| `aether host oracle` | `cmd/host_cmd.go:80` | a documented call to top-level `aether oracle` (`cmd/compatibility_cmds.go:55`) |
| `aether host swarm` | `cmd/host_cmd.go:96` | a documented call to top-level `aether swarm` (`cmd/swarm_cmd.go:103`) |
| `aether host watch` | `cmd/host_cmd.go:88` | a documented call to top-level `aether watch` (`cmd/compatibility_cmds.go:31`) |
| `aether export pheromones` | `cmd/exchange.go:46` | a documented call to `aether import pheromones` (shared leaf `pheromones`) |
| `aether import pheromones` | `cmd/exchange.go:344` | a documented call to `aether export pheromones` (mirror image) |

Confirmed: `/ant-export-signals` and `/ant-import-signals` actually invoke the separate
flat commands `aether export-signals` / `aether import-signals`, not the nested
`export pheromones` / `import pheromones` subcommands — so neither of the last two is
called directly today. This plan does not repair any of the 9; that is separate,
future work, tracked by their allowlist entries.

## Pre- and post-migration counts

| | Count |
|---|---|
| Pre-migration (leaf-keyed) baseline, frozen forever in `orphan_allowlist_baseline_pre_path_migration.json` | 278 |
| — of which unreviewed-pre-existing (`owner_phase: "RECLAIM"`) | 272 |
| — of which skill-lifecycle (`owner_phase: "178"`) | 6 |
| Post-migration (path-keyed) live allowlist / baseline | 293 |
| — carried forward one-to-one (renamed to full path) | 274 |
| — produced by splitting 4 collapsed leaf entries into their real paths | 10 |
| — newly revealed (`path-collision-revealed`) | 9 |

Reconciliation: 274 + 10 + 9 = 293. The 6 skill-lifecycle entries are a subset of the
274 carried one-to-one, not a separate addend.

### Correction — the split entries (orchestrator note, post-merge)

The table above originally read "carried forward unchanged | 278" alongside a separate
"skill-lifecycle | 6" row, which double-counted the 6 and left the +15 change
unreconciled (278 + 9 = 287, not 293). The missing 6 are **split entries**: under the
old leaf keying, one allowlist entry silently covered every command sharing that leaf.
Path keying correctly expands 4 such entries into the 10 real commands they were
standing in for:

| Pre-migration leaf entry | Post-migration path entries |
|---|---|
| `get` | `aether colony-depth get`, `aether parallel-mode get`, `aether plan-granularity get` |
| `set` | `aether colony-depth set`, `aether parallel-mode set`, `aether plan-granularity set` |
| `registry` | `aether export registry`, `aether import registry` |
| `wisdom` | `aether export wisdom`, `aether import wisdom` |

These 6 extra entries are legitimate — each names a real registered command that was
already tolerated, just tolerated invisibly under a shared leaf. No tolerance was
widened: 0 pre-migration leaves lost their entry, and every post-migration entry's leaf
appears in the frozen snapshot or in the reviewed `pathCollisionRevealedOrphans` set.

This is the exact blind spot the plan-checker flagged as non-blocking warning 1 before
execution: `TestPathMigrationDidNotWidenTolerance` compares by **leaf**, so a legitimate
one-to-many split is accepted as "carried forward" without being classified as newly
revealed. The warning predicted it was unlikely to manifest; it manifested, on 4 leaves.
The guard is not defeated — the split is visible in the committed baseline's diff, which
is D-11's stated human backstop, and that is how it was caught here. But the automated
classification remains coarser than the entry counts imply, and a future tightening
should compare full paths where the pre-migration path is determinable.

`python3` verification (pasted output):
```
293 293 True                 # len(live) len(baseline) {names equal}
True                          # all(name.startswith("aether ") for name in live)
True {'name': 'aether colonize', 'reason': 'path-collision-revealed', 'owner_phase': 'RECLAIM'}
True {'name': 'aether closeout', 'reason': 'path-collision-revealed', 'owner_phase': 'RECLAIM'}
6                              # count of owner_phase == "178"
```

## Baseline replacement — deliberate and reviewable

This plan makes exactly one deliberate, on-the-record baseline edit, as D-11 permits:
`cmd/testdata/orphan_allowlist_baseline.json` was replaced to be byte-identical to the
regenerated `orphan_allowlist.json` (293 path-keyed entries), because the key format
changed from bare leaf name to full command path — not because tolerance was widened.
The replacement is reviewable forever against
`cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json`, the frozen,
never-edited-again copy of the baseline exactly as it stood immediately before this
migration (278 entries, bare leaf names). `TestPathMigrationDidNotWidenTolerance`
enforces that every current baseline entry's leaf name either appears in that frozen
snapshot or is one of the 9 explicitly reviewed `pathCollisionRevealedOrphans` — so
"the allowlist may only shrink" (D-10) stays true across the key-format change, not
just across ordinary future edits.

## Verification

- `go test ./cmd -run '<the full named CI alternation, .github/workflows/ci.yml:100>' -count=1 -timeout 900s -v` — all PASS.
- `go test ./... -count=1 -timeout 900s` — all packages green (`cmd` package: 284s, all PASS; no flake observed in `pkg/codex` this run).
- `go vet ./...` — clean.
- `go build ./...` — clean.

## Deviations from Plan

### Auto-fixed issues

**1. [Rule 1 - Bug] `TestCallerEvidenceCreditsCommandSubstitution` half B originally asserted strict path equality against the (still bare-name-keyed, pre-Task-3) allowlist entry**
- **Found during:** Task 2
- **Issue:** An early draft of half B's fixture asserted `resolvedPath == allowlistEntryName` before the allowlist was migrated to paths, which would have permanently broken the test's ability to run meaningfully both before and after Task 3.
- **Fix:** Resolve the allowlist entry (bare or full-path) through `rootCmd.Find`, build the synthetic invocation from the resolved path, and assert evidence against that resolved path — works correctly regardless of the allowlist's key format.
- **Files modified:** cmd/subcommand_reachability_ratchet_test.go
- **Commit:** e1aa5547

### Plan constraint honored (not a deviation, but load-bearing)

**ROADMAP.md is NOT modified by this plan's commits**, per the parallel-execution
worktree-mode instructions (this plan's own `files_modified` lists
`.planning/ROADMAP.md`, but worktree mode reserves that file for the orchestrator).
The intended edit — recorded here for the orchestrator to apply after merge — is:

> In `.planning/ROADMAP.md` § "Phase 172: Wiring Proof" success criterion 1, replace
> the phrase `278 pre-existing orphans, 6 of them tagged owner_phase: "178"` with:
>
> `293 pre-existing orphans [re-measured by 172-09 after keying caller evidence by
> resolved cobra command path instead of bare leaf name; the previous figure of 278
> was measured under leaf-name-keyed evidence, which is the exact defect GAP B/CR-04
> found — aether host colonize silently credited the unrelated top-level aether
> colonize. Of the 293: 6 are tagged owner_phase: "178" (the reviewed skill-*
> lifecycle set, unchanged in count by the migration), 9 are newly revealed by the
> path-keying migration and tagged reason: "path-collision-revealed", and the
> remaining 278 are carried forward unchanged from the pre-migration baseline. See
> .aether/docs/orphan-allowlist-policy.md and the frozen
> cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json snapshot], 6 of them
> tagged owner_phase: "178"`
>
> Also append to the same criterion (after "it may only shrink"):
> `, and this property is proven to survive the path-keying migration by
> TestPathMigrationDidNotWidenTolerance diffing against the frozen pre-migration
> snapshot. A permanent, hermetic regression test
> (TestCallerEvidenceIsNotSharedBetweenSameLeafNames) proves caller evidence for one
> command path can never leak to a same-leaf-name command at a different path, using a
> synthetic fixture that does not depend on colonize/closeout being fixed`

None else — plan executed as written otherwise.

## Known Stubs

None. This plan modifies only test/guard infrastructure and generated data files; no
UI or runtime data paths are touched.

## Threat Flags

None. All five STRIDE threats named in this plan's `<threat_model>` are mitigated as
designed (path-keyed `credit()`/`computeOrphanNames`; `TestPathMigrationDidNotWidenTolerance`
against the frozen snapshot; the in-source `pathCollisionRevealedOrphans` set;
the flag-declaration escape-hatch scan with exactly-one exemption; the `"aether "`
prefix anti-vacuity assertion). No new network endpoints, auth paths, or trust
boundaries are introduced — this plan touches only Go test/guard code and checked-in
JSON/markdown data files.

## Self-Check: PASSED

- FOUND: cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json
- FOUND: .planning/phases/172-wiring-proof/172-09-SUMMARY.md
- FOUND commit: 527488f8 (Task 1)
- FOUND commit: e1aa5547 (Task 2)
- FOUND commit: 9bd46c8e (Task 3)
