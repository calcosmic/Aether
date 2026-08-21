---
phase: 162-switch-on-learning
plan: 05
subsystem: testing

# Dependency graph
requires:
  - phase: 162-01
    provides: "AETHER_HIVE_POLICY defaulting to 'promote', consent gate deleted -- so hive wisdom reaches a worker brief with zero configuration"
  - phase: 162-02
    provides: "consolidation promotion target resolving to repo-local .aether/QUEEN.md's '## Instincts' section, injected into worker context"
provides:
  - "cmd/memory_injection_test.go -- TestWorkerBriefMemoryInjection (populated-vs-wiped sentinel + length-invariant comparison across instincts.json / .aether/QUEEN.md / hub hive/wisdom.json) and TestResolveCodexWorkerContextCarriesMemory (same proof through the real resolveCodexWorkerContext() dispatch funnel), a permanent model-free CI guard for LEARN-05"
  - ".planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md -- the one-time recorded real before/after exhibit from the actual compiled aether binary against a scratch colony"
affects: [162-06-decision-record]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-layer proof for a claim about runtime behavior: a permanent, deterministic, model-free unit test (Layer 1) that lives in CI forever, plus one recorded real-command exhibit (Layer 2) captured once and committed as an artifact -- neither layer alone would have satisfied 'measured, not assumed'"
    - "When a dev repo has no live self-hosted colony to exercise for a real-command exhibit, use the actual compiled binary against an isolated scratch colony addressed via the runtime's own COLONY_DATA_DIR/AETHER_HUB_DIR env overrides, built up entirely through sanctioned CLI subcommands (aether init/plan --synthetic/instinct-create/queen-write-learnings/hive-store) rather than direct file writes -- proves the real code path without touching the dev repo's own tracked or gitignored state"

key-files:
  created:
    - cmd/memory_injection_test.go
    - .planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md
  modified: []

key-decisions:
  - "Layer 2's real before/after exercise ran against a scratch colony in /tmp, not this repo's own .aether/data -- self-hosting was retired in v1.11 and this worktree had no COLONY_STATE.json or instincts.json to move aside. Documented the substitution explicitly in 162-MEMORY-PROOF.md rather than fabricating colony state directly inside this repo's tracked/gitignored paths"
  - "Layer 2 only moved instincts.json and .aether/QUEEN.md aside, per Task 2's literal wording -- hive wisdom.json was left populated, so the exhibit shows Active Instincts and LOCAL QUEEN WISDOM disappearing while HIVE WISDOM stays present and byte-identical, which is itself informative (the two files named in the task are exactly the two sections that vanish)"
  - "Verified acceptance criteria mechanically, not just by reading: temporarily broke the populated subtest by skipping the instincts.json write (test failed as required), separately emptied the seeded hive entries (test failed as required), then restored the test file to its committed state both times"

patterns-established: []

requirements-completed: [LEARN-05]

# Metrics
duration: ~55min
completed: 2026-08-04
---

# Phase 162 Plan 05: Switch On Learning — Memory Injection Proof Summary

**A permanent CI test and a one-time recorded real-command exhibit both prove that populated colony memory (instincts, local wisdom, hive wisdom) demonstrably changes a worker's assembled brief, closing LEARN-05 after four prior milestones declared the learning pipeline "restored" without ever measuring it.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-08-04T14:07:00Z (approx, per worktree base commit timestamp)
- **Completed:** 2026-08-04
- **Tasks:** 2 (Task 1 auto/tdd, Task 2 checkpoint:human-verify)
- **Files modified:** 2 created, 1 requirements doc updated

