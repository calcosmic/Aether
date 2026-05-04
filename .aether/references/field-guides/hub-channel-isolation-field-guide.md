---
schema_version: "1.0"
id: hub-channel-isolation-field-guide
kind: field-guide
category: field-guides
title: Hub Channel Isolation Field Guide
description: "Stable and dev channel separation, hub directory naming, and binary naming conventions."
output_types: [distribution-review, architecture-review, publish-plan]
agent_roles: [builder, watcher, architect, queen, porter]
task_types: [channel, hub, publish, distribution, isolation]
task_keywords: [channel, hub, stable, dev, publish, binary, isolation, directory, stale, drift, version, AETHER_CHANNEL]
workflow_triggers: [build, publish]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---

# Hub Channel Isolation Field Guide

This guide explains how Aether separates the stable and dev channels, including
hub directory paths, binary naming, detection order, and publish commands.

## For Beginners

Aether has two "channels" -- think of them as two separate installations on
your machine. The stable channel is what production projects use. The dev
channel is for developing Aether itself. They never interfere with each other
because they use different directories and different binary names. This
separation lets you work on Aether's code without breaking projects that depend
on the stable version.

## Channel Overview

| Aspect | Stable Channel | Dev Channel |
|--------|---------------|-------------|
| Hub directory | `~/.aether/` | `~/.aether-dev/` |
| Binary name | `aether` | `aether-dev` |
| Purpose | Production projects | Aether source development |
| Published via | `aether publish` | `aether publish --channel dev` |
| Updated via | `aether update` | `aether-dev update` |

## Hub Directory Structure

### Stable Hub (`~/.aether/`)

```
~/.aether/
├── system/           # Companion file source (populated by install/publish)
│   ├── commands/     # Slash command sources
│   ├── agents/       # Agent definition sources
│   ├── skills/       # Skill definitions
│   ├── references/   # Reference library
│   └── rules/        # Development rules
├── QUEEN.md          # Hub-level queen wisdom and user preferences
├── hive/             # Cross-colony wisdom (hive brain)
│   └── wisdom.json   # 200-entry cap, LRU eviction
├── registry/         # Colony registry (tracks all repos)
├── eternal/          # Legacy eternal memory
└── skills/           # Installed skills (colony/ + domain/)
```

### Dev Hub (`~/.aether-dev/`)

Mirrors the stable structure but under `~/.aether-dev/`. This allows
isolated testing of new features, skills, and references without affecting
production colonies.

```
~/.aether-dev/
├── system/           # Dev companion files
├── QUEEN.md          # Dev queen wisdom (separate from stable)
├── hive/             # Dev hive brain
└── ...
```

## Channel Detection Order

When the Aether runtime starts, it determines which channel to use through
this priority order:

1. **Environment variable.** `AETHER_CHANNEL=dev` forces the dev channel.
   `AETHER_CHANNEL=stable` forces the stable channel.

2. **Binary name.** If the binary is named `aether-dev`, the dev channel is
   selected. If named `aether`, the stable channel is selected.

3. **Flag.** The `--channel` flag overrides both env and binary name:
   `aether --channel dev publish`.

4. **Default.** If none of the above indicate a channel, stable is assumed.

This detection order means that installing a dev binary as `aether-dev` in
your PATH automatically routes to the dev hub without any additional
configuration.

## Publish Commands by Channel

### Publishing to Stable

```bash
aether publish
```

This builds the binary, syncs companion files to `~/.aether/system/`, and
verifies version agreement. Production repos can then pull updates with
`aether update --force`.

### Publishing to Dev

```bash
aether publish --channel dev --binary-dest "$HOME/.local/bin"
```

This publishes to `~/.aether-dev/system/` and copies the dev binary to the
specified destination. Target repos use:

```bash
aether-dev update --force
```

### Cross-Channel Safety

The publish system prevents accidental cross-channel contamination:

- Publishing to stable never writes to `~/.aether-dev/`
- Publishing to dev never writes to `~/.aether/`
- Update commands only read from their respective hub

## Version Checking

Each channel tracks its own version. Use these commands to verify:

```bash
aether version --check          # Stable: verify binary and hub agree
aether-dev version --check      # Dev: verify dev binary and dev hub agree
```

A mismatch (exit code non-zero) indicates that the hub is out of sync with
the binary, typically resolved by republishing or re-running update.

## Integrity Verification

The `aether integrity` command validates the full release pipeline chain for
the active channel:

- Source files in `.aether/` match what is in the hub
- Binary version matches hub version
- Companion files are complete (no missing agents, commands, or skills)
- Downstream simulation passes (what would happen if a target repo updated)

Run this before sealing or releasing to catch drift issues early.

## Practical Guidelines

**When developing Aether itself:**
- Use the dev channel to avoid breaking production projects
- `aether publish --channel dev --binary-dest "$HOME/.local/bin"` after changes
- Test in a scratch repo with `aether-dev update --force`

**When using Aether in production projects:**
- Use the stable channel exclusively
- `aether update --force` to pull the latest stable publish
- Never point production projects at the dev hub

**When switching between channels:**
- Check `aether version` vs `aether-dev version` to see which is ahead
- Each channel's queen state is independent
- Pheromones, instincts, and colony state are repo-local and channel-agnostic
