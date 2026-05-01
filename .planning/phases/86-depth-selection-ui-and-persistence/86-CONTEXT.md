# Phase 86: Depth Selection UI and Persistence - Context

**Gathered:** 2026-05-01
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase wires the smart depth defaults (from Phase 85) into the user-facing command flow. When `/ant-plan` runs, the user sees a banner showing the auto-selected planning depth and verification depth with reasons. The user can accept both defaults or override either one via flags. The selected verification depth is persisted into the build packet JSON so `/ant-continue` reads it automatically without requiring the user to re-specify.

This is CLI-only work — "UI" means console output formatting and flag-based interaction, not graphical interfaces.

</domain>

<decisions>
## Implementation Decisions

### Plan Output Format
- **D-01:** `/ant-plan` must display a depth selection banner showing both the planning depth and verification depth smart defaults, along with the reason each was selected (e.g., "Phase 3 of 8 (intermediate) — no high-risk keywords detected"). The `renderSmartDepthReason` and `renderReviewDepthLineWithReason` visual helpers from Phase 85 should be wired in for this output.
- **D-02:** The banner should use the existing stage marker style (`── Depth Selection ──`) for visual consistency with the rest of the Aether CLI output.

### Override Mechanism
- **D-03:** Add a `--verification-depth` flag to `/ant-plan` as a direct mirror of the existing `--planning-depth` flag. Both flags accept the standard depth values (light/standard/deep/heavy/full). When provided, the explicit flag overrides the smart default for that depth. When absent, the smart default is used.

### Build Depth Display
- **D-04:** During `/ant-build`, the active verification depth must be shown in the build stage markers (e.g., `── Stage 2: Verification [standard] ──`) and in the build summary output. The `renderReviewDepthLineWithReason` visual helper from Phase 85 should be wired into the stage marker rendering in `cmd/codex_visuals.go`.

### Claude's Discretion
- Exact formatting of the depth selection banner (spacing, label casing, line width)
- Exact placement of the depth indicator in stage markers (prefix vs suffix, bracket style)
- How the depth value flows through the build packet JSON field naming

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Smart Depth System (Phase 85)
- `cmd/review_depth.go` — Contains `resolveSmartPlanningDepth`, `resolveSmartVerificationDepth`, `phasePositionLevel`, `phaseRiskLevel`, and all smart depth logic
- `cmd/review_depth_test.go` — Tests for smart depth functions (44 subtests)
- `cmd/codex_visuals.go` §1066 — Contains `renderSmartDepthReason` and `renderReviewDepthLineWithReason` visual helpers (ready to wire in)

### Plan Command Integration Points
- `cmd/codex_plan.go` §121 — `CodexPlanOptions` struct (needs `VerificationDepth` field added)
- `cmd/codex_plan.go` §613 — `resolvePlanningDepthSmart` function (pattern to follow for verification depth wiring)
- `cmd/codex_plan.go` §150-173 — Plan generation flow where depth is resolved and used

### Build Command Integration Points
- `cmd/codex_build.go` §150 — Where `resolveVerificationDepth` is called during build (reads from build packet)
- `cmd/codex_build.go` §536 — Stage marker rendering where depth should be displayed

### Continue Command Integration Points
- `cmd/codex_continue_finalize.go` §55-57 — Where continue reads `ReviewDepth` from the plan/build packet
- `cmd/codex_continue_plan.go` §93 — Where continue resolves verification depth for dispatch planning

### Requirements
- `.planning/REQUIREMENTS.md` — DEPTH-04 (user depth selection at plan time), DEPTH-05 (depth persistence across continue)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `renderSmartDepthReason(phase, totalPhases)` — Already produces human-readable reason strings (Phase 85)
- `renderReviewDepthLineWithReason(phase, totalPhases)` — Formats a review depth line with reason (Phase 85)
- `resolvePlanningDepthSmart(depth, phase, totalPhases)` — Pattern to follow: wraps resolve function with smart default fallback
- `resolveVerificationDepth(phase, totalPhases, lightFlag, heavyFlag, depthStr)` — Existing verification depth resolver that needs smart default as final fallback (already done in Phase 85)

### Established Patterns
- Plan flags follow `--planning-depth` naming convention — new `--verification-depth` should match
- Stage markers use `── Stage Name ──` format from `cmd/codex_visuals.go`
- Build packet JSON is written during build dispatch and read during continue

### Integration Points
- `CodexPlanOptions` struct needs a new `VerificationDepth` field
- Plan output rendering in `cmd/codex_plan.go` (around line 160-180) needs depth banner insertion
- Build packet write path needs to include the selected verification depth
- Continue read path needs to consume the stored verification depth from build packet

</code_context>

<specifics>
## Specific Ideas

- The depth banner should appear early in `/ant-plan` output, before plan generation begins, so the user sees what was selected before any work happens
- Both depths should be shown side-by-side for easy comparison
- The banner should include a hint about override flags when defaults are shown

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 86-depth-selection-ui-and-persistence*
*Context gathered: 2026-05-01*
