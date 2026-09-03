---
phase: 199-front-door-and-classic-contract
plan: "06"
subsystem: expert-maintenance
tags: [go, cobra, maintenance, read-only, lifecycle-projection, skills]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "04"
    provides: Shared factual lifecycle projection and focused closeout contract
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Transaction, receipt, recovery, and rollback vocabulary for mutating maintenance operations
provides:
  - One read-only expert maintenance landing with a stable inspection and mutation catalog
  - Typed zero-write integrity and generated/source parity inspection results
  - Live repository and hub skill inventory receipts plus deterministic semantic drift reports
affects: [maintenance, integrity, source-check, skills, help, lifecycle-orientation]

tech-stack:
  added: []
  patterns: [command-owned read-only annotations, typed inspection receipts, live digest-backed inventory]

key-files:
  created:
    - cmd/maintenance_cmd.go
    - cmd/maintenance_cmd_test.go
    - cmd/maintenance_read_199_test.go
    - cmd/maintenance_skills_199_test.go
  modified:
    - cmd/root.go
    - cmd/integrity_cmd.go
    - cmd/source_check.go
    - cmd/skills.go
    - cmd/testdata/command_catalog.json
    - cmd/visual_writer_discipline_test.go
    - .aether/commands/help.yaml
    - .claude/commands/ant-help.md
    - .claude/commands/ant/help.md
    - .opencode/commands/ant/help.md

key-decisions:
  - "Mark inspection commands themselves as read-only and store-free so first-run orientation and lock initialization cannot mutate a supposedly zero-write invocation."
  - "Identify live skills by source class plus relative path and compare SHA-256 content digests, making every scan immediate and independent of an index or cache."
  - "Expose skill inventory only below maintenance while preserving the Phase 191 removal of standalone skill-list, skill-diff, and cache-rebuild commands."

patterns-established:
  - "Maintenance catalog first: opening the expert landing lists stable operations and their recovery contracts but never starts one."
  - "Result before rendering: inspection builders return typed findings, evidence, verification, blockers, next action, and state effect without requiring prose parsing."
  - "Live-read receipts: skill inspection scans current repository and hub sources on every invocation and normalizes deterministic empty collections."

requirements-completed: [CEC-02, LIFE-06]

duration: 29min
completed: 2026-09-03
---

# Phase 199 Plan 06: Expert Maintenance and Live Skill Inspection Summary

**One read-only expert landing now separates inspection from mutation, returns factual lifecycle-aware closeout data, and exposes immediate digest-backed skill inventory and drift without reviving the deleted cache CLI.**

## Performance

- **Duration:** 29 minutes
- **Started:** 2026-09-03T19:51:06Z
- **Completed:** 2026-09-03T20:20:16Z
- **Tasks:** 3/3
- **Files modified:** 14 production, test, catalog, and help files

## Accomplishments

- Added `aether maintenance` with the exact approved help text, one Queen explanation, a stable catalog divided into read-only inspection and explicit mutation, and the shared lifecycle projection in its focused closeout.
- Cataloged integrity, source parity, registry, chamber, context, archive, and live skill inspection alongside update, migration, cleanup, pruning, registry mutation, and chamber creation; every entry names mutation class, preview availability, transaction requirement, receipt type, and recovery action.
- Extracted typed `integrity.inspect` and `source.parity.inspect` results with findings, evidence, verification, blockers, recovery command, next action, and `state_effect: none`, while preserving existing checks and separating rendering from result construction.
- Added `maintenance skills inspect` and `maintenance skills diff`: each invocation scans live repository and hub skill roots, hashes current `SKILL.md` bytes, reports parsed metadata/errors, and deterministically classifies added, removed, changed, and invalid entries.
- Preserved the deleted root cache/list/diff command surface while adding maintenance reachability to canonical and generated help, command-catalog, and visual-output discipline gates.

## Task Commits

Each TDD task was committed as a RED contract followed by its GREEN implementation:

1. **Task 1: Add the expert maintenance landing** — `3310501f` (RED), `fa842ea1` (GREEN)
2. **Task 2: Give inspection commands typed zero-write results** — `ff1ca5e4` (RED), `4e5d0865` (GREEN)
3. **Task 3: Restore inspectable live skill scanning and diffing** — `06469d8a` (RED), `c9dc3319` (GREEN)

**Plan metadata:** committed with this summary.

## Files Created/Modified

- `cmd/maintenance_cmd.go` — expert landing, stable operation catalog, nested skill commands, lifecycle projection, and focused visual closeout.
- `cmd/maintenance_cmd_test.go` — zero-write, visual/JSON parity, operation metadata, lifecycle, and first-run contracts.
- `cmd/integrity_cmd.go` — typed integrity builder, evidence paths, findings, recovery, verification, and separate visual/JSON rendering.
- `cmd/source_check.go` — typed source-parity findings, exact source/generated evidence, verification, next action, and zero-write result finalization.
- `cmd/maintenance_read_199_test.go` — pass, drift, invalid, and missing-input filesystem fingerprint coverage for both inspection paths.
- `cmd/skills.go` — live inventory receipts, SHA-256 digests, metadata validation, receipt/manifest loading, deterministic diffing, and nested maintenance rendering.
- `cmd/maintenance_skills_199_test.go` — exact LiveScan, Diff, Invalid, ReadOnly, and NoLegacyCLI contract suite.
- `cmd/root.go` — command-level read-only/store-free startup guards that prevent incidental first-run or lock-directory writes.
- `.aether/commands/help.yaml`, `.claude/commands/ant-help.md`, `.claude/commands/ant/help.md`, `.opencode/commands/ant/help.md` — expert-maintenance discovery without promoting internals into the ordinary journey.
- `cmd/testdata/command_catalog.json` — registered maintenance hierarchy and stable help metadata.
- `cmd/visual_writer_discipline_test.go` — explicit exemptions for typed renderer helpers used by the refactored inspection paths.

