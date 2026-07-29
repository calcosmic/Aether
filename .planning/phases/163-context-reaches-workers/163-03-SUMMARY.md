---
phase: 163-context-reaches-workers
plan: 03
subsystem: security
tags: [go, permissions, jsonschema, claude-hooks, codex, opencode, docs]

# Dependency graph
requires:
  - phase: 163 (waves 1-2)
    provides: brief/manifest context plumbing this plan's permission fix rides alongside
provides:
  - Exact-subpath write allowlist (`sanctionedDataWritePrefixes`) in `protectedHookWriteReason` so a worker ordered to write phase research, survey, planning, or worker-debug artifacts under `.aether/data/` can actually do it
  - Scout permission profile converged onto the default workspace-write + behavioral-restriction pattern (matching the surveyor precedent) instead of `repository_read_only`
  - Worker `artifacts` sub-schema with named, nullable, typed properties (`research_file`, `survey_file`, `plan_file`) instead of an `{}`-only schema
  - Four distributed docs (both `aether-colony.md` copies, `OPENCODE.md`, `protected-local-state-contract.md`) now name the sanctioned scratch subpaths, checked by a repo-wide test
affects: [163-context-reaches-workers remaining plans, any future phase touching cmd/hook_cmds.go or pkg/codex permission/schema code]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Exact-subpath allowlist checked before a blanket-block switch case, evaluated on full slash-delimited directory segments (never a bare substring/name match)"
    - "Strict JSON schema convention: every declared property listed in required, nullable (type: [T, null]) when optional, rather than omitting it from required — matches the existing handoff sub-schema and the Codex --output-schema strict validator"

key-files:
  created: []
  modified:
    - cmd/hook_cmds.go
    - cmd/hook_cmds_test.go
    - pkg/codex/permission_profile.go
    - pkg/codex/permission_profile_test.go
    - pkg/codex/worker.go
    - pkg/codex/worker_test.go
    - cmd/internal_worker_adapter_test.go
    - cmd/permission_profile_integration_test.go
    - .aether/rules/aether-colony.md
    - .claude/rules/aether-colony.md
    - .opencode/OPENCODE.md
    - .aether/references/contracts/protected-local-state-contract.md

key-decisions:
  - "artifacts schema fields are nullable + listed in required (not omitted from required) to satisfy the pre-existing strict-schema invariant (TestWorkerClaimsSchemaStrictObjects, mirrored by the Codex --output-schema strict validator) — the plan's literal 'keep required empty' text would have broken that invariant"
  - "includer replaces scout as the read-only fixture caste in tests that need a genuinely read-only example, since scout is no longer repository_read_only"
  - "OpenCode's aether-scout.md agent definition was NOT updated to permit write (still declares write:false) — out of scope per Task 3's explicit 'do not edit the 27-caste agent files' instruction; flagged as a follow-up gap"

requirements-completed: [CONTEXT-04]

# Metrics
duration: 30min
completed: 2026-07-29
---

# Phase 163 Plan 03: Contract Bugs — Write Guardrail, Scout Permission, Artifacts Schema Summary

**Exact-subpath allowlist unblocks sanctioned `.aether/data/` writes, scout's permission profile now matches its own phase-research brief, and the artifacts schema accepts named typed fields instead of `{}`-only.**

## Performance

- **Duration:** ~30 min active work (plus long shared-machine contention on `go test ./cmd/...` runs, not reflective of task complexity)
- **Started:** 2026-07-29T18:22:00+02:00 (approx.)
- **Completed:** 2026-07-29T18:46:36+02:00
- **Tasks:** 3/3 completed
- **Files modified:** 12

