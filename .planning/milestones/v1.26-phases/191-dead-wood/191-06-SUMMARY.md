---
phase: 191-dead-wood
plan: 06
subsystem: docs
tags: [documentation, wisdom-pipeline, token-budget, command-playbooks, pkg-learn, host-manifest]

# Dependency graph
requires: []
provides:
  - "workers.md:825's build-work-observation-capture claim traced to its real, cited call site (captureContinueLearning, not memory-capture)"
  - "CLAUDE.md's Trim order list matching cmd/context.go's real 9-tier trimOrder exactly, including both QUEEN wisdom tiers"
  - "CLAUDE.md's Command Playbooks section distinguishing the interactive wrapper's real flow from the separate autopilot/host-driven flow"
affects: [future-CLAUDE.md-edits, future-workers.md-edits, any-phase-relying-on-wisdom-pipeline-or-token-budget-docs]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Doc correction traced to a live call site (file:line), never restated from a prior summary or the ROADMAP's own characterization"

key-files:
  created: []
  modified:
    - .aether/workers.md
    - CLAUDE.md

key-decisions:
  - "workers.md:825 now cites pkg/learn's captureContinueLearning() (cmd/codex_continue_finalize.go:1458), called from both continue paths, as the real automatic capture mechanism -- not a wrapper-invoked memory-capture call"
  - "CLAUDE.md's Trim order list corrected to 9 numbered tiers (splitting QUEEN wisdom into global/local) plus an unnumbered Blockers-never-trimmed note, matching cmd/context.go:1027-1038 exactly"
  - "CLAUDE.md's Command Playbooks section now names two distinct flows: the interactive wrapper (aether build --plan-only -> wrapper spawn -> build-finalize, host-free) and the separate autopilot/host-driven lane (aether host build -> dispatch_manifest -> build-finalize), plus continue.md's own different (dry-run-only, opt-in) host relationship"

patterns-established:
  - "When a doc's Wisdom Pipeline / trim-order / flow claim is corrected, cite the exact file:line the correction is traced to inline in the doc itself, not just in the SUMMARY"

requirements-completed: []

# Metrics
duration: ~20min
completed: 2026-08-21
---

# Phase 191 Plan 06: Correct False Documentation Claims Summary

**Three false claims in `.aether/workers.md` and `CLAUDE.md` corrected, each traced to a live, cited Go call site rather than restated from the ROADMAP's own characterization.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-21T01:04:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Traced `workers.md:825`'s "build work produces observations via `memory-capture` in continue step" claim to its real mechanism: `pkg/learn`'s own in-process `captureContinueLearning()` (`cmd/codex_continue_finalize.go:1458`), invoked from both continue paths (`cmd/codex_continue.go:962` default, `cmd/codex_continue_finalize.go:521` heavy-review) — no `memory-capture` CLI call involved at all.
- Corrected CLAUDE.md's "Trim order" list to match `cmd/context.go:1027-1038`'s real `trimOrder` slice exactly: split the single "QUEEN.md wisdom" entry into "QUEEN wisdom (global)" then "QUEEN wisdom (local)" as two separately-prioritized tiers, and restated Blockers as a never-trimmed exception rather than a false 9th trimmed item.
- Corrected CLAUDE.md's "Command Playbooks" section, which conflated the autopilot-only host-manifest flow with what the interactive wrapper documents. Now names both flows distinctly, and states continue.md's own different (dry-run-only, opt-in) relationship to the host rather than treating it identically to build.md.

## Task Commits

Each task was committed atomically:

1. **Task 1: Trace and correct workers.md:825's memory-capture claim** - `8199f7b0` (docs)
2. **Task 2: Correct CLAUDE.md's Trim order list and Command Playbooks host-build claim** - `0d81f04b` (docs)

## Files Created/Modified

