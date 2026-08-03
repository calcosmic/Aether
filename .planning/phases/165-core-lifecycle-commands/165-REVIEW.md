---
phase: 165-core-lifecycle-commands
reviewed: 2026-08-03T12:05:11Z
depth: standard
files_reviewed: 27
files_reviewed_list:
  - .aether/commands/build.yaml
  - .aether/commands/continue.yaml
  - .aether/commands/init.yaml
  - .aether/commands/plan.yaml
  - .aether/docs/wrapper-host-contract.md
  - .claude/commands/ant-build.md
  - .claude/commands/ant-continue.md
  - .claude/commands/ant-init.md
  - .claude/commands/ant-plan.md
  - .claude/commands/ant/build.md
  - .claude/commands/ant/continue.md
  - .claude/commands/ant/init.md
  - .claude/commands/ant/plan.md
  - .opencode/commands/ant/build.md
  - .opencode/commands/ant/continue.md
  - .opencode/commands/ant/init.md
  - .opencode/commands/ant/plan.md
  - cmd/build_wrapper_ceremony_test.go
  - cmd/continue_wrapper_ceremony_test.go
  - cmd/init_cmd.go
  - cmd/init_wrapper_ceremony_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - cmd/plan_wrapper_ceremony_test.go
  - cmd/recovery_snapshot.go
  - cmd/session_cmds.go
  - cmd/shelf_init.go
  - cmd/shelf_todo_wiring_test.go
findings:
  critical: 1
  warning: 6
  info: 5
  total: 12
status: issues_found
---

# Phase 165: Code Review Report (Re-review after gap-closure 165-07 / 165-08)

**Reviewed:** 2026-08-03T12:05:11Z
**Depth:** standard
**Files Reviewed:** 27
**Status:** issues_found

## Summary

This is the re-review after gap-closure plans 165-07 (shelf-promotion runtime wiring) and 165-08 (read_only contract alignment). The deepest scrutiny went to the newly changed files: `cmd/shelf_init.go`, `cmd/init_cmd.go`, `cmd/recovery_snapshot.go`, `cmd/session_cmds.go`, the init wrapper's Shelf Backlog/Approval stages, the read_only blocks in build/continue/plan wrappers, and `cmd/lifecycle_wrapper_contract_test.go` plus `cmd/shelf_todo_wiring_test.go`.

**What 165-08 fixed is verified good.** The `<read_only>` blocks in all four wrappers now agree with their Guardrails ("never reads or writes, by hand" / "never writes these files by hand"), the WR-01 fence test (`TestLifecycleWrapperReadOnlyBlocksAreConsistent`) covers all 12 surfaces including flat mirrors, and mechanical checks confirm: flat mirrors are byte-identical to canonical sources for all four verbs, and `.opencode` parity holds with exactly the one sanctioned `AskUserQuestion` delta in init.md. All Phase 165 tests pass (`go test ./cmd -run 'TestLifecycle|TestShelf|...'` — PASS). The prior review's CR-01 (hand-append of shelf todos to `active_todos`) is confirmed removed from all init wrapper surfaces and fenced by `TestInitWrapperCeremonyContract`'s forbidden list.

**What 165-07 fixed is only partially sound.** The runtime now seeds `session.ActiveTodos` from promoted shelf entries in `aether init` and preserves them across session refreshes via `mergeShelfTodos`. But tracing the full wrapper-to-runtime flow exposes a data-loss window the tests do not cover: the init wrapper promotes shelf entries *before* user approval, keyed on an exact goal string that the Approval stage can still change or abandon — a cancel, a revised goal, or a failed init strands promoted entries invisibly (new CR-01 below). Additionally, the wiring was closed on only one of two colony-creation paths, and the recovery documents never display the shelf todos the merge so carefully preserves.

## Critical Issues

### CR-01: Shelf promotion persists before Approval and is keyed on an exact goal string Approval can still change — cancel/revise/failed-init silently strands promoted backlog entries

**File:** `.claude/commands/ant/init.md:255-257` (Shelf Backlog stage; identical in `.opencode/commands/ant/init.md` and flat mirror `.claude/commands/ant-init.md`), `cmd/shelf_init.go:124-142` (`promoteShelfEntry`), `cmd/shelf_init.go:171-194` (`promotedShelfTodos`), `cmd/init_cmd.go:214`

**Issue:** The wiring only works when three fragile conditions all hold, and the wrapper's own flow can break each one:

