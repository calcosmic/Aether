---
schema_version: "1.0"
id: brownfield-codebase-map
kind: field-guide
category: field-guides
title: Brownfield Codebase Map
description: "Guide for mapping an existing repository before planning or changing it."
output_types: [codebase-map, colonize-report, source-audit]
agent_roles: [scout, architect, surveyor-nest, queen, surveyor-disciplines]
task_types: [colonize, map, audit, architecture]
task_keywords: [brownfield, codebase, map, architecture, directories, existing, entrypoint, structure, protected]
workflow_triggers: [colonize, plan]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 4200
  sections: [Use When, Map Layers, Questions, Output]
---
# Brownfield Codebase Map

## Use When

Use this when Aether enters an existing repo or when planning depends on current architecture.

For beginners: before changing a house, map the rooms, pipes, and wiring.

## Map Layers

- entrypoints and commands
- package/module layout
- state and data directories
- generated files
- tests and fixtures
- external integrations
- platform-specific surfaces
- local-only runtime state

## Questions

- Where does user-facing behavior enter?
- Which files are source, generated, mirror, or local state?
- What owns persistence?
- What commands prove the app works?
- What is unsafe to touch?
- Where are hidden coupling points?

## Output

Produce:

- architecture summary
- important directories
- likely change surfaces
- protected paths
- test commands
- unknowns and risks
