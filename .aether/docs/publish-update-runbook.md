# Publishing Updates

This runbook is the authoritative workflow for publishing Aether changes and verifying that downstream repos can actually receive them.

## Preflight: Clean Tree Before Publish

`aether publish` builds the binary and hub from the **working tree, with no
clean guard** — uncommitted edits ship silently. Before every publish:

```bash
git status --porcelain   # must be empty
```

If it is not empty, commit or stash first. Publishing from a dirty tree means
the hub and binary contain changes no commit records, which makes "what is
actually deployed" unanswerable.

Also note: when publishing via `go run ./cmd/aether publish`, verify the binary
actually updated afterwards (`aether version --check` matches AND the binary
mtime moved). A past bug sent go-run builds to a temp directory while reporting
success.

## Rule of Thumb

- `aether publish` is the source-checkout publish command. It builds the local channel binary, refreshes the shared hub, and verifies binary/hub version agreement.
- `aether install --package-dir "$PWD"` still publishes companion files and rebuilds the local binary for backward compatibility, but maintainers should prefer `aether publish`.
- `aether update` in another repo refreshes repo-local scaffolding, syncs managed guidance/settings/rules and the supported Codex skill payload, and prunes stale generated asset copies. It does not publish source-checkout changes by itself.
- `aether update --force` should be the default downstream refresh when you need stale Aether-managed repo-local files removed.
- `aether update --download-binary` downloads a published release binary. Use it when you need the released runtime, not an unreleased local source change.
- `.aether/version.json` is the source-checkout release version file. `npm/package.json` must use the exact same version.

## Local Publish vs Public Release

For dummies: local publish is updating your own machine's Aether cupboard.
Public release is shipping a version other people can install from GitHub or
npm.

`aether publish --channel stable --binary-dest "$HOME/.local/bin"` refreshes
the local stable hub, platform home files, and local binary from the source
checkout. Other repos on the same machine can then consume those companion files
with `aether update --force`.

A public release additionally requires the version files to match, the release
commit to be pushed, a `vX.Y.Z` tag to drive the GitHub release workflow,
published release assets, and npm `latest` pointing at the same version.
Downstream `aether update --download-binary` can only fetch a runtime binary
after that public release exists.

## Channel Policy

- Stable/public runtime: `aether` + `~/.aether/`
- Dev/maintainer runtime: `aether-dev` + `~/.aether-dev/`
- npm bootstrap publishes only the stable/public runtime
- Dev installs intentionally skip global Claude/OpenCode/Codex home sync by default so source-development does not overwrite the public command surface on the same machine

## Publish Command

`aether publish` is the primary recommended command for publishing Aether from source. It builds the binary, syncs companion files to the hub, and verifies that binary and hub versions agree. The Codex payload transaction does
not make unrelated binary and companion-file work one atomic operation.

```bash
# In the Aether repo (stable channel, inferred from binary name)
aether publish

# Explicit channel selection
aether publish --channel stable
aether publish --channel dev

# Custom binary destination
aether publish --channel dev --binary-dest "$HOME/.local/bin"

# Skip binary rebuild (use existing binary)
aether publish --skip-build-binary
```

Flags:

| Flag | Description |
|------|-------------|
| `--package-dir` | Source directory (default: current directory) |
| `--home-dir` | User home directory (default: `$HOME`) |
| `--channel` | Runtime channel (`stable` or `dev`; default: infer from binary/env) |
| `--binary-dest` | Destination directory for the built binary |
| `--skip-build-binary` | Skip `go build`; use existing binary |
| `--sync-platform-homes` | Explicit dev install/publish opt-in to shared stable platform homes |

Behavior:
- Builds the binary (unless `--skip-build-binary`)
- Validates channel isolation (rejects cross-channel publish, e.g. dev binary targeting stable hub)
- Syncs companion files to the hub
- On the stable channel, refreshes user-level Claude/OpenCode/Codex assets from the same source checkout; OpenCode is written to the active `~/.opencode/command` and `~/.opencode/agent` paths plus the legacy `~/.config/opencode/...` paths
- On the dev channel, skips user-level platform asset sync by default; explicit `--sync-platform-homes` opts into shared stable homes and reports that scope
- Verifies binary and hub versions agree after sync
- Prints an actionable warning if the hub version changed, including the publish recovery command, `version --check`, `integrity`, and downstream `update --force` commands for the active channel
- Prints actionable TS host warnings if build or hub sync is skipped, including the npm rebuild command and the publish command to rerun
- Prints an advisory note if stable and dev binaries co-locate in the same directory, including channel-specific `version --check` commands

