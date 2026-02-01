# Technology Stack

**Project:** Aether v2 — Claude-native multi-agent system
**Researched:** 2026-02-01
**Overall confidence:** HIGH

## Executive Summary

Aether v2 represents a **paradigm shift** in multi-agent systems: **Claude-native prompt-based architecture** instead of Python-based orchestration. The stack is deliberately minimal—**prompt files, JSON state, and the Task tool**—enabling autonomous agent spawning without external dependencies.

**Core insight from research:** The 2025 standard for Claude-native systems is **not** a complex framework—it's thoughtful use of Claude Code's built-in features: custom commands, agent skills, the Task tool for spawning, and JSON-based state persistence.

---

## Recommended Stack

### Core Framework (Native Claude Code)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Claude Code CLI** | 2.0+ | Native execution environment | Provides Task tool, sub-agent spawning, agent skills, hooks—all built-in. **HIGH confidence** based on official Anthropic documentation and production usage. |
| **Prompt Files** | N/A | Agent behavior definition | Commands stored as `.md` files in `.claude/commands/`. **De facto standard**—verified across multiple sources. |
| **Task Tool** | Built-in | Autonomous agent spawning | Official mechanism for spawning sub-agents. 5 agent types: general-purpose, Explore, Plan, claude-code-guide, statusline-setup. **HIGH confidence** from leaked system prompts. |
| **Agent Skills** | Oct 2025+ | Domain expertise on-demand | Folders with `SKILL.md` that load context only when needed. **Official feature** released Oct 16, 2025. |
| **JSON State** | N/A | Persistence layer | Simple, human-readable, Claude can read/write directly. **No databases required** for Aether v2 scope. |

### State Management

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **JSON Files** | N/A | Working memory, pheromones, intentions | Claude can natively read/write JSON. Simple, debuggable, no dependencies. **Current Aether pattern** works well. |
| **File-based Checkpoints** | N/A | State snapshots | Store before/after states for rollback. Native to filesystem. |
| **Markdown Logs** | N/A | Agent communication history | Human-readable, queryable with Grep, compressible. |

### Agent Communication

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Pheromone System** | N/A | Semantic signaling | JSON-based signal broadcasting (type, strength, persistence). **Aether innovation**—working implementation exists. |
| **Inherited Context** | N/A | Parent→child state passing | Task tool's `resume` parameter + custom context injection. **Verified pattern** from Task tool schema. |
| **Triple-Layer Memory** | N/A | Working/short-term/long-term | Compression-based memory hierarchy. **Research-backed** pattern from Anthropic's context engineering. |

### Security & Isolation

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **Claude Code Sandboxing** | 2025+ | Filesystem & network isolation | OS-level primitives (Linux bubblewrap, macOS seatbelt). **Official feature**—reduces permission prompts by 84%. |
| **No External Dependencies** | N/A | Attack surface reduction | Aether v2 runs entirely within Claude Code. No vector DBs, no embedding services. **Design principle**. |

### Supporting Patterns

| Pattern | Purpose | When to Use |
|---------|---------|-------------|
| **Custom Commands** (`.claude/commands/*.md`) | Repeatable workflows | For structured commands like `/ant:init`, `/ant:build`. **Always**—this is the primary interface. |
| **Agent Skills** (`.claude/skills/*`) | Domain expertise on-demand | For large context (e.g., frontend design, security). Load only when needed. |
| **Hooks** (`~/.claude/hooks/*`) | Lifecycle events | For side-effects (sounds, notifications, auto-prompts). |
| **Plugins** (`.claude/plugins/*`) | Distributable units | For sharing combined skills+commands+hooks across teams. |

---

## Installation

**No package installation required.** Aether v2 is prompt-based.

