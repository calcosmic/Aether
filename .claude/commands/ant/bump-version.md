---
name: ant:bump-version
description: "🚀🐜 Bump version across all files, rebuild, push, and tag for release"
---

You are the **Queen Ant Colony**. Bump the Aether version everywhere and trigger a release.

## Instructions

Parse `$ARGUMENTS` to extract the target version string (e.g. "1.1.0").
If no argument provided, display:
```
Usage: /ant:bump-version <semver>
Example: /ant:bump-version 1.1.0
```
Stop here.

### Step 1: Validate

1. Check the argument matches semver pattern `^[0-9]+\.[0-9]+\.[0-9]+$`. If not, display "Invalid version format. Use X.Y.Z (e.g. 1.1.0)" and stop.
2. Read the current version from `package.json` (the `"version"` field).
3. If the target version equals the current version, display "Already at v{version}. Nothing to do." and stop.
4. Compare major, then minor, then patch numerically. If the target version is not greater than the current version, display "Current version is v{old_version}. v{new_version} is not newer. Nothing to do." and stop.

Store `old_version` and `new_version` for use in replacements.

### Step 2: Verify Files Exist

Check that all target files exist before making any changes. If any are missing, report which one and stop:

```bash
for f in package.json .aether/version.json README.md CLAUDE.md docs/phase3-section-roadmap.md; do
  test -f "$f" || echo "MISSING: $f"
done
```

If any file is missing, display "Cannot proceed: {file} not found." and stop.

### Step 3: Update package.json

Use the Edit tool to replace `"version": "{old_version}"` with `"version": "{new_version}"` in `package.json`.

### Step 4: Update .aether/version.json

Use the Edit tool to replace `"version":"{old_version}"` with `"version":"{new_version}"` in `.aether/version.json`.

### Step 5: Update CLAUDE.md (3 replacements)

Use the Edit tool on `CLAUDE.md` for these three replacements:

1. Header: `> **Current Version:** v{old_version}` → `> **Current Version:** v{new_version}`
2. Table: `| Version | v{old_version} |` → `| Version | v{new_version} |`
3. Footer: `*Updated for Aether v{old_version}` → `*Updated for Aether v{new_version} — {current date in YYYY-MM-DD}`

### Step 6: Update README.md (2 replacements)

Use the Edit tool on `README.md` for these two replacements:

1. Roadmap header: `### 🎉 v{old_version} -- Released (Current)` → `### 🎉 v{new_version} -- Released (Current)`
2. Colony badge: `badge/colony-v{old_version}-gold` → `badge/colony-v{new_version}-gold`

### Step 7: Update docs/phase3-section-roadmap.md

Use the Edit tool to replace `### v{old_version} -- Released (Current)` with `### v{new_version} -- Released (Current)` in `docs/phase3-section-roadmap.md`.

### Step 8: Rebuild Binary

Run using the Bash tool:
```bash
make build
```

If it fails, display the error and stop before committing.

### Step 9: Update Hub

Run using the Bash tool:
```bash
aether install
```

### Step 10: Verify Consistency

Run a quick check that all files agree on the new version:
```bash
echo "package.json:" && grep '"version"' package.json
echo ".aether/version.json:" && cat .aether/version.json
echo "hub:" && cat ~/.aether/version.json
echo "binary:" && ./aether version
```

If any file still shows the old version, report the mismatch and stop.

### Step 11: Commit and Push

```bash
git add package.json .aether/version.json CLAUDE.md README.md docs/phase3-section-roadmap.md
git commit -m "bump version to v{new_version}"
git push origin main
```

If push fails, display the error and tell the user to push manually.

### Step 12: Tag and Push Tag

```bash
git tag v{new_version}
git push origin v{new_version}
```

If the tag already exists, display "Tag v{new_version} already exists." and stop before pushing.

### Step 13: Summary

Display in plain English:

```
Done! Version bumped to v{new_version}.

What happened:
- Updated 6 files (package.json, version.json, CLAUDE.md, README.md, roadmap, hub)
- Rebuilt the aether binary
- Committed and pushed to main
- Tagged v{new_version} and pushed

What happens next:
GoReleaser is now building binaries for Mac, Windows, and Linux.
The release will appear in a few minutes at:
https://github.com/calcosmic/Aether/releases/tag/v{new_version}
```

<failure_modes>
### Build Failure
If `make build` fails:
- Do NOT commit or tag
- Display the build error
- Revert all file changes (restore old_version)
- Tell user to fix the build issue and retry

### Push Failure
If `git push` fails:
- Changes are committed locally but not pushed
- Display the error
- Tell user: "Changes are committed locally. Fix the push issue and run: git push origin main && git push origin v{new_version}"

### Tag Already Exists
If `git tag v{new_version}` fails because the tag exists:
- All file changes are committed and pushed (good)
- Display: "Tag v{new_version} already exists locally or remotely. Delete it first if you want to re-release."
</failure_modes>

<success_criteria>
Command is complete when:
- All 6 files show the new version
- Go binary reports the new version
- Hub version.json shows the new version
- Commit is pushed to main
- Tag is pushed to origin
- User sees the summary with release link
</success_criteria>
