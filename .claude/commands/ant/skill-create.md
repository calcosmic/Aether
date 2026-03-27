---
name: ant:skill-create
description: "Create a custom domain skill using Oracle mini-research and a guided wizard"
---

You are the **Builder**. Create a custom domain skill by researching best practices and guiding the user through a wizard.

The user's input is: `$ARGUMENTS`

## Instructions

### Step 0: Parse Arguments

Extract the topic from `$ARGUMENTS`.

If `$ARGUMENTS` is empty, output:

```
Skill Creator — Usage

  /ant:skill-create "topic"

  Examples:
    /ant:skill-create "kubernetes"
    /ant:skill-create "fastapi"
    /ant:skill-create "terraform"

  This will research the topic, ask a few questions, and generate
  a custom SKILL.md in ~/.aether/skills/domain/
```

Stop here.

Set `TOPIC` to the value of `$ARGUMENTS` (strip surrounding quotes).
Derive `SKILL_NAME` by lowercasing `TOPIC`, replacing spaces with hyphens, and removing any non-alphanumeric/hyphen characters.

Proceed to Step 1.

---

### Step 1: Oracle Mini-Research (5 iterations)

Run a focused mini-research session on the topic. This is a self-contained research cycle -- it does not require an existing Oracle session.

**1a. Web search for best practices**

Use WebSearch to search for:
- `"$TOPIC best practices 2025"`
- `"$TOPIC common mistakes to avoid"`
- `"$TOPIC project structure conventions"`

Collect the top findings from each search (titles, key points, URLs).

**1b. Codebase scan for existing usage**

Use Grep and Glob to scan the current repository for patterns related to the topic:
- Search for imports, config files, or dependencies matching `TOPIC`
- Note any existing conventions or patterns in the codebase

**1c. Compile findings into a research summary**

After 5 iterations of searching and reading (web search results, codebase patterns), compile a structured summary:

```
Research Summary: $TOPIC

Key Best Practices:
  1. ...
  2. ...
  3. ...

Common Pitfalls:
  1. ...
  2. ...

Project Structure Conventions:
  - ...

Codebase Observations:
  - (what was found in the current repo, if anything)

Sources:
  - ...
```

Display this summary to the user before proceeding.

---

### Step 2: Wizard Questions

Use AskUserQuestion to guide the user through skill customization. Ask each question one at a time.

**Question 1: Focus Area**

Based on the Oracle research findings, identify 2-4 distinct aspects of the topic. Present them as options:

```
What aspect of $TOPIC should this skill focus on?
```

Options (dynamically generated from research, for example):
1. **Core patterns and architecture** -- Fundamental design patterns and project structure
2. **Performance and optimization** -- Speed, caching, resource management
3. **Testing and quality** -- Test strategies, coverage, CI/CD integration
4. **Comprehensive guide** -- Cover all major aspects in a single skill

(Adapt the options based on what the research actually found. Always include a "comprehensive" option as the last choice.)

**Question 2: Experience Level**

```
What experience level should this skill target?
```

Options:
1. **Beginner** -- Explain fundamentals, include examples, avoid advanced patterns
2. **Intermediate** -- Assume basics, focus on best practices and common patterns
3. **Advanced** -- Expert-level patterns, performance tuning, edge cases

**Question 3: Custom Rules or Constraints**

```
Any specific rules or constraints to include? (e.g., "always use TypeScript", "prefer composition over inheritance", "no classes")
```

Options:
1. **No specific rules** -- Use standard best practices from the research
2. **Yes, I have rules** -- I want to add custom constraints

If the user selects option 2, ask a follow-up AskUserQuestion with free text:

```
Enter your rules or constraints (free text):
```

Capture the user's custom rules for inclusion in the skill.

---

### Step 3: Generate SKILL.md

Based on the research findings and wizard answers, generate a complete skill file.

**3a. Determine frontmatter values**

