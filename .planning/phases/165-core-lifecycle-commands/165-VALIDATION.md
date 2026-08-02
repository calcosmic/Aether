---
phase: 165
slug: core-lifecycle-commands
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-02
---

# Phase 165 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib), no external test framework |
| **Config file** | none — plain `go test` |
| **Quick run command** | `go test ./cmd/... -run 'TestBuildWrapperCeremonyContract|TestContinueWrapperCeremonyContract|TestPlanWrapperCeremonyContract|TestPlanWrapperCardsParity|TestLifecycleCommandDocsPreferRuntimeCLI|TestBriefPathReferencedAcrossAllFourSurfaces|TestBuildWrapperVerbatimBriefBulletsStayMirrored|TestContinueWrapperSourcesUseFastDevContinue'` |
| **Full suite command** | `go test ./cmd/...` plus `go vet ./...` and `go build ./cmd/aether` |
| **Estimated runtime** | ~2 seconds (quick), ~60 seconds (full) |

---

## Sampling Rate

- **After every task commit:** Run the targeted test for the wrapper/test file the task touches (`go test ./cmd/... -run '<TestNameForThisTask>'`)
- **After every plan wave:** Run `go test ./cmd/...` plus `go vet ./...`
- **Before `/gsd-verify-work`:** Full suite must be green and `go build ./cmd/aether` succeeds
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD (planner fills) | — | — | CMD-01 | — | Stage skeleton (colony beat/Purpose/Reads/Spawns/Stop conditions) in all four wrappers | unit | `go test ./cmd/... -run TestBuildWrapperCeremonyContract` (extended, proportion assertions) | ❌ W0 (extensions) | ⬜ pending |
| TBD | — | — | CMD-01 | — | `<success_criteria>/<failure_modes>/<read_only>` blocks in all four wrappers | unit | new `TestLifecycleWrappersCarryStructuredBlocks` | ❌ W0 | ⬜ pending |
| TBD | — | — | CMD-01 | — | State-carry contract uses current vocabulary; retired `colony_depth` terms forbidden | unit | extend `platform_doc_hygiene_test.go` forbidden lists | ✅ (extend) | ⬜ pending |
| TBD | — | — | CMD-02 | — | Zero envelope-parsing-as-primary-job phrasing in the four wrappers | unit | new `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob` | ❌ W0 | ⬜ pending |
| TBD | — | — | CMD-04 | — | 8 specialist commands unchanged | unit + diff | `go test ./cmd/... -run TestLifecycleCommandDocsPreferRuntimeCLI` + `git diff --stat` on the 8 files empty before final commit | ✅ partial | ⬜ pending |
| TBD | — | — | CMD-05 | — | build.md ownership header + Phase-168 reserved trailer marker present | unit | new `TestBuildMdOwnershipHandshake` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/init_wrapper_ceremony_test.go` — new file; init.md has zero dedicated ceremony-contract test today. Mirror `build_wrapper_ceremony_test.go` shape: required/inOrder/forbidden lists + heading parity vs `.opencode/commands/ant/init.md`
- [ ] Extend `cmd/build_wrapper_ceremony_test.go` — heading parity vs `.opencode` mirror, D-08 ownership header/trailer assertion, stage-skeleton proportion assertion
- [ ] Extend `cmd/continue_wrapper_ceremony_test.go` — heading parity, stage-skeleton proportion
- [ ] Extend `cmd/plan_wrapper_ceremony_test.go` — stage-skeleton proportion (parity already covered by `plan_wrapper_cards_test.go`)
- [ ] Resolve flat-mirror test gap — either extend `wrapperPaths` to include `.claude/commands/ant-build.md` / `ant-init.md` (matching plan/continue) or add an explicit test documenting exclusion

*No framework install needed — Go stdlib testing is already fully wired for this package.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| A user reading `build.md` can describe what each stage does and what context/research a worker receives, without opening Go source | CMD-03 | Inherently qualitative prose-clarity criterion, not a parseable invariant | Read `build.md` top-to-bottom during `/gsd-verify-work`; for each stage, state its purpose and the worker's inputs in one sentence without consulting `cmd/*.go` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
