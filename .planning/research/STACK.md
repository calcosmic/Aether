# Stack Research — Claude Code Subagents

**Domain:** Claude Code subagent definition format for Aether ant colony system
**Researched:** 2026-02-20
**Confidence:** HIGH — format verified against official Claude Code documentation and 4 working repos with production subagents

---

## Context: What This Milestone Is Adding

This is a subsequent milestone research. The existing stack (Bash, jq, Node.js CLI, OpenCode agents in `.opencode/agents/`, GSD agents in `.claude/agents/`) is validated and not re-researched. This file covers ONLY the format requirements for creating Claude Code subagents in `.claude/agents/`.

**What we're adding:** Ant colony worker definitions as Claude Code subagents invocable via the `Task` tool — mapping the 22 existing OpenCode agent roles (queen, builder, watcher, scout, etc.) to Claude Code's `.claude/agents/` format.

---

## Recommended Stack

### Agent File Format

**Decision: YAML frontmatter + markdown/XML body, stored in `.claude/agents/`.**

This is the established Claude Code subagent format, confirmed in official documentation and 4 working repos.

#### Required Frontmatter Fields

| Field | Required | Valid Values | Notes |
|-------|----------|--------------|-------|
| `name` | YES | lowercase letters and hyphens | Must be unique. Claude Code uses this as the agent identifier. E.g., `ant-builder`, `ant-queen` |
| `description` | YES | any string | Claude reads this to decide when to auto-delegate. Write "Use proactively when..." for auto-routing. |

#### Optional Frontmatter Fields (All Confirmed)

| Field | Valid Values | Behavior When Omitted | Notes |
|-------|--------------|----------------------|-------|
| `tools` | Comma-separated tool names (see Tool Names section) | Inherits ALL tools from parent conversation | Allowlist — specifying restricts to only these |
| `disallowedTools` | Comma-separated tool names | None denied | Denylist — removed from inherited/specified list |
| `model` | `sonnet`, `opus`, `haiku`, `inherit` | `inherit` (same as main conversation) | Controls which Anthropic model the subagent uses |
| `color` | UI color name | None | Visual identifier in Claude Code UI. Options observed: `yellow`, `green`, `purple`, `cyan`, `red`, `blue` |
| `permissionMode` | `default`, `acceptEdits`, `dontAsk`, `bypassPermissions`, `plan` | `default` | Controls permission prompt behavior |
| `maxTurns` | integer | Unlimited | Stops agent after N agentic turns |
| `skills` | list of skill names | None | Injects skill content at startup |
| `mcpServers` | list of server names or inline configs | None | MCP server access |
| `hooks` | lifecycle hook definitions | None | PreToolUse, PostToolUse, Stop |
| `memory` | `user`, `project`, `local` | Disabled | Persistent memory across conversations |
| `background` | `true` / `false` | `false` | Run as background task |
| `isolation` | `worktree` | None | Run in temporary git worktree |

#### Tool Names (Claude Code Internal Tools)

Specify these exactly (case-sensitive) in `tools` or `disallowedTools`:

| Tool Name | What It Does |
|-----------|--------------|
| `Read` | Read files |
| `Write` | Write files |
| `Edit` | Edit files (patch-style) |
| `Bash` | Execute bash commands |
| `Grep` | Search file contents |
| `Glob` | Find files by pattern |
| `Task` | Spawn subagents (use `Task(agent-name)` to restrict which) |
| `WebSearch` | Web search |
| `WebFetch` | Fetch web content |
| `mcp__*` | MCP server tools (use server name prefix) |

**MCP tools:** Inherit automatically if not restricted. Specify MCP tools as `mcp__context7__*` to include all tools from a server.

#### Model Values Explained

| Value | Behavior | Use For |
|-------|----------|---------|
| `opus` | Claude Opus 4.6 (most capable, slower, higher cost) | Complex reasoning — queen, architects |
| `sonnet` | Claude Sonnet 4.6 (balanced) | Most worker tasks |
| `haiku` | Haiku (fast, cheap, lower capability) | Simple tasks — status checks, file discovery |
| `inherit` | Same model as parent conversation | Default — safe choice when unsure |

**Recommendation for ant colony:** Use `inherit` for most workers. This lets the operator's model choice propagate. Reserve explicit `haiku` for simple status/lookup tasks and explicit `opus` for the queen role if the queen needs it regardless of operator setting.

### Two Format Styles (Both Work — Choose Based on Role)

#### Style 1: YAML frontmatter + Plain Markdown Body

Used by `everything-claude-code` agents and simpler subagents. Best for straightforward, stateless reviewers and analyzers.

