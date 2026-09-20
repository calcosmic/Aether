# Phase 188: One Truth for Failures and Advances - Context

**Gathered:** 2026-08-19
**Status:** Ready for planning
**Source:** Direct codebase reconnaissance by the planner (no `/gsd-discuss-phase` session exists
for this phase — the owner was asleep when this was requested). Every decision below was reached
by reading the live source, not by inference from documentation, and is marked **auto-decided
(owner asleep) — revisit if wrong** per the planning brief's instruction. None of these are open
questions blocking execution; each is a concrete, conservative choice with its reasoning stated.

<domain>
## Phase Boundary

This phase delivers six independent, self-contained truth-and-discipline fixes, all found by
reading (not assuming) the current code:

1. **One midden.** Failures are written to the flat file `midden.json`, but four production
   consumers (colony-prime, autopilot, immune, memory-health) read from the nested, never-written
   path `midden/midden.json`. All four have been silently blind since the day they were written.
2. **One phase-advance path.** `aether continue` and `aether continue-finalize` each carry their
   own ~60-line copy of the "commit phase completion atomically" logic. One copy has the
   supersession check (`validateRuntimeStateStillCurrent`); the other does not, and additionally
   overwrites its freshly-loaded state with a stale captured variable before writing it back.
3. **One gated way to move `current_phase`.** `aether state-mutate --field current_phase` can be
   invoked today with no `--guard`, bypassing every precondition check that a real phase advance
   would have enforced.
4. **A property ratchet, not a strings grep, guarding the phase-advance write path against ever
   regressing to a non-atomic write.**
5. **A readable record when the autopilot's own build-retry loop gives up**, mirroring the
   worker-level recovery orchestrator's already-good `RecoveryLogEntry` discipline, which this
   one loop does not share.
