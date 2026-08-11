---
phase: 172-wiring-proof
plan: 01
subsystem: cli
tags: [cobra, spawn-tree, cli-wiring, go]

# Dependency graph
requires: []
provides:
  - "spawn-can-spawn accepts the exact positional invocation .aether/workers.md:292 documents (aether spawn-can-spawn {your_depth} --enforce)"
  - "spawn-can-spawn still accepts the flag-only playbook form (aether spawn-can-spawn --depth {depth})"
  - "--enforce with real deny-to-non-zero-exit semantics, wired through outputError/markRenderedCommandError"
  - "spawnCanSpawnDecision: a substitutable allow/deny seam Phase 173 (SPAWN-01) will replace with a real cap decision"
affects: [173-delegation-guard]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Package-level decision-function variable (spawnCanSpawnDecision) as a test seam for a not-yet-real business decision, matching this repo's existing outputError/markRenderedCommandError non-zero-exit convention"

key-files:
  created: [cmd/spawn_enforce_test.go]
  modified: [cmd/spawn.go, cmd/testdata/command_catalog.json]

key-decisions:
  - "D-13 honoured: --enforce carries real semantics now (deny -> non-zero exit via outputError), not an inert accepted-but-ignored flag"
  - "D-14 honoured: fixed cmd/spawn.go (Args: cobra.MaximumNArgs(1)), left .aether/workers.md:292 untouched"
  - "Positional depth wins over --depth when both are present, matching the plan's acceptance criterion for spawn-can-spawn 5 --depth 9"
  - "Deny decision extracted into a package-level func var (spawnCanSpawnDecision) rather than an inline literal, so a test can substitute a deny answer without waiting for Phase 173's real cap logic"

patterns-established:
  - "Substitutable decision-seam func var for machinery that must be reachable by tests before its real logic exists"

requirements-completed: [WIRE-02]

# Metrics
duration: 12min
completed: 2026-08-11
---

# Phase 172 Plan 01: Wire spawn-can-spawn to the documented invocation Summary

**`aether spawn-can-spawn 5 --enforce` — the exact string `.aether/workers.md:292` instructs every worker to run — now resolves and exits 0 instead of erroring twice over (unknown positional, unknown flag), with `--enforce`'s deny path wired through `outputError`/`markRenderedCommandError` and driven by a real test.**

## Performance

- **Duration:** ~35 min (including full-suite regression investigation)
- **Started:** 2026-08-11T16:42:45+02:00 (Task 1 commit)
- **Completed:** 2026-08-11T17:01:33+02:00 (deviation-fix commit)
- **Tasks:** 2 (plus 1 auto-fixed deviation)
- **Files modified:** 3 (2 modified, 1 created)

## Accomplishments
- `spawnCanSpawnCmd.Args` changed from `cobra.NoArgs` to `cobra.MaximumNArgs(1)`, so the documented positional depth (`{your_depth}`) resolves alongside the existing flag-only playbook form (`--depth {depth}`)
- `--enforce` registered with real semantics: when the (currently always-allow) decision denies and `--enforce` is set, the command calls `outputError(1, ...)` → `markRenderedCommandError` → non-zero process exit — the repo's one sanctioned non-zero-exit path, not `os.Exit`
- The allow/deny answer extracted into `spawnCanSpawnDecision`, a package-level function variable Phase 173 (SPAWN-01) will replace with a real depth-cap decision; today it always returns `(true, "")`
- Two new tests prove both halves of D-13/D-14: the manual's own invocation string resolves and executes, and the deny branch is reachable and exits non-zero only when `--enforce` is present

## Task Commits

Each task was committed atomically:

1. **Task 1: Accept the documented invocation and register --enforce with real deny semantics** - `ad57e8b3` (feat)
2. **Task 2: Assert the documented invocation and the deny-to-non-zero-exit wiring** - `6bcb43f7` (test)
3. **Deviation fix: regenerate command catalog golden after --enforce registration** - `91b80579` (fix, Rule 1)

**Plan metadata:** (this commit) - `docs(172-01): complete wire spawn-can-spawn plan`

## Files Created/Modified
- `cmd/spawn.go` - `spawnCanSpawnCmd.Args` relaxed to `cobra.MaximumNArgs(1)`; `--enforce` bool flag registered; `RunE` resolves depth from positional-or-flag (positional wins), calls the new `spawnCanSpawnDecision` seam, and denies via `outputError` when `enforce && !canSpawn`; unparseable positional depth now fails loudly via `outputError` naming the offending value instead of silently defaulting to 0
- `cmd/spawn_enforce_test.go` - `TestSpawnCanSpawnAcceptsDocumentedInvocation` reads `.aether/workers.md` at runtime, extracts the documented invocation line, resolves it through `rootCmd.Find`/the command's own `Args` validator, then executes it in-process and asserts `can_spawn:true, depth:5, exit:0`; `TestSpawnCanSpawnEnforceDeniesWithNonZeroExit` substitutes a deny answer via `spawnCanSpawnDecision` and asserts both the `--enforce` non-zero-exit case (depth named in the error) and the without-`--enforce` zero-exit report case (deny visible in the JSON payload)
- `cmd/testdata/command_catalog.json` - regenerated golden snapshot for `spawn-can-spawn`'s `flags` array (`["depth"]` -> `["depth", "enforce"]`), via `go test ./cmd -run TestAuditCatalogGolden -update-golden`