1. **Promotion runs before consent.** The Shelf Backlog stage runs `aether shelf-promote-batch --ids ... --colony "{goal}"` and `aether shelf-dismiss-batch` *before* the Approval stage. `promoteShelfEntry` immediately flips the entry's status to `promoted` and writes `shelf.json`. If the user then chooses **cancel** at Approval, or `aether init` fails (e.g., active colony present), the entries are already gone from the shelved backlog (`loadActiveShelf` filters `Status == ShelfShelved`) and no colony ever exists to carry them as todos. This directly contradicts init.md's own `<failure_modes>` ("User Cancels At Approval ... Write nothing — no charter call, no pheromone writes, nothing persisted") and the Approval stop condition ("A cancel or a failed `aether init` both end the command with nothing persisted"). The pheromone writes were correctly gated on init success (WR-05 fix, locked by `approval_writes_pheromones_only_after_init_succeeds`); the shelf writes were not.

2. **The linkage key is an exact string match on a mutable value.** `promotedShelfTodos` selects entries where `e.PromotedTo == goal`, and `aether init` receives the goal at Approval. Approval offers "revise goal" as an explicit option; a revision after the Shelf Backlog stage means `PromotedTo` (old goal) never matches the init goal — todos are silently empty while the entries are stranded as promoted-to-a-goal-that-never-existed.

3. **The wrapper's `--colony "{goal}"` placeholder is ambiguous.** The Shelf Backlog stage says `{goal}` unqualified; the cross-stage state defines `refined_goal`, and Approval passes `"<refined goal>"` to init. An LLM executing this spec can plausibly substitute raw `$ARGUMENTS` at the Shelf Backlog stage and `refined_goal` at Approval — guaranteed mismatch, same silent loss.

There is no un-promote command in the reviewed code, so none of these failure paths are recoverable through the documented flow. `cmd/shelf_todo_wiring_test.go` never exercises the wrapper ordering — every test promotes with the identical literal goal it then inits with.

**Fix:** Make promotion atomic with init so it cannot precede consent or diverge from the final goal. Preferred: move the mutation into the runtime — have the wrapper *collect* chosen IDs at the Shelf Backlog stage but pass them to init (`aether init --promote-shelf "id1,id2" --dismiss-shelf "id3" ...`), and have `initCmd` call `promoteShelfEntry` with the actual init goal after state creation succeeds. Then `promotedShelfTodos` cannot mismatch and a cancel/failure persists nothing, matching the documented contract. Minimum viable alternative: move the `shelf-promote-batch`/`shelf-dismiss-batch` calls into the Approval stage after `aether init` returns success (mirroring the pheromone-write gating), use `refined_goal` explicitly, and add a wiring test that inits with a *different* goal than the promoted one and asserts the behavior.

## Warnings

### WR-01: Goal-string trim asymmetry between promotion write and todo read

**File:** `cmd/shelf_init.go:29-48` (batch command), `cmd/shelf_init.go:133` (`PromotedTo` assignment), `cmd/shelf_init.go:180-183` (comparison)
**Issue:** `shelf-promote-batch` stores the `--colony` flag value into `PromotedTo` verbatim (untrimmed), but `promotedShelfTodos` trims only the *query* side (`goal := strings.TrimSpace(colonyGoal)`) and compares `e.PromotedTo == goal` against the untrimmed stored value. A `--colony " Ship v2"` (leading/trailing whitespace, easy for an LLM to produce when interpolating) promotes successfully but can never match any query — including `shelf-promote-batch`'s *own* `todos` output computed three lines later, which would report the promotion succeeded while returning an empty/incomplete todo list.
**Fix:** Trim once at the write site:
```go
colonyGoal = strings.TrimSpace(colonyGoal)   // in shelfPromoteBatchCmd before the loop
// and in promoteShelfEntry:
sf.Entries[i].PromotedTo = strings.TrimSpace(colonyGoal)
```
Then both sides of the comparison are normalized. (Same normalization applies whichever fix CR-01 takes.)

### WR-02: Second colony-creation path (`aether init-ceremony`) never seeds shelf todos — the CR-01 gap is closed on only one of two init surfaces

**File:** `cmd/init_ceremony.go:553-563` (`createCeremonyColony` session literal, `ActiveTodos: []string{}`)
**Issue:** 165-07 wired `promotedShelfTodos` into `cmd/init_cmd.go:214`, but `createCeremonyColony` — the runtime-native guided init used by `aether init-ceremony` (the Codex-facing path) — writes `session.json` with a hard-coded empty `ActiveTodos`. Entries promoted with `--colony <goal>` before an init-ceremony run are stranded exactly as in the pre-fix bug: marked promoted, never surfaced as todos. The phase's own doctrine (CLAUDE.md Definition of Done) warns about capability wired on one path and silently absent on the parallel one.
**Fix:** In `createCeremonyColony`, replace the literal with the same call:
```go
ActiveTodos: promotedShelfTodos(store, goal),
```
and add an init-ceremony case to `shelf_todo_wiring_test.go`.