```markdown
---
name: ant-watcher
description: Monitors build output and test results. Use when checking build status or test failures.
tools: Read, Bash, Grep, Glob
model: inherit
color: green
---

You are an ant colony watcher. Your role is to monitor...

## When invoked:
1. Run test suite
2. Report failures only
...
```

#### Style 2: YAML frontmatter + XML-structured Body

Used by all existing GSD agents in `.claude/agents/` and CDS agents. Best for complex, multi-step agents with distinct phases, rules, and success criteria. XML tags create clear structure Claude navigates.

```markdown
---
name: ant-builder
description: Implements features and fixes. Spawned by queen to execute specific work units.
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
color: yellow
---

<role>
You are an ant colony builder. Execute work units assigned by the queen.
</role>

<constraints>
- Commit after each completed task
- Never touch .aether/data/ or .aether/dreams/
</constraints>

<workflow>
...steps...
</workflow>

<success_criteria>
- [ ] Work unit completed
- [ ] Committed with proper message
</success_criteria>
```

**Recommendation for Aether ant agents:** Use Style 2 (XML body). The existing 11 GSD agents all use this pattern, the system prompt reads as clearly structured XML, and the ant colony workers have multi-step workflows that benefit from explicit XML sections.

### Auto-Routing via Description Field

Claude reads the `description` field to decide when to auto-delegate. Key patterns:

**Trigger auto-delegation:**
- "Use proactively when..." — Claude delegates without being asked
- "Spawned by [command]..." — Claude knows this is invoked by specific slash commands
- "Use when [condition]..." — Conditional routing based on context

**Examples from working repos:**

```
# Proactive auto-routing:
description: Expert code review specialist. Proactively reviews code for quality,
  security, and maintainability. Use immediately after writing or modifying code.
  MUST BE USED for all code changes.

# Spawned by command (orchestrator pattern):
description: Executes GSD plans with atomic commits, deviation handling, checkpoint
  protocols, and state management. Spawned by execute-phase orchestrator or
  execute-plan command.

# Conditional:
description: Debugging specialist for errors, test failures, and unexpected behavior.
  Use proactively when encountering any issues.
```

**For ant colony agents:** Use "Spawned by [ant-command]..." pattern since colony workers are invoked by orchestrator commands, not auto-delegated by Claude.

### How Task Tool Invokes Subagents

The `Task` tool invokes a subagent by its `name` field:

```
Task tool usage:
- subagent_type: "ant-builder"   ← matches `name` field in .claude/agents/ant-builder.md
- prompt: "Build the login feature..."
```

**Resolution order when multiple agents share a name:**
1. CLI `--agents` flag (session-only)
2. `.claude/agents/` (project-level) ← Aether agents live here
3. `~/.claude/agents/` (user-level)
4. Plugin `agents/` directories

**Subagent limitations:**
- Subagents CANNOT spawn other subagents — the Task tool fails if called from within a subagent
- This is a hard platform constraint, not configurable
- Pattern: orchestrator in main conversation → spawns subagents → results return to main conversation
- For nested coordination, use Skills or chain subagents from main conversation

**Task tool in `tools` field:**
- `Task` — can spawn any subagent
- `Task(ant-builder, ant-watcher)` — can only spawn named subagents (allowlist)
- Omit `Task` entirely — agent cannot spawn any subagents

### File Locations

| Location | Scope | Use For |
|----------|-------|---------|
| `/Users/callumcowie/repos/Aether/.claude/agents/` | This project | Ant colony worker definitions — distributed to users via npm |
| `~/.claude/agents/` | All projects on machine | Personal agents not for distribution |

**Distribution note:** `.claude/agents/` files are checked into the Aether git repo and distributed to users as part of the npm package. The hub sync (`aether update`) makes them available in target repos. This is the correct location for colony worker definitions.

### File Naming Convention

All existing GSD agents use `gsd-` prefix. For ant colony agents, use `ant-` prefix:

```
.claude/agents/
  ant-queen.md        # Colony coordinator
  ant-builder.md      # Feature implementation
  ant-watcher.md      # Test monitoring
  ant-scout.md        # Discovery/research
  ant-keeper.md       # State management
  ant-chronicler.md   # Logging/handoffs
  ...
```

Alternatively, keep the existing OpenCode naming without prefix since these are colony roles, not GSD workflow roles. The `name` field in frontmatter is what Claude Code uses for routing — the filename is just for organization.

---

## Supporting Libraries

No new dependencies required. All tooling for creating, validating, and distributing agent files exists in the current stack.