## Accomplishments
- A worker told to write phase research, survey, planning, or worker-debug artifacts under `.aether/data/` can now do so — `protectedHookWriteReason` checks a 4-entry, full-directory-segment allowlist before the blanket block, and the existing negative test (`TestHookPreToolUseBlocksProtectedPath`) stays byte-identical, proving the carve-out did not widen protection elsewhere.
- Scout's permission profile no longer contradicts `renderPhaseResearchBrief`'s own instruction: scout moved off `repositoryReadOnlyCastes` onto the default workspace-write profile with a `.aether/data/phase-research`-scoped behavioral restriction, mirroring the surveyor pattern. `includer` remains the sole read-only caste, proving the carve-out is scout-specific.
- The `artifacts` sub-schema went from `{}`-only (empty properties/required, so the only legal value was `{}`) to three named, nullable, typed fields — `research_file`, `survey_file`, `plan_file` — while `additionalProperties` stays `false`.
- Four documents (`.aether/rules/aether-colony.md`, `.claude/rules/aether-colony.md`, `.opencode/OPENCODE.md`, `.aether/references/contracts/protected-local-state-contract.md`) now name the sanctioned scratch subpaths, and a new test (`TestSanctionedScratchDirsDocumented`) fails if any of the four documents drops one — verified by deliberate deletion and revert.

## Task Commits

1. **Task 1: Exact-subpath write allowlist in the hook that actually enforces** - `fb6df774` (feat)
2. **Task 2: Scout permission and artifacts schema stop contradicting their own briefs** - `ed6306c7` (feat)
3. **Task 3: Distributed rules tell the truth about what is writable** - `f0829822` (docs)

_No TDD RED/GREEN split — plan tasks were marked `tdd="true"` at the behavior-list level but executed as single feat commits per task with tests added alongside implementation, consistent with the plan's per-task action structure (behaviors + implementation + tests described together, not a separate RED-phase commit gate)._

## Files Created/Modified
- `cmd/hook_cmds.go` - Added `sanctionedDataWritePrefixes` (4 entries) and the allowlist check in `protectedHookWriteReason`, evaluated before the blanket `.aether/data/` block; updated the block message to point workers at where they may write
- `cmd/hook_cmds_test.go` - Added `TestHookPreToolUseAllowsSanctionedScratchDirs` (behaviors 2-5, 6, 7) and `TestSanctionedScratchDirsDocumented`
- `pkg/codex/permission_profile.go` - Removed `scout` from `repositoryReadOnlyCastes`; added a `scout` case to `behavioralRestrictionsForCaste` naming `.aether/data/phase-research`
- `pkg/codex/permission_profile_test.go` - Flipped the scout row to `PermissionWorkspaceWrite`; added `TestScoutPermissionProfileAllowsPhaseResearchWrite`; switched `TestCodexReadOnlyProfileSelectsReadOnlySandbox` and `TestResolvePermissionProfileRejectsBroadeningAndStaleContracts` from scout to includer (the surviving read-only caste)
- `pkg/codex/worker.go` - Replaced the empty artifacts `properties`/`required` with named nullable typed fields; updated the response-contract prose to describe the three fields
- `pkg/codex/worker_test.go` - Added `TestWorkerArtifactsSchemaAcceptsNamedFields`
- `cmd/internal_worker_adapter_test.go` - Switched the "broadened profile rejected" fixture from scout to includer (scout is no longer the read-only example)
- `cmd/permission_profile_integration_test.go` - Updated build/planning dispatch assertions for scout's new `PermissionWorkspaceWrite` profile
- `.aether/rules/aether-colony.md`, `.claude/rules/aether-colony.md` - Protected Paths table now carves out the four sanctioned scratch subpaths with a "when a worker writes here" table
- `.opencode/OPENCODE.md` - Same carve-out, with an explicit note that OpenCode enforcement is conduct-only (no `PreToolUse` hook on that platform)
- `.aether/references/contracts/protected-local-state-contract.md` - Corrected the "Any file in `.aether/data/`" update-safety line (with a clarifying note that update/publish still never touch these paths regardless of worker write access) and the "What Lives in `.aether/data/`" table

