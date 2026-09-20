---
phase: 73
slug: rich-init-research
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-28
---

# Phase 73 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./cmd/ -run TestInitResearch -v -count=1` |
| **Full suite command** | `go test ./... -v -count=1` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run TestInitResearch -v -count=1`
- **After every plan wave:** Run `go test ./... -v -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 73-01-01 | 01 | 1 | INIT-03 | — | N/A | unit | `go test ./cmd/ -run "TestParsePackageJsonDeps\|TestParseGoModDeps\|TestParseCargoTomlDeps\|TestParsePyprojectDeps\|TestParseRequirementsTxt\|TestParseGemfileDeps\|TestParsePomXmlDeps\|TestParseMixExsDeps\|TestParseComposerJsonDeps\|TestInitResearchTechStackDetail" -count=1 -v` | ❌ W0 | ⬜ pending |
| 73-02-01 | 02 | 2 | INIT-04 | — | N/A | unit | `go test ./cmd/ -run "TestClassifyDir" -count=1 -v` | ❌ W0 | ⬜ pending |
| 73-02-02 | 02 | 2 | INIT-05 | — | N/A | unit | `go test ./cmd/ -run "TestDeepParse\|TestGovernanceBackwardCompat" -count=1 -v` | ❌ W0 | ⬜ pending |
| 73-03-01 | 03 | 3 | INIT-06, INIT-07 | — | N/A | unit | `go test ./cmd/ -run "TestPheromone\|TestColonyContextSummary\|TestInitResearchFullOutput" -count=1 -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing Go test infrastructure covers all phase requirements.

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