## Decisions Made

- Made read-only behavior a Cobra command annotation enforced by root startup, so correctness does not depend on a caller remembering to suppress store initialization or first-run welcome state.
- Reused `projectLifecycle` for identity, standing, blockers, and next action; maintenance does not infer a competing status or duplicate lifecycle policy.
- Kept integrity/source builders side-effect-free and renderer-agnostic, allowing both direct commands and the maintenance catalog to consume stable result contracts.
- Used source plus relative path as the live skill identity and SHA-256 of exact file bytes as evidence; manifest comparison realigns identities by normalized skill name only at the explicit manifest boundary.
- Kept live inventory and diff nested below `maintenance skills`; no standalone skill cache, list, or diff registration was restored.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Prevented the read-only landing from writing first-run welcome state**

- **Found during:** Task 1 read-only fingerprint verification
- **Issue:** Root persistent startup could create `.aether/data/.welcomed` before the maintenance handler ran, changing repository bytes and mtimes despite a read-only command.
- **Fix:** Added a command-owned `aether.io/read-only` annotation and taught root startup to skip the welcome write for annotated commands.
- **Files modified:** `cmd/maintenance_cmd.go`, `cmd/root.go`
- **Verification:** Landing fingerprints remain identical before and after both visual and JSON invocations, including a first-run fixture.
- **Committed in:** `fa842ea1`

**2. [Rule 1 - Bug] Prevented inspection startup from creating a lock directory**

- **Found during:** Task 2 pass, drift, and missing-source fingerprint verification
- **Issue:** Constructing the shared store for an inspection could initialize `.aether/locks`, violating the command's zero-write contract before result construction.
- **Fix:** Added store-free command metadata and deferred shared-store initialization for the read-only maintenance, integrity, and source-check paths.
- **Files modified:** `cmd/root.go`, `cmd/integrity_cmd.go`, `cmd/source_check.go`, `cmd/maintenance_cmd.go`
- **Verification:** Workspace and hub byte/mtime fingerprints are unchanged for valid, drifting, invalid, and missing inputs.
- **Committed in:** `4e5d0865`

**3. [Rule 3 - Blocking] Reconciled global command-surface gates for the new maintenance hierarchy**

- **Found during:** Task 3 broader command-package verification
- **Issue:** The registered nested commands initially made the catalog golden stale, lacked caller evidence for the reachability ratchet, and moved output through newly extracted renderer functions that the visual-writer audit did not recognize.
- **Fix:** Refreshed the command catalog, added canonical and generated expert-help references, and updated the visual-writer exemptions for the typed renderer boundaries.
- **Files modified:** `cmd/testdata/command_catalog.json`, `.aether/commands/help.yaml`, `.claude/commands/ant-help.md`, `.claude/commands/ant/help.md`, `.opencode/commands/ant/help.md`, `cmd/visual_writer_discipline_test.go`
- **Verification:** Command catalog, registration reachability, visual-output discipline, and source-surface parity focused gates all pass.
- **Committed in:** `c9dc3319`

---

**Total deviations:** 3 auto-fixed (2 correctness bugs, 1 blocking integration issue).
**Impact on plan:** The fixes enforce the promised zero-write boundary and keep the new expert surface consistent with repository-wide command audits; no unrelated runtime capability was added.

## Issues Encountered

- A full `go test ./cmd -count=1` exposed the archived Plan 196 fixture and five legacy Next Up assertions already covered by Phase 199's staged lifecycle migration. Plan-owned catalog, reachability, and output-discipline failures were fixed inline; the remaining named tests are recorded in `deferred-items.md` for their owning renderer plans.

## Verification

- `go test ./cmd -run '^TestMaintenanceLanding(ReadOnly|Catalog|OutputModes)$' -count=1` — passed.
- `go test ./cmd -run '^TestMaintenanceInspection(ReadOnly|StructuredResult|DriftEvidence)$' -count=1` — passed.
- Exact one-function `TestMaintenanceSkills199` source guard plus all LiveScan, Diff, Invalid, ReadOnly, and NoLegacyCLI subtests — passed.
- Related integrity, source-check, root startup, skill, and maintenance regression suites — passed.
- `TestAuditCatalogGolden`, `TestNoRegisteredSubcommandIsUnreferenced`, `TestHumanFacingOutputGoesThroughWriteVisualOutput`, and `TestSourceCheckValidatesCurrentSourceSurfaces` — passed after integration reconciliation.
- Stub scan found no TODO, FIXME, placeholder, coming-soon, unavailable, or goal-blocking unwired value. Explicit empty slices are intentional stable JSON collections.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: filesystem-inspection | `cmd/skills.go`, `cmd/source_check.go`, `cmd/integrity_cmd.go` | The expert inspection lane reads live repository/hub skill files, explicit receipt or manifest paths, generated/source surfaces, version files, and the current executable. It performs no writes, reports parse/read failures as evidence, hashes skill bytes, and is covered by complete before/after filesystem fingerprints; callers should still treat inspected file contents and supplied paths as untrusted input. |

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-18 can attach explicit transaction-backed repair operations to the catalog without changing inspection semantics.
- Plan 199-19 can publish wrapper guidance for the runtime-owned maintenance landing and nested skill operations.
- Command-specific lifecycle renderers can consume the shared factual projection and typed inspection results without parsing visual prose.
- No Plan 199-06 implementation blocker remains.

## Self-Check: PASSED

- All four planned created files and this summary exist.
- All three RED commits and all three GREEN commits are present in Git history.
- The exact plan suites and directly affected command-surface audit gates pass, and `git diff --check` is clean.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-03*