6. **A loud warning — not new refusal logic — when build-finalize accepts a manifest with no
   attempt binding at all** (the `Legacy: true` branch in `validateBuildAttemptManifestBinding`,
   which the 2026-08-17 Audit Addendum explicitly keeps: *"the legacy unbound-manifest branch in
   attempt binding stays (loud warning added in 188)"*).

**Out of scope** (see decisions below for why): converting the ~30 pre-existing
`store.SaveJSON("COLONY_STATE.json", ...)` call sites across the runtime to atomic writes. That is
the ROADMAP's own deferred `LOCK` requirement category (state locking and lifecycle transactions),
not this phase's "one phase-advance discipline" goal. This phase makes the phase-advance path
itself atomic and ratchets it — it does not migrate every unrelated one-shot CLI command's state
write.

</domain>

<decisions>
## Implementation Decisions

### One midden (criterion 1)

- **D-01 (auto-decided, owner asleep — revisit if wrong):** The canonical midden path is the flat
  `midden.json` (resolves to `.aether/data/midden.json`), **not** the nested `midden/midden.json`.

  *Evidence, not inference:* the only two production writers —
  `cmd/midden_cmds.go`'s `midden-write` command (line 463) and `cmd/spawn_budget.go`'s
  `spawnTreeBudgetCeilingToMidden` (line 381) — both call `store.SaveJSON("midden.json", mf)`.
  A repo-wide grep of `cmd/*.go` (excluding tests) found **zero** production writers to
  `midden/midden.json`; every reference to that nested path outside a test fixture is a *reader*:
  `cmd/autopilot.go:89` (`checkAutopilotPauseConditions`), `cmd/context.go:236`
  (`buildResumeDashboardResult`) and `cmd/context.go:859` (`buildContextCapsuleOutput` — this is
  colony-prime's context-capsule builder), `cmd/memory_health.go:77`
  (`loadMemoryHealthSummary`), and `cmd/immune.go:256` (`immuneAutoScarCmd`). This is not a
  judgment call between two valid options; one path is written and one is not.

- **D-02 (auto-decided):** A new shared helper pair lives in `cmd/midden_shared.go`:
  `loadMiddenFile(s *storage.Store) (colony.MiddenFile, error)` (canonical read) and
  `appendMiddenEntry(s *storage.Store, category, source, message string, tags []string) error`
  (canonical write, using `store.UpdateJSONAtomically` — stricter than either existing writer,
  which both do a plain load-then-`SaveJSON`). All four named readers converge on
  `loadMiddenFile`. Migrating the two *existing* writers (`midden_cmds.go`, `spawn_budget.go`) to
  call `appendMiddenEntry` is **discretionary, not required** — they already write the correct
  path; only the read side is broken. Criterion 5's new retry-exhaustion record (see D-11) is
  required to use `appendMiddenEntry`, which is what makes building it in this phase, not later,
  the interface-first move: it is exercised by criterion 5's plan in the very next wave, which
  doubles as proof that criterion 1's fix actually closes the loop end-to-end (a value written
  through the new writer becomes visible through the newly-fixed readers).

- **D-03 (auto-decided, bonus — not named in the ROADMAP criterion):** `cmd/medic_scanner.go`'s
  `scanDataFiles` (the `aether patrol`/`aether medic` health scan) has the identical bug: its
  `structuredFiles` list validates `"midden/midden.json"` (line 511), so Aether's own health
  checker has never validated the real midden file. The ROADMAP's criterion 1 names exactly four
  consumers (colony-prime, autopilot, immune, memory-health) and does not name medic. This is a
  one-line fix directly adjacent to the other four (same root cause, same repo-wide grep found
  it), so it is folded into the same plan as a bonus rather than left broken next to four freshly
  fixed siblings — but it is flagged explicitly here so it reads as a deliberate inclusion, not
  silent scope growth. `cmd/medic_scanner_test.go:435`'s `TestScanDataFilesCorrupted` seeds its
  fixture at the nested path today and **must** be updated to seed at the flat path in the same
  change, or the test will assert against a file the scanner no longer reads.

### One phase-advance path (criterion 2)

- **D-04 (auto-decided):** The shared function is extracted from `runCodexContinue`'s existing
  "ATOMIC STATE COMMIT" block (`cmd/codex_continue.go`, inside the function starting at line 523)
  — that copy is the *complete, correct* one (it already calls `validateRuntimeStateStillCurrent`
  inside the `UpdateJSONAtomically` closure and already handles `errRuntimeStateSuperseded` via
  `continueSupersededResult`). The other copy, `advanceExternalContinue`
  (`cmd/codex_continue_finalize.go:1080`), is rewritten to call the extracted function rather than
  the reverse — it is missing the supersession check entirely and carries a real bug (D-05).
  `continueSupersededResult(startState colony.ColonyState, phase colony.Phase, err error) map[string]interface{}`
  (`cmd/codex_continue.go:3736`) is package-scoped and type-compatible with both call sites — reuse
  it from both, rather than inventing a second superseded-result shape for the finalize path.

- **D-05 (auto-decided — this is a correctness fix, not a style choice):**
  `advanceExternalContinue`'s closure contains `updated = state` as its first line inside
  `UpdateJSONAtomically`'s mutate callback (`cmd/codex_continue_finalize.go:1091`). Because
  `UpdateJSONAtomically` already unmarshals the *current on-disk* file into `updated` before
  calling the closure, this line immediately **discards that fresh read** and replaces it with the
  `state` parameter captured long before this function ran (before verification, before gates,
  before review). Any concurrent write to `COLONY_STATE.json` between that capture and this commit
  is silently thrown away — the exact class of bug this criterion exists to close. The shared
  function must never do this: it operates only on the value `UpdateJSONAtomically` freshly loaded.

- **D-06 (auto-decided):** `advanceExternalContinue` also does a **second, non-atomic** write after
  its atomic block returns — `_ = store.SaveJSON("COLONY_STATE.json", updated)` at
  `cmd/codex_continue_finalize.go:1147`, error discarded, used only to persist Events appended by
  post-advance side effects (housekeeping, worker-closure). The default continue path already
  solves this correctly with `appendRuntimeStateEventsIfCurrent(updated, flowEvents)`
  (`cmd/codex_continue.go`, called after the atomic commit) — a helper that re-checks currency
  before appending rather than blindly overwriting. The unification must delete the trailing
  `SaveJSON` and route the finalize path's side-effect events through
  `appendRuntimeStateEventsIfCurrent` instead. This single change also removes one of the ~30
  non-atomic COLONY_STATE.json write sites the D-07 ratchet baseline would otherwise have had to
  allowlist.

### The ratchet (criterion 4)

- **D-07 (auto-decided, conservative scope-bound under `planner_authority_limits`):** The ratchet
  is **not** "every `store.SaveJSON`/`store.AtomicWrite` call against `COLONY_STATE.json` in the
  repo must become atomic." A repo-wide grep-count found **~30 production call sites** across
  **~25 files** (`abandon_cmd.go`, `build_flow_cmds.go`, `codex_build.go`, `codex_colonize.go`,
  `codex_plan.go` (x3), `codex_workflow_cmds.go`, `colony_cmds.go` (x3), `command_truth.go`,
  `error_cmds.go`, `entomb_cmd.go`, `internal_cmds.go`, `init_cmd.go`, `phase_commit.go`,
  `session_flow_cmds.go` (x4), `state_extra.go` (x2), `state_repair.go`, `state_cmds.go`,
  `state_load.go`, `worktree.go` (x4), plus `codex_continue_finalize.go`'s two sites which D-06
  removes). Converting all of these — most of them one-shot CLI commands (`init`, `colonize`,
  `abandon`, `entomb`, error-recording) with no concurrent writer in practice — is a full state-
  locking migration. It is explicitly named in `STATE.md`'s Deferred Items as `LOCK (4) — state
  locking and lifecycle transactions | Deferred`, and it would consume far more than one agent's
  safe context budget across far more than the 3-5 files a single plan should touch (legitimate
  split reason 1 under `planner_authority_limits`: context cost).

  Instead, the ratchet (in a new `cmd/colony_state_atomicity_ratchet_test.go`, depending on
  criterion 2's `advancePhase()` existing) enforces two things, both structural (AST-based),
  neither a plain string grep:

  1. **Baseline, shrink-only allowlist.** An AST walk of `cmd/*.go` (excluding `_test.go` and the
     ratchet's own file) finds every call of the shape `store.SaveJSON("COLONY_STATE.json", ...)`
     or `store.AtomicWrite("COLONY_STATE.json", ...)`, recorded as (file, enclosing function).
     Compared against a checked-in `cmd/testdata/colony_state_write_allowlist.json`, regenerable
     via an `-update-colony-state-write-allowlist` flag — the exact `-update-orphan-allowlist`
     idiom already established by `cmd/subcommand_reachability_ratchet_test.go:41` and enforced
     shrink-only by the `TestOrphanAllowlistOnlyShrinks` idiom at line 1423 of that file. A site
     not in the allowlist fails the test (a genuinely new non-atomic write appeared). An allowlist
     entry with no matching site fails the test too (keeps the list honest as code changes).
  2. **Zero-tolerance reachability from `advancePhase()`.** A second, much smaller check builds a
     same-package call graph (BFS over `*ast.CallExpr` targets, following `cmd`-package function
     calls transitively) rooted at `advancePhase` and asserts **no allowlist exemption is
     permitted** for anything reachable from it — this is the one path this phase actually
     rewrites, so it alone is held to the full standard the phase's name promises, structurally
     verified rather than asserted in prose.

  Per this repo's own Definition of Done and Phase 187's own finding (a guard that finds nothing
  passes vacuously forever), both checks fail loudly if they discover **zero** call sites at all —
  that means the AST walker itself broke, not that the codebase became perfectly atomic overnight.

  This directly serves "one phase-advance discipline" (the actual three-clause phase goal) without
  claiming a full-repo atomicity migration that did not happen — the exact "claims fixed when it
  isn't" failure `CLAUDE.md`'s Definition of Done exists to prevent.

- **D-08 (auto-decided):** Allowlist entries are keyed by **(file, enclosing function name)**, not
  by line number. Line numbers drift on any unrelated edit above the call site; function names are
  stable identifiers within a file, matching how `cmd/testdata/orphan_allowlist.json` keys by
  command name (a stable identifier) rather than a source location.

### The state-mutate bypass (criterion 3)

- **D-09 (auto-decided — zero live regression risk, confirmed by reading, not assumed):**
  `cmd/state_cmds.go`'s `executeFieldMode`, for `field == "current_phase"`, will require that the
  command was invoked with `--guard phase-advance:<N>` where `<N>` matches the phase number being
  set via `--value`; absent or mismatched, it refuses with a clear, plain-language error and makes
  no write. Confirmed by reading, not assumed: (a) `--guard` already exists as a working, optional
  flag (`enforceGuard` → `runGateCheck("phase-advance", ...)` → `checkAllTasksCompleted` +
  `checkTestsPass` + `checkNoCriticalFlags`), it is simply never *required* for this one
  destructive field; (b) `advancePhase()` (criterion 2) mutates the `colony.ColonyState` struct
  directly in Go and calls `store.UpdateJSONAtomically` directly — it never shells out to
  `aether state-mutate`, so gating the CLI cannot break the internal advance path; (c) a repo-wide
  search for `state-mutate.*--field current_phase` (and the reverse) found **no live wrapper
  caller** — the only two hits are inside `.aether/docs/command-playbooks/*.md`, which
  `CLAUDE.md`'s own "Command Playbooks (Reference Material)" section states plainly have not been
  loaded by the runtime since v1.25. `current_phase`'s other legitimate mutators
  (`cmd/phase_skip.go`'s `phase-skip` command, `cmd/state_extra.go`'s `insert-phase` handling) are
  separate Go code paths that never call `executeFieldMode`, so they are unaffected. This is a
  pure bypass closure with no known legitimate caller to break.

### The legacy manifest warning (criterion 6)

- **D-10 (ROADMAP-locked, not owner-guessed):** The `Legacy: true` branch in
  `validateBuildAttemptManifestBinding` (`cmd/build_attempt.go:461-462`, returned when
  `dispatch_manifest` carries neither `attempt_id` nor `attempt_path`) **stays exactly as-is** —
  the 2026-08-17 Audit Addendum states this directly: *"the legacy unbound-manifest branch in
  attempt binding stays (loud warning added in 188)."* This phase adds visibility only. Confirmed
  by reading `cmd/codex_build_finalize.go:415-495`: today, `binding.Legacy` is set but **never
  read anywhere** — a legacy manifest silently falls through the same `if !binding.Bound { ... }`
  branch as a completely normal fresh completion, indistinguishable in the logs.

- **D-11 (auto-decided, conservative — provide both channels since "logged" is ambiguous):** The
  warning is emitted through **two** channels, both already-established patterns in this exact
  codebase rather than a third invented one: an unconditional
  `fmt.Fprintf(os.Stderr, "warning: ...")` (matching `cmd/init_cmd.go:202`'s existing
  "plain language, every occurrence" convention, and Phase 187's D-02 rule that a destructive- or
  compatibility-relevant path must never handle its edge case silently), **and** an appended
  `colony.Event` string on the state being advanced (matching this exact file's own convention,
  e.g. `"%s|verification_passed|continue|..."`), so the warning survives past the terminal and is
  later queryable (`aether history`, `aether status`) rather than only a fleeting print. Providing
  both is strictly more conservative than choosing one.

### The retry-exhaustion record (criterion 5)

- **D-12 (auto-decided):** The **worker-level** recovery orchestrator
  (`cmd/recovery_orchestrator.go`) already does this well and needs no change: every outcome,
  including exhaustion (`escalateOutcome`), produces a `RecoveryLogEntry` that is persisted to
  `recovery-log-<phase>.json` by both call sites (`cmd/queen_wave_lifecycle.go:154-156` for build,
  `cmd/codex_continue_finalize.go:442-444` for continue-finalize's gate recovery). This phase does
  **not** touch that system — it already satisfies "an exhausted retry loop leaves a readable
  record."  The gap is a different, one-off retry loop: the **autopilot's own phase-build retry**
  inside `runCompatibilityAutopilot` (`cmd/compatibility_cmds.go:386`, the function behind the
  live `aether run` command — confirmed reachable, not dead code, via its two call sites at
  `compatibility_cmds.go:487,533`). On a build failure it sleeps 2s, retries once, and on the
  *second* failure does exactly `_ = syncRunAutopilotState(state, opts, "paused", "")`
  then `return nil, err` (`cmd/compatibility_cmds.go:456-463`) — no count, no first-attempt error
  text, no durable record of either failure. This is the loop this criterion targets.

- **D-13 (auto-decided):** The exhaustion record is written via `appendMiddenEntry` (D-02),
  category `"autopilot_retry_exhausted"`, source `"aether run"`, message naming the phase number
  and both attempts' error text. This is chosen over a bespoke file for two reasons: it is a
  genuine *failure* record (the literal definition of a `MiddenEntry`), and — because criterion 1
  fixes exactly the four readers of this file in the same phase — writing it here is also the most
  direct proof that criterion 1's fix works end-to-end: a value written after criterion 1 ships
  becomes visible through colony-prime, autopilot's own pause-check, immune, and memory-health,
  all four, without further plumbing.

- **D-14 (auto-decided, discretion on mechanics):** Test strategy is two-tier rather than one
  brittle end-to-end harness: (1) extract the "second failure → write a record" behavior into a
  small, directly unit-testable function (e.g. `recordAutopilotRetryExhaustion(phaseID int,
  firstErr, secondErr error, now time.Time) error`), unit-tested directly for its record content
  (fail-then-pass: assert the record exists and names both errors); (2) a structural check
  (reading the call site, or a lighter integration test if a clean forced-failure fixture is
  found) confirming this function is actually invoked from the retry-exhaustion branch in
  `runCompatibilityAutopilot`, not just defined and orphaned — this repo's own recurring failure
  mode per `CLAUDE.md`. No existing test in this codebase forces `runCodexBuildWithOptions` to
  fail deterministically twice in a row; inventing a reliable one is lower-value than proving the
  record-writer's content is correct and that the call site reaches it.

### Claude's Discretion

- Exact wording of the D-11 stderr/Event warning text and the D-09 refusal error text, provided
  each is plain-language (per `CLAUDE.md`'s non-technical-owner rule: no "orphan"/"legacy"/
  "binding" jargon in user-facing text without inline translation) and states what happened and
  why.
- Whether `midden_cmds.go`'s and `spawn_budget.go`'s *existing* writers are refactored onto
  `appendMiddenEntry` (D-02 leaves this optional cleanup, not required).
- Exact `Tags` content on the new `autopilot_retry_exhausted` midden entry (e.g. whether to tag
  `"critical"` so `checkAutopilotPauseConditions`'s own chaos-tag matcher would also catch it on a
  *subsequent* run — a nice-to-have, not required, since criterion 5 only asks for a readable
  record, not a new pause trigger).
- Implementation technique for the D-07 AST walker (manual `go/ast` traversal vs. `go/packages`)
  — provided it is structural (no plain string grep over source text) and satisfies the
  vacuous-check, shrink-only-allowlist, and zero-tolerance-on-`advancePhase` requirements.
- Exact fixture mechanics for D-14's second tier (forced double build-failure) — provided the
  chosen approach is deterministic (no flaky timing/network dependency) and the plan's SUMMARY
  states plainly which tier was actually achieved.

### Folded Todos

Not applicable. This CONTEXT.md was authored directly by the planner because no `/gsd-discuss-phase`
session exists for this phase (owner asleep) — there is no todo backlog to score against it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase governance
- `.planning/ROADMAP.md` § "Phase 188" (line ~958) — the goal and the six success criteria this
  phase is judged against; note `Depends on: nothing` since the 2026-08-18 rescope
- `.planning/ROADMAP.md` § "Phase 192" § "Accepted residue" (line ~985) — confirms D-10 directly:
  *"the legacy unbound-manifest branch in attempt binding stays (loud warning added in 188)"*
- `.planning/STATE.md` § "Deferred Items" — confirms `LOCK (4) — state locking and lifecycle
  transactions | Deferred`, the basis for D-07's scope bound
- `CLAUDE.md` § "Definition of Done" — a requirement is satisfied only when a command exists that
  fails when the requirement is unmet; prefer invariant/proportion tests over named-section checks

### Criterion 1 — midden
- `cmd/midden_cmds.go:415-467` — `midden-write`'s RunE: the real production writer, flat
  `"midden.json"`, load-then-`SaveJSON` (not atomic — left as-is per D-02)
- `cmd/spawn_budget.go:349-382` — `spawnTreeBudgetCeilingToMidden`: the second real production
  writer, same flat path, same load-then-save shape
- `cmd/autopilot.go:49-107` — `checkAutopilotPauseConditions`, live (called from
  `cmd/compatibility_cmds.go:487,533` and `cmd/autopilot.go:197`); the nested-path read is at
  line 89
- `cmd/context.go:116-240` — `buildResumeDashboardResult` (`/ant-resume` dashboard); nested-path
  read at line 236
- `cmd/context.go:435-882` — `buildContextCapsuleOutput`, colony-prime's own context-capsule
  builder; nested-path read at line 859, inside the numbered "9. midden: Load midden.json" step
- `cmd/memory_health.go:25-85` — `loadMemoryHealthSummary`; nested-path read at line 77
- `cmd/immune.go:242-290` — `immuneAutoScarCmd`; nested-path read at line 256, **plus** a second,
  independent bug — it reads `entry["description"]` (line 274) but `colony.MiddenEntry`'s message
  field is JSON-tagged `"message"` (`pkg/colony/midden.go:11`), so even fixing the path alone
  would leave every scar's `description` empty
- `cmd/medic_scanner.go:502-516` — `scanDataFiles`'s `structuredFiles` list, entry
  `{"midden/midden.json", "midden.json"}` (line 511) — the bonus D-03 fix
- `cmd/medic_scanner_test.go:433-449` — `TestScanDataFilesCorrupted`, seeds at the nested path;
  must move to the flat path alongside the D-03 fix
- `cmd/run_autopilot_test.go:228-292` — `TestGoldenAutopilotPauseConditions`, **the parity anchor
  named by `.aether/docs/PARITY_CLASSIC_VS_GO.md`**; its `critical_chaos_findings` case seeds
  `filepath.Join(dataDir, "midden", "midden.json")` (nested) directly via `os.WriteFile` with a
  hand-typed JSON blob using field name `"description"` (not the real `"message"` tag). This test
  currently **passes** — not because the real bug is fixed, but because it seeds and reads the
  *same wrong path*, self-consistently. It does not exercise the real writer at all. Fixing the
  four readers' path **will break this test's fixture** unless the seed is moved to the flat path
  in the same change — do not "fix" the break by reverting the reader path back to nested.
- `pkg/colony/midden.go:1-56` — `MiddenEntry`, `MiddenFile` struct definitions (JSON tags:
  `id`, `timestamp`, `category`, `source`, `message`, `reviewed`, `tags`)

### Criterion 2 — advancePhase()
- `cmd/codex_continue.go:523` — `runCodexContinue`, the function containing the correct atomic
  block (inline, starting around line 908, "--- ATOMIC STATE COMMIT ---")
- `cmd/codex_continue.go:3736-3759` — `continueSupersededResult`, reusable from both paths
- `cmd/codex_build.go:179` — `var errRuntimeStateSuperseded = errors.New("runtime state superseded")`
- `cmd/codex_build.go:2597-2621` — `validateRuntimeStateStillCurrent` full definition: checks
  `state.Paused`, `state.CurrentPhase == phaseNum`, phase index in range, phase status is
  `PhaseInProgress`, `BuildStartedAt` still matches, and `state.State` is one of `allowedStates`
- `cmd/codex_continue_finalize.go:123` — `runCodexContinueFinalize`, the entry point behind
  `aether continue-finalize` (cobra command `continueFinalizeCmd`, line 30)
- `cmd/codex_continue_finalize.go:1080-1080+` — `advanceExternalContinue`, the function to
  rewrite; the bug is at line 1091 (`updated = state`) and the trailing non-atomic write is at
  line 1147
- `cmd/codex_continue_finalize.go:510` — the one call site of `advanceExternalContinue`, inside
  `runCodexContinueFinalize`
- `cmd/codex_workflow_cmds.go:212,263` — `continueCmd`, the cobra command behind `aether continue`,
  calls `runCodexContinue`

### Criterion 3 — state-mutate gating
- `cmd/state_cmds.go:21-66` — `stateMutateCmd`'s `RunE`: `--guard` is read and enforced only
  `if guard != ""` (optional today)
- `cmd/state_cmds.go:111-147` — `enforceGuard`, already implements `phase-advance:<N>` gate
  checking via `runGateCheck`
- `cmd/state_cmds.go:251-324` — `executeFieldMode`, the function to change; the `current_phase`
  case is at lines 272-281 and today performs **no** guard check of its own
- `cmd/state_cmds.go:935-944` — `init()`, flag registration (`--guard`, `--field`, `--value`)
- `cmd/phase_skip.go`, `cmd/state_extra.go:181,186` — the other legitimate `CurrentPhase` mutators;
  neither calls `executeFieldMode`, confirmed unaffected by the gating change

### Criterion 4 — the ratchet
- `cmd/worktree_destruction_reachability_test.go` — the property-guard model named by the planning
  brief: AST-parses `cmd/*.go`, builds a call graph (`buildCmdFuncGraph`), discovers destruction
  sites structurally, and fails loudly on zero findings (see `TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate`,
  line 1590). Note from Phase 187's own `187-VERIFICATION.md`: this exact guard, despite the
  effort behind it, was later shown (by direct adversarial source injection) to be non-sound as a
  *sole* gate — it has no `os.RemoveAll` pattern in its AST vocabulary and its `.Safe`-matching is
  purely syntactic with no data-flow tracing. That is not disqualifying here — D-07 deliberately
  keeps this phase's ratchet narrower (two structural checks: baseline-shrink allowlist,
  zero-tolerance reachability from one named function) rather than attempting the same broad,
  multi-verb destructive-operation coverage that file attempted and only partially achieved.
- `cmd/subcommand_reachability_ratchet_test.go:41` — `var updateOrphanAllowlist = flag.Bool(...)`,
  the exact `-update-<name>-allowlist` idiom to copy
- `cmd/subcommand_reachability_ratchet_test.go:1423` — `TestOrphanAllowlistOnlyShrinks`, the
  shrink-only assertion shape to copy
- `cmd/subcommand_reachability_ratchet_test.go:989-1035` — `loadOrphanAllowlist` /
  `writeOrphanAllowlist`, the read/regenerate pair to copy
- `cmd/testdata/orphan_allowlist.json` — the checked-in allowlist shape to mirror (array of
  objects; here: `{file, function, reason}` rather than `{name, reason, owner_phase}`)
- Repo-wide grep results (captured 2026-08-19, will be re-confirmed by the implementing agent):
  `store.SaveJSON("COLONY_STATE.json"` appears in `abandon_cmd.go:83`, `build_flow_cmds.go:200`,
  `codex_build.go:598`, `codex_continue_finalize.go:1036,1147` (line 1147 removed by D-06),
  `codex_colonize.go:1038`, `codex_plan.go:331,386,432`, `codex_workflow_cmds.go:589`,
  `colony_cmds.go:134,224,308`, `command_truth.go:522`, `error_cmds.go:89`, `entomb_cmd.go:133`,
  `internal_cmds.go:455`, `init_cmd.go:300`, `phase_commit.go:289`, `session_flow_cmds.go:93,210,223,358`,
  `state_extra.go:105,190`, `state_repair.go:53`, `state_cmds.go:318`, `state_load.go:63`,
  `worktree.go:220,447,857,898`. `store.AtomicWrite("COLONY_STATE.json"` appears in
  `autofix.go:106`, `state_extra.go:70`, `state_cmds.go:208,346`, `suggest_approve.go:56,75,226,265,284,436`.

### Criterion 5 — retry-exhaustion record
- `cmd/compatibility_cmds.go:386` — `runCompatibilityAutopilot`, behind the live `run` command
  (`runCompatibilityCmd`, `Use: "run"`, line 231)
- `cmd/compatibility_cmds.go:445-463` — the exact retry-then-give-up block: single 2s-delayed
  retry, then `return nil, err` with no record on the second failure
- `cmd/recovery_orchestrator.go:97-432` — `RecoveryAction`, `RecoveryContext`, `RecoveryOutcome`,
  `escalateOutcome` — the ALREADY-GOOD sibling system (worker-level recovery), the positive analog
  to model the new function's *shape* on (structured record, not a bare error), even though its
  own file needs no change
- `cmd/queen_wave_lifecycle.go:154-156` and `cmd/codex_continue_finalize.go:442-444` — both
  existing `RecoveryLogEntry` persistence call sites, proving that system's record-keeping is real
  and already wired

### Criterion 6 — legacy manifest warning
- `cmd/build_attempt.go:451-522` — `buildAttemptManifestBinding` struct (the `Legacy bool` field)
  and `validateBuildAttemptManifestBinding`; the `Legacy: true` return is at lines 461-462
- `cmd/codex_build_finalize.go:415-495` — the finalize flow that calls
  `validateBuildAttemptManifestBinding` (line 418) and never reads `.Legacy`; the warning insertion
  point is right after that call, before the `if !binding.Bound { attemptRel, err = beginBuildAttempt(...) }`
  branch at line 493
- `cmd/init_cmd.go:200-205` — the plain-language stderr-warning precedent to copy for D-11's first
  channel

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `store.UpdateJSONAtomically` (`pkg/storage/storage.go:121-137`) — read-mutate-write-atomically,
  already the correct primitive; `advancePhase()` must use it, the new `appendMiddenEntry` should
  use it too (stricter than either existing midden writer).
- `continueSupersededResult` — package-scoped, reusable verbatim from the finalize path.
- `appendRuntimeStateEventsIfCurrent` — already exists and already solves "append side-effect
  events without clobbering a currency check"; reuse for D-06 rather than inventing a second
  append helper.
- The `-update-<x>-allowlist` flag + shrink-only-ratchet pattern in
  `subcommand_reachability_ratchet_test.go` is a complete, working, three-times-proven-in-CI
  template; D-07/D-08's ratchet is a direct structural cousin, not a new invention.

### Established Patterns
- **Fail-closed on uncertainty.** `worktreeDestructionSafety` (Phase 187) and
  `validateRuntimeStateStillCurrent` (this phase's own supersession check) share the same rule:
  every "I could not determine X" branch resolves to refusal, never to permission. Any new check
  this phase adds (the ratchet's AST walker, the state-mutate guard) should fail loudly rather
  than silently pass when it cannot determine an answer.
- **Plain-language, every-occurrence reporting.** Phase 187's D-02 (`reportWorktreePreservation`
  unconditional `fmt.Fprintf(os.Stderr, ...)`) is the direct precedent for D-11's warning channel.
- **Grep-ratchet with a self-regenerating, shrink-only allowlist**, not a hand-maintained list —
  established by `subcommand_reachability_ratchet_test.go` and referenced again in the ROADMAP's
  own Phase 191 criteria ("the ratchet allowlist may only shrink").
- **A guard that finds nothing must fail, not pass.** Both `worktree_destruction_reachability_test.go`
  and `subcommand_reachability_ratchet_test.go` include this self-check; D-07's ratchet must too.

### Integration Points
- Criterion 5's plan (retry-exhaustion record) is sequenced **after** criterion 1's plan
  (midden unification) because it calls `appendMiddenEntry`, and its record becomes the
  clearest end-to-end proof that criterion 1 actually works.
- Criterion 4's plan (the ratchet) is sequenced **after** criterion 2's plan (`advancePhase()`)
  because its zero-tolerance check needs `advancePhase` to exist as a named call-graph root.
- Criteria 3 and 6 touch entirely different files (`state_cmds.go` vs. `codex_build_finalize.go`)
  from each other and from every other criterion in this phase — no ordering constraint, bundled
  into one plan purely because each is individually below the ~10% context floor for its own plan.

### Test-Coverage Reality
- The worker-level recovery system (`recovery_orchestrator.go`) is well tested and does not need
  new tests from this phase.
- `runCodexContinue`'s supersession path is presumably covered somewhere already (it is live,
  correct code); `advanceExternalContinue`'s is not covered at all today because the check does
  not exist yet — the new test proving continue-finalize now refuses on staleness is genuinely new
  coverage, not a duplicate.
- No existing test in this codebase forces `runCodexBuildWithOptions` to fail deterministically —
  confirmed by grep across `cmd/*_test.go`. See D-14 for the resulting two-tier test strategy.

</code_context>

<specifics>
## Specific Ideas

- The `TestGoldenAutopilotPauseConditions` finding (see Canonical References, Criterion 1) is the
  single most important piece of evidence in this whole context: it is proof that a *passing* test
  in this codebase can validate a code path's self-consistency while the path itself has never
  once seen real production data. Fixing it correctly (move the fixture, not the reader) is itself
  a small demonstration of exactly the property this repo's `CLAUDE.md` asks every test to have.
- `immune.go`'s `entry["description"]` vs. `MiddenEntry`'s real `"message"` JSON tag (D-01's
  canonical-ref entry) is a second, independent bug layered on top of the path bug — fixing only
  the path would leave `immune-auto-scar` still functionally blind. Both must be fixed together for
  criterion 1's "read by ... immune" to be genuinely true.
- Criterion 2's two functions have **different return signatures**
  (`runCodexContinue` returns 7 values including a bare `colony.Phase`; `advanceExternalContinue`
  returns 6, no bare `colony.Phase`) — the shared `advancePhase()` cannot be a drop-in replacement
  for either call site's full return tuple. It should return a small dedicated result struct
  (updated state, next phase pointer, next command string, final bool) that each of the two much
  larger surrounding functions adapts into its own existing return shape. Do not attempt to
  unify the two callers' full signatures — only the atomic-commit core they both currently
  duplicate.

</specifics>

<deferred>
## Deferred Ideas

- **Full COLONY_STATE.json atomicity migration** (the other ~28 non-atomic write sites named in
  D-07). This is the ROADMAP's own `LOCK` deferred-requirement category. Not started here; the
  ratchet this phase adds will make any *future* work in this direction visible and measurable
  (the baseline count it establishes), but does not perform the migration.
- **Migrating `midden_cmds.go` and `spawn_budget.go`'s existing writers onto `appendMiddenEntry`.**
  Left as Claude's Discretion / a natural follow-up, not required — see D-02.
- **Tagging the new `autopilot_retry_exhausted` midden entry as a chaos-style pause trigger.**
  Not required by criterion 5 (which asks only for a readable record); left as discretion.
- **Consolidating the two duplicate merge-gate / non-atomic-write patterns more broadly** — noted
  by Phase 187's own CONTEXT.md as a candidate for Phase 190 (Lean, Non-Duplicated Delivery); nothing
  here should be read as pre-empting that phase's scope.

</deferred>

---

## Success Criteria Coverage

| # | ROADMAP Success Criterion | Covered By |
|---|---|---|
| 1 | A failure written by any component is read by colony-prime, autopilot, immune and memory-health (midden path unified) | 188-01 |
| 2 | Both continue paths advance through one shared `advancePhase()` with the supersession check | 188-02 |
| 3 | Ungated `state-mutate --field current_phase` is refused | 188-03 |
| 4 | Grep-ratchet against non-atomic COLONY_STATE writes | 188-04 (depends on 188-02) |
| 5 | An exhausted retry loop leaves a readable record of what was tried | 188-05 (depends on 188-01) |
| 6 | A loud warning is logged when build-finalize accepts a legacy unbound manifest | 188-03 |

Every one of the six ROADMAP success criteria is covered by at least one plan. None was found to
be already satisfied in the code (the closest candidate, the worker-level recovery orchestrator
under criterion 5, is a *different* retry loop from the one the ROADMAP's own goal statement and
this phase's reconnaissance identified as unrecorded — see D-12).

---

*Phase: 188-One Truth for Failures and Advances*
*Context gathered: 2026-08-19*
