---
phase: 165-core-lifecycle-commands
plan: 05
subsystem: docs
tags: [go-testing, wrapper-contracts, ceremony-restoration, init-command]

# Dependency graph
requires:
  - phase: 165-core-lifecycle-commands
    plan: 01
    provides: shared lifecycle wrapper test toolkit (canonicalWrapperPaths, assertOrderedHeadingParity, assertStageSkeletonDensity, countMarkerLines, envelopeMechanicsMarkers) and the wrapper-host-contract.md manifest-shape documentation
provides:
  - "cmd/init_wrapper_ceremony_test.go — init.md's first dedicated ceremony contract test (TestInitWrapperCeremonyContract, TestInitWrapperStageSkeletonAndParity)"
  - "Rewritten .claude/commands/ant/init.md and .opencode/commands/ant/init.md carrying the D-05 stage skeleton, success_criteria/failure_modes/read_only blocks, and the restored 👑 intention beat"
  - "Resynced .claude/commands/ant-init.md flat mirror"
  - "Updated .aether/commands/init.yaml with wrapper_additions (stage_skeleton, intention_beat) and two new guardrail bullets"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "init.md keeps its own nine stage names (D-05 permits this) but takes the shared skeleton shape: colony beat line, Purpose, Reads, imperative instructions, Stop conditions — with Spawns omitted throughout since init spawns no workers"
    - "A literal '## Heading' string inside an illustrative fenced code block is parsed by assertStageSkeletonDensity as a real level-2 heading; example blocks must avoid the '## ' prefix to not create a false section boundary"
    - "A backtick-quoted CLI invocation missing its required positional argument (e.g. `aether init --colony-mode --charter-json` with no goal) is flagged by TestCommandCallsMatchCobraContracts even inside YAML guardrail prose; write such mentions as a partial reference (`aether init` alone, with flags named separately) instead of a copy-pasted-looking full invocation"

key-files:
  created:
    - cmd/init_wrapper_ceremony_test.go
  modified:
    - .claude/commands/ant/init.md
    - .opencode/commands/ant/init.md
    - .claude/commands/ant-init.md
    - .aether/commands/init.yaml

key-decisions:
  - "The 👑 intention beat fires at Approval, after user consent, immediately before the runtime `aether init` call — never before, per threat T-165-05-03 (spoofing colony state as pre-existing consent)"
  - "init_ceremony.go retains sole ownership of the init banner; the wrapper narrates the intention moment in prose only, with zero separator/banner rendering (`grep -cE '━━━|────'` returns 0)"
  - "All eight non-exempt stage headings (Codebase Summary, Prior Context, Intent Refinement, Colony Charter, Colony Mode, Pheromone Suggestions, Shelf Backlog, Approval) got the full skeleton; Required Cross-Stage State and Cross-Platform Drift Guard stayed exempt per the plan's explicit exempt-heading list"

patterns-established:
  - "init.md is the one lifecycle wrapper permitted exactly one platform-specific line (AskUserQuestion vs. plain Ask); TestInitWrapperStageSkeletonAndParity/platform_difference_is_the_sanctioned_one pins that delta at exactly one line so any second divergence requires a deliberate decision, not a silent edit"

requirements-completed: [CMD-01, CMD-02]

# Metrics
duration: 45min
completed: 2026-08-03
---

# Phase 165 Plan 05: Init Command Ceremony Restoration Summary

**Rewrote init.md on both platforms onto the uniform D-05 stage skeleton, restored the Classic 👑 intention-setting beat at approval in current `refined_goal` vocabulary, and gave init.md its first dedicated ceremony contract test — while keeping every byte of persistence behind the runtime.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-03
- **Tasks:** 3
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `cmd/init_wrapper_ceremony_test.go` gives init.md a dedicated ceremony contract test for the first time — `TestInitWrapperCeremonyContract` (required/ordered/forbidden substring assertions) and `TestInitWrapperStageSkeletonAndParity` (stage-skeleton density, ordered-heading parity, zero-envelope-parsing-prose, and the sanctioned single-platform-delta check)
- Both `.claude/commands/ant/init.md` and `.opencode/commands/ant/init.md` now carry `## Required Cross-Stage State`, `<success_criteria>`, `<failure_modes>`, and `<read_only>` blocks, plus the D-05 stage skeleton (colony beat, Purpose, Reads, Stop conditions) across all eight non-exempt stages
- The Classic 👑 intention beat is back: `## Approval` now says "Queen has set the colony's intention" followed by the approved `refined_goal`, at the moment of consent, before the runtime `aether init` call
- init.md's CMD-02 obligation (zero envelope-parsing prose) holds — init has no host-manifest step and needs none, confirmed by `no_envelope_parsing_prose` staying green
- The CRITICAL D-2 Frankenstein-state fence (jq state reconstruction, hand-written `COLONY_STATE.json`) is explicitly named and asserted absent, with `aether state-write` fenced pre-emptively even though nothing in the current file names it
- The single sanctioned platform delta (AskUserQuestion vs. plain Ask) is pinned by a dedicated subtest so any future second divergence requires a deliberate decision
- The flat installed mirror `.claude/commands/ant-init.md` is resynced byte-for-byte
- `.aether/commands/init.yaml` documents the stage skeleton and intention beat as `wrapper_additions`, and gained two new guardrail bullets, with every pre-existing key preserved

## Task Commits

Each task was committed atomically:

1. **Task 1: Create cmd/init_wrapper_ceremony_test.go** - `5f46a530` (test) — RED for `TestInitWrapperCeremonyContract` and `stage_skeleton_density`, GREEN for the other three subtests, exactly as the plan specified
2. **Task 2: Rewrite init.md on both platforms** - `212c6811` (feat) — turns the RED assertions GREEN
3. **Task 3: Resync flat mirror and update init.yaml** - `7dcc264e` (fix) — includes an unplanned but necessary fix (see Deviations)

