---
schema_version: "1.0"
id: skills-references-boundary-field-guide
kind: field-guide
category: field-guides
title: Skills And References Boundary Field Guide
description: "When to create a skill vs a reference, and how each system works in the Aether worker pipeline."
output_types: [architecture-review, skill-plan, reference-system]
agent_roles: [architect, queen, builder, chronicler, oracle, watcher]
task_types: [skill, reference, boundary, architecture, create]
task_keywords: [skill, reference, boundary, create, guidance, template, rubric, domain, behavior, injection, index, match]
workflow_triggers: [plan, build]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---

# Skills And References Boundary Field Guide

This guide explains the difference between skills and references in Aether,
when to create each, and how they flow through the worker pipeline.

## For Beginners

Aether has two systems that teach workers how to do their jobs:

- **Skills** teach a worker *how to work* -- coding patterns, testing
  conventions, error handling approaches. They are behavioral guidance.
- **References** teach a worker *what output should look like* -- review
  templates, quality rubrics, example decisions. They are structural templates.

If a skill is a cooking technique, a reference is a recipe card. You need
both to make good food, but they serve different purposes.

## Skills System

### What Skills Are

Skills are reusable behavior modules that shape how workers operate. They
provide domain knowledge and procedural guidance.

### Skill Categories

| Category | Count | Purpose |
|----------|-------|---------|
| Colony skills | 52 | Behavioral patterns (TDD, error handling, commit style) |
| Domain skills | 31+ | Technical knowledge (React patterns, Go idioms, database optimization) |

### Where Skills Live

| Location | Purpose |
|----------|---------|
| `.aether/skills/` | Source of truth (shipped with Aether) |
| `~/.aether/skills/` | Installed skills (hub-level, shared across colonies) |
| `~/.aether/skills/domain/` | User-created custom domain skills |

### How Skills Are Matched

1. `skill-index` builds a cached index of all available skills
2. `skill-detect` detects which domain skills match the codebase
3. `skill-match` scores each skill against:
   - Worker role (builder, watcher, scout, etc.)
   - Active pheromone signals (FOCUS, REDIRECT)
   - Codebase detection patterns
4. Top 3 colony skills + top 3 domain skills are selected per worker

### Skill Injection

- Skills have their own **8,000 character budget**, separate from colony-prime
- Injected into builder and watcher prompts
- Rendered as a dedicated skills section, independent of context capsule trimming

### Creating Custom Skills

Use `/ant-skill-create` to generate a skill from a description, or manually
create a `SKILL.md` file in `~/.aether/skills/domain/` with frontmatter
specifying name, category, detection patterns, and applicable roles.

## References System

### What References Are

References are output templates and review rubrics that guide how agents
structure their work products. They define the *shape* of the output, not the
*behavior* during work.

### Reference Categories

| Category | Purpose | Examples |
|----------|---------|---------|
| contracts | Rules that must be followed | State protection, execution policies |
| field-guides | How-to guides for specific systems | Template selection, budget trimming |
| playbooks | Step-by-step procedures | Source checking, stale publish recovery |
| rubrics | Quality scoring criteria | Builder discipline, watcher evidence |
| examples | Realistic example outputs | Queen decisions, handoff formats |
| templates | Reusable output structures | Report formats, review templates |

### Where References Live

| Location | Purpose |
|----------|---------|
| `.aether/references/{category}/` | Source of truth |
| `~/.aether/system/references/` | Hub staging (populated by publish) |
| `~/.aether/references/` | Installed global library used by workers |

### How References Are Matched

`reference-match` scores references using:
- Agent role (queen, builder, watcher, etc.)
- Task type (build, verify, review, etc.)
- Task keywords
- Output type hint (from `--template` flag for Oracle)

The **top 2 matches** are rendered into a `## References` prompt section.

## Decision Matrix: Skill or Reference?

| If you want to... | Create a... | Why |
|-------------------|-------------|-----|
| Teach a testing convention | Skill | It is behavioral guidance |
| Define what a review report looks like | Reference (rubric) | It is an output structure |
| Share Go error handling patterns | Skill (domain) | It is domain knowledge |
| Show an example Queen decision | Reference (example) | It is a sample output |
| Establish TDD discipline | Skill (colony) | It is a behavioral pattern |
| Define protected path rules | Reference (contract) | It is a rule that must be followed |
| Explain how to fix mirror drift | Reference (playbook) | It is a step-by-step procedure |
| Teach React component patterns | Skill (domain) | It is technical knowledge |
| Score builder output quality | Reference (rubric) | It is a quality evaluation framework |

### Key Distinction

**Create a Skill when:** The content tells a worker *how to approach work*
(conventions, patterns, practices, domain knowledge).

**Create a Reference when:** The content tells a worker *what the result should
look like* (templates, rubrics, examples, contracts, procedures).

## Pipeline Integration

Both systems inject into worker prompts, but at different points:

```
Worker Prompt Assembly
├── Colony-prime context capsule (6K/3K budget)
│   ├── QUEEN.md wisdom
│   ├── Pheromones
│   ├── Instincts
│   └── ... (trim order applies)
├── Skills section (8K budget, independent)
│   ├── Top 3 colony skills
│   └── Top 3 domain skills
└── References section (top 2 matches)
    ├── Reference 1 (full render)
    └── Reference 2 (full render)
```

Skills shape *how* the worker thinks. References shape *what* the worker
produces. Together, they provide both behavioral and structural guidance.
