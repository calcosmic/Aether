---
phase: 165-core-lifecycle-commands
reviewed: 2026-08-03T15:03:33Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - .aether/commands/init.yaml
  - .claude/commands/ant-init.md
  - .claude/commands/ant/init.md
  - .opencode/commands/ant/init.md
  - cmd/command_guide.go
  - cmd/init_ceremony.go
  - cmd/init_cmd.go
  - cmd/init_wrapper_ceremony_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - cmd/recovery_snapshot.go
  - cmd/shelf_init.go
  - cmd/shelf_test.go
  - cmd/shelf_todo_wiring_test.go
  - cmd/testdata/command_catalog.json
findings:
  critical: 1
  warning: 6
  info: 5
  total: 12
status: issues_found
---

# Phase 165: Code Review Report (Gap-Closure Round 2 — plans 165-09 / 165-10)

**Reviewed:** 2026-08-03T15:03:33Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

This round moved shelf-entry promotion inside the `aether init` transaction
(`--promote-shelf` / `--dismiss-shelf` on `initCmd`, applied by
`applyInitShelfSelections`) and rewrote the three init.md wrapper surfaces so
shelf IDs are collected at Shelf Backlog and spent only inside the Approval
stage's init call. The core consent-ordering fix is sound and well-fenced:
`TestFailedInitLeavesShelfEntriesShelved`, `TestInitPromotesUnderRevisedGoal`,
and the new wrapper contract subtests
(`shelf_ids_are_spent_only_inside_the_approval_init_call`) genuinely pin the
behavior, and `TestLifecycleFlatMirrorsMatchCanonical` keeps the flat mirror in
sync. `go build`, `go vet`, and all shelf/init/wrapper tests pass.

However, the atomicity guarantee the wrappers now advertise is stronger than
what the runtime actually provides (WR-01), an ID passed to both flags
double-reports and silently loses its todo (WR-02), and the sibling
`init-ceremony` path — in this review's scope and touched by this diff —
still lacks every sealed-colony safeguard `aether init` was hardened with,
including proceeding with the overwrite when the backup write fails (CR-01).

*For dummies: the main fix works — cancelling or failing colony setup no
longer eats your saved backlog ideas in the normal flow. But the older
"ceremony" setup path can still silently wipe a finished colony's memory, and
the docs promise slightly more all-or-nothing behavior than the code delivers
in rare disk-failure cases.*

## Critical Issues

### CR-01: `createCeremonyColony` can destroy a sealed colony's state without confirmation, and proceeds even when the backup fails