> **Backward compatibility:** `aether install --package-dir "$PWD"` still works but does not include automatic version agreement verification. `aether publish` is the recommended path.

## Standard Local Source Workflow

Use this when you changed files in the Aether repo and want other repos on the same machine to pick them up.

```bash
# In the Aether repo
aether publish

# In each target repo
aether update --force
```

> **Backward compatibility:** `aether install --package-dir "$PWD"` still works as an alternative.

Why this works:
- `aether publish` builds the binary, refreshes `~/.aether/system/` and stable user-level platform assets from the current checkout, and verifies version agreement.
- `update --force` refreshes local scaffolding from the hub, syncs managed guidance/settings/rules, and removes stale generated agents, commands, shipped skills, and old `.aether/` system copies that no longer belong in target repos.

Claude command layout:
- Aether source keeps generated Claude wrappers under `.claude/commands/ant/*.md` for parity with OpenCode generation.
- Stable install/publish writes Claude commands as flat `ant-*.md` files in the global Claude command directory.
- Generated legacy files under `.claude/commands/ant/*.md` are removed on force/update or stable publish so Claude Code exposes `/ant-build`, not an `ant:` namespace.
- Non-Aether custom files in `.claude/commands/ant/` are preserved.

## Isolated Dev Workflow

Use this when you are actively developing Aether itself and do not want unreleased runtime changes to overwrite the public/stable install on the same machine.

```bash
# In the Aether repo
aether publish --channel dev --binary-dest "$HOME/.local/bin"

# In each target repo you want to test against the dev channel
aether-dev update --force
```

> **Backward compatibility:** `go run ./cmd/aether install --channel dev --package-dir "$PWD" --binary-dest "$HOME/.local/bin"` still works.

Why this works:
- the dev channel uses `~/.aether-dev/system/` instead of `~/.aether/system/`
- the dev binary installs as `aether-dev`
- stable `aether` and npm installs remain untouched

## Codex Skill Upgrade and Recovery

The nine ant names and their helper instructions travel as one checked package.
Your own skills and custom project instructions stay intact. A changed skill menu
needs a new Codex chat; an unchanged update stays quiet.

This is the Phase 204.1 distribution contract, to be verified on the combined
candidate at merge. Plan 03 owns the install/publish/update controllers and custom
preservation result plumbing; Plan 05's documentation and notice tests alone do
not establish that wiring. Plan 06 supplies registered-update checks, and Plan 07
supplies the full fresh-client receipt. No release is published by these docs.

### Upgrade the Runtime Before Using the New Payload Route

If the installed executable predates this feature, first bootstrap from the
current source checkout (after the clean-tree preflight):

```bash
go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"
aether version --check
# Then, in a disposable consumer for validation:
aether update --dry-run
aether update
```

Alternatively install a published release that includes payload support. An
ordinary `aether update` refreshes companions; it cannot add this feature to an
old executable. `--download-binary` remains the published-release path, not a way
to fetch an unreleased source change. A runtime rejected by a payload's minimum
version needs upgrading first, even if it would otherwise download a binary later.

### Payload, Ownership and Channels

- The selected hub publishes `system/codex-skills/manifest.json`, nine public
  `ant-*/SKILL.md` entries and three private `support/*.md` bodies. Manifest schema
  `codex-skill-payload/v1` binds versions, inventory, paths, modes and SHA-256
  digests. Install, publish and normal update use the same validated bytes.
- The qualified home destination is `~/.codex/skills/aether/`. Plan 01 observed
  fresh ant-plan discovery on Codex CLI 0.154.0. Private creation, research and
  build-cycle support files resolve relative to the installed public skill.
- `.aether-owned.json` records exact content and modes, not just names. Only
  matching owned files or the frozen fourteen-file v1.0.79 legacy inventory may
  be adopted/retired. Edited or unknown legacy files are preserved and reported.