| What's Needed | How to Do It | Existing Tool |
|---------------|--------------|---------------|
| Agent file creation | Write markdown with YAML frontmatter | File system + editor |
| XML body validation | Frontmatter strip + xmllint | xmllint (system) |
| Agent lint | Existing `npm run lint:agents` approach | AVA + bash |
| Distribution | Check into `.claude/agents/`, npm package | Existing npm pipeline |
| Hub sync | `aether update` copies to target repos | Existing hub system |

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `.claude/agents/` format for Claude Code | Continue OpenCode-only (`.opencode/agents/`) | Users who run Claude Code (not OpenCode) can't use colony workers. Both formats are needed for dual-IDE support |
| XML-structured body (Style 2) | Plain markdown body (Style 1) | Ant colony agents have complex workflows with distinct phases. XML provides navigable structure for multi-step agents. Consistency with existing 11 GSD agents |
| `inherit` model default | Explicit `sonnet` for all workers | `inherit` propagates the operator's model choice. Explicit model locks users into a specific tier |
| `ant-` name prefix | Caste emoji prefix (e.g., `hammer-builder`) | YAML frontmatter `name` must be lowercase letters and hyphens only. Emoji not valid in `name` field |
| `color` field per agent role | No color | Color helps users identify which agent is running in Claude Code UI — valuable in a colony with many agents |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Emoji in `name` field | Must be lowercase letters and hyphens — emoji breaks YAML parsing | Emoji in description or body text only |
| `bypassPermissions` in `permissionMode` | Skips all permission checks — security risk for colony agents that write files and run bash | `default` (standard) or omit field entirely |
| Model references in XML body | Aspirational dead weight — does not affect routing. Confirmed in prior research (v1.3 STACK.md) | Model routing via `model` frontmatter field only |
| Spawning subagents from within subagents | Platform does not support it — Task tool fails when called from subagent context | Orchestrate from main conversation or slash command |
| Using `~/.claude/agents/` for distributed agents | User-level agents don't distribute via npm — only the project lives at `.claude/agents/` | `.claude/agents/` for all colony workers |

---

## Stack Patterns by Role

**For colony orchestrator (queen) invoked by slash command:**
```yaml
name: ant-queen
description: Ant colony queen. Coordinates worker spawning and colony goal. Invoked by /ant:build and /ant:plan.
tools: Read, Write, Bash, Task
model: inherit
color: purple
```

**For worker agents (builder, watcher, etc.) spawned by queen:**
```yaml
name: ant-builder
description: Implements features. Spawned by ant-queen to execute work units.
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
color: yellow
```
Note: Workers OMIT `Task` from tools — they cannot spawn further agents.

**For read-only analysis agents (scout, archaeologist):**
```yaml
name: ant-scout
description: Explores codebase and reports findings. Spawned by ant-queen for discovery phases.
tools: Read, Bash, Grep, Glob
model: haiku
color: cyan
```
Haiku is appropriate here — fast, cheap, read-only work.

---

## Version Compatibility

| Component | Constraint | Notes |
|-----------|------------|-------|
| Claude Code subagents | Current (2026) | Verified against official docs at code.claude.com/docs/en/sub-agents. Format includes fields: `name`, `description`, `tools`, `disallowedTools`, `model`, `permissionMode`, `maxTurns`, `skills`, `mcpServers`, `hooks`, `memory`, `background`, `isolation` |
| Model aliases | `sonnet`, `opus`, `haiku`, `inherit` | These are aliases, not model IDs. Resolved to current models at runtime |
| YAML frontmatter | Standard YAML | Tools field accepts both `tool1, tool2` (comma-separated string) and `["tool1", "tool2"]` (array). Both observed in working repos |
| XML body | Well-formed XML required | Validated with existing xmllint approach. Body must be well-formed XML if using XML style |

---

## Sources

- Official Claude Code documentation at `https://code.claude.com/docs/en/sub-agents` — complete frontmatter field reference, model options, tool names, auto-routing mechanics. HIGH confidence (official docs, verified 2026-02-20)
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-executor.md` — YAML + XML body format, `tools`, `color` field usage. HIGH confidence (working in production)
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-project-researcher.md` — `mcp__context7__*` tool format, `color: cyan`. HIGH confidence (working in production)
- `/Users/callumcowie/repos/everything-claude-code/agents/architect.md` — YAML array syntax for tools `["Read", "Grep", "Glob"]`, `model: opus`. HIGH confidence (working examples)
- `/Users/callumcowie/repos/everything-claude-code/agents/code-reviewer.md` — YAML array syntax, `model: opus`. HIGH confidence (working examples)
- `/Users/callumcowie/repos/superpowers/agents/code-reviewer.md` — `model: inherit` usage, multi-line description format. HIGH confidence (working examples)
- `/Users/callumcowie/repos/cosmic-dev-system/agents/cds-executor.md` — YAML frontmatter comma-separated tools (no array brackets), XML body identical to GSD pattern. HIGH confidence (working in production)

---

*Stack research for: Claude Code subagents for Aether ant colony*
*Researched: 2026-02-20*
