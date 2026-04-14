# Codex CLI Format Specification

> Research findings from the OpenAI Codex CLI source code (openai/codex, Rust).
> Last updated: 2026-04-14

---

## AGENTS.md Schema

### Overview

AGENTS.md is Codex CLI's mechanism for injecting project-level instructions into the agent's
context. It is conceptually similar to CLAUDE.md for Claude Code.

### Format

AGENTS.md is **plain Markdown**. There is no frontmatter, no YAML header, no special metadata
format. The entire file content is treated as raw instructions text and injected verbatim into
the agent's system context.

Key characteristics:
- **No frontmatter** -- the file is read as-is, from first byte to last
- **No special sections** -- any markdown headings, lists, or text are valid
- **No schema validation** -- the content is not parsed for structure; it is concatenated and injected
- **UTF-8 encoded** -- `String::from_utf8_lossy` is used when reading

### File Discovery Rules

Codex discovers AGENTS.md files by walking the directory tree from the project root down to
the current working directory (cwd).

#### Search Algorithm (from `project_doc.rs`)

1. **Find project root:** Walk upwards from cwd looking for root markers (default: `.git`).
   The first directory containing a marker becomes the project root. If no marker is found,
   only the cwd is considered.

2. **Collect files from root to cwd:** Starting at the project root, walk down to cwd
   (inclusive). At each directory level, check for AGENTS.md files. Files are collected in
   order from root to cwd.

3. **Filename priority (per directory):**
   - `AGENTS.override.md` -- highest priority local override (checked first)
   - `AGENTS.md` -- standard filename (checked second)
   - `project_doc_fallback_filenames` -- configured fallbacks in `config.toml` (checked third)

   **Only one file per directory** is selected. If `AGENTS.override.md` exists, `AGENTS.md`
   is skipped in that directory.

4. **Concatenation:** All discovered files are joined with `\n\n` (double newline) separator,
   in order from project root to cwd.

#### Root Markers

Configured via `project_root_markers` in `config.toml`:
- Default: `[".git"]`
- Can be empty (disables parent traversal; only cwd is checked)
- Custom markers supported: e.g., `[".codex-root"]`
- `.git` can be either a file or a directory

#### Byte Budget

Controlled by `project_doc_max_bytes` in `config.toml`:
- Default: non-zero (exact default from source not confirmed; tests use 4096)
- When `0`: discovery is disabled entirely, no AGENTS.md files are read
- When files exceed the budget: files are truncated at the byte boundary
- Budget is shared across all discovered files (root to cwd)

### Configuration Fields (in `config.toml`)

| Field | Type | Description |
|-------|------|-------------|
| `project_doc_max_bytes` | `uint` | Maximum total bytes from AGENTS.md files. 0 disables. |
| `project_doc_fallback_filenames` | `string[]` | Ordered list of fallback filenames when AGENTS.md is missing. |
| `project_root_markers` | `string[]` | Files/dirs that mark the project root. Default: `[".git"]`. |
| `instructions` | `string` | System instructions (separate from AGENTS.md; concatenated with separator). |
| `developer_instructions` | `string` | Developer instructions injected as a `developer` role message. |

### How Instructions Are Assembled

The `get_user_instructions` function assembles the final instruction string:

```
[instructions from config.toml]
  + "\n\n--- project-doc ---\n\n"  (separator, only if both parts exist)
  + [concatenated AGENTS.md files from root to cwd]
  + [JS REPL instructions, if feature enabled]
  + [hierarchical agents message, if feature enabled]
```

- If only `instructions` is present (no AGENTS.md), it is returned alone
- If only AGENTS.md is present (no `instructions`), it is returned alone
- If neither exists, `None` is returned
- The separator string is: `\n\n--- project-doc ---\n\n`

### Hierarchical Agents Message

When the `child_agents_md` feature flag is enabled, Codex appends guidance about AGENTS.md
scope (from `codex-rs/core/hierarchical_agents_message.md`):

> Files called AGENTS.md commonly appear in many places inside a container - at "/", in "~",
> deep within git repositories, or in any other directory; their location is not limited to
> version-controlled folders.
>
> Their purpose is to pass along human guidance to you, the agent. Such guidance can include
> coding standards, explanations of the project layout, steps for building or testing, and even
> wording that must accompany a GitHub pull-request description produced by the agent; all of
> it is to be followed.
>
> Each AGENTS.md governs the entire directory that contains it and every child directory beneath
> that point. Whenever you change a file, you have to comply with every AGENTS.md whose scope
> covers that file. Naming conventions, stylistic rules and similar directives are restricted
> to the code that falls inside that scope unless the document explicitly states otherwise.
>
> When two AGENTS.md files disagree, the one located deeper in the directory structure overrides
> the higher-level file, while instructions given directly in the prompt by the system,
> developer, or user outrank any AGENTS.md content.

### Edge Cases

- **Directories named AGENTS.md** are silently ignored (only regular files are read)
- **Special files** (FIFOs, sockets) are silently ignored
- **Empty files** produce no output (trimmed, not included)
- **Symlinks** are followed (no special handling)
- **Missing cwd** returns `None` gracefully