## Decisions Made
- Artifacts schema fields are nullable and listed in `required` rather than omitted from `required` — the plan's literal text said "keep required empty," but the pre-existing strict-schema invariant (`TestWorkerClaimsSchemaStrictObjects`, and the real Codex `--output-schema` strict validator) requires every declared property to appear in `required`. Making the fields nullable (`type: ["string","null"]`) achieves the same practical outcome (a worker producing no artifacts stays valid) while satisfying that invariant, matching the existing sibling `handoff` schema's convention.
- `includer` replaces `scout` as the "genuinely read-only" fixture caste in four pre-existing tests that hardcoded scout's old profile. This is a direct, necessary consequence of the intentional permission change — those tests encoded the contradiction this plan fixes.
- Left `.opencode/agents/aether-scout.md` untouched (still declares `write: false`, `edit: false`, `bash: false`) per Task 3's explicit "do not edit the 27-caste agent files in this plan" boundary. This means on OpenCode specifically, scout's *real* enforced tool permissions still block writes even though the Go permission profile now permits them — a genuine cross-platform gap, recorded below as a follow-up.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed two pre-existing tests broken by the intentional scout permission change**
- **Found during:** Task 2 (running `go test ./cmd/... -count=1` after the permission profile flip)
- **Issue:** `cmd/internal_worker_adapter_test.go`'s `TestInternalWorkerAdapterRequiresAndValidatesPermissionProfile` and `cmd/permission_profile_integration_test.go`'s `TestBuildAndPlanningManifestsCarryCanonicalPermissionProfiles` both hardcoded scout as `PermissionRepositoryReadOnly` — a direct assertion of the exact contract this plan intentionally changes.
- **Fix:** Switched the "broadened read-only profile rejected" fixture in `internal_worker_adapter_test.go` from scout to includer (the caste that is still genuinely read-only); updated the build/planning dispatch assertions in `permission_profile_integration_test.go` to expect `PermissionWorkspaceWrite` for scout.
- **Files modified:** cmd/internal_worker_adapter_test.go, cmd/permission_profile_integration_test.go
- **Verification:** `go test ./cmd/... -count=1` passes (confirmed twice consecutively after fix, following one earlier transient failure under heavy concurrent-agent machine load — see Issues Encountered)
- **Committed in:** ed6306c7 (Task 2 commit)

**2. [Rule 1 - Bug] Two more pkg/codex tests hardcoded scout as the read-only fixture**
- **Found during:** Task 2 (running `go test ./pkg/codex -count=1`)
- **Issue:** `TestCodexReadOnlyProfileSelectsReadOnlySandbox` asserted the Codex sandbox for caste `scout` is `read-only`, and `TestResolvePermissionProfileRejectsBroadeningAndStaleContracts` used scout as its "read-only caste rejects a broader profile" example — both now false given scout is `PermissionWorkspaceWrite`.
- **Fix:** Rewrote both tests to use `includer` (agent name, TOML fixture, and caste) instead of scout.
- **Files modified:** pkg/codex/permission_profile_test.go
- **Verification:** `go test ./pkg/codex -count=1` passes
- **Committed in:** ed6306c7 (Task 2 commit)

**3. [Rule 1 - Bug] Nullable-typed artifacts fields instead of literally empty required**
- **Found during:** Task 2 (running `TestWorkerClaimsSchemaStrictObjects` after adding named properties with an empty `required` list, as the plan's action text specified)
- **Issue:** The test failed: `$.artifacts object schema required list missing "research_file"`. The codebase enforces a strict-schema convention (every declared property must appear in `required`, matching how the sibling `handoff` schema and the real Codex `--output-schema` strict validator work) that the plan's literal "keep required empty" instruction did not account for.
- **Fix:** Made `research_file`, `survey_file`, and `plan_file` nullable (`"type": ["string", "null"]`) and listed all three in `required`, preserving the practical goal (a worker producing no artifacts still validates, by submitting `null` for each field) without breaking the existing invariant.
- **Files modified:** pkg/codex/worker.go
- **Verification:** `go test ./pkg/codex -run TestWorkerClaimsSchemaStrictObjects -count=1` and `TestWorkerArtifactsSchemaAcceptsNamedFields` both pass
- **Committed in:** ed6306c7 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking, 2 bugs — all direct consequences of the intentional scout-permission and artifacts-schema changes)
**Impact on plan:** All fixes necessary to keep `go test ./cmd/... -count=1` and `go test ./pkg/codex -count=1` green, as the plan's own acceptance criteria require. No scope creep — every touched file was either explicitly listed in the plan's `files_modified` or a pre-existing test directly encoding the old (buggy) contract this plan fixes.