- A custom canonical-name collision or edited owned support file stops the skill
  transaction, even with `--force`. That flag does not widen custom-skill
  ownership. Custom `AGENTS.md` and `.codex/CODEX.md` keep their bytes and receive
  preservation results; documents retaining Aether's managed sentinels still
  follow the existing managed-document refresh policy.
- Missing, partial, corrupt, unknown-schema or incompatible payloads refuse skill
  synchronization. Compatible newer inventories are not pruned to an older
  executable's nine names; installed-version downgrades are refused.
- A copied or removed skill produces one Codex refresh notice, even for removal
  only. Unchanged updates do not. Start a fresh Codex session when advised.
- Dev install/publish leave stable homes untouched by default. Their explicit
  `--sync-platform-homes` opt-in writes the existing stable-home destinations;
  dev update refuses that flag and never implicitly syncs shared homes.

### Recover Without Overwriting Owner Changes

Keep the failed transaction's journal and receipt, then follow its returned
recovery instructions. Failed/interrupted **unverified** transactions can restore
previous bytes and modes. An intervening edit remains preserved and produces
recovery-required status. Do not delete custom files or ownership records to
bypass a refusal. Repair a missing/incompatible payload by installing or publishing
a matching supported version. `LifecycleTransaction.Rollback` refuses an already
verified successful receipt; it is not a post-success downgrade command.