### Integration Implication for Aether

**No special parsing is required.** AGENTS.md has no schema -- it is plain markdown injected
verbatim. Aether can generate AGENTS.md files by simply writing markdown content with no format
conversion, no frontmatter, and no special sections.

---

## Agent Roles (config.toml `[agents]` section)

### Overview

Codex supports defining custom agent roles via `config.toml`. These are distinct from AGENTS.md
-- roles define spawnable sub-agents with their own configuration, while AGENTS.md defines
project instructions.

### Config Format

```toml
[agents]
max_threads = 10
max_depth = 3
job_max_runtime_seconds = 300

[agents.<role_name>]
description = "Human-facing role description (required)"
config_file = "/path/to/role-config.toml"
nickname_candidates = ["nickname1", "nickname2"]
```

### Role File Format (TOML)

Agent role files referenced by `config_file` are **TOML** files with this structure:

```toml
name = "role-name"                        # Required if not declared in config.toml
description = "Role description"          # Required
nickname_candidates = ["name1", "name2"]  # Optional
developer_instructions = "..."            # Required for auto-discovered role files

# Any additional config.toml keys are passed through as role-specific config overrides
model = "gpt-5.2-codex"
approval_policy = "on_request"
```

### Discovery

Roles are discovered from:
1. Explicit declarations in `config.toml` under `[agents.<name>]`
2. Auto-discovery from `agents/` directory (relative to config folder), scanning for `*.toml`
   files recursively

### Validation Rules

- `description` is required for all roles
- `name` is required in role files (or provided as hint from config declaration)
- `nickname_candidates` must be non-empty, unique, and contain only ASCII alphanumeric,
  spaces, hyphens, underscores
- `developer_instructions` is required for auto-discovered role files (not required when role
  name is declared in config.toml)
- Duplicate role names within the same config layer are rejected

### AgentRoleToml Schema Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | `string` | Yes (unless in role file) | Human-facing role documentation |
| `config_file` | `path` | No | Path to role-specific config layer |
| `nickname_candidates` | `string[]` | No | Candidate nicknames for spawned agents |

---

## Plugin System

### Overview

Plugins are bundles of skills, MCP servers, and apps. They are NOT the same as AGENTS.md --
plugins provide capabilities (tools, skills), while AGENTS.md provides instructions.

### Manifest Format

Plugins use a JSON manifest at `.codex-plugin/plugin.json`:

```json
{
  "name": "plugin-name",
  "version": "1.0.0",
  "description": "What this plugin does",
  "skills": "./path/to/skills",
  "mcpServers": "./path/to/mcp-config",
  "apps": "./path/to/apps",
  "interface": {
    "displayName": "Plugin Display Name",
    "shortDescription": "Brief description",
    "longDescription": "Detailed description",
    "developerName": "Author",
    "category": "category",
    "capabilities": ["capability1"],
    "websiteUrl": "https://...",
    "defaultPrompt": "Summarize my inbox",
    "brandColor": "#hex"
  }
}
```

**Path rules:** All paths in the manifest must start with `./` and be relative to the plugin
root. No `..` traversal is allowed.

### Plugin Rendering

When plugins are loaded, they are rendered into the system prompt as a "Plugins" section
wrapped in `PLUGINS_INSTRUCTIONS_OPEN_TAG` / `PLUGINS_INSTRUCTIONS_CLOSE_TAG` tags. Each
explicitly mentioned plugin generates a developer hint pointing at its MCP servers, apps, and
skill prefix.

---

## Key Differences: Claude Code vs Codex CLI

| Aspect | Claude Code | Codex CLI |
|--------|------------|-----------|
| Instructions file | `CLAUDE.md` (root + dirs) | `AGENTS.md` (root + dirs) |
| Instructions format | Plain markdown | Plain markdown |
| Frontmatter on instructions | None | None |
| Discovery | Root markers (.git) + cwd | Root markers (.git) + cwd |
| Hierarchy | Root to cwd, deeper wins | Root to cwd, deeper wins |
| Agent definitions | `.claude/agents/*.md` | `config.toml [agents.*]` + `agents/*.toml` |
| Agent file format | Markdown with frontmatter | TOML with ConfigToml keys |
| Commands/slash | `.claude/commands/*.md` | Not directly equivalent (plugins/skills) |
| Config | `.claude/settings.json` | `~/.codex/config.toml` |
| Skills | Not native | SKILL.md with YAML frontmatter |
| Plugins | Not native | `.codex-plugin/plugin.json` manifest |

---

## Source Files Referenced