- `.aether/workers.md` - Bullet 1 of the Wisdom Pipeline section rewritten to cite the real automatic capture mechanism
- `CLAUDE.md` - "Command Playbooks (Reference Material)" section rewritten to name two distinct execution flows; "Trim order" list rewritten to match the real 9-tier `trimOrder` slice

## Source Traces (Proof)

### Correction 1 — `.aether/workers.md:825`

**Before:**
```
1. **Build work** produces observations (via `memory-capture` in continue step)
```

**After:**
```
1. **Build work** produces observations automatically during continue -- not via a wrapper-invoked
   `memory-capture` call, but through `pkg/learn`'s own in-process capture, `captureContinueLearning()`
   (`cmd/codex_continue_finalize.go:1458`), called from both continue paths (`cmd/codex_continue.go:962`,
   the default path, and `cmd/codex_continue_finalize.go:521`, the heavy-review path). Eligible runs are
   recorded as a hypothesis entry in `pkg/learn`'s own colony store (`.aether/data/learn/`), then handed
   to phase-end consolidation (`runPhaseEndConsolidation`, same call site) for further curation.
```

**Trace, in order:**
1. `grep -rn "memory-capture"` across the repo (excluding stale worktree copies and `.planning/`) found zero hits in `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md`, or any `.claude/agents/ant/*.md` — confirming no wrapper or agent prompt instructs calling it.
2. `cmd/learning.go:212-257` (`memoryCaptureCmd`) confirms `memory-capture` is a real, separately-registered CLI command calling `learn.NewObservationService(store, bus).CaptureWithTrust(...)` — a standalone, human/worker-invocable command with no automatic caller anywhere.
3. `cmd/codex_continue_finalize.go:1448-1458`'s own doc comment on `captureContinueLearning` states directly: *"the single durable-learning capture point for BOTH continue paths... previously lived as a closure inside `runCodexContinueFinalizeExternal` — a command the continue wrapper explicitly forbids on the default fast path — which meant learning had never once fired in normal daily use."*
4. `grep -n "captureContinueLearning" cmd/*.go` confirms exactly two production call sites: `cmd/codex_continue.go:962` (comment: *"Durable learning capture on the DEFAULT path... the fix for 'the colony never learns'"*) and `cmd/codex_continue_finalize.go:521` (comment: *"Shared with the default continue path"*).
5. `captureContinueLearning`'s body (lines 1458-1547) calls `learn.IsLearningEligible`, `learn.CollectEvidence`, `learn.ClassifyEntry`, and `learn.NewColonyStore(store).Add(entry)` directly, in-process — no shell-out, no CLI invocation of any kind.

**Related finding, logged not fixed (in scope boundary per the plan's own instruction):** Bullets 2-5 of workers.md's Wisdom Pipeline section ("Observations auto-promote to instincts after threshold... Instincts stored in COLONY_STATE.json... Hive Brain...") describe the *other* pipeline — the one `memory-capture` -> `ObservationService.CaptureWithTrust` -> `trust-score-compute` -> `learning-promote-auto` -> `instinct-create` -> `queen-promote` -> `hive-promote` feeds. `captureContinueLearning`'s entries use `learn.StatusHypothesis` and transition to `StatusValidated`/`StatusDisproven` within `pkg/learn`'s own `ColonyStore` (`cmd/learning_cmds.go:496-596`), a structurally separate lifecycle from the Observation/Instinct model bullets 2-5 describe — no code path connecting a Hypothesis entry directly into `instinct-create`'s COLONY_STATE.json instincts was found during this trace (their possible link, if any, runs through `runPhaseEndConsolidation`, fired at the identical call site immediately after `captureContinueLearning` on the default path per `cmd/codex_continue.go:969` — full tracing of that function was not undertaken, out of this task's scope). ROADMAP criterion 4 named only the line-825 trigger-mechanism claim for correction; bullets 2-5's broader pipeline-cohesion question is logged here per the plan's explicit "note, don't silently expand scope" instruction, not corrected in this task.