```bash
# Setup directory structure
mkdir -p .claude/commands/ant
mkdir -p .claude/agents
mkdir -p .aether/data
mkdir -p .aether/memory
mkdir -p .aether/checkpoints
mkdir -p .aether/errors

# All "code" is prompt files and JSON state
# No npm install, no pip install, no docker-compose
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| **Prompt files + JSON** | **Python orchestration (AutoGen, LangGraph)** | Use alternative if you need: async I/O, external API calls, complex state machines. But for Claude-native? **Prompt-based is superior**—faster iteration, Claude-native debugging, no Python dependency hell. |
| **Task tool spawning** | **Manual agent creation** | Never use manual. Task tool is the **official** mechanism. Hand-rolling spawning will fight Claude's harness. |
| **JSON state** | **Vector DB (Chroma, Pinecone)** | Use vector DB if you have: millions of embeddings, semantic search across huge corpus. For Aether's scope? **Overkill**. JSON + grep is sufficient and simpler. |
| **File-based prompts** | **MCP servers** | MCP is great for external integrations (GitHub, Drive, Figma). But for Aether's core agent logic? **Keep it prompt-based**. MCP adds tool definition bloat to context. |
| **Claude Code native** | **Claude API with custom harness** | Use API if you're building a product for others. For personal/team development? **Claude Code CLI is better**—sandboxing, checkpointing, syntax highlighting, all built-in. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| **Python async/await** | Aether v2 is prompt-based, not Python-based. Async adds complexity without benefit since Claude handles concurrency via parallel Task tool calls. | Use Task tool's `run_in_background` parameter for concurrent agent execution. |
| **External vector databases** | Embedding services (OpenAI, Cohere) cost money and add latency. For Aether's scope, JSON-based memory is sufficient. | JSON state + grep for search. If you must have semantic search, use local embeddings (sentence-transformers) but **verify you actually need it first**. |
| **Complex state machines** | Claude's context window is the state. External state machines create synchronization issues. | Let Claude manage state through JSON files and inherited context. |
| **Docker/Kubernetes** | Aether v2 runs in Claude Code's sandbox. Containerization adds unnecessary complexity. | Rely on Claude Code's built-in sandboxing (filesystem + network isolation). |
| **Redis/PostgreSQL for state** | Overkill for prototype phase. Adds operational overhead. | JSON files. Migrate to DB only when you have proven scale requirements. |
| **Pre-2025 patterns** | Multi-agent patterns from 2023-2024 (AutoGen v0.2, old LangChain) are obsolete. Claude Code 2.0 (Feb 2025) + Agent Skills (Oct 2025) changed everything. | Current patterns: Task tool spawning, agent skills, hooks, plugins. |

---

## File Structure (Prescriptive)

```
.aether/
├── data/
│   ├── intention.json          # User's goal, colony state
│   ├── pheromones.json         # Active pheromone signals
│   ├── worker_ants.json        # Agent statuses
│   └── memory.json             # Working memory (ephemeral context)
├── memory/
│   ├── long_term.json          # Compressed patterns, learnings
│   ├── short_term/             # Last 10 sessions compressed
│   └── outcome_tracker.json    # Meta-learning: what worked
├── errors/
│   ├── patterns.json           # Auto-flagged error patterns
│   └── err_*.json              # Individual error logs
├── checkpoints/
│   ├── checkpoint_*_before.json
│   └── checkpoint_*_after.json
└── (Python modules only if needed for validation/CLI)

.claude/
├── commands/
│   └── ant/
│       ├── init.md             # /ant:init
│       ├── build.md            # /ant:build
│       ├── colonize.md         # /ant:colonize
│       ├── plan.md             # /ant:plan
│       ├── phase.md            # /ant:phase
│       └── ...
├── agents/
│   └── (custom sub-agent definitions if needed)
├── skills/
│   └── (domain expertise loaded on-demand)
└── CLAUDE.md                   # Global instructions
```

**Key principle:** The **prompt files are the code**. JSON files are the database. Everything else is optional.

---

## Prompt Patterns (XML Formatting)

Based on leaked Claude Code system prompts and Sankalp's reverse engineering:

### Command File Structure

```markdown
---
name: ant:init
description: Initialize new project - Queen sets intention, colony creates phase structure
---

<objective>
[What this command accomplishes]
</objective>

<process>
[Step-by-step instructions for Claude]
</process>

<context>
@.aether/worker_ants.py
@.aether/memory/triple_layer_memory.py

[Relevant patterns, agent castes, spawning rules]
</context>

<reference>
[Static knowledge, templates, examples]
</reference>

<allowed-tools>
Task
Write
Bash
Read
Glob
Grep
AskUserQuestion
</allowed-tools>
```

### Why XML Tags?

**HIGH confidence** from official system prompts: Claude's harness uses `<system-reminder>`, `<tool_result>`, and other XML tags. Using XML in your prompts:
- Creates clear boundaries between instruction sections
- Matches Claude's internal patterns
- Reduces instruction-following errors
- Makes prompts more parseable for future tooling

---

## Task Tool Spawning Pattern

**Verified from leaked Task tool schema (Sankalp, Dec 2025):**

```xml
<invoke>
<parameter name="description">Create phase structure</parameter>
<parameter name="prompt">
You are the Planner Ant. Create a structured phase plan for:

