# Aether Command Source Definitions

This directory contains canonical command definitions in YAML format that can be used to generate
commands for both Claude Code and OpenCode.

## Directory Structure

```
src/commands/
├── README.md           # This file
├── init.yaml           # /ant:init command definition
├── plan.yaml           # /ant:plan command definition
├── build.yaml          # /ant:build command definition
├── ...                 # Other command definitions
└── _meta/
    ├── tool-mapping.yaml   # Tool name translations between platforms
    └── template.yaml       # Template for new commands
```

## Command YAML Format

Each command is defined in YAML with the following structure:

```yaml
name: ant:init
description: Initialize Aether colony - Queen sets intention, colony mobilizes

# Tool mappings (Claude Code -> OpenCode)
tools:
  Read: read
  Write: write
  Bash: bash
  Glob: glob
  Grep: grep
  Task: task
  TaskOutput: (inline)  # OpenCode returns task results inline
  AskUserQuestion: context.ask

# Command content (shared logic)
content: |
  You are the **Queen Ant Colony**. Initialize the colony with the Queen's intention.

  ## Instructions
  ...

# Platform-specific overrides
overrides:
  claude-code:
    task_syntax: |
      Use Task tool with subagent_type="general" and run_in_background: true
  opencode:
    task_syntax: |
      Use task tool with subagent_type: "general"
```

## Generating Commands

Run the generator script to produce commands for both platforms:

```bash
./bin/generate-commands.sh
```

This will:
1. Read each YAML definition
2. Apply tool name translations
3. Apply platform-specific overrides
4. Write to `.claude/commands/ant/` and `.opencode/commands/ant/`

## Tool Name Mapping

| Claude Code | OpenCode | Notes |
|-------------|----------|-------|
| `Read` | `read` | Same semantics |
| `Write` | `write` | Same semantics |
| `Bash` | `bash` | Same semantics |
| `Glob` | `glob` | Same semantics |
| `Grep` | `grep` | Same semantics |
| `Task` | `task` | Different syntax (see below) |
| `TaskOutput` | (implicit) | OpenCode returns inline |
| `AskUserQuestion` | `context.ask` | Different pattern |

## Task Tool Differences

**Claude Code:**
```markdown
Use the Task tool with:
- subagent_type: "general-purpose"
- run_in_background: true
- prompt: "..."

Then use TaskOutput to collect results.
```

**OpenCode:**
```markdown
Use the task tool with:
- subagent_type: "general"
- prompt: "..."

Results return inline (no separate TaskOutput needed).
```

## Adding New Commands

1. Create a new YAML file in this directory
2. Follow the format in `_meta/template.yaml`
3. Run `./bin/generate-commands.sh`
4. Test in both Claude Code and OpenCode
