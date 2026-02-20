<!-- Template: handoff-build-error | Version: 1.0 -->
<!-- Instructions: Fill all {{PLACEHOLDER}} values with real colony data. Remove this comment block before writing to .aether/HANDOFF.md -->

# Colony Session — Build Errors

## Build Status: ISSUES DETECTED

**Phase:** {{PHASE_NUMBER}} — {{PHASE_NAME}}
**Status:** Build completed with failures
**Updated:** {{BUILD_TIMESTAMP}}

## Failed Workers

{{FAILED_WORKERS}}

## Grave Markers Placed

{{GRAVE_MARKERS}}

## Recovery Options

1. Review failures: Check `.aether/data/activity.log`
2. Fix and retry: `/ant:build {{PHASE_NUMBER}}`
3. Swarm fix: `/ant:swarm` for auto-repair
4. Manual fix: Address issues, then `/ant:continue`

## Session Note

Build completed but workers failed. Grave markers placed.
Review failures before advancing.
