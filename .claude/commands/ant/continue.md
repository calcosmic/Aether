---
name: ant:continue
description: "➡️🐜🚪🐜➡️ Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Execute `/ant:continue` through modular playbooks.

## Purpose

This command is intentionally short. It orchestrates verification and phase
advancement via smaller playbooks. This improves instruction-following reliability
without changing continue behavior.

## Rules

1. Do **not** invoke nested slash commands like `/ant:continue-verify`.
2. Use the Read tool to load each playbook file, then execute it.
3. Preserve variables/results from prior stages and pass them forward.
4. Enforce all gate stops exactly as defined in the playbooks.
5. Keep existing behavior and output format from the playbooks.

## Stage Order

Run these stages in order:

1. `.aether/docs/command-playbooks/continue-verify.md`
2. `.aether/docs/command-playbooks/continue-gates.md`
3. `.aether/docs/command-playbooks/continue-advance.md`
4. `.aether/docs/command-playbooks/continue-finalize.md`

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

- `visual_mode`
- `state`
- `current_phase`
- `verification_results`
- `gate_results`
- `advancement_result`
- `next_phase_id`
- `completion_state`

## Final Output

After `continue-finalize.md` finishes, return the normal continue summary and
next-step routing exactly as defined there.
