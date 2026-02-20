# Template System Improvement — Synthesis Overview

**Date:** 2026-02-18
**Status:** Awaiting review
**Companion documents:**
- `2026-02-18-template-architecture-plan.md` — How templates should be structured (format, placeholders, directory layout, before/after examples)
- `2026-02-18-template-schema-system-design.md` — How templates and schemas work together (loading patterns, validation, distribution, migration)

---

## What This Is About

Aether commands embed JSON structures and markdown formats inline — 30-line JSON blocks buried inside 300-line instruction files. When an LLM agent runs `/ant:init`, it reads the whole file and tries to reproduce the JSON exactly from memory. It often doesn't. Fields get dropped, types get wrong, structures drift between runs.

The GSD system (which lives alongside Aether) solved this by creating 30+ template files. Commands say "read this template, fill in the blanks" instead of "reconstruct this structure from the instructions." Two specialist analyses were run in parallel to design Aether's template system. This document synthesizes their findings.

---

## The Big Picture (Plain English)

Right now, Aether has **1 template** (QUEEN.md.template) and **26 structures embedded inline** across 8 command files. Every time an agent runs a command, it's reading the whole command file, finding the structure somewhere in the middle, and trying to reproduce it. That's like giving someone a recipe book and asking them to bake from memory instead of reading the recipe while they cook.

**The fix:** Extract every embedded structure into its own template file. Commands say "read the template at this path, fill in these values, write the result." The agent reads the exact structure at the moment it needs it.

**Three themes emerged:**

1. **Templates are for agents, not for scripts.** The main consumer is an LLM. Templates should include self-describing instructions (`_instructions` field, `_comment_*` annotations) so the agent knows exactly what to do when it reads the file. Shell scripts get simpler templates without annotations.

2. **Schemas are for validators, not for agents.** JSON Schema files go in a separate `schemas/json/` directory. They tell the `validate-state` function what to check. Templates and schemas serve different audiences and should stay separate.

3. **Migration must be additive.** Templates get created and distributed first, before any command file is changed. Once templates are proven in a real colony lifecycle, commands are updated to reference them. No big-bang cutover.

---

## What Both Analyses Agree On (Highest Confidence)

| Finding | LLM Architect | Agent Organizer |
|---------|:---:|:---:|
| colony-state.json is the #1 priority template | Yes | Yes |
| 26 templates identifiable from current commands | Yes | Yes (different count: ~18 core + extensions) |
| Use `__DOUBLE_UNDERSCORE__` for JSON placeholders | Yes (via `${VAR}`) | Yes (`__VAR__`) |
| Use `{{DOUBLE_BRACE}}` for markdown placeholders | Yes | Yes |
| Templates live in `.aether/templates/` | Yes | Yes |
| Subdirectories: `json/`, `md/`, `prompts/`, `results/`, `display/` | Yes | Yes (slightly different: `json/`, `md/`) |
| Templates sync via existing pipeline (sync-to-runtime.sh) | Yes | Yes |
| Add templates to SYSTEM_FILES array | Yes | Yes |
| Additive migration (create first, reference later) | Yes | Yes |
| JSON Schema validation in `.aether/schemas/json/` | Deferred | Yes (full design) |
| Template Registry file for version tracking | -- | Yes |
| Worker result templates have highest build reliability impact | Yes | Yes |

---

## Where They Differ (Needs a Decision)

### 1. Placeholder syntax for JSON templates

**LLM Architect** uses `${VARIABLE}` with `${VARIABLE:default}` syntax — looks like shell variables, includes defaults inline.

**Agent Organizer** uses `__VARIABLE__` — visually louder, catches substitution failures (invalid JSON if left in), no default syntax.

**Recommendation:** Go with `__VARIABLE__` for JSON templates. The failure-detection property (jq will reject `"__GOAL__"` as a value if the agent forgets to substitute) is a real safety net. Use `{{VARIABLE}}` for markdown as both agree.

### 2. Template self-documentation approach

**LLM Architect** uses XML-style sections (`<template>`, `<fields>`, `<placeholders>`, `<example>`, `<validation>`) wrapping the actual template content.

**Agent Organizer** uses underscore-prefixed keys inside the JSON itself (`_template`, `_version`, `_instructions`, `_comment_*`) that get stripped before writing.

**Recommendation:** Both approaches work. The Agent Organizer's approach is simpler for JSON templates (the annotation is in the JSON itself, no wrapper format). The LLM Architect's approach is better for markdown templates (XML sections are natural for wrapping markdown). Consider using both: `_underscore` keys for JSON templates, `<xml>` sections for markdown templates.

### 3. Schema system scope

**LLM Architect** defers JSON Schema to "future enhancement" — the template's `<validation>` section is sufficient for now.