- `name`: Use `SKILL_NAME` (the sanitized, lowercased topic)
- `description`: Generate a concise description like "Use when the project uses $TOPIC" or "Best practices for $TOPIC development"
- `type: domain`
- `domains`: Infer 2-4 relevant domain tags from the research (e.g., `[frontend, components]` for React, `[backend, api]` for FastAPI)
- `agent_roles: [builder]`
- `detect_files`: Infer file patterns that indicate this technology is in use (e.g., `["*.py", "requirements.txt"]` for Python frameworks, `["Dockerfile", "docker-compose.yml"]` for Docker)
- `detect_packages`: Infer package names that indicate this technology (e.g., `["fastapi"]`, `["terraform"]`)
- `priority: normal`
- `version: "1.0"`

**3b. Generate the body content**

Write a comprehensive best-practices guide structured as follows:

```markdown
# $TOPIC Best Practices

## [Aspect heading based on focus area]

[2-3 paragraphs of actionable guidance drawn from Oracle research]
[Adapt depth and complexity to the selected experience level]

## [Next aspect heading]

[More guidance...]

## Common Pitfalls

[List of things to avoid, drawn from research]

## [Additional sections as appropriate for the topic]

[Custom rules or constraints from wizard Question 3, if any, integrated naturally into the guide]
```

Guidelines for body content:
- Write in the same style as existing skills (direct, actionable, no fluff)
- Reference the React SKILL.md at `.aether/skills/domain/react/SKILL.md` for tone and structure
- Beginner level: include more explanation and examples
- Intermediate level: focus on patterns and best practices
- Advanced level: include edge cases, performance tuning, and expert techniques
- If the user provided custom rules, weave them into the relevant sections rather than listing them separately
- Keep total body length between 30-80 lines (matching existing skills)

**3c. Assemble the full SKILL.md content**

Combine the frontmatter and body into a single file:

```markdown
---
name: {SKILL_NAME}
description: {description}
type: domain
domains: [{domain1}, {domain2}]
agent_roles: [builder]
detect_files: ["{pattern1}", "{pattern2}"]
detect_packages: ["{package1}"]
priority: normal
version: "1.0"
---

{body content}
```

---

### Step 4: Write and Verify

**4a. Create the skill directory and write the file**

Run using the Bash tool with description "Creating skill directory...":

```bash
mkdir -p ~/.aether/skills/domain/{SKILL_NAME}
```

Use the Write tool to create `~/.aether/skills/domain/{SKILL_NAME}/SKILL.md` with the assembled content.

**4b. Verify frontmatter parses correctly**

Run using the Bash tool with description "Verifying skill frontmatter...":

```bash
bash .aether/aether-utils.sh skill-parse-frontmatter ~/.aether/skills/domain/{SKILL_NAME}/SKILL.md
```

Check the output. If the result contains `"ok": true` (or the parsed JSON shows the correct name and type), the skill is valid. If parsing fails, fix the frontmatter and retry once.

**4c. Rebuild skill cache**

Run using the Bash tool with description "Rebuilding skill cache...":

```bash
bash .aether/aether-utils.sh skill-cache-rebuild
```

**4d. Show the result**

Display the generated skill to the user:

```
Skill Created: {SKILL_NAME}

  Location:  ~/.aether/skills/domain/{SKILL_NAME}/SKILL.md
  Type:      domain
  Domains:   {domains list}
  Detects:   {detect_files and detect_packages}
  Level:     {experience level}
```

Then show the full content of the SKILL.md file.

**4e. Offer adjustments**

Use AskUserQuestion to ask:

```
Happy with this skill, or want to adjust anything?
```

Options:
1. **Looks good** -- Keep it as-is
2. **Adjust content** -- I want to change some of the guidance
3. **Regenerate** -- Start over with different options

If the user selects option 2, ask what they want to change, make the edits, re-write the file, and re-run `skill-parse-frontmatter` and `skill-cache-rebuild`.

If the user selects option 3, go back to Step 2.

If the user selects option 1, output:

```
Skill "{SKILL_NAME}" is ready. It will automatically activate in projects
that match its detection patterns (files: {detect_files}, packages: {detect_packages}).

You can also view all installed skills with: /ant:skill-list
```

Stop here.
