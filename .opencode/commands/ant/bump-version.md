<!-- Generated from .aether/commands/bump-version.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-bump-version
description: "🚀 Bump version across runtime files, docs, hub publish, commit, push, and tag for release"
---

Use the Go `aether` CLI as the version-file source of truth, then preserve the release workflow.

## Usage

If `$ARGUMENTS` is empty, show:

```text
Usage: /ant-bump-version <semver>
Example: /ant-bump-version 1.1.0
```

Stop here.

## Runtime Bump

Run:

```bash
AETHER_OUTPUT_MODE=visual aether bump-version $ARGUMENTS
```

If the runtime rejects the version or reports no change, stop and report that output directly. Do not hand-edit `.aether/version.json` or `npm/package.json` before the runtime succeeds.

## Release Follow-Up

After the runtime succeeds:

1. Read the old and new versions from the runtime output or from the git diff.
2. Update matching human-facing version references where they exist:
   - `CLAUDE.md`
   - `AGENTS.md`
   - `README.md`
   - `docs/phase3-section-roadmap.md`
3. Run `go test ./...`.
4. Run `aether publish --channel stable --binary-dest "$HOME/.local/bin"`.
5. Verify consistency:

```bash
cat .aether/version.json
grep '"version"' npm/package.json
aether version
aether version --check
```

6. Commit the release files:

```bash
git add .aether/version.json npm/package.json CLAUDE.md AGENTS.md README.md docs/phase3-section-roadmap.md
git commit -m "bump version to v<new_version>"
```

7. Push and tag:

```bash
git push origin main
git tag v<new_version>
git push origin v<new_version>
```

## Failure Rules

- If tests or publish fail, do not commit or tag. Report the failing command and leave the working tree for repair.
- If push fails after commit, report the exact recovery command: `git push origin main && git push origin v<new_version>`.
- If the tag already exists, do not overwrite it unless the user explicitly asks.
- Never edit `.aether/data`, session files, or pheromone files from this command.