| File | Purpose |
|------|---------|
| `codex-rs/core/src/project_doc.rs` | AGENTS.md discovery, reading, concatenation |
| `codex-rs/core/src/project_doc_tests.rs` | Unit tests for AGENTS.md parsing |
| `codex-rs/core/tests/suite/agents_md.rs` | Integration tests for AGENTS.md |
| `codex-rs/core/src/config/agent_roles.rs` | Agent role loading and TOML parsing |
| `codex-rs/core/src/plugins/mod.rs` | Plugin system module |
| `codex-rs/core/src/plugins/manifest.rs` | Plugin manifest (plugin.json) parsing |
| `codex-rs/core/src/plugins/render.rs` | Plugin instruction rendering |
| `codex-rs/core/src/plugins/injection.rs` | Plugin injection into agent context |
| `codex-rs/core/src/external_agent_config.rs` | External agent config migration |
| `codex-rs/core/config.schema.json` | JSON Schema for config.toml |
| `codex-rs/core/hierarchical_agents_message.md` | Hierarchical agents guidance text |
| `AGENTS.md` (repo root) | Example AGENTS.md in Codex's own repo |
| `docs/agents_md.md` | AGENTS.md documentation (links to OpenAI docs) |
| `docs/config.md` | Configuration documentation |

---

## Skill Discovery Paths

Codex discovers skills from multiple root directories, determined by the config layer stack
and the current working directory. The discovery system uses a `SkillRoot` struct with a path
and a `SkillScope` (one of: `repo`, `user`, `system`, `admin`).

### Discovery Order (Highest Priority First)

1. **Repo scope** (SkillScope::Repo) -- project-level skills
2. **User scope** (SkillScope::User) -- user-installed skills
3. **System scope** (SkillScope::System) -- bundled system skills
4. **Admin scope** (SkillScope::Admin) -- system-wide admin skills

When skills have the same name across scopes, the highest-priority scope wins (repo > user > system > admin).

### Root Paths (from `skill_roots_from_layer_stack_inner` in `loader.rs`)

#### Project Layer (ConfigLayerSource::Project)
- **Path:** `{project_root}/.codex/skills/`
- **Scope:** `Repo`
- Discovered from the config layer stack's project layer `config_folder`.

#### User Layer (ConfigLayerSource::User)
Three paths are registered for the user layer:

1. **Path:** `$CODEX_HOME/skills/` (deprecated but kept for backward compat)
   - **Scope:** `User`
   - When `CODEX_HOME` is unset, defaults to `~/.codex/skills/`

2. **Path:** `$HOME/.agents/skills/`
   - **Scope:** `User`
   - The preferred user-level skill location.

3. **Path:** `$CODEX_HOME/skills/.system/`
   - **Scope:** `System`
   - Embedded system skills are cached here by Codex itself.

#### System Layer (ConfigLayerSource::System)
- **Path:** `/etc/codex/skills/`
- **Scope:** `Admin`

#### Plugin Roots
- Additional user-scoped roots registered by plugins via `effective_skill_roots`.

### Repo Agents Skills (`.agents/skills/`)

In addition to `.codex/skills/`, Codex also scans for `.agents/skills/` directories between
the project root and the current working directory:

```rust
fn repo_agents_skill_roots(config_layer_stack, cwd) -> Vec<SkillRoot> {
    // Walk from project root to cwd
    // For each directory, check if `.agents/skills/` exists
    // If so, add as a Repo-scoped root
}
```

This means if you have a monorepo with subdirectories containing `.agents/skills/`, those skills
are discovered as repo-scoped.

### Discovery Algorithm

1. Roots are collected from the config layer stack (project, user, system layers).
2. Plugin skill roots are appended as user-scoped.
3. Repo `.agents/skills/` directories between project root and cwd are appended.
4. Duplicate paths are removed (first occurrence wins).
5. For each root, a BFS scan is performed up to `MAX_SCAN_DEPTH = 6` levels deep.
6. Each root can contain up to `MAX_SKILLS_DIRS_PER_ROOT = 2000` directories.
7. Directories/files starting with `.` are skipped during scan.
8. Symlinks are followed for user, admin, and repo scopes (not system).
9. Any file named `SKILL.md` found during the scan is parsed as a skill.

### Key Constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `SKILLS_FILENAME` | `"SKILL.md"` | The required skill definition file |
| `AGENTS_DIR_NAME` | `".agents"` | Alternate skills parent directory |
| `SKILLS_DIR_NAME` | `"skills"` | Skills subdirectory name |
| `SKILLS_METADATA_DIR` | `"agents"` | Metadata subdirectory within skill |
| `SKILLS_METADATA_FILENAME` | `"openai.yaml"` | UI metadata file |
| `MAX_SCAN_DEPTH` | `6` | Max directory traversal depth |
| `MAX_SKILLS_DIRS_PER_ROOT` | `2000` | Max directories scanned per root |

---

## SKILL.md Format

### Structure

A SKILL.md file consists of two parts:
1. **YAML Frontmatter** (required) -- delimited by `---` lines
2. **Markdown Body** (required) -- instructions and guidance

### Frontmatter Fields

```yaml
---
name: skill-name          # Required. Max 64 chars. Lowercase, hyphens.
description: What it does # Required. Max 1024 chars. Primary triggering mechanism.
metadata:
  short-description: Brief # Optional. Max 1024 chars. Shown in UI.
---
```

