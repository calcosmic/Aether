---
phase: 08-xml-integration
plan: 04
subsystem: xml
tags: [xml, xmllint, entomb, tunnels, pheromone-import, shell, aether-utils]

# Dependency graph
requires:
  - phase: 08-xml-integration
    plan: 03
    provides: colony-archive-xml subcommand in aether-utils.sh
provides:
  - entomb XML tool check (command -v xmllint) + hard-stop export (Step 3.5/4.5 + Step 7.5/6.5)
  - pheromone-import-xml bug fixes: correct signal extraction, merge order, colony prefix support
  - tunnels import flow (Step 6): browse chambers and import pheromone signals into current colony
affects: [entomb, tunnels, pheromone-import-xml]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hard-stop export: XML failure removes chamber (rm -rf) and aborts entomb — colony state NOT reset"
    - "Two-step signal extraction: jq -r result.json | jq .signals (not result.signals integer)"
    - "Colony prefix tagging: imported signal IDs prepended with prefix: before merge"
    - "Merge order: [$imported[], $existing[]] so map(last) keeps current colony on collision"
    - "Pheromone section extraction: xmllint --xpath pheromones element to temp file before import"

key-files:
  created: []
  modified:
    - .claude/commands/ant/entomb.md
    - .aether/commands/claude/entomb.md
    - .opencode/commands/ant/entomb.md
    - .aether/commands/opencode/entomb.md
    - .claude/commands/ant/tunnels.md
    - .aether/commands/claude/tunnels.md
    - .opencode/commands/ant/tunnels.md
    - .aether/commands/opencode/tunnels.md
    - .aether/aether-utils.sh

key-decisions:
  - "entomb tool check uses command -v xmllint (not xml-detect-tools) — consistent with seal.md pattern"
  - "Hard-stop: XML export failure removes chamber directory and aborts; colony state is never reset on failure"
  - "Step numbering differs between Claude Code SoT (3.5/7.5) and OpenCode live (4.5/6.5) due to different ceremony structures — semantics identical"
  - "pheromone-import-xml signal extraction reads result.json (jq -r) then .signals (not integer result.signals)"
  - "Merge order fixed: imported signals first, existing last — map(last) now correctly keeps current colony on ID collision"
  - "tunnels passes extracted pheromone-only temp file to pheromone-import-xml (not combined colony-archive.xml) — XPath signal count scoped to pheromones section only"

patterns-established:
  - "Pheromone section extraction pattern: xmllint --xpath pheromones element, prepend XML declaration via temp file"
  - "Hard-stop pattern: failure removes partial work, preserves original state, clears exit path"

requirements-completed: [XML-01, XML-02, XML-03]

# Metrics
duration: 8min
completed: 2026-02-18
---

# Phase 8 Plan 04: Entomb XML Hard-Stop + Tunnels Import Flow Summary

**XML archiving wired into entomb as a hard requirement (colony cannot be entombed without a successful XML export), plus cross-colony learning enabled by a new import flow in tunnels (browse old colonies and merge their pheromone signals into the current colony).**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-02-18T01:05:59Z
- **Completed:** 2026-02-18T01:13:52Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- All 4 entomb.md files (Claude Code + OpenCode, live + SoT) updated with: xmllint tool check before entombment proceeds (with AskUserQuestion install offer), hard-stop XML export after all files are archived (chamber removed on failure, state NOT reset), and `{xml_archive_line}` in the completion display
- `pheromone-import-xml` in aether-utils.sh has three fixes: signal extraction reads actual signal array from `result.json | .signals` (not the integer count in `result.signals`), merge order swapped so `map(last)` keeps the current colony on ID collision, and optional second argument `colony_prefix` tags imported signal IDs with `${prefix}:` before the merge
- All 4 tunnels.md files updated with conditional import option in Step 3 detail view (shown only when `colony-archive.xml` exists), plus new Step 6 import flow with tool check, pheromone section extraction to temp file, preview with signal count, user confirmation, `pheromone-import-xml` call with source colony prefix, and result display

