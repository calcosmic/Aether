# `/ant:bump-version` Slash Command — Design Spec

**Date:** 2026-04-11
**Status:** Approved

---

## Problem

Version strings live in 7+ locations across the repo. Updating them manually causes drift (package.json says one thing, CLAUDE.md says another, hub says a third). Old git tags confuse `resolveVersion()`. The user is non-technical and should be able to release with one command.

## Solution

A Claude Code slash command (`/ant:bump-version`) that bumps the version everywhere and triggers the existing GoReleaser pipeline automatically.

## Command

```
/ant:bump-version <semver>
```

Example: `/ant:bump-version 1.1.0`

## Files Updated

| # | File | What Changes |
|---|------|-------------|
| 1 | `package.json` | `"version"` field |
| 2 | `.aether/version.json` | `"version"` field |
| 3 | `~/.aether/version.json` (hub) | `"version"` field |
| 4 | `CLAUDE.md` | Header line, table row, footer line (3 replacements) |
| 5 | `README.md` | Roadmap section header, colony badge (2 replacements) |
| 6 | `docs/phase3-section-roadmap.md` | Roadmap section header |

## Execution Steps

1. **Validate**
   - Check argument is valid semver (X.Y.Z)
   - Read current version from `package.json`
   - Verify new version is greater than current (prevent downgrades)

2. **Update files**
   - Update all 6 file locations listed above
   - For markdown files, use targeted replacements on known patterns:
     - CLAUDE.md header: `> **Current Version:** v{old}`
     - CLAUDE.md table: `| Version | v{old} |`
     - CLAUDE.md footer: `*Updated for Aether v{old}`
     - README.md roadmap: `### v{old} -- Released (Current)`
     - README.md badge: `badge/colony-v{old}-gold`
     - roadmap doc: `### v{old} -- Released (Current)`

3. **Rebuild binary**
   - Run `make build` (reads package.json, injects version via ldflags)

4. **Update hub**
   - Run `aether install` (propagates to `~/.aether/version.json`)

5. **Commit and push**
   - `git add` the changed files
   - `git commit -m "bump version to v{semver}"`
   - `git push origin main`

6. **Tag and push**
   - `git tag v{semver}`
   - `git push origin v{semver}`

7. **GoReleaser auto-triggers**
   - The existing `.github/workflows/release.yml` picks up the tag
   - Builds binaries for darwin/linux/windows (amd64 + arm64)
   - Creates GitHub release with archives and checksums

8. **Summary output**
   - Plain English: "Done. v1.1.0 pushed. GoReleaser is building binaries. Release at https://github.com/calcosmic/Aether/releases/tag/v1.1.0"

## Error Handling

- **Invalid semver:** Print error, stop immediately
- **Version not greater:** Print "current is X, you asked for Y which is not newer", stop
- **File not found:** Print which file is missing, stop before any writes
- **`make build` fails:** Print the error, stop before committing
- **`git push` fails:** Print the error, tell user to push manually
- **Tag already exists:** Print "tag v{semver} already exists", stop

## What It Does NOT Do

- No CHANGELOG entry (separate concern)
- No npm publish (not part of current workflow)
- No Homebrew tap update (not yet configured)
- No GitHub release creation manually (GoReleaser handles this)

## Implementation

- New file: `.claude/commands/ant/bump-version.md` (slash command definition)
- No Go code changes needed — it's a Claude command that uses existing tools (Edit, Bash)
- No new dependencies

## Testing

- Manual test: run `/ant:bump-version 1.0.1` and verify all 6 files update
- Verify `aether version` returns the new version after rebuild
- Verify `~/.aether/version.json` matches
- Verify tag was created and pushed
