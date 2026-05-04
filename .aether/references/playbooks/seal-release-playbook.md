---
schema_version: "1.0"
id: seal-release-playbook
kind: playbook
category: playbooks
title: Seal Release Playbook
description: "Playbook for final release readiness and colony sealing."
output_types: [release-review, seal-plan]
agent_roles: [queen, watcher, gatekeeper, auditor, probe, measurer, porter]
task_types: [seal, release, final-review, gate]
task_keywords: [seal, release, final, readiness, crowned, gate, publish, evidence, residual, follow-up]
workflow_triggers: [seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Seal Release Playbook

## Use When

Use this before marking a colony complete or publishing release-sensitive work.

For beginners: seal is the final "are we actually done?" gate.

## Required Checks

- All phases are complete or explicitly skipped with reason.
- No active blockers remain.
- Verification evidence covers user-facing outcomes.
- Security, quality, coverage, and performance risks are reviewed.
- Docs and handoff are current.
- Release or publish steps are clear.

## Final Review Depth

Use stronger review for:

- runtime/state changes
- install/update/publish behavior
- security-sensitive work
- platform parity work
- broad refactors

## Seal Output

Include:

- summary of delivered outcome
- verification evidence
- known residual risks
- follow-up ideas
- user-facing next steps

## Blockers

Do not seal with failing core tests, unverified destructive paths, missing release instructions, or unresolved user corrections.