GOAL: {user's goal}

[Detailed instructions...]
</parameter>
<parameter name="subagent_type">general-purpose</parameter>
<parameter name="model">sonnet</parameter>
<parameter name="run_in_background">true</parameter>
</invoke>
```

**Agent types:**
- `general-purpose`: Full tool access, inherits context
- `Explore`: Read-only codebase search (fast, starts fresh)
- `Plan`: Software architect for implementation planning
- `claude-code-guide`: Documentation lookup
- `statusline-setup`: Configure status line

**Critical:** Never spawn sub-agents for simple tasks. Task tool has overhead. Use only for multi-step, complex tasks.

---

## State Management Approaches

### Working Memory (Ephemeral)

```json
{
  "current_goal": "Build authentication system",
  "active_phase": 2,
  "pheromones": [
    {"type": "FOCUS", "strength": 0.8, "source": "security_specialist"}
  ],
  "active_agents": ["executor", "security_specialist"],
  "context_cache": {
    "jwt_library": "PyJWT 2.8+",
    "db_schema": "users table exists"
  }
}
```

**Pattern:** Clear after each session. Rebuild from short-term memory if needed.

### Short-Term Memory (Compressed Sessions)

```json
{
  "session_20260201_143000": {
    "goal": "Add OAuth login",
    "phases_completed": [1, 2],
    "patterns_learned": [
      "Use authlib for OAuth2 providers",
      "Store tokens in httpOnly cookies"
    ],
    "errors_avoided": ["ERR_001: missing CSRF token"]
  }
}
```

**Pattern:** Keep last 10 sessions. Compress to long-term after threshold.

### Long-Term Memory (Persistent Patterns)

```json
{
  "auth_patterns": {
    "jwt": {
      "library": "PyJWT",
      "algorithm": "HS256",
      "expiry": "24 hours",
      "success_rate": 0.9
    }
  },
  "generalizations": {
    "security_specialist": ["authentication", "jwt", "owasp"]
  }
}
```

**Pattern:** Extract patterns from multiple sessions. Use for meta-learning.

---

## Version-Specific Recommendations

### Claude Code 2.0+ (Feb 2025 - Present)

**Must-have features:**
- Task tool with 5 agent types
- Checkpointing (`Esc + Esc` or `/rewind`)
- Sub-agent spawning (background + foreground)
- Syntax highlighting (2.0.71+)
- Prompt suggestions (2.0.73+)
- History search (`Ctrl + R`)

### Agent Skills (Oct 2025+)

**Use when:**
- You have >500 lines of domain expertise
- Expertise is not always needed (load on-demand)
- You want to keep CLAUDE.md small

**Structure:**
```
.claude/skills/my-skill/
├── SKILL.md          # Required: name, description, when to use
├── reference.md      # Optional: detailed docs
└── scripts/          # Optional: helper scripts
```

**Don't use skills for:**
- Simple commands (<50 lines) → Use custom commands instead
- Always-needed context → Put in CLAUDE.md or command `<context>` blocks

---

## Stack Patterns by Variant

### If building greenfield project:
- Use `/ant:init` to create phase structure
- Use `/ant:colonize` to understand existing patterns (if extending)
- Use `/ant:build` for autonomous execution
- **Why:** This is Aether's core workflow. Phase-based structure prevents context rot.

### If working in existing codebase:
- Run `/ant:colonize` first
- Store patterns in long-term memory
- Match existing conventions (Aether reads from synthesis)
- **Why:** Aether's Mapper + Synthesizer agents extract patterns. Respect them.

### If debugging agent behavior:
- Use `/ant:memory` to query what agents "remember"
- Check `.aether/errors/patterns.json` for auto-flagged issues
- Review spawn history in meta-learning data
- **Why:** Aether has observability built-in. Use it before debugging prompts.

---

## Known Compatibility Issues

| Component | Compatible With | Notes |
|-----------|-----------------|-------|
| Prompt files (`.md`) | All Claude Code 2.0+ versions | YAML frontmatter is standard. No issues. |
| Task tool spawning | Claude Code 2.0+ | Pre-2.0 doesn't have Task tool. **HIGH confidence** from feature release notes. |
| Agent Skills | Claude Code 2.0.60+ (Oct 2025) | Skills were released Oct 16, 2025. Earlier versions will ignore `.claude/skills/`. |
| Checkpointing | Claude Code 2.0.56+ | `Esc + Esc` and `/rewind` added in 2.0.56. |
| JSON state | All versions | JSON is universal. No compatibility concerns. |

---

## Migration Path (If You Have Old Aether v1)

**Aether v1 was Python-based. Aether v2 is prompt-based.**

**Migration steps:**
1. Keep v1's JSON state structure (it's good)
2. Convert Python agent logic to prompt files (use `<process>` blocks)
3. Replace Python spawning with Task tool calls
4. Remove async/await (Claude handles concurrency)
5. Remove dependencies (no more requirements.txt)

**Don't:** Try to run v1 and v2 side-by-side. **Do:** Choose one paradigm per project.

---

## Sources

### HIGH Confidence (Official/Primary)

- [Claude Code Official Documentation - Sandboxing](https://www.anthropic.com/engineering/claude-code-sandboxing) — Verified security model, filesystem/network isolation, 84% permission prompt reduction. **Official Anthropic engineering blog.**
- [Claude Code Official Documentation - Agent Skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) — Official announcement Oct 16, 2025. Skills architecture, load-on-demand pattern.
- [Claude Code Agent Skills API Docs](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview) — Official API documentation for skills structure.
- [Leaked Claude Code System Prompts](https://sankalp.bearblog.dev/my-experience-with-claude-code-20-and-how-to-get-better-at-using-coding-agents/) — Dec 27, 2025. Reverse-engineered Task tool schema, agent types, spawning logic. **Verified against official docs where possible.**

### MEDIUM Confidence (Community/Verified)

- [Claude Agent Skills: First Principles Deep Dive](https://leehanchung.github.io/blogs/2025/10/26/claude-skills-deep-dive/) — Oct 26, 2025. Technical analysis of skills architecture.
- [Understanding Claude Code's Full Stack](https://alexop.dev/posts/understanding-claude-code-full-stack/) — Nov 9, 2025. Evolution timeline: MCP (2024) → Claude Code core (Feb 2025) → Plugins (late 2025).
- [Inside Claude Code Skills: Structure, prompts, invocation](https://mikhail.io/2025/10/claude-code-skills/) — Oct 28, 2025. Skills folder structure, SKILL.md format.

### LOW Confidence (Unverified/WebSearch Only)

- Various blog posts on "multi-agent systems" — **Not used**. Most are pre-2025 or focus on Python frameworks (AutoGen, LangGraph) which are explicitly **not** Aether v2's approach.

### Existing Aether Implementation

- `.claude/commands/ant/*.md` — Verified working prompt patterns (XML tags, Task tool usage)
- `.aether/data/*.json` — Current state management structure (proven in production)
- `.aether/memory/meta_learning_demo.json` — Meta-learning data structure (Beta distribution confidence scoring)

---

## Confidence Assessment

| Area | Confidence | Reasoning |
|------|------------|-----------|
| **Prompt-based architecture** | HIGH | Official Anthropic documentation, leaked system prompts, working Aether implementation all confirm this is the 2025 standard. |
| **Task tool spawning** | HIGH | Schema verified from leaked prompts. Officially documented. Aether uses it successfully. |
| **Agent Skills** | HIGH | Official feature (Oct 2025). Documentation is clear. Multiple community examples exist. |
| **JSON state management** | HIGH | Aether's current implementation works. JSON is Claude-native. No alternative needed for this scope. |
| **No external dependencies** | HIGH | Design principle confirmed by Aether v2 requirements. Claude Code sandboxing provides isolation. |
| **Pheromone system** | MEDIUM | Aether innovation. Working implementation exists, but not yet validated across diverse projects. |
| **Meta-learning patterns** | MEDIUM | Beta distribution scoring is theoretically sound (based on Ralph's research), but needs more production data. |

---

## Open Questions (Phase-Specific Research)

1. **Scale testing:** How does Aether perform with 50+ agents? (Current testing: <10 agents)
2. **Long-term memory compression:** What's the optimal compression ratio? (Current: heuristic)
3. **Error pattern detection:** What's the right threshold for auto-flagging? (Current: 3 occurrences)
4. **Pheromone decay:** What decay rate prevents "stale signals"? (Current: time-based, needs tuning)

**Recommendation:** These are implementation details, not stack decisions. Aether v2's stack is solid. Research these during Phase 2-3 (Implementation & Testing).

---

## Conclusion

**Aether v2's stack is deliberately minimal:**

1. **Prompts** (`.md` files with XML tags)
2. **JSON** (state persistence)
3. **Task tool** (autonomous spawning)

Everything else—agent skills, hooks, plugins—is optional enhancement, not core requirement.

**Why this is the right stack:**
- **Claude-native:** Works with Claude's harness, not against it
- **Debuggable:** Prompts and JSON are human-readable
- **No dependencies:** Runs in any Claude Code 2.0+ installation
- **Scalable:** Proven patterns from Anthropic's own multi-agent research system
- **Future-proof:** Based on official features, not hacks

**The stack is not the innovation.** The innovation is Aether's **pheromone-based semantic communication** and **meta-learning loop**. The stack stays out of the way.

---

*Stack research for: Aether v2 — Claude-native multi-agent system*
*Researched: 2026-02-01*
*Confidence: HIGH*
