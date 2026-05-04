---
schema_version: "1.0"
id: gate-taxonomy
kind: rubric
category: rubrics
title: Gate Taxonomy
description: "How Aether classifies blockers, warnings, and advisory findings during review."
output_types: [quality-gate, code-review, risk-review, gate-output]
agent_roles: [watcher, gatekeeper, auditor, probe, measurer, queen]
task_types: [review, gate, verify, audit, risk]
task_keywords: [blocker, warning, advisory, gate, review, risk, approve, request changes, severity, evidence, pass, block]
workflow_triggers: [continue, seal]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3800
  sections: [Use When, Severity Classes, Gate Decision Rules, Output Format]
---
# Gate Taxonomy

## Use When

Use this when reviewing work for advancement, deciding whether a phase can continue, or translating specialist findings into a clear go/no-go outcome.

For beginners: this prevents every concern from becoming either a panic or a shrug. It sorts issues by whether they actually block shipping.

## Severity Classes

### Blocker

A blocker means the work cannot safely advance. Use this only when at least one of these is true:

- The requested behavior is missing or broken.
- Tests or build fail in a way related to the change.
- A security, data loss, privacy, or destructive-operation risk is credible.
- Install/update/runtime distribution would damage user files.
- The worker output is not reproducible enough to verify.

### Warning

A warning means the work can advance only with an explicit tradeoff noted. Use this when:

- The implementation works but has a known limitation.
- Coverage is narrower than ideal but proportional to risk.
- A follow-up task is needed soon but not required for this phase.
- Behavior changed in a user-visible way that is probably acceptable.

### Advisory

An advisory is a non-blocking improvement. Use this for style, naming, minor duplication, documentation polish, or future hardening that does not affect correctness.

## Gate Decision Rules

- Any blocker prevents advancement.
- Multiple warnings can become a blocker if they point to the same unresolved risk.
- Advisory findings must not block advancement.
- Missing evidence is a blocker only when the changed surface requires evidence.
- Do not demand broad verification for tiny low-risk edits, but do demand precise evidence.

## Output Format

Gate output should be direct:

1. `Decision: pass`, `pass with warnings`, or `block`.
2. Findings ordered by severity.
3. Evidence reviewed.
4. Required next action if blocked.