## Issues Encountered
- One transient failure of `go test ./cmd/... -count=1` (opaque `FAIL github.com/calcosmic/Aether/cmd 386.214s` with no visible `--- FAIL:` line in the captured tail) occurred while several sibling worktree agents were concurrently running the same full test suite on this machine (`uptime` showed load averages of 20-54 at the time). Two subsequent clean, consecutive full runs (`ok github.com/calcosmic/Aether/cmd`, 298-348s each) with zero `--- FAIL` lines confirm this was transient machine-load flakiness, not a real regression from this plan's changes.
- Deliberate-regression proofs were run and reverted for both Task 1 (widened `sanctionedDataWritePrefixes` entry to a bare `/.aether/data/` match — confirmed `TestHookPreToolUseBlocksProtectedPath` fails with "unexpected end of JSON input", i.e. the block stopped firing) and Task 3 (deleted the `planning/` row from `.aether/rules/aether-colony.md` — confirmed `TestSanctionedScratchDirsDocumented` fails with "does not name sanctioned scratch subpath"). Both reverted and confirmed green.

## Known Follow-ups (recorded per Task 3's action text, not fixed in this plan)

**OpenCode agent-parity gap:** `.opencode/agents/aether-scout.md` still declares `write: false`, `edit: false`, `bash: false` in its frontmatter. `validateOpenCodePermissionBoundary` only enforces the stricter all-tools-false check when `profile.Name == PermissionRepositoryReadOnly`, so this doesn't fail any existing test now that scout is `PermissionWorkspaceWrite` — but it means scout's *actual* enforced permissions on OpenCode specifically still block writes, unlike Claude Code and Codex. Editing the 27-caste agent files was explicitly out of scope for this plan (Task 3: "Do NOT edit the 27-caste agent files in this plan — that surface belongs to the agent-parity contract and would collide with parity goldens").

**Repo-wide blanket `.aether/data/` claims sweep:** `grep -rn 'aether/data' .claude/agents/ant .opencode/agents .codex/agents` returns 326 matches across 67 unique files (27 castes × 3 platforms, minus router-only files). This was not filtered down to only the blanket-protection-claim subset (would require per-file judgment on each of 326 lines) — the full grep output and count are recorded here as the honest sweep result per the plan's action text, flagged as a follow-up for whichever phase owns the 27-caste agent-file surface.

**`aether publish` still owed:** This plan edited distributed rule/doc files (`.aether/rules/aether-colony.md`, `.claude/rules/aether-colony.md`, `.opencode/OPENCODE.md`, `.aether/references/contracts/protected-local-state-contract.md`) but did not run `aether publish`, per the plan's explicit instruction to defer publish until all of this phase's distributed edits land.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The write-guardrail evasion, scout permission contradiction, and artifacts-schema dead end are all closed with tests that fail on regression.
- Remaining phase 163 waves (if any) and the eventual publish step can proceed; the publish step should be deferred until all of phase 163's distributed-doc edits are complete, per this plan's own note.
- Known follow-ups above (OpenCode agent-parity gap, 326-match blanket-claim sweep) are explicitly out of this plan's scope and should be triaged by whichever phase owns the 27-caste agent-file surface.

---
*Phase: 163-context-reaches-workers*
*Completed: 2026-07-29*
