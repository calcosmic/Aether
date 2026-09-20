---
phase: 165-core-lifecycle-commands
plan: 09
subsystem: shelf-backlog / init lifecycle
tags: [shelf, init, init-ceremony, recovery-docs, gap-closure]
requires: []
provides:
  - "aether init --promote-shelf/--dismiss-shelf (atomic shelf promotion inside the init transaction)"
  - "applyInitShelfSelections + splitShelfIDs helpers"
  - "init-ceremony shelf-todo seeding (createCeremonyColony)"
  - "CONTEXT.md / HANDOFF.md render merged session.ActiveTodos"
  - "shelf-promote-batch / shelf-dismiss-batch total-failure signalling"
affects:
  - cmd/init_cmd.go
  - cmd/init_ceremony.go
  - cmd/shelf_init.go
  - cmd/recovery_snapshot.go
tech-stack:
  added: []
  patterns:
    - "Mutate state only after the transaction's point of no return (state save), before the response is built"
    - "Normalize both sides of a string equality comparison (TrimSpace) rather than only one"
    - "Prefer a merged/authoritative field, fall back to derivation only when empty"
key-files:
  created: []
  modified:
    - cmd/init_cmd.go
    - cmd/shelf_init.go
    - cmd/init_ceremony.go
    - cmd/recovery_snapshot.go
    - cmd/shelf_todo_wiring_test.go
    - cmd/shelf_test.go
    - cmd/testdata/command_catalog.json
decisions:
  - "Moved shelf-entry mutation into the aether init runtime transaction (review's preferred fix) rather than leaving it in the wrapper before consent, per plan 165-09's scope"
  - "Did not add a status guard to promoteShelfEntry/dismissShelfEntry (re-targeting an already-promoted entry) -- explicitly out of scope per the plan's action notes"
metrics:
  duration: "~70 minutes"
  completed: "2026-08-03"
---

# Phase 165 Plan 09: Atomic Shelf Promotion + Recovery Doc Wiring Summary

Moved shelf-entry promotion out of the wrapper (where it ran before user
consent) and into the `aether init` runtime transaction itself, closing the
Phase 165 Truth 8 blocker (review CR-01) plus four related warnings (WR-01
through WR-05) in the same three files.

## What Changed

**Task 1 — Atomic promotion inside `aether init` (CR-01, WR-01, WR-04):**
`aether init` gained `--promote-shelf` / `--dismiss-shelf` flags. The new
`applyInitShelfSelections` helper (in `cmd/shelf_init.go`) is called
immediately after `store.SaveJSON("COLONY_STATE.json", state)` succeeds and
before `session.json` is built — every refusal branch in `init` (empty goal,
active colony, sealed colony without `--confirm-reinit`, in-progress seal,
invalid JSON, failed state save) returns before that line, so a failed
`aether init` now writes nothing to `shelf.json`. The promotion goal and the
colony goal are literally the same `goal` variable, so a revised goal can
never produce a stranded/mismatched entry. A bad shelf ID is collected into
`shelf_failed` and reported via a stderr warning, but never blocks colony
creation. `promoteShelfEntry` now trims `PromotedTo` with `strings.TrimSpace`
to match the existing query-side trim in `promotedShelfTodos` (WR-01). The
three tests in `cmd/shelf_todo_wiring_test.go` with the leaking
`os.Setenv("AETHER_ROOT", ...)` / broken deferred-restore pattern were fixed
to use `t.Setenv` (WR-04) — the old pattern evaluated its deferred argument at
`defer` time, "restoring" a temp dir Go later deletes and leaking a dangling
env var into the rest of the package's test run.

