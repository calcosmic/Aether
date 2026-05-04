---
schema_version: "1.0"
id: oracle-tech-evaluation
kind: template
category: templates
title: Oracle Technology Evaluation Template
description: "Deep research structure for evaluating technologies, libraries, tools, or architectural options."
output_types: [tech-eval, technology-evaluation, tech-eval-example]
agent_roles: [oracle, architect, scout, queen]
task_types: [research, evaluation, architecture, integration, selection]
task_keywords: [evaluate, compare, versus, vs, library, framework, technology, tradeoff, adoption, ralf, recommendation, matrix]
workflow_triggers: [oracle, plan]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 5200
---
# Oracle Technology Evaluation Template

## Use When

Use this when the Oracle evaluates a technology choice, compares options, or recommends whether Aether should adopt a dependency, platform, framework, API, or architecture pattern.

For beginners: this makes the research answer useful for a builder. It does not just say which option is popular; it explains what will work here and why.

## Evaluation Question

State the decision in one sentence:

`Should Aether use <option> for <specific job> under <constraints>?`

List the options being compared. If the user gave only one option, include "do nothing / keep current approach" as the baseline.

## Aether Context

Describe the parts of Aether affected:

- Runtime code path.
- Companion file distribution.
- Agent or prompt behavior.
- State files and migration concerns.
- Platform parity across Claude, OpenCode, and Codex.
- User-local safety boundaries.

## Evaluation Criteria

Score each option against:

1. Correctness fit.
2. Maintenance cost.
3. Distribution impact.
4. Security and privacy posture.
5. Testability.
6. Failure modes.
7. User complexity.
8. Migration and rollback cost.

## Evidence

Use primary sources where current facts matter: official docs, release notes, source code, API references, standards, or repository issues. Name what each source proves. If browsing was needed, include source links.

## Tradeoff Matrix

Use a compact table:

| Option | Strengths | Risks | Aether Fit | Verdict |
|---|---|---|---|---|

## Recommendation

State one recommendation:

- `Adopt`: use it now.
- `Trial`: spike behind a narrow boundary.
- `Defer`: promising but not now.
- `Reject`: not suitable for Aether.

Include the implementation boundary, validation plan, and rollback path.

## Open Questions

List only questions that affect the decision. Do not include general curiosity.