## Task Commits

1. **Task 1: Wire XML tool check and hard-stop export into /ant:entomb** - `648d75e` (feat)
2. **Task 2: Fix pheromone-import-xml signal extraction, merge order, and colony prefix** - `22959a4` (fix)
3. **Task 3: Add import flow to /ant:tunnels** - `6542168` (feat)

## Files Created/Modified

- `.claude/commands/ant/entomb.md` - Added Step 3.5 (tool check) and Step 7.5 (hard-stop export), `{xml_archive_line}` in display
- `.aether/commands/claude/entomb.md` - SoT copy — identical to live Claude Code copy
- `.opencode/commands/ant/entomb.md` - Added Step 4.5 (tool check) and Step 6.5 (hard-stop export), `{xml_archive_line}` in display
- `.aether/commands/opencode/entomb.md` - SoT copy — Step 3.5/7.5 matching Claude Code ceremony structure
- `.claude/commands/ant/tunnels.md` - Modified Step 3 footer, added Step 6 import flow
- `.aether/commands/claude/tunnels.md` - SoT copy — identical to live Claude Code copy
- `.opencode/commands/ant/tunnels.md` - Adapted Step 3 detail view + Step 6 import flow (OpenCode format)
- `.aether/commands/opencode/tunnels.md` - SoT copy — Step 3/6 matching Claude Code structure
- `.aether/aether-utils.sh` - Fixed `pheromone-import-xml` subcommand (3 changes: extraction, merge order, prefix)

## Decisions Made

- Tool check uses `command -v xmllint` (not `xml-detect-tools`) to stay consistent with the seal.md Step 6.5 pattern established in plan 03
- Hard-stop means: XML failure removes chamber directory and returns immediately — the colony state reset (Step 10) is never reached. Colony remains entombed-ready for a retry
- Step numbering adapts between Claude Code SoT (Step 3.5/7.5) and OpenCode live (Step 4.5/6.5) because the two files have different ceremony structures — the semantics are identical
- Signal extraction from `result.json` is a two-step jq pipe: `jq -r '.result.json // "{}"' | jq -c '.signals // []'` — the result.json field is a serialized JSON string that must be parsed as a second step
- tunnels passes the extracted pheromone-only XML to `pheromone-import-xml` (not the combined archive) so: (1) the XPath signal count is scoped to pheromones only, preventing over-counting from wisdom/registry sections, (2) `pheromone-import-xml` receives the `<pheromones>` root element it was designed for

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None.

## Phase 8 Completion

Phase 8 (XML Integration) is now COMPLETE:
- Plan 01: pheromone-export-xml, pheromone-import-xml, pheromone-validate-xml subcommands
- Plan 02: wisdom-export-xml, wisdom-import-xml, registry-export-xml, registry-import-xml subcommands
- Plan 03: colony-archive-xml combined export + best-effort seal wiring
- Plan 04: entomb hard-stop export + tunnels import flow

## Self-Check

- `.planning/phases/08-xml-integration/08-04-SUMMARY.md` — this file
- `648d75e` — Task 1 commit (entomb XML tool check + hard-stop)
- `22959a4` — Task 2 commit (pheromone-import-xml fixes)
- `6542168` — Task 3 commit (tunnels import flow)

## Self-Check: PASSED

All three commits exist. All 9 modified files verified with grep checks. Key link patterns confirmed:
- entomb: `command -v xmllint` check present, `colony-archive-xml` hard-stop present, `xml_archive_line` in display
- aether-utils.sh: `pix_colony_prefix` second arg, `result.json` extraction, `$new_signals[], $existing.signals[]` order, `map(last)` preserved
- tunnels: `colony-archive.xml found` in Step 3, `pheromone-import-xml "$import_tmp_pheromones" "$source_colony"` in Step 6.4

---
*Phase: 08-xml-integration*
*Completed: 2026-02-18*
