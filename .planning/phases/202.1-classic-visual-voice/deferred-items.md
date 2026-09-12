## Orchestration lesson (recorded 2026-09-12)

Do NOT run the whole-suite gate while executor agents are running. This repo's
full-suite controller uses a `serial-shared-checkout` lane, and executors in
their own worktrees still share the repo root and the `~/.aether/` hub. A
confirmation run started alongside four Wave 2 executors reported
`full-suite controller failed: lane serial-shared-checkout: exit status 1` and
produced two failures that pass in isolation (`TestSkillIsUserCreatedShipped`
and the controller lane itself). Gate runs must be serialised against dispatch.

## Pre-existing failures discovered during Plan 03 full-suite verification

Neither touches a file this plan modified (`cmd/discuss.go`, `cmd/spec_cmd.go`,
`cmd/classic_voice_discuss_spec_test.go`) — confirmed by `git status --short`
showing no other files changed. Not auto-fixed (Scope Boundary). Recorded in
`.planning/WINDOWS.md`.

1. `TestHumanFacingOutputGoesThroughWriteVisualOutput` — `cmd/watch_live.go`'s
   `runColonyLiveRefreshLoop` writes directly to stdout/stderr at lines
   425/426/440, bypassing `writeVisualOutput`. Pre-existing; consistently
   reproducible in isolation.
2. `TestNoWorkerWithoutStatedReason` — already tracked as WINDOWS entry #11
   (Phase 201, `cmd/queen_judgement_test.go`): a Queen-requested Measurer with
   a stated reason is dropped before spawn. Reconfirmed still open.

Every test named in either task's `<acceptance_criteria>`/`<verify>` block
passes; every test directly touching the three files this plan changed
passes (`TestDiscuss*`, `TestSpec*`, `TestVoicedScreensSpeakPlainEnglish`,
`TestCodexVisualsSpecIdentityContract`, `TestMigratedLifecycleSurfaces*`,
`TestTheSevenClosingsSpeakPlainEnglish`).

## Reconciliation: `.planning/WINDOWS.md` is the authoritative register

Plan 03 found the project's existing Broken Windows Ledger, which the
orchestrator's own baseline note above did not consult. **Entry #12 (Phase 202,
opened 2026-09-11) already lists roughly twenty failing tests on this branch**,
including every test the orchestrator independently re-derived at `2757534e`:
`TestCurrentVocabulary199`, `TestGoldenBuildVisualOutput`,
`TestGoldenContinueVisualOutput`, `TestPhase199GateReceipt`,
`TestAuditCatalogGolden`, `TestNoWorkerWithoutStatedReason`,
`TestHumanFacingOutputGoesThroughWriteVisualOutput`, `TestPlanningAdversarial200`
— plus `TestNextActionNeverHardcoded`, which this phase has now FIXED
(`e2d64cd2`) and which entry #12 should therefore be able to drop.

So the "known-red baseline" recorded here is not a discovery; it is an
independent re-derivation of a list the project already maintained. Future
waves should read `WINDOWS.md` first. The value of the re-derivation is that it
is commit-anchored: each one was re-run at `2757534e` and fails there with
byte-identical output, which the ledger entry does not record.

