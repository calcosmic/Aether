<!-- Generated from .aether/commands/skill-create.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:skill-create
description: "Create a custom domain skill using Oracle mini-research and a guided wizard"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Builder**. Create a custom domain skill by researching best practices and guiding the user through a wizard.

## Instructions

### Step 0: Parse Arguments

If `$normalized_args` is empty:
```
Usage: /ant:skill-create "<topic>"

Creates a custom domain skill using Oracle mini-research.

Examples:
  /ant:skill-create "tailwind"
  /ant:skill-create "react native"
  /ant:skill-create "kubernetes"
```
Stop here.

### Step 1: Oracle Mini-Research

Launch a focused research session on the topic (5 iterations max):
1. Web search for best practices, common patterns, and gotchas
2. Codebase scan for existing usage of the technology
3. Compile findings into structured notes

### Step 2: Wizard Questions

Ask the user:
1. What aspect to focus on? (provide 2-4 options based on findings)
2. Experience level? (beginner / intermediate / advanced)
3. Any specific rules or constraints? (free text)

### Step 3: Generate SKILL.md

Create the skill file with proper frontmatter and body:
- name: derived from topic
- description: "Use when working with {topic}"
- type: domain
- domains: inferred from research
- agent_roles: [builder]
- detect_files / detect_packages: inferred from research
- Body: practical best-practices guide from research + wizard answers

### Step 4: Write and Verify

1. Write to `~/.aether/skills/domain/{name}/SKILL.md`
2. Run: `aether skill-parse-frontmatter ~/.aether/skills/domain/{name}/SKILL.md`
3. Run: `aether skill-cache-rebuild`
4. Show the generated skill to the user
5. Ask if they want to adjust anything
