---
schema_version: "1.0"
id: gate-decision-example
kind: example
category: examples
title: Gate Decision Example
description: "Example of a pass-with-warning quality gate decision."
output_types: [gate-output-example, quality-gate]
agent_roles: [watcher, auditor, queen, gatekeeper, probe, measurer]
task_types: [gate, example, review]
task_keywords: [example, gate, warning, pass, evidence, blocker, advisory, residual, severity]
workflow_triggers: [continue]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3200
---
# Gate Decision Example

## Decision

`pass_with_warnings`

## Scope

Reviewed reference loader, reference tests, update sync behavior, and docs updates.

## Evidence

- `go build ./cmd/aether`: pass.
- `go test ./cmd -run Reference -count=1`: pass.
- `reference-match --role oracle --task "evaluate React vs Vue" --output-type tech-eval`: matched `oracle-tech-evaluation`.

## Findings

### Blockers

None.

### Warnings

Full command suite still has unrelated failures in build-dispatch tests. This does not block the reference change but must be addressed before release.

### Advisories

Add more examples in a later reference content pass.

## Next Action

Advance this change only as a scoped reference-library update, not as a full runtime release.
