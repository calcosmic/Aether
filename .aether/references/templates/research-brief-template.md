---
schema_version: "1.0"
id: research-brief-template
kind: template
category: templates
title: Research Brief Template
description: "General Oracle research brief structure for custom topics."
output_types: [research-brief, custom]
agent_roles: [oracle, scout, architect, queen]
task_types: [research, brief, analysis]
task_keywords: [research, investigate, brief, summarize, recommend, options, ralf, findings, evidence, confidence]
workflow_triggers: [oracle, plan]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---
# Research Brief Template

## Research Question

State the question in one sentence. If the request is vague, rewrite it into the decision the user is probably trying to make.

For beginners: this keeps the research from becoming a pile of interesting but unusable facts.

## Context

Explain why the question matters to Aether. Name the affected workflow, worker role, platform, command, or distribution path.

## Findings

Group findings by decision relevance, not by source. Each finding should say what it means for the work.

## Evidence

List the strongest evidence. Use primary sources for current facts. Mark inference separately from sourced fact.

## Options

Describe practical options:

- Option A.
- Option B.
- Keep current behavior.

For each option, name cost, risk, and verification.

## Recommendation

Give a clear recommendation and the conditions under which it changes.

## Implementation Implications

Name likely files, tests, docs, and rollout steps.

## Open Questions

List only unresolved questions that would change the recommendation.