**Agent Organizer** designs the full schema system immediately — 7 JSON Schema files in `.aether/schemas/json/`, wired to `validate-state`.

**Recommendation:** Create the schemas in Phase 1 alongside templates. They're small files (the colony-state schema example is ~40 lines), cost nothing to maintain, and the `validate-state` function already exists. No reason to defer.

### 4. Template count and granularity

**LLM Architect** identifies 26 specific templates with detailed per-template descriptions, including separate prompt templates for each worker type and separate result schemas per worker.

**Agent Organizer** identifies ~18 core templates with a tiered priority system, grouping worker results into a single `worker-result.template.json` and treating prompt templates as a later phase.

**Recommendation:** Start with the Agent Organizer's smaller set (~18 templates). The LLM Architect's granular list (separate scout-broad vs scout-gap, separate builder-result vs watcher-result) is good design but can be split later if needed. Ship fewer, more important templates first.

---

## Unified Priority Roadmap

### Wave 1: Create Templates (no command changes)

Create template files and distribute them. Zero risk — nothing reads them yet.

1. **colony-state.json.template** — extracted from init.md lines 184-213
2. **constraints.json.template** — extracted from init.md lines 219-225
3. **session.json.template** — extracted from aether-utils.sh session-init function
4. **flags.json.template** — extracted from aether-utils.sh flag-add function
5. **manifest.json.template** — extracted from chamber-create function
6. **worker-result.json.template** — extracted from build.md worker output spec
7. **crowned-anthill.md.template** — extracted from seal.md heredoc
8. **handoff.md.template** — extracted from entomb.md/build.md heredocs
9. **REGISTRY.json** — version registry for all templates
10. Add all files to `bin/sync-to-runtime.sh` SYSTEM_FILES array
11. Create JSON schemas for colony-state, constraints, flags, manifest

**Impact:** Templates exist in all repos. Schemas ready for validation. Distribution proven.

### Wave 2: Wire Commands to Templates (critical path)

Update command files to read templates instead of embedding structures.

1. **init.md** — replace inline colony-state JSON with "read template, fill, write"
2. **init.md** — replace inline constraints JSON with template reference
3. **seal.md** — replace heredoc with crowned-anthill.md.template reference
4. **entomb.md** — replace heredoc with handoff.md.template reference
5. **build.md** — replace inline worker result spec with template reference
6. Wire `validate-state` to check against JSON schemas

**Impact:** Commands are shorter and clearer. Structures are defined in one place.

### Wave 3: Worker Prompt Templates

Extract worker prompts from build.md and plan.md into template files.

1. **builder-prompt.md.template** — static prompt sections from build.md
2. **watcher-prompt.md.template** — static verification sections from build.md
3. **chaos-prompt.md.template** — static investigation sections from build.md
4. **scout-prompt.md.template** — research prompt from plan.md
5. **route-setter-prompt.md.template** — planning prompt from plan.md
6. Update build.md and plan.md to reference prompt templates

**Impact:** Prompt consistency across builds. Single place to update worker instructions.

### Wave 4: Remaining Templates and Cleanup

1. **completion-report.md.template** — for init's inheritance parsing
2. **verification-report.md.template** — for watcher output
3. **watch-status.txt.template** — for tmux display
4. Remove all remaining inline structures from command files
5. Update agent definitions to reference output templates
6. Deprecate `{SINGLE_BRACE}` syntax in QUEEN.md.template (migrate to `{{}}`)

**Impact:** Full template coverage. No inline structures remain in any command file.

---

## Key Decisions Needed From You

Before starting implementation, a few choices:

1. **Placeholder syntax** — `__DOUBLE_UNDERSCORE__` for JSON, `{{DOUBLE_BRACE}}` for markdown? Or a different convention?

2. **Template annotation style** — Underscore-prefixed keys inside JSON (`_instructions`, `_comment_*`) that agents strip before writing? Or a wrapper format outside the JSON?

3. **Schema scope** — Create JSON Schema files immediately alongside templates? Or defer validation to a later phase?

4. **Starting scope** — Start with the ~18 core templates (Agent Organizer's list)? Or go for all 26 at once (LLM Architect's list)?

5. **Implementation approach** — Do this as a dedicated colony build phase? Or gradually across sessions?

---

## File Inventory (What's in docs/plans/)

| File | Content | Lines |
|------|---------|-------|
| `2026-02-18-template-improvement-synthesis.md` | This overview | ~160 |
| `2026-02-18-template-architecture-plan.md` | Detailed template format, directory structure, before/after examples for init/seal/build | ~765 |
| `2026-02-18-template-schema-system-design.md` | Template inventory, schema system, loading patterns, variable substitution, versioning, distribution, migration | ~450 |

Read the synthesis first. Dive into the detailed plans for specifics on any area.

---

*Ready for your review. Let me know which direction feels right and I'll start building.*