**Frontmatter parsing rules:**
- Must be delimited by `---` on first and closing lines
- Must have at least one line between delimiters
- `name` and `description` are required (empty values cause parse error)
- Only `name`, `description`, and `metadata.short-description` are parsed from frontmatter
- Whitespace is collapsed: values are sanitized by splitting on whitespace and rejoining with single spaces

**Default name fallback:** If `name` is omitted, the parent directory name is used.

**Namespace:** Skills loaded from plugin paths get namespaced as `plugin-namespace:skill-name`.

### Markdown Body

The body contains the actual instructions for using the skill. It is only loaded when the
skill is triggered (progressive disclosure). Key guidelines from the skill-creator sample:

- Keep under 500 lines to minimize context bloat
- Use imperative/infinitive form
- Include only information another Codex instance would need
- Reference bundled resources (scripts, references, assets) with relative paths
- The `description` in frontmatter is the primary triggering mechanism -- put "when to use" info there, not in the body

---

## Skill Directory Structure

```
skill-name/
  SKILL.md              # Required. Skill definition.
  agents/
    openai.yaml         # Recommended. UI metadata (display_name, icons, etc.)
  scripts/              # Optional. Executable code (Python, Bash, etc.)
  references/           # Optional. Documentation loaded as needed.
  assets/               # Optional. Files used in output (templates, icons, etc.).
```

### openai.yaml (UI Metadata)

Located at `agents/openai.yaml` within the skill directory. This is optional metadata for
skill lists and chips in the UI.

```yaml
interface:
  display_name: "Human Readable Name"    # Max 64 chars
  short_description: "Brief description" # Max 1024 chars
  icon_small: "./assets/icon-small.svg"  # Relative to skill dir, under assets/
  icon_large: "./assets/icon-large.png"  # Relative to skill dir, under assets/
  brand_color: "#RRGGBB"                 # Hex color only
  default_prompt: "Suggested prompt"     # Max 1024 chars

dependencies:
  tools:
    - type: "mcp"                        # Required for each tool dep
      value: "server-name"               # Required
      description: "What this tool does" # Optional
      transport: "stdio"                 # Optional
      command: "npx -y @org/server"      # Optional
      url: "https://..."                 # Optional

policy:
  allow_implicit_invocation: true        # Default: true
  products: []                           # Empty = all products
```

**Field notes:**
- `interface` fields are optional; if none are provided, the entire interface block is ignored
- Icon paths must be relative and under `assets/` (no `..`, no absolute paths)
- `brand_color` must be `#RRGGBB` format
- `policy.allow_implicit_invocation` defaults to `true` if not set
- `policy.products` gates which Codex product variants can use the skill; empty = all

---

## Skill Scopes

| Scope | Source | Example Path |
|-------|--------|-------------|
| `repo` | Project config layer + `.agents/skills/` | `.codex/skills/`, `.agents/skills/` |
| `user` | User config + `$HOME/.agents/skills/` | `~/.codex/skills/`, `~/.agents/skills/` |
| `system` | Bundled skills cache | `$CODEX_HOME/skills/.system/` |
| `admin` | System config layer | `/etc/codex/skills/` |

---

## Skill Invocation

### Explicit Invocation

Users can invoke skills by name using `$skill-name` syntax in their message. Skills can also
be referenced with linked mentions: `[$skill-name](skill://path/to/SKILL.md)`.

### Implicit Invocation

When `policy.allow_implicit_invocation` is `true` (the default), Codex automatically detects
when a command references a skill's scripts or documentation and triggers the skill.

Detection works by:
1. Checking if a shell command references a script inside a skill's `scripts/` directory
2. Checking if a file read references a skill's `SKILL.md` path

### Progressive Disclosure

Skills use a three-level loading system:

1. **Metadata** (name + description) -- Always in context (~100 words)
2. **SKILL.md body** -- Loaded when skill triggers (<5k words recommended)
3. **Bundled resources** -- Loaded as needed by Codex (unlimited, scripts can be executed without reading into context)

---

## Skill Configuration

Skills can be enabled/disabled via config.toml:

```toml
[skills]
bundled.enabled = true  # Enable/disable system bundled skills (default: true)

[[skills.config]]
path = "/path/to/skill"  # Disable by path
enabled = false

[[skills.config]]
name = "skill-name"       # Disable by name
enabled = false
```

---

## Rendering into System Prompt

When skills are loaded, they are rendered into the system prompt as a "Skills" section:

```
## Skills
A skill is a set of local instructions to follow that is stored in a `SKILL.md` file.

### Available skills
- skill-name: Description (file: /path/to/SKILL.md)

### How to use skills
- Discovery: The list above is the skills available in this session...
- Trigger rules: If the user names a skill (with $SkillName) OR the task clearly
  matches a skill's description, you must use that skill...
- How to use a skill (progressive disclosure):
  1) Open its SKILL.md. Read only enough to follow the workflow.
  2) Resolve relative paths relative to the skill directory.
  3) Load only specific files needed from references/.
  4) Prefer running or patching scripts/ instead of retyping code.
  5) Reuse assets/ or templates instead of recreating.
```

