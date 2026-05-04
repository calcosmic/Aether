---
schema_version: "1.0"
id: security-gate-rubric
kind: rubric
category: rubrics
title: Security Gate Rubric
description: "Rubric for Aether security review across secrets, command execution, file writes, and dependency changes."
output_types: [security-review, quality-gate]
agent_roles: [gatekeeper, auditor, watcher, queen, builder]
task_types: [security, audit, gate, dependency, command]
task_keywords: [security, secret, credential, shell, injection, dependency, permission, sanitizer, destructive, evidence]
workflow_triggers: [continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3600
  sections: [Use When, Critical Checks, Blockers, Evidence]
---
# Security Gate Rubric

## Use When

Use this when code executes shell commands, writes files, changes install/update behavior, handles external input, or introduces dependencies.

For beginners: this checks whether a change can leak secrets, run unsafe commands, or damage files.

## Critical Checks

- No secrets or tokens in source or generated output.
- User-provided strings are not interpolated into shell commands unsafely.
- File paths are validated and scoped.
- Destructive operations require explicit intent.
- Dependencies are justified and maintained.
- Logs do not expose sensitive state.
- Prompt injection content is sanitized before storage or injection.

## Blockers

- Hardcoded credentials.
- Unbounded delete/copy behavior.
- Shell injection path.
- Network call with sensitive data and no reason.
- Dependency with known severe vulnerability and no mitigation.

## Evidence

Good evidence includes code inspection, targeted tests for path/shell sanitization, dependency audit output, and secret scan results.
