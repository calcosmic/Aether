---
schema_version: "1.0"
id: spec-refinement-template
kind: template
category: templates
title: Spec Refinement Template
description: "Template for turning vague work into falsifiable implementation specs."
output_types: [spec, implementation-spec]
agent_roles: [architect, builder, watcher, oracle, queen]
task_types: [spec, requirements, implementation, clarify]
task_keywords: [spec, acceptance, constraints, implementation, clarify, falsifiable, boundary, verification]
workflow_triggers: [discuss, plan, build]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---
# Spec Refinement Template

## Intent

Restate the user's desired outcome in one sentence.

## Current Behavior

Describe what the system does now. Reference files or commands if known.

## Desired Behavior

Describe the observable future behavior.

## Acceptance Criteria

Use concrete checks:

- Given `<state>`, when `<action>`, then `<result>`.
- Command `<command>` produces `<observable output>`.
- File `<path>` contains or does not contain `<condition>`.

## Constraints

List safety, platform, distribution, compatibility, and user preference constraints.

## Non-Goals

Exclude tempting adjacent work.

## Implementation Boundary

Name likely modules and files. Keep this as guidance, not a full design unless evidence supports it.

## Verification Plan

List unit tests, integration tests, smoke commands, and manual inspection.

For beginners: this turns "make it better" into "we can prove it is done."
