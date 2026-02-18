---
phase: 08-xml-integration
plan: 03
subsystem: xml
tags: [xml, xmllint, shell, aether-utils, pheromones, wisdom, registry, seal]

# Dependency graph
requires:
  - phase: 08-xml-integration
    provides: pheromone-export-xml, wisdom-export-xml, registry-export-xml subcommands in aether-utils.sh
provides:
  - colony-archive-xml subcommand in aether-utils.sh — combined XML export (pheromones + wisdom + registry)
  - /ant:seal XML archive step — best-effort colony snapshot on seal
affects: [08-xml-integration-04, entomb]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Best-effort export pattern: xmllint check -> attempt export -> informational line on failure, never blocks parent command"
    - "Combined XML archive by stripping XML declarations and wrapping in root element"
    - "cax_ variable prefix for colony-archive-xml subcommand"

key-files:
  created: []
  modified:
    - .aether/aether-utils.sh
    - .aether/exchange/registry-xml.sh
    - .claude/commands/ant/seal.md
    - .aether/commands/claude/seal.md
    - .opencode/commands/ant/seal.md
    - .aether/commands/opencode/seal.md

key-decisions:
  - "colony-archive-xml always filters active-only pheromones — active=false signals excluded from archive snapshots"
  - "Well-formedness validation with xmllint --noout on combined file; no XSD validation of wrapper (individual sections validated by their exchange scripts)"
  - "Step numbering differs between Claude Code (6.5) and OpenCode (5.75) due to different seal ceremony structures — semantics identical"
  - "OpenCode SoT seal.md uses ceremony-style (Step 6.5) matching Claude Code structure rather than archiving-style of the live .opencode/commands/ant/seal.md"

patterns-established:
  - "Best-effort export: command -v xmllint check -> attempt -> result line for ceremony display, no exit on failure"
  - "Combined XML: export each section to temp files, strip declarations, concatenate under single root element"

requirements-completed: [XML-01, XML-02, XML-03]

# Metrics
duration: 25min
completed: 2026-02-18
---

# Phase 8 Plan 03: colony-archive-xml Combined Export + Seal Wiring Summary

**Combined XML export subcommand that merges pheromones (active-only), wisdom, and registry into a single `<colony-archive>` document, automatically triggered by `/ant:seal` as a best-effort milestone snapshot.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-02-18T00:50:00Z
- **Completed:** 2026-02-18T01:15:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- `colony-archive-xml` subcommand added to aether-utils.sh — produces a single well-formed XML file containing all three data types under a `<colony-archive>` root element
- Active-only pheromone filter ensures expired/inactive signals never appear in archive snapshots
- Both Claude Code and OpenCode seal.md files (live + SoT, 4 total) updated with best-effort XML export step that runs between the main seal ceremony steps and reports status in the ceremony display

## Task Commits

1. **Task 1: Add colony-archive-xml combined export subcommand** - `0605952` (feat)
2. **Task 2: Wire best-effort XML export into /ant:seal** - `3a18af6` (feat)

## Files Created/Modified

- `.aether/aether-utils.sh` - Added `colony-archive-xml` case in XML Exchange Commands section (~100 lines)
- `.aether/exchange/registry-xml.sh` - Bug fix: XML-escape colony ID in attribute value (Rule 1 auto-fix)
- `.claude/commands/ant/seal.md` - Added Step 6.5 with XML export + `{xml_export_line}` in ceremony display
- `.aether/commands/claude/seal.md` - SoT copy — identical to live Claude Code copy
- `.opencode/commands/ant/seal.md` - Added Step 5.75 with XML export + `{xml_export_line}` in archive display
- `.aether/commands/opencode/seal.md` - SoT copy — ceremony-style Step 6.5 matching Claude Code

## Decisions Made

- Active-only pheromone filter is always applied for archives (no `--active-only` flag needed — this is the only use case for the combined archive)
- Well-formedness validation only; no XSD validation of the wrapper element (each child section was already validated by its exchange script)
- Step numbering differs between Claude Code seal (Step 6.5) and OpenCode seal (Step 5.75) because the two commands have different ceremony structures — the semantics are identical

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed unescaped `&` in registry XML colony ID attribute**
- **Found during:** Task 1 (colony-archive-xml implementation, first test run)
- **Issue:** `registry-xml.sh` wrote `<colony id="v1.1 Bug Fixes & Update System Repair" ...>` — the `&` in the colony ID was not XML-escaped, producing malformed XML and causing `xmllint --noout` to fail with `xmlParseEntityRef: no name`
- **Fix:** Applied `sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g'` to the `id` variable extraction in `xml-registry-export()` in `registry-xml.sh` — same escaping pattern already used for the `<name>` element body on the next line
- **Files modified:** `.aether/exchange/registry-xml.sh`
- **Verification:** `xmllint --noout /tmp/test-archive.xml` returns exit 0 and combined archive `valid:true`
- **Committed in:** `0605952` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug)
**Impact on plan:** Bug was directly blocking the archive from being well-formed XML. Fix was minimal and consistent with existing escaping pattern in the same function.

## Issues Encountered

None beyond the registry XML escaping bug documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `colony-archive-xml` subcommand is available for plan 04 (entomb) to use with hard-stop semantics
- The subcommand returns JSON with `path`, `valid`, `colony_id`, and `pheromone_count` — all fields plan 04 will need
- Active-only filtering and well-formedness validation already in place

## Self-Check

- `.planning/phases/08-xml-integration/08-03-SUMMARY.md` — this file
- `0605952` — Task 1 commit (colony-archive-xml subcommand)
- `3a18af6` — Task 2 commit (seal.md XML wiring)

---
*Phase: 08-xml-integration*
*Completed: 2026-02-18*
