# Roadmap: Aether

## Milestones

- **v1.0 MVP** - Phases 1-6 (shipped)
- **v1.1 Trusted Context** - Phases 7-11 (shipped)
- **v1.2 Live Dispatch Truth and Recovery** - Phases 12-16 (shipped)
- **v1.3 Visual Truth and Core Hardening** - Phases 17-24 (shipped 2026-04-21)
- **v1.4 Self-Healing Colony** - Phases 25-30 (completed 2026-04-21)
- **v1.5 Runtime Truth Recovery** - Phases 31-38 (completed 2026-04-23, product v1.0.20)
- **v1.6 Release Pipeline Integrity** - Phases 39-46 (in progress)

## Phases

<details>
<summary>v1.0 MVP (Phases 1-6) -- SHIPPED</summary>

- Phase 1: Housekeeping and Foundation
- Phase 2: Colony Scope System
- Phase 3: Restore Build Ceremony
- Phase 4: Restore Continue Ceremony
- Phase 5: Living Watch and Status Surfaces
- Phase 6: Pheromone Visibility and Steering

</details>

<details>
<summary>v1.1 Trusted Context (Phases 7-11) -- SHIPPED</summary>

- Phase 7: Context Ledger and Skill Routing Foundation
- Phase 8: Prompt Integrity and Trust Boundaries
- Phase 9: Trust-Weighted Context Assembly
- Phase 10: Curation Spine and Structural Learning
- Phase 11: Competitive Proof Surfaces and Evaluation

</details>

<details>
<summary>v1.2 Live Dispatch Truth and Recovery (Phases 12-16) -- SHIPPED</summary>

- Phase 12: Dispatch Truth Model and Run Scoping
- Phase 13: Live Workflow Visibility Across Colonize, Plan, and Build
- Phase 14: Worker Execution Robustness and Honest Activity Tracking
- Phase 15: Verification-Led Continue and Partial Success
- Phase 16: Recovery, Reconciliation, and Runtime UX Finalization

</details>

<details>
<summary>v1.3 Visual Truth and Core Hardening (Phases 17-24) -- SHIPPED 2026-04-21</summary>

- Phase 17: Slash Command Format Audit
- Phase 18: Visual UX Restoration -- Caste Identity and Spawn Lists
- Phase 19: Visual UX Restoration -- Stage Separators and Ceremony
- Phase 20: Visual UX Restoration -- Emoji Consistency
- Phase 21: Codex CLI Visual Parity
- Phase 22: Core Path Hardening
- Phase 23: Recovery and Continuity
- Phase 24: Full Instrumentation -- Trace Logging

</details>

<details>
<summary>v1.4 Self-Healing Colony (Phases 25-30) -- COMPLETED 2026-04-21</summary>

- Phase 25: Medic Ant Core -- Health diagnosis command, colony data scanner
- Phase 26: Auto-Repair -- Fix common colony data issues with `--fix` flag
- Phase 27: Medic Skill -- Healthy state specification skill file
- Phase 28: Ceremony Integrity -- Verify wrapper/runtime parity
- Phase 29: Trace Diagnostics -- Remote debugging via trace export analysis
- Phase 30: Medic Worker Integration -- Caste integration, auto-spawn

</details>

<details>
<summary>v1.5 Runtime Truth Recovery (Phases 31-38) -- COMPLETED 2026-04-23</summary>

8 phases, 17 plans, 176 commits. P0 runtime truth fixes, continue unblock, dispatch robustness, cleanup, platform parity, v1.0.20 release, codebase hygiene, Nyquist validation. Product version: v1.0.20. [Full archive -> milestones/v1.5-ROADMAP.md]

</details>

<details>
<summary>v1.6 Release Pipeline Integrity (Phases 39-46) -- IN PROGRESS (Phases 44.1, 44.2 inserted)</summary>

### Phase 39: OpenCode Agent Frontmatter Fix
**Goal:** Fix the urgent blocker where Aether ships invalid OpenCode agent frontmatter that crashes OpenCode startup in downstream repos.
**Requirements:** OPN-01 (R068)
**Success Criteria:**
1. OpenCode launches successfully in a repo after `aether update --force`
2. All .opencode/agents/ files have valid OpenCode-schema frontmatter
3. Install/update validates agent frontmatter before writing to downstream repos
4. E2E test proves OpenCode startup does not fail on agent config
**Depends on:** none (urgent blocker)

### Phase 40: Stable Publish Hardening -- COMPLETE 2026-04-23
**Goal:** Ensure stable publish atomically syncs binary and hub to the same version -- no more 1.0.20 binary with 1.0.19 hub.
**Requirements:** PUB-01 (R059)
**Success Criteria:**
1. `aether install --package-dir "$PWD"` sets hub version.json to match source version
2. After publish, `aether version` and `~/.aether/system/version.json` agree
3. Publish fails loudly if binary and hub cannot be synchronized
4. Reproduce the current 1.0.19/1.0.20 mismatch, then prove it's fixed
**Depends on:** Phase 39 (ship frontmatter fix before touching publish)

### Phase 41: Dev-Channel Isolation -- COMPLETE 2026-04-23
**Goal:** Dev publish touches only `aether-dev` and `~/.aether-dev` -- zero contamination of stable channel.
**Requirements:** PUB-02 (R060)
**Success Criteria:**
1. Dev publish does not modify any file under `~/.aether/` or the `aether` binary
2. Stable publish does not modify any file under `~/.aether-dev/` or `aether-dev` binary
3. Both channels can be published independently without interference
4. Test proves channel isolation with concurrent publish scenarios
**Depends on:** Phase 40

