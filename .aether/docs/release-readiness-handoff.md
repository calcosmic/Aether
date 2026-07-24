# Release Readiness Handoff

## Purpose

This handoff captures the final local smoke and residual-risk expectations before
sealing the current TypeScript-host, command-parity, and release-readiness
cleanup work.

For beginners: this is the final checklist before saying the framework is ready
to ship. It proves the engine still builds, the release path still works, and
auth failures do not accidentally print secrets.

## Required Local Smoke

Run these before seal signoff:

```bash
npm --prefix .aether/ts-host ci
npm --prefix .aether/ts-host run typecheck
npm --prefix .aether/ts-host test
npm --prefix .aether/ts-host run build
go test ./...
go test ./... -race
go vet ./...
go build ./cmd/aether
goreleaser check
AETHER_RELEASE_VERSION="$(node -p "require('./.aether/version.json').version")" goreleaser release --snapshot --clean
AETHER_RELEASE_ACCEPTANCE_DIR=dist go test ./cmd -run '^TestStagedGoReleaserArtifactsInstallThroughPackedNPM$' -count=1 -v
```

The snapshot command assembles the archives and `checksums.txt` consumed by the
npm bootstrap; a raw `goreleaser build` is not sufficient release evidence.
After the snapshot build, run the produced binary's `version` command. CI and
the release workflow use:

```bash
./dist/Aether_linux_amd64_v1/aether version
```

For a local non-Linux run, use the matching produced binary path under `dist/`.

## Classic Command Parity Evidence

Before release signoff, prove the classic command parity matrix and TypeScript
host command spine are still aligned with wrappers and runtime guidance:

```bash
go test ./cmd -run 'TestCodexHostBackedGuidesUseTypeScriptHostSpine|TestWrapperSourcesUseTypeScriptHostManifestSpine|TestClassicCommandParityMatrix' -count=1
go test ./cmd -run '^TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild$' -count=1
npm --prefix .aether/ts-host run typecheck
npm --prefix .aether/ts-host test
```

Release notes may claim that TypeScript conducts host-backed workflows only if
the tests above pass and the wording keeps Go as the state, verification,
finalizer, provider-adapter, provider-diagnostic, and ceremony authority. The TS
host coordinates waves; it does not select or spawn provider CLIs.

## Provider/Auth Evidence

Release smoke must preserve evidence for both auth preflight and launched-worker
diagnostics. The important distinction is that "preflight" means checking
whether a provider is installed and logged in before dispatch, while
"launched-worker diagnostics" means the text captured after a provider process
has already started and failed.

- no-credential fake provider CLIs report unavailable status without requiring
  real Claude, OpenCode, or Codex credentials;
- probe failures expose provider, category, and next action, not raw
  stdout/stderr;
- launched worker failures redact provider secrets in errors, RawOutput,
  worker-debug JSON excerpts, continue/seal reports, and TS host failed-worker
  summaries;
- temp-repo provider smoke uses isolated `HOME`, hub, repo, and `PATH`
  locations so no real user credentials are touched.

Evidence surfaces that should stay green:

- `cmd/blackbox_harness_test.go` exercises the compiled hidden Go adapter with a
  deterministic external provider and proves malformed results fail closed.
- `pkg/codex/platform_dispatch_test.go` and
  `cmd/dispatch_platform_helpers_test.go` cover categorized and redacted
  provider availability diagnostics, including hard-pin no-fallback behavior.
- `pkg/codex/platform_dispatch_test.go`,
  `pkg/codex/worker_test.go`,
  `.aether/ts-host/test/go-bridge.test.ts`, and
  `.aether/ts-host/test/worker-dispatch.test.ts` cover post-launch diagnostic
  redaction before errors, debug artifacts, RawOutput, and TS summaries expose
  provider output.
- `.aether/ts-host/test/worker-dispatch.test.ts` also proves production host
  sources do not import the legacy TypeScript provider launcher.

`.aether/ts-host/test/platform-dispatcher.test.ts` is legacy regression evidence
only. It cannot qualify production providers or justify release claims.

## Permission Profile Evidence

Every production worker request must carry the canonical permission profile from
its Go-authored manifest. Go rejects missing, stale, or broadened profiles before
provider launch.

Release evidence must include:

