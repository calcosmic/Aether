# Playbook: Colonize

## Overview

Analyze an existing codebase to establish territory context and initialize colony state. Colonize runs after `/ant-init` when the repo already contains code.

## Stage 1: Survey Dispatch

### Step 1: Load Colony State

Run `aether load-state`. Verify colony is initialized. Release lock via `aether unload-state`.

### Step 2: Territory Survey

Spawn surveyor ants based on codebase structure:

- **surveyor-nest** — Maps directory structure and file organization
- **surveyor-disciplines** — Documents coding conventions and patterns
- **surveyor-pathogens** — Identifies tech debt and problem areas
- **surveyor-provisions** — Maps dependencies and external integrations

For each surveyor:
- Generate name: `aether generate-ant-name "surveyor-{type}"`
- Log spawn: `aether spawn-log --parent "Queen" --caste "surveyor" --name "{name}" --task "{survey_type}" --depth 0`
- Spawn via Task tool with `subagent_type="aether-surveyor-{type}"`

### Step 3: Collect Survey Results

Wait for all surveyors to complete. Parse JSON results:
- `findings` — key observations
- `patterns` — conventions to follow
- `concerns` — areas of caution
- `recommendations` — suggested improvements

Log completion for each: `aether spawn-complete --name "{name}" --status "completed" --summary "{summary}"`

## Stage 2: State Initialization

### Step 4: Write Survey Artifacts

Write collected findings to `.aether/data/survey/`:
- `DISCIPLINES.md` — coding conventions
- `BLUEPRINT.md` — architecture overview
- `PATHOGENS.md` — tech debt register
- `PROVISIONS.md` — dependency map
- `CHAMBERS.md` — file organization
- `SENTINEL-PROTOCOLS.md` — testing conventions
- `TRAILS.md` — integration paths

### Step 5: Update Colony State

Mark colonize as completed in `COLONY_STATE.json`:
- Set `state` to `"READY"`
- Append event: `"<timestamp>|colonized|colonize|Territory surveyed"`
- Store survey metadata in `memory.survey`

### Step 6: Display Results

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🗺️ C O L O N Y   T E R R I T O R Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Surveyors dispatched: {count}
📄 Documents generated: {count}

Key findings:
  {bullet list of top findings}

Next step: /ant-plan to generate phases
```

### Step 7: Update Session

Run `aether session-update --command "/ant-colonize" --suggested-next "/ant-plan" --summary "Territory surveyed, ready to plan"`