This section is wrapped in open/close tags (`SKILLS_INSTRUCTIONS_OPEN_TAG` / `SKILLS_INSTRUCTIONS_CLOSE_TAG`).

---

## Tool Names and Mapping

> Source: `codex-rs/tools/src/` crate, specifically `tool_registry_plan.rs`, `tool_spec.rs`,
> and individual tool definition files.
> The tool registry is built by `build_tool_registry_plan()` which conditionally adds tools
> based on `ToolsConfig` (feature flags, model capabilities, sandbox policy).

### ToolSpec Types (Serialization Format)

Tools are serialized to the OpenAI Responses API as one of these JSON types:

| Tag (type field) | Rust Variant | When Used |
|------------------|-------------|-----------|
| `"function"` | `ToolSpec::Function(ResponsesApiTool)` | Standard function-calling tools (most tools) |
| `"tool_search"` | `ToolSpec::ToolSearch { .. }` | `tool_search` discovery tool |
| `"local_shell"` | `ToolSpec::LocalShell {}` | `local_shell` tool (no parameters in spec) |
| `"image_generation"` | `ToolSpec::ImageGeneration { output_format }` | Built-in image generation |
| `"web_search"` | `ToolSpec::WebSearch { .. }` | Built-in web search |
| `"custom"` | `ToolSpec::Freeform(FreeformTool)` | Freeform grammar-based tools (apply_patch freeform, js_repl) |

### Core Tool Registry (Built-in Tools)

These are the built-in tools registered by `build_tool_registry_plan()`:

#### Shell / Execution Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `shell` | `command: string[]` (required), `workdir?: string`, `timeout_ms?: number`, `sandbox_permissions?: string`, `justification?: string`, `prefix_rule?: string[]` | Runs a shell command via execvp(). Most commands prefixed with ["bash", "-lc"]. | `shell_type == Default` and `has_environment` | Yes |
| `shell_command` | `command: string` (required), `workdir?: string`, `timeout_ms?: number`, `login?: boolean`, `sandbox_permissions?: string`, `justification?: string`, `prefix_rule?: string[]` | Runs a shell command as a string in the user's default shell. | `shell_type == ShellCommand` | Yes |
| `exec_command` | `cmd: string` (required), `workdir?: string`, `shell?: string`, `tty?: boolean`, `yield_time_ms?: number`, `max_output_tokens?: number`, `login?: boolean`, `sandbox_permissions?: string`, `justification?: string`, `prefix_rule?: string[]` | Runs a command in a PTY, returning output or a session ID. | `shell_type == UnifiedExec` | Yes |
| `write_stdin` | `session_id: number` (required), `chars?: string`, `yield_time_ms?: number`, `max_output_tokens?: number` | Writes characters to an existing unified exec session. | `shell_type == UnifiedExec` | No |
| `local_shell` | (none in spec; type tag only) | Local shell execution (type: "local_shell"). | `shell_type == Local` | Yes |

**Note:** The shell tool name varies by `ConfigShellToolType`:
- `Default` -> `shell` (array-based args via execvp)
- `ShellCommand` -> `shell_command` (string-based, Zsh fork capable)
- `UnifiedExec` -> `exec_command` + `write_stdin` (PTY-based)
- `Local` -> `local_shell` (type: "local_shell")
- `Disabled` -> no shell tool

#### File Editing Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `apply_patch` (freeform) | (freeform grammar-based, lark syntax) | Edit files using a diff-like patch format. FREEFORM tool (not JSON). | `apply_patch_tool_type == Freeform` (default for GPT-5) | No |
| `apply_patch` (function) | `input: string` (required) | Edit files using JSON-wrapped patch format. | `apply_patch_tool_type == Function` (gpt-oss models) | No |

**Note:** `apply_patch` uses `*** Begin Patch` / `*** End Patch` delimiters with `+`/`-`/` ` prefixed lines. Supports `*** Add File:`, `*** Delete File:`, `*** Update File:`, `*** Move to:` operations.

#### Planning Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `update_plan` | `plan: [{step: string, status: string}]` (required), `explanation?: string` | Updates the task plan. Statuses: pending, in_progress, completed. | Always | No |

#### User Interaction Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `request_user_input` | `questions: [{id, header, question, options}]` (required) | Request user input for 1-3 questions. Options need label + description. | Always (restricted by mode) | No |
| `request_permissions` | `permissions: {network: {enabled: bool}, file_system: {read: string[], write: string[]}}` (required), `reason?: string` | Request additional filesystem or network permissions. | `request_permissions_tool_enabled` feature | No |

#### Image Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `view_image` | `path: string` (required), `detail?: string` ("original") | View a local image from the filesystem. | `has_environment` | Yes |
| `image_generation` | (type tag: "image_generation", `output_format: "png"`) | Built-in image generation (Responses API native). | `image_gen_tool` feature | No |

**Note:** `image_generation` and `web_search` are Responses API native tool types (not function tools).

#### Web Search Tool

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `web_search` | (type tag: "web_search", `external_web_access?: bool`, `filters?: {allowed_domains: string[]}`, `user_location?: {...}`, `search_context_size?: string`, `search_content_types?: string[]`) | Built-in web search. | `web_search_mode` is set (Cached or Live) | No |

