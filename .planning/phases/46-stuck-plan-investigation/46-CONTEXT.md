# Phase 46: Stuck-Plan Investigation and Release Decision - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Investigate the stuck `aether plan` issue and make the v1.6 release decision. This is the final phase in v1.6 — once done, the milestone ships.

Two deliverables:
1. Stuck-plan issue: reproduced or proven stale in freshly updated repos
2. Release decision: all v1.6 requirements verified and milestone audit passes

</domain>

<decisions>
## Implementation Decisions

### Stuck-plan Investigation
- **D-01:** Test stuck `aether plan` in a freshly updated downstream repo only. If it works, document the issue as stale-install fallout resolved by Phases 40-43 pipeline hardening. No need to test in the original problematic repo.

### Release Decision Criteria
- **D-02:** Run a full milestone audit (`/gsd-audit-milestone`) before shipping v1.6. Standard checks (go tests pass, version agreement via `aether version --check`, E2E regression tests pass) plus all phases reviewed against original intent.

### Claude's Discretion
- Exact steps for stuck-plan reproduction (which commands to run, what output to check)
- How to structure the milestone audit report
- Whether to include a version bump commit as part of this phase

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Pipeline Infrastructure
- `cmd/publish_cmd.go` — Publish command, hub version synchronization
- `cmd/update_cmd.go` — Update command, stale publish detection
- `cmd/integrity_cmd.go` — Integrity validation, version chain checking
- `cmd/e2e_regression_test.go` — E2E tests proving pipeline works

### Release Documentation
- `.aether/docs/publish-update-runbook.md` — Publish/update operational guide
- `.aether/docs/AETHER-OPERATIONS-GUIDE.md` — Operations guide (if exists)
- `.planning/ROADMAP.md` — Phase 46 success criteria and milestone progress

### Prior Phase Context
- `.planning/phases/40-stable-publish-hardening/` — Publish hardening (version sync)
- `.planning/phases/41-dev-channel-isolation/` — Channel isolation guards
- `.planning/phases/43-release-integrity-checks/` — Integrity validation
- `.planning/phases/45-e2e-regression-coverage/` — E2E regression tests

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `aether version --check` — Verifies binary and hub version agreement (exit 0 = match)
- `aether integrity` — Validates full release chain (source, binary, hub, companion files)
- `aether publish` — Primary publish command with atomic version sync
- E2E test helpers: `createMockSourceCheckout`, `createHubWithExpectedCounts` — reusable for reproduction tests

### Established Patterns
- Test-only phases use `--skip-build-binary` to avoid fragile go build
- Downstream testing pattern: create temp repo, `aether update --force`, verify files exist
- Version agreement check: `readHubVersionAtPath` compares hub version to expected

### Integration Points
- The stuck plan issue likely manifests in `cmd/plan_cmd.go` — the plan command itself
- Fresh downstream repo test requires: publish from source, then `aether update --force` in target repo
- Milestone audit reads all phase VERIFICATION.md files and cross-references against ROADMAP.md success criteria

</code_context>

<specifics>
## Specific Ideas

- The stuck plan issue was reported before the pipeline hardening. Most likely cause was stale hub state (binary v1.0.20 with hub v1.0.19), which would cause plan to fail or hang on version-dependent logic.
- A clean `aether update --force` in a fresh repo should prove the issue is resolved.
- If the issue IS reproducible, this phase includes shipping a fix with a regression test.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 46-stuck-plan-investigation*
*Context gathered: 2026-04-24*