### WR-03: CONTEXT.md and HANDOFF.md recompute tasks from state and never display shelf-seeded todos

**File:** `cmd/recovery_snapshot.go:417` (`activeTasks := sessionActiveTodosFromState(state)`), `cmd/recovery_snapshot.go:575` (`tasks := sessionActiveTodosFromState(state)`)
**Issue:** `syncSessionFromState` (line 134) carefully merges shelf todos into `session.ActiveTodos` so they survive refreshes — but both recovery renderers, which receive that very `session` value as a parameter, ignore it and re-derive tasks from phase state only. Result: a promoted shelf entry exists durably in `session.json` but never appears in CONTEXT.md's "Active Todos" or HANDOFF.md's "Tasks" — the two documents the system describes as "the colony's memory" for context-collapse recovery. The stated goal of the gap closure ("a promoted shelf entry must become a durable colony todo") is met in the data file but not in either human/LLM-facing recovery artifact. (`cmd/context.go:315` gets this right for resume: it prefers `session.ActiveTodos`.)
**Fix:** In both renderers, use the merged list, falling back to derivation only when empty:
```go
activeTasks := session.ActiveTodos
if len(activeTasks) == 0 {
    activeTasks = sessionActiveTodosFromState(state)
}
```
Extend `TestSessionRefreshPreservesShelfTodos` to assert the shelf todo string appears in the written `CONTEXT.md`.

### WR-04: Broken env-var restore in shelf wiring tests — defer evaluates `os.Getenv` immediately, leaking a deleted temp dir into `AETHER_ROOT`

**File:** `cmd/shelf_todo_wiring_test.go:30-31, 93-94, 139-140`
**Issue:** All three integration tests contain:
```go
os.Setenv("AETHER_ROOT", tmpDir)
defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
```
Deferred function *arguments* are evaluated at `defer` time — after the `Setenv` on the previous line — so the "restore" restores `tmpDir` itself. When the test ends, `AETHER_ROOT` remains pointed at a `t.TempDir()` that Go has deleted, for the remainder of the package test run. `saveGlobals` (cmd/testing_main_test.go:116) restores package globals but not environment, so nothing else cleans this up; any later test in the run that relies on ambient `AETHER_ROOT` resolution inherits a dangling path. Other test files in the package handle this correctly (`build_flow_cmds_test.go:24-31` captures the original before setting, or uses `t.Cleanup` + `Unsetenv`).
**Fix:** Replace both lines with `t.Setenv("AETHER_ROOT", tmpDir)` in all three tests — it saves and restores the pre-test value automatically.

### WR-05: `shelf-promote-batch` reports `ok: true` even when every requested ID fails

