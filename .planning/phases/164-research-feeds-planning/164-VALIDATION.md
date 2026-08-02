---
phase: 164
slug: research-feeds-planning
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-02
updated: 2026-08-02
---

# Phase 164 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Go framework** | Go standard `testing` package |
| **Go config file** | none — `go test ./...` at repo root |
| **TS framework** | Node's built-in `node:test` runner via `tsx` (there is **no vitest** in this repo — the earlier draft of this file was wrong; `.aether/ts-host/package.json` is authoritative) |
| **TS config file** | `.aether/ts-host/package.json` (`test`, `test:all` scripts) |
| **Quick run (Go)** | `go test ./cmd/... -run 'TestPhaseResearch\|TestResearch\|TestDepthProposal\|TestPlanManifest\|TestOracleEscalation\|TestPlanWrapperCards\|TestRouteSetterBrief\|TestQueenResearch\|TestReplanReResearches\|TestRenderPhaseResearchBrief\|TestFastPresetSkips\|TestPlanResearchApprove' -count=1` |
| **Quick run (TS, one file)** | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/<name>.test.ts` |
| **Full suite (Go)** | `go test ./... -race` |
| **Full suite (TS)** | `npm --prefix .aether/ts-host run test:all` |
| **Estimated runtime** | Go targeted ~15s, TS targeted ~5s, full Go suite ~90s |

---

## Sampling Rate

- **After every task commit:** the task's own `<automated>` command (each is a targeted `-run` filter or a single TS test file; all run in under 60s)
- **After every plan wave:** `go test ./cmd/... -count=1` plus `npm --prefix .aether/ts-host run test:all`
- **Before `/gsd-verify-work`:** `go test ./... -race` and `npm --prefix .aether/ts-host run test:all` both green, plus `go build ./cmd/aether`
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-T1 | 164-01 | 1 | RESEARCH-05 | T-164-01 | Survey values interpolated into the brief are runtime-derived, never raw user free-text | unit (Go) | `go test ./cmd/... -run 'TestRenderPhaseResearchBriefIncludesSurvey\|TestPlanEmitsPhaseResearchDispatchesFromDraft\|TestPlanFastPresetSkipsPhaseResearch' -count=1` | ✅ extend | ✅ green |
| 01-T2 | 164-01 | 1 | RESEARCH-04 | T-164-03 | Re-research bounded to iteration 1 of a refresh run | unit (Go) | `go test ./cmd/... -run 'TestReplanReResearchesPhases\|TestPhaseResearchDispatchedOncePerPhase\|TestPlanFastPresetSkipsPhaseResearch' -count=1` | ✅ extend | ✅ green |
| 02-T1 | 164-02 | 1 | RESEARCH-01, RESEARCH-02 | T-164-04, T-164-05 | Reason text built from a fixed lookup table, not raw phase description | unit (Go) | `go test ./cmd/... -run 'TestQueenResearchDecision' -count=1` | ❌ new | ✅ green |
| 02-T2 | 164-02 | 1 | RESEARCH-01 | T-164-06 | Override direction durable in `PendingDecision.Resolution` | unit (Go) | `go test ./cmd/... -run 'TestResearchProposalBatchRendersTickToApprove\|TestResearchDecisionRecordsUseExistingStore\|TestQueenResearchDecision' -count=1` | ❌ new | ✅ green |
| 03-T1 | 164-03 | 1 | RESEARCH-08 | — | Depth presets are the default; CLI flags are the only override | unit (TS) | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence.test.ts` | ❌ new | ✅ green |
| 03-T2 | 164-03 | 1 | RESEARCH-07 | T-164-07, T-164-08, T-164-09 | Evidence dominates the score; cited paths resolved inside repoRoot only | unit (TS) | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence.test.ts` | ❌ new | ✅ green |
| 04-T1 | 164-04 | 1 | RESEARCH-09, RESEARCH-10 | T-164-10 | Goal text is measured, never interpolated into reason strings | unit (Go) | `go test ./cmd/... -run 'TestDepthProposalCardKnobs\|TestDepthProposalReasons' -count=1` | ❌ new | ✅ green |
| 04-T2 | 164-04 | 1 | RESEARCH-09, RESEARCH-10 | T-164-11, T-164-12 | Card emits only fixed-set option values; no free-text prompt | unit (Go) | `go test ./cmd/... -run 'TestDepthProposal' -count=1` | ❌ new | ✅ green |
| 05-T1 | 164-05 | 2 | RESEARCH-01, RESEARCH-06 | T-164-13, T-164-14 | `--flip` IDs matched against the in-scope candidate set; scope guard on read and write | unit (Go, CLI) | `go test ./cmd/... -run 'TestPlanResearchApproveRecordsDecisions' -count=1` | ✅ extend | ✅ green |
| 05-T2 | 164-05 | 2 | RESEARCH-01, RESEARCH-06 | T-164-15 | Unanswered batch warns loudly and never blocks | unit (Go) | `go test ./cmd/... -run 'TestPlanManifestCarriesResearchProposal\|TestPlanManifestWarnsWhenResearchBatchUnanswered' -count=1` | ✅ extend | ✅ green |
| 05-T3 | 164-05 | 2 | RESEARCH-02 | T-164-15 | Nil/empty approvals dispatch nothing | unit (Go) | `go test ./cmd/... -run 'TestResearchDispatchGatedOnApproval\|TestFastPresetSkipsUnlessUserFlipsResearchOn\|TestPhaseResearch\|TestReplanReResearchesPhases' -count=1` | ✅ extend | ✅ green |
| 06-T1 | 164-06 | 2 | RESEARCH-07, RESEARCH-08 | T-164-16, T-164-17 | Iteration and budget caps enforced by `ConfidenceLoop`; flag values clamped to documented ranges | typecheck (TS) | `npm --prefix .aether/ts-host run typecheck` | ✅ exists | ✅ green |
| 06-T2 | 164-06 | 2 | RESEARCH-07, RESEARCH-08 | T-164-18 | Ceremony carries metrics only, never research body text | unit (TS) | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence-loop.test.ts` | ❌ new | ✅ green |
| 07-T1 | 164-07 | 3 | RESEARCH-03 | T-164-19, T-164-20 | Research injected under an explicit heading and bounded to 12000 chars | unit (Go) | `go test ./cmd/... -run 'TestRouteSetterBriefIncludesResearchContent\|TestRouteSetterBriefResearchIsBounded' -count=1` | ✅ extend | ✅ green |
| 07-T2 | 164-07 | 3 | RESEARCH-03 | T-164-21 | Failure recorded in two independent places; finalize never errors | unit (Go) | `go test ./cmd/... -run 'TestResearchWorkerFailureWarnsLoudlyWithoutBlocking\|TestPhaseResearchPreserve' -count=1` | ✅ extend | ✅ green |
| 08-T1 | 164-08 | 3 | RESEARCH-07 | T-164-22, T-164-23 | Oracle caste gains a scoped write restriction before escalation ships | unit (Go) | `go test ./cmd/... ./pkg/codex/... -run 'TestOracleEscalation\|TestPermissionProfile' -count=1` | ❌ new | ✅ green |
| 08-T2 | 164-08 | 3 | RESEARCH-07 | T-164-24 | One escalation per phase per run; no nested loop | unit (TS) | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-confidence-loop.test.ts` | ✅ extend | ✅ green |
| 09-T1 | 164-09 | 4 | RESEARCH-09, RESEARCH-10 | T-164-25 | Only fixed-set depth values reach CLI flags | unit (Go) | `go test ./cmd/... -run 'TestPlanManifestCarriesDepthProposal\|TestDepthProposal' -count=1` | ❌ new | ✅ green |
| 09-T2 | 164-09 | 4 | RESEARCH-01, RESEARCH-09 | T-164-26 | Wrapper prints runtime-emitted cards; composes none itself | CLI | `go run ./cmd/aether command-guide plan --platform codex` | ✅ exists | ✅ green |
| 09-T3 | 164-09 | 4 | RESEARCH-01, RESEARCH-10 | T-164-26, T-164-27 | Ordered-heading-set equality catches one-sided wrapper drift | unit (Go) | `go test ./cmd/... -run 'TestPlanWrapperCardsParity' -count=1` | ❌ new | ✅ green |
| 10-T1 | 164-10 | gap 1 | RESEARCH-04, RESEARCH-05 | 164-10/T-01 | `toWorkerDispatches` copies every Go-emitted field through; invariant test fails on any future dropped field | unit (TS) | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/dispatch-field-fidelity.test.ts` | ✅ exists | ✅ green |
| 10-T2 | 164-10 | gap 1 | RESEARCH-01, RESEARCH-02 | 164-10/T-01, 164-10/T-02 | A real plan-time research dispatch resolves at the Go permission boundary for caste scout; broadened profiles are rejected | unit (Go) | `go test ./cmd/... -run 'TestPlanResearchDispatchResolvesAtWorkerBoundary\|TestScoutReadOnlyProfileIsRejectedAtWorkerBoundary\|TestPlanResearchDispatchCarriesSixSectionMission' -count=1` | ✅ exists | ✅ green |
| 11-T1 | 164-11 | gap 2 | RESEARCH-01, RESEARCH-02 | 164-11/T-06 | Escalate resolves phases from the in-progress planning iteration state (fresh-colony empty `Plan.Phases`), with colony-plan fallback | unit (Go, CLI) | `go test ./cmd/... -run 'TestOracleEscalation' -count=1` | ✅ exists | ✅ green |
| 11-T2 | 164-11 | gap 2 | RESEARCH-07 | 164-11/T-05 | A failing escalation degrades to a warning; the plan run survives a real non-zero binary exit | unit (TS, real binary) | `npm --prefix .aether/ts-host exec -- tsx --test .aether/ts-host/test/research-escalation-degrade.test.ts` | ✅ exists | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test framework,
runner or dependency is installed by this phase.

