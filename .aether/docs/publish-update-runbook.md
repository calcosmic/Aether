# Publishing Updates

This runbook is the authoritative workflow for publishing Aether changes and verifying that downstream repos can actually receive them.

## Rule of Thumb

- `aether install --package-dir "$PWD"` publishes companion files from an Aether source checkout into the shared hub on this machine and rebuilds the shared local `aether` binary.
- `aether update` in another repo only pulls companion files from that hub. It does not publish source-checkout changes by itself.
- `aether update --force` should be the default downstream refresh when you need stale Aether-managed files removed.
- `aether update --download-binary` downloads a published release binary. Use it when you need the released runtime, not an unreleased local source change.

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

1. Publish the Go GitHub release first.
2. Verify the release assets exist for every supported platform.
3. Update `npm/package.json`:
   - bump `version` for the npm package itself
   - set `aetherVersion` to the exact Go release version the wrapper should install
4. Run npm package verification locally.
5. Publish the npm package.

Recommended verification:

```bash
npm --prefix npm test
cd npm && npm pack --dry-run
node bin/aether.js --bootstrap-version
node bin/aether.js version
```

Why the order is strict:
- The npm package is only a bootstrap wrapper.
- It downloads a specific published Go release from GitHub Releases.
- If npm is published before the matching Go release exists, the wrapper will point users at a missing or stale runtime.

User-facing rule:
- `npx --yes aether-colony@latest` should always install the same published runtime that the current npm wrapper declares in `npm/package.json`.

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

In a downstream repo, a healthy refresh should no longer show `0/0` for Claude/OpenCode commands.

## Medic Check

Run `aether medic --deep` when you want runtime validation of:
- repo wrapper parity
- hub publish completeness
- ceremony integrity

Medic should flag incomplete hub publishes before you trust downstream `aether update` output.
