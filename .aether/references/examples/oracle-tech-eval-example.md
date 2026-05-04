---
schema_version: "1.0"
id: oracle-tech-eval-example
kind: example
category: examples
title: Oracle Technology Evaluation Example
description: "Example shape for a concise technology evaluation."
output_types: [tech-eval-example, technology-evaluation]
agent_roles: [oracle, architect, queen]
task_types: [evaluation, research, example]
task_keywords: [example, technology, evaluation, recommendation, ralf, confidence, tradeoff, matrix]
workflow_triggers: [oracle]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3400
---
# Oracle Technology Evaluation Example

## Question

Should Aether use a new markdown parser for reference frontmatter and section extraction?

## Context

Aether references need frontmatter parsing, content rendering, and predictable prompt budgets.

## Options

| Option | Strength | Risk | Verdict |
|---|---|---|---|
| Existing YAML plus simple markdown helpers | Small, local, testable | Limited markdown edge cases | Prefer now |
| Full markdown AST dependency | Robust parsing | New dependency and maintenance cost | Defer |
| String-only parser | No dependency | Fragile frontmatter and sections | Reject |

## Recommendation

Use existing YAML parsing with small tested markdown helpers. Revisit an AST dependency only if references need nested section rendering or markdown transformations.

## Confidence

Medium. The local need is simple, but future reference authoring could demand stronger parsing.

## Verification

Unit tests for frontmatter parsing, section extraction, render caps, and CLI match output.