**Related finding #2, logged not fixed:** CLAUDE.md's own "Wisdom Pipeline" stage table (`CLAUDE.md` "Wisdom Pipeline" section, Stage 1: "Observe | `memory-capture \"learning\"` | Records observation...") documents the `memory-capture`/`ObservationService` pipeline generically, without asserting *when* or *by what trigger* it fires for build work specifically — it does not repeat workers.md's precise false claim ("via memory-capture in continue step"), so it was not edited in this task per the plan's explicit scope boundary (correct only if it "shares the exact false claim being corrected here").

### Correction 2 — CLAUDE.md's Trim order list

**Before:**
```
**Trim order** (first trimmed = lowest retention priority):
1. Rolling summary (trimmed first -- lowest retention priority)
2. Phase learnings
3. Key decisions
4. Hive wisdom
5. Context capsule
6. User preferences
7. QUEEN.md wisdom
8. Pheromone signals (trimmed last -- highest retention priority)
9. Blockers (NEVER trimmed)
```

**After:**
```
**Trim order** (first trimmed = lowest retention priority):
1. Rolling summary (trimmed first -- lowest retention priority)
2. Phase learnings
3. Key decisions
4. Hive wisdom
5. Context capsule
6. User preferences
7. QUEEN wisdom (global)
8. QUEEN wisdom (local)
9. Pheromone signals / active signals (trimmed last -- highest retention priority)

Blockers are NEVER trimmed, regardless of budget.
```

**Source proof:** `cmd/context.go:1027-1038`, read directly:
```go
trimOrder := []string{
    "--- ROLLING SUMMARY ---",
    "--- PHASE LEARNINGS ---",
    "--- KEY DECISIONS ---",
    "--- HIVE WISDOM",
    "--- CONTEXT CAPSULE ---",
    "--- USER PREFERENCES ---",
    "--- QUEEN WISDOM (Global) ---",
    "--- QUEEN WISDOM (Local) ---",
    "--- ACTIVE SIGNALS",
    // BLOCKERS is NEVER trimmed
}
```
9 real trim-priority entries with QUEEN wisdom split into two separately-prioritized tiers (Global before Local), matching the corrected doc 1:1. The old doc's item 9 ("Blockers (NEVER trimmed)") was self-contradictory as written (labeled both a numbered trim-order item AND "never trimmed"); the code confirms Blockers is not in `trimOrder` at all, only a code comment beside it — now reflected as an unnumbered closing note, not a 9th/10th trim-priority rank.

### Correction 3 — CLAUDE.md's Command Playbooks section