### Phase 42: Downstream Stale-Publish Detection
**Goal:** `aether update --force` and `aether-dev update --force` detect and report stale/incomplete publishes instead of silently succeeding.
**Requirements:** PUB-03 (R061), PUB-04 (R061)
**Success Criteria:**
1. Downstream update detects when hub version is older than source/binary version
2. Downstream update reports exactly what is stale (binary, companion files, or both)
3. Update returns non-zero exit code on stale/incomplete publish
4. Works for both stable and dev channels independently
**Depends on:** Phase 40, Phase 41

### Phase 43: Release Integrity Checks and Diagnostics
**Goal:** Single integrity check validates the full chain (source -> binary -> hub -> downstream) and medic flags incomplete publishes with recovery commands.
**Requirements:** REL-01 (R062), REL-02 (R063)
**Success Criteria:**
1. `aether` command validates source version, binary version, hub version, and companion surfaces together
2. Medic flags incomplete stable/dev publishes and prints exact recovery command
3. Integrity check is runnable both locally (source repo) and downstream (consumer repo)
4. Diagnostic output is human-readable and actionable
**Depends on:** Phase 42

### Phase 44: Doc Alignment and Archive Consistency
**Goal:** Operations guide, runbook, and AGENTS.md match actual runtime behavior exactly. Archived milestone evidence is internally consistent.
**Requirements:** REL-03 (R064), EVD-01 (R066)
**Plans:** 2 plans

Plans:
- [ ] 44-01-PLAN.md -- Core docs alignment (operations guide, runbook, CLAUDE.md, CODEX.md, OPENCODE.md)
- [ ] 44-02-PLAN.md -- Agent/skill docs and v1.5 archive consistency

**Success Criteria:**
1. AETHER-OPERATIONS-GUIDE.md verification checklist passes as written
2. publish-update-runbook.md steps match actual command behavior
3. AGENTS.md operator flows are accurate for both channels
4. Archived v1.5 docs no longer contain internal contradictions
5. Any behavior changes from Phases 39-43 are reflected in docs
**Depends on:** Phase 43 (docs must reflect final behavior)

### Phase 44.1: Downstream Runtime Bugs (INSERTED)
**Goal:** Fix three runtime bugs found during downstream testing in Sodalitas: false Codex skills count warning, rigid plan --refresh guard, and low default scout timeout.
**Requirements:** PUB-03 (R061), EVD-02 (R067)
**Success Criteria:**
1. `aether update --force` reports correct Codex skills count (no false "1 found, expected 29")
2. `aether plan --refresh` is allowed when Phase 1 is "ready" with no built artifacts (not blocked)
3. Default scout/worker timeout raised to 15m (configurable)
4. Downstream repro confirms all three fixes work in a real repo
**Depends on:** Phase 44

### Phase 44.2: Command Hygiene and Agent Parity (INSERTED)
**Goal:** Fix aether-medic.md agent body parity mismatch between Claude and OpenCode, and rename all Aether slash commands from colon format (`ant:command`) to hyphen format (`ant-command`) to comply with current Claude Code skill naming rules.
**Requirements:** PUB-03 (R061), REL-01 (R059)
**Plans:** 1 plan

Plans:
- [x] 44.2-01-PLAN.md -- Medic parity fix and colon-to-hyphen rename across repo

**Success Criteria:**
1. `TestClaudeOpenCodeAgentContentParity` passes (aether-medic.md matches between Claude and OpenCode)
2. All 50 slash commands use hyphen format (e.g., `ant-build` not `ant:build`)
3. All command references in YAML sources, wrappers, docs, and tests updated
4. `go test ./...` passes with zero failures
**Depends on:** Phase 44.1

### Phase 45: End-to-End Regression Coverage
**Goal:** Automated E2E tests for stable and dev publish/update flows that catch regressions before they ship.
**Requirements:** REL-04 (R065)
**Plans:** 1 plan

Plans:
- [ ] 45-01-PLAN.md -- Four E2E regression tests for stable/dev publish-update pipeline

**Success Criteria:**
1. E2E test for stable publish -> downstream update -> version agreement
2. E2E test for dev publish -> dev downstream update -> version agreement
3. E2E test for stale publish detection (intentionally stale hub)
4. E2E test for channel isolation (dev publish does not touch stable)
5. Tests runnable in CI (`go test`)
**Depends on:** Phase 43

### Phase 46: Stuck-Plan Investigation and Release Decision
**Goal:** Verify the stuck `aether plan` issue; make the v1.6 release decision.
**Requirements:** EVD-02 (R067)
**Success Criteria:**
1. Stuck `aether plan` issue is reproduced or proven stale in freshly updated repos
2. If reproducible: fix shipped with regression test
3. If stale-install fallout: documented as resolved by pipeline hardening
4. All v1.6 requirements verified and milestone audit passes
5. Source, binary, hub, and downstream versions all agree for both channels
**Depends on:** Phase 44, Phase 45

</details>

## Progress

| Milestone | Phases | Status | Completed |
|-----------|--------|--------|-----------|
| v1.0 | 1-6 | Complete | 2026-04-21 |
| v1.1 | 7-11 | Complete | 2026-04-21 |
| v1.2 | 12-16 | Complete | 2026-04-21 |
| v1.3 | 17-24 | Complete | 2026-04-21 |
| v1.4 | 25-30 | Complete | 2026-04-21 |
| v1.5 | 31-38 | Complete | 2026-04-23 |
| v1.6 | 39-46 | In Progress | -- |
