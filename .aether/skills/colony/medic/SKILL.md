---
name: medic
description: Colony health diagnostics and repair — healthy state specifications for all colony data files, failure modes, and remedies
type: colony
domains: [health, diagnostics, repair, data-integrity, validation]
agent_roles: [builder, watcher, medic]
priority: normal
version: "1.0"
---

# Medic: Colony Data Health Reference

## Purpose

This skill defines what healthy colony data looks like. Workers should reference this when modifying colony state files or diagnosing colony issues. The Medic worker (`aether medic`) uses these rules to scan for problems and repair them.

## COLONY_STATE.json Healthy State

### Schema

| Field | Type | Required | Valid Values |
|-------|------|----------|-------------|
| version | string | yes | "3.0" |
| goal | *string | yes | non-empty string |
| state | State | yes | IDLE, READY, EXECUTING, BUILT, COMPLETED |
| scope | ColonyScope | yes | "project", "meta" |
| current_phase | int | yes | >= 0 |
| colony_version | int | no | starts at 1 |
| session_id | *string | no | format: `{unix}_{hex}` |
| parallel_mode | ParallelMode | no | "in-repo", "worktree" |
| paused | bool | no | only valid when state is READY |
| milestone | string | no | milestone name |
| worktrees | []WorktreeEntry | no | each has status field |
| signals | []Signal | **deprecated** | should be empty — migrated to pheromones.json |
| events | []string | no | pipe-delimited: `timestamp\|type\|source\|description` |
| plan | Plan | no | phases with PhaseStatus and tasks |
| memory | Memory | no | instincts, learnings, errors |

### State Machine

Legal transitions (from `pkg/colony/colony.go:490`):
- IDLE → READY
- READY → EXECUTING, COMPLETED
- EXECUTING → BUILT, COMPLETED
- BUILT → READY, COMPLETED
- COMPLETED → IDLE

### Consistency Rules

- If `state` is EXECUTING, `current_phase` must be > 0
- If `paused` is true, `state` should be READY
- If `parallel_mode` is "worktree", worktrees array should be maintained
- Events entries must be pipe-delimited with at least 2 segments

### Common Failures

| Failure | Severity | Detection |
|---------|----------|-----------|
| Missing or empty goal | critical | goal field nil or empty |
| Invalid state value | critical | not in IDLE/READY/EXECUTING/BUILT/COMPLETED |
| EXECUTING with phase 0 | warning | state mismatch |
| Paused but not READY | warning | flag/state conflict |
| Deprecated signals present | warning | signals array non-empty |
| Orphaned worktrees | warning | status="orphaned" entries |
| Invalid parallel_mode | warning | not "in-repo" or "worktree" |
| Malformed event entries | warning | missing pipe delimiters |
| Legacy state strings | warning | PAUSED/PLANNED/SEALED/ENTOMBED |

### Remedies

