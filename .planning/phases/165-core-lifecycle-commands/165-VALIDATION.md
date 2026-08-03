---
phase: 165
slug: core-lifecycle-commands
status: complete
nyquist_compliant: true
wave_0_complete: true
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
| **Estimated runtime** | ~2 seconds (quick), ~5-7 minutes (full — measured 275-440s across Wave 2/3 executor runs) |

---

## Sampling Rate

- **After every task commit:** Run the targeted test for the wrapper/test file the task touches (`go test ./cmd/... -run '<TestNameForThisTask>'`)
- **After every plan wave:** Run `go test ./cmd/...` plus `go vet ./...`
- **Before `/gsd-verify-work`:** Full suite must be green and `go build ./cmd/aether` succeeds
- **Max feedback latency:** 60 seconds (quick command); the full suite is sampled once per wave, within its own cadence

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 165-02 T1, 165-03 T1, 165-04 T1, 165-05 T1 | 02, 03, 04, 05 | 2 | CMD-01 | T-165-06-02 | Per-wrapper stage skeleton (colony beat/Purpose/Reads/Spawns/Stop conditions) density | unit | `go test ./cmd/... -run 'TestBuildWrapperStageSkeletonAndParity|TestContinueWrapperStageSkeletonAndParity|TestPlanWrapperStageSkeleton|TestInitWrapperStageSkeletonAndParity'` | ✅ | ✅ green |
| 165-06 T1 | 06 | 3 | CMD-01 | T-165-06-02 | Cross-wrapper structured-block markers (`<success_criteria>`/`<failure_modes>`/`<read_only>`/`## Required Cross-Stage State`) present in all eight canonical files | unit | `go test ./cmd/... -run TestLifecycleWrappersCarryStructuredBlocks` | ✅ | ✅ green |
| 165-01 T2, 165-06 T1 | 01, 06 | 1, 3 | CMD-01 | T-165-06-02 | State-carry contract uses current vocabulary; retired `colony_depth`/`visual_mode`/`verbose_mode`/`suggest_enabled`/`synthesis_status` terms forbidden | unit | `go test ./cmd/... -run 'TestLifecycleWrappersAvoidRetiredDepthVocabulary|TestLifecycleWrappersCarryStructuredBlocks'` | ✅ | ✅ green |
| 165-06 T1 | 06 | 3 | CMD-02 | T-165-06-02 | Method marker lines outnumber envelope-mechanics marker lines by >=3:1 in all eight canonical wrappers; contract pointer referenced exactly once (zero for init) | unit | `go test ./cmd/... -run TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob` | ✅ | ✅ green |
| 165-06 T2 | 06 | 3 | CMD-04 | T-165-06-03 | 17 specialist/delight surfaces (7 Claude wrappers + 7 OpenCode mirrors + 3 sage agent definitions) pinned by SHA-256, count asserted at exactly 17, 7 command-guide verbs resolve in-process | unit + hash ledger | `go test ./cmd/... -run TestSpecialistCommandSurfacesUnchanged` | ✅ | ✅ green |
| 165-02 T1 | 02 | 2 | CMD-05 | T-165-06-01 | build.md ownership header (PHASE-160 record) + Phase-168 reserved trailer marker present, in order | unit | `go test ./cmd/... -run TestBuildMdOwnershipHandshake` | ✅ | ✅ green |

*Status legend: ⬜=not-yet-run · ✅=green · ❌=red · ⚠️=flaky (all rows above are ✅ green; no row is outstanding)*

---

## Wave 0 Requirements

- [x] `cmd/init_wrapper_ceremony_test.go` — created by **165-05 T1** (`5f46a530`): `TestInitWrapperCeremonyContract` + `TestInitWrapperStageSkeletonAndParity`, mirroring `build_wrapper_ceremony_test.go`'s required/inOrder/forbidden shape plus heading parity vs `.opencode/commands/ant/init.md`
- [x] Extend `cmd/build_wrapper_ceremony_test.go` — done by **165-02 T1** (`4ec50399`): heading parity vs `.opencode` mirror, `TestBuildMdOwnershipHandshake` (D-08 header/trailer), `TestBuildWrapperStageSkeletonAndParity` (proportion assertion)
- [x] Extend `cmd/continue_wrapper_ceremony_test.go` — done by **165-03 T1** (`e8aa6981`): heading parity, `TestContinueWrapperStageSkeletonAndParity` (5 subtests)
- [x] Extend `cmd/plan_wrapper_ceremony_test.go` — done by **165-04 T1** (`178108bb`): `TestPlanWrapperStageSkeleton` (4 subtests; parity already covered by `plan_wrapper_cards_test.go`)
- [x] Resolve flat-mirror test gap — done by **165-01 T2/T3** (`0834b9d3`, `ba8874e4`): `TestLifecycleFlatMirrorsMatchCanonical` extends coverage to `.claude/commands/ant-build.md`/`ant-init.md` (matching plan/continue), and both mirrors resynced byte-for-byte

*No framework install needed — Go stdlib testing is already fully wired for this package.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions | Verdict | Date |
|----------|-------------|------------|-------------------|---------|------|
| A user reading `build.md` can describe what each stage does and what context/research a worker receives, without opening Go source | CMD-03 | Inherently qualitative prose-clarity criterion, not a parseable invariant | Read `build.md` top-to-bottom; for each stage, state its purpose and the worker's inputs in one sentence without consulting `cmd/*.go` | **Approved** — all nine stages (Colony Context, Active Signals, Phase Framing, Dispatch Manifest, Guided Boundary Gate, Runtime Spawn Ceremony, Worker Spawning, Finalize, After the Build) were describable in one sentence each for purpose and worker inputs, without opening any file under `cmd/`; the file reads in Queen/colony voice, not protocol specification. No stages flagged for another pass. | 2026-08-03 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** Approved (2026-08-03) — CMD-03 human read-through checkpoint confirmed by the reviewer; no stage required another pass. All six per-task/CMD-04 automated assertions plus the CMD-03 manual verdict are recorded above with real test names, task references, and commit hashes.