See [Runtime Update Architecture](../../RUNTIME%20UPDATE%20ARCHITECTURE.md#codex-skill-distribution-contract)
for the complete file ownership and transaction boundary.

### Focused Proof and Merge Checks

The [Plan 01 receipt](../../.planning/phases/204.1-codex-ant-skill-surface/evidence/discovery-tracer.json)
proves its recorded client/root and ant-plan entry routing only. The following
named checks map the distribution claims to their owners:

| Claim | Named checks | Evidence owner |
|-------|--------------|----------------|
| Clean install, strict inventory and private support | `TestCodexAntSkillInstallTracer`, `TestCodexAntSkillInventory`, `TestCodexAntSkillCleanInstallCollision` | Plan 01 |
| Exact legacy retirement, edited-file preservation, mode/absence guards, recovery | `TestCodexAntSkillLegacyMigration`, `TestCodexAntSkillOwnership`, `TestCodexAntSkillPostPlanCreationRefused`, `TestCodexAntSkillPostPlanModeChangeRefused`, `TestCodexAntSkillRollback` | Plan 02 |
| Shared published payload, refusal, repeat, channels, newer versions | `TestCodexAntSkillPublishedPayload`, `TestCodexAntSkillPayloadValidation`, `TestCodexAntSkillPublishRepeat`, `TestCodexAntSkillPublishChannel`, `TestCodexAntSkillRegisteredUpdate`, `TestCodexAntSkillRegisteredUpdateRepeat`, `TestCodexAntSkillUpdatePreview`, `TestCodexAntSkillUpdateVersions`, `TestCodexAntSkillUpdateChannel` | Plan 03; check at merge |
| Managed docs, custom docs, copied/removal-only notices | `TestCodexAntSkillManagedGuidance`, `TestCodexAntSkillCustomGuidancePreserved`, `TestCodexAntSkillRestartNotice` | Plan 05 |
| Combined update, custom preservation receipts, interruption/conflict recovery | `TestCodexAntSkillUpdateMatrix`, `TestCodexAntSkillUpdatePostPlanCreationRefused`, `TestCodexAntSkillUpdatePostPlanModeChangeRefused`, `TestCodexAntSkillUpdateFailureRecovery`, `TestCodexAntSkillUpdatePreservationReceipt` | Plan 06; check at merge |
| All nine names in fresh clients after install/upgrade | `TestCodexAntSkillFreshHost`, `TestCodexAntSkillFreshHostUpgrade`, `TestCodexAntSkillReceiptValidation` | Plan 07; actual receipt required |

Run the focused generated-guidance check from source:

```bash
go test ./cmd -run '^TestCodexAntSkill(ManagedGuidance|CustomGuidancePreserved|RestartNotice)$' -count=1 -timeout 90m
```

After Plans 03 and 06 are merged, discover then execute their named checks. Zero
matched tests is not success; retain exact discovered/executed identities and raw
exit statuses. See [Plan 06's normal/race commands](../../.planning/phases/204.1-codex-ant-skill-surface/204.1-06-PLAN.md)
and [Plan 07's explicit live command](../../.planning/phases/204.1-codex-ant-skill-surface/204.1-07-PLAN.md).
A skipped or unavailable live client does not qualify the surface. These checks
prove installation and entry routing, not full lifecycle or native-worker parity.

## Published Release Workflow

Use this when you need the published runtime binary as well as refreshed companion files.

```bash
aether update --force --download-binary
```

That command syncs companion files first, then downloads the published binary.

## npm Bootstrap Release Workflow

Use this when you are publishing the public `npx` entrypoint for non-Go users.

Release order matters:

1. Set `.aether/version.json` to the release version.
2. Set `npm/package.json` `version` to the exact same release version.
3. Commit the version change.
4. Push the commit.
5. Create an annotated Git tag: `git tag -a vX.Y.Z -m "vX.Y.Z"`.
6. Push only that tag: `git push origin vX.Y.Z`.
7. Let the GitHub `Release` workflow publish the Go release first, then the npm bootstrap if `NPM_TOKEN` is configured.

Release metadata gates:
- The GitHub `Release` workflow checks that the pushed tag version, `.aether/version.json`, and `npm/package.json` all match before building or publishing release assets.
- The CI and release workflows run `.aether/ts-host` `ci`, `typecheck`, `test`, and `build`, including deterministic provider/auth tests that use fake CLIs rather than real credentials.
- The npm bootstrap job repeats that metadata check before publishing, so npm cannot intentionally move to a version that does not match the Go release tag.

Auth gates:
- The Go release publish step uses the workflow-provided `GITHUB_TOKEN` with `contents: write`; if release creation is blocked, verify the workflow run and release asset status before publishing npm.
- The npm bootstrap job runs only when `NPM_TOKEN` is configured and the GoReleaser job is publishing a real release. The workflow resolves token availability in the GoReleaser job output instead of referencing `secrets.NPM_TOKEN` directly in a job conditional. If `NPM_TOKEN` is missing, the Go release can still succeed, but npm `latest` will not move.
- Local GoReleaser fallback requires a usable GitHub token, for example `export GITHUB_TOKEN="$(gh auth token)"`, before `goreleaser release --clean`.

Recommended verification:

```bash
npm --prefix npm test
cd npm && npm pack --dry-run
node bin/aether.js --bootstrap-version
node bin/aether.js version
test "$(node -p "require('./npm/package.json').version")" = "$(node -p "require('./.aether/version.json').version")"
```

Why the order is strict:
- The npm package is only a bootstrap wrapper.
- It downloads the published Go release with the exact same version.
- If the npm package version and the Aether release version diverge, users will see version drift immediately.
- Push release tags one at a time. GitHub's workflow docs say push events are not created for tags when more than three tags are pushed at once, so do not use `git push --tags` for release publication.

User-facing rule:
- `npx --yes aether-colony@latest` should always install the same published runtime as the current `latest` GitHub release.
- The `latest` npm dist-tag should point at the same version as the current stable Aether release, even though historical npm versions like `5.x` still exist in the registry history.
- The npm package page README comes from `npm/README.md` in the published tarball, not from the root GitHub `README.md`.
- Updating the npm website README requires publishing a new npm package version; editing `npm/README.md` in git alone does not change the live npm page.

## Release Workflow Fallback

If you pushed a release tag and GitHub does not create a `Release` run or release assets, do not publish npm yet.

Failure signature:
- `git push origin vX.Y.Z` succeeds
- `gh run list --workflow Release` shows no run for the tag
- `gh release view vX.Y.Z` reports `release not found`

Preferred fallback:

```bash
gh workflow run Release -f tag=vX.Y.Z
```

Optional validation-only check:

```bash
gh workflow run Release -f tag=vX.Y.Z -f dry_run=true
```

Then verify:

```bash
gh run list --workflow Release --limit 5
```

If GitHub responds with `HTTP 422: Actions has been disabled for this user`, the workflow exists but this actor cannot dispatch it. In that case, use the local GoReleaser fallback below or have another maintainer trigger the workflow.

Second fallback, only if GitHub workflow dispatch is unavailable or broken:

```bash
export GITHUB_TOKEN="$(gh auth token)"
goreleaser release --clean
```

Then verify the release exists and has assets before publishing npm:

```bash
gh release view vX.Y.Z --json tagName,url,assets
```

Why the order still matters:
- the npm bootstrap downloads the published GitHub release assets directly
- if npm moves first, `npx --yes aether-colony@latest` can point users at a version whose release archives do not exist yet

Manual npm fallback, only if the release exists but npm automation is unavailable:

```bash
cd npm
npm publish --access public
```

Then verify:

```bash
npm view aether-colony dist-tags --json
```

## Integrity Check

`aether integrity` validates the full release pipeline chain. It auto-detects whether you are in the Aether source repo or a consumer repo and runs the appropriate checks.

```bash
# In the Aether source repo (5 checks)
aether integrity

# In a consumer repo (4 checks)
aether integrity

# Force source-repo context
aether integrity --source

# JSON output
aether integrity --json

# Check dev channel
aether integrity --channel dev
```

Flags:

| Flag | Description |
|------|-------------|
| `--json` | Output structured JSON instead of visual report |
| `--channel stable\|dev` | Override channel detection |
| `--source` | Force source-repo checks (5 checks instead of 4) |

Source repo checks: source version, binary version, hub version, hub companion files, downstream simulation.
Consumer repo checks: binary version, hub version, hub companion files, downstream simulation.

Exit codes: `0` = all checks pass, non-zero = failures found.

> **Note:** `aether medic --deep` includes integrity scanning automatically via `scanIntegrity()`. Use `aether integrity` directly when you want a focused release-pipeline validation.

## Stale Publish Detection

Every `aether update` automatically runs stale publish detection. This checks whether the hub publish is complete and fresh by comparing binary and hub versions and verifying companion-file completeness.

Classifications:

| Classification | Meaning | Behavior |
|---|---|---|
| `ok` | Binary and hub versions agree, companion files complete | Update proceeds normally |
| `info` | Companion files are incomplete (counts below expected) | Update proceeds, warning displayed |
| `warning` | Hub version is ahead of binary version | Update proceeds, warning displayed |
| `critical` | Hub version is behind binary version (stale publish) | **Update blocked**, non-zero exit code |

Recovery commands printed on failure:

```bash
# For stable channel
aether publish

# For dev channel
aether publish --channel dev
```

Current source inventories contain the following. Legacy count diagnostics are
a coarse lower bound, not proof that every current entry was published:
- 64 Claude commands
- 64 OpenCode commands
- 28 OpenCode agent assets (27 castes plus `aether-worker-router`)
- 27 Codex agents
- 86 hub shipped skills

The Codex payload has nine public skills and three private support bodies plus
its manifest. Validate its actual bytes and ownership with the focused checks
above; legacy companion counts alone are not a skill-surface receipt.

## Release Gate

The release gate has two layers.

GoReleaser `before.hooks` block the release if any hook fails:

- `go mod tidy` — ensures module metadata is normalized.
- `git diff --exit-code -- go.mod go.sum` — blocks release if `go mod tidy` changed module files.
- `go test ./cmd -run TestDocCLIAlignment -v` — Doc-CLI alignment smoke test. Host-critical flag mismatches between YAML documentation and the Go CLI will fail the release.

The GitHub `Release` workflow adds the release/auth gates around GoReleaser:
- Verifies the tag version, `.aether/version.json`, and `npm/package.json` match before publishing.
- Runs `goreleaser check`, `go build`, `go vet ./...`, `go test ./... -count=1`, `go test ./... -race -count=1`, narrator package verification, a full GoReleaser snapshot release (archives plus checksums), and an exact-version binary smoke test before release publication.
- Packs the npm bootstrap and installs the actual current-platform snapshot archive through it before the publishing step.
- Runs `.aether/ts-host` install, typecheck, tests, and build so provider/auth preflight behavior is exercised under deterministic no-credential conditions.
- Publishes Go release assets with the workflow `GITHUB_TOKEN`.
- Publishes npm only when the GoReleaser job reports `NPM_TOKEN` availability and the Go release is a real publish, not a dry run.

This ensures documentation, CLI flags, release metadata, and publish credentials cannot silently diverge on a shipped release.

## Go Binary Change Checklist

Use this checklist any time the change touches `cmd/`, `pkg/`, `.goreleaser.yml`, version resolution, install/update flows, binary download logic, or anything else that can affect the shipped Go runtime.

Required checks:

```bash
go test ./... -count=1
go test ./... -race -count=1
go build ./cmd/aether
aether version
```

If the change touches `aether install`, `aether update`, version resolution, or binary publishing/bootstrap logic, also run:

```bash
go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"
aether version
aether integrity --source --channel stable
```

Then verify at least one downstream repo:

```bash
cd /path/to/target-repo
aether update --force
```

If the public install path is affected, also verify:

```bash
cd /path/to/Aether/npm
npm --prefix . test
npm pack --dry-run
```

## Bootstrap Workflow When Publish/Install Changed

If the change you made affects `aether install`, `aether publish`, version resolution, or hub detection, the currently installed binary may still be running old logic. Bootstrap the new publisher directly from source once:

```bash
go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"
```

Why this is different:
- `go run` executes the new publish/install/update code from the source checkout immediately.
- `--binary-dest "$HOME/.local/bin"` rebuilds the shared `aether` binary to a stable path on `PATH` instead of a temporary Go build location.
- `publish` verifies binary and hub versions agree before returning success.

After that bootstrap run, downstream repos should use:

```bash
aether update --force
```

## Failure Signatures

These outputs mean the hub publish is incomplete and downstream repos cannot recover on their own:

- `Commands (claude) — 0 copied, 0 unchanged`
- `Commands (opencode) — 0 copied, 0 unchanged`
- `Agents (opencode)` count is below 27

Root cause:
- The target repo is updating from `~/.aether/system/`.
- If wrapper commands or OpenCode agents were never published into that hub layout, `aether update` has nothing to copy.

Fix:

```bash
# In the Aether repo
aether publish

# In the target repo
aether update --force
```

If the publish bug is inside `publish` itself, use the fallback install bootstrap once:

```bash
go run ./cmd/aether install --package-dir "$PWD" --binary-dest "$HOME/.local/bin"
```

## Verification Checklist

On the publishing machine, verify the hub contains the expected surfaces:

```bash
find "$HOME/.aether/system/commands"/claude -maxdepth 1 -type f | wc -l
find "$HOME/.aether/system/commands"/opencode -maxdepth 1 -type f | wc -l
find ~/.aether/system/agents -maxdepth 1 -type f | wc -l
find ~/.aether/system/codex -maxdepth 1 -type f | wc -l
find "$HOME/.aether/system/skills" -name SKILL.md | wc -l
find "$HOME/.aether/system/codex-skills" -name SKILL.md | wc -l
```

Expected counts:
- Claude commands: `64`
- OpenCode commands: `64`
- OpenCode agents: `28` (27 castes plus the restricted router)
- Codex agents: `27`
- Hub shipped skills: `86`
- Codex public skill payload: `9` public `SKILL.md` entries, plus `3` ordinary
  private support files and the versioned manifest (not counted as public skills)

Release metadata should also agree:
- `.aether/version.json` version equals `npm/package.json` version
- `aether version` equals the intended release version after rebuilding from source
- `aether version --check` returns exit 0 (binary and hub versions agree)
- `npm view aether-colony dist-tags --json` reports `latest` at the same stable release version

In a downstream repo, a healthy refresh should prune stale generated repo-local commands/agents instead of trying to copy them back into the repo.

## Medic Check

Run `aether medic --deep` when you want runtime validation of:
- repo wrapper parity
- hub publish completeness
- ceremony integrity
- **release integrity** (binary vs hub version agreement, stale publish detection — via `scanIntegrity()`)

Medic should flag incomplete hub publishes before you trust downstream `aether update` output.
Medic should also treat a missing GitHub release after a pushed tag as a release-integrity failure and recommend `gh workflow run Release -f tag=vX.Y.Z` before falling back to local GoReleaser or manual npm publish.

For a focused release-pipeline validation without the broader medic health checks, use `aether integrity` directly (see [Integrity Check](#integrity-check) above).
