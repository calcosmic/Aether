# update -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/update_cmd.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --channel | string | no | "" | Runtime channel to update from: stable or dev (default: infer) |
| --download-binary | bool | no | false | Also download a binary from GitHub Releases |
| --binary-version | string | no | "" | Binary version to download (default: installed version) |
| --dry-run | bool | no | false | Show what would be updated without making changes |
| --force | bool | no | false | Overwrite modified companion files and remove stale ones |

### Arguments
None

### Environment
None

## Outputs

### Stdout
Text output with sync report: files copied, unchanged, skipped, and any stale publish warnings.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| .aether/ | update | Companion files synced from hub |
| .claude/commands/ant/ | update | Claude Code command wrappers synced |
| .opencode/commands/ant/ | update | OpenCode command wrappers synced |
| .claude/agents/ant/ | update | Claude Code agent definitions synced |
| .opencode/agents/ | update | OpenCode agent definitions synced |
| {binary path}/aether | create | When --download-binary is used |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Hub not populated, stale publish detected, or sync failure |

## State Mutations

### Colony State Transitions
None -- does not modify colony state.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| .aether/ (companion dirs) | update | Companion files refreshed from hub |
| .claude/commands/ant/ | update | Claude wrappers refreshed |
| .opencode/commands/ant/ | update | OpenCode wrappers refreshed |

## Preconditions

- Hub must have published content (~/.aether/system/ or ~/.aether-dev/system/)
- Must run from a target repo (not the Aether source repo itself)
- Local colony data (COLONY_STATE.json, pheromones, etc.) is never overwritten