- Go: `go test ./...` already runs `cmd/` and `pkg/codex/` tests; new test files
  are added to the existing package.
- TS: `.aether/ts-host/package.json` already defines `test` and `test:all`
  scripts driving `node --test` via `tsx`; new test files land in
  `.aether/ts-host/test/` and are picked up by the existing `test/*.test.ts` glob.
- Every task in every plan carries an `<automated>` command. No task depends on a
  test file that does not exist by the time that task runs (files marked ❌ new
  are created by the task that first references them).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live confidence readout visible during a plan run | RESEARCH-07 | Terminal ceremony rendering during a live multi-worker run; the automated test asserts the lines are emitted, not that they are legible | Run `/ant-plan` at deep depth on a colony with at least two research-warranted phases; watch for per-phase `── ... confidence NN% (delta ...) ──` lines climbing while research runs |
| Two-tap plan flow feels like selection, not a form | RESEARCH-10 | Interaction quality is a human judgement | Run `/ant-plan`; confirm exactly two cards appear, that accepting each is a single confirmation, and that changing a knob needs only a knob name and a number |
| Caste colour and emoji render for research Scouts and an escalated Oracle | RESEARCH-07 (D-09) | ANSI colour rendering in a real terminal | Run `/ant-plan` at deep depth with `AETHER_FORCE_COLOR=1`; confirm the Scout lines and any Oracle escalation line carry the caste emoji, colour and deterministic ant name |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planner-signed 2026-08-02

---

## Validation Audit 2026-08-02

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All 20 plan-time task rows verified green by executing their mapped automated
commands (13 named Go tests confirmed to run individually, 17+13 TS tests pass,
typecheck clean, `command-guide plan` probe exits zero). Four rows added for
gap-closure plans 164-10/164-11, whose tests (7 TS + 5 Go, including a
real-binary failure drill) all pass. Full `go test ./...` and the 555-test
ts-host suite were green at post-merge gates earlier today.
