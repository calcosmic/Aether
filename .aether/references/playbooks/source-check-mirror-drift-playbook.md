---
schema_version: "1.0"
id: source-check-mirror-drift-playbook
kind: playbook
category: playbooks
title: Source Check And Source Drift Playbook
description: "What aether source-check validates and how to fix drift between canonical sources and generated wrappers."
output_types: [source-audit, drift-report, parity-review]
agent_roles: [builder, watcher, architect, queen, chronicler]
task_types: [source, wrapper, drift, check, parity]
task_keywords: [source-check, drift, wrapper, agent, generated, yaml, parity, embed, asset, fix, mismatched]
workflow_triggers: [build, seal]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---

# Source Check And Source Drift Playbook

This playbook describes what `aether source-check` validates, how to interpret
drift reports, and the steps to fix drift between source files and generated
wrappers.

## For Beginners

Aether has a "source chain" -- files are defined in one place (the source) and
then published or generated into platform-specific surfaces. For example,
a command defined in `.aether/commands/*.yaml` gets turned into markdown files
in `.claude/commands/ant/` and `.opencode/commands/ant/`.
When the source changes but the generated wrappers do not, "drift" occurs.
This playbook shows how to detect and fix that drift.

## What Source-Check Validates

The `aether source-check` command validates source surfaces and generated wrapper
relationships.

### 1. Command Wrappers

**Source:** `.aether/commands/*.yaml` (YAML command definitions)
**Generated surfaces:** Markdown headers in `.claude/commands/ant/*.md`,
`.opencode/commands/ant/*.md`

The check verifies that each YAML source file has corresponding wrapper files
in each platform directory, and that the generated headers in those wrappers
match the YAML source metadata (name, description, arguments).

**What drift looks like:**
- A YAML file has a new description but the markdown headers still show the
  old description
- A YAML file is added but no corresponding markdown wrapper exists
- A markdown wrapper exists but its YAML source has been removed

### 2. Canonical Source Surfaces

**Sources:**
- `.claude/agents/ant/*.md` (Claude agent definitions)
- `.opencode/agents/*.md` (OpenCode agent definitions)
- `.codex/agents/*.toml` (Codex agent definitions)
- `.aether/commands/*.yaml`
- `.aether/skills/**/SKILL.md`
- `.aether/templates/`, `.aether/docs/`, `.aether/utils/`
- Aether repo companion sources for exchange modules, references, and workers

The check verifies:
- Canonical source directories and files are present in the Aether repo
- Retired packaging mirror directories have not been recreated
- Platform command wrappers point at real YAML sources

**What drift looks like:**
- A source directory is missing from the Aether checkout
- A retired packaging mirror directory is recreated by mistake
- A Codex TOML file missing a newly added agent

### 3. Published Assets

The source checkout is the authority. `aether publish` or
`aether install --package-dir` pushes companion files to the global hub and
platform homes. Target repos keep repo-local state only.

## Running Source-Check

```bash
aether source-check
```

Output includes:

```
Source Check Results
────────────────────
Canonical sources:   pass
Retired mirrors:     pass
Command wrappers:    pass
```

Each mismatched or missing entry includes:
- `source_path`: Where the authoritative file lives
- `path`: Which source or generated wrapper is affected
- `issue`: Description of the drift

## Fixing Drift

### Step 1: Identify the Drift

Run `aether source-check` and review the output. Focus on mismatched and
missing entries. Each entry tells you exactly which source and mirror are out
of sync.

### Step 2: Determine the Direction

Ask: did the source change, or did the generated wrapper/platform source change?

- **Source changed, generated wrapper stale.** This is the common case. The
  source was updated and the wrapper was not regenerated. Fix by updating the
  generated wrapper to match the source.

- **Generated wrapper changed, source stale.** Someone edited a generated
  wrapper without updating the YAML. Fix by porting valid changes back to the
  YAML source, then regenerating the wrapper.

- **Both changed.** Merge the changes. The source is authoritative, so
  prioritize source changes and incorporate any valid wrapper-only additions.

### Step 3: Fix Command Wrapper Drift

For command wrappers:

1. Edit the source YAML in `.aether/commands/*.yaml`
2. Regenerate the markdown wrappers by running the publish or install command:
   ```bash
   aether publish
   ```
   Or for dev channel:
   ```bash
   aether publish --channel dev --binary-dest "$HOME/.local/bin"
   ```
3. The publish process regenerates all wrappers from their YAML sources

### Step 4: Fix Agent Source Drift

For agent sources:

1. Edit the canonical platform source directly:
   - Claude: `.claude/agents/ant/*.md`
   - OpenCode: `.opencode/agents/*.md`
   - Codex: `.codex/agents/*.toml`
2. Keep role purpose, safety boundaries, and output expectations aligned.
3. Publish from the Aether repo so the hub and platform homes refresh from
   those canonical sources.

### Step 5: Fix Embedded Asset Drift

For embedded assets:

1. Update the source file in `.aether/templates/`, `.aether/docs/`, or
   `.aether/skills/`
2. Rebuild the binary to pick up the embedded changes:
   ```bash
   go build ./cmd/aether
   ```
3. Republish to propagate to the hub

### Step 6: Verify the Fix

```bash
aether source-check
```

All categories should show zero mismatched and zero missing entries.

## Prevention

- **Always edit the source first.** Do not recreate retired packaging mirrors.
  The source chain is: YAML/platform sources -> generated wrappers -> hub.
- **Run source-check before committing.** Add it to your pre-commit or CI
  workflow.
- **Publish after source changes.** The publish command regenerates mirrors.
- **Use `aether integrity` for comprehensive validation.** It includes
  source-check as part of the full pipeline validation.

## Common Pitfalls

- Editing `.claude/commands/ant/build.md` directly instead of the YAML source.
  The header says "DO NOT EDIT DIRECTLY" for this reason.
- Adding an agent to one platform but forgetting the corresponding OpenCode or
  Codex source update.
- Forgetting that Codex agents use TOML format, not markdown, so they need
  a separate translation step.
- Running source-check after editing but before publishing -- the mirrors
  are not yet updated at that point.
