---
schema_version: "1.0"
id: oracle-ralf-playbook
kind: playbook
category: playbooks
title: Oracle RALF Playbook
description: "Playbook for iterative Oracle research using plan, evidence, gaps, and synthesis."
output_types: [oracle-plan, research-playbook]
agent_roles: [oracle, queen, architect, scout]
task_types: [oracle, research, synthesis, evaluation]
task_keywords: [oracle, RALF, research, synthesis, gaps, confidence, ralf, template, loop, iterate, evidence]
workflow_triggers: [oracle]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---
# Oracle RALF Playbook

## Use When

Use this for `aether oracle` iterative research.

For beginners: Oracle research should narrow uncertainty across iterations until it can recommend a next move.

## Loop Shape

1. Clarify the research question.
2. Build a research plan.
3. Gather evidence from local code, docs, web, or both.
4. Write findings.
5. Identify gaps.
6. Synthesize recommendation.
7. Stop when confidence target and completeness criteria are met.

## Evidence Rules

- Local architecture claims need file or command evidence.
- Current external facts need current primary sources.
- Inference must be labeled.
- Missing evidence should lower confidence.

## Output Types

Use the matching template:

- tech-eval
- architecture-review
- bug-investigation
- PRD
- research-brief

## Completion

The Oracle is done when the recommendation is actionable and the remaining unknowns are named.