#### Code Mode Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `exec` | (freeform grammar-based) | Code mode execution: runs multiple tool calls in a single cell, with top-level await support. | `code_mode_enabled` feature | No |
| `wait` | `cell_id?: string`, `yield_time_ms?: number`, `max_tokens?: number`, `terminate?: boolean` | Wait on a yielded exec cell, returns new output or completion. | `code_mode_enabled` feature | No |

**Note:** Code mode wraps other tools (shell, apply_patch, view_image, etc.) into a single `exec` call with structured output.

#### JS REPL Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `js_repl` | (freeform grammar-based) | Runs JavaScript in a persistent Node kernel with top-level await. | `js_repl_enabled` feature | No |
| `js_repl_reset` | (no parameters) | Restarts the js_repl kernel and clears bindings. | `js_repl_enabled` feature | No |

#### Directory Listing (Experimental)

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `list_dir` | `dir_path: string` (required), `offset?: number`, `limit?: number`, `depth?: number` | Lists entries in a local directory with 1-indexed entry numbers. | `experimental_supported_tools` includes "list_dir" | Yes |

#### Test Sync (Internal)

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `test_sync_tool` | `sleep_before_ms?: number`, `sleep_after_ms?: number`, `barrier?: {id, participants, timeout_ms}` | Internal synchronization helper for Codex integration tests. | `experimental_supported_tools` includes "test_sync_tool" | Yes |

### Multi-Agent / Collaboration Tools

These tools are available when `collab_tools` feature is enabled. There are two versions (v1 and v2).

#### V1 Multi-Agent Tools

| Tool Name | Parameters | Description | Parallel |
|-----------|-----------|-------------|----------|
| `spawn_agent` | `message?: string`, `items?: array`, `agent_type?: string`, `fork_context?: boolean`, `model?: string`, `reasoning_effort?: string` | Spawn a sub-agent for a well-scoped task. | No |
| `send_input` | `target: string` (required), `message?: string`, `items?: array`, `interrupt?: boolean` | Send a message to an existing agent. | No |
| `resume_agent` | `id: string` (required) | Resume a previously closed agent. | No |
| `wait_agent` | `targets: string[]` (required), `timeout_ms?: number` | Wait for agents to reach a final status. | No |
| `close_agent` | `target: string` (required) | Close an agent and its descendants. | No |

#### V2 Multi-Agent Tools

| Tool Name | Parameters | Description | Parallel |
|-----------|-----------|-------------|----------|
| `spawn_agent` | `task_name: string` (required), `message: string` (required), `agent_type?: string`, `fork_turns?: string`, `model?: string`, `reasoning_effort?: string` | Spawns an agent with a canonical task name (path-based). | No |
| `send_message` | `target: string` (required), `message: string` (required) | Send a string message without triggering a new turn. | No |
| `followup_task` | `target: string` (required), `message: string` (required), `interrupt?: boolean` | Send a message and trigger a new turn in the target. | No |
| `wait_agent` | `timeout_ms?: number` | Wait for a mailbox update from any live agent. | No |
| `close_agent` | `target: string` (required) | Close an agent and its descendants. | No |
| `list_agents` | `path_prefix?: string` | List live agents in the current root thread tree. | No |

**Version selection:** `multi_agent_v2` feature flag selects V2; otherwise V1 is used.

### Agent Job Tools (Batch CSV Processing)

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `spawn_agents_on_csv` | `csv_path: string` (required), `instruction: string` (required), `id_column?: string`, `output_csv_path?: string`, `max_concurrency?: number`, `max_workers?: number`, `max_runtime_seconds?: number`, `output_schema?: object` | Process a CSV by spawning one worker per row. | `agent_jobs_tools` feature | No |
| `report_agent_job_result` | `job_id: string` (required), `item_id: string` (required), `result: object` (required), `stop?: boolean` | Worker-only tool to report job results. | `agent_jobs_worker_tools` feature | No |

### MCP Tools

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `list_mcp_resources` | `server?: string`, `cursor?: string` | Lists MCP server resources. | MCP tools configured | Yes |
| `list_mcp_resource_templates` | `server?: string`, `cursor?: string` | Lists parameterized MCP resource templates. | MCP tools configured | Yes |
| `read_mcp_resource` | `server: string` (required), `uri: string` (required) | Read a specific MCP resource. | MCP tools configured | Yes |
| `tool_search` | `query: string` (required), `limit?: number` | Search over MCP tool metadata with BM25, exposes matching tools for next model call. | `search_tool` feature + deferred MCP tools | Yes |
| (dynamic MCP tools) | Varies per MCP server | Tools from connected MCP servers, namespaced as `namespace__tool_name`. | MCP servers connected | No |

**Note:** MCP tools use namespacing. Tool names follow the pattern `namespace__tool_name` where namespace is derived from the connector or server name.

### Tool Discovery / Suggestion