**File:** `cmd/shelf_init.go:43-63`
**Issue:** Per-ID failures are collected into a `failed` array, but the command always terminates with `outputOK(...)` — including when `promoted` is empty and *all* IDs failed (typo'd ID, entry already dismissed, unreadable shelf). The init wrapper's Shelf Backlog stage instructs the LLM only to surface the `todos` array; nothing tells it to inspect `failed`. A total failure therefore renders as a success with an empty carry-forward list, and the user's chosen promotions vanish without any error signal. (Note also that `promoteShelfEntry`/`dismissShelfEntry` apply no status guard, so a re-run can silently retarget an entry already promoted to a different colony.)
**Fix:** When `len(promoted) == 0 && len(failed) > 0`, return `outputError` naming the failed IDs; otherwise include a non-zero `failed_count` and have the wrapper spec surface failures ("If `failed` is non-empty, tell the user which IDs did not promote").

### WR-06: Stale RED/GREEN narrative in `TestLifecycleFlatMirrorsMatchCanonical` doc comment

**File:** `cmd/lifecycle_wrapper_contract_test.go:129-134`
**Issue:** The comment states the test "is currently RED for build and init (both proven drifted by research) and GREEN for plan and continue. Task 3 of this plan resyncs build and init and turns this fully green." Verified today: all four verbs' flat mirrors are byte-identical to canonical and the test passes. A comment describing a permanent invariant test as intentionally failing invites a future reader to dismiss a *real* future failure as "the known RED state." This is the same documentation-truth class the phase itself polices.
**Fix:** Rewrite the comment to describe the standing invariant only (mirrors must be byte-identical; `aether install`/`update` copy, never hand-edit), dropping the point-in-time RED/GREEN status.

## Info

### IN-01: Dead code in `formatShelfForInit` — `phaseStr` assigned and discarded

**File:** `cmd/shelf_init.go:249-253`
**Issue:** The `phase == 0` branch assigns `phaseStr := "unknown"` then immediately discards it with `_ = phaseStr`; the string is never printed. Either the "unknown" phase was meant to be rendered or the two branches should collapse.
**Fix:** Delete the two dead lines, or render the intended `(from phase unknown)` suffix.

### IN-02: `no_permissive_read_phrasing` subtest scans the whole file but its failure message claims the phrase is in the `<read_only>` block

**File:** `cmd/lifecycle_wrapper_contract_test.go:662-678`
**Issue:** The check runs `strings.Contains(string(content), permissiveReadOnlyReadPhrase)` over the entire file, but the error text asserts "`<read_only>` block uses the permissive phrase." A future wrapper edit that quotes "may read but never write" anywhere else (e.g., inside a guardrail explaining what *not* to say) fails with a misleading diagnosis.
**Fix:** Scope the check to `extractReadOnlyBlock(text)` (already available two functions up), or reword the message to say the phrase may not appear anywhere in the file.

### IN-03: Redundant entries in the init wrapper forbidden list

**File:** `cmd/init_wrapper_ceremony_test.go:71-74`
**Issue:** The forbidden markers ``"append `[shelf:"`` and ``"to `active_todos`"`` are strictly subsumed by the bare `"active_todos"` entry two lines below — any text matching the first two necessarily matches the third. Harmless, but the redundancy suggests the narrower fences predate the broad one and can confuse maintenance.
**Fix:** Keep the broad `"active_todos"` fence and drop the two subsumed entries (or keep all three with a comment noting the subsumption is intentional belt-and-braces).

### IN-04: Wrapper labels dismissal as "Delete permanently" but the runtime only marks status dismissed

**File:** `.claude/commands/ant/init.md:252-256` (and mirrors), `cmd/shelf_init.go:144-161`
**Issue:** Shelf Backlog option 3 is presented to the user as "Delete permanently," but the mapped command `shelf-dismiss-batch` sets `Status = dismissed` and the entry remains in `shelf.json` indefinitely. Overstating destruction is the safe direction, but the wording misleads a user who later expects the data to be gone (or, conversely, doesn't realize it is recoverable).
**Fix:** Rename the option "Dismiss (remove from backlog)" or make dismissal actually delete the entry — pick one and align wrapper wording with runtime behavior.

### IN-05: `mergeShelfTodos` dedupes identical derived task strings

**File:** `cmd/shelf_init.go:202-223`
**Issue:** The `seen` map dedupes derived (phase-task) entries against each other, not just against shelf entries — two distinct incomplete tasks that happen to share identical goal text collapse to one line in `active_todos`. Cosmetic in practice, but it means the todo count can under-report open work.
**Fix:** Only consult/populate `seen` for shelf-prefixed strings when appending derived entries, or accept and document the collapse.

---

## Verification Performed

- `go test ./cmd/ -run 'TestLifecycle|TestShelf|TestInitSeeds|TestSessionRefreshPreserves|TestMergeShelfTodos|TestInitWrapper|TestBuildWrapper|TestContinueWrapper|TestPlanWrapper|TestBuildMdOwnership|TestBriefPath|TestSpecialistCommand|TestWrapperHostContract' -count=1` — **PASS**
- `go vet ./cmd/` — clean
- `diff` across all 12 wrapper surfaces: flat mirrors byte-identical to canonical for all four verbs; `.opencode` parity exact except init.md's single sanctioned `AskUserQuestion` line
- Prior-review CR-01 (hand-append to `active_todos`): confirmed absent from all init wrapper surfaces; fence markers present in `TestInitWrapperCeremonyContract`
- Prior-review WR-05 (pheromone gating): confirmed fixed in Approval stage and locked by `approval_writes_pheromones_only_after_init_succeeds`
- Full call-chain trace: `shelf-promote-batch` → `promoteShelfEntry` → `aether init` (`promotedShelfTodos`) → `session.json` → `syncSessionFromState`/`session-update` (`mergeShelfTodos`) → recovery renderers

---

_Reviewed: 2026-08-03T12:05:11Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