## Accomplishments
- Layer 1: `TestWorkerBriefMemoryInjection` proves, deterministically and with no model calls, that a populated colony (real instinct, real local QUEEN.md wisdom, real hub hive wisdom) produces a strictly longer, sentinel-bearing `buildColonyPrimeOutput(true).PromptSection` than an identical wiped colony — and that the hive wisdom arrives with `AETHER_HIVE_POLICY` completely unset, confirming Phase 162-01's default-to-promote flip actually took effect.
- Layer 1 also adds `TestResolveCodexWorkerContextCarriesMemory`, proving the same thing through `resolveCodexWorkerContext()` — the single funnel every real dispatch path calls — not just the lower-level assembly function.
- Layer 2: `.planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md` records a real `aether build 1 --print-brief --full` run (actual compiled binary, actual CLI subcommands, no internal Go calls) with memory populated versus the same command after `instincts.json` and `.aether/QUEEN.md` were moved aside — the assembled Context Capsule measurably shrank from 958 to 615 characters, losing exactly the two sections backed by the moved files, and returned to exactly 958 characters after restoration.
- Both acceptance-criteria "does deleting the fixture actually break the test" checks were run by hand: skipping the `instincts.json` write failed the populated subtest; emptying the seeded hive entries failed it too. This is the mechanical proof, not just an assertion, that the test does what it claims.

## Task Commits

Each task was committed atomically:

1. **Task 1: Layer 1 — a permanent deterministic populated-versus-wiped brief test** - `41dc5bd1` (test)
2. **Task 2: Layer 2 — record the one-time real before/after exhibit** - `ca5dd40d` (docs)

**Plan metadata:** (this commit) - docs: complete plan

## Files Created/Modified
- `cmd/memory_injection_test.go` - `TestWorkerBriefMemoryInjection` (populated/wiped subtests + length invariant) and `TestResolveCodexWorkerContextCarriesMemory`
- `.planning/phases/162-switch-on-learning/162-MEMORY-PROOF.md` - recorded real before/after exhibit with exact commands, four raw outputs, a diff table, and a restoration confirmation
- `.planning/REQUIREMENTS.md` - LEARN-05 checkbox and traceability table row marked complete

## Decisions Made
- Used a scratch colony (`/tmp/aether-memory-proof/scratch`, addressed via `COLONY_DATA_DIR`/`AETHER_HUB_DIR`) for the Layer 2 real-command exercise instead of this repo's own `.aether/data`, since this repo currently runs no self-hosted colony (self-hosting was retired in v1.11) and there was nothing here to populate or move aside. Every write to the scratch colony went through the real `aether` CLI (`init`, `plan --synthetic`, `instinct-create`, `queen-write-learnings`, `hive-init`, `hive-store`) — never a direct file write — matching the project's own rule that colony state is written through the CLI, not hand-edited.
- Moved only the two files Task 2 names (`instincts.json`, `.aether/QUEEN.md`) aside for the wiped run, leaving hive wisdom populated; the resulting diff cleanly isolates which sections come from which file.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' acceptance criteria were met without needing an architectural change, a scope addition, or a blocking-issue workaround. The one adaptation (scratch colony instead of this repo's live state for Layer 2) was anticipated by Task 2's own instructions being written generically ("in this repo") without assuming a self-hosted colony exists here, and is documented transparently in the exhibit itself rather than treated as a silent substitution.

## Issues Encountered

- `hive-store` initially rejected the wisdom text as "not admissible" (an anti-hallucination gate requiring cited text to name a real file, command, or error). Reworded the seeded hive wisdom text to reference the actual `aether build --print-brief` command and `cmd/build_print_brief.go`, which passed admissibility and remained a plausible, real piece of wisdom rather than an arbitrary sentinel string.
- The Write tool's protected-path hook blocks any write to a path containing `.aether/data/*` or `.aether/QUEEN.md`-shaped segments regardless of whether the target is inside this worktree or a `/tmp` scratch directory. Worked around this by using the real `aether` CLI subcommands to populate the scratch colony (the sanctioned path) instead of direct file writes — this is also a more faithful proof, since it exercises the actual write path a real colony uses.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

LEARN-05 is closed: the learning loop's effect on a worker's brief is now measured by a permanent CI test and documented by a real recorded exhibit, not merely asserted. Remaining open LEARN requirements (LEARN-03: reconcile the two competing learning systems; LEARN-04: decide the Hive Brain default deliberately — largely addressed by 162-01's default flip but not yet formally closed as its own requirement) are tracked separately in `.planning/REQUIREMENTS.md` and are not blocked by this plan. Phase 162 plan 06 (decision record) can now cite this plan's artifacts as evidence.

---
*Phase: 162-switch-on-learning*
*Completed: 2026-08-04*
