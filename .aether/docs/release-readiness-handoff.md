# Release Readiness Handoff

## Purpose

This handoff captures the final local smoke and residual-risk expectations before
sealing the Provider Auth Clarity and Release Hardening colony.

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
goreleaser build --snapshot --clean
```

After the snapshot build, run the produced binary's `version` command. CI and
the release workflow use:

```bash
./dist/Aether_linux_amd64_v1/aether version
```

For a local non-Linux run, use the matching produced binary path under `dist/`.

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

- `.aether/ts-host/test/platform-dispatcher.test.ts` covers deterministic
  no-credential fake providers for Claude, OpenCode, and Codex.
- `pkg/codex/platform_dispatch_test.go` and
  `cmd/dispatch_platform_helpers_test.go` cover categorized and redacted
  provider availability diagnostics.
- `pkg/codex/platform_dispatch_test.go`,
  `pkg/codex/worker_test.go`,
  `.aether/ts-host/test/go-bridge.test.ts`, and
  `.aether/ts-host/test/worker-dispatch.test.ts` cover post-launch diagnostic
  redaction before errors, debug artifacts, RawOutput, and TS summaries expose
  provider output.

## Seal-Time Blockers

Block seal if any of these are true:

- any required smoke command fails;
- provider/auth no-credential smoke lacks clear fake-provider evidence;
- post-launch worker diagnostic redaction evidence is missing or failing;
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
  deterministic fake-provider coverage.

## Publish Handoff

Actual publish is not implied by seal readiness. Before publishing, maintainers
still need:

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
