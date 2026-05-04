---
schema_version: "1.0"
id: codebase-conventions-field-guide
kind: field-guide
category: field-guides
title: Codebase Conventions Field Guide
description: "Guide for identifying and preserving local coding, testing, naming, and layout conventions."
output_types: [conventions-map, code-review]
agent_roles: [surveyor-disciplines, scout, builder, watcher, queen]
task_types: [conventions, codebase, review, map]
task_keywords: [conventions, patterns, naming, tests, style, layout, error, fixture, style]
workflow_triggers: [colonize, build, continue]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3600
  sections: [Use When, Scan Targets, Output, Review Use]
---
# Codebase Conventions Field Guide

## Use When

Use this before implementing changes in unfamiliar code or when reviewing style drift.

For beginners: match the house style unless there is a good reason not to.

## Scan Targets

- naming patterns
- test layout
- error handling
- logging style
- command output style
- fixture structure
- helper APIs
- generated file conventions
- platform mirror rules

## Output

Produce:

- established patterns
- exceptions
- files that demonstrate the pattern
- risky areas where patterns conflict

## Review Use

Use conventions to guide implementation, but do not reject correct code only because a minor style choice differs.
