---
schema_version: "1.0"
id: reference-library-contract
kind: contract
category: contracts
title: Reference Library Contract
description: "Structural contract for Aether global references."
output_types: [reference-system, distribution-review, architecture-review]
agent_roles: [queen, architect, builder, watcher, oracle, scout]
task_types: [reference, distribution, install, update, architecture]
task_keywords: [reference, references, library, global, hub, install, update, distribution, category, frontmatter, index]
workflow_triggers: [plan, build, continue]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Reference Library Contract

## Purpose

References are global structural guidance for Aether workers. They are templates, rubrics, contracts, playbooks, examples, and field guides. They are not domain skills and they are not target-repo project files.

For beginners: skills tell a worker how to do a kind of work. References give the worker the form, checklist, or quality bar for the output.

## Source Layout

The repo source of truth is:

```text
.aether/references/{category}/{id}.md
```

Allowed categories:

- `contracts`
- `examples`
- `field-guides`
- `playbooks`
- `rubrics`
- `templates`

Each markdown file must be named after its `id`. Do not use repeated generic names like `REFERENCE.md`.

## Installed Layout

The active global library is installed to:

```text
~/.aether/references/{category}/{id}.md
```

The staged shipped copy is:

```text
~/.aether/system/references/{category}/{id}.md
```

## Target Repo Rule

Target project repositories must not receive `.aether/references/` in v1. This avoids confusing source truth, installed truth, and project-local state.

## Frontmatter Requirements

Each reference needs:

- `schema_version`
- `id`
- `kind`
- `category`
- `title`
- `description`
- `output_types`
- `agent_roles`
- `task_types` or `task_keywords`
- `priority`
- `version`

## Matching Rule

Match references by:

- Agent role.
- Requested output type.
- Task type and task keywords.
- Workflow trigger when available.

Inject the top 2 matches. Scoring weights: output_type match = 4 points, role match = 3 points, task match = 2 points. Task matching uses substring containment after normalization (lowercase, remove hyphens/underscores). References should be precise, not a giant dump of every useful document.
