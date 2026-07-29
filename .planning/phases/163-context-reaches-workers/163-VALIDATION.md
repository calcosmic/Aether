---
phase: 163
slug: context-reaches-workers
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-29
---

# Phase 163 — Validation Strategy

> Per-phase validation contract, derived from `163-RESEARCH.md` § Validation
> Architecture. Requirement → command map; every requirement keeps at least one
> command that fails when the requirement is unmet (CLAUDE.md Definition of Done).

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) via `go test` |
| **Quick run** | `go test ./cmd/... -run 'TestBuildWorkerBrief\|TestHookPreToolUse\|TestCharter\|TestPrintBrief\|TestSuggestAnalyze\|TestPermissionProfile' -count=1` |
| **Full suite** | `go test ./... -count=1 -race` |

Phase-gate extras: `go build ./cmd/aether`, `go vet ./...`, `aether integrity`.

## Sampling Rate

- **Per task commit:** the specific new test(s) for that requirement
- **Per wave merge:** `go test ./cmd/... ./pkg/codex/... -count=1`
- **Phase gate:** full suite + race + build + vet + integrity green before `/gsd-verify-work`

## Requirement → Test Map

| Requirement | Behavior | Command | Exists |
|---|---|---|---|
| CONTEXT-01/04 (reworded per D-07) | Survey + phase research present in briefs on both dispatch paths | `go test ./cmd -run TestBuildWorkerBriefIncludesSurveyAndResearch -count=1` | ⚠️ partial coverage may exist — planner greps existing `resolveSurveySection`/`resolvePhaseResearchSection` assertions before writing new (research Pitfall 5) |
| CONTEXT-02 | Colony-prime capsule reaches the wrapper-manifest path, sourced from `resolveCodexWorkerContext()` | `go test ./cmd -run TestBuildManifestCarriesContextCapsuleOnce -count=1` | ❌ W0 |
| CONTEXT-03 | Capsule carried once at manifest level, not duplicated per dispatch | same test — asserts `dispatch.Brief` lacks the capsule marker for N>1 dispatches | ❌ W0 |
| CONTEXT-05 | `suggest-analyze` runs at build-finalize; suggestions surface end-of-build (D-11) | `go test ./cmd -run TestBuildFinalizeCollectsSuggestAnalyzeResults -count=1` | ❌ W0 |
| CONTEXT-06 | Charter rules in briefs | `go test ./cmd -run TestBuildWorkerBriefIncludesCharter -count=1` | ❌ W0 |
| CONTEXT-06/D-09 | Mechanically-checkable charter rules gate at continue | `go test ./cmd -run TestContinueCharterComplianceGate -count=1` | ❌ W0 |
| CONTEXT-07/D-06 | `--print-brief` checklist default + `--full` raw + budget totals | `go test ./cmd -run TestPrintBriefChecklistDefaultAndFullFlag -count=1` | ❌ W0 (existing `printWorkerBriefs` has zero coverage today) |
| CONTEXT-08/D-03 | Assembled context measured and bounded against a real budget | `go test ./cmd -run TestAssembledContextStaysUnderBudgetCeiling -count=1` (style model: `TestBuildWorkerBriefIsMostlyTask`, cmd/codex_build_test.go:3186) | ❌ W0 |
| CONTEXT-09 (descoped per D-08) | Validation via inspector + presence tests; no staged benchmark | covered by the CONTEXT-07 and presence tests above | — |
| D-04 | Sanctioned scratch dirs writable; rest of `.aether/data/` still blocked | `go test ./cmd -run TestHookPreToolUseAllowsSanctionedScratchDirs -count=1`; existing `TestHookPreToolUseBlocksProtectedPath` (cmd/hook_cmds_test.go:34) must pass **unmodified** as the negative case | ❌ W0 / ✅ negative |
| D-05a | Scout permission profile permits the write its brief orders | `go test ./pkg/codex -run TestScoutPermissionProfileAllowsPhaseResearchWrite -count=1` | ❌ W0 |
| D-05b | Artifacts schema accepts named fields (not `{}`-only, not `additionalProperties:true`) | `go test ./pkg/codex -run TestWorkerArtifactsSchemaAcceptsNamedFields -count=1` | ❌ W0 |
| D-10 | Staleness warning renders in brief + inspector | `go test ./cmd -run TestBuildWorkerBriefWarnsOnStaleSurvey -count=1` | ❌ W0 |

## Security guardrails (from research § Security Domain — binding on plans)

- D-04 carve-out is an **explicit allowlist of exact subpaths**, never a broadened
  substring match; `TestHookPreToolUseBlocksProtectedPath` stays green unmodified.
- D-05b fix adds **named, typed properties**; flipping `additionalProperties` to
  `true` is prohibited.
- Charter enters prompts as a `colonyPrimeSection` so it inherits the existing
  `colony.AssessPromptSource` integrity pipeline — no direct string concat.

## Wave 0 Gaps

- [ ] `cmd/build_print_brief_test.go` — new file (zero existing coverage)
- [ ] `cmd/hook_cmds_test.go` — sanctioned-dir companion test
- [ ] Manifest-level capsule test (CONTEXT-02/03)
- [ ] `pkg/codex/permission_profile_test.go` — scout contradiction test
- [ ] `pkg/codex/worker_test.go` — artifacts named-fields test
- [ ] `cmd/codex_build_finalize_test.go` — suggest-analyze collection test
- [ ] Charter-in-brief + charter-gate tests (copy `checkAntiPatternGate` shape)

## Manual-Only Verifications

| Behavior | Why manual | Instructions |
|---|---|---|
| Brief checklist is legible at a glance | terminal-readability judgement | run `aether build <n> --print-brief` on a real colony; can you answer "did the charter arrive?" in ten seconds |
| Real-world cheap-model improvement (D-08) | deliberately descoped to lived usage | user develops real repos on a cheap model after the phase ships |

## Validation Sign-Off

- [x] All tasks have automated verify or Wave 0 dependency — checker's Dimension 8 pass: 18/18 tasks, one `<automated>` command each
- [x] No 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references — all 7 gaps assigned to tasks
- [x] No watch-mode flags; feedback latency < 30s per task
- [x] `nyquist_compliant: true` set

**Approval:** approved 2026-07-29 (plan-checker: 1 mechanical blocker fixed, 3 warnings resolved)
