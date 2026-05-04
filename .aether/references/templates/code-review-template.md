---
schema_version: "1.0"
id: code-review-template
kind: template
category: templates
title: Code Review Template
description: "Output format for concise, severity-ordered Aether code reviews."
output_types: [code-review, review-output]
agent_roles: [watcher, auditor, scout, architect, queen]
task_types: [review, code, audit]
task_keywords: [review, code review, findings, bugs, tests, severity, blocker, warning, advisory, residual]
workflow_triggers: [continue, seal]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Code Review Template

## Findings

Lead with findings. Order by severity.

Use this format:

```text
Severity: blocker | warning | advisory
File: path/to/file.go:123
Issue: what is wrong
Impact: why it matters
Fix: concrete correction
```

If no issues are found, say:

```text
No blocking findings found.
Residual risk: <test gap or remaining uncertainty>
```

## Open Questions

Only include questions that affect correctness, safety, or scope.

## Verification Reviewed

List tests, commands, diffs, or inspections reviewed.

## Summary

Keep this secondary. Do not bury findings under a long summary.

For beginners: this structure makes the review useful because the important problems come first.
