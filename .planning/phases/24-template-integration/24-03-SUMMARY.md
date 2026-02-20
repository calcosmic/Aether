---
phase: 24-template-integration
plan: 03
subsystem: templates
tags: [handoff, templates, build, distribution]

# Dependency graph
requires:
  - phase: 21-template-foundation
    provides: Template conventions (HTML comment header, {{PLACEHOLDER}} syntax, validate-package.sh registration pattern)
  - phase: 24-template-integration
    provides: 24-01 and 24-02 template wiring patterns (entomb.md wired in Phase 21; build.md heredocs identified in 24-RESEARCH.md)
provides:
  - Two new build HANDOFF templates (error recovery + success summary)
  - build.md (Claude Code) wired to both templates via template-read instructions
  - build.md (OpenCode) wired to both templates via identical template-read instructions
  - validate-package.sh updated to require and distribute both new templates
affects: [build-command-users, template-distribution, handoff-generation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Heredoc replacement with template-read instructions: resolve hub path first, then .aether/, error if missing"
    - "Template-not-found error format: 'Template missing: {name}. Run aether update to fix.'"
    - "jq block preserved untouched when replacing surrounding HANDOFF heredoc"

key-files:
  created:
    - .aether/templates/handoff-build-error.template.md
    - .aether/templates/handoff-build-success.template.md
  modified:
    - bin/validate-package.sh
    - .claude/commands/ant/build.md
    - .opencode/commands/ant/build.md

key-decisions:
  - "Bash code block closing fence added after jq block in both build.md files — original heredoc was inside same code block, replacement is prose so fence needed to be explicit"
  - "Template instructions placed as prose (not code block) — matches approach used for entomb.md in Phase 21"
  - "Both platforms wired simultaneously with identical template read instructions — no platform-specific differences in HANDOFF logic"

patterns-established:
  - "Build HANDOFF templates follow same Phase 21 header convention (<!-- Template: name | Version: 1.0 -->)"
  - "Template resolution order: hub (~/.aether/system/) first, then local (.aether/), error if missing"

requirements-completed: [WIRE-05]

# Metrics
duration: 4min
completed: 2026-02-20
---

# Phase 24 Plan 03: Template Integration — Build HANDOFF Wiring Summary

**Two build HANDOFF templates created and both build.md files wired to templates, eliminating all inline heredocs from the build command on both platforms.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-20T00:02:09Z
- **Completed:** 2026-02-20T00:06:52Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Created handoff-build-error.template.md following Phase 21 conventions (HTML comment header, {{PLACEHOLDER}} syntax, 5 placeholders)
- Created handoff-build-success.template.md following Phase 21 conventions (11 placeholders covering goal, phase, build status, counts)
- Registered both templates in validate-package.sh REQUIRED_FILES array — package validation passes
- Wired Claude Code and OpenCode build.md Step 5.9 (error) and Step 6.5 (success) heredocs to templates with identical instructions
- last-build-result.json jq block preserved intact in both files; all HANDOFF_EOF markers removed

## Task Commits

Each task was committed atomically:

1. **Task 1: Create build HANDOFF templates and register for distribution** - `1b32846` (feat)
2. **Task 2: Wire build.md HANDOFF heredocs to templates (both platforms)** - `b242800` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `.aether/templates/handoff-build-error.template.md` - Build error recovery handoff template with 5 placeholders
- `.aether/templates/handoff-build-success.template.md` - Build success handoff template with 11 placeholders
- `bin/validate-package.sh` - Both new templates added to REQUIRED_FILES array
- `.claude/commands/ant/build.md` - Error (Step 5.9) and success (Step 6.5) HANDOFF heredocs replaced with template-read instructions
- `.opencode/commands/ant/build.md` - Same replacements as Claude Code, applied identically

## Decisions Made

- Bash code block closing fence added explicitly after jq block in both build.md files. The original file had one large `bash` code block containing both the jq write and the HANDOFF heredoc. When the heredoc was replaced with prose, the code block needed explicit closure — the replacement added ` ``` ` after the jq line.
- Template instructions written as prose (not inside a code block), matching the pattern used for entomb.md template wiring in Phase 21.
- Both platforms updated simultaneously with identical template read instructions — no platform-specific differences justified.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added closing code fence after jq block in both build.md files**
- **Found during:** Task 2 (Wire build.md HANDOFF heredocs)
- **Issue:** The original bash code block contained both the jq write and the HANDOFF heredoc. Replacing the heredoc with prose left an unclosed ` ```bash ` fence, which would break markdown rendering.
- **Fix:** Added ` ``` ` closing fence after `}' > .aether/data/last-build-result.json` in both files.
- **Files modified:** `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`
- **Verification:** Code block structure verified by reading the modified sections.
- **Committed in:** b242800 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug)
**Impact on plan:** Necessary for correct markdown rendering. No scope creep.

## Issues Encountered

- Pre-existing 2 test failures in validate-state.test.js (documented in STATE.md, out of scope for this plan)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 24 plan 03 complete — all build HANDOFF heredocs now use templates
- Template integration phase (24) is now complete across all targeted commands
- Templates distributed via validate-package.sh and npm install -g . workflow

---
*Phase: 24-template-integration*
*Completed: 2026-02-20*