## Files Created/Modified

- `cmd/init_wrapper_ceremony_test.go` - New: `TestInitWrapperCeremonyContract` (required/ordered/forbidden lists) and `TestInitWrapperStageSkeletonAndParity` (4 subtests: `stage_skeleton_density`, `ordered_heading_parity`, `no_envelope_parsing_prose`, `platform_difference_is_the_sanctioned_one`)
- `.claude/commands/ant/init.md` - Rewritten with `## Required Cross-Stage State`, `<success_criteria>`/`<failure_modes>`/`<read_only>` blocks, D-05 skeleton on all 8 non-exempt stages, and the restored 👑 intention beat at Approval
- `.opencode/commands/ant/init.md` - Same rewrite, differing from the `.claude` version on exactly one line (the ask-tool wording)
- `.claude/commands/ant-init.md` - Resynced to exact byte content of `.claude/commands/ant/init.md`
- `.aether/commands/init.yaml` - Added `wrapper_additions` (`stage_skeleton`, `intention_beat`) and two new `guardrails` entries; all pre-existing keys (`name`, `description`, `source_of_truth`, `runtime`, `codex_orchestration`, `ceremony_contract`, `intent_refinement`) untouched

## Decisions Made

- Placed `<success_criteria>`, `<failure_modes>`, `<read_only>` immediately after the opening prose and before `## Required Cross-Stage State`, mirroring the ordering convention already used in `.claude/commands/ant/lay-eggs.md`
- Wrote one short colony beat line per non-exempt stage (🐜 for the eight regular stages, 👑 for Approval specifically) rather than inventing a new glyph convention, per "Keep 🐜 and 👑 glyphs for beats"
- Used the illustrative `Prior Context — {count} archived colonies` example block without a literal `## ` prefix, after discovering the proportion test parses any `## `-prefixed line — including ones inside example fences — as a real section heading (see Deviations)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Example block's literal `## ` line broke the stage-skeleton density test**
- **Found during:** Task 2, first verification pass
- **Issue:** The `## Prior Context` section's illustrative code-block example contained a literal line `## Prior Context — {count} archived colonies` (carried over from the original file). `assertStageSkeletonDensity` treats any line starting with `## ` as a real section boundary, comment-stripping only — not fence-aware — so this created a phantom section with no `**Purpose:**` line, failing `stage_skeleton_density` on both platform files.
- **Fix:** Changed the example line to `Prior Context — {count} archived colonies` (dropped the `## ` prefix) in both `.claude/commands/ant/init.md` and `.opencode/commands/ant/init.md`. No semantic content changed — it is still clearly an illustrative block header inside the fence.
- **Files modified:** `.claude/commands/ant/init.md`, `.opencode/commands/ant/init.md`
- **Commit:** `212c6811`

**2. [Rule 1 - Bug] New init.yaml guardrail bullet failed the CLI contract audit**
- **Found during:** Task 3, full `go test ./cmd/...` verification run
- **Issue:** The new guardrail bullet "All persistence goes through `aether init --colony-mode --charter-json` and `aether pheromone-write`..." backtick-quoted a partial `aether init` invocation with flags but no required positional goal argument. `TestCommandCallsMatchCobraContracts` (added in an earlier phase to catch exactly this class of drift) correctly flagged it: cobra's own `Args` validator on `init` expects 1 positional argument, received 0.
- **Fix:** Reworded the bullet to `` "All persistence goes through the `aether init` call in Approval (with `--colony-mode` and `--charter-json`) and `aether pheromone-write`; the wrapper never writes state files." `` — the bare `` `aether init` `` backtick span with zero args, outside a fence, is correctly treated by the audit as a naming reference rather than an invocation, so it is exempt from the positional-argument check. The actual full invocation with its positional goal argument was already correct and untouched elsewhere in `init.md`.
- **Files modified:** `.aether/commands/init.yaml`
- **Commit:** `7dcc264e`

## Issues Encountered

The full `go test ./cmd/...` run took ~5-6.5 minutes per invocation (405s first run, 324s second run after the fix), consistent with the other Wave 2 executor's noted test economy constraints. Ran in the background and awaited via notification both times rather than blocking on a foreground call with the default 120s timeout.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- init.md now has parity with the other lifecycle wrappers' method-carrying pattern (Required Cross-Stage State, success_criteria/failure_modes/read_only blocks, D-05 skeleton) established by this and the sibling Wave 2 plans (165-02, 165-03, 165-04)
- `TestInitWrapperCeremonyContract` and `TestInitWrapperStageSkeletonAndParity` are permanent regression guards — any future edit that reintroduces state-write vocabulary, drops the intention beat, or lets a second platform delta slip in will fail loudly
- `TestLifecycleFlatMirrorsMatchCanonical` (from Plan 01) now also passes for the `init` verb with this plan's changes included
- No blockers identified for Wave 3 (the CMD-02 proportion test across all four wrappers, per 165-06)

---
*Phase: 165-core-lifecycle-commands*
*Completed: 2026-08-03*

## Self-Check: PASSED

- FOUND: `cmd/init_wrapper_ceremony_test.go`
- FOUND: `.claude/commands/ant/init.md`
- FOUND: `.opencode/commands/ant/init.md`
- FOUND: `.claude/commands/ant-init.md`
- FOUND: `.aether/commands/init.yaml`
- FOUND: `.planning/phases/165-core-lifecycle-commands/165-05-SUMMARY.md`
- FOUND: commit `5f46a530` (Task 1)
- FOUND: commit `212c6811` (Task 2)
- FOUND: commit `7dcc264e` (Task 3)
- FOUND: commit `063cf291` (SUMMARY)