| Tool Name | Parameters | Description | Condition | Parallel |
|-----------|-----------|-------------|-----------|----------|
| `tool_suggest` | `tool_type: string` (required), `action_type: string` (required), `tool_id: string` (required), `suggest_reason: string` (required) | Suggest a missing connector or plugin for the user to install. | `tool_suggest` feature + apps + plugins | Yes |

### Complete Tool Handler Kinds (Internal)

These are the internal handler kinds used in the registry:

| Handler Kind | Registered Tool Name(s) |
|-------------|------------------------|
| `Shell` | `shell`, `container.exec`, `local_shell` |
| `ShellCommand` | `shell_command` |
| `UnifiedExec` | `exec_command`, `write_stdin` |
| `ApplyPatch` | `apply_patch` |
| `Plan` | `update_plan` |
| `RequestUserInput` | `request_user_input` |
| `RequestPermissions` | `request_permissions` |
| `ViewImage` | `view_image` |
| `CodeModeExecute` | `exec` |
| `CodeModeWait` | `wait` |
| `JsRepl` | `js_repl` |
| `JsReplReset` | `js_repl_reset` |
| `ListDir` | `list_dir` |
| `TestSync` | `test_sync_tool` |
| `SpawnAgentV1` | `spawn_agent` (v1) |
| `SendInputV1` | `send_input` (v1) |
| `ResumeAgentV1` | `resume_agent` |
| `WaitAgentV1` | `wait_agent` (v1) |
| `CloseAgentV1` | `close_agent` (v1) |
| `SpawnAgentV2` | `spawn_agent` (v2) |
| `SendMessageV2` | `send_message` |
| `FollowupTaskV2` | `followup_task` |
| `WaitAgentV2` | `wait_agent` (v2) |
| `CloseAgentV2` | `close_agent` (v2) |
| `ListAgentsV2` | `list_agents` |
| `AgentJobs` | `spawn_agents_on_csv`, `report_agent_job_result` |
| `Mcp` | MCP tool names (dynamic) |
| `McpResource` | `list_mcp_resources`, `list_mcp_resource_templates`, `read_mcp_resource` |
| `ToolSearch` | `tool_search` |
| `ToolSuggest` | `tool_suggest` |
| `DynamicTool` | Dynamic tool names |

### Model-Specific Tool Behavior

Tools vary per model based on `ModelInfo` fields:

| ModelInfo Field | Effect on Tools |
|-----------------|----------------|
| `shell_type` (`ConfigShellToolType`) | Determines which shell tool variant is used: `Default` (shell), `ShellCommand` (shell_command), `UnifiedExec` (exec_command), `Local` (local_shell), `Disabled` (no shell) |
| `apply_patch_tool_type` | `Some(Freeform)` = grammar-based patch; `Some(Function)` = JSON-wrapped patch; `None` = feature flag decides |
| `supports_parallel_tool_calls` | Whether the model supports parallel tool calls |
| `supports_search_tool` | Whether `tool_search` is available |
| `supports_image_detail_original` | Whether `view_image` accepts `detail: "original"` |
| `input_modalities` | If includes `Image`, enables `image_gen_tool` |
| `web_search_tool_type` | `Text` or `TextAndImage` content types for web search |
| `experimental_supported_tools` | Enables experimental tools like `list_dir`, `test_sync_tool` |

### Tool Condition Summary

A tool is included in the registry when ALL of its conditions are met:

```
shell (Default):      has_environment AND shell_type == Default
shell_command:        has_environment AND shell_type == ShellCommand
exec_command:         has_environment AND shell_type == UnifiedExec
local_shell:          has_environment AND shell_type == Local
apply_patch:          has_environment AND apply_patch_tool_type is set
update_plan:          always
request_user_input:   always (mode-restricted)
request_permissions:  request_permissions_tool_enabled
view_image:           has_environment
list_dir:             has_environment AND "list_dir" in experimental_supported_tools
test_sync_tool:       "test_sync_tool" in experimental_supported_tools
web_search:           web_search_mode is Some(Cached) or Some(Live)
image_generation:     image_gen_tool AND supports Image input modality
code_mode (exec/wait): code_mode_enabled feature
js_repl / js_repl_reset: has_environment AND js_repl_enabled
collab tools:         collab_tools feature
agent jobs:           agent_jobs_tools feature
mcp resources:        mcp_tools is Some
tool_search:          search_tool feature AND deferred_mcp_tools is Some
tool_suggest:         tool_suggest AND apps AND plugins features AND non-empty discoverable_tools
```

### Sandbox Permissions

Shell tools accept sandbox permission controls:

| Field | Values | Description |
|-------|--------|-------------|
| `sandbox_permissions` | `"use_default"` (default), `"with_additional_permissions"`, `"require_escalated"` | Permission level for the command |
| `justification` | string | Required when `require_escalated`; explains why sandbox bypass is needed |
| `prefix_rule` | string[] | Suggested prefix pattern for future similar commands |
| `additional_permissions` | `{network: {enabled: bool}, file_system: {read: string[], write: string[]}}` | Additional permissions (when `exec_permission_approvals_enabled`) |

---

