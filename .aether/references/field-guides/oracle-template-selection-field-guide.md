---
schema_version: "1.0"
id: oracle-template-selection-field-guide
kind: field-guide
category: field-guides
title: Oracle Template Selection Field Guide
description: "How Oracle selects templates for different research tasks and when each template applies."
output_types: [oracle-plan, research-brief, tech-eval, architecture-review, bug-investigation]
agent_roles: [oracle, architect, queen, scout]
task_types: [oracle, research, template, evaluation, investigation]
task_keywords: [oracle, template, ralf, research, tech-eval, architecture, bug, prd, custom, loop, selection, output-type]
workflow_triggers: [oracle, plan]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---

# Oracle Template Selection Field Guide

This guide explains the Oracle's template system, when to use each template,
and how the `--template` flag affects research output structure and reference
matching.

## For Beginners

The Oracle is Aether's deep-research agent. It runs a structured investigation
called a RALF loop (Research-Analyze-Learn-Formulate). The template you choose
tells the Oracle what shape the final output should take. Think of templates as
different report formats: a technology evaluation looks very different from a
bug investigation, even though both involve research.

## Available Templates

### 1. tech-eval

**Purpose:** Evaluate a technology, library, or tool for adoption.

**Use when:**
- Deciding whether to adopt a new dependency
- Comparing multiple libraries for a task
- Assessing a framework's fitness for the project
- Evaluating a toolchain change

**Output structure:**
- Executive summary (adopt / reject / investigate further)
- Criteria matrix (performance, maturity, ecosystem, license, compatibility)
- Risk assessment with mitigation strategies
- Recommendation with confidence level
- Implementation notes if adopting

**Example invocation:**
```bash
/ant-oracle --template tech-eval "Evaluate using SQLite vs PostgreSQL for the state store"
```

### 2. architecture-review

**Purpose:** Analyze or propose architectural patterns and decisions.

**Use when:**
- Designing a new subsystem
- Evaluating an existing architecture for scalability
- Planning a major refactoring
- Assessing separation of concerns

**Output structure:**
- Current state analysis (what exists now)
- Proposed changes with diagrams or descriptions
- Trade-off analysis (performance, complexity, maintainability)
- Migration path if changing existing architecture
- Affected components and risk areas

**Example invocation:**
```bash
/ant-oracle --template architecture-review "Review the pheromone system for cross-colony scalability"
```

### 3. bug-investigation

**Purpose:** Deep investigation of a specific bug or failure.

**Use when:**
- A bug is intermittent or hard to reproduce
- Multiple failures share a suspected root cause
- Performance degradation needs root-cause analysis
- A test is flaky and the cause is unclear

**Output structure:**
- Symptom description (what the user sees)
- Reproduction steps (confirmed or suspected)
- Root cause analysis with evidence chain
- Timeline of contributing events
- Recommended fix with confidence level
- Prevention measures to avoid recurrence

**Example invocation:**
```bash
/ant-oracle --template bug-investigation "Workers occasionally receive stale handoff data"
```

### 4. best-practices (also: research-brief)

**Purpose:** Research best practices for a domain, language, or pattern.

**Use when:**
- Establishing conventions for a new codebase area
- Researching idiomatic patterns for a language or framework
- Comparing industry approaches to a problem
- Preparing guidance for the colony

**Output structure:**
- Summary of findings
- Recommended practices with rationale
- Anti-patterns to avoid
- Applicability assessment for this project
- References and sources
- Actionable recommendations ranked by impact

**Example invocation:**
```bash
/ant-oracle --template best-practices "Go error handling patterns for CLI applications"
```

### 5. custom / PRD

**Purpose:** Free-form research or product requirements investigation.

**Use when:**
- The research does not fit other templates
- Exploring a new domain or market
- Building a product requirements document
- User research or competitive analysis

**Output structure:**
- Flexible -- the Oracle adapts to the topic
- Typically includes: findings, analysis, recommendations
- May incorporate elements of other templates as needed

**Example invocation:**
```bash
/ant-oracle --template custom "Research how other agent orchestration systems handle fault tolerance"
```

## Template and Reference Matching

The `--template` flag does more than set the output structure. It also acts as
an `--output-type` hint for the reference matching system.

When a template is specified, `reference-match` receives the template name as
the `output-type` parameter. This means:

- `--template tech-eval` matches references with `output_types` including
  `tech-eval` or general evaluation types
- `--template architecture-review` matches architecture-specific references
- Custom templates fall back to general research references

The top two matched references are injected into the Oracle's prompt as a
`## References` section, providing structural guidance alongside the template.

## RALF Loop and Templates

The Oracle executes a RALF (Research-Analyze-Learn-Formulate) loop regardless
of which template is selected. The template affects the Formulate stage:

1. **Research** -- Gather information (same for all templates)
2. **Analyze** -- Process findings (same for all templates)
3. **Learn** -- Extract patterns and insights (same for all templates)
4. **Formulate** -- Structure output (template-specific)

The first three stages produce internal working notes. The Formulate stage
produces the deliverable that follows the template structure.

## Template Selection Guide

| If the user wants to... | Use this template |
|------------------------|-------------------|
| Choose between technologies | `tech-eval` |
| Design or review a system | `architecture-review` |
| Fix a mysterious bug | `bug-investigation` |
| Learn best practices | `best-practices` |
| Explore a new topic | `custom` |
| Write requirements | `custom` (or `prd` alias) |
| Compare approaches | `tech-eval` or `architecture-review` |

When in doubt, `tech-eval` covers most evaluative tasks and
`architecture-review` covers most design tasks. The Oracle can also suggest
a template if the user's question implies a specific output shape.
