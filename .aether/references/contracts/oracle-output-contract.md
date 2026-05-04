---
schema_version: "1.0"
id: oracle-output-contract
kind: contract
category: contracts
title: Oracle Output Contract
description: "Minimum standard for Oracle research and recommendations."
output_types: [research-brief, tech-eval, architecture-review, bug-investigation, prd]
agent_roles: [oracle, queen, architect]
task_types: [research, evaluation, architecture, bug, requirements]
task_keywords: [oracle, recommendation, confidence, evidence, research, decision, ralf, template, loop, synthesis]
workflow_triggers: [oracle]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4600
---
# Oracle Output Contract

## Purpose

The Oracle turns uncertainty into a decision-ready document. It must not produce a long essay that leaves the builder guessing what to do.

For beginners: the Oracle's job is to research, judge, and explain the next move.

## Required Sections

Every Oracle output must include:

1. `Question`: the decision or investigation being answered.
2. `Context`: why it matters to Aether.
3. `Findings`: the important facts.
4. `Evidence`: sources, files, commands, or observations.
5. `Options`: realistic choices.
6. `Recommendation`: one clear path.
7. `Confidence`: high, medium, or low, with reasons.
8. `Risks`: what could go wrong.
9. `Implementation Implications`: files, workflows, or tests affected.
10. `Open Questions`: only questions that could change the decision.

## Evidence Rules

- Current external facts need current sources.
- Local architecture claims need file references or command output.
- Inference must be labeled as inference.
- Missing evidence should reduce confidence.
- Do not cite stale docs as runtime truth when code disagrees.

## Confidence Rubric

### High

Multiple strong sources agree, the local code path is inspected, and the recommendation has a clear validation path.

### Medium

Evidence supports the recommendation, but at least one meaningful unknown remains.

### Low

The answer is provisional because sources are incomplete, the code path is not inspectable, or assumptions dominate the decision.

## Recommendation Rules

The recommendation must include:

- What to do.
- Why it is better than alternatives.
- How to verify it.
- How to reverse it if wrong.