## Decisions Made
- Positional depth wins over `--depth` when both are present in the same invocation (plan's explicit acceptance criterion, e.g. `spawn-can-spawn 5 --depth 9` reports depth 5)
- Kept the success payload shape unchanged (`can_spawn`, `depth` only) rather than adding a `reason` field, per the plan's "keep the success payload shape unchanged" instruction — the deny reason is only surfaced in the `--enforce` error message
- `spawnCanSpawnDecision` implemented as a package-level function *variable* (not a plain function) specifically so `TestSpawnCanSpawnEnforceDeniesWithNonZeroExit` can substitute a deny answer for the duration of one test case via `defer`, matching the plan's explicit ask ("extract the decision into a small package-level function... structure the deny decision so a test can drive it")

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Regenerated command catalog golden after --enforce registration**
- **Found during:** post-Task-2 broader verification (`go test ./cmd/... -count=1 -timeout 900s`, the plan's own verification item 5)
- **Issue:** `cmd/audit_catalog_test.go`'s `TestAuditCatalogGolden` compares the live cobra command tree (name, short description, and **flags**) against a checked-in snapshot at `cmd/testdata/command_catalog.json`. Task 1 registering `--enforce` on `spawn-can-spawn` correctly changed the live catalog but the plan did not call out updating this golden file, so the audit failed deterministically (`got 244436 bytes, want 244419 bytes`). Reproduced in isolation with `go test ./cmd -run TestAuditCatalogGolden -count=1 -v` before touching anything else, to separate this real, deterministic failure from unrelated noise the shared build machine was producing from other concurrently-running worktree agents' `go test ./cmd/...` processes (observed up to 5 identical processes running at once).
- **Fix:** `go test ./cmd -run TestAuditCatalogGolden -count=1 -update-golden`, which regenerated `cmd/testdata/command_catalog.json` — a 1-line diff adding `"enforce"` to `spawn-can-spawn`'s `flags` array.
- **Files modified:** `cmd/testdata/command_catalog.json`
- **Verification:** `go test ./cmd -run 'TestAuditCatalog|TestCatalog|TestSpawnCanSpawn|TestCLIFlagAudit|TestCommandCallsMatchCobraContracts' -count=1 -v` — all pass; `python3 scripts/verify_catalog_classified.py --strict` — all checks passed.
- **Committed in:** `91b80579`

---

**Total deviations:** 1 auto-fixed (1 bug — a golden-file test directly downstream of Task 1's own change)
**Impact on plan:** Necessary for correctness; the golden file is the checked-in proof that the command catalog matches the real binary, and Task 1's own change was what moved it. No scope creep — no other file was touched to fix this.

## Issues Encountered
- Full-package regression runs (`go test ./cmd/... -count=1 -timeout 900s`, ~5 minutes each) were contended by multiple other worktree agents' concurrently-running identical test suites on this shared build machine (confirmed via `ps aux` showing up to 5 simultaneous `go test ./cmd/...` processes), which made results slow and, on at least one run, ambiguous about which test had failed. Isolating and re-running the specific failing test (`TestAuditCatalogGolden`) directly, outside the full-suite noise, gave a clean, reproducible signal and let the real regression (above) be found and fixed with confidence, separate from the machine's shared-resource flakiness.
- The stash/unstash red-then-green verification in Task 2's acceptance criteria used a manual temporary edit + `git checkout --` instead of `git stash` (`cmd/spawn.go` was already committed clean from Task 1 at that point in the sequence, so there was nothing to stash). The observable red-then-green transcript is identical; recorded below.

### Red-then-green transcript (Task 2 acceptance criterion)

**RED** — with the `--enforce` flag registration removed from `cmd/spawn.go`:
```
=== RUN   TestSpawnCanSpawnAcceptsDocumentedInvocation
    spawn_enforce_test.go:101: documented invocation uses --enforce, which is not registered on spawn-can-spawn
--- FAIL: TestSpawnCanSpawnAcceptsDocumentedInvocation (0.00s)
FAIL
```

**GREEN** — after restoring `cmd/spawn.go` (`git checkout -- cmd/spawn.go`):
```
=== RUN   TestSpawnCanSpawnAcceptsDocumentedInvocation
--- PASS: TestSpawnCanSpawnAcceptsDocumentedInvocation (0.00s)
=== RUN   TestSpawnCanSpawnEnforceDeniesWithNonZeroExit
--- PASS: TestSpawnCanSpawnEnforceDeniesWithNonZeroExit (0.00s)
PASS
ok  	github.com/calcosmic/Aether/cmd	0.731s
```

### Decision seam Phase 173 will replace

`spawnCanSpawnDecision` in `cmd/spawn.go` — signature `func(depth int) (bool, string)`, currently always returning `(true, "")`. Phase 173 (SPAWN-01) implements the real depth-cap decision by replacing this function's body (or reassigning the variable); no other call site needs to change.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- WIRE-02 satisfied: the documented worker spawn-protocol command now executes as written
- Phase 173 (SPAWN-01) has a named, tested seam (`spawnCanSpawnDecision`) to implement the real allow/deny decision against — no further wiring needed on its side, only the decision logic itself
- Both existing calling conventions (positional `.aether/workers.md:292` form and flag-only playbook form) continue to work side by side

---
*Phase: 172-wiring-proof*
*Completed: 2026-08-11*

## Self-Check: PASSED

- FOUND: cmd/spawn.go
- FOUND: cmd/spawn_enforce_test.go
- FOUND: cmd/testdata/command_catalog.json
- FOUND: .planning/phases/172-wiring-proof/172-01-SUMMARY.md
- FOUND commit: ad57e8b3 (Task 1)
- FOUND commit: 6bcb43f7 (Task 2)
- FOUND commit: 91b80579 (deviation fix)