**File:** `cmd/init_ceremony.go:464-494`
**Issue:** The `aether init` path was deliberately hardened (see
`cmd/init_cmd.go:76-94` and its comment "Destroying a colony's memory requires
saying so out loud"): `StateCOMPLETED` counts as sealed, re-init requires
`--confirm-reinit`, and the pre-overwrite backup is mandatory — a failed backup
aborts the init. The `init-ceremony` path (`createCeremonyColony`) has none of
this:

1. Only `Milestone == "Crowned Anthill"` is treated as sealed;
   `StateCOMPLETED` is not (line 472), so the two commands classify the same
   colony differently.
2. There is no confirmation that a sealed colony is about to be replaced. The
   user approves a launch brief that never mentions the existing colony.
3. The backup is best-effort and its failure is silently swallowed:

```go
if err := os.MkdirAll(backupDir, 0755); err == nil {
    backupFile := filepath.Join(...)
    if err := copyFile(statePath, backupFile); err == nil {
        fmt.Fprintf(os.Stderr, "warning: backed up previous colony state to %s\n", backupFile)
    }
}
```

If `MkdirAll` or `copyFile` fails, execution falls through to
`store.SaveJSON("COLONY_STATE.json", state)` at line 548 and the only copy of
the sealed colony's history (phases, instincts, decisions, errors) is
destroyed with no backup and no message. This is a data-loss path on a
supported (Codex-facing) colony-creation command, and it directly contradicts
the guarantee the primary path enforces.
**Fix:** Mirror `init_cmd.go`: treat `existing.State == colony.StateCOMPLETED`
as sealed-equivalent; refuse the overwrite when the backup cannot be written
(`return fmt.Errorf("cannot back up previous colony state: %v — refusing to overwrite without a backup", err)`);
and surface the sealed-colony replacement to the user during the ceremony
(an explicit prompt via `promptNumberedChoice`, or a required
`--confirm-reinit` flag on `init-ceremony`), reporting the backup path and
restore command as `init_cmd.go:161` does.

## Warnings

### WR-01: Post-state-save failures strand a shelf promotion — "a failed init persists nothing" only holds for refusal branches

**File:** `cmd/init_cmd.go:208-258` (also `cmd/shelf_init.go:136-143`, `.claude/commands/ant/init.md:295-297`, `.aether/commands/init.yaml:48`)
**Issue:** `applyInitShelfSelections` runs immediately after the
`COLONY_STATE.json` save, but three fallible steps still follow it:
`store.SaveJSON("session.json", ...)` (line 232), `syncColonyArtifacts`
(line 237), and `store.AppendJSONL("activity.log", ...)` (line 255). If any of
them fails, `aether init` returns an `ok:false` envelope — i.e. "a failed
init" from the wrapper's point of view — but `shelf.json` has already been
mutated. The code comment at line 208-213 ("a failed `aether init` writes
nothing to shelf.json") and the wrapper's Stop conditions ("A cancel or a
failed `aether init` both end the command with nothing persisted — no charter,
no pheromones, no shelf promotion or dismissal") are only true for the refusal
branches above the state save. Worse, the retry is then blocked ("colony
already initialized"), and any session later recreated by
`syncSessionFromState` starts from `ActiveTodos: []string{}`
(`cmd/recovery_snapshot.go:112`) without consulting the shelf — so the
promoted entry is off the backlog and its todo is never seeded. That is
exactly the stranded-promotion failure this gap-closure round set out to
eliminate, still reachable through a mid-init I/O failure.
**Fix:** Make the shelf write the last fallible mutation in the transaction:
resolve the selections against the loaded shelf file in memory first (yielding
the promoted entries and their todo strings), build `session.ActiveTodos` from
that in-memory result plus `promotedShelfTodos`, and only call
`writeShelfFile` after `activity.log` has been appended. Alternatively, soften
the wrapper/comment claim to "a refused init writes nothing to the shelf" and
document the partial-failure window.

### WR-02: An ID passed to both `--promote-shelf` and `--dismiss-shelf` is reported as both promoted and dismissed, and its todo silently vanishes

**File:** `cmd/shelf_init.go:144-168`
**Issue:** `applyInitShelfSelections` applies all promotes, then all
dismisses, with no overlap check. For an ID present in both lists:
`promoteShelfEntry` succeeds (ID lands in `shelf_promoted`), then
`dismissShelfEntry` overwrites the status to `ShelfDismissed` (ID also lands
in `shelf_dismissed`). The final state is dismissed, so `promotedShelfTodos`
(which filters on `Status == colony.ShelfPromoted`, line 250) never emits the
todo — yet the init result envelope tells the wrapper the promotion
succeeded, and the wrapper's Approval step relays that to the user
("what this colony will carry forward"). The entry also retains a stale
`PromotedTo` pointing at the new goal while dismissed. The wrapper's
three-way menu makes overlap unlikely from a well-behaved LLM, but the
runtime is the trust boundary and must not report contradictory results.
**Fix:** Reject or de-duplicate the intersection before applying:
```go
promoteIDs := splitShelfIDs(promoteRaw)
dismissSet := make(map[string]bool)
for _, id := range splitShelfIDs(dismissRaw) { dismissSet[id] = true }
for _, id := range promoteIDs {
    if dismissSet[id] {
        failed = append(failed, id) // conflicting instruction — apply neither
        delete(dismissSet, id)
        continue
    }
    ...
}
```

### WR-03: init-ceremony "Reject" never returns to a goal prompt — it dead-ends in a misleading "research failed" error

**File:** `cmd/init_ceremony.go:302-316` (interacts with `cmd/init_research.go:1811-1814`)
**Issue:** Choosing "Reject — return to goal prompt" sets `goal = ""` and
`continue`s the outer loop, which immediately calls
`runCeremonyResearch("", target)`. `initResearchCmd.RunE` silently returns
`nil` with no output when `--goal` is empty, so `runCeremonyResearch` returns
"init-research produced no output", and the command exits with
`research failed: init-research produced no output`. The promised "goal
prompt" does not exist — `promptString` is defined at line 40 but never called
in this flow. The user who rejects a brief gets a cryptic error instead of a
chance to enter a new goal.
**Fix:** After Reject, prompt for the new goal before looping:
```go
case 3: // Reject
    fmt.Fprintln(os.Stderr, "  Brief rejected.")
    goal = promptString("Enter a new colony goal (empty to abort)")
    if goal == "" {
        outputError(1, "init ceremony aborted: no goal provided", nil)
        return nil
    }
```
(and drop the now-dead `if goal == "" { continue }` at line 314).

### WR-04: `promptNumberedChoice` spins forever on stdin EOF

**File:** `cmd/init_ceremony.go:22-37` (loop at 246-312)
**Issue:** The `ReadString('\n')` error is discarded. On EOF (Ctrl-D at the
approval prompt, or exhausted piped input), `ReadString` returns `("", err)`
immediately on every call, `Atoi` yields 0, the `default` branch prints
"Invalid choice" and re-prompts — an unbounded busy loop flooding stderr with
no way to terminate except SIGINT.
**Fix:** Propagate the read error and treat it as cancel:
```go
response, err := reader.ReadString('\n')
if err != nil && response == "" {
    return -1 // caller treats as abort
}
```
and handle `-1` in the ceremony loop by aborting cleanly.

### WR-05: `isTerm` dereferences a nil `FileInfo` when `Stat` fails

**File:** `cmd/init_ceremony.go:597-600`
**Issue:** `fi, _ := f.Stat()` ignores the error; if `Stat` fails (e.g. stdin
closed by the parent process), `fi` is nil and `fi.Mode()` panics with a nil
pointer dereference before the ceremony even starts.
**Fix:**
```go
func isTerm(f *os.File) bool {
    fi, err := f.Stat()
    if err != nil {
        return false
    }
    return (fi.Mode() & os.ModeCharDevice) != 0
}
```

### WR-06: Canonical run-command templates omit `--promote-shelf` / `--dismiss-shelf` while the surrounding prose requires them

**File:** `cmd/command_guide.go:166` (RunCommand), `.aether/commands/init.yaml:5` (runtime.command) and `:25` (intent_refinement)
**Issue:** The new command-guide pre-step instructs the Codex flow to "carry
the chosen IDs into the init call with `--promote-shelf` / `--dismiss-shelf`",
and init.yaml's guardrail (line 48) makes the same demand — but both
machine-facing command templates still read
`aether init --colony-mode <...> --charter-json '<...>' "<refined goal>"`
with no shelf flags. A consumer that copies the RunCommand template verbatim
(the exact drift the repo's own Cross-Platform Drift Guard exists to prevent)
silently drops the user's shelf choices; the entries stay shelved and no
error surfaces. The Claude/OpenCode wrappers carry the flags, so the three
surfaces now disagree on the canonical invocation.
**Fix:** Add the optional flags to both templates, e.g.
`AETHER_OUTPUT_MODE=visual aether init --colony-mode <colony|orchestrator> --charter-json '<synthesized charter JSON>' [--promote-shelf "<ids>"] [--dismiss-shelf "<ids>"] "<refined goal>"`,
and mirror the same string into `.aether/commands/init.yaml` runtime.command.

## Info

### IN-01: `formatShelfForInit` is dead code containing further dead code

**File:** `cmd/shelf_init.go:292-328`
**Issue:** `formatShelfForInit` has no non-test caller. Inside it, the
`phase == 0` branch assigns `phaseStr := "unknown"` and immediately discards
it (`_ = phaseStr`, lines 318-319). It also silently drops any entry whose
category is outside the four hard-coded values, so its numbering would skip
entries if a new category were added.
**Fix:** Delete the function (and its dead branch), or wire it to a caller and
remove the `phaseStr` placeholder.

### IN-02: Wrappers never state that shelf ID lists must be comma-separated

**File:** `.claude/commands/ant/init.md:287` (same line in the other two surfaces)
**Issue:** The Approval call shows `--promote-shelf "{promoted_shelf_ids}"`
without specifying the list format. `splitShelfIDs` splits only on commas, so
a space- or newline-joined list becomes one unmatched ID and every choice
lands in `shelf_failed`. The flag help text says comma-separated; the wrapper
should too.
**Fix:** In the Approval bullet, note: "comma-separated, e.g.
`--promote-shelf \"shelf_1,shelf_2\"`".

### IN-03: "Delete permanently" is a soft dismiss, and status transitions are unguarded

**File:** `.claude/commands/ant/init.md:256`; `cmd/shelf_init.go:211-228`
**Issue:** The Shelf Backlog menu offers "Delete permanently", but
`--dismiss-shelf` only flips the entry's status to `dismissed`; the entry
remains in `shelf.json` indefinitely. User-visible behavior (gone from the
backlog listing) matches, but "permanently" overstates it. Relatedly,
`promoteShelfEntry`/`dismissShelfEntry` don't check the current status, so an
already-dismissed or already-promoted-elsewhere entry can be silently
re-promoted or re-dismissed by ID.
**Fix:** Either rename the option ("Remove from backlog") or make dismiss
actually delete the entry; optionally guard `promoteShelfEntry` to only act on
`ShelfShelved` entries and return an error otherwise (surfacing via
`shelf_failed`).

### IN-04: Shelf-seeded todos have no completion or removal path within a colony

**File:** `cmd/shelf_init.go:269-290`; `cmd/recovery_snapshot.go:134`
**Issue:** `mergeShelfTodos` unconditionally re-pins every `[shelf:`-prefixed
entry from the existing session on every refresh. Nothing in the reviewed
surface can mark a shelf-derived todo done, so once seeded it persists in
`active_todos`, CONTEXT.md, and HANDOFF.md until the next `aether init`
removes `session.json` — even after the underlying work ships. Deliberate
retention-vs-clobber trade-off from this round, but worth a follow-up
mechanism (e.g. a `session-todo-complete` subcommand or matching a shelf todo
against completed phase tasks).
**Fix:** Track as a follow-up; at minimum document the lifetime in the shelf
docs.

### IN-05: `initShelfBacklogSection` doc comment claims it fails via `t.Fatalf`

**File:** `cmd/init_wrapper_ceremony_test.go:219-232`
**Issue:** The comment says "Fails the calling test via t.Fatalf when the
heading is absent", but the function just returns `""`; the caller performs
the Fatalf. Harmless today, but a future caller relying on the documented
behavior would silently pass on a missing heading.
**Fix:** Correct the comment: "Returns \"\" when the heading is absent;
callers must treat that as fatal."

---

## Verified Sound

- Consent ordering: `--promote-shelf`/`--dismiss-shelf` are parsed up front
  but applied only after every refusal branch (active colony, sealed colony
  without `--confirm-reinit`, in-progress seal, invalid inputs, failed state
  save) — `TestFailedInitLeavesShelfEntriesShelved` pins the refusal case.
- Goal coupling: promotion uses the same `goal` variable the colony is created
  with, so a revised goal cannot orphan a promotion
  (`TestInitPromotesUnderRevisedGoal`).
- Non-blocking failures: a bad shelf ID never fails colony creation and is
  reported via `shelf_failed` (`TestInitReportsUnpromotableShelfIDs`); the
  earlier round's WR-05 total-failure envelope fix on the batch commands is
  present and tested.
- Wrapper surfaces: all three init.md files are consistent
  (`TestLifecycleFlatMirrorsMatchCanonical` green; the single sanctioned
  AskUserQuestion delta is enforced); forbidden vocabulary now fences
  `shelf-promote-batch`/`shelf-dismiss-batch`; `--promote-shelf` appears only
  after `## Approval`; Shelf Backlog contains only the read-only
  `aether shelf-list` (whose `result.total`/`result.entries` keys match
  `cmd/shelf_cmd.go:39-40`).
- `promoteShelfEntry` trims `PromotedTo` symmetrically with the query-side
  trim in `promotedShelfTodos` (prior round's WR-01 closed).
- Test hygiene: the broken `defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))`
  no-op restore pattern was replaced with `t.Setenv`.
- `cmd/testdata/command_catalog.json` correctly registers the two new flags.
- `go build ./cmd/...`, `go vet ./cmd/`, and all 13 targeted shelf/init/wrapper
  tests pass.

---

_Reviewed: 2026-08-03T15:03:33Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