**Task 2 — Both colony-creation paths, both recovery documents (WR-02,
WR-03):** `createCeremonyColony` (the Codex-facing `init-ceremony` path) now
seeds `ActiveTodos: promotedShelfTodos(store, goal)` instead of a hard-coded
empty slice, so a shelf entry promoted earlier under this goal surfaces here
too, not only on the `aether init` path. `renderContextSnapshot` and
`renderHandoffSnapshot` in `cmd/recovery_snapshot.go` now prefer the merged
`session.ActiveTodos` list (which already carries shelf-seeded todos via the
prior `mergeShelfTodos` wiring) and fall back to
`sessionActiveTodosFromState(state)` only when that list is empty — the same
precedence `cmd/context.go`'s resume path already uses. A promoted shelf idea
now appears in `CONTEXT.md` and `HANDOFF.md`, not only in `session.json`.

**Task 3 — Total batch failure fails visibly (WR-05):** `shelf-promote-batch`
and `shelf-dismiss-batch` now return `outputError` (naming the failed IDs)
when every requested ID fails, instead of an `ok:true` envelope with an empty
result. A partial failure (at least one ID succeeded) still returns
`ok:true`, with a new `failed_count` key alongside the existing
`promoted`/`dismissed`/`failed`/`count` keys so a caller has a positive signal
to branch on.

## Verification

- `go build ./cmd/aether` exits 0; `go vet ./...` clean
- `go test ./cmd/... ./pkg/...` passes in full (269.9s for `cmd`, all `pkg/*` green)
- `aether init --help` lists both `--promote-shelf` and `--dismiss-shelf`
- `applyInitShelfSelections(store` (line 214) sits below
  `store.SaveJSON("COLONY_STATE.json", state)` (line 203) and above
  `ActiveTodos:` (line 228) in `cmd/init_cmd.go`
- `grep -c 'strings.TrimSpace(colonyGoal)' cmd/shelf_init.go` → 2 (query side + new write side)
- `grep -rn 'defer os.Setenv("AETHER_ROOT"' cmd/shelf_todo_wiring_test.go` → no matches; `t.Setenv("AETHER_ROOT"` → 7 occurrences
- `grep -rn 'ActiveTodos: \[\]string{}' cmd/init_ceremony.go` → no matches
- `grep -c 'session.ActiveTodos' cmd/recovery_snapshot.go` → 3
- `grep -c 'failed_count' cmd/shelf_init.go` → 2; `grep -c 'outputError' cmd/shelf_init.go` → 7
- Regenerated `cmd/testdata/command_catalog.json` golden fixture for the two new `init` flags (`TestAuditCatalogGolden` passes)
- One transient failure was observed and diagnosed as a test artifact, not a regression: `TestCLICompiledInstallToSealJourney` failed once during a background run because it detected `cmd/shelf_init.go` as a dirty working-tree file — caused by this executor's own concurrent commit-splitting edits mid-run, not by the code under test. Re-ran in isolation against a clean working tree and it passed.

## Deviations from Plan

None — plan executed exactly as written across all three tasks. The
TDD RED/GREEN gate was followed per task: each task's tests were written and
confirmed failing before the corresponding implementation, then confirmed
passing after.

## TDD Gate Compliance

All three tasks (`tdd="true"`) followed RED → GREEN:
- Task 1: `6f067044` (test, RED confirmed) → `5de17a58` (feat, GREEN confirmed)
- Task 2: `a4bd8f99` (test, RED confirmed) → `28522270` (feat, GREEN confirmed)
- Task 3: `9a9d4885` (test, RED confirmed) → `e8b69cd6` (feat, GREEN confirmed)

No REFACTOR commits were needed.

## Self-Check: PASSED

- `cmd/init_cmd.go` contains `applyInitShelfSelections(store` — FOUND
- `cmd/shelf_init.go` contains `func applyInitShelfSelections` and `func splitShelfIDs` — FOUND
- `cmd/init_ceremony.go` contains `promotedShelfTodos(store, goal)` — FOUND
- `cmd/recovery_snapshot.go` contains `session.ActiveTodos` at 3 sites — FOUND
- Commits `6f067044`, `5de17a58`, `a4bd8f99`, `28522270`, `9a9d4885`, `e8b69cd6` all present in `git log` — FOUND
