---
schema_version: "1.0"
id: doc-ingestion-conflict-field-guide
kind: field-guide
category: field-guides
title: Documentation Ingestion Conflict Field Guide
description: "How Aether reconciles user docs, generated docs, command wrappers, and runtime source truth."
output_types: [documentation-review, source-audit, conflict-review]
agent_roles: [chronicler, scout, oracle, queen, architect, watcher]
task_types: [documentation, ingest, audit, reconcile, update]
task_keywords: [docs, documentation, conflict, source of truth, wrapper, generated, stale, reconcile, authority, drift]
workflow_triggers: [colonize, plan, build, update]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: sections
  max_chars: 3800
  sections: [Use When, Authority Order, Conflict Handling, Output Requirements]
---
# Documentation Ingestion Conflict Field Guide

## Use When

Use this when documentation, command wrappers, generated mirrors, and runtime code disagree.

For beginners: docs are useful, but executable code and source-of-truth files decide what the system really does.

## Authority Order

When sources conflict, prefer:

1. Latest explicit user correction.
2. Executable runtime code.
3. Source-of-truth companion files in `.aether/`.
4. Platform-native maintained files in `.claude/`, `.opencode/`, and `.codex/`.
5. Distributed docs and command playbooks.
6. Generated mirrors and installed hub copies.
7. Old notes, stale README text, and local scratch files.

Never treat `~/.aether/` as source truth when working inside the Aether repo. It is an install target.

## Conflict Handling

- Preserve useful intent from stale docs, but rewrite it against current code.
- If generated mirrors differ from source files, update the source and mirror together or document why not.
- Do not copy branding, command names, or workflow names from another system without converting them into Aether language and behavior.
- If the user rejects a model, update both docs and implementation assumptions.

## Output Requirements

Documentation review output should state:

- Which source won.
- Which sources were stale.
- What was changed.
- What still needs regeneration or publication.
