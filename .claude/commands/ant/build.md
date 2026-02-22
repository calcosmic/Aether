---
name: ant:build
description: "🔨🐜🏗️🐜🔨 Build a phase with pure emergence - colony self-organizes and completes tasks"
---

You are the **Queen**. Execute `/ant:build` through modular playbooks.

The phase to build is: `$ARGUMENTS`

## Purpose

This command is intentionally short. It orchestrates build execution by loading
smaller playbooks in sequence. This improves instruction-following reliability
without changing build behavior.

## Rules

1. Do **not** invoke nested slash commands like `/ant:build-prep`.
2. Use the Read tool to load each playbook file, then execute it.
3. Preserve variables/results from prior stages and pass them forward.
4. Stop immediately on hard failure conditions defined in any stage.
5. Keep existing behavior and output format from the playbooks.

## Stage Order

Run these stages in order:

1. `.aether/docs/command-playbooks/build-prep.md`
2. `.aether/docs/command-playbooks/build-context.md`
3. `.aether/docs/command-playbooks/build-wave.md`
4. `.aether/docs/command-playbooks/build-verify.md`
5. `.aether/docs/command-playbooks/build-complete.md`

## Execution Contract

For each stage:

1. Read the file with the Read tool.
2. Execute the instructions exactly as written.
3. Keep an in-memory stage result record:
   - `stage_name`
   - `status` (`ok` or `failed`)
   - `key_outputs` (values needed downstream)
4. If `status == failed`, halt and report the failure with recovery options.

## Required Cross-Stage State

Carry these values forward when produced:

- `phase_id`
- `visual_mode`
- `verbose_mode`
- `suggest_enabled`
- `cli_model_override`
- `prompt_section`
- `wave_results`
- `verification_status`
- `synthesis_status`
- `next_action`

## Final Output

After `build-complete.md` finishes, return the normal build summary and next-step
routing exactly as defined there.
