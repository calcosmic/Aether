---
phase: 27-deterministic-fallback-dedup
plan: 02
subsystem: learning
tags: [bash, jq, git-diff, awk, instinct-create, fallback, playbook]

# Dependency graph
requires:
  - phase: 27-01
    provides: "_normalize_text helper, _jaccard_similarity helper, fuzzy dedup in instinct-create"
provides:
  - _learning_extract_fallback subcommand for git-diff-based learning extraction
  - continue-advance.md Step 2.4 wiring for fallback activation
  - continue-finalize.md wisdom summary with fallback count display
affects: [28-integration-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "jq for file grouping/sorting instead of bash associative arrays (bash 3.2 compatible)"
    - "jq -sc for compact single-line JSON output between pipeline stages"
    - "Cross-stage echo pattern for fallback_count (same as hive_promoted_count)"

key-files:
  created:
    - tests/integration/fallback-extraction.test.js
  modified:
    - .aether/utils/learning.sh
    - .aether/aether-utils.sh
    - .aether/docs/command-playbooks/continue-advance.md
    - .aether/docs/command-playbooks/continue-finalize.md

key-decisions:
  - "Used jq for file grouping/sorting instead of bash associative arrays (bash 3.2 compatibility)"
  - "Added extra parentheses around `or` expressions in jq if-conditions (jq parser requirement)"
  - "Redirected instinct-create stdout to /dev/null inside fallback to avoid polluting JSON output"

patterns-established:
  - "jq-based file categorization and grouping: parse git stat to JSON, filter/categorize/group with jq, iterate categories in bash"
  - "Pre-flight guard pattern: check git HEAD~1 exists and COLONY_STATE.json exists before processing"

requirements-completed: [PIPE-03]

# Metrics
duration: 15min
completed: 2026-03-27
---

# Phase 27 Plan 02: Deterministic Fallback Extraction Summary

**Git-diff-based fallback learning extraction that fires when AI builders skip learning output, producing structured learnings through the existing instinct pipeline**

## Performance

- **Duration:** 15 min
- **Started:** 2026-03-27T13:45:24Z
- **Completed:** 2026-03-27T14:00:00Z
- **Tasks:** 2
- **Files modified:** 4
- **Files created:** 1

## Accomplishments
- Added `_learning_extract_fallback` function: parses git diff stat, filters noise files, categorizes by type (testing/source/documentation/configuration), groups by category, sorts by change magnitude, caps at 5 learnings
- Each fallback learning feeds through instinct-create with confidence 0.5 (same pipeline as AI learnings, no provenance tag)
- Wired Step 2.4 into continue-advance.md: checks patterns_count from COLONY_STATE.json, fires fallback only when empty
- Updated continue-finalize.md wisdom summary: displays "N learnings recorded (M from fallback)"
- 6 integration tests covering: fires on empty learnings, skips trivial changes, respects 5-learning cap, always includes test additions, no-git guard, no-colony-state guard

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement _learning_extract_fallback in learning.sh and register subcommand** - `1861df1` (feat)
2. **Task 2: Wire fallback into continue-advance and continue-finalize, write integration test** - `63cdbb9` (feat)

## Files Created/Modified
- `.aether/utils/learning.sh` - Added _learning_extract_fallback function (git-diff-based learning extraction)
- `.aether/aether-utils.sh` - Registered learning-extract-fallback subcommand in case block
- `.aether/docs/command-playbooks/continue-advance.md` - Added Step 2.4 for deterministic fallback extraction
- `.aether/docs/command-playbooks/continue-finalize.md` - Updated wisdom summary with fallback count
- `tests/integration/fallback-extraction.test.js` - Created 6 integration tests for fallback behavior

## Decisions Made
- Used jq for file grouping/sorting instead of bash associative arrays. macOS ships bash 3.2 which doesn't support `declare -A`. jq is already a project dependency and handles grouping/sorting elegantly.
- Added extra parentheses around `or` expressions in jq if-conditions. jq parses `if A or B then` incorrectly when A is a pipe expression -- `(if (A or B) then ...)` is required.
- Redirected instinct-create stdout to /dev/null inside fallback. The instinct-create subcommand outputs JSON to stdout, which would pollute the fallback's JSON response. Both stdout and stderr are redirected.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed bash 3.2 incompatibility with associative arrays**
- **Found during:** Task 1
- **Issue:** Plan used `declare -A` (associative arrays) for category grouping. macOS bash 3.2 doesn't support this -- produces "invalid option" error.
- **Fix:** Rewrote the grouping logic to use jq instead. Parse git stat to JSON, then use jq `group_by()` and `sort_by()` for categorization and sorting.
- **Files modified:** .aether/utils/learning.sh
- **Committed in:** `1861df1` (part of Task 1 commit)

**2. [Rule 1 - Bug] Fixed git diff stat summary line parsing**
- **Found during:** Task 1
- **Issue:** git diff --stat outputs a summary line like "13 files changed, 697 insertions(+), 128 deletions(-)" at the end. The awk parser treated this as a file path with no `|` separator, producing an entry with the entire summary as a path.
- **Fix:** Added awk guard `if ($0 ~ /files? changed/) next` to skip summary lines.
- **Files modified:** .aether/utils/learning.sh
- **Committed in:** `1861df1` (part of Task 1 commit)

**3. [Rule 1 - Bug] Fixed trailing spaces in git stat file paths**
- **Found during:** Task 1
- **Issue:** git diff --stat pads file paths with trailing spaces for alignment. The awk parser included these trailing spaces in the path, causing `.aether/data/COLONY_STATE.json` filter to fail (trailing spaces prevented `startswith` match).
- **Fix:** Added `gsub(/^[[:space:]]+|[[:space:]]+$/, "", fpath)` to trim whitespace from extracted path.
- **Files modified:** .aether/utils/learning.sh
- **Committed in:** `1861df1` (part of Task 1 commit)

**4. [Rule 1 - Bug] Fixed jq `or` operator parsing in if-conditions**
- **Found during:** Task 1
- **Issue:** jq's `if A or B then` syntax doesn't work when A is a pipe expression like `(.path | test(...))`. jq parses `if (.path | test("a")) or (.path | test("b")) then` as `(if (.path | test("a"))) or ((.path | test("b")) then ...)` which is invalid.
- **Fix:** Wrapped the entire condition in parentheses: `if ((.path | test("a")) or (.path | test("b"))) then`.
- **Files modified:** .aether/utils/learning.sh
- **Committed in:** `1861df1` (part of Task 1 commit)

**5. [Rule 1 - Bug] Fixed jq -s producing multi-line JSON**
- **Found during:** Task 1
- **Issue:** `jq -s '.'` without `-c` flag produces pretty-printed JSON. When piped to a second jq call, the multi-line input causes `.[]` to iterate over lines (strings) instead of array elements, producing "Cannot index string with string" errors.
- **Fix:** Changed to `jq -sc '.'` (compact mode) for single-line JSON output between pipeline stages.
- **Files modified:** .aether/utils/learning.sh
- **Committed in:** `1861df1` (part of Task 1 commit)

**6. [Rule 1 - Bug] Fixed instinct-create JSON polluting fallback output**
- **Found during:** Task 1
- **Issue:** The fallback function calls `bash "$0" instinct-create ... 2>/dev/null` which only redirects stderr. The instinct-create JSON output goes to stdout, mixing with the fallback's own JSON output.
- **Fix:** Changed to `>/dev/null 2>&1` to redirect both stdout and stderr of instinct-create subprocess calls.
- **Files modified:** .aether/utils/learning.sh
- **Committed in:** `1861df1` (part of Task 1 commit)

---

**Total deviations:** 6 auto-fixed (6 bugs)
**Impact on plan:** All auto-fixes were necessary for correctness on bash 3.2 / macOS. No scope creep -- all fixes address the same implementation approach specified in the plan, just with platform-specific adaptations.

## Issues Encountered
- bash 3.2 associative array limitation required complete rewrite of the grouping logic from bash to jq. The jq approach is actually cleaner and more maintainable.
- jq's `or` operator has surprising precedence rules in if-conditions. Extra parentheses are required when the condition contains pipe expressions.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 27 is complete (2/2 plans: fuzzy dedup + fallback extraction)
- 28-integration-validation can proceed -- both learning pipeline features are implemented and tested
- The fallback wiring in continue-advance.md is NON-BLOCKING (failures don't stop the continue flow)

---
*Phase: 27-deterministic-fallback-dedup*
*Completed: 2026-03-27*
