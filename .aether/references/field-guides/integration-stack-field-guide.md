---
schema_version: "1.0"
id: integration-stack-field-guide
kind: field-guide
category: field-guides
title: Integration Stack Field Guide
description: "Guide for mapping external tools, APIs, package managers, and runtime dependencies."
output_types: [integration-map, provisions-map]
agent_roles: [surveyor-provisions, ambassador, scout, architect, queen]
task_types: [integration, dependency, stack, map]
task_keywords: [integration, dependency, package, API, stack, external, binary, lockfile, offline, pinned]
workflow_triggers: [colonize, plan, build]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3800
  sections: [Use When, Scan Targets, Risk Questions, Output]
---
# Integration Stack Field Guide

## Use When

Use this when a task depends on external tools, APIs, package managers, binaries, or hosted services.

For beginners: this lists what the project relies on outside its own code.

## Scan Targets

- package manifests
- lockfiles
- CI workflows
- Dockerfiles
- environment variable examples
- API clients
- SDK setup
- binary download code
- deployment config

## Risk Questions

- Is the dependency current and maintained?
- Is the version pinned?
- Does install/update require network access?
- Are credentials needed?
- What fails offline?
- What is the rollback path?

## Output

Produce a stack summary, critical dependencies, external service assumptions, and verification commands.