**Before:**
```
`.aether/docs/command-playbooks/*.md` are reference documentation for the
build/continue methodology. Since v1.25 they are NOT loaded by the runtime and
NOT injected into worker or orchestrator prompts — execution behavior lives in
the host-manifest flow (`aether host build` → spawn from `dispatch_manifest` →
`build-finalize`), and workers receive the lean runtime-composed brief.

Authority note:
- `.claude/commands/ant/build.md` and `continue.md` describe the host-manifest
  flow directly; there is no playbook indirection.
```

**After:** names two distinct flows (interactive wrapper: `aether build --plan-only` -> wrapper spawn -> `aether build-finalize`; autopilot/host-driven: `aether host build` -> `dispatch_manifest` -> `build-finalize`, reached only via `aether run`), and corrects the Authority note to state that `build.md` describes the plan-only/wrapper-driven flow (never touching the TS host) while `continue.md`'s relationship to the host is different again (default path never touches it; opt-in heavy-review path reaches `aether host continue --dry-run` only, with the wrapper still spawning reviewers).

**Source proof:**
- `.claude/commands/ant/build.md:348`: *"Do NOT run `aether host build` from this wrapper; the TS host hop is off the interactive build path. Fetch the manifest with `aether build $ARGUMENTS --plan-only` and never run a command that dispatches workers itself."*
- `.claude/commands/ant/build.md:105-108, 253, 295`: confirms the real flow — `aether build $ARGUMENTS --plan-only` for the manifest (line 108), wrapper-driven spawn (line 253: *"The wrapper spawns workers. The runtime does NOT dispatch workers in plan-only mode."*), `aether build-finalize` for completion (line 295).
- `cmd/host_cmd.go:55-59`: confirms `aether host build <phase>` is a real, separately-registered subcommand (`RunE: makeHostSubcommand("build", true)`), distinct from `aether build --plan-only`.
- `.aether/ts-host/src/host.ts:1064-1097, 1200` and `lifecycle.ts:353-459`: confirms the TS host's own internal flow — it calls Go's `aether build --plan-only` to obtain `dispatch_manifest`, dispatches itself, then calls `build-finalize` — matching the shape the old CLAUDE.md text described, but as the **autopilot-only** lane, not what the interactive wrapper does.
- `.claude/commands/ant/continue.md:15, 32-40`: *"Manifest generation | TS host (`aether host continue --dry-run`) for heavy review"* and *"No reviewer spawning happens on this [default] path — verification runs inside the runtime process itself."* — confirms continue's default path never touches the host at all, and its host touch point exists only for the opt-in heavy-review path.
- `.claude/commands/ant/continue.md:264`: *"Do NOT run `aether host continue --classic-ceremony` without `--dry-run` from this wrapper; that triggers the TS host dispatched path which duplicates the wrapper's own reviewer spawning. Always use `aether host continue --dry-run`."* — confirms continue's host touch point is dry-run only, with the wrapper still performing the actual reviewer spawning (unlike build's host lane, where the host dispatches itself).

## Decisions Made

- Scoped workers.md's correction to bullet 1 (the named false claim) only; bullets 2-5's pipeline-cohesion question and CLAUDE.md's own Wisdom Pipeline table are logged as related-but-out-of-scope findings above, per the plan's explicit "note, don't silently expand scope" instruction (matching threat T-191-06-02's mitigation).
- Confirmed via `scripts/version-sync.sh` that neither the Token Budget section nor the Command Playbooks section falls inside a version-sync-managed region (which only rewrites the header version line, the Quick Reference table's Version row, and the footer date) — both corrections were safe to edit directly with no conflict.
- Did not edit CLAUDE.md's owner-communication top block ("READ THIS BEFORE YOU WRITE ANYTHING TO THE OWNER") — confirmed unchanged after edits (still at line 6).

## Deviations from Plan

None - plan executed exactly as written. Both named corrections were traced to live call sites before any edit was made; no architectural changes, no bug fixes, no blocking issues encountered.

## Issues Encountered

None. The worktree's base was stale on spawn (HEAD was several phases behind at `6577f51c`, not the expected `916db8fa` "docs(phase-191): begin phase"); corrected via the mandatory `<worktree_branch_check>` hard reset before any plan work began, per the documented recovery pattern for this repo's GSD worktree stale-base issue.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Both named CLAUDE.md claims and the workers.md claim now match live source exactly, closing ROADMAP Phase 191 criterion 4 for this plan's scope.
- Two related-but-out-of-scope findings (CLAUDE.md's Wisdom Pipeline table's generic `memory-capture` framing; workers.md bullets 2-5 describing a structurally different pkg/learn pipeline than the corrected bullet 1) are recorded above as candidates for a future ruling — not blockers for 191-07's final verification.
- 191-07 (final verification, Wave 2) can proceed once all Wave-1 plans land; this plan touched only `.aether/workers.md` and `CLAUDE.md`, with zero file overlap with any sibling plan.

## Self-Check: PASSED

- FOUND: `.planning/phases/191-dead-wood/191-06-SUMMARY.md`
- FOUND: commit `8199f7b0` (Task 1: workers.md correction)
- FOUND: commit `0d81f04b` (Task 2: CLAUDE.md corrections)

---
*Phase: 191-dead-wood*
*Completed: 2026-08-21*
