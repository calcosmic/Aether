---
schema_version: "1.0"
id: continue-verification-playbook
kind: playbook
category: playbooks
title: Continue Verification Playbook
description: "Operational playbook for `aether continue` verification and advancement."
output_types: [continue-plan, verification-playbook]
agent_roles: [queen, watcher, auditor, probe, gatekeeper, measurer]
task_types: [continue, verify, advance, gate]
task_keywords: [continue, verification, advance, gate, watcher, claims, evidence, midden, blocker, depth]
workflow_triggers: [continue]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---
# Continue Verification Playbook

## Use When

Use this for `aether continue`, phase advancement, and post-build verification.

For beginners: continue is the quality check and phase movement step.

## Inputs

- Last build claims.
- Worker handoffs.
- Changed files.
- Phase acceptance criteria.
- Active blockers and pheromones.
- Queen-selected review depth.

## Verification Flow

1. Check that build artifacts exist and are fresh.
2. Map claims to evidence.
3. Run focused verification.
4. Run gates appropriate to risk.
5. Record failures in midden or gate results.
6. Advance only if blockers are clear.

## Advancement Rules

- Build completion is not enough.
- Missing evidence blocks high-risk work.
- Warnings can advance only when explicit.
- State updates must be consistent and recoverable.

## Output

Return:

- decision
- evidence
- blockers
- warnings
- next phase or completion state

## Failure Handling

If verification fails, keep the phase active and give the exact next repair task.