```bash
go test ./pkg/codex -run 'TestPermission|TestCodexReadOnly|TestClaudeWorkspace|TestShippedOpenCode' -count=1
go test ./cmd -run 'TestInternalWorkerAdapterRequiresAndValidatesPermissionProfile|TestBuildAndPlanningManifestsCarryCanonicalPermissionProfiles' -count=1
npm --prefix .aether/ts-host test
```

The supported release contract is `repository_read_only` for Scout and Includer,
and `workspace_write` for current write-capable castes. `scoped_write` and
`test_write` must fail closed until path-level enforcement exists. Probe,
review-ledger, survey, and documentation scopes are behavioral restrictions
inside a workspace boundary and must not be advertised as stronger isolation.

## Durable Build Run Evidence

Every supported build worker must carry the exact Go-authored execution binding.
Release evidence must prove stale results fail closed, parallel process registry
writes do not lose PIDs, wrapper retries reuse terminal results, and native workers
journal terminal evidence before wave completion:

```bash
go test ./pkg/codex -run 'TestWorkspaceFingerprint|TestExecutionBinding|TestProcessTracker' -count=1
go test ./pkg/codex -run 'TestProcessTrackerConcurrentUpsertsPreserveEveryProvider' -race -count=1
go test ./cmd -run 'TestPlanOnlyManifestCarriesJournalBoundExecutionIdentity|TestInternalBuildAdapter|TestTerminalWorkerJournal|TestNativeBuildDispatch|TestCLIInterruptedBuild' -count=1
npm --prefix .aether/ts-host test
```

Do not describe this as universal provider stream reattachment. Wrapper-hosted
builds can stage and replay a terminal completion after host loss. Native direct
builds preserve worker evidence and exact cancellation but still force-retry if
the parent dies before aggregate lifecycle commit.

## Seal-Time Blockers

Block seal if any of these are true:

- any required smoke command fails;
- provider/auth no-credential smoke lacks clear fake-provider evidence;
- post-launch worker diagnostic redaction evidence is missing or failing;
- a production TS host source imports `platform-dispatcher.ts` or launches a
  provider CLI directly;
- an explicit `AETHER_WORKER_PLATFORM` pin falls back to another provider;
- `control-ts` can return success or mutate the live `.aether/data` store;
- a worker manifest/request omits its permission profile, broadens it, or launches
  after the selected host fails permission attestation;
- a build worker request/result is missing its execution binding, carries a stale
  run/attempt/manifest/workspace/owner value, or is accepted after exact-run
  cancellation fails;
- Claude uses `bypassPermissions`, OpenCode bypasses `aether-worker-router`, or
  Codex grants an additional writable `CODEX_HOME`;
- release documentation claims `govulncheck` passed without actual tool output;
- release documentation claims supply-chain hardening from SHA-pinned Actions
  while workflows still use version tags.

For beginners: blockers are the things that stop the release now. Residual risks
are known imperfections that can ship only because they are written down and
accepted.

## Residual Risks

- GitHub Actions in release workflows are tag-pinned rather than SHA-pinned.
  This is acceptable for the current milestone but should be revisited before a
  stricter supply-chain hardening release.
- `govulncheck` was not available during the Gatekeeper review, so Go CVE status
  is not claimed by this handoff.
- Live third-party provider credentials are intentionally excluded from normal
  CI. Manual live-provider validation remains optional and should not replace
  deterministic fake-provider coverage. When live validation is in scope, use
  `.aether/docs/manual-provider-smoke-checklist.md`.
- Network access is still provider-managed. The current permission contract does
  not claim a cross-provider network sandbox.
- Narrow in-workspace scopes such as test-only and review-ledger-only are not yet
  host-enforced profiles.

## Publish Handoff

Actual public release is not implied by seal readiness.

For dummies: `aether publish --channel stable --binary-dest "$HOME/.local/bin"`
updates the local machine so other repos can test the current checkout. A public
release is the extra GitHub/npm step that makes the same version installable by
everyone else.

Before publishing publicly, maintainers still need:

- an intentionally curated working tree with no unrelated edits or generated
  verification artifacts;
- release metadata agreement between the pushed tag, `.aether/version.json`, and
  `npm/package.json`;
- tag readiness, including the final annotated release target and the decision
  to run GitHub release publishing rather than only a dry run;
- GitHub release asset verification before npm `latest` moves.

## Seal Guidance

Seal reviewers should treat missing smoke evidence for provider diagnostic
redaction as a release blocker. Tag pinning and absent `govulncheck` are
documented residual risks unless the release scope is expanded to solve them.
