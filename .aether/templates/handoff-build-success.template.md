<!-- Template: handoff-build-success | Version: 1.0 -->
<!-- Instructions: Fill all {{PLACEHOLDER}} values with real colony data. Remove this comment block before writing to .aether/HANDOFF.md -->

# Colony Session — Build Complete

## Quick Resume

Run `/ant:continue` to advance phase, or `/ant:resume-colony` to restore full context.

## State at Build Completion

- Goal: "{{GOAL}}"
- Phase: {{PHASE_NUMBER}} — {{PHASE_NAME}}
- Build Status: {{BUILD_STATUS}}
- Updated: {{BUILD_TIMESTAMP}}

## Build Summary

{{BUILD_SUMMARY}}

## Tasks

- Completed: {{TASKS_COMPLETED}}
- Failed: {{TASKS_FAILED}}

## Files Changed

- Created: {{FILES_CREATED}} files
- Modified: {{FILES_MODIFIED}} files

## Next Steps

- If verification passed: `/ant:continue` to advance to next phase
- If issues found: `/ant:flags` to review blockers
- To pause: `/ant:pause-colony`

## Session Note

{{SESSION_NOTE}}
