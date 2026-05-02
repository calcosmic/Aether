---
phase: 91-hive-intelligence
plan: 02
subsystem: skills
tags: [skills, lifecycle, crud, progressive-disclosure, fts5-search, cli]

# Dependency graph
requires:
  - phase: 91-01
    provides: "SQLiteColonyStore with DB() accessor, skills table schema, FTS5 Search method"
provides:
  - "SkillService with CRUD operations (create, get, patch, archive, pin, list)"
  - "Progressive disclosure via BuildSkillIndex returning lightweight entries"
  - "CLI commands: hive-search, skill-create, skill-patch, skill-archive, skill-pin, skill-view, skill-list-lifecycle"
  - "Path traversal prevention via validateSkillName"
  - "Pinned skill immutability (patch and archive blocked)"
affects: [91-03, 91-04]

# Tech tracking
tech-stack:
  added: []
  patterns: [file-metadata-duality, progressive-disclosure-index, path-validation]

key-files:
  created:
    - pkg/learn/skills.go
    - pkg/learn/skills_test.go
    - cmd/hive_search.go
    - cmd/skill_lifecycle.go
  modified: []

key-decisions:
  - "SkillService takes *sql.DB directly (not SQLiteColonyStore) for separation of concerns"
  - "Description truncated to 200 chars in YAML frontmatter for progressive disclosure"
  - "Archive moves files between stage directories rather than copying"
  - "Pinned skills blocked at both PatchSkill and ArchiveSkill (not just auto-transitions)"

patterns-established:
  - "File-metadata duality: SKILL.md on disk + SQLite row for lifecycle tracking"
  - "Progressive disclosure: BuildSkillIndex returns name/description/roles only, full content loaded on match"
  - "Path resolution: storage.ResolveAetherRoot for project root, store.BasePath() for colony.db"

requirements-completed: [HIVE-05, SKIL-01, SKIL-02, SKIL-03]

# Metrics
duration: 6min
completed: 2026-05-02
---

# Phase 91 Plan 02: Skill Lifecycle and Search CLI Summary

**Skill lifecycle CRUD with file-metadata duality, progressive disclosure index, path traversal prevention, pinned skill immutability, and FTS5 search via CLI**

## Performance

- **Duration:** 6 min
- **Started:** 2026-05-02T11:26:20Z
- **Completed:** 2026-05-02T11:32:30Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- SkillService provides full lifecycle CRUD: create (writes SKILL.md + SQLite row), get, patch (updates content + increments patch_count), archive (moves file between stage dirs), pin/unpin, list (with optional stage filter)
- Progressive disclosure via BuildSkillIndex returns lightweight SkillIndexEntry with name/description/roles only (description truncated to 200 chars)
- CLI commands registered: hive-search (FTS5 with filters), skill-create, skill-patch, skill-archive, skill-pin, skill-view, skill-list-lifecycle
- Path traversal prevented: validateSkillName rejects "/", "\\", "..", null bytes, empty names
- Pinned skills are immutable to both patching and archiving operations

## Task Commits

Each task was committed atomically:

1. **Task 1: Skill lifecycle types, CRUD operations, and progressive disclosure** - `a33bf1de` (test: RED), `17c7e056` (feat: GREEN)
2. **Task 2: CLI commands for hive search and skill lifecycle** - `7ef9f04e` (feat)

_Note: TDD cycle followed -- RED test commit first, then GREEN implementation commit._

## Files Created/Modified
- `pkg/learn/skills.go` - SkillMetadata, SkillIndexEntry, SkillEvidenceFrontmatter types; SkillService with CreateSkill, GetSkill, PatchSkill, ArchiveSkill, PinSkill, UnpinSkill, ListSkills, BuildSkillIndex; validateSkillName path traversal prevention
- `pkg/learn/skills_test.go` - 12 test functions covering CRUD, evidence frontmatter, pinning, archiving, listing, progressive disclosure, name validation, not-found cases
- `cmd/hive_search.go` - hive-search CLI command with limit, classification, min-confidence flags
- `cmd/skill_lifecycle.go` - skill-create, skill-patch, skill-archive, skill-pin, skill-view, skill-list-lifecycle CLI commands

## Decisions Made
- SkillService takes `*sql.DB` directly (not SQLiteColonyStore wrapper) for clean separation -- callers create the store for lifecycle management but pass the raw DB to the service
- Description field in YAML frontmatter capped at 200 characters for progressive disclosure -- keeps index entries lean
- Archive uses `os.Rename` (atomic on same filesystem) to move SKILL.md between stage directories
- Path resolution follows established cmd/ patterns: `storage.ResolveAetherRoot()` for project root, `filepath.Join(store.BasePath(), "colony.db")` for database path

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestSkillBuildIndex test assertion**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** Test expected "Long detailed content" string to not appear in index description, but the test content was only 69 chars -- shorter than the 200-char truncation limit, so it was included verbatim. The test was asserting the wrong thing.
- **Fix:** Changed test to use content longer than 200 chars via `strings.Repeat`, and changed assertion to check `len(entry.Description) > 200` instead of string matching.
- **Files modified:** pkg/learn/skills_test.go
- **Verification:** All 12 skill tests pass

---

**Total deviations:** 1 auto-fixed (1 bug -- test assertion)
**Impact on plan:** Fix necessary for test correctness. No scope creep.

## Issues Encountered
- `go build ./cmd/aether` fails with pre-existing `embedded_assets.go` embed pattern error (`.aether/rules: no matching files found`). This issue existed before this plan and is unrelated to our changes. All `go test` and `go vet` checks pass on the learn package.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SkillService is ready for use by Phase 91-03 (Curator) and 91-04 (Auto-Skill)
- BuildSkillIndex provides progressive disclosure data structure for skill-inject
- CLI commands follow established patterns for path resolution and output formatting
- Skills table schema (from 91-01) has all columns needed for lifecycle tracking

## Self-Check: PASSED

All files found: pkg/learn/skills.go, pkg/learn/skills_test.go, cmd/hive_search.go, cmd/skill_lifecycle.go, 91-02-SUMMARY.md
All commits found: a33bf1de, 17c7e056, 7ef9f04e, 880b3db5
All 12 skill tests pass.

---
*Phase: 91-hive-intelligence*
*Completed: 2026-05-02*