- **Legacy states**: Auto-normalized by `normalizeLegacyColonyState()` in `cmd/state_load.go`
- **Orphaned worktrees**: Remove entries with status="orphaned" (verify git worktree doesn't exist first)
- **Deprecated signals**: Migrate to pheromones.json, clear signals array
- **EXECUTING without phase**: Reset to READY
- **Invalid parallel_mode**: Report for manual correction

## pheromones.json Healthy State

### Schema

| Field | Type | Required | Valid Values |
|-------|------|----------|-------------|
| id | string | yes | unique identifier |
| type | PheromoneType | yes | FOCUS, REDIRECT, FEEDBACK |
| active | bool | yes | true/false |
| content | json.RawMessage | yes | valid JSON (supports nested objects) |
| created_at | string | yes | parseable timestamp |
| expires_at | *string | no | parseable timestamp |
| content_hash | *string | no | SHA-256 for dedup |
| strength | *float64 | no | 0.0-1.0 |
| priority | string | no | "normal", "high", "low" |

### Content Rules

- Content is `json.RawMessage`, NOT a plain string
- Supports nested objects like `{"text": "..."}`
- Max 500 characters (enforced by `SanitizeSignalContent()`)
- No XML structural tags, no prompt injection patterns, no shell injection

### Common Failures

| Failure | Severity | Detection |
|---------|----------|-----------|
| Missing signal ID | warning | id field empty |
| Invalid signal type | warning | not FOCUS/REDIRECT/FEEDBACK |
| Expired but still active | warning | expires_at past, active=true |
| Duplicate content hash | warning | same hash with different IDs |
| Invalid content JSON | critical | content not valid JSON |

### Remedies

- **Expired signals**: Set active=false, set archived_at timestamp
- **Missing IDs**: Generate `sig_{unix}_{4hex}`
- **Invalid types**: Default to FOCUS (safest)
- **Duplicate hashes**: Keep most recent, archive older

## session.json Healthy State

### Schema

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| session_id | string | yes | unique session identifier |
| started_at | string | yes | parseable timestamp |
| colony_goal | string | yes | should match COLONY_STATE goal |
| current_phase | int | yes | should match COLONY_STATE current_phase |
| current_milestone | string | no | milestone name |
| suggested_next | string | no | next recommended command |
| context_cleared | bool | no | whether context was cleared |
| baseline_commit | string | no | git commit hash |
| last_command | string | no | last command run |
| last_command_at | string | no | timestamp of last command |
| active_todos | []string | no | active todo items |
| summary | string | no | session summary |

### Cross-Reference Rules

- `current_phase` must match `COLONY_STATE.json` current_phase
- `colony_goal` must match `COLONY_STATE.json` goal

### Staleness Thresholds

- Last activity >7 days: warning
- Last activity >30 days: critical

### Common Failures

| Failure | Severity | Detection |
|---------|----------|-----------|
| Phase mismatch with COLONY_STATE | warning | values differ |
| Goal mismatch with COLONY_STATE | warning | strings differ |
| Missing session_id | warning | field empty |
| Invalid started_at | warning | not parseable |
| Stale session | warning/critical | >7d or >30d |

### Remedies

- **Phase/goal mismatch**: Sync from COLONY_STATE.json
- **Stale session**: User should run `/ant-resume` to refresh

## Data Files Healthy State

### midden/midden.json

- **Location**: `.aether/data/midden/midden.json` (subdirectory — different from other data files)
- **Structure**: `MiddenFile` with archived_signals, entries, spawn_metrics
- **Common failures**: corrupted JSON
- **Remedy**: JSON recovery with `--force`

### instincts.json

- **Structure**: `InstinctsFile` with entries (trigger, action, domain, trust_score, confidence, provenance)
- **Note**: Two instinct representations exist — simple `Instinct` in ColonyState.Memory and rich `InstinctEntry` here
- **Common failures**: corrupted JSON
- **Remedy**: JSON recovery with `--force`

### learning-observations.json

- **Structure**: `LearningFile` with observations (content_hash, content, wisdom_type, trust_score)
- **Common failures**: corrupted JSON
- **Remedy**: JSON recovery with `--force`

### assumptions.json

- **Structure**: `AssumptionsFile` with assumptions (id, phase, category, confidence, validated)
- **Valid confidence**: "confident", "likely", "unclear"
- **Common failures**: corrupted JSON, invalid confidence values
- **Remedy**: JSON recovery with `--force`

### pending-decisions.json

- **Structure**: `FlagsFile` with entries (id, type, description, resolved)
- **Valid types**: "blocker", "issue", "note"
- **Common failures**: corrupted JSON
- **Remedy**: JSON recovery with `--force`

### constraints.json

- **Ghost file**: Go struct is empty `{}`, content is ignored by all Go code
- **Expected state**: `{}` (empty object)
- **If content present**: warning — content is never read
- **Remedy**: Reset to `{}`

### Cache Files

- `.cache_COLONY_STATE.json` — expected, auto-rebuilt by runtime
- `.cache_instincts.json` — expected, auto-rebuilt by runtime
- **Not corruption** — these are legitimate cache files

## JSONL Files Healthy State

### trace.jsonl

- **Structure**: Each line is a `TraceEntry` (id, run_id, timestamp, level, topic, payload, source)
- **Valid levels**: state, phase, pheromone, error, recovery, intervention, token, artifact
- **Rotation**: 50MB default (`pkg/trace/rotate.go`)
- **Common failures**: malformed lines, approaching rotation limit (>45MB)
- **Remedy**: malformed lines auto-skipped by `ReadJSONL`; rotation is automatic

### event-bus.jsonl

- **Structure**: Each line is an `Event` (id, topic, payload, source, timestamp, ttl_days, expires_at)
- **TTL**: default 30 days, cleanup via bus
- **Common failures**: malformed lines, expired events
- **Remedy**: malformed lines auto-skipped; TTL cleanup runs automatically

### spawn-tree.txt

- **Structure**: TSV format with `SpawnEntry` rows
- **Common failures**: malformed format
- **Remedy**: regenerate on next build

## Wrapper Parity (Deep Scan)

### Expected Counts

| Surface | Count | Path |
|---------|-------|------|
| YAML commands | 50 | `.aether/commands/*.yaml` |
| Claude commands | 50 | `.claude/commands/ant/*.md` |
| OpenCode commands | 50 | `.opencode/commands/ant/*.md` |
| Claude agents | 25 | `.claude/agents/ant/*.md` |
| OpenCode agents | 25 | `.opencode/agents/*.md` |
| Codex agents | 25 | `.codex/agents/*.toml` |
| Claude mirror | 25 | `.aether/agents-claude/*.md` |
| Codex mirror | 25 | `.aether/agents-codex/*.toml` |
| Colony skills | 11 | `.aether/skills/colony/*/SKILL.md` |
| Domain skills | 18 | `.aether/skills/domain/*/SKILL.md` |

### Consistency Rules

- Command counts must match across YAML, Claude, and OpenCode
- Agent counts must match across Claude, OpenCode, Codex, and mirrors

### Common Failures

- Count mismatch after adding/removing commands
- Missing files after incomplete `aether update`

### Remedies

- Run `aether install` to regenerate wrappers from YAML sources
- Check `.aether/commands/*.yaml` as source of truth

## Hub Publish Integrity (Deep Scan)

### Expected Hub Counts

| Surface | Count | Path |
|---------|-------|------|
| Hub Claude commands | 50 | `~/.aether/system/commands/claude/*.md` |
| Hub OpenCode commands | 50 | `~/.aether/system/commands/opencode/*.md` |
| Hub OpenCode agents | 25 | `~/.aether/system/agents/*.md` |
| Hub Codex agents | 25 | `~/.aether/system/codex/*.toml` |
| Hub Codex skills | 29 | `~/.aether/system/skills-codex/*/*/SKILL.md` |

### Failure Signatures

- `aether update --force` reports `Commands (claude) — 0 copied, 0 unchanged`
- `aether update --force` reports `Commands (opencode) — 0 copied, 0 unchanged`
- `aether update --force` reports fewer than 25 OpenCode agents

### Remedies

- For unreleased local source changes on this machine: run `aether publish` from the Aether repo (builds binary, syncs hub, verifies version agreement), then `aether update --force` in target repos. `aether install --package-dir "$PWD"` also works as backward-compatible fallback
- For isolated source-development on the same machine: run `aether publish --channel dev --binary-dest "$HOME/.local/bin"` from the Aether repo, then `aether-dev update --force` in target repos. `go run ./cmd/aether install --channel dev --package-dir "$PWD" --binary-dest "$HOME/.local/bin"` also works as fallback
- If the change modified `aether install` itself: bootstrap with `go run ./cmd/aether install --package-dir "$PWD" --binary-dest "$HOME/.local/bin"`
- For published release runtime updates: run `aether update --force --download-binary`
- For first-time published installs on a machine with no hub yet: run `npx --yes aether-colony@latest`

## Release Version Integrity

- `aether medic --deep` includes release integrity scanning via scanIntegrity() which checks binary/hub version agreement and stale publish classification. Issues surface with severity and recovery commands inline.
- `aether integrity` is the dedicated command for full release pipeline validation. Auto-detects source vs consumer repo context. Source repos get 5 checks (source version, binary version, hub version, companion files, downstream simulation); consumer repos get 4 checks (binary version, hub version, companion files, downstream simulation). Flags: `--json` (machine-readable output), `--channel stable|dev` (override channel), `--source` (force source-repo checks). Cross-reference the operations guide for full workflow documentation.
- `.aether/version.json` is the source-checkout release version file
- `npm/package.json` version must equal `.aether/version.json`
- If these differ, report release version drift before trusting docs or publish instructions
- Release docs are part of the health check: `README.md`, `npm/README.md`, `AGENTS.md`, `CLAUDE.md`, `.codex/CODEX.md`, `.opencode/OPENCODE.md`, `RUNTIME UPDATE ARCHITECTURE.md`, `.aether/docs/publish-update-runbook.md`, `CHANGELOG.md`, and roadmap docs should agree on the install/update story
- For public installs, `npx --yes aether-colony@latest` should resolve to the same stable version as the current GitHub release
- The npm package page README comes from `npm/README.md` in the published package, not the root repo README
- Updating the npm website README requires a fresh npm publish; editing `npm/README.md` in git is not enough
- If install/update/version/binary-download logic changed, treat downstream `aether update --force`, local `aether version`, and npm bootstrap verification as part of release integrity
- Stable/public runtime uses `aether` + `~/.aether/`; dev/maintainer runtime uses `aether-dev` + `~/.aether-dev/`
- Dev installs intentionally skip global Claude/OpenCode/Codex home sync by default so source-development does not overwrite the stable command surface on the same machine
- If a release tag exists but `gh run list --workflow Release` shows no run and `gh release view vX.Y.Z` shows no release, report that the GitHub release did not materialize and recommend `gh workflow run Release -f tag=vX.Y.Z` first
- If `gh workflow run Release -f tag=vX.Y.Z` returns `HTTP 422: Actions has been disabled for this user`, report that as a GitHub Actions actor/policy failure and recommend another maintainer or the local GoReleaser fallback
- If GitHub workflow dispatch is unavailable or still fails, recommend `export GITHUB_TOKEN=\"$(gh auth token)\" && goreleaser release --clean` before npm publish
- Treat `git push --tags` as a release risk. Push a single annotated release tag instead

## Version Compatibility

- COLONY_STATE.json `version` field is "3.0" (string)
- Legacy state names (PAUSED, PLANNED, SEALED, ENTOMBED) are auto-normalized on load
- No structural migration framework exists
- Source-checkout release version detection uses `.aether/version.json`; installed/runtime detection uses ldflags or the hub version
- Cache files are expected and auto-rebuilt — not data corruption
