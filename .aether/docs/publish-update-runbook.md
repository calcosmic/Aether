# Publishing Updates

This runbook is the authoritative workflow for publishing Aether changes and verifying that downstream repos can actually receive them.

## Rule of Thumb

- `aether install --package-dir "$PWD"` publishes companion files from an Aether source checkout into the shared hub on this machine and rebuilds the shared local `aether` binary.
- `aether update` in another repo only pulls companion files from that hub. It does not publish source-checkout changes by itself.
- `aether update --force` should be the default downstream refresh when you need stale Aether-managed files removed.
- `aether update --download-binary` downloads a published release binary. Use it when you need the released runtime, not an unreleased local source change.
- `.aether/version.json` is the source-checkout release version file. `npm/package.json` must use the exact same version.

## Channel Policy

- Stable/public runtime: `aether` + `~/.aether/`
- Dev/maintainer runtime: `aether-dev` + `~/.aether-dev/`
- npm bootstrap publishes only the stable/public runtime
- Dev installs intentionally skip global Claude/OpenCode/Codex home sync by default so source-development does not overwrite the public command surface on the same machine

## Publish Command

`aether publish` is the primary recommended command for publishing Aether from source. It builds the binary, syncs companion files to the hub, and verifies that binary and hub versions agree atomically.

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

Behavior:
- Builds the binary (unless `--skip-build-binary`)
- Validates channel isolation (rejects cross-channel publish, e.g. dev binary targeting stable hub)
- Syncs companion files to the hub
- Verifies binary and hub versions agree after sync
- Prints a warning if hub version changed
- Prints an advisory note if stable and dev binaries co-locate in the same directory

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
- `aether publish` builds the binary, refreshes `~/.aether/system/` from the current checkout, and verifies version agreement.
- `update --force` refreshes tracked companion files from the hub and removes stale managed files.

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

Companion file completeness checks verify expected counts:
- 50 Claude commands
- 50 OpenCode commands
- 25 OpenCode agents
- 25 Codex agents
- 29 Codex skills

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
go run ./cmd/aether install --package-dir "$PWD" --binary-dest "$HOME/.local/bin"
aether version
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

## Bootstrap Workflow When `install` Itself Changed

If the change you made affects `aether install`, the currently installed binary may still be running the old publish logic. Bootstrap the new installer directly from source once:

```bash
go run ./cmd/aether install --package-dir "$PWD" --binary-dest "$HOME/.local/bin"
```

Why this is different:
- `go run` executes the new install code from the source checkout immediately.
- `--binary-dest "$HOME/.local/bin"` rebuilds the shared `aether` binary to a stable path on `PATH` instead of a temporary Go build location.

After that bootstrap run, downstream repos should use:

```bash
aether update --force
```

## Failure Signatures

These outputs mean the hub publish is incomplete and downstream repos cannot recover on their own:

- `Commands (claude) — 0 copied, 0 unchanged`
- `Commands (opencode) — 0 copied, 0 unchanged`
- `Agents (opencode)` count is below 25

Root cause:
- The target repo is updating from `~/.aether/system/`.
- If wrapper commands or OpenCode agents were never published into that hub layout, `aether update` has nothing to copy.

Fix:

```bash
# In the Aether repo
aether install --package-dir "$PWD"

# In the target repo
aether update --force
```

If the publish bug is inside `install` itself, use the bootstrap workflow above first.

## Verification Checklist

On the publishing machine, verify the hub contains the expected surfaces:

```bash
find ~/.aether/system/commands/claude -maxdepth 1 -type f | wc -l
find ~/.aether/system/commands/opencode -maxdepth 1 -type f | wc -l
find ~/.aether/system/agents -maxdepth 1 -type f | wc -l
find ~/.aether/system/codex -maxdepth 1 -type f | wc -l
find ~/.aether/system/skills-codex -name SKILL.md | wc -l
```

Expected counts:
- Claude commands: `50`
- OpenCode commands: `50`
- OpenCode agents: `25`
- Codex agents: `25`
- Codex skills: `29`

Release metadata should also agree:
- `.aether/version.json` version equals `npm/package.json` version
- `aether version` equals the intended release version after rebuilding from source
- `aether version --check` returns exit 0 (binary and hub versions agree)
- `npm view aether-colony dist-tags --json` reports `latest` at the same stable release version

In a downstream repo, a healthy refresh should no longer show `0/0` for Claude/OpenCode commands.

## Medic Check

Run `aether medic --deep` when you want runtime validation of:
- repo wrapper parity
- hub publish completeness
- ceremony integrity
- **release integrity** (binary vs hub version agreement, stale publish detection — via `scanIntegrity()`)

Medic should flag incomplete hub publishes before you trust downstream `aether update` output.
Medic should also treat a missing GitHub release after a pushed tag as a release-integrity failure and recommend `gh workflow run Release -f tag=vX.Y.Z` before falling back to local GoReleaser or manual npm publish.

For a focused release-pipeline validation without the broader medic health checks, use `aether integrity` directly (see [Integrity Check](#integrity-check) above).
