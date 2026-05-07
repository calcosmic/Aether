# publish -- Lifecycle Contract

**Last verified:** 2026-05-07
**Source files:** cmd/publish_cmd.go

## Inputs

### Flags
| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| --package-dir | string | no | "" | Source directory (default: current directory) |
| --home-dir | string | no | "" | User home directory (default: $HOME) |
| --channel | string | no | "" | Runtime channel: stable or dev (default: infer from binary/env) |
| --binary-dest | string | no | "" | Destination directory for the built binary |
| --skip-build-binary | bool | no | false | Skip go build and use existing binary |

### Arguments
None

### Environment
None

## Outputs

### Stdout
Text output via `fmt.Fprint`/`fmt.Fprintf`. Publish progress and results.

### Files Created/Modified
| File | Operation | When |
|------|-----------|------|
| ~/.aether/system/ (or ~/.aether-dev/system/) | create/update | Companion files synced to hub |
| ~/.aether/system/agents/ | create/update | Agent definitions published |
| ~/.aether/system/commands/ | create/update | Command definitions published |
| ~/.aether/system/skills/ | create/update | Skill files published |
| ~/.aether/system/templates/ | create/update | Template files published |
| ~/.aether/system/docs/ | create/update | Documentation files published |
| ~/.aether/system/workers.md | create/update | Worker definitions published |
| {binary-dest}/aether | create | Binary built and placed |

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Not in Aether repo, build failure, version mismatch, or sync failure |

## State Mutations

### Colony State Transitions
None -- does not modify colony state.

### Data Artifacts Modified
| Artifact | Write Type | Content Changed |
|----------|------------|-----------------|
| ~/.aether/system/ | create/update | All companion files synced from source |
| ~/.aether/publish-manifest.json | create/update | Publish manifest tracking |

## Preconditions

- Must run from the Aether source repo (or specify --package-dir)
- Git working tree should be clean for reliable version detection
- Go toolchain available for binary build (unless --skip-build-binary)