## Relevant Source Files

| File | Purpose |
|------|---------|
| `codex-rs/tools/src/tool_registry_plan.rs` | Master registry plan: conditionally builds tool list |
| `codex-rs/tools/src/tool_config.rs` | `ToolsConfig` and `ToolsConfigParams`: feature-flag-driven config |
| `codex-rs/tools/src/tool_spec.rs` | `ToolSpec` enum (function, freeform, web_search, etc.) |
| `codex-rs/tools/src/tool_name.rs` | `ToolName` struct (plain or namespaced) |
| `codex-rs/tools/src/local_tool.rs` | `shell`, `shell_command`, `exec_command`, `write_stdin`, `request_permissions` |
| `codex-rs/tools/src/apply_patch_tool.rs` | `apply_patch` (freeform + function variants) |
| `codex-rs/tools/src/plan_tool.rs` | `update_plan` |
| `codex-rs/tools/src/request_user_input_tool.rs` | `request_user_input` |
| `codex-rs/tools/src/view_image.rs` | `view_image` |
| `codex-rs/tools/src/image_detail.rs` | Image detail level handling |
| `codex-rs/tools/src/agent_tool.rs` | `spawn_agent`, `send_input`, `send_message`, `wait_agent`, `close_agent`, etc. |
| `codex-rs/tools/src/agent_job_tool.rs` | `spawn_agents_on_csv`, `report_agent_job_result` |
| `codex-rs/tools/src/code_mode.rs` | `exec` (code_mode), `wait` (code_mode) |
| `codex-rs/tools/src/js_repl_tool.rs` | `js_repl`, `js_repl_reset` |
| `codex-rs/tools/src/utility_tool.rs` | `list_dir`, `test_sync_tool` |
| `codex-rs/tools/src/mcp_tool.rs` | MCP tool conversion to Responses API format |
| `codex-rs/tools/src/mcp_resource_tool.rs` | `list_mcp_resources`, `list_mcp_resource_templates`, `read_mcp_resource` |
| `codex-rs/tools/src/tool_discovery.rs` | `tool_search`, `tool_suggest` |
| `codex-rs/tools/src/responses_api.rs` | `ResponsesApiTool`, `FreeformTool` serialization types |
| `codex-rs/tools/src/tool_definition.rs` | `ToolDefinition` (base metadata + schema) |
| `codex-rs/tools/src/tool_registry_plan_types.rs` | `ToolHandlerKind` enum, `ToolRegistryPlan` |
| `codex-rs/tools/src/registry.rs` | `ToolRegistry`: runtime dispatch of tool calls to handlers |
| `codex-rs/protocol/src/openai_models.rs` | `ModelInfo`, `ConfigShellToolType`, `ApplyPatchToolType` |
| `codex-rs/models-manager/src/model_info.rs` | Per-model metadata construction |
| `codex-rs/core/src/tools/spec.rs` | `build_specs_with_discoverable_tools()`: wires plan to handlers |
| `codex-rs/code-mode/src/lib.rs` | `PUBLIC_TOOL_NAME = "exec"`, `WAIT_TOOL_NAME = "wait"` |
| `codex-rs/core-skills/src/loader.rs` | Skill root discovery and SKILL.md parsing |
| `codex-rs/core-skills/src/model.rs` | Data structures (SkillMetadata, SkillScope, etc.) |
| `codex-rs/core-skills/src/manager.rs` | SkillsManager with caching and loading |
| `codex-rs/core-skills/src/injection.rs` | Explicit/implicit skill invocation detection |
| `codex-rs/core-skills/src/render.rs` | Rendering skills section into system prompt |
| `codex-rs/core-skills/src/system.rs` | System skills installation and cache |
| `codex-rs/skills/src/lib.rs` | Embedded system skills (include_dir) |
| `codex-rs/config/src/skills_config.rs` | TOML config types for skill enable/disable |
| `codex-rs/core/src/skills.rs` | Core re-exports and dependency resolution |
| `codex-rs/skills/src/assets/samples/` | Sample skills (skill-creator, imagegen, etc.) |
| `codex-rs/core/src/project_doc.rs` | AGENTS.md discovery, reading, concatenation |
| `codex-rs/core/src/project_doc_tests.rs` | Unit tests for AGENTS.md parsing |
| `codex-rs/core/tests/suite/agents_md.rs` | Integration tests for AGENTS.md |
| `codex-rs/core/src/config/agent_roles.rs` | Agent role loading and TOML parsing |
| `codex-rs/core/src/plugins/mod.rs` | Plugin system module |
| `codex-rs/core/src/plugins/manifest.rs` | Plugin manifest (plugin.json) parsing |
| `codex-rs/core/src/plugins/render.rs` | Plugin instruction rendering |
| `codex-rs/core/src/plugins/injection.rs` | Plugin injection into agent context |
| `codex-rs/core/src/external_agent_config.rs` | External agent config migration |
| `codex-rs/core/config.schema.json` | JSON Schema for config.toml |
| `codex-rs/core/hierarchical_agents_message.md` | Hierarchical agents guidance text |
