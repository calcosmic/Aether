---
phase: 200-iterative-planning
plan: 25
subsystem: command-surfaces
tags: [specification, command-wrappers, public-inventory, claude, opencode, codex, tdd]

requires:
  - phase: 200-10
    provides: Go-owned specification inspect, revision, approval, and projection-repair command
  - phase: 200-19
    provides: typed specification presentation and platform-native command spelling contract
  - phase: 200-20
    provides: settled Discuss handoff to exact specification review
provides:
  - canonical specification wrapper guidance for every Go-backed operation
  - synchronized Claude flat/nested and OpenCode /ant-spec surfaces
  - closed public inventory coverage for Discuss and Specification
  - runtime, canonical-source, managed-projection, and Codex-spelling parity enforcement
affects: [phase-200-verification, phase-201, specification, discuss, plan, source-check, public-command-inventory]

tech-stack:
  added: []
  patterns: [Go-owned mutation authority, YAML-owned wrapper guidance, immutable specification revisions, closed command inventory]

key-files:
  created:
    - .aether/commands/spec.yaml
    - .claude/commands/ant-spec.md
    - .claude/commands/ant/spec.md
    - .opencode/commands/ant/spec.md
  modified:
    - .aether/commands/classic-command-parity.json
    - cmd/classic_command_parity_test.go
    - cmd/wrapper_command_names.go

key-decisions:
  - "Specification wrappers render and submit only Go-issued operations; editor completion, file existence, generic confirmation, planning stop, and candidate readiness grant no approval."
  - "Specification approval and exact plan-candidate acceptance remain separate authority transitions with separate commands and receipts."
  - "Claude and OpenCode expose /ant-spec while Codex remains runtime-native through aether spec with no native $ant alias."

patterns-established:
  - "Specification projection parity: canonical YAML plus byte-identical Claude flat/nested and OpenCode wrappers describe the same runtime-backed routes."
  - "Closed public inventory: every listed row must resolve to public Cobra registration, canonical YAML, Claude flat/nested wrappers, and an OpenCode wrapper."

requirements-completed: [SYNTH-02, PLAN-05, PLAN-06]

duration: 8 min
completed: 2026-09-08
---

# Phase 200 Plan 25: Specification Surface Parity Summary

**A real `/ant-spec` workflow now exposes Go-owned specification review, immutable revision, exact approval, and projection repair on Claude and OpenCode, while the closed public inventory proves direct Codex and managed-wrapper truth.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-08T01:12:53Z
- **Completed:** 2026-09-08T01:21:01Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added the canonical specification command source with inspect, add, modify, remove, exact approve, and projection-repair routes backed only by `aether spec`.
- Synchronized byte-identical Claude flat, Claude nested, and OpenCode wrappers that render all nine typed specification body categories and preserve exact revision/hash bindings.
- Kept draft approval, planning stop, and exact plan-candidate acceptance visibly and operationally separate.
- Added `discuss` and `spec` to the normal public journey without changing existing command identities or classifications.
- Extended the closed inventory gate to verify public Cobra registration, canonical YAML, every managed primary-platform path, and direct Codex spelling.

## Task Commits

1. **Task 1: Create and synchronize the real specification wrapper** — `d7f1b9e9` (feat)
2. **Task 2 RED: Define public Discuss/Specification parity behavior** — `50a455f5` (test)
3. **Task 2 GREEN: Extend the public command parity inventory** — `9ca85a12` (feat)

## Files Created/Modified

- `.aether/commands/spec.yaml` — Canonical Go-authority contract for specification inspection, revision, approval, repair, impact, and platform spelling.
- `.claude/commands/ant-spec.md` — Managed flat Claude `/ant-spec` entrypoint.
- `.claude/commands/ant/spec.md` — Managed nested Claude `/ant-spec` entrypoint.
- `.opencode/commands/ant/spec.md` — Managed OpenCode `/ant-spec` entrypoint with identical semantics.
- `.aether/commands/classic-command-parity.json` — Adds Discuss and Specification to the normal public workflow and aligns Plan's already-shipped description.
- `cmd/classic_command_parity_test.go` — Enforces the 22-row closed inventory against Cobra and every canonical/managed path, including platform spellings.
- `cmd/wrapper_command_names.go` — Registers `spec` as a real slash-wrapper verb for Claude/OpenCode hint translation.

## Decisions Made

