# Publishing Updates

This runbook is the authoritative workflow for publishing Aether changes and verifying that downstream repos can actually receive them.

## Rule of Thumb

- `aether install --package-dir "$PWD"` publishes companion files from an Aether source checkout into the shared hub on this machine and rebuilds the shared local `aether` binary.
- `aether update` in another repo only pulls companion files from that hub. It does not publish source-checkout changes by itself.
- `aether update --force` should be the default downstream refresh when you need stale Aether-managed files removed.
- `aether update --download-binary` downloads a published release binary. Use it when you need the released runtime, not an unreleased local source change.
- `.aether/version.json` is the source-checkout release version file. `npm/package.json` must use the exact same version.

## Standard Local Source Workflow

Use this when you changed files in the Aether repo and want other repos on the same machine to pick them up.

```bash
# In the Aether repo
aether install --package-dir "$PWD"

# In each target repo
aether update --force
```

Why this works:
- `install` refreshes `~/.aether/system/` from the current checkout.
- In a source checkout, `install` also rebuilds the shared local `aether` binary unless `--skip-build-binary` is used.
- `update --force` refreshes tracked companion files from the hub and removes stale managed files.

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
5. Create and push the Git tag `vX.Y.Z`.
6. Let the release workflow publish the Go release first, then the npm bootstrap if `NPM_TOKEN` is configured.

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

User-facing rule:
- `npx --yes aether-colony@latest` should always install the same published runtime as the current `latest` GitHub release.
- The `latest` npm dist-tag should point at the same version as the current stable Aether release, even though historical npm versions like `5.x` still exist in the registry history.

Manual fallback if npm is not automated in CI yet:

```bash
cd npm
npm publish --access public
```

Then verify:

```bash
npm view aether-colony dist-tags --json
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
- `npm view aether-colony dist-tags --json` reports `latest` at the same stable release version

In a downstream repo, a healthy refresh should no longer show `0/0` for Claude/OpenCode commands.

## Medic Check

Run `aether medic --deep` when you want runtime validation of:
- repo wrapper parity
- hub publish completeness
- ceremony integrity

Medic should flag incomplete hub publishes before you trust downstream `aether update` output.