- The wrapper never parses `.aether/SPEC.md` or computes stable IDs, deltas, scope impact, tokens, receipts, or authority. It invokes structured Go results and renders them.
- All nine body categories stay separately named and ordered: outcomes, included behaviors, exclusions, binding decisions, requirements, acceptance checks, negative expectations, recovery expectations, and affected public paths.
- A specification change always produces a predecessor-linked successor draft. The exact current revision ID, full content hash, and runtime-issued token are required for approval.
- Direct Codex guidance remains `aether discuss`, `aether spec`, and `aether plan`; native `$ant-*` aliases are not advertised.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Registered Specification in the central wrapper allowlist**
- **Found during:** Task 2 GREEN
- **Issue:** The new managed wrappers existed, but `wrapperCommandNames` did not include `spec`; Claude/OpenCode Next Up translation would therefore display raw `aether spec` instead of `/ant-spec`.
- **Fix:** Added `spec` to the central slash-wrapper allowlist so runtime hints use the platform's real public spelling.
- **Files modified:** `cmd/wrapper_command_names.go`
- **Verification:** `TestWrapperCommandNamesMatchCanonicalCorpus` and `TestDiscussAndSpecRegisteredAcrossPublicSurfaces` pass.
- **Committed in:** `9ca85a12`

**2. [Rule 3 - Blocking] Reconciled the public Plan description with its canonical Phase 200 wrapper**
- **Found during:** Task 2 GREEN
- **Issue:** The closed parity test exposed a pre-existing mismatch: Plan's canonical YAML already described the evidence-backed candidate flow, while the inventory and expected row retained the earlier depth-scoped wording.
- **Fix:** Updated only Plan's description in the inventory and expected row; its public name, Cobra name, ordering, and `normal` classification remain unchanged.
- **Files modified:** `.aether/commands/classic-command-parity.json`, `cmd/classic_command_parity_test.go`
- **Verification:** `TestClassicCommandParity` passes across all 22 rows.
- **Committed in:** `9ca85a12`

---

**Total deviations:** 2 auto-fixed (1 missing critical functionality, 1 blocking parity drift).
**Impact on plan:** Both fixes were necessary to make the declared cross-platform inventory truthful; no new runtime authority or unrelated feature was added.

## Issues Encountered

- The TDD RED run failed exactly as intended: the 20-row manifest was two commands short and specifically lacked `discuss`.
- Commit hooks completed successfully and reported only their existing package-validation warning summary.

## TDD Gate Compliance

- **RED:** `50a455f5` added the missing-command and cross-platform assertions; the focused run reported one passing test and two expected failures (`20, want 22` and missing `discuss`).
- **GREEN:** `9ca85a12` followed RED and made all four focused parity/wrapper-name tests pass.
- **REFACTOR:** No separate refactor was needed; the minimal inventory and allowlist changes satisfied the contract.

## Verification

- `go test ./cmd -run 'Test.*Spec.*Wrapper|TestSpecCommand' -count=1` — 10 passed after Task 1.
- TDD RED `go test ./cmd -run 'TestClassicCommandParity|Test.*Spec.*Registered' -count=1` — failed for the intended missing inventory behavior.
- GREEN `go test ./cmd -run 'TestClassicCommandParity|Test.*Spec.*Registered|TestWrapperCommandNamesMatchCanonicalCorpus' -count=1` — 4 passed.
- Final `go test ./cmd -run 'Test.*Spec.*Wrapper|TestClassicCommandParity|Test.*Spec.*Registered' -count=1` — 3 passed.
- `go run ./cmd/aether source-check` — 16 canonical sources, 5 retired mirrors, and 126 generated wrappers passed with no findings.
- `git diff --check` — passed before task commits.
- Stub scan across all seven plan files — no functional stubs; matches were ordinary Go empty-value checks only.

## Known Stubs

None.

## User Setup Required

None - no external service or local configuration is required.

## Next Phase Readiness

- The public Discuss → Specification → Planning path now has real, synchronized primary-platform entrypoints and direct Codex spelling.
- Phase 200 verification can rely on a closed 22-command inventory rather than documentation-only claims.
- No unresolved Plan 200-25 blocker remains; numerically earlier incomplete plans should retain the execution pointer if still pending.

## Self-Check: PASSED

- All seven implementation/test/wrapper files and this summary exist.
- Task commits `d7f1b9e9`, `50a455f5`, and `9ca85a12` are present in repository history.
- Final focused tests, source-check, and whitespace validation pass.

---
*Phase: 200-iterative-planning*
*Completed: 2026-09-08*
